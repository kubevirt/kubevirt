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
	"net/http"

	"github.com/emicklei/go-restful/v3"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/controller"
	metrics "kubevirt.io/kubevirt/pkg/monitoring/metrics/virt-api"
)

const (
	hotplugVolumeNotEnabledError = "Enable DeclarativeHotplugVolumes or HotplugVolumes feature gate to use this API."
)

// VMAddVolumeRequestHandler handles the subresource for hot plugging a volume and disk.
func (app *SubresourceAPIApp) VMAddVolumeRequestHandler(request *restful.Request, response *restful.Response) {
	app.addVolumeRequestHandler(request, response, false)
}

// VMRemoveVolumeRequestHandler handles the subresource for hot plugging a volume and disk.
func (app *SubresourceAPIApp) VMRemoveVolumeRequestHandler(request *restful.Request, response *restful.Response) {
	app.removeVolumeRequestHandler(request, response, false)
}

// VMIAddVolumeRequestHandler handles the subresource for hot plugging a volume and disk.
func (app *SubresourceAPIApp) VMIAddVolumeRequestHandler(request *restful.Request, response *restful.Response) {
	app.addVolumeRequestHandler(request, response, true)
}

// VMIRemoveVolumeRequestHandler handles the subresource for hot plugging a volume and disk.
func (app *SubresourceAPIApp) VMIRemoveVolumeRequestHandler(request *restful.Request, response *restful.Response) {
	app.removeVolumeRequestHandler(request, response, true)
}

func (app *SubresourceAPIApp) hotplugVolumesEnabled() bool {
	return app.clusterConfig.HotplugVolumesEnabled() || app.clusterConfig.DeclarativeHotplugVolumesEnabled()
}

func (app *SubresourceAPIApp) ephemeralHotplugSupported() bool {
	return app.clusterConfig.HotplugVolumesEnabled()
}

func (app *SubresourceAPIApp) addVolumeRequestHandler(request *restful.Request, response *restful.Response, ephemeral bool) {
	name := request.PathParameter("name")
	namespace := request.PathParameter("namespace")

	if !app.hotplugVolumesEnabled() {
		writeError(errors.NewBadRequest(hotplugVolumeNotEnabledError), response)
		return
	}

	if request.Request.Body == nil {
		writeError(errors.NewBadRequest("Request with no body, a new name is expected as the request body"), response)
		return
	}

	opts := &v1.AddVolumeOptions{}
	defer request.Request.Body.Close()
	if err := decodeBody(request, opts); err != nil {
		writeError(err, response)
		return
	}

	if opts.Name == "" {
		writeError(errors.NewBadRequest("AddVolumeOptions requires name to be set"), response)
		return
	} else if opts.Disk == nil {
		writeError(errors.NewBadRequest("AddVolumeOptions requires disk to not be nil"), response)
		return
	} else if opts.VolumeSource == nil {
		writeError(errors.NewBadRequest("AddVolumeOptions requires VolumeSource to not be nil"), response)
		return
	}

	opts.Disk.Name = opts.Name
	volumeRequest := v1.VirtualMachineVolumeRequest{
		AddVolumeOptions: opts,
	}
	if opts.VolumeSource.DataVolume != nil {
		opts.VolumeSource.DataVolume.Hotpluggable = true
	} else if opts.VolumeSource.PersistentVolumeClaim != nil {
		opts.VolumeSource.PersistentVolumeClaim.Hotpluggable = true
	}

	// inject into VMI if ephemeral, else set as a request on the VM to both make permanent and hotplug.
	if ephemeral {
		if err := app.vmiVolumePatch(name, namespace, &volumeRequest); err != nil {
			writeError(err, response)
			return
		}
		metrics.NewEphemeralHotplugVolume()
	} else if app.clusterConfig.HotplugVolumesEnabled() {
		if err := app.vmVolumePatchStatus(name, namespace, &volumeRequest); err != nil {
			writeError(err, response)
			return
		}
	} else {
		if err := app.vmVolumePatch(name, namespace, &volumeRequest); err != nil {
			writeError(err, response)
			return
		}
	}

	response.WriteHeader(http.StatusAccepted)
}

