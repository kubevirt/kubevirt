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
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright The KubeVirt Authors.
 *
 */

package main

import (
	"context"
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/util/certificate"
	"k8s.io/client-go/util/flowcontrol"

	k8sv1 "k8s.io/api/core/v1"
	k8coresv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"

	"kubevirt.io/kubevirt/pkg/healthz"
	synccontrollermetrics "kubevirt.io/kubevirt/pkg/monitoring/metrics/virt-synchronization-controller"
	"kubevirt.io/kubevirt/pkg/service"
	kvtls "kubevirt.io/kubevirt/pkg/util/tls"
	"kubevirt.io/kubevirt/pkg/virt-controller/leaderelectionconfig"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/kubevirt/scheme"
	"kubevirt.io/client-go/log"
	clientutil "kubevirt.io/client-go/util"

	"kubevirt.io/kubevirt/pkg/certificates/bootstrap"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/synchronization-controller"
	"kubevirt.io/kubevirt/pkg/util/ratelimiter"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	virthandler "kubevirt.io/kubevirt/pkg/virt-handler"
)

const (
	// Default port that virt-handler listens on.
	defaultPort = 9185
	// Default address that virt-handler listens on.
	defaultHost = "0.0.0.0"

	// Default ConfigMap name of CA
	defaultCAConfigMapName = "kubevirt-ca"

	// Default certificate and key paths
	defaultClientCertFilePath = "/etc/virt-sync-controller/clientcertificates/tls.crt"
	defaultClientKeyFilePath  = "/etc/virt-sync-controller/clientcertificates/tls.key"
	defaultTlsCertFilePath    = "/etc/virt-sync-controller/servercertificates/tls.crt"
	defaultTlsKeyFilePath     = "/etc/virt-sync-controller/servercertificates/tls.key"
	// Virt-handler migration certs for terminating TLS on the proxy data path
	defaultMigrationClientCertFilePath = "/etc/virt-handler/migrationclientcertificates/tls.crt"
	defaultMigrationClientKeyFilePath  = "/etc/virt-handler/migrationclientcertificates/tls.key"
	defaultMigrationServerCertFilePath = "/etc/virt-handler/migrationservercertificates/tls.crt"
	defaultMigrationServerKeyFilePath  = "/etc/virt-handler/migrationservercertificates/tls.key"

	defaultGracefulShutdownSeconds = 30
	maxRetryCount                  = 10
	leaseName                      = "virt-synchronization-controller"
)

var (
	apiHealthVersion = new(healthz.KubeApiHealthzVersion)
)

type synchronizationControllerApp struct {
	service.ServiceListen

	virtCli        kubecli.KubevirtClient
	namespace      string
	LeaderElection leaderelectionconfig.Configuration

	caConfigMapName             string
	clientCertFilePath          string
	clientKeyFilePath           string
	serverCertFilePath          string
	serverKeyFilePath           string
	migrationClientCertFilePath string
	migrationClientKeyFilePath  string
	migrationServerCertFilePath string
	migrationServerKeyFilePath  string
	externallyManaged           bool
	ip                          string
	podIP                       string

	serverTLSConfig            *tls.Config
	clientTLSConfig            *tls.Config
	migrationClientTLSConfig   *tls.Config
	migrationServerTLSConfig   *tls.Config
	clientcertmanager          certificate.Manager
	servercertmanager          certificate.Manager
	migrationclientcertmanager certificate.Manager
	migrationservercertmanager certificate.Manager
	clusterConfig              *virtconfig.ClusterConfig
	reloadableRateLimiter      *ratelimiter.ReloadableRateLimiter
	caManager                  kvtls.ClientCAManager

	ctx context.Context
}

func (app *synchronizationControllerApp) prepareCertManager() (err error) {
	app.clientcertmanager = bootstrap.NewFileCertificateManager(app.clientCertFilePath, app.clientKeyFilePath)
	app.servercertmanager = bootstrap.NewFileCertificateManager(app.serverCertFilePath, app.serverKeyFilePath)
	app.migrationclientcertmanager = bootstrap.NewFileCertificateManager(app.migrationClientCertFilePath, app.migrationClientKeyFilePath)
	app.migrationservercertmanager = bootstrap.NewFileCertificateManager(app.migrationServerCertFilePath, app.migrationServerKeyFilePath)
	return
}

