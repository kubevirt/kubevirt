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
	"crypto/sha256"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "kubevirt.io/api/core/v1"

	kvcorev1 "kubevirt.io/client-go/kubevirt/typed/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"

	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libmigration"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
)

var _ = Describe("[sig-compute]VSOCK", Serial, decorators.SigCompute, decorators.VSOCK, func() {
	const guestAgentPort = 1234

	BeforeEach(func() {
		config.EnableFeatureGate(featuregate.VSOCKGate)
	})

	Context("VM creation", func() {
		DescribeTable("should expose a VSOCK device", func(useVirtioTransitional bool) {
			By("Creating a VMI with VSOCK enabled")
			vmi := libvmifact.NewFedora(libnet.WithMasqueradeNetworking())
			vmi.Spec.Domain.Devices.UseVirtioTransitional = &useVirtioTransitional
			vmi.Spec.Domain.Devices.AutoattachVSOCK = pointer.P(true)
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, flags.StartupTimeoutSecondsSmall())
			Expect(vmi.Status.VSOCKCID).NotTo(BeNil())

			By("Logging in as root")
			Expect(console.LoginToFedora(vmi)).To(Succeed())

			By("Ensuring a vsock device is present")
			Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: "ls /dev/vsock-vhost\n"},
				&expect.BExp{R: "/dev/vsock-vhost"},
			}, 300)).To(Succeed(), "Could not find a vsock-vhost device")
			Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: "ls /dev/vsock\n"},
				&expect.BExp{R: "/dev/vsock"},
			}, 300)).To(Succeed(), "Could not find a vsock device")
		},
			Entry("Use virtio transitional", true),
			Entry("Use virtio non-transitional", false),
		)
	})

	It("should retain the CID for migration target", decorators.RequiresTwoSchedulableNodes, func() {
		By("Creating a VMI with VSOCK enabled")
		vmi := libvmifact.NewFedora(libnet.WithMasqueradeNetworking())
		vmi.Spec.Domain.Devices.AutoattachVSOCK = pointer.P(true)
		vmi = libvmops.RunVMIAndExpectLaunch(vmi, flags.StartupTimeoutSecondsSmall())
		Expect(vmi.Status.VSOCKCID).NotTo(BeNil())
		cid := *vmi.Status.VSOCKCID

		By("Migrating the VMI")
		migration := libmigration.New(vmi.Name, vmi.Namespace)
		libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(kubevirt.Client(), migration)

		By("Ensuring the CID survived the migration")
		vmi, err := matcher.ThisVMI(vmi)()
		Expect(err).ToNot(HaveOccurred())
		Expect(vmi.Status.VSOCKCID).To(HaveValue(Equal(cid)))
	})

	DescribeTable("communicating with VMI via VSOCK", func(useTLS bool) {
		vmi := libvmifact.NewFedora(
			libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
		)
		vmi.Spec.Domain.Devices.AutoattachVSOCK = pointer.P(true)
		vmi = libvmops.RunVMIAndExpectLaunch(vmi, flags.StartupTimeoutSecondsSmall())

		By("Logging in as root")
		Expect(console.LoginToFedora(vmi)).To(Succeed())

		By("copying the guest agent binary")
		copyExampleGuestAgent(vmi)

		By("starting the guest agent binary")
		Expect(startExampleGuestAgent(vmi, useTLS, guestAgentPort)).To(Succeed())

		expectVSOCKEchoViaAPI(vmi, guestAgentPort, useTLS)
	},
		Entry("should succeed with TLS on both sides", true),
		Entry("should succeed without TLS on both sides", false),
	)

	It("should return err if the port is invalid", func() {
		By("Creating a VMI with VSOCK enabled")
		vmi := libvmifact.NewFedora(libnet.WithMasqueradeNetworking())
		vmi.Spec.Domain.Devices.AutoattachVSOCK = pointer.P(true)
		vmi = libvmops.RunVMIAndExpectLaunch(vmi, flags.StartupTimeoutSecondsSmall())

		By("Connect to the guest on invalid port")
		_, err := kubevirt.Client().VirtualMachineInstance(vmi.Namespace).VSOCK(
			vmi.Name, &v1.VSOCKOptions{TargetPort: uint32(0)})
		Expect(err).To(HaveOccurred())
	})

	It("should return err if no app listeners on the port", func() {
		By("Creating a VMI with VSOCK enabled")
		vmi := libvmifact.NewFedora(libnet.WithMasqueradeNetworking())
		vmi.Spec.Domain.Devices.AutoattachVSOCK = pointer.P(true)
		vmi = libvmops.RunVMIAndExpectLaunch(vmi, flags.StartupTimeoutSecondsSmall())

		By("Connect to the guest on the unused port")
		cliConn, svrConn := net.Pipe()
		defer func() {
			_ = cliConn.Close()
			_ = svrConn.Close()
		}()
		vsockStream, err := kubevirt.Client().VirtualMachineInstance(vmi.Namespace).VSOCK(
			vmi.Name, &v1.VSOCKOptions{TargetPort: uint32(9999)})
		Expect(err).NotTo(HaveOccurred())
		Expect(vsockStream.Stream(kvcorev1.StreamOptions{
			In:  svrConn,
			Out: svrConn,
		})).NotTo(Succeed())
	})
})

