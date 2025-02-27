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
 * Copyright The KubeVirt Authors
 *
 */

package rest

import (
	"context"
	"fmt"
	"net/http"

	"github.com/emicklei/go-restful/v3"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
	kutil "kubevirt.io/kubevirt/pkg/util"
)

const (
	pvcVolumeModeErr          = "pvc should be filesystem pvc"
	pvcAccessModeErr          = "pvc access mode can't be read only"
	pvcSizeErrFmt             = "pvc size [%s] should be bigger then [%s]"
	memoryDumpNameConflictErr = "can't request memory dump for pvc [%s] while pvc [%s] is still associated as the memory dump pvc"
)

func (app *SubresourceAPIApp) fetchPersistentVolumeClaim(name string, namespace string) (*k8sv1.PersistentVolumeClaim, *errors.StatusError) {
	pvc, err := app.virtCli.CoreV1().PersistentVolumeClaims(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, errors.NewNotFound(v1.Resource("persistentvolumeclaim"), name)
		}
		return nil, errors.NewInternalError(fmt.Errorf("unable to retrieve pvc [%s]: %v", name, err))
	}
	return pvc, nil
}

func (app *SubresourceAPIApp) fetchCDIConfig() (*cdiv1.CDIConfig, *errors.StatusError) {
	cdiConfig, err := app.virtCli.CdiClient().CdiV1beta1().CDIConfigs().Get(context.Background(), storagetypes.ConfigName, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, errors.NewInternalError(fmt.Errorf("unable to retrieve cdi config: %v", err))
	}
	return cdiConfig, nil
}

func (app *SubresourceAPIApp) validateMemoryDumpClaim(vmi *v1.VirtualMachineInstance, claimName, namespace string) *errors.StatusError {
	pvc, err := app.fetchPersistentVolumeClaim(claimName, namespace)
	if err != nil {
		return err
	}
	if storagetypes.IsPVCBlock(pvc.Spec.VolumeMode) {
		return errors.NewConflict(v1.Resource("persistentvolumeclaim"), claimName, fmt.Errorf(pvcVolumeModeErr))
	}

	if storagetypes.IsReadOnlyAccessMode(pvc.Spec.AccessModes) {
		return errors.NewConflict(v1.Resource("persistentvolumeclaim"), claimName, fmt.Errorf(pvcAccessModeErr))
	}

	pvcSize := pvc.Spec.Resources.Requests.Storage()
	scaledPvcSize := resource.NewScaledQuantity(pvcSize.ScaledValue(resource.Kilo), resource.Kilo)

	expectedMemoryDumpSize := kutil.CalcExpectedMemoryDumpSize(vmi)
	cdiConfig, err := app.fetchCDIConfig()
	if err != nil {
		return err
	}
	var expectedPvcSize *resource.Quantity
	var overheadErr error
	if cdiConfig == nil {
		log.Log.Object(vmi).V(3).Infof(storagetypes.FSOverheadMsg)
		expectedPvcSize, overheadErr = storagetypes.GetSizeIncludingDefaultFSOverhead(expectedMemoryDumpSize)
	} else {
		expectedPvcSize, overheadErr = storagetypes.GetSizeIncludingFSOverhead(expectedMemoryDumpSize, pvc.Spec.StorageClassName, pvc.Spec.VolumeMode, cdiConfig)
	}
	if overheadErr != nil {
		return errors.NewInternalError(overheadErr)
	}
	if scaledPvcSize.Cmp(*expectedPvcSize) < 0 {
		return errors.NewConflict(v1.Resource("persistentvolumeclaim"), claimName, fmt.Errorf(pvcSizeErrFmt, scaledPvcSize.String(), expectedPvcSize.String()))
	}

	return nil
}

func (app *SubresourceAPIApp) validateMemoryDumpRequest(vm *v1.VirtualMachine, memoryDumpReq *v1.VirtualMachineMemoryDumpRequest) *errors.StatusError {
	if memoryDumpReq.ClaimName == "" && vm.Status.MemoryDumpRequest == nil {
		return errors.NewBadRequest("Memory dump requires claim name to be set")
	} else if vm.Status.MemoryDumpRequest != nil && memoryDumpReq.ClaimName != "" {
		if vm.Status.MemoryDumpRequest.ClaimName != memoryDumpReq.ClaimName {
			return errors.NewConflict(v1.Resource("virtualmachine"), vm.Name, fmt.Errorf(memoryDumpNameConflictErr, memoryDumpReq.ClaimName, vm.Status.MemoryDumpRequest.ClaimName))
		}
	} else if vm.Status.MemoryDumpRequest != nil {
		memoryDumpReq.ClaimName = vm.Status.MemoryDumpRequest.ClaimName
	}

	vmi, statErr := app.FetchVirtualMachineInstance(vm.Namespace, vm.Name)
	if statErr != nil {
		return statErr
	}

	if !vmi.IsRunning() {
		return errors.NewConflict(v1.Resource("virtualmachineinstance"), vm.Name, fmt.Errorf(vmiNotRunning))
	}

	if statErr = app.validateMemoryDumpClaim(vmi, memoryDumpReq.ClaimName, vm.Namespace); statErr != nil {
		return statErr
	}

	return nil
}

