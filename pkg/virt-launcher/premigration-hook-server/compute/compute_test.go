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
 */

package compute_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"libvirt.org/go/libvirtxml"

	v1 "kubevirt.io/api/core/v1"

	cmdv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"
	"kubevirt.io/kubevirt/pkg/virt-launcher/premigration-hook-server/compute"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Compute Hooks", func() {
	Context("CPU Dedicated Hook", func() {
		It("should change CPU pinning according to migration metadata", func() {
			domXML := `<domain type="kvm" id="1">
  <name>kubevirt</name>
  <vcpu placement="static">2</vcpu>
  <cputune>
    <vcpupin vcpu="0" cpuset="4"></vcpupin>
    <vcpupin vcpu="1" cpuset="5"></vcpupin>
  </cputune>
</domain>`
			expectedXML := `<domain type="kvm" id="1">
  <name>kubevirt</name>
  <vcpu placement="static">2</vcpu>
  <cputune>
    <vcpupin vcpu="0" cpuset="6"></vcpupin>
    <vcpupin vcpu="1" cpuset="7"></vcpupin>
  </cputune>
  <cpu>
    <topology sockets="1" cores="2" threads="1"></topology>
  </cpu>
</domain>`

			By("creating a VMI with dedicated CPU cores")
			vmi := &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kubevirt",
					Namespace: "testns",
				},
				Spec: v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{
						CPU: &v1.CPU{
							Cores:                 2,
							DedicatedCPUPlacement: true,
						},
					},
				},
			}

			By("making up a target topology")
			topology := &cmdv1.Topology{NumaCells: []*cmdv1.Cell{{
				Id: 0,
				Cpus: []*cmdv1.CPU{
					{
						Id:       6,
						Siblings: []uint32{6},
					},
					{
						Id:       7,
						Siblings: []uint32{7},
					},
				},
			}}}
			targetNodeTopology, err := json.Marshal(topology)
			Expect(err).NotTo(HaveOccurred(), "failed to marshall the topology")

			By("saving that topology in the migration state of the VMI")
			vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
				TargetNodeTopology: string(targetNodeTopology),
			}

			By("parsing the input domain XML")
			var domain libvirtxml.Domain
			err = domain.Unmarshal(domXML)
			Expect(err).NotTo(HaveOccurred(), "failed to parse input domain XML")

			By("running the CPU dedicated hook")
			hook := compute.NewCPUDedicatedHook(func() ([]int, error) { return []int{6, 7}, nil })
			hook(vmi, &domain)

			By("marshaling the modified domain back to XML")
			newXML, err := domain.Marshal()
			Expect(err).NotTo(HaveOccurred(), "failed to marshal modified domain")

			By("ensuring the generated XML is accurate")
			Expect(newXML).To(Equal(expectedXML), "the target XML is not as expected")
		})
	})
})
