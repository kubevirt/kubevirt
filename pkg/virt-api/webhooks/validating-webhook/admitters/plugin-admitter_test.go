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

package admitters

import (
	"context"
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/api/plugin"
	pluginv1alpha1 "kubevirt.io/api/plugin/v1alpha1"

	"kubevirt.io/kubevirt/pkg/testutils"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
)

var _ = Describe("Validating Plugin Admitter", func() {
	var config *virtconfig.ClusterConfig
	var kvStore cache.Store

	enablePluginFeatureGate := func() {
		testutils.UpdateFakeKubeVirtClusterConfig(kvStore, &v1.KubeVirt{
			Spec: v1.KubeVirtSpec{
				Configuration: v1.KubeVirtConfiguration{
					DeveloperConfiguration: &v1.DeveloperConfiguration{
						FeatureGates: []string{featuregate.PluginsGate},
					},
				},
			},
		})
	}

	disablePluginFeatureGate := func() {
		testutils.UpdateFakeKubeVirtClusterConfig(kvStore, &v1.KubeVirt{
			Spec: v1.KubeVirtSpec{
				Configuration: v1.KubeVirtConfiguration{
					DeveloperConfiguration: &v1.DeveloperConfiguration{
						DisabledFeatureGates: []string{featuregate.PluginsGate},
					},
				},
			},
		})
	}

	admit := func(p *pluginv1alpha1.Plugin) *admissionv1.AdmissionResponse {
		admitter := NewPluginAdmitter(config)
		return admitter.Admit(context.Background(), createPluginAdmissionReview(p))
	}

	BeforeEach(func() {
		config, _, kvStore = testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{})
		enablePluginFeatureGate()
	})

	It("should reject when Plugins feature gate is disabled", func() {
		disablePluginFeatureGate()
		resp := admit(newValidPluginWithDomainHook())
		Expect(resp.Allowed).To(BeFalse())
		Expect(resp.Result.Message).To(ContainSubstring("Plugins feature gate is not enabled"))
	})

	It("should accept a valid Plugin", func() {
		resp := admit(newValidPluginWithDomainHook())
		Expect(resp.Allowed).To(BeTrue())
	})

	It("should accept an empty spec with no hooks or admission references", func() {
		p := newMinimalPlugin()
		resp := admit(p)
		Expect(resp.Allowed).To(BeTrue())
	})

	It("should reject CEL domain hook with invalid expression syntax", func() {
		p := newMinimalPlugin()
		p.Spec.DomainHooks = []pluginv1alpha1.DomainHook{{
			CEL: &pluginv1alpha1.CELDomainHook{Expression: "invalid!!! syntax"},
		}}
		resp := admit(p)
		Expect(resp.Allowed).To(BeFalse())
		Expect(resp.Result.Details.Causes).To(ContainElement(
			WithTransform(func(c metav1.StatusCause) string { return c.Message }, ContainSubstring("invalid CEL mutation expression")),
		))
	})

	It("should reject CEL domain hook with unknown type name", func() {
		p := newMinimalPlugin()
		p.Spec.DomainHooks = []pluginv1alpha1.DomainHook{{
			CEL: &pluginv1alpha1.CELDomainHook{Expression: `Domainn{Title: "test"}`},
		}}
		resp := admit(p)
		Expect(resp.Allowed).To(BeFalse())
		Expect(resp.Result.Details.Causes).To(ContainElement(
			WithTransform(func(c metav1.StatusCause) string { return c.Message }, ContainSubstring("invalid CEL mutation expression")),
		))
	})

	It("should reject domain hook with invalid condition expression", func() {
		p := newMinimalPlugin()
		p.Spec.DomainHooks = []pluginv1alpha1.DomainHook{{
			CEL:       &pluginv1alpha1.CELDomainHook{Expression: `Domain{Title: "test"}`},
			Condition: "invalid!!! condition",
		}}
		resp := admit(p)
		Expect(resp.Allowed).To(BeFalse())
		Expect(resp.Result.Details.Causes).To(ContainElement(
			WithTransform(func(c metav1.StatusCause) string { return c.Message }, ContainSubstring("invalid CEL condition expression")),
		))
	})

	It("should reject condition that returns non-bool", func() {
		p := newMinimalPlugin()
		p.Spec.DomainHooks = []pluginv1alpha1.DomainHook{{
			CEL:       &pluginv1alpha1.CELDomainHook{Expression: `Domain{Title: "test"}`},
			Condition: `vmi.Name`,
		}}
		resp := admit(p)
		Expect(resp.Allowed).To(BeFalse())
		Expect(resp.Result.Details.Causes).To(ContainElement(
			WithTransform(func(c metav1.StatusCause) string { return c.Message }, ContainSubstring("invalid CEL condition expression")),
		))
	})

	It("should accept valid CEL expression with condition", func() {
		p := newMinimalPlugin()
		p.Spec.DomainHooks = []pluginv1alpha1.DomainHook{{
			CEL:       &pluginv1alpha1.CELDomainHook{Expression: `Domain{Title: "test"}`},
			Condition: `vmi.Name == "test"`,
		}}
		resp := admit(p)
		Expect(resp.Allowed).To(BeTrue())
	})

	It("should reject CEL domain hook with type-incorrect field value", func() {
		p := newMinimalPlugin()
		p.Spec.DomainHooks = []pluginv1alpha1.DomainHook{{
			CEL: &pluginv1alpha1.CELDomainHook{Expression: `Domain{Memory: "not-a-struct"}`},
		}}
		resp := admit(p)
		Expect(resp.Allowed).To(BeFalse())
		Expect(resp.Result.Details.Causes).To(ContainElement(
			WithTransform(func(c metav1.StatusCause) string { return c.Message }, ContainSubstring("invalid CEL mutation expression")),
		))
	})

	It("should reject node hook with invalid condition expression", func() {
		p := newMinimalPlugin()
		p.Spec.NodeHooks = []pluginv1alpha1.NodeHook{{
			Socket:         "/var/run/kubevirt/plugins/test.sock",
			PermittedHooks: []pluginv1alpha1.NodeHookPoint{pluginv1alpha1.NodeHookPreVMStart},
			Condition:      "invalid!!! condition",
		}}
		resp := admit(p)
		Expect(resp.Allowed).To(BeFalse())
		Expect(resp.Result.Details.Causes).To(ContainElement(
			WithTransform(func(c metav1.StatusCause) string { return c.Message }, ContainSubstring("invalid CEL condition expression")),
		))
	})

	It("should accept node hook with valid condition expression", func() {
		p := newMinimalPlugin()
		p.Spec.NodeHooks = []pluginv1alpha1.NodeHook{{
			Socket:         "/var/run/kubevirt/plugins/test.sock",
			PermittedHooks: []pluginv1alpha1.NodeHookPoint{pluginv1alpha1.NodeHookPreVMStart},
			Condition:      `vmi.spec.domain.cpu.cores > 0`,
		}}
		resp := admit(p)
		Expect(resp.Allowed).To(BeTrue())
	})

	Context("sidecar socketPath validation", func() {
		It("should accept a valid sidecar socketPath", func() {
			p := newMinimalPlugin()
			p.Name = "my-plugin"
			p.Spec.DomainHooks = []pluginv1alpha1.DomainHook{{
				Sidecar: &pluginv1alpha1.SidecarDomainHook{
					SocketPath: "/var/run/kubevirt-plugin/my-plugin/hook.sock",
				},
			}}
			resp := admit(p)
			Expect(resp.Allowed).To(BeTrue())
		})

		DescribeTable("should reject socketPath with unclean path segments", func(socketPath string) {
			p := newMinimalPlugin()
			p.Name = "my-plugin"
			p.Spec.DomainHooks = []pluginv1alpha1.DomainHook{{
				Sidecar: &pluginv1alpha1.SidecarDomainHook{SocketPath: socketPath},
			}}
			resp := admit(p)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Details.Causes).To(ContainElement(
				WithTransform(func(c metav1.StatusCause) string { return c.Message }, ContainSubstring("must be a clean path")),
			))
		},
			Entry("path traversal with ..", "/var/run/kubevirt-plugin/my-plugin/../other/hook.sock"),
			Entry("dot segment", "/var/run/kubevirt-plugin/my-plugin/./hook.sock"),
			Entry("double slash", "/var/run/kubevirt-plugin/my-plugin//hook.sock"),
			Entry("trailing slash", "/var/run/kubevirt-plugin/my-plugin/hook.sock/"),
		)

	})
})

func newMinimalPlugin() *pluginv1alpha1.Plugin {
	return &pluginv1alpha1.Plugin{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-plugin",
		},
		Spec: pluginv1alpha1.PluginSpec{},
	}
}

func newValidPluginWithDomainHook() *pluginv1alpha1.Plugin {
	p := newMinimalPlugin()
	p.Spec.DomainHooks = []pluginv1alpha1.DomainHook{{
		CEL: &pluginv1alpha1.CELDomainHook{Expression: `Domain{Title: "test"}`},
	}}
	return p
}

func createPluginAdmissionReview(p *pluginv1alpha1.Plugin) *admissionv1.AdmissionReview {
	raw, err := json.Marshal(p)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	return &admissionv1.AdmissionReview{
		Request: &admissionv1.AdmissionRequest{
			Resource: metav1.GroupVersionResource{
				Group:    plugin.GroupName,
				Version:  pluginv1alpha1.SchemeGroupVersion.Version,
				Resource: plugin.ResourcePluginPlural,
			},
			Object: runtime.RawExtension{
				Raw: raw,
			},
			Operation: admissionv1.Create,
		},
	}
}
