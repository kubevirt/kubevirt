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
	"io/ioutil"
	golog "log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/emicklei/go-restful"
	"github.com/golang/glog"
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

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	clientutil "kubevirt.io/client-go/util"
	"kubevirt.io/kubevirt/pkg/certificates"
	containerdisk "kubevirt.io/kubevirt/pkg/container-disk"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/service"
	"kubevirt.io/kubevirt/pkg/util"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-controller/leaderelectionconfig"
	"kubevirt.io/kubevirt/pkg/virt-controller/rest"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
	"kubevirt.io/kubevirt/pkg/virt-controller/watch/drain/disruptionbudget"
	"kubevirt.io/kubevirt/pkg/virt-controller/watch/drain/evacuation"
)

const (
	defaultPort = 8182

	defaultHost = "0.0.0.0"

	launcherImage = "virt-launcher"

	imagePullSecret = ""

	virtShareDir = "/var/run/kubevirt"

	ephemeralDiskDir = virtShareDir + "-ephemeral-disks"

	defaultControllerThreads = 3
)

var (
	containerDiskDir = filepath.Join(util.VirtShareDir, "/container-disks")
)

type VirtControllerApp struct {
	service.ServiceListen

	clientSet       kubecli.KubevirtClient
	templateService services.TemplateService
	restClient      *clientrest.RESTClient
	informerFactory controller.KubeInformerFactory
	podInformer     cache.SharedIndexInformer

	nodeInformer   cache.SharedIndexInformer
	nodeController *NodeController

	vmiCache      cache.Store
	vmiController *VMIController
	vmiInformer   cache.SharedIndexInformer
	vmiRecorder   record.EventRecorder

	clusterConfig *virtconfig.ClusterConfig

	persistentVolumeClaimCache    cache.Store
	persistentVolumeClaimInformer cache.SharedIndexInformer

	rsController *VMIReplicaSet
	rsInformer   cache.SharedIndexInformer

	vmController *VMController
	vmInformer   cache.SharedIndexInformer

	dataVolumeInformer cache.SharedIndexInformer

	migrationController *MigrationController
	migrationInformer   cache.SharedIndexInformer

	LeaderElection leaderelectionconfig.Configuration

	launcherImage              string
	imagePullSecret            string
	virtShareDir               string
	virtLibDir                 string
	ephemeralDiskDir           string
	containerDiskDir           string
	readyChan                  chan bool
	kubevirtNamespace          string
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
}

var _ service.Service = &VirtControllerApp{}

