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
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"time"

	kvtls "kubevirt.io/kubevirt/pkg/util/tls"
	"kubevirt.io/kubevirt/pkg/virt-handler/seccomp"
	"kubevirt.io/kubevirt/pkg/virt-handler/vsock"

	"github.com/emicklei/go-restful/v3"
	"github.com/golang/glog"
	flag "github.com/spf13/pflag"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	utilwait "k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/scheme"
	k8coresv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/certificate"
	"k8s.io/client-go/util/flowcontrol"

	"kubevirt.io/kubevirt/pkg/safepath"

	"kubevirt.io/kubevirt/pkg/util/ratelimiter"

	"kubevirt.io/kubevirt/pkg/virt-handler/node-labeller/api"

	"kubevirt.io/kubevirt/pkg/monitoring/domainstats/downwardmetrics"

	"kubevirt.io/kubevirt/pkg/healthz"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	clientutil "kubevirt.io/client-go/util"

	"kubevirt.io/kubevirt/pkg/certificates/bootstrap"
	containerdisk "kubevirt.io/kubevirt/pkg/container-disk"
	"kubevirt.io/kubevirt/pkg/controller"
	inotifyinformer "kubevirt.io/kubevirt/pkg/inotify-informer"
	_ "kubevirt.io/kubevirt/pkg/monitoring/client/prometheus"               // import for prometheus metrics
	promdomain "kubevirt.io/kubevirt/pkg/monitoring/domainstats/prometheus" // import for prometheus metrics
	"kubevirt.io/kubevirt/pkg/monitoring/profiler"
	_ "kubevirt.io/kubevirt/pkg/monitoring/reflector/prometheus" // import for prometheus metrics
	_ "kubevirt.io/kubevirt/pkg/monitoring/workqueue/prometheus" // import for prometheus metrics
	"kubevirt.io/kubevirt/pkg/service"
	"kubevirt.io/kubevirt/pkg/util"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	virthandler "kubevirt.io/kubevirt/pkg/virt-handler"
	virtcache "kubevirt.io/kubevirt/pkg/virt-handler/cache"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
	dmetricsmanager "kubevirt.io/kubevirt/pkg/virt-handler/dmetrics-manager"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
	migrationproxy "kubevirt.io/kubevirt/pkg/virt-handler/migration-proxy"
	nodelabeller "kubevirt.io/kubevirt/pkg/virt-handler/node-labeller"
	"kubevirt.io/kubevirt/pkg/virt-handler/rest"
	"kubevirt.io/kubevirt/pkg/virt-handler/selinux"
	virt_api "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/watchdog"
)

const (
	defaultWatchdogTimeout = 30 * time.Second

	// Default port that virt-handler listens on.
	defaultPort = 8185

	// Default address that virt-handler listens on.
	defaultHost = "0.0.0.0"

	hostOverride = ""

	podIpAddress = ""

	// This value reflects in the max number of VMIs per node
	maxDevices = 1000

	maxRequestsInFlight = 3
	// Default port that virt-handler listens to console requests
	defaultConsoleServerPort = 8186

	// Default period for resyncing virt-launcher domain cache
	defaultDomainResyncPeriodSeconds = 300

	// Default seconds to wait for migration connections to terminate before shutting down
	defaultGracefulShutdownSeconds = 300

	// Default ConfigMap name of CA
	defaultCAConfigMapName = "kubevirt-ca"

	// Default certificate and key paths
	defaultClientCertFilePath = "/etc/virt-handler/clientcertificates/tls.crt"
	defaultClientKeyFilePath  = "/etc/virt-handler/clientcertificates/tls.key"
	defaultTlsCertFilePath    = "/etc/virt-handler/servercertificates/tls.crt"
	defaultTlsKeyFilePath     = "/etc/virt-handler/servercertificates/tls.key"

	// Default network-status downward API file path
	defaultNetworkStatusFilePath = "/etc/podinfo/network-status"
)

