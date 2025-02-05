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

package rest

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	kubevirtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/pointer"
)

var _ = Describe("Object Graph", func() {
	var (
		virtClient *kubecli.MockKubevirtClient
		ctrl       *gomock.Controller
		kubeClient *fake.Clientset
		vmClient   *kubecli.MockVirtualMachineInterface
		vmiClient  *kubecli.MockVirtualMachineInstanceInterface
		graph      *ObjectGraph
		vm         *kubevirtv1.VirtualMachine
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		kubeClient = fake.NewSimpleClientset()
		vmClient = kubecli.NewMockVirtualMachineInterface(ctrl)
		vmiClient = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)
		graph = nil

		virtClient.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()
		virtClient.EXPECT().VirtualMachine(k8smetav1.NamespaceDefault).Return(vmClient).AnyTimes()
		virtClient.EXPECT().VirtualMachine("").Return(vmClient).AnyTimes()
		virtClient.EXPECT().VirtualMachineInstance(k8smetav1.NamespaceDefault).Return(vmiClient).AnyTimes()
		virtClient.EXPECT().VirtualMachineInstance("").Return(vmiClient).AnyTimes()

		vm = &kubevirtv1.VirtualMachine{
			ObjectMeta: k8smetav1.ObjectMeta{
				Name:      "test-vm",
				Namespace: "test-namespace",
			},
			Spec: kubevirtv1.VirtualMachineSpec{
				Template: &kubevirtv1.VirtualMachineInstanceTemplateSpec{
					Spec: kubevirtv1.VirtualMachineInstanceSpec{},
				},
			},
		}
	})

	It("should generate the correct object graph for VirtualMachine", func() {
		vm = &kubevirtv1.VirtualMachine{
			ObjectMeta: k8smetav1.ObjectMeta{
				Name:      "test-vm",
				Namespace: "test-namespace",
			},
			Spec: kubevirtv1.VirtualMachineSpec{
				Template: &kubevirtv1.VirtualMachineInstanceTemplateSpec{
					Spec: kubevirtv1.VirtualMachineInstanceSpec{
						Domain: kubevirtv1.DomainSpec{
							CPU: &kubevirtv1.CPU{
								Cores:   2,
								Sockets: 1,
								Threads: 1,
							},
							Resources: kubevirtv1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceMemory: resource.MustParse("4Gi"),
								},
							},
							Devices: kubevirtv1.Devices{
								Disks: []kubevirtv1.Disk{
									{
										Name: "rootdisk",
										DiskDevice: kubevirtv1.DiskDevice{
											Disk: &kubevirtv1.DiskTarget{
												Bus: "virtio",
											},
										},
									},
									{
										Name: "cloudinitdisk",
										DiskDevice: kubevirtv1.DiskDevice{
											Disk: &kubevirtv1.DiskTarget{
												Bus: "virtio",
											},
										},
									},
								},
								Interfaces: []kubevirtv1.Interface{
									{
										Name: "default",
										InterfaceBindingMethod: kubevirtv1.InterfaceBindingMethod{
											Bridge: &kubevirtv1.InterfaceBridge{},
										},
									},
								},
							},
						},
						AccessCredentials: []kubevirtv1.AccessCredential{
							{
								SSHPublicKey: &kubevirtv1.SSHPublicKeyAccessCredential{
									Source: kubevirtv1.SSHPublicKeyAccessCredentialSource{
										Secret: &kubevirtv1.AccessCredentialSecretSource{
											SecretName: "test-ssh-secret",
										},
									},
								},
							},
						},
						Volumes: []kubevirtv1.Volume{
							{
								Name: "rootdisk",
								VolumeSource: kubevirtv1.VolumeSource{
									PersistentVolumeClaim: &kubevirtv1.PersistentVolumeClaimVolumeSource{
										PersistentVolumeClaimVolumeSource: corev1.PersistentVolumeClaimVolumeSource{
											ClaimName: "test-root-disk-pvc",
										},
									},
								},
							},
							{
								Name: "datavolumedisk",
								VolumeSource: kubevirtv1.VolumeSource{
									DataVolume: &kubevirtv1.DataVolumeSource{
										Name: "test-datavolume",
									},
								},
							},
							{
								Name: "cloudinitdisk",
								VolumeSource: kubevirtv1.VolumeSource{
									CloudInitNoCloud: &kubevirtv1.CloudInitNoCloudSource{
										UserData: "#!/bin/bash\necho 'Hello World' > /root/welcome.txt\n",
									},
								},
							},
						},
						Networks: []kubevirtv1.Network{
							{
								Name: "default",
								NetworkSource: kubevirtv1.NetworkSource{
									Pod: &kubevirtv1.PodNetwork{},
								},
							},
						},
					},
				},
			},
			Status: kubevirtv1.VirtualMachineStatus{
				Created: true,
			},
		}

		kubeClient.Fake.PrependReactor("list", "pods", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			return true, &corev1.PodList{Items: []corev1.Pod{
				{
					ObjectMeta: k8smetav1.ObjectMeta{
						Name:      "virt-launcher-test-vmi",
						Namespace: "test-namespace",
						Labels: map[string]string{
							"kubevirt.io": "virt-launcher",
						},
						Annotations: map[string]string{
							"kubevirt.io/domain": vm.Name,
						},
					}}},
			}, nil
		})

		graph = NewObjectGraph(virtClient)
		graphNodes, err := graph.GetObjectGraph(vm)
		Expect(err).NotTo(HaveOccurred())
		Expect(graphNodes.Children).To(HaveLen(4))
		Expect(graphNodes.ObjectReference.Name).To(Equal("test-vm"))
		Expect(graphNodes.Children[0].ObjectReference.Name).To(Equal("test-vm"))

		// Child nodes of the VMI
		Expect(graphNodes.Children[0].Children).To(HaveLen(1))
		Expect(graphNodes.Children[0].Children[0].ObjectReference.Name).To(Equal("virt-launcher-test-vmi"))

		Expect(graphNodes.Children[1].ObjectReference.Name).To(Equal("test-ssh-secret"))
		Expect(graphNodes.Children[2].ObjectReference.Name).To(Equal("test-root-disk-pvc"))
		Expect(graphNodes.Children[3].ObjectReference.Name).To(Equal("test-datavolume"))
		Expect(graphNodes.Children[3].ObjectReference.Kind).To(Equal("datavolumes"))

		// Child nodes of DV
		Expect(graphNodes.Children[3].Children).To(HaveLen(1))
		Expect(graphNodes.Children[3].Children[0].ObjectReference.Name).To(Equal("test-datavolume"))
		Expect(graphNodes.Children[3].Children[0].ObjectReference.Kind).To(Equal("persistentvolumeclaims"))
	})

	It("should handle error when listing pods", func() {
		vm.Status.Created = true
		kubeClient.Fake.PrependReactor("list", "pods", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			return true, nil, fmt.Errorf("error listing pods")
		})
		graph = NewObjectGraph(virtClient)
		graphNodes, err := graph.GetObjectGraph(vm)
		Expect(err).To(HaveOccurred())
		Expect(graphNodes.Children).To(HaveLen(1)) // Should still return the VMI
	})

	It("should include backend storage PVC in the graph", func() {
		vm.Spec.Template.Spec.Domain.Devices.TPM = &kubevirtv1.TPMDevice{
			Persistent: pointer.P(true),
		}
		pvc := &corev1.PersistentVolumeClaim{
			ObjectMeta: k8smetav1.ObjectMeta{
				Name:      "backend-storage-pvc",
				Namespace: "test-namespace",
				Labels: map[string]string{
					"persistent-state-for": vm.Name,
				},
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{
					corev1.ReadWriteOnce,
				},
				Resources: corev1.VolumeResourceRequirements{},
			},
		}

		kubeClient.Fake.PrependReactor("list", "persistentvolumeclaims", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			return true, &corev1.PersistentVolumeClaimList{Items: []corev1.PersistentVolumeClaim{*pvc}}, nil
		})

		graph = NewObjectGraph(virtClient)
		graphNodes, err := graph.GetObjectGraph(vm)
		Expect(err).NotTo(HaveOccurred())
		Expect(graphNodes.Children).To(HaveLen(1))
		Expect(graphNodes.Children[0].ObjectReference.Name).To(Equal("backend-storage-pvc"))
	})

	It("should return empty graph for unrelated objects", func() {
		pod := &corev1.Pod{
			ObjectMeta: k8smetav1.ObjectMeta{
				Name:      "test-pod",
				Namespace: "test-namespace",
			},
		}

		graph = NewObjectGraph(virtClient)
		graphNodes, err := graph.GetObjectGraph(pod)
		Expect(err).NotTo(HaveOccurred())
		Expect(graphNodes.Children).To(BeEmpty())
	})
})
