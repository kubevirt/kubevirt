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
 * Copyright 2017, 2018 Red Hat, Inc.
 *
 */

package watch

import (
	"context"
	"crypto/tls"
	golog "log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/hooks"

	containerdisk "kubevirt.io/kubevirt/pkg/container-disk"
	kvtls "kubevirt.io/kubevirt/pkg/util/tls"

	"kubevirt.io/kubevirt/pkg/monitoring/migration"
	"kubevirt.io/kubevirt/pkg/monitoring/migrationstats"

	clonev1alpha1 "kubevirt.io/api/clone/v1alpha1"

	"kubevirt.io/kubevirt/pkg/virt-controller/watch/clone"

	"kubevirt.io/kubevirt/pkg/instancetype"

	"github.com/emicklei/go-restful/v3"
	vsv1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	flag "github.com/spf13/pflag"
	k8sv1 "k8s.io/api/core/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	k8coresv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	clientrest "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/flowcontrol"

	"kubevirt.io/kubevirt/pkg/util/ratelimiter"

	"kubevirt.io/kubevirt/pkg/virt-controller/watch/topology"

	"kubevirt.io/kubevirt/pkg/healthz"
	"kubevirt.io/kubevirt/pkg/monitoring/profiler"

	exportv1 "kubevirt.io/api/export/v1alpha1"
	poolv1 "kubevirt.io/api/pool/v1alpha1"
	snapshotv1 "kubevirt.io/api/snapshot/v1alpha1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	clientutil "kubevirt.io/client-go/util"

	"kubevirt.io/kubevirt/pkg/certificates/bootstrap"
	"kubevirt.io/kubevirt/pkg/controller"
	clusterutil "kubevirt.io/kubevirt/pkg/util/cluster"

	"kubevirt.io/kubevirt/pkg/monitoring/perfscale"
	"kubevirt.io/kubevirt/pkg/monitoring/virt-controller/metrics"
	vmiprom "kubevirt.io/kubevirt/pkg/monitoring/vmistats" // import for prometheus metrics
	vmprom "kubevirt.io/kubevirt/pkg/monitoring/vmstats"
	"kubevirt.io/kubevirt/pkg/service"
	"kubevirt.io/kubevirt/pkg/storage/export/export"
	"kubevirt.io/kubevirt/pkg/storage/snapshot"
	"kubevirt.io/kubevirt/pkg/util"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-controller/leaderelectionconfig"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
	"kubevirt.io/kubevirt/pkg/virt-controller/watch/drain/disruptionbudget"
	"kubevirt.io/kubevirt/pkg/virt-controller/watch/drain/evacuation"
	workloadupdater "kubevirt.io/kubevirt/pkg/virt-controller/watch/workload-updater"

	"kubevirt.io/kubevirt/pkg/network/netbinding"
)

const (
	defaultPort = 8182

	defaultHost = "0.0.0.0"

	launcherImage       = "virt-launcher"
	exporterImage       = "virt-exportserver"
	launcherQemuTimeout = 240

	imagePullSecret = ""

	virtShareDir = "/var/run/kubevirt"

	ephemeralDiskDir = virtShareDir + "-ephemeral-disks"

	defaultControllerThreads         = 3
	defaultSnapshotControllerThreads = 6
	defaultVMIControllerThreads      = 10

	defaultLauncherSubGid                 = 107
	defaultSnapshotControllerResyncPeriod = 5 * time.Minute
	defaultNodeTopologyUpdatePeriod       = 30 * time.Second

	defaultPromCertFilePath = "/etc/virt-controller/certificates/tls.crt"
	defaultPromKeyFilePath  = "/etc/virt-controller/certificates/tls.key"
)

var (
	containerDiskDir = filepath.Join(util.VirtShareDir, "container-disks")
	hotplugDiskDir   = filepath.Join(util.VirtShareDir, "hotplug-disks")

	apiHealthVersion = new(healthz.KubeApiHealthzVersion)
)

