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
	"io"

	"github.com/emicklei/go-restful/v3"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/controller"
)

// PauseVMI pauses a running VMI. The optional PauseOptions body is decoded to
// detect a dry-run request before the body is proxied to virt-handler
func (app *SubresourceAPIApp) PauseVMI(ctx context.Context, namespace, name string, body io.ReadCloser) *errors.StatusError {
	validate := func(vmi *v1.VirtualMachineInstance) *errors.StatusError {
		if vmi.Status.Phase != v1.Running {
			return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf(vmNotRunning))
		}
		if vmi.Spec.LivenessProbe != nil && vmi.Spec.LivenessProbe.GuestAgentPing == nil {
			return errors.NewForbidden(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf("Pausing VMIs with a non-GuestAgentPing LivenessProbe is not supported"))
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
	if body != nil {
		if err := decodeBodyReader(body, bodyStruct); err != nil {
			return err
		}
	}
	var dryRun bool
	if len(bodyStruct.DryRun) > 0 && bodyStruct.DryRun[0] == metav1.DryRunAll {
		dryRun = true
	}
	return app.connectVirtHandler(ctx, namespace, name, body, validate, nil, getURL, dryRun)
}

func (app *SubresourceAPIApp) PauseVMIRequestHandler(request *restful.Request, response *restful.Response) {
	name := request.PathParameter("name")
	namespace := request.PathParameter("namespace")
	if statusErr := app.PauseVMI(request.Request.Context(), namespace, name, request.Request.Body); statusErr != nil {
		writeError(statusErr, response)
	}
}

// UnpauseVMI resumes a paused VMI. It first ensures the owning VM (if any) has
// no snapshot in progress then proxies the request to virt-handler
func (app *SubresourceAPIApp) UnpauseVMI(ctx context.Context, namespace, name string, body io.ReadCloser) *errors.StatusError {
	// Check VM status - only continue if VM doesn't exist or if it exists without snapshot in progress
	vm, statusErr := app.fetchVirtualMachine(name, namespace)
	if statusErr != nil {
		if !errors.IsNotFound(statusErr) {
			return statusErr
		}
	} else if vm.Status.SnapshotInProgress != nil {
		// VM exists - check if snapshot is in progress
		return errors.NewConflict(v1.Resource("virtualmachine"), name, fmt.Errorf(vmSnapshotInprogress))
	}

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
	if body != nil {
		if err := decodeBodyReader(body, bodyStruct); err != nil {
			return err
		}
	}
	var dryRun bool
	if len(bodyStruct.DryRun) > 0 && bodyStruct.DryRun[0] == metav1.DryRunAll {
		dryRun = true
	}
	return app.connectVirtHandler(ctx, namespace, name, body, validate, nil, getURL, dryRun)
}

func (app *SubresourceAPIApp) UnpauseVMIRequestHandler(request *restful.Request, response *restful.Response) {
	name := request.PathParameter("name")
	namespace := request.PathParameter("namespace")
	if statusErr := app.UnpauseVMI(request.Request.Context(), namespace, name, request.Request.Body); statusErr != nil {
		writeError(statusErr, response)
	}
}

// FreezeVMI freezes the filesystems of a running VMI
// The request body is proxied unchanged to virt-handler
func (app *SubresourceAPIApp) FreezeVMI(ctx context.Context, namespace, name string, body io.ReadCloser) *errors.StatusError {
	validate := func(vmi *v1.VirtualMachineInstance) *errors.StatusError {
		if vmi.Status.Phase != v1.Running {
			return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf(vmNotRunning))
		}
		return nil
	}

	getURL := func(vmi *v1.VirtualMachineInstance, conn kubecli.VirtHandlerConn) (string, error) {
		return conn.FreezeURI(vmi)
	}

	return app.connectVirtHandler(ctx, namespace, name, body, validate, nil, getURL, false)
}

func (app *SubresourceAPIApp) FreezeVMIRequestHandler(request *restful.Request, response *restful.Response) {
	name := request.PathParameter("name")
	namespace := request.PathParameter("namespace")
	if statusErr := app.FreezeVMI(request.Request.Context(), namespace, name, request.Request.Body); statusErr != nil {
		writeError(statusErr, response)
	}
}

