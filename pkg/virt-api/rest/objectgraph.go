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
	"fmt"
	"strings"

	"github.com/emicklei/go-restful/v3"
	"github.com/pkg/errors"

	k8sv1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8serrors "k8s.io/apimachinery/pkg/util/errors"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	storageutils "kubevirt.io/kubevirt/pkg/storage/utils"
)

const (
	// ObjectGraphDependencyLabel is used to specify the type of dependency for a node in the object graph.
	// Possible values are: "storage", "network", "compute", "config".
	ObjectGraphDependencyLabel = "kubevirt.io/dependency-type"

	// TODO: Add support for additional labels. For example "backup-mandatory", "restore-mandatory", etc.
)

// DependencyType represents the type of dependency in the object graph
type DependencyType string

const (
	DependencyTypeStorage DependencyType = "storage"
	DependencyTypeNetwork DependencyType = "network"
	DependencyTypeCompute DependencyType = "compute"
	DependencyTypeConfig  DependencyType = "config"
)

type ObjectGraph struct {
	client   kubecli.KubevirtClient
	graphMap map[string]schema.GroupKind
	options  *v1.ObjectGraphOptions
}

func NewObjectGraph(client kubecli.KubevirtClient, opts *v1.ObjectGraphOptions) *ObjectGraph {
	return &ObjectGraph{
		client:   client,
		graphMap: objectGraphMap,
		options:  opts,
	}
}

// objectGraphMap represents the graph of objects that can be potentially related to a KubeVirt resource
// This is not strictly needed, but helps to keep the list of objects that we are appending to the graph.
var objectGraphMap = map[string]schema.GroupKind{
	"virtualmachines":                    {Group: "kubevirt.io", Kind: "VirtualMachine"},
	"virtualmachineinstances":            {Group: "kubevirt.io", Kind: "VirtualMachineInstance"},
	"virtualmachineinstancetypes":        {Group: "instancetype.kubevirt.io", Kind: "VirtualMachineInstancetype"},
	"virtualmachineclusterinstancetypes": {Group: "instancetype.kubevirt.io", Kind: "VirtualMachineClusterInstancetype"},
	"virtualmachinepreferences":          {Group: "instancetype.kubevirt.io", Kind: "VirtualMachinePreference"},
	"virtualmachineclusterpreferences":   {Group: "instancetype.kubevirt.io", Kind: "VirtualMachineClusterPreference"},
	"datavolumes":                        {Group: "cdi.kubevirt.io", Kind: "DataVolume"},
	"controllerrevisions":                {Group: "apps", Kind: "ControllerRevision"},
	"configmaps":                         {Group: "", Kind: "ConfigMap"},
	"persistentvolumeclaims":             {Group: "", Kind: "PersistentVolumeClaim"},
	"serviceaccounts":                    {Group: "", Kind: "ServiceAccount"},
	"secrets":                            {Group: "", Kind: "Secret"},
	"pods":                               {Group: "", Kind: "Pod"},
	"networkattachmentdefinitions":       {Group: "k8s.cni.cncf.io", Kind: "NetworkAttachmentDefinition"},
	"ipamclaims":                         {Group: "k8s.cni.cncf.io", Kind: "IPAMClaim"},
}

// getResourceDependencyType returns the dependency type for a given resource
func getResourceDependencyType(resource string) DependencyType {
	switch resource {
	case "datavolumes", "persistentvolumeclaims":
		return DependencyTypeStorage
	case "pods", "virtualmachines", "virtualmachineinstances":
		return DependencyTypeCompute
	case "configmaps", "secrets", "virtualmachineinstancetypes", "virtualmachineclusterinstancetypes",
		"virtualmachinepreferences", "virtualmachineclusterpreferences", "controllerrevisions":
		return DependencyTypeConfig
	case "serviceaccounts", "networkattachmentdefinitions", "ipamclaims":
		return DependencyTypeNetwork
	default:
		return DependencyTypeCompute
	}
}

