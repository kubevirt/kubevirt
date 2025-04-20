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
	"strings"

	"github.com/emicklei/go-restful/v3"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/pointer"
)

func (app *SubresourceAPIApp) StartVMRequestHandler(request *restful.Request, response *restful.Response) {
	name := request.PathParameter("name")
	namespace := request.PathParameter("namespace")

	vm, statusErr := app.fetchVirtualMachine(name, namespace)
	if statusErr != nil {
		writeError(statusErr, response)
		return
	}

	vmi, err := app.virtCli.VirtualMachineInstance(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		writeError(errors.NewInternalError(err), response)
		return
	}

	if vmi != nil && !vmi.IsFinal() && vmi.Status.Phase != v1.Unknown && vmi.Status.Phase != v1.VmPhaseUnset {
		writeError(errors.NewConflict(v1.Resource("virtualmachine"), name, fmt.Errorf("VM is already running")), response)
		return
	}
	if controller.NewVirtualMachineConditionManager().HasConditionWithStatus(vm, v1.VirtualMachineManualRecoveryRequired, k8sv1.ConditionTrue) {
		writeError(errors.NewConflict(v1.Resource("virtualmachine"), name, fmt.Errorf(volumeMigrationManualRecoveryRequiredErr)), response)
		return
	}

	startPaused := false
	startChangeRequestData := make(map[string]string)
	bodyStruct := &v1.StartOptions{}
	if request.Request.Body != nil {
		if err := decodeBody(request, bodyStruct); err != nil {
			writeError(err, response)
			return
		}
		startPaused = bodyStruct.Paused
	}
	if startPaused {
		startChangeRequestData[v1.StartRequestDataPausedKey] = v1.StartRequestDataPausedTrue
	}

	var patchErr error

	runStrategy, err := vm.RunStrategy()
	if err != nil {
		writeError(errors.NewInternalError(err), response)
		return
	}
	// RunStrategyHalted         -> spec.running = true / send start request for paused start
	// RunStrategyManual         -> send start request
	// RunStrategyAlways         -> doesn't make sense
	// RunStrategyRerunOnFailure -> doesn't make sense
	// RunStrategyOnce           -> doesn't make sense
	switch runStrategy {
	case v1.RunStrategyHalted:
		pausedStartStrategy := v1.StartStrategyPaused
		// Send start request if VM should start paused. virt-controller will update RunStrategy upon this request.
		// No need to send the request if StartStrategy is already set to Paused in VMI Spec.
		if startPaused && (vm.Spec.Template == nil || vm.Spec.Template.Spec.StartStrategy != &pausedStartStrategy) {
			patchBytes, err := getChangeRequestJson(vm, v1.VirtualMachineStateChangeRequest{
				Action: v1.StartRequest,
				Data:   startChangeRequestData,
			})
			if err != nil {
				writeError(errors.NewInternalError(err), response)
				return
			}
			log.Log.Object(vm).V(4).Infof(patchingVMStatusFmt, string(patchBytes))
			_, patchErr = app.virtCli.VirtualMachine(vm.Namespace).PatchStatus(context.Background(), vm.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{DryRun: bodyStruct.DryRun})
		} else {
			patchBytes, err := getRunningPatch(vm, true)
			if err != nil {
				writeError(errors.NewInternalError(err), response)
				return
			}
			log.Log.Object(vm).V(4).Infof(patchingVMFmt, string(patchBytes))
			_, patchErr = app.virtCli.VirtualMachine(namespace).Patch(context.Background(), vm.GetName(), types.JSONPatchType, patchBytes, metav1.PatchOptions{DryRun: bodyStruct.DryRun})
		}

	case v1.RunStrategyRerunOnFailure, v1.RunStrategyManual:
		needsRestart := false
		if (runStrategy == v1.RunStrategyRerunOnFailure && vmi != nil && vmi.Status.Phase == v1.Succeeded) ||
			(runStrategy == v1.RunStrategyManual && vmi != nil && vmi.IsFinal()) {
			needsRestart = true
		} else if runStrategy == v1.RunStrategyRerunOnFailure && vmi != nil && vmi.Status.Phase == v1.Failed {
			writeError(errors.NewConflict(v1.Resource("virtualmachine"), name, fmt.Errorf("%v does not support starting VM from failed state", v1.RunStrategyRerunOnFailure)), response)
			return
		}

		var patchBytes []byte
		if needsRestart {
			patchBytes, err = getChangeRequestJson(vm,
				v1.VirtualMachineStateChangeRequest{Action: v1.StopRequest, UID: &vmi.UID},
				v1.VirtualMachineStateChangeRequest{Action: v1.StartRequest, Data: startChangeRequestData})
		} else {
			patchBytes, err = getChangeRequestJson(vm,
				v1.VirtualMachineStateChangeRequest{Action: v1.StartRequest, Data: startChangeRequestData})
		}
		if err != nil {
			writeError(errors.NewInternalError(err), response)
			return
		}
		log.Log.Object(vm).V(4).Infof(patchingVMStatusFmt, string(patchBytes))
		_, patchErr = app.virtCli.VirtualMachine(vm.Namespace).PatchStatus(context.Background(), vm.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{DryRun: bodyStruct.DryRun})
	case v1.RunStrategyAlways, v1.RunStrategyOnce:
		writeError(errors.NewConflict(v1.Resource("virtualmachine"), name, fmt.Errorf("%v does not support manual start requests", runStrategy)), response)
		return
	}

	if patchErr != nil {
		if strings.Contains(patchErr.Error(), jsonpatchTestErr) {
			writeError(errors.NewConflict(v1.Resource("virtualmachine"), name, patchErr), response)
		} else {
			writeError(errors.NewInternalError(patchErr), response)
		}
		return
	}

	response.WriteHeader(http.StatusAccepted)
}

