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
	"math/rand"
	"net/http"
	"strconv"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"

	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests/framework/k8s"
)

// GetCertsForPods returns the used certificates for all pods matching  the label selector
func GetCertsForPods(labelSelector, namespace, port string) ([][]byte, error) {
	pods, err := k8s.Client().CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{LabelSelector: labelSelector})
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
	const (
		basePort  = 4321
		portRange = 6000
	)
	//nolint:gosec
	randPort := strconv.Itoa(basePort + rand.Intn(portRange))
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
		err := ForwardPorts(pod, []string{fmt.Sprintf("%s:%s", randPort, port)}, stopChan, timeout*time.Second)
		ExpectWithOffset(offset, err).ToNot(HaveOccurred())

		conn, err := tls.Dial("tcp4", fmt.Sprintf("localhost:%s", randPort), conf)
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

func ForwardPorts(pod *k8sv1.Pod, ports []string, stop chan struct{}, readyTimeout time.Duration) error {
	errChan := make(chan error, 1)
	readyChan := make(chan struct{})
	go func() {
		req := k8s.Client().CoreV1().RESTClient().Post().
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
		forwarder, err := portforward.New(dialer, ports, stop, readyChan, GinkgoWriter, GinkgoWriter)
		if err != nil {
			errChan <- err
			return
		}
		err = forwarder.ForwardPorts()
		if err != nil {
			errChan <- err
		}
	}()

	select {
	case err := <-errChan:
		return err
	case <-readyChan:
		return nil
	case <-time.After(readyTimeout):
		return fmt.Errorf("failed to forward ports, timed out")
	}
}
