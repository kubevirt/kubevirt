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

package plugins_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	pluginv1alpha1 "kubevirt.io/api/plugin/v1alpha1"

	virtwrapApi "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/plugins"
)

var _ = Describe("Domain Hook Pipeline", func() {
	var (
		vmi  *v1.VirtualMachineInstance
		spec *virtwrapApi.DomainSpec
	)

	BeforeEach(func() {
		vmi = &v1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-vmi",
				Namespace: "default",
				Labels:    map[string]string{"app": "test"},
			},
		}
		spec = &virtwrapApi.DomainSpec{}
		spec.Type = "kvm"
		spec.Name = "test-vm"
	})

	Context("with no plugins", func() {
		It("should return the original spec unchanged", func() {
			result, xmlStr, err := plugins.ApplyDomainHooks(nil, vmi, spec, pluginv1alpha1.InvocationContextBoot)
			Expect(err).NotTo(HaveOccurred())
			Expect(xmlStr).To(BeEmpty())
			Expect(result).To(Equal(spec))
		})

		It("should return original spec for empty plugin list", func() {
			result, xmlStr, err := plugins.ApplyDomainHooks([]pluginv1alpha1.Plugin{}, vmi, spec, pluginv1alpha1.InvocationContextBoot)
			Expect(err).NotTo(HaveOccurred())
			Expect(xmlStr).To(BeEmpty())
			Expect(result).To(Equal(spec))
		})
	})

	Context("with a single CEL hook", func() {
		It("should apply a simple mutation", func() {
			plugin := pluginv1alpha1.Plugin{
				ObjectMeta: metav1.ObjectMeta{Name: "test-plugin"},
				Spec: pluginv1alpha1.PluginSpec{
					DomainHooks: []pluginv1alpha1.DomainHook{
						{
							CEL: &pluginv1alpha1.CELDomainHook{Expression: `Domain{Title: "modified"}`},
						},
					},
				},
			}

			result, xmlStr, err := plugins.ApplyDomainHooks([]pluginv1alpha1.Plugin{plugin}, vmi, spec, pluginv1alpha1.InvocationContextBoot)
			Expect(err).NotTo(HaveOccurred())
			Expect(xmlStr).NotTo(BeEmpty())
			Expect(xmlStr).To(ContainSubstring("modified"))
			Expect(result).NotTo(BeNil())
		})
	})

	Context("with hook conditions", func() {
		It("should skip hooks when condition is false", func() {
			plugin := pluginv1alpha1.Plugin{
				ObjectMeta: metav1.ObjectMeta{Name: "conditional"},
				Spec: pluginv1alpha1.PluginSpec{
					DomainHooks: []pluginv1alpha1.DomainHook{
						{
							Condition: `vmi.Labels["app"] == "nonexistent"`,
							CEL:       &pluginv1alpha1.CELDomainHook{Expression: `Domain{Title: "should-not-appear"}`},
						},
					},
				},
			}

			result, xmlStr, err := plugins.ApplyDomainHooks([]pluginv1alpha1.Plugin{plugin}, vmi, spec, pluginv1alpha1.InvocationContextBoot)
			Expect(err).NotTo(HaveOccurred())
			Expect(xmlStr).NotTo(ContainSubstring("should-not-appear"))
			Expect(result).NotTo(BeNil())
		})

		It("should apply hooks when condition is true", func() {
			plugin := pluginv1alpha1.Plugin{
				ObjectMeta: metav1.ObjectMeta{Name: "conditional"},
				Spec: pluginv1alpha1.PluginSpec{
					DomainHooks: []pluginv1alpha1.DomainHook{
						{
							Condition: `vmi.Labels["app"] == "test"`,
							CEL:       &pluginv1alpha1.CELDomainHook{Expression: `Domain{Title: "applied"}`},
						},
					},
				},
			}

			result, xmlStr, err := plugins.ApplyDomainHooks([]pluginv1alpha1.Plugin{plugin}, vmi, spec, pluginv1alpha1.InvocationContextBoot)
			Expect(err).NotTo(HaveOccurred())
			Expect(xmlStr).To(ContainSubstring("applied"))
			Expect(result).NotTo(BeNil())
		})
	})

	Context("no-op hooks", func() {
		It("should preserve all domain fields through a no-op pipeline round-trip", func() {
			spec.Devices.Inputs = []virtwrapApi.Input{{Type: v1.InputTypeTablet, Bus: "usb"}}
			spec.Devices.Watchdogs = []virtwrapApi.Watchdog{{Model: "i6300esb", Action: "none"}}

			plugin := pluginv1alpha1.Plugin{
				ObjectMeta: metav1.ObjectMeta{Name: "noop"},
				Spec: pluginv1alpha1.PluginSpec{
					DomainHooks: []pluginv1alpha1.DomainHook{
						{
							CEL: &pluginv1alpha1.CELDomainHook{Expression: `Domain{}`},
						},
					},
				},
			}

			result, xmlStr, err := plugins.ApplyDomainHooks([]pluginv1alpha1.Plugin{plugin}, vmi, spec, pluginv1alpha1.InvocationContextBoot)
			Expect(err).NotTo(HaveOccurred())
			Expect(xmlStr).NotTo(BeEmpty())
			Expect(result.Type).To(Equal("kvm"))
			Expect(result.Name).To(Equal("test-vm"))
			Expect(result.Devices.Inputs).To(HaveLen(1))
			Expect(result.Devices.Inputs[0].Type).To(Equal(v1.InputTypeTablet))
			Expect(result.Devices.Watchdogs).To(HaveLen(1))
			Expect(result.Devices.Watchdogs[0].Model).To(Equal("i6300esb"))
		})

		It("should preserve QEMU commandline args through the plugin pipeline", func() {
			spec.XmlNS = "http://libvirt.org/schemas/domain/qemu/1.0"
			spec.QEMUCmd = &virtwrapApi.Commandline{
				QEMUArg: []virtwrapApi.Arg{
					{Value: "-fw_cfg"},
					{Value: "name=opt/test,string=value"},
				},
				QEMUEnv: []virtwrapApi.Env{
					{Name: "QEMU_TEST", Value: "1"},
				},
			}

			plugin := pluginv1alpha1.Plugin{
				ObjectMeta: metav1.ObjectMeta{Name: "noop"},
				Spec: pluginv1alpha1.PluginSpec{
					DomainHooks: []pluginv1alpha1.DomainHook{
						{
							CEL: &pluginv1alpha1.CELDomainHook{Expression: `Domain{}`},
						},
					},
				},
			}

			result, xmlStr, err := plugins.ApplyDomainHooks([]pluginv1alpha1.Plugin{plugin}, vmi, spec, pluginv1alpha1.InvocationContextBoot)
			Expect(err).NotTo(HaveOccurred())
			Expect(xmlStr).NotTo(BeEmpty())
			Expect(result.QEMUCmd).NotTo(BeNil())
			Expect(result.QEMUCmd.QEMUArg).To(HaveLen(2))
			Expect(result.QEMUCmd.QEMUArg[0].Value).To(Equal("-fw_cfg"))
			Expect(result.QEMUCmd.QEMUArg[1].Value).To(Equal("name=opt/test,string=value"))
			Expect(result.QEMUCmd.QEMUEnv).To(HaveLen(1))
			Expect(result.QEMUCmd.QEMUEnv[0].Name).To(Equal("QEMU_TEST"))
			Expect(result.QEMUCmd.QEMUEnv[0].Value).To(Equal("1"))
		})
	})

	Context("compound conditions", func() {
		It("should apply hook when compound label and namespace condition matches", func() {
			plugin := pluginv1alpha1.Plugin{
				ObjectMeta: metav1.ObjectMeta{Name: "compound"},
				Spec: pluginv1alpha1.PluginSpec{
					DomainHooks: []pluginv1alpha1.DomainHook{
						{
							Condition: `vmi.Labels["app"] == "test" && vmi.Namespace == "default"`,
							CEL:       &pluginv1alpha1.CELDomainHook{Expression: `Domain{Title: "compound-match"}`},
						},
					},
				},
			}

			_, xmlStr, err := plugins.ApplyDomainHooks([]pluginv1alpha1.Plugin{plugin}, vmi, spec, pluginv1alpha1.InvocationContextBoot)
			Expect(err).NotTo(HaveOccurred())
			Expect(xmlStr).To(ContainSubstring("compound-match"))
		})

		It("should skip hook when one part of compound condition is false", func() {
			plugin := pluginv1alpha1.Plugin{
				ObjectMeta: metav1.ObjectMeta{Name: "compound"},
				Spec: pluginv1alpha1.PluginSpec{
					DomainHooks: []pluginv1alpha1.DomainHook{
						{
							Condition: `vmi.Labels["app"] == "test" && vmi.Namespace == "wrong-ns"`,
							CEL:       &pluginv1alpha1.CELDomainHook{Expression: `Domain{Title: "should-not-appear"}`},
						},
					},
				},
			}

			_, xmlStr, err := plugins.ApplyDomainHooks([]pluginv1alpha1.Plugin{plugin}, vmi, spec, pluginv1alpha1.InvocationContextBoot)
			Expect(err).NotTo(HaveOccurred())
			Expect(xmlStr).NotTo(ContainSubstring("should-not-appear"))
		})
	})

	Context("plugin ordering", func() {
		It("should produce deterministic results when multiple plugins compete on the same field", func() {
			pluginBeta := pluginv1alpha1.Plugin{
				ObjectMeta: metav1.ObjectMeta{Name: "beta-plugin"},
				Spec: pluginv1alpha1.PluginSpec{
					DomainHooks: []pluginv1alpha1.DomainHook{
						{
							CEL: &pluginv1alpha1.CELDomainHook{Expression: `Domain{Title: "from-beta"}`},
						},
					},
				},
			}
			pluginAlpha := pluginv1alpha1.Plugin{
				ObjectMeta: metav1.ObjectMeta{Name: "alpha-plugin"},
				Spec: pluginv1alpha1.PluginSpec{
					DomainHooks: []pluginv1alpha1.DomainHook{
						{
							CEL: &pluginv1alpha1.CELDomainHook{Expression: `Domain{Title: "from-alpha"}`},
						},
					},
				},
			}

			for i := 0; i < 10; i++ {
				_, xmlStr, err := plugins.ApplyDomainHooks(
					[]pluginv1alpha1.Plugin{pluginBeta, pluginAlpha}, vmi, spec, pluginv1alpha1.InvocationContextBoot)
				Expect(err).NotTo(HaveOccurred())
				Expect(xmlStr).To(ContainSubstring("from-beta"))
				Expect(xmlStr).NotTo(ContainSubstring("from-alpha"))
			}
		})

		It("should apply hooks in spec order within a single plugin", func() {
			plugin := pluginv1alpha1.Plugin{
				ObjectMeta: metav1.ObjectMeta{Name: "multi-hook"},
				Spec: pluginv1alpha1.PluginSpec{
					DomainHooks: []pluginv1alpha1.DomainHook{
						{
							CEL: &pluginv1alpha1.CELDomainHook{Expression: `Domain{Title: "first"}`},
						},
						{
							CEL: &pluginv1alpha1.CELDomainHook{Expression: `Domain{Title: "second"}`},
						},
					},
				},
			}

			result, xmlStr, err := plugins.ApplyDomainHooks([]pluginv1alpha1.Plugin{plugin}, vmi, spec, pluginv1alpha1.InvocationContextBoot)
			Expect(err).NotTo(HaveOccurred())
			Expect(xmlStr).To(ContainSubstring("second"))
			Expect(xmlStr).NotTo(ContainSubstring("first"))
			Expect(result).NotTo(BeNil())
		})

		It("should apply plugins in alphabetical order by name", func() {
			pluginB := pluginv1alpha1.Plugin{
				ObjectMeta: metav1.ObjectMeta{Name: "b-plugin"},
				Spec: pluginv1alpha1.PluginSpec{
					DomainHooks: []pluginv1alpha1.DomainHook{
						{
							CEL: &pluginv1alpha1.CELDomainHook{Expression: `Domain{Description: "from-b"}`},
						},
					},
				},
			}
			pluginA := pluginv1alpha1.Plugin{
				ObjectMeta: metav1.ObjectMeta{Name: "a-plugin"},
				Spec: pluginv1alpha1.PluginSpec{
					DomainHooks: []pluginv1alpha1.DomainHook{
						{
							CEL: &pluginv1alpha1.CELDomainHook{Expression: `Domain{Title: "from-a"}`},
						},
					},
				},
			}

			result, xmlStr, err := plugins.ApplyDomainHooks([]pluginv1alpha1.Plugin{pluginB, pluginA}, vmi, spec, pluginv1alpha1.InvocationContextBoot)
			Expect(err).NotTo(HaveOccurred())
			Expect(xmlStr).To(ContainSubstring("from-a"))
			Expect(xmlStr).To(ContainSubstring("from-b"))
			Expect(result).NotTo(BeNil())
		})
	})

	Context("plugin-level condition", func() {
		It("should skip all hooks when plugin condition is false", func() {
			plugin := pluginv1alpha1.Plugin{
				ObjectMeta: metav1.ObjectMeta{Name: "filtered"},
				Spec: pluginv1alpha1.PluginSpec{
					Condition: `vmi.Labels["app"] == "nonexistent"`,
					DomainHooks: []pluginv1alpha1.DomainHook{
						{
							CEL: &pluginv1alpha1.CELDomainHook{Expression: `Domain{Title: "should-not-appear"}`},
						},
					},
				},
			}

			result, xmlStr, err := plugins.ApplyDomainHooks([]pluginv1alpha1.Plugin{plugin}, vmi, spec, pluginv1alpha1.InvocationContextBoot)
			Expect(err).NotTo(HaveOccurred())
			Expect(xmlStr).NotTo(ContainSubstring("should-not-appear"))
			Expect(result).NotTo(BeNil())
		})

		It("should apply hooks when plugin condition is true", func() {
			plugin := pluginv1alpha1.Plugin{
				ObjectMeta: metav1.ObjectMeta{Name: "matching"},
				Spec: pluginv1alpha1.PluginSpec{
					Condition: `vmi.Labels["app"] == "test"`,
					DomainHooks: []pluginv1alpha1.DomainHook{
						{
							CEL: &pluginv1alpha1.CELDomainHook{Expression: `Domain{Title: "plugin-matched"}`},
						},
					},
				},
			}

			_, xmlStr, err := plugins.ApplyDomainHooks([]pluginv1alpha1.Plugin{plugin}, vmi, spec, pluginv1alpha1.InvocationContextBoot)
			Expect(err).NotTo(HaveOccurred())
			Expect(xmlStr).To(ContainSubstring("plugin-matched"))
		})

		It("should hard-fail on invalid plugin condition regardless of failure strategy", func() {
			plugin := pluginv1alpha1.Plugin{
				ObjectMeta: metav1.ObjectMeta{Name: "bad-condition"},
				Spec: pluginv1alpha1.PluginSpec{
					Condition:       `invalid!!! expression`,
					FailureStrategy: pluginv1alpha1.FailureStrategyIgnore,
					DomainHooks: []pluginv1alpha1.DomainHook{
						{
							CEL: &pluginv1alpha1.CELDomainHook{Expression: `Domain{Title: "irrelevant"}`},
						},
					},
				},
			}

			_, _, err := plugins.ApplyDomainHooks([]pluginv1alpha1.Plugin{plugin}, vmi, spec, pluginv1alpha1.InvocationContextBoot)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("condition"))
		})

		It("should evaluate hook condition only when plugin condition passes", func() {
			plugin := pluginv1alpha1.Plugin{
				ObjectMeta: metav1.ObjectMeta{Name: "both-conditions"},
				Spec: pluginv1alpha1.PluginSpec{
					Condition: `vmi.Labels["app"] == "test"`,
					DomainHooks: []pluginv1alpha1.DomainHook{
						{
							Condition: `vmi.Namespace == "wrong"`,
							CEL:       &pluginv1alpha1.CELDomainHook{Expression: `Domain{Title: "should-not-appear"}`},
						},
						{
							Condition: `vmi.Namespace == "default"`,
							CEL:       &pluginv1alpha1.CELDomainHook{Expression: `Domain{Title: "both-passed"}`},
						},
					},
				},
			}

			_, xmlStr, err := plugins.ApplyDomainHooks([]pluginv1alpha1.Plugin{plugin}, vmi, spec, pluginv1alpha1.InvocationContextBoot)
			Expect(err).NotTo(HaveOccurred())
			Expect(xmlStr).To(ContainSubstring("both-passed"))
			Expect(xmlStr).NotTo(ContainSubstring("should-not-appear"))
		})
	})

	Context("failure strategy", func() {
		DescribeTable("should resolve failure strategy correctly", func(pluginStrategy, hookStrategy pluginv1alpha1.FailureStrategy, expectErr bool) {
			plugin := pluginv1alpha1.Plugin{
				ObjectMeta: metav1.ObjectMeta{Name: "test-strategy"},
				Spec: pluginv1alpha1.PluginSpec{
					FailureStrategy: pluginStrategy,
					DomainHooks: []pluginv1alpha1.DomainHook{
						{
							FailureStrategy: hookStrategy,
							CEL:             &pluginv1alpha1.CELDomainHook{Expression: `invalid!!! expression`},
						},
					},
				},
			}

			_, _, err := plugins.ApplyDomainHooks([]pluginv1alpha1.Plugin{plugin}, vmi, spec, pluginv1alpha1.InvocationContextBoot)
			if expectErr {
				Expect(err).To(HaveOccurred())
			} else {
				Expect(err).NotTo(HaveOccurred())
			}
		},
			Entry("should abort on Fail strategy", pluginv1alpha1.FailureStrategy(""), pluginv1alpha1.FailureStrategyFail, true),
			Entry("should continue on Ignore strategy", pluginv1alpha1.FailureStrategy(""), pluginv1alpha1.FailureStrategyIgnore, false),
			Entry("should default to Fail when no strategy is set", pluginv1alpha1.FailureStrategy(""), pluginv1alpha1.FailureStrategy(""), true),
			Entry("should fall back to plugin-level Ignore when hook has none", pluginv1alpha1.FailureStrategyIgnore, pluginv1alpha1.FailureStrategy(""), false),
			Entry("should let hook-level Fail override plugin-level Ignore", pluginv1alpha1.FailureStrategyIgnore, pluginv1alpha1.FailureStrategyFail, true),
		)
	})

	Context("sidecar hooks", func() {
		It("should fail when sidecar socket is unreachable and failureStrategy is Fail", func() {
			plugin := pluginv1alpha1.Plugin{
				ObjectMeta: metav1.ObjectMeta{Name: "sidecar-plugin"},
				Spec: pluginv1alpha1.PluginSpec{
					DomainHooks: []pluginv1alpha1.DomainHook{
						{
							Sidecar: &pluginv1alpha1.SidecarDomainHook{SocketPath: "/tmp/test.sock"},
						},
					},
				},
			}

			_, _, err := plugins.ApplyDomainHooks([]pluginv1alpha1.Plugin{plugin}, vmi, spec, pluginv1alpha1.InvocationContextBoot)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not ready"))
		})

		It("should skip sidecar hooks gracefully when failureStrategy is Ignore", func() {
			plugin := pluginv1alpha1.Plugin{
				ObjectMeta: metav1.ObjectMeta{Name: "sidecar-plugin"},
				Spec: pluginv1alpha1.PluginSpec{
					DomainHooks: []pluginv1alpha1.DomainHook{
						{
							Sidecar:         &pluginv1alpha1.SidecarDomainHook{SocketPath: "/tmp/test.sock"},
							FailureStrategy: pluginv1alpha1.FailureStrategyIgnore,
						},
					},
				},
			}

			result, xmlStr, err := plugins.ApplyDomainHooks([]pluginv1alpha1.Plugin{plugin}, vmi, spec, pluginv1alpha1.InvocationContextBoot)
			Expect(err).NotTo(HaveOccurred())
			Expect(xmlStr).NotTo(BeEmpty())
			Expect(result).NotTo(BeNil())
		})
	})

	Context("plugin store", func() {
		It("should return nil when no plugins are set", func() {
			Expect(plugins.GetPlugins()).To(BeNil())
		})

		It("should store and retrieve plugins", func() {
			pluginList := []pluginv1alpha1.Plugin{
				{ObjectMeta: metav1.ObjectMeta{Name: "stored-plugin"}},
			}
			plugins.SetPlugins(pluginList)
			defer plugins.SetPlugins(nil)

			retrieved := plugins.GetPlugins()
			Expect(retrieved).To(HaveLen(1))
			Expect(retrieved[0].Name).To(Equal("stored-plugin"))
		})
	})
})
