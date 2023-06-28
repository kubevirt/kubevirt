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

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests/clientcmd"
)

var _ = Describe("Remove volume command", func() {
	var vmInterface *kubecli.MockVirtualMachineInterface
	var vmiInterface *kubecli.MockVirtualMachineInstanceInterface
	var ctrl *gomock.Controller

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		kubecli.GetKubevirtClientFromClientConfig = kubecli.GetMockKubevirtClientFromClientConfig
		kubecli.MockKubevirtClientInstance = kubecli.NewMockKubevirtClient(ctrl)
		vmInterface = kubecli.NewMockVirtualMachineInterface(ctrl)
		vmiInterface = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)
	})

	expectVMIEndpointRemoveVolumeError := func(vmiName, volumeName string) {
		kubecli.MockKubevirtClientInstance.
			EXPECT().
			VirtualMachineInstance(k8smetav1.NamespaceDefault).
			Return(vmiInterface).
			Times(1)
		vmiInterface.EXPECT().RemoveVolume(context.Background(), vmiName, gomock.Any()).DoAndReturn(func(ctx context.Context, arg0, arg1 interface{}) interface{} {
			Expect(arg1.(*v1.RemoveVolumeOptions).Name).To(Equal(volumeName))
			return fmt.Errorf("error removing")
		})
	}

	expectVMIEndpointRemoveVolume := func(vmiName, volumeName string, useDv bool) {
		kubecli.MockKubevirtClientInstance.
			EXPECT().
			VirtualMachineInstance(k8smetav1.NamespaceDefault).
			Return(vmiInterface).
			Times(1)
		vmiInterface.EXPECT().RemoveVolume(context.Background(), vmiName, gomock.Any()).DoAndReturn(func(ctx context.Context, arg0, arg1 interface{}) interface{} {
			Expect(arg1.(*v1.RemoveVolumeOptions).Name).To(Equal(volumeName))
			return nil
		})
	}

	expectVMEndpointRemoveVolume := func(vmiName, volumeName string, useDv bool) {
		kubecli.MockKubevirtClientInstance.
			EXPECT().
			VirtualMachine(k8smetav1.NamespaceDefault).
			Return(vmInterface).
			Times(1)
		vmInterface.EXPECT().RemoveVolume(context.Background(), vmiName, gomock.Any()).DoAndReturn(func(ctx context.Context, arg0, arg1 interface{}) interface{} {
			Expect(arg1.(*v1.RemoveVolumeOptions).Name).To(Equal(volumeName))
			return nil
		})
	}

	DescribeTable("should report error if call returns error according to option", func(isDryRun bool) {
		expectVMIEndpointRemoveVolumeError("testvmi", "testvolume")
		commandAndArgs := []string{"removevolume", "testvmi", "--volume-name=testvolume"}
		if isDryRun {
			commandAndArgs = append(commandAndArgs, "--dry-run")
		}
		cmdAdd := clientcmd.NewRepeatableVirtctlCommand(commandAndArgs...)
		err := cmdAdd()

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("error removing"))
	},
		Entry("with default", false),
		Entry("with dry-run arg", true),
	)

	DescribeTable("should fail with missing required or invalid parameters", func(errorString string, args ...string) {
		commandAndArgs := append([]string{"removevolume"}, args...)
		cmdAdd := clientcmd.NewRepeatableVirtctlCommand(commandAndArgs...)
		err := cmdAdd()

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(errorString))
	},
		Entry("removevolume no args", "argument validation failed"),
		Entry("removevolume name, missing required volume-name", "required flag(s)", "testvmi"),
		Entry("removevolume name, invalid extra parameter", "unknown flag", "testvmi", "--volume-name=blah", "--invalid=test"),
	)

	DescribeTable("should call correct endpoint", func(vmiName, volumeName string, useDv bool, expectFunc func(vmiName, volumeName string, useDv bool), args ...string) {
		expectFunc(vmiName, volumeName, useDv)
		commandAndArgs := append([]string{"removevolume", vmiName, fmt.Sprintf("--volume-name=%s", volumeName)}, args...)
		cmd := clientcmd.NewRepeatableVirtctlCommand(commandAndArgs...)

		Expect(cmd()).To(Succeed())
	},
		Entry("removevolume dv, no persist should call VMI endpoint", "testvmi", "testvolume", true, expectVMIEndpointRemoveVolume),
		Entry("removevolume pvc, no persist should call VMI endpoint", "testvmi", "testvolume", false, expectVMIEndpointRemoveVolume),
		Entry("removevolume dv, with persist should call VM endpoint", "testvmi", "testvolume", true, expectVMEndpointRemoveVolume, "--persist"),
		Entry("removevolume pvc, with persist should call VM endpoint", "testvmi", "testvolume", false, expectVMEndpointRemoveVolume, "--persist"),

		Entry("removevolume dv, no persist with dry-run should call VMI endpoint", "testvmi", "testvolume", true, expectVMIEndpointRemoveVolume, "--dry-run"),
		Entry("removevolume pvc, no persist with dry-run should call VMI endpoint", "testvmi", "testvolume", false, expectVMIEndpointRemoveVolume, "--dry-run"),
		Entry("removevolume dv, with persist with dry-run should call VM endpoint", "testvmi", "testvolume", true, expectVMEndpointRemoveVolume, "--persist", "--dry-run"),
		Entry("removevolume pvc, with persist with dry-run should call VM endpoint", "testvmi", "testvolume", false, expectVMEndpointRemoveVolume, "--persist", "--dry-run"),
	)
})