// Update synchronization controller log verbosity on relevant config changes
func (app *synchronizationControllerApp) shouldChangeLogVerbosity() {
	verbosity := app.clusterConfig.GetVirtSynchronizationControllerVerbosity()
	if verbosity == 0 {
		// If the verbosity gets set in kubevirt CR, this will not be 0, but it is otherwise.
		verbosity = 2
	}
	err := log.Log.SetVerbosityLevel(int(verbosity))
	if err != nil {
		log.Log.Errorf("unable to change log verbosity to %d, %v", verbosity, err)
	} else {
		log.Log.V(2).Infof("set verbosity to %d", verbosity)
	}
}

// Update synchronization controller rate limiter
func (app *synchronizationControllerApp) shouldChangeRateLimiter() {
	config := app.clusterConfig.GetConfig()
	qps := config.HandlerConfiguration.RestClient.RateLimiter.TokenBucketRateLimiter.QPS
	burst := config.HandlerConfiguration.RestClient.RateLimiter.TokenBucketRateLimiter.Burst
	app.reloadableRateLimiter.Set(flowcontrol.NewTokenBucketRateLimiter(qps, burst))
	log.Log.V(2).Infof("setting rate limiter to %v QPS and %v Burst", qps, burst)
}

func (app *synchronizationControllerApp) setupTLS(factory controller.KubeInformerFactory) error {
	kubevirtCAConfigInformer := factory.KubeVirtCAConfigMap()
	if err := kubevirtCAConfigInformer.SetWatchErrorHandler(func(r *cache.Reflector, err error) {
		apiHealthVersion.Clear()
		cache.DefaultWatchErrorHandler(context.TODO(), r, err)
	}); err != nil {
		return err
	}

	app.caManager = kvtls.NewCAManager(kubevirtCAConfigInformer.GetStore(), app.namespace, app.caConfigMapName)

	app.serverTLSConfig = kvtls.SetupTLSForVirtSynchronizationControllerServer(app.caManager, app.servercertmanager, app.externallyManaged, app.clusterConfig)
	app.clientTLSConfig = kvtls.SetupTLSForVirtSynchronizationControllerClients(app.caManager, app.clientcertmanager, app.externallyManaged)

	// Terminating TLS for virt-handler ↔ proxy:
	// - client config (migration client cert): dial target virt-handler
	// - server config (virt-handler server cert): accept source virt-handler
	app.migrationClientTLSConfig = kvtls.SetupTLSForVirtHandlerClients(app.caManager, app.migrationclientcertmanager, app.externallyManaged)
	app.migrationServerTLSConfig = kvtls.SetupTLSForVirtHandlerServer(app.caManager, app.migrationservercertmanager, app.externallyManaged, app.clusterConfig, []string{"virt-handler", "migration"})

	return nil
}

