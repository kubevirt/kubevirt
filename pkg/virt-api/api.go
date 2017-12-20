package virt_api

import (
	"log"
	"net/http"

	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful-openapi"
	kithttp "github.com/go-kit/kit/transport/http"
	openapispec "github.com/go-openapi/spec"
	flag "github.com/spf13/pflag"
	"golang.org/x/net/context"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"io/ioutil"
	"os"

	"k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/cert/triple"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/healthz"
	"kubevirt.io/kubevirt/pkg/kubecli"
	mime "kubevirt.io/kubevirt/pkg/rest"
	"kubevirt.io/kubevirt/pkg/rest/endpoints"
	"kubevirt.io/kubevirt/pkg/rest/filter"
	"kubevirt.io/kubevirt/pkg/service"
	"kubevirt.io/kubevirt/pkg/virt-api/rest"
)

const (
	// Default port that virt-api listens on.
	defaultPort = 8183

	// Default address that virt-api listens on.
	defaultHost = "0.0.0.0"
)

type VirtAPIApp struct {
	service.ServiceListen
	SwaggerUI string
}

var _ service.Service = &VirtAPIApp{}

func (app *VirtAPIApp) Compose() {
	ctx := context.Background()
	vmGVR := schema.GroupVersionResource{Group: v1.GroupVersion.Group, Version: v1.GroupVersion.Version, Resource: "virtualmachines"}
	subresourcesvmGVR := schema.GroupVersionResource{Group: "subresources.kubevirt.io", Version: v1.GroupVersion.Version, Resource: "virtualmachines"}
	migrationGVR := schema.GroupVersionResource{Group: v1.GroupVersion.Group, Version: v1.GroupVersion.Version, Resource: "migrations"}
	vmrsGVR := schema.GroupVersionResource{Group: v1.GroupVersion.Group, Version: v1.GroupVersion.Version, Resource: "virtualmachinereplicasets"}

	ws, err := rest.GroupVersionProxyBase(ctx, v1.GroupVersion)
	if err != nil {
		log.Fatal(err)
	}

	ws, err = rest.GenericResourceProxy(ws, ctx, vmGVR, &v1.VirtualMachine{}, v1.VirtualMachineGroupVersionKind.Kind, &v1.VirtualMachineList{})
	if err != nil {
		log.Fatal(err)
	}

	ws, err = rest.GenericResourceProxy(ws, ctx, migrationGVR, &v1.Migration{}, v1.MigrationGroupVersionKind.Kind, &v1.MigrationList{})
	if err != nil {
		log.Fatal(err)
	}

	ws, err = rest.GenericResourceProxy(ws, ctx, vmrsGVR, &v1.VirtualMachineReplicaSet{}, v1.VMReplicaSetGroupVersionKind.Kind, &v1.VirtualMachineReplicaSetList{})
	if err != nil {
		log.Fatal(err)
	}

	virtCli, err := kubecli.GetKubevirtClient()
	if err != nil {
		log.Fatal(err)
	}

	//  TODO, allow Encoder and Decoders per type and combine the endpoint logic
	spice := endpoints.MakeGoRestfulWrapper(endpoints.NewHandlerBuilder().Get().
		Endpoint(rest.NewSpiceEndpoint(virtCli.RestClient(), vmGVR)).Encoder(
		endpoints.NewMimeTypeAwareEncoder(endpoints.NewEncodeINIResponse(http.StatusOK),
			map[string]kithttp.EncodeResponseFunc{
				mime.MIME_INI:  endpoints.NewEncodeINIResponse(http.StatusOK),
				mime.MIME_JSON: endpoints.NewEncodeJsonResponse(http.StatusOK),
				mime.MIME_YAML: endpoints.NewEncodeYamlResponse(http.StatusOK),
			})).Build(ctx))

	subws, err := rest.GroupVersionProxyBase(ctx, subresourcesvmGVR.GroupVersion())
	if err != nil {
		log.Fatal(err)
	}

	subws.Route(subws.GET(rest.ResourcePath(subresourcesvmGVR)+rest.SubResourcePath("spice")).
		To(spice).Produces(mime.MIME_INI, mime.MIME_JSON, mime.MIME_YAML).
		Param(rest.NamespaceParam(subws)).Param(rest.NameParam(subws)).
		Operation("spice").
		Doc("Returns a remote-viewer configuration file. Run `man 1 remote-viewer` to learn more about the configuration format.").
		Returns(http.StatusOK, "remote-viewer configuration file" /*os.File{}*/, nil))
	// TODO: That os.File doesn't work as I expect.
	// I need end up with response_type="file", but I am getting response_type="os.File"

	subws.Route(subws.GET(rest.ResourcePath(subresourcesvmGVR) + rest.SubResourcePath("console")).
		To(rest.NewConsoleResource(virtCli, virtCli.CoreV1()).Console).
		Param(restful.QueryParameter("console", "Name of the serial console to connect to")).
		Param(rest.NamespaceParam(subws)).Param(rest.NameParam(subws)).
		Operation("console").
		Doc("Open a websocket connection to a serial console on the specified VM."))
	// TODO: Add 'Returns', but I don't know what return type to put there.

	restful.Add(ws)
	restful.Add(subws)

	ws.Route(ws.GET("/healthz").
		To(healthz.KubeConnectionHealthzFunc).
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON).
		Operation("checkHealth").
		Doc("Health endpoint").
		Returns(http.StatusOK, "OK", nil).
		Returns(http.StatusInternalServerError, "Unhealthy", nil))
	ws, err = rest.ResourceProxyAutodiscovery(ctx, vmGVR)
	if err != nil {
		log.Fatal(err)
	}

	restful.Add(ws)

	restful.Filter(filter.RequestLoggingFilter())
	restful.Filter(restful.OPTIONSFilter())
}

