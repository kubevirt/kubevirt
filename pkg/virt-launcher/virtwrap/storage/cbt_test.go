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
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"

	v1 "kubevirt.io/api/core/v1"
	api2 "kubevirt.io/client-go/api"

	"kubevirt.io/kubevirt/pkg/storage/cbt"
	"kubevirt.io/kubevirt/pkg/storage/volumepath"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	convertertypes "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/types"
)

const (
	testVmName    = "testvmi"
	testNamespace = "testnamespace"
)

var _ = Describe("Changed Block Tracking", func() {
	var (
		vmi                      *v1.VirtualMachineInstance
		converterContext         *convertertypes.ConverterContext
		createQCOW2OverlayCalled int
	)

	setupCBTTest := func() {
		vmi = newVMI(testNamespace, testVmName)
		converterContext = &convertertypes.ConverterContext{
			IsBlockPVC: make(map[string]bool),
			IsBlockDV:  make(map[string]bool),
		}
		createQCOW2OverlayCalled = 0
		CreateQCOW2Overlay = func(overlayPath, imagePath string) error {
			createQCOW2OverlayCalled++
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

	Context("createQCOW2Overlay", func() {
		var dir string

		BeforeEach(func() {
			dir = GinkgoT().TempDir()
		})

		It("should reuse an existing overlay", func() {
			overlay := filepath.Join(dir, "disk0.qcow2")
			Expect(os.WriteFile(overlay, []byte("overlay"), 0o600)).To(Succeed())

			Expect(createQCOW2Overlay(overlay, filepath.Join(dir, "missing.img"))).To(Succeed())
			Expect(os.ReadFile(overlay)).To(Equal([]byte("overlay")))
		})

		It("should refuse an overlay that is a symlink", func() {
			target := filepath.Join(dir, "target")
			Expect(os.WriteFile(target, []byte("target"), 0o600)).To(Succeed())
			overlay := filepath.Join(dir, "disk0.qcow2")
			Expect(os.Symlink(target, overlay)).To(Succeed())

			Expect(createQCOW2Overlay(overlay, filepath.Join(dir, "missing.img"))).ToNot(Succeed())
			Expect(os.ReadFile(target)).To(Equal([]byte("target")))
		})
	})

	Context("reconcileQCOW2OverlaysInDir", func() {
		var dir string

		BeforeEach(func() {
			dir = GinkgoT().TempDir()
		})

		It("should remove orphaned overlays and keep desired ones", func() {
			diskA := filepath.Join(dir, "disk-a.qcow2")
			diskB := filepath.Join(dir, "disk-b.qcow2")
			Expect(os.WriteFile(diskA, []byte("a"), 0o600)).To(Succeed())
			Expect(os.WriteFile(diskB, []byte("b"), 0o600)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("keep"), 0o600)).To(Succeed())

			Expect(reconcileQCOW2OverlaysInDir(dir, map[string]struct{}{diskA: {}})).To(Succeed())

			Expect(diskA).To(BeAnExistingFile())
			Expect(diskB).NotTo(BeAnExistingFile())
			Expect(filepath.Join(dir, "readme.txt")).To(BeAnExistingFile())
		})

		It("should remove all overlays when the desired set is empty", func() {
			Expect(os.WriteFile(filepath.Join(dir, "disk-a.qcow2"), []byte("a"), 0o600)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(dir, "disk-b.qcow2"), []byte("b"), 0o600)).To(Succeed())

			Expect(reconcileQCOW2OverlaysInDir(dir, nil)).To(Succeed())

			Expect(filepath.Join(dir, "disk-a.qcow2")).NotTo(BeAnExistingFile())
			Expect(filepath.Join(dir, "disk-b.qcow2")).NotTo(BeAnExistingFile())
		})

		It("should remove leftover overlay temp files", func() {
			diskA := filepath.Join(dir, "disk-a.qcow2")
			Expect(os.WriteFile(diskA+".tmp", []byte("tmp"), 0o600)).To(Succeed())
			Expect(os.WriteFile(diskA, []byte("a"), 0o600)).To(Succeed())

			Expect(reconcileQCOW2OverlaysInDir(dir, map[string]struct{}{diskA: {}})).To(Succeed())

			Expect(diskA).To(BeAnExistingFile())
			Expect(diskA + ".tmp").NotTo(BeAnExistingFile())
		})

		It("should skip symlink overlays", func() {
			target := filepath.Join(dir, "target.img")
			link := filepath.Join(dir, "disk-a.qcow2")
			Expect(os.WriteFile(target, []byte("target"), 0o600)).To(Succeed())
			Expect(os.Symlink(target, link)).To(Succeed())

			Expect(reconcileQCOW2OverlaysInDir(dir, nil)).To(Succeed())

			Expect(link).To(BeAnExistingFile())
			Expect(os.ReadFile(target)).To(Equal([]byte("target")))
		})

		It("should remove orphaned overlays under nested directories", func() {
			nested := filepath.Join(dir, "claim", "uid")
			Expect(os.MkdirAll(nested, 0o750)).To(Succeed())
			keep := filepath.Join(dir, "disk-a.qcow2")
			orphan := filepath.Join(nested, "disk-b.qcow2")
			Expect(os.WriteFile(keep, []byte("a"), 0o600)).To(Succeed())
			Expect(os.WriteFile(orphan, []byte("b"), 0o600)).To(Succeed())

			Expect(reconcileQCOW2OverlaysInDir(dir, map[string]struct{}{keep: {}})).To(Succeed())

			Expect(keep).To(BeAnExistingFile())
			Expect(orphan).NotTo(BeAnExistingFile())
		})

		It("should no-op when the CBT directory does not exist", func() {
			Expect(reconcileQCOW2OverlaysInDir(filepath.Join(dir, "missing"), map[string]struct{}{
				filepath.Join(dir, "missing", "disk-a.qcow2"): {},
			})).To(Succeed())
		})
	})

	Context("addLiveDomainCBTOverlayPaths", func() {
		It("should only keep disk sources under the CBT directory", func() {
			dir := GinkgoT().TempDir()
			under := filepath.Join(dir, "disk-a.qcow2")
			nested := filepath.Join(dir, "claim", "uid", "disk-a.qcow2")
			outside := filepath.Join(filepath.Dir(dir), "outside.qcow2")

			desired := map[string]struct{}{}
			addLiveDomainCBTOverlayPaths(desired, dir, &api.DomainSpec{
				Devices: api.Devices{
					Disks: []api.Disk{
						{Source: api.DiskSource{File: under}},
						{Source: api.DiskSource{File: nested}},
						{Source: api.DiskSource{File: outside}},
						{Source: api.DiskSource{File: ""}},
					},
				},
			})

			Expect(desired).To(HaveKey(under))
			Expect(desired).To(HaveKey(nested))
			Expect(desired).NotTo(HaveKey(outside))
		})

		It("should no-op for a nil domain spec", func() {
			desired := map[string]struct{}{}
			addLiveDomainCBTOverlayPaths(desired, "/cbt", nil)
			Expect(desired).To(BeEmpty())
		})

		It("should keep a live-domain overlay path that GetQCOW2OverlayPath would not produce", func() {
			// Simulates a migration/upgrade target whose domain still references
			// an older nested layout while this launcher would create flat paths.
			dir := GinkgoT().TempDir()
			currentLayout := filepath.Join(dir, "disk-a.qcow2")
			legacyLayout := filepath.Join(dir, "claim", "uid", "disk-a.qcow2")
			orphan := filepath.Join(dir, "disk-b.qcow2")
			Expect(os.MkdirAll(filepath.Dir(legacyLayout), 0o750)).To(Succeed())
			Expect(os.WriteFile(currentLayout, []byte("a"), 0o600)).To(Succeed())
			Expect(os.WriteFile(legacyLayout, []byte("legacy"), 0o600)).To(Succeed())
			Expect(os.WriteFile(orphan, []byte("b"), 0o600)).To(Succeed())

			desired := map[string]struct{}{currentLayout: {}}
			addLiveDomainCBTOverlayPaths(desired, dir, &api.DomainSpec{
				Devices: api.Devices{
					Disks: []api.Disk{
						{Source: api.DiskSource{File: legacyLayout}},
					},
				},
			})
			Expect(reconcileQCOW2OverlaysInDir(dir, desired)).To(Succeed())

			Expect(currentLayout).To(BeAnExistingFile())
			Expect(legacyLayout).To(BeAnExistingFile())
			Expect(orphan).NotTo(BeAnExistingFile())
		})
	})

	Context("isPathUnderDir", func() {
		DescribeTable("path containment",
			func(path, dir string, expected bool) {
				Expect(isPathUnderDir(path, dir)).To(Equal(expected))
			},
			Entry("nested file", "/cbt/sub/f.qcow2", "/cbt", true),
			Entry("path equals dir", "/cbt", "/cbt", true),
			Entry("empty path", "", "/cbt", false),
			Entry("sibling escape", "/cbt-other/f.qcow2", "/cbt", false),
			Entry("parent traversal", "/cbt/../../etc/passwd", "/cbt", false),
		)
	})

	Context("CleanupChangedBlockTrackingOverlays", func() {
		It("should no-op when ChangedBlockTracking status is nil", func() {
			vmi := newVMI(testNamespace, testVmName)
			Expect(CleanupChangedBlockTrackingOverlays(vmi, nil)).To(Succeed())
		})

		It("should no-op when CBT is still enabled", func() {
			vmi := newVMI(testNamespace, testVmName)
			cbt.SetCBTState(&vmi.Status.ChangedBlockTracking, v1.ChangedBlockTrackingEnabled)
			Expect(CleanupChangedBlockTrackingOverlays(vmi, nil)).To(Succeed())
		})

		It("should skip wipe when the live domain still references a CBT overlay", func() {
			vmi := newVMI(testNamespace, testVmName)
			cbt.SetCBTState(&vmi.Status.ChangedBlockTracking, v1.ChangedBlockTrackingDisabled)
			cbtDir := cbt.PathForCBT(vmi)
			Expect(CleanupChangedBlockTrackingOverlays(vmi, &api.DomainSpec{
				Devices: api.Devices{
					Disks: []api.Disk{
						{Source: api.DiskSource{File: filepath.Join(cbtDir, "disk-a.qcow2")}},
					},
				},
			})).To(Succeed())
		})
	})

	Context("buildCBTOverlayKeepPaths", func() {
		It("should include eligible volumes and skip ineligible ones", func() {
			vmi := newVMI(testNamespace, testVmName)
			vmi.Spec.Volumes = []v1.Volume{
				newPVCVolume("pvc-volume", "test-pvc", false),
				newConfigMapVolume("config-map-volume"),
			}

			_, keep := buildCBTOverlayKeepPaths(vmi, nil)
			Expect(keep).To(HaveKey(cbt.GetQCOW2OverlayPath(vmi, "pvc-volume")))
			Expect(keep).NotTo(HaveKey(cbt.GetQCOW2OverlayPath(vmi, "config-map-volume")))
		})

		It("should keep live-domain overlay paths not produced by GetQCOW2OverlayPath", func() {
			vmi := newVMI(testNamespace, testVmName)
			vmi.Spec.Volumes = []v1.Volume{
				newPVCVolume("disk-a", "test-pvc", false),
			}
			cbtDir := cbt.PathForCBT(vmi)
			legacy := filepath.Join(cbtDir, "claim", "uid", "disk-a.qcow2")

			_, keep := buildCBTOverlayKeepPaths(vmi, &api.DomainSpec{
				Devices: api.Devices{
					Disks: []api.Disk{
						{Source: api.DiskSource{File: legacy}},
					},
				},
			})
			Expect(keep).To(HaveKey(cbt.GetQCOW2OverlayPath(vmi, "disk-a")))
			Expect(keep).To(HaveKey(legacy))
		})

		It("should only contain live-domain paths when no eligible volumes remain", func() {
			vmi := newVMI(testNamespace, testVmName)
			vmi.Spec.Volumes = nil
			cbtDir := cbt.PathForCBT(vmi)
			legacy := filepath.Join(cbtDir, "orphan-still-open.qcow2")

			_, keep := buildCBTOverlayKeepPaths(vmi, &api.DomainSpec{
				Devices: api.Devices{
					Disks: []api.Disk{
						{Source: api.DiskSource{File: legacy}},
					},
				},
			})
			Expect(keep).To(Equal(map[string]struct{}{legacy: {}}))
		})
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

			var capturedPaths []string
			CreateQCOW2Overlay = func(overlayPath, imagePath string) error {
				createQCOW2OverlayCalled++
				capturedPaths = append(capturedPaths, imagePath)
				return nil
			}

			err := ApplyChangedBlockTracking(vmi, converterContext)
			Expect(err).ToNot(HaveOccurred())
			Expect(createQCOW2OverlayCalled).To(Equal(2))
			Expect(capturedPaths).To(ConsistOf(volumepath.BlockDevice("pvc-volume"), volumepath.BlockDevice("dv-volume")))
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
			CreateQCOW2Overlay = func(overlayPath, imagePath string) error {
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
			Expect(capturedPaths).To(ContainElement(volumepath.HotplugFilesystem("hotplug-pvc-volume")))
			Expect(capturedPaths).To(ContainElement(volumepath.HotplugFilesystem("hotplug-dv-volume")))
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
			CreateQCOW2Overlay = func(overlayPath, imagePath string) error {
				createQCOW2OverlayCalled++
				capturedPath = imagePath
				return nil
			}

			err := ApplyChangedBlockTracking(vmi, converterContext)
			Expect(err).ToNot(HaveOccurred())
			Expect(createQCOW2OverlayCalled).To(Equal(1))
			Expect(converterContext.ApplyCBT).To(HaveKey("hotplug-block-volume"))
			// Verify hotplug block path is used
			Expect(capturedPath).To(Equal(volumepath.HotplugBlockDevice("hotplug-block-volume")))
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
			CreateQCOW2Overlay = func(overlayPath, imagePath string) error {
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

	Context("DeleteQCOW2Overlay", func() {
		var dir string

		BeforeEach(func() {
			dir = GinkgoT().TempDir()
		})

		touch := func(path string) string {
			Expect(os.MkdirAll(filepath.Dir(path), 0o755)).To(Succeed())
			Expect(os.WriteFile(path, nil, 0o600)).To(Succeed())
			return path
		}

		overlayDisk := func(overlay string) api.Disk {
			return api.Disk{Source: api.DiskSource{
				File:      overlay,
				DataStore: &api.DataStore{Source: &api.DiskSource{File: "/var/run/kubevirt-private/vmi-disks/disk0/disk.img"}},
			}}
		}

		DescribeTable("should remove only the file of a flat overlay", func(name string) {
			overlay := touch(filepath.Join(dir, name))
			other := touch(filepath.Join(dir, "other.qcow2"))

			Expect(DeleteQCOW2Overlay(overlayDisk(overlay))).To(Succeed())
			Expect(overlay).ToNot(BeAnExistingFile())
			Expect(other).To(BeAnExistingFile())
		},
			Entry("of a volume", "disk0.qcow2"),
		)

		It("should ignore disks without an overlay", func() {
			image := touch(filepath.Join(dir, "disk.img"))

			Expect(DeleteQCOW2Overlay(api.Disk{Source: api.DiskSource{File: image}})).To(Succeed())
			Expect(image).To(BeAnExistingFile())
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

		DescribeTable("should leave overlay creation to the premigration hook", func(setMigrationState func(*v1.VirtualMachineInstance)) {
			vmi.Spec.Volumes = []v1.Volume{
				newPVCVolume("pvc-volume", "test-pvc", false),
				newDVVolume("hotplug-dv-volume", "test-dv", true),
			}
			converterContext.HotplugVolumes = map[string]v1.VolumeStatus{
				"hotplug-dv-volume": {Name: "hotplug-dv-volume", Phase: v1.VolumeReady, HotplugVolume: &v1.HotplugVolumeStatus{}},
			}
			setMigrationState(vmi)

			Expect(ApplyChangedBlockTrackingForMigration(vmi, converterContext)).To(Succeed())
			Expect(createQCOW2OverlayCalled).To(Equal(0))
			Expect(converterContext.ApplyCBT).To(HaveKeyWithValue("pvc-volume", cbt.GetQCOW2OverlayPath(vmi, "pvc-volume")))
			Expect(converterContext.ApplyCBT).To(HaveKeyWithValue("hotplug-dv-volume", cbt.GetQCOW2OverlayPath(vmi, "hotplug-dv-volume")))
		},
			Entry("with RWO backend storage", setRWOMigrationState),
			Entry("with RWX backend storage", setRWXMigrationState),
		)
	})
})

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
