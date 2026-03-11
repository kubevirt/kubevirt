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

package vfio

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	testPCIAddress   = "0000:00:01.0"
	testMDevUUID     = "00001111-2222-3333-4444-555566667777"
	testVFIOCDevName = "vfio0"
)

var (
	originalPCIDeviceBasePath  = pciDeviceBasePath
	originalMDevDeviceBasePath = mdevDeviceBasePath
	originalIOMMUFDPath        = iommufdPath
	originalVFIOCDevBasePath   = vfioCDevBasePath
)

var _ = Describe("VFIO specification", func() {
	var (
		vfioSpec VFIOSpec

		fakeRootPath string

		fakePCIDeviceBasePath  string
		fakeMDevDeviceBasePath string
		fakeVFIOCDevBasePath   string

		testPCIDevicePath  string
		testMDevDevicePath string
	)

	createTempRootStructure := func() {
		var err error
		fakeRootPath, err = os.MkdirTemp("", "fake_root")
		Expect(err).ToNot(HaveOccurred())

		fakePCIDeviceBasePath = filepath.Join(fakeRootPath, pciDeviceBasePath)
		testPCIDevicePath = filepath.Join(fakePCIDeviceBasePath, testPCIAddress)
		err = os.MkdirAll(testPCIDevicePath, 0o755)
		Expect(err).ToNot(HaveOccurred())

		fakeMDevDeviceBasePath = filepath.Join(fakeRootPath, mdevDeviceBasePath)
		testMDevDevicePath = filepath.Join(fakeMDevDeviceBasePath, testMDevUUID)
		err = os.MkdirAll(testMDevDevicePath, 0o755)
		Expect(err).ToNot(HaveOccurred())

		fakeVFIOCDevBasePath = filepath.Join(fakeRootPath, vfioCDevBasePath)
		err = os.MkdirAll(fakeVFIOCDevBasePath, 0o755)
		Expect(err).ToNot(HaveOccurred())
	}

	setIOMMUFDPath := func(exist bool) {
		if exist {
			// use /dev/null for avoiding file creation
			iommufdPath = "/dev/null"
		} else {
			iommufdPath = filepath.Join(fakeRootPath, "/dev/obviously-not-exists")
		}
	}

	setupSysfsVFIOCDevFiles := func(createFiles bool) {
		if createFiles {
			for _, dirPath := range []string{testPCIDevicePath, testMDevDevicePath} {
				path := filepath.Join(dirPath, "vfio-dev", testVFIOCDevName)
				err := os.MkdirAll(path, 0o755)
				Expect(err).ToNot(HaveOccurred())
			}
		}

		pciDeviceBasePath = fakePCIDeviceBasePath
		mdevDeviceBasePath = fakeMDevDeviceBasePath
	}

	cleanupSysfsVFIOCDevFiles := func() {
		for _, dirPath := range []string{testPCIDevicePath, testMDevDevicePath} {
			path := filepath.Join(dirPath, "vfio-dev")
			os.RemoveAll(path)
		}
	}

	setupDevfsVFIOCDevFiles := func(createFiles bool) {
		if createFiles {
			path := filepath.Join(fakeVFIOCDevBasePath, testVFIOCDevName)
			err := os.WriteFile(path, []byte("\n"), 0o644)
			Expect(err).ToNot(HaveOccurred())
		}

		vfioCDevBasePath = fakeVFIOCDevBasePath
	}

	cleanupDevfsVFIOCDevFiles := func() {
		path := filepath.Join(fakeVFIOCDevBasePath, testVFIOCDevName)
		os.RemoveAll(path)
	}

	resetAllPaths := func() {
		pciDeviceBasePath = originalPCIDeviceBasePath
		mdevDeviceBasePath = originalMDevDeviceBasePath
		iommufdPath = originalIOMMUFDPath
		vfioCDevBasePath = originalVFIOCDevBasePath
	}

	BeforeEach(func() {
		createTempRootStructure()
	})

	AfterEach(func() {
		if fakeRootPath != "" {
			os.RemoveAll(fakeRootPath)
		}
	})

	Context("determines whether device can be assigned via IOMMUFD", func() {
		DescribeTableSubtree("when domain xml is capable of IOMMUFD setting", func(withIOMMUFD bool, withVFIOCDev bool, vfioCDevExistsOnNode bool) {
			shouldAssignDeviceViaIOMMUFD := func(withIOMMUFD bool, withVFIOCDev bool, vfioCDevExistsOnNode bool) bool {
				switch {
				// device plugins/dra drivers allocate both iommufd and vfio cdev
				case withIOMMUFD && withVFIOCDev && vfioCDevExistsOnNode:
					return true
				// as designed, fall back to the legacy usage if not get both
				case withIOMMUFD, withVFIOCDev && vfioCDevExistsOnNode:
					return false
				// cannot happen
				case withVFIOCDev && !vfioCDevExistsOnNode:
					return false
				// the legacy usage
				default:
					return false
				}
			}

			BeforeEach(func() {
				setIOMMUFDPath(withIOMMUFD)
				setupSysfsVFIOCDevFiles(vfioCDevExistsOnNode)
				setupDevfsVFIOCDevFiles(withVFIOCDev)

				vfioSpec = NewVFIOSpec(true)
			})

			AfterEach(func() {
				cleanupSysfsVFIOCDevFiles()
				cleanupDevfsVFIOCDevFiles()
				resetAllPaths()
			})

			It("should return the correct result for PCI device", func() {
				result := vfioSpec.IsPCIAssignableViaIOMMUFD(testPCIAddress)
				if shouldAssignDeviceViaIOMMUFD(withIOMMUFD, withVFIOCDev, vfioCDevExistsOnNode) {
					Expect(result).To(BeTrue())
				} else {
					Expect(result).To(BeFalse())
				}
			})

			It("should return the correct result for mediated device", func() {
				result := vfioSpec.IsMDevAssignableViaIOMMUFD(testMDevUUID)
				if shouldAssignDeviceViaIOMMUFD(withIOMMUFD, withVFIOCDev, vfioCDevExistsOnNode) {
					Expect(result).To(BeTrue())
				} else {
					Expect(result).To(BeFalse())
				}
			})
		},
			Entry("w/ IOMMUFD and VFIO cdev been allocated", true, true, true),
			Entry("w/ only IOMMUFD been allocated, while VFIO cdev exists on node", true, false, true),
			Entry("w/ only IOMMUFD been allocated, while VFIO cdev does not exist on node", true, false, false),
			Entry("w/ only VFIO cdev been allocated", false, true, true),
			Entry("w/o IOMMUFD and VFIO cdev been allocated, while VFIO cdev exists on node", false, false, true),
			Entry("w/o IOMMUFD and VFIO cdev been allocated, while VFIO cdev does not exist on node", false, false, false),
		)

		DescribeTableSubtree("when domain xml is not capable of iommufd setting", func(withIOMMUFD bool, withVFIOCDev bool, vfioCDevExistsOnNode bool) {
			BeforeEach(func() {
				setIOMMUFDPath(withIOMMUFD)
				setupSysfsVFIOCDevFiles(vfioCDevExistsOnNode)
				setupDevfsVFIOCDevFiles(withVFIOCDev)

				vfioSpec = NewVFIOSpec(false)
			})

			AfterEach(func() {
				cleanupSysfsVFIOCDevFiles()
				cleanupDevfsVFIOCDevFiles()
				resetAllPaths()
			})

			It("should return false for PCI device", func() {
				result := vfioSpec.IsPCIAssignableViaIOMMUFD(testPCIAddress)
				Expect(result).To(BeFalse())
			})

			It("should return false for mediated device", func() {
				result := vfioSpec.IsMDevAssignableViaIOMMUFD(testMDevUUID)
				Expect(result).To(BeFalse())
			})
		},
			Entry("w/ IOMMUFD and VFIO cdev been allocated", true, true, true),
			Entry("w/ only IOMMUFD been allocated, while VFIO cdev exists on node", true, false, true),
			Entry("w/ only IOMMUFD been allocated, while VFIO cdev does not exist on node", true, false, false),
			Entry("w/ only VFIO cdev been allocated", false, true, true),
			Entry("w/o IOMMUFD and VFIO cdev been allocated, while VFIO cdev exists on node", false, false, true),
			Entry("w/o IOMMUFD and VFIO cdev been allocated, while VFIO cdev does not exist on node", false, false, false),
		)
	})

	// to ensure no false positive
	Context("when device does not exist", func() {
		BeforeEach(func() {
			setIOMMUFDPath(true)
			setupSysfsVFIOCDevFiles(true)
			setupDevfsVFIOCDevFiles(true)

			vfioSpec = NewVFIOSpec(true)
		})

		AfterEach(func() {
			cleanupSysfsVFIOCDevFiles()
			cleanupDevfsVFIOCDevFiles()
			resetAllPaths()
		})

		It("should return false for PCI device", func() {
			result := vfioSpec.IsPCIAssignableViaIOMMUFD("unavailable_pci")
			Expect(result).To(BeFalse())
		})

		It("should return false for mediated device", func() {
			result := vfioSpec.IsMDevAssignableViaIOMMUFD("unavailable_mdev")
			Expect(result).To(BeFalse())
		})
	})
})
