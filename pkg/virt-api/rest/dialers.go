package rest

import (
	"fmt"
	"net"

	"github.com/gorilla/websocket"

	restful "github.com/emicklei/go-restful/v3"
	"k8s.io/apimachinery/pkg/api/errors"
	netutils "k8s.io/utils/net"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/virt-api/definitions"
)

type netDial struct {
	request *restful.Request
}

type handlerDial struct {
	getURL URLResolver
	app    *SubresourceAPIApp
}

func (h handlerDial) Dial(vmi *v1.VirtualMachineInstance) (*websocket.Conn, *errors.StatusError) {
	url, _, statusError := h.app.getVirtHandlerFor(vmi, h.getURL)
	if statusError != nil {
		return nil, statusError
	}
	conn, _, err := kubecli.Dial(url, h.app.handlerTLSConfiguration)
	if err != nil {
		return nil, errors.NewInternalError(fmt.Errorf("dialing virt-handler: %w", err))
	}
	return conn, nil
}

func (h handlerDial) DialUnderlying(vmi *v1.VirtualMachineInstance) (net.Conn, *errors.StatusError) {
	conn, err := h.Dial(vmi)
	if err != nil {
		return nil, err
	}
	return conn.UnderlyingConn(), nil
}

func (n netDial) Dial(vmi *v1.VirtualMachineInstance) (*websocket.Conn, *errors.StatusError) {
	panic("don't call me")
}

func (n netDial) DialUnderlying(vmi *v1.VirtualMachineInstance) (net.Conn, *errors.StatusError) {
	logger := log.Log.Object(vmi)

	targetIP, err := getTargetInterfaceIP(vmi)
	if err != nil {
		logger.Reason(err).Error("Can't establish TCP tunnel.")
		return nil, errors.NewBadRequest(err.Error())
	}

	port := n.request.PathParameter(definitions.PortParamName)
	if len(port) < 1 {
		return nil, errors.NewBadRequest("port must not be empty")
	}

	protocol := "tcp"
	if protocolParam := n.request.PathParameter(definitions.ProtocolParamName); len(protocolParam) > 0 {
		protocol = protocolParam
	}

	addr := fmt.Sprintf("%s:%s", targetIP, port)
	if netutils.IsIPv6String(targetIP) {
		addr = fmt.Sprintf("[%s]:%s", targetIP, port)
	}
	conn, err := net.Dial(protocol, addr)
	if err != nil {
		logger.Reason(err).Errorf("Can't dial %s %s", protocol, addr)
		return nil, errors.NewInternalError(fmt.Errorf("dialing VM: %w", err))
	}
	return conn, nil
}

func (app *SubresourceAPIApp) virtHandlerDialer(getURL URLResolver) dialer {
	return handlerDial{
		getURL: getURL,
		app:    app,
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