type virtHandlerApp struct {
	service.ServiceListen
	HostOverride              string
	PodIpAddress              string
	VirtShareDir              string
	VirtPrivateDir            string
	VirtLibDir                string
	KubeletPodsDir            string
	KubeletRoot               string
	WatchdogTimeoutDuration   time.Duration
	MaxDevices                int
	MaxRequestsInFlight       int
	domainResyncPeriodSeconds int
	gracefulShutdownSeconds   int

	// Remember whether we have already installed the custom SELinux policy or not
	customSELinuxPolicyInstalled bool
	semoduleLock                 sync.Mutex

	caConfigMapName    string
	clientCertFilePath string
	clientKeyFilePath  string
	serverCertFilePath string
	serverKeyFilePath  string
	externallyManaged  bool

	virtCli   kubecli.KubevirtClient
	namespace string

	serverTLSConfig       *tls.Config
	clientTLSConfig       *tls.Config
	consoleServerPort     int
	clientcertmanager     certificate.Manager
	servercertmanager     certificate.Manager
	promTLSConfig         *tls.Config
	clusterConfig         *virtconfig.ClusterConfig
	reloadableRateLimiter *ratelimiter.ReloadableRateLimiter
	caManager             kvtls.ClientCAManager
}

var (
	_                service.Service = &virtHandlerApp{}
	apiHealthVersion                 = new(healthz.KubeApiHealthzVersion)
)

func (app *virtHandlerApp) prepareCertManager() (err error) {
	app.clientcertmanager = bootstrap.NewFileCertificateManager(app.clientCertFilePath, app.clientKeyFilePath)
	app.servercertmanager = bootstrap.NewFileCertificateManager(app.serverCertFilePath, app.serverKeyFilePath)
	return
}

