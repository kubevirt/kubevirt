/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2017 Red Hat, Inc.
 *
 */

package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/golang/glog"
	flag "github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/api/errors"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes/scheme"
	k8coresv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/cert"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/certificates"
	"kubevirt.io/kubevirt/pkg/certificates/triple"
	"kubevirt.io/kubevirt/pkg/controller"
	inotifyinformer "kubevirt.io/kubevirt/pkg/inotify-informer"
	_ "kubevirt.io/kubevirt/pkg/monitoring/client/prometheus"    // import for prometheus metrics
	_ "kubevirt.io/kubevirt/pkg/monitoring/reflector/prometheus" // import for prometheus metrics
	promvm "kubevirt.io/kubevirt/pkg/monitoring/vms/prometheus"  // import for prometheus metrics
	_ "kubevirt.io/kubevirt/pkg/monitoring/workqueue/prometheus" // import for prometheus metrics
	"kubevirt.io/kubevirt/pkg/service"
	"kubevirt.io/kubevirt/pkg/util"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	virthandler "kubevirt.io/kubevirt/pkg/virt-handler"
	virtcache "kubevirt.io/kubevirt/pkg/virt-handler/cache"
	virtlauncher "kubevirt.io/kubevirt/pkg/virt-launcher"
	virt_api "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const (
	defaultWatchdogTimeout = 15 * time.Second

	// Default port that virt-handler listens on.
	defaultPort = 8185

	// Default address that virt-handler listens on.
	defaultHost = "0.0.0.0"

	hostOverride = ""

	podIpAddress = ""

	virtShareDir = "/var/run/kubevirt"

	// This value is derived from default MaxPods in Kubelet Config
	maxDevices = 110

	clientCertBytesValue  = "client-cert-bytes"
	clientKeyBytesValue   = "client-key-bytes"
	signingCertBytesValue = "signing-cert-bytes"

	// selfsigned cert secret name
	virtHandlerCertSecretName = "kubevirt-virt-handler-certs"
	maxRequestsInFlight       = 3
)

type virtHandlerApp struct {
	service.ServiceListen
	HostOverride            string
	PodIpAddress            string
	VirtShareDir            string
	WatchdogTimeoutDuration time.Duration
	MaxDevices              int
	MaxRequestsInFlight     int

	signingCertBytes []byte
	clientCertBytes  []byte
	clientKeyBytes   []byte

	virtCli   kubecli.KubevirtClient
	namespace string

	migrationTLSConfig *tls.Config
}

var _ service.Service = &virtHandlerApp{}

func (app *virtHandlerApp) getSelfSignedCert() error {
	var ok bool

	caKeyPair, _ := triple.NewCA("kubevirt.io")
	clientKeyPair, _ := triple.NewClientKeyPair(caKeyPair,
		"kubevirt.io:system:node:virt-handler",
		nil,
	)

	secret := &k8sv1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      virtHandlerCertSecretName,
			Namespace: app.namespace,
			Labels: map[string]string{
				v1.AppLabel: "virt-api-aggregator",
			},
		},
		Type: "Opaque",
		Data: map[string][]byte{
			clientCertBytesValue:  cert.EncodeCertPEM(clientKeyPair.Cert),
			clientKeyBytesValue:   cert.EncodePrivateKeyPEM(clientKeyPair.Key),
			signingCertBytesValue: cert.EncodeCertPEM(caKeyPair.Cert),
		},
	}
	_, err := app.virtCli.CoreV1().Secrets(app.namespace).Create(secret)
	if errors.IsAlreadyExists(err) {
		secret, err = app.virtCli.CoreV1().Secrets(app.namespace).Get(virtHandlerCertSecretName, metav1.GetOptions{})
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	// retrieve self signed cert info from secret
	app.clientCertBytes, ok = secret.Data[clientCertBytesValue]
	if !ok {
		return fmt.Errorf("%s value not found in %s virt-api secret", clientCertBytesValue, virtHandlerCertSecretName)
	}
	app.clientKeyBytes, ok = secret.Data[clientKeyBytesValue]
	if !ok {
		return fmt.Errorf("%s value not found in %s virt-api secret", clientKeyBytesValue, virtHandlerCertSecretName)
	}
	app.signingCertBytes, ok = secret.Data[signingCertBytesValue]
	if !ok {
		return fmt.Errorf("%s value not found in %s virt-api secret", signingCertBytesValue, virtHandlerCertSecretName)
	}
	return nil
}

