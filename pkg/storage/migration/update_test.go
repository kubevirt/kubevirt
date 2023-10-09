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
 * Copyright 2024 The KubeVirt Authors.
 *
 */

package migration

import (
	"context"
	"fmt"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/equality"
	virtstoragev1alpha1 "kubevirt.io/api/storage/v1alpha1"
	"kubevirt.io/client-go/kubecli"

	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
)

var _ = Describe("VolumeMigrationUpdater", func() {
	var (
		ctrl       *gomock.Controller
		virtClient *kubecli.MockKubevirtClient
		updater    VolumeMigrationUpdater
	)
	const testNs = "test"

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		updater = NewVolumeMigrationUpdater(virtClient)
	})

	Context("Update volumes on the VMI", func() {
		Context("Update volumes after successful migration on a VMI", func() {
			var (
				vmiInterface *kubecli.MockVirtualMachineInstanceInterface
			)
			BeforeEach(func() {
				vmiInterface = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)
				virtClient.EXPECT().VirtualMachineInstance(gomock.Any()).Return(vmiInterface).AnyTimes()
				vmiInterface.EXPECT().Update(context.Background(), gomock.Any()).AnyTimes()
			})

			DescribeTable("UpdateVMIWithMigrationVolumes",
				func(pvcs []string, migVols []virtstoragev1alpha1.MigratedVolume, expectedErr error) {
					phase := virtstoragev1alpha1.VolumeMigrationPhaseSucceeded
					vmi := createVMIWithPVCs("testvmi", testNs, pvcs...)
					updateMigrateVolumesVMIStatus(vmi, migVols, &phase)
					vmi, err := updater.UpdateVMIWithMigrationVolumes(vmi)
					if expectedErr != nil {
						Expect(err).Should(MatchError(expectedErr))
						return
					}
					Expect(err).ShouldNot(HaveOccurred())
					mapVol := make(map[string]bool)
					for _, v := range migVols {
						mapVol[v.DestinationClaim] = true
					}
					for _, v := range vmi.Spec.Volumes {
						name := storagetypes.PVCNameFromVirtVolume(&v)
						if name == "" {
							continue
						}
						if _, ok := mapVol[name]; ok {
							delete(mapVol, name)
						}
					}
					Expect(mapVol).Should(BeEmpty())
				},
				Entry("successful update simple VMI", []string{"src1"}, []virtstoragev1alpha1.MigratedVolume{{SourceClaim: "src1", DestinationClaim: "dest1"}}, nil),
				Entry("successful update VMI with multiple PVCs", []string{"src1", "src2", "src3"}, []virtstoragev1alpha1.MigratedVolume{{SourceClaim: "src1", DestinationClaim: "dest1"}}, nil),
				Entry("successful update VMI with multiple PVCs and migrated volumes", []string{"src1", "src2", "src3"}, []virtstoragev1alpha1.MigratedVolume{
					{SourceClaim: "src1", DestinationClaim: "dest1"},
					{SourceClaim: "src2", DestinationClaim: "dest2"},
					{SourceClaim: "src3", DestinationClaim: "dest3"},
				}, nil),
				Entry("failed to update missing migrated volume", []string{"src1", "src2", "src3"},
					[]virtstoragev1alpha1.MigratedVolume{{SourceClaim: "src4", DestinationClaim: "dest4"}},
					fmt.Errorf("failed to replace the source volumes with the destination volumes in the VMI")),
			)
		})

		It("should not update volumes if volume migration hasn't succeeded", func() {
			phase := virtstoragev1alpha1.VolumeMigrationPhaseRunning
			migVols := []virtstoragev1alpha1.MigratedVolume{{SourceClaim: "src", DestinationClaim: "dest"}}
			vmi := createVMIWithPVCs("testvmi", testNs, "src")
			updateMigrateVolumesVMIStatus(vmi, migVols, &phase)
			vmiUpdated, err := updater.UpdateVMIWithMigrationVolumes(vmi)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(equality.Semantic.DeepEqual(vmi, vmiUpdated)).Should(BeTrue())

		})
	})

	Context("Update volumes on the VM", func() {
		Context("Update volumes after successful migration on a VMI", func() {
			var (
				vmInterface *kubecli.MockVirtualMachineInterface
			)
			BeforeEach(func() {
				vmInterface = kubecli.NewMockVirtualMachineInterface(ctrl)
				virtClient.EXPECT().VirtualMachine(gomock.Any()).Return(vmInterface).AnyTimes()
				vmInterface.EXPECT().Update(context.Background(), gomock.Any()).AnyTimes()
			})

			DescribeTable("UpdateVMWithMigrationVolumes",
				func(pvcs []string, migVols []virtstoragev1alpha1.MigratedVolume, expectedErr error) {
					phase := virtstoragev1alpha1.VolumeMigrationPhaseSucceeded
					vmi := createVMIWithPVCs("testvmi", testNs, pvcs...)
					updateMigrateVolumesVMIStatus(vmi, migVols, &phase)
					vm := createVMFromVMI(vmi)

					vm, err := updater.UpdateVMWithMigrationVolumes(vm, vmi)
					if expectedErr != nil {
						Expect(err).Should(MatchError(expectedErr))
						return
					}
					Expect(err).ShouldNot(HaveOccurred())
					mapVol := make(map[string]bool)
					for _, v := range migVols {
						mapVol[v.DestinationClaim] = true
					}
					for _, v := range vm.Spec.Template.Spec.Volumes {
						name := storagetypes.PVCNameFromVirtVolume(&v)
						if name == "" {
							continue
						}
						if _, ok := mapVol[name]; ok {
							delete(mapVol, name)
						}
					}
					Expect(mapVol).Should(BeEmpty())
				},
				Entry("successful update simple VM", []string{"src1"}, []virtstoragev1alpha1.MigratedVolume{{SourceClaim: "src1", DestinationClaim: "dest1"}}, nil),
				Entry("successful update VM with multiple PVCs", []string{"src1", "src2", "src3"}, []virtstoragev1alpha1.MigratedVolume{{SourceClaim: "src1", DestinationClaim: "dest1"}}, nil),
				Entry("successful update VM with multiple PVCs and migrated volumes", []string{"src1", "src2", "src3"}, []virtstoragev1alpha1.MigratedVolume{
					{SourceClaim: "src1", DestinationClaim: "dest1"},
					{SourceClaim: "src2", DestinationClaim: "dest2"},
					{SourceClaim: "src3", DestinationClaim: "dest3"},
				}, nil),
				Entry("failed to update missing migrated volume", []string{"src1", "src2", "src3"},
					[]virtstoragev1alpha1.MigratedVolume{{SourceClaim: "src4", DestinationClaim: "dest4"}},
					fmt.Errorf("failed to replace the source volumes with the destination volumes in the VM")),
			)
		})

		It("should not update volumes if volume migration hasn't succeeded", func() {
			phase := virtstoragev1alpha1.VolumeMigrationPhaseRunning
			migVols := []virtstoragev1alpha1.MigratedVolume{{SourceClaim: "src", DestinationClaim: "dest"}}
			vmi := createVMIWithPVCs("testvmi", testNs, "src")
			updateMigrateVolumesVMIStatus(vmi, migVols, &phase)
			vm := createVMFromVMI(vmi)
			vmUpdated, err := updater.UpdateVMWithMigrationVolumes(vm, vmi)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(equality.Semantic.DeepEqual(vm, vmUpdated)).Should(BeTrue())

		})

	})

})
