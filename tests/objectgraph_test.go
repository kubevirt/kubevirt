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
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	libdv "kubevirt.io/kubevirt/pkg/libdv"
	"kubevirt.io/kubevirt/pkg/libvmi"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	. "kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libsecret"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/libvmops"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe("[sig-storage]ObjectGraph", decorators.SigStorage, func() {
	var (
		virtClient kubecli.KubevirtClient
	)

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	Context("with VM", func() {
		var (
			vm     *v1.VirtualMachine
			secret *corev1.Secret
			pvc    *corev1.PersistentVolumeClaim
		)

		BeforeEach(func() {
			By("Creating a PVC")
			pvc = &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "test-pvc-",
					Namespace:    testsuite.GetTestNamespace(nil),
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
					Resources: corev1.VolumeResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse("1Gi"),
						},
					},
				},
			}
			var err error
			pvc, err = virtClient.CoreV1().PersistentVolumeClaims(testsuite.GetTestNamespace(nil)).Create(context.Background(), pvc, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Creating a Secret")
			secret = libsecret.New(fmt.Sprintf("test-secret-%s", pvc.Name), libsecret.DataString{"token": "test-token"})
			secret, err = virtClient.CoreV1().Secrets(testsuite.GetTestNamespace(nil)).Create(context.Background(), secret, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Creating a VM with dependencies")
			vm = libvmi.NewVirtualMachine(
				libvmi.New(
					libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
					libvmi.WithPersistentVolumeClaim("disk0", pvc.Name),
					libvmi.WithAccessCredentialUserPassword(secret.Name),
					libvmi.WithMemoryRequest("100M"),
					libvmi.WithTPM(true),
				),
				libvmi.WithRunStrategy(v1.RunStrategyAlways),
			)
			vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).Create(context.Background(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			By("Waiting for VMI to start")
			Eventually(ThisVMIWith(vm.Namespace, vm.Name), 120*time.Second, 1*time.Second).Should(BeRunning())
		})

		It("should return object graph for VM with PVC, backend storage PVC and Secret", func() {
			By("Getting the object graph for the VM")
			objectGraph, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).ObjectGraph(context.Background(), vm.Name, &v1.ObjectGraphOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(objectGraph).ToNot(BeNil())

			By("Verifying the VM is the root node")
			Expect(objectGraph.ObjectReference.Name).To(Equal(vm.Name))
			Expect(objectGraph.ObjectReference.Kind).To(Equal("VirtualMachine"))

			By("Verifying dependencies are included")
			Expect(objectGraph.Children).To(HaveLen(4)) // Includes VMI too
			pvcFound := false
			secretFound := false
			tpmFound := false
			for _, child := range objectGraph.Children {
				if child.ObjectReference.Kind == "PersistentVolumeClaim" && child.ObjectReference.Name == pvc.Name {
					pvcFound = true
				}
				if child.ObjectReference.Kind == "PersistentVolumeClaim" && strings.HasPrefix(child.ObjectReference.Name, "persistent-state-for-") {
					tpmFound = true
				}
				if child.ObjectReference.Kind == "Secret" && child.ObjectReference.Name == secret.Name {
					secretFound = true
				}
			}
			Expect(pvcFound).To(BeTrue())
			Expect(secretFound).To(BeTrue())
			Expect(tpmFound).To(BeTrue())
		})

		It("should filter dependencies using label selector", func() {
			By("Getting the object graph filtered for storage dependencies")
			objectGraph, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).ObjectGraph(context.Background(), vm.Name, &v1.ObjectGraphOptions{
				LabelSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"kubevirt.io/dependency-type": "storage",
					},
				},
			})
			Expect(err).ToNot(HaveOccurred())

			By("Verifying only storage dependencies are returned")
			for _, child := range objectGraph.Children {
				Expect(child.Labels["kubevirt.io/dependency-type"]).To(Equal("storage"))
			}
		})

		It("should detect hotplugged disks in the object graph", func() {
			By("Hotplugging a disk")
			hotplugPVC := libstorage.NewPVC("hotplug-pvc", "1Gi", "")
			hotplugPVC, err := virtClient.CoreV1().PersistentVolumeClaims(testsuite.GetTestNamespace(nil)).Create(context.Background(), hotplugPVC, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			hotplugOptions := &v1.AddVolumeOptions{
				Name: hotplugPVC.Name,
				Disk: &v1.Disk{
					DiskDevice: v1.DiskDevice{},
					Serial:     hotplugPVC.Name,
				},
				VolumeSource: &v1.HotplugVolumeSource{
					PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
						PersistentVolumeClaimVolumeSource: corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: hotplugPVC.Name,
						},
					},
				},
				DryRun: nil,
			}

			Eventually(func() error {
				return virtClient.VirtualMachine(vm.Namespace).AddVolume(context.Background(), vm.Name, hotplugOptions)
			}, 3*time.Second, 1*time.Second).Should(Succeed())

			// Wait for hotplug volume to be added
			Eventually(func() bool {
				updatedVM, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
				if err != nil {
					return false
				}
				for _, volume := range updatedVM.Spec.Template.Spec.Volumes {
					if volume.Name == hotplugPVC.Name {
						return true
					}
				}
				return false
			}, 90*time.Second, 1*time.Second).Should(BeTrue())

			By("Getting the object graph for the VM")
			objectGraph, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).ObjectGraph(context.Background(), vm.Name, &v1.ObjectGraphOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(objectGraph).ToNot(BeNil())

			By("Verifying the hotplugged PVC is included")
			hotplugFound := false
			for _, child := range objectGraph.Children {
				if child.ObjectReference.Kind == "PersistentVolumeClaim" && child.ObjectReference.Name == hotplugPVC.Name {
					hotplugFound = true
				}
			}
			Expect(hotplugFound).To(BeTrue())
		})

		It("Object Graph should detect newly added resources", func() {
			By("Creating DataVolume")
			dv := libdv.NewDataVolume(
				libdv.WithRegistryURLSource(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskCirros)),
				libdv.WithStorage(),
				libdv.WithForceBindAnnotation(),
			)
			dv, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(nil)).Create(context.Background(), dv, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			libstorage.EventuallyDV(dv, 240, HaveSucceeded())

			By("Getting the initial object graph for the VM")
			initialObjectGraph, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).ObjectGraph(context.Background(), vm.Name, &v1.ObjectGraphOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(initialObjectGraph).ToNot(BeNil())
			initialDependencyCount := len(initialObjectGraph.Children)

			By("Verifying the DataVolume is not included in the original graph")
			dvFound := false
			for _, child := range initialObjectGraph.Children {
				if child.ObjectReference.Kind == "DataVolume" && child.ObjectReference.Name == dv.Name {
					dvFound = true
					break
				}
			}
			// DV shouldn't be part of the initial graph
			Expect(dvFound).To(BeFalse())

			vm = libvmops.StopVirtualMachine(vm)

			By("Adding the DataVolume to the VM")
			vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v1.Volume{
				Name: "test-datavolume",
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: dv.Name,
					},
				},
			})
			vm.Spec.Template.Spec.Domain.Devices.Disks = append(vm.Spec.Template.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "test-datavolume",
				DiskDevice: v1.DiskDevice{
					Disk: &v1.DiskTarget{
						Bus: "virtio",
					},
				},
			})
			vm, err = virtClient.VirtualMachine(vm.Namespace).Update(context.Background(), vm, metav1.UpdateOptions{})
			Expect(err).ToNot(HaveOccurred())

			vm = libvmops.StartVirtualMachine(vm)

			By("Getting the updated object graph for the VM")
			updatedObjectGraph, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).ObjectGraph(context.Background(), vm.Name, &v1.ObjectGraphOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(len(updatedObjectGraph.Children)).To(BeNumerically(">", initialDependencyCount))

			By("Verifying the DataVolume is included in the updated object graph")
			for _, child := range updatedObjectGraph.Children {
				if child.ObjectReference.Kind == "DataVolume" && child.ObjectReference.Name == dv.Name {
					dvFound = true
					break
				}
			}
			Expect(dvFound).To(BeTrue())
		})
	})

	Context("with VMI", func() {
		var vmi *v1.VirtualMachineInstance

		BeforeEach(func() {
			By("Creating and starting a VMI")
			vmi = libvmi.New(
				libvmi.WithMemoryRequest("128Mi"),
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			)
			var err error
			vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
		})

		It("should return object graph for running VMI with launcher pod", func() {
			By("Waiting for VMI to be running")
			Eventually(func() bool {
				updatedVmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Get(context.Background(), vmi.Name, metav1.GetOptions{})
				if err != nil {
					return false
				}
				return updatedVmi.Status.Phase == v1.Running
			}, 180*time.Second, 1*time.Second).Should(BeTrue())

			By("Getting the object graph for the VMI")
			objectGraph, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).ObjectGraph(context.Background(), vmi.Name, &v1.ObjectGraphOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(objectGraph).ToNot(BeNil())

			By("Verifying the VMI is the root node")
			Expect(objectGraph.ObjectReference.Name).To(Equal(vmi.Name))
			Expect(objectGraph.ObjectReference.Kind).To(Equal("VirtualMachineInstance"))

			By("Verifying launcher pod is included")
			Expect(objectGraph.Children).To(HaveLen(1))
			Expect(objectGraph.Children[0].ObjectReference.Name).To(ContainSubstring("virt-launcher"))
			Expect(objectGraph.Children[0].ObjectReference.Kind).To(Equal("Pod"))
		})
	})

	Context("with optional resources", func() {
		var vm *v1.VirtualMachine

		BeforeEach(func() {
			By("Creating a VM with instance type")
			vm = libvmi.NewVirtualMachine(
				libvmi.New(
					libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
				),
			)

			// this would be optional
			vm.Spec.Instancetype = &v1.InstancetypeMatcher{
				Name: "test-instancetype",
				Kind: "VirtualMachineInstancetype",
			}

			var err error
			vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).Create(context.Background(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
		})

		It("should exclude optional resources when IncludeOptionalNodes is false", func() {
			By("Getting object graph with optional nodes excluded")
			includeOptional := false
			objectGraph, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).ObjectGraph(context.Background(), vm.Name, &v1.ObjectGraphOptions{
				IncludeOptionalNodes: &includeOptional,
			})
			Expect(err).ToNot(HaveOccurred())

			By("Verifying optional resources are excluded")
			for _, child := range objectGraph.Children {
				Expect(child.ObjectReference.Name).ToNot(Equal("test-instancetype"))
			}

			By("Getting object graph with optional nodes included")
			objectGraphWithOptional, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).ObjectGraph(context.Background(), vm.Name, &v1.ObjectGraphOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Verifying the graph with optional nodes has more dependencies")
			Expect(len(objectGraphWithOptional.Children)).To(BeNumerically(">", len(objectGraph.Children)))
		})
	})
})
