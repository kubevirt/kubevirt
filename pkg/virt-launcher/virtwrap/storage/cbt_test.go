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

package storage

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"

	v1 "kubevirt.io/api/core/v1"
	api2 "kubevirt.io/client-go/api"

	"kubevirt.io/kubevirt/pkg/storage/cbt"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter"
)

const (
	testVmName    = "testvmi"
	testNamespace = "testnamespace"
)

func newVMI(namespace, name string) *v1.VirtualMachineInstance {
	vmi := api2.NewMinimalVMIWithNS(namespace, name)
	v1.SetObjectDefaults_VirtualMachineInstance(vmi)
	return vmi
}

var _ = Describe("Changed Block Tracking", func() {
	Context("ShouldCreateQCOW2Overlay", func() {
		DescribeTable("should return correct value based on ChangedBlockTracking state", func(state v1.ChangedBlockTrackingState, expected bool) {
			vmi := newVMI(testNamespace, testVmName)
			cbt.SetCBTState(&vmi.Status.ChangedBlockTracking, state)

			result := ShouldCreateQCOW2Overlay(vmi)
			Expect(result).To(Equal(expected))
		},
			Entry("when state is Initializing", v1.ChangedBlockTrackingInitializing, true),
			Entry("when state is Enabled", v1.ChangedBlockTrackingEnabled, false),
			Entry("when state is Disabled", v1.ChangedBlockTrackingDisabled, false),
			Entry("when state is Undefined", v1.ChangedBlockTrackingUndefined, false),
		)
	})

	Context("ApplyChangedBlockTracking", func() {
		var (
			vmi                      *v1.VirtualMachineInstance
			converterContext         *converter.ConverterContext
			createQCOW2OverlayCalled int
			blockDevCalled           int
		)

		BeforeEach(func() {
			vmi = newVMI(testNamespace, testVmName)
			cbt.SetCBTState(&vmi.Status.ChangedBlockTracking, v1.ChangedBlockTrackingInitializing)
			converterContext = &converter.ConverterContext{
				IsBlockPVC: make(map[string]bool),
				IsBlockDV:  make(map[string]bool),
			}
			createQCOW2OverlayCalled = 0
			blockDevCalled = 0
			CreateQCOW2Overlay = func(overlayPath, imagePath string, blockDev bool) error {
				createQCOW2OverlayCalled++
				if blockDev {
					blockDevCalled++
				}
				return nil
			}
		})

		It("should skip volumes that don't support CBT", func() {
			vmi.Spec.Volumes = []v1.Volume{
				{
					Name: "config-map-volume",
					VolumeSource: v1.VolumeSource{
						ConfigMap: &v1.ConfigMapVolumeSource{},
					},
				},
				{
					Name: "secret-volume",
					VolumeSource: v1.VolumeSource{
						Secret: &v1.SecretVolumeSource{},
					},
				},
			}

			err := ApplyChangedBlockTracking(vmi, converterContext)
			Expect(err).ToNot(HaveOccurred())
			Expect(converterContext.ApplyCBT).To(BeEmpty())
			Expect(createQCOW2OverlayCalled).To(Equal(0))
		})

		It("should process fs volumes that support CBT", func() {
			vmi.Spec.Volumes = []v1.Volume{
				{
					Name: "pvc-volume",
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
							PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
								ClaimName: "test-pvc",
							},
						},
					},
				},
				{
					Name: "dv-volume",
					VolumeSource: v1.VolumeSource{
						DataVolume: &v1.DataVolumeSource{
							Name: "test-dv",
						},
					},
				},
				{
					Name: "host-disk-volume",
					VolumeSource: v1.VolumeSource{
						HostDisk: &v1.HostDisk{
							Path: "/path/to/disk",
						},
					},
				},
			}
			converterContext.IsBlockPVC["pvc-volume"] = false
			converterContext.IsBlockDV["dv-volume"] = false

			err := ApplyChangedBlockTracking(vmi, converterContext)
			Expect(err).ToNot(HaveOccurred())
			Expect(createQCOW2OverlayCalled).To(Equal(3))
			Expect(blockDevCalled).To(Equal(0))
			Expect(converterContext.ApplyCBT).To(HaveKey("pvc-volume"))
			Expect(converterContext.ApplyCBT["pvc-volume"]).To(ContainSubstring("pvc-volume.qcow2"))
			Expect(converterContext.ApplyCBT).To(HaveKey("dv-volume"))
			Expect(converterContext.ApplyCBT["dv-volume"]).To(ContainSubstring("dv-volume.qcow2"))
			Expect(converterContext.ApplyCBT).To(HaveKey("host-disk-volume"))
			Expect(converterContext.ApplyCBT["host-disk-volume"]).To(ContainSubstring("host-disk-volume.qcow2"))
		})

		It("should process block volumes that support CBT", func() {
			vmi.Spec.Volumes = []v1.Volume{
				{
					Name: "pvc-volume",
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
							PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
								ClaimName: "test-pvc",
							},
						},
					},
				},
				{
					Name: "dv-volume",
					VolumeSource: v1.VolumeSource{
						DataVolume: &v1.DataVolumeSource{
							Name: "test-dv",
						},
					},
				},
			}
			converterContext.IsBlockPVC["pvc-volume"] = true
			converterContext.IsBlockDV["dv-volume"] = true

			err := ApplyChangedBlockTracking(vmi, converterContext)
			Expect(err).ToNot(HaveOccurred())
			Expect(createQCOW2OverlayCalled).To(Equal(2))
			Expect(blockDevCalled).To(Equal(2))
			Expect(converterContext.ApplyCBT).To(HaveKey("pvc-volume"))
			Expect(converterContext.ApplyCBT["pvc-volume"]).To(ContainSubstring("pvc-volume.qcow2"))
			Expect(converterContext.ApplyCBT).To(HaveKey("dv-volume"))
			Expect(converterContext.ApplyCBT["dv-volume"]).To(ContainSubstring("dv-volume.qcow2"))
		})

		It("should apply cbt to domain but skip creation when CBT is already enabled", func() {
			vmi.Spec.Volumes = []v1.Volume{
				{
					Name: "pvc-volume",
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
							PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
								ClaimName: "test-pvc",
							},
						},
					},
				},
			}
			cbt.SetCBTState(&vmi.Status.ChangedBlockTracking, v1.ChangedBlockTrackingEnabled)

			err := ApplyChangedBlockTracking(vmi, converterContext)
			Expect(err).ToNot(HaveOccurred())
			Expect(createQCOW2OverlayCalled).To(Equal(0))
			Expect(converterContext.ApplyCBT).To(HaveKey("pvc-volume"))
			Expect(converterContext.ApplyCBT["pvc-volume"]).To(ContainSubstring("pvc-volume.qcow2"))
		})

		It("should return error when overlay creation fails", func() {
			vmi.Spec.Volumes = []v1.Volume{
				{
					Name: "pvc-volume",
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
							PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
								ClaimName: "test-pvc",
							},
						},
					},
				},
			}
			cbt.SetCBTState(&vmi.Status.ChangedBlockTracking, v1.ChangedBlockTrackingInitializing)

			errMsg := "failed to create overlay"
			// Mock createQCOW2Overlay to return error
			CreateQCOW2Overlay = func(overlayPath, imagePath string, blockDev bool) error {
				createQCOW2OverlayCalled++
				return fmt.Errorf("%s", errMsg)
			}

			err := ApplyChangedBlockTracking(vmi, converterContext)
			Expect(err).To(HaveOccurred())
			Expect(createQCOW2OverlayCalled).To(Equal(1))
			Expect(err.Error()).To(ContainSubstring(errMsg))
			Expect(converterContext.ApplyCBT).To(BeEmpty())
		})
	})
})
