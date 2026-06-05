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

package mutators_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	admissionv1 "k8s.io/api/admission/v1"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks/mutating-webhook/mutators"
	"kubevirt.io/kubevirt/pkg/virtiofs"
)

var _ = Describe("VirtLauncherPodMutator", func() {
	var ctrl *gomock.Controller
	var virtClient *kubecli.MockKubevirtClient
	var vmiInterface *kubecli.MockVirtualMachineInstanceInterface

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		vmiInterface = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("when pod is not a virt-launcher", func() {
		It("should allow without mutation", func() {
			pod := &k8sv1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "regular-pod",
					Namespace: "default",
				},
			}

			clusterConfig, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{})
			mutator := mutators.NewVirtLauncherPodMutator(clusterConfig, virtClient)

			ar := createPodAdmissionReview(pod)
			response := mutator.Mutate(ar)

			Expect(response.Allowed).To(BeTrue())
			Expect(response.Patch).To(BeNil())
		})
	})

	Context("when virt-launcher pod has no containerPath volumes", func() {
		It("should allow without mutation", func() {
			vmi := newVMIWithoutContainerPath()
			pod := newVirtLauncherPod(vmi)

			virtClient.EXPECT().VirtualMachineInstance(pod.Namespace).Return(vmiInterface)
			vmiInterface.EXPECT().Get(gomock.Any(), vmi.Name, gomock.Any()).Return(vmi, nil)

			clusterConfig, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{})
			mutator := mutators.NewVirtLauncherPodMutator(clusterConfig, virtClient)

			ar := createPodAdmissionReview(pod)
			response := mutator.Mutate(ar)

			Expect(response.Allowed).To(BeTrue())
			Expect(response.Patch).To(BeNil())
		})
	})

	Context("when virt-launcher pod has containerPath volumes", func() {
		DescribeTable("should handle readOnly field correctly", func(readOnly *bool, expectInjection bool) {
			vmi := newVMIWithContainerPath()
			vmi.Spec.Volumes[0].ContainerPath.ReadOnly = readOnly
			pod := newVirtLauncherPodWithVolumeMounts(vmi)

			virtClient.EXPECT().VirtualMachineInstance(pod.Namespace).Return(vmiInterface)
			vmiInterface.EXPECT().Get(gomock.Any(), vmi.Name, gomock.Any()).Return(vmi, nil)

			clusterConfig, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{})
			mutator := mutators.NewVirtLauncherPodMutator(clusterConfig, virtClient)

			ar := createPodAdmissionReview(pod)
			response := mutator.Mutate(ar)

			Expect(response.Allowed).To(BeTrue())
			if expectInjection {
				Expect(response.Patch).ToNot(BeNil())
				var patches []map[string]any
				Expect(json.Unmarshal(response.Patch, &patches)).To(Succeed())
				Expect(patches).To(HaveLen(1))
				Expect(patches[0]["op"]).To(Equal("add"))
				container := patches[0]["value"].(map[string]any)
				Expect(container["name"]).To(Equal(virtiofs.ContainerPathVirtiofsContainerName("token-volume")))
			} else {
				Expect(response.Patch).To(BeNil())
			}
		},
			Entry("should inject when readOnly is true", pointer.P(true), true),
			Entry("should not inject when readOnly is false", pointer.P(false), false),
			Entry("should not inject when readOnly is nil", nil, false),
		)

		It("should not mutate if virtiofs container already exists", func() {
			vmi := newVMIWithContainerPath()
			pod := newVirtLauncherPodWithVolumeMounts(vmi)

			// Add the virtiofs container to simulate idempotency
			pod.Spec.Containers = append(pod.Spec.Containers, k8sv1.Container{
				Name: virtiofs.ContainerPathVirtiofsContainerName("token-volume"),
			})

			virtClient.EXPECT().VirtualMachineInstance(pod.Namespace).Return(vmiInterface)
			vmiInterface.EXPECT().Get(gomock.Any(), vmi.Name, gomock.Any()).Return(vmi, nil)

			clusterConfig, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{})
			mutator := mutators.NewVirtLauncherPodMutator(clusterConfig, virtClient)

			ar := createPodAdmissionReview(pod)
			response := mutator.Mutate(ar)

			Expect(response.Allowed).To(BeTrue())
			Expect(response.Patch).To(BeNil())
		})
	})
})

