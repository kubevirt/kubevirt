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
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

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

var _ = Describe("Changed Block Tracking", func() {
	var (
		vmi                      *v1.VirtualMachineInstance
		converterContext         *converter.ConverterContext
		createQCOW2OverlayCalled int
		blockDevCalled           int
	)

	setupCBTTest := func() {
		vmi = newVMI(testNamespace, testVmName)
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
	}

	Context("ShouldCreateQCOW2Overlay", func() {
		DescribeTable("should return correct value based on ChangedBlockTracking state and hotplug", func(state v1.ChangedBlockTrackingState, isHotplug bool, hotplugPhase v1.VolumePhase, expected bool) {
			vmi := newVMI(testNamespace, testVmName)
			cbt.SetCBTState(&vmi.Status.ChangedBlockTracking, state)

			result := shouldCreateQCOW2Overlay(vmi, isHotplug, hotplugPhase)
			Expect(result).To(Equal(expected))
		},
			Entry("when state is Initializing", v1.ChangedBlockTrackingInitializing, false, v1.VolumePhase(""), true),
			Entry("when state is Enabled", v1.ChangedBlockTrackingEnabled, false, v1.VolumePhase(""), false),
			Entry("when state is Disabled", v1.ChangedBlockTrackingDisabled, false, v1.VolumePhase(""), false),
			Entry("when state is Undefined", v1.ChangedBlockTrackingUndefined, false, v1.VolumePhase(""), false),
			Entry("when state is Initializing and hotplug mounted", v1.ChangedBlockTrackingInitializing, true, v1.HotplugVolumeMounted, true),
			Entry("when state is Enabled and hotplug mounted", v1.ChangedBlockTrackingEnabled, true, v1.HotplugVolumeMounted, true),
			Entry("when state is Enabled and hotplug ready", v1.ChangedBlockTrackingEnabled, true, v1.VolumeReady, false),
			Entry("when state is Disabled and hotplug mounted", v1.ChangedBlockTrackingDisabled, true, v1.HotplugVolumeMounted, false),
			Entry("when state is Undefined and hotplug mounted", v1.ChangedBlockTrackingUndefined, true, v1.HotplugVolumeMounted, false),
		)
	})

	Context("isBackendStorageRWO", func() {
		DescribeTable("should return correct value based on migration state",
			func(sourcePVC, targetPVC string, expected bool) {
				vmi := newVMI(testNamespace, testVmName)
				vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
					SourcePersistentStatePVCName: sourcePVC,
					TargetPersistentStatePVCName: targetPVC,
				}

				result := isBackendStorageRWO(vmi)
				Expect(result).To(Equal(expected))
			},
			Entry("RWX backend storage", "shared-pvc", "shared-pvc", false),
			Entry("RWO backend storage", "source-pvc", "target-pvc", true),
		)
	})

	Context("ApplyChangedBlockTracking", func() {
		BeforeEach(func() {
			setupCBTTest()
			cbt.SetCBTState(&vmi.Status.ChangedBlockTracking, v1.ChangedBlockTrackingInitializing)
		})

		It("should skip volumes that don't support CBT", func() {
			vmi.Spec.Volumes = []v1.Volume{
				newConfigMapVolume("config-map-volume"),
				newSecretVolume("secret-volume"),
			}

			err := ApplyChangedBlockTracking(vmi, converterContext)
			Expect(err).ToNot(HaveOccurred())
			Expect(converterContext.ApplyCBT).To(BeEmpty())
			Expect(createQCOW2OverlayCalled).To(Equal(0))
		})

		It("should process fs volumes that support CBT", func() {
			vmi.Spec.Volumes = []v1.Volume{
				newPVCVolume("pvc-volume", "test-pvc", false),
				newDVVolume("dv-volume", "test-dv", false),
				newHostDiskVolume("host-disk-volume", "/path/to/disk"),
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
				newPVCVolume("pvc-volume", "test-pvc", false),
				newDVVolume("dv-volume", "test-dv", false),
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

		It("should process hotplug volumes with correct paths", func() {
			vmi.Spec.Volumes = []v1.Volume{
				newPVCVolume("hotplug-pvc-volume", "test-hotplug-pvc", true),
				newDVVolume("hotplug-dv-volume", "test-hotplug-dv", true),
			}
			converterContext.IsBlockPVC["hotplug-pvc-volume"] = false
			converterContext.IsBlockDV["hotplug-dv-volume"] = false
			converterContext.HotplugVolumes = map[string]v1.VolumeStatus{
				"hotplug-pvc-volume": {Name: "hotplug-pvc-volume", Phase: v1.HotplugVolumeMounted, HotplugVolume: &v1.HotplugVolumeStatus{}},
				"hotplug-dv-volume":  {Name: "hotplug-dv-volume", Phase: v1.HotplugVolumeMounted, HotplugVolume: &v1.HotplugVolumeStatus{}},
			}

			var capturedPaths []string
			CreateQCOW2Overlay = func(overlayPath, imagePath string, blockDev bool) error {
				createQCOW2OverlayCalled++
				capturedPaths = append(capturedPaths, imagePath)
				return nil
			}

			err := ApplyChangedBlockTracking(vmi, converterContext)
			Expect(err).ToNot(HaveOccurred())
			Expect(createQCOW2OverlayCalled).To(Equal(2))
			Expect(converterContext.ApplyCBT).To(HaveKey("hotplug-pvc-volume"))
			Expect(converterContext.ApplyCBT).To(HaveKey("hotplug-dv-volume"))
			// Verify hotplug paths are used
			Expect(capturedPaths).To(ContainElement(converter.GetHotplugFilesystemVolumePath("hotplug-pvc-volume")))
			Expect(capturedPaths).To(ContainElement(converter.GetHotplugFilesystemVolumePath("hotplug-dv-volume")))
		})

		It("should process hotplug block volumes with correct paths", func() {
			vmi.Spec.Volumes = []v1.Volume{
				newPVCVolume("hotplug-block-volume", "test-hotplug-block-pvc", true),
			}
			converterContext.IsBlockPVC["hotplug-block-volume"] = true
			converterContext.HotplugVolumes = map[string]v1.VolumeStatus{
				"hotplug-block-volume": {Name: "hotplug-block-volume", Phase: v1.HotplugVolumeMounted, HotplugVolume: &v1.HotplugVolumeStatus{}},
			}

			var capturedPath string
			CreateQCOW2Overlay = func(overlayPath, imagePath string, blockDev bool) error {
				createQCOW2OverlayCalled++
				capturedPath = imagePath
				Expect(blockDev).To(BeTrue())
				return nil
			}

			err := ApplyChangedBlockTracking(vmi, converterContext)
			Expect(err).ToNot(HaveOccurred())
			Expect(createQCOW2OverlayCalled).To(Equal(1))
			Expect(converterContext.ApplyCBT).To(HaveKey("hotplug-block-volume"))
			// Verify hotplug block path is used
			Expect(capturedPath).To(Equal(converter.GetHotplugBlockDeviceVolumePath("hotplug-block-volume")))
		})

		It("should apply cbt to domain but skip creation when CBT is already enabled", func() {
			vmi.Spec.Volumes = []v1.Volume{
				newPVCVolume("pvc-volume", "test-pvc", false),
			}
			cbt.SetCBTState(&vmi.Status.ChangedBlockTracking, v1.ChangedBlockTrackingEnabled)

			err := ApplyChangedBlockTracking(vmi, converterContext)
			Expect(err).ToNot(HaveOccurred())
			Expect(createQCOW2OverlayCalled).To(Equal(0))
			Expect(converterContext.ApplyCBT).To(HaveKey("pvc-volume"))
			Expect(converterContext.ApplyCBT["pvc-volume"]).To(ContainSubstring("pvc-volume.qcow2"))
		})

		It("should create overlay for hotplug volume when CBT is already enabled", func() {
			vmi.Spec.Volumes = []v1.Volume{
				newPVCVolume("hotplug-pvc-volume", "test-pvc", true),
			}
			cbt.SetCBTState(&vmi.Status.ChangedBlockTracking, v1.ChangedBlockTrackingEnabled)
			converterContext.HotplugVolumes = map[string]v1.VolumeStatus{
				"hotplug-pvc-volume": {Name: "hotplug-pvc-volume", Phase: v1.HotplugVolumeMounted, HotplugVolume: &v1.HotplugVolumeStatus{}},
			}

			err := ApplyChangedBlockTracking(vmi, converterContext)
			Expect(err).ToNot(HaveOccurred())
			Expect(createQCOW2OverlayCalled).To(Equal(1))
			Expect(converterContext.ApplyCBT).To(HaveKey("hotplug-pvc-volume"))
			Expect(converterContext.ApplyCBT["hotplug-pvc-volume"]).To(ContainSubstring("hotplug-pvc-volume.qcow2"))
		})

		It("should skip overlay creation for hotplug volume when phase is VolumeReady", func() {
			vmi.Spec.Volumes = []v1.Volume{
				newPVCVolume("hotplug-pvc-volume", "test-pvc", true),
			}
			cbt.SetCBTState(&vmi.Status.ChangedBlockTracking, v1.ChangedBlockTrackingEnabled)
			converterContext.HotplugVolumes = map[string]v1.VolumeStatus{
				"hotplug-pvc-volume": {Name: "hotplug-pvc-volume", Phase: v1.VolumeReady, HotplugVolume: &v1.HotplugVolumeStatus{}},
			}

			err := ApplyChangedBlockTracking(vmi, converterContext)
			Expect(err).ToNot(HaveOccurred())
			Expect(createQCOW2OverlayCalled).To(Equal(0))
			Expect(converterContext.ApplyCBT).To(HaveKey("hotplug-pvc-volume"))
			Expect(converterContext.ApplyCBT["hotplug-pvc-volume"]).To(ContainSubstring("hotplug-pvc-volume.qcow2"))
		})

		It("should return error when overlay creation fails", func() {
			vmi.Spec.Volumes = []v1.Volume{
				newPVCVolume("pvc-volume", "test-pvc", false),
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

	Context("runOverlayQMPSession", func() {
		const overlayPath = "/test/overlay.qcow2"
		const overlaySize int64 = 1024

		It("should send dismiss and quit only after concluded", func() {
			qmpOutput := strings.Join([]string{
				`{"QMP": {"version": {"qemu": {"micro": 0, "minor": 2, "major": 9}}}}`,
				`{"return": {}}`,
				`{"timestamp": {"seconds": 1, "microseconds": 0}, "event": "JOB_STATUS_CHANGE", "data": {"status": "created", "id": "create"}}`,
				`{"timestamp": {"seconds": 1, "microseconds": 0}, "event": "JOB_STATUS_CHANGE", "data": {"status": "running", "id": "create"}}`,
				`{"return": {}}`,
				`{"timestamp": {"seconds": 1, "microseconds": 0}, "event": "JOB_STATUS_CHANGE", "data": {"status": "waiting", "id": "create"}}`,
				`{"timestamp": {"seconds": 1, "microseconds": 0}, "event": "JOB_STATUS_CHANGE", "data": {"status": "pending", "id": "create"}}`,
				`{"timestamp": {"seconds": 1, "microseconds": 0}, "event": "JOB_STATUS_CHANGE", "data": {"status": "concluded", "id": "create"}}`,
				`{"return": [{"id": "create", "type": "create", "status": "concluded"}]}`,
				`{"return": {}}`,
				`{"return": {}}`,
			}, "\n")
			stdout := strings.NewReader(qmpOutput)

			var stdinBuf writeCloserBuffer
			ctx := context.Background()

			output, err := runOverlayQMPSession(ctx, &stdinBuf, stdout, overlaySize, overlayPath)
			Expect(err).ToNot(HaveOccurred())
			Expect(output).To(ContainSubstring(`"status": "concluded"`))

			written := stdinBuf.String()
			Expect(written).To(ContainSubstring("blockdev-create"))
			Expect(written).To(ContainSubstring("query-jobs"))
			Expect(written).To(ContainSubstring("job-dismiss"))
			Expect(written).To(ContainSubstring("quit"))
		})

		It("should return error when job concludes with error", func() {
			qmpOutput := strings.Join([]string{
				`{"QMP": {"version": {"qemu": {"micro": 0, "minor": 2, "major": 9}}}}`,
				`{"return": {}}`,
				`{"timestamp": {"seconds": 1, "microseconds": 0}, "event": "JOB_STATUS_CHANGE", "data": {"status": "created", "id": "create"}}`,
				`{"timestamp": {"seconds": 1, "microseconds": 0}, "event": "JOB_STATUS_CHANGE", "data": {"status": "running", "id": "create"}}`,
				`{"timestamp": {"seconds": 1, "microseconds": 0}, "event": "JOB_STATUS_CHANGE", "data": {"status": "aborting", "id": "create"}}`,
				`{"timestamp": {"seconds": 1, "microseconds": 0}, "event": "JOB_STATUS_CHANGE", "data": {"status": "concluded", "id": "create"}}`,
				`{"return": [{"id": "create", "type": "create", "status": "concluded", "error": "Could not create file: No such file or directory"}]}`,
				`{"return": {}}`,
				`{"return": {}}`,
			}, "\n")
			stdout := strings.NewReader(qmpOutput)

			var stdinBuf writeCloserBuffer
			ctx := context.Background()

			_, err := runOverlayQMPSession(ctx, &stdinBuf, stdout, overlaySize, overlayPath)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("blockdev-create job failed"))
			Expect(err.Error()).To(ContainSubstring("Could not create file"))

			written := stdinBuf.String()
			Expect(written).To(ContainSubstring("query-jobs"))
			Expect(written).To(ContainSubstring("job-dismiss"))
			Expect(written).To(ContainSubstring("quit"))
		})

		It("should still send init commands when daemon exits without concluding", func() {
			qmpOutput := strings.Join([]string{
				`{"QMP": {"version": {"qemu": {"micro": 0, "minor": 2, "major": 9}}}}`,
				`{"return": {}}`,
				`{"error": {"class": "GenericError", "desc": "something went wrong"}}`,
			}, "\n")
			stdout := strings.NewReader(qmpOutput)

			var stdinBuf writeCloserBuffer
			ctx := context.Background()

			_, err := runOverlayQMPSession(ctx, &stdinBuf, stdout, overlaySize, overlayPath)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("exited without job concluding"))
			Expect(err.Error()).To(ContainSubstring(overlayPath))

			written := stdinBuf.String()
			Expect(written).To(ContainSubstring("qmp_capabilities"))
			Expect(written).To(ContainSubstring("blockdev-create"))
			Expect(written).NotTo(ContainSubstring("job-dismiss"))
		})

		It("should return error on context timeout", func() {
			stdoutR, stdoutW := io.Pipe()
			defer stdoutW.Close()

			var stdinBuf writeCloserBuffer
			ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
			defer cancel()

			go func() {
				<-ctx.Done()
				stdoutW.Close()
			}()

			_, err := runOverlayQMPSession(ctx, &stdinBuf, stdoutR, overlaySize, overlayPath)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("timed out"))
			Expect(err.Error()).To(ContainSubstring(overlayPath))

			Expect(stdinBuf.String()).NotTo(ContainSubstring("job-dismiss"))
		})

		It("should include overlay size in blockdev-create command", func() {
			qmpOutput := `{"event": "JOB_STATUS_CHANGE", "data": {"status": "concluded", "id": "create"}}`
			stdout := strings.NewReader(qmpOutput)

			var stdinBuf writeCloserBuffer
			ctx := context.Background()
			const testSize int64 = 107374182400

			_, err := runOverlayQMPSession(ctx, &stdinBuf, stdout, testSize, overlayPath)
			Expect(err).ToNot(HaveOccurred())
			Expect(stdinBuf.String()).To(ContainSubstring(fmt.Sprintf(`"size": %d`, testSize)))
		})

		It("should not panic on multiple concluded events", func() {
			qmpOutput := strings.Join([]string{
				`{"return": {}}`,
				`{"event": "JOB_STATUS_CHANGE", "data": {"status": "concluded", "id": "create"}}`,
				`{"event": "JOB_STATUS_CHANGE", "data": {"status": "concluded", "id": "create"}}`,
			}, "\n")
			stdout := strings.NewReader(qmpOutput)

			var stdinBuf writeCloserBuffer
			ctx := context.Background()

			Expect(func() {
				_, _ = runOverlayQMPSession(ctx, &stdinBuf, stdout, overlaySize, overlayPath)
			}).ToNot(Panic())
		})

		It("should capture output lines after concluded event", func() {
			qmpOutput := strings.Join([]string{
				`{"event": "JOB_STATUS_CHANGE", "data": {"status": "concluded", "id": "create"}}`,
				`{"return": {}}`,
				`{"return": {}}`,
				`{"event": "SHUTDOWN"}`,
			}, "\n")
			stdout := strings.NewReader(qmpOutput)

			var stdinBuf writeCloserBuffer
			ctx := context.Background()

			output, err := runOverlayQMPSession(ctx, &stdinBuf, stdout, overlaySize, overlayPath)
			Expect(err).ToNot(HaveOccurred())
			Expect(output).To(ContainSubstring("SHUTDOWN"))
		})

		It("should return error on empty stdout", func() {
			stdout := strings.NewReader("")

			var stdinBuf writeCloserBuffer
			ctx := context.Background()

			_, err := runOverlayQMPSession(ctx, &stdinBuf, stdout, overlaySize, overlayPath)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("exited without job concluding"))
		})
	})

	Context("ApplyChangedBlockTrackingForMigration", func() {
		BeforeEach(func() {
			setupCBTTest()
		})

		It("should skip volumes that don't support CBT", func() {
			vmi.Spec.Volumes = []v1.Volume{
				newConfigMapVolume("config-map-volume"),
				newSecretVolume("secret-volume"),
			}

			err := ApplyChangedBlockTrackingForMigration(vmi, converterContext)
			Expect(err).ToNot(HaveOccurred())
			Expect(converterContext.ApplyCBT).To(BeEmpty())
			Expect(createQCOW2OverlayCalled).To(Equal(0))
		})

		It("should use existing overlay for RWX backend storage (no overlay creation)", func() {
			vmi.Spec.Volumes = []v1.Volume{
				newPVCVolume("pvc-volume", "test-pvc", false),
				newDVVolume("dv-volume", "test-dv", false),
			}
			setRWXMigrationState(vmi)

			err := ApplyChangedBlockTrackingForMigration(vmi, converterContext)
			Expect(err).ToNot(HaveOccurred())
			Expect(createQCOW2OverlayCalled).To(Equal(0))
			Expect(converterContext.ApplyCBT).To(HaveKey("pvc-volume"))
			Expect(converterContext.ApplyCBT["pvc-volume"]).To(ContainSubstring("pvc-volume.qcow2"))
			Expect(converterContext.ApplyCBT).To(HaveKey("dv-volume"))
			Expect(converterContext.ApplyCBT["dv-volume"]).To(ContainSubstring("dv-volume.qcow2"))
		})

		DescribeTable("should create overlays for RWO backend storage",
			func(isBlock bool, expectedPathFunc func(string) string) {
				vmi.Spec.Volumes = []v1.Volume{
					newPVCVolume("pvc-volume", "test-pvc", false),
					newDVVolume("dv-volume", "test-dv", false),
				}
				converterContext.IsBlockPVC["pvc-volume"] = isBlock
				converterContext.IsBlockDV["dv-volume"] = isBlock
				setRWOMigrationState(vmi)

				var capturedPaths []string
				CreateQCOW2Overlay = func(overlayPath, imagePath string, blockDev bool) error {
					createQCOW2OverlayCalled++
					capturedPaths = append(capturedPaths, imagePath)
					Expect(blockDev).To(Equal(isBlock))
					return nil
				}

				err := ApplyChangedBlockTrackingForMigration(vmi, converterContext)
				Expect(err).ToNot(HaveOccurred())
				Expect(createQCOW2OverlayCalled).To(Equal(2))
				Expect(converterContext.ApplyCBT).To(HaveKey("pvc-volume"))
				Expect(converterContext.ApplyCBT).To(HaveKey("dv-volume"))
				Expect(capturedPaths).To(ContainElement(expectedPathFunc("pvc-volume")))
				Expect(capturedPaths).To(ContainElement(expectedPathFunc("dv-volume")))
			},
			Entry("filesystem volumes", false, converter.GetFilesystemVolumePath),
			Entry("block volumes", true, converter.GetBlockDeviceVolumePath),
		)

		DescribeTable("should create overlays for hotplug volumes with RWO backend",
			func(volumeName string, isBlock bool, expectedPathFunc func(string) string) {
				vmi.Spec.Volumes = []v1.Volume{
					newPVCVolume(volumeName, "test-hotplug-pvc", true),
				}
				converterContext.IsBlockPVC[volumeName] = isBlock
				converterContext.HotplugVolumes = map[string]v1.VolumeStatus{
					volumeName: {Name: volumeName, Phase: v1.VolumeReady, HotplugVolume: &v1.HotplugVolumeStatus{}},
				}
				setRWOMigrationState(vmi)

				var capturedPath string
				CreateQCOW2Overlay = func(overlayPath, imagePath string, blockDev bool) error {
					createQCOW2OverlayCalled++
					capturedPath = imagePath
					Expect(blockDev).To(Equal(isBlock))
					return nil
				}

				err := ApplyChangedBlockTrackingForMigration(vmi, converterContext)
				Expect(err).ToNot(HaveOccurred())
				Expect(createQCOW2OverlayCalled).To(Equal(1))
				Expect(converterContext.ApplyCBT).To(HaveKey(volumeName))
				Expect(capturedPath).To(Equal(expectedPathFunc(volumeName)))
			},
			Entry("filesystem volume", "hotplug-fs-volume", false, converter.GetHotplugFilesystemVolumePath),
			Entry("block volume", "hotplug-block-volume", true, converter.GetHotplugBlockDeviceVolumePath),
		)

		It("should use existing overlay for hotplug volumes with RWX backend (no overlay creation)", func() {
			vmi.Spec.Volumes = []v1.Volume{
				newPVCVolume("hotplug-pvc-volume", "test-hotplug-pvc", true),
			}
			converterContext.HotplugVolumes = map[string]v1.VolumeStatus{
				"hotplug-pvc-volume": {Name: "hotplug-pvc-volume", Phase: v1.VolumeReady, HotplugVolume: &v1.HotplugVolumeStatus{}},
			}
			setRWXMigrationState(vmi)

			err := ApplyChangedBlockTrackingForMigration(vmi, converterContext)
			Expect(err).ToNot(HaveOccurred())
			Expect(createQCOW2OverlayCalled).To(Equal(0))
			Expect(converterContext.ApplyCBT).To(HaveKey("hotplug-pvc-volume"))
		})
	})
})

type writeCloserBuffer struct {
	bytes.Buffer
}

func (w *writeCloserBuffer) Close() error { return nil }

func newVMI(namespace, name string) *v1.VirtualMachineInstance {
	vmi := api2.NewMinimalVMIWithNS(namespace, name)
	v1.SetObjectDefaults_VirtualMachineInstance(vmi)
	return vmi
}

func newPVCVolume(name, claimName string, hotpluggable bool) v1.Volume {
	return v1.Volume{
		Name: name,
		VolumeSource: v1.VolumeSource{
			PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
				PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
					ClaimName: claimName,
				},
				Hotpluggable: hotpluggable,
			},
		},
	}
}

func newDVVolume(name, dvName string, hotpluggable bool) v1.Volume {
	return v1.Volume{
		Name: name,
		VolumeSource: v1.VolumeSource{
			DataVolume: &v1.DataVolumeSource{
				Name:         dvName,
				Hotpluggable: hotpluggable,
			},
		},
	}
}

func newConfigMapVolume(name string) v1.Volume {
	return v1.Volume{
		Name: name,
		VolumeSource: v1.VolumeSource{
			ConfigMap: &v1.ConfigMapVolumeSource{},
		},
	}
}

func newSecretVolume(name string) v1.Volume {
	return v1.Volume{
		Name: name,
		VolumeSource: v1.VolumeSource{
			Secret: &v1.SecretVolumeSource{},
		},
	}
}

func newHostDiskVolume(name, path string) v1.Volume {
	return v1.Volume{
		Name: name,
		VolumeSource: v1.VolumeSource{
			HostDisk: &v1.HostDisk{
				Path: path,
			},
		},
	}
}

func setRWOMigrationState(vmi *v1.VirtualMachineInstance) {
	vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
		SourcePersistentStatePVCName: "source-pvc",
		TargetPersistentStatePVCName: "target-pvc",
	}
}

func setRWXMigrationState(vmi *v1.VirtualMachineInstance) {
	vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
		SourcePersistentStatePVCName: "shared-pvc",
		TargetPersistentStatePVCName: "shared-pvc",
	}
}
