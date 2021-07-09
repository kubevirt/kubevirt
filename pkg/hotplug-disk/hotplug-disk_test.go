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
 * Copyright 2020 Red Hat, Inc.
 *
 */

package hotplugdisk

import (
	"io/ioutil"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"

	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
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
		tempDir, err = ioutil.TempDir("/tmp", "hp-disk-test")
		Expect(err).ToNot(HaveOccurred())
		podsBaseDir = filepath.Join(tempDir, "podsBaseDir")
		err = os.MkdirAll(filepath.Join(podsBaseDir), os.FileMode(0755))
		hotplug = &hotplugDiskManager{
			podsBaseDir:       podsBaseDir,
			targetPodBasePath: nil,
		}

		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		if tempDir != "" {
			os.RemoveAll(tempDir)
		}
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

	It("GetFileSystemDiskTargetPathFromHostView should create the volume directory", func() {
		testUID := types.UID("abcd")
		_ = os.MkdirAll(TargetPodBasePath(podsBaseDir, testUID), 0755)
		res, err := hotplug.GetFileSystemDiskTargetPathFromHostView(testUID, "testvolume", true)
		Expect(err).ToNot(HaveOccurred())
		testPath := filepath.Join(TargetPodBasePath(podsBaseDir, testUID), "testvolume")
		exists, _ := diskutils.FileExists(testPath)
		Expect(exists).To(BeTrue())
		Expect(res).To(Equal(testPath))
	})

	It("GetFileSystemDiskTargetPathFromHostView should return the volume directory", func() {
		testUID := types.UID("abcd")
		_ = os.MkdirAll(TargetPodBasePath(podsBaseDir, testUID), 0755)
		testPath := filepath.Join(TargetPodBasePath(podsBaseDir, testUID), "testvolume")
		err = os.MkdirAll(testPath, os.FileMode(0755))
		res, err := hotplug.GetFileSystemDiskTargetPathFromHostView(testUID, "testvolume", false)
		Expect(err).ToNot(HaveOccurred())
		Expect(res).To(Equal(testPath))
	})

	It("GetFileSystemDiskTargetPathFromHostView should fail on invalid UID", func() {
		testUID := types.UID("abcde")
		_, err := hotplug.GetFileSystemDiskTargetPathFromHostView(testUID, "testvolume", false)
		Expect(err).To(HaveOccurred())
	})
})
