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
 * Copyright The KubeVirt Authors.
 *
 */

package rest

import (
	"context"
	"strings"

	"github.com/pkg/errors"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8serrors "k8s.io/apimachinery/pkg/util/errors"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	storageutils "kubevirt.io/kubevirt/pkg/storage/utils"
)

type ObjectGraph struct {
	client   kubecli.KubevirtClient
	graphMap map[string]schema.GroupResource
}

// objectGraphMap represents the graph of objects that can be potentially related to a KubeVirt resource
var objectGraphMap = map[string]schema.GroupResource{
	"virtualmachineinstances":           {Group: "kubevirt.io", Resource: "virtualmachineinstances"},
	"virtualmachineinstancetype":        {Group: "instancetype.kubevirt.io", Resource: "virtualmachineinstancetype"},
	"virtualmachineclusterinstancetype": {Group: "instancetype.kubevirt.io", Resource: "virtualmachineclusterinstancetype"},
	"virtualmachinepreference":          {Group: "instancetype.kubevirt.io", Resource: "virtualmachinepreference"},
	"virtualmachineclusterpreference":   {Group: "instancetype.kubevirt.io", Resource: "virtualmachineclusterpreference"},
	"datavolumes":                       {Group: "cdi.kubevirt.io", Resource: "datavolumes"},
	"controllerrevisions":               {Group: "apps", Resource: "controllerrevisions"},
	"configmaps":                        {Group: "", Resource: "configmaps"},
	"persistentvolumeclaims":            {Group: "", Resource: "persistentvolumeclaims"},
	"serviceaccounts":                   {Group: "", Resource: "serviceaccounts"},
	"secrets":                           {Group: "", Resource: "secrets"},
	"pods":                              {Group: "", Resource: "pods"},
}

func NewObjectGraph(client kubecli.KubevirtClient) *ObjectGraph {
	return &ObjectGraph{
		client:   client,
		graphMap: objectGraphMap,
	}
}

func (og *ObjectGraph) addGraphNode(name, namespace, resource string, resources []v1.ObjectGraphNode) []v1.ObjectGraphNode {
	if groupResource, ok := og.graphMap[resource]; ok {
		resources = append(resources, v1.ObjectGraphNode{
			ObjectReference: k8sv1.TypedObjectReference{
				Name:      name,
				Namespace: &namespace,
				APIGroup:  &groupResource.Group,
				Kind:      groupResource.Resource,
			},
		})
	}
	return resources
}

func (og *ObjectGraph) GetObjectGraph(obj interface{}) (v1.ObjectGraphNodeList, error) {
	switch obj := obj.(type) {
	case *v1.VirtualMachine:
		return og.virtualMachineObjectGraph(obj)
	case *v1.VirtualMachineInstance:
		return og.virtualMachineInstanceObjectGraph(obj)
	default:
		// No specific backup graph for the passed object
		return v1.ObjectGraphNodeList{}, nil
	}
}

func (og *ObjectGraph) virtualMachineObjectGraph(vm *v1.VirtualMachine) (v1.ObjectGraphNodeList, error) {
	var resources []v1.ObjectGraphNode
	var err error
	namespace := vm.GetNamespace()

	if vm.Spec.Instancetype != nil {
		resources = og.addInstanceType(*vm.Spec.Instancetype, vm.GetNamespace(), resources)
	}
	if vm.Spec.Preference != nil {
		resources = og.addPreferenceType(*vm.Spec.Preference, vm.GetNamespace(), resources)
	}

	var errs []error
	if vm.Status.Created {
		resources = og.addGraphNode(vm.GetName(), namespace, "virtualmachineinstances", resources)
		resources, err = og.addLauncherPod(vm.GetName(), vm.GetNamespace(), resources)
		if err != nil {
			errs = append(errs, err)
		}
	}

	resources = og.addAccessCredentials(vm.Spec.Template.Spec.AccessCredentials, vm.GetNamespace(), resources)
	resources, err = og.addVolumeGraph(vm, vm.GetNamespace(), resources)
	if err != nil {
		errs = append(errs, err)
	}

	objectGraph := v1.ObjectGraphNodeList{
		Items: resources,
	}

	if len(errs) > 0 {
		return objectGraph, k8serrors.NewAggregate(errs)
	}

	return objectGraph, nil
}

