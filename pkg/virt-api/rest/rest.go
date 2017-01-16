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

func NewSpiceSubResourceEndpoint(cli *rest.RESTClient, coreCli *kubernetes.Clientset, gvr schema.GroupVersionResource) endpoint.Endpoint {
	return func(ctx context.Context, payload interface{}) (interface{}, error) {
		metadata := payload.(*endpoints.Metadata)
		result := cli.Get().Namespace(metadata.Namespace).Resource(gvr.Resource).Name(metadata.Name).Do()
		if result.Error() != nil {
			return nil, middleware.NewInternalServerError(result.Error())
		}
		obj, err := result.Get()
		if err != nil {
			return nil, middleware.NewInternalServerError(result.Error())
		}

		vm := obj.(*v1.VM)

		if vm.Status.Phase != v1.Running {
			return nil, middleware.NewResourceNotFoundError("VM is not running")
		}

		// TODO allow specifying the spice device. For now select the first one.
		for _, d := range vm.Spec.Domain.Devices.Graphics {
			if strings.ToLower(d.Type) == "spice" {
				port := d.Port
				podList, err := coreCli.Core().Pods(api.NamespaceDefault).List(unfinishedVMPodSelector(vm))
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

				config := "[virt-viewer]\n" +
					"type=spice\n" +
					fmt.Sprintf("host=%s\n", ip) +
					fmt.Sprintf("port=%d\n", port)

				if len(spiceProxy) > 0 {
					config = config + fmt.Sprintf("proxy=http://%s\n", spiceProxy)
				}
				return config, nil
			}
		}

		return nil, middleware.NewResourceNotFoundError("No spice device attached to the VM found.")
	}
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
