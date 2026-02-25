package rest

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/emicklei/go-restful/v3"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/controller"
)

const hotplugHostDevicesWithDraNotEnabledError string = "Enable HotplugHostDevicesWithDRA feature gate to use this API."

// VMAddResourceClaimRequestHandler handles the subresource for hot plugging a resource claim and host device.
func (app *SubresourceAPIApp) VMAddResourceClaimRequestHandler(request *restful.Request, response *restful.Response) {
	app.addResourceClaimRequestHandler(request, response, false)
}

// VMRemoveResourceClaimRequestHandler handles the subresource for hot plugging a resource claim and host device.
func (app *SubresourceAPIApp) VMRemoveResourceClaimRequestHandler(request *restful.Request, response *restful.Response) {
	app.removeResourceClaimRequestHandler(request, response, false)
}

// VMIAddResourceClaimRequestHandler handles the subresource for hot plugging a resource claim and host device.
func (app *SubresourceAPIApp) VMIAddResourceClaimRequestHandler(request *restful.Request, response *restful.Response) {
	app.addResourceClaimRequestHandler(request, response, true)
}

// VMIRemoveResourceClaimRequestHandler handles the subresource for hot plugging a resource claim and host device.
func (app *SubresourceAPIApp) VMIRemoveResourceClaimRequestHandler(request *restful.Request, response *restful.Response) {
	app.removeResourceClaimRequestHandler(request, response, true)
}

func (app *SubresourceAPIApp) addResourceClaimRequestHandler(request *restful.Request, response *restful.Response, ephemeral bool) {
	name := request.PathParameter("name")
	namespace := request.PathParameter("namespace")

	if !app.hotplugHostDevicesWithDRAEnabled() {
		writeError(k8serrors.NewBadRequest(hotplugHostDevicesWithDraNotEnabledError), response)
		return
	}

	if request.Request.Body == nil {
		writeError(k8serrors.NewBadRequest("Request with no body, a new name is expected as the request body"), response)
		return
	}

	opts := &v1.AddResourceClaimOptions{}
	defer request.Request.Body.Close()
	if err := decodeBody(request, opts); err != nil {
		writeError(err, response)
		return
	}

	switch {
	case opts.Name == "":
		writeError(k8serrors.NewBadRequest("AddResourceClaimOptions requires name to be set"), response)
		return
	case opts.HostDevice == nil:
		writeError(k8serrors.NewBadRequest("AddResourceClaimOptions requires hostDevice to not be nil"), response)
		return
	case opts.ResourceClaim == nil:
		writeError(k8serrors.NewBadRequest("AddResourceClaimOptions requires resourceClaim to not be nil"), response)
		return
	}

	// override the name and hotpluggable fields
	opts.HostDevice.Name = opts.Name
	opts.ResourceClaim.PodResourceClaim.Name = opts.Name
	opts.ResourceClaim.Hotpluggable = true

	resourceClaimRequest := v1.VirtualMachineResourceClaimRequest{
		AddResourceClaimOptions: opts,
	}

	// inject into VMI if ephemeral, else set as a request on the VM to both make permanent and hotplug.
	if ephemeral {
		if err := app.vmiResourceClaimPatch(name, namespace, &resourceClaimRequest); err != nil {
			writeError(err, response)
		}
		return
	}
	if err := app.vmResourceClaimPatchStatus(name, namespace, &resourceClaimRequest); err != nil {
		writeError(err, response)
		return
	}

	response.WriteHeader(http.StatusAccepted)
}

func (app *SubresourceAPIApp) removeResourceClaimRequestHandler(request *restful.Request, response *restful.Response, ephemeral bool) {
	name := request.PathParameter("name")
	namespace := request.PathParameter("namespace")

	if !app.hotplugHostDevicesWithDRAEnabled() {
		writeError(k8serrors.NewBadRequest(hotplugHostDevicesWithDraNotEnabledError), response)
		return
	}

	if request.Request.Body == nil {
		writeError(k8serrors.NewBadRequest("Request with no body, a new name is expected as the request body"),
			response)
		return
	}
	opts := &v1.RemoveResourceClaimOptions{}
	defer request.Request.Body.Close()
	if err := decodeBody(request, opts); err != nil {
		writeError(err, response)
		return
	}

	if opts.Name == "" {
		writeError(k8serrors.NewBadRequest("RemoveResourceClaimOptions requires name to be set"), response)
		return
	}

	resourceClaimRequest := v1.VirtualMachineResourceClaimRequest{
		RemoveResourceClaimOptions: opts,
	}

	// inject into VMI if ephemeral, else set as a request on the VM to both make permanent and hotplug.
	if ephemeral {
		if err := app.vmiResourceClaimPatch(name, namespace, &resourceClaimRequest); err != nil {
			writeError(err, response)
			return
		}

	} else {
		if err := app.vmResourceClaimPatchStatus(name, namespace, &resourceClaimRequest); err != nil {
			writeError(err, response)
			return
		}
	}

	response.WriteHeader(http.StatusAccepted)
}

