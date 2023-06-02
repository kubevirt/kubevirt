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
 * Copyright 2023 Red Hat, Inc.
 *
 */

package rest

import (
	"context"
	"encoding/json"
	goerrors "errors"
	"fmt"
	"io"
	"net/http"

	"github.com/emicklei/go-restful/v3"

	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/network/vmispec"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

func generateVMIInterfaceRequestPatch(vmi *v1.VirtualMachineInstance, interfaceRequest *v1.VirtualMachineInterfaceRequest) (string, error) {
	vmiCopy := vmi.DeepCopy()
	switch {
	case interfaceRequest.AddInterfaceOptions != nil:
		vmiCopy.Spec = *ApplyAddInterfaceRequestOnVMISpec(&vmiCopy.Spec, interfaceRequest)
	case interfaceRequest.RemoveInterfaceOptions != nil:
		vmiCopy.Spec = *ApplyRemoveInterfaceRequestOnVMISpec(&vmiCopy.Spec, interfaceRequest)
	}

	if equality.Semantic.DeepEqual(vmiCopy.Spec, vmi.Spec) {
		return "", nil
	}

	oldIfacesJSON, err := json.Marshal(vmi.Spec.Domain.Devices.Interfaces)
	if err != nil {
		return "", err
	}

	newIfacesJSON, err := json.Marshal(vmiCopy.Spec.Domain.Devices.Interfaces)
	if err != nil {
		return "", err
	}

	oldNetworksJSON, err := json.Marshal(vmi.Spec.Networks)
	if err != nil {
		return "", err
	}

	newNetworksJSON, err := json.Marshal(vmiCopy.Spec.Networks)
	if err != nil {
		return "", err
	}

	const verb = "add"
	testNetworks := fmt.Sprintf(`{ "op": "test", "path": "/spec/networks", "value": %s}`, string(oldNetworksJSON))
	updateNetworks := fmt.Sprintf(`{ "op": %q, "path": "/spec/networks", "value": %s}`, verb, string(newNetworksJSON))

	testInterfaces := fmt.Sprintf(`{ "op": "test", "path": "/spec/domain/devices/interfaces", "value": %s}`, string(oldIfacesJSON))
	updateInterfaces := fmt.Sprintf(`{ "op": %q, "path": "/spec/domain/devices/interfaces", "value": %s}`, verb, string(newIfacesJSON))

	patch := fmt.Sprintf("[%s, %s, %s, %s]", testNetworks, testInterfaces, updateNetworks, updateInterfaces)
	return patch, nil
}

func generateVMInterfaceRequestPatch(vm *v1.VirtualMachine, interfaceRequest *v1.VirtualMachineInterfaceRequest) (string, error) {
	const verb = "add"

	vmCopy := vm.DeepCopy()
	switch {
	case interfaceRequest.AddInterfaceOptions != nil:
		addAddInterfaceRequests(vm, interfaceRequest, vmCopy)
	case interfaceRequest.RemoveInterfaceOptions != nil:
		addRemoveInterfaceRequests(vm, interfaceRequest, vmCopy)
	}

	if equality.Semantic.DeepEqual(vm.Status.InterfaceRequests, vmCopy.Status.InterfaceRequests) {
		return "", nil
	}

	oldJson, err := json.Marshal(vm.Status.InterfaceRequests)
	if err != nil {
		return "", err
	}
	newJson, err := json.Marshal(vmCopy.Status.InterfaceRequests)
	if err != nil {
		return "", err
	}

	test := fmt.Sprintf(`{ "op": "test", "path": "/status/interfaceRequests", "value": %s}`, string(oldJson))
	update := fmt.Sprintf(`{ "op": %q, "path": "/status/interfaceRequests", "value": %s}`, verb, string(newJson))
	patch := fmt.Sprintf("[%s, %s]", test, update)

	return patch, nil
}

func addAddInterfaceRequests(vm *v1.VirtualMachine, ifaceRequest *v1.VirtualMachineInterfaceRequest, vmCopy *v1.VirtualMachine) {
	canonicalIfaceName := ifaceRequest.AddInterfaceOptions.Name
	existingIfaceAddRequests := filterAddInterfaceRequests(vm, canonicalIfaceName)
	existingIfaceRemoveRequests := filterRemoveInterfaceRequests(vm, canonicalIfaceName)

	if len(existingIfaceAddRequests) == 0 {
		if len(existingIfaceRemoveRequests) > 0 {
			log.Log.Object(vm).Warningf("cannot handle interface %q add request because there is an existing remove request", canonicalIfaceName)
			return
		}
		vmCopy.Status.InterfaceRequests = append(vm.Status.InterfaceRequests, *ifaceRequest)
	}
}

func addRemoveInterfaceRequests(vm *v1.VirtualMachine, ifaceRequest *v1.VirtualMachineInterfaceRequest, vmCopy *v1.VirtualMachine) {
	canonicalIfaceName := ifaceRequest.RemoveInterfaceOptions.Name
	existingIfaceRemoveRequests := filterRemoveInterfaceRequests(vm, canonicalIfaceName)
	existingIfaceAddRequests := filterAddInterfaceRequests(vm, canonicalIfaceName)

	if len(existingIfaceRemoveRequests) == 0 {
		if len(existingIfaceAddRequests) > 0 {
			log.Log.Object(vm).Warningf("cannot handle interface %q remove request because there is an existing add request", canonicalIfaceName)
			return
		}
		vmCopy.Status.InterfaceRequests = append(vm.Status.InterfaceRequests, *ifaceRequest)
	}
}

func filterAddInterfaceRequests(vm *v1.VirtualMachine, canonicalIfaceName string) []v1.VirtualMachineInterfaceRequest {
	return filterInterfaceRequests(
		vm.Status.InterfaceRequests,
		func(ifaceReq v1.VirtualMachineInterfaceRequest) bool {
			return ifaceReq.AddInterfaceOptions != nil && ifaceReq.AddInterfaceOptions.Name == canonicalIfaceName
		},
	)
}

func filterRemoveInterfaceRequests(vm *v1.VirtualMachine, canonicalIfaceName string) []v1.VirtualMachineInterfaceRequest {
	return filterInterfaceRequests(
		vm.Status.InterfaceRequests,
		func(ifaceReq v1.VirtualMachineInterfaceRequest) bool {
			return ifaceReq.RemoveInterfaceOptions != nil && ifaceReq.RemoveInterfaceOptions.Name == canonicalIfaceName
		},
	)
}

func ApplyAddInterfaceRequestOnVMISpec(vmiSpec *v1.VirtualMachineInstanceSpec, request *v1.VirtualMachineInterfaceRequest) *v1.VirtualMachineInstanceSpec {
	iface2AddName := request.AddInterfaceOptions.Name
	iface2Add := vmispec.LookupInterfaceByName(vmiSpec.Domain.Devices.Interfaces, iface2AddName)
	if iface2Add == nil {
		newInterface := v1.Interface{
			InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}},
			Name:                   iface2AddName,
		}
		newNetwork := v1.Network{
			Name: iface2AddName,
			NetworkSource: v1.NetworkSource{
				Multus: &v1.MultusNetwork{
					NetworkName: request.AddInterfaceOptions.NetworkAttachmentDefinitionName,
				},
			},
		}

		vmiSpec.Domain.Devices.Interfaces = append(vmiSpec.Domain.Devices.Interfaces, newInterface)
		vmiSpec.Networks = append(vmiSpec.Networks, newNetwork)
	}
	return vmiSpec
}

