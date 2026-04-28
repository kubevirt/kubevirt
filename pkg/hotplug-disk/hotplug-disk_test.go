/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package hotplugdisk

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"

	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	"kubevirt.io/kubevirt/pkg/unsafepath"
)

var _ = Describe("HotplugDisk", func() {
	var (
		tempDir     string
		err         error
		hotplug     *hotplugDiskManager
		podsBaseDir string
	)

	BeforeEach(func() {
		// Create some directories and files in temporary location.
		tempDir = GinkgoT().TempDir()
		podsBaseDir = filepath.Join(tempDir, "podsBaseDir")
		err = os.MkdirAll(filepath.Join(podsBaseDir), os.FileMode(0755))
		hotplug = &hotplugDiskManager{
			podsBaseDir: podsBaseDir,
		}

		Expect(err).ToNot(HaveOccurred())
	})

	It("GetHotplugTargetPodPathOnHost should return the correct path", func() {
		testUID := types.UID("abcd")
		_ = os.MkdirAll(TargetPodBasePath(podsBaseDir, testUID), 0755)
		_, err := hotplug.GetHotplugTargetPodPathOnHost(testUID)
		Expect(err).ToNot(HaveOccurred())
	})

	It("GetHotplugTargetPodPathOnHost should return error on incorrect path", func() {
		testUID := types.UID("abcde")
		_, err := hotplug.GetHotplugTargetPodPathOnHost(testUID)
		Expect(err).To(HaveOccurred())
	})

	It("GetFileSystemDirectoryTargetPathFromHostView should create the volume directory", func() {
		testUID := types.UID("abcd")
		_ = os.MkdirAll(TargetPodBasePath(podsBaseDir, testUID), 0755)
		res, err := hotplug.GetFileSystemDirectoryTargetPathFromHostView(testUID, "testvolume", true)
		Expect(err).ToNot(HaveOccurred())
		testPath := filepath.Join(TargetPodBasePath(podsBaseDir, testUID), "testvolume")
		exists, _ := diskutils.FileExists(testPath)
		Expect(exists).To(BeTrue())
		Expect(unsafepath.UnsafeAbsolute(res.Raw())).To(Equal(testPath))
	})

	It("GetFileSystemDirectoryTargetPathFromHostView should return the volume directory", func() {
		testUID := types.UID("abcd")
		_ = os.MkdirAll(TargetPodBasePath(podsBaseDir, testUID), 0755)
		testPath := filepath.Join(TargetPodBasePath(podsBaseDir, testUID), "testvolume")
		err = os.MkdirAll(testPath, os.FileMode(0755))
		res, err := hotplug.GetFileSystemDirectoryTargetPathFromHostView(testUID, "testvolume", false)
		Expect(err).ToNot(HaveOccurred())
		Expect(unsafepath.UnsafeAbsolute(res.Raw())).To(Equal(testPath))
	})

	It("GetFileSystemDirectoryTargetPathFromHostView should fail on invalid UID", func() {
		testUID := types.UID("abcde")
		_, err := hotplug.GetFileSystemDirectoryTargetPathFromHostView(testUID, "testvolume", false)
		Expect(err).To(HaveOccurred())
	})

	It("GetFileSystemDiskTargetPathFromHostView should create the disk image file", func() {
		testUID := types.UID("abcd")
		_ = os.MkdirAll(TargetPodBasePath(podsBaseDir, testUID), 0755)
		res, err := hotplug.GetFileSystemDiskTargetPathFromHostView(testUID, "testvolume", true)
		Expect(err).ToNot(HaveOccurred())
		targetPath := filepath.Join(TargetPodBasePath(podsBaseDir, testUID), "testvolume.img")
		exists, _ := diskutils.FileExists(targetPath)
		Expect(exists).To(BeTrue())
		Expect(unsafepath.UnsafeAbsolute(res.Raw())).To(Equal(targetPath))
	})

	It("GetFileSystemDiskTargetPathFromHostView should return the disk image file", func() {
		testUID := types.UID("abcd")
		_ = os.MkdirAll(TargetPodBasePath(podsBaseDir, testUID), 0755)
		targetPath := filepath.Join(TargetPodBasePath(podsBaseDir, testUID), "testvolume.img")
		err = os.MkdirAll(targetPath, os.FileMode(0755))
		res, err := hotplug.GetFileSystemDiskTargetPathFromHostView(testUID, "testvolume", false)
		Expect(err).ToNot(HaveOccurred())
		Expect(unsafepath.UnsafeAbsolute(res.Raw())).To(Equal(targetPath))
	})

	It("GetFileSystemDiskTargetPathFromHostView should fail on invalid UID", func() {
		testUID := types.UID("abcde")
		_, err := hotplug.GetFileSystemDiskTargetPathFromHostView(testUID, "testvolume", false)
		Expect(err).To(HaveOccurred())
	})
})
