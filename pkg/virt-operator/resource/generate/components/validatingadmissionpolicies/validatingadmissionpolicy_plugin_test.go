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

package validatingadmissionpolicies_test

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/cel-go/cel"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"

	pluginv1alpha1 "kubevirt.io/api/plugin/v1alpha1"

	vap "kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components/validatingadmissionpolicies"
)

func evalCEL(expression string, object map[string]interface{}) bool {
	return evalCELWithVars(expression, object, map[string]interface{}{})
}

func evalCELValue(expression string, object map[string]interface{}, vars map[string]interface{}) interface{} {
	env, err := cel.NewEnv(
		cel.Variable("object", cel.DynType),
		cel.Variable("variables", cel.DynType),
	)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	ast, issues := env.Compile(expression)
	ExpectWithOffset(1, issues.Err()).ToNot(HaveOccurred())

	prg, err := env.Program(ast)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	out, _, err := prg.Eval(map[string]interface{}{
		"object":    object,
		"variables": vars,
	})
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	return out.Value()
}

func evalCELWithVars(expression string, object map[string]interface{}, vars map[string]interface{}) bool {
	return evalCELValue(expression, object, vars).(bool)
}

func pluginToUnstructured(plugin *pluginv1alpha1.Plugin) map[string]interface{} {
	data, err := json.Marshal(plugin)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	var obj map[string]interface{}
	ExpectWithOffset(1, json.Unmarshal(data, &obj)).ToNot(HaveOccurred())
	return obj
}

func evaluatePolicy(policy *admissionregistrationv1.ValidatingAdmissionPolicy, obj map[string]interface{}) error {
	vars := map[string]interface{}{}
	for _, v := range policy.Spec.Variables {
		val := evalCELValue(v.Expression, obj, vars)
		vars[v.Name] = val
	}

	var msgs []string
	for _, v := range policy.Spec.Validations {
		if !evalCELWithVars(v.Expression, obj, vars) {
			msgs = append(msgs, v.Message)
		}
	}
	if len(msgs) > 0 {
		return fmt.Errorf("%s", strings.Join(msgs, "; "))
	}
	return nil
}

func admitPlugin(plugin *pluginv1alpha1.Plugin) error {
	return evaluatePolicy(vap.NewPluginValidatingAdmissionPolicy(), pluginToUnstructured(plugin))
}

func warnPlugin(plugin *pluginv1alpha1.Plugin) error {
	return evaluatePolicy(vap.NewPluginWarningAdmissionPolicy(), pluginToUnstructured(plugin))
}

func generateCELDomainPlugin(expression, condition string) *pluginv1alpha1.Plugin {
	return &pluginv1alpha1.Plugin{
		Spec: pluginv1alpha1.PluginSpec{
			DomainHooks: []pluginv1alpha1.DomainHook{
				{
					CEL:       &pluginv1alpha1.CELDomainHook{Expression: expression},
					Condition: condition,
				},
			},
		},
	}
}

