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

	"github.com/emicklei/go-restful/v3"
	"github.com/pkg/errors"

	k8sv1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8serrors "k8s.io/apimachinery/pkg/util/errors"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	storageutils "kubevirt.io/kubevirt/pkg/storage/utils"
)

const (
	virtLauncherLabel = "kubevirt.io=virt-launcher"
	domainAnnotation  = "kubevirt.io/domain"
)

type ObjectGraph struct {
	client   kubecli.KubevirtClient
	graphMap map[string]schema.GroupResource
}

func NewObjectGraph(client kubecli.KubevirtClient) *ObjectGraph {
	return &ObjectGraph{
		client:   client,
		graphMap: objectGraphMap,
	}
}

// objectGraphMap represents the graph of objects that can be potentially related to a KubeVirt resource
// This is not strictly needed, but helps to keep the list of objects that we are appending to the graph.
var objectGraphMap = map[string]schema.GroupResource{
	"virtualmachines":                    {Group: "kubevirt.io", Resource: "virtualmachines"},
	"virtualmachineinstances":            {Group: "kubevirt.io", Resource: "virtualmachineinstances"},
	"virtualmachineinstancetypes":        {Group: "instancetype.kubevirt.io", Resource: "virtualmachineinstancetypes"},
	"virtualmachineclusterinstancetypes": {Group: "instancetype.kubevirt.io", Resource: "virtualmachineclusterinstancetypes"},
	"virtualmachinepreferences":          {Group: "instancetype.kubevirt.io", Resource: "virtualmachinepreferences"},
	"virtualmachineclusterpreferences":   {Group: "instancetype.kubevirt.io", Resource: "virtualmachineclusterpreferences"},
	"datavolumes":                        {Group: "cdi.kubevirt.io", Resource: "datavolumes"},
	"controllerrevisions":                {Group: "apps", Resource: "controllerrevisions"},
	"configmaps":                         {Group: "", Resource: "configmaps"},
	"persistentvolumeclaims":             {Group: "", Resource: "persistentvolumeclaims"},
	"serviceaccounts":                    {Group: "", Resource: "serviceaccounts"},
	"secrets":                            {Group: "", Resource: "secrets"},
	"pods":                               {Group: "", Resource: "pods"},
}

func (app *SubresourceAPIApp) handleObjectGraph(request *restful.Request, response *restful.Response, fetchFunc func(string, string) (interface{}, *apierrors.StatusError)) {
	name := request.PathParameter("name")
	namespace := request.PathParameter("namespace")

	if !app.clusterConfig.ObjectGraphEnabled() {
		writeError(apierrors.NewBadRequest("ObjectGraph feature gate not enabled: Unable to return object graph."), response)
		return
	}

	obj, statErr := fetchFunc(namespace, name)
	if statErr != nil {
		writeError(statErr, response)
		return
	}

	graph, err := NewObjectGraph(app.virtCli).GetObjectGraph(obj)
	if err != nil {
		writeError(apierrors.NewInternalError(err), response)
		return
	}

	if err := response.WriteEntity(graph); err != nil {
		log.Log.Reason(err).Error("Failed to write HTTP response.")
	}
}

func (app *SubresourceAPIApp) VMIObjectGraph(request *restful.Request, response *restful.Response) {
	app.handleObjectGraph(request, response, func(ns, name string) (interface{}, *apierrors.StatusError) {
		return app.FetchVirtualMachineInstance(ns, name)
	})
}

func (app *SubresourceAPIApp) VMObjectGraph(request *restful.Request, response *restful.Response) {
	app.handleObjectGraph(request, response, func(ns, name string) (interface{}, *apierrors.StatusError) {
		return app.fetchVirtualMachine(name, ns)
	})
}

func (og *ObjectGraph) newGraphNode(name, namespace, resource string, children []v1.ObjectGraphNode, optional bool) *v1.ObjectGraphNode {
	if groupResource, ok := og.graphMap[resource]; ok {
		node := &v1.ObjectGraphNode{
			ObjectReference: k8sv1.TypedObjectReference{
				Name:      name,
				Namespace: &namespace,
				APIGroup:  &groupResource.Group,
				Kind:      groupResource.Resource,
			},
			Children: children,
		}
		if optional {
			node.Optional = &optional
		}
		return node
	}
	return nil
}

func (og *ObjectGraph) GetObjectGraph(obj interface{}) (v1.ObjectGraphNode, error) {
	switch obj := obj.(type) {
	case *v1.VirtualMachine:
		return og.virtualMachineObjectGraph(obj)
	case *v1.VirtualMachineInstance:
		return og.virtualMachineInstanceObjectGraph(obj)
	default:
		return v1.ObjectGraphNode{}, nil
	}
}

