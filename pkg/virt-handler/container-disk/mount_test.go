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

package container_disk

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"kubevirt.io/client-go/api"

	"kubevirt.io/kubevirt/pkg/checkpoint"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"

	gomock "github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomega_types "github.com/onsi/gomega/types"

	containerdisk "kubevirt.io/kubevirt/pkg/container-disk"

	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"

	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
)

var _ = Describe("ContainerDisk", func() {
	var tmpDir string
	var m *mounter
	var err error
	var vmi *v1.VirtualMachineInstance

	BeforeEach(func() {
		tmpDir, err = os.MkdirTemp("", "containerdisktest")
		Expect(err).ToNot(HaveOccurred())
		vmi = api.NewMinimalVMI("fake-vmi")
		vmi.UID = "1234"

		m = &mounter{
			mountRecords:           make(map[types.UID]*vmiMountTargetRecord),
			checkpointManager:      checkpoint.NewSimpleCheckpointManager(tmpDir),
			suppressWarningTimeout: 1 * time.Minute,
			socketPathGetter:       containerdisk.NewSocketPathGetter(""),
		}
	})

	AfterEach(func() {
		os.RemoveAll(tmpDir)
	})

	BeforeEach(func() {
		vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
			Name: "test",
			VolumeSource: v1.VolumeSource{
				ContainerDisk: testutils.NewFakeContainerDiskSource(),
			},
		})
	})

	detectsSocket := func(detector *isolation.MockPodIsolationDetector) {
		detector.EXPECT().DetectForSocket(vmi, "someting").Return(nil, nil)
	}

	noSocketDetected := func(detector *isolation.MockPodIsolationDetector) {
		detector.EXPECT().DetectForSocket(vmi, "someting").Return(nil, fmt.Errorf("Not Found"))
	}

	detectForSocketNotCalled := func(detector *isolation.MockPodIsolationDetector) {
		detector.EXPECT().DetectForSocket(gomock.Any(), gomock.Any()).Times(0)
	}

	Context("checking if containerDisks are ready", func() {

		DescribeTable("should", func(
			pathGetter containerdisk.SocketPathGetter,
			mockSetup func(*isolation.MockPodIsolationDetector),
			addedDelay time.Duration,
			errorMatcher gomega_types.GomegaMatcher,
			shouldBeReady bool,
		) {
			ctrl := gomock.NewController(GinkgoT())
			mockIsolationDetector := isolation.NewMockPodIsolationDetector(ctrl)
			m.podIsolationDetector = mockIsolationDetector
			mockSetup(mockIsolationDetector)

			m.socketPathGetter = pathGetter
			ready, err := m.ContainerDisksReady(vmi, time.Now().Add(addedDelay))
			Expect(err).To(errorMatcher)
			Expect(ready).To(Equal(shouldBeReady))
		},
			Entry("return false and no error if we are still within the tolerated retry period",
				func(*v1.VirtualMachineInstance, int) (string, error) { return "", fmt.Errorf("not found") },
				detectForSocketNotCalled,
				time.Duration(0),
				Succeed(),
				false,
			),
			Entry("return false and an error if we are outside the tolerated retry period",
				func(*v1.VirtualMachineInstance, int) (string, error) { return "", fmt.Errorf("not found") },
				detectForSocketNotCalled,
				-2*time.Minute,
				HaveOccurred(),
				false,
			),
			Entry("return true and no error once everything is ready and we are within the tolerated retry period",
				func(*v1.VirtualMachineInstance, int) (string, error) { return "someting", nil },
				detectsSocket,
				time.Duration(0),
				Succeed(),
				true,
			),
			Entry("return true and no error once everything is ready when we are outside of the tolerated retry period",
				func(*v1.VirtualMachineInstance, int) (string, error) { return "someting", nil },
				detectsSocket,
				-2*time.Minute,
				Succeed(),
				true,
			),
			Entry("return false and no error if we are still within the tolerated retry period but socket is not there",
				func(*v1.VirtualMachineInstance, int) (string, error) { return "someting", nil },
				noSocketDetected,
				time.Duration(0),
				Succeed(),
				false,
			),
			Entry("return false and an error if we are outside the tolerated retry period but socket is not there",
				func(*v1.VirtualMachineInstance, int) (string, error) { return "someting", nil },
				noSocketDetected,
				-2*time.Minute,
				HaveOccurred(),
				false,
			),
		)

		Context("with kernelBoot container", func() {

			BeforeEach(func() {
				vmi.Spec.Volumes = []v1.Volume{}

				vmi.Spec.Domain.Firmware = &v1.Firmware{
					KernelBoot: &v1.KernelBoot{
						Container: &v1.KernelBootContainer{
							KernelPath: "/fake-kernel",
							InitrdPath: "/fake-initrd",
						},
					},
				}
			})

			DescribeTable("should", func(
				pathGetter containerdisk.KernelBootSocketPathGetter,
				mockSetup func(*isolation.MockPodIsolationDetector),
				addedDelay time.Duration,
				errorMatcher gomega_types.GomegaMatcher,
				shouldBeReady bool,
			) {
				ctrl := gomock.NewController(GinkgoT())
				mockIsolationDetector := isolation.NewMockPodIsolationDetector(ctrl)
				m.podIsolationDetector = mockIsolationDetector
				mockSetup(mockIsolationDetector)

				m.kernelBootSocketPathGetter = pathGetter
				ready, err := m.ContainerDisksReady(vmi, time.Now().Add(addedDelay))
				Expect(err).To(errorMatcher)
				Expect(ready).To(Equal(shouldBeReady))
			},
				Entry("return false and no error if we are still within the tolerated retry period",
					func(*v1.VirtualMachineInstance) (string, error) { return "", fmt.Errorf("not found") },
					detectForSocketNotCalled,
					time.Duration(0),
					Succeed(),
					false,
				),
				Entry("return false and an error if we are outside the tolerated retry period",
					func(*v1.VirtualMachineInstance) (string, error) { return "", fmt.Errorf("not found") },
					detectForSocketNotCalled,
					-2*time.Minute,
					HaveOccurred(),
					false,
				),
				Entry("return true and no error once everything is ready and we are within the tolerated retry period",
					func(*v1.VirtualMachineInstance) (string, error) { return "someting", nil },
					detectsSocket,
					time.Duration(0),
					Succeed(),
					true,
				),
				Entry("return true and no error once everything is ready when we are outside of the tolerated retry period",
					func(*v1.VirtualMachineInstance) (string, error) { return "someting", nil },
					detectsSocket,
					-2*time.Minute,
					Succeed(),
					true,
				),

				Entry("return false and no error if we are still within the tolerated retry period but socket is not there",
					func(*v1.VirtualMachineInstance) (string, error) { return "someting", nil },
					noSocketDetected,
					time.Duration(0),
					Succeed(),
					false,
				),
				Entry("return false and an error if we are outside the tolerated retry period but socket is not there",
					func(*v1.VirtualMachineInstance) (string, error) { return "someting", nil },
					noSocketDetected,
					-2*time.Minute,
					HaveOccurred(),
					false,
				),
			)
		})
	})

	Context("verify mount target recording for vmi", func() {
		It("should set and get same results", func() {

			// verify reading non-existent results just returns empty slice
			record, err := m.getMountTargetRecord(vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(record).To(BeNil())

			// verify setting a result works
			record = &vmiMountTargetRecord{
				MountTargetEntries: []vmiMountTargetEntry{
					{
						TargetFile: "sometargetfile",
						SocketFile: "somesocketfile",
					},
				},
			}
			err = m.setMountTargetRecord(vmi, record)
			Expect(err).ToNot(HaveOccurred())

			// verify the file actually exists
			recordFile := filepath.Join(tmpDir, string(vmi.UID))
			exists, err := diskutils.FileExists(recordFile)
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeTrue())

			// verify we can read a result
			record, err = m.getMountTargetRecord(vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(record.MountTargetEntries).To(HaveLen(1))
			Expect(record.MountTargetEntries[0].TargetFile).To(Equal("sometargetfile"))
			Expect(record.MountTargetEntries[0].SocketFile).To(Equal("somesocketfile"))

			// verify we can read a result directly from disk if the entry
			// doesn't exist in the map
			delete(m.mountRecords, vmi.UID)
			record, err = m.getMountTargetRecord(vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(record.MountTargetEntries).To(HaveLen(1))
			Expect(record.MountTargetEntries[0].TargetFile).To(Equal("sometargetfile"))
			Expect(record.MountTargetEntries[0].SocketFile).To(Equal("somesocketfile"))

			// verify the cache is populated again with the mount info after reading from disk
			_, ok := m.mountRecords[vmi.UID]
			Expect(ok).To(BeTrue())

			// verify delete results
			err = m.deleteMountTargetRecord(vmi)
			Expect(err).ToNot(HaveOccurred())

			// verify the file is actually removed
			exists, err = diskutils.FileExists(recordFile)
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeFalse())

			// verify deleting results that don't exist won't fail
			err = m.deleteMountTargetRecord(vmi)
			Expect(err).ToNot(HaveOccurred())

			// verify reading deleted results just returns empty slice
			record, err = m.getMountTargetRecord(vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(record).To(BeNil())
		})
	})
})
