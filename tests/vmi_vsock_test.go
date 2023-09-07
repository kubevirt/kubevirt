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
 * Copyright 2022 Red Hat, Inc.
 *
 */

package tests_test

import (
	"encoding/xml"
	"net"
	"os"
	"time"

	"kubevirt.io/kubevirt/tests/libmigration"

	"kubevirt.io/kubevirt/tests/decorators"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/tests/libssh"

	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libvmi"

	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
)

var _ = Describe("[sig-compute]VSOCK", Serial, decorators.SigCompute, func() {
	var virtClient kubecli.KubevirtClient
	var err error

	BeforeEach(func() {
		tests.EnableFeatureGate(virtconfig.VSOCKGate)
		checks.SkipTestIfNoFeatureGate(virtconfig.VSOCKGate)
		virtClient = kubevirt.Client()
	})

	Context("VM creation", func() {
		DescribeTable("should expose a VSOCK device", func(useVirtioTransitional bool) {
			By("Creating a VMI with VSOCK enabled")
			vmi := tests.NewRandomFedoraVMI()
			vmi.Spec.Domain.Devices.UseVirtioTransitional = &useVirtioTransitional
			vmi.Spec.Domain.Devices.AutoattachVSOCK = pointer.Bool(true)
			vmi = tests.RunVMIAndExpectLaunch(vmi, 60)
			Expect(vmi.Status.VSOCKCID).NotTo(BeNil())

			By("creating valid libvirt domain")

			domain, err := tests.GetRunningVirtualMachineInstanceDomainXML(virtClient, vmi)
			Expect(err).ToNot(HaveOccurred())
			domSpec := &api.DomainSpec{}
			Expect(xml.Unmarshal([]byte(domain), domSpec)).To(Succeed())
			Expect(domSpec.Devices.VSOCK.CID.Auto).To(Equal("no"))
			Expect(domSpec.Devices.VSOCK.CID.Address).To(Equal(*vmi.Status.VSOCKCID))

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
										Key:      "kubernetes.io/hostname",
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

		It("should retain the CID for migration target", func() {
			By("Creating a VMI with VSOCK enabled")
			vmi := tests.NewRandomFedoraVMI()
			vmi.Spec.Domain.Devices.AutoattachVSOCK = pointer.Bool(true)
			vmi = tests.RunVMIAndExpectLaunch(vmi, 60)
			Expect(vmi.Status.VSOCKCID).NotTo(BeNil())

			By("creating valid libvirt domain")
			domain, err := tests.GetRunningVirtualMachineInstanceDomainXML(virtClient, vmi)
			Expect(err).ToNot(HaveOccurred())
			domSpec := &api.DomainSpec{}
			Expect(xml.Unmarshal([]byte(domain), domSpec)).To(Succeed())
			Expect(domSpec.Devices.VSOCK.CID.Auto).To(Equal("no"))
			Expect(domSpec.Devices.VSOCK.CID.Address).To(Equal(*vmi.Status.VSOCKCID))

			By("Creating a new VMI with VSOCK enabled on the same node")
			node := vmi.Status.NodeName
			vmi2 := tests.NewRandomFedoraVMI()
			vmi2.Spec.Domain.Devices.AutoattachVSOCK = pointer.Bool(true)
			vmi2.Spec.Affinity = affinity(node)
			vmi2 = tests.RunVMIAndExpectLaunch(vmi2, 60)
			Expect(vmi2.Status.VSOCKCID).NotTo(BeNil())

			By("creating valid libvirt domain")
			domain2, err := tests.GetRunningVirtualMachineInstanceDomainXML(virtClient, vmi2)
			Expect(err).ToNot(HaveOccurred())
			domSpec2 := &api.DomainSpec{}
			Expect(xml.Unmarshal([]byte(domain2), domSpec2)).To(Succeed())
			Expect(domSpec2.Devices.VSOCK.CID.Auto).To(Equal("no"))
			Expect(domSpec2.Devices.VSOCK.CID.Address).To(Equal(*vmi2.Status.VSOCKCID))

			By("Migrating the 2nd VMI")
			checks.SkipIfMigrationIsNotPossible()
			migration := tests.NewRandomMigration(vmi2.Name, vmi2.Namespace)
			libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

			domain2, err = tests.GetRunningVirtualMachineInstanceDomainXML(virtClient, vmi2)
			Expect(err).ToNot(HaveOccurred())
			domSpec2 = &api.DomainSpec{}
			Expect(xml.Unmarshal([]byte(domain2), domSpec2)).To(Succeed())
			Expect(domSpec2.Devices.VSOCK.CID.Auto).To(Equal("no"))
			Expect(domSpec2.Devices.VSOCK.CID.Address).To(Equal(*vmi2.Status.VSOCKCID))
		})
	})

	DescribeTable("communicating with VMI via VSOCK", func(useTLS bool) {
		if flags.KubeVirtExampleGuestAgentPath == "" {
			Skip("example guest agent path is not specified")
		}
		privateKeyPath, publicKey, err := libssh.GenerateKeyPair(GinkgoT().TempDir())
		Expect(err).ToNot(HaveOccurred())
		userData := libssh.RenderUserDataWithKey(publicKey)
		vmi := libvmi.NewFedora(
			libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
			libvmi.WithCloudInitNoCloudUserData(userData, false),
		)
		vmi.Spec.Domain.Devices.AutoattachVSOCK = pointer.Bool(true)
		vmi = tests.RunVMIAndExpectLaunch(vmi, 60)

		By("Logging in as root")
		err = console.LoginToFedora(vmi)
		Expect(err).ToNot(HaveOccurred())

		By("copying the guest agent binary")
		Expect(os.Setenv("SSH_AUTH_SOCK", "/dev/null")).To(Succeed())
		Expect(libssh.SCPToVMI(vmi, privateKeyPath, flags.KubeVirtExampleGuestAgentPath, "/usr/bin/")).To(Succeed())

		By("starting the guest agent binary")
		Expect(tests.StartExampleGuestAgent(vmi, useTLS, 1234)).To(Succeed())
		time.Sleep(2 * time.Second)

		virtClient := kubevirt.Client()

		By("Connect to the guest via API")
		cliConn, svrConn := net.Pipe()
		defer func() {
			_ = cliConn.Close()
			_ = svrConn.Close()
		}()
		stopChan := make(chan error)
		go func() {
			defer GinkgoRecover()
			vsock, err := virtClient.VirtualMachineInstance(vmi.Namespace).VSOCK(vmi.Name, &v1.VSOCKOptions{TargetPort: uint32(1234), UseTLS: pointer.Bool(useTLS)})
			if err != nil {
				stopChan <- err
				return
			}
			stopChan <- vsock.Stream(kubecli.StreamOptions{
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
		Entry("should succeed with TLS on both sides", true),
		Entry("should succeed without TLS on both sides", false),
	)

	It("should return err if the port is invalid", func() {
		virtClient := kubevirt.Client()

		By("Creating a VMI with VSOCK enabled")
		vmi := tests.NewRandomFedoraVMI()
		vmi.Spec.Domain.Devices.AutoattachVSOCK = pointer.Bool(true)
		vmi = tests.RunVMIAndExpectLaunch(vmi, 60)

		By("Connect to the guest on invalide port")
		_, err = virtClient.VirtualMachineInstance(vmi.Namespace).VSOCK(vmi.Name, &v1.VSOCKOptions{TargetPort: uint32(0)})
		Expect(err).To(HaveOccurred())
	})

	It("should return err if no app listerns on the port", func() {
		virtClient := kubevirt.Client()

		By("Creating a VMI with VSOCK enabled")
		vmi := tests.NewRandomFedoraVMI()
		vmi.Spec.Domain.Devices.AutoattachVSOCK = pointer.Bool(true)
		vmi = tests.RunVMIAndExpectLaunch(vmi, 60)

		By("Connect to the guest on the unused port")
		cliConn, svrConn := net.Pipe()
		defer func() {
			_ = cliConn.Close()
			_ = svrConn.Close()
		}()
		vsock, err := virtClient.VirtualMachineInstance(vmi.Namespace).VSOCK(vmi.Name, &v1.VSOCKOptions{TargetPort: uint32(9999)})
		Expect(err).NotTo(HaveOccurred())
		Expect(vsock.Stream(kubecli.StreamOptions{
			In:  svrConn,
			Out: svrConn,
		})).NotTo(Succeed())
	})
})
