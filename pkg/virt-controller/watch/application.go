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
	golog "log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"kubevirt.io/kubevirt/pkg/flavor"

	"github.com/emicklei/go-restful"
	vsv1beta1 "github.com/kubernetes-csi/external-snapshotter/v2/pkg/apis/volumesnapshot/v1beta1"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	flag "github.com/spf13/pflag"
	k8sv1 "k8s.io/api/core/v1"
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

	snapshotv1 "kubevirt.io/api/snapshot/v1alpha1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	clientutil "kubevirt.io/client-go/util"
	"kubevirt.io/kubevirt/pkg/certificates/bootstrap"
	containerdisk "kubevirt.io/kubevirt/pkg/container-disk"
	"kubevirt.io/kubevirt/pkg/controller"

	"kubevirt.io/kubevirt/pkg/monitoring/perfscale"
	vmiprom "kubevirt.io/kubevirt/pkg/monitoring/vmistats" // import for prometheus metrics
	"kubevirt.io/kubevirt/pkg/service"
	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/util/webhooks"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-controller/leaderelectionconfig"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
	"kubevirt.io/kubevirt/pkg/virt-controller/watch/drain/disruptionbudget"
	"kubevirt.io/kubevirt/pkg/virt-controller/watch/drain/evacuation"
	"kubevirt.io/kubevirt/pkg/virt-controller/watch/snapshot"
	workloadupdater "kubevirt.io/kubevirt/pkg/virt-controller/watch/workload-updater"
)

const (
	defaultPort = 8182

	defaultHost = "0.0.0.0"

	launcherImage       = "virt-launcher"
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
	containerDiskDir = filepath.Join(util.VirtShareDir, "/container-disks")
	hotplugDiskDir   = filepath.Join(util.VirtShareDir, "/hotplug-disks")

	leaderGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "kubevirt_virt_controller_leading",
			Help: "Indication for an operating virt-controller.",
		},
	)

	readyGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "kubevirt_virt_controller_ready",
			Help: "Indication for a virt-controller that is ready to take the lead.",
		},
	)

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

	kubeVirtInformer cache.SharedIndexInformer

	clusterConfig *virtconfig.ClusterConfig

	pdbInformer cache.SharedIndexInformer

	persistentVolumeClaimCache    cache.Store
	persistentVolumeClaimInformer cache.SharedIndexInformer

	rsController *VMIReplicaSet
	rsInformer   cache.SharedIndexInformer

	vmController *VMController
	vmInformer   cache.SharedIndexInformer

	controllerRevisionInformer cache.SharedIndexInformer

	dataVolumeInformer cache.SharedIndexInformer
	cdiInformer        cache.SharedIndexInformer
	cdiConfigInformer  cache.SharedIndexInformer

	migrationController *MigrationController
	migrationInformer   cache.SharedIndexInformer

	workloadUpdateController *workloadupdater.WorkloadUpdateController

	snapshotController        *snapshot.VMSnapshotController
	restoreController         *snapshot.VMRestoreController
	vmSnapshotInformer        cache.SharedIndexInformer
	vmSnapshotContentInformer cache.SharedIndexInformer
	vmRestoreInformer         cache.SharedIndexInformer
	storageClassInformer      cache.SharedIndexInformer
	allPodInformer            cache.SharedIndexInformer

	crdInformer cache.SharedIndexInformer

	flavorInformer        cache.SharedIndexInformer
	clusterFlavorInformer cache.SharedIndexInformer

	LeaderElection leaderelectionconfig.Configuration

	launcherImage              string
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
	vmControllerThreads               int
	migrationControllerThreads        int
	evacuationControllerThreads       int
	disruptionBudgetControllerThreads int
	launcherSubGid                    int64
	snapshotControllerThreads         int
	restoreControllerThreads          int
	snapshotControllerResyncPeriod    time.Duration

	caConfigMapName          string
	promCertFilePath         string
	promKeyFilePath          string
	nodeTopologyUpdater      topology.NodeTopologyUpdater
	nodeTopologyUpdatePeriod time.Duration
	reloadableRateLimiter    *ratelimiter.ReloadableRateLimiter
	leaderElector            *leaderelection.LeaderElector
}