func (app *SubresourceAPIApp) handleObjectGraph(request *restful.Request, response *restful.Response, fetchFunc func(string, string) (any, *apierrors.StatusError)) {
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

	objectGraphOpts := &v1.ObjectGraphOptions{}
	defer request.Request.Body.Close()
	if err := decodeBody(request, objectGraphOpts); err != nil {
		writeError(err, response)
		return
	}

	graph, err := NewObjectGraph(app.virtCli, objectGraphOpts).GetObjectGraph(obj)
	if err != nil {
		writeError(apierrors.NewInternalError(err), response)
		return
	}

	if err := response.WriteEntity(graph); err != nil {
		log.Log.Reason(err).Error("Failed to write HTTP response.")
	}
}

func (app *SubresourceAPIApp) VMIObjectGraph(request *restful.Request, response *restful.Response) {
	app.handleObjectGraph(request, response, func(ns, name string) (any, *apierrors.StatusError) {
		return app.FetchVirtualMachineInstance(ns, name)
	})
}

func (app *SubresourceAPIApp) VMObjectGraph(request *restful.Request, response *restful.Response) {
	app.handleObjectGraph(request, response, func(ns, name string) (any, *apierrors.StatusError) {
		return app.fetchVirtualMachine(name, ns)
	})
}

func (og *ObjectGraph) newGraphNode(name, namespace, resource string, children []v1.ObjectGraphNode, optional bool) *v1.ObjectGraphNode {
	groupKind, ok := og.graphMap[resource]
	if !ok {
		return nil
	}

	nodeLabels := map[string]string{
		ObjectGraphDependencyLabel: string(getResourceDependencyType(resource)),
	}

	node := &v1.ObjectGraphNode{
		ObjectReference: k8sv1.TypedObjectReference{
			Name:      name,
			Namespace: &namespace,
			APIGroup:  &groupKind.Group,
			Kind:      groupKind.Kind,
		},
		Labels:   nodeLabels,
		Children: children,
	}

	if optional {
		node.Optional = &optional
	}

	return node
}

func (og *ObjectGraph) shouldIncludeNode(node *v1.ObjectGraphNode) bool {
	if og.options == nil {
		return true
	}

	// Exclude optional nodes only if IncludeOptionalNodes is explicitly set to false
	if node.Optional != nil && *node.Optional &&
		og.options.IncludeOptionalNodes != nil && !*og.options.IncludeOptionalNodes {
		return false
	}

	if og.options.LabelSelector != nil {
		selector, err := metav1.LabelSelectorAsSelector(og.options.LabelSelector)
		if err != nil {
			log.Log.Reason(err).Error("Invalid label selector")
			return true // Include node if selector is invalid
		}

		if !selector.Matches(labels.Set(node.Labels)) {
			return false
		}
	}

	return true
}

func (og *ObjectGraph) filterNodes(nodes []v1.ObjectGraphNode) []v1.ObjectGraphNode {
	filtered := make([]v1.ObjectGraphNode, 0, len(nodes))

	for _, node := range nodes {
		if og.shouldIncludeNode(&node) {
			node.Children = og.filterNodes(node.Children)
			log.Log.V(3).Infof("Including node: %s/%s (%s) in the object graph", *node.ObjectReference.Namespace, node.ObjectReference.Name, node.ObjectReference.Kind)
			filtered = append(filtered, node)
		}
	}

	return filtered
}

func (og *ObjectGraph) GetObjectGraph(obj any) (v1.ObjectGraphNode, error) {
	var root v1.ObjectGraphNode
	var err error

	switch obj := obj.(type) {
	case *v1.VirtualMachine:
		root, err = og.virtualMachineObjectGraph(obj)
	case *v1.VirtualMachineInstance:
		root, err = og.virtualMachineInstanceObjectGraph(obj)
	default:
		return v1.ObjectGraphNode{}, nil
	}

	if err != nil {
		return root, err
	}

	root.Children = og.filterNodes(root.Children)
	return root, nil
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

	// Main storage nodes
	volumeNodes, err := og.addVolumeGraph(vm, vm.GetNamespace())
	children = append(children, volumeNodes...)
	errs = append(errs, err)
	// Main network nodes
	networkNodes, err := og.handleNetworkNodes(vm.Spec.Template.Spec, vm.GetName(), vm.GetNamespace())
	children = append(children, networkNodes...)
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
	// Main storage nodes
	volumeNodes, err := og.addVolumeGraph(vmi, vmi.GetNamespace())
	children = append(children, volumeNodes...)
	errs = append(errs, err)
	// Main network nodes
	networkNodes, err := og.handleNetworkNodes(vmi.Spec, vmi.GetName(), vmi.GetNamespace())
	children = append(children, networkNodes...)
	errs = append(errs, err)

	return children, errs
}

