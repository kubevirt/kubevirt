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
 * Copyright The KubeVirt Authors
 *
 */

package virtctl

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"

	"github.com/mitchellh/go-vnc"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

type ctxKeyTypeVNC string

const connectedKeyVNC ctxKeyTypeVNC = "connected"

var _ = Describe(SIG("[sig-compute]VNC", decorators.SigCompute, decorators.WgArm64, Ordered, decorators.OncePerOrderedCleanup, func() {
	var vmi *v1.VirtualMachineInstance
	const proxyConnectTimeout = time.Second * 5

	BeforeAll(func() {
		var err error
		vmi = libvmifact.NewGuestless(libvmi.WithAutoattachGraphicsDevice(true))
		vmi, err = kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).
			Create(context.Background(), vmi, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		vmi = libwait.WaitForSuccessfulVMIStart(vmi)
	})

	It("[rfe_id:127][crit:medium][vendor:cnv-qe@redhat.com][level:component]"+
		"[test_id:4272]should connect to vnc with --proxy-only flag", func() {
		By("Invoking virtctl vnc with --proxy-only")
		r, w, _ := os.Pipe()
		cmd := newVirtctlCommand(
			"vnc",
			vmi.Name,
			"--namespace", vmi.Namespace,
			"--proxy-only",
		)
		cmd.SetOut(w)

		go func() {
			defer GinkgoRecover()
			Expect(cmd.Execute()).To(Succeed())
		}()

		var result map[string]interface{}
		Eventually(func() error {
			return json.NewDecoder(r).Decode(&result)
		}, 60*time.Second).Should(Succeed())

		ctx, cancel := context.WithTimeout(context.Background(), proxyConnectTimeout)
		DeferCleanup(cancel)
		verifyProxyConnection(ctx, fmt.Sprintf("%v", result["port"]), vmi.Name, true)
	})

	It("[rfe_id:127][crit:medium][vendor:cnv-qe@redhat.com][level:component]"+
		"[test_id:5274]should connect to vnc with --proxy-only flag to the specified port", func() {
		const (
			portRangeFirst = 33333
			portRangeLast  = 33433
		)
		port, found := findOpenPort(portRangeFirst, portRangeLast)
		Expect(found).To(BeTrue())
		testPort := strconv.FormatInt(int64(port), 10)

		By("Invoking virtctl vnc with --proxy-only")
		go func() {
			defer GinkgoRecover()
			err := newRepeatableVirtctlCommand(
				"vnc",
				vmi.Name,
				"--namespace", vmi.Namespace,
				"--proxy-only",
				"--port", testPort,
			)()
			Expect(err).ToNot(HaveOccurred())
		}()

		ctx, cancel := context.WithTimeout(context.Background(), proxyConnectTimeout)
		DeferCleanup(cancel)
		verifyProxyConnection(ctx, testPort, vmi.Name, true)
	})

	It("[rfe_id:127][crit:medium][vendor:cnv-qe@redhat.com][level:component]"+
		"[test_id:11667]should allow creating a VNC screenshot in PNG format", func() {
		// The default resolution is 720x400 for the vga/boch device used on amd64 and ppcl64,
		// while it is 1280x800 for the virtio device used on arm64 and s390x.
		size := image.Point{720, 400}
		if vmi.Spec.Architecture == "arm64" || vmi.Spec.Architecture == "s390x" {
			size = image.Point{1280, 800}
		}

		By("gathering screenshots until we are past the first boot screen and see the expected resolution")
		path := filepath.Join(GinkgoT().TempDir(), "screenshot.png")
		Eventually(func(g Gomega) image.Point {
			err := newRepeatableVirtctlCommand(
				"vnc", "screenshot",
				vmi.Name,
				"--namespace", vmi.Namespace,
				"--file="+path,
			)()
			g.Expect(err).ToNot(HaveOccurred())

			file, err := os.Open(path)
			g.Expect(err).ToNot(HaveOccurred())
			defer file.Close()

			img, err := png.Decode(file)
			g.Expect(err).ToNot(HaveOccurred())
			return img.Bounds().Size()
		}, 60*time.Second).Should(Equal(size), "screenshot.png should have the expected size")
	})

	DescribeTable("Dual connection. Validate default behavior and with 'force' option", func(force1, force2 bool) {
		const (
			portRangeFirst = 33333
			portRangeLast  = 33433
		)

		newVirtctlVNC := func(force bool) (string, *cobra.Command, context.CancelFunc) {
			// Define local proxy port
			portInt, found := findOpenPort(portRangeFirst, portRangeLast)
			Expect(found).To(BeTrue())
			port := strconv.FormatInt(int64(portInt), 10)
			forceStr := strconv.FormatBool(force)

			cmd := newVirtctlCommand("vnc",
				vmi.Name,
				"--namespace", vmi.Namespace,
				"--port", port,
				"--proxy-only",
				"--force="+forceStr,
			)
			connect := make(chan struct{})
			ctx := context.WithValue(cmd.Context(), connectedKeyVNC, connect)
			ctx, cancel := context.WithTimeout(ctx, proxyConnectTimeout)
			cmd.SetContext(ctx)
			return port, cmd, cancel
		}

		doProxyAsync := func(ctx context.Context, port string, success bool) (chan struct{}, chan struct{}) {
			proxy := make(chan struct{})
			go func() {
				defer GinkgoRecover()
				verifyProxyConnection(ctx, port, vmi.Name, success)
				proxy <- struct{}{}
			}()

			connected, ok := ctx.Value(connectedKeyVNC).(chan struct{})
			Expect(ok).To(BeTrue())
			return proxy, connected
		}

		By("session 01: virtctl to connect to remote VNC using --proxy-only")
		port1, cmd1, cancel1 := newVirtctlVNC(force1)
		DeferCleanup(cancel1)

		go func() {
			defer GinkgoRecover()
			err := cmd1.Execute()
			// First connection fails if Second connection succeeds
			Expect(err != nil).To(Equal(force2))
		}()

		By("session 01: vnc client to connect to proxy")
		proxy1, connected1 := doProxyAsync(cmd1.Context(), port1, true)
		// Wait connect
		select {
		case <-connected1:
		case <-cmd1.Context().Done():
			// Timeout
			Expect(cmd1.Context().Err()).ToNot(HaveOccurred())
		}

		By("session 02: virtctl to connect to remote VNC using --proxy-only")
		port2, cmd2, cancel2 := newVirtctlVNC(force2)
		DeferCleanup(cancel2)

		go func() {
			defer GinkgoRecover()
			err := cmd2.Execute()
			if force2 {
				Expect(err).ToNot(HaveOccurred())
			} else {
				Expect(err).To(HaveOccurred())
			}
		}()

		By("session 02: vnc client to connect to proxy")
		proxy2, connected2 := doProxyAsync(cmd2.Context(), port2, force2)
		// Wait connect or timeout
		select {
		case <-connected2:
		case <-cmd2.Context().Done():
			Expect(force2).To(BeFalse())
		}
		<-proxy1
		<-proxy2
	},
		Entry("Second session without force", false, false),
		Entry("Second session with force", false, true),
		// Force in the first session should not change the behavior
		Entry("First with force, Second session without force", true, false),
		Entry("First with force, Second session with force", true, true),
	)
}))