func (app *SubresourceAPIApp) StopVMRequestHandler(request *restful.Request, response *restful.Response) {
	// RunStrategyHalted         -> force stop if grace period in request is shorter than before, otherwise doesn't make sense
	// RunStrategyManual         -> send stop request
	// RunStrategyAlways         -> spec.running = false
	// RunStrategyRerunOnFailure -> send stop request
	// RunStrategyOnce           -> spec.running = false

	name := request.PathParameter("name")
	namespace := request.PathParameter("namespace")

	bodyStruct := &v1.StopOptions{}
	if request.Request.Body != nil {
		if err := decodeBody(request, bodyStruct); err != nil {
			writeError(err, response)
			return
		}
	}

	vm, statusErr := app.fetchVirtualMachine(name, namespace)
	if statusErr != nil {
		writeError(statusErr, response)
		return
	}

	runStrategy, err := vm.RunStrategy()
	if err != nil {
		writeError(errors.NewInternalError(err), response)
		return
	}

	hasVMI := true
	vmi, err := app.virtCli.VirtualMachineInstance(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil && errors.IsNotFound(err) {
		hasVMI = false
	} else if err != nil {
		writeError(errors.NewInternalError(err), response)
		return
	}

	var oldGracePeriodSeconds int64
	var patchErr error
	if hasVMI && !vmi.IsFinal() && bodyStruct.GracePeriod != nil {
		patchSet := patch.New()
		// used for stopping a VM with RunStrategyHalted
		if vmi.Spec.TerminationGracePeriodSeconds != nil {
			oldGracePeriodSeconds = *vmi.Spec.TerminationGracePeriodSeconds
			patchSet.AddOption(patch.WithTest("/spec/terminationGracePeriodSeconds", *vmi.Spec.TerminationGracePeriodSeconds))
		} else {
			patchSet.AddOption(patch.WithTest("/spec/terminationGracePeriodSeconds", nil))
		}

		patchSet.AddOption(patch.WithReplace("/spec/terminationGracePeriodSeconds", *bodyStruct.GracePeriod))
		patchBytes, err := patchSet.GeneratePayload()
		if err != nil {
			writeError(errors.NewInternalError(err), response)
			return
		}

		log.Log.Object(vmi).V(2).Infof("Patching VMI: %s", string(patchBytes))
		_, err = app.virtCli.VirtualMachineInstance(namespace).Patch(context.Background(), vmi.GetName(), types.JSONPatchType, patchBytes, metav1.PatchOptions{DryRun: bodyStruct.DryRun})
		if err != nil {
			writeError(errors.NewInternalError(err), response)
			return
		}
	}

	switch runStrategy {
	case v1.RunStrategyHalted:
		if !hasVMI || vmi.IsFinal() {
			writeError(errors.NewConflict(v1.Resource("virtualmachine"), name, fmt.Errorf(vmNotRunning)), response)
			return
		}
		if bodyStruct.GracePeriod == nil || (vmi.Spec.TerminationGracePeriodSeconds != nil && *bodyStruct.GracePeriod >= oldGracePeriodSeconds) {
			writeError(errors.NewConflict(v1.Resource("virtualmachine"), name, fmt.Errorf("%v only supports manual stop requests with a shorter graceperiod", v1.RunStrategyHalted)), response)
			return
		}
		// same behavior as RunStrategyManual
		patchErr, err = app.patchVMStatusStopped(vmi, vm, response, bodyStruct)
		if err != nil {
			return
		}
	case v1.RunStrategyRerunOnFailure, v1.RunStrategyManual:
		if !hasVMI || vmi.IsFinal() {
			writeError(errors.NewConflict(v1.Resource("virtualmachine"), name, fmt.Errorf(vmNotRunning)), response)
			return
		}
		// pass the buck and ask virt-controller to stop the VM. this way the
		// VM will retain RunStrategy = manual
		patchErr, err = app.patchVMStatusStopped(vmi, vm, response, bodyStruct)
		if err != nil {
			return
		}
	case v1.RunStrategyAlways, v1.RunStrategyOnce:
		patchBytes, err := getRunningPatch(vm, false)
		if err != nil {
			writeError(errors.NewInternalError(err), response)
			return
		}
		log.Log.Object(vm).V(4).Infof(patchingVMFmt, string(patchBytes))
		_, patchErr = app.virtCli.VirtualMachine(namespace).Patch(context.Background(), vm.GetName(), types.JSONPatchType, patchBytes, metav1.PatchOptions{DryRun: bodyStruct.DryRun})
	}

	if patchErr != nil {
		if strings.Contains(patchErr.Error(), jsonpatchTestErr) {
			writeError(errors.NewConflict(v1.Resource("virtualmachine"), name, patchErr), response)
		} else {
			writeError(errors.NewInternalError(patchErr), response)
		}
		return
	}

	response.WriteHeader(http.StatusAccepted)
}

func (app *SubresourceAPIApp) PauseVMIRequestHandler(request *restful.Request, response *restful.Response) {

	validate := func(vmi *v1.VirtualMachineInstance) *errors.StatusError {
		if vmi.Status.Phase != v1.Running {
			return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf(vmNotRunning))
		}
		if vmi.Spec.LivenessProbe != nil {
			return errors.NewForbidden(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf("Pausing VMIs with LivenessProbe is currently not supported"))
		}
		condManager := controller.NewVirtualMachineInstanceConditionManager()
		if condManager.HasCondition(vmi, v1.VirtualMachineInstancePaused) {
			return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf("VMI is already paused"))
		}
		return nil
	}

	getURL := func(vmi *v1.VirtualMachineInstance, conn kubecli.VirtHandlerConn) (string, error) {
		return conn.PauseURI(vmi)
	}

	bodyStruct := &v1.PauseOptions{}
	if request.Request.Body != nil {
		if err := decodeBody(request, bodyStruct); err != nil {
			writeError(err, response)
			return
		}
	}
	var dryRun bool
	if len(bodyStruct.DryRun) > 0 && bodyStruct.DryRun[0] == metav1.DryRunAll {
		dryRun = true
	}
	app.putRequestHandler(request, response, validate, getURL, dryRun)

}

