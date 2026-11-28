/*
This file is part of the KubeVirt project

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

Copyright The KubeVirt Authors.
*/

package instancetype_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	virtv1 "kubevirt.io/api/core/v1"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
)

func (s *instancetypeSuite) describeApplication() {
	DescribeTableSubtree(
		"Instancetype and preference application", func(scenario string) {
			var vm *virtv1.VirtualMachine
			var it *instancetypev1beta1.VirtualMachineInstancetype
			var pref *instancetypev1beta1.VirtualMachinePreference

			BeforeEach(func(ctx SpecContext) {
				it = newInstancetype()
				it.Spec.Annotations = map[string]string{"required-annotation-1": "1", "required-annotation-2": "2"}
				pref = newPreference()
				pref.Spec.Devices = &instancetypev1beta1.DevicePreferences{
					PreferredDiskBus: virtv1.DiskBusSATA, PreferredInterfaceModel: virtv1.VirtIO,
				}
				pref.Spec.Features = &instancetypev1beta1.FeaturePreferences{
					PreferredHyperv: &virtv1.FeatureHyperv{
						VAPIC:   &virtv1.FeatureState{Enabled: pointer.P(true)},
						Relaxed: &virtv1.FeatureState{Enabled: pointer.P(true)},
					},
				}
				pref.Spec.Firmware = &instancetypev1beta1.FirmwarePreferences{PreferredUseBios: pointer.P(true)}
				pref.Spec.PreferredTerminationGracePeriodSeconds = pointer.P(int64(15))
				pref.Spec.PreferredSubdomain = pointer.P("non-existent-subdomain")
				pref.Spec.Annotations = map[string]string{
					"preferred-annotation-1": "1", "preferred-annotation-2": "use-vm-value",
					"required-annotation-1": "use-instancetype-value",
				}
				vmi := libvmi.New(libvmi.WithMemoryRequest("128Mi"),
					libvmi.WithEmptyDisk("empty", virtv1.DiskBusVirtio, resource.MustParse("1Mi")))
				var options []libvmi.VMOption
				if scenario == "namespaced" {
					var err error
					it, err = s.f.VirtClient().VirtualMachineInstancetype(s.namespace).Create(ctx, it, metav1.CreateOptions{})
					Expect(err).NotTo(HaveOccurred())
					pref, err = s.f.VirtClient().VirtualMachinePreference(s.namespace).Create(ctx, pref, metav1.CreateOptions{})
					Expect(err).NotTo(HaveOccurred())
					options = []libvmi.VMOption{libvmi.WithInstancetype(it.Name), libvmi.WithPreference(pref.Name)}
				} else {
					clusterPref := createTestClusterPreference(ctx, s.f, &instancetypev1beta1.VirtualMachineClusterPreference{
						ObjectMeta: metav1.ObjectMeta{GenerateName: "clusterpreference-"}, Spec: pref.Spec,
					})
					options = []libvmi.VMOption{libvmi.WithClusterPreference(clusterPref.Name)}
					if scenario == "cluster" {
						clusterIT := createTestClusterInstancetype(ctx, s.f, &instancetypev1beta1.VirtualMachineClusterInstancetype{
							ObjectMeta: metav1.ObjectMeta{GenerateName: "clusterinstancetype-"}, Spec: it.Spec,
						})
						options = append(options, libvmi.WithClusterInstancetype(clusterIT.Name))
					}
				}
				options = append(options, libvmi.WithRunStrategy(virtv1.RunStrategyAlways))
				vm = libvmi.NewVirtualMachine(vmi, options...)
				vm.Spec.Template.ObjectMeta.Annotations = map[string]string{"preferred-annotation-2": "2"}
				if scenario == "disks" {
					vm.Spec.Template.Spec.Domain.Devices.Disks = nil
				}
				var err error
				vm, err = s.f.VirtClient().VirtualMachine(s.namespace).Create(ctx, vm, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())
			})

			It("should apply the selected preferences to the VMI", func(ctx SpecContext) {
				vmi := waitForScheduledVMI(ctx, s.f, vm)
				switch scenario {
				case "cluster":
					Expect(vm.Spec.Instancetype.Kind).To(BeEmpty())
					Expect(vm.Spec.Preference.Kind).To(BeEmpty())
					Expect(vmi.Spec.Domain.CPU.Sockets).To(Equal(it.Spec.CPU.Guest))
					Expect(*vmi.Spec.Domain.Memory.Guest).To(Equal(it.Spec.Memory.Guest))
				case "interface":
					Expect(vmi.Spec.Domain.Devices.Interfaces).To(HaveLen(1))
					Expect(vmi.Spec.Domain.Devices.Interfaces[0].Model).To(Equal(pref.Spec.Devices.PreferredInterfaceModel))
				case "disks":
					Expect(vmi.Spec.Domain.Devices.Disks).To(HaveLen(1))
					Expect(vmi.Spec.Domain.Devices.Disks[0].Disk.Bus).To(Equal(pref.Spec.Devices.PreferredDiskBus))
				case "namespaced":
					Expect(vmi.Spec.Domain.CPU.Sockets).To(Equal(it.Spec.CPU.Guest))
					Expect(*vmi.Spec.Domain.Memory.Guest).To(Equal(it.Spec.Memory.Guest))
					Expect(vmi.Spec.Domain.Devices.Disks).To(HaveLen(1))
					Expect(vmi.Spec.Domain.Devices.Disks[0].Disk.Bus).To(Equal(pref.Spec.Devices.PreferredDiskBus))
					Expect(*vmi.Spec.Domain.Features.Hyperv).To(Equal(*pref.Spec.Features.PreferredHyperv))
					Expect(vmi.Spec.Domain.Firmware.Bootloader.BIOS).NotTo(BeNil())
					Expect(vmi.Spec.TerminationGracePeriodSeconds).To(Equal(pref.Spec.PreferredTerminationGracePeriodSeconds))
					Expect(vmi.Spec.Subdomain).To(Equal(*pref.Spec.PreferredSubdomain))
					Expect(vmi.Annotations).To(HaveKeyWithValue(virtv1.InstancetypeAnnotation, it.Name))
					Expect(vmi.Annotations).NotTo(HaveKey(virtv1.ClusterInstancetypeAnnotation))
					Expect(vmi.Annotations).To(HaveKeyWithValue(virtv1.PreferenceAnnotation, pref.Name))
					Expect(vmi.Annotations).NotTo(HaveKey(virtv1.ClusterPreferenceAnnotation))
					Expect(vmi.Annotations).To(HaveKeyWithValue("required-annotation-1", "1"))
					Expect(vmi.Annotations).To(HaveKeyWithValue("required-annotation-2", "2"))
					Expect(vmi.Annotations).To(HaveKeyWithValue("preferred-annotation-1", "1"))
					Expect(vmi.Annotations).To(HaveKeyWithValue("preferred-annotation-2", "2"))
				}
			})
		},
		Entry("cluster references without kind [test_id:CNV-9094]", "cluster"),
		Entry("namespaced references [test_id:CNV-9095]", "namespaced"),
		Entry("default network interface [test_id:CNV-9302]", "interface"),
		Entry("default volume disks [test_id:CNV-9303]", "disks"),
	)
}