func (app *virtHandlerApp) Run() {
	// HostOverride should default to os.Hostname(), to make sure we handle errors ensure it here.
	if app.HostOverride == "" {
		defaultHostName, err := os.Hostname()
		if err != nil {
			panic(err)
		}
		app.HostOverride = defaultHostName
	}

	if app.PodIpAddress == "" {
		panic(fmt.Errorf("no pod ip detected"))
	}

	logger := log.Log
	logger.V(1).Level(log.INFO).Log("hostname", app.HostOverride)

	// Create event recorder
	var err error
	app.virtCli, err = kubecli.GetKubevirtClient()
	if err != nil {
		panic(err)
	}
	broadcaster := record.NewBroadcaster()
	broadcaster.StartRecordingToSink(&k8coresv1.EventSinkImpl{Interface: app.virtCli.CoreV1().Events(k8sv1.NamespaceAll)})
	// Scheme is used to create an ObjectReference from an Object (e.g. VirtualMachineInstance) during Event creation
	recorder := broadcaster.NewRecorder(scheme.Scheme, k8sv1.EventSource{Component: "virt-handler", Host: app.HostOverride})

	vmiSourceLabel, err := labels.Parse(fmt.Sprintf(v1.NodeNameLabel+" in (%s)", app.HostOverride))
	if err != nil {
		panic(err)
	}
	vmiTargetLabel, err := labels.Parse(fmt.Sprintf(v1.MigrationTargetNodeNameLabel+" in (%s)", app.HostOverride))
	if err != nil {
		panic(err)
	}

	// Wire VirtualMachineInstance controller

	vmSourceSharedInformer := cache.NewSharedIndexInformer(
		controller.NewListWatchFromClient(app.virtCli.RestClient(), "virtualmachineinstances", k8sv1.NamespaceAll, fields.Everything(), vmiSourceLabel),
		&v1.VirtualMachineInstance{},
		0,
		cache.Indexers{},
	)

	vmTargetSharedInformer := cache.NewSharedIndexInformer(
		controller.NewListWatchFromClient(app.virtCli.RestClient(), "virtualmachineinstances", k8sv1.NamespaceAll, fields.Everything(), vmiTargetLabel),
		&v1.VirtualMachineInstance{},
		0,
		cache.Indexers{},
	)

	// Wire Domain controller
	domainSharedInformer, err := virtcache.NewSharedInformer(app.VirtShareDir, int(app.WatchdogTimeoutDuration.Seconds()), recorder, vmSourceSharedInformer.GetStore())
	if err != nil {
		panic(err)
	}

	virtlauncher.InitializeSharedDirectories(app.VirtShareDir)

	app.namespace, err = util.GetNamespace()
	if err != nil {
		glog.Fatalf("Error searching for namespace: %v", err)
	}

	if err := app.getSelfSignedCert(); err != nil {
		glog.Fatalf("Error loading self signed certificates: %v", err)
	}

	if err := app.setupTLS(); err != nil {
		glog.Fatalf("Error constructing migration tls config: %v", err)
	}

	factory := controller.NewKubeInformerFactory(app.virtCli.RestClient(), app.virtCli, app.namespace)

	gracefulShutdownInformer := cache.NewSharedIndexInformer(
		inotifyinformer.NewFileListWatchFromClient(
			virtlauncher.GracefulShutdownTriggerDir(app.VirtShareDir)),
		&virt_api.Domain{},
		0,
		cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})

	vmController := virthandler.NewController(
		recorder,
		app.virtCli,
		app.HostOverride,
		app.PodIpAddress,
		app.VirtShareDir,
		vmSourceSharedInformer,
		vmTargetSharedInformer,
		domainSharedInformer,
		gracefulShutdownInformer,
		int(app.WatchdogTimeoutDuration.Seconds()),
		app.MaxDevices,
		virtconfig.NewClusterConfig(factory.ConfigMap(), factory.CRD(), app.namespace),
		app.migrationTLSConfig,
	)

	certsDirectory, err := ioutil.TempDir("", "certsdir")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(certsDirectory)

	certStore, err := certificates.GenerateSelfSignedCert(certsDirectory, "virt-handler", app.namespace)
	if err != nil {
		glog.Fatalf("unable to generate certificates: %v", err)
	}

	promvm.SetupCollector(app.virtCli, app.VirtShareDir, app.HostOverride)

	// Bootstrapping. From here on the startup order matters
	stop := make(chan struct{})
	defer close(stop)
	factory.Start(stop)
	cache.WaitForCacheSync(stop, factory.ConfigMap().HasSynced)

	go vmController.Run(10, stop)

	logger.V(1).Infof("metrics: max concurrent requests=%d", app.MaxRequestsInFlight)
	http.Handle("/metrics", promvm.Handler(app.MaxRequestsInFlight))

	err = http.ListenAndServeTLS(app.ServiceListen.Address(), certStore.CurrentPath(), certStore.CurrentPath(), nil)
	if err != nil {
		log.Log.Reason(err).Error("Serving prometheus failed.")
		panic(err)
	}
}