func (app *VirtAPIApp) ConfigureOpenAPIService() {
	restful.DefaultContainer.Add(restfulspec.NewOpenAPIService(CreateOpenAPIConfig()))
	http.Handle("/swagger-ui/", http.StripPrefix("/swagger-ui/", http.FileServer(http.Dir(app.SwaggerUI))))
}

func CreateOpenAPIConfig() restfulspec.Config {
	return restfulspec.Config{
		WebServices:    restful.RegisteredWebServices(),
		WebServicesURL: "",
		APIPath:        "/swaggerapi",
		PostBuildSwaggerObjectHandler: addInfoToSwaggerObject,
	}
}

func addInfoToSwaggerObject(swo *openapispec.Swagger) {
	swo.Info = &openapispec.Info{
		InfoProps: openapispec.InfoProps{
			Title:       "KubeVirt API",
			Description: "This is KubeVirt API an add-on for Kubernetes.",
			Contact: &openapispec.ContactInfo{
				Name:  "kubevirt-dev",
				Email: "kubevirt-dev@googlegroups.com",
				URL:   "https://github.com/kubevirt/kubevirt",
			},
			License: &openapispec.License{
				Name: "Apache 2.0",
				URL:  "https://www.apache.org/licenses/LICENSE-2.0",
			},
		},
	}
}

func (app *VirtAPIApp) Run() {

	caKeyPair, _ := triple.NewCA("kubevirt.io")
	keyPair, _ := triple.NewServerKeyPair(
		caKeyPair,
		"virt-api.kube-system.pod.cluster.local",
		"virt-api",
		"kube-system",
		"cluster.local",
		nil,
		nil,
	)
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	ioutil.WriteFile(dir+"/key.pem", cert.EncodePrivateKeyPEM(keyPair.Key), 0600)
	ioutil.WriteFile(dir+"/cert.pem", cert.EncodeCertPEM(keyPair.Cert), 0600)

	errors := make(chan error)

	go func() {
		errors <- http.ListenAndServe(app.Address(), nil)
	}()

	go func() {
		errors <- http.ListenAndServeTLS(app.BindAddress+":"+"8443", dir+"/cert.pem", dir+"/key.pem", nil)
	}()
	log.Fatal(<-errors)
}

func (app *VirtAPIApp) AddFlags() {
	app.InitFlags()

	app.BindAddress = defaultHost
	app.Port = defaultPort

	app.AddCommonFlags()

	flag.StringVar(&app.SwaggerUI, "swagger-ui", "third_party/swagger-ui",
		"swagger-ui location")
}