type VirtControllerApp struct {
	service.ServiceListen

	clientSet       kubecli.KubevirtClient
	templateService services.TemplateService
	restClient      *clientrest.RESTClient
	informerFactory controller.KubeInformerFactory
	kvPodInformer   cache.SharedIndexInformer

	nodeInformer   cache.SharedIndexInformer
	nodeController *NodeController

	vmiCache      cache.Store
	vmiController *VMIController
	vmiInformer   cache.SharedIndexInformer
	vmiRecorder   record.EventRecorder

	namespaceInformer cache.SharedIndexInformer
	namespaceStore    cache.Store

	kubeVirtInformer cache.SharedIndexInformer

	clusterConfig *virtconfig.ClusterConfig

	pdbInformer cache.SharedIndexInformer

	persistentVolumeClaimCache    cache.Store
	persistentVolumeClaimInformer cache.SharedIndexInformer

	rsController *VMIReplicaSet
	rsInformer   cache.SharedIndexInformer

	poolController *PoolController
	poolInformer   cache.SharedIndexInformer

	vmController *VMController
	vmInformer   cache.SharedIndexInformer

	controllerRevisionInformer cache.SharedIndexInformer

	dataVolumeInformer cache.SharedIndexInformer
	dataSourceInformer cache.SharedIndexInformer
	cdiInformer        cache.SharedIndexInformer
	cdiConfigInformer  cache.SharedIndexInformer

	migrationController *MigrationController
	migrationInformer   cache.SharedIndexInformer

	workloadUpdateController *workloadupdater.WorkloadUpdateController

	caExportConfigMapInformer    cache.SharedIndexInformer
	exportRouteConfigMapInformer cache.SharedInformer
	exportServiceInformer        cache.SharedIndexInformer
	exportController             *export.VMExportController
	snapshotController           *snapshot.VMSnapshotController
	restoreController            *snapshot.VMRestoreController
	vmExportInformer             cache.SharedIndexInformer
	routeCache                   cache.Store
	ingressCache                 cache.Store
	unmanagedSecretInformer      cache.SharedIndexInformer
	vmSnapshotInformer           cache.SharedIndexInformer
	vmSnapshotContentInformer    cache.SharedIndexInformer
	vmRestoreInformer            cache.SharedIndexInformer
	storageClassInformer         cache.SharedIndexInformer
	allPodInformer               cache.SharedIndexInformer
	resourceQuotaInformer        cache.SharedIndexInformer

	crdInformer cache.SharedIndexInformer

	migrationPolicyInformer cache.SharedIndexInformer

	vmCloneInformer   cache.SharedIndexInformer
	vmCloneController *clone.VMCloneController

	instancetypeInformer        cache.SharedIndexInformer
	clusterInstancetypeInformer cache.SharedIndexInformer
	preferenceInformer          cache.SharedIndexInformer
	clusterPreferenceInformer   cache.SharedIndexInformer

	LeaderElection leaderelectionconfig.Configuration

	launcherImage              string
	exporterImage              string
	launcherQemuTimeout        int
	imagePullSecret            string
	virtShareDir               string
	virtLibDir                 string
	ephemeralDiskDir           string
	containerDiskDir           string
	hotplugDiskDir             string
	readyChan                  chan bool
	kubevirtNamespace          string
	host                       string
	evacuationController       *evacuation.EvacuationController
	disruptionBudgetController *disruptionbudget.DisruptionBudgetController

	ctx context.Context

	// indicates if controllers were started with or without CDI/DataVolume support
	hasCDI bool
	// the channel used to trigger re-initialization.
	reInitChan chan string

	// number of threads for each controller
	nodeControllerThreads             int
	vmiControllerThreads              int
	rsControllerThreads               int
	poolControllerThreads             int
	vmControllerThreads               int
	migrationControllerThreads        int
	evacuationControllerThreads       int
	disruptionBudgetControllerThreads int
	launcherSubGid                    int64
	exportControllerThreads           int
	snapshotControllerThreads         int
	restoreControllerThreads          int
	snapshotControllerResyncPeriod    time.Duration
	cloneControllerThreads            int

	caConfigMapName          string
	promCertFilePath         string
	promKeyFilePath          string
	nodeTopologyUpdater      topology.NodeTopologyUpdater
	nodeTopologyUpdatePeriod time.Duration
	reloadableRateLimiter    *ratelimiter.ReloadableRateLimiter
	leaderElector            *leaderelection.LeaderElector

	onOpenshift bool
}

var _ service.Service = &VirtControllerApp{}

func init() {
	utilruntime.Must(vsv1.AddToScheme(scheme.Scheme))
	utilruntime.Must(snapshotv1.AddToScheme(scheme.Scheme))
	utilruntime.Must(exportv1.AddToScheme(scheme.Scheme))
	utilruntime.Must(poolv1.AddToScheme(scheme.Scheme))
	utilruntime.Must(clonev1alpha1.AddToScheme(scheme.Scheme))

	metrics.SetupMetrics()
}