func createPodAdmissionReview(pod *k8sv1.Pod) *admissionv1.AdmissionReview {
	podBytes, err := json.Marshal(pod)
	Expect(err).ToNot(HaveOccurred())

	return &admissionv1.AdmissionReview{
		Request: &admissionv1.AdmissionRequest{
			Resource: metav1.GroupVersionResource{
				Group:    "",
				Version:  "v1",
				Resource: "pods",
			},
			Object: runtime.RawExtension{
				Raw: podBytes,
			},
		},
	}
}

func newVMIWithoutContainerPath() *v1.VirtualMachineInstance {
	return &v1.VirtualMachineInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-vmi",
			Namespace: "default",
			UID:       "test-uid",
		},
		Spec: v1.VirtualMachineInstanceSpec{
			Domain: v1.DomainSpec{
				Devices: v1.Devices{},
			},
		},
	}
}

func newVMIWithContainerPath() *v1.VirtualMachineInstance {
	return &v1.VirtualMachineInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-vmi",
			Namespace: "default",
			UID:       "test-uid",
		},
		Spec: v1.VirtualMachineInstanceSpec{
			Domain: v1.DomainSpec{
				Devices: v1.Devices{
					Filesystems: []v1.Filesystem{
						{
							Name:     "token-volume",
							Virtiofs: &v1.FilesystemVirtiofs{},
						},
					},
				},
			},
			Volumes: []v1.Volume{
				{
					Name: "token-volume",
					VolumeSource: v1.VolumeSource{
						ContainerPath: &v1.ContainerPathVolumeSource{
							Path:     "/var/run/secrets/token",
							ReadOnly: pointer.P(true),
						},
					},
				},
			},
		},
	}
}

func newVirtLauncherPod(vmi *v1.VirtualMachineInstance) *k8sv1.Pod {
	return &k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "virt-launcher-" + vmi.Name,
			Namespace: vmi.Namespace,
			Labels: map[string]string{
				v1.AppLabel: "virt-launcher",
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: v1.VirtualMachineInstanceGroupVersionKind.GroupVersion().String(),
					Kind:       v1.VirtualMachineInstanceGroupVersionKind.Kind,
					Name:       vmi.Name,
					UID:        vmi.UID,
				},
			},
		},
		Spec: k8sv1.PodSpec{
			Containers: []k8sv1.Container{
				{
					Name:  "compute",
					Image: "virt-launcher:latest",
				},
			},
		},
	}
}

func newVirtLauncherPodWithVolumeMounts(vmi *v1.VirtualMachineInstance) *k8sv1.Pod {
	pod := newVirtLauncherPod(vmi)

	// Add the virtiofs socket volume
	pod.Spec.Volumes = append(pod.Spec.Volumes, k8sv1.Volume{
		Name: virtiofs.VirtioFSContainers,
		VolumeSource: k8sv1.VolumeSource{
			EmptyDir: &k8sv1.EmptyDirVolumeSource{},
		},
	})

	// Add the token volume that would be injected by an external mutator
	pod.Spec.Volumes = append(pod.Spec.Volumes, k8sv1.Volume{
		Name: "aws-iam-token",
		VolumeSource: k8sv1.VolumeSource{
			Projected: &k8sv1.ProjectedVolumeSource{},
		},
	})

	// Add volume mount to compute container
	pod.Spec.Containers[0].VolumeMounts = []k8sv1.VolumeMount{
		{
			Name:      virtiofs.VirtioFSContainers,
			MountPath: virtiofs.VirtioFSContainersMountBaseDir,
		},
		{
			Name:      "aws-iam-token",
			MountPath: "/var/run/secrets/token",
			ReadOnly:  true,
		},
	}

	return pod
}
