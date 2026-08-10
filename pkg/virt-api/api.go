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
 * Copyright The KubeVirt Authors.
 *
 */

package virt_api

import (
	"context"
	"crypto/tls"
	goflag "flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	kvtls "kubevirt.io/kubevirt/pkg/util/tls"

	restful "github.com/emicklei/go-restful/v3"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	flag "github.com/spf13/pflag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	certificate2 "k8s.io/client-go/util/certificate"
	"k8s.io/client-go/util/flowcontrol"
	aggregatorclient "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"

	"kubevirt.io/kubevirt/pkg/util/ratelimiter"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	clientutil "kubevirt.io/client-go/util"

	"kubevirt.io/kubevirt/pkg/certificates/bootstrap"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/healthz"
	clientmetrics "kubevirt.io/kubevirt/pkg/monitoring/metrics/common/client"
	metrics "kubevirt.io/kubevirt/pkg/monitoring/metrics/virt-api"
	netadmitter "kubevirt.io/kubevirt/pkg/network/admitter"
	"kubevirt.io/kubevirt/pkg/service"
	storageadmitters "kubevirt.io/kubevirt/pkg/storage/admitters"

	"kubevirt.io/kubevirt/pkg/monitoring/profiler"
	"kubevirt.io/kubevirt/pkg/virt-api/apiserver"
	"kubevirt.io/kubevirt/pkg/virt-api/apiserver/cluster/clusterprofiler"
	"kubevirt.io/kubevirt/pkg/virt-api/apiserver/cluster/expand"
	"kubevirt.io/kubevirt/pkg/virt-api/apiserver/cluster/guestfs"
	versionhandler "kubevirt.io/kubevirt/pkg/virt-api/apiserver/cluster/version"
	"kubevirt.io/kubevirt/pkg/virt-api/apiserver/storage/virtualmachine"
	"kubevirt.io/kubevirt/pkg/virt-api/apiserver/storage/virtualmachineinstance"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
	mutating_webhook "kubevirt.io/kubevirt/pkg/virt-api/webhooks/mutating-webhook"
	validating_webhook "kubevirt.io/kubevirt/pkg/virt-api/webhooks/validating-webhook"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
)

const (
	// Default port that virt-api listens on.
	defaultPort = 443

	DefaultConsoleServerPort = 8186

	defaultCAConfigMapName     = "kubevirt-ca"
	defaultTlsCertFilePath     = "/etc/virt-api/certificates/tls.crt"
	defaultTlsKeyFilePath      = "/etc/virt-api/certificates/tls.key"
	defaultHandlerCertFilePath = "/etc/virt-handler/clientcertificates/tls.crt"
	defaultHandlerKeyFilePath  = "/etc/virt-handler/clientcertificates/tls.key"
)

type VirtApi interface {
	Run()
	AddFlags()
	Execute()
}

type virtAPIApp struct {
	apiServer *apiserver.APIServer

	// LEGACY(virt-api-migration): still passed by virt-operator but no longer read
	SubresourcesOnly bool
	virtClient       kubecli.KubevirtClient
	k8sClient        kubernetes.Interface
	aggregatorClient *aggregatorclient.Clientset
	clusterConfig    *virtconfig.ClusterConfig

	namespace               string
	host                    string
	consoleServerPort       int
	handlerTLSConfiguration *tls.Config
	handlerCertManager      certificate2.Manager
	handlerCertFilePath     string
	handlerKeyFilePath      string

	// Serving certificate handed to the GenericAPIServer.
	caConfigMapName   string
	externallyManaged bool

	reloadableRateLimiter        *ratelimiter.ReloadableRateLimiter
	reloadableWebhookRateLimiter *ratelimiter.ReloadableRateLimiter

	// indicates if controllers were started with or without CDI/DataSource support
	hasCDIDataSource bool
	// the channel used to trigger re-initialization.
	reInitChan chan string

	kubeVirtServiceAccounts map[string]struct{}
}

var _ service.Service = &virtAPIApp{}

var apiHealthVersion = new(healthz.KubeApiHealthzVersion)

