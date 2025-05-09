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
package services

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/testutils"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

var _ = Describe("Container disk", func() {
	Context("image pull policy", func() {

		var svc TemplateService

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			config, _, _ := testutils.NewFakeClusterConfigUsingKVWithCPUArch(&v1.KubeVirt{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kubevirt",
					Namespace: "kubevirt",
				},
				Spec: v1.KubeVirtSpec{
					Configuration: v1.KubeVirtConfiguration{
						DeveloperConfiguration: &v1.DeveloperConfiguration{},
					},
				},
			}, "amd64")
			svc = NewTemplateService("kubevirt/virt-launcher",
				240,
				"/var/run/kubevirt",
				"/var/run/kubevirt-ephemeral-disks",
				"/var/run/kubevirt/container-disks",
				v1.HotplugDiskDir,
				"pull-secret-1",
				cache.NewIndexer(cache.DeletionHandlingMetaNamespaceKeyFunc, nil),
				kubecli.NewMockKubevirtClient(ctrl),
				config,
				107,
				"kubevirt/vmexport",
				cache.NewIndexer(cache.DeletionHandlingMetaNamespaceKeyFunc, nil),
				cache.NewIndexer(cache.DeletionHandlingMetaNamespaceKeyFunc, nil),
			)
		})

		DescribeTable("should", func(image string, policy k8sv1.PullPolicy) {
			vmi := &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testvmi", Namespace: "namespace1", UID: "1234",
				},
			}
			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "test",
				DiskDevice: v1.DiskDevice{
					Disk: &v1.DiskTarget{
						Bus: v1.DiskBusVirtio,
					},
				},
			})
			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				Name: "test",
				VolumeSource: v1.VolumeSource{
					ContainerDisk: &v1.ContainerDiskSource{
						Image:           image,
						ImagePullPolicy: policy,
					},
				},
			})

			pod, err := svc.RenderLaunchManifest(vmi)
			Expect(err).ToNot(HaveOccurred())
			container := pod.Spec.Containers[1]
			Expect(container.ImagePullPolicy).To(Equal(policy))
		},
			Entry("pass through Never to the pod", "test@sha256:9c2b78e11c25b3fd0b24b0ed684a112052dff03eee4ca4bdcc4f3168f9a14396", k8sv1.PullNever),
			Entry("pass through IfNotPresent to the pod", "test:latest", k8sv1.PullIfNotPresent),
		)
	})
})
