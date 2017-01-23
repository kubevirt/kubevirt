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
	"reflect"
	"strings"
)

type ResponseHandlerFunc func(rest.Result) (interface{}, error)

func AddGenericResourceProxy(ws *restful.WebService, ctx context.Context, gvr schema.GroupVersionResource, ptr runtime.Object, response ResponseHandlerFunc) error {
	cli, err := kubecli.GetRESTClient()
	if err != nil {
		return err
	}
	// We don't have to set root here, since the whole webservice has that prefix:
	// ws.Path(ResourcePathBase(gvr.GroupVersion()))

	example := reflect.ValueOf(ptr).Elem().Interface()
	delete := endpoints.NewHandlerBuilder().Delete().Endpoint(NewGenericDeleteEndpoint(cli, gvr, response)).Build(ctx)
	put := endpoints.NewHandlerBuilder().Put(ptr).Endpoint(NewGenericPutEndpoint(cli, gvr, response)).Build(ctx)
	post := endpoints.NewHandlerBuilder().Post(ptr).Endpoint(NewGenericPostEndpoint(cli, gvr, response)).Build(ctx)
	get := endpoints.NewHandlerBuilder().Get().Endpoint(NewGenericGetEndpoint(cli, gvr, response)).Build(ctx)

	ws.Route(ws.POST(ResourcePath(gvr)).
		Produces(mime.MIME_JSON, mime.MIME_YAML).
		Consumes(mime.MIME_JSON, mime.MIME_YAML).
		To(endpoints.MakeGoRestfulWrapper(post)).Reads(example).Writes(example))

	ws.Route(ws.PUT(ResourcePath(gvr)).
		Produces(mime.MIME_JSON, mime.MIME_YAML).
		Consumes(mime.MIME_JSON, mime.MIME_YAML).
		To(endpoints.MakeGoRestfulWrapper(put)).Reads(example).Writes(example).Doc("test2"))

	ws.Route(ws.DELETE(ResourcePath(gvr)).
		Produces(mime.MIME_JSON, mime.MIME_YAML).
		Consumes(mime.MIME_JSON, mime.MIME_YAML).
		To(endpoints.MakeGoRestfulWrapper(delete)).Writes(metav1.Status{}).Doc("test3"))

	ws.Route(ws.GET(ResourcePath(gvr)).
		Produces(mime.MIME_JSON, mime.MIME_YAML).
		To(endpoints.MakeGoRestfulWrapper(get)).Writes(example).Doc("test4"))
	return nil
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

func ResourcePathBase(gvr schema.GroupVersion) string {
	return fmt.Sprintf("/apis/%s/%s", gvr.Group, gvr.Version)
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