func Execute() {
	var err error
	var app = VirtControllerApp{}

	app.LeaderElection = leaderelectionconfig.DefaultLeaderElectionConfiguration()

	service.Setup(&app)

	app.readyChan = make(chan bool, 1)

	log.InitializeLogging("virt-controller")

	app.reloadableRateLimiter = ratelimiter.NewReloadableRateLimiter(flowcontrol.NewTokenBucketRateLimiter(virtconfig.DefaultVirtControllerQPS, virtconfig.DefaultVirtControllerBurst))
	clientConfig, err := kubecli.GetKubevirtClientConfig()
	if err != nil {
		panic(err)
	}
	clientConfig.RateLimiter = app.reloadableRateLimiter
	app.clientSet, err = kubecli.GetKubevirtClientFromRESTConfig(clientConfig)
	if err != nil {
		golog.Fatal(err)
	}

	app.restClient = app.clientSet.RestClient()

	// Bootstrapping. From here on the initialization order is important
	app.kubevirtNamespace, err = clientutil.GetNamespace()
	if err != nil {
		golog.Fatalf("Error searching for namespace: %v", err)
	}

	host, err := os.Hostname()
	if err != nil {
		golog.Fatalf("unable to get hostname: %v", err)
	}
	app.host = host

	ctx, cancel := context.WithCancel(context.Background())
	stopChan := ctx.Done()
	app.ctx = ctx

	app.informerFactory = controller.NewKubeInformerFactory(app.restClient, app.clientSet, nil, app.kubevirtNamespace)

	app.crdInformer = app.informerFactory.CRD()
	app.kubeVirtInformer = app.informerFactory.KubeVirt()

	if err := app.kubeVirtInformer.SetWatchErrorHandler(func(r *cache.Reflector, err error) {
		apiHealthVersion.Clear()
		cache.DefaultWatchErrorHandler(r, err)
	}); err != nil {
		golog.Fatalf("failed to set the watch error handler: %v", err)
	}
	app.informerFactory.Start(stopChan)

	cache.WaitForCacheSync(stopChan, app.crdInformer.HasSynced, app.kubeVirtInformer.HasSynced)
	app.clusterConfig, err = virtconfig.NewClusterConfig(app.crdInformer, app.kubeVirtInformer, app.kubevirtNamespace)
	if err != nil {
		panic(err)
	}

	app.reInitChan = make(chan string, 10)
	app.hasCDI = app.clusterConfig.HasDataVolumeAPI()
	app.clusterConfig.SetConfigModifiedCallback(app.configModificationCallback)
	app.clusterConfig.SetConfigModifiedCallback(app.shouldChangeLogVerbosity)
	app.clusterConfig.SetConfigModifiedCallback(app.shouldChangeRateLimiter)

	webService := new(restful.WebService)
	webService.Path("/").Consumes(restful.MIME_JSON).Produces(restful.MIME_JSON)
	webService.Route(webService.GET("/healthz").To(healthz.KubeConnectionHealthzFuncFactory(app.clusterConfig, apiHealthVersion)).Doc("Health endpoint"))
	webService.Route(webService.GET("/leader").To(app.leaderProbe).Doc("Leader endpoint"))

	componentProfiler := profiler.NewProfileManager(app.clusterConfig)
	webService.Route(webService.GET("/start-profiler").To(componentProfiler.HandleStartProfiler).Doc("start profiler endpoint"))
	webService.Route(webService.GET("/stop-profiler").To(componentProfiler.HandleStopProfiler).Doc("stop profiler endpoint"))
	webService.Route(webService.GET("/dump-profiler").To(componentProfiler.HandleDumpProfiler).Doc("dump profiler results endpoint"))

	restful.Add(webService)

	app.vmiInformer = app.informerFactory.VMI()
	app.kvPodInformer = app.informerFactory.KubeVirtPod()
	app.nodeInformer = app.informerFactory.KubeVirtNode()
	app.namespaceStore = app.informerFactory.Namespace().GetStore()
	app.namespaceInformer = app.informerFactory.Namespace()
	app.vmiCache = app.vmiInformer.GetStore()
	app.vmiRecorder = app.newRecorder(k8sv1.NamespaceAll, "virtualmachine-controller")

	app.rsInformer = app.informerFactory.VMIReplicaSet()
	app.poolInformer = app.informerFactory.VMPool()

	app.persistentVolumeClaimInformer = app.informerFactory.PersistentVolumeClaim()
	app.persistentVolumeClaimCache = app.persistentVolumeClaimInformer.GetStore()

	app.pdbInformer = app.informerFactory.K8SInformerFactory().Policy().V1().PodDisruptionBudgets().Informer()

	app.vmInformer = app.informerFactory.VirtualMachine()

	app.migrationInformer = app.informerFactory.VirtualMachineInstanceMigration()

	app.controllerRevisionInformer = app.informerFactory.ControllerRevision()

	app.vmExportInformer = app.informerFactory.VirtualMachineExport()
	app.vmSnapshotInformer = app.informerFactory.VirtualMachineSnapshot()
	app.vmSnapshotContentInformer = app.informerFactory.VirtualMachineSnapshotContent()
	app.vmRestoreInformer = app.informerFactory.VirtualMachineRestore()
	app.storageClassInformer = app.informerFactory.StorageClass()
	app.caExportConfigMapInformer = app.informerFactory.KubeVirtExportCAConfigMap()
	app.exportRouteConfigMapInformer = app.informerFactory.ExportRouteConfigMap()
	app.unmanagedSecretInformer = app.informerFactory.UnmanagedSecrets()
	app.allPodInformer = app.informerFactory.Pod()
	app.exportServiceInformer = app.informerFactory.ExportService()
	app.resourceQuotaInformer = app.informerFactory.ResourceQuota()

	if app.hasCDI {
		app.dataVolumeInformer = app.informerFactory.DataVolume()
		app.cdiInformer = app.informerFactory.CDI()
		app.cdiConfigInformer = app.informerFactory.CDIConfig()
		app.dataSourceInformer = app.informerFactory.DataSource()
		log.Log.Infof("CDI detected, DataVolume integration enabled")
	} else {
		// Add a dummy DataVolume informer in the event datavolume support
		// is disabled. This lets the controller continue to work without
		// requiring a separate branching code path.
		app.dataVolumeInformer = app.informerFactory.DummyDataVolume()
		app.cdiInformer = app.informerFactory.DummyCDI()
		app.cdiConfigInformer = app.informerFactory.DummyCDIConfig()
		app.dataSourceInformer = app.informerFactory.DummyDataSource()
		log.Log.Infof("CDI not detected, DataVolume integration disabled")
	}

	onOpenShift, err := clusterutil.IsOnOpenShift(app.clientSet)
	if err != nil {
		golog.Fatalf("Error determining cluster type: %v", err)
	}
	if onOpenShift {
		log.Log.Info("we are on openshift")
		app.routeCache = app.informerFactory.OperatorRoute().GetStore()
	} else {
		log.Log.Info("we are on kubernetes")
		app.routeCache = app.informerFactory.DummyOperatorRoute().GetStore()
	}
	app.ingressCache = app.informerFactory.Ingress().GetStore()
	app.migrationPolicyInformer = app.informerFactory.MigrationPolicy()

	app.vmCloneInformer = app.informerFactory.VirtualMachineClone()

	app.instancetypeInformer = app.informerFactory.VirtualMachineInstancetype()
	app.clusterInstancetypeInformer = app.informerFactory.VirtualMachineClusterInstancetype()
	app.preferenceInformer = app.informerFactory.VirtualMachinePreference()
	app.clusterPreferenceInformer = app.informerFactory.VirtualMachineClusterPreference()

	app.onOpenshift = onOpenShift

	app.initCommon()
	app.initReplicaSet()
	app.initPool()
	app.initVirtualMachines()
	app.initDisruptionBudgetController()
	app.initEvacuationController()
	app.initSnapshotController()
	app.initRestoreController()
	app.initExportController()
	app.initWorkloadUpdaterController()
	app.initCloneController()
	go app.Run()

	<-app.reInitChan
	cancel()
}

