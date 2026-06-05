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

package convert_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	ocpv1 "github.com/openshift/api/template/v1"
	ocptemplateclient "github.com/openshift/client-go/template/clientset/versioned"
	ocptemplatefake "github.com/openshift/client-go/template/clientset/versioned/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/yaml"

	virtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/virt-template-api/core/v1alpha1"

	"kubevirt.io/kubevirt/pkg/virtctl/template/convert"
	"kubevirt.io/kubevirt/pkg/virtctl/testing"
)

const (
	nameFlag   = "name"
	fileFlag   = "file"
	outputFlag = "output"

	formatYAML = "yaml"
	formatJSON = "json"
)

var _ = Describe("Convert command", func() {
	Context("With file input", func() {
		var file string

		BeforeEach(func() {
			file = filepath.Join(GinkgoT().TempDir(), "template")
		})

		DescribeTable("should convert OpenShift Template to VirtualMachineTemplate", func(
			marshalFn func(any) []byte,
			unmarshalFn func([]byte, any) error,
			output string,
		) {
			ocpTpl := newOpenShiftTemplate()
			Expect(os.WriteFile(file, marshalFn(ocpTpl), 0o600)).To(Succeed())

			out, err := runCmd(
				setFlag(fileFlag, file),
				setFlag(outputFlag, output),
			)
			Expect(err).ToNot(HaveOccurred())

			var tpl v1alpha1.VirtualMachineTemplate
			Expect(unmarshalFn(out, &tpl)).To(Succeed())

			verifyConversion(&tpl, ocpTpl)
		},
			Entry("when template is JSON and output format is JSON", marshalJSON, json.Unmarshal, formatJSON),
			Entry("when template is JSON and output format is YAML", marshalJSON, unmarshalYAML, formatYAML),
			Entry("when template is YAML and output format is JSON", marshalYAML, json.Unmarshal, formatJSON),
			Entry("when template is YAML and output format is YAML", marshalYAML, unmarshalYAML, formatYAML),
		)

		It("should fail when template file is in invalid format", func() {
			Expect(os.WriteFile(file, []byte("invalid: yaml: content: ["), 0o600)).To(Succeed())

			_, err := runCmd(setFlag(fileFlag, file))
			Expect(err).To(MatchError(ContainSubstring("error decoding Template")))
		})

		It("should fail when template has no objects", func() {
			tpl := newOpenShiftTemplate()
			tpl.Objects = []runtime.RawExtension{}
			Expect(os.WriteFile(file, marshalYAML(tpl), 0o600)).To(Succeed())

			_, err := runCmd(setFlag(fileFlag, file))
			Expect(err).To(MatchError("template must contain exactly one object, found 0"))
		})

		It("should fail when template has multiple objects", func() {
			tpl := newOpenShiftTemplate()
			tpl.Objects = append(tpl.Objects, runtime.RawExtension{
				Object: &virtv1.VirtualMachine{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "kubevirt.io/v1",
						Kind:       "VirtualMachine",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "another-vm",
					},
				},
			})
			Expect(os.WriteFile(file, marshalYAML(tpl), 0o600)).To(Succeed())

			_, err := runCmd(setFlag(fileFlag, file))
			Expect(err).To(MatchError("template must contain exactly one object, found 2"))
		})
	})

	Context("With remote template", func() {
		var (
			ocpClient             *ocptemplatefake.Clientset
			origGetTemplateClient func(*rest.Config) (ocptemplateclient.Interface, error)
			ocpTpl                *ocpv1.Template
		)

		BeforeEach(func() {
			ocpClient = ocptemplatefake.NewSimpleClientset()
			origGetTemplateClient = convert.GetTemplateClient
			convert.GetTemplateClient = func(_ *rest.Config) (ocptemplateclient.Interface, error) {
				return ocpClient, nil
			}

			var err error
			ocpTpl, err = ocpClient.TemplateV1().Templates(metav1.NamespaceDefault).
				Create(context.Background(), newOpenShiftTemplate(), metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			convert.GetTemplateClient = origGetTemplateClient
		})

		DescribeTable("should convert remote OpenShift Template", func(
			positional bool,
			output string,
			unmarshalFn func([]byte, any) error,
		) {
			args := []string{
				setFlag(outputFlag, output),
			}
			if positional {
				args = append(args, ocpTpl.Name)
			} else {
				args = append(args, setFlag(nameFlag, ocpTpl.Name))
			}
			out, err := runCmd(args...)
			Expect(err).ToNot(HaveOccurred())

			var tpl v1alpha1.VirtualMachineTemplate
			Expect(unmarshalFn(out, &tpl)).To(Succeed())

			verifyConversion(&tpl, ocpTpl)
		},
			Entry("when name is passed as positional arg and output format is JSON", true, formatJSON, json.Unmarshal),
			Entry("when name is passed as positional arg and output format is YAML", true, formatYAML, unmarshalYAML),
			Entry("when name is passed as flag and output format is JSON", false, formatJSON, json.Unmarshal),
			Entry("when name is passed as flag and output format is YAML", false, formatYAML, unmarshalYAML),
		)

		DescribeTable("should fail when remote template does not exist", func(positional bool) {
			args := []string{}
			if positional {
				args = append(args, "something")
			} else {
				args = append(args, setFlag(nameFlag, "something"))
			}
			_, err := runCmd(args...)
			Expect(err).To(MatchError(ContainSubstring("not found")))
		},
			Entry("when name is passed as positional arg", true),
			Entry("when name is passed as flag", false),
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

func runCmd(extraArgs ...string) ([]byte, error) {
	args := append([]string{"template", "convert"}, extraArgs...)
	return testing.NewRepeatableVirtctlCommandWithOut(args...)()
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

func newOpenShiftTemplate() *ocpv1.Template {
	return &ocpv1.Template{
		TypeMeta: metav1.TypeMeta{
			APIVersion: ocpv1.GroupVersion.String(),
			Kind:       "Template",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-template",
			Namespace: metav1.NamespaceDefault,
			Labels: map[string]string{
				"test-label": "test-value",
			},
			Annotations: map[string]string{
				"test-annotation": "test-value",
			},
		},
		Message: "This is a test template",
		Parameters: []ocpv1.Parameter{
			{
				Name:        "NAME",
				DisplayName: "VM Name",
				Description: "The name of the VM",
				Required:    true,
			},
			{
				Name:        "PREFERENCE",
				DisplayName: "VM Preference",
				Description: "The preference for the VM",
				Required:    false,
			},
		},
		Objects: []runtime.RawExtension{
			{
				Object: &virtv1.VirtualMachine{
					TypeMeta: metav1.TypeMeta{
						APIVersion: virtv1.GroupVersion.String(),
						Kind:       "VirtualMachine",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "${NAME}",
					},
					Spec: virtv1.VirtualMachineSpec{
						Preference: &virtv1.PreferenceMatcher{
							Name: "${PREFERENCE}",
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
	}
}

func verifyConversion(tpl *v1alpha1.VirtualMachineTemplate, ocpTpl *ocpv1.Template) {
	// Verify metadata is preserved
	Expect(tpl.Name).To(Equal(ocpTpl.Name))
	Expect(tpl.Namespace).To(Equal(ocpTpl.Namespace))
	Expect(tpl.Labels).To(Equal(ocpTpl.Labels))
	Expect(tpl.Annotations).To(Equal(ocpTpl.Annotations))

	// Verify spec fields
	Expect(tpl.Spec.Message).To(Equal(ocpTpl.Message))
	Expect(tpl.Spec.Parameters).To(HaveLen(len(ocpTpl.Parameters)))

	// Verify parameters are converted correctly
	for i, param := range ocpTpl.Parameters {
		Expect(tpl.Spec.Parameters[i].Name).To(Equal(param.Name))
		Expect(tpl.Spec.Parameters[i].DisplayName).To(Equal(param.DisplayName))
		Expect(tpl.Spec.Parameters[i].Description).To(Equal(param.Description))
		Expect(tpl.Spec.Parameters[i].Value).To(Equal(param.Value))
		Expect(tpl.Spec.Parameters[i].Generate).To(Equal(param.Generate))
		Expect(tpl.Spec.Parameters[i].From).To(Equal(param.From))
		Expect(tpl.Spec.Parameters[i].Required).To(Equal(param.Required))
	}

	// Verify VirtualMachine was converted correctly
	// Marshal to JSON first to allow comparing runtime.RawExtension Raw and Object
	tplData, err := json.Marshal(tpl.Spec.VirtualMachine)
	Expect(err).ToNot(HaveOccurred())
	Expect(ocpTpl.Objects).To(HaveLen(1))
	ocpTplData, err := json.Marshal(ocpTpl.Objects[0])
	Expect(err).ToNot(HaveOccurred())
	Expect(tplData).To(MatchJSON(ocpTplData))
}
