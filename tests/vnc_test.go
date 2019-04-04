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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package tests_test

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/rest"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/util/subresources"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("[rfe_id:127][crit:medium][vendor:cnv-qe@redhat.com][level:component]VNC", func() {

	flag.Parse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)
	var vmi *v1.VirtualMachineInstance

	Describe("[rfe_id:127][crit:medium][vendor:cnv-qe@redhat.com][level:component]A new VirtualMachineInstance", func() {
		tests.BeforeAll(func() {
			tests.BeforeTestCleanup()
			vmi = tests.NewRandomVMI()
			Expect(virtClient.RestClient().Post().Resource("virtualmachineinstances").Namespace(tests.NamespaceTestDefault).Body(vmi).Do().Error()).To(Succeed())
			tests.WaitForSuccessfulVMIStart(vmi)
		})

		Context("with VNC connection", func() {

			vncConnect := func() {
				pipeInReader, _ := io.Pipe()
				pipeOutReader, pipeOutWriter := io.Pipe()
				defer pipeInReader.Close()
				defer pipeOutReader.Close()

				k8ResChan := make(chan error)
				readStop := make(chan string)

				go func() {
					GinkgoRecover()
					vnc, err := virtClient.VirtualMachineInstance(vmi.ObjectMeta.Namespace).VNC(vmi.ObjectMeta.Name)
					if err != nil {
						k8ResChan <- err
						return
					}

					k8ResChan <- vnc.Stream(kubecli.StreamOptions{
						In:  pipeInReader,
						Out: pipeOutWriter,
					})
				}()
				// write to FD <- pipeOutReader
				By("Reading from the VNC socket")
				go func() {
					GinkgoRecover()
					buf := make([]byte, 1024, 1024)
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

				response := ""

				select {
				case response = <-readStop:
				case err = <-k8ResChan:
					Expect(err).ToNot(HaveOccurred())
				case <-time.After(45 * time.Second):
					Fail("Timout reached while waiting for valid VNC server response")
				}

				// This is the response capture by wireshark when the VNC server is contacted.
				// This verifies that the test is able to establish a connection with VNC and
				// communicate.
				By("Checking the response from VNC server")
				Expect(response).To(Equal("RFB 003.008\n"))
				Expect(err).To(BeNil())
			}

			It("[test_id:1611]should allow accessing the VNC device", func() {
				vncConnect()
			})

			It("should allow accessing the VNC device multiple times", func() {

				for i := 0; i < 10; i++ {
					vncConnect()
				}
			})
		})

		table.DescribeTable("[rfe_id:127][crit:medium][vendor:cnv-qe@redhat.com][level:component]should upgrade websocket connection which look like coming from a browser", func(subresource string) {
			config, err := kubecli.GetKubevirtClientConfig()
			Expect(err).ToNot(HaveOccurred())
			// Browsers need a subprotocol, since they will have to use the subprotocol mechanism to forward the bearer token.
			// As a consequence they need a subprotocol match.
			rt, err := upgradeCheckRoundTripperFromConfig(config, []string{"fantasy.protocol", subresources.PlainStreamProtocolName})
			Expect(err).ToNot(HaveOccurred())
			wrappedRoundTripper, err := rest.HTTPWrappersForConfig(config, rt)
			Expect(err).ToNot(HaveOccurred())
			req, err := kubecli.RequestFromConfig(config, vmi.Name, vmi.Namespace, subresource)

			// Add an Origin header to look more like an arbitrary browser
			if req.Header == nil {
				req.Header = http.Header{}
			}
			req.Header.Add("Origin", config.Host)
			_, err = wrappedRoundTripper.RoundTrip(req)
			Expect(err).ToNot(HaveOccurred())
			Expect(rt.Response.Header.Get("Sec-Websocket-Protocol")).To(Equal(subresources.PlainStreamProtocolName))
		},
			table.Entry("[test_id:1612]for vnc", "vnc"),
			table.Entry("[test_id:1613]for serial console", "console"),
		)

		It("[test_id:1614]should upgrade websocket connections without a subprotocol given", func() {
			config, err := kubecli.GetKubevirtClientConfig()
			Expect(err).ToNot(HaveOccurred())
			// If no subprotocol is given, we still want to upgrade to be backward compatible
			rt, err := upgradeCheckRoundTripperFromConfig(config, nil)
			Expect(err).ToNot(HaveOccurred())
			wrappedRoundTripper, err := rest.HTTPWrappersForConfig(config, rt)
			Expect(err).ToNot(HaveOccurred())
			req, err := kubecli.RequestFromConfig(config, vmi.Name, vmi.Namespace, "vnc")
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
		WriteBufferSize: kubecli.WebsocketMessageBufferSize,
		ReadBufferSize:  kubecli.WebsocketMessageBufferSize,
		Subprotocols:    subprotocols,
	}

	// Create a roundtripper which will pass in the final underlying websocket connection to a callback
	return &checkUpgradeRoundTripper{
		Dialer: dialer,
	}, nil
}
