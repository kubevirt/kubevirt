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

package vnc

// Invalidate stale bazel remote cache entries after CI CacheNotFoundException.

import (
	"context"
	"fmt"
	"net/http"

	"k8s.io/apimachinery/pkg/api/errors"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	apimetrics "kubevirt.io/kubevirt/pkg/monitoring/metrics/virt-api"
	"kubevirt.io/kubevirt/pkg/virt-api/streaming"
)

// Handler serves the VirtualMachineInstance VNC streaming and screenshot subresources.
type Handler struct {
	streamer *streaming.Streamer
}

func NewHandler(streamer *streaming.Streamer) *Handler {
	return &Handler{streamer: streamer}
}

// StreamVNC proxies the VNC display of the named VMI as a raw, bidirectional
// websocket stream to virt-handler
func (h *Handler) StreamVNC(ctx context.Context, namespace, name string, preserveSession bool, w http.ResponseWriter, req *http.Request) *errors.StatusError {
	activeConnectionMetric := apimetrics.NewActiveVNCConnection(namespace, name)
	defer activeConnectionMetric.Dec()
	defer apimetrics.SetVMILastConnectionTimestamp(namespace, name)

	return h.streamer.StreamRaw(ctx, namespace, name, w, req, validateVMIForVNC,
		func(vmi *v1.VirtualMachineInstance, conn kubecli.VirtHandlerConn) (string, error) {
			return conn.VNCURI(vmi, preserveSession)
		},
	)
}

// Screenshot fetches a VNC screenshot for the named VMI from virt-handler.
func (h *Handler) Screenshot(ctx context.Context, namespace, name string) ([]byte, *errors.StatusError) {
	vmi, statusErr := h.streamer.FetchAndValidateVMI(ctx, namespace, name, validateVMIForVNC)
	if statusErr != nil {
		return nil, statusErr
	}

	conn := h.streamer.VirtHandlerConnFor(vmi)
	url, err := conn.ScreenshotURI(vmi)
	if err != nil {
		return nil, errors.NewBadRequest(err.Error())
	}

	response, err := conn.Get(url, "")
	if err != nil {
		log.Log.Reason(err).Error("Failed to get VNC screenshot from virt-handler")
		return nil, errors.NewInternalError(err)
	}
	return []byte(response), nil
}

func validateVMIForVNC(vmi *v1.VirtualMachineInstance) *errors.StatusError {
	// can't proceed if there are no graphics devices present
	if vmi.Spec.Domain.Devices.AutoattachGraphicsDevice != nil && !*vmi.Spec.Domain.Devices.AutoattachGraphicsDevice {
		err := fmt.Errorf("No graphics devices are present.")
		log.Log.Object(vmi).Reason(err).Error("Can't establish VNC connection.")
		return errors.NewBadRequest(err.Error())
	}
	if !vmi.IsRunning() {
		return errors.NewBadRequest("VMI is not running")
	}
	return nil
}