func (app *virtHandlerApp) markNodeAsUnschedulable(logger *log.FilteredLogger) {
	data := []byte(fmt.Sprintf(`{"metadata": { "labels": {"%s": "false"}}}`, v1.NodeSchedulable))
	_, err := app.virtCli.CoreV1().Nodes().Patch(context.Background(), app.HostOverride, types.StrategicMergePatchType, data, metav1.PatchOptions{})
	if err != nil {
		logger.Reason(err).Error("Unable to mark node as unschedulable")
	}
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
	logger.V(1).Infof("hostname %s", app.HostOverride)
	var err error

	// Copy container-disk binary
	targetFile := filepath.Join(app.VirtLibDir, "/init/usr/bin/container-disk")
	err = os.MkdirAll(filepath.Dir(targetFile), os.ModePerm)
	if err != nil {
		panic(err)
	}
	err = copy("/usr/bin/container-disk", targetFile)
	if err != nil {
		panic(err)
	}

	app.reloadableRateLimiter = ratelimiter.NewReloadableRateLimiter(flowcontrol.NewTokenBucketRateLimiter(virtconfig.DefaultVirtHandlerQPS, virtconfig.DefaultVirtHandlerBurst))
	clientConfig, err := kubecli.GetKubevirtClientConfig()
	if err != nil {
		panic(err)
	}
	clientConfig.RateLimiter = app.reloadableRateLimiter
	app.virtCli, err = kubecli.GetKubevirtClientFromRESTConfig(clientConfig)
	if err != nil {
		panic(err)
	}

	app.markNodeAsUnschedulable(logger)

	app.namespace, err = clientutil.GetNamespace()
	if err != nil {
		glog.Fatalf("Error searching for namespace: %v", err)
	}

	go func() {
		sigint := make(chan os.Signal, 1)

		signal.Notify(sigint, syscall.SIGTERM)

		<-sigint

		app.markNodeAsUnschedulable(logger)
		os.Exit(0)
	}()

	// Create event recorder
	broadcaster := record.NewBroadcaster()
	broadcaster.StartRecordingToSink(&k8coresv1.EventSinkImpl{Interface: app.virtCli.CoreV1().Events(k8sv1.NamespaceAll)})
	// Scheme is used to create an ObjectReference from an Object (e.g. VirtualMachineInstance) during Event creation
	recorder := broadcaster.NewRecorder(scheme.Scheme, k8sv1.EventSource{Component: "virt-handler", Host: app.HostOverride})

	// Wire VirtualMachineInstance controller
	factory := controller.NewKubeInformerFactory(app.virtCli.RestClient(), app.virtCli, nil, app.namespace)

	vmiSourceInformer := factory.VMISourceHost(app.HostOverride)
	vmiTargetInformer := factory.VMITargetHost(app.HostOverride)

	// Wire Domain controller
	domainSharedInformer, err := virtcache.NewSharedInformer(app.VirtShareDir, int(app.WatchdogTimeoutDuration.Seconds()), recorder, vmiSourceInformer.GetStore(), time.Duration(app.domainResyncPeriodSeconds)*time.Second)
	if err != nil {
		panic(err)
	}

	// Legacy directory for watchdog files
	err = os.MkdirAll(watchdog.WatchdogFileDirectory(app.VirtShareDir), 0755)
	if err != nil {
		panic(err)
	}

	// Legacy Directory for graceful shutdown trigger files.
	err = os.MkdirAll(filepath.Join(app.VirtShareDir, "graceful-shutdown-trigger"), 0755)
	if err != nil {
		panic(err)
	}

	// We keep a record on disk of every VMI virt-handler starts.
	// That record isn't deleted from this node until the VMI
	// is completely torn down.
	err = virtcache.InitializeGhostRecordCache(filepath.Join(app.VirtPrivateDir, "ghost-records"))
	if err != nil {
		panic(err)
	}

	cmdclient.SetPodsBaseDir("/pods")
	cmdclient.SetLegacyBaseDir(app.VirtShareDir)
	containerdisk.SetKubeletPodsDirectory(app.KubeletPodsDir)
	err = os.MkdirAll(cmdclient.LegacySocketsDirectory(), 0755)
	if err != nil {
		panic(err)
	}

	if err := app.prepareCertManager(); err != nil {
		glog.Fatalf("Error preparing the certificate manager: %v", err)
	}

	// Legacy support, Remove this informer once we no longer support
	// VMIs with graceful shutdown trigger
	gracefulShutdownInformer := cache.NewSharedIndexInformer(
		inotifyinformer.NewFileListWatchFromClient(filepath.Join(app.VirtShareDir, "graceful-shutdown-trigger")),
		&virt_api.Domain{},
		0,
		cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})

	podIsolationDetector := isolation.NewSocketBasedIsolationDetector(app.VirtShareDir)
	app.clusterConfig, err = virtconfig.NewClusterConfig(factory.CRD(), factory.KubeVirt(), app.namespace)
	if err != nil {
		panic(err)
	}
	// set log verbosity
	app.clusterConfig.SetConfigModifiedCallback(app.shouldChangeLogVerbosity)
	app.clusterConfig.SetConfigModifiedCallback(app.shouldChangeRateLimiter)
	app.clusterConfig.SetConfigModifiedCallback(app.shouldInstallKubevirtSeccompProfile)

	if err := app.setupTLS(factory); err != nil {
		glog.Fatalf("Error constructing migration tls config: %v", err)
	}
	vsockMgr := vsock.NewVSOCKHypervisorService(1, app.caManager)

	vsockConfigCallback := func() {
		if app.clusterConfig.VSOCKEnabled() {
			vsockMgr.Start()
		} else {
			vsockMgr.Stop()
		}
	}

	app.clusterConfig.SetConfigModifiedCallback(vsockConfigCallback)

	migrationProxy := migrationproxy.NewMigrationProxyManager(app.serverTLSConfig, app.clientTLSConfig, app.clusterConfig)

	stop := make(chan struct{})
	defer close(stop)
	var capabilities *api.Capabilities
	var hostCpuModel string
	nodeLabellerrecorder := broadcaster.NewRecorder(scheme.Scheme, k8sv1.EventSource{Component: "node-labeller", Host: app.HostOverride})
	nodeLabellerController, err := nodelabeller.NewNodeLabeller(app.clusterConfig, app.virtCli, app.HostOverride, app.namespace, nodeLabellerrecorder)
	if err != nil {
		panic(err)
	}
	capabilities = nodeLabellerController.HostCapabilities()

	// Node labelling is only relevant on x86_64 arch.
	if virtconfig.IsAMD64(runtime.GOARCH) {
		hostCpuModel = nodeLabellerController.GetHostCpuModel().Name

		go nodeLabellerController.Run(10, stop)
	}

	migrationIpAddress := app.PodIpAddress
	migrationIpAddress, err = virthandler.FindMigrationIP(defaultNetworkStatusFilePath, migrationIpAddress)
	if err != nil {
		log.Log.Reason(err)
		return
	}

	downwardMetricsManager := dmetricsmanager.NewDownwardMetricsManager(app.HostOverride)

	vmController, err := virthandler.NewController(
		recorder,
		app.virtCli,
		app.HostOverride,
		migrationIpAddress,
		app.VirtShareDir,
		app.VirtPrivateDir,
		app.KubeletPodsDir,
		vmiSourceInformer,
		vmiTargetInformer,
		domainSharedInformer,
		gracefulShutdownInformer,
		int(app.WatchdogTimeoutDuration.Seconds()),
		app.MaxDevices,
		app.clusterConfig,
		podIsolationDetector,
		migrationProxy,
		downwardMetricsManager,
		capabilities,
		hostCpuModel,
	)
	if err != nil {
		panic(err)
	}

	promErrCh := make(chan error)
	go app.runPrometheusServer(promErrCh)

	lifecycleHandler := rest.NewLifecycleHandler(
		recorder,
		vmiSourceInformer,
		app.VirtShareDir,
	)

	promdomain.SetupDomainStatsCollector(app.virtCli, app.VirtShareDir, app.HostOverride, app.MaxRequestsInFlight, vmiSourceInformer)
	if err := downwardmetrics.RunDownwardMetricsCollector(context.Background(), app.HostOverride, vmiSourceInformer, podIsolationDetector); err != nil {
		panic(fmt.Errorf("failed to set up the downwardMetrics collector: %v", err))
	}

	go app.clientcertmanager.Start()
	go app.servercertmanager.Start()

	// Bootstrapping. From here on the startup order matters

	factory.Start(stop)
	go gracefulShutdownInformer.Run(stop)
	go domainSharedInformer.Run(stop)

	se, exists, err := selinux.NewSELinux()
	if err == nil && exists {
		// relabel tun device

		devTun, err := safepath.JoinAndResolveWithRelativeRoot("/", "/dev/net/tun")
		if err != nil {
			panic(err)
		}
		devNull, err := safepath.JoinAndResolveWithRelativeRoot("/", "/dev/null")
		if err != nil {
			panic(err)
		}
		err = selinux.RelabelFiles(util.UnprivilegedContainerSELinuxLabel, se.IsPermissive(), devTun, devNull)
		if err != nil {
			panic(fmt.Errorf("error relabeling required files: %v", err))
		}
	} else if err != nil {
		//an error occurred
		panic(fmt.Errorf("failed to detect the presence of selinux: %v", err))
	}

	cache.WaitForCacheSync(stop, vmiSourceInformer.HasSynced, factory.CRD().HasSynced, factory.KubeVirt().HasSynced)

	// This callback can only be called only after the KubeVirt CR has synced,
	// to avoid installing the SELinux policy when the feature gate is set
	app.clusterConfig.SetConfigModifiedCallback(app.shouldInstallSELinuxPolicy)

	go vmController.Run(10, stop)

	doneCh := make(chan string)
	defer close(doneCh)

	consoleHandler := rest.NewConsoleHandler(
		podIsolationDetector,
		vmiSourceInformer,
		app.clientcertmanager,
	)

	errCh := make(chan error)
	go app.runServer(errCh, consoleHandler, lifecycleHandler)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)
	// start graceful shutdown handler
	go func() {
		connectionInterval := 10 * time.Second
		connectionTimeout := time.Duration(app.gracefulShutdownSeconds) * time.Second

		s := <-c
		log.Log.Infof("Received signal %s, initiating graceful shutdown", s.String())

		// This triggers the migration proxy to no longer accept new connections
		migrationProxy.InitiateGracefulShutdown()

		err := utilwait.PollImmediate(connectionInterval, connectionTimeout, func() (done bool, err error) {
			count := migrationProxy.OpenListenerCount()
			if count > 0 {
				log.Log.Infof("waiting for %d migration listeners to terminate", count)
				return false, nil
			}
			return true, nil
		})

		if err != nil {
			errCh <- fmt.Errorf("Timed out waiting for migration listeners to terminate: %v", err)
		} else {
			doneCh <- "migration proxy cleanly shutdown"
		}
	}()

	// wait exit condition
	select {
	case err := <-errCh:
		log.Log.Reason(err).Errorf("exiting due to error")
		panic(err)
	case doneMsg := <-doneCh:
		log.Log.Infof("cleanly exiting with reason: %s", doneMsg)
	}
}