func (app *SubresourceAPIApp) vmMemoryDumpRequestPatchStatus(name, namespace string, memoryDumpReq *v1.VirtualMachineMemoryDumpRequest, removeRequest bool) *errors.StatusError {
	vm, statErr := app.fetchVirtualMachine(name, namespace)
	if statErr != nil {
		return statErr
	}

	if !removeRequest {
		statErr = app.validateMemoryDumpRequest(vm, memoryDumpReq)
		if statErr != nil {
			return statErr
		}
	}

	patchBytes, err := generateVMMemoryDumpRequestPatch(vm, memoryDumpReq, removeRequest)
	if err != nil {
		return errors.NewConflict(v1.Resource("virtualmachine"), name, err)
	}

	log.Log.Object(vm).V(4).Infof(patchingVMFmt, string(patchBytes))
	if _, err = app.virtCli.VirtualMachine(vm.Namespace).PatchStatus(context.Background(), vm.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{}); err != nil {
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

func (app *SubresourceAPIApp) MemoryDumpVMRequestHandler(request *restful.Request, response *restful.Response) {
	name := request.PathParameter("name")
	namespace := request.PathParameter("namespace")

	if !app.clusterConfig.HotplugVolumesEnabled() {
		writeError(errors.NewBadRequest("Unable to memory dump because HotplugVolumes feature gate is not enabled."), response)
		return
	}

	if request.Request.Body == nil {
		writeError(errors.NewBadRequest("Request with no body"), response)
		return
	}
	memoryDumpReq := &v1.VirtualMachineMemoryDumpRequest{}
	defer request.Request.Body.Close()
	if err := decodeBody(request, memoryDumpReq); err != nil {
		writeError(err, response)
		return
	}

	memoryDumpReq.Phase = v1.MemoryDumpAssociating
	isRemoveRequest := false
	if err := app.vmMemoryDumpRequestPatchStatus(name, namespace, memoryDumpReq, isRemoveRequest); err != nil {
		writeError(err, response)
		return
	}

	response.WriteHeader(http.StatusAccepted)
}

func (app *SubresourceAPIApp) RemoveMemoryDumpVMRequestHandler(request *restful.Request, response *restful.Response) {
	name := request.PathParameter("name")
	namespace := request.PathParameter("namespace")

	removeReq := &v1.VirtualMachineMemoryDumpRequest{
		Phase:  v1.MemoryDumpDissociating,
		Remove: true,
	}
	isRemoveRequest := true
	if err := app.vmMemoryDumpRequestPatchStatus(name, namespace, removeReq, isRemoveRequest); err != nil {
		writeError(err, response)
		return
	}

	response.WriteHeader(http.StatusAccepted)
}

func addMemoryDumpRequest(vm, vmCopy *v1.VirtualMachine, memoryDumpReq *v1.VirtualMachineMemoryDumpRequest) error {
	claimName := memoryDumpReq.ClaimName
	if vm.Status.MemoryDumpRequest != nil {
		if vm.Status.MemoryDumpRequest.Phase == v1.MemoryDumpDissociating {
			return fmt.Errorf("can't dump memory for pvc [%s] a remove memory dump request is in progress", claimName)
		}
		if vm.Status.MemoryDumpRequest.Phase != v1.MemoryDumpCompleted && vm.Status.MemoryDumpRequest.Phase != v1.MemoryDumpFailed {
			return fmt.Errorf("memory dump request for pvc [%s] already in progress", claimName)
		}
	}
	vmCopy.Status.MemoryDumpRequest = memoryDumpReq
	return nil
}

func removeMemoryDumpRequest(vm, vmCopy *v1.VirtualMachine, memoryDumpReq *v1.VirtualMachineMemoryDumpRequest) error {
	if vm.Status.MemoryDumpRequest == nil {
		return fmt.Errorf("can't remove memory dump association for vm %s, no association found", vm.Name)
	}

	claimName := vm.Status.MemoryDumpRequest.ClaimName
	if vm.Status.MemoryDumpRequest.Remove {
		return fmt.Errorf("memory dump remove request for pvc [%s] already exists", claimName)
	}
	memoryDumpReq.ClaimName = claimName
	vmCopy.Status.MemoryDumpRequest = memoryDumpReq
	return nil
}

func generateVMMemoryDumpRequestPatch(vm *v1.VirtualMachine, memoryDumpReq *v1.VirtualMachineMemoryDumpRequest, removeRequest bool) ([]byte, error) {
	vmCopy := vm.DeepCopy()

	if !removeRequest {
		if err := addMemoryDumpRequest(vm, vmCopy, memoryDumpReq); err != nil {
			return nil, err
		}
	} else {
		if err := removeMemoryDumpRequest(vm, vmCopy, memoryDumpReq); err != nil {
			return nil, err
		}
	}

	patchSet := patch.New(patch.WithTest("/status/memoryDumpRequest", vm.Status.MemoryDumpRequest))
	if vm.Status.MemoryDumpRequest != nil {
		patchSet.AddOption(patch.WithReplace("/status/memoryDumpRequest", vmCopy.Status.MemoryDumpRequest))
	} else {
		patchSet.AddOption(patch.WithAdd("/status/memoryDumpRequest", vmCopy.Status.MemoryDumpRequest))
	}

	return patchSet.GeneratePayload()
}
