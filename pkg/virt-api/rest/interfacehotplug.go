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
	"encoding/json"
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

	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

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