func (app *virtHandlerApp) installCustomSELinuxPolicy(se selinux.SELinux) {
	// Install KubeVirt's virt-launcher policy
	err := se.InstallPolicy("/var/run/kubevirt")
	if err != nil {
		panic(fmt.Errorf("failed to install virt-launcher selinux policy: %v", err))
	}
	app.customSELinuxPolicyInstalled = true
}

// Update virt-handler log verbosity on relevant config changes
func (app *virtHandlerApp) shouldChangeLogVerbosity() {
	verbosity := app.clusterConfig.GetVirtHandlerVerbosity(app.HostOverride)
	log.Log.SetVerbosityLevel(int(verbosity))
	log.Log.V(2).Infof("set verbosity to %d", verbosity)
}

// Update virt-handler rate limiter
func (app *virtHandlerApp) shouldChangeRateLimiter() {
	config := app.clusterConfig.GetConfig()
	qps := config.HandlerConfiguration.RestClient.RateLimiter.TokenBucketRateLimiter.QPS
	burst := config.HandlerConfiguration.RestClient.RateLimiter.TokenBucketRateLimiter.Burst
	app.reloadableRateLimiter.Set(flowcontrol.NewTokenBucketRateLimiter(qps, burst))
	log.Log.V(2).Infof("setting rate limiter to %v QPS and %v Burst", qps, burst)
}

