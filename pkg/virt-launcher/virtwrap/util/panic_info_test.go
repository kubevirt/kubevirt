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

package util

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("Panic Info", func() {
	Context("ParseGuestPanicLogLine", func() {
		It("should parse hyper-v panic log line with all arguments", func() {
			line := "2024-09-30 16:30:49.506+0000: panic hyper-v: arg1='0x5a', arg2='0x1', arg3='0x2', arg4='0x3', arg5='0x4'"

			result := ParseGuestPanicLogLine(line)

			Expect(result).NotTo(BeNil())
			Expect(result.Type).To(Equal("hyper-v"))
			Expect(result.Arg1).To(Equal(uint64(0x5a)))
			Expect(result.Arg2).To(Equal(uint64(0x1)))
			Expect(result.Arg3).To(Equal(uint64(0x2)))
			Expect(result.Arg4).To(Equal(uint64(0x3)))
			Expect(result.Arg5).To(Equal(uint64(0x4)))
		})

		It("should parse hyper-v panic log line with partial arguments", func() {
			line := "panic hyper-v: arg1='0xABC'"

			result := ParseGuestPanicLogLine(line)

			Expect(result).NotTo(BeNil())
			Expect(result.Type).To(Equal("hyper-v"))
			Expect(result.Arg1).To(Equal(uint64(0xABC)))
			Expect(result.Arg2).To(Equal(uint64(0)))
		})

		It("should parse s390 panic log line", func() {
			line := "panic s390:"

			result := ParseGuestPanicLogLine(line)

			Expect(result).NotTo(BeNil())
			Expect(result.Type).To(Equal("s390"))
		})

		It("should return nil for non-panic line", func() {
			line := "2024-09-30 16:30:49.506+0000: normal log message"

			result := ParseGuestPanicLogLine(line)

			Expect(result).To(BeNil())
		})

		It("should return nil for empty line", func() {
			result := ParseGuestPanicLogLine("")
			Expect(result).To(BeNil())
		})
	})

	Context("ParseGuestPanicEvent", func() {
		It("should parse Hyper-V panic event details", func() {
			jsonDetails := `{"action":"pause","info":{"type":"hyper-v","arg1":90,"arg2":1,"arg3":2,"arg4":3,"arg5":4}}`

			result := ParseGuestPanicEvent(jsonDetails)

			Expect(result).NotTo(BeNil())
			Expect(result.Type).To(Equal("hyper-v"))
			Expect(result.Arg1).To(Equal(uint64(90)))
			Expect(result.Arg2).To(Equal(uint64(1)))
			Expect(result.Arg3).To(Equal(uint64(2)))
			Expect(result.Arg4).To(Equal(uint64(3)))
			Expect(result.Arg5).To(Equal(uint64(4)))
		})

		It("should return empty struct for panic event without info", func() {
			jsonDetails := `{"action":"pause"}`

			result := ParseGuestPanicEvent(jsonDetails)

			Expect(result).NotTo(BeNil())
			Expect(result.Type).To(Equal(""))
		})

		It("should return nil for empty input", func() {
			result := ParseGuestPanicEvent("")
			Expect(result).To(BeNil())
		})

		It("should return nil for invalid JSON", func() {
			result := ParseGuestPanicEvent("not valid json")
			Expect(result).To(BeNil())
		})
	})

	Context("FormatGuestPanicInfo", func() {
		It("should format Hyper-V panic info with bugcheck code", func() {
			panicInfo := &api.GuestPanicInfo{
				Type: "hyper-v",
				Arg1: 0x5a,
				Arg2: 0x1,
				Arg3: 0x2,
				Arg4: 0x0,
				Arg5: 0x0,
			}

			result := FormatGuestPanicInfo(panicInfo)

			Expect(result).To(Equal("GuestPanicked: '0x5a', '0x1', '0x2', '0x0', '0x0'"))
		})

		It("should format non-Hyper-V panic with type", func() {
			panicInfo := &api.GuestPanicInfo{
				Type: "s390",
			}

			result := FormatGuestPanicInfo(panicInfo)

			Expect(result).To(Equal("GuestPanicked: type=s390"))
		})

		It("should return simple message for nil panic info", func() {
			result := FormatGuestPanicInfo(nil)
			Expect(result).To(Equal("GuestPanicked"))
		})

		It("should return simple message for empty panic info", func() {
			panicInfo := &api.GuestPanicInfo{}

			result := FormatGuestPanicInfo(panicInfo)

			Expect(result).To(Equal("GuestPanicked"))
		})
	})

	Context("ReadPanicInfoFromLog", func() {
		var tmpDir string

		BeforeEach(func() {
			var err error
			tmpDir, err = os.MkdirTemp("", "panic-log-test")
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			os.RemoveAll(tmpDir)
		})

		It("should read panic info from log file", func() {
			logFile := filepath.Join(tmpDir, "test.log")
			content := `2024-09-30 16:30:48.000+0000: normal log line
2024-09-30 16:30:49.506+0000: panic hyper-v: arg1='0x5a', arg2='0x1', arg3='0x2', arg4='0x0', arg5='0x0'
2024-09-30 16:30:50.000+0000: more log lines
`
			err := os.WriteFile(logFile, []byte(content), 0644)
			Expect(err).NotTo(HaveOccurred())

			result, err := ReadPanicInfoFromLog(logFile)

			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(BeNil())
			Expect(result.Type).To(Equal("hyper-v"))
			Expect(result.Arg1).To(Equal(uint64(0x5a)))
		})

		It("should return empty struct when no panic line found", func() {
			logFile := filepath.Join(tmpDir, "test.log")
			content := `2024-09-30 16:30:48.000+0000: normal log line
2024-09-30 16:30:50.000+0000: more log lines
`
			err := os.WriteFile(logFile, []byte(content), 0644)
			Expect(err).NotTo(HaveOccurred())

			result, err := ReadPanicInfoFromLog(logFile)

			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(BeNil())
			Expect(result.Type).To(Equal(""))
		})

		It("should return last panic info when multiple panics in log", func() {
			logFile := filepath.Join(tmpDir, "test.log")
			content := `panic hyper-v: arg1='0x1'
panic hyper-v: arg1='0x2'
panic hyper-v: arg1='0x3'
`
			err := os.WriteFile(logFile, []byte(content), 0644)
			Expect(err).NotTo(HaveOccurred())

			result, err := ReadPanicInfoFromLog(logFile)

			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(BeNil())
			Expect(result.Arg1).To(Equal(uint64(0x3)))
		})

		It("should return error for non-existent file", func() {
			_, err := ReadPanicInfoFromLog("/non/existent/file.log")

			Expect(err).To(HaveOccurred())
		})
	})
})
