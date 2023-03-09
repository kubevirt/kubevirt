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

	"github.com/emicklei/go-restful"

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
	vmiCopy.Spec = *ApplyInterfaceRequestOnVMISpec(&vmiCopy.Spec, interfaceRequest)

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
	if interfaceRequest.AddInterfaceOptions != nil {
		addAddInterfaceRequests(vm, interfaceRequest, vmCopy)
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
	canonicalIfaceName := dynamicIfaceName(ifaceRequest)
	if ifaceRequest.AddInterfaceOptions != nil {
		existingIface := filterInterfaceRequests(
			vm.Status.InterfaceRequests,
			func(ifaceReq v1.VirtualMachineInterfaceRequest) bool {
				return dynamicIfaceName(&ifaceReq) == canonicalIfaceName
			},
		)

		if len(existingIface) == 0 {
			vmCopy.Status.InterfaceRequests = append(vm.Status.InterfaceRequests, *ifaceRequest)
		}
	}
}

func dynamicIfaceName(plugRequest *v1.VirtualMachineInterfaceRequest) string {
	if plugRequest.AddInterfaceOptions != nil {
		return plugRequest.AddInterfaceOptions.InterfaceName
	}
	panic(fmt.Errorf("must provide the `AddInterfaceOptions.InterfaceName` attribute"))
}

func ApplyInterfaceRequestOnVMISpec(vmiSpec *v1.VirtualMachineInstanceSpec, request *v1.VirtualMachineInterfaceRequest) *v1.VirtualMachineInstanceSpec {
	canonicalIfaceName := dynamicIfaceName(request)
	existingIface := vmispec.FilterInterfacesSpec(vmiSpec.Domain.Devices.Interfaces, func(iface v1.Interface) bool {
		return iface.Name == canonicalIfaceName
	})

	if len(existingIface) == 0 {
		newInterface := v1.Interface{
			InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}},
			Name:                   canonicalIfaceName,
		}
		newNetwork := v1.Network{
			Name: canonicalIfaceName,
			NetworkSource: v1.NetworkSource{
				Multus: &v1.MultusNetwork{
					NetworkName: request.AddInterfaceOptions.NetworkName,
				},
			},
		}

		vmiSpec.Domain.Devices.Interfaces = append(vmiSpec.Domain.Devices.Interfaces, newInterface)
		vmiSpec.Networks = append(vmiSpec.Networks, newNetwork)
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
	interfaceRequest, err := app.newInterfaceRequest(request)
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
	interfaceRequest, err := app.newInterfaceRequest(request)
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

func (app *SubresourceAPIApp) newInterfaceRequest(request *restful.Request) (v1.VirtualMachineInterfaceRequest, error) {
	opts := &v1.AddInterfaceOptions{}
	if request.Request.Body != nil {
		defer func() { _ = request.Request.Body.Close() }()
		err := yaml.NewYAMLOrJSONDecoder(request.Request.Body, 1024).Decode(opts)
		switch err {
		case io.EOF, nil:
			break
		default:
			return v1.VirtualMachineInterfaceRequest{}, fmt.Errorf("cannot unmarshal Request body to struct, error: %v", err)
		}
	} else {
		return v1.VirtualMachineInterfaceRequest{}, fmt.Errorf("`networkName` and `interfaceName` are expected")
	}

	if opts.NetworkName == "" {
		return v1.VirtualMachineInterfaceRequest{}, fmt.Errorf("AddInterfaceOptions requires `networkName` to be set")
	}
	if opts.InterfaceName == "" {
		return v1.VirtualMachineInterfaceRequest{}, fmt.Errorf("AddInterfaceOptions requires `interfaceName` to be set")
	}

	return v1.VirtualMachineInterfaceRequest{AddInterfaceOptions: opts}, nil
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
		log.Log.Object(vmi).Errorf("unable to patch vmi: %v", err)
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
		log.Log.Object(vm).Errorf("unable to patch vm status: %v", err)
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
