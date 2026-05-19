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

package libpod

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"

	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests/framework/kubevirt"
)

// GetCertsForPods returns the used certificates for all pods matching  the label selector
func GetCertsForPods(labelSelector, namespace, port string) ([][]byte, error) {
	cli := kubevirt.Client()
	pods, err := cli.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{LabelSelector: labelSelector})
	Expect(err).ToNot(HaveOccurred())
	Expect(pods.Items).ToNot(BeEmpty())

	var certs [][]byte

	//nolint:gocritic
	for _, pod := range pods.Items {
		podCopy := pod
		err := func() error {
			certs = append(certs, getCert(&podCopy, port))
			return nil
		}()
		if err != nil {
			return nil, err
		}
	}
	return certs, nil
}

func getCert(pod *k8sv1.Pod, port string) []byte {
	var rawCert []byte
	mutex := &sync.Mutex{}
	conf := &tls.Config{
		//nolint:gosec
		InsecureSkipVerify: true,
		VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
			mutex.Lock()
			defer mutex.Unlock()
			rawCert = rawCerts[0]
			return nil
		},
	}

	var certificate []byte
	const offset = 2
	EventuallyWithOffset(offset, func() []byte {
		stopChan := make(chan struct{})
		defer close(stopChan)
		const timeout = 10
		localPort, err := ForwardPorts(pod, []string{"0:" + port}, stopChan, timeout*time.Second)
		ExpectWithOffset(offset, err).ToNot(HaveOccurred())

		conn, err := tls.Dial("tcp4", fmt.Sprintf("localhost:%d", localPort), conf)
		if err == nil {
			defer conn.Close()
		}
		mutex.Lock()
		defer mutex.Unlock()
		certificate = make([]byte, len(rawCert))
		copy(certificate, rawCert)
		return certificate
	}, 40*time.Second, 1*time.Second).Should(Not(BeEmpty()))

	return certificate
}

// ForwardPorts starts port-forwarding from a pod's remote port to a local port
// and waits until the tunnel is ready. Returns the assigned local port number.
// Pass "0:remotePort" in ports to let the OS pick a free ephemeral local port.
func ForwardPorts(pod *k8sv1.Pod, ports []string, stop chan struct{}, readyTimeout time.Duration) (uint16, error) {
	if len(ports) != 1 {
		return 0, fmt.Errorf("ForwardPorts requires exactly one port mapping, got %d", len(ports))
	}
	errChan := make(chan error, 1)
	readyChan := make(chan struct{})
	var forwarder *portforward.PortForwarder
	go func() {
		cli := kubevirt.Client()

		req := cli.CoreV1().RESTClient().Post().
			Resource("pods").
			Namespace(pod.Namespace).
			Name(pod.Name).
			SubResource("portforward")

		kubevirtClientConfig, err := kubecli.GetKubevirtClientConfig()
		if err != nil {
			errChan <- err
			return
		}
		transport, upgrader, err := spdy.RoundTripperFor(kubevirtClientConfig)
		if err != nil {
			errChan <- err
			return
		}
		dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, "POST", req.URL())
		forwarder, err = portforward.New(dialer, ports, stop, readyChan, GinkgoWriter, GinkgoWriter)
		if err != nil {
			errChan <- err
			return
		}
		if err = forwarder.ForwardPorts(); err != nil {
			errChan <- err
		}
	}()

	// Wait for forwarding to be ready, then get ports synchronously
	select {
	case err := <-errChan:
		return 0, err
	case <-readyChan:
		assignedPorts, err := forwarder.GetPorts()
		if err != nil {
			return 0, err
		}
		if len(assignedPorts) == 0 {
			return 0, fmt.Errorf("no ports were forwarded")
		}
		return assignedPorts[0].Local, nil
	case <-time.After(readyTimeout):
		return 0, fmt.Errorf("failed to forward ports, timed out")
	}
}
