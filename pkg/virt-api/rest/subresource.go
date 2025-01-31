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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package rest

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/emicklei/go-restful/v3"
	"k8s.io/apimachinery/pkg/api/errors"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/util/yaml"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/instancetype/expand"
	"kubevirt.io/kubevirt/pkg/instancetype/find"
	preferenceFind "kubevirt.io/kubevirt/pkg/instancetype/preference/find"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

const (
	unmarshalRequestErrFmt                   = "Can not unmarshal Request body to struct, error: %s"
	vmNotRunning                             = "VM is not running"
	patchingVMFmt                            = "Patching VM: %s"
	jsonpatchTestErr                         = "jsonpatch test operation does not apply"
	patchingVMStatusFmt                      = "Patching VM status: %s"
	vmiNotRunning                            = "VMI is not running"
	vmiNotPaused                             = "VMI is not paused"
	vmiGuestAgentErr                         = "VMI does not have guest agent connected"
	vmiNoAttestationErr                      = "Attestation not requested for VMI"
	prepConnectionErrFmt                     = "Cannot prepare connection %s"
	getRequestErrFmt                         = "Cannot GET request %s"
	pvcVolumeModeErr                         = "pvc should be filesystem pvc"
	pvcAccessModeErr                         = "pvc access mode can't be read only"
	pvcSizeErrFmt                            = "pvc size [%s] should be bigger then [%s]"
	memoryDumpNameConflictErr                = "can't request memory dump for pvc [%s] while pvc [%s] is still associated as the memory dump pvc"
	featureGateDisabledErrFmt                = "'%s' feature gate is not enabled"
	defaultProfilerComponentPort             = 8443
	volumeMigrationManualRecoveryRequiredErr = "VM recovery required: Volume migration failed, leaving some volumes pointing to non-consistent targets; manual intervention is needed to reassign them to their original volumes."
)

type instancetypeVMExpander interface {
	Expand(vm *v1.VirtualMachine) (*v1.VirtualMachine, error)
}

type SubresourceAPIApp struct {
	virtCli                 kubecli.KubevirtClient
	consoleServerPort       int
	profilerComponentPort   int
	handlerTLSConfiguration *tls.Config
	credentialsLock         *sync.Mutex
	clusterConfig           *virtconfig.ClusterConfig
	instancetypeExpander    instancetypeVMExpander
	handlerHttpClient       *http.Client
}

func NewSubresourceAPIApp(virtCli kubecli.KubevirtClient, consoleServerPort int, tlsConfiguration *tls.Config, clusterConfig *virtconfig.ClusterConfig) *SubresourceAPIApp {
	// When this method is called from tools/openapispec.go when running 'make generate',
	// the virtCli is nil, and accessing GeneratedKubeVirtClient() would cause nil dereference.
	var instancetypeExpander instancetypeVMExpander
	if virtCli != nil {
		instancetypeExpander = expand.New(
			clusterConfig,
			find.NewSpecFinder(nil, nil, nil, virtCli),
			preferenceFind.NewSpecFinder(nil, nil, nil, virtCli),
		)
	}

	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfiguration,
		},
		Timeout: 10 * time.Second,
	}

	return &SubresourceAPIApp{
		virtCli:                 virtCli,
		consoleServerPort:       consoleServerPort,
		profilerComponentPort:   defaultProfilerComponentPort,
		credentialsLock:         &sync.Mutex{},
		handlerTLSConfiguration: tlsConfiguration,
		clusterConfig:           clusterConfig,
		instancetypeExpander:    instancetypeExpander,
		handlerHttpClient:       httpClient,
	}
}

type validation func(*v1.VirtualMachineInstance) (err *errors.StatusError)

// This function prototype is used with putRequestHandlerWithErrorPostProcessing.
// The errorPostProcessing function will get called if an error occurs when attempting
// to make a request to virt-handler. Depending on where in the stack the error occurred
// the VMI might be nil.
//
// Use this function to inject more human readible context into the error response.
type errorPostProcessing func(*v1.VirtualMachineInstance, error) (err error)
type URLResolver func(*v1.VirtualMachineInstance, kubecli.VirtHandlerConn) (string, error)