func (app *SubresourceAPIApp) UnpauseVMIRequestHandler(request *restful.Request, response *restful.Response) {

	validate := func(vmi *v1.VirtualMachineInstance) *errors.StatusError {
		if vmi.Status.Phase != v1.Running {
			return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf(vmiNotRunning))
		}
		condManager := controller.NewVirtualMachineInstanceConditionManager()
		if !condManager.HasCondition(vmi, v1.VirtualMachineInstancePaused) {
			return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf(vmiNotPaused))
		}
		return nil
	}
	getURL := func(vmi *v1.VirtualMachineInstance, conn kubecli.VirtHandlerConn) (string, error) {
		return conn.UnpauseURI(vmi)
	}

	bodyStruct := &v1.UnpauseOptions{}
	if request.Request.Body != nil {
		if err := decodeBody(request, bodyStruct); err != nil {
			writeError(err, response)
			return
		}
	}
	var dryRun bool
	if len(bodyStruct.DryRun) > 0 && bodyStruct.DryRun[0] == metav1.DryRunAll {
		dryRun = true
	}
	app.putRequestHandler(request, response, validate, getURL, dryRun)

}

func (app *SubresourceAPIApp) FreezeVMIRequestHandler(request *restful.Request, response *restful.Response) {

	validate := func(vmi *v1.VirtualMachineInstance) *errors.StatusError {
		if vmi.Status.Phase != v1.Running {
			return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf(vmNotRunning))
		}
		return nil
	}

	getURL := func(vmi *v1.VirtualMachineInstance, conn kubecli.VirtHandlerConn) (string, error) {
		return conn.FreezeURI(vmi)
	}

	app.putRequestHandler(request, response, validate, getURL, false)
}