func (app *virtHandlerApp) AddFlags() {
	app.InitFlags()

	app.BindAddress = defaultHost
	app.Port = defaultPort

	app.AddCommonFlags()

	flag.StringVar(&app.HostOverride, "hostname-override", hostOverride,
		"Name under which the node is registered in Kubernetes, where this virt-handler instance is running on")

	flag.StringVar(&app.PodIpAddress, "pod-ip-address", podIpAddress,
		"The pod ip address")

	flag.StringVar(&app.VirtShareDir, "kubevirt-share-dir", virtShareDir,
		"Shared directory between virt-handler and virt-launcher")

	flag.DurationVar(&app.WatchdogTimeoutDuration, "watchdog-timeout", defaultWatchdogTimeout,
		"Watchdog file timeout")

	// TODO: the Device Plugin API does not allow for infinitely available (shared) devices
	// so the current approach is to register an arbitrary number.
	// This should be deprecated if the API allows for shared resources in the future
	flag.IntVar(&app.MaxDevices, "max-devices", maxDevices,
		"Number of devices to register with Kubernetes device plugin framework")

	flag.IntVar(&app.MaxRequestsInFlight, "max-metric-requests", maxRequestsInFlight,
		"Number of concurrent requests to the metrics endpoint")
}

func (app *virtHandlerApp) setupTLS() error {

	clientCert, err := tls.X509KeyPair(app.clientCertBytes, app.clientKeyBytes)
	if err != nil {
		return err
	}

	caCert, err := cert.ParseCertsPEM(app.signingCertBytes)
	if err != nil {
		return err
	}

	certPool := x509.NewCertPool()

	for _, crt := range caCert {
		certPool.AddCert(crt)
	}

	app.migrationTLSConfig = &tls.Config{
		MinVersion: tls.VersionTLS12,
		ClientCAs:  certPool,
		GetClientCertificate: func(info *tls.CertificateRequestInfo) (certificate *tls.Certificate, e error) {
			return &clientCert, nil
		},
		GetCertificate: func(info *tls.ClientHelloInfo) (i *tls.Certificate, e error) {
			return &clientCert, nil
		},
		// Neither the client nor the server should validate anything itself, `VerifyPeerCertificate` is still executed
		InsecureSkipVerify: true,
		// XXX: We need to verify the cert ourselves because we don't have DNS or IP on the certs at the moment
		VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {

			// impossible with RequireAnyClientCert
			if len(rawCerts) == 0 {
				return fmt.Errorf("no client certificate provided.")
			}

			c, err := x509.ParseCertificate(rawCerts[0])
			if err != nil {
				return fmt.Errorf("failed to parse peer certificate: %v", err)
			}
			_, err = c.Verify(x509.VerifyOptions{
				Roots:     certPool,
				KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
			})

			if err != nil {
				return fmt.Errorf("could not verify peer certificate: %v", err)
			}
			return nil
		},
		ClientAuth: tls.RequireAndVerifyClientCert,
	}

	return nil
}

func main() {
	app := &virtHandlerApp{}
	service.Setup(app)
	log.InitializeLogging("virt-handler")
	app.Run()
}
