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
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"kubevirt.io/client-go/kubecli"
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

// DeleteAllObjects deletes a collection with the given selector
func DeleteAllObjects(virtCli kubecli.KubevirtClient, resourceKind string, listOpts *metav1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}

	// get all objects with the benchmark label
	result := &unstructured.Unstructured{}
	err := virtCli.RestClient().Get().
		Resource(resourceKind).
		VersionedParams(listOpts, metav1.ParameterCodec).
		Timeout(timeout).
		Do(context.Background()).
		Into(result)
	if err != nil {
		return err
	}

	list, err := result.ToList()
	if err != nil {
		return err
	}
	for _, obj := range list.Items {
		err := virtCli.RestClient().Delete().
			Namespace(obj.GetNamespace()).
			Resource(resourceKind).
			Name(obj.GetName()).
			Body(&metav1.DeleteOptions{}).
			Do(context.Background()).
			Error()
		if !errors.IsNotFound(err) {
			return err
		}
	}
	return nil
}

func AddLabels(obj *unstructured.Unstructured, uuid string) {
	labels := map[string]string{
		config.WorkloadLabel: uuid,
	}
	obj.SetLabels(labels)
}