func (app *SubresourceAPIApp) UnfreezeVMIRequestHandler(request *restful.Request, response *restful.Response) {

	validate := func(vmi *v1.VirtualMachineInstance) *errors.StatusError {
		if vmi.Status.Phase != v1.Running {
			return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf(vmiNotRunning))
		}
		return nil
	}
	getURL := func(vmi *v1.VirtualMachineInstance, conn kubecli.VirtHandlerConn) (string, error) {
		return conn.UnfreezeURI(vmi)
	}
	app.putRequestHandler(request, response, validate, getURL, false)

}

func (app *SubresourceAPIApp) ResetVMIRequestHandler(request *restful.Request, response *restful.Response) {

	// Post process any error responses in order to append human
	// readable explanation for why the reset may have failed.
	errorPostProcessing := func(vmi *v1.VirtualMachineInstance, err error) error {
		// VMI reset request could have been sent while VMI was in the process of transitioning
		// from scheduled to running.
		if vmi != nil && !vmi.IsRunning() {
			return fmt.Errorf("Failed to reset non-running VMI with phase %s: %v", vmi.Status.Phase, err)
		}
		return err
	}

	getURL := func(vmi *v1.VirtualMachineInstance, conn kubecli.VirtHandlerConn) (string, error) {
		return conn.ResetURI(vmi)
	}

	app.putRequestHandlerWithErrorPostProcessing(request, response, nil, errorPostProcessing, getURL, false)
}

func (app *SubresourceAPIApp) RestartVMRequestHandler(request *restful.Request, response *restful.Response) {
	// RunStrategyHalted         -> doesn't make sense
	// RunStrategyManual         -> send restart request
	// RunStrategyAlways         -> send restart request
	// RunStrategyRerunOnFailure -> send restart request
	// RunStrategyOnce           -> doesn't make sense
	name := request.PathParameter("name")
	namespace := request.PathParameter("namespace")

	bodyStruct := &v1.RestartOptions{}

	if request.Request.Body != nil {
		if err := decodeBody(request, bodyStruct); err != nil {
			writeError(err, response)
			return
		}
	}
	if bodyStruct.GracePeriodSeconds != nil {
		if *bodyStruct.GracePeriodSeconds > 0 {
			writeError(errors.NewBadRequest(fmt.Sprintf("For force restart, only gracePeriod=0 is supported for now")), response)
			return
		} else if *bodyStruct.GracePeriodSeconds < 0 {
			writeError(errors.NewBadRequest(fmt.Sprintf("gracePeriod has to be greater or equal to 0")), response)
			return
		}
	}

	vm, statusErr := app.fetchVirtualMachine(name, namespace)
	if statusErr != nil {
		writeError(statusErr, response)
		return
	}
	if controller.NewVirtualMachineConditionManager().HasConditionWithStatus(vm,
		v1.VirtualMachineConditionType(v1.VirtualMachineInstanceVolumesChange), k8sv1.ConditionTrue) {
		writeError(errors.NewConflict(v1.Resource("virtualmachine"), name, fmt.Errorf(volumeMigrationManualRecoveryRequiredErr)), response)
		return
	}

	runStrategy, err := vm.RunStrategy()
	if err != nil {
		writeError(errors.NewInternalError(err), response)
		return
	}
	if runStrategy == v1.RunStrategyHalted || runStrategy == v1.RunStrategyOnce {
		writeError(errors.NewConflict(v1.Resource("virtualmachine"), name, fmt.Errorf("RunStategy %v does not support manual restart requests", runStrategy)), response)
		return
	}

	vmi, err := app.virtCli.VirtualMachineInstance(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			writeError(errors.NewInternalError(err), response)
			return
		}
		writeError(errors.NewConflict(v1.Resource("virtualmachine"), name, fmt.Errorf("VM is not running: %v", v1.RunStrategyHalted)), response)
		return
	}

	patchBytes, err := getChangeRequestJson(vm,
		v1.VirtualMachineStateChangeRequest{Action: v1.StopRequest, UID: &vmi.UID},
		v1.VirtualMachineStateChangeRequest{Action: v1.StartRequest})
	if err != nil {
		writeError(errors.NewInternalError(err), response)
		return
	}

	log.Log.Object(vm).V(4).Infof(patchingVMFmt, string(patchBytes))
	_, err = app.virtCli.VirtualMachine(vm.Namespace).PatchStatus(context.Background(), vm.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{DryRun: bodyStruct.DryRun})
	if err != nil {
		if strings.Contains(err.Error(), jsonpatchTestErr) {
			writeError(errors.NewConflict(v1.Resource("virtualmachine"), name, err), response)
		} else {
			writeError(errors.NewInternalError(err), response)
		}
		return
	}

	// Only force restart with GracePeriodSeconds=0 is supported for now
	// Here we are deleting the Pod because CRDs don't support gracePeriodSeconds at the moment
	if bodyStruct.GracePeriodSeconds != nil {
		if *bodyStruct.GracePeriodSeconds == 0 {
			vmiPodname, err := app.findPod(namespace, vmi)
			if err != nil {
				writeError(errors.NewInternalError(err), response)
				return
			}
			if vmiPodname == "" {
				response.WriteHeader(http.StatusAccepted)
				return
			}
			// set terminationGracePeriod to 1 (which is the shorted safe restart period) and delete the VMI pod to trigger a swift restart.
			err = app.virtCli.CoreV1().Pods(namespace).Delete(context.Background(), vmiPodname, metav1.DeleteOptions{GracePeriodSeconds: pointer.P(int64(1))})
			if err != nil {
				if !errors.IsNotFound(err) {
					writeError(errors.NewInternalError(err), response)
					return
				}
			}
		}
	}

	response.WriteHeader(http.StatusAccepted)
}