// expectVSOCKEchoViaAPI retries because the agent binds its VSOCK port only
// after the shell backgrounded it.
func expectVSOCKEchoViaAPI(vmi *v1.VirtualMachineInstance, port uint32, useTLS bool) {
	GinkgoHelper()

	By("Echoing a message off the guest via the API")
	Eventually(func() error {
		return vsockEchoViaAPI(vmi, port, useTLS)
	}, 60*time.Second, time.Second).Should(Succeed())
}

func vsockEchoViaAPI(vmi *v1.VirtualMachineInstance, port uint32, useTLS bool) error {
	vsockStream, err := kubevirt.Client().VirtualMachineInstance(vmi.Namespace).VSOCK(
		vmi.Name, &v1.VSOCKOptions{TargetPort: port, UseTLS: pointer.P(useTLS)})
	if err != nil {
		return err
	}

	cliConn, svrConn := net.Pipe()
	defer func() {
		_ = cliConn.Close()
		_ = svrConn.Close()
	}()

	streamErr := make(chan error, 1)
	go func() {
		streamErr <- vsockStream.Stream(kvcorev1.StreamOptions{
			In:  svrConn,
			Out: svrConn,
		})
	}()

	if err = cliConn.SetDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return err
	}

	const message = "Hello World?"
	if _, err = cliConn.Write([]byte(message)); err != nil {
		return fmt.Errorf("failed to write to the guest: %w", err)
	}

	buf := make([]byte, 1024)
	n, err := cliConn.Read(buf)
	if err != nil {
		return fmt.Errorf("failed to read from the guest: %w", err)
	}
	if got := string(buf[:n]); got != message {
		return fmt.Errorf("the guest echoed %q, expected %q", got, message)
	}

	select {
	case err = <-streamErr:
		return fmt.Errorf("the stream must still be open, but it ended with: %w", err)
	default:
	}

	return nil
}

func copyExampleGuestAgent(vmi *v1.VirtualMachineInstance) {
	const (
		port           = 4444
		guestAgentPath = "/usr/bin/example-guest-agent"
	)

	if flags.KubeVirtExampleGuestAgentPath == "" {
		Fail(`"example-guest-agent-path" argument is not specified`, 1)
	}
	err := console.RunCommand(vmi, fmt.Sprintf("nc -l %d > %s < /dev/null 2>/dev/null &", port, guestAgentPath), 60*time.Second)
	Expect(err).ToNot(HaveOccurred())

	file, err := os.Open(flags.KubeVirtExampleGuestAgentPath)
	Expect(err).ToNot(HaveOccurred())
	defer file.Close()

	var stream kvcorev1.StreamInterface
	Eventually(func() error {
		stream, err = kubevirt.Client().VirtualMachineInstance(vmi.Namespace).PortForward(vmi.Name, port, "tcp")
		return err
	}, 60*time.Second, 1*time.Second).Should(Succeed())

	conn := stream.AsConn()
	sha256Hasher := sha256.New()
	_, err = io.Copy(conn, io.TeeReader(file, sha256Hasher))
	Expect(err).ToNot(HaveOccurred())
	err = conn.Close()
	Expect(err).ToNot(HaveOccurred())

	expectedSHA256 := fmt.Sprintf("%x", sha256Hasher.Sum(nil))
	guestAgentSHA256Command := fmt.Sprintf("sha256sum %s | awk '{print $1}'", guestAgentPath)
	Eventually(func() error {
		guestSHA256Output, err := console.RunCommandAndStoreOutput(vmi, guestAgentSHA256Command, 30*time.Second)
		if err != nil {
			return err
		}
		guestSHA256 := strings.TrimSpace(guestSHA256Output)
		if guestSHA256 != expectedSHA256 {
			return fmt.Errorf("guest agent sha256 mismatch: got %q, expected %q", guestSHA256, expectedSHA256)
		}
		return nil
	}, 2*time.Minute, 10*time.Second).Should(Succeed(), "should validate the guest agent file was copied correctly")
}

func startExampleGuestAgent(vmi *v1.VirtualMachineInstance, useTLS bool, port uint32) error {
	serverArgs := fmt.Sprintf("--port %v", port)
	if useTLS {
		serverArgs = strings.Join([]string{serverArgs, "--use-tls"}, " ")
	}

	return console.SafeExpectBatch(vmi, []expect.Batcher{
		&expect.BSnd{S: "chmod +x /usr/bin/example-guest-agent\n"},
		&expect.BExp{R: ""},
		&expect.BSnd{S: console.EchoLastReturnValue},
		&expect.BExp{R: console.ShellSuccess},
		&expect.BSnd{S: fmt.Sprintf("/usr/bin/example-guest-agent %s 2>&1 &\n", serverArgs)},
		&expect.BExp{R: ""},
		&expect.BSnd{S: console.EchoLastReturnValue},
		&expect.BExp{R: console.ShellSuccess},
	}, 60)
}
