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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kubevirt.io/client-go/kubecli"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/ephemeral-disk/fake"
	"kubevirt.io/kubevirt/pkg/hypervisor"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/pkg/os/disk"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter"
	archconverter "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/arch"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libkubevirt"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe("[HyperVLayered] HyperVLayered integration tests", decorators.HyperVLayered, func() {
	var virtClient kubecli.KubevirtClient
	var vmi *v1.VirtualMachineInstance
	var hypervisorConfig *v1.HypervisorConfiguration

	BeforeEach(func() {
		virtClient = kubevirt.Client()
		vmi = libvmifact.NewFedora()

		kv := libkubevirt.GetCurrentKv(virtClient)

		clusterConfig, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&kv.Spec.Configuration)

		hypervisorConfig = clusterConfig.GetHypervisor()

		if hypervisorConfig.Name != v1.HyperVLayeredHypervisorName {
			Skip(fmt.Sprintf(
				"Skipping HyperVLayered integration tests: hypervisor.Name=%q (need %q)",
				hypervisorConfig.Name, v1.HyperVLayeredHypervisorName,
			))
		}
	})

	Context("VMI created with HyperVLayered", func() {
		It("should request 'devices.kubevirt.io/mshv' instead of 'devices.kubevirt.io/kvm' in virt-launcher pod spec", func() {
			vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			// Wait for VMI to be running
			vmi = libwait.WaitForSuccessfulVMIStart(vmi)

			// Ensure the VMI is fully booted
			Expect(console.LoginToFedora(vmi)).To(Succeed())


			// Get the virt-launcher pod
			// Check the compute container resources
			computeContainer, err := libpod.LookupComputeContainerFromVmi(vmi)
			Expect(err).ToNot(HaveOccurred())

			Expect(computeContainer.Resources.Limits).To(HaveKey(k8sv1.ResourceName(services.HyperVDevice)),
				"virt-launcher pod should request 'devices.kubevirt.io/mshv' when hyperv-layered hypervisor is used")
			Expect(computeContainer.Resources.Limits).ToNot(HaveKey(k8sv1.ResourceName(services.KvmDevice)),
				"virt-launcher pod should NOT request 'devices.kubevirt.io/kvm' when hyperv-layered hypervisor is used")
			Expect(computeContainer.Resources.Limits[k8sv1.ResourceName(services.HyperVDevice)]).To(Equal(resource.MustParse("1")),
				"virt-launcher pod should request 1 'devices.kubevirt.io/mshv' when hyperv-layered hypervisor is used")
		})

		It("should generate libvirt domain xml with hyperv domain type", func() {
			domain := &api.Domain{}
			c := &converter.ConverterContext{
				Architecture:         archconverter.NewConverter("amd64"),
				Hypervisor:           hypervisor.NewHypervisor(hypervisorConfig.Name),
				EphemeraldiskCreator: &fake.MockEphemeralDiskImageCreator{},
				AllowEmulation:       true,
			}

			c.DisksInfo = map[string]*disk.DiskInfo{}
			for _, vol := range vmi.Spec.Volumes {
				c.DisksInfo[vol.Name] = &disk.DiskInfo{}
			}

			err := converter.Convert_v1_VirtualMachineInstance_To_api_Domain(vmi, domain, c)
			Expect(err).ToNot(HaveOccurred())
			Expect(domain.Spec.Type).To(Equal("hyperv"), "libvirt XML domain type should be 'hyperv' when hyperv-layered hypervisor is used")
		})

	})

})
