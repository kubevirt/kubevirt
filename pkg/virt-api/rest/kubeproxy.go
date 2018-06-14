/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2017 Red Hat, Inc.
 *
 */

package rest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/emicklei/go-restful"
	jsonpatch "github.com/evanphx/json-patch"
	"github.com/go-kit/kit/endpoint"
	"golang.org/x/net/context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"

	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/middleware"
	mime "kubevirt.io/kubevirt/pkg/rest"
	"kubevirt.io/kubevirt/pkg/rest/endpoints"
)

type ResponseHandlerFunc func(rest.Result) (interface{}, error)

func GroupVersionProxyBase(ctx context.Context, gv schema.GroupVersion) (*restful.WebService, error) {
	ws := new(restful.WebService)
	ws.Doc("The KubeVirt API, a virtual machine management.")
	ws.Path(GroupVersionBasePath(gv))

	virtClient, err := kubecli.GetKubevirtClient()
	if err != nil {
		return nil, err
	}
	autodiscover := endpoints.NewHandlerBuilder().Get().Decoder(endpoints.NoopDecoder).Endpoint(NewAutodiscoveryEndpoint(virtClient.RestClient())).Build(ctx)
	ws.Route(
		ws.GET("/").Produces(mime.MIME_JSON).Writes(metav1.APIResourceList{}).
			To(endpoints.MakeGoRestfulWrapper(autodiscover)).
			Operation("getAPIResources").
			Doc("Get KubeVirt API Resources").
			Returns(http.StatusOK, "OK", metav1.APIResourceList{}).
			Returns(http.StatusNotFound, "Not Found", nil),
	)
	return ws, nil
}

