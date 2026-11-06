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
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/emicklei/go-restful/v3"

	k8sv1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8serrors "k8s.io/apimachinery/pkg/util/errors"

	"k8s.io/client-go/kubernetes"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"
	templateapi "kubevirt.io/virt-template-api/core"
	templatev1beta1 "kubevirt.io/virt-template-api/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/pointer"
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

const (
	clusterInstancetypeResource = "virtualmachineclusterinstancetypes"
	clusterPreferenceResource   = "virtualmachineclusterpreferences"
)

type ObjectGraph struct {
	virtClient kubecli.KubevirtClient
	k8sClient  kubernetes.Interface
	graphMap   map[string]schema.GroupKind
	options    *v1.ObjectGraphOptions
}

func NewObjectGraph(virtClient kubecli.KubevirtClient, k8sClient kubernetes.Interface, opts *v1.ObjectGraphOptions) *ObjectGraph {
	return &ObjectGraph{
		virtClient: virtClient,
		k8sClient:  k8sClient,
		graphMap:   objectGraphMap,
		options:    opts,
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
	"datasources":                        {Group: "cdi.kubevirt.io", Kind: "DataSource"},
	"volumesnapshots":                    {Group: "snapshot.storage.k8s.io", Kind: "VolumeSnapshot"},
	templateapi.PluralResourceName:       {Group: templateapi.GroupName, Kind: "VirtualMachineTemplate"},
}

// getResourceDependencyType returns the dependency type for a given resource
func getResourceDependencyType(resource string) DependencyType {
	switch resource {
	case "datavolumes", "persistentvolumeclaims", "datasources", "volumesnapshots":
		return DependencyTypeStorage
	case "pods", "virtualmachines", "virtualmachineinstances":
		return DependencyTypeCompute
	case "configmaps", "secrets", "virtualmachineinstancetypes", "virtualmachineclusterinstancetypes",
		"virtualmachinepreferences", "virtualmachineclusterpreferences", "controllerrevisions", "serviceaccounts",
		templateapi.PluralResourceName:
		return DependencyTypeConfig
	case "networkattachmentdefinitions":
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

	graph, err := NewObjectGraph(app.virtClient, app.k8sClient, objectGraphOpts).GetObjectGraph(obj)
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

func (app *SubresourceAPIApp) VMTemplateObjectGraph(request *restful.Request, response *restful.Response) {
	app.handleObjectGraph(request, response, func(ns, name string) (any, *apierrors.StatusError) {
		return app.fetchVirtualMachineTemplate(ns, name)
	})
}

func (app *SubresourceAPIApp) fetchVirtualMachineTemplate(namespace, name string) (*templatev1beta1.VirtualMachineTemplate, *apierrors.StatusError) {
	if app.templateClient == nil {
		return nil, apierrors.NewServiceUnavailable("VirtualMachineTemplate support is not configured")
	}

	tpl, err := app.templateClient.TemplateV1beta1().VirtualMachineTemplates(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			groupResource := schema.GroupResource{Group: templateapi.GroupName, Resource: templateapi.SingularResourceName}
			return nil, apierrors.NewNotFound(groupResource, name)
		}
		return nil, apierrors.NewInternalError(fmt.Errorf("unable to retrieve VirtualMachineTemplate [%s]: %v", name, err))
	}
	return tpl, nil
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
	case *templatev1beta1.VirtualMachineTemplate:
		root, err = og.virtualMachineTemplateObjectGraph(obj)
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
	children, errs := og.buildChildrenFromVM(vm, storageutils.WithAllVolumes)
	root := og.newGraphNode(vm.GetName(), vm.GetNamespace(), "virtualmachines", children, false)
	if root == nil {
		return v1.ObjectGraphNode{}, errors.New("could not create root graph node")
	}
	return *root, k8serrors.NewAggregate(errs)
}

func (og *ObjectGraph) virtualMachineTemplateObjectGraph(tpl *templatev1beta1.VirtualMachineTemplate) (v1.ObjectGraphNode, error) {
	vm, err := decodeTemplate(tpl)
	if err != nil {
		return v1.ObjectGraphNode{}, err
	}

	children, errs := og.buildChildrenFromVM(vm, storageutils.WithRegularVolumes)
	dvTemplates := vm.Spec.DataVolumeTemplates

	markDataVolumeTemplateNodesOptional(children, dvTemplates)

	sourceNodes := og.dataVolumeTemplateSourceNodes(dvTemplates, vm.GetNamespace())
	children = append(children, sourceNodes...)
	children = resolveAndMarkTemplateParameters(children, tpl.Spec.Parameters)

	root := og.newGraphNode(tpl.GetName(), tpl.GetNamespace(), templateapi.PluralResourceName, children, false)
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

func (og *ObjectGraph) buildChildrenFromVM(vm *v1.VirtualMachine, volumeOpts ...storageutils.VolumeOption) ([]v1.ObjectGraphNode, []error) {
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
	volumeNodes, err := og.addVolumeGraph(vm, vm.GetNamespace(), volumeOpts...)
	children = append(children, volumeNodes...)
	errs = append(errs, err)
	// Main network nodes
	networkNodes := og.handleNetworkNodes(vm.Spec.Template.Spec, vm.GetName(), vm.GetNamespace())
	children = append(children, networkNodes...)

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
	volumeNodes, err := og.addVolumeGraph(vmi, vmi.GetNamespace(), storageutils.WithAllVolumes)
	children = append(children, volumeNodes...)
	errs = append(errs, err)
	// Main network nodes
	networkNodes := og.handleNetworkNodes(vmi.Spec, vmi.GetName(), vmi.GetNamespace())
	children = append(children, networkNodes...)

	return children, errs
}

func (og *ObjectGraph) addVolumeGraph(obj any, namespace string, volumeOpts ...storageutils.VolumeOption) ([]v1.ObjectGraphNode, error) {
	var nodes []v1.ObjectGraphNode
	volumes, err := storageutils.GetVolumes(obj, og.virtClient, volumeOpts...)
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
		case volume.ServiceAccount != nil:
			node := og.newGraphNode(volume.ServiceAccount.ServiceAccountName, namespace, "serviceaccounts", nil, false)
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
	pods, err := og.k8sClient.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{
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
		matcher := vm.Spec.Instancetype
		return og.getMatcherResource(matcher, vm.Status.InstancetypeRef, vm.GetNamespace(), clusterInstancetypeResource)
	}
	return nil
}

func (og *ObjectGraph) getPreferenceNode(vm *v1.VirtualMachine) []v1.ObjectGraphNode {
	if vm.Spec.Preference != nil {
		matcher := vm.Spec.Preference
		return og.getMatcherResource(matcher, vm.Status.PreferenceRef, vm.GetNamespace(), clusterPreferenceResource)
	}
	return nil
}

func (og *ObjectGraph) getMatcherResource(
	matcher v1.Matcher, statusRef *v1.InstancetypeStatusRef, namespace, defaultResource string,
) []v1.ObjectGraphNode {
	var nodes []v1.ObjectGraphNode
	resource := defaultResource
	if matcher.GetKind() != "" {
		resource = strings.ToLower(matcher.GetKind()) + "s"
	}
	switch resource {
	case clusterInstancetypeResource, clusterPreferenceResource:
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

func (og *ObjectGraph) handleNetworkNodes(vmiSpec v1.VirtualMachineInstanceSpec, vmName, namespace string) []v1.ObjectGraphNode {
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
		}
	}

	return nodes
}

func decodeTemplate(tpl *templatev1beta1.VirtualMachineTemplate) (*v1.VirtualMachine, error) {
	if tpl.Spec.VirtualMachine == nil || len(tpl.Spec.VirtualMachine.Raw) == 0 {
		return nil, fmt.Errorf("virtual machine template %s/%s has no embedded VirtualMachine", tpl.GetNamespace(), tpl.GetName())
	}

	raw := map[string]interface{}{}
	if err := json.Unmarshal(tpl.Spec.VirtualMachine.Raw, &raw); err != nil {
		return nil, fmt.Errorf("failed to decode embedded VirtualMachine of template %s/%s: %w", tpl.GetNamespace(), tpl.GetName(), err)
	}

	template, found, err := unstructured.NestedFieldNoCopy(raw, "spec", "template")
	if err != nil {
		return nil, fmt.Errorf("failed to decode embedded VirtualMachine of template %s/%s: %w", tpl.GetNamespace(), tpl.GetName(), err)
	}
	if !found || template == nil {
		return nil, fmt.Errorf("embedded VirtualMachine of template %s/%s is missing spec.template", tpl.GetNamespace(), tpl.GetName())
	}

	templateSpec, _, err := unstructured.NestedMap(raw, "spec", "template", "spec")
	if err != nil {
		return nil, fmt.Errorf("failed to decode embedded VirtualMachine of template %s/%s: %w", tpl.GetNamespace(), tpl.GetName(), err)
	}

	vmiSpec := v1.VirtualMachineInstanceSpec{
		Volumes:           decodeNestedSliceLenient(templateSpec, "volumes", volumeFromRaw),
		Networks:          decodeNestedSliceLenient(templateSpec, "networks", networkFromRaw),
		AccessCredentials: decodeNestedSliceLenient(templateSpec, "accessCredentials", accessCredentialFromRaw),
	}

	var instancetype *v1.InstancetypeMatcher
	if err := decodeNestedField(raw, &instancetype, "spec", "instancetype"); err != nil {
		return nil, fmt.Errorf("failed to decode instance type of template %s/%s: %w", tpl.GetNamespace(), tpl.GetName(), err)
	}
	var preference *v1.PreferenceMatcher
	if err := decodeNestedField(raw, &preference, "spec", "preference"); err != nil {
		return nil, fmt.Errorf("failed to decode preference of template %s/%s: %w", tpl.GetNamespace(), tpl.GetName(), err)
	}

	name, _, _ := unstructured.NestedString(raw, "metadata", "name")
	namespace, _, _ := unstructured.NestedString(raw, "metadata", "namespace")

	vm := &v1.VirtualMachine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1.VirtualMachineSpec{
			Instancetype:        instancetype,
			Preference:          preference,
			DataVolumeTemplates: decodeDataVolumeTemplates(raw),
			Template: &v1.VirtualMachineInstanceTemplateSpec{
				Spec: vmiSpec,
			},
		},
	}

	if !containsUnresolvedTemplateParam(vm.Namespace) {
		vm.Namespace = tpl.GetNamespace()
	}
	return vm, nil
}

func decodeNestedField(obj map[string]interface{}, out interface{}, fields ...string) error {
	value, found, err := unstructured.NestedFieldNoCopy(obj, fields...)
	if err != nil || !found {
		return err
	}

	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, out)
}

func decodeNestedSliceLenient[T any](obj map[string]interface{}, field string, fromRaw func(map[string]interface{}) *T) []T {
	items, found, err := unstructured.NestedSlice(obj, field)
	if err != nil || !found {
		return nil
	}

	decoded := make([]T, 0, len(items))
	for _, item := range items {
		data, err := json.Marshal(item)
		if err != nil {
			continue
		}
		var out T
		if err := json.Unmarshal(data, &out); err != nil {
			if itemMap, ok := item.(map[string]interface{}); ok {
				if fallback := fromRaw(itemMap); fallback != nil {
					decoded = append(decoded, *fallback)
				}
			}
			continue
		}
		decoded = append(decoded, out)
	}
	return decoded
}

func volumeFromRaw(volume map[string]interface{}) *v1.Volume {
	if name, _, _ := unstructured.NestedString(volume, "persistentVolumeClaim", "claimName"); name != "" {
		return &v1.Volume{
			VolumeSource: v1.VolumeSource{
				PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
					PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{ClaimName: name},
				},
			},
		}
	}
	if name, _, _ := unstructured.NestedString(volume, "dataVolume", "name"); name != "" {
		return &v1.Volume{
			VolumeSource: v1.VolumeSource{
				DataVolume: &v1.DataVolumeSource{Name: name},
			},
		}
	}
	if name, _, _ := unstructured.NestedString(volume, "configMap", "name"); name != "" {
		return &v1.Volume{
			VolumeSource: v1.VolumeSource{
				ConfigMap: &v1.ConfigMapVolumeSource{
					LocalObjectReference: k8sv1.LocalObjectReference{Name: name},
				},
			},
		}
	}
	if name, _, _ := unstructured.NestedString(volume, "secret", "secretName"); name != "" {
		return &v1.Volume{
			VolumeSource: v1.VolumeSource{
				Secret: &v1.SecretVolumeSource{SecretName: name},
			},
		}
	}
	if name, _, _ := unstructured.NestedString(volume, "serviceAccount", "serviceAccountName"); name != "" {
		return &v1.Volume{
			VolumeSource: v1.VolumeSource{
				ServiceAccount: &v1.ServiceAccountVolumeSource{ServiceAccountName: name},
			},
		}
	}
	if userDataRef, networkDataRef := cloudInitSecretRefs(volume, "cloudInitNoCloud"); userDataRef != nil || networkDataRef != nil {
		return &v1.Volume{
			VolumeSource: v1.VolumeSource{
				CloudInitNoCloud: &v1.CloudInitNoCloudSource{
					UserDataSecretRef:    userDataRef,
					NetworkDataSecretRef: networkDataRef,
				},
			},
		}
	}
	if userDataRef, networkDataRef := cloudInitSecretRefs(volume, "cloudInitConfigDrive"); userDataRef != nil || networkDataRef != nil {
		return &v1.Volume{
			VolumeSource: v1.VolumeSource{
				CloudInitConfigDrive: &v1.CloudInitConfigDriveSource{
					UserDataSecretRef:    userDataRef,
					NetworkDataSecretRef: networkDataRef,
				},
			},
		}
	}
	return nil
}