func ApplyRemoveInterfaceRequestOnVMISpec(vmiSpec *v1.VirtualMachineInstanceSpec, request *v1.VirtualMachineInterfaceRequest) *v1.VirtualMachineInstanceSpec {
	iface2RemoveName := request.RemoveInterfaceOptions.Name
	iface2Remove := vmispec.LookupInterfaceByName(vmiSpec.Domain.Devices.Interfaces, iface2RemoveName)
	if iface2Remove != nil {
		iface2Remove.State = v1.InterfaceStateAbsent
	}
	return vmiSpec
}

// VMAddInterfaceRequestHandler handles the subresource for hot plugging a network interface.
func (app *SubresourceAPIApp) VMAddInterfaceRequestHandler(request *restful.Request, response *restful.Response) {
	if !app.clusterConfig.HotplugNetworkInterfacesEnabled() {
		writeError(featureGateNotEnableError(), response)
		return
	}

	name := request.PathParameter("name")
	namespace := request.PathParameter("namespace")
	interfaceRequest, err := app.newAddInterfaceRequest(request)
	if err != nil {
		writeError(errors.NewBadRequest(err.Error()), response)
		return
	}

	if err := app.vmInterfacePatchStatus(name, namespace, &interfaceRequest); err != nil {
		writeError(err, response)
		return
	}

	response.WriteHeader(http.StatusAccepted)
}

