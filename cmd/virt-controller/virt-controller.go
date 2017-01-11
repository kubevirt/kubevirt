package main

import (
	"flag"
	"net/http"
	"strconv"

	golog "log"

	"github.com/emicklei/go-restful"
	"github.com/facebookgo/inject"
	"k8s.io/client-go/tools/cache"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/logging"
	"kubevirt.io/kubevirt/pkg/util"
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
	vmHandler, err := watch.NewVMResourceEventHandler()
	if err != nil {
		golog.Fatal(err)
	}
	podHandler, err := watch.NewPodResourceEventHandler()
	if err != nil {
		golog.Fatal(err)
	}
	vmCache, err := util.NewVMCache()
	if err != nil {
		golog.Fatal(err)
	}

	clientSet, err := kubecli.Get()

	if err != nil {
		golog.Fatal(err)
	}

	g.Provide(
		&inject.Object{Value: clientSet},
		&inject.Object{Value: templateService},
		&inject.Object{Value: vmService},
		&inject.Object{Value: vmHandler},
		&inject.Object{Value: podHandler},
		&inject.Object{Value: vmCache},
	)

	err = g.Populate()
	if err != nil {
		golog.Fatal(err)
	}
	restful.Add(rest.WebService)

	// Bootstrapping. From here on the initialization order is important
	stop := make(chan struct{})
	defer close(stop)

	// Warm up the vmCache before the pod watcher is started
	go vmCache.Run(stop)
	cache.WaitForCacheSync(stop, vmCache.HasSynced)

	// Start wachting vms
	vmController, err := watch.NewVMInformer(vmHandler)
	if err != nil {
		golog.Fatal(err)
	}
	go vmController.Run(stop)

	// Start watching pods
	podController, err := watch.NewPodInformer(podHandler)
	if err != nil {
		golog.Fatal(err)
	}
	go podController.Run(stop)

	httpLogger := logger.With("service", "http")

	httpLogger.Info().Log("action", "listening", "interface", *host, "port", *port)
	if err := http.ListenAndServe(*host+":"+strconv.Itoa(*port), nil); err != nil {
		golog.Fatal(err)
	}
}
