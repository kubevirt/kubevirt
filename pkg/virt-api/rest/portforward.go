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
	"fmt"

	restful "github.com/emicklei/go-restful/v3"
	"k8s.io/apimachinery/pkg/api/errors"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/controller"
	apimetrics "kubevirt.io/kubevirt/pkg/monitoring/metrics/virt-api"
	netvmispec "kubevirt.io/kubevirt/pkg/network/vmispec"
)

func (app *SubresourceAPIApp) PortForwardRequestHandler(fetcher vmiFetcher) restful.RouteFunction {
	return func(request *restful.Request, response *restful.Response) {
		activeTunnelMetric := apimetrics.NewActivePortForwardTunnel(request.PathParameter("namespace"), request.PathParameter("name"))
		defer activeTunnelMetric.Dec()

		defer apimetrics.SetVMILastConnectionTimestamp(request.PathParameter("namespace"), request.PathParameter("name"))

		streamer := NewWebsocketStreamer(
			fetcher,
			validateVMIForPortForward,
			netDial{request: request, launcherPodIndexer: app.launcherPodIndexer},
		)

		streamer.Handle(request, response)
	}
}

func validateVMIForPortForward(vmi *v1.VirtualMachineInstance) *errors.StatusError {
	condManager := controller.NewVirtualMachineInstanceConditionManager()
	if condManager.HasCondition(vmi, v1.VirtualMachineInstancePaused) {
		return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf("VMI is paused"))
	}
	// virt-api reaches the VM over the launcher pod IP, which only exists for the
	// pod network. A VMI without a pod network is unreachable this way, so reject
	// it up front instead of failing later during target resolution.
	if netvmispec.LookupPodNetwork(vmi.Spec.Networks) == nil {
		return errors.NewBadRequest(fmt.Sprintf("VMI %s is not connected to the pod network", vmi.Name))
	}
	return nil
}
