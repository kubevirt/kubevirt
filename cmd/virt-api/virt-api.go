package main

import (
	"flag"
	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful/swagger"
	kithttp "github.com/go-kit/kit/transport/http"
	"golang.org/x/net/context"
	"k8s.io/client-go/pkg/runtime/schema"
	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/logging"
	mime "kubevirt.io/kubevirt/pkg/rest"
	"kubevirt.io/kubevirt/pkg/rest/endpoints"
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
	vmGVR := schema.GroupVersionResource{Group: v1.GroupVersion.Group, Version: v1.GroupVersion.Version, Resource: "vms"}
	migrationGVR := schema.GroupVersionResource{Group: v1.GroupVersion.Group, Version: v1.GroupVersion.Version, Resource: "migrations"}

	// FIXME the whole newResponseHandler is just a hack, see the method itself for details
	err := rest.AddGenericResourceProxy(rest.WebService, ctx, vmGVR, &v1.VM{}, v1.GroupVersionKind.Kind, &v1.VMList{})
	if err != nil {
		log.Fatal(err)
	}
	err = rest.AddGenericResourceProxy(rest.WebService, ctx, migrationGVR, &v1.Migration{}, "Migration", &v1.MigrationList{})
	if err != nil {
		log.Fatal(err)
	}

	cli, err := kubecli.GetRESTClient()
	if err != nil {
		log.Fatal(err)
	}
	coreCli, err := kubecli.Get()
	if err != nil {
		log.Fatal(err)
	}

	//  TODO, allow Encoder and Decoders per type and combine the endpoint logic
	spice := endpoints.MakeGoRestfulWrapper(endpoints.NewHandlerBuilder().Get().
		Endpoint(rest.NewSpiceEndpoint(cli, coreCli, vmGVR)).Encoder(
		endpoints.NewMimeTypeAwareEncoder(endpoints.NewEncodeJsonResponse(http.StatusOK),
			map[string]kithttp.EncodeResponseFunc{
				mime.MIME_INI:  endpoints.NewEncodeINIResponse(http.StatusOK),
				mime.MIME_JSON: endpoints.NewEncodeJsonResponse(http.StatusOK),
				mime.MIME_YAML: endpoints.NewEncodeYamlResponse(http.StatusOK),
			})).Build(ctx))

	rest.WebService.Route(rest.WebService.GET(rest.ResourcePath(vmGVR)+rest.SubResourcePath("spice")).
		To(spice).Produces(mime.MIME_INI, mime.MIME_JSON, mime.MIME_YAML).
		Param(rest.NamespaceParam(rest.WebService)).Param(rest.NameParam(rest.WebService)).
		Doc("Returns a remote-viewer configuration file. Run `man 1 remote-viewer` to learn more about the configuration format."))

	config := swagger.Config{
		WebServices:     restful.RegisteredWebServices(), // you control what services are visible
		WebServicesUrl:  "http://localhost:8183",
		ApiPath:         "/swaggerapi",
		SwaggerPath:     "/swagger-ui/",
		SwaggerFilePath: *swaggerui,
	}
	swagger.InstallSwaggerService(config)

	log.Fatal(http.ListenAndServe(*host+":"+strconv.Itoa(*port), nil))
}