// Detects if a config has been applied that requires
// re-initializing virt-controller.
func (vca *VirtControllerApp) configModificationCallback() {
	newHasCDI := vca.clusterConfig.HasDataVolumeAPI()
	if newHasCDI != vca.hasCDI {
		if newHasCDI {
			log.Log.Infof("Reinitialize virt-controller, cdi api has been introduced")
		} else {
			log.Log.Infof("Reinitialize virt-controller, cdi api has been removed")
		}
		vca.reInitChan <- "reinit"
	}
}

// Update virt-controller rate limiter
func (vca *VirtControllerApp) shouldChangeRateLimiter() {
	config := vca.clusterConfig.GetConfig()
	qps := config.ControllerConfiguration.RestClient.RateLimiter.TokenBucketRateLimiter.QPS
	burst := config.ControllerConfiguration.RestClient.RateLimiter.TokenBucketRateLimiter.Burst
	vca.reloadableRateLimiter.Set(flowcontrol.NewTokenBucketRateLimiter(qps, burst))
	log.Log.V(2).Infof("setting rate limiter to %v QPS and %v Burst", qps, burst)
}

// Update virt-controller log verbosity on relevant config changes
func (vca *VirtControllerApp) shouldChangeLogVerbosity() {
	verbosity := vca.clusterConfig.GetVirtControllerVerbosity(vca.host)
	if err := log.Log.SetVerbosityLevel(int(verbosity)); err != nil {
		log.Log.Warningf("failed up update log verbosity to %d: %v", verbosity, err)
	} else {
		log.Log.V(2).Infof("set log verbosity to %d", verbosity)
	}
}

func (vca *VirtControllerApp) Run() {
	logger := log.Log

	promCertManager := bootstrap.NewFileCertificateManager(vca.promCertFilePath, vca.promKeyFilePath)
	go promCertManager.Start()
	promTLSConfig := kvtls.SetupPromTLS(promCertManager, vca.clusterConfig)

	go func() {
		httpLogger := logger.With("service", "http")
		_ = httpLogger.Level(log.INFO).Log("action", "listening", "interface", vca.BindAddress, "port", vca.Port)
		http.Handle("/metrics", promhttp.Handler())
		server := http.Server{
			Addr:      vca.Address(),
			Handler:   http.DefaultServeMux,
			TLSConfig: promTLSConfig,
			// Disable HTTP/2
			// See CVE-2023-44487
			TLSNextProto: map[string]func(*http.Server, *tls.Conn, http.Handler){},
		}
		if err := server.ListenAndServeTLS("", ""); err != nil {
			golog.Fatal(err)
		}
	}()

	if err := vca.setupLeaderElector(); err != nil {
		golog.Fatal(err)
	}

	metrics.SetVirtControllerReady()
	vca.leaderElector.Run(vca.ctx)
	metrics.SetVirtControllerNotReady()
	panic("unreachable")
}

