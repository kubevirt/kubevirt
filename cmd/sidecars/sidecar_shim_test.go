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

package main

import (
	"errors"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/hooks"
)

const helloSHA256 = "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"

var _ = Describe("Sidecar shim checksum verification", func() {
	setChecksumEnvironment := func(algorithm, value string) {
		Expect(os.Setenv(hooks.HookChecksumAlgorithmEnvVar, algorithm)).To(Succeed())
		Expect(os.Setenv(hooks.HookChecksumValueEnvVar, value)).To(Succeed())
		DeferCleanup(os.Unsetenv, hooks.HookChecksumAlgorithmEnvVar)
		DeferCleanup(os.Unsetenv, hooks.HookChecksumValueEnvVar)
	}

	It("allows hooks without checksum configuration", func() {
		Expect(verifyExecutableChecksum("/does/not/exist", os.ReadFile)).To(Succeed())
	})

	It("accepts a hook matching the configured checksum", func() {
		hookPath := hooks.OnDefineDomainHookPath
		setChecksumEnvironment(hooks.SHA256ChecksumAlgorithm, helloSHA256)
		readFile := func(path string) ([]byte, error) {
			Expect(path).To(Equal(hookPath))
			return []byte("hello"), nil
		}

		Expect(verifyExecutableChecksum(hookPath, readFile)).To(Succeed())
	})

	It("rejects a hook that does not match the configured checksum", func() {
		hookPath := hooks.OnDefineDomainHookPath
		setChecksumEnvironment(hooks.SHA256ChecksumAlgorithm, helloSHA256)
		readFile := func(path string) ([]byte, error) {
			Expect(path).To(Equal(hookPath))
			return []byte("modified"), nil
		}

		Expect(verifyExecutableChecksum(hookPath, readFile)).To(
			MatchError(ContainSubstring("checksum mismatch")),
		)
	})

	It("rejects incomplete checksum configuration", func() {
		Expect(os.Setenv(hooks.HookChecksumAlgorithmEnvVar, hooks.SHA256ChecksumAlgorithm)).To(Succeed())
		DeferCleanup(os.Unsetenv, hooks.HookChecksumAlgorithmEnvVar)

		Expect(verifyExecutableChecksum(hooks.OnDefineDomainHookPath, os.ReadFile)).To(
			MatchError("incomplete hook checksum configuration"),
		)
	})

	It("rejects incomplete checksum configuration when only the value is set", func() {
		Expect(os.Setenv(hooks.HookChecksumValueEnvVar, helloSHA256)).To(Succeed())
		DeferCleanup(os.Unsetenv, hooks.HookChecksumValueEnvVar)

		Expect(verifyExecutableChecksum(hooks.OnDefineDomainHookPath, os.ReadFile)).To(
			MatchError("incomplete hook checksum configuration"),
		)
	})

	It("fails when the hook cannot be read", func() {
		setChecksumEnvironment(hooks.SHA256ChecksumAlgorithm, helloSHA256)
		readFile := func(string) ([]byte, error) {
			return nil, errors.New("permission denied")
		}

		Expect(verifyExecutableChecksum(hooks.OnDefineDomainHookPath, readFile)).To(
			MatchError(ContainSubstring("permission denied")),
		)
	})
})
