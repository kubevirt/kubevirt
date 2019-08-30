/*
Copyright 2018 The CDI Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"

	jsondiff "github.com/appscode/jsonpatch"
	jsonpatch "github.com/evanphx/json-patch"
	"github.com/go-logr/logr"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/jsonmergepatch"
	"k8s.io/apimachinery/pkg/util/mergepatch"
)

func mergeLabelsAndAnnotations(src, dest metav1.Object) {
	// allow users to add labels but not change ours
	for k, v := range src.GetLabels() {
		if dest.GetLabels() == nil {
			dest.SetLabels(map[string]string{})
		}

		dest.GetLabels()[k] = v
	}

	// same for annotations
	for k, v := range src.GetAnnotations() {
		if dest.GetAnnotations() == nil {
			dest.SetAnnotations(map[string]string{})
		}

		dest.GetAnnotations()[k] = v
	}
}

func mergeObject(desiredObj, currentObj runtime.Object) (runtime.Object, error) {
	desiredObj = desiredObj.DeepCopyObject()
	desiredMetaObj := desiredObj.(metav1.Object)
	currentMetaObj := currentObj.(metav1.Object)

	v, ok := currentMetaObj.GetAnnotations()[lastAppliedConfigAnnotation]
	if !ok {
		return nil, fmt.Errorf("%T %s/%s missing last applied config",
			currentMetaObj, currentMetaObj.GetNamespace(), currentMetaObj.GetName())
	}

	original := []byte(v)

	// setting the timestamp saves unnecessary updates because creation timestamp is nulled
	desiredMetaObj.SetCreationTimestamp(currentMetaObj.GetCreationTimestamp())
	modified, err := json.Marshal(desiredObj)
	if err != nil {
		return nil, err
	}

	current, err := json.Marshal(currentObj)
	if err != nil {
		return nil, err
	}

	preconditions := []mergepatch.PreconditionFunc{
		mergepatch.RequireKeyUnchanged("apiVersion"),
		mergepatch.RequireKeyUnchanged("kind"),
		mergepatch.RequireMetadataKeyUnchanged("name"),
	}

	patch, err := jsonmergepatch.CreateThreeWayJSONMergePatch(original, modified, current, preconditions...)
	if err != nil {
		return nil, err
	}

	newCurrent, err := jsonpatch.MergePatch(current, patch)
	if err != nil {
		return nil, err
	}

	result := newDefaultInstance(currentObj)
	if err = json.Unmarshal(newCurrent, result); err != nil {
		return nil, err
	}

	return result, nil
}

func deployClusterResources() bool {
	return strings.ToLower(os.Getenv("DEPLOY_CLUSTER_RESOURCES")) != "false"
}

func logJSONDiff(logger logr.Logger, objA, objB interface{}) {
	aBytes, _ := json.Marshal(objA)
	bBytes, _ := json.Marshal(objB)
	patches, _ := jsondiff.CreatePatch(aBytes, bBytes)
	pBytes, _ := json.Marshal(patches)
	logger.Info("DIFF", "obj", objA, "patch", string(pBytes))
}

func checkDeploymentReady(deployment *appsv1.Deployment) bool {
	desiredReplicas := deployment.Spec.Replicas
	if desiredReplicas == nil {
		desiredReplicas = &[]int32{1}[0]
	}

	if *desiredReplicas != deployment.Status.Replicas ||
		deployment.Status.Replicas != deployment.Status.ReadyReplicas {
		return false
	}

	return true
}

func newDefaultInstance(obj runtime.Object) runtime.Object {
	typ := reflect.ValueOf(obj).Elem().Type()
	return reflect.New(typ).Interface().(runtime.Object)
}
