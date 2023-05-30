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

package definitions

import (
	"fmt"
	"net/http"
	"path"
	"reflect"

	"kubevirt.io/api/clone"
	clonev1lpha1 "kubevirt.io/api/clone/v1alpha1"

	"kubevirt.io/api/instancetype"

	"kubevirt.io/api/migrations"

	migrationsv1 "kubevirt.io/api/migrations/v1alpha1"

	restful "github.com/emicklei/go-restful/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	v1 "kubevirt.io/api/core/v1"
	exportv1 "kubevirt.io/api/export/v1alpha1"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"
	poolv1alpha1 "kubevirt.io/api/pool/v1alpha1"
	snapshotv1 "kubevirt.io/api/snapshot/v1alpha1"

	mime "kubevirt.io/kubevirt/pkg/rest"
)

const (
	obj   = " object."
	objs  = " objects."
	watch = "Watch a "
)

func ComposeAPIDefinitions() []*restful.WebService {
	var result []*restful.WebService
	for _, f := range []func() []*restful.WebService{
		kubevirtApiServiceDefinitions,
		snapshotApiServiceDefinitions,
		exportApiServiceDefinitions,
		instancetypeApiServiceDefinitions,
		migrationPoliciesApiServiceDefinitions,
		poolApiServiceDefinitions,
		vmCloneDefinitions,
	} {
		result = append(result, f()...)
	}

	return result
}

func kubevirtApiServiceDefinitions() []*restful.WebService {
	vmiGVR := schema.GroupVersionResource{Group: v1.GroupVersion.Group, Version: v1.GroupVersion.Version, Resource: "virtualmachineinstances"}
	vmirsGVR := schema.GroupVersionResource{Group: v1.GroupVersion.Group, Version: v1.GroupVersion.Version, Resource: "virtualmachineinstancereplicasets"}
	vmipGVR := schema.GroupVersionResource{Group: v1.GroupVersion.Group, Version: v1.GroupVersion.Version, Resource: "virtualmachineinstancepresets"}
	vmGVR := schema.GroupVersionResource{Group: v1.GroupVersion.Group, Version: v1.GroupVersion.Version, Resource: "virtualmachines"}
	migrationGVR := schema.GroupVersionResource{Group: v1.GroupVersion.Group, Version: v1.GroupVersion.Version, Resource: "virtualmachineinstancemigrations"}
	kubeVirtGVR := schema.GroupVersionResource{Group: v1.GroupVersion.Group, Version: v1.GroupVersion.Version, Resource: "kubevirt"}

	ws, err := groupVersionProxyBase(v1.GroupVersion)
	if err != nil {
		panic(err)
	}

	ws, err = genericNamespacedResourceProxy(ws, kubeVirtGVR, &v1.KubeVirt{}, v1.KubeVirtGroupVersionKind.Kind, &v1.KubeVirtList{})
	if err != nil {
		panic(err)
	}
	ws, err = genericNamespacedResourceProxy(ws, vmiGVR, &v1.VirtualMachineInstance{}, v1.VirtualMachineInstanceGroupVersionKind.Kind, &v1.VirtualMachineInstanceList{})
	if err != nil {
		panic(err)
	}

	ws, err = genericNamespacedResourceProxy(ws, vmirsGVR, &v1.VirtualMachineInstanceReplicaSet{}, v1.VirtualMachineInstanceReplicaSetGroupVersionKind.Kind, &v1.VirtualMachineInstanceReplicaSetList{})
	if err != nil {
		panic(err)
	}

	ws, err = genericNamespacedResourceProxy(ws, vmipGVR, &v1.VirtualMachineInstancePreset{}, v1.VirtualMachineInstancePresetGroupVersionKind.Kind, &v1.VirtualMachineInstancePresetList{})
	if err != nil {
		panic(err)
	}

	ws, err = genericNamespacedResourceProxy(ws, vmGVR, &v1.VirtualMachine{}, v1.VirtualMachineGroupVersionKind.Kind, &v1.VirtualMachineList{})
	if err != nil {
		panic(err)
	}

	ws, err = genericNamespacedResourceProxy(ws, migrationGVR, &v1.VirtualMachineInstanceMigration{}, v1.VirtualMachineInstanceMigrationGroupVersionKind.Kind, &v1.VirtualMachineInstanceMigrationList{})
	if err != nil {
		panic(err)
	}

	ws2, err := resourceProxyAutodiscovery(vmiGVR)
	if err != nil {
		panic(err)
	}

	return []*restful.WebService{ws, ws2}
}

