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
 * Copyright 2023 Red Hat, Inc.
 *
 */

package vm_test

import (
	"context"
	"fmt"
	"os"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests/clientcmd"
)

const (
	fileInput     = "--file"
	outputFormat  = "--output"
	nonExistingVM = "non-existing-vm"
	invalidFormat = "test-format"
)

var _ = Describe("Expand command", func() {
	var vm *v1.VirtualMachine
	var vmInterface *kubecli.MockVirtualMachineInterface
	var expandSpecInterface *kubecli.MockExpandSpecInterface
	var ctrl *gomock.Controller
	const vmName = "testvm"

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		kubecli.GetKubevirtClientFromClientConfig = kubecli.GetMockKubevirtClientFromClientConfig
		kubecli.MockKubevirtClientInstance = kubecli.NewMockKubevirtClient(ctrl)
		expandSpecInterface = kubecli.NewMockExpandSpecInterface(ctrl)
		vmInterface = kubecli.NewMockVirtualMachineInterface(ctrl)
		vm = kubecli.NewMinimalVM(vmName)
	})

	It("should fail with missing input parameters", func() {
		cmd := clientcmd.NewRepeatableVirtctlCommand("expand")
		err := cmd()
		Expect(err).To(HaveOccurred())
		Expect(err).Should(MatchError("error invalid arguments - VirtualMachine name or file must be provided"))
	})

	It("should succeed when called on vm", func() {
		kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachine(k8smetav1.NamespaceDefault).Return(vmInterface).Times(1)
		vmInterface.EXPECT().GetWithExpandedSpec(context.Background(), vmName)

		cmd := clientcmd.NewRepeatableVirtctlCommand("expand", "--vm", vmName)
		Expect(cmd()).To(Succeed())
	})

	DescribeTable("should succeed when called with", func(formatName string) {
		kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachine(k8smetav1.NamespaceDefault).Return(vmInterface).Times(1)
		vmInterface.EXPECT().GetWithExpandedSpec(context.Background(), vmName)

		cmd := clientcmd.NewRepeatableVirtctlCommand("expand", outputFormat, formatName, "--vm", vmName)
		Expect(cmd()).To(Succeed())
	},
		Entry("supported format json", "json"),
		Entry("supported format yaml", "yaml"),
	)

	It("should fail when called on non existing vm", func() {
		kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachine(k8smetav1.NamespaceDefault).Return(vmInterface).Times(1)
		vmInterface.EXPECT().GetWithExpandedSpec(context.Background(), nonExistingVM).Return(nil, fmt.Errorf("\"%s\" not found", nonExistingVM))

		cmd := clientcmd.NewRepeatableVirtctlCommand("expand", "--vm", nonExistingVM)
		err := cmd()

		Expect(err).To(HaveOccurred())
		Expect(err).Should(MatchError("error expanding VirtualMachine - non-existing-vm in namespace - default: \"non-existing-vm\" not found"))
	})

	It("should fail when called with non supported output format", func() {
		cmd := clientcmd.NewRepeatableVirtctlCommand("expand", outputFormat, invalidFormat, "--vm", vmName)
		err := cmd()

		Expect(err).To(HaveOccurred())
		Expect(err).Should(MatchError("error not supported output format defined: test-format"))
	})

	Context("with file input", func() {
		var file *os.File

		const (
			vmSpec = `apiVersion: kubevirt.io/v1
kind: VirtualMachine
metadata:
  name: testvm
spec:
  runStrategy: Always
  template:
    spec:
      domain:
        devices: {}
        machine:
          type: q35
        resources: {}
        volumes:
status:
`
			invalidYaml = `apiVersion: kubevirt.io/v1kind: VirtualMachine`
		)
		BeforeEach(func() {
			var err error
			file, err = os.CreateTemp(GinkgoT().TempDir(), "file-*")
			Expect(err).ToNot(HaveOccurred())
		})

		It("should succeed when called with valid file input", func() {
			Expect(os.WriteFile(file.Name(), []byte(vmSpec), 0666)).To(Succeed())
			Expect(yaml.Unmarshal([]byte(vmSpec), vm)).To(Succeed())

			kubecli.MockKubevirtClientInstance.EXPECT().ExpandSpec(k8smetav1.NamespaceDefault).Return(expandSpecInterface).Times(1)
			expandSpecInterface.EXPECT().ForVirtualMachine(vm).Return(vm, nil).Times(1)

			cmd := clientcmd.NewRepeatableVirtctlCommand("expand", fileInput, file.Name())
			err := cmd()

			Expect(err).ToNot(HaveOccurred())
		})

		It("should fail when called with invalid vm defined in file input", func() {
			Expect(os.WriteFile(file.Name(), []byte(vmSpec), 0666)).To(Succeed())
			Expect(yaml.Unmarshal([]byte(vmSpec), vm)).To(Succeed())

			kubecli.MockKubevirtClientInstance.EXPECT().ExpandSpec(k8smetav1.NamespaceDefault).Return(expandSpecInterface).Times(1)
			expandSpecInterface.EXPECT().ForVirtualMachine(vm).Return(nil, fmt.Errorf("error expanding vm from file")).Times(1)

			cmd := clientcmd.NewRepeatableVirtctlCommand("expand", fileInput, file.Name())
			err := cmd()

			Expect(err).To(HaveOccurred())
			Expect(err).Should(MatchError("error expanding VirtualMachine - testvm in namespace - default: error expanding vm from file"))
		})

		It("should fail when called with invalid yaml in file input", func() {
			Expect(os.WriteFile(file.Name(), []byte(invalidYaml), 0666)).To(Succeed())
			Expect(yaml.Unmarshal([]byte(invalidYaml), vm)).ToNot(Succeed())

			cmd := clientcmd.NewRepeatableVirtctlCommand("expand", fileInput, file.Name())
			err := cmd()

			Expect(err).To(HaveOccurred())
			Expect(err).Should(MatchError("error decoding VirtualMachine error converting YAML to JSON: yaml: mapping values are not allowed in this context"))
		})
	})
})
