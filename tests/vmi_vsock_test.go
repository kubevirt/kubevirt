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

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"

	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/tests/framework/checks"

	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
)

var _ = Describe("[Serial][sig-compute]VSOCK", func() {
	var virtClient kubecli.KubevirtClient
	var err error

	BeforeEach(func() {
		checks.SkipTestIfNoFeatureGate(virtconfig.VSOCKGate)
		virtClient, err = kubecli.GetKubevirtClient()
		Expect(err).ToNot(HaveOccurred())
	})

	Context("VM creation", func() {
		It("should expose a VSOCK device", func() {
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
		})
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
			tests.RunMigrationAndExpectCompletion(virtClient, migration, tests.MigrationWaitTime)

			domain2, err = tests.GetRunningVirtualMachineInstanceDomainXML(virtClient, vmi2)
			Expect(err).ToNot(HaveOccurred())
			domSpec2 = &api.DomainSpec{}
			Expect(xml.Unmarshal([]byte(domain2), domSpec2)).To(Succeed())
			Expect(domSpec2.Devices.VSOCK.CID.Auto).To(Equal("no"))
			Expect(domSpec2.Devices.VSOCK.CID.Address).To(Equal(*vmi2.Status.VSOCKCID))
		})
	})
})
