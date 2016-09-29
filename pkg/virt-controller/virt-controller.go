package main

import (
	"flag"
	"golang.org/x/net/context"
	"net/http"
	"os"
	"strconv"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/levels"

	"kubevirt/core/pkg/virt-controller/endpoints"
	"kubevirt/core/pkg/virt-controller/rest"
	"kubevirt/core/pkg/virt-controller/services"
)

func main() {

	host := flag.String("address", "0.0.0.0", "Address to bind to")
	port := flag.Int("port", 8080, "Port to listen on")

	logger := log.NewLogfmtLogger(os.Stderr)
	flag.Parse()
	ctx := context.Background()
	svc := services.MakeVMService(logger)

	handlers := rest.Handlers{
		RawDomainHandler: endpoints.MakeRawDomainHandler(ctx, endpoints.MakeRawDomainEndpoint(svc)),
	}

	http.Handle("/", rest.DefineRoutes(&handlers))
	httpLogger := levels.New(logger).With("component", "http")
	httpLogger.Info().Log("action", "listening", "interface", *host, "port", *port)
	http.ListenAndServe(*host+":"+strconv.Itoa(*port), nil)
}
