package watch

import (
	"flag"
	golog "log"
	"net/http"
	"strconv"

	"github.com/emicklei/go-restful"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	k8coresv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	clientrest "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	"k8s.io/client-go/util/workqueue"

	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	v12 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"

	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/logging"
	"kubevirt.io/kubevirt/pkg/virt-controller/rest"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
)

type VirtControllerApp struct {
	clientSet       kubecli.KubevirtClient
	templateService services.TemplateService
	restClient      *clientrest.RESTClient
	vmService       services.VMService
	informerFactory controller.KubeInformerFactory
	podInformer     cache.SharedIndexInformer

	migrationCache      cache.Store
	migrationController *MigrationController
	migrationInformer   cache.SharedIndexInformer
	migrationQueue      workqueue.RateLimitingInterface
	migrationRecorder   record.EventRecorder

	vmCache      cache.Store
	vmController *VMController
	vmInformer   cache.SharedIndexInformer
	vmQueue      workqueue.RateLimitingInterface

	rsController *VMReplicaSet
	rsInformer   cache.SharedIndexInformer

	host          string
	port          int
	launcherImage string
	migratorImage string
	socketDir     string
}

func Execute() {
	var err error
	var app VirtControllerApp = VirtControllerApp{}

	app.DefineFlags()

	logging.InitializeLogging("virt-controller")

	app.clientSet, err = kubecli.GetKubevirtClient()

	if err != nil {
		golog.Fatal(err)
	}

	app.restClient = app.clientSet.RestClient()

	restful.Add(rest.WebService)

	// Bootstrapping. From here on the initialization order is important

	app.informerFactory = controller.NewKubeInformerFactory(app.restClient, app.clientSet)

	app.vmInformer = app.informerFactory.VM()
	app.migrationInformer = app.informerFactory.Migration()
	app.podInformer = app.informerFactory.KubeVirtPod()

	app.vmQueue = workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	app.vmCache = app.vmInformer.GetStore()
	app.vmInformer.AddEventHandler(controller.NewResourceEventHandlerFuncsForWorkqueue(app.vmQueue))
	app.podInformer.AddEventHandler(controller.NewResourceEventHandlerFuncsForFunc(vmLabelHandler(app.vmQueue)))

	app.migrationQueue = workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	app.migrationInformer.AddEventHandler(controller.NewResourceEventHandlerFuncsForWorkqueue(app.migrationQueue))
	app.podInformer.AddEventHandler(controller.NewResourceEventHandlerFuncsForFunc(migrationJobLabelHandler(app.migrationQueue)))
	app.podInformer.AddEventHandler(controller.NewResourceEventHandlerFuncsForFunc(migrationPodLabelHandler(app.migrationQueue)))
	app.migrationCache = app.migrationInformer.GetStore()

	broadcaster := record.NewBroadcaster()
	broadcaster.StartRecordingToSink(&k8coresv1.EventSinkImpl{Interface: app.clientSet.CoreV1().Events(k8sv1.NamespaceAll)})
	app.migrationRecorder = broadcaster.NewRecorder(scheme.Scheme, k8sv1.EventSource{Component: "virt-migration-controller"})

	app.rsInformer = app.informerFactory.VMReplicaSet()

	app.initCommon()
	app.initReplicaSet()
	app.Run()
}
func (vca *VirtControllerApp) Run() {
	logger := logging.DefaultLogger()
	stop := make(chan struct{})
	defer close(stop)
	vca.informerFactory.Start(stop)
	go vca.vmController.Run(3, stop)
	go vca.migrationController.Run(3, stop)
	go vca.rsController.Run(3, stop)
	httpLogger := logger.With("service", "http")
	httpLogger.Info().Log("action", "listening", "interface", vca.host, "port", vca.port)
	if err := http.ListenAndServe(vca.host+":"+strconv.Itoa(vca.port), nil); err != nil {
		golog.Fatal(err)
	}
}

func (vca *VirtControllerApp) initCommon() {
	var err error
	vca.templateService, err = services.NewTemplateService(vca.launcherImage, vca.migratorImage, vca.socketDir)
	if err != nil {
		golog.Fatal(err)
	}
	vca.vmService = services.NewVMService(vca.clientSet, vca.restClient, vca.templateService)
	vca.vmController = NewVMController(vca.restClient, vca.vmService, vca.vmQueue, vca.vmCache, vca.vmInformer, vca.podInformer, nil, vca.clientSet)
	vca.migrationController = NewMigrationController(vca.restClient, vca.vmService, vca.clientSet, vca.migrationQueue, vca.migrationInformer, vca.podInformer, vca.migrationCache, vca.migrationRecorder)
}

func (vca *VirtControllerApp) initReplicaSet() {
	broadcaster := record.NewBroadcaster()
	broadcaster.StartRecordingToSink(&v12.EventSinkImpl{Interface: vca.clientSet.CoreV1().Events(v1.NamespaceAll)})
	// TODO what is scheme used for in Recorder?
	recorder := broadcaster.NewRecorder(scheme.Scheme, v1.EventSource{Component: "virtualmachinereplicaset-controller"})

	vca.rsController = NewVMReplicaSet(vca.vmInformer, vca.rsInformer, recorder, vca.clientSet, controller.BurstReplicas)
}

func (vca *VirtControllerApp) DefineFlags() {
	flag.StringVar(&vca.host, "listen", "0.0.0.0", "Address and port where to listen on")
	flag.IntVar(&vca.port, "port", 8182, "Port to listen on")
	flag.StringVar(&vca.launcherImage, "launcher-image", "virt-launcher", "Shim container for containerized VMs")
	flag.StringVar(&vca.migratorImage, "migrator-image", "virt-handler", "Container which orchestrates a VM migration")
	flag.StringVar(&vca.socketDir, "socket-dir", "/var/run/kubevirt", "Directory where to look for sockets for cgroup detection")
	flag.Parse()
}
