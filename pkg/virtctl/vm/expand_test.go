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

package vm_test

import (
	"context"
	"errors"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	kubevirtfake "kubevirt.io/client-go/kubevirt/fake"
	kvtesting "kubevirt.io/client-go/testing"

	"kubevirt.io/kubevirt/pkg/virtctl/testing"
	virtctl "kubevirt.io/kubevirt/pkg/virtctl/vm"
)

var _ = Describe("Expand command", func() {
	const (
		vmName = "testvm"
	)

	var ctrl *gomock.Controller
	var virtClient *kubevirtfake.Clientset

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		kubecli.GetKubevirtClientFromClientConfig = kubecli.GetMockKubevirtClientFromClientConfig
		kubecli.MockKubevirtClientInstance = kubecli.NewMockKubevirtClient(ctrl)
		virtClient = kubevirtfake.NewSimpleClientset()
	})

	It("should fail with missing input parameters", func() {
		cmd := testing.NewRepeatableVirtctlCommand(virtctl.COMMAND_EXPAND)
		Expect(cmd()).Should(MatchError("error invalid arguments - VirtualMachine name or file must be provided"))
	})

	It("should fail when called with non supported output format", func() {
		cmd := testing.NewRepeatableVirtctlCommand(virtctl.COMMAND_EXPAND, "--vm", vmName, "--output", "test-format")
		Expect(cmd()).Should(MatchError("error not supported output format defined: test-format"))
	})

	It("should fail when called on non existing vm", func() {
		kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachine(k8smetav1.NamespaceDefault).Return(virtClient.KubevirtV1().VirtualMachines(k8smetav1.NamespaceDefault)).Times(1)
		cmd := testing.NewRepeatableVirtctlCommand(virtctl.COMMAND_EXPAND, "--vm", "non-existing-vm")
		Expect(cmd()).Should(MatchError("error expanding VirtualMachine - non-existing-vm in namespace - default: virtualmachines.kubevirt.io \"non-existing-vm\" not found"))
	})

	It("should fail when input file does not exist", func() {
		cmd := testing.NewRepeatableVirtctlCommand(virtctl.COMMAND_EXPAND, "--file", "invalid/path")
		Expect(cmd()).To(MatchError("error reading file open invalid/path: no such file or directory"))
	})

	DescribeTable("should succeed when called on vm with", func(extraArgs []string) {
		kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachine(k8smetav1.NamespaceDefault).Return(virtClient.KubevirtV1().VirtualMachines(k8smetav1.NamespaceDefault)).Times(1)
		_, err := virtClient.KubevirtV1().VirtualMachines(k8smetav1.NamespaceDefault).Create(context.Background(), kubecli.NewMinimalVM(vmName), k8smetav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		args := append([]string{virtctl.COMMAND_EXPAND, "--vm", vmName}, extraArgs...)
		cmd := testing.NewRepeatableVirtctlCommand(args...)

		Expect(cmd()).To(Succeed())
		Expect(kvtesting.FilterActions(&virtClient.Fake, "get", "virtualmachines", "expand-spec")).To(HaveLen(1))
	},
		Entry("implicit default format", nil),
		Entry("supported format json", []string{"--output", "json"}),
		Entry("supported format yaml", []string{"--output", "yaml"}),
	)

	Context("expanding a vm spec", func() {
		const invalidYaml = `apiVersion: kubevirt.io/v1kind: VirtualMachine`

		var expandSpecInterface *kubecli.MockExpandSpecInterface
		var vm *v1.VirtualMachine
		var vmSpec []byte

		BeforeEach(func() {
			expandSpecInterface = kubecli.NewMockExpandSpecInterface(ctrl)
			vm = kubecli.NewMinimalVM(vmName)

			var err error
			vmSpec, err = yaml.Marshal(vm)
			Expect(err).ToNot(HaveOccurred())
		})

		Context("with file input", func() {
			var file *os.File

			BeforeEach(func() {
				var err error
				file, err = os.CreateTemp(GinkgoT().TempDir(), "file-*")
				Expect(err).ToNot(HaveOccurred())
			})

			Context("with valid file input", func() {
				BeforeEach(func() {
					_, err := file.Write(vmSpec)
					Expect(err).ToNot(HaveOccurred())
					err = file.Close()
					Expect(err).ToNot(HaveOccurred())
				})

				It("should expand vm spec", func() {
					kubecli.MockKubevirtClientInstance.EXPECT().ExpandSpec(k8smetav1.NamespaceDefault).Return(expandSpecInterface).Times(1)
					expandSpecInterface.EXPECT().ForVirtualMachine(vm).Return(vm, nil).Times(1)
					cmd := testing.NewRepeatableVirtctlCommand(virtctl.COMMAND_EXPAND, "--file", file.Name())
					Expect(cmd()).ToNot(HaveOccurred())
				})

				It("should handle error returned by API", func() {
					kubecli.MockKubevirtClientInstance.EXPECT().ExpandSpec(k8smetav1.NamespaceDefault).Return(expandSpecInterface).Times(1)
					expandSpecInterface.EXPECT().ForVirtualMachine(vm).Return(nil, errors.New("error expanding vm")).Times(1)
					cmd := testing.NewRepeatableVirtctlCommand(virtctl.COMMAND_EXPAND, "--file", file.Name())
					Expect(cmd()).Should(MatchError("error expanding VirtualMachine - testvm in namespace - default: error expanding vm"))
				})
			})

			It("should fail when called with invalid yaml in file input", func() {
				_, err := file.Write([]byte(invalidYaml))
				Expect(err).ToNot(HaveOccurred())
				err = file.Close()
				Expect(err).ToNot(HaveOccurred())
				cmd := testing.NewRepeatableVirtctlCommand(virtctl.COMMAND_EXPAND, "--file", file.Name())
				Expect(cmd()).Should(MatchError("error decoding VirtualMachine error converting YAML to JSON: yaml: mapping values are not allowed in this context"))
			})
		})

		Context("with stdin input", func() {
			var (
				oldStdin *os.File
				r        *os.File
				w        *os.File
			)

			writeToStdin := func(data []byte) {
				go func() {
					defer w.Close()
					_, _ = w.Write(data)
				}()
			}

			BeforeEach(func() {
				var err error
				r, w, err = os.Pipe()
				Expect(err).ToNot(HaveOccurred())

				oldStdin = os.Stdin
				os.Stdin = r

				DeferCleanup(func() {
					os.Stdin = oldStdin
					r.Close()
				})
			})

			It("should succeed when called with valid stdin input", func() {
				kubecli.MockKubevirtClientInstance.EXPECT().ExpandSpec(k8smetav1.NamespaceDefault).Return(expandSpecInterface).Times(1)
				expandSpecInterface.EXPECT().ForVirtualMachine(vm).Return(vm, nil).Times(1)

				writeToStdin(vmSpec)
				cmd := testing.NewRepeatableVirtctlCommand(virtctl.COMMAND_EXPAND, "--file", "-")
				Expect(cmd()).To(Succeed())
			})

			It("should fail when called with invalid yaml in stdin input", func() {
				writeToStdin([]byte(invalidYaml))
				cmd := testing.NewRepeatableVirtctlCommand(virtctl.COMMAND_EXPAND, "--file", "-")
				Expect(cmd()).To(MatchError("error decoding VirtualMachine error converting YAML to JSON: yaml: mapping values are not allowed in this context"))
			})
		})
	})
})