// Install the SELinux policy when the feature gate that disables it gets removed
func (app *virtHandlerApp) shouldInstallSELinuxPolicy() {
	app.semoduleLock.Lock()
	defer app.semoduleLock.Unlock()
	if app.customSELinuxPolicyInstalled {
		return
	}
	if !app.clusterConfig.CustomSELinuxPolicyDisabled() {
		se, exists, err := selinux.NewSELinux()
		if err == nil && exists {
			app.installCustomSELinuxPolicy(se)
		}
	}
}

// Update virt-handler rate limiter
func (app *virtHandlerApp) shouldInstallKubevirtSeccompProfile() {
	enabled := app.clusterConfig.KubevirtSeccompProfileEnabled()
	if !enabled {
		log.DefaultLogger().Info("Kubevirt Seccomp profile is not enabled")
		return
	}

	installPath := filepath.Join("/proc/1/root", app.KubeletRoot)
	if err := seccomp.InstallPolicy(installPath); err != nil {
		log.DefaultLogger().Errorf("Failed to install Kubevirt Seccomp profile, %v", err)
		return
	}
	log.DefaultLogger().Infof("Kubevirt Seccomp profile was installed at %s", installPath)

}

func (app *virtHandlerApp) runPrometheusServer(errCh chan error) {
	mux := restful.NewContainer()
	webService := new(restful.WebService)
	webService.Path("/").Consumes(restful.MIME_JSON).Produces(restful.MIME_JSON)
	webService.Route(webService.GET("/healthz").To(healthz.KubeConnectionHealthzFuncFactory(app.clusterConfig, apiHealthVersion)).Doc("Health endpoint"))

	componentProfiler := profiler.NewProfileManager(app.clusterConfig)
	webService.Route(webService.GET("/start-profiler").To(componentProfiler.HandleStartProfiler).Doc("start profiler endpoint"))
	webService.Route(webService.GET("/stop-profiler").To(componentProfiler.HandleStopProfiler).Doc("stop profiler endpoint"))
	webService.Route(webService.GET("/dump-profiler").To(componentProfiler.HandleDumpProfiler).Doc("dump profiler results endpoint"))

	mux.Add(webService)
	log.Log.V(1).Infof("metrics: max concurrent requests=%d", app.MaxRequestsInFlight)
	mux.Handle("/metrics", promdomain.Handler(app.MaxRequestsInFlight))
	server := http.Server{
		Addr:      app.ServiceListen.Address(),
		Handler:   mux,
		TLSConfig: app.promTLSConfig,
	}
	errCh <- server.ListenAndServeTLS("", "")
}