func GenericResourceProxy(ws *restful.WebService, ctx context.Context, gvr schema.GroupVersionResource, objPointer runtime.Object, objKind string, objListPointer runtime.Object) (*restful.WebService, error) {

	objResponseHandler := newResponseHandler(schema.GroupVersionKind{Group: gvr.Group, Version: gvr.Version, Kind: objKind}, objPointer)
	objListResponseHandler := newResponseHandler(schema.GroupVersionKind{Group: gvr.Group, Version: gvr.Version, Kind: objKind + "List"}, objListPointer)
	cli, err := kubecli.GetKubevirtClient()
	if err != nil {
		return nil, err
	}

	objExample := reflect.ValueOf(objPointer).Elem().Interface()
	listExample := reflect.ValueOf(objListPointer).Elem().Interface()

	delete := endpoints.NewHandlerBuilder().Delete().Endpoint(NewGenericDeleteEndpoint(cli.RestClient(), gvr, newStatusResponseHandler())).Build(ctx)
	put := endpoints.NewHandlerBuilder().Put(objPointer).Endpoint(NewGenericPutEndpoint(cli.RestClient(), gvr, objResponseHandler)).Build(ctx)
	patch := endpoints.NewHandlerBuilder().Patch().Endpoint(NewGenericPatchEndpoint(cli.RestClient(), gvr, objResponseHandler)).Build(ctx)
	post := endpoints.NewHandlerBuilder().Post(objPointer).Endpoint(NewGenericPostEndpoint(cli.RestClient(), gvr, objResponseHandler)).Build(ctx)
	get := endpoints.NewHandlerBuilder().Get().Endpoint(NewGenericGetEndpoint(cli.RestClient(), gvr, objResponseHandler)).Build(ctx)
	getListAllNamespaces := endpoints.NewHandlerBuilder().Get().Endpoint(NewGenericGetListEndpoint(cli.RestClient(), gvr, objListResponseHandler)).Decoder(endpoints.NotNamespacedDecodeRequestFunc).Build(ctx)
	getList := endpoints.NewHandlerBuilder().Get().Endpoint(NewGenericGetListEndpoint(cli.RestClient(), gvr, objListResponseHandler)).Decoder(endpoints.NamespaceDecodeRequestFunc).Build(ctx)
	deleteList := endpoints.NewHandlerBuilder().Delete().Endpoint(NewGenericDeleteListEndpoint(cli.RestClient(), gvr, objListResponseHandler)).Decoder(endpoints.NamespaceDecodeRequestFunc).Build(ctx)

	ws.Route(addPostParams(
		ws.POST(ResourceBasePath(gvr)).
			Produces(mime.MIME_JSON, mime.MIME_YAML).
			Consumes(mime.MIME_JSON, mime.MIME_YAML).
			Operation("createNamespaced"+objKind).
			To(endpoints.MakeGoRestfulWrapper(post)).Reads(objExample).Writes(objExample).
			Doc("Create a "+objKind+" object.").
			Returns(http.StatusOK, "OK", objExample).
			Returns(http.StatusCreated, "Created", objExample).
			Returns(http.StatusAccepted, "Accepted", objExample).
			Returns(http.StatusUnauthorized, "Unauthorized", nil), ws,
	))

	ws.Route(addPutParams(
		ws.PUT(ResourcePath(gvr)).
			Produces(mime.MIME_JSON, mime.MIME_YAML).
			Consumes(mime.MIME_JSON, mime.MIME_YAML).
			Operation("replaceNamespaced"+objKind).
			To(endpoints.MakeGoRestfulWrapper(put)).Reads(objExample).Writes(objExample).
			Doc("Update a "+objKind+" object.").
			Returns(http.StatusOK, "OK", objExample).
			Returns(http.StatusCreated, "Create", objExample).
			Returns(http.StatusUnauthorized, "Unauthorized", nil), ws,
	))

	ws.Route(addDeleteParams(
		ws.DELETE(ResourcePath(gvr)).
			Produces(mime.MIME_JSON, mime.MIME_YAML).
			Consumes(mime.MIME_JSON, mime.MIME_YAML).
			Operation("deleteNamespaced"+objKind).
			To(endpoints.MakeGoRestfulWrapper(delete)).
			Reads(metav1.DeleteOptions{}).Writes(metav1.Status{}).
			Doc("Delete a "+objKind+" object.").
			Returns(http.StatusOK, "OK", metav1.Status{}).
			Returns(http.StatusUnauthorized, "Unauthorized", nil), ws,
	))

	ws.Route(addGetParams(
		ws.GET(ResourcePath(gvr)).
			Produces(mime.MIME_JSON, mime.MIME_YAML, mime.MIME_JSON_STREAM).
			Operation("readNamespaced"+objKind).
			To(endpoints.MakeGoRestfulWrapper(get)).Writes(objExample).
			Doc("Get a "+objKind+" object.").
			Returns(http.StatusOK, "OK", objExample).
			Returns(http.StatusUnauthorized, "Unauthorized", nil), ws,
	))

	ws.Route(addGetAllNamespacesListParams(
		ws.GET(gvr.Resource).
			Produces(mime.MIME_JSON, mime.MIME_YAML, mime.MIME_JSON_STREAM).
			Operation("list"+objKind+"ForAllNamespaces").
			To(endpoints.MakeGoRestfulWrapper(getListAllNamespaces)).Writes(listExample).
			Doc("Get a list of all "+objKind+" objects.").
			Returns(http.StatusOK, "OK", listExample).
			Returns(http.StatusUnauthorized, "Unauthorized", nil), ws,
	))

	ws.Route(addPatchParams(
		ws.PATCH(ResourcePath(gvr)).
			Consumes(mime.MIME_JSON_PATCH, mime.MIME_MERGE_PATCH).
			Produces(mime.MIME_JSON).
			Operation("patchNamespaced"+objKind).
			To(endpoints.MakeGoRestfulWrapper(patch)).
			Writes(objExample).Reads(metav1.Patch{}).
			Doc("Patch a "+objKind+" object.").
			Returns(http.StatusOK, "OK", objExample).
			Returns(http.StatusUnauthorized, "Unauthorized", nil), ws,
	))

	// TODO, implement watch. For now it is here to provide swagger doc only
	ws.Route(addWatchGetListParams(
		ws.GET("/watch/"+gvr.Resource).
			Produces(mime.MIME_JSON).
			Operation("watch"+objKind+"ListForAllNamespaces").
			To(NotImplementedYet).Writes(metav1.WatchEvent{}).
			Doc("Watch a "+objKind+"List object.").
			Returns(http.StatusOK, "OK", metav1.WatchEvent{}).
			Returns(http.StatusUnauthorized, "Unauthorized", nil), ws,
	))

	// TODO, implement watch. For now it is here to provide swagger doc only
	ws.Route(addWatchNamespacedGetListParams(
		ws.GET("/watch"+ResourceBasePath(gvr)).
			Operation("watchNamespaced"+objKind).
			Produces(mime.MIME_JSON).
			To(NotImplementedYet).Writes(metav1.WatchEvent{}).
			Doc("Watch a "+objKind+" object.").
			Returns(http.StatusOK, "OK", metav1.WatchEvent{}).
			Returns(http.StatusUnauthorized, "Unauthorized", nil), ws,
	))

	ws.Route(addGetNamespacedListParams(
		ws.GET(ResourceBasePath(gvr)).
			Produces(mime.MIME_JSON, mime.MIME_YAML, mime.MIME_JSON_STREAM).
			Operation("listNamespaced"+objKind).
			Writes(listExample).
			To(endpoints.MakeGoRestfulWrapper(getList)).
			Doc("Get a list of "+objKind+" objects.").
			Returns(http.StatusOK, "OK", listExample).
			Returns(http.StatusUnauthorized, "Unauthorized", nil), ws,
	))

	ws.Route(addDeleteListParams(
		ws.DELETE(ResourceBasePath(gvr)).
			Operation("deleteCollectionNamespaced"+objKind).
			Produces(mime.MIME_JSON, mime.MIME_YAML).
			To(endpoints.MakeGoRestfulWrapper(deleteList)).Writes(metav1.Status{}).
			Doc("Delete a collection of "+objKind+" objects.").
			Returns(http.StatusOK, "OK", metav1.Status{}).
			Returns(http.StatusUnauthorized, "Unauthorized", nil), ws,
	))

	return ws, nil
}

