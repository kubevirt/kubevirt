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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package services_test

import (
	"errors"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	kubev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/hooks"
	"kubevirt.io/kubevirt/pkg/log"
	. "kubevirt.io/kubevirt/pkg/virt-controller/services"
)

const namespaceKubevirt = "kubevirt"

var _ = Describe("Template", func() {

	log.Log.SetIOWriter(GinkgoWriter)
	configCache := cache.NewIndexer(cache.DeletionHandlingMetaNamespaceKeyFunc, nil)
	pvcCache := cache.NewIndexer(cache.DeletionHandlingMetaNamespaceKeyFunc, nil)
	svc := NewTemplateService("kubevirt/virt-launcher",
		"/var/run/kubevirt",
		"/var/run/kubevirt-ephemeral-disks",
		"pull-secret-1",
		configCache,
		pvcCache)

	Describe("Rendering", func() {
		Context("launch template with correct parameters", func() {
			It("should work", func() {
				trueVar := true
				annotations := map[string]string{
					hooks.HookSidecarListAnnotationName: `[{"image": "some-image:v1", "imagePullPolicy": "IfNotPresent"}]`,
				}
				pod, err := svc.RenderLaunchManifest(&v1.VirtualMachineInstance{ObjectMeta: metav1.ObjectMeta{Name: "testvmi", Namespace: "testns", UID: "1234", Annotations: annotations}, Spec: v1.VirtualMachineInstanceSpec{Domain: v1.DomainSpec{}}})
				Expect(err).ToNot(HaveOccurred())

				Expect(len(pod.Spec.Containers)).To(Equal(2))
				Expect(pod.Spec.Containers[0].Image).To(Equal("kubevirt/virt-launcher"))
				Expect(pod.ObjectMeta.Labels).To(Equal(map[string]string{
					v1.AppLabel:       "virt-launcher",
					v1.CreatedByLabel: "1234",
				}))
				Expect(pod.ObjectMeta.Annotations).To(Equal(map[string]string{
					v1.DomainAnnotation:  "testvmi",
					v1.OwnedByAnnotation: "virt-controller",
				}))
				Expect(pod.ObjectMeta.OwnerReferences).To(Equal([]metav1.OwnerReference{{
					APIVersion:         v1.VirtualMachineInstanceGroupVersionKind.GroupVersion().String(),
					Kind:               v1.VirtualMachineInstanceGroupVersionKind.Kind,
					Name:               "testvmi",
					UID:                "1234",
					Controller:         &trueVar,
					BlockOwnerDeletion: &trueVar,
				}}))
				Expect(pod.ObjectMeta.GenerateName).To(Equal("virt-launcher-testvmi-"))
				Expect(pod.Spec.NodeSelector).To(Equal(map[string]string{
					v1.NodeSchedulable: "true",
				}))
				Expect(pod.Spec.Containers[0].Command).To(Equal([]string{"/usr/bin/virt-launcher",
					"--qemu-timeout", "5m",
					"--name", "testvmi",
					"--uid", "1234",
					"--namespace", "testns",
					"--kubevirt-share-dir", "/var/run/kubevirt",
					"--ephemeral-disk-dir", "/var/run/kubevirt-ephemeral-disks",
					"--readiness-file", "/tmp/healthy",
					"--grace-period-seconds", "45",
					"--hook-sidecars", "1"}))
				Expect(pod.Spec.Containers[1].Name).To(Equal("hook-sidecar-0"))
				Expect(pod.Spec.Containers[1].Image).To(Equal("some-image:v1"))
				Expect(pod.Spec.Containers[1].ImagePullPolicy).To(Equal(kubev1.PullPolicy("IfNotPresent")))
				Expect(*pod.Spec.TerminationGracePeriodSeconds).To(Equal(int64(60)))
				By("setting the right hostname")
				Expect(pod.Spec.Hostname).To(Equal("testvmi"))
				Expect(pod.Spec.Subdomain).To(BeEmpty())
			})
		})
		Context("with multus annotation", func() {
			It("should add multus networks in the pod annotation", func() {
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "testvmi",
						Namespace: "default",
						UID:       "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{
						Domain: v1.DomainSpec{},
						Networks: []v1.Network{
							{Name: "default",
								NetworkSource: v1.NetworkSource{
									Multus: &v1.CniNetwork{NetworkName: "default"},
								}},
							{Name: "test1",
								NetworkSource: v1.NetworkSource{
									Multus: &v1.CniNetwork{NetworkName: "test1"},
								}},
						},
					},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())
				value, ok := pod.Annotations["k8s.v1.cni.cncf.io/networks"]
				Expect(ok).To(Equal(true))
				Expect(value).To(Equal("default,test1"))
			})
		})
		Context("with genie annotation", func() {
			It("should add genie networks in the pod annotation", func() {
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "testvmi",
						Namespace: "default",
						UID:       "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{
						Domain: v1.DomainSpec{},
						Networks: []v1.Network{
							{Name: "default",
								NetworkSource: v1.NetworkSource{
									Genie: &v1.CniNetwork{NetworkName: "default"},
								}},
							{Name: "test1",
								NetworkSource: v1.NetworkSource{
									Genie: &v1.CniNetwork{NetworkName: "test1"},
								}},
						},
					},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())
				value, ok := pod.Annotations["cni"]
				Expect(ok).To(Equal(true))
				Expect(value).To(Equal("default,test1"))
			})
		})
		Context("with node selectors", func() {
			It("should add node selectors to template", func() {

				nodeSelector := map[string]string{
					"kubernetes.io/hostname": "master",
					v1.NodeSchedulable:       "true",
				}
				annotations := map[string]string{
					hooks.HookSidecarListAnnotationName: `[{"image": "some-image:v1", "imagePullPolicy": "IfNotPresent"}]`,
				}
				vmi := v1.VirtualMachineInstance{ObjectMeta: metav1.ObjectMeta{Name: "testvmi", Namespace: "default", UID: "1234", Annotations: annotations}, Spec: v1.VirtualMachineInstanceSpec{NodeSelector: nodeSelector, Domain: v1.DomainSpec{}}}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				Expect(len(pod.Spec.Containers)).To(Equal(2))
				Expect(pod.Spec.Containers[0].Image).To(Equal("kubevirt/virt-launcher"))
				Expect(pod.ObjectMeta.Labels).To(Equal(map[string]string{
					v1.AppLabel:       "virt-launcher",
					v1.CreatedByLabel: "1234",
				}))
				Expect(pod.ObjectMeta.GenerateName).To(Equal("virt-launcher-testvmi-"))
				Expect(pod.Spec.NodeSelector).To(Equal(map[string]string{
					"kubernetes.io/hostname": "master",
					v1.NodeSchedulable:       "true",
				}))
				Expect(pod.Spec.Containers[0].Command).To(Equal([]string{"/usr/bin/virt-launcher",
					"--qemu-timeout", "5m",
					"--name", "testvmi",
					"--uid", "1234",
					"--namespace", "default",
					"--kubevirt-share-dir", "/var/run/kubevirt",
					"--ephemeral-disk-dir", "/var/run/kubevirt-ephemeral-disks",
					"--readiness-file", "/tmp/healthy",
					"--grace-period-seconds", "45",
					"--hook-sidecars", "1"}))
				Expect(pod.Spec.Containers[1].Name).To(Equal("hook-sidecar-0"))
				Expect(pod.Spec.Containers[1].Image).To(Equal("some-image:v1"))
				Expect(pod.Spec.Containers[1].ImagePullPolicy).To(Equal(kubev1.PullPolicy("IfNotPresent")))
				Expect(pod.Spec.Containers[1].VolumeMounts[0].MountPath).To(Equal(hooks.HookSocketsSharedDirectory))
				Expect(pod.Spec.Volumes[0].EmptyDir.Medium).To(Equal(kubev1.StorageMedium("")))
				Expect(pod.Spec.Volumes[1].HostPath.Path).To(Equal("/var/run/kubevirt"))
				Expect(pod.Spec.Containers[0].VolumeMounts[1].MountPath).To(Equal("/var/run/kubevirt"))
				Expect(*pod.Spec.TerminationGracePeriodSeconds).To(Equal(int64(60)))
			})
			It("should add default cpu/memory resources to the sidecar container if cpu pinning was requested", func() {
				nodeSelector := map[string]string{
					"kubernetes.io/hostname": "master",
					v1.NodeSchedulable:       "true",
				}
				annotations := map[string]string{
					hooks.HookSidecarListAnnotationName: `[{"image": "some-image:v1", "imagePullPolicy": "IfNotPresent"}]`,
				}
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "testvmi",
						Namespace:   "default",
						UID:         "1234",
						Annotations: annotations,
					},
					Spec: v1.VirtualMachineInstanceSpec{
						NodeSelector: nodeSelector,
						Domain: v1.DomainSpec{
							CPU: &v1.CPU{
								Cores: 2,
								DedicatedCPUPlacement: true,
							},
						},
					},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())
				cpu := resource.MustParse("200m")
				mem := resource.MustParse("64M")
				Expect(pod.Spec.Containers[1].Resources.Limits.Memory().Cmp(mem)).To(BeZero())
				Expect(pod.Spec.Containers[1].Resources.Limits.Cpu().Cmp(cpu)).To(BeZero())

				found := false
				caps := pod.Spec.Containers[0].SecurityContext.Capabilities
				for _, cap := range caps.Add {
					if cap == CAP_SYS_NICE {
						found = true
					}
				}
				Expect(found).To(BeTrue(), "Expected compute container to be granted SYS_NICE capability")
				Expect(pod.Spec.NodeSelector).Should(HaveKeyWithValue(v1.CPUManager, "true"))
			})
			It("should add node affinity to pod", func() {
				nodeAffinity := kubev1.NodeAffinity{}
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{Name: "testvmi", Namespace: "default", UID: "1234"},
					Spec: v1.VirtualMachineInstanceSpec{
						Affinity: &kubev1.Affinity{NodeAffinity: &nodeAffinity},
						Domain:   v1.DomainSpec{},
					},
				}
				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				Expect(pod.Spec.Affinity).To(BeEquivalentTo(&kubev1.Affinity{NodeAffinity: &nodeAffinity}))
			})

			It("should add pod affinity to pod", func() {
				podAffinity := kubev1.PodAffinity{}
				vm := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{Name: "testvm", Namespace: "default", UID: "1234"},
					Spec: v1.VirtualMachineInstanceSpec{
						Affinity: &kubev1.Affinity{PodAffinity: &podAffinity},
						Domain:   v1.DomainSpec{},
					},
				}
				pod, err := svc.RenderLaunchManifest(&vm)
				Expect(err).ToNot(HaveOccurred())

				Expect(pod.Spec.Affinity).To(BeEquivalentTo(&kubev1.Affinity{PodAffinity: &podAffinity}))
			})

			It("should add pod anti-affinity to pod", func() {
				podAntiAffinity := kubev1.PodAntiAffinity{}
				vm := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{Name: "testvm", Namespace: "default", UID: "1234"},
					Spec: v1.VirtualMachineInstanceSpec{
						Affinity: &kubev1.Affinity{PodAntiAffinity: &podAntiAffinity},
						Domain:   v1.DomainSpec{},
					},
				}
				pod, err := svc.RenderLaunchManifest(&vm)
				Expect(err).ToNot(HaveOccurred())

				Expect(pod.Spec.Affinity).To(BeEquivalentTo(&kubev1.Affinity{PodAntiAffinity: &podAntiAffinity}))
			})

			It("should add tolerations to pod", func() {
				podToleration := kubev1.Toleration{Key: "test"}
				vm := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{Name: "testvm", Namespace: "default", UID: "1234"},
					Spec: v1.VirtualMachineInstanceSpec{
						Tolerations: []kubev1.Toleration{
							{
								Key: podToleration.Key,
							},
						},
						Domain: v1.DomainSpec{},
					},
				}
				pod, err := svc.RenderLaunchManifest(&vm)
				Expect(err).ToNot(HaveOccurred())
				Expect(pod.Spec.Tolerations).To(BeEquivalentTo([]kubev1.Toleration{{Key: podToleration.Key}}))
			})

			It("should use the hostname and subdomain if specified on the vm", func() {
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{Name: "testvm",
						Namespace: "default",
						UID:       "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{
						Domain:    v1.DomainSpec{},
						Hostname:  "myhost",
						Subdomain: "mydomain",
					},
				}
				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())
				Expect(pod.Spec.Hostname).To(Equal(vmi.Spec.Hostname))
				Expect(pod.Spec.Subdomain).To(Equal(vmi.Spec.Subdomain))
			})

			It("should add vmi labels to pod", func() {
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{Name: "testvmi",
						Namespace: "default",
						UID:       "1234",
						Labels: map[string]string{
							"key1": "val1",
							"key2": "val2",
						},
					},
					Spec: v1.VirtualMachineInstanceSpec{
						Domain: v1.DomainSpec{},
					},
				}
				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				Expect(pod.Labels).To(Equal(
					map[string]string{
						"key1":            "val1",
						"key2":            "val2",
						v1.AppLabel:       "virt-launcher",
						v1.CreatedByLabel: "1234",
					},
				))
			})

			It("should not add empty affinity to pod", func() {
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{Name: "testvm", Namespace: "default", UID: "1234"},
					Spec: v1.VirtualMachineInstanceSpec{
						Domain: v1.DomainSpec{},
					},
				}
				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				Expect(pod.Spec.Affinity).To(BeNil())
			})
		})
		Context("with cpu and memory constraints", func() {
			It("should add cpu and memory constraints to a template", func() {

				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "testvmi",
						Namespace: "default",
						UID:       "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{
						Domain: v1.DomainSpec{
							Resources: v1.ResourceRequirements{
								Requests: kubev1.ResourceList{
									kubev1.ResourceCPU:    resource.MustParse("1m"),
									kubev1.ResourceMemory: resource.MustParse("1G"),
								},
								Limits: kubev1.ResourceList{
									kubev1.ResourceCPU:    resource.MustParse("2m"),
									kubev1.ResourceMemory: resource.MustParse("2G"),
								},
							},
						},
					},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				Expect(pod.Spec.Containers[0].Resources.Requests.Cpu().String()).To(Equal("1m"))
				Expect(pod.Spec.Containers[0].Resources.Limits.Cpu().String()).To(Equal("2m"))
				Expect(pod.Spec.Containers[0].Resources.Requests.Memory().String()).To(Equal("1099507557"))
				Expect(pod.Spec.Containers[0].Resources.Limits.Memory().String()).To(Equal("2099507557"))
			})
			It("should overcommit guest overhead if selected, by only adding the overhead to memory limits", func() {

				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "testvmi",
						Namespace: "default",
						UID:       "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{
						Domain: v1.DomainSpec{
							Resources: v1.ResourceRequirements{
								OvercommitGuestOverhead: true,
								Requests: kubev1.ResourceList{
									kubev1.ResourceMemory: resource.MustParse("1G"),
								},
								Limits: kubev1.ResourceList{
									kubev1.ResourceMemory: resource.MustParse("2G"),
								},
							},
						},
					},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				Expect(pod.Spec.Containers[0].Resources.Requests.Memory().String()).To(Equal("1G"))
				Expect(pod.Spec.Containers[0].Resources.Limits.Memory().String()).To(Equal("2099507557"))
			})
			It("should not add unset resources", func() {

				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "testvmi",
						Namespace: "default",
						UID:       "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{
						Domain: v1.DomainSpec{
							CPU: &v1.CPU{Cores: 3},
							Resources: v1.ResourceRequirements{
								Requests: kubev1.ResourceList{
									kubev1.ResourceCPU:    resource.MustParse("1m"),
									kubev1.ResourceMemory: resource.MustParse("64M"),
								},
							},
						},
					},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				Expect(vmi.Spec.Domain.Resources.Requests.Memory().String()).To(Equal("64M"))
				Expect(pod.Spec.Containers[0].Resources.Requests.Cpu().String()).To(Equal("1m"))
				Expect(pod.Spec.Containers[0].Resources.Requests.Memory().ToDec().ScaledValue(resource.Mega)).To(Equal(int64(179)))

				// Limits for KVM and TUN devices should be requested.
				Expect(pod.Spec.Containers[0].Resources.Limits).ToNot(BeNil())
			})

			table.DescribeTable("should check autoattachGraphicsDevicse", func(autoAttach *bool, memory int) {

				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "testvmi",
						Namespace: "default",
						UID:       "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{
						Domain: v1.DomainSpec{

							CPU: &v1.CPU{Cores: 3},
							Resources: v1.ResourceRequirements{
								Requests: kubev1.ResourceList{
									kubev1.ResourceCPU:    resource.MustParse("1m"),
									kubev1.ResourceMemory: resource.MustParse("64M"),
								},
							},
						},
					},
				}
				vmi.Spec.Domain.Devices = v1.Devices{
					AutoattachGraphicsDevice: autoAttach,
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())
				Expect(pod.Spec.Containers[0].Resources.Requests.Memory().ToDec().ScaledValue(resource.Mega)).To(Equal(int64(memory)))
			},
				table.Entry("and consider graphics overhead if it is not set", nil, 179),
				table.Entry("and consider graphics overhead if it is set to true", True(), 179),
				table.Entry("and not consider graphics overhead if it is set to false", False(), 162),
			)
		})

		Context("with hugepages constraints", func() {
			table.DescribeTable("should add to the template constraints ", func(value string) {
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "testvmi",
						Namespace: "default",
						UID:       "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{
						Domain: v1.DomainSpec{
							Memory: &v1.Memory{
								Hugepages: &v1.Hugepages{
									PageSize: value,
								},
							},
							Resources: v1.ResourceRequirements{
								Requests: kubev1.ResourceList{
									kubev1.ResourceMemory: resource.MustParse("64M"),
								},
								Limits: kubev1.ResourceList{
									kubev1.ResourceMemory: resource.MustParse("64M"),
								},
							},
						},
					},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				Expect(pod.Spec.Containers[0].Resources.Requests.Memory().ToDec().ScaledValue(resource.Mega)).To(Equal(int64(98)))
				Expect(pod.Spec.Containers[0].Resources.Limits.Memory().ToDec().ScaledValue(resource.Mega)).To(Equal(int64(98)))

				hugepageType := kubev1.ResourceName(kubev1.ResourceHugePagesPrefix + value)
				hugepagesRequest := pod.Spec.Containers[0].Resources.Requests[hugepageType]
				hugepagesLimit := pod.Spec.Containers[0].Resources.Limits[hugepageType]
				Expect(hugepagesRequest.ToDec().ScaledValue(resource.Mega)).To(Equal(int64(64)))
				Expect(hugepagesLimit.ToDec().ScaledValue(resource.Mega)).To(Equal(int64(64)))

				Expect(len(pod.Spec.Volumes)).To(Equal(4))
				Expect(pod.Spec.Volumes[0].EmptyDir).ToNot(BeNil())
				Expect(pod.Spec.Volumes[0].EmptyDir.Medium).To(Equal(kubev1.StorageMediumHugePages))

				Expect(len(pod.Spec.Containers[0].VolumeMounts)).To(Equal(4))
				Expect(pod.Spec.Containers[0].VolumeMounts[3].MountPath).To(Equal("/dev/hugepages"))
			},
				table.Entry("hugepages-2Mi", "2Mi"),
				table.Entry("hugepages-1Gi", "1Gi"),
			)
		})

		Context("with file mode pvc source", func() {
			It("should add volume to template", func() {
				namespace := "testns"
				pvcName := "pvcFile"
				pvc := kubev1.PersistentVolumeClaim{
					TypeMeta:   metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: pvcName},
				}
				err := pvcCache.Add(&pvc)
				Expect(err).ToNot(HaveOccurred(), "Added PVC to cache successfully")

				volumeName := "pvc-volume"
				volumes := []v1.Volume{
					{
						Name: volumeName,
						VolumeSource: v1.VolumeSource{
							PersistentVolumeClaim: &kubev1.PersistentVolumeClaimVolumeSource{ClaimName: pvcName},
						},
					},
				}
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testvmi", Namespace: namespace, UID: "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{Volumes: volumes, Domain: v1.DomainSpec{}},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred(), "Render manifest successfully")

				Expect(pod.Spec.Containers[0].VolumeDevices).To(BeEmpty(), "No devices in manifest for 1st container")

				Expect(pod.Spec.Containers[0].VolumeMounts).ToNot(BeEmpty(), "Some mounts in manifest for 1st container")
				Expect(len(pod.Spec.Containers[0].VolumeMounts)).To(Equal(4), "4 mounts in manifest for 1st container")
				Expect(pod.Spec.Containers[0].VolumeMounts[3].Name).To(Equal(volumeName), "1st mount in manifest for 1st container has correct name")

				Expect(pod.Spec.Volumes).ToNot(BeEmpty(), "Found some volumes in manifest")
				Expect(len(pod.Spec.Volumes)).To(Equal(4), "Found 4 volumes in manifest")
				Expect(pod.Spec.Volumes[0].PersistentVolumeClaim).ToNot(BeNil(), "Found PVC volume")
				Expect(pod.Spec.Volumes[0].PersistentVolumeClaim.ClaimName).To(Equal(pvcName), "Found PVC volume with correct name")
			})
		})

		Context("with blockdevice mode pvc source", func() {
			It("should add device to template", func() {
				namespace := "testns"
				pvcName := "pvcDevice"
				mode := kubev1.PersistentVolumeBlock
				pvc := kubev1.PersistentVolumeClaim{
					TypeMeta:   metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: pvcName},
					Spec: kubev1.PersistentVolumeClaimSpec{
						VolumeMode: &mode,
					},
				}
				err := pvcCache.Add(&pvc)
				Expect(err).ToNot(HaveOccurred(), "Added PVC to cache successfully")
				volumeName := "pvc-volume"
				volumes := []v1.Volume{
					{
						Name: volumeName,
						VolumeSource: v1.VolumeSource{
							PersistentVolumeClaim: &kubev1.PersistentVolumeClaimVolumeSource{ClaimName: pvcName},
						},
					},
				}
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testvmi", Namespace: namespace, UID: "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{Volumes: volumes, Domain: v1.DomainSpec{}},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred(), "Render manifest successfully")

				Expect(pod.Spec.Containers[0].VolumeDevices).ToNot(BeEmpty(), "Found some devices for 1st container")
				Expect(len(pod.Spec.Containers[0].VolumeDevices)).To(Equal(1), "Found 1 device for 1st container")
				Expect(pod.Spec.Containers[0].VolumeDevices[0].Name).To(Equal(volumeName), "Found device for 1st container with correct name")

				Expect(pod.Spec.Containers[0].VolumeMounts).ToNot(BeEmpty(), "Found some mounts in manifest for 1st container")
				Expect(len(pod.Spec.Containers[0].VolumeMounts)).To(Equal(3), "Found 3 mounts in manifest for 1st container")

				Expect(pod.Spec.Volumes).ToNot(BeEmpty(), "Found some volumes in manifest")
				Expect(len(pod.Spec.Volumes)).To(Equal(4), "Found 4 volumes in manifest")
				Expect(pod.Spec.Volumes[0].PersistentVolumeClaim).ToNot(BeNil(), "Found PVC volume")
				Expect(pod.Spec.Volumes[0].PersistentVolumeClaim.ClaimName).To(Equal(pvcName), "Found PVC volume with correct name")
			})
		})

		Context("with non existing pvc source", func() {
			It("should result in an error", func() {
				namespace := "testns"
				pvcName := "pvcNotExisting"
				volumeName := "pvc-volume"
				volumes := []v1.Volume{
					{
						Name: volumeName,
						VolumeSource: v1.VolumeSource{
							PersistentVolumeClaim: &kubev1.PersistentVolumeClaimVolumeSource{ClaimName: pvcName},
						},
					},
				}
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testvmi", Namespace: namespace, UID: "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{Volumes: volumes, Domain: v1.DomainSpec{}},
				}

				_, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).To(HaveOccurred(), "Render manifest results in an error")
				Expect(err).To(BeAssignableToTypeOf(PvcNotFoundError(errors.New(""))), "Render manifest results in an PvsNotFoundError")
			})
		})

		Context("with launcher's pull secret", func() {
			It("should contain launcher's secret in pod spec", func() {
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testvmi", Namespace: "default", UID: "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{Domain: v1.DomainSpec{}},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				Expect(len(pod.Spec.ImagePullSecrets)).To(Equal(1))
				Expect(pod.Spec.ImagePullSecrets[0].Name).To(Equal("pull-secret-1"))
			})

		})

		Context("with RegistryDisk pull secrets", func() {
			volumes := []v1.Volume{
				{
					Name: "registrydisk1",
					VolumeSource: v1.VolumeSource{
						RegistryDisk: &v1.RegistryDiskSource{
							Image:           "my-image-1",
							ImagePullSecret: "pull-secret-2",
						},
					},
				},
				{
					Name: "registrydisk2",
					VolumeSource: v1.VolumeSource{
						RegistryDisk: &v1.RegistryDiskSource{
							Image: "my-image-2",
						},
					},
				},
			}

			vmi := v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testvmi", Namespace: "default", UID: "1234",
				},
				Spec: v1.VirtualMachineInstanceSpec{Volumes: volumes, Domain: v1.DomainSpec{}},
			}

			It("should add secret to pod spec", func() {
				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				Expect(len(pod.Spec.ImagePullSecrets)).To(Equal(2))

				// RegistryDisk secrets come first
				Expect(pod.Spec.ImagePullSecrets[0].Name).To(Equal("pull-secret-2"))
				Expect(pod.Spec.ImagePullSecrets[1].Name).To(Equal("pull-secret-1"))
			})

			It("should deduplicate identical secrets", func() {
				volumes[1].VolumeSource.RegistryDisk.ImagePullSecret = "pull-secret-2"

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				Expect(len(pod.Spec.ImagePullSecrets)).To(Equal(2))

				// RegistryDisk secrets come first
				Expect(pod.Spec.ImagePullSecrets[0].Name).To(Equal("pull-secret-2"))
				Expect(pod.Spec.ImagePullSecrets[1].Name).To(Equal("pull-secret-1"))
			})
		})

		Context("with slirp interface", func() {
			It("Should have empty port list in the pod manifest", func() {
				slirpInterface := v1.InterfaceSlirp{}
				domain := v1.DomainSpec{}
				domain.Devices.Interfaces = []v1.Interface{{Name: "testnet", InterfaceBindingMethod: v1.InterfaceBindingMethod{Slirp: &slirpInterface}}}
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testvmi", Namespace: "default", UID: "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{Domain: domain},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				Expect(len(pod.Spec.Containers)).To(Equal(1))
				Expect(len(pod.Spec.Containers[0].Ports)).To(Equal(0))
			})
			It("Should create a port list in the pod manifest", func() {
				slirpInterface := v1.InterfaceSlirp{}
				ports := []v1.Port{{Name: "http", Port: 80}, {Protocol: "UDP", Port: 80}, {Port: 90}, {Name: "other-http", Port: 80}}
				domain := v1.DomainSpec{}
				domain.Devices.Interfaces = []v1.Interface{{Name: "testnet", Ports: ports, InterfaceBindingMethod: v1.InterfaceBindingMethod{Slirp: &slirpInterface}}}
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testvmi", Namespace: "default", UID: "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{Domain: domain},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				Expect(len(pod.Spec.Containers)).To(Equal(1))
				Expect(len(pod.Spec.Containers[0].Ports)).To(Equal(4))
				Expect(pod.Spec.Containers[0].Ports[0].Name).To(Equal("http"))
				Expect(pod.Spec.Containers[0].Ports[0].ContainerPort).To(Equal(int32(80)))
				Expect(pod.Spec.Containers[0].Ports[0].Protocol).To(Equal(kubev1.Protocol("TCP")))
				Expect(pod.Spec.Containers[0].Ports[1].Name).To(Equal(""))
				Expect(pod.Spec.Containers[0].Ports[1].ContainerPort).To(Equal(int32(80)))
				Expect(pod.Spec.Containers[0].Ports[1].Protocol).To(Equal(kubev1.Protocol("UDP")))
				Expect(pod.Spec.Containers[0].Ports[2].Name).To(Equal(""))
				Expect(pod.Spec.Containers[0].Ports[2].ContainerPort).To(Equal(int32(90)))
				Expect(pod.Spec.Containers[0].Ports[2].Protocol).To(Equal(kubev1.Protocol("TCP")))
				Expect(pod.Spec.Containers[0].Ports[3].Name).To(Equal("other-http"))
				Expect(pod.Spec.Containers[0].Ports[3].ContainerPort).To(Equal(int32(80)))
				Expect(pod.Spec.Containers[0].Ports[3].Protocol).To(Equal(kubev1.Protocol("TCP")))
			})
			It("Should create a port list in the pod manifest with multiple interfaces", func() {
				slirpInterface1 := v1.InterfaceSlirp{}
				slirpInterface2 := v1.InterfaceSlirp{}
				ports1 := []v1.Port{{Name: "http", Port: 80}}
				ports2 := []v1.Port{{Name: "other-http", Port: 80}}
				domain := v1.DomainSpec{}
				domain.Devices.Interfaces = []v1.Interface{
					{Name: "testnet",
						Ports: ports1,
						InterfaceBindingMethod: v1.InterfaceBindingMethod{Slirp: &slirpInterface1}},
					{Name: "testnet",
						Ports: ports2,
						InterfaceBindingMethod: v1.InterfaceBindingMethod{Slirp: &slirpInterface2}}}

				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testvmi", Namespace: "default", UID: "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{Domain: domain},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				Expect(len(pod.Spec.Containers)).To(Equal(1))
				Expect(len(pod.Spec.Containers[0].Ports)).To(Equal(2))
				Expect(pod.Spec.Containers[0].Ports[0].Name).To(Equal("http"))
				Expect(pod.Spec.Containers[0].Ports[0].ContainerPort).To(Equal(int32(80)))
				Expect(pod.Spec.Containers[0].Ports[0].Protocol).To(Equal(kubev1.Protocol("TCP")))
				Expect(pod.Spec.Containers[0].Ports[1].Name).To(Equal("other-http"))
				Expect(pod.Spec.Containers[0].Ports[1].ContainerPort).To(Equal(int32(80)))
				Expect(pod.Spec.Containers[0].Ports[1].Protocol).To(Equal(kubev1.Protocol("TCP")))
			})
		})

		Context("with pod networking", func() {
			It("Should require tun device by default", func() {
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testvmi", Namespace: "default", UID: "1234",
					},
				}
				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				tun, ok := pod.Spec.Containers[0].Resources.Limits[TunDevice]
				Expect(ok).To(BeTrue())
				Expect(int(tun.Value())).To(Equal(1))

				found := false
				caps := pod.Spec.Containers[0].SecurityContext.Capabilities
				for _, cap := range caps.Add {
					if cap == CAP_NET_ADMIN {
						found = true
					}
				}
				Expect(found).To(BeTrue(), "Expected compute container to be granted NET_ADMIN capability")
			})

			It("Should require tun device if explicitly requested", func() {
				domain := v1.DomainSpec{}
				autoAttach := true
				domain.Devices.AutoattachPodInterface = &autoAttach

				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testvmi", Namespace: "default", UID: "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{Domain: domain},
				}
				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				tun, ok := pod.Spec.Containers[0].Resources.Limits[TunDevice]
				Expect(ok).To(BeTrue())
				Expect(int(tun.Value())).To(Equal(1))

				found := false
				caps := pod.Spec.Containers[0].SecurityContext.Capabilities
				for _, cap := range caps.Add {
					if cap == CAP_NET_ADMIN {
						found = true
					}
				}
				Expect(found).To(BeTrue(), "Expected compute container to be granted NET_ADMIN capability")
			})

			It("Should not require tun device if explicitly rejected", func() {
				domain := v1.DomainSpec{}
				autoAttach := false
				domain.Devices.AutoattachPodInterface = &autoAttach

				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testvmi", Namespace: "default", UID: "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{Domain: domain},
				}
				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				_, ok := pod.Spec.Containers[0].Resources.Limits[TunDevice]
				Expect(ok).To(BeFalse())

				found := false
				caps := pod.Spec.Containers[0].SecurityContext.Capabilities
				for _, cap := range caps.Add {
					if cap == CAP_NET_ADMIN {
						found = true
					}
				}
				Expect(found).To(BeFalse(), "Expected compute container to not be granted NET_ADMIN capability")
			})
		})

		Context("with a configMap volume source", func() {
			It("Should add the ConfigMap to template", func() {
				volumes := []v1.Volume{
					{
						Name: "configmap-volume",
						VolumeSource: v1.VolumeSource{
							ConfigMap: &v1.ConfigMapVolumeSource{
								LocalObjectReference: kubev1.LocalObjectReference{
									Name: "test-configmap",
								},
							},
						},
					},
				}
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testvmi", Namespace: "default", UID: "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{Volumes: volumes, Domain: v1.DomainSpec{}},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				Expect(pod.Spec.Volumes).ToNot(BeEmpty())
				Expect(len(pod.Spec.Volumes)).To(Equal(4))
				Expect(pod.Spec.Volumes[0].ConfigMap).ToNot(BeNil())
				Expect(pod.Spec.Volumes[0].ConfigMap.LocalObjectReference.Name).To(Equal("test-configmap"))
			})
		})

		Context("with a secret volume source", func() {
			It("should add the Secret to template", func() {
				volumes := []v1.Volume{
					{
						Name: "secret-volume",
						VolumeSource: v1.VolumeSource{
							Secret: &v1.SecretVolumeSource{
								SecretName: "test-secret",
							},
						},
					},
				}
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testvmi", Namespace: "default", UID: "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{Volumes: volumes, Domain: v1.DomainSpec{}},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				Expect(pod.Spec.Volumes).ToNot(BeEmpty())
				Expect(len(pod.Spec.Volumes)).To(Equal(4))
				Expect(pod.Spec.Volumes[0].Secret).ToNot(BeNil())
				Expect(pod.Spec.Volumes[0].Secret.SecretName).To(Equal("test-secret"))
			})
		})
	})
	Describe("ConfigMap", func() {
		var cmListWatch *cache.ListWatch
		var cmInformer cache.SharedIndexInformer
		var cmStore cache.Store
		var stopChan chan struct{}

		BeforeEach(func() {
			stopChan = make(chan struct{})
		})

		AfterEach(func() {
			close(stopChan)
		})

		It("Should return false if configmap is not present", func() {
			cmListWatch = MakeFakeConfigMapWatcher([]kubev1.ConfigMap{})
			cmInformer = cache.NewSharedIndexInformer(cmListWatch, &v1.VirtualMachineInstance{}, time.Second, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
			cmStore = cmInformer.GetStore()
			go cmInformer.Run(stopChan)
			cache.WaitForCacheSync(stopChan, cmInformer.HasSynced)

			result, err := IsEmulationAllowed(cmStore)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeFalse())
		})

		It("Should return false if configmap doesn't have useEmulation set", func() {
			cfgMap := kubev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespaceKubevirt,
					Name:      "kubevirt-config",
				},
				Data: map[string]string{},
			}
			cmListWatch = MakeFakeConfigMapWatcher([]kubev1.ConfigMap{cfgMap})
			cmInformer = cache.NewSharedIndexInformer(cmListWatch, &v1.VirtualMachineInstance{}, time.Second, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
			cmStore = cmInformer.GetStore()
			go cmInformer.Run(stopChan)
			cache.WaitForCacheSync(stopChan, cmInformer.HasSynced)

			result, err := IsEmulationAllowed(cmStore)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeFalse())
		})

		It("Should return true if useEmulation = true", func() {
			cfgMap := kubev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespaceKubevirt,
					Name:      "kubevirt-config",
				},
				Data: map[string]string{UseEmulationKey: "true"},
			}
			cmListWatch = MakeFakeConfigMapWatcher([]kubev1.ConfigMap{cfgMap})
			cmInformer = cache.NewSharedIndexInformer(cmListWatch, &v1.VirtualMachineInstance{}, time.Second, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
			cmStore = cmInformer.GetStore()
			go cmInformer.Run(stopChan)
			cache.WaitForCacheSync(stopChan, cmInformer.HasSynced)

			result, err := IsEmulationAllowed(cmStore)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeTrue())
		})

		It("Should return IfNotPresent if configmap doesn't have imagePullPolicy set", func() {
			cfgMap := kubev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespaceKubevirt,
					Name:      "kubevirt-config",
				},
				Data: map[string]string{},
			}
			cmListWatch = MakeFakeConfigMapWatcher([]kubev1.ConfigMap{cfgMap})
			cmInformer = cache.NewSharedIndexInformer(cmListWatch, &v1.VirtualMachineInstance{}, time.Second, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
			cmStore = cmInformer.GetStore()
			go cmInformer.Run(stopChan)
			cache.WaitForCacheSync(stopChan, cmInformer.HasSynced)

			result, err := GetImagePullPolicy(cmStore)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(kubev1.PullIfNotPresent))
		})

		It("Should return Always if imagePullPolicy = Always", func() {
			cfgMap := kubev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespaceKubevirt,
					Name:      "kubevirt-config",
				},
				Data: map[string]string{ImagePullPolicyKey: "Always"},
			}
			cmListWatch = MakeFakeConfigMapWatcher([]kubev1.ConfigMap{cfgMap})
			cmInformer = cache.NewSharedIndexInformer(cmListWatch, &v1.VirtualMachineInstance{}, time.Second, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
			cmStore = cmInformer.GetStore()
			go cmInformer.Run(stopChan)
			cache.WaitForCacheSync(stopChan, cmInformer.HasSynced)

			result, err := GetImagePullPolicy(cmStore)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(kubev1.PullAlways))
		})

		It("Should return an error if imagePullPolicy is not valid", func() {
			cfgMap := kubev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespaceKubevirt,
					Name:      "kubevirt-config",
				},
				Data: map[string]string{ImagePullPolicyKey: "IHaveNoStrongFeelingsOneWayOrTheOther"},
			}
			cmListWatch = MakeFakeConfigMapWatcher([]kubev1.ConfigMap{cfgMap})
			cmInformer = cache.NewSharedIndexInformer(cmListWatch, &v1.VirtualMachineInstance{}, time.Second, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
			cmStore = cmInformer.GetStore()
			go cmInformer.Run(stopChan)
			cache.WaitForCacheSync(stopChan, cmInformer.HasSynced)

			_, err := GetImagePullPolicy(cmStore)
			Expect(err).To(HaveOccurred())
		})
	})
	Describe("ServiceAccountName", func() {

		It("Should add service account if present", func() {
			serviceAccountName := "testAccount"
			volumes := []v1.Volume{
				{
					Name: "serviceaccount-volume",
					VolumeSource: v1.VolumeSource{
						ServiceAccount: &v1.ServiceAccountVolumeSource{
							ServiceAccountName: serviceAccountName,
						},
					},
				},
			}
			vmi := v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testvmi", Namespace: "default", UID: "1234",
				},
				Spec: v1.VirtualMachineInstanceSpec{Volumes: volumes, Domain: v1.DomainSpec{}},
			}

			pod, err := svc.RenderLaunchManifest(&vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(pod.Spec.ServiceAccountName).To(Equal(serviceAccountName), "ServiceAccount matches")
			Expect(*pod.Spec.AutomountServiceAccountToken).To(BeTrue(), "Token automount is enabled")
		})

		It("Should not add service account if not present", func() {
			vmi := v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testvmi", Namespace: "default", UID: "1234",
				},
				Spec: v1.VirtualMachineInstanceSpec{Domain: v1.DomainSpec{}},
			}

			pod, err := svc.RenderLaunchManifest(&vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(pod.Spec.ServiceAccountName).To(BeEmpty(), "ServiceAccount is empty")
			Expect(*pod.Spec.AutomountServiceAccountToken).To(BeFalse(), "Token automount is disabled")
		})

	})
})

func MakeFakeConfigMapWatcher(configMaps []kubev1.ConfigMap) *cache.ListWatch {
	cmListWatch := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			return &kubev1.ConfigMapList{Items: configMaps}, nil
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			fakeWatch := watch.NewFake()
			for _, cfgMap := range configMaps {
				fakeWatch.Add(&cfgMap)
			}
			return watch.NewFake(), nil
		},
	}
	return cmListWatch
}

func True() *bool {
	b := true
	return &b
}

func False() *bool {
	b := false
	return &b
}

func TestTemplate(t *testing.T) {
	log.Log.SetIOWriter(GinkgoWriter)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Template")
}
