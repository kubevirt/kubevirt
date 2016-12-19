package main

import (
	"flag"
	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful/swagger"
	"golang.org/x/net/context"
	"k8s.io/client-go/1.5/pkg/api/unversioned"
	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/logging"
	"kubevirt.io/kubevirt/pkg/virt-api/rest"
	"log"
	"net/http"
	"strconv"
)

func main() {

	logging.InitializeLogging("virt-api")
	swaggerui := flag.String("swagger-ui", "third_party/swagger-ui", "swagger-ui location")
	host := flag.String("listen", "0.0.0.0", "Address and port where to listen on")
	port := flag.Int("port", 8183, "Port to listen on")
	flag.Parse()

	ctx := context.Background()
	gvr := unversioned.GroupVersionResource{Group: v1.GroupVersion.Group, Version: v1.GroupVersion.Version, Resource: "vms"}
	gvk := unversioned.GroupVersionKind{Group: v1.GroupVersion.Group, Version: v1.GroupVersion.Version, Kind: "VM"}

	// FIXME the whole NewResponseHandler is just a hack, see the method itself for details
	err := rest.AddGenericResourceProxy(rest.WebService, ctx, gvr, &v1.VM{}, rest.NewResponseHandler(gvk, &v1.VM{}))
	if err != nil {
		log.Fatal(err)
	}

	config := swagger.Config{
		WebServices:     restful.RegisteredWebServices(), // you control what services are visible
		WebServicesUrl:  "http://localhost:8183",
		ApiPath:         "/apidocs.json",
		SwaggerPath:     "/apidocs/",
		SwaggerFilePath: *swaggerui,
	}
	swagger.InstallSwaggerService(config)

	log.Fatal(http.ListenAndServe(*host+":"+strconv.Itoa(*port), nil))
}