func (app *SubresourceAPIApp) vmiResourceClaimPatch(name, namespace string, resourceClaimRequest *v1.VirtualMachineResourceClaimRequest) *k8serrors.StatusError {
	vmi, statErr := app.FetchVirtualMachineInstance(namespace, name)
	if statErr != nil {
		return statErr
	}

	if !vmi.IsRunning() {
		return k8serrors.NewConflict(v1.Resource("virtualmachineinstance"), name, fmt.Errorf(vmiNotRunning))
	}

	err := verifyResourceClaimOptions(vmi.Spec.ResourceClaims, resourceClaimRequest)
	if err != nil {
		return k8serrors.NewConflict(v1.Resource("virtualmachineinstance"), name, err)
	}

	patchBytes, err := generateResourceClaimRequestPatch(&vmi.Spec, resourceClaimRequest)
	if err != nil {
		return k8serrors.NewConflict(v1.Resource("virtualmachineinstance"), name, err)
	}

	dryRunOption := getResourceClaimDryRunOption(resourceClaimRequest)
	log.Log.Object(vmi).V(4).Infof("Patching VMI: %s", string(patchBytes))

	if _, err := app.virtCli.VirtualMachineInstance(vmi.Namespace).Patch(context.Background(), vmi.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{DryRun: dryRunOption}); err != nil {
		log.Log.Object(vmi).Errorf("unable to patch vmi: %v", err)
		if k8serrors.IsInvalid(err) {
			var statErr *k8serrors.StatusError
			if errors.As(err, &statErr) {
				return statErr
			}
		}
		return k8serrors.NewInternalError(fmt.Errorf("unable to patch vmi: %v", err))
	}
	return nil
}

func (app *SubresourceAPIApp) vmResourceClaimPatchStatus(name, namespace string, resourceClaimRequest *v1.VirtualMachineResourceClaimRequest) *k8serrors.StatusError {
	vm, statErr := app.fetchVirtualMachine(name, namespace)
	if statErr != nil {
		return statErr
	}

	err := verifyResourceClaimOptions(vm.Spec.Template.Spec.ResourceClaims, resourceClaimRequest)
	if err != nil {
		return k8serrors.NewConflict(v1.Resource("virtualmachine"), name, err)
	}

	patchBytes, err := generateVMResourceClaimRequestPatch(vm, resourceClaimRequest)
	if err != nil {
		return k8serrors.NewConflict(v1.Resource("virtualmachine"), name, err)
	}

	dryRunOption := getResourceClaimDryRunOption(resourceClaimRequest)
	log.Log.Object(vm).V(4).Infof(patchingVMStatusFmt, string(patchBytes))

	if _, err = app.virtCli.VirtualMachine(vm.Namespace).PatchStatus(context.Background(), vm.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{DryRun: dryRunOption}); err != nil {
		log.Log.Object(vm).Errorf("unable to patch vm status: %v", err)
		if k8serrors.IsInvalid(err) {
			var statErr *k8serrors.StatusError
			if errors.As(err, &statErr) {
				return statErr
			}
		}
		return k8serrors.NewInternalError(fmt.Errorf("unable to patch vm status: %v", err))
	}
	return nil
}

func verifyResourceClaimOptions(resourceClaims []v1.ResourceClaim, resourceClaimRequest *v1.VirtualMachineResourceClaimRequest) error {
	add := resourceClaimRequest.AddResourceClaimOptions
	remove := resourceClaimRequest.RemoveResourceClaimOptions

	switch {
	case add != nil:
		return verifyResourceClaimAddRequest(resourceClaims, add)
	case remove != nil:
		return verifyResourceClaimRemoveRequest(resourceClaims, remove)
	default:
		return fmt.Errorf("Must specify either AddResourceClaimOptions or RemoveResourceClaimOptions")
	}
}

