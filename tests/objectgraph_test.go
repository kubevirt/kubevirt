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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libsecret"
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
				),
			)
			vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).Create(context.Background(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
		})

		It("should return object graph for VM with PVC and Secret", func() {
			By("Getting the object graph for the VM")
			objectGraph, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).ObjectGraph(context.Background(), vm.Name, &v1.ObjectGraphOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(objectGraph).ToNot(BeNil())

			By("Verifying the VM is the root node")
			Expect(objectGraph.ObjectReference.Name).To(Equal(vm.Name))
			Expect(objectGraph.ObjectReference.Kind).To(Equal("VirtualMachine"))

			By("Verifying dependencies are included")
			Expect(objectGraph.Children).To(HaveLen(2))
			pvcFound := false
			secretFound := false
			for _, child := range objectGraph.Children {
				if child.ObjectReference.Kind == "PersistentVolumeClaim" && child.ObjectReference.Name == pvc.Name {
					pvcFound = true
				}
				if child.ObjectReference.Kind == "Secret" && child.ObjectReference.Name == secret.Name {
					secretFound = true
				}
			}
			Expect(pvcFound).To(BeTrue())
			Expect(secretFound).To(BeTrue())
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
	})

	Context("with VMI", func() {
		var vmi *v1.VirtualMachineInstance

		BeforeEach(func() {
			By("Creating and starting a VMI")
			vmi = libvmi.New(
				libvmi.WithResourceMemory("128Mi"),
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

	// TODO: Need to update kubevirtCI to install the IPAMClaim CRD.
	// Skipping until the CRD is available.
	PContext("Network resources", decorators.Multus, func() {
		const nadName = "test-network-attachment"

		It("should include network resources in object graph", func() {
			const bridgeName = "br10"
			netAttachDef := libnet.NewBridgeNetAttachDef(nadName, bridgeName)
			_, err := libnet.CreateNetAttachDef(context.Background(), testsuite.GetTestNamespace(nil), netAttachDef)
			Expect(err).ToNot(HaveOccurred())

			const secondaryNetName = "secondary-net"
			defaultModelIface := libvmi.InterfaceDeviceWithBridgeBinding(secondaryNetName)
			defaultModelIface.PciAddress = "0000:03:00.0"
			vm := libvmi.NewVirtualMachine(
				libvmi.New(
					libvmi.WithInterface(defaultModelIface),
					libvmi.WithNetwork(libvmi.MultusNetwork(secondaryNetName, nadName)),
				))

			// TODO: Need to update kubevirtCI to install the IPAMClaim CRD.
			ipamClaim, err := libnet.CreateIPAMClaimForVM(context.Background(), testsuite.GetTestNamespace(nil), vm.Name, "test-ipam-claim", "", "")
			Expect(err).ToNot(HaveOccurred())

			vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).Create(context.Background(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Getting object graph")
			objectGraph, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).ObjectGraph(context.Background(), vm.Name, &v1.ObjectGraphOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Verifying NetworkAttachmentDefinition is included")
			hasNetworkAttachment := false
			for _, child := range objectGraph.Children {
				if child.ObjectReference.Kind == "NetworkAttachmentDefinition" &&
					child.ObjectReference.Name == netAttachDef.Name {
					hasNetworkAttachment = true
					break
				}
			}
			Expect(hasNetworkAttachment).To(BeTrue())

			By("Verifying IPAMClaim is included")
			hasIPAMClaim := false
			for _, child := range objectGraph.Children {
				if child.ObjectReference.Kind == "IPAMClaim" &&
					child.ObjectReference.Name == ipamClaim.GetName() {
					hasIPAMClaim = true
					break
				}
			}
			Expect(hasIPAMClaim).To(BeTrue())
		})
	})
})
