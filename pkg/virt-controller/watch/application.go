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

	"k8s.io/client-go/kubernetes"

	kubeinformers "kubevirt.io/kubevirt/pkg/informers"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/logging"
	"kubevirt.io/kubevirt/pkg/virt-controller/rest"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
)

var (
	flags               Flags = Flags{}
	host                string
	port                int
	templateService     services.TemplateService
	restClient          *clientrest.RESTClient
	clientSet           *kubernetes.Clientset
	vmService           services.VMService
	informerFactory     kubeinformers.KubeInformerFactory
	vmInformer          cache.SharedIndexInformer
	migrationInformer   cache.SharedIndexInformer
	podInformer         cache.SharedIndexInformer
	vmController        *VMController
	migrationController *MigrationController
	migrationQueue      workqueue.RateLimitingInterface
	vmCache             cache.Store
)

type Flags struct {
	host          string
	port          int
	launcherImage string
	migratorImage string
}

func Execute() {
	var err error

	DefineFlags()

	logging.InitializeLogging("virt-controller")
	logger := logging.DefaultLogger()

	templateService, err = services.NewTemplateService(flags.launcherImage, flags.migratorImage)
	if err != nil {
		golog.Fatal(err)
	}

	clientSet, err = kubecli.Get()

	if err != nil {
		golog.Fatal(err)
	}

	restClient, err = kubecli.GetRESTClient()
	if err != nil {
		golog.Fatal(err)
	}

	vmService = services.NewVMService(clientSet, restClient, templateService)

	restful.Add(rest.WebService)

	// Bootstrapping. From here on the initialization order is important

	informerFactory = kubeinformers.NewKubeInformerFactory(restClient, clientSet)

	vmInformer = informerFactory.VM()
	migrationInformer = informerFactory.Migration()
	podInformer = informerFactory.KubeVirtPod()
	vmController = NewVMController(vmService, nil, restClient, clientSet, vmInformer, podInformer)

	migrationQueue = workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	migrationInformer.AddEventHandler(kubecli.NewResourceEventHandlerFuncsForWorkqueue(migrationQueue))
	podInformer.AddEventHandler(kubecli.NewResourceEventHandlerFuncsForFunc(migrationJobLabelHandler(migrationQueue)))
	podInformer.AddEventHandler(kubecli.NewResourceEventHandlerFuncsForFunc(migrationPodLabelHandler(migrationQueue)))
	vmCache = migrationInformer.GetStore()

	migrationController = NewMigrationController(restClient, vmService, clientSet, migrationQueue, migrationInformer, podInformer, vmCache)

	stop := make(chan struct{})
	defer close(stop)
	informerFactory.Start(stop)

	go vmController.Run(1, stop)

	//FIXME when we have more than one worker, we need a lock on the VM
	go migrationController.Run(1, stop)

	httpLogger := logger.With("service", "http")

	httpLogger.Info().Log("action", "listening", "interface", host, "port", port)
	if err := http.ListenAndServe(host+":"+strconv.Itoa(port), nil); err != nil {
		golog.Fatal(err)
	}
}
func DefineFlags() {
	flag.StringVar(&flags.host, "listen", "0.0.0.0", "Address and port where to listen on")
	flag.IntVar(&flags.port, "port", 8182, "Port to listen on")
	flag.StringVar(&flags.launcherImage, "launcher-image", "virt-launcher", "Shim container for containerized VMs")
	flag.StringVar(&flags.migratorImage, "migrator-image", "virt-handler", "Container which orchestrates a VM migration")
	flag.Parse()
}