var _ service.Service = &VirtControllerApp{}

func init() {
	vsv1beta1.AddToScheme(scheme.Scheme)
	snapshotv1.AddToScheme(scheme.Scheme)

	prometheus.MustRegister(leaderGauge)
	prometheus.MustRegister(readyGauge)
}

func Execute() {
	var err error
	var app VirtControllerApp = VirtControllerApp{}

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
	app.informerFactory.Start(stopChan)

	app.kubeVirtInformer.SetWatchErrorHandler(func(r *cache.Reflector, err error) {
		apiHealthVersion.Clear()
		cache.DefaultWatchErrorHandler(r, err)
	})

	cache.WaitForCacheSync(stopChan, app.crdInformer.HasSynced, app.kubeVirtInformer.HasSynced)
	app.clusterConfig = virtconfig.NewClusterConfig(app.crdInformer, app.kubeVirtInformer, app.kubevirtNamespace)

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

	app.vmiCache = app.vmiInformer.GetStore()
	app.vmiRecorder = app.newRecorder(k8sv1.NamespaceAll, "virtualmachine-controller")

	app.rsInformer = app.informerFactory.VMIReplicaSet()

	app.persistentVolumeClaimInformer = app.informerFactory.PersistentVolumeClaim()
	app.persistentVolumeClaimCache = app.persistentVolumeClaimInformer.GetStore()

	app.pdbInformer = app.informerFactory.K8SInformerFactory().Policy().V1beta1().PodDisruptionBudgets().Informer()

	app.vmInformer = app.informerFactory.VirtualMachine()

	app.migrationInformer = app.informerFactory.VirtualMachineInstanceMigration()

	app.controllerRevisionInformer = app.informerFactory.ControllerRevision()

	app.vmSnapshotInformer = app.informerFactory.VirtualMachineSnapshot()
	app.vmSnapshotContentInformer = app.informerFactory.VirtualMachineSnapshotContent()
	app.vmRestoreInformer = app.informerFactory.VirtualMachineRestore()
	app.storageClassInformer = app.informerFactory.StorageClass()
	app.allPodInformer = app.informerFactory.Pod()

	if app.hasCDI {
		app.dataVolumeInformer = app.informerFactory.DataVolume()
		app.cdiInformer = app.informerFactory.CDI()
		app.cdiConfigInformer = app.informerFactory.CDIConfig()
		log.Log.Infof("CDI detected, DataVolume integration enabled")
	} else {
		// Add a dummy DataVolume informer in the event datavolume support
		// is disabled. This lets the controller continue to work without
		// requiring a separate branching code path.
		app.dataVolumeInformer = app.informerFactory.DummyDataVolume()
		app.cdiInformer = app.informerFactory.DummyCDI()
		app.cdiConfigInformer = app.informerFactory.DummyCDIConfig()
		log.Log.Infof("CDI not detected, DataVolume integration disabled")
	}

	app.flavorInformer = app.informerFactory.VirtualMachineFlavor()
	app.clusterFlavorInformer = app.informerFactory.VirtualMachineClusterFlavor()

	app.initCommon()
	app.initReplicaSet()
	app.initVirtualMachines()
	app.initDisruptionBudgetController()
	app.initEvacuationController()
	app.initSnapshotController()
	app.initRestoreController()
	app.initWorkloadUpdaterController()
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
func (app *VirtControllerApp) shouldChangeRateLimiter() {
	config := app.clusterConfig.GetConfig()
	qps := config.ControllerConfiguration.RestClient.RateLimiter.TokenBucketRateLimiter.QPS
	burst := config.ControllerConfiguration.RestClient.RateLimiter.TokenBucketRateLimiter.Burst
	app.reloadableRateLimiter.Set(flowcontrol.NewTokenBucketRateLimiter(qps, burst))
	log.Log.V(2).Infof("setting rate limiter to %v QPS and %v Burst", qps, burst)
}

// Update virt-controller log verbosity on relevant config changes
func (vca *VirtControllerApp) shouldChangeLogVerbosity() {
	verbosity := vca.clusterConfig.GetVirtControllerVerbosity(vca.host)
	log.Log.SetVerbosityLevel(int(verbosity))
	log.Log.V(2).Infof("set log verbosity to %d", verbosity)
}

func (vca *VirtControllerApp) Run() {
	logger := log.Log

	promCertManager := bootstrap.NewFileCertificateManager(vca.promCertFilePath, vca.promKeyFilePath)
	go promCertManager.Start()
	promTLSConfig := webhooks.SetupPromTLS(promCertManager)

	go func() {
		httpLogger := logger.With("service", "http")
		httpLogger.Level(log.INFO).Log("action", "listening", "interface", vca.BindAddress, "port", vca.Port)
		http.Handle("/metrics", promhttp.Handler())
		server := http.Server{
			Addr:      vca.Address(),
			Handler:   http.DefaultServeMux,
			TLSConfig: promTLSConfig,
		}
		if err := server.ListenAndServeTLS("", ""); err != nil {
			golog.Fatal(err)
		}
	}()

	if err := vca.setupLeaderElector(); err != nil {
		golog.Fatal(err)
	}

	readyGauge.Set(1)
	vca.leaderElector.Run(vca.ctx)
	readyGauge.Set(0)
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

		vmiprom.SetupVMICollector(vca.vmiInformer)
		perfscale.RegisterPerfScaleMetrics(vca.vmiInformer)

		go vca.evacuationController.Run(vca.evacuationControllerThreads, stop)
		go vca.disruptionBudgetController.Run(vca.disruptionBudgetControllerThreads, stop)
		go vca.nodeController.Run(vca.nodeControllerThreads, stop)
		go vca.vmiController.Run(vca.vmiControllerThreads, stop)
		go vca.rsController.Run(vca.rsControllerThreads, stop)
		go vca.vmController.Run(vca.vmControllerThreads, stop)
		go vca.migrationController.Run(vca.migrationControllerThreads, stop)
		go vca.snapshotController.Run(vca.snapshotControllerThreads, stop)
		go vca.restoreController.Run(vca.restoreControllerThreads, stop)
		go vca.workloadUpdateController.Run(stop)
		go vca.nodeTopologyUpdater.Run(vca.nodeTopologyUpdatePeriod, stop)

		cache.WaitForCacheSync(stop, vca.persistentVolumeClaimInformer.HasSynced)
		close(vca.readyChan)
		leaderGauge.Set(1)
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

	containerdisk.SetLocalDirectory(vca.ephemeralDiskDir + "/container-disk-data")
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
	)

	topologyHinter := topology.NewTopologyHinter(vca.nodeInformer.GetStore(), vca.vmiInformer.GetStore(), runtime.GOARCH, vca.clusterConfig)

	vca.vmiController = NewVMIController(vca.templateService,
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

	recorder := vca.newRecorder(k8sv1.NamespaceAll, "node-controller")
	vca.nodeController = NewNodeController(vca.clientSet, vca.nodeInformer, vca.vmiInformer, recorder)
	vca.migrationController = NewMigrationController(
		vca.templateService,
		vca.vmiInformer,
		vca.kvPodInformer,
		vca.migrationInformer,
		vca.nodeInformer,
		vca.persistentVolumeClaimInformer,
		vca.pdbInformer,
		vca.vmiRecorder,
		vca.clientSet,
		vca.clusterConfig,
	)

	vca.nodeTopologyUpdater = topology.NewNodeTopologyUpdater(vca.clientSet, topologyHinter, vca.nodeInformer)
}

func (vca *VirtControllerApp) initReplicaSet() {
	recorder := vca.newRecorder(k8sv1.NamespaceAll, "virtualmachinereplicaset-controller")
	vca.rsController = NewVMIReplicaSet(vca.vmiInformer, vca.rsInformer, recorder, vca.clientSet, controller.BurstReplicas)
}

func (vca *VirtControllerApp) initVirtualMachines() {
	recorder := vca.newRecorder(k8sv1.NamespaceAll, "virtualmachine-controller")

	vca.vmController = NewVMController(
		vca.vmiInformer,
		vca.vmInformer,
		vca.dataVolumeInformer,
		vca.persistentVolumeClaimInformer,
		vca.controllerRevisionInformer,
		flavor.NewMethods(vca.flavorInformer.GetStore(), vca.clusterFlavorInformer.GetStore()),
		recorder,
		vca.clientSet)
}

func (vca *VirtControllerApp) initDisruptionBudgetController() {
	recorder := vca.newRecorder(k8sv1.NamespaceAll, "disruptionbudget-controller")
	vca.disruptionBudgetController = disruptionbudget.NewDisruptionBudgetController(
		vca.vmiInformer,
		vca.pdbInformer,
		vca.allPodInformer,
		vca.migrationInformer,
		recorder,
		vca.clientSet,
	)

}

func (vca *VirtControllerApp) initWorkloadUpdaterController() {
	recorder := vca.newRecorder(k8sv1.NamespaceAll, "workload-update-controller")
	vca.workloadUpdateController = workloadupdater.NewWorkloadUpdateController(
		vca.launcherImage,
		vca.vmiInformer,
		vca.kvPodInformer,
		vca.migrationInformer,
		vca.kubeVirtInformer,
		recorder,
		vca.clientSet,
		vca.clusterConfig)
}

func (vca *VirtControllerApp) initEvacuationController() {
	recorder := vca.newRecorder(k8sv1.NamespaceAll, "disruptionbudget-controller")
	vca.evacuationController = evacuation.NewEvacuationController(
		vca.vmiInformer,
		vca.migrationInformer,
		vca.nodeInformer,
		vca.kvPodInformer,
		recorder,
		vca.clientSet,
		vca.clusterConfig,
	)
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
	vca.snapshotController.Init()
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
		Recorder:                  recorder,
	}
	vca.restoreController.Init()
}

