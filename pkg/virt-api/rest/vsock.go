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
 */

package rest

import (
	"fmt"

	restful "github.com/emicklei/go-restful/v3"
	"k8s.io/apimachinery/pkg/api/errors"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/util"
)

func (app *SubresourceAPIApp) VSOCKRequestHandler(request *restful.Request, response *restful.Response) {
	streamer := NewRawStreamer(
		app.FetchVirtualMachineInstance,
		validateVMIForVSOCK,
		app.virtHandlerDialer(func(vmi *v1.VirtualMachineInstance, conn kubecli.VirtHandlerConn) (string, error) {
			tls := "true"
			if request.QueryParameter("tls") != "" {
				tls = request.QueryParameter("tls")
			}
			return conn.VSOCKURI(vmi, request.QueryParameter("port"), tls)
		}),
	)

	streamer.Handle(request, response)
}

func validateVMIForVSOCK(vmi *v1.VirtualMachineInstance) *errors.StatusError {
	if !util.IsAutoAttachVSOCK(vmi) {
		err := fmt.Errorf("VSOCK is not attached.")
		log.Log.Object(vmi).Reason(err).Error("Can't establish Vsock connection.")
		return errors.NewBadRequest(err.Error())
	}
	return nil
}