var _ = Describe("Plugin ValidatingAdmissionPolicy", func() {
	Context("Validation policy", func() {
		Context("oneOf: exactly one of cel or sidecar per domain hook", func() {
			It("should accept CEL-only hook", func() {
				plugin := generateCELDomainPlugin("x", "")
				Expect(admitPlugin(plugin)).To(Succeed())
			})

			It("should accept sidecar-only hook", func() {
				plugin := &pluginv1alpha1.Plugin{
					Spec: pluginv1alpha1.PluginSpec{
						DomainHooks: []pluginv1alpha1.DomainHook{
							{Sidecar: &pluginv1alpha1.SidecarDomainHook{SocketPath: "/s"}},
						},
					},
				}
				Expect(admitPlugin(plugin)).To(Succeed())
			})

			It("should reject when both cel and sidecar are set", func() {
				plugin := &pluginv1alpha1.Plugin{
					Spec: pluginv1alpha1.PluginSpec{
						DomainHooks: []pluginv1alpha1.DomainHook{
							{
								CEL:     &pluginv1alpha1.CELDomainHook{Expression: "x"},
								Sidecar: &pluginv1alpha1.SidecarDomainHook{SocketPath: "/s"},
							},
						},
					},
				}
				Expect(admitPlugin(plugin)).To(MatchError(ContainSubstring("exactly one of cel or sidecar")))
			})

			It("should reject when neither cel nor sidecar is set", func() {
				plugin := &pluginv1alpha1.Plugin{
					Spec: pluginv1alpha1.PluginSpec{
						DomainHooks: []pluginv1alpha1.DomainHook{{}},
					},
				}
				Expect(admitPlugin(plugin)).To(MatchError(ContainSubstring("exactly one of cel or sidecar")))
			})

			It("should accept when domainHooks is absent", func() {
				plugin := &pluginv1alpha1.Plugin{}
				Expect(admitPlugin(plugin)).To(Succeed())
			})
		})

		Context("failureStrategy enum on domain hooks", func() {
			It("should accept Fail", func() {
				plugin := generateCELDomainPlugin("x", "")
				plugin.Spec.DomainHooks[0].FailureStrategy = pluginv1alpha1.FailureStrategyFail
				Expect(admitPlugin(plugin)).To(Succeed())
			})

			It("should accept Ignore", func() {
				plugin := generateCELDomainPlugin("x", "")
				plugin.Spec.DomainHooks[0].FailureStrategy = pluginv1alpha1.FailureStrategyIgnore
				Expect(admitPlugin(plugin)).To(Succeed())
			})

			It("should accept when failureStrategy is absent", func() {
				plugin := generateCELDomainPlugin("x", "")
				Expect(admitPlugin(plugin)).To(Succeed())
			})

			It("should reject invalid failureStrategy", func() {
				plugin := generateCELDomainPlugin("x", "")
				plugin.Spec.DomainHooks[0].FailureStrategy = pluginv1alpha1.FailureStrategy("Crash")
				Expect(admitPlugin(plugin)).To(MatchError(ContainSubstring("failureStrategy must be either")))
			})
		})

		Context("failureStrategy enum on node hooks", func() {
			It("should accept Fail", func() {
				plugin := &pluginv1alpha1.Plugin{
					Spec: pluginv1alpha1.PluginSpec{
						NodeHooks: []pluginv1alpha1.NodeHook{
							{
								Socket:          "/s",
								PermittedHooks:  []pluginv1alpha1.NodeHookPoint{pluginv1alpha1.NodeHookPreVMStart},
								FailureStrategy: pluginv1alpha1.FailureStrategyFail,
							},
						},
					},
				}
				Expect(admitPlugin(plugin)).To(Succeed())
			})

			It("should accept Ignore", func() {
				plugin := &pluginv1alpha1.Plugin{
					Spec: pluginv1alpha1.PluginSpec{
						NodeHooks: []pluginv1alpha1.NodeHook{
							{
								Socket:          "/s",
								PermittedHooks:  []pluginv1alpha1.NodeHookPoint{pluginv1alpha1.NodeHookPreVMStart},
								FailureStrategy: pluginv1alpha1.FailureStrategyIgnore,
							},
						},
					},
				}
				Expect(admitPlugin(plugin)).To(Succeed())
			})

			It("should reject invalid failureStrategy", func() {
				plugin := &pluginv1alpha1.Plugin{
					Spec: pluginv1alpha1.PluginSpec{
						NodeHooks: []pluginv1alpha1.NodeHook{
							{
								Socket:          "/s",
								PermittedHooks:  []pluginv1alpha1.NodeHookPoint{pluginv1alpha1.NodeHookPreVMStart},
								FailureStrategy: pluginv1alpha1.FailureStrategy("Abort"),
							},
						},
					},
				}
				Expect(admitPlugin(plugin)).To(MatchError(ContainSubstring("failureStrategy must be either")))
			})

			It("should accept when nodeHooks is absent", func() {
				plugin := &pluginv1alpha1.Plugin{}
				Expect(admitPlugin(plugin)).To(Succeed())
			})
		})

		Context("nodeHookPoint enum on node hooks", func() {
			It("should accept valid hook points", func() {
				plugin := &pluginv1alpha1.Plugin{
					Spec: pluginv1alpha1.PluginSpec{
						NodeHooks: []pluginv1alpha1.NodeHook{
							{
								Socket:         "/s",
								PermittedHooks: []pluginv1alpha1.NodeHookPoint{pluginv1alpha1.NodeHookPreVMStart, pluginv1alpha1.NodeHookPostVMStop},
							},
						},
					},
				}
				Expect(admitPlugin(plugin)).To(Succeed())
			})

			It("should reject invalid hook points", func() {
				plugin := &pluginv1alpha1.Plugin{
					Spec: pluginv1alpha1.PluginSpec{
						NodeHooks: []pluginv1alpha1.NodeHook{
							{
								Socket:         "/s",
								PermittedHooks: []pluginv1alpha1.NodeHookPoint{"InvalidPoint"},
							},
						},
					},
				}
				Expect(admitPlugin(plugin)).To(MatchError(ContainSubstring("permittedHooks must only contain valid")))
			})
		})

		Context("failureStrategy enum on plugin", func() {
			It("should accept Fail", func() {
				plugin := generateCELDomainPlugin("x", "")
				plugin.Spec.FailureStrategy = pluginv1alpha1.FailureStrategyFail
				Expect(admitPlugin(plugin)).To(Succeed())
			})

			It("should accept Ignore", func() {
				plugin := generateCELDomainPlugin("x", "")
				plugin.Spec.FailureStrategy = pluginv1alpha1.FailureStrategyIgnore
				Expect(admitPlugin(plugin)).To(Succeed())
			})

			It("should accept when failureStrategy is absent", func() {
				plugin := generateCELDomainPlugin("x", "")
				Expect(admitPlugin(plugin)).To(Succeed())
			})

			It("should reject invalid failureStrategy", func() {
				plugin := generateCELDomainPlugin("x", "")
				plugin.Spec.FailureStrategy = pluginv1alpha1.FailureStrategy("Crash")
				Expect(admitPlugin(plugin)).To(MatchError(ContainSubstring("failureStrategy must be either")))
			})
		})
	})

	Context("Warning policy", func() {
		It("should not warn when domainHooks is present", func() {
			plugin := generateCELDomainPlugin("x", "")
			Expect(warnPlugin(plugin)).To(Succeed())
		})

		It("should not warn when nodeHooks is present", func() {
			plugin := &pluginv1alpha1.Plugin{
				Spec: pluginv1alpha1.PluginSpec{
					NodeHooks: []pluginv1alpha1.NodeHook{
						{Socket: "/s", PermittedHooks: []pluginv1alpha1.NodeHookPoint{pluginv1alpha1.NodeHookPreVMStart}},
					},
				},
			}
			Expect(warnPlugin(plugin)).To(Succeed())
		})

		It("should not warn when validatingAdmissionPolicies is present", func() {
			plugin := &pluginv1alpha1.Plugin{
				Spec: pluginv1alpha1.PluginSpec{
					ValidatingAdmissionPolicies: []pluginv1alpha1.AdmissionReference{{Name: "p"}},
				},
			}
			Expect(warnPlugin(plugin)).To(Succeed())
		})

		It("should not warn when mutatingAdmissionWebhooks is present", func() {
			plugin := &pluginv1alpha1.Plugin{
				Spec: pluginv1alpha1.PluginSpec{
					MutatingAdmissionWebhooks: []pluginv1alpha1.AdmissionReference{{Name: "w"}},
				},
			}
			Expect(warnPlugin(plugin)).To(Succeed())
		})

		It("should warn when spec is empty", func() {
			plugin := &pluginv1alpha1.Plugin{}
			Expect(warnPlugin(plugin)).To(MatchError(ContainSubstring("no hooks or admission references")))
		})
	})

	Context("Bindings", func() {
		It("validation binding should reference validation policy with Deny action", func() {
			policy := vap.NewPluginValidatingAdmissionPolicy()
			binding := vap.NewPluginValidatingAdmissionPolicyBinding()
			Expect(binding.Spec.PolicyName).To(Equal(policy.Name))
			Expect(binding.Spec.ValidationActions).To(ConsistOf(admissionregistrationv1.Deny))
		})

		It("warning binding should reference warning policy with Warn action", func() {
			policy := vap.NewPluginWarningAdmissionPolicy()
			binding := vap.NewPluginWarningAdmissionPolicyBinding()
			Expect(binding.Spec.PolicyName).To(Equal(policy.Name))
			Expect(binding.Spec.ValidationActions).To(ConsistOf(admissionregistrationv1.Warn))
		})
	})
})
