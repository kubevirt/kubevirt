package main

import (
	"flag"
	"golang.org/x/net/context"
	"net/http"
	"os"
	"strconv"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/levels"
	golog "log"

	"github.com/facebookgo/inject"
	"kubevirt/core/pkg/middleware"
	"kubevirt/core/pkg/virt-controller/endpoints"
	"kubevirt/core/pkg/virt-controller/rest"
	"kubevirt/core/pkg/virt-controller/services"
)

func main() {

	host := flag.String("listen", "0.0.0.0", "Address and port where to listen on")
	port := flag.Int("port", 8182, "Port to listen on")
	templateFile := flag.String("launcher-template", "./templates/manifest-template.yaml", "Pod manifest template for VMs")
	dockerRegistry := flag.String("docker-registry", "kubevirt", "Organization or private docker registry URL")
	launcherImage := flag.String("launcher-image", "virt-launcher", "Shim container for containerized VMs")
	apiServer := flag.String("api-server", "127.0.0.1:8080", "Kubernetes api server location")

	logger := log.NewLogfmtLogger(os.Stderr)
	flag.Parse()
	ctx := context.Background()

	var g inject.Graph

	vmService := services.NewVMService(logger)
	templateService, err := services.NewTemplateService(logger, *templateFile, *dockerRegistry, *launcherImage)
	if err != nil {
		golog.Fatal(err)
	}

	g.Provide(
		&inject.Object{Value: services.NewKubeService(logger, *apiServer)},
		&inject.Object{Value: templateService},
		&inject.Object{Value: vmService},
	)

	g.Populate()

	handlers := rest.Handlers{
		RawDomainHandler: endpoints.MakeRawDomainHandler(ctx, middleware.InternalErrorMiddleware(logger)(endpoints.MakeRawDomainEndpoint(vmService))),
	}

	http.Handle("/", rest.DefineRoutes(&handlers))
	httpLogger := levels.New(logger).With("component", "http")
	httpLogger.Info().Log("action", "listening", "interface", *host, "port", *port)
	if err := http.ListenAndServe(*host+":"+strconv.Itoa(*port), nil); err != nil {
		golog.Fatal(err)
	}
}
