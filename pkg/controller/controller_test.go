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

package controller_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomegaTypes "github.com/onsi/gomega/types"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/pointer"
)

var _ = Describe("Controller", func() {

	Context("using pod utility functions", func() {
		Context("IsPodReady", func() {
			DescribeTable("should return", func(phase k8sv1.PodPhase, matcher gomegaTypes.GomegaMatcher) {
				pod := &k8sv1.Pod{
					Status: k8sv1.PodStatus{
						Phase: phase,
					},
				}
				Expect(controller.IsPodReady(pod)).To(matcher)
			},
				Entry("false if pod is in succeeded phase", k8sv1.PodSucceeded, BeFalse()),
				Entry("false if pod is in failed phase", k8sv1.PodFailed, BeFalse()),
				Entry("true if pod is in running phase", k8sv1.PodRunning, BeTrue()),
			)

			It("should return false if the compute container is terminated", func() {
				pod := &k8sv1.Pod{
					Status: k8sv1.PodStatus{
						Phase: k8sv1.PodRunning,
						ContainerStatuses: []k8sv1.ContainerStatus{
							{
								Name: "compute",
								State: k8sv1.ContainerState{
									Running:    &k8sv1.ContainerStateRunning{},
									Terminated: &k8sv1.ContainerStateTerminated{},
								},
							},
						},
					},
				}
				Expect(controller.IsPodReady(pod)).To(BeFalse())
			})

			It("should return false if the compute container is not running", func() {
				pod := &k8sv1.Pod{
					Status: k8sv1.PodStatus{
						Phase: k8sv1.PodRunning,
						ContainerStatuses: []k8sv1.ContainerStatus{
							{
								Name:  "compute",
								State: k8sv1.ContainerState{},
							},
						},
					},
				}
				Expect(controller.IsPodReady(pod)).To(BeFalse())
			})

			It("should return false if the istio-proxy container is not running", func() {
				pod := &k8sv1.Pod{
					Status: k8sv1.PodStatus{
						Phase: k8sv1.PodRunning,
						ContainerStatuses: []k8sv1.ContainerStatus{
							{
								Name: "compute",
								State: k8sv1.ContainerState{
									Running: &k8sv1.ContainerStateRunning{},
								},
							},
							{
								Name:  "istio-proxy",
								State: k8sv1.ContainerState{},
							},
						},
					},
				}
				Expect(controller.IsPodReady(pod)).To(BeFalse())
			})

			It("should return false if the a container reports that it is not ready", func() {
				pod := &k8sv1.Pod{
					Status: k8sv1.PodStatus{
						Phase: k8sv1.PodRunning,
						ContainerStatuses: []k8sv1.ContainerStatus{
							{
								Name: "compute",
								State: k8sv1.ContainerState{
									Running: &k8sv1.ContainerStateRunning{},
								},
								Ready: true,
							},
							{
								Name:  "fake-container",
								Ready: false,
							},
						},
					},
				}
				Expect(controller.IsPodReady(pod)).To(BeFalse())
			})

			It("should return false if the pod is being deleted", func() {
				pod := &k8sv1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						DeletionTimestamp: pointer.P(metav1.Now()),
					},
				}
				Expect(controller.IsPodReady(pod)).To(BeFalse())
			})

		})

		Context("IsPodFailedOrGoingDown", func() {
			DescribeTable("should return", func(phase k8sv1.PodPhase, matcher gomegaTypes.GomegaMatcher) {
				pod := &k8sv1.Pod{
					Status: k8sv1.PodStatus{
						Phase: phase,
					},
				}
				Expect(controller.IsPodFailedOrGoingDown(pod)).To(matcher)
			},
				Entry("true if pod is in failed phase", k8sv1.PodFailed, BeTrue()),
				Entry("false if pod is in scheduled phase", k8sv1.PodPending, BeFalse()),
				Entry("false if pod is in running phase", k8sv1.PodRunning, BeFalse()),
			)

			It("should return true if the compute container is terminated with exit code != 0", func() {
				pod := &k8sv1.Pod{
					Status: k8sv1.PodStatus{
						Phase: k8sv1.PodRunning,
						ContainerStatuses: []k8sv1.ContainerStatus{
							{
								Name: "compute",
								State: k8sv1.ContainerState{
									Terminated: &k8sv1.ContainerStateTerminated{
										ExitCode: int32(1),
									},
								},
							},
						},
					},
				}
				Expect(controller.IsPodFailedOrGoingDown(pod)).To(BeTrue())
			})

			It("should return true if the pod is being deleted", func() {
				pod := &k8sv1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						DeletionTimestamp: pointer.P(metav1.Now()),
					},
				}
				Expect(controller.IsPodFailedOrGoingDown(pod)).To(BeTrue())
			})
		})

		Context("PodExists", func() {
			DescribeTable("should return", func(pod *k8sv1.Pod, matcher gomegaTypes.GomegaMatcher) {
				Expect(controller.PodExists(pod)).To(matcher)
			},
				Entry("true if pod exists", &k8sv1.Pod{}, BeTrue()),
				Entry("false if pod is nil", nil, BeFalse()),
			)
		})

		Context("GetHotplugVolumes", func() {
			DescribeTable("should not return the new volume", func(volume v1.Volume) {

				vmi := &v1.VirtualMachineInstance{
					Spec: v1.VirtualMachineInstanceSpec{
						Volumes: []v1.Volume{volume},
					},
				}
				pod := &k8sv1.Pod{
					Spec: k8sv1.PodSpec{
						Volumes: []k8sv1.Volume{{Name: "existing"}},
					},
				}
				Expect(controller.GetHotplugVolumes(vmi, pod)).To(BeEmpty())
			},
				Entry("if it already exist", v1.Volume{Name: "existing"}),
				Entry("with HostDisk", v1.Volume{Name: "new", VolumeSource: v1.VolumeSource{HostDisk: &v1.HostDisk{}}}),
				Entry("with CloudInitNoCloud", v1.Volume{Name: "new", VolumeSource: v1.VolumeSource{CloudInitNoCloud: &v1.CloudInitNoCloudSource{}}}),
				Entry("with CloudInitConfigDrive", v1.Volume{Name: "new", VolumeSource: v1.VolumeSource{CloudInitConfigDrive: &v1.CloudInitConfigDriveSource{}}}),
				Entry("with Sysprep", v1.Volume{Name: "new", VolumeSource: v1.VolumeSource{Sysprep: &v1.SysprepSource{}}}),
				Entry("with ContainerDisk", v1.Volume{Name: "new", VolumeSource: v1.VolumeSource{ContainerDisk: &v1.ContainerDiskSource{}}}),
				Entry("with Ephemeral", v1.Volume{Name: "new", VolumeSource: v1.VolumeSource{Ephemeral: &v1.EphemeralVolumeSource{}}}),
				Entry("with EmptyDisk", v1.Volume{Name: "new", VolumeSource: v1.VolumeSource{EmptyDisk: &v1.EmptyDiskSource{}}}),
				Entry("with ConfigMap", v1.Volume{Name: "new", VolumeSource: v1.VolumeSource{ConfigMap: &v1.ConfigMapVolumeSource{}}}),
				Entry("with Secret", v1.Volume{Name: "new", VolumeSource: v1.VolumeSource{Secret: &v1.SecretVolumeSource{}}}),
				Entry("with DownwardAPI", v1.Volume{Name: "new", VolumeSource: v1.VolumeSource{DownwardAPI: &v1.DownwardAPIVolumeSource{}}}),
				Entry("with ServiceAccount", v1.Volume{Name: "new", VolumeSource: v1.VolumeSource{ServiceAccount: &v1.ServiceAccountVolumeSource{}}}),
				Entry("with DownwardMetrics", v1.Volume{Name: "new", VolumeSource: v1.VolumeSource{DownwardMetrics: &v1.DownwardMetricsVolumeSource{}}}),
			)

			DescribeTable("should return the new volume", func(volume *v1.Volume) {
				vmi := &v1.VirtualMachineInstance{
					Spec: v1.VirtualMachineInstanceSpec{
						Volumes: []v1.Volume{*volume},
					},
				}
				pod := &k8sv1.Pod{
					Spec: k8sv1.PodSpec{
						Volumes: []k8sv1.Volume{{Name: "existing"}},
					},
				}
				Expect(controller.GetHotplugVolumes(vmi, pod)).To(ContainElement(volume))
			},
				Entry("with DataVolume", &v1.Volume{Name: "new", VolumeSource: v1.VolumeSource{DataVolume: &v1.DataVolumeSource{}}}),
				Entry("with PersistentVolumeClaim", &v1.Volume{Name: "new", VolumeSource: v1.VolumeSource{PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{}}}),
				Entry("with MemoryDump", &v1.Volume{Name: "new", VolumeSource: v1.VolumeSource{MemoryDump: &v1.MemoryDumpVolumeSource{}}}),
			)
		})
	})
})
