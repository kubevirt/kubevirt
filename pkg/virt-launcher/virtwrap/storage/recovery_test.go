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
	"encoding/json"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	osdisk "kubevirt.io/kubevirt/pkg/os/disk"
	"kubevirt.io/kubevirt/pkg/pointer"
	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
	"kubevirt.io/kubevirt/pkg/virt-launcher/metadata"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("Overlay recovery", func() {
	var (
		dir     string
		vmi     *v1.VirtualMachineInstance
		domain  *api.Domain
		backing map[string]string
		format  string
	)

	fileDisk := func(volume, device, path string) api.Disk {
		return api.Disk{
			Alias: api.NewUserDefinedAlias(volume), Target: api.DiskTarget{Device: device},
			Type: "file", Driver: &api.DiskDriver{Type: "raw"}, Source: api.DiskSource{File: path},
		}
	}

	BeforeEach(func() {
		dir = GinkgoT().TempDir()
		vmi = libvmi.New(libvmi.WithDataVolume("root", "root-dv"), libvmi.WithPersistentVolumeClaim("data", "data-pvc"))
		domain = &api.Domain{}
		// Device letters can change on a replacement launcher, volume names do not
		domain.Spec.Devices.Disks = []api.Disk{
			fileDisk("data", "vda", "/base/data.img"),
			fileDisk("root", "vdb", "/base/root.img"),
		}
		backing = map[string]string{"root": "/base/root.img", "data": "/base/data.img"}
		format = "qcow2"
		for name := range backing {
			Expect(os.WriteFile(overlayPath(dir, name), []byte("qcow2"), 0o600)).To(Succeed())
		}

		diskInfo := overlayDiskInfo
		overlayDiskInfo = func(path string) (*osdisk.DiskInfo, error) {
			if _, err := os.Stat(path); err != nil {
				return nil, err
			}
			name := filepath.Base(path)
			name = name[len(overlayFilePrefix) : len(name)-len(overlayFileSuffix)]
			return &osdisk.DiskInfo{Format: format, BackingFile: backing[name]}, nil
		}
		DeferCleanup(func() { overlayDiskInfo = diskInfo })
	})

	diskOf := func(volume string) api.Disk {
		for _, disk := range domain.Spec.Devices.Disks {
			if disk.Alias.GetName() == volume {
				return disk
			}
		}
		Fail("no disk for volume " + volume)
		return api.Disk{}
	}

	expectOnOverlay := func(volume string, base api.DiskSource, baseType string) {
		disk := diskOf(volume)
		Expect(disk.Type).To(Equal("file"))
		Expect(disk.Source).To(Equal(api.DiskSource{File: overlayPath(dir, volume)}))
		Expect(disk.Driver.Type).To(Equal("qcow2"))
		Expect(disk.BackingStore).To(Equal(&api.BackingStore{Type: baseType, Format: &api.BackingStoreFormat{Type: "raw"}, Source: &base}))
	}

	It("should start every disk that still has an overlay on it", func() {
		Expect(recoverOverlayDisks(domain, dir)).To(Succeed())
		expectOnOverlay("root", api.DiskSource{File: "/base/root.img"}, "file")
		expectOnOverlay("data", api.DiskSource{File: "/base/data.img"}, "file")
	})

	It("should start a disk without an overlay on its base image", func() {
		Expect(os.Remove(overlayPath(dir, "root"))).To(Succeed())

		Expect(recoverOverlayDisks(domain, dir)).To(Succeed())
		Expect(diskOf("root")).To(Equal(fileDisk("root", "vdb", "/base/root.img")))
		expectOnOverlay("data", api.DiskSource{File: "/base/data.img"}, "file")
	})

	It("should keep a block device as the backing image of its overlay", func() {
		domain.Spec.Devices.Disks[0].Type = "block"
		domain.Spec.Devices.Disks[0].Source = api.DiskSource{Dev: "/dev/data"}
		backing["data"] = "/dev/data"

		Expect(recoverOverlayDisks(domain, dir)).To(Succeed())
		expectOnOverlay("data", api.DiskSource{Dev: "/dev/data"}, "block")
	})

	DescribeTable("should refuse without changing any disk", func(change func()) {
		change()
		original := make([]api.Disk, len(domain.Spec.Devices.Disks))
		for i := range domain.Spec.Devices.Disks {
			domain.Spec.Devices.Disks[i].DeepCopyInto(&original[i])
		}

		Expect(recoverOverlayDisks(domain, dir)).ToNot(Succeed())
		Expect(domain.Spec.Devices.Disks).To(Equal(original))
	},
		Entry("an overlay of another disk", func() { backing["root"] = "/base/data.img" }),
		Entry("an overlay that is not qcow2", func() { format = "raw" }),
		Entry("a base image that was replaced", func() { domain.Spec.Devices.Disks[1].Source.File = "/base/replacement.img" }),
		Entry("a disk without a driver", func() { domain.Spec.Devices.Disks[1].Driver = nil }),
		Entry("an overlay of a volume the VMI no longer has", func() {
			Expect(os.WriteFile(overlayPath(dir, "hotplugged"), []byte("qcow2"), 0o600)).To(Succeed())
		}),
	)

	Context("the domain", func() {
		var (
			cache   *metadata.Cache
			manager *StorageManager
		)

		BeforeEach(func() {
			cache = metadata.NewCache()
			manager = NewStorageManager(nil, cache)

			reference, err := json.Marshal(storagetypes.SnapshotOverlayReference{ContentUID: "content", ScratchPVCUID: "scratch"})
			Expect(err).ToNot(HaveOccurred())
			vmi.Annotations = map[string]string{
				storagetypes.PendingOverlayRecoveryAnnotation:   "snap-scratch-content",
				storagetypes.OverlaySnapshotReferenceAnnotation: string(reference),
			}
			vmi.Spec.UtilityVolumes = []v1.UtilityVolume{{
				Name:                              "snap-scratch-content",
				PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{ClaimName: "snap-scratch-content"},
				Type:                              pointer.P(v1.SnapshotOverlay),
			}}

			mountDir, mounted := overlayVolumeMountDir, verifyOverlayDirMounted
			overlayVolumeMountDir = func(string) string { return dir }
			verifyOverlayDirMounted = func(string) error { return nil }
			DeferCleanup(func() { overlayVolumeMountDir, verifyOverlayDirMounted = mountDir, mounted })
		})

		It("should report the transaction as committing before the domain starts", func() {
			Expect(manager.RecoverOverlays(vmi, domain)).To(Succeed())

			overlay, _ := cache.SnapshotOverlay.Load()
			Expect(overlay.Phase).To(Equal(api.SnapshotOverlayCommitting))
			Expect(overlay.ContentUID).To(BeEquivalentTo("content"))
			Expect(overlay.ScratchPVCUID).To(BeEquivalentTo("scratch"))
			expectOnOverlay("root", api.DiskSource{File: "/base/root.img"}, "file")
		})

		// An empty scratch still needs a matching completion to be released
		It("should report committing when every overlay was already committed", func() {
			Expect(os.Remove(overlayPath(dir, "root"))).To(Succeed())
			Expect(os.Remove(overlayPath(dir, "data"))).To(Succeed())

			Expect(manager.RecoverOverlays(vmi, domain)).To(Succeed())

			overlay, _ := cache.SnapshotOverlay.Load()
			Expect(overlay.Phase).To(Equal(api.SnapshotOverlayCommitting))
			Expect(diskOf("root").BackingStore).To(BeNil())
		})

		It("should refuse a scratch volume of another transaction", func() {
			vmi.Annotations[storagetypes.PendingOverlayRecoveryAnnotation] = "snap-scratch-other"

			Expect(manager.RecoverOverlays(vmi, domain)).ToNot(Succeed())
			overlay, _ := cache.SnapshotOverlay.Load()
			Expect(overlay.Phase).To(BeEmpty())
		})

		It("should leave a VMI that recovers nothing alone", func() {
			delete(vmi.Annotations, storagetypes.PendingOverlayRecoveryAnnotation)

			Expect(manager.RecoverOverlays(vmi, domain)).To(Succeed())
			overlay, _ := cache.SnapshotOverlay.Load()
			Expect(overlay.Phase).To(BeEmpty())
			Expect(diskOf("root").BackingStore).To(BeNil())
		})
	})
})