// UnfreezeVMI thaws the filesystems of a running VMI.
func (app *SubresourceAPIApp) UnfreezeVMI(ctx context.Context, namespace, name string, body io.ReadCloser) *errors.StatusError {
	validate := func(vmi *v1.VirtualMachineInstance) *errors.StatusError {
		if vmi.Status.Phase != v1.Running {
			return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf(vmiNotRunning))
		}
		return nil
	}
	getURL := func(vmi *v1.VirtualMachineInstance, conn kubecli.VirtHandlerConn) (string, error) {
		return conn.UnfreezeURI(vmi)
	}
	return app.connectVirtHandler(ctx, namespace, name, body, validate, nil, getURL, false)
}

func (app *SubresourceAPIApp) UnfreezeVMIRequestHandler(request *restful.Request, response *restful.Response) {
	name := request.PathParameter("name")
	namespace := request.PathParameter("namespace")
	if statusErr := app.UnfreezeVMI(request.Request.Context(), namespace, name, request.Request.Body); statusErr != nil {
		writeError(statusErr, response)
	}
}

// ResetVMI triggers a reset of a running VMI.
func (app *SubresourceAPIApp) ResetVMI(ctx context.Context, namespace, name string, body io.ReadCloser) *errors.StatusError {
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

	return app.connectVirtHandler(ctx, namespace, name, body, nil, errorPostProcessing, getURL, false)
}

func (app *SubresourceAPIApp) ResetVMIRequestHandler(request *restful.Request, response *restful.Response) {
	name := request.PathParameter("name")
	namespace := request.PathParameter("namespace")
	if statusErr := app.ResetVMI(request.Request.Context(), namespace, name, request.Request.Body); statusErr != nil {
		writeError(statusErr, response)
	}
}

// SoftRebootVMI issues an ACPI soft reboot of a running VMI
func (app *SubresourceAPIApp) SoftRebootVMI(ctx context.Context, namespace, name string, body io.ReadCloser) *errors.StatusError {
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

	return app.connectVirtHandler(ctx, namespace, name, body, validate, nil, getURL, false)
}

func (app *SubresourceAPIApp) SoftRebootVMIRequestHandler(request *restful.Request, response *restful.Response) {
	name := request.PathParameter("name")
	namespace := request.PathParameter("namespace")
	if statusErr := app.SoftRebootVMI(request.Request.Context(), namespace, name, request.Request.Body); statusErr != nil {
		writeError(statusErr, response)
	}
}

func (app *SubresourceAPIApp) BackupVMI(ctx context.Context, namespace, name string, body io.ReadCloser) *errors.StatusError {
	validate := func(vmi *v1.VirtualMachineInstance) *errors.StatusError {
		if vmi.Status.Phase != v1.Running {
			return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf(vmNotRunning))
		}
		return nil
	}

	getURL := func(vmi *v1.VirtualMachineInstance, conn kubecli.VirtHandlerConn) (string, error) {
		return conn.BackupURI(vmi)
	}

	return app.connectVirtHandler(ctx, namespace, name, body, validate, nil, getURL, false)
}

func (app *SubresourceAPIApp) RedefineCheckpointVMI(ctx context.Context, namespace, name string, body io.ReadCloser) *errors.StatusError {
	validate := func(vmi *v1.VirtualMachineInstance) *errors.StatusError {
		if vmi.Status.ChangedBlockTracking == nil ||
			vmi.Status.ChangedBlockTracking.State != v1.ChangedBlockTrackingEnabled {
			return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name,
				fmt.Errorf("ChangedBlockTracking is not enabled"))
		}
		return nil
	}

	getURL := func(vmi *v1.VirtualMachineInstance, conn kubecli.VirtHandlerConn) (string, error) {
		return conn.RedefineCheckpointURI(vmi)
	}

	return app.connectVirtHandler(ctx, namespace, name, body, validate, nil, getURL, false)
}