func (og *ObjectGraph) virtualMachineObjectGraph(vm *v1.VirtualMachine) (v1.ObjectGraphNode, error) {
	children, errs := og.buildChildrenFromVM(vm)
	root := og.newGraphNode(vm.GetName(), vm.GetNamespace(), "virtualmachines", children, false)
	if root == nil {
		return v1.ObjectGraphNode{}, errors.New("could not create root graph node")
	}
	return *root, k8serrors.NewAggregate(errs)
}

func (og *ObjectGraph) virtualMachineInstanceObjectGraph(vmi *v1.VirtualMachineInstance) (v1.ObjectGraphNode, error) {
	children, errs := og.buildChildrenFromVMI(vmi)
	root := og.newGraphNode(vmi.GetName(), vmi.GetNamespace(), "virtualmachineinstances", children, false)
	if root == nil {
		return v1.ObjectGraphNode{}, errors.New("could not create root graph node")
	}
	return *root, k8serrors.NewAggregate(errs)
}

func (og *ObjectGraph) buildChildrenFromVM(vm *v1.VirtualMachine) ([]v1.ObjectGraphNode, []error) {
	var children []v1.ObjectGraphNode
	var errs []error

	if vm.Status.Created {
		vmiNode := og.newGraphNode(vm.GetName(), vm.GetNamespace(), "virtualmachineinstances", nil, false)
		if vmiNode != nil {
			if podNode, err := og.getLauncherPodNode(vm.GetName(), vm.GetNamespace()); err == nil && podNode != nil {
				vmiNode.Children = append(vmiNode.Children, *podNode)
			} else if err != nil {
				errs = append(errs, err)
			}
			children = append(children, *vmiNode)
		}
	}

	children = append(children, og.getInstanceTypeNode(vm)...)
	children = append(children, og.getPreferenceNode(vm)...)
	children = append(children, og.getAccessCredentialNodes(vm.Spec.Template.Spec.AccessCredentials, vm.GetNamespace())...)

	volumeNodes, err := og.addVolumeGraph(vm, vm.GetNamespace())
	children = append(children, volumeNodes...)
	errs = append(errs, err)

	return children, errs
}

func (og *ObjectGraph) buildChildrenFromVMI(vmi *v1.VirtualMachineInstance) ([]v1.ObjectGraphNode, []error) {
	var children []v1.ObjectGraphNode
	var errs []error

	if podNode, err := og.getLauncherPodNode(vmi.GetName(), vmi.GetNamespace()); err == nil && podNode != nil {
		children = append(children, *podNode)
	} else if err != nil {
		errs = append(errs, err)
	}

	children = append(children, og.getAccessCredentialNodes(vmi.Spec.AccessCredentials, vmi.GetNamespace())...)
	volumeNodes, err := og.addVolumeGraph(vmi, vmi.GetNamespace())
	children = append(children, volumeNodes...)
	errs = append(errs, err)

	return children, errs
}

func (og *ObjectGraph) addVolumeGraph(obj interface{}, namespace string) ([]v1.ObjectGraphNode, error) {
	var nodes []v1.ObjectGraphNode
	volumes, err := storageutils.GetVolumes(obj, og.client, storageutils.WithAllVolumes)
	if err != nil && !storageutils.IsErrNoBackendPVC(err) {
		return nil, err
	}
	for _, volume := range volumes {
		switch {
		case volume.DataVolume != nil:
			child := og.newGraphNode(volume.DataVolume.Name, namespace, "persistentvolumeclaims", nil, false)
			node := og.newGraphNode(volume.DataVolume.Name, namespace, "datavolumes", []v1.ObjectGraphNode{*child}, false)
			if node != nil {
				nodes = append(nodes, *node)
			}
		case volume.PersistentVolumeClaim != nil:
			node := og.newGraphNode(volume.PersistentVolumeClaim.ClaimName, namespace, "persistentvolumeclaims", nil, false)
			if node != nil {
				nodes = append(nodes, *node)
			}
		case volume.ConfigMap != nil:
			node := og.newGraphNode(volume.ConfigMap.Name, namespace, "configmaps", nil, false)
			if node != nil {
				nodes = append(nodes, *node)
			}
		case volume.Secret != nil:
			node := og.newGraphNode(volume.Secret.SecretName, namespace, "secrets", nil, false)
			if node != nil {
				nodes = append(nodes, *node)
			}
		case volume.CloudInitNoCloud != nil:
			if volume.CloudInitNoCloud.UserDataSecretRef != nil {
				node := og.newGraphNode(volume.CloudInitNoCloud.UserDataSecretRef.Name, namespace, "secrets", nil, false)
				if node != nil {
					nodes = append(nodes, *node)
				}
			}
			if volume.CloudInitNoCloud.NetworkDataSecretRef != nil {
				node := og.newGraphNode(volume.CloudInitNoCloud.NetworkDataSecretRef.Name, namespace, "secrets", nil, false)
				if node != nil {
					nodes = append(nodes, *node)
				}
			}
		case volume.CloudInitConfigDrive != nil:
			if volume.CloudInitConfigDrive.UserDataSecretRef != nil {
				node := og.newGraphNode(volume.CloudInitConfigDrive.UserDataSecretRef.Name, namespace, "secrets", nil, false)
				if node != nil {
					nodes = append(nodes, *node)
				}
			}
			if volume.CloudInitConfigDrive.NetworkDataSecretRef != nil {
				node := og.newGraphNode(volume.CloudInitConfigDrive.NetworkDataSecretRef.Name, namespace, "secrets", nil, false)
				if node != nil {
					nodes = append(nodes, *node)
				}
			}
		}
	}
	return nodes, err
}