func (app *SubresourceAPIApp) SoftRebootVMIRequestHandler(request *restful.Request, response *restful.Response) {

	validate := func(vmi *v1.VirtualMachineInstance) *errors.StatusError {
		if vmi.Status.Phase != v1.Running {
			return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf(vmNotRunning))
		}
		condManager := controller.NewVirtualMachineInstanceConditionManager()
		if condManager.HasConditionWithStatus(vmi, v1.VirtualMachineInstancePaused, k8sv1.ConditionTrue) {
			return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf("VMI is paused"))
		}
		if !condManager.HasCondition(vmi, v1.VirtualMachineInstanceAgentConnected) {
			if features := vmi.Spec.Domain.Features; features != nil && features.ACPI.Enabled != nil && !(*features.ACPI.Enabled) {
				return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf("VMI neither have the agent connected nor the ACPI feature enabled"))
			}
		}
		return nil
	}

	getURL := func(vmi *v1.VirtualMachineInstance, conn kubecli.VirtHandlerConn) (string, error) {
		return conn.SoftRebootURI(vmi)
	}

	app.putRequestHandler(request, response, validate, getURL, false)
}

func (app *SubresourceAPIApp) MigrateVMRequestHandler(request *restful.Request, response *restful.Response) {
	name := request.PathParameter("name")
	namespace := request.PathParameter("namespace")

	bodyStruct := &v1.MigrateOptions{}
	if request.Request.Body != nil {
		if err := decodeBody(request, bodyStruct); err != nil {
			writeError(err, response)
			return
		}
	}
	_, err := app.fetchVirtualMachine(name, namespace)
	if err != nil {
		writeError(err, response)
		return
	}

	vmi, err := app.FetchVirtualMachineInstance(namespace, name)
	if err != nil {
		writeError(err, response)
		return
	}

	if vmi.Status.Phase != v1.Running {
		writeError(errors.NewConflict(v1.Resource("virtualmachine"), name, fmt.Errorf(vmNotRunning)), response)
		return
	}

	createMigrationJob := func() *errors.StatusError {
		_, err := app.virtCli.VirtualMachineInstanceMigration(namespace).Create(context.Background(), &v1.VirtualMachineInstanceMigration{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "kubevirt-migrate-vm-",
			},
			Spec: v1.VirtualMachineInstanceMigrationSpec{
				VMIName:           name,
				AddedNodeSelector: bodyStruct.AddedNodeSelector,
			},
		}, metav1.CreateOptions{DryRun: bodyStruct.DryRun})
		if err != nil {
			return errors.NewInternalError(err)
		}
		return nil
	}

	if err = createMigrationJob(); err != nil {
		writeError(err, response)
		return
	}

	response.WriteHeader(http.StatusAccepted)
}

