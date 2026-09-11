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

package compute

import (
	"fmt"
	"strings"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"

	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libdomain"
	"kubevirt.io/kubevirt/tests/libkubevirt"
	kvconfig "kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
)

var _ = Describe(SIG("IOMMUFD", Serial, decorators.Kubernetes137, func() {
	BeforeEach(func() {
		kvconfig.EnableFeatureGate(featuregate.HostDevicesGate, featuregate.IOMMUFDGate)
	})

	Context("with ephemeral disk", func() {
		// Emulated sound cards provided by kubevirtci and bound to vfio-pci by gocli.
		hostSoundCards := []string{
			"8086:2668", // Intel HD Audio Controller (ich6)
			"8086:293e", // Intel HD Audio Controller (ich9)
		}

		It("Should successfully passthrough emulated PCI devices with IOMMUFD", func() {
			deviceName := "example.org/soundcard-iommufd"

			By("Adding the emulated sound cards to the permitted host devices")
			virtClient := kubevirt.Client()
			kv := libkubevirt.GetCurrentKv(virtClient)
			config := kv.Spec.Configuration
			config.PermittedHostDevices = &v1.PermittedHostDevices{}
			var hostDevs []v1.HostDevice
			for i, id := range hostSoundCards {
				config.PermittedHostDevices.PciHostDevices = append(config.PermittedHostDevices.PciHostDevices, v1.PciHostDevice{
					PCIVendorSelector: id,
					ResourceName:      deviceName,
				})
				hostDevs = append(hostDevs, v1.HostDevice{
					Name:       fmt.Sprintf("sound%d", i),
					DeviceName: deviceName,
				})
			}
			kvconfig.UpdateKubeVirtConfigValueAndWait(config)

			By("Creating a Fedora VMI with the sound cards as host devices")
			vmi := libvmifact.NewFedora(
				libnet.WithMasqueradeNetworking(),
				libvmi.WithHostDevice(hostDevs[0]),
				libvmi.WithHostDevice(hostDevs[1]),
			)
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, flags.VMIStartupTimeout())

			By("Verifying the launcher pod has the IOMMUFD device resource")
			compute, err := libpod.LookupComputeContainerFromVmi(vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(compute.Resources.Limits).To(HaveKeyWithValue(k8sv1.ResourceName(services.IOMMUFDDevice), resource.MustParse("1")))

			By("Verifying the domain XML contains IOMMUFD configuration")
			domSpec, err := libdomain.GetRunningVMIDomainSpec(vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(domSpec.IOMMUFD).ToNot(BeNil(), "IOMMUFD element should be present in domain XML")
			Expect(domSpec.IOMMUFD.Enabled).To(Equal("yes"))
			Expect(domSpec.IOMMUFD.FDGroup).To(Equal("iommu"))

			By("Making sure the sound cards are present inside the VMI")
			Expect(console.LoginToFedora(vmi)).To(Succeed())
			for _, id := range hostSoundCards {
				Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: "grep -c " + strings.Replace(id, ":", "", 1) + " /proc/bus/pci/devices\n"},
					&expect.BExp{R: console.RetValue("1")},
				}, 15)).To(Succeed(), "Device not found")
			}
		})
	})
}))
