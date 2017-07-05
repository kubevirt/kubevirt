package watch

import (
	"flag"
	golog "log"
	"net/http"
	"strconv"

	"github.com/emicklei/go-restful"
	clientrest "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	kubeinformers "kubevirt.io/kubevirt/pkg/informers"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/logging"
	"kubevirt.io/kubevirt/pkg/virt-controller/rest"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
)

var (
	host                string
	port                int
	templateService     services.TemplateService
	restClient          *clientrest.RESTClient
	vmService           services.VMService
	informerFactory     kubeinformers.KubeInformerFactory
	vmInformer          cache.SharedIndexInformer
	migrationInformer   cache.SharedIndexInformer
	podInformer         cache.SharedIndexInformer
	vmController        *VMController
	migrationController *MigrationController
)

func Execute() {
	var err error

	logging.InitializeLogging("virt-controller")
	flag.StringVar(&host, "listen", "0.0.0.0", "Address and port where to listen on")
	flag.IntVar(&port, "port", 8182, "Port to listen on")
	launcherImage := flag.String("launcher-image", "virt-launcher", "Shim container for containerized VMs")
	migratorImage := flag.String("migrator-image", "virt-handler", "Container which orchestrates a VM migration")

	logger := logging.DefaultLogger()
	flag.Parse()

	templateService, err = services.NewTemplateService(*launcherImage, *migratorImage)
	if err != nil {
		golog.Fatal(err)
	}

	clientSet, err := kubecli.Get()

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
	migrationController = NewMigrationController(vmService, restClient, clientSet, migrationInformer, podInformer)

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
