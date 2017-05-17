package rest

import (
	"encoding/json"
	"fmt"
	"github.com/asaskevich/govalidator"
	"github.com/emicklei/go-restful"
	"github.com/evanphx/json-patch"
	"github.com/go-kit/kit/endpoint"
	"golang.org/x/net/context"
	"k8s.io/client-go/pkg/api"
	metav1 "k8s.io/client-go/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/fields"
	"k8s.io/client-go/pkg/labels"
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
	"time"
)

type ResponseHandlerFunc func(rest.Result) (interface{}, error)

func GroupVersionProxyBase(ctx context.Context, gv schema.GroupVersion) (*restful.WebService, error) {
	ws := new(restful.WebService)
	ws.Path(GroupVersionBasePath(gv))

	cli, err := kubecli.GetRESTClient()
	if err != nil {
		return nil, err
	}
	autodiscover := endpoints.NewHandlerBuilder().Get().Decoder(endpoints.NoopDecoder).Endpoint(NewAutodiscoveryEndpoint(cli)).Build(ctx)
	ws.Route(ws.GET("/").Produces(mime.MIME_JSON).
		Returns(http.StatusOK, "OK", metav1.APIResourceList{}).
		Returns(http.StatusNotFound, "Not Found", nil).
		Writes(metav1.APIResourceList{}).To(endpoints.MakeGoRestfulWrapper(autodiscover)))
	return ws, nil
}