func (og *ObjectGraph) getLauncherPodNode(name, namespace string) (*v1.ObjectGraphNode, error) {
	pods, err := og.client.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: virtLauncherLabel,
	})
	if err != nil {
		return nil, err
	}
	for _, pod := range pods.Items {
		if pod.Annotations[domainAnnotation] == name {
			return og.newGraphNode(pod.GetName(), namespace, "pods", nil, false), nil
		}
	}
	return nil, nil
}

func (og *ObjectGraph) getAccessCredentialNodes(acs []v1.AccessCredential, namespace string) []v1.ObjectGraphNode {
	var nodes []v1.ObjectGraphNode
	for _, ac := range acs {
		if ac.SSHPublicKey != nil && ac.SSHPublicKey.Source.Secret != nil {
			nodes = append(nodes, *og.newGraphNode(ac.SSHPublicKey.Source.Secret.SecretName, namespace, "secrets", nil, false))
		} else if ac.UserPassword != nil && ac.UserPassword.Source.Secret != nil {
			nodes = append(nodes, *og.newGraphNode(ac.UserPassword.Source.Secret.SecretName, namespace, "secrets", nil, false))
		}
	}
	return nodes
}

func (og *ObjectGraph) getInstanceTypeNode(vm *v1.VirtualMachine) []v1.ObjectGraphNode {
	if vm.Spec.Instancetype != nil {
		return og.getInstanceTypeMatcherResource(vm.Spec.Instancetype, vm.Status.InstancetypeRef, vm.GetNamespace())
	}
	return nil
}

func (og *ObjectGraph) getPreferenceNode(vm *v1.VirtualMachine) []v1.ObjectGraphNode {
	if vm.Spec.Preference != nil {
		return og.getInstanceTypeMatcherResource(vm.Spec.Preference, vm.Status.PreferenceRef, vm.GetNamespace())
	}
	return nil
}

func (og *ObjectGraph) getInstanceTypeMatcherResource(matcher v1.Matcher, statusRef *v1.InstancetypeStatusRef, namespace string) []v1.ObjectGraphNode {
	var nodes []v1.ObjectGraphNode
	switch resource := strings.ToLower(matcher.GetKind()) + "s"; resource {
	case "virtualmachineclusterinstancetypes", "virtualmachineclusterpreferences":
		nodes = append(nodes, *og.newGraphNode(matcher.GetName(), "", resource, nil, true))
	case "virtualmachineinstancetypes", "virtualmachinepreferences":
		nodes = append(nodes, *og.newGraphNode(matcher.GetName(), namespace, resource, nil, true))
	}
	if statusRef != nil && statusRef.ControllerRevisionRef != nil {
		nodes = append(nodes, *og.newGraphNode(statusRef.ControllerRevisionRef.Name, namespace, "controllerrevisions", nil, false))
	}
	// Handle cases where the VM Status hasn't been populated yet by falling back to any spec provided RevisionName
	if statusRef == nil && matcher.GetRevisionName() != "" {
		nodes = append(nodes, *og.newGraphNode(matcher.GetRevisionName(), namespace, "controllerrevisions", nil, false))
	}
	return nodes
}

// TODO: Add more as needed
