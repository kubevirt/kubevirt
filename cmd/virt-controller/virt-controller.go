package main

import (
	"flag"
	golog "log"
	"net/http"
	"strconv"

	"github.com/emicklei/go-restful"
	"github.com/facebookgo/inject"
	clientrest "k8s.io/client-go/rest"

	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/logging"
	"kubevirt.io/kubevirt/pkg/virt-controller/rest"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
	"kubevirt.io/kubevirt/pkg/virt-controller/watch"
)

func main() {

	logging.InitializeLogging("virt-controller")
	host := flag.String("listen", "0.0.0.0", "Address and port where to listen on")
	port := flag.Int("port", 8182, "Port to listen on")
	launcherImage := flag.String("launcher-image", "virt-launcher", "Shim container for containerized VMs")

	logger := logging.DefaultLogger()
	flag.Parse()

	var g inject.Graph

	vmService := services.NewVMService()
	templateService, err := services.NewTemplateService(*launcherImage)
	if err != nil {
		golog.Fatal(err)
	}

	clientSet, err := kubecli.Get()

	if err != nil {
		golog.Fatal(err)
	}

	var restClient *clientrest.RESTClient
	restClient, err = kubecli.GetRESTClient()
	if err != nil {
		golog.Fatal(err)
	}

	g.Provide(
		&inject.Object{Value: restClient},
		&inject.Object{Value: clientSet},
		&inject.Object{Value: templateService},
		&inject.Object{Value: vmService},
	)

	err = g.Populate()
	if err != nil {
		golog.Fatal(err)
	}
	restful.Add(rest.WebService)

	// Bootstrapping. From here on the initialization order is important
	stop := make(chan struct{})
	defer close(stop)

	// Start wachting vms
	restClient, err = kubecli.GetRESTClient()
	if err != nil {
		golog.Fatal(err)
	}

	vmController := watch.NewVMController(vmService, nil, restClient, clientSet)
	go vmController.Run(1, stop)

	//FIXME when we have more than one worker, we need a lock on the VM
	migrationController := watch.NewMigrationController(vmService, restClient, clientSet)
	go migrationController.Run(1, stop)

	httpLogger := logger.With("service", "http")

	httpLogger.Info().Log("action", "listening", "interface", *host, "port", *port)
	if err := http.ListenAndServe(*host+":"+strconv.Itoa(*port), nil); err != nil {
		golog.Fatal(err)
	}
}