func GenericResourceProxy(ws *restful.WebService, ctx context.Context, gvr schema.GroupVersionResource, objPointer runtime.Object, objKind string, objListPointer runtime.Object) (*restful.WebService, error) {

	objResponseHandler := newResponseHandler(schema.GroupVersionKind{Group: gvr.Group, Version: gvr.Version, Kind: objKind}, objPointer)
	objListResponseHandler := newResponseHandler(schema.GroupVersionKind{Group: gvr.Group, Version: gvr.Version, Kind: objKind + "List"}, objListPointer)
	cli, err := kubecli.GetRESTClient()
	if err != nil {
		return nil, err
	}

	objExample := reflect.ValueOf(objPointer).Elem().Interface()
	listExample := reflect.ValueOf(objListPointer).Elem().Interface()

	delete := endpoints.NewHandlerBuilder().Delete().Endpoint(NewGenericDeleteEndpoint(cli, gvr, newStatusResponseHandler())).Build(ctx)
	put := endpoints.NewHandlerBuilder().Put(objPointer).Endpoint(NewGenericPutEndpoint(cli, gvr, objResponseHandler)).Build(ctx)
	patch := endpoints.NewHandlerBuilder().Patch().Endpoint(NewGenericPatchEndpoint(cli, gvr, objResponseHandler)).Build(ctx)
	post := endpoints.NewHandlerBuilder().Post(objPointer).Endpoint(NewGenericPostEndpoint(cli, gvr, objResponseHandler)).Build(ctx)
	get := endpoints.NewHandlerBuilder().Get().Endpoint(NewGenericGetEndpoint(cli, gvr, objResponseHandler)).Build(ctx)
	getList := endpoints.NewHandlerBuilder().Get().Endpoint(NewGenericGetListEndpoint(cli, gvr, objListResponseHandler)).Decoder(endpoints.NamespaceDecodeRequestFunc).Build(ctx)
	deleteList := endpoints.NewHandlerBuilder().Delete().Endpoint(NewGenericDeleteListEndpoint(cli, gvr, objListResponseHandler)).Decoder(endpoints.NamespaceDecodeRequestFunc).Build(ctx)

	ws.Route(addPostParams(
		ws.POST(ResourceBasePath(gvr)).
			Produces(mime.MIME_JSON, mime.MIME_YAML).
			Consumes(mime.MIME_JSON, mime.MIME_YAML).
			Returns(http.StatusCreated, "Created", objExample).
			Returns(http.StatusNotFound, "Not Found", nil).
			To(endpoints.MakeGoRestfulWrapper(post)).Reads(objExample).Writes(objExample), ws,
	))

	ws.Route(addPutParams(
		ws.PUT(ResourcePath(gvr)).
			Produces(mime.MIME_JSON, mime.MIME_YAML).
			Consumes(mime.MIME_JSON, mime.MIME_YAML).
			Returns(http.StatusOK, "Updated", objExample).
			Returns(http.StatusNotFound, "Not Found", nil).
			To(endpoints.MakeGoRestfulWrapper(put)).Reads(objExample).Writes(objExample), ws,
	))

	ws.Route(addDeleteParams(
		ws.DELETE(ResourcePath(gvr)).
			Produces(mime.MIME_JSON, mime.MIME_YAML).
			Consumes(mime.MIME_JSON, mime.MIME_YAML).
			Returns(http.StatusNoContent, "Deleted", nil).
			Returns(http.StatusNotFound, "Not Found", nil).
			To(endpoints.MakeGoRestfulWrapper(delete)).Writes(metav1.Status{}), ws,
	))

	ws.Route(addGetParams(
		ws.GET(ResourcePath(gvr)).
			Produces(mime.MIME_JSON, mime.MIME_YAML).
			Returns(http.StatusOK, "OK", objExample).
			Returns(http.StatusNotFound, "Not Found", nil).
			To(endpoints.MakeGoRestfulWrapper(get)).Writes(objExample), ws,
	))

	ws.Route(addPatchParams(
		ws.PATCH(ResourcePath(gvr)).
			Consumes(mime.MIME_JSON_PATCH).
			Produces(mime.MIME_JSON, mime.MIME_YAML).
			Returns(http.StatusOK, "OK", objExample).
			Returns(http.StatusNotFound, "Not Found", nil).
			To(endpoints.MakeGoRestfulWrapper(patch)).Writes(objExample), ws,
	))

	// TODO, implement watch. For now it is here to provide swagger doc only
	ws.Route(addNotNamespacedWatchGetListParams(
		ws.GET("/watch/"+gvr.Resource).
			Produces(mime.MIME_JSON).
			Returns(http.StatusOK, "OK", objExample).
			Returns(http.StatusNotFound, "Not Found", nil).
			To(NotImplementedYet).Writes(objExample), ws,
	))

	// TODO, implement watch. For now it is here to provide swagger doc only
	ws.Route(addWatchGetListParams(
		ws.GET("/watch"+ResourceBasePath(gvr)).
			Returns(http.StatusOK, "OK", objExample).
			Returns(http.StatusNotFound, "Not Found", nil).
			Produces(mime.MIME_JSON).
			To(NotImplementedYet).Writes(objExample), ws,
	))

	ws.Route(addWatchGetListParams(
		ws.GET(ResourceBasePath(gvr)).
			Produces(mime.MIME_JSON, mime.MIME_YAML).
			Returns(http.StatusOK, "OK", listExample).
			Returns(http.StatusNotFound, "Not Found", nil).
			Writes(listExample).
			To(endpoints.MakeGoRestfulWrapper(getList)), ws,
	))

	ws.Route(addDeleteListParams(
		ws.DELETE(ResourceBasePath(gvr)).
			Returns(http.StatusOK, "OK", listExample).
			Returns(http.StatusNotFound, "Not Found", nil).
			Produces(mime.MIME_JSON, mime.MIME_YAML).
			To(endpoints.MakeGoRestfulWrapper(deleteList)).Writes(listExample), ws,
	))

	return ws, nil
}

func ResourceProxyAutodiscovery(ctx context.Context, gvr schema.GroupVersionResource) (*restful.WebService, error) {
	cli, err := kubecli.GetRESTClient()
	if err != nil {
		return nil, err
	}
	autodiscover := endpoints.NewHandlerBuilder().Get().Decoder(endpoints.NoopDecoder).Endpoint(NewAutodiscoveryEndpoint(cli)).Build(ctx)
	ws := new(restful.WebService)
	ws.Route(ws.GET(GroupBasePath(gvr.GroupVersion())).Produces(mime.MIME_JSON).
		Returns(http.StatusOK, "OK", metav1.APIGroup{}).
		Returns(http.StatusNotFound, "Not Found", nil).
		Writes(metav1.APIGroup{}).To(endpoints.MakeGoRestfulWrapper(autodiscover)))
	return ws, nil
}

func addNotNamespacedWatchGetListParams(builder *restful.RouteBuilder, ws *restful.WebService) *restful.RouteBuilder {
	return builder.Param(fieldSelectorParam(ws)).Param(labelSelectorParam(ws)).
		Param(ws.QueryParameter("resourceVersion", "When specified with a watch call, shows changes that occur after that particular version of a resource. Defaults to changes from the beginning of history.")).
		Param(ws.QueryParameter("timeoutSeconds", "TimeoutSeconds for the list/watch call.").DataType("integer"))
}

