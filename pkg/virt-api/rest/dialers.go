package rest

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/gorilla/websocket"

	"github.com/emicklei/go-restful/v3"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	netutils "k8s.io/utils/net"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	kvcorev1 "kubevirt.io/client-go/kubevirt/typed/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/virt-api/definitions"
)

type netDial struct {
	request *restful.Request
	app     *SubresourceAPIApp
}

type handlerDial struct {
	getURL URLResolver
	app    *SubresourceAPIApp
}

func (h handlerDial) Dial(vmi *v1.VirtualMachineInstance) (*websocket.Conn, *k8serrors.StatusError) {
	url, _, statusError := h.app.getVirtHandlerFor(vmi, h.getURL)
	if statusError != nil {
		return nil, statusError
	}
	conn, _, err := kvcorev1.Dial(url, h.app.handlerTLSConfiguration)
	if err != nil {
		return nil, k8serrors.NewInternalError(fmt.Errorf("dialing virt-handler: %w", err))
	}
	return conn, nil
}

func (h handlerDial) DialUnderlying(vmi *v1.VirtualMachineInstance) (net.Conn, *k8serrors.StatusError) {
	conn, err := h.Dial(vmi)
	if err != nil {
		return nil, err
	}
	return conn.UnderlyingConn(), nil
}

func (n netDial) Dial(vmi *v1.VirtualMachineInstance) (*websocket.Conn, *k8serrors.StatusError) {
	panic("don't call me")
}

func (n netDial) DialUnderlying(vmi *v1.VirtualMachineInstance) (net.Conn, *k8serrors.StatusError) {
	logger := log.Log.Object(vmi)

	targetIP, err := getTargetInterfaceIP(n.app, vmi)
	if err != nil {
		logger.Reason(err).Error("Can't establish TCP tunnel.")
		return nil, k8serrors.NewBadRequest(err.Error())
	}

	port := n.request.PathParameter(definitions.PortParamName)
	if len(port) < 1 {
		return nil, k8serrors.NewBadRequest("port must not be empty")
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
		return nil, k8serrors.NewInternalError(fmt.Errorf("dialing VM: %w", err))
	}
	return conn, nil
}

func (app *SubresourceAPIApp) virtHandlerDialer(getURL URLResolver) dialer {
	return handlerDial{
		getURL: getURL,
		app:    app,
	}
}

func (app *SubresourceAPIApp) getVirtHandlerFor(vmi *v1.VirtualMachineInstance, getVirtHandlerURL URLResolver) (url string, conn kubecli.VirtHandlerConn, statusError *k8serrors.StatusError) {
	var err error
	if conn, err = app.getVirtHandlerConnForVMI(vmi); err != nil {
		statusError = k8serrors.NewBadRequest(err.Error())
		log.Log.Object(vmi).Reason(statusError).Error("Unable to establish connection to virt-handler")
		return
	}
	if url, err = getVirtHandlerURL(vmi, conn); err != nil {
		statusError = k8serrors.NewBadRequest(err.Error())
		log.Log.Object(vmi).Reason(statusError).Error("Unable to retrieve target handler URL")
		return
	}
	return
}

func (app *SubresourceAPIApp) getVirtHandlerConnForVMI(vmi *v1.VirtualMachineInstance) (kubecli.VirtHandlerConn, error) {
	if !vmi.IsRunning() && !vmi.IsScheduled() {
		return nil, errors.New(fmt.Sprintf("Unable to connect to VirtualMachineInstance because phase is %s instead of %s or %s", vmi.Status.Phase, v1.Running, v1.Scheduled))
	}
	return kubecli.NewVirtHandlerClient(app.virtCli, app.handlerHttpClient).Port(app.consoleServerPort).ForNode(vmi.Status.NodeName), nil
}

// get the first available interface IP
// if no interface is present, return error
func getTargetInterfaceIP(app *SubresourceAPIApp, vmi *v1.VirtualMachineInstance) (string, error) {
	log.Log.Object(vmi).Warningf("DELETEME, getTargetInterfaceIP")
	virtLauncherPodList, err := app.virtCli.CoreV1().Pods(vmi.Namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: labels.Set{v1.VirtualMachineNameLabel: vmi.Name}.String(),
	})
	if err != nil {
		return "", fmt.Errorf("failed looking for virt launcher pod for vm '%s/%s': %w", vmi.Namespace, vmi.Name, err)
	}
	virtLauncherPod := virtLauncherPodList.Items[0]
	virtLauncherPodIP := virtLauncherPod.Status.PodIP
	if virtLauncherPodIP == "" {
		return "", fmt.Errorf("missing ip at virt launcher pod %q for vm '%s/%s': %w", virtLauncherPod.Name, vmi.Namespace, vmi.Name, err)
	}
	log.Log.Object(vmi).Warningf("DELETEME, getTargetInterfaceIP, virtLauncherPodList.Items[0].Status.PodIP: %s", virtLauncherPodList.Items[0].Status.PodIP)
	return virtLauncherPodList.Items[0].Status.PodIP, nil
}