func (vca *VirtControllerApp) onStartedLeading() func(ctx context.Context) {
	return func(ctx context.Context) {
		stop := ctx.Done()
		vca.informerFactory.Start(stop)

		golog.Printf("STARTING controllers with following threads : "+
			"node %d, vmi %d, replicaset %d, vm %d, migration %d, evacuation %d, disruptionBudget %d",
			vca.nodeControllerThreads, vca.vmiControllerThreads, vca.rsControllerThreads,
			vca.vmControllerThreads, vca.migrationControllerThreads, vca.evacuationControllerThreads,
			vca.disruptionBudgetControllerThreads)

		vmiprom.SetupVMICollector(
			vca.vmiInformer,
			vca.clusterInstancetypeInformer, vca.instancetypeInformer,
			vca.clusterPreferenceInformer, vca.preferenceInformer,
			vca.clusterConfig,
		)
		vmprom.SetupVMCollector(vca.vmInformer)
		perfscale.RegisterPerfScaleMetrics(vca.vmiInformer)
		if vca.migrationInformer == nil {
			vca.migrationInformer = vca.informerFactory.VirtualMachineInstanceMigration()
		}
		golog.Printf("\nvca.migrationInformer :%v\n", vca.migrationInformer)
		migration.RegisterMigrationMetrics(vca.migrationInformer)
		migrationstats.SetupMigrationsCollector(vca.migrationInformer)

		go vca.evacuationController.Run(vca.evacuationControllerThreads, stop)
		go vca.disruptionBudgetController.Run(vca.disruptionBudgetControllerThreads, stop)
		go vca.nodeController.Run(vca.nodeControllerThreads, stop)
		go vca.vmiController.Run(vca.vmiControllerThreads, stop)
		go vca.rsController.Run(vca.rsControllerThreads, stop)
		go vca.poolController.Run(vca.poolControllerThreads, stop)
		go vca.vmController.Run(vca.vmControllerThreads, stop)
		go vca.migrationController.Run(vca.migrationControllerThreads, stop)
		go func() {
			if err := vca.snapshotController.Run(vca.snapshotControllerThreads, stop); err != nil {
				log.Log.Warningf("error running the snapshot controller: %v", err)
			}
		}()
		go func() {
			if err := vca.restoreController.Run(vca.restoreControllerThreads, stop); err != nil {
				log.Log.Warningf("error running the restore controller: %v", err)
			}
		}()
		go func() {
			if err := vca.exportController.Run(vca.exportControllerThreads, stop); err != nil {
				log.Log.Warningf("error running the export controller: %v", err)
			}
		}()
		go vca.workloadUpdateController.Run(stop)
		go vca.nodeTopologyUpdater.Run(vca.nodeTopologyUpdatePeriod, stop)
		go func() {
			if err := vca.vmCloneController.Run(vca.cloneControllerThreads, stop); err != nil {
				log.Log.Warningf("error running the clone controller: %v", err)
			}
		}()

		cache.WaitForCacheSync(stop, vca.persistentVolumeClaimInformer.HasSynced, vca.namespaceInformer.HasSynced, vca.resourceQuotaInformer.HasSynced)
		close(vca.readyChan)
		metrics.SetVirtControllerLeading()
	}
}

func (vca *VirtControllerApp) newRecorder(namespace string, componentName string) record.EventRecorder {
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartRecordingToSink(&k8coresv1.EventSinkImpl{Interface: vca.clientSet.CoreV1().Events(namespace)})
	return eventBroadcaster.NewRecorder(scheme.Scheme, k8sv1.EventSource{Component: componentName})
}

func (vca *VirtControllerApp) initCommon() {
	var err error

	virtClient, err := kubecli.GetKubevirtClient()
	if err != nil {
		golog.Fatal(err)
	}

	containerdisk.SetLocalDirectoryOnly(filepath.Join(vca.ephemeralDiskDir, "container-disk-data"))
	vca.templateService = services.NewTemplateService(vca.launcherImage,
		vca.launcherQemuTimeout,
		vca.virtShareDir,
		vca.virtLibDir,
		vca.ephemeralDiskDir,
		vca.containerDiskDir,
		vca.hotplugDiskDir,
		vca.imagePullSecret,
		vca.persistentVolumeClaimCache,
		virtClient,
		vca.clusterConfig,
		vca.launcherSubGid,
		vca.exporterImage,
		vca.resourceQuotaInformer.GetStore(),
		vca.namespaceStore,
		services.WithSidecarCreator(
			func(vmi *v1.VirtualMachineInstance, _ *v1.KubeVirtConfiguration) (hooks.HookSidecarList, error) {
				return hooks.UnmarshalHookSidecarList(vmi)
			}),
		services.WithSidecarCreator(
			func(vmi *v1.VirtualMachineInstance, kvc *v1.KubeVirtConfiguration) (hooks.HookSidecarList, error) {
				return netbinding.NetBindingPluginSidecarList(vmi, kvc, vca.vmiRecorder)
			}),
	)

	topologyHinter := topology.NewTopologyHinter(vca.nodeInformer.GetStore(), vca.vmiInformer.GetStore(), vca.clusterConfig)

	vca.vmiController, err = NewVMIController(vca.templateService,
		vca.vmiInformer,
		vca.vmInformer,
		vca.kvPodInformer,
		vca.persistentVolumeClaimInformer,
		vca.vmiRecorder,
		vca.clientSet,
		vca.dataVolumeInformer,
		vca.cdiInformer,
		vca.cdiConfigInformer,
		vca.clusterConfig,
		topologyHinter,
	)
	if err != nil {
		panic(err)
	}

	recorder := vca.newRecorder(k8sv1.NamespaceAll, "node-controller")
	vca.nodeController, err = NewNodeController(vca.clientSet, vca.nodeInformer, vca.vmiInformer, recorder)
	if err != nil {
		panic(err)
	}
	vca.migrationController, err = NewMigrationController(
		vca.templateService,
		vca.vmiInformer,
		vca.kvPodInformer,
		vca.migrationInformer,
		vca.nodeInformer,
		vca.persistentVolumeClaimInformer,
		vca.pdbInformer,
		vca.migrationPolicyInformer,
		vca.resourceQuotaInformer,
		vca.vmiRecorder,
		vca.clientSet,
		vca.clusterConfig,
	)
	if err != nil {
		panic(err)
	}

	vca.nodeTopologyUpdater = topology.NewNodeTopologyUpdater(vca.clientSet, topologyHinter, vca.nodeInformer)
}