func cloudInitSecretRefs(volume map[string]interface{}, source string) (userData, networkData *k8sv1.LocalObjectReference) {
	if name, _, _ := unstructured.NestedString(volume, source, "secretRef", "name"); name != "" {
		userData = &k8sv1.LocalObjectReference{Name: name}
	}
	if name, _, _ := unstructured.NestedString(volume, source, "networkDataSecretRef", "name"); name != "" {
		networkData = &k8sv1.LocalObjectReference{Name: name}
	}
	return userData, networkData
}

func networkFromRaw(network map[string]interface{}) *v1.Network {
	if name, _, _ := unstructured.NestedString(network, "multus", "networkName"); name != "" {
		return &v1.Network{
			NetworkSource: v1.NetworkSource{
				Multus: &v1.MultusNetwork{NetworkName: name},
			},
		}
	}
	return nil
}

func accessCredentialFromRaw(accessCredential map[string]interface{}) *v1.AccessCredential {
	if name, _, _ := unstructured.NestedString(accessCredential, "sshPublicKey", "source", "secret", "secretName"); name != "" {
		return &v1.AccessCredential{
			SSHPublicKey: &v1.SSHPublicKeyAccessCredential{
				Source: v1.SSHPublicKeyAccessCredentialSource{
					Secret: &v1.AccessCredentialSecretSource{SecretName: name},
				},
			},
		}
	}
	if name, _, _ := unstructured.NestedString(accessCredential, "userPassword", "source", "secret", "secretName"); name != "" {
		return &v1.AccessCredential{
			UserPassword: &v1.UserPasswordAccessCredential{
				Source: v1.UserPasswordAccessCredentialSource{
					Secret: &v1.AccessCredentialSecretSource{SecretName: name},
				},
			},
		}
	}
	return nil
}