func ResourceProxyAutodiscovery(ctx context.Context, gvr schema.GroupVersionResource) (*restful.WebService, error) {
	virtClient, err := kubecli.GetKubevirtClient()
	if err != nil {
		return nil, err
	}
	autodiscover := endpoints.NewHandlerBuilder().Get().Decoder(endpoints.NoopDecoder).Endpoint(NewAutodiscoveryEndpoint(virtClient.RestClient())).Build(ctx)
	ws := new(restful.WebService)
	ws.Path(GroupBasePath(gvr.GroupVersion()))
	ws.Route(ws.GET("/").
		Produces(mime.MIME_JSON).Writes(metav1.APIGroup{}).
		To(endpoints.MakeGoRestfulWrapper(autodiscover)).
		Doc("Get a KubeVirt API group").
		Operation("getAPIGroup").
		Returns(http.StatusOK, "OK", metav1.APIGroup{}).
		Returns(http.StatusNotFound, "Not Found", nil))
	return ws, nil
}

func addCollectionParams(builder *restful.RouteBuilder, ws *restful.WebService) *restful.RouteBuilder {
	return builder.Param(continueParam(ws)).
		Param(fieldSelectorParam(ws)).
		Param(includeUninitializedParam(ws)).
		Param(labelSelectorParam(ws)).
		Param(limitParam(ws)).
		Param(resourceVersionParam(ws)).
		Param(timeoutSecondsParam(ws)).
		Param(watchParam(ws))
}

func addWatchGetListParams(builder *restful.RouteBuilder, ws *restful.WebService) *restful.RouteBuilder {
	return addCollectionParams(builder, ws)
}

func addWatchNamespacedGetListParams(builder *restful.RouteBuilder, ws *restful.WebService) *restful.RouteBuilder {
	return addWatchGetListParams(builder.Param(NamespaceParam(ws)), ws)
}

func addGetAllNamespacesListParams(builder *restful.RouteBuilder, ws *restful.WebService) *restful.RouteBuilder {
	return addCollectionParams(builder, ws)
}

func addDeleteListParams(builder *restful.RouteBuilder, ws *restful.WebService) *restful.RouteBuilder {
	return addCollectionParams(builder, ws)
}

func addGetParams(builder *restful.RouteBuilder, ws *restful.WebService) *restful.RouteBuilder {
	return builder.Param(NameParam(ws)).
		Param(NamespaceParam(ws)).
		Param(exactParam(ws)).
		Param(exportParam(ws))
}

func addGetNamespacedListParams(builder *restful.RouteBuilder, ws *restful.WebService) *restful.RouteBuilder {
	return addCollectionParams(builder.Param(NamespaceParam(ws)), ws)
}

func addPostParams(builder *restful.RouteBuilder, ws *restful.WebService) *restful.RouteBuilder {
	return builder.Param(NamespaceParam(ws))
}

func addPutParams(builder *restful.RouteBuilder, ws *restful.WebService) *restful.RouteBuilder {
	return builder.Param(NamespaceParam(ws)).Param(NameParam(ws))
}

