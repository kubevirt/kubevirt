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
	"errors"
	"fmt"
	"net"

	"github.com/gorilla/websocket"

	"github.com/emicklei/go-restful/v3"
	k8sv1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

// resolveTargetIP returns the IP virt-api should dial to reach the VMI. It
// prefers the current virt-launcher pod IP, which is always routable from the
// pod network, and falls back to the first reported guest interface IP. The
// guest interface IP may not be routable from the pod network (for example
// when the VM is attached to a VPC network), so it is only used when the
// launcher pod IP cannot be determined.
func (n netDial) resolveTargetIP(vmi *v1.VirtualMachineInstance) (string, error) {
	if podIP := n.launcherPodIP(vmi); podIP != "" {
		return podIP, nil
	}
	return getTargetInterfaceIP(vmi)
}

// launcherPodIP looks up the IP of the virt-launcher pod currently backing the
// VMI. It returns an empty string (rather than an error) when the pod cannot be
// found or has no IP yet, so the caller can fall back to the interface IP.
func (n netDial) launcherPodIP(vmi *v1.VirtualMachineInstance) string {
	if n.app == nil {
		return ""
	}
	pods, err := n.app.virtCli.CoreV1().Pods(vmi.Namespace).List(
		n.request.Request.Context(),
		metav1.ListOptions{LabelSelector: fmt.Sprintf("%s=%s", v1.AppLabel, "virt-launcher")},
	)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Warning("failed to list virt-launcher pods; falling back to the VMI interface IP")
		return ""
	}
	if pod := currentLauncherPod(vmi, pods.Items); pod != nil {
		return pod.Status.PodIP
	}
	return ""
}

// currentLauncherPod selects the virt-launcher pod backing the VMI right now:
// the most recently created pod owned by the VMI and scheduled to the VMI's
// node. This mirrors the selection performed by controller.CurrentVMIPod and
// ensures that during a migration the target pod is only chosen once the VMI
// node has been handed over.
func currentLauncherPod(vmi *v1.VirtualMachineInstance, pods []k8sv1.Pod) *k8sv1.Pod {
	var current *k8sv1.Pod
	for i := range pods {
		pod := &pods[i]
		if !metav1.IsControlledBy(pod, vmi) {
			continue
		}
		if vmi.Status.NodeName != "" && vmi.Status.NodeName != pod.Spec.NodeName {
			continue
		}
		if current == nil || current.CreationTimestamp.Before(&pod.CreationTimestamp) {
			current = pod
		}
	}
	return current
}

// get the first available interface IP
// if no interface is present, return error
func getTargetInterfaceIP(vmi *v1.VirtualMachineInstance) (string, error) {
	interfaces := vmi.Status.Interfaces
	if len(interfaces) < 1 {
		return "", fmt.Errorf("no network interfaces are present")
	}
	return interfaces[0].IP, nil
}