func (og *ObjectGraph) addVolumeGraph(obj any, namespace string) ([]v1.ObjectGraphNode, error) {
	var nodes []v1.ObjectGraphNode
	volumes, err := storageutils.GetVolumes(obj, og.client, storageutils.WithAllVolumes)
	if err != nil {
		if !storageutils.IsErrNoBackendPVC(err) {
			return nil, err
		}
		err = fmt.Errorf("failed to get backend volume (%w), VM might still be provisioning. Proceeding with the remaining volumes", err)
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
		LabelSelector: fmt.Sprintf("%s=%s", v1.AppLabel, "virt-launcher"),
	})
	if err != nil {
		return nil, err
	}
	for _, pod := range pods.Items {
		for _, ownerRef := range pod.OwnerReferences {
			if ownerRef.Kind == "VirtualMachineInstance" && ownerRef.Name == name {
				return og.newGraphNode(pod.GetName(), namespace, "pods", nil, false), nil
			}
		}
		// fallback to check annotations in case the owner reference is not set
		if pod.Annotations[v1.DomainAnnotation] == name {
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

func (og *ObjectGraph) handleNetworkNodes(vmiSpec v1.VirtualMachineInstanceSpec, vmName, namespace string) ([]v1.ObjectGraphNode, error) {
	var nodes []v1.ObjectGraphNode

	for _, net := range vmiSpec.Networks {
		if net.Multus != nil && net.Multus.NetworkName != "" {
			parts := strings.Split(net.Multus.NetworkName, "/")
			name := net.Multus.NetworkName
			nadNamespace := namespace

			if len(parts) == 2 {
				nadNamespace = parts[0]
				name = parts[1]
			}

			node := og.newGraphNode(name, nadNamespace, "networkattachmentdefinitions", nil, true)
			if node != nil {
				nodes = append(nodes, *node)
			}
		} else if net.Pod != nil {
			// TODO: Consider handling this case
			continue
		}
	}

	// IPAM Claims are important for cross cluster live migration because the fixed IP needs to be created in the target cluster.
	// Marking them as optional since these are madantory only for cross cluster live migration.
	if len(vmiSpec.Networks) > 0 {
		available, err := IsIPAMClaimAvailable(og.client)
		if err != nil || !available {
			log.Log.V(3).Info("IPAMClaims are not enabled in this cluster, skipping IPAMClaims nodes in the object graph")
			return nodes, err
		}
		ipamClaims, err := og.client.DynamicClient().Resource(schema.GroupVersionResource{
			Group:    "k8s.cni.cncf.io",
			Version:  "v1alpha1",
			Resource: "ipamclaims",
		}).Namespace(namespace).List(context.TODO(), metav1.ListOptions{
			// Following the label convention used in https://github.com/kubevirt/ipam-extensions.
			// This repo creates (and manage the lifecycle of) IPAMClaims on behalf of KubeVirt virtual machines.
			LabelSelector: fmt.Sprintf("%s=%s", v1.VirtualMachineLabel, vmName),
		})
		if meta.IsNoMatchError(err) {
			return nodes, nil
		}
		if err != nil {
			log.Log.Reason(err).Error("Failed to list IPAM claims")
			return nodes, err
		}
		for _, claim := range ipamClaims.Items {
			node := og.newGraphNode(claim.GetName(), namespace, "ipamclaims", nil, true)
			if node != nil {
				nodes = append(nodes, *node)
			}
		}
	}

	return nodes, nil
}

func IsIPAMClaimAvailable(client kubecli.KubevirtClient) (bool, error) {
	// Check if the IPAMClaims resource is available in the cluster
	_, err := client.Discovery().ServerResourcesForGroupVersion("k8s.cni.cncf.io/v1alpha1")
	if err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		return false, errors.Wrap(err, "failed to check IPAMClaims API availability")
	}
	return true, nil
}

// TODO: Add more as needed.
// For example vmexports, vmsnapshots, vmimigrations, etc.