func snapshotApiServiceDefinitions() []*restful.WebService {
	vmsGVR := snapshotv1.SchemeGroupVersion.WithResource("virtualmachinesnapshots")
	vmscGVR := snapshotv1.SchemeGroupVersion.WithResource("virtualmachinesnapshotcontents")
	vmrGVR := snapshotv1.SchemeGroupVersion.WithResource("virtualmachinerestores")

	ws, err := groupVersionProxyBase(schema.GroupVersion{Group: snapshotv1.SchemeGroupVersion.Group, Version: snapshotv1.SchemeGroupVersion.Version})
	if err != nil {
		panic(err)
	}

	ws, err = genericNamespacedResourceProxy(ws, vmsGVR, &snapshotv1.VirtualMachineSnapshot{}, "VirtualMachineSnapshot", &snapshotv1.VirtualMachineSnapshotList{})
	if err != nil {
		panic(err)
	}

	ws, err = genericNamespacedResourceProxy(ws, vmscGVR, &snapshotv1.VirtualMachineSnapshotContent{}, "VirtualMachineSnapshotContent", &snapshotv1.VirtualMachineSnapshotContentList{})
	if err != nil {
		panic(err)
	}

	ws, err = genericNamespacedResourceProxy(ws, vmrGVR, &snapshotv1.VirtualMachineRestore{}, "VirtualMachineRestore", &snapshotv1.VirtualMachineRestoreList{})
	if err != nil {
		panic(err)
	}

	ws2, err := resourceProxyAutodiscovery(vmsGVR)
	if err != nil {
		panic(err)
	}
	return []*restful.WebService{ws, ws2}
}

func exportApiServiceDefinitions() []*restful.WebService {
	exportsGVR := exportv1.SchemeGroupVersion.WithResource("virtualmachineexports")

	ws, err := groupVersionProxyBase(schema.GroupVersion{Group: exportv1.SchemeGroupVersion.Group, Version: exportv1.SchemeGroupVersion.Version})
	if err != nil {
		panic(err)
	}

	ws, err = genericNamespacedResourceProxy(ws, exportsGVR, &exportv1.VirtualMachineExport{}, "VirtualMachineExport", &exportv1.VirtualMachineExportList{})
	if err != nil {
		panic(err)
	}

	ws2, err := resourceProxyAutodiscovery(exportsGVR)
	if err != nil {
		panic(err)
	}
	return []*restful.WebService{ws, ws2}
}

func migrationPoliciesApiServiceDefinitions() []*restful.WebService {
	mpGVR := migrationsv1.SchemeGroupVersion.WithResource(migrations.ResourceMigrationPolicies)

	ws, err := groupVersionProxyBase(schema.GroupVersion{Group: migrationsv1.SchemeGroupVersion.Group, Version: migrationsv1.SchemeGroupVersion.Version})
	if err != nil {
		panic(err)
	}

	ws, err = genericClusterResourceProxy(ws, mpGVR, &migrationsv1.MigrationPolicy{}, migrationsv1.MigrationPolicyKind.Kind, &migrationsv1.MigrationPolicyList{})
	if err != nil {
		panic(err)
	}

	ws2, err := resourceProxyAutodiscovery(mpGVR)
	if err != nil {
		panic(err)
	}
	return []*restful.WebService{ws, ws2}
}

