package watch

import (
	"flag"
	golog "log"
	"net/http"
	"strconv"

	"github.com/emicklei/go-restful"
	clientrest "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	"k8s.io/client-go/util/workqueue"

	kubeinformers "kubevirt.io/kubevirt/pkg/informers"
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
	informerFactory kubeinformers.KubeInformerFactory
	podInformer     cache.SharedIndexInformer

	migrationCache      cache.Store
	migrationController *MigrationController
	migrationInformer   cache.SharedIndexInformer
	migrationQueue      workqueue.RateLimitingInterface

	vmCache      cache.Store
	vmController *VMController
	vmInformer   cache.SharedIndexInformer
	vmQueue      workqueue.RateLimitingInterface

	host          string
	port          int
	launcherImage string
	migratorImage string
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

	app.restClient, err = kubecli.GetRESTClient()
	if err != nil {
		golog.Fatal(err)
	}

	restful.Add(rest.WebService)

	// Bootstrapping. From here on the initialization order is important

	app.informerFactory = kubeinformers.NewKubeInformerFactory(app.restClient, app.clientSet)

	app.vmInformer = app.informerFactory.VM()
	app.migrationInformer = app.informerFactory.Migration()
	app.podInformer = app.informerFactory.KubeVirtPod()

	app.vmQueue = workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	app.vmCache = app.vmInformer.GetStore()
	app.vmInformer.AddEventHandler(kubecli.NewResourceEventHandlerFuncsForWorkqueue(app.vmQueue))
	app.podInformer.AddEventHandler(kubecli.NewResourceEventHandlerFuncsForFunc(vmLabelHandler(app.vmQueue)))

	app.migrationQueue = workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	app.migrationInformer.AddEventHandler(kubecli.NewResourceEventHandlerFuncsForWorkqueue(app.migrationQueue))
	app.podInformer.AddEventHandler(kubecli.NewResourceEventHandlerFuncsForFunc(migrationJobLabelHandler(app.migrationQueue)))
	app.podInformer.AddEventHandler(kubecli.NewResourceEventHandlerFuncsForFunc(migrationPodLabelHandler(app.migrationQueue)))
	app.migrationCache = app.migrationInformer.GetStore()

	app.initCommon()
	app.Run()
}
func (vca *VirtControllerApp) Run() {
	logger := logging.DefaultLogger()
	stop := make(chan struct{})
	defer close(stop)
	vca.informerFactory.Start(stop)
	go vca.vmController.Run(3, stop)
	//FIXME when we have more than one worker, we need a lock on the VM
	go vca.migrationController.Run(3, stop)
	httpLogger := logger.With("service", "http")
	httpLogger.Info().Log("action", "listening", "interface", vca.host, "port", vca.port)
	if err := http.ListenAndServe(vca.host+":"+strconv.Itoa(vca.port), nil); err != nil {
		golog.Fatal(err)
	}
}

func (vca *VirtControllerApp) initCommon() {
	var err error
	vca.templateService, err = services.NewTemplateService(vca.launcherImage, vca.migratorImage)
	if err != nil {
		golog.Fatal(err)
	}
	vca.vmService = services.NewVMService(vca.clientSet, vca.restClient, vca.templateService)
	vca.vmController = NewVMController(vca.restClient, vca.vmService, vca.vmQueue, vca.vmCache, vca.vmInformer, vca.podInformer, nil, vca.clientSet)
	vca.migrationController = NewMigrationController(vca.restClient, vca.vmService, vca.clientSet, vca.migrationQueue, vca.migrationInformer, vca.podInformer, vca.migrationCache)
}

func (vca *VirtControllerApp) DefineFlags() {
	flag.StringVar(&vca.host, "listen", "0.0.0.0", "Address and port where to listen on")
	flag.IntVar(&vca.port, "port", 8182, "Port to listen on")
	flag.StringVar(&vca.launcherImage, "launcher-image", "virt-launcher", "Shim container for containerized VMs")
	flag.StringVar(&vca.migratorImage, "migrator-image", "virt-handler", "Container which orchestrates a VM migration")
	flag.Parse()
}