func (og *ObjectGraph) virtualMachineInstanceObjectGraph(vmi *v1.VirtualMachineInstance) (v1.ObjectGraphNodeList, error) {
	var resources []v1.ObjectGraphNode
	var errs []error

	resources, err := og.addLauncherPod(vmi.GetName(), vmi.GetNamespace(), resources)
	if err != nil {
		errs = append(errs, err)
	}
	resources = og.addAccessCredentials(vmi.Spec.AccessCredentials, vmi.GetNamespace(), resources)
	resources, err = og.addVolumeGraph(vmi, vmi.GetNamespace(), resources)
	if err != nil {
		errs = append(errs, err)
	}

	objectGraph := v1.ObjectGraphNodeList{
		Items: resources,
	}

	if len(errs) > 0 {
		return objectGraph, k8serrors.NewAggregate(errs)
	}

	return objectGraph, nil
}

func (og *ObjectGraph) addVolumeGraph(obj interface{}, namespace string, resources []v1.ObjectGraphNode) ([]v1.ObjectGraphNode, error) {
	volumes, err := storageutils.GetVolumes(obj, og.client, storageutils.WithAllVolumes)
	if err != nil {
		return resources, err
	}

	for _, volume := range volumes {
		switch {
		case volume.DataVolume != nil:
			resources = og.addGraphNode(volume.DataVolume.Name, namespace, "datavolumes", resources)
			resources = og.addGraphNode(volume.DataVolume.Name, namespace, "persistentvolumeclaims", resources)
		case volume.PersistentVolumeClaim != nil:
			resources = og.addGraphNode(volume.PersistentVolumeClaim.ClaimName, namespace, "persistentvolumeclaims", resources)
		case volume.MemoryDump != nil:
			resources = og.addGraphNode(volume.MemoryDump.ClaimName, namespace, "persistentvolumeclaims", resources)
		case volume.ConfigMap != nil:
			resources = og.addGraphNode(volume.ConfigMap.Name, namespace, "configmaps", resources)
		case volume.Secret != nil:
			resources = og.addGraphNode(volume.Secret.SecretName, namespace, "secrets", resources)
		case volume.ServiceAccount != nil:
			resources = og.addGraphNode(volume.ServiceAccount.ServiceAccountName, namespace, "serviceaccounts", resources)
		}
	}
	return resources, err
}

func (og *ObjectGraph) addAccessCredentials(acs []v1.AccessCredential, namespace string, resources []v1.ObjectGraphNode) []v1.ObjectGraphNode {
	for _, ac := range acs {
		if ac.SSHPublicKey != nil && ac.SSHPublicKey.Source.Secret != nil {
			resources = og.addGraphNode(ac.SSHPublicKey.Source.Secret.SecretName, namespace, "secrets", resources)
		} else if ac.UserPassword != nil && ac.UserPassword.Source.Secret != nil {
			resources = og.addGraphNode(ac.UserPassword.Source.Secret.SecretName, namespace, "secrets", resources)
		}
	}
	return resources
}

func (og *ObjectGraph) addLauncherPod(vmiName, vmiNamespace string, resources []v1.ObjectGraphNode) ([]v1.ObjectGraphNode, error) {
	pods, err := og.client.CoreV1().Pods(vmiNamespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: "kubevirt.io=virt-launcher",
	})
	if err != nil {
		return resources, errors.Wrapf(err, "failed to get launcher pod from VMI %s/%s", vmiNamespace, vmiName)
	}

	for _, pod := range pods.Items {
		if pod.Annotations["kubevirt.io/domain"] == vmiName {
			return og.addGraphNode(pod.GetName(), vmiNamespace, "pods", resources), nil
		}
	}

	return resources, nil
}

func (og *ObjectGraph) addInstanceType(instanceType v1.InstancetypeMatcher, namespace string, resources []v1.ObjectGraphNode) []v1.ObjectGraphNode {
	instanceKind := strings.ToLower(instanceType.Kind)
	switch instanceKind {
	case "virtualmachineclusterinstancetype":
		resources = og.addGraphNode(instanceType.Name, "", instanceKind, resources)
	case "virtualmachineinstancetype":
		resources = og.addGraphNode(instanceType.Name, namespace, instanceKind, resources)
	}
	resources = og.addGraphNode(instanceType.RevisionName, namespace, "controllerrevisions", resources)
	return resources
}

func (og *ObjectGraph) addPreferenceType(preference v1.PreferenceMatcher, namespace string, resources []v1.ObjectGraphNode) []v1.ObjectGraphNode {
	preferenceKind := strings.ToLower(preference.Kind)
	switch preferenceKind {
	case "virtualmachineclusterpreference":
		resources = og.addGraphNode(preference.Name, "", preferenceKind, resources)
	case "virtualmachinepreference":
		resources = og.addGraphNode(preference.Name, namespace, preferenceKind, resources)
	}
	resources = og.addGraphNode(preference.RevisionName, namespace, "controllerrevisions", resources)
	return resources
}