func addWatchGetListParams(builder *restful.RouteBuilder, ws *restful.WebService) *restful.RouteBuilder {
	return builder.Param(NamespaceParam(ws)).Param(fieldSelectorParam(ws)).Param(labelSelectorParam(ws)).
		Param(ws.QueryParameter("resourceVersion", "When specified with a watch call, shows changes that occur after that particular version of a resource. Defaults to changes from the beginning of history.")).
		Param(ws.QueryParameter("timeoutSeconds", "TimeoutSeconds for the list/watch call.").DataType("integer"))
}

func addDeleteListParams(builder *restful.RouteBuilder, ws *restful.WebService) *restful.RouteBuilder {
	return builder.Param(NamespaceParam(ws)).Param(fieldSelectorParam(ws)).Param(labelSelectorParam(ws))
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

func addPatchParams(builder *restful.RouteBuilder, ws *restful.WebService) *restful.RouteBuilder {
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

func NewGenericGetListEndpoint(cli *rest.RESTClient, gvr schema.GroupVersionResource, response ResponseHandlerFunc) endpoint.Endpoint {
	return func(ctx context.Context, payload interface{}) (interface{}, error) {
		metadata := payload.(*endpoints.Metadata)
		listOptions, err := listOptionsFromMetadata(metadata)
		if err != nil {
			return middleware.NewBadRequestError(err.Error()), nil
		}
		result := cli.Get().Namespace(metadata.Namespace).
			FieldsSelectorParam(listOptions.FieldSelector).
			LabelsSelectorParam(listOptions.LabelSelector).
			Timeout(time.Duration(*listOptions.TimeoutSeconds) * time.Second).
			Resource(gvr.Resource).Do()
		return response(result)
	}
}

func NewGenericDeleteListEndpoint(cli *rest.RESTClient, gvr schema.GroupVersionResource, response ResponseHandlerFunc) endpoint.Endpoint {
	return func(ctx context.Context, payload interface{}) (interface{}, error) {
		metadata := payload.(*endpoints.Metadata)
		listOptions, err := listOptionsFromMetadata(metadata)
		if err != nil {
			return middleware.NewBadRequestError(err.Error()), nil
		}
		result := cli.Delete().Namespace(metadata.Namespace).
			FieldsSelectorParam(listOptions.FieldSelector).
			LabelsSelectorParam(listOptions.LabelSelector).
			Timeout(time.Duration(*listOptions.TimeoutSeconds) * time.Second).
			Resource(gvr.Resource).Do()
		return response(result)
	}
}

func listOptionsFromMetadata(metadata *endpoints.Metadata) (*api.ListOptions, error) {
	listOptions := &api.ListOptions{}
	if metadata.Headers.FieldSelector != "" {
		fieldSelector, err := fields.ParseSelector(metadata.Headers.FieldSelector)
		if err != nil {
			return nil, err
		}
		listOptions.FieldSelector = fieldSelector
	}
	if metadata.Headers.LabelSelector != "" {
		labelSelector, err := labels.Parse(metadata.Headers.LabelSelector)
		if err != nil {
			return nil, err
		}
		listOptions.LabelSelector = labelSelector
	}

	listOptions.ResourceVersion = metadata.Headers.ResourceVersion
	listOptions.TimeoutSeconds = &metadata.Headers.TimeoutSeconds
	return listOptions, nil
}

func NewGenericPutEndpoint(cli *rest.RESTClient, gvr schema.GroupVersionResource, response ResponseHandlerFunc) endpoint.Endpoint {
	return func(ctx context.Context, payload interface{}) (interface{}, error) {
		obj := payload.(*endpoints.PutObject)
		result := cli.Put().Namespace(obj.Metadata.Namespace).Resource(gvr.Resource).Name(obj.Metadata.Name).Body(obj.Payload).Do()
		return response(result)
	}
}

func NewGenericPatchEndpoint(cli *rest.RESTClient, gvr schema.GroupVersionResource, response ResponseHandlerFunc) endpoint.Endpoint {
	return func(ctx context.Context, payload interface{}) (interface{}, error) {
		obj := payload.(*endpoints.PatchObject)
		result := cli.Get().Namespace(obj.Metadata.Namespace).Resource(gvr.Resource).Name(obj.Metadata.Name).Do()
		if result.Error() != nil {
			return middleware.NewKubernetesError(result), nil
		}
		// Check if we can deserialize into something we expected
		originalBody, err := result.Get()
		if err != nil {
			return middleware.NewKubernetesError(result), nil
		}

		patchedBody, err := patchJson(obj.PatchType, obj.Patch, originalBody)
		if err != nil {
			return err, nil
		}

		ok, err := govalidator.ValidateStruct(patchedBody)
		if !ok {
			return middleware.NewUnprocessibleEntityError(err), nil
		}

		result = cli.Put().Namespace(obj.Metadata.Namespace).Resource(gvr.Resource).Name(obj.Metadata.Name).Body(patchedBody).Do()
		return response(result)
	}
}

func patchJson(patchType api.PatchType, patch interface{}, orig runtime.Object) (runtime.Object, error) {

	var rawPatched []byte
	rawOriginal, err := json.Marshal(orig)
	if err != nil {
		return nil, middleware.NewInternalServerError(err)
	}

	rawPatch, err := json.Marshal(patch)
	if err != nil {
		return nil, middleware.NewInternalServerError(err)
	}

	switch patchType {
	case api.MergePatchType:
		if rawPatched, err = jsonpatch.MergePatch(rawOriginal, rawPatch); err != nil {
			return nil, middleware.NewUnprocessibleEntityError(err)
		}
	case api.JSONPatchType:
		p, err := jsonpatch.DecodePatch(rawPatch)
		if err != nil {
			return nil, middleware.NewUnprocessibleEntityError(err)
		}
		if rawPatched, err = p.Apply(rawOriginal); err != nil {
			return nil, middleware.NewUnprocessibleEntityError(err)
		}
	default:
		return nil, middleware.NewInternalServerError(fmt.Errorf("Patch type %s is unknown", patchType))
	}
	patchedObj := reflect.New(reflect.TypeOf(orig).Elem()).Interface().(runtime.Object)
	if err = json.Unmarshal(rawPatched, patchedObj); err != nil {
		return nil, middleware.NewUnprocessibleEntityError(err)
	}
	return patchedObj, nil
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
func newResponseHandler(gvk schema.GroupVersionKind, ptr runtime.Object) ResponseHandlerFunc {
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

//FIXME this is basically one big workaround because version and kind are not filled by the restclient
func newStatusResponseHandler() ResponseHandlerFunc {
	return func(result rest.Result) (interface{}, error) {
		obj, err := result.Get()
		if err != nil {
			return middleware.NewKubernetesError(result), nil
		}
		if reflect.TypeOf(obj).Elem() == reflect.TypeOf(metav1.Status{}) {
			obj.(*metav1.Status).Kind = "Status"
			obj.(*metav1.Status).APIVersion = "v1"
		}
		return obj, nil

	}
}

func NewAutodiscoveryEndpoint(cli *rest.RESTClient) endpoint.Endpoint {
	return func(ctx context.Context, _ interface{}) (interface{}, error) {
		request := ctx.Value(endpoints.ReqKey).(*restful.Request)
		result := cli.Get().AbsPath(request.SelectedRoutePath()).SetHeader("Accept", mime.MIME_JSON).Do()
		obj, err := result.Get()
		if err != nil {
			return middleware.NewKubernetesError(result), nil
		}
		return obj, nil
	}
}

func GroupBasePath(gvr schema.GroupVersion) string {
	return fmt.Sprintf("/apis/%s", gvr.Group)
}

func GroupVersionBasePath(gvr schema.GroupVersion) string {
	return fmt.Sprintf("/apis/%s/%s", gvr.Group, gvr.Version)
}

func ResourceBasePath(gvr schema.GroupVersionResource) string {
	return fmt.Sprintf("/namespaces/{namespace:[a-z0-9][a-z0-9\\-]*}/%s", gvr.Resource)
}

func ResourcePath(gvr schema.GroupVersionResource) string {
	return fmt.Sprintf("/namespaces/{namespace:[a-z0-9][a-z0-9\\-]*}/%s/{name:[a-z0-9][a-z0-9\\-]*}", gvr.Resource)
}

func SubResourcePath(subResource string) string {
	if !strings.HasPrefix(subResource, "/") {
		return "/" + subResource
	}
	return subResource
}
