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
	. "kubevirt.io/kubevirt/pkg/virt-controller/services"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	kubev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/log"
)

var _ = Describe("Template", func() {

	log.Log.SetIOWriter(GinkgoWriter)
	svc, err := NewTemplateService("kubevirt/virt-launcher", "/var/run/kubevirt")

	Describe("Rendering", func() {
		Context("launch template with correct parameters", func() {
			It("should work", func() {

				Expect(err).To(BeNil())
				pod, err := svc.RenderLaunchManifest(&v1.VirtualMachine{ObjectMeta: metav1.ObjectMeta{Name: "testvm", Namespace: "testns", UID: "1234"}, Spec: v1.VirtualMachineSpec{Domain: v1.DomainSpec{}}})

				Expect(err).To(BeNil())
				Expect(pod.Spec.Containers[0].Image).To(Equal("kubevirt/virt-launcher"))
				Expect(pod.ObjectMeta.Labels).To(Equal(map[string]string{
					v1.AppLabel:    "virt-launcher",
					v1.DomainLabel: "testvm",
					v1.VMUIDLabel:  "1234",
				}))
				Expect(pod.ObjectMeta.GenerateName).To(Equal("virt-launcher-testvm-----"))
				Expect(pod.Spec.NodeSelector).To(BeEmpty())
				Expect(pod.Spec.Containers[0].Command).To(Equal([]string{"/entrypoint.sh",
					"--qemu-timeout", "5m",
					"--name", "testvm",
					"--namespace", "testns",
					"--kubevirt-share-dir", "/var/run/kubevirt",
					"--readiness-file", "/tmp/healthy",
					"--grace-period-seconds", "45"}))
				Expect(*pod.Spec.TerminationGracePeriodSeconds).To(Equal(int64(60)))
			})
		})
		Context("with node selectors", func() {
			It("should add node selectors to template", func() {

				nodeSelector := map[string]string{
					"kubernetes.io/hostname": "master",
				}
				vm := v1.VirtualMachine{ObjectMeta: metav1.ObjectMeta{Name: "testvm", Namespace: "default", UID: "1234"}, Spec: v1.VirtualMachineSpec{NodeSelector: nodeSelector, Domain: v1.DomainSpec{}}}

				pod, err := svc.RenderLaunchManifest(&vm)

				Expect(err).To(BeNil())
				Expect(pod.Spec.Containers[0].Image).To(Equal("kubevirt/virt-launcher"))
				Expect(pod.ObjectMeta.Labels).To(Equal(map[string]string{
					v1.AppLabel:    "virt-launcher",
					v1.DomainLabel: "testvm",
					v1.VMUIDLabel:  "1234",
				}))
				Expect(pod.ObjectMeta.GenerateName).To(Equal("virt-launcher-testvm-----"))
				Expect(pod.Spec.NodeSelector).To(Equal(map[string]string{
					"kubernetes.io/hostname": "master",
				}))
				Expect(pod.Spec.Containers[0].Command).To(Equal([]string{"/entrypoint.sh",
					"--qemu-timeout", "5m",
					"--name", "testvm",
					"--namespace", "default",
					"--kubevirt-share-dir", "/var/run/kubevirt",
					"--readiness-file", "/tmp/healthy",
					"--grace-period-seconds", "45"}))
				Expect(pod.Spec.Volumes[0].HostPath.Path).To(Equal("/var/run/kubevirt"))
				Expect(pod.Spec.Containers[0].VolumeMounts[0].MountPath).To(Equal("/var/run/kubevirt"))
				Expect(*pod.Spec.TerminationGracePeriodSeconds).To(Equal(int64(60)))
			})

			It("should add node affinity to pod", func() {
				nodeAffinity := kubev1.NodeAffinity{}
				vm := v1.VirtualMachine{
					ObjectMeta: metav1.ObjectMeta{Name: "testvm", Namespace: "default", UID: "1234"},
					Spec: v1.VirtualMachineSpec{
						Affinity: &v1.Affinity{NodeAffinity: &nodeAffinity},
						Domain:   v1.DomainSpec{},
					},
				}
				pod, err := svc.RenderLaunchManifest(&vm)

				Expect(err).To(BeNil())
				Expect(pod.Spec.Affinity).To(BeEquivalentTo(&kubev1.Affinity{NodeAffinity: &nodeAffinity}))
			})

			It("should not add empty node affinity to pod", func() {
				vm := v1.VirtualMachine{
					ObjectMeta: metav1.ObjectMeta{Name: "testvm", Namespace: "default", UID: "1234"},
					Spec: v1.VirtualMachineSpec{
						Domain: v1.DomainSpec{},
					},
				}
				pod, err := svc.RenderLaunchManifest(&vm)

				Expect(err).To(BeNil())
				Expect(pod.Spec.Affinity).To(BeNil())
			})
		})

		Context("with pvs source", func() {
			It("should add pvc to template", func() {
				volumes := []v1.Volume{
					{
						Name: "pvc-volume",
						VolumeSource: v1.VolumeSource{
							PersistentVolumeClaim: &kubev1.PersistentVolumeClaimVolumeSource{ClaimName: "nfs-pvc"},
						},
					},
				}
				vm := v1.VirtualMachine{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testvm", Namespace: "default", UID: "1234",
					},
					Spec: v1.VirtualMachineSpec{Volumes: volumes, Domain: v1.DomainSpec{}},
				}

				pod, err := svc.RenderLaunchManifest(&vm)
				Expect(err).To(BeNil())

				Expect(pod.Spec.Volumes).ToNot(BeEmpty())
				Expect(len(pod.Spec.Volumes)).To(Equal(5))
				Expect(pod.Spec.Volumes[0].PersistentVolumeClaim).ToNot(BeNil())
				Expect(pod.Spec.Volumes[0].PersistentVolumeClaim.ClaimName).To(Equal("nfs-pvc"))
			})
		})

		Context("with ISCSI source", func() {
			It("should add ISCSI source to template", func() {
				volumes := []v1.Volume{
					v1.Volume{
						Name: "iscsi-volume",
						VolumeSource: v1.VolumeSource{
							ISCSI: &kubev1.ISCSIVolumeSource{
								TargetPortal: "127.0.0.1:6543",
								IQN:          "iqn.2009-02.com.test:for.all",
								Lun:          1,
							},
						},
					},
				}
				vm := v1.VirtualMachine{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testvm", Namespace: "default", UID: "1234",
					},
					Spec: v1.VirtualMachineSpec{Volumes: volumes, Domain: v1.DomainSpec{}},
				}

				pod, err := svc.RenderLaunchManifest(&vm)
				Expect(err).To(BeNil())

				Expect(pod.Spec.Volumes).ToNot(BeEmpty())
				Expect(len(pod.Spec.Volumes)).To(Equal(5))
				Expect(pod.Spec.Volumes[0].ISCSI).ToNot(BeNil())
				Expect(pod.Spec.Volumes[0].ISCSI.TargetPortal).To(Equal("127.0.0.1:6543"))
				Expect(pod.Spec.Volumes[0].ISCSI.Lun).To(Equal(int32(1)))
			})
		})
	})
})