func Execute() {
	var err error
	var app VirtControllerApp = VirtControllerApp{}

	app.LeaderElection = leaderelectionconfig.DefaultLeaderElectionConfiguration()

	service.Setup(&app)

	app.readyChan = make(chan bool, 1)

	log.InitializeLogging("virt-controller")

	app.clientSet, err = kubecli.GetKubevirtClient()

	if err != nil {
		golog.Fatal(err)
	}

	app.restClient = app.clientSet.RestClient()

	webService := rest.WebService
	webService.Route(webService.GET("/leader").To(app.leaderProbe).Doc("Leader endpoint"))
	restful.Add(webService)

	// Bootstrapping. From here on the initialization order is important
	app.kubevirtNamespace, err = clientutil.GetNamespace()
	if err != nil {
		golog.Fatalf("Error searching for namespace: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	stopChan := ctx.Done()
	app.ctx = ctx

	app.informerFactory = controller.NewKubeInformerFactory(app.restClient, app.clientSet, app.kubevirtNamespace)

	configMapInformer := app.informerFactory.ConfigMap()
	crdInformer := app.informerFactory.CRD()
	app.informerFactory.Start(stopChan)

	cache.WaitForCacheSync(stopChan, configMapInformer.HasSynced, crdInformer.HasSynced)
	app.clusterConfig = virtconfig.NewClusterConfig(configMapInformer, crdInformer, app.kubevirtNamespace)

	app.reInitChan = make(chan string, 10)
	app.hasCDI = app.clusterConfig.HasDataVolumeAPI()
	app.clusterConfig.SetConfigModifiedCallback(app.configModificationCallback)

	app.vmiInformer = app.informerFactory.VMI()
	app.podInformer = app.informerFactory.KubeVirtPod()
	app.nodeInformer = app.informerFactory.KubeVirtNode()

	app.vmiCache = app.vmiInformer.GetStore()
	app.vmiRecorder = app.getNewRecorder(k8sv1.NamespaceAll, "virtualmachine-controller")

	app.rsInformer = app.informerFactory.VMIReplicaSet()

	app.persistentVolumeClaimInformer = app.informerFactory.PersistentVolumeClaim()
	app.persistentVolumeClaimCache = app.persistentVolumeClaimInformer.GetStore()

	app.informerFactory.K8SInformerFactory().Policy().V1beta1().PodDisruptionBudgets().Informer()

	app.vmInformer = app.informerFactory.VirtualMachine()

	app.migrationInformer = app.informerFactory.VirtualMachineInstanceMigration()

	if app.hasCDI {
		app.dataVolumeInformer = app.informerFactory.DataVolume()
		log.Log.Infof("CDI detected, DataVolume integration enabled")
	} else {
		// Add a dummy DataVolume informer in the event datavolume support
		// is disabled. This lets the controller continue to work without
		// requiring a separate branching code path.
		app.dataVolumeInformer = app.informerFactory.DummyDataVolume()
		log.Log.Infof("CDI not detected, DataVolume integration disabled")
	}

	app.initCommon()
	app.initReplicaSet()
	app.initVirtualMachines()
	app.initDisruptionBudgetController()
	app.initEvacuationController()
	go app.Run()

	select {
	case <-app.reInitChan:
		cancel()
	}
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

func (vca *VirtControllerApp) Run() {
	logger := log.Log

	stop := vca.ctx.Done()

	certsDirectory, err := ioutil.TempDir("", "certsdir")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(certsDirectory)

	certStore, err := certificates.GenerateSelfSignedCert(certsDirectory, "virt-controller", vca.kubevirtNamespace)
	if err != nil {
		glog.Fatalf("unable to generate certificates: %v", err)
	}
	go func() {
		httpLogger := logger.With("service", "http")
		httpLogger.Level(log.INFO).Log("action", "listening", "interface", vca.BindAddress, "port", vca.Port)
		http.Handle("/metrics", promhttp.Handler())
		if err := http.ListenAndServeTLS(vca.Address(), certStore.CurrentPath(), certStore.CurrentPath(), nil); err != nil {
			golog.Fatal(err)
		}
	}()

	recorder := vca.getNewRecorder(k8sv1.NamespaceAll, leaderelectionconfig.DefaultEndpointName)

	id, err := os.Hostname()
	if err != nil {
		golog.Fatalf("unable to get hostname: %v", err)
	}

	rl, err := resourcelock.New(vca.LeaderElection.ResourceLock,
		vca.kubevirtNamespace,
		leaderelectionconfig.DefaultEndpointName,
		vca.clientSet.CoreV1(),
		vca.clientSet.CoordinationV1(),
		resourcelock.ResourceLockConfig{
			Identity:      id,
			EventRecorder: recorder,
		})
	if err != nil {
		golog.Fatal(err)
	}

	leaderElector, err := leaderelection.NewLeaderElector(
		leaderelection.LeaderElectionConfig{
			Lock:          rl,
			LeaseDuration: vca.LeaderElection.LeaseDuration.Duration,
			RenewDeadline: vca.LeaderElection.RenewDeadline.Duration,
			RetryPeriod:   vca.LeaderElection.RetryPeriod.Duration,
			Callbacks: leaderelection.LeaderCallbacks{
				OnStartedLeading: func(ctx context.Context) {
					vca.informerFactory.Start(stop)

					golog.Printf("STARTING controllers with following threads : "+
						"node %d, vmi %d, replicaset %d, vm %d, migration %d, evacuation %d, disruptionBudget %d",
						vca.nodeControllerThreads, vca.vmiControllerThreads, vca.rsControllerThreads,
						vca.vmControllerThreads, vca.migrationControllerThreads, vca.evacuationControllerThreads,
						vca.disruptionBudgetControllerThreads)

					go vca.evacuationController.Run(vca.evacuationControllerThreads, stop)
					go vca.disruptionBudgetController.Run(vca.disruptionBudgetControllerThreads, stop)
					go vca.nodeController.Run(vca.nodeControllerThreads, stop)
					go vca.vmiController.Run(vca.vmiControllerThreads, stop)
					go vca.rsController.Run(vca.rsControllerThreads, stop)
					go vca.vmController.Run(vca.vmControllerThreads, stop)
					go vca.migrationController.Run(vca.migrationControllerThreads, stop)
					cache.WaitForCacheSync(stop, vca.persistentVolumeClaimInformer.HasSynced)
					close(vca.readyChan)
				},
				OnStoppedLeading: func() {
					golog.Fatal("leaderelection lost")
				},
			},
		})
	if err != nil {
		golog.Fatal(err)
	}

	leaderElector.Run(vca.ctx)
	panic("unreachable")
}

func (vca *VirtControllerApp) getNewRecorder(namespace string, componentName string) record.EventRecorder {
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
		vca.virtShareDir,
		vca.virtLibDir,
		vca.ephemeralDiskDir,
		vca.containerDiskDir,
		vca.imagePullSecret,
		vca.persistentVolumeClaimCache,
		virtClient,
		vca.clusterConfig,
	)

	vca.vmiController = NewVMIController(vca.templateService, vca.vmiInformer, vca.podInformer, vca.vmiRecorder, vca.clientSet, vca.dataVolumeInformer)
	recorder := vca.getNewRecorder(k8sv1.NamespaceAll, "node-controller")
	vca.nodeController = NewNodeController(vca.clientSet, vca.nodeInformer, vca.vmiInformer, recorder)
	vca.migrationController = NewMigrationController(vca.templateService, vca.vmiInformer, vca.podInformer, vca.migrationInformer, vca.vmiRecorder, vca.clientSet, vca.clusterConfig)
}

func (vca *VirtControllerApp) initReplicaSet() {
	recorder := vca.getNewRecorder(k8sv1.NamespaceAll, "virtualmachinereplicaset-controller")
	vca.rsController = NewVMIReplicaSet(vca.vmiInformer, vca.rsInformer, recorder, vca.clientSet, controller.BurstReplicas)
}

func (vca *VirtControllerApp) initVirtualMachines() {
	recorder := vca.getNewRecorder(k8sv1.NamespaceAll, "virtualmachine-controller")

	vca.vmController = NewVMController(
		vca.vmiInformer,
		vca.vmInformer,
		vca.dataVolumeInformer,
		recorder,
		vca.clientSet)
}

func (vca *VirtControllerApp) initDisruptionBudgetController() {
	recorder := vca.getNewRecorder(k8sv1.NamespaceAll, "disruptionbudget-controller")
	vca.disruptionBudgetController = disruptionbudget.NewDisruptionBudgetController(
		vca.vmiInformer,
		vca.informerFactory.K8SInformerFactory().Policy().V1beta1().PodDisruptionBudgets().Informer(),
		recorder,
		vca.clientSet,
	)

}

func (vca *VirtControllerApp) initEvacuationController() {
	recorder := vca.getNewRecorder(k8sv1.NamespaceAll, "disruptionbudget-controller")
	vca.evacuationController = evacuation.NewEvacuationController(
		vca.vmiInformer,
		vca.migrationInformer,
		vca.nodeInformer,
		recorder,
		vca.clientSet,
		vca.clusterConfig,
	)
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

	// allows user-defined threads based on the underlying hardware in use
	flag.IntVar(&vca.nodeControllerThreads, "node-controller-threads", defaultControllerThreads,
		"Number of goroutines to run for node controller")

	flag.IntVar(&vca.vmiControllerThreads, "vmi-controller-threads", defaultControllerThreads,
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
}