// VMRemoveInterfaceRequestHandler handles the subresource for hot unplugging a network interface.
func (app *SubresourceAPIApp) VMRemoveInterfaceRequestHandler(request *restful.Request, response *restful.Response) {
	if !app.clusterConfig.HotplugNetworkInterfacesEnabled() {
		writeError(featureGateNotEnableError(), response)
		return
	}

	name := request.PathParameter("name")
	namespace := request.PathParameter("namespace")
	interfaceRequest, err := app.newRemoveInterfaceRequest(request)
	if err != nil {
		writeError(errors.NewBadRequest(err.Error()), response)
		return
	}

	if err := app.vmInterfacePatchStatus(name, namespace, &interfaceRequest); err != nil {
		writeError(err, response)
		return
	}
	response.WriteHeader(http.StatusAccepted)
}

// VMIAddInterfaceRequestHandler handles the subresource for hot plugging a network interface.
func (app *SubresourceAPIApp) VMIAddInterfaceRequestHandler(request *restful.Request, response *restful.Response) {
	if !app.clusterConfig.HotplugNetworkInterfacesEnabled() {
		writeError(featureGateNotEnableError(), response)
		return
	}

	name := request.PathParameter("name")
	namespace := request.PathParameter("namespace")
	interfaceRequest, err := app.newAddInterfaceRequest(request)
	if err != nil {
		writeError(errors.NewBadRequest(err.Error()), response)
		return
	}

	if err := app.vmiInterfacePatch(name, namespace, &interfaceRequest); err != nil {
		writeError(err, response)
		return
	}
	response.WriteHeader(http.StatusAccepted)
}

// VMIRemoveInterfaceRequestHandler handles the subresource for hot unplugging a network interface.
func (app *SubresourceAPIApp) VMIRemoveInterfaceRequestHandler(request *restful.Request, response *restful.Response) {
	if !app.clusterConfig.HotplugNetworkInterfacesEnabled() {
		writeError(featureGateNotEnableError(), response)
		return
	}

	name := request.PathParameter("name")
	namespace := request.PathParameter("namespace")
	interfaceRequest, err := app.newRemoveInterfaceRequest(request)
	if err != nil {
		writeError(errors.NewBadRequest(err.Error()), response)
		return
	}

	if err := app.vmiInterfacePatch(name, namespace, &interfaceRequest); err != nil {
		writeError(err, response)
		return
	}
	response.WriteHeader(http.StatusAccepted)
}

func (app *SubresourceAPIApp) newAddInterfaceRequest(request *restful.Request) (v1.VirtualMachineInterfaceRequest, error) {
	var vmInterfaceRequest v1.VirtualMachineInterfaceRequest
	interfaceRequestOptions, err := decodeInterfaceRequest(request, &v1.AddInterfaceOptions{})
	if err != nil {
		return vmInterfaceRequest, err
	}

	if interfaceRequestOptions.NetworkAttachmentDefinitionName == "" {
		return vmInterfaceRequest, fmt.Errorf("AddInterfaceOptions requires `networkAttachmentDefinitionName` to be set")
	}
	if interfaceRequestOptions.Name == "" {
		return vmInterfaceRequest, fmt.Errorf("AddInterfaceOptions requires `name` to be set")
	}

	vmInterfaceRequest.AddInterfaceOptions = interfaceRequestOptions
	return vmInterfaceRequest, nil
}

