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

package tests_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"kubevirt.io/kubevirt/tests/decorators"

	"kubevirt.io/kubevirt/tests/testsuite"

	"github.com/gorilla/websocket"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/rest"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	kvcorev1 "kubevirt.io/client-go/kubevirt/typed/core/v1"
	"kubevirt.io/client-go/log"
	"kubevirt.io/client-go/subresources"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libwait"
)

var _ = Describe("[rfe_id:127][crit:medium][vendor:cnv-qe@redhat.com][level:component][sig-compute]VNC", decorators.SigCompute, decorators.WgArm64, func() {

	var vmi *v1.VirtualMachineInstance

	Describe("[rfe_id:127][crit:medium][vendor:cnv-qe@redhat.com][level:component]A new VirtualMachineInstance", func() {
		BeforeEach(func() {
			var err error
			vmi = libvmifact.NewGuestless(libvmi.WithAutoattachGraphicsDevice(true))
			vmi, err = kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			vmi = libwait.WaitForSuccessfulVMIStart(vmi)
		})

		Context("with VNC connection", func() {
			vncConnect := func() {
				pipeOutReader, pipeOutWriter := io.Pipe()
				defer pipeOutReader.Close()

				k8ResChan := make(chan error)
				readStop := make(chan string)

				go func() {
					defer GinkgoRecover()
					vnc, err := kubevirt.Client().VirtualMachineInstance(vmi.ObjectMeta.Namespace).VNC(vmi.ObjectMeta.Name)
					if err != nil {
						k8ResChan <- err
						return
					}

					pipeInReader, _ := io.Pipe()
					defer pipeInReader.Close()

					k8ResChan <- vnc.Stream(kvcorev1.StreamOptions{
						In:  pipeInReader,
						Out: pipeOutWriter,
					})
				}()
				// write to FD <- pipeOutReader
				By("Reading from the VNC socket")
				go func() {
					defer GinkgoRecover()
					buf := make([]byte, 1024)
					// reading qemu vnc server
					n, err := pipeOutReader.Read(buf)
					if err != nil && err != io.EOF {
						Expect(err).ToNot(HaveOccurred())
						return
					}
					if n == 0 && err == io.EOF {
						log.Log.Info("zero bytes read from vnc socket.")
						return
					}
					readStop <- string(buf[0:n])
				}()

				select {
				case response := <-readStop:
					// This is the response capture by wireshark when the VNC server is contacted.
					// This verifies that the test is able to establish a connection with VNC and
					// communicate.
					By("Checking the response from VNC server")
					Expect(response).To(Equal("RFB 003.008\n"))
				case err := <-k8ResChan:
					Expect(err).ToNot(HaveOccurred())
				case <-time.After(45 * time.Second):
					Fail("Timeout reached while waiting for valid VNC server response")
				}
			}

			It("[test_id:1611]should allow accessing the VNC device multiple times", decorators.Conformance, func() {

				for i := 0; i < 10; i++ {
					vncConnect()
				}
			})
		})

		DescribeTable("[rfe_id:127][crit:medium][vendor:cnv-qe@redhat.com][level:component]should upgrade websocket connection which look like coming from a browser", func(subresource string) {
			config, err := kubecli.GetKubevirtClientConfig()
			Expect(err).ToNot(HaveOccurred())
			// Browsers need a subprotocol, since they will have to use the subprotocol mechanism to forward the bearer token.
			// As a consequence they need a subprotocol match.
			rt, err := upgradeCheckRoundTripperFromConfig(config, []string{"fantasy.protocol", subresources.PlainStreamProtocolName})
			Expect(err).ToNot(HaveOccurred())
			wrappedRoundTripper, err := rest.HTTPWrappersForConfig(config, rt)
			Expect(err).ToNot(HaveOccurred())
			req, err := kvcorev1.RequestFromConfig(config, "virtualmachineinstances", vmi.Name, vmi.Namespace, subresource, url.Values{})
			Expect(err).ToNot(HaveOccurred())

			// Add an Origin header to look more like an arbitrary browser
			if req.Header == nil {
				req.Header = http.Header{}
			}
			req.Header.Add("Origin", config.Host)
			_, err = wrappedRoundTripper.RoundTrip(req)
			Expect(err).ToNot(HaveOccurred())
			Expect(rt.Response.Header.Get("Sec-Websocket-Protocol")).To(Equal(subresources.PlainStreamProtocolName))
		},
			Entry("[test_id:1612]for vnc", "vnc"),
			// TODO: This should be moved to console tests
			Entry("[test_id:1613]for serial console", "console"),
		)

		It("[test_id:1614]should upgrade websocket connections without a subprotocol given", func() {
			config, err := kubecli.GetKubevirtClientConfig()
			Expect(err).ToNot(HaveOccurred())
			// If no subprotocol is given, we still want to upgrade to be backward compatible
			rt, err := upgradeCheckRoundTripperFromConfig(config, nil)
			Expect(err).ToNot(HaveOccurred())
			wrappedRoundTripper, err := rest.HTTPWrappersForConfig(config, rt)
			Expect(err).ToNot(HaveOccurred())
			req, err := kvcorev1.RequestFromConfig(config, "virtualmachineinstances", vmi.Name, vmi.Namespace, "vnc", url.Values{})
			Expect(err).ToNot(HaveOccurred())
			_, err = wrappedRoundTripper.RoundTrip(req)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})

// checkUpgradeRoundTripper checks if an upgrade confirmation is received from the server
type checkUpgradeRoundTripper struct {
	Dialer   *websocket.Dialer
	Response *http.Response
}

func (t *checkUpgradeRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	conn, resp, err := t.Dialer.Dial(r.URL.String(), r.Header)
	t.Response = resp
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("%v: %d", err, resp.StatusCode))
	Expect(resp.StatusCode).To(Equal(http.StatusSwitchingProtocols))
	conn.Close()
	return nil, nil
}

// upgradeCheckRoundTripperFromConfig returns a wrapped roundtripper which checks if an upgrade confirmation from servers is received
func upgradeCheckRoundTripperFromConfig(config *rest.Config, subprotocols []string) (*checkUpgradeRoundTripper, error) {

	// Configure TLS
	tlsConfig, err := rest.TLSConfigFor(config)
	if err != nil {
		return nil, err
	}

	// Configure the websocket dialer
	dialer := &websocket.Dialer{
		Proxy:           http.ProxyFromEnvironment,
		TLSClientConfig: tlsConfig,
		WriteBufferSize: kvcorev1.WebsocketMessageBufferSize,
		ReadBufferSize:  kvcorev1.WebsocketMessageBufferSize,
		Subprotocols:    subprotocols,
	}

	// Create a roundtripper which will pass in the final underlying websocket connection to a callback
	return &checkUpgradeRoundTripper{
		Dialer: dialer,
	}, nil
}
