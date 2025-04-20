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

package hotplug_volume

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	invalidFindmntByVolume = "{\"filesystems\": [{:\"/t\", \"source\":\"/dev/testvolume\", \"fstype\":\"xfs\", \"options\":\"rw,relatime,seclabel,attr2,inode64,logbufs=8,logbsize=32k,noquota\"}]}"
)

var _ = Describe("findmnt", func() {
	callFindMntByVolume := func() ([]FindmntInfo, error) {
		findMntByVolume = func(volumeName string, pid int) ([]byte, error) {
			return []byte(fmt.Sprintf(findmntByVolumeRes, "testvolume", "/test/path")), nil
		}
		return LookupFindmntInfoByVolume("test", 1234)
	}

	callFindMntByDevice := func() ([]FindmntInfo, error) {
		findMntByDevice = func(volumeName string) ([]byte, error) {
			return []byte(fmt.Sprintf(findmntByVolumeRes, "testvolume", "/test/path")), nil
		}
		return LookupFindmntInfoByDevice("test")
	}

	callFindMntByVolumeBrokenFindmnt := func() ([]FindmntInfo, error) {
		findMntByVolume = func(volumeName string, pid int) ([]byte, error) {
			return []byte(""), fmt.Errorf("findmnt is busted")
		}
		return LookupFindmntInfoByVolume("test", 1234)
	}

	callFindMntByDeviceBrokenFindmnt := func() ([]FindmntInfo, error) {
		findMntByDevice = func(volumeName string) ([]byte, error) {
			return []byte(""), fmt.Errorf("findmnt is busted")
		}
		return LookupFindmntInfoByDevice("test")
	}

	callFindMntByVolumeInvalidJson := func() ([]FindmntInfo, error) {
		findMntByVolume = func(volumeName string, pid int) ([]byte, error) {
			return []byte(invalidFindmntByVolume), nil
		}
		return LookupFindmntInfoByVolume("test", 1234)
	}

	callFindMntByDeviceInvalidJson := func() ([]FindmntInfo, error) {
		findMntByDevice = func(volumeName string) ([]byte, error) {
			return []byte(invalidFindmntByVolume), nil
		}
		return LookupFindmntInfoByDevice("test")
	}

	AfterEach(func() {
		findMntByVolume = orgFindMntByVolume
		findMntByDevice = orgFindMntByDevice
	})

	DescribeTable("Should return a list of values, with valid input", func(findMntFunc func() ([]FindmntInfo, error)) {
		res, err := findMntFunc()
		Expect(err).ToNot(HaveOccurred())
		Expect(res).To(HaveLen(1))
		Expect(res[0].GetSourcePath()).To(Equal("/test/path"))
		Expect(res[0].Target).To(Equal("/testvolume"))
		Expect(res[0].Fstype).To(Equal("xfs"))
		Expect(res[0].GetOptions()).To(HaveLen(8))
		Expect(res[0].GetOptions()[0]).To(Equal("rw"))
		Expect(res[0].GetOptions()[1]).To(Equal("relatime"))
		Expect(res[0].GetOptions()[2]).To(Equal("seclabel"))
		Expect(res[0].GetOptions()[3]).To(Equal("attr2"))
		Expect(res[0].GetOptions()[4]).To(Equal("inode64"))
		Expect(res[0].GetOptions()[5]).To(Equal("logbufs=8"))
		Expect(res[0].GetOptions()[6]).To(Equal("logbsize=32k"))
		Expect(res[0].GetOptions()[7]).To(Equal("noquota"))
	},
		Entry("for findmntbyvolume", callFindMntByVolume),
		Entry("for findmntbydevice", callFindMntByDevice),
	)

	DescribeTable("Should return an error if findmnt fails", func(findMntFunc func() ([]FindmntInfo, error)) {
		_, err := findMntFunc()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("findmnt is busted"))
		Expect(err.Error()).To(ContainSubstring("test"))
	},
		Entry("for findmntbyvolume", callFindMntByVolumeBrokenFindmnt),
		Entry("for findmntbydevice", callFindMntByDeviceBrokenFindmnt),
	)

	DescribeTable("Should return an error if unmarshalling fails", func(findMntFunc func() ([]FindmntInfo, error)) {
		_, err := findMntFunc()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("unable to unmarshal"))
	},
		Entry("for findmntbyvolume", callFindMntByVolumeInvalidJson),
		Entry("for findmntbydevice", callFindMntByDeviceInvalidJson),
	)

	It("GetSourcePath should properly match source field", func() {
		test := FindmntInfo{
			Source: "/dev/test",
		}
		Expect(test.GetSourcePath()).To(Equal("/dev/test"))
		test2 := FindmntInfo{
			Source: "/dev/test[/mnt/something/else/]",
		}
		Expect(test2.GetSourcePath()).To(Equal("/mnt/something/else/"))
		test3 := FindmntInfo{
			Source: "/dev/test[/mnt/something/else/[/more]]",
		}
		Expect(test3.GetSourcePath()).To(Equal("/mnt/something/else/[/more]"))
	})

	It("GetSourceDevice should return the device", func() {
		test := FindmntInfo{
			Source: "/dev/test",
		}
		Expect(test.GetSourceDevice()).To(Equal(""))
		test2 := FindmntInfo{
			Source: "/dev/test[/mnt/something/else/]",
		}
		Expect(test2.GetSourceDevice()).To(Equal("/dev/test"))
		test3 := FindmntInfo{
			Source: "/dev/test[/mnt/something/else/[/more]]",
		}
		Expect(test3.GetSourceDevice()).To(Equal("/dev/test"))
		test4 := FindmntInfo{
			Source: "/path/to/somewhere",
		}
		Expect(test4.GetSourceDevice()).To(Equal(""))
	})

	It("GetOptions should properly return a list", func() {
		test := FindmntInfo{
			Options: "aa,bb,cc,dd",
		}
		Expect(test.GetOptions()).To(HaveLen(4))
		Expect(test.GetOptions()[0]).To(Equal("aa"))
		Expect(test.GetOptions()[1]).To(Equal("bb"))
		Expect(test.GetOptions()[2]).To(Equal("cc"))
		Expect(test.GetOptions()[3]).To(Equal("dd"))
	})
})
