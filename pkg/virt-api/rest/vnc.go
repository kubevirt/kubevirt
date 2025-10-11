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
	"slices"
	"strconv"
	"time"

	restful "github.com/emicklei/go-restful/v3"
	"k8s.io/apimachinery/pkg/api/errors"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	apimetrics "kubevirt.io/kubevirt/pkg/monitoring/metrics/virt-api"
	"kubevirt.io/kubevirt/pkg/virt-api/definitions"
)

func (app *SubresourceAPIApp) patchService(name, namespace string, service *v1.ServiceStatus) error {
	var bytes []byte
	var err error

	bytes, err = patch.GeneratePatchPayload(patch.PatchOperation{
		Op:    patch.PatchReplaceOp,
		Path:  "/status/serviceStatus",
		Value: service,
	},
	)
	if err != nil {
		return err
	}

	_, err = app.virtCli.VirtualMachineInstance(namespace).Patch(
		context.Background(),
		name,
		types.JSONPatchType,
		bytes,
		k8smetav1.PatchOptions{},
	)
	if err != nil {
		return err
	}

	return nil
}

func (app *SubresourceAPIApp) vncAddConnection(name, namespace string) (*v1.VNCStatusInfo, *errors.StatusError) {
	app.serviceLock.Lock()
	defer app.serviceLock.Unlock()

	vmi, statErr := app.FetchVirtualMachineInstance(namespace, name)
	if statErr != nil {
		return nil, statErr
	}

	service := vmi.Status.ServiceStatus
	if service == nil {
		service = &v1.ServiceStatus{}
	}

	conn := &v1.VNCStatusInfo{
		Since: time.Now().Format(time.RFC3339),
	}
	service.VNCStatuses = append(service.VNCStatuses, conn)

	if err := app.patchService(name, namespace, service); err != nil {
		return nil, errors.NewInternalError(fmt.Errorf("unable to patch vm status: %v", err))
	}
	return conn, nil
}

func (app *SubresourceAPIApp) vncRemoveConnection(name, namespace string, conn *v1.VNCStatusInfo) *errors.StatusError {
	app.serviceLock.Lock()
	defer app.serviceLock.Unlock()

	vmi, statErr := app.FetchVirtualMachineInstance(namespace, name)
	if statErr != nil {
		return statErr
	}

	service := vmi.Status.ServiceStatus
	if service == nil || len(service.VNCStatuses) == 0 {
		return nil
	}

	i := slices.IndexFunc(service.VNCStatuses, func(info *v1.VNCStatusInfo) bool {
		return info.Since == conn.Since
	})

	if i < 0 {
		return errors.NewInternalError(fmt.Errorf("failed to find: %s", conn.Since))
	}

	// The order is not important. We only support 1 active socket but
	// the request might be accepted or not by virt-handler.
	last := len(service.VNCStatuses) - 1
	service.VNCStatuses[i], service.VNCStatuses[last] = service.VNCStatuses[last], service.VNCStatuses[i]
	service.VNCStatuses = service.VNCStatuses[:last]

	if err := app.patchService(name, namespace, service); err != nil {
		return errors.NewInternalError(fmt.Errorf("unable to patch vm status: %v", err))
	}
	return nil
}

func (app *SubresourceAPIApp) VNCRequestHandler(request *restful.Request, response *restful.Response) {
	name, namespace := request.PathParameter("name"), request.PathParameter("namespace")
	activeConnectionMetric := apimetrics.NewActiveVNCConnection(namespace, name)
	defer activeConnectionMetric.Dec()

	defer apimetrics.SetVMILastConnectionTimestamp(request.PathParameter("namespace"), request.PathParameter("name"))

	// Default is false: drops the current VNC session if any
	preserveSessionParam := false

	// Check the request as QueryParameter assumes them to exist
	if request.Request != nil && request.Request.URL != nil {
		val, err := strconv.ParseBool(request.QueryParameter(definitions.PreserveSessionParamName))
		if err != nil {
			log.DefaultLogger().Reason(err).Warningf("Failed to parse VNC's query parameter: %s", definitions.PreserveSessionParamName)
		}
		preserveSessionParam = val
	}

	streamer := NewRawStreamer(
		app.FetchVirtualMachineInstance,
		vmiHasDisplay,
		app.virtHandlerDialer(func(vmi *v1.VirtualMachineInstance, conn kubecli.VirtHandlerConn) (string, error) {
			return conn.VNCURI(vmi, preserveSessionParam)
		}),
	)

	conn, err := app.vncAddConnection(name, namespace)
	if err != nil {
		log.Log.Reason(err).Info("Failed to patch VMI status")
	}

	streamer.Handle(request, response)

	if err := app.vncRemoveConnection(name, namespace, conn); err != nil {
		log.Log.Reason(err).Info("Failed to patch VMI status")
	}
}

func (app *SubresourceAPIApp) ScreenshotRequestHandler(request *restful.Request, response *restful.Response) {
	getURL := func(vmi *v1.VirtualMachineInstance, conn kubecli.VirtHandlerConn) (string, error) {
		return conn.ScreenshotURI(vmi)
	}

	// Screenshot without Display fails with:
	//   `Requested operation is not valid: no screens to take screenshot from`
	app.httpGetRequestBinaryHandler(request, response, vmiHasDisplay, getURL)
}

func vmiHasDisplay(vmi *v1.VirtualMachineInstance) *errors.StatusError {
	// If there are no graphics devices present, we can't proceed
	if vmi.Spec.Domain.Devices.AutoattachGraphicsDevice != nil && !*vmi.Spec.Domain.Devices.AutoattachGraphicsDevice {
		err := fmt.Errorf("No graphics devices are present.")
		log.Log.Object(vmi).Reason(err).Error("Can't establish VNC connection.")
		return errors.NewBadRequest(err.Error())
	}
	if !vmi.IsRunning() {
		return errors.NewBadRequest(vmiNotRunning)
	}
	return nil
}
