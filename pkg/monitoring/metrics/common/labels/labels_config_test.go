/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
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