func NewVirtApi() VirtApi {

	app := &virtAPIApp{}
	app.apiServer = apiserver.New().
		WithSecureServingPort(defaultPort).
		WithSecureServingCert(defaultTlsCertFilePath, defaultTlsKeyFilePath)

	return app
}

func (app *virtAPIApp) Execute() {
	if err := metrics.SetupMetrics(); err != nil {
		panic(err)
	}

	app.reloadableRateLimiter = ratelimiter.NewReloadableRateLimiter(flowcontrol.NewTokenBucketRateLimiter(virtconfig.DefaultVirtAPIQPS, virtconfig.DefaultVirtAPIBurst))
	app.reloadableWebhookRateLimiter = ratelimiter.NewReloadableRateLimiter(flowcontrol.NewTokenBucketRateLimiter(virtconfig.DefaultVirtWebhookClientQPS, virtconfig.DefaultVirtWebhookClientBurst))

	clientmetrics.RegisterRestConfigHooks()
	clientConfig, err := kubecli.GetKubevirtClientConfig()
	if err != nil {
		panic(err)
	}
	clientConfig.RateLimiter = app.reloadableRateLimiter
	app.virtClient, err = kubecli.GetKubevirtClientFromRESTConfig(clientConfig)
	if err != nil {
		panic(err)
	}
	app.k8sClient, err = kubecli.GetK8sClientFromRESTConfig(clientConfig)
	if err != nil {
		panic(err)
	}

	app.aggregatorClient = aggregatorclient.NewForConfigOrDie(clientConfig)

	app.namespace, err = clientutil.GetNamespace()
	if err != nil {
		panic(err)
	}

	app.kubeVirtServiceAccounts = webhooks.KubeVirtServiceAccounts(app.namespace)

	app.reInitChan = make(chan string, 10)

	app.Run()
}

func (app *virtAPIApp) prepareCertManager() {
	app.handlerCertManager = bootstrap.NewFileCertificateManager(app.handlerCertFilePath, app.handlerKeyFilePath)
}