func (app *SubresourceAPIApp) removeVolumeRequestHandler(request *restful.Request, response *restful.Response, ephemeral bool) {
	name := request.PathParameter("name")
	namespace := request.PathParameter("namespace")

	if !app.hotplugVolumesEnabled() {
		writeError(errors.NewBadRequest(hotplugVolumeNotEnabledError), response)
		return
	}

	if request.Request.Body == nil {
		writeError(errors.NewBadRequest("Request with no body, a new name is expected as the request body"),
			response)
		return
	}
	opts := &v1.RemoveVolumeOptions{}
	defer request.Request.Body.Close()
	if err := decodeBody(request, opts); err != nil {
		writeError(err, response)
		return
	}

	if opts.Name == "" {
		writeError(errors.NewBadRequest("RemoveVolumeOptions requires name to be set"), response)
		return
	}
	volumeRequest := v1.VirtualMachineVolumeRequest{
		RemoveVolumeOptions: opts,
	}

	// inject into VMI if ephemeral, else set as a request on the VM to both make permanent and hotplug.
	if ephemeral {
		if err := app.vmiVolumePatch(name, namespace, &volumeRequest); err != nil {
			writeError(err, response)
			return
		}
	} else if app.clusterConfig.HotplugVolumesEnabled() {
		if err := app.vmVolumePatchStatus(name, namespace, &volumeRequest); err != nil {
			writeError(err, response)
			return
		}
	} else {
		if err := app.vmVolumePatch(name, namespace, &volumeRequest); err != nil {
			writeError(err, response)
			return
		}
	}

	response.WriteHeader(http.StatusAccepted)
}

func (app *SubresourceAPIApp) vmVolumePatch(name, namespace string, volumeRequest *v1.VirtualMachineVolumeRequest) *errors.StatusError {
	vm, statErr := app.fetchVirtualMachine(name, namespace)
	if statErr != nil {
		return statErr
	}

	err := verifyVolumeOption(vm.Spec.Template.Spec.Volumes, volumeRequest)
	if err != nil {
		return errors.NewConflict(v1.Resource("virtualmachine"), name, err)
	}

	patchBytes, err := generateVolumeRequestPatchVM(&vm.Spec.Template.Spec, volumeRequest)
	if err != nil {
		return errors.NewConflict(v1.Resource("virtualmachine"), name, err)
	}

	dryRunOption := getDryRunOption(volumeRequest)
	log.Log.Object(vm).V(4).Infof("Patching VM: %s", string(patchBytes))
	if _, err := app.virtCli.VirtualMachine(vm.Namespace).Patch(context.Background(), vm.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{DryRun: dryRunOption}); err != nil {
		log.Log.Object(vm).Errorf("unable to patch vm: %v", err)
		if errors.IsInvalid(err) {
			if statErr, ok := err.(*errors.StatusError); ok {
				return statErr
			}
		}
		return errors.NewInternalError(fmt.Errorf("unable to patch vm: %v", err))
	}
	return nil
}

func (app *SubresourceAPIApp) vmiVolumePatch(name, namespace string, volumeRequest *v1.VirtualMachineVolumeRequest) *errors.StatusError {
	vmi, statErr := app.FetchVirtualMachineInstance(namespace, name)
	if statErr != nil {
		return statErr
	}

	if ownedByVirtualMachine(vmi) && !app.ephemeralHotplugSupported() {
		return errors.NewBadRequest(fmt.Sprintf("VMI %s/%s is owned by a VM", vmi.Namespace, vmi.Name))
	}

	if !vmi.IsRunning() {
		return errors.NewConflict(v1.Resource("virtualmachineinstance"), name, fmt.Errorf(vmiNotRunning))
	}

	err := verifyVolumeOption(vmi.Spec.Volumes, volumeRequest)
	if err != nil {
		return errors.NewConflict(v1.Resource("virtualmachineinstance"), name, err)
	}

	patchBytes, err := generateVolumeRequestPatchVMI(&vmi.Spec, volumeRequest)
	if err != nil {
		return errors.NewConflict(v1.Resource("virtualmachineinstance"), name, err)
	}

	dryRunOption := getDryRunOption(volumeRequest)
	log.Log.Object(vmi).V(4).Infof("Patching VMI: %s", string(patchBytes))
	if _, err := app.virtCli.VirtualMachineInstance(vmi.Namespace).Patch(context.Background(), vmi.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{DryRun: dryRunOption}); err != nil {
		log.Log.Object(vmi).Errorf("unable to patch vmi: %v", err)
		if errors.IsInvalid(err) {
			if statErr, ok := err.(*errors.StatusError); ok {
				return statErr
			}
		}
		return errors.NewInternalError(fmt.Errorf("unable to patch vmi: %v", err))
	}
	return nil
}

