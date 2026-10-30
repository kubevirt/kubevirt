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

package hooks_test

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/hooks"
)

const helloSHA256 = "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"

var _ = Describe("Checksum", func() {
	DescribeTable("validating checksums",
		func(checksum hooks.Checksum, expectedError string) {
			err := hooks.ValidateChecksum(checksum)
			if expectedError == "" {
				Expect(err).ToNot(HaveOccurred())
				return
			}

			Expect(err).To(MatchError(ContainSubstring(expectedError)))
		},
		Entry("accepts a SHA-256 checksum", hooks.Checksum{Algorithm: hooks.SHA256ChecksumAlgorithm, Value: helloSHA256}, ""),
		Entry("rejects an unsupported algorithm", hooks.Checksum{Algorithm: "sha512", Value: helloSHA256}, "unsupported checksum algorithm"),
		Entry("rejects invalid hexadecimal data",
			hooks.Checksum{Algorithm: hooks.SHA256ChecksumAlgorithm, Value: "not-hex"}, "decode sha256 checksum"),
		Entry("rejects a checksum with the wrong length",
			hooks.Checksum{Algorithm: hooks.SHA256ChecksumAlgorithm, Value: "abcd"}, "invalid sha256 checksum length"),
	)

	It("accepts content matching the expected checksum", func() {
		err := hooks.VerifyChecksum([]byte("hello"), hooks.Checksum{
			Algorithm: hooks.SHA256ChecksumAlgorithm,
			Value:     helloSHA256,
		})

		Expect(err).ToNot(HaveOccurred())
	})

	It("rejects content that does not match the expected checksum", func() {
		err := hooks.VerifyChecksum([]byte("modified"), hooks.Checksum{
			Algorithm: hooks.SHA256ChecksumAlgorithm,
			Value:     helloSHA256,
		})

		Expect(err).To(MatchError("checksum mismatch"))
	})

	It("accepts an uppercase hexadecimal checksum", func() {
		err := hooks.VerifyChecksum([]byte("hello"), hooks.Checksum{
			Algorithm: hooks.SHA256ChecksumAlgorithm,
			Value:     strings.ToUpper(helloSHA256),
		})

		Expect(err).ToNot(HaveOccurred())
	})

	It("reports a malformed checksum instead of a mismatch", func() {
		err := hooks.VerifyChecksum([]byte("hello"), hooks.Checksum{
			Algorithm: hooks.SHA256ChecksumAlgorithm,
			Value:     "not-hex",
		})

		Expect(err).To(MatchError(ContainSubstring("decode sha256 checksum")))
	})

	DescribeTable("validating ConfigMap checksum configuration",
		func(configMap hooks.ConfigMap, expectedError string) {
			err := hooks.ValidateConfigMapChecksum(configMap)
			if expectedError == "" {
				Expect(err).ToNot(HaveOccurred())
				return
			}

			Expect(err).To(MatchError(ContainSubstring(expectedError)))
		},
		Entry("allows a ConfigMap without a checksum", hooks.ConfigMap{}, ""),
		Entry("accepts a complete checksum configuration", hooks.ConfigMap{
			Name:     "hooks",
			Key:      "onDefineDomain.sh",
			HookPath: hooks.OnDefineDomainHookPath,
			Checksum: &hooks.Checksum{Algorithm: hooks.SHA256ChecksumAlgorithm, Value: helloSHA256},
		}, ""),
		Entry("accepts the preCloudInitIso hook path", hooks.ConfigMap{
			Name:     "hooks",
			Key:      "preCloudInitIso.sh",
			HookPath: hooks.PreCloudInitIsoHookPath,
			Checksum: &hooks.Checksum{Algorithm: hooks.SHA256ChecksumAlgorithm, Value: helloSHA256},
		}, ""),
		Entry("rejects an unsupported checksum algorithm", hooks.ConfigMap{
			Name:     "hooks",
			Key:      "onDefineDomain.sh",
			HookPath: hooks.OnDefineDomainHookPath,
			Checksum: &hooks.Checksum{Algorithm: "sha1", Value: helloSHA256},
		}, "unsupported checksum algorithm"),
		Entry("requires a ConfigMap name", hooks.ConfigMap{
			Key:      "onDefineDomain.sh",
			HookPath: hooks.OnDefineDomainHookPath,
			Checksum: &hooks.Checksum{Algorithm: hooks.SHA256ChecksumAlgorithm, Value: helloSHA256},
		}, "config map name is required"),
		Entry("requires a ConfigMap key", hooks.ConfigMap{
			Name:     "hooks",
			HookPath: hooks.OnDefineDomainHookPath,
			Checksum: &hooks.Checksum{Algorithm: hooks.SHA256ChecksumAlgorithm, Value: helloSHA256},
		}, "config map key is required"),
		Entry("rejects an unsupported hook path", hooks.ConfigMap{
			Name:     "hooks",
			Key:      "hook.sh",
			HookPath: "/usr/bin/otherHook",
			Checksum: &hooks.Checksum{Algorithm: hooks.SHA256ChecksumAlgorithm, Value: helloSHA256},
		}, "unsupported hook path"),
	)
})