func decodeDataVolumeTemplates(raw map[string]interface{}) []v1.DataVolumeTemplateSpec {
	items, found, err := unstructured.NestedSlice(raw, "spec", "dataVolumeTemplates")
	if err != nil || !found {
		return nil
	}

	var dvTemplates []v1.DataVolumeTemplateSpec
	for _, item := range items {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		name, _, _ := unstructured.NestedString(itemMap, "metadata", "name")
		if name == "" {
			continue
		}
		dvTemplates = append(dvTemplates, v1.DataVolumeTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{Name: name},
			Spec: cdiv1.DataVolumeSpec{
				Source:    decodeDataVolumeSource(itemMap),
				SourceRef: decodeDataVolumeSourceRef(itemMap),
			},
		})
	}
	return dvTemplates
}

func decodeDataVolumeSource(dvt map[string]interface{}) *cdiv1.DataVolumeSource {
	if name, _, _ := unstructured.NestedString(dvt, "spec", "source", "pvc", "name"); name != "" {
		namespace, _, _ := unstructured.NestedString(dvt, "spec", "source", "pvc", "namespace")
		return &cdiv1.DataVolumeSource{
			PVC: &cdiv1.DataVolumeSourcePVC{Name: name, Namespace: namespace},
		}
	}

	if name, _, _ := unstructured.NestedString(dvt, "spec", "source", "snapshot", "name"); name != "" {
		namespace, _, _ := unstructured.NestedString(dvt, "spec", "source", "snapshot", "namespace")
		return &cdiv1.DataVolumeSource{
			Snapshot: &cdiv1.DataVolumeSourceSnapshot{Name: name, Namespace: namespace},
		}
	}

	return nil
}