func addDeleteParams(builder *restful.RouteBuilder, ws *restful.WebService) *restful.RouteBuilder {
	return builder.Param(NamespaceParam(ws)).Param(NameParam(ws)).
		Param(gracePeriodSecondsParam(ws)).
		Param(orphanDependentsParam(ws)).
		Param(propagationPolicyParam(ws))
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

func resourceVersionParam(ws *restful.WebService) *restful.Parameter {
	return ws.QueryParameter("resourceVersion", "When specified with a watch call, shows changes that occur after that particular version of a resource. Defaults to changes from the beginning of history.")
}

func timeoutSecondsParam(ws *restful.WebService) *restful.Parameter {
	return ws.QueryParameter("timeoutSeconds", "TimeoutSeconds for the list/watch call.").DataType("integer")
}

func includeUninitializedParam(ws *restful.WebService) *restful.Parameter {
	return ws.QueryParameter("includeUninitialized", "If true, partially initialized resources are included in the response.").DataType("boolean")
}

func watchParam(ws *restful.WebService) *restful.Parameter {
	return ws.QueryParameter("watch", "Watch for changes to the described resources and return them as a stream of add, update, and remove notifications. Specify resourceVersion.").DataType("boolean")
}

func limitParam(ws *restful.WebService) *restful.Parameter {
	return ws.QueryParameter("limit", "limit is a maximum number of responses to return for a list call. If more items exist, the server will set the `continue` field on the list metadata to a value that can be used with the same initial query to retrieve the next set of results. Setting a limit may return fewer than the requested amount of items (up to zero items) in the event all requested objects are filtered out and clients should only use the presence of the continue field to determine whether more results are available. Servers may choose not to support the limit argument and will return all of the available results. If limit is specified and the continue field is empty, clients may assume that no more results are available. This field is not supported if watch is true.\n\nThe server guarantees that the objects returned when using continue will be identical to issuing a single list call without a limit - that is, no objects created, modified, or deleted after the first request is issued will be included in any subsequent continued requests. This is sometimes referred to as a consistent snapshot, and ensures that a client that is using limit to receive smaller chunks of a very large result can ensure they see all possible objects. If objects are updated during a chunked list the version of the object that was present at the time the first list result was calculated is returned.").DataType("integer")
}

func continueParam(ws *restful.WebService) *restful.Parameter {
	return ws.QueryParameter("continue", "The continue option should be set when retrieving more results from the server. Since this value is server defined, clients may only use the continue value from a previous query result with identical query parameters (except for the value of continue) and the server may reject a continue value it does not recognize. If the specified continue value is no longer valid whether due to expiration (generally five to fifteen minutes) or a configuration change on the server the server will respond with a 410 ResourceExpired error indicating the client must restart their list without the continue field. This field is not supported when watch is true. Clients may start a watch from the last resourceVersion value returned by the server and not miss any modifications.")
}

func exactParam(ws *restful.WebService) *restful.Parameter {
	return ws.QueryParameter("exact", "Should the export be exact. Exact export maintains cluster-specific fields like 'Namespace'.").DataType("boolean")
}

func exportParam(ws *restful.WebService) *restful.Parameter {
	return ws.QueryParameter("export", "Should this value be exported. Export strips fields that a user can not specify.").DataType("boolean")
}

func gracePeriodSecondsParam(ws *restful.WebService) *restful.Parameter {
	return ws.QueryParameter("gracePeriodSeconds", "The duration in seconds before the object should be deleted. Value must be non-negative integer. The value zero indicates delete immediately. If this value is nil, the default grace period for the specified type will be used. Defaults to a per object value if not specified. zero means delete immediately.").DataType("integer")
}

func orphanDependentsParam(ws *restful.WebService) *restful.Parameter {
	return ws.QueryParameter("orphanDependents", "Deprecated: please use the PropagationPolicy, this field will be deprecated in 1.7. Should the dependent objects be orphaned. If true/false, the \"orphan\" finalizer will be added to/removed from the object's finalizers list. Either this field or PropagationPolicy may be set, but not both.").DataType("boolean")
}

func propagationPolicyParam(ws *restful.WebService) *restful.Parameter {
	return ws.QueryParameter("propagationPolicy", "Whether and how garbage collection will be performed. Either this field or OrphanDependents may be set, but not both. The default policy is decided by the existing finalizer set in the metadata.finalizers and the resource-specific default policy. Acceptable values are: 'Orphan' - orphan the dependents; 'Background' - allow the garbage collector to delete the dependents in the background; 'Foreground' - a cascading policy that deletes all dependents in the foreground.")
}

func NewGenericDeleteEndpoint(cli *rest.RESTClient, gvr schema.GroupVersionResource, response ResponseHandlerFunc) endpoint.Endpoint {
	return func(ctx context.Context, payload interface{}) (interface{}, error) {
		p := payload.(*endpoints.PutObject)
		del := p.Payload
		if p.Payload == nil {
			del = &metav1.DeleteOptions{}
		}
		result := cli.Delete().Namespace(p.Metadata.Namespace).Resource(gvr.Resource).Name(p.Metadata.Name).Body(del).Do()
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
			Timeout(time.Duration(*listOptions.TimeoutSeconds) * time.Second).
			Resource(gvr.Resource).Do()
		return response(result)
	}
}

func listOptionsFromMetadata(metadata *endpoints.Metadata) (*metav1.ListOptions, error) {
	listOptions := &metav1.ListOptions{}
	if metadata.Headers.FieldSelector != "" {
		listOptions.FieldSelector = metadata.Headers.FieldSelector
	}
	if metadata.Headers.LabelSelector != "" {
		listOptions.LabelSelector = metadata.Headers.LabelSelector
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

func patchJson(patchType types.PatchType, patch interface{}, orig runtime.Object) (runtime.Object, error) {

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
	case types.MergePatchType:
		if rawPatched, err = jsonpatch.MergePatch(rawOriginal, rawPatch); err != nil {
			return nil, middleware.NewUnprocessibleEntityError(err)
		}
	case types.JSONPatchType:
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