func verifyResourceClaimAddRequest(resourceClaims []v1.ResourceClaim, add *v1.AddResourceClaimOptions) error {
	for _, resourceClaim := range resourceClaims {
		if resourceClaimNameExists(resourceClaim, add.Name) {
			return fmt.Errorf("Unable to add ResourceClaim [%s] because ResourceClaim with that name already exists", add.Name)
		}

		switch {
		case resourceClaim.PodResourceClaim.ResourceClaimName != nil:
			if add.ResourceClaim.PodResourceClaim.ResourceClaimName != nil {
				if *add.ResourceClaim.PodResourceClaim.ResourceClaimName == *resourceClaim.PodResourceClaim.ResourceClaimName {
					return fmt.Errorf("Unable to add ResourceClaimName [%s] because it already exists", add.Name)
				}
			}
		case resourceClaim.PodResourceClaim.ResourceClaimTemplateName != nil:
			if add.ResourceClaim.PodResourceClaim.ResourceClaimTemplateName != nil {
				if *add.ResourceClaim.PodResourceClaim.ResourceClaimTemplateName == *resourceClaim.PodResourceClaim.ResourceClaimTemplateName {
					return fmt.Errorf("Unable to add ResourceClaimTemplateName [%s] because it already exists", add.Name)
				}
			}
		}
	}

	return nil
}

func verifyResourceClaimRemoveRequest(resourceClaims []v1.ResourceClaim, remove *v1.RemoveResourceClaimOptions) error {
	for _, resourceClaim := range resourceClaims {
		if resourceClaimNameExists(resourceClaim, remove.Name) {
			if !resourceClaimHotpluggable(resourceClaim) {
				return fmt.Errorf("Unable to remove ResourceClaim [%s] because it is not hotpluggable", remove.Name)
			}
			return nil
		}
	}
	return fmt.Errorf("Unable to remove ResourceClaim [%s] because it does not exist", remove.Name)
}

func resourceClaimOrTemplateName(resourceClaim *v1.ResourceClaim) string {
	if resourceClaim == nil {
		return ""
	}
	if resourceClaim.ResourceClaimTemplateName != nil {
		return *resourceClaim.ResourceClaimTemplateName
	}
	if resourceClaim.ResourceClaimName != nil {
		return *resourceClaim.ResourceClaimName
	}
	return ""
}

func resourceClaimNameOrTemplateExists(resourceClaim *v1.ResourceClaim, resourceClaimOrTmplName string) bool {
	return resourceClaimOrTemplateName(resourceClaim) == resourceClaimOrTmplName
}

func resourceClaimNameExists(resourceClaim v1.ResourceClaim, resourceClaimName string) bool {
	return resourceClaim.Name == resourceClaimName
}

func resourceClaimHotpluggable(resourceClaim v1.ResourceClaim) bool {
	return resourceClaim.Hotpluggable
}

func generateResourceClaimRequestPatch(vmiSpec *v1.VirtualMachineInstanceSpec, resourceClaimRequest *v1.VirtualMachineResourceClaimRequest) ([]byte, error) {
	const resourceClaimPath = "/spec/resourceClaims"
	const hostDevicePath = "/spec/domain/devices/hostDevices"

	vmiSpecCopy := controller.ApplyResourceClaimRequestOnVMISpec(vmiSpec.DeepCopy(), resourceClaimRequest)

	patchSet := patch.New(
		patch.WithTest(resourceClaimPath, vmiSpec.ResourceClaims),
		patch.WithTest(hostDevicePath, vmiSpec.Domain.Devices.HostDevices),
	)

	if len(vmiSpec.ResourceClaims) > 0 {
		patchSet.AddOption(patch.WithReplace(resourceClaimPath, vmiSpecCopy.ResourceClaims))
	} else {
		patchSet.AddOption(patch.WithAdd(resourceClaimPath, vmiSpecCopy.ResourceClaims))
	}

	if len(vmiSpec.Domain.Devices.HostDevices) > 0 {
		patchSet.AddOption(patch.WithReplace(hostDevicePath, vmiSpecCopy.Domain.Devices.HostDevices))
	} else {
		patchSet.AddOption(patch.WithAdd(hostDevicePath, vmiSpecCopy.Domain.Devices.HostDevices))
	}

	return patchSet.GeneratePayload()
}

