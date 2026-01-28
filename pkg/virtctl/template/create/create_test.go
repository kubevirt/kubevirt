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

package create_test

import (
	"context"
	"fmt"
	"regexp"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/rand"
	k8stesting "k8s.io/client-go/testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"

	templateclient "kubevirt.io/virt-template-client-go/virttemplate"
	templatefake "kubevirt.io/virt-template-client-go/virttemplate/fake"

	"kubevirt.io/kubevirt/pkg/virtctl/template/create"
	"kubevirt.io/kubevirt/pkg/virtctl/testing"
)

const (
	vmNameFlag      = "vm-name"
	vmNamespaceFlag = "vm-namespace"
	nameFlag        = "name"
)

var nameRegex = regexp.MustCompile(`virtualmachinetemplaterequest\.template\.kubevirt\.io/(\S+) created`)

var _ = Describe("Create command", func() {
	const (
		testVMName      = "test-vm"
		testVMNamespace = "test-namespace"
		testName        = "test-template"
	)

	var (
		tplClient             *templatefake.Clientset
		origGetTemplateClient func(*rest.Config) (templateclient.Interface, error)
	)

	BeforeEach(func() {
		tplClient = templatefake.NewSimpleClientset()
		tplClient.PrependReactor("create", "*", generateName)
		origGetTemplateClient = create.GetTemplateClient
		create.GetTemplateClient = func(_ *rest.Config) (templateclient.Interface, error) {
			return tplClient, nil
		}
	})

	AfterEach(func() {
		create.GetTemplateClient = origGetTemplateClient
	})

	Context("Creating VirtualMachineTemplateRequest", func() {
		It("should create a VirtualMachineTemplateRequest with vm-name only", func() {
			out, err := runCmd(setFlag(vmNameFlag, testVMName))
			Expect(err).ToNot(HaveOccurred())

			name := parseName(out)
			tplReq, err := tplClient.TemplateV1alpha1().VirtualMachineTemplateRequests(metav1.NamespaceDefault).
				Get(context.Background(), name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			Expect(tplReq.Spec.VirtualMachineRef.Name).To(Equal(testVMName))
			Expect(tplReq.Spec.VirtualMachineRef.Namespace).To(Equal(metav1.NamespaceDefault))
		})

		It("should create a VirtualMachineTemplateRequest with vm-name and vm-namespace", func() {
			out, err := runCmd(
				setFlag(vmNameFlag, testVMName),
				setFlag(vmNamespaceFlag, testVMNamespace),
			)
			Expect(err).ToNot(HaveOccurred())

			name := parseName(out)
			tplReq, err := tplClient.TemplateV1alpha1().VirtualMachineTemplateRequests(metav1.NamespaceDefault).
				Get(context.Background(), name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			Expect(tplReq.Spec.VirtualMachineRef.Name).To(Equal(testVMName))
			Expect(tplReq.Spec.VirtualMachineRef.Namespace).To(Equal(testVMNamespace))
		})

		It("should set template name when --name flag is provided", func() {
			out, err := runCmd(
				setFlag(vmNameFlag, testVMName),
				setFlag(nameFlag, testName),
			)
			Expect(err).ToNot(HaveOccurred())

			tplReqName := parseName(out)
			tplReq, err := tplClient.TemplateV1alpha1().VirtualMachineTemplateRequests(metav1.NamespaceDefault).
				Get(context.Background(), tplReqName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			Expect(tplReq.Spec.TemplateName).To(Equal(testName))
		})
	})

	Context("Error handling", func() {
		It("should fail when --vm-name is not provided", func() {
			_, err := runCmd()
			Expect(err).To(MatchError(ContainSubstring("required flag(s) \"vm-name\" not set")))
		})

		It("should fail when positional arguments are provided", func() {
			_, err := runCmd("unexpected-arg", setFlag(vmNameFlag, testVMName))
			Expect(err).To(MatchError(ContainSubstring("unknown command")))
		})
	})
})

func parseName(out []byte) string {
	matches := nameRegex.FindSubmatch(out)
	ExpectWithOffset(1, matches).To(HaveLen(2), "expected output to match pattern: %s", string(out))
	return string(matches[1])
}

func setFlag(flag, parameter string) string {
	return fmt.Sprintf("--%s=%s", flag, parameter)
}

func runCmd(extraArgs ...string) ([]byte, error) {
	args := append([]string{"template", "create"}, extraArgs...)
	return testing.NewRepeatableVirtctlCommandWithOut(args...)()
}

func generateName(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
	createAction, ok := action.(k8stesting.CreateAction)
	if !ok {
		return false, nil, fmt.Errorf("not a CreateAction")
	}
	obj := createAction.GetObject()

	metaObj, ok := obj.(metav1.Object)
	if !ok {
		return false, nil, fmt.Errorf("object is not a metav1.Object")
	}

	if metaObj.GetName() == "" && metaObj.GetGenerateName() != "" {
		// Simulating k8s behavior: append random string to prefix
		generatedName := metaObj.GetGenerateName() + rand.String(5)
		metaObj.SetName(generatedName)
		metaObj.SetGenerateName("")
	}

	return false, obj, nil
}
