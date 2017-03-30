package main

import (
	"flag"
	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful/swagger"
	kithttp "github.com/go-kit/kit/transport/http"
	"golang.org/x/net/context"
	"k8s.io/client-go/pkg/runtime/schema"
	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/healthz"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/logging"
	mime "kubevirt.io/kubevirt/pkg/rest"
	"kubevirt.io/kubevirt/pkg/rest/endpoints"
	"kubevirt.io/kubevirt/pkg/rest/filter"
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

	ws, err := rest.GroupVersionProxyBase(ctx, v1.GroupVersion)
	if err != nil {
		log.Fatal(err)
	}

	ws, err = rest.GenericResourceProxy(ws, ctx, vmGVR, &v1.VM{}, v1.GroupVersionKind.Kind, &v1.VMList{})
	if err != nil {
		log.Fatal(err)
	}

	ws, err = rest.GenericResourceProxy(ws, ctx, migrationGVR, &v1.Migration{}, "Migration", &v1.MigrationList{})
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
	virtCli, err := kubecli.GetKubevirtClient()
	if err != nil {
		log.Fatal(err)
	}

	//  TODO, allow Encoder and Decoders per type and combine the endpoint logic
	spice := endpoints.MakeGoRestfulWrapper(endpoints.NewHandlerBuilder().Get().
		Endpoint(rest.NewSpiceEndpoint(cli, coreCli, vmGVR)).Encoder(
		endpoints.NewMimeTypeAwareEncoder(endpoints.NewEncodeINIResponse(http.StatusOK),
			map[string]kithttp.EncodeResponseFunc{
				mime.MIME_INI:  endpoints.NewEncodeINIResponse(http.StatusOK),
				mime.MIME_JSON: endpoints.NewEncodeJsonResponse(http.StatusOK),
				mime.MIME_YAML: endpoints.NewEncodeYamlResponse(http.StatusOK),
			})).Build(ctx))

	ws.Route(ws.GET(rest.ResourcePath(vmGVR)+rest.SubResourcePath("spice")).
		To(spice).Produces(mime.MIME_INI, mime.MIME_JSON, mime.MIME_YAML).
		Param(rest.NamespaceParam(ws)).Param(rest.NameParam(ws)).
		Doc("Returns a remote-viewer configuration file. Run `man 1 remote-viewer` to learn more about the configuration format."))

	ws.Route(ws.GET(rest.ResourcePath(vmGVR) + rest.SubResourcePath("console")).
		To(rest.NewConsoleResource(virtCli, coreCli.CoreV1()).Console).
		Param(restful.QueryParameter("console", "Name of the serial console to connect to")).
		Param(rest.NamespaceParam(ws)).Param(rest.NameParam(ws)).
		Doc("Open a websocket connection to a serial console on the specified VM."))

	restful.Add(ws)

	ws.Route(ws.GET("/healthz").To(healthz.KubeConnectionHealthzFunc).Consumes(restful.MIME_JSON).Produces(restful.MIME_JSON).Doc("Health endpoint"))
	ws, err = rest.ResourceProxyAutodiscovery(ctx, vmGVR)
	if err != nil {
		log.Fatal(err)
	}

	restful.Add(ws)

	restful.Filter(filter.RequestLoggingFilter())
	restful.Filter(restful.OPTIONSFilter())

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
