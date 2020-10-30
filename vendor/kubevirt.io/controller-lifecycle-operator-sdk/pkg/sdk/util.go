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

package sdk

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"

	v1 "k8s.io/api/core/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"

	jsondiff "github.com/appscode/jsonpatch"
	jsonpatch "github.com/evanphx/json-patch"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/jsonmergepatch"
	"k8s.io/apimachinery/pkg/util/mergepatch"
)

const statusKey = "status"
const capitalStatusKey = "Status"

var log = logf.Log.WithName("sdk")

func MergeLabelsAndAnnotations(src, dest metav1.Object) {
	// allow users to add labels but not change ours. The operator supplies the src, so if someone altered dest it will get restored.
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

func MergeObject(desiredObj, currentObj runtime.Object, lastAppliedConfigAnnotation string) (runtime.Object, error) {
	desiredObj = desiredObj.DeepCopyObject()
	desiredMetaObj := desiredObj.(metav1.Object)
	currentMetaObj := currentObj.(metav1.Object)

	v, ok := currentMetaObj.GetAnnotations()[lastAppliedConfigAnnotation]
	if !ok {
		log.Info("Resource missing last applied config", "resource", currentMetaObj)
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

	result := NewDefaultInstance(currentObj)
	if err = json.Unmarshal(newCurrent, result); err != nil {
		return nil, err
	}

	return result, nil
}

func StripStatusFromObject(obj runtime.Object) (runtime.Object, error) {
	modified, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	modified, err = StripStatusByte(modified)
	if err != nil {
		return nil, err
	}
	result := NewDefaultInstance(obj)
	if err = json.Unmarshal(modified, result); err != nil {
		return nil, err
	}

	return result, nil
}

func StripStatusByte(in []byte) ([]byte, error) {
	var result map[string]interface{}
	json.Unmarshal(in, &result)

	if _, ok := result[statusKey]; ok {
		delete(result, statusKey)
	}
	if _, ok := result[capitalStatusKey]; ok {
		delete(result, capitalStatusKey)
	}
	return json.Marshal(result)
}

func DeployClusterResources() bool {
	return strings.ToLower(os.Getenv("DEPLOY_CLUSTER_RESOURCES")) != "false"
}

func LogJSONDiff(logger logr.Logger, objA, objB interface{}) {
	aBytes, _ := json.Marshal(objA)
	bBytes, _ := json.Marshal(objB)
	patches, _ := jsondiff.CreatePatch(aBytes, bBytes)
	pBytes, _ := json.Marshal(patches)
	logger.Info("DIFF", "obj", objA, "patch", string(pBytes))
}

func CheckDeploymentReady(deployment *appsv1.Deployment) bool {
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

func NewDefaultInstance(obj runtime.Object) runtime.Object {
	typ := reflect.ValueOf(obj).Elem().Type()
	return reflect.New(typ).Interface().(runtime.Object)
}

func ContainsStringValue(values []string, value string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}

func IsMutable(obj runtime.Object) bool {
	switch obj.(type) {
	case *v1.ConfigMap, *v1.Secret:
		return true
	}
	return false
}

func SetLabel(key, value string, obj metav1.Object) {
	if obj.GetLabels() == nil {
		obj.SetLabels(make(map[string]string))
	}
	obj.GetLabels()[key] = value
}

func SameResource(obj1, obj2 runtime.Object) bool {
	metaObj1 := obj1.(metav1.Object)
	metaObj2 := obj2.(metav1.Object)

	if reflect.TypeOf(obj1) != reflect.TypeOf(obj2) ||
		metaObj1.GetNamespace() != metaObj2.GetNamespace() ||
		metaObj1.GetName() != metaObj2.GetName() {
		return false
	}

	return true
}

// SetLastAppliedConfiguration writes last applied configuration to given annotation
func SetLastAppliedConfiguration(obj metav1.Object, lastAppliedConfigAnnotation string) error {
	bytes, err := json.Marshal(obj)
	if err != nil {
		return err
	}

	if obj.GetAnnotations() == nil {
		obj.SetAnnotations(make(map[string]string))
	}

	obj.GetAnnotations()[lastAppliedConfigAnnotation] = string(bytes)

	return nil
}

// GetOperatorToplevel returns the top level source directory of the operator.
// Can be overridden using the environment variable "OPERATOR_DIR".
func GetOperatorToplevel() string {
	// When running unit tests, we pass the OPERATOR_DIR environment variable, because
	// the tests run in their own directory and module.
	cwd := os.Getenv("OPERATOR_DIR")
	if cwd == "" {
		cwd, _ = os.Getwd()
	}
	return cwd
}