func verifyProxyConnection(
	ctx context.Context,
	port string,
	vmiName string,
	expectSuccess bool,
) {
	Eventually(func(g Gomega) {
		// In case the VNC connection fails, the caller will timeout
		// the context and we can break out of Eventually()
		if err := ctx.Err(); err != nil {
			return
		}
		conn, err := net.Dial("tcp", "127.0.0.1:"+port)
		g.Expect(err).ToNot(HaveOccurred())
		defer conn.Close()

		clientConn, err := vnc.Client(conn, &vnc.ClientConfig{
			ServerMessageCh: make(chan vnc.ServerMessage),
			ServerMessages:  []vnc.ServerMessage{new(vnc.FramebufferUpdateMessage)},
		})
		if expectSuccess {
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(clientConn.DesktopName).To(ContainSubstring(vmiName))
			defer clientConn.Close()
		} else {
			g.Expect(err).To(HaveOccurred())
		}

		// Let caller know VNC connect() phase is done
		if connected, ok := ctx.Value(connectedKeyVNC).(chan struct{}); ok {
			connected <- struct{}{}
		}
		<-ctx.Done()
	}, 60*time.Second).Should(Succeed())
}

func findOpenPort(start, end int) (int, bool) {
	Expect(end).To(BeNumerically(">=", start), "Start <= End")
	const host = "localhost"
	for port := start; port < end; port++ {
		addr := net.JoinHostPort(host, strconv.FormatInt(int64(port), 10))
		if conn, err := net.Listen("tcp", addr); err == nil {
			err = conn.Close()
			Expect(err).ToNot(HaveOccurred())
			return port, true
		}
	}
	return -1, false
}
