package rest

import (
	"flag"
	"fmt"
	"github.com/emicklei/go-restful"
	"github.com/go-kit/kit/endpoint"
	"golang.org/x/net/context"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api"
	kubev1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/fields"
	"k8s.io/client-go/pkg/labels"
	"k8s.io/client-go/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/healthz"
	"kubevirt.io/kubevirt/pkg/middleware"
	"kubevirt.io/kubevirt/pkg/rest/endpoints"
	"strings"
)

var WebService *restful.WebService
var spiceProxy string

func init() {
	WebService = new(restful.WebService)
	WebService.Path("/").Consumes(restful.MIME_JSON).Produces(restful.MIME_JSON)
	WebService.ApiVersion(v1.GroupVersion.String()).Doc("help")
	restful.Add(WebService)
	WebService.Route(WebService.GET("/apis/" + v1.GroupVersion.String() + "/healthz").To(healthz.KubeConnectionHealthzFunc).Doc("Health endpoint"))
	// TODO should be reloadable, use configmaps and update on every access? Watch a config file and reload?
	flag.StringVar(&spiceProxy, "spice-proxy", "", "Spice proxy to use when spice access is requested")
}

func NewSpiceEndpoint(cli *rest.RESTClient, coreCli *kubernetes.Clientset, gvr schema.GroupVersionResource) endpoint.Endpoint {
	return func(ctx context.Context, payload interface{}) (interface{}, error) {
		metadata := payload.(*endpoints.Metadata)
		obj, err := cli.Get().Namespace(metadata.Namespace).Resource(gvr.Resource).Name(metadata.Name).Do().Get()
		if err != nil {
			return nil, middleware.NewInternalServerError(err)
		}

		vm := obj.(*v1.VM)
		spice, err := spiceFromVM(vm, coreCli)
		if err != nil {
			return nil, err

		}

		return spice, nil
	}
}

func spiceFromVM(vm *v1.VM, coreCli *kubernetes.Clientset) (*v1.Spice, error) {

	if vm.Status.Phase != v1.Running {
		return nil, middleware.NewResourceNotFoundError("VM is not running")
	}

	// TODO allow specifying the spice device. For now select the first one.
	for _, d := range vm.Spec.Domain.Devices.Graphics {
		if strings.ToLower(d.Type) == "spice" {
			port := d.Port
			podList, err := coreCli.CoreV1().Pods(api.NamespaceDefault).List(unfinishedVMPodSelector(vm))
			if err != nil {
				return nil, middleware.NewInternalServerError(err)
			}

			// The pod could just have failed now
			if len(podList.Items) == 0 {
				// TODO is that the right return code?
				return nil, middleware.NewResourceNotFoundError("VM is not running")
			}

			pod := podList.Items[0]
			ip := pod.Status.PodIP

			spice := v1.NewSpice(vm.GetObjectMeta().GetName())
			spice.Info = v1.SpiceInfo{
				Type: "spice",
				Host: ip,
				Port: port,
			}
			if spiceProxy != "" {
				spice.Info.Proxy = fmt.Sprintf("http://%s", spiceProxy)
			}
			return spice, nil
		}
	}

	return nil, middleware.NewResourceNotFoundError("No spice device attached to the VM found.")
}

// TODO for now just copied from VMService
func unfinishedVMPodSelector(vm *v1.VM) kubev1.ListOptions {
	fieldSelector := fields.ParseSelectorOrDie(
		"status.phase!=" + string(kubev1.PodFailed) +
			",status.phase!=" + string(kubev1.PodSucceeded))
	labelSelector, err := labels.Parse(fmt.Sprintf(v1.DomainLabel+" in (%s)", vm.GetObjectMeta().GetName()))
	if err != nil {
		panic(err)
	}
	return kubev1.ListOptions{FieldSelector: fieldSelector.String(), LabelSelector: labelSelector.String()}
}