func decodeDataVolumeSourceRef(dvt map[string]interface{}) *cdiv1.DataVolumeSourceRef {
	name, _, _ := unstructured.NestedString(dvt, "spec", "sourceRef", "name")
	if name == "" {
		return nil
	}
	kind, _, _ := unstructured.NestedString(dvt, "spec", "sourceRef", "kind")
	sourceRef := &cdiv1.DataVolumeSourceRef{Kind: kind, Name: name}
	if namespace, found, _ := unstructured.NestedString(dvt, "spec", "sourceRef", "namespace"); found {
		sourceRef.Namespace = pointer.P(namespace)
	}
	return sourceRef
}

func containsUnresolvedTemplateParam(s string) bool {
	return strings.Contains(s, "${")
}

func resolveParameterDefaults(s string, params []templatev1beta1.Parameter) string {
	for _, p := range params {
		if p.Value == "" {
			continue
		}
		s = strings.ReplaceAll(s, fmt.Sprintf("${%s}", p.Name), p.Value)
		s = strings.ReplaceAll(s, fmt.Sprintf("${{%s}}", p.Name), p.Value)
	}
	return s
}

func resolveAndMarkTemplateParameters(nodes []v1.ObjectGraphNode, params []templatev1beta1.Parameter) []v1.ObjectGraphNode {
	for i := range nodes {
		nodes[i].ObjectReference.Name = resolveParameterDefaults(nodes[i].ObjectReference.Name, params)
		if nodes[i].ObjectReference.Namespace != nil {
			nodes[i].ObjectReference.Namespace = pointer.P(resolveParameterDefaults(*nodes[i].ObjectReference.Namespace, params))
		}
		if nodeHasUnresolvedTemplateParam(nodes[i]) {
			nodes[i].Optional = pointer.P(true)
		}
		nodes[i].Children = resolveAndMarkTemplateParameters(nodes[i].Children, params)
	}
	return nodes
}

