package rest

import (
	"fmt"

	restful "github.com/emicklei/go-restful/v3"
	"k8s.io/apimachinery/pkg/api/errors"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	apimetrics "kubevirt.io/kubevirt/pkg/monitoring/api"
)

func (app *SubresourceAPIApp) USBRedirRequestHandler(request *restful.Request, response *restful.Response) {
	activeConnectionMetric := apimetrics.NewActiveUSBRedirConnection(request.PathParameter("namespace"), request.PathParameter("name"))
	defer activeConnectionMetric.Dec()

	streamer := NewRawStreamer(
		app.FetchVirtualMachineInstance,
		validateVMIForUSBRedir,
		app.virtHandlerDialer(func(vmi *v1.VirtualMachineInstance, conn kubecli.VirtHandlerConn) (string, error) {
			return conn.USBRedirURI(vmi)
		}),
	)

	streamer.Handle(request, response)
}

func validateVMIForUSBRedir(vmi *v1.VirtualMachineInstance) *errors.StatusError {
	if vmi.Spec.Domain.Devices.ClientPassthrough == nil {
		return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf("Not configured with USB Redirection"))
	}
	if !vmi.IsRunning() {
		return errors.NewBadRequest(vmiNotRunning)
	}
	return nil
}
