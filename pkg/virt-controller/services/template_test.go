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
	"k8s.io/apimachinery/pkg/util/uuid"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/log"
)

var _ = Describe("Template", func() {

	log.Log.SetIOWriter(GinkgoWriter)
	svc, err := NewTemplateService("kubevirt/virt-launcher", "kubevirt/virt-handler", "/var/run/kubevirt")

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
		Context("migration", func() {
			var (
				srcIp      = kubev1.NodeAddress{}
				destIp     = kubev1.NodeAddress{}
				srcNodeIp  = kubev1.Node{}
				destNodeIp = kubev1.Node{}
				srcNode    kubev1.Node
				targetNode kubev1.Node
			)

			BeforeEach(func() {
				srcIp = kubev1.NodeAddress{
					Type:    kubev1.NodeInternalIP,
					Address: "127.0.0.2",
				}
				destIp = kubev1.NodeAddress{
					Type:    kubev1.NodeInternalIP,
					Address: "127.0.0.3",
				}
				srcNodeIp = kubev1.Node{
					Status: kubev1.NodeStatus{
						Addresses: []kubev1.NodeAddress{srcIp},
					},
				}
				destNodeIp = kubev1.Node{
					Status: kubev1.NodeStatus{
						Addresses: []kubev1.NodeAddress{destIp},
					},
				}
				srcNode = kubev1.Node{
					Status: kubev1.NodeStatus{
						Addresses: []kubev1.NodeAddress{srcIp, destIp},
					},
				}
				targetNode = kubev1.Node{
					Status: kubev1.NodeStatus{
						Addresses: []kubev1.NodeAddress{destIp, srcIp},
					},
				}
			})

			Context("migration template", func() {
				var vm *v1.VirtualMachine
				var destPod *kubev1.Pod
				var hostInfo *v1.MigrationHostInfo
				BeforeEach(func() {
					vm = v1.NewMinimalVM("testvm")
					vm.GetObjectMeta().SetUID(uuid.NewUUID())
					destPod, err = svc.RenderLaunchManifest(vm)
					Expect(err).ToNot(HaveOccurred())
					destPod.Status.PodIP = "127.0.0.1"
					hostInfo = &v1.MigrationHostInfo{PidNS: "pidns", Controller: []string{"cpu", "memory"}, Slice: "slice"}
				})
				Context("with correct parameters", func() {

					It("should never restart", func() {
						job, err := svc.RenderMigrationJob(vm, &srcNodeIp, &destNodeIp, destPod, hostInfo)
						Expect(err).ToNot(HaveOccurred())
						Expect(job.Spec.RestartPolicy).To(Equal(kubev1.RestartPolicyNever))
					})
					It("should use the first ip it finds", func() {
						job, err := svc.RenderMigrationJob(vm, &srcNode, &targetNode, destPod, hostInfo)
						Expect(err).ToNot(HaveOccurred())
						refCommand := []string{
							"/migrate", "testvm", "--source", "qemu+tcp://127.0.0.2/system",
							"--dest", "qemu+tcp://127.0.0.3/system",
							"--node-ip", "127.0.0.3", "--namespace", "default",
							"--slice", "slice", "--controller", "cpu,memory",
						}
						Expect(job.Spec.Containers[0].Command).To(Equal(refCommand))
					})
				})
				Context("with incorrect parameters", func() {
					It("should error on missing source address", func() {
						srcNode.Status.Addresses = []kubev1.NodeAddress{}
						job, err := svc.RenderMigrationJob(vm, &srcNode, &targetNode, destPod, hostInfo)
						Expect(err).To(HaveOccurred())
						Expect(job).To(BeNil())
					})
					It("should error on missing destination address", func() {
						targetNode.Status.Addresses = []kubev1.NodeAddress{}
						job, err := svc.RenderMigrationJob(vm, &srcNode, &targetNode, destPod, hostInfo)
						Expect(err).To(HaveOccurred())
						Expect(job).To(BeNil())
					})
				})
			})
		})
	})

})