func instancetypeApiServiceDefinitions() []*restful.WebService {
	instancetypeGVR := instancetypev1beta1.SchemeGroupVersion.WithResource(instancetype.PluralResourceName)
	clusterInstancetypeGVR := instancetypev1beta1.SchemeGroupVersion.WithResource(instancetype.ClusterPluralResourceName)
	preferenceGVR := instancetypev1beta1.SchemeGroupVersion.WithResource(instancetype.PluralPreferenceResourceName)
	clusterPreferenceGVR := instancetypev1beta1.SchemeGroupVersion.WithResource(instancetype.ClusterPluralPreferenceResourceName)

	ws, err := groupVersionProxyBase(instancetypev1beta1.SchemeGroupVersion)
	if err != nil {
		panic(err)
	}

	ws, err = genericNamespacedResourceProxy(ws, instancetypeGVR, &instancetypev1beta1.VirtualMachineInstancetype{}, "VirtualMachineInstancetype", &instancetypev1beta1.VirtualMachineInstancetypeList{})
	if err != nil {
		panic(err)
	}

	ws, err = genericClusterResourceProxy(ws, clusterInstancetypeGVR, &instancetypev1beta1.VirtualMachineClusterInstancetype{}, "VirtualMachineClusterInstancetype", &instancetypev1beta1.VirtualMachineClusterInstancetypeList{})
	if err != nil {
		panic(err)
	}

	ws, err = genericNamespacedResourceProxy(ws, preferenceGVR, &instancetypev1beta1.VirtualMachinePreference{}, "VirtualMachinePreference", &instancetypev1beta1.VirtualMachinePreferenceList{})
	if err != nil {
		panic(err)
	}

	ws, err = genericClusterResourceProxy(ws, clusterPreferenceGVR, &instancetypev1beta1.VirtualMachineClusterPreference{}, "VirtualMachineClusterPreference", &instancetypev1beta1.VirtualMachineClusterPreferenceList{})
	if err != nil {
		panic(err)
	}

	ws2, err := resourceProxyAutodiscovery(instancetypeGVR)
	if err != nil {
		panic(err)
	}

	return []*restful.WebService{ws, ws2}
}

func poolApiServiceDefinitions() []*restful.WebService {
	poolGVR := poolv1alpha1.SchemeGroupVersion.WithResource("virtualmachinepools")

	ws, err := groupVersionProxyBase(poolv1alpha1.SchemeGroupVersion)
	if err != nil {
		panic(err)
	}

	ws, err = genericNamespacedResourceProxy(ws, poolGVR, &poolv1alpha1.VirtualMachinePool{}, "VirtualMachinePool", &poolv1alpha1.VirtualMachinePoolList{})
	if err != nil {
		panic(err)
	}

	ws2, err := resourceProxyAutodiscovery(poolGVR)
	if err != nil {
		panic(err)
	}

	return []*restful.WebService{ws, ws2}
}

func vmCloneDefinitions() []*restful.WebService {
	mpGVR := clonev1lpha1.SchemeGroupVersion.WithResource(clone.ResourceVMClonePlural)

	ws, err := groupVersionProxyBase(schema.GroupVersion{Group: clonev1lpha1.SchemeGroupVersion.Group, Version: clonev1lpha1.SchemeGroupVersion.Version})
	if err != nil {
		panic(err)
	}

	ws, err = genericClusterResourceProxy(ws, mpGVR, &clonev1lpha1.VirtualMachineClone{}, clonev1lpha1.VirtualMachineCloneKind.Kind, &clonev1lpha1.VirtualMachineCloneList{})
	if err != nil {
		panic(err)
	}

	ws2, err := resourceProxyAutodiscovery(mpGVR)
	if err != nil {
		panic(err)
	}
	return []*restful.WebService{ws, ws2}
}

func groupVersionProxyBase(gv schema.GroupVersion) (*restful.WebService, error) {
	ws := new(restful.WebService)
	ws.Doc("The KubeVirt API, a virtual machine management.")
	ws.Path(GroupVersionBasePath(gv))

	ws.Route(
		ws.GET("/").Produces(mime.MIME_JSON).Writes(metav1.APIResourceList{}).
			To(noop).
			Operation(fmt.Sprintf("getAPIResources-%s-%s", gv.Group, gv.Version)).
			Doc("Get KubeVirt API Resources").
			Returns(http.StatusOK, "OK", metav1.APIResourceList{}).
			Returns(http.StatusNotFound, "Not Found", ""),
	)
	return ws, nil
}

