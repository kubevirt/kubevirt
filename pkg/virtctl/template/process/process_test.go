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

package process_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	k8stesting "k8s.io/client-go/testing"
	"sigs.k8s.io/yaml"

	virtv1 "kubevirt.io/api/core/v1"
	kvtesting "kubevirt.io/client-go/testing"

	templateapi "kubevirt.io/virt-template-api/core"
	"kubevirt.io/virt-template-api/core/subresourcesv1alpha1"
	"kubevirt.io/virt-template-api/core/v1alpha1"
	templateclient "kubevirt.io/virt-template-client-go/virttemplate"
	templatefake "kubevirt.io/virt-template-client-go/virttemplate/fake"
	"kubevirt.io/virt-template-engine/template"

	"kubevirt.io/kubevirt/pkg/virtctl/template/process"
	"kubevirt.io/kubevirt/pkg/virtctl/testing"
)

const (
	param1Name        = "NAME"
	param1Placeholder = "${NAME}"
	param1Val         = "test-vm"
	param2Name        = "PREFERENCE"
	param2Placeholder = "${PREFERENCE}"
	param2Val         = "fedora"
	param3Name        = "COUNT"

	nameFlag        = "name"
	fileFlag        = "file"
	outputFlag      = "output"
	paramFlag       = "param"
	paramsFileFlag  = "params-file"
	localFlag       = "local"
	createFlag      = "create"
	printParamsFlag = "print-params"

	formatYAML = "yaml"
	formatJSON = "json"
)