func (app *virtHandlerApp) runServer(errCh chan error, consoleHandler *rest.ConsoleHandler, lifecycleHandler *rest.LifecycleHandler) {
	ws := new(restful.WebService)
	ws.Route(ws.GET("/v1/namespaces/{namespace}/virtualmachineinstances/{name}/console").To(consoleHandler.SerialHandler))
	ws.Route(ws.GET("/v1/namespaces/{namespace}/virtualmachineinstances/{name}/vnc").To(consoleHandler.VNCHandler))
	ws.Route(ws.GET("/v1/namespaces/{namespace}/virtualmachineinstances/{name}/usbredir").To(consoleHandler.USBRedirHandler))
	ws.Route(ws.PUT("/v1/namespaces/{namespace}/virtualmachineinstances/{name}/pause").To(lifecycleHandler.PauseHandler))
	ws.Route(ws.PUT("/v1/namespaces/{namespace}/virtualmachineinstances/{name}/unpause").To(lifecycleHandler.UnpauseHandler))
	ws.Route(ws.PUT("/v1/namespaces/{namespace}/virtualmachineinstances/{name}/freeze").To(lifecycleHandler.FreezeHandler).Reads(v1.FreezeUnfreezeTimeout{}))
	ws.Route(ws.PUT("/v1/namespaces/{namespace}/virtualmachineinstances/{name}/unfreeze").To(lifecycleHandler.UnfreezeHandler))
	ws.Route(ws.PUT("/v1/namespaces/{namespace}/virtualmachineinstances/{name}/softreboot").To(lifecycleHandler.SoftRebootHandler))
	ws.Route(ws.GET("/v1/namespaces/{namespace}/virtualmachineinstances/{name}/guestosinfo").To(lifecycleHandler.GetGuestInfo).Produces(restful.MIME_JSON).Consumes(restful.MIME_JSON).Returns(http.StatusOK, "OK", v1.VirtualMachineInstanceGuestAgentInfo{}))
	ws.Route(ws.GET("/v1/namespaces/{namespace}/virtualmachineinstances/{name}/userlist").To(lifecycleHandler.GetUsers).Produces(restful.MIME_JSON).Consumes(restful.MIME_JSON).Returns(http.StatusOK, "OK", v1.VirtualMachineInstanceGuestOSUserList{}))
	ws.Route(ws.GET("/v1/namespaces/{namespace}/virtualmachineinstances/{name}/filesystemlist").To(lifecycleHandler.GetFilesystems).Produces(restful.MIME_JSON).Consumes(restful.MIME_JSON).Returns(http.StatusOK, "OK", v1.VirtualMachineInstanceFileSystemList{}))
	ws.Route(ws.GET("/v1/namespaces/{namespace}/virtualmachineinstances/{name}/vsock").Param(restful.QueryParameter("port", "Target VSOCK port")).To(consoleHandler.VSOCKHandler))
	ws.Route(ws.GET("/v1/namespaces/{namespace}/virtualmachineinstances/{name}/sev/fetchcertchain").To(lifecycleHandler.SEVFetchCertChainHandler).Produces(restful.MIME_JSON).Consumes(restful.MIME_JSON).Returns(http.StatusOK, "OK", v1.SEVPlatformInfo{}))
	ws.Route(ws.GET("/v1/namespaces/{namespace}/virtualmachineinstances/{name}/sev/querylaunchmeasurement").To(lifecycleHandler.SEVQueryLaunchMeasurementHandler).Produces(restful.MIME_JSON).Consumes(restful.MIME_JSON).Returns(http.StatusOK, "OK", v1.SEVMeasurementInfo{}))
	ws.Route(ws.PUT("/v1/namespaces/{namespace}/virtualmachineinstances/{name}/sev/injectlaunchsecret").To(lifecycleHandler.SEVInjectLaunchSecretHandler))
	restful.DefaultContainer.Add(ws)
	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", app.ServiceListen.BindAddress, app.consoleServerPort),
		Handler: restful.DefaultContainer,
		// we use migration TLS also for console connections (initiated by virt-api)
		TLSConfig:   app.serverTLSConfig,
		IdleTimeout: 60 * time.Second,
	}
	errCh <- server.ListenAndServeTLS("", "")
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

	flag.StringVar(&app.VirtShareDir, "kubevirt-share-dir", util.VirtShareDir,
		"Shared directory between virt-handler and virt-launcher")

	flag.StringVar(&app.VirtPrivateDir, "kubevirt-private-dir", util.VirtPrivateDir,
		"private directory for virt-handler state")

	flag.StringVar(&app.VirtLibDir, "kubevirt-lib-dir", util.VirtLibDir,
		"Shared lib directory between virt-handler and virt-launcher")

	flag.StringVar(&app.KubeletPodsDir, "kubelet-pods-dir", util.KubeletPodsDir,
		"Path for pod directory (matching host's path for kubelet root)")

	flag.StringVar(&app.KubeletRoot, "kubelet-root", util.KubeletRoot,
		"Path for Kubelet root")

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

	flag.BoolVar(&app.externallyManaged, "externally-managed", false,
		"Allow intermediate certificates to be used in building up the chain of trust when certificates are externally managed")

	flag.DurationVar(&app.WatchdogTimeoutDuration, "watchdog-timeout", defaultWatchdogTimeout,
		"Watchdog file timeout")

	// TODO: the Device Plugin API does not allow for infinitely available (shared) devices
	// so the current approach is to register an arbitrary number.
	// This should be deprecated if the API allows for shared resources in the future
	flag.IntVar(&app.MaxDevices, "max-devices", maxDevices,
		"Number of devices to register with Kubernetes device plugin framework")

	flag.IntVar(&app.MaxRequestsInFlight, "max-metric-requests", maxRequestsInFlight,
		"Number of concurrent requests to the metrics endpoint")

	flag.IntVar(&app.consoleServerPort, "console-server-port", defaultConsoleServerPort,
		"The port virt-handler listens on for console requests")

	flag.IntVar(&app.domainResyncPeriodSeconds, "domain-resync-period-seconds", defaultDomainResyncPeriodSeconds,
		"Recurring period for resyncing all known virt-launcher domains.")

	flag.IntVar(&app.gracefulShutdownSeconds, "graceful-shutdown-seconds", defaultGracefulShutdownSeconds,
		"The number of seconds to wait for existing migration connections to close before shutting down virt-handler.")
}

