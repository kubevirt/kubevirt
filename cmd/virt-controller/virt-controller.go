package main

import (
	"flag"
	"net/http"
	"os"
	"strconv"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/levels"
	golog "log"

	"github.com/emicklei/go-restful"
	"github.com/facebookgo/inject"
	"k8s.io/client-go/1.5/tools/cache"
	"kubevirt/core/pkg/kubecli"
	"kubevirt/core/pkg/virt-controller/rest"
	"kubevirt/core/pkg/virt-controller/services"
	"kubevirt/core/pkg/virt-controller/watch"
)

func main() {

	host := flag.String("listen", "0.0.0.0", "Address and port where to listen on")
	port := flag.Int("port", 8182, "Port to listen on")
	templateFile := flag.String("launcher-template", "./templates/manifest-template.yaml", "Pod manifest template for VMs")
	dockerRegistry := flag.String("docker-registry", "kubevirt", "Organization or private docker registry URL")
	launcherImage := flag.String("launcher-image", "virt-launcher", "Shim container for containerized VMs")

	logger := log.NewLogfmtLogger(os.Stderr)
	flag.Parse()

	var g inject.Graph

	vmService := services.NewVMService(logger)
	templateService, err := services.NewTemplateService(logger, *templateFile, *dockerRegistry, *launcherImage)
	if err != nil {
		golog.Fatal(err)
	}
	vmHandler, err := watch.NewVMResourceEventHandler(logger)
	if err != nil {
		golog.Fatal(err)
	}
	podHandler, err := watch.NewPodResourceEventHandler(logger)
	if err != nil {
		golog.Fatal(err)
	}
	vmCache, err := watch.NewVMCache()
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

	httpLogger := levels.New(logger).With("component", "http")
	httpLogger.Info().Log("action", "listening", "interface", *host, "port", *port)
	if err := http.ListenAndServe(*host+":"+strconv.Itoa(*port), nil); err != nil {
		golog.Fatal(err)
	}
}