var _ = Describe("Process command", func() {
	Context("With file input", func() {
		var file string

		BeforeEach(func() {
			file = filepath.Join(GinkgoT().TempDir(), "template")
		})

		DescribeTable("should print parameters when --print-params is set", func(
			marshalFn func(any) []byte,
			unmarshalFn func([]byte, any) error,
			output string,
		) {
			tpl := newVirtualMachineTemplate()
			Expect(os.WriteFile(file, marshalFn(tpl), 0o600)).To(Succeed())

			out, err := runCmd(
				setFlag(fileFlag, file),
				setFlag(printParamsFlag, "true"),
				setFlag(outputFlag, output),
			)
			Expect(err).ToNot(HaveOccurred())

			var params []v1alpha1.Parameter
			Expect(unmarshalFn(out, &params)).To(Succeed())
			Expect(params).To(Equal(tpl.Spec.Parameters))
		},
			Entry("when params-file is JSON and output format is JSON", marshalJSON, json.Unmarshal, formatJSON),
			Entry("when params-file is JSON and output format is YAML", marshalJSON, unmarshalYAML, formatYAML),
			Entry("when params-file is YAML and output format is JSON", marshalYAML, json.Unmarshal, formatJSON),
			Entry("when params-file is YAML and output format is YAML", marshalYAML, unmarshalYAML, formatYAML),
		)

		DescribeTable("should process template with parameters from flags", func(
			marshalFn func(any) []byte,
			unmarshalFn func([]byte, any) error,
			output string,
		) {
			Expect(os.WriteFile(file, marshalFn(newVirtualMachineTemplate()), 0o600)).To(Succeed())

			out, err := runCmd(
				setFlag(fileFlag, file),
				setParamFlag(param1Name, param1Val),
				setParamFlag(param2Name, param2Val),
				setFlag(outputFlag, output),
			)
			Expect(err).ToNot(HaveOccurred())

			var vm virtv1.VirtualMachine
			Expect(unmarshalFn(out, &vm)).To(Succeed())
			Expect(vm.Name).To(Equal(param1Val))
			Expect(vm.Spec.Preference.Name).To(Equal(param2Val))
		},
			Entry("when template is JSON and output format is JSON", marshalJSON, json.Unmarshal, formatJSON),
			Entry("when template is JSON and output format is YAML", marshalJSON, unmarshalYAML, formatYAML),
			Entry("when template is YAML and output format is JSON", marshalYAML, json.Unmarshal, formatJSON),
			Entry("when template is YAML and output format is YAML", marshalYAML, unmarshalYAML, formatYAML),
		)

		DescribeTable("should process template with parameters from file", func(
			marshalFn func(any) []byte,
			unmarshalFn func([]byte, any) error,
			output string,
		) {
			Expect(os.WriteFile(file, marshalFn(newVirtualMachineTemplate()), 0o600)).To(Succeed())

			params := map[string]string{
				param1Name: param1Val,
				param2Name: param2Val,
			}
			paramsFile := filepath.Join(GinkgoT().TempDir(), "params")
			Expect(os.WriteFile(paramsFile, marshalFn(params), 0o600)).To(Succeed())

			out, err := runCmd(
				setFlag(fileFlag, file),
				setFlag(paramsFileFlag, paramsFile),
				setFlag(outputFlag, output),
			)
			Expect(err).ToNot(HaveOccurred())

			var vm virtv1.VirtualMachine
			Expect(unmarshalFn(out, &vm)).To(Succeed())
			Expect(vm.Name).To(Equal(param1Val))
			Expect(vm.Spec.Preference.Name).To(Equal(param2Val))
		},
			Entry("when input format is JSON and output format is JSON", marshalJSON, json.Unmarshal, formatJSON),
			Entry("when input format is JSON and output format is YAML", marshalJSON, unmarshalYAML, formatYAML),
			Entry("when input format is YAML and output format is JSON", marshalYAML, json.Unmarshal, formatJSON),
			Entry("when input format is YAML and output format is YAML", marshalYAML, unmarshalYAML, formatYAML),
		)

		It("should fail when template file is in invalid format", func() {
			Expect(os.WriteFile(file, []byte("invalid: yaml: content: ["), 0o600)).To(Succeed())

			_, err := runCmd(setFlag(fileFlag, file))
			Expect(err).To(MatchError(ContainSubstring("error decoding VirtualMachineTemplate")))
		})

		It("should fail when params file is in invalid format", func() {
			Expect(os.WriteFile(file, []byte("invalid: [yaml"), 0o600)).To(Succeed())

			_, err := runCmd(
				"something",
				setFlag(paramsFileFlag, file),
			)
			Expect(err).To(MatchError(ContainSubstring("error reading params file")))
		})

		It("should fail when template contains undefined parameter reference", func() {
			tpl := newVirtualMachineTemplateWithSpec(
				&v1alpha1.VirtualMachineTemplateSpec{
					Parameters: []v1alpha1.Parameter{
						{
							Name: param1Name,
						},
					},
					VirtualMachine: &runtime.RawExtension{
						Raw: []byte(`{"metadata":{"name":"${NAME}"},"spec":{"preference":{"name":"${PREFERENCE}"}}}`),
					},
				},
			)
			Expect(os.WriteFile(file, marshalYAML(tpl), 0o600)).To(Succeed())

			_, err := runCmd(
				setFlag(fileFlag, file),
				setParamFlag(param1Name, param1Val),
			)
			Expect(err).To(MatchError(ContainSubstring("references undefined parameter PREFERENCE")))
		})

		It("should warn when template contains unused parameter", func() {
			tpl := newVirtualMachineTemplateWithSpec(
				&v1alpha1.VirtualMachineTemplateSpec{
					Parameters: []v1alpha1.Parameter{
						{
							Name: param1Name,
						},
						{
							Name: param2Name,
						},
					},
					VirtualMachine: &runtime.RawExtension{
						Raw: []byte(`{"metadata":{"name":"${NAME}"}}`),
					},
				},
			)
			Expect(os.WriteFile(file, marshalYAML(tpl), 0o600)).To(Succeed())

			out, errOut, err := runCmdWithErr(
				setFlag(fileFlag, file),
				setParamFlag(param1Name, param1Val),
				setParamFlag(param2Name, param2Val),
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(errOut)).To(ContainSubstring("PREFERENCE is defined but never referenced"))

			var vm virtv1.VirtualMachine
			Expect(yaml.Unmarshal(out, &vm)).To(Succeed())
			Expect(vm.Name).To(Equal(param1Val))
		})

		It("should fail and warn when template contains both undefined and unused parameters", func() {
			tpl := newVirtualMachineTemplateWithSpec(
				&v1alpha1.VirtualMachineTemplateSpec{
					Parameters: []v1alpha1.Parameter{
						{
							Name: param1Name,
						},
						{
							Name: param3Name,
						},
					},
					VirtualMachine: &runtime.RawExtension{
						Raw: []byte(`{"metadata":{"name":"${NAME}"},"spec":{"preference":{"name":"${PREFERENCE}"}}}`),
					},
				},
			)
			Expect(os.WriteFile(file, marshalYAML(tpl), 0o600)).To(Succeed())

			_, errOut, err := runCmdWithErr(setFlag(fileFlag, file))
			Expect(err).To(MatchError(ContainSubstring("references undefined parameter PREFERENCE")))
			Expect(string(errOut)).To(ContainSubstring("COUNT is defined but never referenced"))
		})
	})

	Context("With remote template", func() {
		var (
			tplClient             *templatefake.Clientset
			origGetTemplateClient func(*rest.Config) (templateclient.Interface, error)
			tpl                   *v1alpha1.VirtualMachineTemplate
		)

		BeforeEach(func() {
			tplClient = templatefake.NewSimpleClientset()
			origGetTemplateClient = process.GetTemplateClient
			process.GetTemplateClient = func(_ *rest.Config) (templateclient.Interface, error) {
				return tplClient, nil
			}

			var err error
			tpl, err = tplClient.TemplateV1alpha1().VirtualMachineTemplates(metav1.NamespaceDefault).
				Create(context.Background(), newVirtualMachineTemplate(), metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			process.GetTemplateClient = origGetTemplateClient
		})

		DescribeTable("should print parameters when --print-params is set", func(
			positional bool,
			output string,
			unmarshalFn func([]byte, any) error,
		) {
			args := []string{
				setFlag(printParamsFlag, "true"),
				setFlag(outputFlag, output),
			}
			if positional {
				args = append(args, tpl.Name)
			} else {
				args = append(args, setFlag(nameFlag, tpl.Name))
			}
			out, err := runCmd(args...)
			Expect(err).ToNot(HaveOccurred())

			var params []v1alpha1.Parameter
			Expect(unmarshalFn(out, &params)).To(Succeed())
			Expect(params).To(Equal(tpl.Spec.Parameters))
		},
			Entry("when name is passed as positional arg and output format is JSON", true, formatJSON, json.Unmarshal),
			Entry("when name is passed as positional arg and output format is YAML", true, formatYAML, unmarshalYAML),
			Entry("when name is passed as flag and output format is JSON", false, formatJSON, json.Unmarshal),
			Entry("when name is passed as flag and output format is YAML", false, formatYAML, unmarshalYAML),
		)

		Context("should call subresource APIs", func() {
			const (
				subresourceCreate  = "create"
				subresourceProcess = "process"
			)

			var expectedSubresource string

			BeforeEach(func() {
				tplClient.PrependReactor("create", "virtualmachinetemplates", func(a k8stesting.Action) (bool, runtime.Object, error) {
					c, ok := a.(k8stesting.CreateActionImpl)
					Expect(ok).To(BeTrue())

					Expect(c.Name).To(Equal(tpl.Name))
					Expect(c.GetSubresource()).To(Equal(expectedSubresource))

					opts, ok := c.GetObject().(*subresourcesv1alpha1.ProcessOptions)
					Expect(ok).To(BeTrue())
					Expect(opts.Parameters).To(Equal(map[string]string{
						param1Name: param1Val,
						param2Name: param2Val,
					}))

					var err error
					tpl.Spec.Parameters, err = template.MergeParameters(tpl.Spec.Parameters, opts.Parameters)
					Expect(err).ToNot(HaveOccurred())
					vm, _, err := template.GetDefaultProcessor().Process(tpl)
					Expect(err).ToNot(HaveOccurred())

					return true, &subresourcesv1alpha1.ProcessedVirtualMachineTemplate{VirtualMachine: vm}, nil
				})
			})

			DescribeTable("should call subresource with parameters from flags", func(positional, create bool) {
				var notExpectedSubresource string
				if create {
					expectedSubresource = subresourceCreate
					notExpectedSubresource = subresourceProcess
				} else {
					expectedSubresource = subresourceProcess
					notExpectedSubresource = subresourceCreate
				}

				args := []string{
					setFlag(createFlag, strconv.FormatBool(create)),
					setParamFlag(param1Name, param1Val),
					setParamFlag(param2Name, param2Val),
				}
				if positional {
					args = append(args, tpl.Name)
				} else {
					args = append(args, setFlag(nameFlag, tpl.Name))
				}
				_, err := runCmd(args...)
				Expect(err).ToNot(HaveOccurred())

				Expect(kvtesting.FilterActions(&tplClient.Fake, "create", templateapi.PluralResourceName, expectedSubresource)).To(HaveLen(1))
				Expect(kvtesting.FilterActions(&tplClient.Fake, "create", templateapi.PluralResourceName, notExpectedSubresource)).To(BeEmpty())
			},
				Entry("call process when name is passed as positional arg", true, false),
				Entry("call process when name is passed as flag", false, false),
				Entry("call create when name is passed as positional arg", true, true),
				Entry("call create when name is passed as flag", false, true),
			)

			DescribeTable("should call subresource with parameters from file", func(positional, create bool, marshalFn func(any) []byte) {
				var notExpectedSubresource string
				if create {
					expectedSubresource = subresourceCreate
					notExpectedSubresource = subresourceProcess
				} else {
					expectedSubresource = subresourceProcess
					notExpectedSubresource = subresourceCreate
				}

				params := map[string]string{
					param1Name: param1Val,
					param2Name: param2Val,
				}
				paramsFile := filepath.Join(GinkgoT().TempDir(), "params")
				Expect(os.WriteFile(paramsFile, marshalFn(params), 0o600)).To(Succeed())

				args := []string{
					setFlag(createFlag, strconv.FormatBool(create)),
					setFlag(paramsFileFlag, paramsFile),
				}
				if positional {
					args = append(args, tpl.Name)
				} else {
					args = append(args, setFlag(nameFlag, tpl.Name))
				}
				_, err := runCmd(args...)
				Expect(err).ToNot(HaveOccurred())

				Expect(kvtesting.FilterActions(&tplClient.Fake, "create", templateapi.PluralResourceName, expectedSubresource)).To(HaveLen(1))
				Expect(kvtesting.FilterActions(&tplClient.Fake, "create", templateapi.PluralResourceName, notExpectedSubresource)).To(BeEmpty())
			},
				Entry("call process when name is passed as positional arg and input format is JSON", true, false, marshalJSON),
				Entry("call process when name is passed as positional arg and input format is YAML", true, false, marshalYAML),
				Entry("call process when name is passed as flag and input format is JSON", false, false, marshalJSON),
				Entry("call process when name is passed as flag and input format is YAML", false, false, marshalYAML),
				Entry("call create when name is passed as positional arg and input format is JSON", true, true, marshalJSON),
				Entry("call create when name is passed as positional arg and input format is YAML", true, true, marshalYAML),
				Entry("call create when name is passed as flag and input format is JSON", false, true, marshalJSON),
				Entry("call create when name is passed as flag and input format is YAML", false, true, marshalYAML),
			)
		})

		DescribeTable("should use local processing when --local flag is set", func(positional bool) {
			args := []string{
				setFlag(localFlag, "true"),
				setParamFlag(param1Name, param1Val),
				setParamFlag(param2Name, param2Val),
			}
			if positional {
				args = append(args, tpl.Name)
			} else {
				args = append(args, setFlag(nameFlag, tpl.Name))
			}
			out, err := runCmd(args...)
			Expect(err).ToNot(HaveOccurred())

			var vm virtv1.VirtualMachine
			Expect(yaml.Unmarshal(out, &vm)).To(Succeed())
			Expect(vm.Name).To(Equal(param1Val))
			Expect(vm.Spec.Preference.Name).To(Equal(param2Val))

			Expect(kvtesting.FilterActions(&tplClient.Fake, "create", templateapi.PluralResourceName, "process")).To(BeEmpty())
		},
			Entry("when name is passed as positional arg", true),
			Entry("when name is passed as flag", false),
		)

		DescribeTable("should fail when remote template does not exist", func(positional, local bool) {
			args := []string{
				setFlag(localFlag, strconv.FormatBool(local)),
			}
			if positional {
				args = append(args, "something")
			} else {
				args = append(args, setFlag(nameFlag, "something"))
			}
			_, err := runCmd(args...)
			Expect(err).To(MatchError(ContainSubstring("not found")))
		},
			Entry("when name is passed as positional arg and processing remotely", true, false),
			Entry("when name is passed as flag and processing remotely", false, false),
			Entry("when name is passed as positional arg and processing locally", true, true),
			Entry("when name is passed as flag and processing locally", false, true),
		)
	})

	Context("Error handling", func() {
		It("should fail when both --name and --file are provided", func() {
			_, err := runCmd(
				setFlag(nameFlag, "something"),
				setFlag(fileFlag, "something"),
			)
			Expect(err).To(MatchError("if any flags in the group [file name] are set none of the others can be; [file name] were all set"))
		})

		It("should fail when both --file and --local are provided", func() {
			_, err := runCmd(
				setFlag(fileFlag, "something"),
				setFlag(localFlag, "false"),
			)
			Expect(err).To(MatchError("if any flags in the group [file local] are set none of the others can be; [file local] were all set"))
		})

		It("should fail when both --param and --params-file are provided", func() {
			_, err := runCmd(
				setFlag(fileFlag, "something"),
				setFlag(paramsFileFlag, "something"),
				setParamFlag("something", "something"),
			)
			Expect(err).To(MatchError(
				"if any flags in the group [param params-file] are set none of the others can be; [param params-file] were all set",
			))
		})

		It("should fail when both --create and --file are provided", func() {
			_, err := runCmd(
				setFlag(createFlag, "true"),
				setFlag(fileFlag, "something"),
			)
			Expect(err).To(MatchError("if any flags in the group [create file] are set none of the others can be; [create file] were all set"))
		})

		It("should fail when both --create and --local are provided", func() {
			_, err := runCmd(
				setFlag(createFlag, "true"),
				setFlag(localFlag, "true"),
			)
			Expect(err).To(MatchError("if any flags in the group [create local] are set none of the others can be; [create local] were all set"))
		})

		It("should fail when both --create and --print-params are provided", func() {
			_, err := runCmd(
				setFlag(createFlag, "true"),
				setFlag(printParamsFlag, "true"),
			)
			Expect(err).To(MatchError(
				"if any flags in the group [create print-params] are set none of the others can be; [create print-params] were all set",
			))
		})

		It("should fail when positional arg and --file flag are both provided", func() {
			_, err := runCmd(
				"something",
				setFlag(fileFlag, "something"),
			)
			Expect(err).To(MatchError("only one of name or file can be provided"))
		})

		It("should fail when positional arg and --name flag are both provided", func() {
			_, err := runCmd(
				"something",
				setFlag(nameFlag, "something"),
			)
			Expect(err).To(MatchError("provide name either from positional argument or from flag"))
		})

		It("should fail when no template name or file provided", func() {
			_, err := runCmd()
			Expect(err).To(MatchError("name or file must be provided"))
		})

		It("should fail with invalid output format", func() {
			_, err := runCmd(
				setFlag(fileFlag, "something"),
				setFlag(outputFlag, "xml"),
			)
			Expect(err).To(MatchError("not supported output format: xml"))
		})

		It("should fail when template file does not exist", func() {
			_, err := runCmd(setFlag(fileFlag, filepath.Join(GinkgoT().TempDir(), "something")))
			Expect(err).To(MatchError(ContainSubstring("no such file or directory")))
		})
	})
})

func setFlag(flag, parameter string) string {
	return fmt.Sprintf("--%s=%s", flag, parameter)
}

func setParamFlag(k, v string) string {
	return fmt.Sprintf("--%s=%s=%s", paramFlag, k, v)
}

func runCmd(extraArgs ...string) ([]byte, error) {
	args := append([]string{"template", "process"}, extraArgs...)
	return testing.NewRepeatableVirtctlCommandWithOut(args...)()
}

func runCmdWithErr(extraArgs ...string) (out, errOut []byte, err error) {
	args := append([]string{"template", "process"}, extraArgs...)
	return testing.NewRepeatableVirtctlCommandWithOutAndErr(args...)()
}

func marshalYAML(obj any) []byte {
	data, err := yaml.Marshal(obj)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	return data
}

func marshalJSON(obj any) []byte {
	data, err := json.Marshal(obj)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	return data
}

func unmarshalYAML(data []byte, out any) error {
	return yaml.Unmarshal(data, out)
}

func newVirtualMachineTemplateWithSpec(spec *v1alpha1.VirtualMachineTemplateSpec) *v1alpha1.VirtualMachineTemplate {
	return &v1alpha1.VirtualMachineTemplate{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.GroupVersion.String(),
			Kind:       "VirtualMachineTemplate",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-template",
			Namespace: metav1.NamespaceDefault,
		},
		Spec: *spec,
	}
}

func newVirtualMachineTemplate() *v1alpha1.VirtualMachineTemplate {
	return newVirtualMachineTemplateWithSpec(
		&v1alpha1.VirtualMachineTemplateSpec{
			Parameters: []v1alpha1.Parameter{
				{
					Name: param1Name,
				},
				{
					Name: param2Name,
				},
			},
			VirtualMachine: &runtime.RawExtension{
				Object: &virtv1.VirtualMachine{
					TypeMeta: metav1.TypeMeta{
						APIVersion: virtv1.GroupVersion.String(),
						Kind:       "VirtualMachine",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: param1Placeholder,
					},
					Spec: virtv1.VirtualMachineSpec{
						Preference: &virtv1.PreferenceMatcher{
							Name: param2Placeholder,
						},
						Template: &virtv1.VirtualMachineInstanceTemplateSpec{
							Spec: virtv1.VirtualMachineInstanceSpec{
								Domain: virtv1.DomainSpec{},
							},
						},
					},
				},
			},
		},
	)
}
