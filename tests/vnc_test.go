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
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"kubevirt.io/kubevirt/tests/decorators"

	"github.com/mitchellh/go-vnc"

	"kubevirt.io/kubevirt/tests/clientcmd"
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

	launcherApi "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"

	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libwait"
)

var _ = Describe("[rfe_id:127][crit:medium][arm64][vendor:cnv-qe@redhat.com][level:component][sig-compute]VNC", decorators.SigCompute, func() {

	var vmi *v1.VirtualMachineInstance

	Describe("[rfe_id:127][crit:medium][vendor:cnv-qe@redhat.com][level:component]A new VirtualMachineInstance", func() {
		BeforeEach(func() {
			var err error
			vmi = libvmifact.NewGuestless()
			vmi, err = kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			vmi = libwait.WaitForSuccessfulVMIStart(vmi)
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
					defer GinkgoRecover()
					vnc, err := kubevirt.Client().VirtualMachineInstance(vmi.ObjectMeta.Namespace).VNC(vmi.ObjectMeta.Name, false)
					if err != nil {
						k8ResChan <- err
						return
					}

					k8ResChan <- vnc.Stream(kvcorev1.StreamOptions{
						In:  pipeInReader,
						Out: pipeOutWriter,
					})
				}()
				// write to FD <- pipeOutReader
				By("Reading from the VNC socket")
				go func() {
					defer GinkgoRecover()
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
					Fail("Timout reached while waiting for valid VNC server response")
				}
			}

			It("[test_id:1611]should allow accessing the VNC device multiple times", func() {

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

		It("[test_id:4272]should connect to vnc with --proxy-only flag", func() {

			By("Invoking virtctl vnc with --proxy-only")
			proxyOnlyCommand := clientcmd.NewVirtctlCommand("vnc", "--proxy-only", "--namespace", vmi.Namespace, vmi.Name)

			r, w, _ := os.Pipe()
			proxyOnlyCommand.SetOut(w)

			// Run this as go routine to keep proxy open in the background
			go func() {
				defer GinkgoRecover()
				Expect(proxyOnlyCommand.Execute()).ToNot(HaveOccurred())
			}()

			var result map[string]interface{}
			Eventually(func() error {
				return json.NewDecoder(r).Decode(&result)
			}, 60*time.Second).ShouldNot(HaveOccurred())

			port := result["port"]
			addr := fmt.Sprintf("127.0.0.1:%v", port)

			nc, err := net.Dial("tcp", addr)
			Expect(err).ToNot(HaveOccurred())
			defer nc.Close()

			ch := make(chan vnc.ServerMessage)

			c, err := vnc.Client(nc, &vnc.ClientConfig{
				Exclusive:       false,
				ServerMessageCh: ch,
				ServerMessages:  []vnc.ServerMessage{new(vnc.FramebufferUpdateMessage)},
			})
			Expect(err).ToNot(HaveOccurred())
			defer c.Close()
			Expect(c.DesktopName).To(ContainSubstring(vmi.Name))
		})

		It("[test_id:5274]should connect to vnc with --proxy-only flag to the specified port", func() {
			testPort := "33333"

			By("Invoking virtctl vnc with --proxy-only")
			proxyOnlyCommand := clientcmd.NewVirtctlCommand("vnc", "--proxy-only", "--port", testPort, "--namespace", vmi.Namespace, vmi.Name)

			// Run this as go routine to keep proxy open in the background
			go func() {
				defer GinkgoRecover()
				Expect(proxyOnlyCommand.Execute()).ToNot(HaveOccurred())
			}()

			addr := fmt.Sprintf("127.0.0.1:%s", testPort)

			// Run this under Eventually so we don't dial connection before proxy has started
			Eventually(func() error {
				nc, err := net.Dial("tcp", addr)
				if err != nil {
					return err
				}
				defer nc.Close()

				ch := make(chan vnc.ServerMessage)

				c, err := vnc.Client(nc, &vnc.ClientConfig{
					Exclusive:       false,
					ServerMessageCh: ch,
					ServerMessages:  []vnc.ServerMessage{new(vnc.FramebufferUpdateMessage)},
				})
				Expect(err).ToNot(HaveOccurred())
				defer c.Close()
				Expect(c.DesktopName).To(ContainSubstring(vmi.Name))

				return nil
			}, 60*time.Second).ShouldNot(HaveOccurred())
		})

		It("should allow creating a VNC screenshot in PNG format", func() {
			filePath := filepath.Join(GinkgoT().TempDir(), "screenshot.png")
			domain, err := tests.GetRunningVMIDomainSpec(vmi)
			Expect(err).ToNot(HaveOccurred())
			// According to the video device type to set the expected resolution
			// The default resolution is 720x400 for vga device while it is 1280x800 for virtio-gpu device
			xres, yres := getResolution(domain)
			// Sometimes we can see initially a 640x480 resolution if we connect very early
			By("gathering screenshots until we are past the first boot screen and see the expected 720x400 resolution")
			Eventually(func() image.Image {
				cmd := clientcmd.NewVirtctlCommand("vnc", "screenshot", "--namespace", vmi.Namespace, "--file", filePath, vmi.Name)
				Expect(cmd.Execute()).To(Succeed())

				f, err := os.Open(filePath)
				Expect(err).ToNot(HaveOccurred())
				defer f.Close()

				img, err := png.Decode(f)
				Expect(err).ToNot(HaveOccurred())
				return img
			}, 10*time.Second).Should(HaveResolution(xres, yres))
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

type ResolutionMatcher struct {
	X, Y int
}

func (h ResolutionMatcher) Match(actual interface{}) (success bool, err error) {
	x, y, err := imgSize(actual)
	if err != nil {
		return false, nil
	}
	return x == h.X && y == h.Y, nil
}

func (h ResolutionMatcher) FailureMessage(actual interface{}) (message string) {
	x, y, err := imgSize(actual)
	if err != nil {
		return err.Error()
	}
	return fmt.Sprintf("Expected (X: %d, Y: %d) to match (X: %d, Y: %d)", x, y, h.X, h.Y)
}

func (h ResolutionMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	x, y, err := imgSize(actual)
	if err != nil {
		return err.Error()
	}
	return fmt.Sprintf("Expected (X: %d, Y: %d) to not match (X: %d, Y: %d)", x, y, h.X, h.Y)
}

func getResolution(domain *launcherApi.DomainSpec) (X, Y int) {
	videoType := domain.Devices.Video[0].Model.Type
	if videoType == "virtio" {
		X = 1280
		Y = 800
	} else {
		X = 720
		Y = 400
	}
	return X, Y
}

func HaveResolution(X, Y int) ResolutionMatcher {
	return ResolutionMatcher{X: X, Y: Y}
}

func imgSize(actual interface{}) (X, Y int, err error) {
	if actual == nil {
		return -1, -1, fmt.Errorf("expected an object of type image.Image but got nil")
	}
	img, ok := actual.(image.Image)
	if !ok {
		return -1, -1, fmt.Errorf("expected an object of type image.Image")
	}
	size := img.Bounds().Size()
	return size.X, size.Y, nil
}