func (vca *VirtControllerApp) initReplicaSet() {
	var err error
	recorder := vca.newRecorder(k8sv1.NamespaceAll, "virtualmachinereplicaset-controller")
	vca.rsController, err = NewVMIReplicaSet(vca.vmiInformer, vca.rsInformer, recorder, vca.clientSet, controller.BurstReplicas)
	if err != nil {
		panic(err)
	}
}

func (vca *VirtControllerApp) initPool() {
	var err error
	recorder := vca.newRecorder(k8sv1.NamespaceAll, "virtualmachinepool-controller")
	vca.poolController, err = NewPoolController(vca.clientSet,
		vca.vmiInformer,
		vca.vmInformer,
		vca.poolInformer,
		vca.controllerRevisionInformer,
		recorder,
		controller.BurstReplicas)
	if err != nil {
		panic(err)
	}
}

func (vca *VirtControllerApp) initVirtualMachines() {
	var err error
	recorder := vca.newRecorder(k8sv1.NamespaceAll, "virtualmachine-controller")

	instancetypeMethods := &instancetype.InstancetypeMethods{
		InstancetypeStore:        vca.instancetypeInformer.GetStore(),
		ClusterInstancetypeStore: vca.clusterInstancetypeInformer.GetStore(),
		PreferenceStore:          vca.preferenceInformer.GetStore(),
		ClusterPreferenceStore:   vca.clusterPreferenceInformer.GetStore(),
		ControllerRevisionStore:  vca.controllerRevisionInformer.GetStore(),
		Clientset:                vca.clientSet,
	}

	vca.vmController, err = NewVMController(
		vca.vmiInformer,
		vca.vmInformer,
		vca.dataVolumeInformer,
		vca.dataSourceInformer,
		vca.namespaceStore,
		vca.persistentVolumeClaimInformer,
		vca.controllerRevisionInformer,
		vca.kvPodInformer,
		instancetypeMethods,
		recorder,
		vca.clientSet,
		vca.clusterConfig)
	if err != nil {
		panic(err)
	}
}

func (vca *VirtControllerApp) initDisruptionBudgetController() {
	var err error
	recorder := vca.newRecorder(k8sv1.NamespaceAll, "disruptionbudget-controller")
	vca.disruptionBudgetController, err = disruptionbudget.NewDisruptionBudgetController(
		vca.vmiInformer,
		vca.pdbInformer,
		vca.allPodInformer,
		vca.migrationInformer,
		recorder,
		vca.clientSet,
		vca.clusterConfig,
	)
	if err != nil {
		panic(err)
	}
}

func (vca *VirtControllerApp) initWorkloadUpdaterController() {
	var err error
	recorder := vca.newRecorder(k8sv1.NamespaceAll, "workload-update-controller")
	vca.workloadUpdateController, err = workloadupdater.NewWorkloadUpdateController(
		vca.launcherImage,
		vca.vmiInformer,
		vca.kvPodInformer,
		vca.migrationInformer,
		vca.kubeVirtInformer,
		recorder,
		vca.clientSet,
		vca.clusterConfig)
	if err != nil {
		panic(err)
	}
}

func (vca *VirtControllerApp) initEvacuationController() {
	var err error
	recorder := vca.newRecorder(k8sv1.NamespaceAll, "disruptionbudget-controller")
	vca.evacuationController, err = evacuation.NewEvacuationController(
		vca.vmiInformer,
		vca.migrationInformer,
		vca.nodeInformer,
		vca.kvPodInformer,
		recorder,
		vca.clientSet,
		vca.clusterConfig,
	)
	if err != nil {
		panic(err)
	}
}

func (vca *VirtControllerApp) initSnapshotController() {
	recorder := vca.newRecorder(k8sv1.NamespaceAll, "snapshot-controller")
	vca.snapshotController = &snapshot.VMSnapshotController{
		Client:                    vca.clientSet,
		VMSnapshotInformer:        vca.vmSnapshotInformer,
		VMSnapshotContentInformer: vca.vmSnapshotContentInformer,
		VMInformer:                vca.vmInformer,
		VMIInformer:               vca.vmiInformer,
		StorageClassInformer:      vca.storageClassInformer,
		PVCInformer:               vca.persistentVolumeClaimInformer,
		CRDInformer:               vca.crdInformer,
		PodInformer:               vca.allPodInformer,
		DVInformer:                vca.dataVolumeInformer,
		CRInformer:                vca.controllerRevisionInformer,
		Recorder:                  recorder,
		ResyncPeriod:              vca.snapshotControllerResyncPeriod,
	}
	if err := vca.snapshotController.Init(); err != nil {
		panic(err)
	}
}

