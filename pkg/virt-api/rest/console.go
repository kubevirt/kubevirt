package rest

import (
	"fmt"

	restful "github.com/emicklei/go-restful/v3"
	"k8s.io/apimachinery/pkg/api/errors"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	apimetrics "kubevirt.io/kubevirt/pkg/monitoring/api"
)

func (app *SubresourceAPIApp) ConsoleRequestHandler(request *restful.Request, response *restful.Response) {
	activeConnectionMetric := apimetrics.NewActiveConsoleConnection(request.PathParameter("namespace"), request.PathParameter("name"))
	defer activeConnectionMetric.Dec()

	streamer := NewRawStreamer(
		app.FetchVirtualMachineInstance,
		validateVMIForConsole,
		app.virtHandlerDialer(func(vmi *v1.VirtualMachineInstance, conn kubecli.VirtHandlerConn) (string, error) {
			return conn.ConsoleURI(vmi)
		}),
	)

	streamer.Handle(request, response)
}

func validateVMIForConsole(vmi *v1.VirtualMachineInstance) *errors.StatusError {
	if vmi.Spec.Domain.Devices.AutoattachSerialConsole != nil && *vmi.Spec.Domain.Devices.AutoattachSerialConsole == false {
		err := fmt.Errorf("No serial consoles are present.")
		log.Log.Object(vmi).Reason(err).Error("Can't establish a serial console connection.")
		return errors.NewBadRequest(err.Error())
	}
	if vmi.Status.Phase == v1.Failed {
		return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf("VMI is in failed status"))
	}
	if !vmi.IsRunning() {
		return errors.NewBadRequest(vmiNotRunning)
	}
	return nil
}
