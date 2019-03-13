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

package main

import (
	"fmt"
	"net/http"
	"reflect"

	restful "github.com/emicklei/go-restful"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	cdiv1alpha1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1alpha1"
)

// code stolen/adapted from https://github.com/kubevirt/kubevirt/blob/master/pkg/virt-api/rest/definitions.go

const (
	mimeJSON       string = "application/json"
	mimeJSONPatch  string = "application/json-patch+json"
	mimeJSONStream string = "application/json;stream=watch"
	mimeMergePatch string = "application/merge-patch+json"
	mimeYAML       string = "application/yaml"
	mimeText       string = "text/plain"
	mimeINI        string = "text/plain"
)

// DataVolumeAPI returns the DataVolume API for DataVolumes
func DataVolumeAPI() []*restful.WebService {

	dvGVR := schema.GroupVersionResource{
		Group:    cdiv1alpha1.SchemeGroupVersion.Group,
		Version:  cdiv1alpha1.SchemeGroupVersion.Version,
		Resource: "datavolumes",
	}

	cdiGVR := schema.GroupVersionResource{
		Group:    cdiv1alpha1.SchemeGroupVersion.Group,
		Version:  cdiv1alpha1.SchemeGroupVersion.Version,
		Resource: "cdis",
	}

	ws, err := groupVersionProxyBase(cdiv1alpha1.SchemeGroupVersion)
	if err != nil {
		panic(err)
	}

	ws, err = genericResourceProxy(ws, dvGVR, &cdiv1alpha1.DataVolume{}, "DataVolume", &cdiv1alpha1.DataVolumeList{})
	if err != nil {
		panic(err)
	}

	ws, err = genericResourceProxy(ws, cdiGVR, &cdiv1alpha1.CDI{}, "CDI", &cdiv1alpha1.CDIList{})
	if err != nil {
		panic(err)
	}

	ws1, err := resourceProxyAutodiscovery(dvGVR)
	if err != nil {
		panic(err)
	}

	return []*restful.WebService{ws, ws1}
}

func groupVersionProxyBase(gv schema.GroupVersion) (*restful.WebService, error) {
	ws := new(restful.WebService)
	ws.Doc("The KubeVirt API, a virtual machine management.")
	ws.Path(groupVersionBasePath(gv))

	ws.Route(
		ws.GET("/").Produces(mimeJSON).Writes(metav1.APIResourceList{}).
			To(noOp).
			Operation("getAPIResources").
			Doc("Get KubeVirt API Resources").
			Returns(http.StatusOK, "OK", metav1.APIResourceList{}).
			Returns(http.StatusNotFound, "Not Found", nil),
	)
	return ws, nil
}

func genericResourceProxy(ws *restful.WebService, gvr schema.GroupVersionResource, objPointer runtime.Object, objKind string, objListPointer runtime.Object) (*restful.WebService, error) {

	objExample := reflect.ValueOf(objPointer).Elem().Interface()
	listExample := reflect.ValueOf(objListPointer).Elem().Interface()

	ws.Route(addPostParams(
		ws.POST(resourceBasePath(gvr)).
			Produces(mimeJSON, mimeYAML).
			Consumes(mimeJSON, mimeYAML).
			Operation("createNamespaced"+objKind).
			To(noOp).Reads(objExample).Writes(objExample).
			Doc("Create a "+objKind+" object.").
			Returns(http.StatusOK, "OK", objExample).
			Returns(http.StatusCreated, "Created", objExample).
			Returns(http.StatusAccepted, "Accepted", objExample).
			Returns(http.StatusUnauthorized, "Unauthorized", nil), ws,
	))

	ws.Route(addPutParams(
		ws.PUT(resourcePath(gvr)).
			Produces(mimeJSON, mimeYAML).
			Consumes(mimeJSON, mimeYAML).
			Operation("replaceNamespaced"+objKind).
			To(noOp).Reads(objExample).Writes(objExample).
			Doc("Update a "+objKind+" object.").
			Returns(http.StatusOK, "OK", objExample).
			Returns(http.StatusCreated, "Create", objExample).
			Returns(http.StatusUnauthorized, "Unauthorized", nil), ws,
	))

	ws.Route(addDeleteParams(
		ws.DELETE(resourcePath(gvr)).
			Produces(mimeJSON, mimeYAML).
			Consumes(mimeJSON, mimeYAML).
			Operation("deleteNamespaced"+objKind).
			To(noOp).
			Reads(metav1.DeleteOptions{}).Writes(metav1.Status{}).
			Doc("Delete a "+objKind+" object.").
			Returns(http.StatusOK, "OK", metav1.Status{}).
			Returns(http.StatusUnauthorized, "Unauthorized", nil), ws,
	))

	ws.Route(addGetParams(
		ws.GET(resourcePath(gvr)).
			Produces(mimeJSON, mimeYAML, mimeJSONStream).
			Operation("readNamespaced"+objKind).
			To(noOp).Writes(objExample).
			Doc("Get a "+objKind+" object.").
			Returns(http.StatusOK, "OK", objExample).
			Returns(http.StatusUnauthorized, "Unauthorized", nil), ws,
	))

	ws.Route(addGetAllNamespacesListParams(
		ws.GET(gvr.Resource).
			Produces(mimeJSON, mimeYAML, mimeJSONStream).
			Operation("list"+objKind+"ForAllNamespaces").
			To(noOp).Writes(listExample).
			Doc("Get a list of all "+objKind+" objects.").
			Returns(http.StatusOK, "OK", listExample).
			Returns(http.StatusUnauthorized, "Unauthorized", nil), ws,
	))

	ws.Route(addPatchParams(
		ws.PATCH(resourcePath(gvr)).
			Consumes(mimeJSONPatch, mimeMergePatch).
			Produces(mimeJSON).
			Operation("patchNamespaced"+objKind).
			To(noOp).
			Writes(objExample).Reads(metav1.Patch{}).
			Doc("Patch a "+objKind+" object.").
			Returns(http.StatusOK, "OK", objExample).
			Returns(http.StatusUnauthorized, "Unauthorized", nil), ws,
	))

	// TODO, implement watch. For now it is here to provide swagger doc only
	ws.Route(addWatchGetListParams(
		ws.GET("/watch/"+gvr.Resource).
			Produces(mimeJSON).
			Operation("watch"+objKind+"ListForAllNamespaces").
			To(noOp).Writes(metav1.WatchEvent{}).
			Doc("Watch a "+objKind+"List object.").
			Returns(http.StatusOK, "OK", metav1.WatchEvent{}).
			Returns(http.StatusUnauthorized, "Unauthorized", nil), ws,
	))

	// TODO, implement watch. For now it is here to provide swagger doc only
	ws.Route(addWatchNamespacedGetListParams(
		ws.GET("/watch"+resourceBasePath(gvr)).
			Operation("watchNamespaced"+objKind).
			Produces(mimeJSON).
			To(noOp).Writes(metav1.WatchEvent{}).
			Doc("Watch a "+objKind+" object.").
			Returns(http.StatusOK, "OK", metav1.WatchEvent{}).
			Returns(http.StatusUnauthorized, "Unauthorized", nil), ws,
	))

	ws.Route(addGetNamespacedListParams(
		ws.GET(resourceBasePath(gvr)).
			Produces(mimeJSON, mimeYAML, mimeJSONStream).
			Operation("listNamespaced"+objKind).
			Writes(listExample).
			To(noOp).
			Doc("Get a list of "+objKind+" objects.").
			Returns(http.StatusOK, "OK", listExample).
			Returns(http.StatusUnauthorized, "Unauthorized", nil), ws,
	))

	ws.Route(addDeleteListParams(
		ws.DELETE(resourceBasePath(gvr)).
			Operation("deleteCollectionNamespaced"+objKind).
			Produces(mimeJSON, mimeYAML).
			To(noOp).Writes(metav1.Status{}).
			Doc("Delete a collection of "+objKind+" objects.").
			Returns(http.StatusOK, "OK", metav1.Status{}).
			Returns(http.StatusUnauthorized, "Unauthorized", nil), ws,
	))

	return ws, nil
}