func (app *SubresourceAPIApp) findPod(namespace string, vmi *v1.VirtualMachineInstance) (string, error) {
	fieldSelector := fields.ParseSelectorOrDie("status.phase==" + string(k8sv1.PodRunning))
	labelSelector, err := labels.Parse(fmt.Sprintf(v1.AppLabel + "=virt-launcher," + v1.CreatedByLabel + "=" + string(vmi.UID)))
	if err != nil {
		return "", err
	}
	selector := metav1.ListOptions{FieldSelector: fieldSelector.String(), LabelSelector: labelSelector.String()}
	podList, err := app.virtCli.CoreV1().Pods(namespace).List(context.Background(), selector)
	if err != nil {
		return "", err
	}
	if len(podList.Items) == 0 {
		return "", nil
	} else if len(podList.Items) == 1 {
		return podList.Items[0].ObjectMeta.Name, nil
	} else {
		// If we have 2 running pods, we might have a migration. Find the new pod!
		if vmi.Status.MigrationState != nil && vmi.Status.MigrationState.Completed {
			for _, pod := range podList.Items {
				if pod.Name == vmi.Status.MigrationState.TargetPod {
					return pod.Name, nil
				}
			}
		} else {
			// fallback to old behaviour
			return podList.Items[0].ObjectMeta.Name, nil
		}
	}
	return "", nil
}

func (app *SubresourceAPIApp) patchVMStatusStopped(vmi *v1.VirtualMachineInstance, vm *v1.VirtualMachine, response *restful.Response, bodyStruct *v1.StopOptions) (error, error) {
	patchBytes, err := getChangeRequestJson(vm,
		v1.VirtualMachineStateChangeRequest{Action: v1.StopRequest, UID: &vmi.UID})
	if err != nil {
		writeError(errors.NewInternalError(err), response)
		return nil, err
	}
	log.Log.Object(vm).V(4).Infof(patchingVMStatusFmt, string(patchBytes))
	_, err = app.virtCli.VirtualMachine(vm.Namespace).PatchStatus(context.Background(), vm.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{DryRun: bodyStruct.DryRun})
	return err, nil
}

func getChangeRequestJson(vm *v1.VirtualMachine, changes ...v1.VirtualMachineStateChangeRequest) ([]byte, error) {
	patchSet := patch.New()
	// Special case: if there's no status field at all, add one.
	newStatus := v1.VirtualMachineStatus{}
	if equality.Semantic.DeepEqual(vm.Status, newStatus) {
		newStatus.StateChangeRequests = append(newStatus.StateChangeRequests, changes...)
		patchSet.AddOption(patch.WithAdd("/status", newStatus))
	} else {
		patchSet.AddOption(patch.WithTest("/status/stateChangeRequests", vm.Status.StateChangeRequests))
		switch {
		case len(vm.Status.StateChangeRequests) == 0:
			patchSet.AddOption(patch.WithAdd("/status/stateChangeRequests", changes))
		case len(changes) == 1 && changes[0].Action == v1.StopRequest:
			// If this is a stopRequest, replace all existing StateChangeRequests.
			patchSet.AddOption(patch.WithReplace("/status/stateChangeRequests", changes))
		default:
			return nil, fmt.Errorf("unable to complete request: stop/start already underway")
		}
	}

	if vm.Status.StartFailure != nil {
		patchSet.AddOption(patch.WithRemove("/status/startFailure"))
	}

	return patchSet.GeneratePayload()
}

func getRunningPatch(vm *v1.VirtualMachine, running bool) ([]byte, error) {
	runStrategy := v1.RunStrategyHalted
	if running {
		runStrategy = v1.RunStrategyAlways
	}

	if vm.Spec.RunStrategy != nil {
		return patch.New(
			patch.WithTest("/spec/runStrategy", vm.Spec.RunStrategy),
			patch.WithReplace("/spec/runStrategy", runStrategy),
		).GeneratePayload()
	}

	return patch.New(
		patch.WithTest("/spec/running", vm.Spec.Running),
		patch.WithReplace("/spec/running", running),
	).GeneratePayload()
}