// webhookMuxHandlers returns the admission webhooks served directly from the
// GenericAPIServer's NonGoRestfulMux
func (app *virtAPIApp) webhookMuxHandlers(informers *webhooks.Informers) []apiserver.MuxHandler {
	return []apiserver.MuxHandler{
		{
			Path: components.VMValidatePath,
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				validating_webhook.ServeVMs(w, r, app.clusterConfig, app.virtClient, informers, app.kubeVirtServiceAccounts)
			}),
		},
		{
			Path: components.VMMutatePath,
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mutating_webhook.ServeVMs(w, r, app.clusterConfig, app.virtClient)
			}),
		},
		{
			Path: components.VMIMutatePath,
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mutating_webhook.ServeVMIs(w, r, app.clusterConfig, informers, app.kubeVirtServiceAccounts)
			}),
		},
		{
			Path: components.MigrationMutatePath,
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mutating_webhook.ServeMigrationCreate(w, r)
			}),
		},
		{
			Path: components.VMCloneCreateMutatePath,
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mutating_webhook.ServeClones(w, r)
			}),
		},
		{
			Path: components.VirtLauncherPodMutatePath,
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mutating_webhook.ServeVirtLauncherPods(w, r, app.clusterConfig, app.virtClient)
			}),
		},
		{
			Path: components.VMICreateValidatePath,
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				validating_webhook.ServeVMICreate(w, r, app.clusterConfig, app.kubeVirtServiceAccounts,
					func(field *field.Path, vmiSpec *v1.VirtualMachineInstanceSpec, clusterCfg *virtconfig.ClusterConfig) []metav1.StatusCause {
						return netadmitter.Validate(field, vmiSpec, clusterCfg)
					},
					storageadmitters.Validate,
				)
			}),
		},
		{
			Path: components.VMIUpdateValidatePath,
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				validating_webhook.ServeVMIUpdate(w, r, app.clusterConfig, app.kubeVirtServiceAccounts)
			}),
		},
		{
			Path: components.VMIRSValidatePath,
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				validating_webhook.ServeVMIRS(w, r, app.clusterConfig)
			}),
		},
		{
			Path: components.VMPoolValidatePath,
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				validating_webhook.ServeVMPool(w, r, app.clusterConfig, app.kubeVirtServiceAccounts)
			}),
		},
		{
			Path: components.VMIPresetValidatePath,
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				validating_webhook.ServeVMIPreset(w, r)
			}),
		},
		{
			Path: components.MigrationCreateValidatePath,
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				validating_webhook.ServeMigrationCreate(w, r, app.clusterConfig, app.virtClient, app.kubeVirtServiceAccounts)
			}),
		},
		{
			Path: components.MigrationUpdateValidatePath,
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				validating_webhook.ServeMigrationUpdate(w, r)
			}),
		},
		{
			Path: components.VMSnapshotValidatePath,
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				validating_webhook.ServeVMSnapshots(w, r, app.clusterConfig, app.virtClient)
			}),
		},
		{
			Path: components.VMRestoreValidatePath,
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				validating_webhook.ServeVMRestores(w, r, app.clusterConfig, app.virtClient, informers)
			}),
		},
		{
			Path: components.VMBackupValidatePath,
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				validating_webhook.ServeVMBackups(w, r, app.clusterConfig, app.virtClient, informers)
			}),
		},
		{
			Path: components.VMBackupTrackerValidatePath,
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				validating_webhook.ServeVMBackupTrackers(w, r, app.clusterConfig)
			}),
		},
		{
			Path: components.VMExportValidatePath,
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				validating_webhook.ServeVMExports(w, r, app.clusterConfig)
			}),
		},
		{
			Path: components.VMInstancetypeValidatePath,
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				validating_webhook.ServeVmInstancetypes(w, r)
			}),
		},
		{
			Path: components.VMClusterInstancetypeValidatePath,
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				validating_webhook.ServeVmClusterInstancetypes(w, r)
			}),
		},
		{
			Path: components.VMPreferenceValidatePath,
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				validating_webhook.ServeVmPreferences(w, r)
			}),
		},
		{
			Path: components.VMClusterPreferenceValidatePath,
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				validating_webhook.ServeVmClusterPreferences(w, r)
			}),
		},
		{
			Path: components.StatusValidatePath,
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				validating_webhook.ServeStatusValidation(w, r, app.clusterConfig, app.virtClient, informers, app.kubeVirtServiceAccounts)
			}),
		},
		{
			Path: components.PodEvictionValidatePath,
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				validating_webhook.ServePodEvictionInterceptor(w, r, app.clusterConfig, app.virtClient, app.k8sClient)
			}),
		},
		{
			Path: components.MigrationPolicyCreateValidatePath,
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				validating_webhook.ServeMigrationPolicies(w, r, app.clusterConfig)
			}),
		},
		{
			Path: components.VMCloneCreateValidatePath,
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				validating_webhook.ServeVirtualMachineClones(w, r, app.clusterConfig, app.virtClient)
			}),
		},
		{
			Path: components.PluginValidatePath,
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				validating_webhook.ServePlugins(w, r, app.clusterConfig)
			}),
		},
	}
}

