package rest

import (
	"fmt"

	restful "github.com/emicklei/go-restful/v3"
	"k8s.io/apimachinery/pkg/api/errors"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	apimetrics "kubevirt.io/kubevirt/pkg/monitoring/metrics/virt-api"
)

func (app *SubresourceAPIApp) VNCRequestHandler(request *restful.Request, response *restful.Response) {
	activeConnectionMetric := apimetrics.NewActiveVNCConnection(request.PathParameter("namespace"), request.PathParameter("name"))
	defer activeConnectionMetric.Dec()

	defer apimetrics.SetVMILastConnectionTimestamp(request.PathParameter("namespace"), request.PathParameter("name"))

	streamer := NewRawStreamer(
		app.FetchVirtualMachineInstance,
		vmiHasDisplay,
		app.virtHandlerDialer(func(vmi *v1.VirtualMachineInstance, conn kubecli.VirtHandlerConn) (string, error) {
			return conn.VNCURI(vmi)
		}),
	)

	streamer.Handle(request, response)
}

func (app *SubresourceAPIApp) ScreenshotRequestHandler(request *restful.Request, response *restful.Response) {
	getURL := func(vmi *v1.VirtualMachineInstance, conn kubecli.VirtHandlerConn) (string, error) {
		return conn.ScreenshotURI(vmi)
	}

	// Screenshot without Display fails with:
	//   `Requested operation is not valid: no screens to take screenshot from`
	app.httpGetRequestHandler(request, response, vmiHasDisplay, getURL, nil)
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