func (app *SubresourceAPIApp) prepareConnection(request *restful.Request, validate validation, getVirtHandlerURL URLResolver) (vmi *v1.VirtualMachineInstance, url string, conn kubecli.VirtHandlerConn, statusError *errors.StatusError) {

	vmiName := request.PathParameter("name")
	namespace := request.PathParameter("namespace")

	vmi, statusError = app.fetchAndValidateVirtualMachineInstance(namespace, vmiName, validate)
	if statusError != nil {
		return
	}

	url, conn, statusError = app.getVirtHandlerFor(vmi, getVirtHandlerURL)
	if statusError != nil {
		return
	}

	return
}

func (app *SubresourceAPIApp) fetchAndValidateVirtualMachineInstance(namespace, vmiName string, validate validation) (vmi *v1.VirtualMachineInstance, statusError *errors.StatusError) {
	vmi, statusError = app.FetchVirtualMachineInstance(namespace, vmiName)
	if statusError != nil {
		log.Log.Reason(statusError).Errorf("Failed to gather vmi %s in namespace %s.", vmiName, namespace)
		return
	}

	if statusError = validate(vmi); statusError != nil {
		return
	}
	return
}

func (app *SubresourceAPIApp) putRequestHandler(request *restful.Request, response *restful.Response, preValidate validation, getVirtHandlerURL URLResolver, dryRun bool) {

	app.putRequestHandlerWithErrorPostProcessing(request, response, preValidate, nil, getVirtHandlerURL, dryRun)
}

func (app *SubresourceAPIApp) putRequestHandlerWithErrorPostProcessing(request *restful.Request, response *restful.Response, preValidate validation, errorPostProcessing errorPostProcessing, getVirtHandlerURL URLResolver, dryRun bool) {

	if preValidate == nil {
		preValidate = func(vmi *v1.VirtualMachineInstance) *errors.StatusError { return nil }
	}
	if errorPostProcessing == nil {
		errorPostProcessing = func(vmi *v1.VirtualMachineInstance, err error) error { return err }
	}

	vmi, url, conn, statusErr := app.prepareConnection(request, preValidate, getVirtHandlerURL)
	if statusErr != nil {
		err := errorPostProcessing(vmi, fmt.Errorf(statusErr.ErrStatus.Message))
		statusErr.ErrStatus.Message = err.Error()
		writeError(statusErr, response)
		return
	}

	if dryRun {
		return
	}
	err := conn.Put(url, request.Request.Body)
	if err != nil {
		err = errorPostProcessing(vmi, err)
		writeError(errors.NewInternalError(err), response)
		return
	}
}

func (app *SubresourceAPIApp) httpGetRequestHandler(request *restful.Request, response *restful.Response, validate validation, getURL URLResolver, v interface{}) {
	_, url, conn, err := app.prepareConnection(request, validate, getURL)
	if err != nil {
		log.Log.Errorf(prepConnectionErrFmt, err.Error())
		response.WriteError(http.StatusInternalServerError, err)
		return
	}

	resp, conErr := conn.Get(url)
	if conErr != nil {
		log.Log.Errorf(getRequestErrFmt, conErr.Error())
		response.WriteError(http.StatusInternalServerError, conErr)
		return
	}

	if err := json.Unmarshal([]byte(resp), &v); err != nil {
		log.Log.Reason(err).Error("error unmarshalling response")
		response.WriteError(http.StatusInternalServerError, err)
		return
	}

	response.WriteEntity(v)
}

func (app *SubresourceAPIApp) fetchVirtualMachine(name string, namespace string) (*v1.VirtualMachine, *errors.StatusError) {

	vm, err := app.virtCli.VirtualMachine(namespace).Get(context.Background(), name, k8smetav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, errors.NewNotFound(v1.Resource("virtualmachine"), name)
		}
		return nil, errors.NewInternalError(fmt.Errorf("unable to retrieve vm [%s]: %v", name, err))
	}
	return vm, nil
}