func (app *virtAPIApp) Run() {
	host, err := os.Hostname()
	if err != nil {
		panic(fmt.Errorf("unable to get hostname: %v", err))
	}
	app.host = host

	// Get/Set selfsigned cert
	app.prepareCertManager()

	// Run informers for webhooks usage
	kubeInformerFactory := controller.NewKubeInformerFactory(app.virtClient.RestClient(), app.virtClient, app.k8sClient, app.aggregatorClient, app.namespace)

	kubeVirtInformer := kubeInformerFactory.KubeVirt()
	// A broken watch means the cached config may be stale, so the healthz payload
	// has to re-resolve the Kubernetes API version on the next probe.
	if err := kubeVirtInformer.SetWatchErrorHandler(func(r *cache.Reflector, err error) {
		apiHealthVersion.Clear()
		cache.DefaultWatchErrorHandler(context.TODO(), r, err)
	}); err != nil {
		panic(err)
	}

	kubeInformerFactory.KubeVirtCAConfigMap()
	crdInformer := kubeInformerFactory.CRD()
	vmiPresetInformer := kubeInformerFactory.VirtualMachinePreset()
	vmRestoreInformer := kubeInformerFactory.VirtualMachineRestore()
	vmBackupInformer := kubeInformerFactory.VirtualMachineBackup()
	namespaceInformer := kubeInformerFactory.Namespace()

	stopChan := make(chan struct{}, 1)
	defer close(stopChan)
	kubeInformerFactory.Start(stopChan)
	kubeInformerFactory.WaitForCacheSync(stopChan)

	app.clusterConfig, err = virtconfig.NewClusterConfig(crdInformer, kubeVirtInformer, app.namespace)
	if err != nil {
		panic(err)
	}
	app.hasCDIDataSource = app.clusterConfig.HasDataSourceAPI()
	app.clusterConfig.SetConfigModifiedCallback(app.configModificationCallback)
	app.clusterConfig.SetConfigModifiedCallback(app.shouldChangeLogVerbosity)
	app.clusterConfig.SetConfigModifiedCallback(app.shouldChangeRateLimiter)

	var dataSourceInformer cache.SharedIndexInformer
	if app.hasCDIDataSource {
		dataSourceInformer = kubeInformerFactory.DataSource()
		log.Log.Infof("CDI detected, DataSource integration enabled")
	} else {
		// Add a dummy DataSource informer in the event datasource support
		// is disabled. This lets the controller continue to work without
		// requiring a separate branching code path.
		dataSourceInformer = kubeInformerFactory.DummyDataSource()
		log.Log.Infof("CDI not detected, DataSource integration disabled")
	}

	// It is safe to call kubeInformerFactory.Start multiple times.
	// The function is idempotent and will only start the informers that
	// have not been started yet
	kubeInformerFactory.Start(stopChan)
	kubeInformerFactory.WaitForCacheSync(stopChan)

	webhookInformers := &webhooks.Informers{
		VMIPresetInformer:  vmiPresetInformer,
		VMRestoreInformer:  vmRestoreInformer,
		VMBackupInformer:   vmBackupInformer,
		DataSourceInformer: dataSourceInformer,
		NamespaceInformer:  namespaceInformer,
	}

	go app.handlerCertManager.Start()

	app.setupHandlerTLS(kubeInformerFactory)
	metrics.SetVirtAPIReady()

	ctx := app.signalAwareContext()
	if err := app.startAggregatedAPIServer(ctx, webhookInformers); err != nil && err != http.ErrServerClosed {
		panic(err)
	}
}

// This is where the dial virt-handler is done for console, vnc, etc
func (app *virtAPIApp) setupHandlerTLS(informerFactory controller.KubeInformerFactory) {
	kubevirtCAConfigInformer := informerFactory.KubeVirtCAConfigMap()
	kubevirtCAManager := kvtls.NewCAManager(kubevirtCAConfigInformer.GetStore(), app.namespace, app.caConfigMapName)
	app.handlerTLSConfiguration = kvtls.SetupTLSForVirtHandlerClients(kubevirtCAManager, app.handlerCertManager, app.externallyManaged)
}

func (app *virtAPIApp) signalAwareContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)

	go func() {
		select {
		case s := <-c:
			log.Log.Infof("Received signal %s, initiating graceful shutdown", s.String())
		case msg := <-app.reInitChan:
			log.Log.Infof("Received signal to reInitialize virt-api [%s], initiating graceful shutdown", msg)
		}
		metrics.SetVirtAPINotReady()
		cancel()
	}()

	return ctx
}