func (vca *VirtControllerApp) initRestoreController() {
	recorder := vca.newRecorder(k8sv1.NamespaceAll, "restore-controller")
	vca.restoreController = &snapshot.VMRestoreController{
		Client:                    vca.clientSet,
		VMRestoreInformer:         vca.vmRestoreInformer,
		VMSnapshotInformer:        vca.vmSnapshotInformer,
		VMSnapshotContentInformer: vca.vmSnapshotContentInformer,
		VMInformer:                vca.vmInformer,
		VMIInformer:               vca.vmiInformer,
		DataVolumeInformer:        vca.dataVolumeInformer,
		PVCInformer:               vca.persistentVolumeClaimInformer,
		StorageClassInformer:      vca.storageClassInformer,
		VolumeSnapshotProvider:    vca.snapshotController,
		Recorder:                  recorder,
		CRInformer:                vca.controllerRevisionInformer,
	}
	if err := vca.restoreController.Init(); err != nil {
		panic(err)
	}
}

func (vca *VirtControllerApp) initExportController() {
	recorder := vca.newRecorder(k8sv1.NamespaceAll, "export-controller")
	vca.exportController = &export.VMExportController{
		TemplateService:             vca.templateService,
		Client:                      vca.clientSet,
		VMExportInformer:            vca.vmExportInformer,
		PVCInformer:                 vca.persistentVolumeClaimInformer,
		PodInformer:                 vca.allPodInformer,
		DataVolumeInformer:          vca.dataVolumeInformer,
		ServiceInformer:             vca.exportServiceInformer,
		Recorder:                    recorder,
		ConfigMapInformer:           vca.caExportConfigMapInformer,
		IngressCache:                vca.ingressCache,
		RouteCache:                  vca.routeCache,
		KubevirtNamespace:           vca.kubevirtNamespace,
		RouteConfigMapInformer:      vca.exportRouteConfigMapInformer,
		SecretInformer:              vca.unmanagedSecretInformer,
		VolumeSnapshotProvider:      vca.snapshotController,
		VMSnapshotInformer:          vca.vmSnapshotInformer,
		VMSnapshotContentInformer:   vca.vmSnapshotContentInformer,
		VMInformer:                  vca.vmInformer,
		VMIInformer:                 vca.vmiInformer,
		CRDInformer:                 vca.crdInformer,
		KubeVirtInformer:            vca.kubeVirtInformer,
		InstancetypeInformer:        vca.instancetypeInformer,
		ClusterInstancetypeInformer: vca.clusterInstancetypeInformer,
		PreferenceInformer:          vca.preferenceInformer,
		ClusterPreferenceInformer:   vca.clusterPreferenceInformer,
		ControllerRevisionInformer:  vca.controllerRevisionInformer,
	}
	if err := vca.exportController.Init(); err != nil {
		panic(err)
	}
}

func (vca *VirtControllerApp) initCloneController() {
	var err error
	recorder := vca.newRecorder(k8sv1.NamespaceAll, "clone-controller")
	vca.vmCloneController, err = clone.NewVmCloneController(
		vca.clientSet, vca.vmCloneInformer, vca.vmSnapshotInformer, vca.vmRestoreInformer, vca.vmInformer, vca.vmSnapshotContentInformer, recorder,
	)
	if err != nil {
		panic(err)
	}
}

func (vca *VirtControllerApp) leaderProbe(_ *restful.Request, response *restful.Response) {
	res := map[string]interface{}{}

	select {
	case _, opened := <-vca.readyChan:
		if !opened {
			res["apiserver"] = map[string]interface{}{"leader": "true"}
			if err := response.WriteHeaderAndJson(http.StatusOK, res, restful.MIME_JSON); err != nil {
				log.Log.Warningf("failed to return 200 OK reply: %v", err)
			}
			return
		}
	default:
	}
	res["apiserver"] = map[string]interface{}{"leader": "false"}
	if err := response.WriteHeaderAndJson(http.StatusOK, res, restful.MIME_JSON); err != nil {
		log.Log.Warningf("failed to return 200 OK reply: %v", err)
	}
}

