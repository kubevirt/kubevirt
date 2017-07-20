/*
 * This file is part of the kubevirt project
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
	"kubevirt.io/kubevirt/pkg/logging"
)

var _ = Describe("Template", func() {

	logging.DefaultLogger().SetIOWriter(GinkgoWriter)
	svc, err := NewTemplateService("kubevirt/virt-launcher", "kubevirt/virt-handler")

	Describe("Rendering", func() {
		Context("launch template with correct parameters", func() {
			It("should work", func() {

				Expect(err).To(BeNil())
				pod, err := svc.RenderLaunchManifest(&v1.VM{ObjectMeta: metav1.ObjectMeta{Name: "testvm", UID: "1234"}, Spec: v1.VMSpec{Domain: &v1.DomainSpec{}}})

				Expect(err).To(BeNil())
				Expect(pod.Spec.Containers[0].Image).To(Equal("kubevirt/virt-launcher"))
				Expect(pod.ObjectMeta.Labels).To(Equal(map[string]string{
					v1.AppLabel:    "virt-launcher",
					v1.DomainLabel: "testvm",
					v1.VMUIDLabel:  "1234",
				}))
				Expect(pod.ObjectMeta.GenerateName).To(Equal("virt-launcher-testvm-----"))
				Expect(pod.Spec.NodeSelector).To(BeEmpty())
				Expect(pod.Spec.Containers[0].Command).To(Equal([]string{"/virt-launcher", "-qemu-timeout", "60s"}))
			})
		})
		Context("with node selectors", func() {
			It("should add node selectors to template", func() {

				nodeSelector := map[string]string{
					"kubernetes.io/hostname": "master",
				}
				vm := v1.VM{ObjectMeta: metav1.ObjectMeta{Name: "testvm", UID: "1234"}, Spec: v1.VMSpec{NodeSelector: nodeSelector, Domain: &v1.DomainSpec{}}}

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
				Expect(pod.Spec.Containers[0].Command).To(Equal([]string{"/virt-launcher", "-qemu-timeout", "60s"}))
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
				var vm *v1.VM
				var destPod *kubev1.Pod
				BeforeEach(func() {
					vm = v1.NewMinimalVM("testvm")
					vm.GetObjectMeta().SetUID(uuid.NewUUID())
					destPod, err = svc.RenderLaunchManifest(vm)
					Expect(err).ToNot(HaveOccurred())
					destPod.Status.PodIP = "127.0.0.1"
				})
				Context("with correct parameters", func() {

					It("should never restart", func() {
						job, err := svc.RenderMigrationJob(vm, &srcNodeIp, &destNodeIp, destPod)
						Expect(err).ToNot(HaveOccurred())
						Expect(job.Spec.RestartPolicy).To(Equal(kubev1.RestartPolicyNever))
					})
					It("should use the first ip it finds", func() {
						job, err := svc.RenderMigrationJob(vm, &srcNode, &targetNode, destPod)
						Expect(err).ToNot(HaveOccurred())
						refCommand := []string{
							"/migrate", "testvm", "--source", "qemu+tcp://127.0.0.2/system",
							"--dest", "qemu+tcp://127.0.0.3/system",
							"--node-ip", "127.0.0.3", "--namespace", "default"}
						Expect(job.Spec.Containers[0].Command).To(Equal(refCommand))
					})
				})
				Context("with incorrect parameters", func() {
					It("should error on missing source address", func() {
						srcNode.Status.Addresses = []kubev1.NodeAddress{}
						job, err := svc.RenderMigrationJob(vm, &srcNode, &targetNode, destPod)
						Expect(err).To(HaveOccurred())
						Expect(job).To(BeNil())
					})
					It("should error on missing destination address", func() {
						targetNode.Status.Addresses = []kubev1.NodeAddress{}
						job, err := svc.RenderMigrationJob(vm, &srcNode, &targetNode, destPod)
						Expect(err).To(HaveOccurred())
						Expect(job).To(BeNil())
					})
				})
			})
		})
	})

})