func (app *SubresourceAPIApp) newRemoveInterfaceRequest(request *restful.Request) (v1.VirtualMachineInterfaceRequest, error) {
	var vmInterfaceRequest v1.VirtualMachineInterfaceRequest
	interfaceRequestOptions, err := decodeInterfaceRequest(request, &v1.RemoveInterfaceOptions{})
	if err != nil {
		return vmInterfaceRequest, err
	}

	if interfaceRequestOptions.Name == "" {
		return vmInterfaceRequest, fmt.Errorf("RemoveInterfaceOptions requires `name` to be set")
	}

	vmInterfaceRequest.RemoveInterfaceOptions = interfaceRequestOptions
	return vmInterfaceRequest, nil
}

func decodeInterfaceRequest[T any](request *restful.Request, opts *T) (*T, error) {
	if request.Request.Body != nil {
		defer func() { _ = request.Request.Body.Close() }()
		err := yaml.NewYAMLOrJSONDecoder(request.Request.Body, 1024).Decode(opts)
		switch err {
		case io.EOF, nil:
			break
		default:
			return nil, fmt.Errorf("cannot unmarshal Request body to struct, error: %v", err)
		}
	} else {
		return nil, fmt.Errorf("no request body, unable to decode options")
	}
	return opts, nil
}

func (app *SubresourceAPIApp) vmiInterfacePatch(vmName string, namespace string, interfaceRequest *v1.VirtualMachineInterfaceRequest) *errors.StatusError {
	vmi, statErr := app.FetchVirtualMachineInstance(namespace, vmName)
	if statErr != nil {
		return statErr
	}

	if !vmi.IsRunning() {
		return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmName, fmt.Errorf("VMI is not running"))
	}

	patch, err := generateVMIInterfaceRequestPatch(vmi, interfaceRequest)
	if err != nil {
		return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmName, err)
	} else if patch == "" {
		log.Log.Object(vmi).V(4).Infof("Empty patch; nothing to do for VMI %s/%s", namespace, vmName)
		return nil
	}

	log.Log.Object(vmi).V(4).Infof("Patching VMI: %s", patch)
	if _, err := app.virtCli.VirtualMachineInstance(vmi.Namespace).Patch(context.Background(), vmi.Name, types.JSONPatchType, []byte(patch), &k8smetav1.PatchOptions{}); err != nil {
		log.Log.Object(vmi).V(1).Errorf("unable to patch vmi: %v", err)
		var statusError *errors.StatusError
		if goerrors.As(err, &statusError) {
			return statusError
		}
		return errors.NewInternalError(fmt.Errorf("unable to patch vmi: %v", err))
	}
	return nil
}

func (app *SubresourceAPIApp) vmInterfacePatchStatus(vmName string, namespace string, interfaceRequest *v1.VirtualMachineInterfaceRequest) *errors.StatusError {
	vm, statErr := app.fetchVirtualMachine(vmName, namespace)
	if statErr != nil {
		return statErr
	}

	patch, err := generateVMInterfaceRequestPatch(vm, interfaceRequest)
	if err != nil {
		return errors.NewConflict(v1.Resource("virtualmachine"), vmName, err)
	}

	if patch == "" {
		return nil
	}

	log.Log.Object(vm).V(4).Infof("Patching VM: %s", patch)
	if err := app.statusUpdater.PatchStatus(vm, types.JSONPatchType, []byte(patch), &k8smetav1.PatchOptions{}); err != nil {
		log.Log.Object(vm).V(1).Errorf("unable to patch vm status: %v", err)
		if errors.IsInvalid(err) {
			if statErr, ok := err.(*errors.StatusError); ok {
				return statErr
			}
		}
		return errors.NewInternalError(fmt.Errorf("unable to patch vm status: %v", err))
	}
	return nil
}

func filterInterfaceRequests(ifaceRequests []v1.VirtualMachineInterfaceRequest, p func(iface v1.VirtualMachineInterfaceRequest) bool) []v1.VirtualMachineInterfaceRequest {
	var filteredIfaceRequests []v1.VirtualMachineInterfaceRequest
	for _, iface := range ifaceRequests {
		if p(iface) {
			filteredIfaceRequests = append(filteredIfaceRequests, iface)
		}
	}
	return filteredIfaceRequests
}

func featureGateNotEnableError() *errors.StatusError {
	return errors.NewBadRequest(
		fmt.Sprintf(
			"Unable to Add Interface because the %q feature gate is not enabled.",
			virtconfig.HotplugNetworkIfacesGate,
		),
	)
}