// FetchVirtualMachineInstance by namespace and name
func (app *SubresourceAPIApp) FetchVirtualMachineInstance(namespace, name string) (*v1.VirtualMachineInstance, *errors.StatusError) {

	vmi, err := app.virtCli.VirtualMachineInstance(namespace).Get(context.Background(), name, k8smetav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, errors.NewNotFound(v1.Resource("virtualmachineinstance"), name)
		}
		return nil, errors.NewInternalError(fmt.Errorf("unable to retrieve vmi [%s]: %v", name, err))
	}
	return vmi, nil
}

// FetchVirtualMachineInstanceForVM by namespace and name
func (app *SubresourceAPIApp) FetchVirtualMachineInstanceForVM(namespace, name string) (*v1.VirtualMachineInstance, *errors.StatusError) {
	vm, err := app.virtCli.VirtualMachine(namespace).Get(context.Background(), name, k8smetav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, errors.NewNotFound(v1.Resource("virtualmachine"), name)
		}
		return nil, errors.NewInternalError(fmt.Errorf("unable to retrieve vm [%s]: %v", name, err))
	}

	if !vm.Status.Created {
		return nil, errors.NewConflict(v1.Resource("virtualmachine"), vm.Name, fmt.Errorf("VMI is not started"))
	}

	vmi, err := app.virtCli.VirtualMachineInstance(namespace).Get(context.Background(), name, k8smetav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, errors.NewNotFound(v1.Resource("virtualmachineinstance"), name)
		}
		return nil, errors.NewInternalError(fmt.Errorf("unable to retrieve vmi [%s]: %v", name, err))
	}

	for _, ref := range vmi.OwnerReferences {
		if ref.UID == vm.UID {
			return vmi, nil
		}
	}

	return nil, errors.NewInternalError(fmt.Errorf("unable to retrieve vmi [%s] for vm: %v", name, err))
}

func writeError(error *errors.StatusError, response *restful.Response) {
	errStatus := error.ErrStatus.DeepCopy()
	errStatus.Kind = "Status"
	errStatus.APIVersion = "v1"
	err := response.WriteHeaderAndJson(int(error.Status().Code), errStatus, restful.MIME_JSON)
	if err != nil {
		log.Log.Reason(err).Error("Failed to write http response.")
	}
}

// GuestOSInfo handles the subresource for providing VM guest agent information
func (app *SubresourceAPIApp) GuestOSInfo(request *restful.Request, response *restful.Response) {
	validate := func(vmi *v1.VirtualMachineInstance) *errors.StatusError {
		if vmi == nil || vmi.Status.Phase != v1.Running {
			return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf(vmiNotRunning))
		}
		condManager := controller.NewVirtualMachineInstanceConditionManager()
		if !condManager.HasCondition(vmi, v1.VirtualMachineInstanceAgentConnected) {
			return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf(vmiGuestAgentErr))
		}
		return nil
	}
	getURL := func(vmi *v1.VirtualMachineInstance, conn kubecli.VirtHandlerConn) (string, error) {
		return conn.GuestInfoURI(vmi)
	}

	app.httpGetRequestHandler(request, response, validate, getURL, v1.VirtualMachineInstanceGuestAgentInfo{})
}

// UserList handles the subresource for providing VM guest user list
func (app *SubresourceAPIApp) UserList(request *restful.Request, response *restful.Response) {
	validate := func(vmi *v1.VirtualMachineInstance) *errors.StatusError {
		if vmi == nil || vmi.Status.Phase != v1.Running {
			return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf(vmiNotRunning))
		}
		condManager := controller.NewVirtualMachineInstanceConditionManager()
		if !condManager.HasCondition(vmi, v1.VirtualMachineInstanceAgentConnected) {
			return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf(vmiGuestAgentErr))
		}
		return nil
	}
	getURL := func(vmi *v1.VirtualMachineInstance, conn kubecli.VirtHandlerConn) (string, error) {
		return conn.UserListURI(vmi)
	}

	app.httpGetRequestHandler(request, response, validate, getURL, v1.VirtualMachineInstanceGuestOSUserList{})
}