func (vca *VirtControllerApp) leaderProbe(_ *restful.Request, response *restful.Response) {
	res := map[string]interface{}{}

	select {
	case _, opened := <-vca.readyChan:
		if !opened {
			res["apiserver"] = map[string]interface{}{"leader": "true"}
			response.WriteHeaderAndJson(http.StatusOK, res, restful.MIME_JSON)
			return
		}
	default:
	}
	res["apiserver"] = map[string]interface{}{"leader": "false"}
	response.WriteHeaderAndJson(http.StatusOK, res, restful.MIME_JSON)
}

func (vca *VirtControllerApp) AddFlags() {
	vca.InitFlags()

	leaderelectionconfig.BindFlags(&vca.LeaderElection)

	vca.BindAddress = defaultHost
	vca.Port = defaultPort

	vca.AddCommonFlags()

	flag.StringVar(&vca.launcherImage, "launcher-image", launcherImage,
		"Shim container for containerized VMIs")

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

	flag.DurationVar(&vca.snapshotControllerResyncPeriod, "snapshot-controller-resync-period", defaultSnapshotControllerResyncPeriod,
		"Number of goroutines to run for snapshot controller")

	flag.DurationVar(&vca.nodeTopologyUpdatePeriod, "node-topology-update-period", defaultNodeTopologyUpdatePeriod,
		"Update period for the node topology updater")

	flag.StringVar(&vca.promCertFilePath, "prom-cert-file", defaultPromCertFilePath,
		"Client certificate used to prove the identity of the virt-controller when it must call out Promethus during a request")

	flag.StringVar(&vca.promKeyFilePath, "prom-key-file", defaultPromKeyFilePath,
		"Private key for the client certificate used to prove the identity of the virt-controller when it must call out Promethus during a request")
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