func (vca *VirtControllerApp) AddFlags() {
	vca.InitFlags()

	leaderelectionconfig.BindFlags(&vca.LeaderElection)

	vca.BindAddress = defaultHost
	vca.Port = defaultPort

	vca.AddCommonFlags()

	flag.StringVar(&vca.launcherImage, "launcher-image", launcherImage,
		"Shim container for containerized VMIs")

	flag.StringVar(&vca.exporterImage, "exporter-image", exporterImage,
		"Container for exporting VMs and VM images")

	flag.IntVar(&vca.launcherQemuTimeout, "launcher-qemu-timeout", launcherQemuTimeout,
		"Amount of time to wait for qemu")

	flag.StringVar(&vca.imagePullSecret, "image-pull-secret", imagePullSecret,
		"Secret to use for pulling virt-launcher and/or registry disks")

	flag.StringVar(&vca.virtShareDir, "kubevirt-share-dir", util.VirtShareDir,
		"Shared directory between virt-handler and virt-launcher")

	flag.StringVar(&vca.virtLibDir, "kubevirt-lib-dir", util.VirtLibDir,
		"Shared lib directory between virt-handler and virt-launcher")

	flag.StringVar(&vca.ephemeralDiskDir, "ephemeral-disk-dir", ephemeralDiskDir,
		"Base directory for ephemeral disk data")

	flag.StringVar(&vca.containerDiskDir, "container-disk-dir", containerDiskDir,
		"Base directory for container disk data")

	flag.StringVar(&vca.hotplugDiskDir, "hotplug-disk-dir", hotplugDiskDir,
		"Base directory for hotplug disk data")

	// allows user-defined threads based on the underlying hardware in use
	flag.IntVar(&vca.nodeControllerThreads, "node-controller-threads", defaultControllerThreads,
		"Number of goroutines to run for node controller")

	flag.IntVar(&vca.vmiControllerThreads, "vmi-controller-threads", defaultVMIControllerThreads,
		"Number of goroutines to run for vmi controller")

	flag.IntVar(&vca.rsControllerThreads, "rs-controller-threads", defaultControllerThreads,
		"Number of goroutines to run for replicaset controller")

	flag.IntVar(&vca.poolControllerThreads, "pool-controller-threads", defaultControllerThreads,
		"Number of goroutines to run for pool controller")

	flag.IntVar(&vca.vmControllerThreads, "vm-controller-threads", defaultControllerThreads,
		"Number of goroutines to run for vm controller")

	flag.IntVar(&vca.migrationControllerThreads, "migration-controller-threads", defaultControllerThreads,
		"Number of goroutines to run for migration controller")

	flag.IntVar(&vca.evacuationControllerThreads, "evacuation-controller-threads", defaultControllerThreads,
		"Number of goroutines to run for evacuation controller")

	flag.IntVar(&vca.disruptionBudgetControllerThreads, "disruption-budget-controller-threads", defaultControllerThreads,
		"Number of goroutines to run for disruption budget controller")

	flag.Int64Var(&vca.launcherSubGid, "launcher-subgid", defaultLauncherSubGid,
		"ID of subgroup to virt-launcher")

	flag.IntVar(&vca.snapshotControllerThreads, "snapshot-controller-threads", defaultSnapshotControllerThreads,
		"Number of goroutines to run for snapshot controller")

	flag.IntVar(&vca.restoreControllerThreads, "restore-controller-threads", defaultControllerThreads,
		"Number of goroutines to run for restore controller")

	flag.IntVar(&vca.exportControllerThreads, "export-controller-threads", defaultControllerThreads,
		"Number of goroutines to run for virtual machine export controller")

	flag.DurationVar(&vca.snapshotControllerResyncPeriod, "snapshot-controller-resync-period", defaultSnapshotControllerResyncPeriod,
		"Number of goroutines to run for snapshot controller")

	flag.DurationVar(&vca.nodeTopologyUpdatePeriod, "node-topology-update-period", defaultNodeTopologyUpdatePeriod,
		"Update period for the node topology updater")

	flag.StringVar(&vca.promCertFilePath, "prom-cert-file", defaultPromCertFilePath,
		"Client certificate used to prove the identity of the virt-controller when it must call out Promethus during a request")

	flag.StringVar(&vca.promKeyFilePath, "prom-key-file", defaultPromKeyFilePath,
		"Private key for the client certificate used to prove the identity of the virt-controller when it must call out Promethus during a request")

	flag.IntVar(&vca.cloneControllerThreads, "clone-controller-threads", defaultControllerThreads,
		"Number of goroutines to run for clone controller")
}

func (vca *VirtControllerApp) setupLeaderElector() (err error) {
	clientConfig, err := kubecli.GetKubevirtClientConfig()
	if err != nil {
		return
	}

	clientConfig.RateLimiter =
		flowcontrol.NewTokenBucketRateLimiter(
			virtconfig.DefaultVirtControllerQPS,
			virtconfig.DefaultVirtControllerBurst)

	clientSet, err := kubecli.GetKubevirtClientFromRESTConfig(clientConfig)
	if err != nil {
		return
	}

	rl, err := resourcelock.New(vca.LeaderElection.ResourceLock,
		vca.kubevirtNamespace,
		leaderelectionconfig.DefaultEndpointName,
		clientSet.CoreV1(),
		clientSet.CoordinationV1(),
		resourcelock.ResourceLockConfig{
			Identity:      vca.host,
			EventRecorder: vca.newRecorder(k8sv1.NamespaceAll, leaderelectionconfig.DefaultEndpointName),
		})

	if err != nil {
		return
	}

	vca.leaderElector, err = leaderelection.NewLeaderElector(
		leaderelection.LeaderElectionConfig{
			Lock:          rl,
			LeaseDuration: vca.LeaderElection.LeaseDuration.Duration,
			RenewDeadline: vca.LeaderElection.RenewDeadline.Duration,
			RetryPeriod:   vca.LeaderElection.RetryPeriod.Duration,
			Callbacks: leaderelection.LeaderCallbacks{
				OnStartedLeading: vca.onStartedLeading(),
				OnStoppedLeading: func() {
					golog.Fatal("leaderelection lost")
				},
			},
		})

	return
}
