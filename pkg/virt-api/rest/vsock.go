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
