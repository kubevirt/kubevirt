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

package vsock

import (
	"context"
	"fmt"
	"net/http"

	"k8s.io/apimachinery/pkg/api/errors"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/virt-api/streaming"
)

// Handler serves the VirtualMachineInstance VSOCK streaming subresource.
type Handler struct {
	streamer *streaming.Streamer
}

func NewHandler(streamer *streaming.Streamer) *Handler {
	return &Handler{streamer: streamer}
}

// StreamVSOCK proxies the VSOCK channel of the named VMI as a raw, bidirectional
// websocket stream to virt-handler
func (h *Handler) StreamVSOCK(ctx context.Context, namespace, name, port, tls string, w http.ResponseWriter, req *http.Request) *errors.StatusError {
	return h.streamer.StreamRaw(ctx, namespace, name, w, req, validateVMIForVSOCK,
		func(vmi *v1.VirtualMachineInstance, conn kubecli.VirtHandlerConn) (string, error) {
			return conn.VSOCKURI(vmi, port, tls)
		},
	)
}

func validateVMIForVSOCK(vmi *v1.VirtualMachineInstance) *errors.StatusError {
	if !util.IsAutoAttachVSOCK(vmi) {
		err := fmt.Errorf("VSOCK is not attached.")
		log.Log.Object(vmi).Reason(err).Error("Can't establish Vsock connection.")
		return errors.NewBadRequest(err.Error())
	}
	return nil
}
