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

package usbredir

import (
	"context"
	"fmt"
	"net/http"

	"k8s.io/apimachinery/pkg/api/errors"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	apimetrics "kubevirt.io/kubevirt/pkg/monitoring/metrics/virt-api"
	"kubevirt.io/kubevirt/pkg/virt-api/streaming"
)

// Handler serves the VirtualMachineInstance USB redirection streaming subresource.
type Handler struct {
	streamer *streaming.Streamer
}

func NewHandler(streamer *streaming.Streamer) *Handler {
	return &Handler{streamer: streamer}
}

// StreamUSBRedir proxies the USB redirection channel of the named VMI as a raw,
// bidirectional websocket stream to virt-handler
func (h *Handler) StreamUSBRedir(ctx context.Context, namespace, name string, w http.ResponseWriter, req *http.Request) *errors.StatusError {
	activeConnectionMetric := apimetrics.NewActiveUSBRedirConnection(namespace, name)
	defer activeConnectionMetric.Dec()
	defer apimetrics.SetVMILastConnectionTimestamp(namespace, name)

	return h.streamer.StreamRaw(ctx, namespace, name, w, req, validateVMIForUSBRedir,
		func(vmi *v1.VirtualMachineInstance, conn kubecli.VirtHandlerConn) (string, error) {
			return conn.USBRedirURI(vmi)
		},
	)
}

func validateVMIForUSBRedir(vmi *v1.VirtualMachineInstance) *errors.StatusError {
	if vmi.Spec.Architecture == "s390x" {
		return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf("No USB support on s390x"))
	}
	if vmi.Spec.Domain.Devices.ClientPassthrough == nil {
		return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf("Not configured with USB Redirection"))
	}
	if !vmi.IsRunning() {
		return errors.NewBadRequest("VMI is not running")
	}
	return nil
}
