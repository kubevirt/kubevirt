package rest

import (
	"fmt"

	restful "github.com/emicklei/go-restful/v3"
	"k8s.io/apimachinery/pkg/api/errors"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/controller"
	apimetrics "kubevirt.io/kubevirt/pkg/monitoring/api"
)

func (app *SubresourceAPIApp) PortForwardRequestHandler(fetcher vmiFetcher) restful.RouteFunction {
	return func(request *restful.Request, response *restful.Response) {
		activeTunnelMetric := apimetrics.NewActivePortForwardTunnel(request.PathParameter("namespace"), request.PathParameter("name"))
		defer activeTunnelMetric.Dec()

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
