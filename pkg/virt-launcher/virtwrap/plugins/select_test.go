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

package plugins

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pluginv1alpha1 "kubevirt.io/api/plugin/v1alpha1"
)

var _ = Describe("selectHooks", func() {
	var evalCondition func(expr string) (bool, error)

	BeforeEach(func() {
		evalCondition = func(expr string) (bool, error) {
			switch expr {
			case "":
				return true, nil
			case "true":
				return true, nil
			case "false":
				return false, nil
			case "error":
				return false, errors.New("condition evaluation boom")
			default:
				return true, nil
			}
		}
	})

	Context("hook point filtering", func() {
		It("selects a CEL hook matching the hook point", func() {
			plugin := pluginv1alpha1.Plugin{
				ObjectMeta: metav1.ObjectMeta{Name: "plugin"},
				Spec: pluginv1alpha1.PluginSpec{
					LauncherHooks: []pluginv1alpha1.LauncherHook{
						{CEL: &pluginv1alpha1.CELLauncherHook{HookPoint: pluginv1alpha1.LauncherHookGuestDefinition, Expression: "Domain{}"}},
					},
				},
			}

			resolved, err := selectHooks([]pluginv1alpha1.Plugin{plugin}, pluginv1alpha1.LauncherHookGuestDefinition, evalCondition)
			Expect(err).NotTo(HaveOccurred())
			Expect(resolved).To(HaveLen(1))
			Expect(resolved[0].pluginName).To(Equal("plugin"))
		})

		It("skips a CEL hook with an unsupported hook point", func() {
			plugin := pluginv1alpha1.Plugin{
				ObjectMeta: metav1.ObjectMeta{Name: "plugin"},
				Spec: pluginv1alpha1.PluginSpec{
					LauncherHooks: []pluginv1alpha1.LauncherHook{
						{CEL: &pluginv1alpha1.CELLauncherHook{HookPoint: pluginv1alpha1.LauncherHookPoint("Unsupported"), Expression: "Domain{}"}},
					},
				},
			}

			resolved, err := selectHooks([]pluginv1alpha1.Plugin{plugin}, pluginv1alpha1.LauncherHookGuestDefinition, evalCondition)
			Expect(err).NotTo(HaveOccurred())
			Expect(resolved).To(BeEmpty())
		})

		It("selects a sidecar hook whose permittedHooks include the hook point", func() {
			plugin := pluginv1alpha1.Plugin{
				ObjectMeta: metav1.ObjectMeta{Name: "plugin"},
				Spec: pluginv1alpha1.PluginSpec{
					LauncherHooks: []pluginv1alpha1.LauncherHook{
						{Sidecar: &pluginv1alpha1.SidecarLauncherHook{SocketPath: "/tmp/hook.sock", PermittedHooks: []pluginv1alpha1.LauncherHookPoint{pluginv1alpha1.LauncherHookPoint("Unsupported"), pluginv1alpha1.LauncherHookGuestDefinition}}},
					},
				},
			}

			resolved, err := selectHooks([]pluginv1alpha1.Plugin{plugin}, pluginv1alpha1.LauncherHookGuestDefinition, evalCondition)
			Expect(err).NotTo(HaveOccurred())
			Expect(resolved).To(HaveLen(1))
		})

		It("skips a sidecar hook with only an unsupported hook point", func() {
			plugin := pluginv1alpha1.Plugin{
				ObjectMeta: metav1.ObjectMeta{Name: "plugin"},
				Spec: pluginv1alpha1.PluginSpec{
					LauncherHooks: []pluginv1alpha1.LauncherHook{
						{Sidecar: &pluginv1alpha1.SidecarLauncherHook{SocketPath: "/tmp/hook.sock", PermittedHooks: []pluginv1alpha1.LauncherHookPoint{pluginv1alpha1.LauncherHookPoint("Unsupported")}}},
					},
				},
			}

			resolved, err := selectHooks([]pluginv1alpha1.Plugin{plugin}, pluginv1alpha1.LauncherHookGuestDefinition, evalCondition)
			Expect(err).NotTo(HaveOccurred())
			Expect(resolved).To(BeEmpty())
		})
	})

	Context("plugin condition", func() {
		It("skips all hooks in a plugin when the plugin condition evaluates false", func() {
			plugin := pluginv1alpha1.Plugin{
				ObjectMeta: metav1.ObjectMeta{Name: "plugin"},
				Spec: pluginv1alpha1.PluginSpec{
					Condition: "false",
					LauncherHooks: []pluginv1alpha1.LauncherHook{
						{CEL: &pluginv1alpha1.CELLauncherHook{HookPoint: pluginv1alpha1.LauncherHookGuestDefinition, Expression: "Domain{}"}},
					},
				},
			}

			resolved, err := selectHooks([]pluginv1alpha1.Plugin{plugin}, pluginv1alpha1.LauncherHookGuestDefinition, evalCondition)
			Expect(err).NotTo(HaveOccurred())
			Expect(resolved).To(BeEmpty())
		})

		It("includes hooks when the plugin condition evaluates true", func() {
			plugin := pluginv1alpha1.Plugin{
				ObjectMeta: metav1.ObjectMeta{Name: "plugin"},
				Spec: pluginv1alpha1.PluginSpec{
					Condition: "true",
					LauncherHooks: []pluginv1alpha1.LauncherHook{
						{CEL: &pluginv1alpha1.CELLauncherHook{HookPoint: pluginv1alpha1.LauncherHookGuestDefinition, Expression: "Domain{}"}},
					},
				},
			}

			resolved, err := selectHooks([]pluginv1alpha1.Plugin{plugin}, pluginv1alpha1.LauncherHookGuestDefinition, evalCondition)
			Expect(err).NotTo(HaveOccurred())
			Expect(resolved).To(HaveLen(1))
		})

		It("returns an error when the plugin condition evaluation fails", func() {
			plugin := pluginv1alpha1.Plugin{
				ObjectMeta: metav1.ObjectMeta{Name: "plugin"},
				Spec: pluginv1alpha1.PluginSpec{
					Condition: "error",
					LauncherHooks: []pluginv1alpha1.LauncherHook{
						{CEL: &pluginv1alpha1.CELLauncherHook{HookPoint: pluginv1alpha1.LauncherHookGuestDefinition, Expression: "Domain{}"}},
					},
				},
			}

			resolved, err := selectHooks([]pluginv1alpha1.Plugin{plugin}, pluginv1alpha1.LauncherHookGuestDefinition, evalCondition)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("plugin plugin condition evaluation failed"))
			Expect(resolved).To(BeNil())
		})
	})

	Context("hook condition", func() {
		It("skips a hook whose condition evaluates false", func() {
			plugin := pluginv1alpha1.Plugin{
				ObjectMeta: metav1.ObjectMeta{Name: "plugin"},
				Spec: pluginv1alpha1.PluginSpec{
					LauncherHooks: []pluginv1alpha1.LauncherHook{
						{Condition: "false", CEL: &pluginv1alpha1.CELLauncherHook{HookPoint: pluginv1alpha1.LauncherHookGuestDefinition, Expression: "Domain{}"}},
					},
				},
			}

			resolved, err := selectHooks([]pluginv1alpha1.Plugin{plugin}, pluginv1alpha1.LauncherHookGuestDefinition, evalCondition)
			Expect(err).NotTo(HaveOccurred())
			Expect(resolved).To(BeEmpty())
		})

		It("returns an error when the hook condition evaluation fails", func() {
			plugin := pluginv1alpha1.Plugin{
				ObjectMeta: metav1.ObjectMeta{Name: "plugin"},
				Spec: pluginv1alpha1.PluginSpec{
					LauncherHooks: []pluginv1alpha1.LauncherHook{
						{Condition: "error", CEL: &pluginv1alpha1.CELLauncherHook{HookPoint: pluginv1alpha1.LauncherHookGuestDefinition, Expression: "Domain{}"}},
					},
				},
			}

			resolved, err := selectHooks([]pluginv1alpha1.Plugin{plugin}, pluginv1alpha1.LauncherHookGuestDefinition, evalCondition)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("plugin plugin hook 0 condition evaluation failed"))
			Expect(resolved).To(BeNil())
		})

		It("only evaluates the hook condition once the plugin condition has passed", func() {
			plugin := pluginv1alpha1.Plugin{
				ObjectMeta: metav1.ObjectMeta{Name: "plugin"},
				Spec: pluginv1alpha1.PluginSpec{
					Condition: "false",
					LauncherHooks: []pluginv1alpha1.LauncherHook{
						{Condition: "error", CEL: &pluginv1alpha1.CELLauncherHook{HookPoint: pluginv1alpha1.LauncherHookGuestDefinition, Expression: "Domain{}"}},
					},
				},
			}

			resolved, err := selectHooks([]pluginv1alpha1.Plugin{plugin}, pluginv1alpha1.LauncherHookGuestDefinition, evalCondition)
			Expect(err).NotTo(HaveOccurred())
			Expect(resolved).To(BeEmpty())
		})
	})

	Context("failure strategy resolution", func() {
		DescribeTable("resolves the effective failure strategy", func(pluginStrategy, hookStrategy, expected pluginv1alpha1.FailureStrategy) {
			plugin := pluginv1alpha1.Plugin{
				ObjectMeta: metav1.ObjectMeta{Name: "plugin"},
				Spec: pluginv1alpha1.PluginSpec{
					FailureStrategy: pluginStrategy,
					LauncherHooks: []pluginv1alpha1.LauncherHook{
						{FailureStrategy: hookStrategy, CEL: &pluginv1alpha1.CELLauncherHook{HookPoint: pluginv1alpha1.LauncherHookGuestDefinition, Expression: "Domain{}"}},
					},
				},
			}

			resolved, err := selectHooks([]pluginv1alpha1.Plugin{plugin}, pluginv1alpha1.LauncherHookGuestDefinition, evalCondition)
			Expect(err).NotTo(HaveOccurred())
			Expect(resolved).To(HaveLen(1))
			Expect(resolved[0].failureStrategy).To(Equal(expected))
		},
			Entry("defaults to Fail when neither plugin nor hook set a strategy", pluginv1alpha1.FailureStrategy(""), pluginv1alpha1.FailureStrategy(""), pluginv1alpha1.FailureStrategyFail),
			Entry("falls back to the plugin-level strategy when the hook has none", pluginv1alpha1.FailureStrategyIgnore, pluginv1alpha1.FailureStrategy(""), pluginv1alpha1.FailureStrategyIgnore),
			Entry("lets the hook-level strategy override the plugin-level strategy", pluginv1alpha1.FailureStrategyIgnore, pluginv1alpha1.FailureStrategyFail, pluginv1alpha1.FailureStrategyFail),
		)
	})

	Context("plugin ordering", func() {
		It("sorts plugins alphabetically by name regardless of input order", func() {
			pluginB := pluginv1alpha1.Plugin{
				ObjectMeta: metav1.ObjectMeta{Name: "b-plugin"},
				Spec: pluginv1alpha1.PluginSpec{
					LauncherHooks: []pluginv1alpha1.LauncherHook{
						{CEL: &pluginv1alpha1.CELLauncherHook{HookPoint: pluginv1alpha1.LauncherHookGuestDefinition, Expression: "Domain{}"}},
					},
				},
			}
			pluginA := pluginv1alpha1.Plugin{
				ObjectMeta: metav1.ObjectMeta{Name: "a-plugin"},
				Spec: pluginv1alpha1.PluginSpec{
					LauncherHooks: []pluginv1alpha1.LauncherHook{
						{CEL: &pluginv1alpha1.CELLauncherHook{HookPoint: pluginv1alpha1.LauncherHookGuestDefinition, Expression: "Domain{}"}},
					},
				},
			}

			resolved, err := selectHooks([]pluginv1alpha1.Plugin{pluginB, pluginA}, pluginv1alpha1.LauncherHookGuestDefinition, evalCondition)
			Expect(err).NotTo(HaveOccurred())
			Expect(resolved).To(HaveLen(2))
			Expect(resolved[0].pluginName).To(Equal("a-plugin"))
			Expect(resolved[1].pluginName).To(Equal("b-plugin"))
		})

		It("preserves hook declaration order within a single plugin", func() {
			plugin := pluginv1alpha1.Plugin{
				ObjectMeta: metav1.ObjectMeta{Name: "plugin"},
				Spec: pluginv1alpha1.PluginSpec{
					LauncherHooks: []pluginv1alpha1.LauncherHook{
						{CEL: &pluginv1alpha1.CELLauncherHook{HookPoint: pluginv1alpha1.LauncherHookGuestDefinition, Expression: "Domain{Title: \"first\"}"}},
						{CEL: &pluginv1alpha1.CELLauncherHook{HookPoint: pluginv1alpha1.LauncherHookGuestDefinition, Expression: "Domain{Title: \"second\"}"}},
					},
				},
			}

			resolved, err := selectHooks([]pluginv1alpha1.Plugin{plugin}, pluginv1alpha1.LauncherHookGuestDefinition, evalCondition)
			Expect(err).NotTo(HaveOccurred())
			Expect(resolved).To(HaveLen(2))
			Expect(resolved[0].index).To(Equal(0))
			Expect(resolved[1].index).To(Equal(1))
			Expect(resolved[0].hook.CEL.Expression).To(ContainSubstring("first"))
			Expect(resolved[1].hook.CEL.Expression).To(ContainSubstring("second"))
		})
	})
})
