package rest

import (
	"fmt"
	"github.com/emicklei/go-restful"
	"github.com/go-kit/kit/endpoint"
	"golang.org/x/net/context"
	metav1 "k8s.io/client-go/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/runtime"
	"k8s.io/client-go/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/middleware"
	mime "kubevirt.io/kubevirt/pkg/rest"
	"kubevirt.io/kubevirt/pkg/rest/endpoints"
	"net/http"
	"reflect"
	"strings"
)

type ResponseHandlerFunc func(rest.Result) (interface{}, error)

func AddGenericResourceProxy(ws *restful.WebService, ctx context.Context, gvr schema.GroupVersionResource, objPointer runtime.Object, objListPointer runtime.Object, response ResponseHandlerFunc) error {
	cli, err := kubecli.GetRESTClient()
	if err != nil {
		return err
	}
	// We don't have to set root here, since the whole webservice has that prefix:
	// ws.Path(GroupVersionBasePath(gvr.GroupVersion()))

	objExample := reflect.ValueOf(objPointer).Elem().Interface()
	listExample := reflect.ValueOf(objListPointer).Elem().Interface()

	delete := endpoints.NewHandlerBuilder().Delete().Endpoint(NewGenericDeleteEndpoint(cli, gvr, response)).Build(ctx)
	put := endpoints.NewHandlerBuilder().Put(objPointer).Endpoint(NewGenericPutEndpoint(cli, gvr, response)).Build(ctx)
	post := endpoints.NewHandlerBuilder().Post(objPointer).Endpoint(NewGenericPostEndpoint(cli, gvr, response)).Build(ctx)
	get := endpoints.NewHandlerBuilder().Get().Endpoint(NewGenericGetEndpoint(cli, gvr, response)).Build(ctx)

	ws.Route(addPostParams(
		ws.POST(ResourcePath(gvr)).
			Produces(mime.MIME_JSON, mime.MIME_YAML).
			Consumes(mime.MIME_JSON, mime.MIME_YAML).
			To(endpoints.MakeGoRestfulWrapper(post)).Reads(objExample).Writes(objExample), ws,
	))

	ws.Route(addPutParams(
		ws.PUT(ResourcePath(gvr)).
			Produces(mime.MIME_JSON, mime.MIME_YAML).
			Consumes(mime.MIME_JSON, mime.MIME_YAML).
			To(endpoints.MakeGoRestfulWrapper(put)).Reads(objExample).Writes(objExample).Doc("test2"), ws,
	))

	ws.Route(addDeleteParams(
		ws.DELETE(ResourcePath(gvr)).
			Produces(mime.MIME_JSON, mime.MIME_YAML).
			Consumes(mime.MIME_JSON, mime.MIME_YAML).
			To(endpoints.MakeGoRestfulWrapper(delete)).Writes(metav1.Status{}).Doc("test3"), ws,
	))

	ws.Route(addGetParams(
		ws.GET(ResourcePath(gvr)).
			Produces(mime.MIME_JSON, mime.MIME_YAML).
			To(endpoints.MakeGoRestfulWrapper(get)).Writes(objExample).Doc("test4"), ws,
	))

	// TODO, implement watch. For now it is here to provide swagger doc only
	ws.Route(addWatchGetListParams(
		ws.GET("/watch/"+gvr.Resource).
			Produces(mime.MIME_JSON).
			To(NotImplementedYet).Writes(objExample), ws,
	))

	ws.Route(addWatchGetListParams(
		ws.GET("/watch"+ResourceBasePath(gvr)).
			Produces(mime.MIME_JSON).
			To(NotImplementedYet).Writes(objExample), ws,
	))

	// TODO List all vms in namespace
	ws.Route(addWatchGetListParams(
		ws.GET(ResourceBasePath(gvr)).
			Produces(mime.MIME_JSON).
			Writes(listExample).
			To(NotImplementedYet).Writes(objExample), ws,
	))

	// TODO Delete all vms in namespace
	ws.Route(addDeleteListParams(
		ws.DELETE(ResourceBasePath(gvr)).
			Produces(mime.MIME_JSON).
			To(NotImplementedYet).Writes(objExample), ws,
	))
	return nil
}

func addWatchGetListParams(builder *restful.RouteBuilder, ws *restful.WebService) *restful.RouteBuilder {
	return builder.Param(NamespaceParam(ws)).Param(fieldSelectorParam(ws)).Param(labelSelectorParam(ws)).
		Param(ws.QueryParameter("resourceVersion", "When specified with a watch call, shows changes that occur after that particular version of a resource. Defaults to changes from the beginning of history.")).
		Param(ws.QueryParameter("timeoutSeconds", "Timeout for the list/watch call.").DataType("int"))
}

func addDeleteListParams(builder *restful.RouteBuilder, ws *restful.WebService) *restful.RouteBuilder {
	return builder.Param(NameParam(ws)).Param(fieldSelectorParam(ws)).Param(labelSelectorParam(ws))
}