func (app *synchronizationControllerApp) Run() {
	var err error

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	app.ctx = ctx

	envIP, _ := os.LookupEnv("MY_POD_IP")
	app.podIP = envIP
	ip, err := virthandler.FindMigrationIP(envIP)
	app.ip = ip

	app.LeaderElection = leaderelectionconfig.DefaultLeaderElectionConfiguration()

	app.reloadableRateLimiter = ratelimiter.NewReloadableRateLimiter(flowcontrol.NewTokenBucketRateLimiter(virtconfig.DefaultVirtControllerQPS, virtconfig.DefaultVirtHandlerBurst))
	var clientConfig *rest.Config
	retryCount := 0
	for retryCount = 0; retryCount < maxRetryCount; retryCount++ {
		clientConfig, err = kubecli.GetKubevirtClientConfig()
		if err != nil {
			log.Log.Errorf("unable to get kubevirt client config %v", err)
			waitTime := 2 ^ (retryCount + 1)
			time.Sleep(time.Duration(waitTime) * time.Millisecond)
			continue
		}
		break
	}
	if retryCount >= maxRetryCount {
		panic(fmt.Errorf("unable to get kubevirt client config after %d retries %v", maxRetryCount, err))
	}

	clientConfig.RateLimiter = app.reloadableRateLimiter
	for retryCount = 0; retryCount < maxRetryCount; retryCount++ {
		app.virtCli, err = kubecli.GetKubevirtClientFromRESTConfig(clientConfig)
		if err != nil {
			log.Log.Errorf("unable to get kubevirt client from rest config %v", err)
			waitTime := 2 ^ (retryCount + 1)
			time.Sleep(time.Duration(waitTime) * time.Millisecond)
			continue
		}
		break
	}
	if retryCount >= maxRetryCount {
		panic(fmt.Errorf("unable to get kubevirt client from rest config after %d retries %v", maxRetryCount, err))
	}

	app.namespace, err = clientutil.GetNamespace()
	if err != nil {
		log.Log.Criticalf("Error searching for namespace: %v", err)
		os.Exit(2)
	}
	log.Log.V(1).Infof("running in namespace %s", app.namespace)
	factory := controller.NewKubeInformerFactory(app.virtCli.RestClient(), app.virtCli, app.virtCli, nil, app.namespace)

	vmiInformer := factory.VMI()
	migrationInformer := factory.VirtualMachineInstanceMigration()

	if err := app.prepareCertManager(); err != nil {
		log.Log.Criticalf("Error preparing the certificate manager: %v", err)
		os.Exit(2)
	}

	app.clusterConfig, err = virtconfig.NewClusterConfig(factory.CRD(), factory.KubeVirt(), app.namespace)
	if err != nil {
		panic(err)
	}
	// set log verbosity
	app.clusterConfig.SetConfigModifiedCallback(app.shouldChangeLogVerbosity)
	// set rate limiter
	app.clusterConfig.SetConfigModifiedCallback(app.shouldChangeRateLimiter)

	if err := app.setupTLS(factory); err != nil {
		log.Log.Criticalf("Error constructing migration tls config: %v", err)
		os.Exit(2)
	}

	stop := app.ctx.Done()

	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt,
		os.Kill,
		syscall.SIGHUP,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)
	go func() {
		s := <-sigint
		log.Log.Infof("received signal %s, initiating graceful shutdown", s.String())
		cancel()
	}()

	synchronizationController, err := synchronization.NewSynchronizationController(
		app.virtCli,
		vmiInformer,
		migrationInformer,
		app.clientTLSConfig,
		app.serverTLSConfig,
		app.migrationClientTLSConfig,
		app.migrationServerTLSConfig,
		app.BindAddress,
		app.Port,
		app.ip,
	)
	if err != nil {
		panic(err)
	}

	go app.clientcertmanager.Start()
	go app.servercertmanager.Start()
	go app.migrationclientcertmanager.Start()
	go app.migrationservercertmanager.Start()

	factory.Start(stop)
	app.runWithLeaderElection(synchronizationController, stop)
}

func (app *synchronizationControllerApp) runWithLeaderElection(synchronizationController *synchronization.SynchronizationController, stop <-chan struct{}) {
	recorder := app.getNewRecorder(k8sv1.NamespaceAll, leaseName)

	id, err := os.Hostname()
	if err != nil {
		log.Log.Criticalf("unable to get hostname: %v", err)
		panic(err)
	}

	tlsConfig := kvtls.SetupPromTLS(app.servercertmanager, app.clusterConfig)
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", app.healthzHandler)
	mux.Handle("/metrics", synccontrollermetrics.Handler(0))

	log.Log.V(2).Infof("Listing on %s", app.Address())

	// Serve metrics/healthz on the primary pod IP (kubelet probes) and localhost
	// (kubectl port-forward). Do not use 0.0.0.0 — that would also expose 8443 on
	// secondary networks (internal migration / cross-cluster Multus interfaces).
	metricsHosts := []string{"127.0.0.1"}
	if app.podIP != "" && app.podIP != "127.0.0.1" {
		metricsHosts = append(metricsHosts, app.podIP)
	}
	var metricsServers []*http.Server
	for _, host := range metricsHosts {
		metricsAddr := net.JoinHostPort(host, "8443")
		server := &http.Server{
			Addr:      metricsAddr,
			Handler:   mux,
			TLSConfig: tlsConfig.Clone(),
			// Disable HTTP/2
			// See CVE-2023-44487
			TLSNextProto: map[string]func(*http.Server, *tls.Conn, http.Handler){},
		}
		metricsServers = append(metricsServers, server)
		go func(s *http.Server) {
			log.Log.V(2).Infof("/healthz and /metrics listening on %s", s.Addr)
			for {
				if err := s.ListenAndServeTLS("", ""); err != nil {
					if errors.Is(err, http.ErrServerClosed) {
						log.Log.V(1).Infof("shut down http server on %s", s.Addr)
						return
					}
					log.Log.Errorf("unable to listen and serve TLS on %s: %v, retrying in 1 second", s.Addr, err)
				}
				time.Sleep(time.Second)
			}
		}(server)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-stop
		app.close()
		httpShutdownCTX, httpShutdownCancel := context.WithTimeout(context.Background(), defaultGracefulShutdownSeconds*time.Second)
		defer httpShutdownCancel()

		for _, s := range metricsServers {
			if err := s.Shutdown(httpShutdownCTX); err != nil {
				log.Log.Errorf("server shutdown error on %s: %v", s.Addr, err)
			}
		}
		log.Log.V(1).Info("completed stop function")
	}()

	rl, err := resourcelock.New(app.LeaderElection.ResourceLock,
		app.namespace,
		leaseName,
		app.virtCli.CoreV1(),
		app.virtCli.CoordinationV1(),
		resourcelock.ResourceLockConfig{
			Identity:      id,
			EventRecorder: recorder,
		})
	if err != nil {
		panic(err)
	}

	controllerContext, controllerCancel := context.WithCancel(context.Background())

	wg.Add(1)
	leaderElector, err := leaderelection.NewLeaderElector(
		leaderelection.LeaderElectionConfig{
			Lock:          rl,
			LeaseDuration: app.LeaderElection.LeaseDuration.Duration,
			RenewDeadline: app.LeaderElection.RenewDeadline.Duration,
			RetryPeriod:   app.LeaderElection.RetryPeriod.Duration,
			Callbacks: leaderelection.LeaderCallbacks{
				OnStartedLeading: func(ctx context.Context) {
					// run app
					if err := synchronizationController.Run(10, controllerContext.Done()); err != nil {
						panic(err)
					}
					log.Log.Info("successfully shut down controller")
					wg.Done()
				},
				OnStoppedLeading: func() {
					log.Log.Error("leaderelection lost, shutting down controller")
					controllerCancel()
				},
			},
		})
	if err != nil {
		panic(err)
	}
	leaderElector.Run(app.ctx)
	wg.Wait()
}