// FilesystemList handles the subresource for providing guest filesystem list
func (app *SubresourceAPIApp) FilesystemList(request *restful.Request, response *restful.Response) {
	validate := func(vmi *v1.VirtualMachineInstance) *errors.StatusError {
		if vmi == nil || vmi.Status.Phase != v1.Running {
			return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf(vmiNotRunning))
		}
		condManager := controller.NewVirtualMachineInstanceConditionManager()
		if !condManager.HasCondition(vmi, v1.VirtualMachineInstanceAgentConnected) {
			return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf(vmiGuestAgentErr))
		}
		return nil
	}
	getURL := func(vmi *v1.VirtualMachineInstance, conn kubecli.VirtHandlerConn) (string, error) {
		return conn.FilesystemListURI(vmi)
	}

	app.httpGetRequestHandler(request, response, validate, getURL, v1.VirtualMachineInstanceFileSystemList{})
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

func volumeHotpluggable(volume v1.Volume) bool {
	return (volume.DataVolume != nil && volume.DataVolume.Hotpluggable) || (volume.PersistentVolumeClaim != nil && volume.PersistentVolumeClaim.Hotpluggable)
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

func volumeSourceExists(volume v1.Volume, volumeName string) bool {
	return (volume.DataVolume != nil && volume.DataVolume.Name == volumeName) ||
		(volume.PersistentVolumeClaim != nil && volume.PersistentVolumeClaim.ClaimName == volumeName)
}

func volumeExists(volume v1.Volume, volumeName string) bool {
	return volumeNameExists(volume, volumeName) || volumeSourceExists(volume, volumeName)
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

func generateVMIVolumeRequestPatch(vmi *v1.VirtualMachineInstance, volumeRequest *v1.VirtualMachineVolumeRequest) ([]byte, error) {
	vmiCopy := vmi.DeepCopy()
	vmiCopy.Spec = *controller.ApplyVolumeRequestOnVMISpec(&vmiCopy.Spec, volumeRequest)

	patchSet := patch.New(
		patch.WithTest("/spec/volumes", vmi.Spec.Volumes),
		patch.WithTest("/spec/domain/devices/disks", vmi.Spec.Domain.Devices.Disks),
	)

	if len(vmi.Spec.Volumes) > 0 {
		patchSet.AddOption(patch.WithReplace("/spec/volumes", vmiCopy.Spec.Volumes))
	} else {
		patchSet.AddOption(patch.WithAdd("/spec/volumes", vmiCopy.Spec.Volumes))
	}

	if len(vmi.Spec.Domain.Devices.Disks) > 0 {
		patchSet.AddOption(patch.WithReplace("/spec/domain/devices/disks", vmiCopy.Spec.Domain.Devices.Disks))
	} else {
		patchSet.AddOption(patch.WithAdd("/spec/domain/devices/disks", vmiCopy.Spec.Domain.Devices.Disks))
	}

	return patchSet.GeneratePayload()
}

func (app *SubresourceAPIApp) addVolumeRequestHandler(request *restful.Request, response *restful.Response, ephemeral bool) {
	name := request.PathParameter("name")
	namespace := request.PathParameter("namespace")

	if !app.clusterConfig.HotplugVolumesEnabled() {
		writeError(errors.NewBadRequest("Unable to Add Volume because HotplugVolumes feature gate is not enabled."), response)
		return
	}

	opts := &v1.AddVolumeOptions{}
	if request.Request.Body != nil {
		defer request.Request.Body.Close()
		err := yaml.NewYAMLOrJSONDecoder(request.Request.Body, 1024).Decode(opts)
		switch err {
		case io.EOF, nil:
			break
		default:
			writeError(errors.NewBadRequest(fmt.Sprintf(unmarshalRequestErrFmt, err)), response)
			return
		}
	} else {
		writeError(errors.NewBadRequest("Request with no body, a new name is expected as the request body"), response)
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
	} else {
		if err := app.vmVolumePatchStatus(name, namespace, &volumeRequest); err != nil {
			writeError(err, response)
			return
		}
	}

	response.WriteHeader(http.StatusAccepted)
}

func (app *SubresourceAPIApp) removeVolumeRequestHandler(request *restful.Request, response *restful.Response, ephemeral bool) {
	name := request.PathParameter("name")
	namespace := request.PathParameter("namespace")

	if !app.clusterConfig.HotplugVolumesEnabled() {
		writeError(errors.NewBadRequest("Unable to Remove Volume because HotplugVolumes feature gate is not enabled."), response)
		return
	}

	opts := &v1.RemoveVolumeOptions{}
	if request.Request.Body != nil {
		defer request.Request.Body.Close()
		err := yaml.NewYAMLOrJSONDecoder(request.Request.Body, 1024).Decode(opts)
		switch err {
		case io.EOF, nil:
			break
		default:
			writeError(errors.NewBadRequest(fmt.Sprintf(unmarshalRequestErrFmt,
				err)), response)
			return
		}
	} else {
		writeError(errors.NewBadRequest("Request with no body, a new name is expected as the request body"),
			response)
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
	} else {
		if err := app.vmVolumePatchStatus(name, namespace, &volumeRequest); err != nil {
			writeError(err, response)
			return
		}
	}

	response.WriteHeader(http.StatusAccepted)
}

func (app *SubresourceAPIApp) vmiVolumePatch(name, namespace string, volumeRequest *v1.VirtualMachineVolumeRequest) *errors.StatusError {
	vmi, statErr := app.FetchVirtualMachineInstance(namespace, name)
	if statErr != nil {
		return statErr
	}

	if !vmi.IsRunning() {
		return errors.NewConflict(v1.Resource("virtualmachineinstance"), name, fmt.Errorf(vmiNotRunning))
	}

	err := verifyVolumeOption(vmi.Spec.Volumes, volumeRequest)
	if err != nil {
		return errors.NewConflict(v1.Resource("virtualmachineinstance"), name, err)
	}

	patchBytes, err := generateVMIVolumeRequestPatch(vmi, volumeRequest)
	if err != nil {
		return errors.NewConflict(v1.Resource("virtualmachineinstance"), name, err)
	}

	dryRunOption := app.getDryRunOption(volumeRequest)
	log.Log.Object(vmi).V(4).Infof("Patching VMI: %s", string(patchBytes))
	if _, err := app.virtCli.VirtualMachineInstance(vmi.Namespace).Patch(context.Background(), vmi.Name, types.JSONPatchType, patchBytes, k8smetav1.PatchOptions{DryRun: dryRunOption}); err != nil {
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

	dryRunOption := app.getDryRunOption(volumeRequest)
	log.Log.Object(vm).V(4).Infof(patchingVMFmt, string(patchBytes))
	if _, err = app.virtCli.VirtualMachine(vm.Namespace).PatchStatus(context.Background(), vm.Name, types.JSONPatchType, patchBytes, k8smetav1.PatchOptions{DryRun: dryRunOption}); err != nil {
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

func (app *SubresourceAPIApp) getDryRunOption(volumeRequest *v1.VirtualMachineVolumeRequest) []string {
	var dryRunOption []string
	if options := volumeRequest.AddVolumeOptions; options != nil && options.DryRun != nil && options.DryRun[0] == k8smetav1.DryRunAll {
		dryRunOption = volumeRequest.AddVolumeOptions.DryRun
	} else if options := volumeRequest.RemoveVolumeOptions; options != nil && options.DryRun != nil && options.DryRun[0] == k8smetav1.DryRunAll {
		dryRunOption = volumeRequest.RemoveVolumeOptions.DryRun
	}
	return dryRunOption
}

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
