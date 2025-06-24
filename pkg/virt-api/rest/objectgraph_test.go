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

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/pkg/pointer"
)

var _ = Describe("Object Graph", func() {
	var (
		virtClient *kubecli.MockKubevirtClient
		kubeClient *fake.Clientset
		vm         *v1.VirtualMachine
	)

	BeforeEach(func() {
		ctrl := gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		kubeClient = fake.NewSimpleClientset()

		virtClient.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()

		vm = &v1.VirtualMachine{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-vm",
				Namespace: "test-namespace",
			},
			Spec: v1.VirtualMachineSpec{
				Template: &v1.VirtualMachineInstanceTemplateSpec{
					Spec: v1.VirtualMachineInstanceSpec{},
				},
			},
		}
	})

	Context("with empty options", func() {
		It("should generate the correct object graph for VirtualMachine", func() {
			vm = &v1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vm",
					Namespace: "test-namespace",
				},
				Spec: v1.VirtualMachineSpec{
					Template: &v1.VirtualMachineInstanceTemplateSpec{
						Spec: v1.VirtualMachineInstanceSpec{
							Domain: v1.DomainSpec{
								CPU: &v1.CPU{
									Cores:   2,
									Sockets: 1,
									Threads: 1,
								},
								Resources: v1.ResourceRequirements{
									Requests: k8sv1.ResourceList{
										k8sv1.ResourceMemory: resource.MustParse("4Gi"),
									},
								},
								Devices: v1.Devices{
									Disks: []v1.Disk{
										{
											Name: "rootdisk",
											DiskDevice: v1.DiskDevice{
												Disk: &v1.DiskTarget{
													Bus: "virtio",
												},
											},
										},
										{
											Name: "cloudinitdisk",
											DiskDevice: v1.DiskDevice{
												Disk: &v1.DiskTarget{
													Bus: "virtio",
												},
											},
										},
									},
									Interfaces: []v1.Interface{
										{
											Name: "default",
											InterfaceBindingMethod: v1.InterfaceBindingMethod{
												Bridge: &v1.InterfaceBridge{},
											},
										},
									},
								},
							},
							AccessCredentials: []v1.AccessCredential{
								{
									SSHPublicKey: &v1.SSHPublicKeyAccessCredential{
										Source: v1.SSHPublicKeyAccessCredentialSource{
											Secret: &v1.AccessCredentialSecretSource{
												SecretName: "test-ssh-secret",
											},
										},
									},
								},
							},
							Volumes: []v1.Volume{
								{
									Name: "rootdisk",
									VolumeSource: v1.VolumeSource{
										PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
											PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
												ClaimName: "test-root-disk-pvc",
											},
										},
									},
								},
								{
									Name: "datavolumedisk",
									VolumeSource: v1.VolumeSource{
										DataVolume: &v1.DataVolumeSource{
											Name: "test-datavolume",
										},
									},
								},
								{
									Name: "cloudinitdisk",
									VolumeSource: v1.VolumeSource{
										CloudInitNoCloud: &v1.CloudInitNoCloudSource{
											UserData: "#!/bin/bash\necho 'Hello World' > /root/welcome.txt\n",
										},
									},
								},
							},
							Networks: []v1.Network{
								{
									Name: "default",
									NetworkSource: v1.NetworkSource{
										Pod: &v1.PodNetwork{},
									},
								},
							},
						},
					},
				},
				Status: v1.VirtualMachineStatus{
					Created: true,
				},
			}

			kubeClient.Fake.PrependReactor("list", "pods", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				return true, &k8sv1.PodList{Items: []k8sv1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{
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

			graph := NewObjectGraph(virtClient, &v1.ObjectGraphOptions{})
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
			Expect(graphNodes.Children[3].ObjectReference.Kind).To(Equal("DataVolume"))
			// Child nodes of the DV
			Expect(graphNodes.Children[3].Children).To(HaveLen(1))
			Expect(graphNodes.Children[3].Children[0].ObjectReference.Name).To(Equal("test-datavolume"))
			Expect(graphNodes.Children[3].Children[0].ObjectReference.Kind).To(Equal("PersistentVolumeClaim"))
		})

		It("should generate object graph for VirtualMachineInstance", func() {
			vmi := &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vmi",
					Namespace: "test-namespace",
				},
				Spec: v1.VirtualMachineInstanceSpec{
					AccessCredentials: []v1.AccessCredential{
						{
							SSHPublicKey: &v1.SSHPublicKeyAccessCredential{
								Source: v1.SSHPublicKeyAccessCredentialSource{
									Secret: &v1.AccessCredentialSecretSource{
										SecretName: "vmi-ssh-secret",
									},
								},
							},
						},
					},
					Volumes: []v1.Volume{
						{
							Name: "root-disk",
							VolumeSource: v1.VolumeSource{
								PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
									PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
										ClaimName: "vmi-root-pvc",
									},
								},
							},
						},
					},
				},
			}

			kubeClient.Fake.PrependReactor("list", "pods", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				return true, &k8sv1.PodList{Items: []k8sv1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "virt-launcher-test-vmi-pod",
							Namespace: "test-namespace",
							Labels: map[string]string{
								"kubevirt.io": "virt-launcher",
							},
							OwnerReferences: []metav1.OwnerReference{
								{
									Kind: "VirtualMachineInstance",
									Name: "test-vmi",
								},
							},
						}}},
				}, nil
			})

			graph := NewObjectGraph(virtClient, &v1.ObjectGraphOptions{})
			graphNodes, err := graph.GetObjectGraph(vmi)
			Expect(err).NotTo(HaveOccurred())
			Expect(graphNodes.ObjectReference.Name).To(Equal("test-vmi"))
			Expect(graphNodes.ObjectReference.Kind).To(Equal("VirtualMachineInstance"))

			childMap := make(map[string]string)
			for _, child := range graphNodes.Children {
				childMap[child.ObjectReference.Name] = child.ObjectReference.Kind
			}

			Expect(childMap).To(HaveKey("virt-launcher-test-vmi-pod"))
			Expect(childMap["virt-launcher-test-vmi-pod"]).To(Equal("Pod"))
			Expect(childMap).To(HaveKey("vmi-ssh-secret"))
			Expect(childMap["vmi-ssh-secret"]).To(Equal("Secret"))
			Expect(childMap).To(HaveKey("vmi-root-pvc"))
			Expect(childMap["vmi-root-pvc"]).To(Equal("PersistentVolumeClaim"))
		})

		It("should handle error when listing pods", func() {
			vm.Status.Created = true
			kubeClient.Fake.PrependReactor("list", "pods", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				return true, nil, fmt.Errorf("error listing pods")
			})
			graph := NewObjectGraph(virtClient, &v1.ObjectGraphOptions{})
			graphNodes, err := graph.GetObjectGraph(vm)
			Expect(err).To(HaveOccurred())
			Expect(graphNodes.Children).To(HaveLen(1))
		})

		It("should include backend storage PVC in the graph", func() {
			vm.Spec.Template.Spec.Domain.Devices.TPM = &v1.TPMDevice{
				Persistent: pointer.P(true),
			}
			pvc := &k8sv1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "backend-storage-pvc",
					Namespace: "test-namespace",
					Labels: map[string]string{
						"persistent-state-for": vm.Name,
					},
				},
				Spec: k8sv1.PersistentVolumeClaimSpec{
					AccessModes: []k8sv1.PersistentVolumeAccessMode{
						k8sv1.ReadWriteOnce,
					},
					Resources: k8sv1.VolumeResourceRequirements{},
				},
			}

			kubeClient.Fake.PrependReactor("list", "persistentvolumeclaims", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				return true, &k8sv1.PersistentVolumeClaimList{Items: []k8sv1.PersistentVolumeClaim{*pvc}}, nil
			})

			graph := NewObjectGraph(virtClient, &v1.ObjectGraphOptions{})
			graphNodes, err := graph.GetObjectGraph(vm)
			Expect(err).NotTo(HaveOccurred())
			Expect(graphNodes.Children).To(HaveLen(1))
			Expect(graphNodes.Children[0].ObjectReference.Name).To(Equal("backend-storage-pvc"))
		})

		It("should return empty graph for unrelated objects", func() {
			pod := &k8sv1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod",
					Namespace: "test-namespace",
				},
			}

			graph := NewObjectGraph(virtClient, &v1.ObjectGraphOptions{})
			graphNodes, err := graph.GetObjectGraph(pod)
			Expect(err).NotTo(HaveOccurred())
			Expect(graphNodes.Children).To(BeEmpty())
		})

		It("should handle VM with instance type and preference", func() {
			vm.Spec.Instancetype = &v1.InstancetypeMatcher{
				Name: "test-instancetype",
				Kind: "VirtualMachineInstancetype",
			}
			vm.Spec.Preference = &v1.PreferenceMatcher{
				Name: "test-preference",
				Kind: "VirtualMachinePreference",
			}
			vm.Status.InstancetypeRef = &v1.InstancetypeStatusRef{
				ControllerRevisionRef: &v1.ControllerRevisionRef{
					Name: "test-instancetype-revision",
				},
			}
			vm.Status.PreferenceRef = &v1.InstancetypeStatusRef{
				ControllerRevisionRef: &v1.ControllerRevisionRef{
					Name: "test-preference-revision",
				},
			}

			graph := NewObjectGraph(virtClient, &v1.ObjectGraphOptions{})
			graphNodes, err := graph.GetObjectGraph(vm)
			Expect(err).NotTo(HaveOccurred())
			Expect(graphNodes.Children).To(HaveLen(4))

			childMap := make(map[string]v1.ObjectGraphNode)
			for _, child := range graphNodes.Children {
				childMap[child.ObjectReference.Name] = child
			}

			Expect(childMap).To(HaveKey("test-instancetype"))
			instanceTypeNode := childMap["test-instancetype"]
			Expect(instanceTypeNode.ObjectReference.Kind).To(Equal("VirtualMachineInstancetype"))
			Expect(*instanceTypeNode.Optional).To(BeTrue())

			Expect(childMap).To(HaveKey("test-preference"))
			preferenceNode := childMap["test-preference"]
			Expect(preferenceNode.ObjectReference.Kind).To(Equal("VirtualMachinePreference"))
			Expect(*preferenceNode.Optional).To(BeTrue())

			Expect(childMap).To(HaveKey("test-instancetype-revision"))
			Expect(childMap["test-instancetype-revision"].ObjectReference.Kind).To(Equal("ControllerRevision"))
			Expect(childMap).To(HaveKey("test-preference-revision"))
			Expect(childMap["test-preference-revision"].ObjectReference.Kind).To(Equal("ControllerRevision"))
		})

		It("should handle VM with cluster instance type and preference", func() {
			vm.Spec.Instancetype = &v1.InstancetypeMatcher{
				Name: "test-cluster-instancetype",
				Kind: "VirtualMachineClusterInstancetype",
			}
			vm.Spec.Preference = &v1.PreferenceMatcher{
				Name: "test-cluster-preference",
				Kind: "VirtualMachineClusterPreference",
			}

			graph := NewObjectGraph(virtClient, &v1.ObjectGraphOptions{})
			graphNodes, err := graph.GetObjectGraph(vm)
			Expect(err).NotTo(HaveOccurred())

			childMap := make(map[string]v1.ObjectGraphNode)
			for _, child := range graphNodes.Children {
				childMap[child.ObjectReference.Name] = child
			}

			Expect(childMap).To(HaveKey("test-cluster-instancetype"))
			instanceTypeNode := childMap["test-cluster-instancetype"]
			Expect(instanceTypeNode.ObjectReference.Kind).To(Equal("VirtualMachineClusterInstancetype"))
			Expect(*instanceTypeNode.ObjectReference.Namespace).To(Equal(""))

			Expect(childMap).To(HaveKey("test-cluster-preference"))
			preferenceNode := childMap["test-cluster-preference"]
			Expect(preferenceNode.ObjectReference.Kind).To(Equal("VirtualMachineClusterPreference"))
			Expect(*preferenceNode.ObjectReference.Namespace).To(Equal(""))
		})

		It("should handle VM with multiple access credentials", func() {
			vm.Spec.Template.Spec.AccessCredentials = []v1.AccessCredential{
				{
					SSHPublicKey: &v1.SSHPublicKeyAccessCredential{
						Source: v1.SSHPublicKeyAccessCredentialSource{
							Secret: &v1.AccessCredentialSecretSource{
								SecretName: "ssh-secret",
							},
						},
					},
				},
				{
					UserPassword: &v1.UserPasswordAccessCredential{
						Source: v1.UserPasswordAccessCredentialSource{
							Secret: &v1.AccessCredentialSecretSource{
								SecretName: "password-secret",
							},
						},
					},
				},
			}

			graph := NewObjectGraph(virtClient, &v1.ObjectGraphOptions{})
			graphNodes, err := graph.GetObjectGraph(vm)
			Expect(err).NotTo(HaveOccurred())

			childMap := make(map[string]bool)
			for _, child := range graphNodes.Children {
				if child.ObjectReference.Kind == "Secret" {
					childMap[child.ObjectReference.Name] = true
				}
			}

			Expect(childMap).To(HaveKey("ssh-secret"))
			Expect(childMap).To(HaveKey("password-secret"))
		})

		It("should handle VM without status.created", func() {
			vm.Status.Created = false

			graph := NewObjectGraph(virtClient, &v1.ObjectGraphOptions{})
			graphNodes, err := graph.GetObjectGraph(vm)
			Expect(err).NotTo(HaveOccurred())
			Expect(graphNodes.ObjectReference.Name).To(Equal("test-vm"))

			vmiFound := false
			for _, child := range graphNodes.Children {
				if child.ObjectReference.Kind == "VirtualMachineInstance" {
					vmiFound = true
				}
			}
			Expect(vmiFound).To(BeFalse())
		})

		It("should find launcher pod by owner reference", func() {
			vm.Status.Created = true

			kubeClient.Fake.PrependReactor("list", "pods", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				return true, &k8sv1.PodList{Items: []k8sv1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "virt-launcher-pod-ownerref",
							Namespace: "test-namespace",
							Labels: map[string]string{
								"kubevirt.io": "virt-launcher",
							},
							OwnerReferences: []metav1.OwnerReference{
								{
									Kind: "VirtualMachineInstance",
									Name: vm.Name,
								},
							},
						}}},
				}, nil
			})

			graph := NewObjectGraph(virtClient, &v1.ObjectGraphOptions{})
			graphNodes, err := graph.GetObjectGraph(vm)
			Expect(err).NotTo(HaveOccurred())

			vmiChild := graphNodes.Children[0]
			Expect(vmiChild.Children).To(HaveLen(1))
			Expect(vmiChild.Children[0].ObjectReference.Name).To(Equal("virt-launcher-pod-ownerref"))
		})

		It("should handle pod not found", func() {
			vm.Status.Created = true

			kubeClient.Fake.PrependReactor("list", "pods", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				return true, &k8sv1.PodList{Items: []k8sv1.Pod{}}, nil
			})

			graph := NewObjectGraph(virtClient, &v1.ObjectGraphOptions{})
			graphNodes, err := graph.GetObjectGraph(vm)
			Expect(err).NotTo(HaveOccurred())

			vmiChild := graphNodes.Children[0]
			Expect(vmiChild.Children).To(BeEmpty())
		})

		It("should handle newGraphNode with invalid resource", func() {
			graph := NewObjectGraph(virtClient, &v1.ObjectGraphOptions{})
			node := graph.newGraphNode("test", "default", "invalid-resource", nil, false)
			Expect(node).To(BeNil())
		})
	})

	Context("with options", func() {
		BeforeEach(func() {
			vm.Spec.Instancetype = &v1.InstancetypeMatcher{
				Name: "test-instancetype",
				Kind: "VirtualMachineInstancetype",
			}
			vm.Spec.Preference = &v1.PreferenceMatcher{
				Name: "test-preference",
				Kind: "VirtualMachinePreference",
			}
			vm.Spec.Template.Spec.AccessCredentials = []v1.AccessCredential{
				{
					SSHPublicKey: &v1.SSHPublicKeyAccessCredential{
						Source: v1.SSHPublicKeyAccessCredentialSource{
							Secret: &v1.AccessCredentialSecretSource{
								SecretName: "test-secret",
							},
						},
					},
				},
			}
		})

		It("should exclude optional resources when IncludeOptionalNodes is false", func() {
			options := &v1.ObjectGraphOptions{
				IncludeOptionalNodes: pointer.P(false),
			}
			graph := NewObjectGraph(virtClient, options)
			graphNodes, err := graph.GetObjectGraph(vm)
			Expect(err).NotTo(HaveOccurred())

			childNames := make(map[string]bool)
			for _, child := range graphNodes.Children {
				childNames[child.ObjectReference.Name] = true
			}

			Expect(childNames).NotTo(HaveKey("test-instancetype"))
			Expect(childNames).NotTo(HaveKey("test-preference"))
			Expect(childNames).To(HaveKey("test-secret"))
		})

		It("should filter by label selector for config dependencies", func() {
			options := &v1.ObjectGraphOptions{
				LabelSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						ObjectGraphDependencyLabel: "config",
					},
				},
			}
			graph := NewObjectGraph(virtClient, options)
			graphNodes, err := graph.GetObjectGraph(vm)
			Expect(err).NotTo(HaveOccurred())

			// Should only return config-related nodes
			for _, child := range graphNodes.Children {
				Expect(child.Labels[ObjectGraphDependencyLabel]).To(Equal("config"))
			}
		})

		It("should filter by label selector for storage dependencies", func() {
			vm.Spec.Template.Spec.Volumes = []v1.Volume{
				{
					Name: "test-volume",
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
							PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
								ClaimName: "test-pvc",
							},
						},
					},
				},
			}

			options := &v1.ObjectGraphOptions{
				LabelSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						ObjectGraphDependencyLabel: "storage",
					},
				},
			}
			graph := NewObjectGraph(virtClient, options)
			graphNodes, err := graph.GetObjectGraph(vm)
			Expect(err).NotTo(HaveOccurred())

			// Should only return storage-related nodes
			for _, child := range graphNodes.Children {
				Expect(child.Labels[ObjectGraphDependencyLabel]).To(Equal("storage"))
			}
		})
	})
})
