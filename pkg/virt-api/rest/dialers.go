package rest

import (
	"fmt"
	"net"

	restful "github.com/emicklei/go-restful"
	"k8s.io/apimachinery/pkg/api/errors"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
)

func netDialer(request *restful.Request) dialer {
	return func(vmi *v1.VirtualMachineInstance) (net.Conn, *errors.StatusError) {
		logger := log.Log.Object(vmi)

		targetIP, err := getTargetInterfaceIP(vmi)
		if err != nil {
			logger.Reason(err).Error("Can't establish TCP tunnel.")
			return nil, errors.NewBadRequest(err.Error())
		}

		port := request.PathParameter(PortParamName)
		if len(port) < 1 {
			return nil, errors.NewBadRequest("port must not be empty")
		}

		protocol := "tcp"
		if protocolParam := request.PathParameter(ProtocolParamName); len(protocolParam) > 0 {
			protocol = protocolParam
		}

		addr := fmt.Sprintf("%s:%s", targetIP, port)
		conn, err := net.Dial(protocol, addr)
		if err != nil {
			logger.Reason(err).Errorf("Can't dial %s %s", protocol, addr)
			return nil, errors.NewInternalError(fmt.Errorf("dialing VM: %w", err))
		}
		return conn, nil
	}
}

func (app *SubresourceAPIApp) virtHandlerDialer(getURL URLResolver) dialer {
	return func(vmi *v1.VirtualMachineInstance) (net.Conn, *errors.StatusError) {
		url, _, statusError := app.getVirtHandlerFor(vmi, getURL)
		if statusError != nil {
			return nil, statusError
		}
		conn, _, err := kubecli.Dial(url, app.handlerTLSConfiguration)
		if err != nil {
			return nil, errors.NewInternalError(fmt.Errorf("dialing virt-handler: %w", err))
		}
		return conn.UnderlyingConn(), nil
	}
}

func (app *SubresourceAPIApp) getVirtHandlerFor(vmi *v1.VirtualMachineInstance, getVirtHandlerURL URLResolver) (url string, conn kubecli.VirtHandlerConn, statusError *errors.StatusError) {
	var err error
	if conn, err = app.getVirtHandlerConnForVMI(vmi); err != nil {
		statusError = errors.NewBadRequest(err.Error())
		log.Log.Object(vmi).Reason(statusError).Error("Unable to establish connection to virt-handler")
		return
	}
	if url, err = getVirtHandlerURL(vmi, conn); err != nil {
		statusError = errors.NewBadRequest(err.Error())
		log.Log.Object(vmi).Reason(statusError).Error("Unable to retrieve target handler URL")
		return
	}
	return
}