func generateVMResourceClaimRequestPatch(vm *v1.VirtualMachine, resourceClaimRequest *v1.VirtualMachineResourceClaimRequest) ([]byte, error) {
	vmCopy := vm.DeepCopy()

	// We only validate the list against other items in the list at this point.
	// The VM validation webhook will validate the list against the VMI spec
	// during the Patch command
	if resourceClaimRequest.AddResourceClaimOptions != nil {
		if err := addAddResourceClaimRequests(vm, resourceClaimRequest, vmCopy); err != nil {
			return nil, err
		}
	} else if resourceClaimRequest.RemoveResourceClaimOptions != nil {
		if err := addRemoveResourceClaimRequests(vm, resourceClaimRequest, vmCopy); err != nil {
			return nil, err
		}
	}

	patchSet := patch.New(
		patch.WithTest("/status/resourceClaimRequests", vm.Status.ResourceClaimRequests),
	)

	if len(vm.Status.ResourceClaimRequests) > 0 {
		patchSet.AddOption(patch.WithReplace("/status/resourceClaimRequests", vmCopy.Status.ResourceClaimRequests))
	} else {
		patchSet.AddOption(patch.WithAdd("/status/resourceClaimRequests", vmCopy.Status.ResourceClaimRequests))
	}
	return patchSet.GeneratePayload()
}

func addAddResourceClaimRequests(vm *v1.VirtualMachine, resourceClaimRequest *v1.VirtualMachineResourceClaimRequest, vmCopy *v1.VirtualMachine) error {
	name := resourceClaimRequest.AddResourceClaimOptions.Name
	for _, request := range vm.Status.ResourceClaimRequests {
		if addResourceClaimRequestExists(request, name) {
			return fmt.Errorf("add resource claim request for resource claim [%s] already exists", name)
		}
		if removeResourceClaimRequestExists(request, name) {
			return fmt.Errorf("unable to add resource claim since a remove resource claim request for resource claim [%s] already exists and is still being processed", name)
		}
	}
	vmCopy.Status.ResourceClaimRequests = append(vm.Status.ResourceClaimRequests, *resourceClaimRequest)
	return nil
}

func addRemoveResourceClaimRequests(vm *v1.VirtualMachine, resourceClaimRequest *v1.VirtualMachineResourceClaimRequest, vmCopy *v1.VirtualMachine) error {
	name := resourceClaimRequest.RemoveResourceClaimOptions.Name
	var resourceClaimRequestsList []v1.VirtualMachineResourceClaimRequest
	for _, request := range vm.Status.ResourceClaimRequests {
		if addResourceClaimRequestExists(request, name) {
			// Filter matching AddVolume requests from the new list.
			continue
		}
		if removeResourceClaimRequestExists(request, name) {
			return fmt.Errorf("a remove resource claim request for resource claim [%s] already exists and is still being processed", name)
		}
		resourceClaimRequestsList = append(resourceClaimRequestsList, request)
	}
	resourceClaimRequestsList = append(resourceClaimRequestsList, *resourceClaimRequest)
	vmCopy.Status.ResourceClaimRequests = resourceClaimRequestsList
	return nil
}

func addResourceClaimRequestExists(request v1.VirtualMachineResourceClaimRequest, name string) bool {
	return request.AddResourceClaimOptions != nil && request.AddResourceClaimOptions.Name == name
}

func removeResourceClaimRequestExists(request v1.VirtualMachineResourceClaimRequest, name string) bool {
	return request.RemoveResourceClaimOptions != nil && request.RemoveResourceClaimOptions.Name == name
}

func getResourceClaimDryRunOption(resourceClaimRequest *v1.VirtualMachineResourceClaimRequest) []string {
	var dryRunOption []string
	if options := resourceClaimRequest.AddResourceClaimOptions; options != nil && options.DryRun != nil && options.DryRun[0] == metav1.DryRunAll {
		dryRunOption = resourceClaimRequest.AddResourceClaimOptions.DryRun
	} else if options := resourceClaimRequest.RemoveResourceClaimOptions; options != nil && options.DryRun != nil && options.DryRun[0] == metav1.DryRunAll {
		dryRunOption = resourceClaimRequest.RemoveResourceClaimOptions.DryRun
	}
	return dryRunOption
}

func (app *SubresourceAPIApp) hotplugHostDevicesWithDRAEnabled() bool {
	return app.clusterConfig.HotplugHostDevicesWithDRAEnabled()
}