func nodeHasUnresolvedTemplateParam(node v1.ObjectGraphNode) bool {
	if containsUnresolvedTemplateParam(node.ObjectReference.Name) {
		return true
	}
	return node.ObjectReference.Namespace != nil && containsUnresolvedTemplateParam(*node.ObjectReference.Namespace)
}

func markDataVolumeTemplateNodesOptional(nodes []v1.ObjectGraphNode, dvTemplates []v1.DataVolumeTemplateSpec) {
	for i := range nodes {
		if nodes[i].ObjectReference.Kind != "DataVolume" {
			continue
		}
		if !hasDataVolumeTemplate(dvTemplates, nodes[i].ObjectReference.Name) {
			continue
		}
		markNodeTreeOptional(&nodes[i])
	}
}

func markNodeTreeOptional(node *v1.ObjectGraphNode) {
	node.Optional = pointer.P(true)
	for i := range node.Children {
		markNodeTreeOptional(&node.Children[i])
	}
}

func hasDataVolumeTemplate(dvTemplates []v1.DataVolumeTemplateSpec, name string) bool {
	for i := range dvTemplates {
		if dvTemplates[i].Name == name {
			return true
		}
	}
	return false
}

func (og *ObjectGraph) dataVolumeTemplateSourceNodes(dvTemplates []v1.DataVolumeTemplateSpec, namespace string) []v1.ObjectGraphNode {
	var nodes []v1.ObjectGraphNode

	for i := range dvTemplates {
		spec := dvTemplates[i].Spec

		if spec.Source != nil && spec.Source.PVC != nil {
			pvcNamespace := spec.Source.PVC.Namespace
			if pvcNamespace == "" {
				pvcNamespace = namespace
			}
			node := og.newGraphNode(spec.Source.PVC.Name, pvcNamespace, "persistentvolumeclaims", nil, false)
			if node != nil {
				nodes = append(nodes, *node)
			}
		}

		if spec.Source != nil && spec.Source.Snapshot != nil {
			snapshotNamespace := spec.Source.Snapshot.Namespace
			if snapshotNamespace == "" {
				snapshotNamespace = namespace
			}
			node := og.newGraphNode(spec.Source.Snapshot.Name, snapshotNamespace, "volumesnapshots", nil, false)
			if node != nil {
				nodes = append(nodes, *node)
			}
		}

		if spec.SourceRef != nil && spec.SourceRef.Kind == "DataSource" {
			dataSourceNamespace := namespace
			if spec.SourceRef.Namespace != nil {
				dataSourceNamespace = *spec.SourceRef.Namespace
			}
			node := og.newGraphNode(spec.SourceRef.Name, dataSourceNamespace, "datasources", nil, false)
			if node != nil {
				nodes = append(nodes, *node)
			}
		}
	}

	return nodes
}

// TODO: Add more as needed.
// For example vmexports, vmsnapshots, vmimigrations, etc.