func (app *virtAPIApp) startAggregatedAPIServer(ctx context.Context, webhookInformers *webhooks.Informers) error {
	s := app.apiServer.
		WithLongRunningSubresources("console", "vnc", "usbredir", "vsock", "portforward").
		// expand-vm-spec, version, guestfs and the cluster-profiler endpoints are
		// GET/PUT to the collection path without a resource name, which cannot be
		// expressed as a rest.Storage. So serve them as plain http.Handlers
		// (like the webhooks) for every subresource version.
		WithAPIHandlers(app.clusterLevelAPIHandlers()...).
		WithAlwaysAllowPaths(apiserver.ClusterLevelAllowPaths(v1.SubresourceGroupVersions)...).
		WithAlwaysAllowPaths(apiserver.ComponentProfilerPaths()...).
		WithMuxHandlers(app.healthAndMetricsMuxHandlers()...).
		WithMuxHandlers(app.webhookMuxHandlers(webhookInformers)...).
		WithMuxHandlers(app.componentProfilerMuxHandlers()...)

	scheme := apiserver.NewScheme()

	// Register each version gets its own storage instances.
	apiGroups := apiserver.APIGroups{}
	for _, gv := range v1.SubresourceGroupVersions {
		storage := virtualmachine.NewStorageMap(app.virtClient, app.k8sClient, app.consoleServerPort, app.handlerTLSConfiguration, app.clusterConfig)
		for resource, store := range virtualmachineinstance.NewStorageMap(app.virtClient, app.k8sClient, app.consoleServerPort, app.handlerTLSConfiguration, app.clusterConfig) {
			storage[resource] = store
		}
		apiGroups[gv] = storage
	}

	log.Log.Info("starting aggregated API server (GenericAPIServer) on port %d as the single virt-api listener")

	return s.Run(
		ctx,
		"virt-api-aggregated",
		scheme,
		apiserver.NewOpenAPIConfig(scheme),
		apiserver.NewOpenAPIV3Config(scheme),
		apiGroups,
	)
}

func (app *virtAPIApp) healthAndMetricsMuxHandlers() []apiserver.MuxHandler {
	healthzHandler := healthz.KubeConnectionHealthzFuncFactory(app.clusterConfig, apiHealthVersion)
	return []apiserver.MuxHandler{
		{Path: "/metrics", Handler: promhttp.Handler()},
		{
			Path: "/healthz",
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				healthzHandler(restful.NewRequest(r), restful.NewResponse(w))
			}),
		},
	}
}

// clusterLevelAPIHandlers returns the ConditionalAPIHandlers for the cluster
// level subresources of the subresources.kubevirt.io group that are GET/PUT to
// the collection path without a resource name (expand-vm-spec, version, guestfs
// and the cluster-profiler endpoints)
func (app *virtAPIApp) clusterLevelAPIHandlers() []apiserver.ConditionalAPIHandler {
	clusterProfilerHandler := clusterprofiler.NewHandler(app.virtClient, app.k8sClient, app.clusterConfig)

	matchesResource := func(resource string) func(*request.RequestInfo) bool {
		return func(info *request.RequestInfo) bool {
			return info.IsResourceRequest &&
				info.APIGroup == v1.SubresourceGroupName &&
				info.Resource == resource
		}
	}

	return []apiserver.ConditionalAPIHandler{
		{
			Matches: matchesResource("expand-vm-spec"),
			Handler: expand.NewHandler(app.clusterConfig, app.virtClient),
		},
		{
			Matches: matchesResource("version"),
			Handler: versionhandler.NewHandler(),
		},
		{
			Matches: matchesResource("guestfs"),
			Handler: guestfs.NewHandler(app.clusterConfig),
		},
		{
			Matches: matchesResource("healthz"),
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				healthz.KubeConnectionHealthzFuncFactory(app.clusterConfig, apiHealthVersion)(restful.NewRequest(r), restful.NewResponse(w))
			}),
		},
		{
			Matches: matchesResource("start-cluster-profiler"),
			Handler: http.HandlerFunc(clusterProfilerHandler.StartClusterProfilerHTTP),
		},
		{
			Matches: matchesResource("stop-cluster-profiler"),
			Handler: http.HandlerFunc(clusterProfilerHandler.StopClusterProfilerHTTP),
		},
		{
			Matches: matchesResource("dump-cluster-profiler"),
			Handler: http.HandlerFunc(clusterProfilerHandler.DumpClusterProfilerHTTP),
		},
	}
}

