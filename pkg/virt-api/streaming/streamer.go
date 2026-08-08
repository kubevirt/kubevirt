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
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	kvcorev1 "kubevirt.io/client-go/kubevirt/typed/core/v1"
)

const streamTimeout = 10 * time.Second

// this would validate a VMI is eligible for a given streaming subresource
type VMIValidator func(*v1.VirtualMachineInstance) *errors.StatusError

// this would  returns the virt-handler URL to dial for the streaming target
type URLResolver func(*v1.VirtualMachineInstance, kubecli.VirtHandlerConn) (string, error)

// Streamer proxies a connection between an aggregated API client and virt-handler
type Streamer struct {
	virtCli           kubecli.KubevirtClient
	consoleServerPort int
	tlsConfig         *tls.Config
	httpClient        *http.Client
}

func NewStreamer(virtCli kubecli.KubevirtClient, consoleServerPort int, tlsConfig *tls.Config) *Streamer {
	return &Streamer{
		virtCli:           virtCli,
		consoleServerPort: consoleServerPort,
		tlsConfig:         tlsConfig,
		httpClient: &http.Client{
			Transport: &http.Transport{TLSClientConfig: tlsConfig},
			Timeout:   streamTimeout,
		},
	}
}

// StreamRaw fetches and validates the VMI, dials virt-handler, upgrades the
// client req to a websocket and copies bytes in both directions until
// either side closes.
func (s *Streamer) StreamRaw(ctx context.Context, namespace, name string, w http.ResponseWriter, req *http.Request, validate VMIValidator, getURL URLResolver) *errors.StatusError {
	serverConn, statusErr := s.dialVirtHandler(ctx, namespace, name, validate, getURL)
	if statusErr != nil {
		return statusErr
	}

	upgrader := kvcorev1.NewUpgrader()
	upgrader.HandshakeTimeout = streamTimeout
	clientConn, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		// The upgrader alr wrote the failure resp to the client
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

	results := make(chan error, 2)
	go func() {
		_, copyErr := io.Copy(serverConn, clientConn.UnderlyingConn())
		results <- copyErr
	}()
	go func() {
		_, copyErr := io.Copy(clientConn.UnderlyingConn(), serverConn)
		results <- copyErr
	}()

	<-results

	cancel()
	<-results
	return nil
}

// fetchAndValidateVMI retrieves the named VMI and runs the subresource specific validation
func (s *Streamer) fetchAndValidateVMI(ctx context.Context, namespace, name string, validate VMIValidator) (*v1.VirtualMachineInstance, *errors.StatusError) {
	vmi, err := s.virtCli.VirtualMachineInstance(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, errors.NewNotFound(v1.Resource("virtualmachineinstance"), name)
		}
		return nil, errors.NewInternalError(fmt.Errorf("unable to retrieve vmi [%s]: %w", name, err))
	}

	if statusErr := validate(vmi); statusErr != nil {
		return nil, statusErr
	}

	return vmi, nil
}

// dialVirtHandler resolves the virt-handler endpoint for the VMI
// and returns the net.Conn of the websocket connection
func (s *Streamer) dialVirtHandler(ctx context.Context, namespace, name string, validate VMIValidator, getURL URLResolver) (net.Conn, *errors.StatusError) {
	vmi, statusErr := s.fetchAndValidateVMI(ctx, namespace, name, validate)
	if statusErr != nil {
		return nil, statusErr
	}

	if !vmi.IsRunning() && !vmi.IsScheduled() {
		return nil, errors.NewBadRequest(fmt.Sprintf("Unable to connect to VirtualMachineInstance because phase is %s instead of %s or %s", vmi.Status.Phase, v1.Running, v1.Scheduled))
	}

	conn := kubecli.NewVirtHandlerClient(s.virtCli, s.httpClient).Port(s.consoleServerPort).ForNode(vmi.Status.NodeName)
	url, err := getURL(vmi, conn)
	if err != nil {
		return nil, errors.NewBadRequest(err.Error())
	}

	wsConn, resp, err := kvcorev1.Dial(url, s.tlsConfig)
	if err != nil {
		return nil, errors.NewInternalError(kvcorev1.EnrichError(err, resp))
	}
	return wsConn.UnderlyingConn(), nil
}
