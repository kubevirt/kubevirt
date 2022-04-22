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
 * Copyright 2021 IBM, Inc.
 *
 */

package object

import (
	"context"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/tools/perfscale-load-generator/config"
)

func CreateObject(virtCli kubecli.KubevirtClient, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	result := &unstructured.Unstructured{}
	err := virtCli.RestClient().Post().
		Namespace(obj.GetNamespace()).
		Resource(GetObjectResource(obj)).
		Body(obj).
		Do(context.Background()).
		Into(result)
	return result, err
}

func DeleteObject(virtCli kubecli.KubevirtClient, obj unstructured.Unstructured, resourceKind string, gracePeriod int64) {
	err := virtCli.RestClient().Delete().
		Namespace(obj.GetNamespace()).
		Name(obj.GetName()).
		Resource(resourceKind).
		Body(&metav1.DeleteOptions{GracePeriodSeconds: &gracePeriod}).
		Do(context.Background()).Error()
	if err != nil && !errors.IsNotFound(err) {
		log.Log.V(2).Errorf("Error deleting obj %s %s: %v", resourceKind, obj.GetName(), err)
	}
	return
}

func ListObjects(virtCli kubecli.KubevirtClient, resourceKind string, listOpts *metav1.ListOptions, namespace string) (*unstructured.UnstructuredList, error) {
	result := &unstructured.UnstructuredList{}
	err := virtCli.RestClient().Get().
		Resource(resourceKind).
		Namespace(namespace).
		VersionedParams(listOpts, scheme.ParameterCodec).
		Do(context.Background()).
		Into(result)
	if err != nil {
		log.Log.V(3).Infof("error LISTing obj(s) %s", resourceKind)
		return nil, err
	}
	return result, err
}

func FindObject(virtCli kubecli.KubevirtClient, obj *config.ObjectSpec, count int) (*unstructured.Unstructured, string) {
	result := &unstructured.Unstructured{}
	for replica := 1; replica <= count; replica++ {
		templateData := GenerateObjectTemplateData(obj, replica)
		newObject, err := RenderObject(templateData, obj.ObjectTemplate)
		objType := GetObjectResource(newObject)
		if err != nil {
			log.Log.Errorf("error rendering obj: %v", err)
		}
		err = virtCli.RestClient().Get().
			Namespace(newObject.GetNamespace()).
			Name(newObject.GetName()).
			Resource(objType).
			Do(context.Background()).
			Into(result)
		if err != nil {
			log.Log.V(3).Errorf("Error matching object %s/%s", newObject.GetNamespace(), newObject.GetName())
		} else if result != nil {
			log.Log.V(3).Infof("Found matching object %s/%s", newObject.GetNamespace(), newObject.GetName())
			return result, objType
		}
		log.Log.V(3).Infof("Searching for matching object %s/%s to scrape job label", newObject.GetNamespace(), newObject.GetName())
	}
	log.Log.V(2).Infof("Didn't find any matching objects. The previous job had already been cleaned up")
	return nil, ""
}

// DeleteAllObjectsInNamespaces deletes a collection of objects in a set of namespace with a given selector
func DeleteAllObjectsInNamespaces(virtCli kubecli.KubevirtClient, resourceKind string, listOpts *metav1.ListOptions) {
	gracePeriod := int64(0)
	result, err := ListObjects(virtCli, resourceKind, listOpts, "")
	if err != nil {
		return
	}

	log.Log.V(3).Infof("Number of %s to delete: %d", resourceKind, len(result.Items))
	for _, item := range result.Items {
		log.Log.V(3).Infof("Deleting obj %s", item.GetName())
		DeleteObject(virtCli, item, resourceKind, gracePeriod)
	}
}
