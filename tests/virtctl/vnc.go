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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

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

var _ = Describe(SIG("[sig-compute]VNC", decorators.SigCompute, decorators.WgArm64, func() {
	var vmi *v1.VirtualMachineInstance

	BeforeEach(func() {
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

		verifyProxyConnection(fmt.Sprintf("%v", result["port"]), vmi.Name)
	})

	It("[rfe_id:127][crit:medium][vendor:cnv-qe@redhat.com][level:component]"+
		"[test_id:5274]should connect to vnc with --proxy-only flag to the specified port", func() {
		const testPort = "33333"

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

		verifyProxyConnection(testPort, vmi.Name)
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
}))

func verifyProxyConnection(port, vmiName string) {
	Eventually(func(g Gomega) {
		conn, err := net.Dial("tcp", "127.0.0.1:"+port)
		g.Expect(err).ToNot(HaveOccurred())
		defer conn.Close()

		clientConn, err := vnc.Client(conn, &vnc.ClientConfig{
			ServerMessageCh: make(chan vnc.ServerMessage),
			ServerMessages:  []vnc.ServerMessage{new(vnc.FramebufferUpdateMessage)},
		})
		g.Expect(err).ToNot(HaveOccurred())
		defer clientConn.Close()

		g.Expect(clientConn.DesktopName).To(ContainSubstring(vmiName))
	}, 60*time.Second).Should(Succeed())
}
