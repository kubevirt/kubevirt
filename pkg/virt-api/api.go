package virt_api

import (
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/cert/triple"

	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful-openapi"
	openapispec "github.com/go-openapi/spec"
	flag "github.com/spf13/pflag"
	"golang.org/x/net/context"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/healthz"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/rest/filter"
	"kubevirt.io/kubevirt/pkg/service"
	"kubevirt.io/kubevirt/pkg/virt-api/rest"
)

const (
	// Default port that virt-api listens on.
	defaultPort = 443

	// Default address that virt-api listens on.
	defaultHost = "0.0.0.0"
)

type VirtAPIApp struct {
	service.ServiceListen
	SwaggerUI string
	virtCli   kubecli.KubevirtClient
}

var _ service.Service = &VirtAPIApp{}

func (app *VirtAPIApp) Compose() {
	ctx := context.Background()
	vmGVR := schema.GroupVersionResource{Group: v1.GroupVersion.Group, Version: v1.GroupVersion.Version, Resource: "virtualmachines"}
	vmrsGVR := schema.GroupVersionResource{Group: v1.GroupVersion.Group, Version: v1.GroupVersion.Version, Resource: "virtualmachinereplicasets"}
	vmpGVR := schema.GroupVersionResource{Group: v1.GroupVersion.Group, Version: v1.GroupVersion.Version, Resource: "virtualmachinepresets"}
	subresourcesvmGVR := schema.GroupVersionResource{Group: v1.SubresourceGroupVersion.Group, Version: v1.SubresourceGroupVersion.Version, Resource: "virtualmachines"}

	virtCli, err := kubecli.GetKubevirtClient()
	if err != nil {
		panic(err)
	}

	app.virtCli = virtCli

	ws, err := rest.GroupVersionProxyBase(ctx, v1.GroupVersion)
	if err != nil {
		panic(err)
	}

	ws, err = rest.GenericResourceProxy(ws, ctx, vmGVR, &v1.VirtualMachine{}, v1.VirtualMachineGroupVersionKind.Kind, &v1.VirtualMachineList{})
	if err != nil {
		panic(err)
	}

	ws, err = rest.GenericResourceProxy(ws, ctx, vmrsGVR, &v1.VirtualMachineReplicaSet{}, v1.VMReplicaSetGroupVersionKind.Kind, &v1.VirtualMachineReplicaSetList{})
	if err != nil {
		panic(err)
	}

	ws, err = rest.GenericResourceProxy(ws, ctx, vmpGVR, &v1.VirtualMachinePreset{}, v1.VirtualMachineGroupVersionKind.Kind, &v1.VirtualMachinePresetList{})
	if err != nil {
		log.Fatal(err)
	}

	subws := new(restful.WebService)
	subws.Doc("The KubeVirt API, a virtual machine management.")
	subws.Path(rest.GroupVersionBasePath(v1.SubresourceGroupVersion))

	subresourceApp := &rest.SubresourceAPIApp{
		VirtCli: app.virtCli,
	}
	subws.Route(subws.GET(rest.ResourcePath(subresourcesvmGVR) + rest.SubResourcePath("console")).
		To(subresourceApp.ConsoleRequestHandler).
		Param(rest.NamespaceParam(subws)).Param(rest.NameParam(subws)).
		Operation("console").
		Doc("Open a websocket connection to a serial console on the specified VM."))

	subws.Route(subws.GET(rest.ResourcePath(subresourcesvmGVR) + rest.SubResourcePath("vnc")).
		To(subresourceApp.VNCRequestHandler).
		Param(rest.NamespaceParam(subws)).Param(rest.NameParam(subws)).
		Operation("vnc").
		Doc("Open a websocket connection to connect to VNC on the specified VM."))

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
		panic(err)
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
		errors <- http.ListenAndServeTLS(app.BindAddress+":"+"8443", dir+"/cert.pem", dir+"/key.pem", nil)
	}()
	panic(<-errors)
	//	panic(http.ListenAndServe(app.Address(), nil))
}

func (app *VirtAPIApp) AddFlags() {
	app.InitFlags()

	app.BindAddress = defaultHost
	app.Port = defaultPort

	app.AddCommonFlags()

	flag.StringVar(&app.SwaggerUI, "swagger-ui", "third_party/swagger-ui",
		"swagger-ui location")
}
