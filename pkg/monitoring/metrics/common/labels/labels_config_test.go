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

package labels

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
)

var _ = Describe("labels config", func() {
	var cfg *configImpl

	BeforeEach(func() {
		_, err := New(nil)
		Expect(err).To(HaveOccurred())
		// since New(nil) returns (nil, err), construct directly for tests
		cfg = &configImpl{
			allowlist:  append([]string{}, defaultAllowlist...),
			ignorelist: append([]string{}, defaultIgnorelist...),
			allowAll:   true,
		}
	})

	Context("default configuration", func() {
		It("should report any label when allowlist is '*'", func() {
			Expect(cfg.ShouldReport("environment")).To(BeTrue())
			Expect(cfg.ShouldReport("team")).To(BeTrue())
		})
	})

	Context("ConfigMap updates", func() {
		It("should prioritize ignorelist over allowlist when overlapping", func() {
			cm := &k8sv1.ConfigMap{Data: map[string]string{
				"allowlist":  "*",
				"ignorelist": "vm.kubevirt.io/template",
			}}
			cfg.updateFromConfigMap(cm)

			Expect(cfg.ShouldReport("vm.kubevirt.io/template")).To(BeFalse())
			Expect(cfg.ShouldReport("team")).To(BeTrue())
		})

		It("should not report when allowlist is empty", func() {
			cm := &k8sv1.ConfigMap{Data: map[string]string{
				"allowlist":  "",
				"ignorelist": "",
			}}
			cfg.updateFromConfigMap(cm)
			Expect(cfg.ShouldReport("a")).To(BeFalse())
		})

		It("should not treat '*' in ignorelist as wildcard", func() {
			cm := &k8sv1.ConfigMap{Data: map[string]string{
				"allowlist":  "*",
				"ignorelist": "*",
			}}
			cfg.updateFromConfigMap(cm)
			Expect(cfg.ShouldReport("environment")).To(BeTrue())
			Expect(cfg.ShouldReport("team")).To(BeTrue())
		})
	})
})
