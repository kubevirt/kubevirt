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
 */

package disk

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Validation", func() {

	var diskInfo DiskInfo
	var sizeStub int64

	BeforeEach(func() {
		diskInfo = DiskInfo{}
		sizeStub = 12345
	})

	Context("verify qcow2", func() {

		It("should return error if format is not qcow2", func() {
			diskInfo.Format = "not qcow2"
			err := VerifyQCOW2(&diskInfo)
			Expect(err).Should(HaveOccurred())
		})

		It("should return error if backing file exists", func() {
			diskInfo.Format = "qcow2"
			diskInfo.BackingFile = "my-super-awesome-file"
			err := VerifyQCOW2(&diskInfo)
			Expect(err).Should(HaveOccurred())
		})

		It("should run successfully", func() {
			diskInfo.Format = "qcow2"
			diskInfo.ActualSize = sizeStub
			diskInfo.VirtualSize = sizeStub
			err := VerifyQCOW2(&diskInfo)
			Expect(err).ShouldNot(HaveOccurred())
		})

	})

	Context("verify image", func() {

		It("should be successful if image is raw", func() {
			diskInfo.Format = "raw"
			err := VerifyImage(&diskInfo)
			Expect(err).ShouldNot(HaveOccurred())
		})

		It("should succeed on qcow2 valid disk info", func() {
			diskInfo.Format = "qcow2"
			diskInfo.ActualSize = sizeStub
			diskInfo.VirtualSize = sizeStub
			err := VerifyImage(&diskInfo)
			Expect(err).ShouldNot(HaveOccurred())
		})

		It("should fail on unknown format", func() {
			diskInfo.Format = "unknown format"
			err := VerifyImage(&diskInfo)
			Expect(err).Should(HaveOccurred())
		})

	})

})
