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
	"net/http"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/emicklei/go-restful"
	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	flag "github.com/spf13/pflag"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes/scheme"
	k8coresv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/certificate"

	"kubevirt.io/kubevirt/pkg/certificates/bootstrap"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/controller"
	inotifyinformer "kubevirt.io/kubevirt/pkg/inotify-informer"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
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

	virtShareDir = "/var/run/kubevirt"

	// This value is derived from default MaxPods in Kubelet Config
	maxDevices = 110
)

type virtHandlerApp struct {
	service.ServiceListen
	HostOverride            string
	VirtShareDir            string
	WatchdogTimeoutDuration time.Duration
	MaxDevices              int
}

var _ service.Service = &virtHandlerApp{}

func (app *virtHandlerApp) Run() {
	// HostOverride should default to os.Hostname(), to make sure we handle errors ensure it here.
	if app.HostOverride == "" {
		defaultHostName, err := os.Hostname()
		if err != nil {
			panic(err)
		}
		app.HostOverride = defaultHostName
	}

	if app.PodName == "" {
		app.PodName = app.HostOverride
	}

	logger := log.Log
	logger.V(1).Level(log.INFO).Log("hostname", app.HostOverride)

	// Create event recorder
	virtCli, err := kubecli.GetKubevirtClient()
	if err != nil {
		panic(err)
	}
	broadcaster := record.NewBroadcaster()
	broadcaster.StartRecordingToSink(&k8coresv1.EventSinkImpl{Interface: virtCli.CoreV1().Events(k8sv1.NamespaceAll)})
	// Scheme is used to create an ObjectReference from an Object (e.g. VirtualMachineInstance) during Event creation
	recorder := broadcaster.NewRecorder(scheme.Scheme, k8sv1.EventSource{Component: app.GetName(), Host: app.HostOverride})

	if err != nil {
		panic(err)
	}

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
		controller.NewListWatchFromClient(virtCli.RestClient(), "virtualmachineinstances", k8sv1.NamespaceAll, fields.Everything(), vmiSourceLabel),
		&v1.VirtualMachineInstance{},
		0,
		cache.Indexers{},
	)

	vmTargetSharedInformer := cache.NewSharedIndexInformer(
		controller.NewListWatchFromClient(virtCli.RestClient(), "virtualmachineinstances", k8sv1.NamespaceAll, fields.Everything(), vmiTargetLabel),
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

	namespace, err := util.GetNamespace()
	if err != nil {
		glog.Fatalf("Error searching for namespace: %v", err)
	}

	factory := controller.NewKubeInformerFactory(virtCli.RestClient(), virtCli, namespace)

	stop := make(chan struct{})
	defer close(stop)

	app.SetupCertificateManager(virtCli, bootstrap.LoadCertConfigForNode)

	gracefulShutdownInformer := cache.NewSharedIndexInformer(
		inotifyinformer.NewFileListWatchFromClient(
			virtlauncher.GracefulShutdownTriggerDir(app.VirtShareDir)),
		&virt_api.Domain{},
		0,
		cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})

	vmController := virthandler.NewController(
		recorder,
		virtCli,
		app.HostOverride,
		app.PodIpAddress.String(),
		app.VirtShareDir,
		vmSourceSharedInformer,
		vmTargetSharedInformer,
		domainSharedInformer,
		gracefulShutdownInformer,
		int(app.WatchdogTimeoutDuration.Seconds()),
		app.MaxDevices,
		virtconfig.NewClusterConfig(factory.ConfigMap().GetStore(), namespace),
		newMigrationTLSConfig(app.RootCAPool, app.CertificateManager),
	)

	factory.Start(stop)
	cache.WaitForCacheSync(stop, factory.ConfigMap().HasSynced)
	go vmController.Run(3, stop)

	handler := http.NewServeMux()
	server := &http.Server{
		Addr:      app.ServiceListen.Address(),
		TLSConfig: app.PromTLSConfig,
		Handler:   handler,
	}

	promvm.SetupCollector(app.VirtShareDir)

	handler.Handle("/metrics", promhttp.Handler())
	handler.Handle("/", restful.DefaultContainer)
	err = server.ListenAndServeTLS("", "")
	if err != nil {
		log.Log.Reason(err).Error("Serving prometheus failed.")
		panic(err)
	}
}

func newMigrationTLSConfig(certPool *x509.CertPool, manager certificate.Manager) *tls.Config {
	migrationTLSConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
		ClientAuth: tls.RequireAndVerifyClientCert,
		ClientCAs:  certPool,
		RootCAs:    certPool,
		GetClientCertificate: func(info *tls.CertificateRequestInfo) (*tls.Certificate, error) {
			cert := manager.Current()
			if cert == nil {
				return nil, fmt.Errorf("no serving certificate available for virt-handler")
			}
			return cert, nil
		},
		GetCertificate: func(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
			cert := manager.Current()
			if cert == nil {
				return nil, fmt.Errorf("no serving certificate available for virt-handler")
			}
			return cert, nil
		},
		VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
			c, err := x509.ParseCertificate(rawCerts[0])
			if err != nil {
				return fmt.Errorf("failed to parse certificate: %v", err)
			}
			if !reflect.DeepEqual([]string{"kubevirt.io:system"}, c.Subject.Organization) {
				return fmt.Errorf("expected organization kubevirt.io:system, but got: %v", c.Subject.Organization)
			}
			if !strings.HasPrefix(c.Subject.CommonName, "kubevirt.io:system:nodes") {
				return fmt.Errorf("expected common name with kubevirt.io:system:nodes prefix, but got: %s", c.Subject.CommonName)
			}
			return nil
		},
	}
	return migrationTLSConfig
}

func (app *virtHandlerApp) AddFlags() {
	app.InitFlags()

	app.BindAddress = defaultHost
	app.Port = defaultPort

	app.AddCommonFlags()

	flag.StringVar(&app.HostOverride, "hostname-override", hostOverride,
		"Name under which the node is registered in Kubernetes, where this virt-handler instance is running on")

	flag.StringVar(&app.VirtShareDir, "kubevirt-share-dir", virtShareDir,
		"Shared directory between virt-handler and virt-launcher")

	flag.DurationVar(&app.WatchdogTimeoutDuration, "watchdog-timeout", defaultWatchdogTimeout,
		"Watchdog file timeout")

	// TODO: the Device Plugin API does not allow for infinitely available (shared) devices
	// so the current approach is to register an arbitrary number.
	// This should be deprecated if the API allows for shared resources in the future
	flag.IntVar(&app.MaxDevices, "max-devices", maxDevices,
		"Number of devices to register with Kubernetes device plugin framework")
}

func main() {
	app := &virtHandlerApp{ServiceListen: service.ServiceListen{Name: "virt-handler"}}
	service.Setup(app)
	app.Run()
}
