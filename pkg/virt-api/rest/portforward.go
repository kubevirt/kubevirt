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

	"kubevirt.io/kubevirt/pkg/controller"
	apimetrics "kubevirt.io/kubevirt/pkg/monitoring/metrics/virt-api"
)

func (app *SubresourceAPIApp) PortForwardRequestHandler(fetcher vmiFetcher) restful.RouteFunction {
	return func(request *restful.Request, response *restful.Response) {
		activeTunnelMetric := apimetrics.NewActivePortForwardTunnel(request.PathParameter("namespace"), request.PathParameter("name"))
		defer activeTunnelMetric.Dec()

		defer apimetrics.SetVMILastConnectionTimestamp(request.PathParameter("namespace"), request.PathParameter("name"))

		streamer := NewWebsocketStreamer(
			fetcher,
			validateVMIForPortForward,
			netDial{request: request},
		)

		streamer.Handle(request, response)
	}
}

func validateVMIForPortForward(vmi *v1.VirtualMachineInstance) *errors.StatusError {
	condManager := controller.NewVirtualMachineInstanceConditionManager()
	if condManager.HasCondition(vmi, v1.VirtualMachineInstancePaused) {
		return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf("VMI is paused"))
	}
	return nil
}