func genericNamespacedResourceProxy(ws *restful.WebService, gvr schema.GroupVersionResource, objPointer runtime.Object, objKind string, objListPointer runtime.Object) (*restful.WebService, error) {

	objExample := reflect.ValueOf(objPointer).Elem().Interface()
	listExample := reflect.ValueOf(objListPointer).Elem().Interface()

	ws.Route(addNamespaceParam(ws,
		createOperation(ws, NamespacedResourceBasePath(gvr), objExample).
			Operation("createNamespaced"+objKind).
			Doc("Create a "+objKind+obj),
	))

	ws.Route(addNamespaceParam(ws,
		replaceOperation(ws, NamespacedResourcePath(gvr), objExample).
			Operation("replaceNamespaced"+objKind).
			Doc("Update a "+objKind+obj),
	))

	ws.Route(addNamespaceParam(ws,
		deleteOperation(ws, NamespacedResourcePath(gvr)).
			Operation("deleteNamespaced"+objKind).
			Doc("Delete a "+objKind+obj),
	))

	ws.Route(addNamespaceParam(ws,
		readOperation(ws, NamespacedResourcePath(gvr), objExample).
			Operation("readNamespaced"+objKind).
			Doc("Get a "+objKind+obj),
	))

	ws.Route(
		listOperation(ws, gvr.Resource, listExample).
			Operation("list" + objKind + "ForAllNamespaces").
			Doc("Get a list of all " + objKind + objs),
	)

	ws.Route(addNamespaceParam(ws,
		patchOperation(ws, NamespacedResourcePath(gvr), objExample).
			Operation("patchNamespaced"+objKind).
			Doc("Patch a "+objKind+obj),
	))

	// TODO, implement watch. For now it is here to provide swagger doc only
	ws.Route(
		watchOperation(ws, "/watch/"+gvr.Resource).
			Operation("watch" + objKind + "ListForAllNamespaces").
			Doc(watch + objKind + "List object."),
	)

	// TODO, implement watch. For now it is here to provide swagger doc only
	ws.Route(addNamespaceParam(ws,
		watchOperation(ws, "/watch"+NamespacedResourceBasePath(gvr)).
			Operation("watchNamespaced"+objKind).
			Doc(watch+objKind+obj),
	))

	ws.Route(addNamespaceParam(ws,
		listOperation(ws, NamespacedResourceBasePath(gvr), listExample).
			Operation("listNamespaced"+objKind).
			Doc("Get a list of "+objKind+objs),
	))

	ws.Route(
		deleteCollectionOperation(ws, NamespacedResourceBasePath(gvr)).
			Operation("deleteCollectionNamespaced" + objKind).
			Doc("Delete a collection of " + objKind + objs),
	)

	return ws, nil
}

func genericClusterResourceProxy(ws *restful.WebService, gvr schema.GroupVersionResource, objPointer runtime.Object, objKind string, objListPointer runtime.Object) (*restful.WebService, error) {

	objExample := reflect.ValueOf(objPointer).Elem().Interface()
	listExample := reflect.ValueOf(objListPointer).Elem().Interface()

	ws.Route(
		createOperation(ws, ClusterResourceBasePath(gvr), objExample).
			Operation("create" + objKind).
			Doc("Create a " + objKind + obj),
	)

	ws.Route(
		replaceOperation(ws, ClusterResourcePath(gvr), objExample).
			Operation("replace" + objKind).
			Doc("Update a " + objKind + obj),
	)

	ws.Route(
		deleteOperation(ws, ClusterResourcePath(gvr)).
			Operation("delete" + objKind).
			Doc("Delete a " + objKind + obj),
	)

	ws.Route(
		readOperation(ws, ClusterResourcePath(gvr), objExample).
			Operation("read" + objKind).
			Doc("Get a " + objKind + obj),
	)

	ws.Route(
		listOperation(ws, gvr.Resource, listExample).
			Operation("list" + objKind).
			Doc("Get a list of " + objKind + objs),
	)

	ws.Route(
		patchOperation(ws, ClusterResourcePath(gvr), objExample).
			Operation("patch" + objKind).
			Doc("Patch a " + objKind + obj),
	)

	// TODO, implement watch. For now it is here to provide swagger doc only
	ws.Route(
		watchOperation(ws, "/watch/"+gvr.Resource).
			Operation("watch" + objKind + "ListForAllNamespaces").
			Doc(watch + objKind + "List object."),
	)

	ws.Route(
		deleteCollectionOperation(ws, ClusterResourceBasePath(gvr)).
			Operation("deleteCollection" + objKind).
			Doc("Delete a collection of " + objKind + objs),
	)

	return ws, nil
}

func resourceProxyAutodiscovery(gvr schema.GroupVersionResource) (*restful.WebService, error) {
	ws := new(restful.WebService)
	ws.Path(GroupBasePath(gvr.GroupVersion()))
	ws.Route(ws.GET("/").
		Produces(mime.MIME_JSON).Writes(metav1.APIGroup{}).
		To(noop).
		Doc("Get a KubeVirt API group").
		Operation("getAPIGroup-"+gvr.Group).
		Returns(http.StatusOK, "OK", metav1.APIGroup{}).
		Returns(http.StatusNotFound, "Not Found", ""))
	return ws, nil
}

