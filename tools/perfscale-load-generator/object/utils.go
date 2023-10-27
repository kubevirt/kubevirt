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
 * Copyright the KubeVirt Authors.
 *
 */

package object

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/tools/perfscale-load-generator/config"
)

const (
	VMIResource           = "virtualmachineinstances"
	VMResource            = "virtualmachines"
	VMIReplicaSetResource = "virtualmachineinstancereplicasets"
)

// GetObjectResource returns the resource type of an object
// The kubevirt API understand the resource type as plural
func GetObjectResource(obj *unstructured.Unstructured) string {
	switch res := strings.ToLower(obj.GroupVersionKind().Kind); res {
	case "virtualmachineinstance":
		return VMIResource
	case "virtualmachine":
		return VMResource
	case "virtualmachineinstancereplicaset":
		return VMIReplicaSetResource
	default:
		return res
	}
}

// CreateObjectReplicaSpec returns the last created object to provies information for wait and delete the objects
func CreateObjectReplicaSpec(obj *config.ObjectSpec, objIdx *int, uuid string) (*unstructured.Unstructured, error) {
	var err error
	var newObject *unstructured.Unstructured
	templateData := GenerateObjectTemplateData(obj, *objIdx)
	if newObject, err = RenderObject(templateData, obj.ObjectTemplate); err != nil {
		return nil, fmt.Errorf("error rendering obj: %v", err)
	}
	*objIdx += 1
	config.AddLabels(newObject, uuid)
	return newObject, err
}

func CreateObjectReplica(client kubecli.KubevirtClient, objSpec *config.ObjectSpec, objIdx *int, uuid string) (*unstructured.Unstructured, error) {
	var err error
	var obj *unstructured.Unstructured
	if obj, err = CreateObjectReplicaSpec(objSpec, objIdx, uuid); err != nil {
		return nil, err
	}

	var newObject *unstructured.Unstructured
	if newObject, err := CreateObject(client, obj); err != nil {
		log.Log.Errorf("error creating obj %s: %v", newObject.GroupVersionKind().Kind, err)
		return nil, err
	}
	return newObject, nil
}
