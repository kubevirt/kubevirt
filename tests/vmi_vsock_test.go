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
	"crypto/md5"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/client-go/kubecli"
	kvcorev1 "kubevirt.io/client-go/kubevirt/typed/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
	"kubevirt.io/kubevirt/pkg/vsock"

	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libdomain"
	"kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libmigration"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
)

var _ = Describe("[sig-compute]VSOCK", Serial, decorators.SigCompute, decorators.VSOCK, func() {
	var virtClient kubecli.KubevirtClient
	var err error

	BeforeEach(func() {
		config.EnableFeatureGate(featuregate.VSOCKGate)
		virtClient = kubevirt.Client()
	})

	Context("VM creation", func() {
		DescribeTable("should expose a VSOCK device", func(useVirtioTransitional bool) {
			By("Creating a VMI with VSOCK enabled")
			vmi := libvmifact.NewFedora(libnet.WithMasqueradeNetworking())
			vmi.Spec.Domain.Devices.UseVirtioTransitional = &useVirtioTransitional
			vmi.Spec.Domain.Devices.AutoattachVSOCK = pointer.P(true)
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, libvmops.StartupTimeoutSecondsSmall)
			Expect(vmi.Status.VSOCKCID).NotTo(BeNil())

			By("creating valid libvirt domain")

			domSpec, err := libdomain.GetRunningVMIDomainSpec(vmi)
			Expect(err).ToNot(HaveOccurred())

			nsMode, err := getVsockNsMode(vmi)
			Expect(err).ToNot(HaveOccurred())

			Expect(domSpec.Devices.VSOCK.CID.Auto).To(Equal("no"))

			expectedCID := *vmi.Status.VSOCKCID
			if nsMode == vsock.ModeLocal {
				expectedCID = uint32(vsock.LocalCID)
			}
			Expect(domSpec.Devices.VSOCK.CID.Address).To(Equal(expectedCID))

			By("Logging in as root")
			err = console.LoginToFedora(vmi)
			Expect(err).ToNot(HaveOccurred())

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

	Context("Live migration", func() {
		affinity := func(nodeName string) *k8sv1.Affinity {
			return &k8sv1.Affinity{
				NodeAffinity: &k8sv1.NodeAffinity{
					PreferredDuringSchedulingIgnoredDuringExecution: []k8sv1.PreferredSchedulingTerm{
						{
							Preference: k8sv1.NodeSelectorTerm{
								MatchExpressions: []k8sv1.NodeSelectorRequirement{
									{
										Key:      k8sv1.LabelHostname,
										Operator: k8sv1.NodeSelectorOpIn,
										Values:   []string{nodeName},
									},
								},
							},
							Weight: 1,
						},
					},
				},
			}
		}

		It("should retain the CID for migration target", decorators.RequiresTwoSchedulableNodes, func() {
			By("Creating a VMI with VSOCK enabled")
			vmi := libvmifact.NewFedora(libnet.WithMasqueradeNetworking())
			vmi.Spec.Domain.Devices.AutoattachVSOCK = pointer.P(true)
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, libvmops.StartupTimeoutSecondsSmall)
			Expect(vmi.Status.VSOCKCID).NotTo(BeNil())

			By("creating valid libvirt domain")
			domSpec, err := libdomain.GetRunningVMIDomainSpec(vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(domSpec.Devices.VSOCK.CID.Auto).To(Equal("no"))
			Expect(domSpec.Devices.VSOCK.CID.Address).To(Equal(*vmi.Status.VSOCKCID))

			By("Creating a new VMI with VSOCK enabled on the same node")
			node := vmi.Status.NodeName
			vmi2 := libvmifact.NewFedora(libnet.WithMasqueradeNetworking())
			vmi2.Spec.Domain.Devices.AutoattachVSOCK = pointer.P(true)
			vmi2.Spec.Affinity = affinity(node)
			vmi2 = libvmops.RunVMIAndExpectLaunch(vmi2, libvmops.StartupTimeoutSecondsSmall)
			Expect(vmi2.Status.VSOCKCID).NotTo(BeNil())

			By("creating valid libvirt domain")
			domSpec2, err := libdomain.GetRunningVMIDomainSpec(vmi2)
			Expect(err).ToNot(HaveOccurred())

			Expect(domSpec2.Devices.VSOCK.CID.Auto).To(Equal("no"))
			Expect(domSpec2.Devices.VSOCK.CID.Address).To(Equal(*vmi2.Status.VSOCKCID))

			By("Migrating the 2nd VMI")
			migration := libmigration.New(vmi2.Name, vmi2.Namespace)
			libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

			domSpec2, err = libdomain.GetRunningVMIDomainSpec(vmi2)
			Expect(err).ToNot(HaveOccurred())

			Expect(domSpec2.Devices.VSOCK.CID.Auto).To(Equal("no"))
			Expect(domSpec2.Devices.VSOCK.CID.Address).To(Equal(*vmi2.Status.VSOCKCID))
		})
	})

	DescribeTable("communicating with VMI via VSOCK", func(useTLS bool) {
		if flags.KubeVirtExampleGuestAgentPath == "" {
			Fail(`"example-guest-agent-path" argument is not specified`)
		}

		vmi := libvmifact.NewFedora(
			libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
		)
		vmi.Spec.Domain.Devices.AutoattachVSOCK = pointer.P(true)
		vmi = libvmops.RunVMIAndExpectLaunch(vmi, libvmops.StartupTimeoutSecondsSmall)

		By("Logging in as root")
		err = console.LoginToFedora(vmi)
		Expect(err).ToNot(HaveOccurred())

		By("copying the guest agent binary")
		copyExampleGuestAgent(vmi)

		By("starting the guest agent binary")
		Expect(startExampleGuestAgent(vmi, useTLS, 1234)).To(Succeed())
		time.Sleep(2 * time.Second)

		By("Connect to the guest via API")
		cliConn, svrConn := net.Pipe()
		defer func() {
			_ = cliConn.Close()
			_ = svrConn.Close()
		}()
		stopChan := make(chan error)
		go func() {
			defer GinkgoRecover()
			vsock, err := kubevirt.Client().VirtualMachineInstance(vmi.Namespace).VSOCK(vmi.Name, &v1.VSOCKOptions{TargetPort: uint32(1234), UseTLS: pointer.P(useTLS)})
			if err != nil {
				stopChan <- err
				return
			}
			stopChan <- vsock.Stream(kvcorev1.StreamOptions{
				In:  svrConn,
				Out: svrConn,
			})
		}()

		Expect(cliConn.SetDeadline(time.Now().Add(10 * time.Second))).To(Succeed())

		By("Writing to the Guest")
		message := "Hello World?"
		_, err = cliConn.Write([]byte(message))
		Expect(err).NotTo(HaveOccurred())

		By("Reading from the Guest")
		buf := make([]byte, 1024, 1024)
		n, err := cliConn.Read(buf)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(buf[0:n])).To(Equal(message))

		select {
		case err := <-stopChan:
			Expect(err).NotTo(HaveOccurred())
		default:
		}
	},
		// TODO: The TLS handshake will fail when using local namespace,
		//   because the certificate server is listening in global namespace.
		//   This will be fixed in a future commit. See "Change 4" in the VEP:
		//   https://github.com/kubevirt/enhancements/blob/main/veps/sig-compute/222-vsock-netns-vep/vsock-netns-vep.md#change-4-on-demand-vsock-ca-service
		Entry("should succeed with TLS on both sides", true),
		Entry("should succeed without TLS on both sides", false),
	)

	It("should return err if the port is invalid", func() {
		virtClient := kubevirt.Client()

		By("Creating a VMI with VSOCK enabled")
		vmi := libvmifact.NewFedora(libnet.WithMasqueradeNetworking())
		vmi.Spec.Domain.Devices.AutoattachVSOCK = pointer.P(true)
		vmi = libvmops.RunVMIAndExpectLaunch(vmi, libvmops.StartupTimeoutSecondsSmall)

		By("Connect to the guest on invalid port")
		_, err = virtClient.VirtualMachineInstance(vmi.Namespace).VSOCK(vmi.Name, &v1.VSOCKOptions{TargetPort: uint32(0)})
		Expect(err).To(HaveOccurred())
	})

	It("should return err if no app listerns on the port", func() {
		virtClient := kubevirt.Client()

		By("Creating a VMI with VSOCK enabled")
		vmi := libvmifact.NewFedora(libnet.WithMasqueradeNetworking())
		vmi.Spec.Domain.Devices.AutoattachVSOCK = pointer.P(true)
		vmi = libvmops.RunVMIAndExpectLaunch(vmi, libvmops.StartupTimeoutSecondsSmall)

		By("Connect to the guest on the unused port")
		cliConn, svrConn := net.Pipe()
		defer func() {
			_ = cliConn.Close()
			_ = svrConn.Close()
		}()
		vsock, err := virtClient.VirtualMachineInstance(vmi.Namespace).VSOCK(vmi.Name, &v1.VSOCKOptions{TargetPort: uint32(9999)})
		Expect(err).NotTo(HaveOccurred())
		Expect(vsock.Stream(kvcorev1.StreamOptions{
			In:  svrConn,
			Out: svrConn,
		})).NotTo(Succeed())
	})
})

func copyExampleGuestAgent(vmi *v1.VirtualMachineInstance) {
	const (
		port           = 4444
		guestAgentPath = "/usr/bin/example-guest-agent"
	)

	err := console.RunCommand(vmi, fmt.Sprintf("nc -vl %d > %s < /dev/null &", port, guestAgentPath), 60*time.Second)
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
	md5Hasher := md5.New()
	_, err = io.Copy(conn, io.TeeReader(file, md5Hasher))
	Expect(err).ToNot(HaveOccurred())
	err = conn.Close()
	Expect(err).ToNot(HaveOccurred())

	expectedMD5 := fmt.Sprintf("%x", md5Hasher.Sum(nil))
	guestAgentMD5Command := fmt.Sprintf("md5sum %s | awk '{print $1}'", guestAgentPath)
	Eventually(func() error {
		guestMD5Output, err := console.RunCommandAndStoreOutput(vmi, guestAgentMD5Command, 30*time.Second)
		if err != nil {
			return err
		}
		guestMD5 := strings.TrimSpace(guestMD5Output)
		if guestMD5 != expectedMD5 {
			return fmt.Errorf("guest agent md5 mismatch: got %q, expected %q", guestMD5, expectedMD5)
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

func getVsockNsMode(vmi *v1.VirtualMachineInstance) (string, error) {
	vmiPod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
	if err != nil {
		return "", err
	}

	stdout, stderr, err := exec.ExecuteCommandOnPodWithResults(
		vmiPod,
		libpod.LookupComputeContainer(vmiPod).Name,
		[]string{"cat", vsock.NsModePath},
	)
	if err != nil {
		return "", fmt.Errorf("could not read %s (remotely on pod %s): %v: %s, %s", vsock.NsModePath, vmiPod.Name, err, stdout, stderr)
	}
	return strings.TrimSpace(stdout), nil
}
