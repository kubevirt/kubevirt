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

package render

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	virtv1 "kubevirt.io/api/core/v1"
)

var _ = Describe("Render", func() {

	Describe("PodFromVM", func() {
		It("should render a pod from a basic containerDisk VM", func() {
			vm := &virtv1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testvm",
					Namespace: "default",
				},
				Spec: virtv1.VirtualMachineSpec{
					Template: &virtv1.VirtualMachineInstanceTemplateSpec{
						Spec: virtv1.VirtualMachineInstanceSpec{
							Domain: virtv1.DomainSpec{
								Resources: virtv1.ResourceRequirements{
									Requests: k8sv1.ResourceList{
										k8sv1.ResourceMemory: resource.MustParse("128Mi"),
									},
								},
							},
							Volumes: []virtv1.Volume{
								{
									Name: "containerdisk",
									VolumeSource: virtv1.VolumeSource{
										ContainerDisk: &virtv1.ContainerDiskSource{
											Image: "quay.io/kubevirt/cirros-container-disk-demo:latest",
										},
									},
								},
							},
						},
					},
				},
			}

			pod, err := PodFromVM(vm, Options{
				LauncherImage: "quay.io/kubevirt/virt-launcher:v1.8.0",
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(pod).ToNot(BeNil())
			Expect(pod.Name).ToNot(BeEmpty())

			computeFound := false
			for _, c := range pod.Spec.Containers {
				if c.Name == "compute" {
					computeFound = true
					Expect(c.Image).To(Equal("quay.io/kubevirt/virt-launcher:v1.8.0"))
					break
				}
			}
			Expect(computeFound).To(BeTrue(), "compute container should exist")
		})

		It("should use custom launcher image", func() {
			vm := minimalVM("custom-image-vm")

			pod, err := PodFromVM(vm, Options{
				LauncherImage: "my-registry/my-launcher:v2.0.0",
			})
			Expect(err).ToNot(HaveOccurred())

			for _, c := range pod.Spec.Containers {
				if c.Name == "compute" {
					Expect(c.Image).To(Equal("my-registry/my-launcher:v2.0.0"))
					break
				}
			}
		})

		It("should handle VM with empty namespace", func() {
			vm := minimalVM("no-ns-vm")
			vm.Namespace = ""

			pod, err := PodFromVM(vm, Options{})
			Expect(err).ToNot(HaveOccurred())
			Expect(pod).ToNot(BeNil())
		})

		It("should produce stable output for the same input", func() {
			vm := minimalVM("stable-vm")

			pod1, err := PodFromVM(vm.DeepCopy(), Options{})
			Expect(err).ToNot(HaveOccurred())

			pod2, err := PodFromVM(vm.DeepCopy(), Options{})
			Expect(err).ToNot(HaveOccurred())

			Expect(pod1.Name).To(Equal(pod2.Name))
		})

		It("should handle PVC volumes", func() {
			vm := &virtv1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pvc-vm",
					Namespace: "default",
				},
				Spec: virtv1.VirtualMachineSpec{
					Template: &virtv1.VirtualMachineInstanceTemplateSpec{
						Spec: virtv1.VirtualMachineInstanceSpec{
							Domain: virtv1.DomainSpec{
								Resources: virtv1.ResourceRequirements{
									Requests: k8sv1.ResourceList{
										k8sv1.ResourceMemory: resource.MustParse("128Mi"),
									},
								},
								Devices: virtv1.Devices{
									Disks: []virtv1.Disk{
										{Name: "datadisk", DiskDevice: virtv1.DiskDevice{Disk: &virtv1.DiskTarget{Bus: virtv1.DiskBusVirtio}}},
									},
								},
							},
							Volumes: []virtv1.Volume{
								{
									Name: "datadisk",
									VolumeSource: virtv1.VolumeSource{
										PersistentVolumeClaim: &virtv1.PersistentVolumeClaimVolumeSource{
											PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
												ClaimName: "my-pvc",
											},
										},
									},
								},
							},
						},
					},
				},
			}

			pod, err := PodFromVM(vm, Options{})
			Expect(err).ToNot(HaveOccurred())

			found := false
			for _, v := range pod.Spec.Volumes {
				if v.PersistentVolumeClaim != nil && v.PersistentVolumeClaim.ClaimName == "my-pvc" {
					found = true
					break
				}
			}
			Expect(found).To(BeTrue(), "pod should have PVC volume for my-pvc")
		})
	})

	Describe("PodFromVMI", func() {
		It("should render a pod from a VMI directly", func() {
			vmi := &virtv1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testvmi",
					Namespace: "default",
				},
				Spec: virtv1.VirtualMachineInstanceSpec{
					Domain: virtv1.DomainSpec{
						Resources: virtv1.ResourceRequirements{
							Requests: k8sv1.ResourceList{
								k8sv1.ResourceMemory: resource.MustParse("64Mi"),
							},
						},
					},
				},
			}

			pod, err := PodFromVMI(vmi, Options{})
			Expect(err).ToNot(HaveOccurred())
			Expect(pod).ToNot(BeNil())
			Expect(pod.Name).ToNot(BeEmpty())
		})
	})
})

func minimalVM(name string) *virtv1.VirtualMachine {
	return &virtv1.VirtualMachine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
		},
		Spec: virtv1.VirtualMachineSpec{
			Template: &virtv1.VirtualMachineInstanceTemplateSpec{
				Spec: virtv1.VirtualMachineInstanceSpec{
					Domain: virtv1.DomainSpec{
						Resources: virtv1.ResourceRequirements{
							Requests: k8sv1.ResourceList{
								k8sv1.ResourceMemory: resource.MustParse("64Mi"),
							},
						},
					},
				},
			},
		},
	}
}
