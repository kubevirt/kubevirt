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
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"

	"github.com/emicklei/go-restful/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"go.uber.org/mock/gomock"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	kubevirtfake "kubevirt.io/client-go/kubevirt/fake"
	templatev1beta1 "kubevirt.io/virt-template-api/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/testutils"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
)

var _ = Describe("Object Graph", func() {
	var (
		kvClient   *kubecli.MockKubevirtClient
		kubeClient *fake.Clientset
		vm         *v1.VirtualMachine
	)

	BeforeEach(func() {
		ctrl := gomock.NewController(GinkgoT())
		kvClient = kubecli.NewMockKubevirtClient(ctrl)
		kubeClient = fake.NewClientset()
		kvClient.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()

		vm = &v1.VirtualMachine{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testVMName,
				Namespace: metav1.NamespaceDefault,
			},
			Spec: v1.VirtualMachineSpec{
				Template: &v1.VirtualMachineInstanceTemplateSpec{
					Spec: v1.VirtualMachineInstanceSpec{},
				},
			},
		}
	})

	Context("endpoint handler", func() {
		var (
			request    *restful.Request
			response   *restful.Response
			recorder   *httptest.ResponseRecorder
			virtClient *kubevirtfake.Clientset
			kv         *v1.KubeVirt
			kvStore    cache.Store
			app        *SubresourceAPIApp
		)

		BeforeEach(func() {
			request = restful.NewRequest(&http.Request{})
			request.PathParameters()["name"] = testVMName
			request.PathParameters()["namespace"] = metav1.NamespaceDefault
			recorder = httptest.NewRecorder()
			response = restful.NewResponse(recorder)
			backend := ghttp.NewTLSServer()
			backendAddr := strings.Split(backend.Addr(), ":")
			backendPort, err := strconv.Atoi(backendAddr[1])
			Expect(err).ToNot(HaveOccurred())

			virtClient = kubevirtfake.NewSimpleClientset()
			kvClient.EXPECT().VirtualMachineInstance(metav1.NamespaceDefault).Return(virtClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault)).AnyTimes()
			kvClient.EXPECT().VirtualMachine(metav1.NamespaceDefault).Return(virtClient.KubevirtV1().VirtualMachines(metav1.NamespaceDefault)).AnyTimes()

			kv = &v1.KubeVirt{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kubevirt",
					Namespace: "kubevirt",
				},
				Spec: v1.KubeVirtSpec{
					Configuration: v1.KubeVirtConfiguration{
						DeveloperConfiguration: &v1.DeveloperConfiguration{
							FeatureGates: []string{featuregate.ObjectGraph},
						},
					},
				},
				Status: v1.KubeVirtStatus{
					Phase: v1.KubeVirtPhaseDeployed,
				},
			}
			var config *virtconfig.ClusterConfig
			config, _, kvStore = testutils.NewFakeClusterConfigUsingKV(kv)
			app = NewSubresourceAPIApp(kvClient, kubeClient, backendPort, &tls.Config{InsecureSkipVerify: true}, config)
		})

		disableFeatureGates := func() {
			newKV := kv.DeepCopy()
			newKV.Spec.Configuration.DeveloperConfiguration.FeatureGates = []string{}
			testutils.UpdateFakeKubeVirtClusterConfig(kvStore, newKV)
		}

		AfterEach(func() {
			testutils.UpdateFakeKubeVirtClusterConfig(kvStore, kv)
		})

		When("VMIObjectGraph API request arrives", func() {
			It("should return an error if the FG is not enabled", func() {
				disableFeatureGates()
				app.VMIObjectGraph(request, response)
				ExpectStatusErrorWithCode(recorder, http.StatusBadRequest)
				ExpectMessage(recorder, Equal("ObjectGraph feature gate not enabled: Unable to return object graph."))
			})

			It("should return an error if the VMI is not found", func() {
				app.VMIObjectGraph(request, response)
				ExpectStatusErrorWithCode(recorder, http.StatusNotFound)
				ExpectMessage(recorder, Equal("virtualmachineinstance.kubevirt.io \"testvm\" not found"))
			})
		})

		When("VMObjectGraph API request arrives", func() {
			It("should return an error if the FG is not enabled", func() {
				disableFeatureGates()
				app.VMObjectGraph(request, response)
				ExpectStatusErrorWithCode(recorder, http.StatusBadRequest)
				ExpectMessage(recorder, Equal("ObjectGraph feature gate not enabled: Unable to return object graph."))
			})

			It("should return an error if the VM is not found", func() {
				app.VMObjectGraph(request, response)
				ExpectStatusErrorWithCode(recorder, http.StatusNotFound)
				ExpectMessage(recorder, Equal("virtualmachine.kubevirt.io \"testvm\" not found"))
			})
		})

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
								{
									Name: "serviceAccount",
									VolumeSource: v1.VolumeSource{
										ServiceAccount: &v1.ServiceAccountVolumeSource{
											ServiceAccountName: "test-serviceAccount",
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

			graph := NewObjectGraph(kvClient, kubeClient, &v1.ObjectGraphOptions{})
			graphNodes, err := graph.GetObjectGraph(vm)
			Expect(err).NotTo(HaveOccurred())
			Expect(graphNodes.Children).To(HaveLen(5))
			Expect(graphNodes.ObjectReference.Name).To(Equal("test-vm"))
			Expect(graphNodes.Children[0].ObjectReference.Name).To(Equal("test-vm"))
			// Child nodes of the VMI
			Expect(graphNodes.Children[0].Children).To(HaveLen(1))
			Expect(graphNodes.Children[0].Children[0].ObjectReference.Name).To(Equal("virt-launcher-test-vmi"))

			Expect(graphNodes.Children[1].ObjectReference.Name).To(Equal("test-ssh-secret"))
			Expect(graphNodes.Children[2].ObjectReference.Name).To(Equal("test-root-disk-pvc"))
			Expect(graphNodes.Children[3].ObjectReference.Name).To(Equal("test-datavolume"))
			Expect(graphNodes.Children[3].ObjectReference.Kind).To(Equal("DataVolume"))
			Expect(graphNodes.Children[4].ObjectReference.Name).To(Equal("test-serviceAccount"))
			Expect(graphNodes.Children[4].ObjectReference.Kind).To(Equal("ServiceAccount"))
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

			graph := NewObjectGraph(kvClient, kubeClient, &v1.ObjectGraphOptions{})
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
			graph := NewObjectGraph(kvClient, kubeClient, &v1.ObjectGraphOptions{})
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

			graph := NewObjectGraph(kvClient, kubeClient, &v1.ObjectGraphOptions{})
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

			graph := NewObjectGraph(kvClient, kubeClient, &v1.ObjectGraphOptions{})
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

			graph := NewObjectGraph(kvClient, kubeClient, &v1.ObjectGraphOptions{})
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

			graph := NewObjectGraph(kvClient, kubeClient, &v1.ObjectGraphOptions{})
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

			graph := NewObjectGraph(kvClient, kubeClient, &v1.ObjectGraphOptions{})
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

			graph := NewObjectGraph(kvClient, kubeClient, &v1.ObjectGraphOptions{})
			graphNodes, err := graph.GetObjectGraph(vm)
			Expect(err).NotTo(HaveOccurred())
			Expect(graphNodes.ObjectReference.Name).To(Equal(testVMName))

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

			graph := NewObjectGraph(kvClient, kubeClient, &v1.ObjectGraphOptions{})
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

			graph := NewObjectGraph(kvClient, kubeClient, &v1.ObjectGraphOptions{})
			graphNodes, err := graph.GetObjectGraph(vm)
			Expect(err).NotTo(HaveOccurred())

			vmiChild := graphNodes.Children[0]
			Expect(vmiChild.Children).To(BeEmpty())
		})

		It("should handle newGraphNode with invalid resource", func() {
			graph := NewObjectGraph(kvClient, kubeClient, &v1.ObjectGraphOptions{})
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
			graph := NewObjectGraph(kvClient, kubeClient, options)
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
			graph := NewObjectGraph(kvClient, kubeClient, options)
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
			graph := NewObjectGraph(kvClient, kubeClient, options)
			graphNodes, err := graph.GetObjectGraph(vm)
			Expect(err).NotTo(HaveOccurred())

			// Should only return storage-related nodes
			for _, child := range graphNodes.Children {
				Expect(child.Labels[ObjectGraphDependencyLabel]).To(Equal("storage"))
			}
		})
	})

	Context("Network handling", func() {
		It("should handle VM with Multus network attachment definitions", func() {
			vm.Spec.Template.Spec.Networks = []v1.Network{
				{
					Name: "default",
					NetworkSource: v1.NetworkSource{
						Pod: &v1.PodNetwork{},
					},
				},
				{
					Name: "multus-net",
					NetworkSource: v1.NetworkSource{
						Multus: &v1.MultusNetwork{
							NetworkName: "test-nad",
						},
					},
				},
			}

			graph := NewObjectGraph(kvClient, kubeClient, &v1.ObjectGraphOptions{})
			graphNodes, err := graph.GetObjectGraph(vm)
			Expect(err).NotTo(HaveOccurred())

			// Check for NetworkAttachmentDefinition in children
			childMap := make(map[string]string)
			for _, child := range graphNodes.Children {
				childMap[child.ObjectReference.Name] = child.ObjectReference.Kind
			}

			Expect(childMap).To(HaveKey("test-nad"))
			Expect(childMap["test-nad"]).To(Equal("NetworkAttachmentDefinition"))
		})

		It("should filter network dependencies by label selector", func() {
			vm.Spec.Template.Spec.Networks = []v1.Network{
				{
					Name: "multus-net",
					NetworkSource: v1.NetworkSource{
						Multus: &v1.MultusNetwork{
							NetworkName: "test-nad",
						},
					},
				},
			}

			options := &v1.ObjectGraphOptions{
				LabelSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						ObjectGraphDependencyLabel: "network",
					},
				},
			}

			graph := NewObjectGraph(kvClient, kubeClient, options)
			graphNodes, err := graph.GetObjectGraph(vm)
			Expect(err).NotTo(HaveOccurred())

			// Should only return network-related nodes
			for _, child := range graphNodes.Children {
				Expect(child.Labels[ObjectGraphDependencyLabel]).To(Equal("network"))
			}
		})
	})

	Context("with VirtualMachineTemplate", func() {
		var embeddedVM *v1.VirtualMachine

		BeforeEach(func() {
			embeddedVM = &v1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-vm",
				},
				Spec: v1.VirtualMachineSpec{
					Template: &v1.VirtualMachineInstanceTemplateSpec{
						Spec: v1.VirtualMachineInstanceSpec{
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
						},
					},
				},
			}
		})

		newVMTemplate := func(namespace string, embeddedVM *v1.VirtualMachine) *templatev1beta1.VirtualMachineTemplate {
			raw, err := json.Marshal(embeddedVM)
			Expect(err).NotTo(HaveOccurred())
			return &templatev1beta1.VirtualMachineTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-template",
					Namespace: namespace,
				},
				Spec: templatev1beta1.VirtualMachineTemplateSpec{
					VirtualMachine: &runtime.RawExtension{Raw: raw},
				},
			}
		}

		It("should generate the object graph for a VirtualMachineTemplate", func() {
			tpl := newVMTemplate("template-namespace", embeddedVM)

			graph := NewObjectGraph(kvClient, kubeClient, &v1.ObjectGraphOptions{})
			graphNodes, err := graph.GetObjectGraph(tpl)
			Expect(err).NotTo(HaveOccurred())

			Expect(graphNodes.ObjectReference.Kind).To(Equal("VirtualMachineTemplate"))
			Expect(*graphNodes.ObjectReference.APIGroup).To(Equal("template.kubevirt.io"))
			Expect(graphNodes.ObjectReference.Name).To(Equal("test-template"))
			Expect(*graphNodes.ObjectReference.Namespace).To(Equal("template-namespace"))
			Expect(graphNodes.Labels).To(HaveKeyWithValue(ObjectGraphDependencyLabel, string(DependencyTypeConfig)))

			childMap := make(map[string]v1.ObjectGraphNode)
			for _, child := range graphNodes.Children {
				childMap[child.ObjectReference.Name] = child
			}

			Expect(childMap).To(HaveKey("test-ssh-secret"))
			Expect(childMap["test-ssh-secret"].ObjectReference.Kind).To(Equal("Secret"))
			Expect(*childMap["test-ssh-secret"].ObjectReference.Namespace).To(Equal("template-namespace"))

			Expect(childMap).To(HaveKey("test-root-disk-pvc"))
			Expect(childMap["test-root-disk-pvc"].ObjectReference.Kind).To(Equal("PersistentVolumeClaim"))
		})

		It("should not emit a VirtualMachineInstance node even if the embedded status claims Created", func() {
			embeddedVM.Status.Created = true
			tpl := newVMTemplate("template-namespace", embeddedVM)

			kubeClient.Fake.PrependReactor("list", "pods", func(action testing.Action) (bool, runtime.Object, error) {
				Fail("launcher pod lookup should not happen for a VirtualMachineTemplate")
				return true, nil, nil
			})

			graph := NewObjectGraph(kvClient, kubeClient, &v1.ObjectGraphOptions{})
			graphNodes, err := graph.GetObjectGraph(tpl)
			Expect(err).NotTo(HaveOccurred())

			for _, child := range graphNodes.Children {
				Expect(child.ObjectReference.Kind).NotTo(Equal("VirtualMachineInstance"))
			}
		})

		It("should not look up a backend storage PVC even if the embedded VM requires persistent TPM", func() {
			embeddedVM.Spec.Template.Spec.Domain.Devices.TPM = &v1.TPMDevice{
				Persistent: pointer.P(true),
			}
			tpl := newVMTemplate("template-namespace", embeddedVM)

			// A decoy backend PVC that would (wrongly) match if the template path ever
			// performed a live backend-storage lookup keyed by the embedded VM's name.
			decoyPVC := k8sv1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "unrelated-backend-storage-pvc",
					Namespace: "template-namespace",
					Labels: map[string]string{
						"persistent-state-for": embeddedVM.Name,
					},
				},
			}
			kubeClient.Fake.PrependReactor("list", "persistentvolumeclaims", func(action testing.Action) (bool, runtime.Object, error) {
				return true, &k8sv1.PersistentVolumeClaimList{Items: []k8sv1.PersistentVolumeClaim{decoyPVC}}, nil
			})

			graph := NewObjectGraph(kvClient, kubeClient, &v1.ObjectGraphOptions{})
			graphNodes, err := graph.GetObjectGraph(tpl)
			Expect(err).NotTo(HaveOccurred())

			for _, child := range graphNodes.Children {
				Expect(child.ObjectReference.Name).NotTo(Equal("unrelated-backend-storage-pvc"))
			}
		})

		It("should mark nodes with unresolved template parameters as optional", func() {
			embeddedVM.Spec.Template.Spec.Volumes[0].PersistentVolumeClaim.ClaimName = "${DISK_NAME}"
			tpl := newVMTemplate("template-namespace", embeddedVM)

			graph := NewObjectGraph(kvClient, kubeClient, &v1.ObjectGraphOptions{})
			graphNodes, err := graph.GetObjectGraph(tpl)
			Expect(err).NotTo(HaveOccurred())

			childMap := make(map[string]v1.ObjectGraphNode)
			for _, child := range graphNodes.Children {
				childMap[child.ObjectReference.Name] = child
			}

			Expect(childMap).To(HaveKey("${DISK_NAME}"))
			Expect(childMap["${DISK_NAME}"].Optional).NotTo(BeNil())
			Expect(*childMap["${DISK_NAME}"].Optional).To(BeTrue())

			options := &v1.ObjectGraphOptions{IncludeOptionalNodes: pointer.P(false)}
			graph = NewObjectGraph(kvClient, kubeClient, options)
			graphNodes, err = graph.GetObjectGraph(tpl)
			Expect(err).NotTo(HaveOccurred())
			for _, child := range graphNodes.Children {
				Expect(child.ObjectReference.Name).NotTo(Equal("${DISK_NAME}"))
			}
		})

		It("should mark nodes using the non-string ${{PARAM}} form as optional", func() {
			// The engine substitutes both "${NAME}" and "${{NAME}}", so either form left in a
			// name means the node cannot correspond to an object that exists today.
			embeddedVM.Spec.Template.Spec.Volumes[0].PersistentVolumeClaim.ClaimName = "${{DISK_NAME}}"
			tpl := newVMTemplate("template-namespace", embeddedVM)

			graph := NewObjectGraph(kvClient, kubeClient, &v1.ObjectGraphOptions{})
			graphNodes, err := graph.GetObjectGraph(tpl)
			Expect(err).NotTo(HaveOccurred())

			childMap := make(map[string]v1.ObjectGraphNode)
			for _, child := range graphNodes.Children {
				childMap[child.ObjectReference.Name] = child
			}

			Expect(childMap).To(HaveKey("${{DISK_NAME}}"))
			Expect(childMap["${{DISK_NAME}}"].Optional).NotTo(BeNil())
			Expect(*childMap["${{DISK_NAME}}"].Optional).To(BeTrue())
		})

		It("should not treat a non-parameterized namespace as unresolved", func() {
			embeddedVM.Namespace = "Hardcoded_Elsewhere"
			tpl := newVMTemplate("template-namespace", embeddedVM)

			graph := NewObjectGraph(kvClient, kubeClient, &v1.ObjectGraphOptions{})
			graphNodes, err := graph.GetObjectGraph(tpl)
			Expect(err).NotTo(HaveOccurred())
			Expect(graphNodes.Children).NotTo(BeEmpty())

			for _, child := range graphNodes.Children {
				Expect(child.Optional).To(BeNil(), "node %s should not be optional", child.ObjectReference.Name)
				Expect(*child.ObjectReference.Namespace).To(Equal("template-namespace"))
			}
		})

		It("should not treat a hardcoded non-DNS1123 resource name as unresolved", func() {
			embeddedVM.Spec.Template.Spec.Volumes = append(embeddedVM.Spec.Template.Spec.Volumes, v1.Volume{
				Name: "serviceaccount-disk",
				VolumeSource: v1.VolumeSource{
					ServiceAccount: &v1.ServiceAccountVolumeSource{
						ServiceAccountName: "test-ServiceAccount",
					},
				},
			})
			tpl := newVMTemplate("template-namespace", embeddedVM)

			graph := NewObjectGraph(kvClient, kubeClient, &v1.ObjectGraphOptions{})
			graphNodes, err := graph.GetObjectGraph(tpl)
			Expect(err).NotTo(HaveOccurred())

			childMap := make(map[string]v1.ObjectGraphNode)
			for _, child := range graphNodes.Children {
				childMap[child.ObjectReference.Name] = child
			}

			Expect(childMap).To(HaveKey("test-ServiceAccount"))
			Expect(childMap["test-ServiceAccount"].Optional).To(BeNil())
		})

		It("should fall back to the instance type/preference RevisionName since the embedded status is discarded", func() {
			embeddedVM.Spec.Instancetype = &v1.InstancetypeMatcher{
				Name:         "test-instancetype",
				Kind:         "VirtualMachineInstancetype",
				RevisionName: "test-instancetype-revision",
			}
			embeddedVM.Spec.Preference = &v1.PreferenceMatcher{
				Name:         "test-preference",
				Kind:         "VirtualMachinePreference",
				RevisionName: "test-preference-revision",
			}
			embeddedVM.Status.InstancetypeRef = &v1.InstancetypeStatusRef{
				ControllerRevisionRef: &v1.ControllerRevisionRef{Name: "status-instancetype-revision"},
			}
			tpl := newVMTemplate("template-namespace", embeddedVM)

			graph := NewObjectGraph(kvClient, kubeClient, &v1.ObjectGraphOptions{})
			graphNodes, err := graph.GetObjectGraph(tpl)
			Expect(err).NotTo(HaveOccurred())

			childMap := make(map[string]v1.ObjectGraphNode)
			for _, child := range graphNodes.Children {
				childMap[child.ObjectReference.Name] = child
			}

			Expect(childMap).To(HaveKey("test-instancetype"))
			Expect(childMap).To(HaveKey("test-preference"))
			Expect(childMap).To(HaveKey("test-instancetype-revision"))
			Expect(childMap).To(HaveKey("test-preference-revision"))
			Expect(childMap).NotTo(HaveKey("status-instancetype-revision"))
		})

		It("should return an error when the embedded VirtualMachine is syntactically invalid JSON", func() {
			tpl := &templatev1beta1.VirtualMachineTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "bad-json-template",
					Namespace: "template-namespace",
				},
				Spec: templatev1beta1.VirtualMachineTemplateSpec{
					VirtualMachine: &runtime.RawExtension{Raw: []byte("{not valid json")},
				},
			}

			graph := NewObjectGraph(kvClient, kubeClient, &v1.ObjectGraphOptions{})
			_, err := graph.GetObjectGraph(tpl)
			Expect(err).To(MatchError(ContainSubstring("failed to decode embedded VirtualMachine")))
		})

		It("should not fail decoding when an unrelated field holds an unresolved non-string ${{PARAM}} value", func() {
			vmJSON, err := json.Marshal(embeddedVM)
			Expect(err).NotTo(HaveOccurred())

			var vmMap map[string]interface{}
			Expect(json.Unmarshal(vmJSON, &vmMap)).To(Succeed())

			templateSpec := vmMap["spec"].(map[string]interface{})["template"].(map[string]interface{})["spec"].(map[string]interface{})
			templateSpec["domain"] = map[string]interface{}{
				"cpu": map[string]interface{}{"cores": "${{CORES}}"},
			}

			patchedRaw, err := json.Marshal(vmMap)
			Expect(err).NotTo(HaveOccurred())

			tpl := &templatev1beta1.VirtualMachineTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-template",
					Namespace: "template-namespace",
				},
				Spec: templatev1beta1.VirtualMachineTemplateSpec{
					VirtualMachine: &runtime.RawExtension{Raw: patchedRaw},
				},
			}

			graph := NewObjectGraph(kvClient, kubeClient, &v1.ObjectGraphOptions{})
			graphNodes, err := graph.GetObjectGraph(tpl)
			Expect(err).NotTo(HaveOccurred())

			childMap := make(map[string]v1.ObjectGraphNode)
			for _, child := range graphNodes.Children {
				childMap[child.ObjectReference.Name] = child
			}

			Expect(childMap).To(HaveKey("test-ssh-secret"))
			Expect(childMap).To(HaveKey("test-root-disk-pvc"))
		})

		It("should mark a nested child optional when its parent DataVolume name is an unresolved parameter", func() {
			embeddedVM.Spec.Template.Spec.Volumes = append(embeddedVM.Spec.Template.Spec.Volumes, v1.Volume{
				Name: "datavolumedisk",
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: "${DV_NAME}",
					},
				},
			})
			tpl := newVMTemplate("template-namespace", embeddedVM)

			graph := NewObjectGraph(kvClient, kubeClient, &v1.ObjectGraphOptions{})
			graphNodes, err := graph.GetObjectGraph(tpl)
			Expect(err).NotTo(HaveOccurred())

			childMap := make(map[string]v1.ObjectGraphNode)
			for _, child := range graphNodes.Children {
				childMap[child.ObjectReference.Name] = child
			}

			Expect(childMap).To(HaveKey("${DV_NAME}"))
			dvNode := childMap["${DV_NAME}"]
			Expect(dvNode.Optional).NotTo(BeNil())
			Expect(*dvNode.Optional).To(BeTrue())

			Expect(dvNode.Children).To(HaveLen(1))
			pvcChild := dvNode.Children[0]
			Expect(pvcChild.ObjectReference.Kind).To(Equal("PersistentVolumeClaim"))
			Expect(pvcChild.Optional).NotTo(BeNil(), "nested PVC child should also be marked optional")
			Expect(*pvcChild.Optional).To(BeTrue())
		})

		It("should mark every dependent resource as optional when the embedded namespace is an unresolved parameter", func() {
			embeddedVM.Namespace = "${TARGET_NAMESPACE}"
			tpl := newVMTemplate("template-namespace", embeddedVM)

			graph := NewObjectGraph(kvClient, kubeClient, &v1.ObjectGraphOptions{})
			graphNodes, err := graph.GetObjectGraph(tpl)
			Expect(err).NotTo(HaveOccurred())
			Expect(graphNodes.Children).NotTo(BeEmpty())

			for _, child := range graphNodes.Children {
				Expect(child.Optional).NotTo(BeNil(), "node %s should be marked optional", child.ObjectReference.Name)
				Expect(*child.Optional).To(BeTrue(), "node %s should be marked optional", child.ObjectReference.Name)
				Expect(*child.ObjectReference.Namespace).To(Equal("${TARGET_NAMESPACE}"),
					"node %s should surface the unresolved namespace instead of the template's namespace", child.ObjectReference.Name)
			}

			options := &v1.ObjectGraphOptions{IncludeOptionalNodes: pointer.P(false)}
			graph = NewObjectGraph(kvClient, kubeClient, options)
			graphNodes, err = graph.GetObjectGraph(tpl)
			Expect(err).NotTo(HaveOccurred())
			Expect(graphNodes.Children).To(BeEmpty())
		})

		It("should return an error when the template has no embedded VirtualMachine", func() {
			tpl := &templatev1beta1.VirtualMachineTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "empty-template",
					Namespace: "template-namespace",
				},
			}

			graph := NewObjectGraph(kvClient, kubeClient, &v1.ObjectGraphOptions{})
			_, err := graph.GetObjectGraph(tpl)
			Expect(err).To(HaveOccurred())
		})

		It("should return an error, not panic, when the embedded VirtualMachine has no spec.template", func() {
			incompleteVM := &v1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{Name: "incomplete-vm"},
			}
			tpl := newVMTemplate("template-namespace", incompleteVM)

			graph := NewObjectGraph(kvClient, kubeClient, &v1.ObjectGraphOptions{})
			var err error
			Expect(func() {
				_, err = graph.GetObjectGraph(tpl)
			}).NotTo(Panic())
			Expect(err).To(HaveOccurred())
		})

		It("should return a childless graph when spec.template is present but empty", func() {
			tpl := &templatev1beta1.VirtualMachineTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "empty-template",
					Namespace: "template-namespace",
				},
				Spec: templatev1beta1.VirtualMachineTemplateSpec{
					VirtualMachine: &runtime.RawExtension{
						Raw: []byte(`{"spec":{"template":{}}}`),
					},
				},
			}

			graph := NewObjectGraph(kvClient, kubeClient, &v1.ObjectGraphOptions{})
			graphNodes, err := graph.GetObjectGraph(tpl)
			Expect(err).NotTo(HaveOccurred())
			Expect(graphNodes.ObjectReference.Name).To(Equal("empty-template"))
			Expect(graphNodes.Children).To(BeEmpty())
		})
	})
})