func (app *virtHandlerApp) setupTLS(factory controller.KubeInformerFactory) error {
	kubevirtCAConfigInformer := factory.KubeVirtCAConfigMap()
	kubevirtCAConfigInformer.SetWatchErrorHandler(func(r *cache.Reflector, err error) {
		apiHealthVersion.Clear()
		cache.DefaultWatchErrorHandler(r, err)
	})
	app.caManager = kvtls.NewCAManager(kubevirtCAConfigInformer.GetStore(), app.namespace, app.caConfigMapName)

	app.promTLSConfig = kvtls.SetupPromTLS(app.servercertmanager, app.clusterConfig)
	app.serverTLSConfig = kvtls.SetupTLSForVirtHandlerServer(app.caManager, app.servercertmanager, app.externallyManaged, app.clusterConfig)
	app.clientTLSConfig = kvtls.SetupTLSForVirtHandlerClients(app.caManager, app.clientcertmanager, app.externallyManaged)

	return nil
}

func main() {
	app := &virtHandlerApp{}
	service.Setup(app)
	log.InitializeLogging("virt-handler")
	app.Run()
}

func copy(sourceFile string, targetFile string) error {

	if err := os.RemoveAll(targetFile); err != nil {
		return fmt.Errorf("failed to remove target file: %v", err)
	}
	target, err := os.Create(targetFile)
	if err != nil {
		return fmt.Errorf("failed to crate target file: %v", err)
	}
	defer target.Close()
	source, err := os.Open(sourceFile)
	if err != nil {
		return fmt.Errorf("failed to open source file: %v", err)
	}
	defer source.Close()
	_, err = io.Copy(target, source)
	if err != nil {
		return fmt.Errorf("failed to copy file: %v", err)
	}
	err = os.Chmod(targetFile, 0555)
	if err != nil {
		return fmt.Errorf("failed to make file executable: %v", err)
	}
	return nil
}