func addGetParams(builder *restful.RouteBuilder, ws *restful.WebService) *restful.RouteBuilder {
	return builder.Param(NamespaceParam(ws)).Param(NameParam(ws)).
		Param(ws.QueryParameter("export", "Should this value be exported. Export strips fields that a user can not specify.").DataType("boolean")).
		Param(ws.QueryParameter("exact", "Should the export be exact. Exact export maintains cluster-specific fields like 'Namespace'").DataType("boolean"))
}

func addPostParams(builder *restful.RouteBuilder, ws *restful.WebService) *restful.RouteBuilder {
	return builder.Param(NamespaceParam(ws))
}

func addPutParams(builder *restful.RouteBuilder, ws *restful.WebService) *restful.RouteBuilder {
	return builder.Param(NamespaceParam(ws)).Param(NameParam(ws))
}

func addDeleteParams(builder *restful.RouteBuilder, ws *restful.WebService) *restful.RouteBuilder {
	return builder.Param(NamespaceParam(ws)).Param(NameParam(ws))
}

func NameParam(ws *restful.WebService) *restful.Parameter {
	return ws.PathParameter("name", "Name of the resource").Required(true)
}

func NamespaceParam(ws *restful.WebService) *restful.Parameter {
	return ws.PathParameter("namespace", "Object name and auth scope, such as for teams and projects").Required(true)
}

func labelSelectorParam(ws *restful.WebService) *restful.Parameter {
	return ws.QueryParameter("labelSelector", "A selector to restrict the list of returned objects by their labels. Defaults to everything")
}

func fieldSelectorParam(ws *restful.WebService) *restful.Parameter {
	return ws.QueryParameter("fieldSelector", "A selector to restrict the list of returned objects by their fields. Defaults to everything.")
}

func NewGenericDeleteEndpoint(cli *rest.RESTClient, gvr schema.GroupVersionResource, response ResponseHandlerFunc) endpoint.Endpoint {
	return func(ctx context.Context, payload interface{}) (interface{}, error) {
		metadata := payload.(*endpoints.Metadata)
		result := cli.Delete().Namespace(metadata.Namespace).Resource(gvr.Resource).Name(metadata.Name).Do()
		return response(result)
	}
}

func NewGenericPutEndpoint(cli *rest.RESTClient, gvr schema.GroupVersionResource, response ResponseHandlerFunc) endpoint.Endpoint {
	return func(ctx context.Context, payload interface{}) (interface{}, error) {
		obj := payload.(*endpoints.PutObject)
		result := cli.Put().Namespace(obj.Metadata.Namespace).Resource(gvr.Resource).Name(obj.Metadata.Name).Body(obj.Payload).Do()
		return response(result)
	}
}

func NewGenericPostEndpoint(cli *rest.RESTClient, gvr schema.GroupVersionResource, response ResponseHandlerFunc) endpoint.Endpoint {
	return func(ctx context.Context, payload interface{}) (interface{}, error) {
		obj := payload.(*endpoints.PutObject)
		result := cli.Post().Namespace(obj.Metadata.Namespace).Resource(gvr.Resource).Body(obj.Payload).Do()
		return response(result)
	}
}

func NewGenericGetEndpoint(cli *rest.RESTClient, gvr schema.GroupVersionResource, response ResponseHandlerFunc) endpoint.Endpoint {
	return func(ctx context.Context, payload interface{}) (interface{}, error) {
		metadata := payload.(*endpoints.Metadata)
		result := cli.Get().Namespace(metadata.Namespace).Resource(gvr.Resource).Name(metadata.Name).Do()
		return response(result)
	}
}

func NotImplementedYet(request *restful.Request, response *restful.Response) {
	response.AddHeader("Content-Type", "text/plain")
	response.WriteHeader(http.StatusInternalServerError)
	response.Write([]byte("Not implemented yet, use the native apiserver endpoint."))

}

//FIXME this is basically one big workaround because version and kind are not filled by the restclient
func NewResponseHandler(gvk schema.GroupVersionKind, ptr runtime.Object) ResponseHandlerFunc {
	return func(result rest.Result) (interface{}, error) {
		obj, err := result.Get()
		if err != nil {
			return middleware.NewKubernetesError(result), nil
		}
		if reflect.TypeOf(obj).Elem() == reflect.TypeOf(ptr).Elem() {
			obj.(runtime.Object).GetObjectKind().SetGroupVersionKind(gvk)
		}
		return obj, nil

	}
}

func GroupVersionBasePath(gvr schema.GroupVersion) string {
	return fmt.Sprintf("/apis/%s/%s", gvr.Group, gvr.Version)
}

func ResourceBasePath(gvr schema.GroupVersionResource) string {
	return fmt.Sprintf("/namespaces/{namespace}/%s", gvr.Resource)
}

func ResourcePath(gvr schema.GroupVersionResource) string {
	return fmt.Sprintf("/namespaces/{namespace}/%s/{name}", gvr.Resource)
}

func SubResourcePath(subResource string) string {
	if !strings.HasPrefix(subResource, "/") {
		return "/" + subResource
	}
	return subResource
}