func (app *synchronizationControllerApp) getNewRecorder(namespace string, componentName string) record.EventRecorder {
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartRecordingToSink(&k8coresv1.EventSinkImpl{Interface: app.virtCli.CoreV1().Events(namespace)})
	return eventBroadcaster.NewRecorder(scheme.Scheme, k8sv1.EventSource{Component: componentName})
}

func (app *synchronizationControllerApp) close() {
	log.Log.V(1).Info("stopping client and server cert managers")
	// release resources associated with the application
	app.clientcertmanager.Stop()
	app.servercertmanager.Stop()
	app.migrationclientcertmanager.Stop()
	app.migrationservercertmanager.Stop()
}

func (app *synchronizationControllerApp) healthzHandler(w http.ResponseWriter, _ *http.Request) {
	io.WriteString(w, "OK")
}

func (app *synchronizationControllerApp) AddFlags() {
	app.InitFlags()
	app.AddCommonFlags()

	if app.BindAddress == "" {
		app.BindAddress = defaultHost
	}
	if app.Port == 0 {
		app.Port = defaultPort
	}
	flag.StringVar(&app.caConfigMapName, "ca-configmap-name", defaultCAConfigMapName,
		"The name of configmap containing CA certificates to authenticate requests presenting client certificates with matching CommonName")

	flag.StringVar(&app.clientCertFilePath, "client-cert-file", defaultClientCertFilePath,
		"Client certificate used to prove the identity of the virt-handler when it must call out during a request")

	flag.StringVar(&app.clientKeyFilePath, "client-key-file", defaultClientKeyFilePath,
		"Private key for the client certificate used to prove the identity of the virt-handler when it must call out during a request")

	flag.StringVar(&app.serverCertFilePath, "tls-cert-file", defaultTlsCertFilePath,
		"File containing the default x509 Certificate for HTTPS")

	flag.StringVar(&app.serverKeyFilePath, "tls-key-file", defaultTlsKeyFilePath,
		"File containing the default x509 private key matching --tls-cert-file")

	flag.StringVar(&app.migrationClientCertFilePath, "migration-client-cert-file", defaultMigrationClientCertFilePath,
		"Client certificate with CN='migration' used to connect to virt-handler migration proxies")

	flag.StringVar(&app.migrationClientKeyFilePath, "migration-client-key-file", defaultMigrationClientKeyFilePath,
		"Private key for the migration client certificate")

	flag.StringVar(&app.migrationServerCertFilePath, "migration-server-cert-file", defaultMigrationServerCertFilePath,
		"Server certificate (virt-handler server cert) used to accept TLS from source virt-handlers")

	flag.StringVar(&app.migrationServerKeyFilePath, "migration-server-key-file", defaultMigrationServerKeyFilePath,
		"Private key for the migration server certificate")

	flag.BoolVar(&app.externallyManaged, "externally-managed", false,
		"Allow intermediate certificates to be used in building up the chain of trust when certificates are externally managed")
}

func main() {
	app := &synchronizationControllerApp{}
	service.Setup(app)
	app.Run()
	log.Log.Info("successfully shutdown")
}
