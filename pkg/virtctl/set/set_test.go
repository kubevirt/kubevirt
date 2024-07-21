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
 * Copyright 2024 The KubeVirt Contributors
 *
 */

package set_test

import (
	"context"
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/golang/mock/gomock"

	"k8s.io/apimachinery/pkg/api/resource"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests/clientcmd"
)

var _ = Describe("Set command", func() {
	var vmInterface *kubecli.MockVirtualMachineInterface
	var ctrl *gomock.Controller
	const vmName = "testvm"
	const expectedCpuSocketsCount = 2

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		kubecli.GetKubevirtClientFromClientConfig = kubecli.GetMockKubevirtClientFromClientConfig
		kubecli.MockKubevirtClientInstance = kubecli.NewMockKubevirtClient(ctrl)
		vmInterface = kubecli.NewMockVirtualMachineInterface(ctrl)
		kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachine(k8smetav1.NamespaceDefault).Return(vmInterface).AnyTimes()
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("Input parameters validation", func() {
		It("should fail with missing input parameters", func() {
			cmd := clientcmd.NewRepeatableVirtctlCommand("set")
			Expect(cmd()).To(MatchError("argument validation failed"))
		})

		It("should fail with missing input parameters", func() {
			cmd := clientcmd.NewRepeatableVirtctlCommand("set", vmName)
			Expect(cmd()).To(MatchError("at least one of --cpu or --memory must be set"))
		})

		It("should fail with invalid CPU count", func() {
			cmd := clientcmd.NewRepeatableVirtctlCommand("set", vmName, "--cpu=invalid")
			Expect(cmd()).To(MatchError(ContainSubstring("invalid argument \"invalid\" for \"--cpu\" flag: strconv.ParseUint: parsing \"invalid\": invalid syntax")))
		})

		It("should fail with zero CPU count", func() {
			cmd := clientcmd.NewRepeatableVirtctlCommand("set", vmName, "--cpu=0")
			Expect(cmd()).To(MatchError(ContainSubstring("invalid CPU count: 0; must be greater than 0")))
		})

		It("should fail with negative CPU count", func() {
			cmd := clientcmd.NewRepeatableVirtctlCommand("set", vmName, "--cpu=-1")
			Expect(cmd()).To(MatchError(ContainSubstring("invalid argument \"-1\" for \"--cpu\" flag: strconv.ParseUint: parsing \"-1\": invalid syntax")))
		})

		It("should fail with invalid memory size", func() {
			cmd := clientcmd.NewRepeatableVirtctlCommand("set", vmName, "--memory=invalidSize")
			Expect(cmd()).To(MatchError(ContainSubstring("invalid memory size: invalidSize")))
		})

		It("should fail with zero memory size", func() {
			cmd := clientcmd.NewRepeatableVirtctlCommand("set", vmName, "--memory=0")
			Expect(cmd()).To(MatchError(ContainSubstring("memory size must be greater than zero")))
		})

		It("should fail with negative memory size", func() {
			cmd := clientcmd.NewRepeatableVirtctlCommand("set", vmName, "--memory=-1Gi")
			Expect(cmd()).To(MatchError(ContainSubstring("memory size must be greater than zero")))
		})

		It("should fail with invalid timeout flag", func() {
			cmd := clientcmd.NewRepeatableVirtctlCommand("set", vmName, "--cpu=2", "--timeout=invalid")
			Expect(cmd()).To(MatchError(ContainSubstring("invalid argument \"invalid\" for \"--timeout\" flag")))
		})
	})

	Context("Patch CPU and Memory", func() {
		It("should succeed with valid CPU", func() {
			vm := kubecli.NewMinimalVM(vmName)
			vm.ObjectMeta.Namespace = k8smetav1.NamespaceDefault
			vm.Spec.Template = &v1.VirtualMachineInstanceTemplateSpec{
				Spec: v1.VirtualMachineInstanceSpec{},
			}
			vmInterface.EXPECT().Get(gomock.Any(), vmName, k8smetav1.GetOptions{}).Return(vm, nil).Times(1)
			vmInterface.EXPECT().Patch(gomock.Any(), vmName, types.JSONPatchType, gomock.Any(), k8smetav1.PatchOptions{}, gomock.Any()).DoAndReturn(
				func(ctx context.Context, name string, pt types.PatchType, data []byte, options k8smetav1.PatchOptions, subresources ...string) (*v1.VirtualMachine, error) {
					patches := []map[string]interface{}{}
					json.Unmarshal(data, &patches)
					for _, patch := range patches {
						if patch["op"] == "replace" && patch["path"] == "/spec/template/spec/domain/cpu/sockets" {
							if vm.Spec.Template.Spec.Domain.CPU == nil {
								vm.Spec.Template.Spec.Domain.CPU = &v1.CPU{}
							}
							vm.Spec.Template.Spec.Domain.CPU.Sockets = expectedCpuSocketsCount
						}
					}
					return vm, nil
				}).Times(1)

			cmd := clientcmd.NewRepeatableVirtctlCommand("set", vmName, "--cpu=2")
			Expect(cmd()).To(Succeed())
		})

		It("should succeed with valid Memory", func() {
			vm := kubecli.NewMinimalVM(vmName)
			vm.ObjectMeta.Namespace = k8smetav1.NamespaceDefault
			vm.Spec.Template = &v1.VirtualMachineInstanceTemplateSpec{}
			vmInterface.EXPECT().Get(gomock.Any(), vmName, k8smetav1.GetOptions{}).Return(vm, nil).Times(1)
			vmInterface.EXPECT().Patch(gomock.Any(), vmName, types.JSONPatchType, gomock.Any(), k8smetav1.PatchOptions{}, gomock.Any()).DoAndReturn(
				func(ctx context.Context, name string, pt types.PatchType, data []byte, options k8smetav1.PatchOptions, subresources ...string) (*v1.VirtualMachine, error) {
					patches := []map[string]interface{}{}
					json.Unmarshal(data, &patches)
					for _, patch := range patches {
						if patch["op"] == "replace" && patch["path"] == "/spec/template/spec/domain/memory" {
							if vm.Spec.Template.Spec.Domain.Memory == nil {
								vm.Spec.Template.Spec.Domain.Memory = &v1.Memory{}
							}
							expectedMemorySize := resource.MustParse("2048Mi")
							vm.Spec.Template.Spec.Domain.Memory.Guest = &expectedMemorySize
						}
					}
					return vm, nil
				}).Times(1)

			cmd := clientcmd.NewRepeatableVirtctlCommand("set", vmName, "--memory=2048Mi")
			Expect(cmd()).To(Succeed())
		})

		It("should succeed with valid CPU and Memory with timeout flag", func() {
			vm := kubecli.NewMinimalVM(vmName)
			vm.ObjectMeta.Namespace = k8smetav1.NamespaceDefault
			vm.Spec.Template = &v1.VirtualMachineInstanceTemplateSpec{
				Spec: v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{},
				},
			}
			vmInterface.EXPECT().Get(gomock.Any(), vmName, k8smetav1.GetOptions{}).Return(vm, nil).Times(1)
			vmInterface.EXPECT().Patch(gomock.Any(), vmName, types.JSONPatchType, gomock.Any(), k8smetav1.PatchOptions{}, gomock.Any()).DoAndReturn(
				func(ctx context.Context, name string, pt types.PatchType, data []byte, options k8smetav1.PatchOptions, subresources ...string) (*v1.VirtualMachine, error) {
					patches := []map[string]interface{}{}
					json.Unmarshal(data, &patches)
					for _, patch := range patches {
						if patch["op"] == "replace" && patch["path"] == "/spec/template/spec/domain/cpu/sockets" {
							if vm.Spec.Template.Spec.Domain.CPU == nil {
								vm.Spec.Template.Spec.Domain.CPU = &v1.CPU{}
							}
							vm.Spec.Template.Spec.Domain.CPU.Sockets = expectedCpuSocketsCount
						}
						if patch["op"] == "replace" && patch["path"] == "/spec/template/spec/domain/memory" {
							if vm.Spec.Template.Spec.Domain.Memory == nil {
								vm.Spec.Template.Spec.Domain.Memory = &v1.Memory{}
							}
							expectedMemorySize := resource.MustParse("2048Mi")
							vm.Spec.Template.Spec.Domain.Memory.Guest = &expectedMemorySize
						}
					}
					return vm, nil
				}).Times(1)

			cmd := clientcmd.NewRepeatableVirtctlCommand("set", vmName, "--cpu=2", "--memory=2048Mi", "--timeout=2m")
			Expect(cmd()).To(Succeed())
		})
	})
})
