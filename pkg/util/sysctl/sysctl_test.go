/*
Copyright 2015 The Kubernetes Authors.
Copyright 2020 The KubeVirt Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

Originally copied from https://github.com/kubernetes/kubernetes/blob/d8695d06b7191db56ebbbc0340da263833c9bb6f/pkg/util/sysctl/sysctl.go
*/

package sysctl

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestSysctl(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Sysctl Suite")
}

var _ = Describe("Sysctl", func() {
	var (
		tmpDir string
		svc    *procSysctl
	)

	BeforeEach(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "sysctl-test")
		Expect(err).ToNot(HaveOccurred())

		svc = &procSysctl{
			sysctlBase: tmpDir,
		}
	})

	AfterEach(func() {
		os.RemoveAll(tmpDir)
	})

	Context("New", func() {
		It("should create a new instance with default base", func() {
			impl := New()
			Expect(impl).ToNot(BeNil())
			
			// Verify internal state
			s, ok := impl.(*procSysctl)
			Expect(ok).To(BeTrue())
			Expect(s.sysctlBase).To(Equal(sysctlBase))
		})
	})

	Context("Zero Value", func() {
		It("should default to /proc/sys if uninitialized", func() {
			// This covers the "if p.sysctlBase == empty" path
			zeroImpl := &procSysctl{}
			Expect(zeroImpl.getBase()).To(Equal(sysctlBase))
		})
	})

	Context("GetSysctl", func() {
		It("should return error if file does not exist", func() {
			val, err := svc.GetSysctl("net/ipv4/does_not_exist")
			Expect(err).To(HaveOccurred())
			Expect(val).To(Equal("-1"))
		})

		It("should read value from file", func() {
			sysctlPath := filepath.Join(tmpDir, "some_setting")
			err := os.WriteFile(sysctlPath, []byte("1\n"), 0644)
			Expect(err).ToNot(HaveOccurred())

			val, err := svc.GetSysctl("some_setting")
			Expect(err).ToNot(HaveOccurred())
			Expect(val).To(Equal("1"))
		})
	})

	Context("SetSysctl", func() {
		It("should write value to file", func() {
			subdir := filepath.Join(tmpDir, "net", "ipv4")
			err := os.MkdirAll(subdir, 0755)
			Expect(err).ToNot(HaveOccurred())
			
			err = svc.SetSysctl("net/ipv4/test_setting", "1")
			Expect(err).ToNot(HaveOccurred())

			content, err := os.ReadFile(filepath.Join(tmpDir, "net/ipv4/test_setting"))
			Expect(err).ToNot(HaveOccurred())
			Expect(string(content)).To(Equal("1"))
		})
	})
})