func resourceProxyAutodiscovery(gvr schema.GroupVersionResource) (*restful.WebService, error) {
	ws := new(restful.WebService)
	ws.Path(groupBasePath(gvr.GroupVersion()))
	ws.Route(ws.GET("/").
		Produces(mimeJSON).Writes(metav1.APIGroup{}).
		To(noOp).
		Doc("Get a KubeVirt CDI API group").
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
	return addWatchGetListParams(builder.Param(namespaceParam(ws)), ws)
}

func addGetAllNamespacesListParams(builder *restful.RouteBuilder, ws *restful.WebService) *restful.RouteBuilder {
	return addCollectionParams(builder, ws)
}

func addDeleteListParams(builder *restful.RouteBuilder, ws *restful.WebService) *restful.RouteBuilder {
	return addCollectionParams(builder, ws)
}

func addGetParams(builder *restful.RouteBuilder, ws *restful.WebService) *restful.RouteBuilder {
	return builder.Param(nameParam(ws)).
		Param(namespaceParam(ws)).
		Param(exactParam(ws)).
		Param(exportParam(ws))
}

func addGetNamespacedListParams(builder *restful.RouteBuilder, ws *restful.WebService) *restful.RouteBuilder {
	return addCollectionParams(builder.Param(namespaceParam(ws)), ws)
}

func addPostParams(builder *restful.RouteBuilder, ws *restful.WebService) *restful.RouteBuilder {
	return builder.Param(namespaceParam(ws))
}

func addPutParams(builder *restful.RouteBuilder, ws *restful.WebService) *restful.RouteBuilder {
	return builder.Param(namespaceParam(ws)).Param(nameParam(ws))
}

func addDeleteParams(builder *restful.RouteBuilder, ws *restful.WebService) *restful.RouteBuilder {
	return builder.Param(namespaceParam(ws)).Param(nameParam(ws)).
		Param(gracePeriodSecondsParam(ws)).
		Param(orphanDependentsParam(ws)).
		Param(propagationPolicyParam(ws))
}

func addPatchParams(builder *restful.RouteBuilder, ws *restful.WebService) *restful.RouteBuilder {
	return builder.Param(namespaceParam(ws)).Param(nameParam(ws))
}

func nameParam(ws *restful.WebService) *restful.Parameter {
	return ws.PathParameter("name", "Name of the resource").Required(true)
}

func namespaceParam(ws *restful.WebService) *restful.Parameter {
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

func groupBasePath(gvr schema.GroupVersion) string {
	return fmt.Sprintf("/apis/%s", gvr.Group)
}

func groupVersionBasePath(gvr schema.GroupVersion) string {
	return fmt.Sprintf("/apis/%s/%s", gvr.Group, gvr.Version)
}

func resourceBasePath(gvr schema.GroupVersionResource) string {
	return fmt.Sprintf("/namespaces/{namespace:[a-z0-9][a-z0-9\\-]*}/%s", gvr.Resource)
}

func resourcePath(gvr schema.GroupVersionResource) string {
	return fmt.Sprintf("/namespaces/{namespace:[a-z0-9][a-z0-9\\-]*}/%s/{name:[a-z0-9][a-z0-9\\-]*}", gvr.Resource)
}

func noOp(request *restful.Request, response *restful.Response) {}