func createOperation(ws *restful.WebService, subPath string, objExample interface{}) *restful.RouteBuilder {
	return ws.POST(subPath).
		Produces(mime.MIME_JSON, mime.MIME_YAML).
		Consumes(mime.MIME_JSON, mime.MIME_YAML).
		To(noop).Reads(objExample).Writes(objExample).
		Returns(http.StatusOK, "OK", objExample).
		Returns(http.StatusCreated, "Created", objExample).
		Returns(http.StatusAccepted, "Accepted", objExample).
		Returns(http.StatusUnauthorized, "Unauthorized", "")
}

func replaceOperation(ws *restful.WebService, subPath string, objExample interface{}) *restful.RouteBuilder {
	return addPutParams(ws,
		ws.PUT(subPath).
			Produces(mime.MIME_JSON, mime.MIME_YAML).
			Consumes(mime.MIME_JSON, mime.MIME_YAML).
			To(noop).Reads(objExample).Writes(objExample).
			Returns(http.StatusOK, "OK", objExample).
			Returns(http.StatusCreated, "Create", objExample).
			Returns(http.StatusUnauthorized, "Unauthorized", ""),
	)
}

func patchOperation(ws *restful.WebService, subPath string, objExample interface{}) *restful.RouteBuilder {
	return addPatchParams(ws,
		ws.PATCH(subPath).
			Consumes(mime.MIME_JSON_PATCH, mime.MIME_MERGE_PATCH).
			Produces(mime.MIME_JSON).
			To(noop).
			Writes(objExample).Reads(metav1.Patch{}).
			Returns(http.StatusOK, "OK", objExample).
			Returns(http.StatusUnauthorized, "Unauthorized", ""),
	)
}

func deleteOperation(ws *restful.WebService, subPath string) *restful.RouteBuilder {
	return addDeleteParams(ws,
		ws.DELETE(subPath).
			Produces(mime.MIME_JSON, mime.MIME_YAML).
			Consumes(mime.MIME_JSON, mime.MIME_YAML).
			To(noop).
			Reads(metav1.DeleteOptions{}).Writes(metav1.Status{}).
			Returns(http.StatusOK, "OK", metav1.Status{}).
			Returns(http.StatusUnauthorized, "Unauthorized", ""),
	)
}

func deleteCollectionOperation(ws *restful.WebService, subPath string) *restful.RouteBuilder {
	return addDeleteListParams(ws,
		ws.DELETE(subPath).
			Produces(mime.MIME_JSON, mime.MIME_YAML).
			To(noop).Writes(metav1.Status{}).
			Returns(http.StatusOK, "OK", metav1.Status{}).
			Returns(http.StatusUnauthorized, "Unauthorized", ""),
	)
}

func readOperation(ws *restful.WebService, subPath string, objExample interface{}) *restful.RouteBuilder {
	return addGetParams(ws,
		ws.GET(subPath).
			Produces(mime.MIME_JSON, mime.MIME_YAML, mime.MIME_JSON_STREAM).
			To(noop).Writes(objExample).
			Returns(http.StatusOK, "OK", objExample).
			Returns(http.StatusUnauthorized, "Unauthorized", ""),
	)
}

func listOperation(ws *restful.WebService, subPath string, listExample interface{}) *restful.RouteBuilder {
	return addGetListParams(ws,
		ws.GET(subPath).
			Produces(mime.MIME_JSON, mime.MIME_YAML, mime.MIME_JSON_STREAM).
			To(noop).Writes(listExample).
			Returns(http.StatusOK, "OK", listExample).
			Returns(http.StatusUnauthorized, "Unauthorized", ""),
	)
}

func watchOperation(ws *restful.WebService, subPath string) *restful.RouteBuilder {
	return addWatchGetListParams(ws,
		ws.GET(subPath).
			Produces(mime.MIME_JSON).
			To(noop).Writes(metav1.WatchEvent{}).
			Returns(http.StatusOK, "OK", metav1.WatchEvent{}).
			Returns(http.StatusUnauthorized, "Unauthorized", ""),
	)
}

func addCollectionParams(ws *restful.WebService, builder *restful.RouteBuilder) *restful.RouteBuilder {
	return builder.Param(continueParam(ws)).
		Param(fieldSelectorParam(ws)).
		Param(includeUninitializedParam(ws)).
		Param(labelSelectorParam(ws)).
		Param(limitParam(ws)).
		Param(resourceVersionParam(ws)).
		Param(timeoutSecondsParam(ws)).
		Param(watchParam(ws))
}

func addNamespaceParam(ws *restful.WebService, builder *restful.RouteBuilder) *restful.RouteBuilder {
	return builder.Param(NamespaceParam(ws))
}

