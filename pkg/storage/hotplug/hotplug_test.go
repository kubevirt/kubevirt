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
 * Copyright 2025 The KubeVirt Contributors
 *
 */
package hotplug

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/kubevirt/fake"

	"kubevirt.io/kubevirt/pkg/libvmi"
	libvmistatus "kubevirt.io/kubevirt/pkg/libvmi/status"
)

type addVolFunc func(diskName, pvcName string, diskOpts ...libvmi.DiskOption) libvmi.Option

var _ = Describe("Volume Hotplug", func() {
	var virtClient *kubecli.MockKubevirtClient
	var virtFakeClient *fake.Clientset

	BeforeEach(func() {
		virtClient = kubecli.NewMockKubevirtClient(gomock.NewController(GinkgoT()))
		virtFakeClient = fake.NewSimpleClientset()

		// Set up mock client
		virtClient.EXPECT().VirtualMachineInstance(metav1.NamespaceDefault).Return(
			virtFakeClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault),
		).AnyTimes()
		virtClient.EXPECT().VirtualMachine(metav1.NamespaceDefault).Return(
			virtFakeClient.KubevirtV1().VirtualMachines(metav1.NamespaceDefault),
		).AnyTimes()
	})

	Context("declarative volume hotplug", func() {
		handle := func(vm *v1.VirtualMachine, vmi *v1.VirtualMachineInstance) *v1.VirtualMachineInstance {
			vm, err := virtFakeClient.KubevirtV1().VirtualMachines(metav1.NamespaceDefault).Create(context.TODO(), vm, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			vmi, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			Expect(HandleDeclarativeVolumes(virtClient, vm, vmi)).To(Succeed())

			vmi, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault).Get(context.TODO(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			return vmi
		}

		It("should do nothing if VMI not running", func() {
			opts := []libvmi.Option{
				libvmi.WithDataVolume("perm", "perm"),
			}
			allOpts := append(opts, libvmi.WithHotplugDataVolume("hotplugdisk_1", "hotplugvolume_1"))
			origVMI := libvmi.New(opts...)
			postVMI := libvmi.New(allOpts...)
			postVMI.Name = origVMI.Name
			vm := libvmi.NewVirtualMachine(postVMI)
			result := handle(vm, origVMI)
			Expect(result.Spec).To(Equal(origVMI.Spec))
			Expect(result.Spec.Domain.Devices.Disks).To(HaveLen(1))
			Expect(result.Spec.Volumes).To(HaveLen(1))
		})

		It("should do nothing if non hotplug volume added", func() {
			opts := []libvmi.Option{
				libvmi.WithDataVolume("perm", "perm"),
				libvmistatus.WithStatus(libvmistatus.New(libvmistatus.WithPhase(v1.Running))),
			}
			allOpts := append(opts, libvmi.WithDataVolume("hotplugdisk_1", "hotplugvolume_1"))
			origVMI := libvmi.New(opts...)
			postVMI := libvmi.New(allOpts...)
			postVMI.Name = origVMI.Name
			vm := libvmi.NewVirtualMachine(postVMI)
			result := handle(vm, origVMI)
			Expect(result.Spec).To(Equal(origVMI.Spec))
			Expect(result.Spec.Domain.Devices.Disks).To(HaveLen(1))
			Expect(result.Spec.Volumes).To(HaveLen(1))
		})

		DescribeTable("should add hotplug volumes to VMI", func(f addVolFunc, numDisks int) {
			opts := []libvmi.Option{
				libvmi.WithDataVolume("perm", "perm"),
				libvmistatus.WithStatus(libvmistatus.New(libvmistatus.WithPhase(v1.Running))),
			}
			allOpts := opts
			for i := 1; i <= numDisks; i++ {
				allOpts = append(allOpts, f(fmt.Sprintf("hotplugdisk_%d", i), fmt.Sprintf("hotplugvolume_%d", i)))
			}
			origVMI := libvmi.New(opts...)
			postVMI := libvmi.New(allOpts...)
			postVMI.Name = origVMI.Name
			vm := libvmi.NewVirtualMachine(postVMI)
			result := handle(vm, origVMI)
			Expect(result.Spec).To(Equal(postVMI.Spec))
			Expect(result.Spec.Domain.Devices.Disks).To(HaveLen(numDisks + 1)) // +1 for the existing disk
			Expect(result.Spec.Volumes).To(HaveLen(numDisks + 1))              // +1 for the existing volume
		},
			Entry("With one DataVolume", libvmi.WithHotplugDataVolume, 1),
			Entry("With five DataVolumes", libvmi.WithHotplugDataVolume, 5),
			Entry("With one PVC", libvmi.WithHotplugPersistentVolumeClaim, 1),
			Entry("With five PVC", libvmi.WithHotplugPersistentVolumeClaim, 5),
		)

		DescribeTable("should remove hotplug volumes from VMI", func(f addVolFunc, numDisks, index int) {
			opts := []libvmi.Option{
				libvmi.WithDataVolume("perm", "perm"),
				libvmistatus.WithStatus(libvmistatus.New(libvmistatus.WithPhase(v1.Running))),
			}
			for i := 1; i <= numDisks; i++ {
				opts = append(opts, f(fmt.Sprintf("hotplugdisk_%d", i), fmt.Sprintf("hotplugvolume_%d", i)))
			}
			origVMI := libvmi.New(opts...)
			postVMI := origVMI.DeepCopy()
			postVMI.Spec.Domain.Devices.Disks = removeFromSlice(postVMI.Spec.Domain.Devices.Disks, index+1)
			postVMI.Spec.Volumes = removeFromSlice(postVMI.Spec.Volumes, index+1)
			vm := libvmi.NewVirtualMachine(postVMI)
			result := handle(vm, origVMI)
			Expect(result.Spec).To(Equal(postVMI.Spec))
			Expect(result.Spec.Domain.Devices.Disks).To(HaveLen(numDisks))
			Expect(result.Spec.Volumes).To(HaveLen(numDisks))
		},
			Entry("With one DataVolume", libvmi.WithHotplugDataVolume, 1, 0),
			Entry("With three DataVolumes index 0", libvmi.WithHotplugDataVolume, 3, 0),
			Entry("With three DataVolumes index 1", libvmi.WithHotplugDataVolume, 3, 1),
			Entry("With three DataVolumes index 2", libvmi.WithHotplugDataVolume, 3, 2),
			Entry("With one PVC", libvmi.WithHotplugPersistentVolumeClaim, 1, 0),
			Entry("With three PVCs index 0", libvmi.WithHotplugPersistentVolumeClaim, 3, 0),
			Entry("With three PVCs index 1", libvmi.WithHotplugPersistentVolumeClaim, 3, 1),
			Entry("With three PVCs index 2", libvmi.WithHotplugPersistentVolumeClaim, 3, 2),
		)

		It("should not add hotplug volume to VMI if vmi has status for the volume", func() {
			opts := []libvmi.Option{
				libvmi.WithDataVolume("perm", "perm"),
				libvmistatus.WithStatus(
					libvmistatus.New(
						libvmistatus.WithPhase(v1.Running),
						libvmistatus.WithVolumeStatus(
							v1.VolumeStatus{
								Name: "hotplugdisk_1",
							},
						),
					),
				),
			}
			allOpts := append(opts, libvmi.WithHotplugDataVolume("hotplugdisk_1", "hotplugvolume_1"))
			origVMI := libvmi.New(opts...)
			postVMI := libvmi.New(allOpts...)
			postVMI.Name = origVMI.Name
			vm := libvmi.NewVirtualMachine(postVMI)
			result := handle(vm, origVMI)
			Expect(result.Spec).To(Equal(origVMI.Spec))
			Expect(result.Spec.Domain.Devices.Disks).To(HaveLen(1))
			Expect(result.Spec.Volumes).To(HaveLen(1))
		})

		It("should remove volume when volume changes", func() {
			opts := []libvmi.Option{
				libvmi.WithDataVolume("perm", "perm"),
				libvmi.WithHotplugDataVolume("hotplugdisk_1", "hotplugvolume_1"),
				libvmistatus.WithStatus(libvmistatus.New(libvmistatus.WithPhase(v1.Running))),
			}
			origVMI := libvmi.New(opts...)
			postVMI := origVMI.DeepCopy()
			postVMI.Spec.Volumes[1].VolumeSource.DataVolume.Name = "changed"
			vm := libvmi.NewVirtualMachine(postVMI)
			result := handle(vm, origVMI)
			Expect(result.Spec.Domain.Devices.Disks).To(HaveLen(1))
			Expect(result.Spec.Domain.Devices.Disks[0].Name).To(Equal("perm"))
			Expect(result.Spec.Volumes).To(HaveLen(1))
			Expect(result.Spec.Volumes[0].Name).To(Equal("perm"))
		})

		It("should not add hotplug volume to VMI with migration updatestrategy", func() {
			opts := []libvmi.Option{
				libvmi.WithDataVolume("perm", "perm"),
				libvmistatus.WithStatus(libvmistatus.New(libvmistatus.WithPhase(v1.Running))),
			}
			allOpts := append(opts, libvmi.WithDataVolume("hotplugdisk_1", "hotplugvolume_1"))
			origVMI := libvmi.New(opts...)
			postVMI := libvmi.New(allOpts...)
			postVMI.Name = origVMI.Name
			vm := libvmi.NewVirtualMachine(postVMI, libvmi.WithUpdateVolumeStrategy(v1.UpdateVolumesStrategyMigration))
			result := handle(vm, origVMI)
			Expect(result.Spec).To(Equal(origVMI.Spec))
			Expect(result.Spec.Domain.Devices.Disks).To(HaveLen(1))
			Expect(result.Spec.Volumes).To(HaveLen(1))
		})
	})
})

func removeFromSlice[T any](slice []T, index int) []T {
	if index < 0 || index >= len(slice) {
		return slice
	}
	return append(slice[:index], slice[index+1:]...)
}