// componentProfilerMuxHandlers returns the component level profiler endpoints
// served directly from the GenericAPIServer's NonGoRestfulMux. These are the
// per-process endpoints the cluster-profiler fans out to
func (app *virtAPIApp) componentProfilerMuxHandlers() []apiserver.MuxHandler {
	componentProfiler := profiler.NewProfileManager(app.clusterConfig)
	adapt := func(h func(*restful.Request, *restful.Response)) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h(restful.NewRequest(r), restful.NewResponse(w))
		})
	}
	return []apiserver.MuxHandler{
		{Path: "/start-profiler", Handler: adapt(componentProfiler.HandleStartProfiler)},
		{Path: "/stop-profiler", Handler: adapt(componentProfiler.HandleStopProfiler)},
		{Path: "/dump-profiler", Handler: adapt(componentProfiler.HandleDumpProfiler)},
	}
}

// Detects if a config has been applied that requires
// re-initializing virt-api.
func (app *virtAPIApp) configModificationCallback() {
	newHasCDI := app.clusterConfig.HasDataSourceAPI()
	if newHasCDI != app.hasCDIDataSource {
		if newHasCDI {
			log.Log.Infof("Reinitialize virt-api, cdi DataSource api has been introduced")
		} else {
			log.Log.Infof("Reinitialize virt-api, cdi DataSource api has been removed")
		}
		app.reInitChan <- "reinit due to CDI api change"
	}
}

// Update virt-api log verbosity on relevant config changes
func (app *virtAPIApp) shouldChangeLogVerbosity() {
	verbosity := app.clusterConfig.GetVirtAPIVerbosity(app.host)
	log.Log.SetVerbosityLevel(int(verbosity))
	log.Log.V(2).Infof("set log verbosity to %d", verbosity)
}

// Update virt-handler rate limiter
func (app *virtAPIApp) shouldChangeRateLimiter() {
	config := app.clusterConfig.GetConfig()
	qps := config.APIConfiguration.RestClient.RateLimiter.TokenBucketRateLimiter.QPS
	burst := config.APIConfiguration.RestClient.RateLimiter.TokenBucketRateLimiter.Burst
	app.reloadableRateLimiter.Set(flowcontrol.NewTokenBucketRateLimiter(qps, burst))
	log.Log.V(2).Infof("setting rate limiter for the API to %v QPS and %v Burst", qps, burst)
	qps = config.WebhookConfiguration.RestClient.RateLimiter.TokenBucketRateLimiter.QPS
	burst = config.WebhookConfiguration.RestClient.RateLimiter.TokenBucketRateLimiter.Burst
	app.reloadableWebhookRateLimiter.Set(flowcontrol.NewTokenBucketRateLimiter(qps, burst))
	log.Log.V(2).Infof("setting rate limiter for webhooks to %v QPS and %v Burst", qps, burst)
}

func (app *virtAPIApp) AddFlags() {
	flag.CommandLine.AddGoFlag(log.VerbosityFlag())
	flag.CommandLine.AddGoFlag(goflag.CommandLine.Lookup("kubeconfig"))
	flag.CommandLine.AddGoFlag(goflag.CommandLine.Lookup("master"))

	app.apiServer.AddFlags(flag.CommandLine)

	flag.BoolVar(&app.SubresourcesOnly, "subresources-only", false,
		"Only serve subresource endpoints")
	flag.IntVar(&app.consoleServerPort, "console-server-port", DefaultConsoleServerPort,
		"The port virt-handler listens on for console requests")
	flag.StringVar(&app.caConfigMapName, "ca-configmap-name", defaultCAConfigMapName,
		"The name of configmap containing CA certificates to authenticate requests presenting client certificates with matching CommonName")
	flag.StringVar(&app.handlerCertFilePath, "handler-cert-file", defaultHandlerCertFilePath,
		"Client certificate used to prove the identity of the virt-api when it must call virt-handler during a request")
	flag.StringVar(&app.handlerKeyFilePath, "handler-key-file", defaultHandlerKeyFilePath,
		"Private key for the client certificate used to prove the identity of the virt-api when it must call virt-handler during a request")
	flag.BoolVar(&app.externallyManaged, "externally-managed", false,
		"Allow intermediate certificates to be used in building up the chain of trust when certificates are externally managed")
}