func (app *SubresourceAPIApp) vmVolumePatchStatus(name, namespace string, volumeRequest *v1.VirtualMachineVolumeRequest) *errors.StatusError {
	vm, statErr := app.fetchVirtualMachine(name, namespace)
	if statErr != nil {
		return statErr
	}

	err := verifyVolumeOption(vm.Spec.Template.Spec.Volumes, volumeRequest)
	if err != nil {
		return errors.NewConflict(v1.Resource("virtualmachine"), name, err)
	}

	patchBytes, err := generateVMVolumeRequestPatch(vm, volumeRequest)
	if err != nil {
		return errors.NewConflict(v1.Resource("virtualmachine"), name, err)
	}

	dryRunOption := getDryRunOption(volumeRequest)
	log.Log.Object(vm).V(4).Infof(patchingVMFmt, string(patchBytes))
	if _, err = app.virtCli.VirtualMachine(vm.Namespace).PatchStatus(context.Background(), vm.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{DryRun: dryRunOption}); err != nil {
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

func getDryRunOption(volumeRequest *v1.VirtualMachineVolumeRequest) []string {
	var dryRunOption []string
	if options := volumeRequest.AddVolumeOptions; options != nil && options.DryRun != nil && options.DryRun[0] == metav1.DryRunAll {
		dryRunOption = volumeRequest.AddVolumeOptions.DryRun
	} else if options := volumeRequest.RemoveVolumeOptions; options != nil && options.DryRun != nil && options.DryRun[0] == metav1.DryRunAll {
		dryRunOption = volumeRequest.RemoveVolumeOptions.DryRun
	}
	return dryRunOption
}

func verifyVolumeOption(volumes []v1.Volume, volumeRequest *v1.VirtualMachineVolumeRequest) error {
	foundRemoveVol := false
	for _, volume := range volumes {
		if volumeRequest.AddVolumeOptions != nil {
			volSourceName := volumeSourceName(volumeRequest.AddVolumeOptions.VolumeSource)
			if volumeNameExists(volume, volumeRequest.AddVolumeOptions.Name) {
				return fmt.Errorf("Unable to add volume [%s] because volume with that name already exists", volumeRequest.AddVolumeOptions.Name)
			}
			if volumeSourceExists(volume, volSourceName) {
				return fmt.Errorf("Unable to add volume source [%s] because it already exists", volSourceName)
			}
		} else if volumeRequest.RemoveVolumeOptions != nil && volumeExists(volume, volumeRequest.RemoveVolumeOptions.Name) {
			if !volumeHotpluggable(volume) {
				return fmt.Errorf("Unable to remove volume [%s] because it is not hotpluggable", volume.Name)
			}
			foundRemoveVol = true
			break
		}
	}

	if volumeRequest.RemoveVolumeOptions != nil && !foundRemoveVol {
		return fmt.Errorf("Unable to remove volume [%s] because it does not exist", volumeRequest.RemoveVolumeOptions.Name)
	}

	return nil
}

func volumeSourceExists(volume v1.Volume, volumeName string) bool {
	return (volume.DataVolume != nil && volume.DataVolume.Name == volumeName) ||
		(volume.PersistentVolumeClaim != nil && volume.PersistentVolumeClaim.ClaimName == volumeName)
}

func volumeExists(volume v1.Volume, volumeName string) bool {
	return volumeNameExists(volume, volumeName) || volumeSourceExists(volume, volumeName)
}

func volumeNameExists(volume v1.Volume, volumeName string) bool {
	return volume.Name == volumeName
}

func volumeSourceName(volumeSource *v1.HotplugVolumeSource) string {
	if volumeSource.DataVolume != nil {
		return volumeSource.DataVolume.Name
	}
	if volumeSource.PersistentVolumeClaim != nil {
		return volumeSource.PersistentVolumeClaim.ClaimName
	}
	return ""
}

func generateVolumeRequestPatchVM(vmiSpec *v1.VirtualMachineInstanceSpec, volumeRequest *v1.VirtualMachineVolumeRequest) ([]byte, error) {
	return generateVolumeRequestPatch("/spec/template", vmiSpec, volumeRequest)
}

func generateVolumeRequestPatchVMI(vmiSpec *v1.VirtualMachineInstanceSpec, volumeRequest *v1.VirtualMachineVolumeRequest) ([]byte, error) {
	return generateVolumeRequestPatch("", vmiSpec, volumeRequest)
}

func generateVolumeRequestPatch(prefix string, vmiSpec *v1.VirtualMachineInstanceSpec, volumeRequest *v1.VirtualMachineVolumeRequest) ([]byte, error) {
	volumePath := prefix + "/spec/volumes"
	diskPath := prefix + "/spec/domain/devices/disks"
	vmiSpecCopy := *controller.ApplyVolumeRequestOnVMISpec(vmiSpec.DeepCopy(), volumeRequest)

	patchSet := patch.New(
		patch.WithTest(volumePath, vmiSpec.Volumes),
		patch.WithTest(diskPath, vmiSpec.Domain.Devices.Disks),
	)

	if len(vmiSpec.Volumes) > 0 {
		patchSet.AddOption(patch.WithReplace(volumePath, vmiSpecCopy.Volumes))
	} else {
		patchSet.AddOption(patch.WithAdd(volumePath, vmiSpecCopy.Volumes))
	}

	if len(vmiSpec.Domain.Devices.Disks) > 0 {
		patchSet.AddOption(patch.WithReplace(diskPath, vmiSpecCopy.Domain.Devices.Disks))
	} else {
		patchSet.AddOption(patch.WithAdd(diskPath, vmiSpecCopy.Domain.Devices.Disks))
	}

	return patchSet.GeneratePayload()
}

func volumeHotpluggable(volume v1.Volume) bool {
	return (volume.DataVolume != nil && volume.DataVolume.Hotpluggable) || (volume.PersistentVolumeClaim != nil && volume.PersistentVolumeClaim.Hotpluggable)
}

func generateVMVolumeRequestPatch(vm *v1.VirtualMachine, volumeRequest *v1.VirtualMachineVolumeRequest) ([]byte, error) {
	vmCopy := vm.DeepCopy()

	// We only validate the list against other items in the list at this point.
	// The VM validation webhook will validate the list against the VMI spec
	// during the Patch command
	if volumeRequest.AddVolumeOptions != nil {
		if err := addAddVolumeRequests(vm, volumeRequest, vmCopy); err != nil {
			return nil, err
		}
	} else if volumeRequest.RemoveVolumeOptions != nil {
		if err := addRemoveVolumeRequests(vm, volumeRequest, vmCopy); err != nil {
			return nil, err
		}
	}

	patchSet := patch.New(
		patch.WithTest("/status/volumeRequests", vm.Status.VolumeRequests),
	)

	if len(vm.Status.VolumeRequests) > 0 {
		patchSet.AddOption(patch.WithReplace("/status/volumeRequests", vmCopy.Status.VolumeRequests))
	} else {
		patchSet.AddOption(patch.WithAdd("/status/volumeRequests", vmCopy.Status.VolumeRequests))
	}
	return patchSet.GeneratePayload()
}

func addAddVolumeRequests(vm *v1.VirtualMachine, volumeRequest *v1.VirtualMachineVolumeRequest, vmCopy *v1.VirtualMachine) error {
	name := volumeRequest.AddVolumeOptions.Name
	for _, request := range vm.Status.VolumeRequests {
		if err := validateAddVolumeRequest(request, name); err != nil {
			return err
		}
	}
	vmCopy.Status.VolumeRequests = append(vm.Status.VolumeRequests, *volumeRequest)
	return nil
}

func validateAddVolumeRequest(request v1.VirtualMachineVolumeRequest, name string) error {
	if addVolumeRequestExists(request, name) {
		return fmt.Errorf("add volume request for volume [%s] already exists", name)
	}
	if removeVolumeRequestExists(request, name) {
		return fmt.Errorf("unable to add volume since a remove volume request for volume [%s] already exists and is still being processed", name)
	}
	return nil
}

func addRemoveVolumeRequests(vm *v1.VirtualMachine, volumeRequest *v1.VirtualMachineVolumeRequest, vmCopy *v1.VirtualMachine) error {
	name := volumeRequest.RemoveVolumeOptions.Name
	var volumeRequestsList []v1.VirtualMachineVolumeRequest
	for _, request := range vm.Status.VolumeRequests {
		if addVolumeRequestExists(request, name) {
			// Filter matching AddVolume requests from the new list.
			continue
		}
		if removeVolumeRequestExists(request, name) {
			return fmt.Errorf("a remove volume request for volume [%s] already exists and is still being processed", name)
		}
		volumeRequestsList = append(volumeRequestsList, request)
	}
	volumeRequestsList = append(volumeRequestsList, *volumeRequest)
	vmCopy.Status.VolumeRequests = volumeRequestsList
	return nil
}

func removeVolumeRequestExists(request v1.VirtualMachineVolumeRequest, name string) bool {
	return request.RemoveVolumeOptions != nil && request.RemoveVolumeOptions.Name == name
}

func addVolumeRequestExists(request v1.VirtualMachineVolumeRequest, name string) bool {
	return request.AddVolumeOptions != nil && request.AddVolumeOptions.Name == name
}

func ownedByVirtualMachine(vmi *v1.VirtualMachineInstance) bool {
	owner := metav1.GetControllerOf(vmi)
	if owner != nil && owner.Kind == "VirtualMachine" && owner.APIVersion == v1.SchemeGroupVersion.String() {
		return true
	}

	return false
}
