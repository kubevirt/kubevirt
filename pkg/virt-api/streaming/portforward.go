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

package streaming

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"k8s.io/apimachinery/pkg/api/errors"

	v1 "kubevirt.io/api/core/v1"
	kvcorev1 "kubevirt.io/client-go/kubevirt/typed/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/controller"
	apimetrics "kubevirt.io/kubevirt/pkg/monitoring/metrics/virt-api"
)

const keepAliveTimeout = 1 * time.Minute

// StreamPortForward tunnels a TCP/UDP connection to the given port of the named
// VMI. Unlike the console/vnc/usbredir/vsock streams, portforward dials the VM
// directly and exchanges data with the client using websocket message framing
// plus periodic keep-alive pings, preserving the behavior of the legacy
// WebsocketStreamer based handler
func (s *Streamer) StreamPortForward(ctx context.Context, namespace, name, port, protocol string, w http.ResponseWriter, req *http.Request) *errors.StatusError {
	activeTunnelMetric := apimetrics.NewActivePortForwardTunnel(namespace, name)
	defer activeTunnelMetric.Dec()
	defer apimetrics.SetVMILastConnectionTimestamp(namespace, name)

	vmi, statusErr := s.FetchAndValidateVMI(ctx, namespace, name, validateVMIForPortForward)
	if statusErr != nil {
		return statusErr
	}

	serverConn, statusErr := dialVMPort(vmi, port, protocol)
	if statusErr != nil {
		return statusErr
	}

	upgrader := kvcorev1.NewUpgrader()
	upgrader.HandshakeTimeout = streamTimeout
	clientConn, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		serverConn.Close()
		return nil
	}

	streamCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		<-streamCtx.Done()
		serverConn.Close()
		clientConn.Close()
	}()
	go keepAliveClientStream(streamCtx, clientConn, cancel)

	results := make(chan error, 2)
	// server -> client, framed as websocket binary messages
	go func() {
		_, copyErr := kvcorev1.CopyTo(clientConn, serverConn)
		results <- copyErr
	}()
	// client -> server, de-framed from the websocket messages
	go func() {
		_, copyErr := kvcorev1.CopyFrom(serverConn, clientConn)
		results <- copyErr
	}()

	<-results
	cancel()
	<-results
	return nil
}

func validateVMIForPortForward(vmi *v1.VirtualMachineInstance) *errors.StatusError {
	condManager := controller.NewVirtualMachineInstanceConditionManager()
	if condManager.HasCondition(vmi, v1.VirtualMachineInstancePaused) {
		return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf("VMI is paused"))
	}
	return nil
}

// dialVMPort opens a raw TCP/UDP connection to the first interface IP of the VMI
func dialVMPort(vmi *v1.VirtualMachineInstance, port, protocol string) (net.Conn, *errors.StatusError) {
	logger := log.Log.Object(vmi)

	targetIP, err := getTargetInterfaceIP(vmi)
	if err != nil {
		logger.Reason(err).Error("Can't establish TCP tunnel.")
		return nil, errors.NewBadRequest(err.Error())
	}

	if len(port) < 1 {
		return nil, errors.NewBadRequest("port must not be empty")
	}

	if len(protocol) < 1 {
		protocol = "tcp"
	}

	// net.JoinHostPort brackets IPv6 addresses (e.g. [::1]:8080) instead of the manual formatting
	// avoid the vet warning
	addr := net.JoinHostPort(targetIP, port)
	conn, err := net.Dial(protocol, addr)
	if err != nil {
		logger.Reason(err).Errorf("Can't dial %s %s", protocol, addr)
		return nil, errors.NewInternalError(fmt.Errorf("dialing VM: %w", err))
	}
	return conn, nil
}

// getTargetInterfaceIP returns the first available interface IP of the VMI
func getTargetInterfaceIP(vmi *v1.VirtualMachineInstance) (string, error) {
	interfaces := vmi.Status.Interfaces
	if len(interfaces) < 1 {
		return "", fmt.Errorf("no network interfaces are present")
	}
	return interfaces[0].IP, nil
}

// keepAliveClientStream pings the client websocket periodically and cancels the
// stream if the client stops responding
func keepAliveClientStream(ctx context.Context, conn *websocket.Conn, cancel func()) {
	pingTicker := time.NewTicker(1 * time.Second)
	defer pingTicker.Stop()
	conn.SetReadDeadline(time.Now().Add(keepAliveTimeout))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(keepAliveTimeout))
		return nil
	})

	for {
		select {
		case <-ctx.Done():
			return
		case <-pingTicker.C:
			if err := conn.WriteControl(websocket.PingMessage, []byte("keep alive"), time.Now().Add(keepAliveTimeout)); err != nil {
				log.Log.Reason(err).Error("Failed to write control message to client websocket connection")
				cancel()
				return
			}
		}
	}
}
