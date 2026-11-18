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
	"net"

	"github.com/gorilla/websocket"

	"github.com/emicklei/go-restful/v3"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/cache"
	netutils "k8s.io/utils/net"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	kvcorev1 "kubevirt.io/client-go/kubevirt/typed/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/virt-api/definitions"
)

type netDial struct {
	request            *restful.Request
	launcherPodIndexer cache.Indexer
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
	conn, resp, err := kvcorev1.Dial(url, h.app.handlerTLSConfiguration)
	if err != nil {
		return nil, k8serrors.NewInternalError(kvcorev1.EnrichError(err, resp))
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

	targetIP, err := n.resolveTargetIP(vmi)
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
	// Port-forward only supports TCP and UDP. Reject any other client-supplied
	// network type instead of forwarding it to net.Dial.
	if protocol != "tcp" && protocol != "udp" {
		return nil, k8serrors.NewBadRequest(fmt.Sprintf("unsupported protocol %q", protocol))
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
		return nil, fmt.Errorf("Unable to connect to VirtualMachineInstance because phase is %s instead of %s or %s", vmi.Status.Phase, v1.Running, v1.Scheduled)
	}
	return kubecli.NewVirtHandlerClient(app.virtClient, app.handlerHttpClient).Port(app.consoleServerPort).ForNode(vmi.Status.NodeName), nil
}

// resolveTargetIP returns the launcher pod IP, the only address virt-api and
// virt-launcher are guaranteed to share. The guest-reported interface IP is
// never dialed: it reflects the guest's own view of its addresses and is
// untrusted input. Callers must ensure the VMI is connected to the pod network;
// validateVMIForPortForward enforces this before the dialer runs.
func (n netDial) resolveTargetIP(vmi *v1.VirtualMachineInstance) (string, error) {
	if n.launcherPodIndexer == nil {
		return "", fmt.Errorf("launcher pod indexer is not available")
	}
	pod, err := controller.CurrentVMIPod(vmi, n.launcherPodIndexer)
	if err != nil {
		return "", fmt.Errorf("failed to look up the virt-launcher pod: %w", err)
	}
	if pod == nil {
		return "", fmt.Errorf("virt-launcher pod was not found")
	}
	if pod.Status.PodIP == "" {
		return "", fmt.Errorf("virt-launcher pod IP is not available yet")
	}
	return pod.Status.PodIP, nil
}
