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

var (
	orgTargetPodBasePath = targetPodBasePath
	orgPodsBaseDir       = podsBaseDir
)

var _ = Describe("HotplugDisk", func() {
	var (
		tempDir string
		err     error
	)

	BeforeEach(func() {
		// Create some directories and files in temporary location.
		tempDir, err = ioutil.TempDir("/tmp", "hp-disk-test")
		Expect(err).ToNot(HaveOccurred())
		err = os.MkdirAll(filepath.Join(tempDir, "abcd"), os.FileMode(0755))
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		if tempDir != "" {
			os.RemoveAll(tempDir)
		}
		targetPodBasePath = orgTargetPodBasePath
		SetKubeletPodsDirectory(orgPodsBaseDir)
	})

	It("GetHotplugTargetPodPathOnHost should return the correct path", func() {
		testUID := types.UID("abcd")
		SetKubeletPodsDirectory(tempDir)
		targetPodBasePath = func(podUID types.UID) string {
			Expect(podUID).To(Equal(testUID))
			return string(testUID)
		}
		_, err := GetHotplugTargetPodPathOnHost(testUID)
		Expect(err).ToNot(HaveOccurred())
	})

	It("GetHotplugTargetPodPathOnHost should return error on incorrect path", func() {
		testUID := types.UID("abcde")
		SetKubeletPodsDirectory(tempDir)
		targetPodBasePath = func(podUID types.UID) string {
			Expect(podUID).To(Equal(testUID))
			return string(testUID)
		}
		_, err := GetHotplugTargetPodPathOnHost(testUID)
		Expect(err).To(HaveOccurred())
	})

	It("GetFileSystemDiskTargetPathFromHostView should create the volume directory", func() {
		testUID := types.UID("abcd")
		SetKubeletPodsDirectory(tempDir)
		targetPodBasePath = func(podUID types.UID) string {
			Expect(podUID).To(Equal(testUID))
			return string(testUID)
		}
		res, err := GetFileSystemDiskTargetPathFromHostView(testUID, "testvolume")
		Expect(err).ToNot(HaveOccurred())
		testPath := filepath.Join(tempDir, string(testUID), "testvolume")
		exists, _ := diskutils.FileExists(testPath)
		Expect(exists).To(BeTrue())
		Expect(res).To(Equal(testPath))
	})

	It("GetFileSystemDiskTargetPathFromHostView should return the volume directory", func() {
		testUID := types.UID("abcd")
		SetKubeletPodsDirectory(tempDir)
		targetPodBasePath = func(podUID types.UID) string {
			Expect(podUID).To(Equal(testUID))
			return string(testUID)
		}
		testPath := filepath.Join(tempDir, string(testUID), "testvolume")
		err = os.MkdirAll(testPath, os.FileMode(0755))
		res, err := GetFileSystemDiskTargetPathFromHostView(testUID, "testvolume")
		Expect(err).ToNot(HaveOccurred())
		Expect(res).To(Equal(testPath))
	})

	It("GetFileSystemDiskTargetPathFromHostView should fail on invalid UID", func() {
		testUID := types.UID("abcde")
		SetKubeletPodsDirectory(tempDir)
		targetPodBasePath = func(podUID types.UID) string {
			Expect(podUID).To(Equal(testUID))
			return string(testUID)
		}
		_, err := GetFileSystemDiskTargetPathFromHostView(testUID, "testvolume")
		Expect(err).To(HaveOccurred())
	})
})