func addWatchGetListParams(ws *restful.WebService, builder *restful.RouteBuilder) *restful.RouteBuilder {
	return addCollectionParams(ws, builder)
}

func addGetListParams(ws *restful.WebService, builder *restful.RouteBuilder) *restful.RouteBuilder {
	return addCollectionParams(ws, builder)
}

func addDeleteListParams(ws *restful.WebService, builder *restful.RouteBuilder) *restful.RouteBuilder {
	return addCollectionParams(ws, builder)
}

func addGetParams(ws *restful.WebService, builder *restful.RouteBuilder) *restful.RouteBuilder {
	return builder.Param(NameParam(ws)).
		Param(exactParam(ws)).
		Param(exportParam(ws))
}

func addPutParams(ws *restful.WebService, builder *restful.RouteBuilder) *restful.RouteBuilder {
	return builder.Param(NameParam(ws))
}

func addDeleteParams(ws *restful.WebService, builder *restful.RouteBuilder) *restful.RouteBuilder {
	return builder.Param(NameParam(ws)).
		Param(gracePeriodSecondsParam(ws)).
		Param(orphanDependentsParam(ws)).
		Param(propagationPolicyParam(ws))
}

func addPatchParams(ws *restful.WebService, builder *restful.RouteBuilder) *restful.RouteBuilder {
	return builder.Param(NameParam(ws))
}

const (
	NamespaceParamName  = "namespace"
	NameParamName       = "name"
	MoveCursorParamName = "moveCursor"
)

func NameParam(ws *restful.WebService) *restful.Parameter {
	return ws.PathParameter(NameParamName, "Name of the resource").Required(true)
}

func NamespaceParam(ws *restful.WebService) *restful.Parameter {
	return ws.PathParameter(NamespaceParamName, "Object name and auth scope, such as for teams and projects").Required(true)
}

func MoveCursorParam(ws *restful.WebService) *restful.Parameter {
	return ws.QueryParameter(MoveCursorParamName, "Move the cursor on the VNC display to wake up the screen").DataType("boolean").DefaultValue("false")
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

func GroupBasePath(gvr schema.GroupVersion) string {
	return fmt.Sprintf("/apis/%s", gvr.Group)
}

func GroupVersionBasePath(gvr schema.GroupVersion) string {
	return fmt.Sprintf("/apis/%s/%s", gvr.Group, gvr.Version)
}

func NamespacedResourceBasePath(gvr schema.GroupVersionResource) string {
	return fmt.Sprintf("/namespaces/{namespace:[a-z0-9][a-z0-9\\-]*}/%s", gvr.Resource)
}

func NamespacedResourcePath(gvr schema.GroupVersionResource) string {
	return fmt.Sprintf("/namespaces/{namespace:[a-z0-9][a-z0-9\\-]*}/%s/{name:[a-z0-9][a-z0-9\\-]*}", gvr.Resource)
}

func ClusterResourceBasePath(gvr schema.GroupVersionResource) string {
	return gvr.Resource
}

func ClusterResourcePath(gvr schema.GroupVersionResource) string {
	return fmt.Sprintf("%s/{name:[a-z0-9][a-z0-9\\-]*}", gvr.Resource)
}

func SubResourcePath(subResource string) string {
	return path.Join("/", subResource)
}

const (
	PortParamName     = "port"
	TLSParamName      = "tls"
	PortPath          = "/{port:[0-9]+}"
	ProtocolParamName = "protocol"
	ProtocolPath      = "/{protocol:tcp|udp}"
)

func PortForwardPortParameter(ws *restful.WebService) *restful.Parameter {
	return ws.PathParameter(PortParamName, "The target port for portforward on the VirtualMachineInstance.")
}

func PortForwardProtocolParameter(ws *restful.WebService) *restful.Parameter {
	return ws.PathParameter(ProtocolParamName, "The protocol for portforward on the VirtualMachineInstance.")
}

func noop(_ *restful.Request, _ *restful.Response) {}

func VSOCKPortParameter(ws *restful.WebService) *restful.Parameter {
	return ws.QueryParameter(PortParamName, "The port which the VSOCK application listens to.").DataType("integer").Required(true)
}

func VSOCKTLSParameter(ws *restful.WebService) *restful.Parameter {
	return ws.QueryParameter(TLSParamName, "Weather to request a TLS encrypted session from the VSOCK application.").DataType("boolean").Required(false)
}
