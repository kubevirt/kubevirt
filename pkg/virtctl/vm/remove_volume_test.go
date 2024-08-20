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
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/golang/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests/clientcmd"
)

var _ = Describe("Remove volume command", func() {
	const (
		vmiName    = "testvmi"
		volumeName = "testvolume"
	)

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

	expectVMIEndpointRemoveVolumeError := func() {
		kubecli.MockKubevirtClientInstance.
			EXPECT().
			VirtualMachineInstance(metav1.NamespaceDefault).
			Return(vmiInterface).
			Times(1)
		vmiInterface.EXPECT().RemoveVolume(context.Background(), vmiName, gomock.Any()).DoAndReturn(func(_ context.Context, _, arg1 interface{}) interface{} {
			return errors.New("error removing")
		})
	}

	expectVMEndpointRemoveVolumeError := func() {
		kubecli.MockKubevirtClientInstance.
			EXPECT().
			VirtualMachine(metav1.NamespaceDefault).
			Return(vmInterface).
			Times(1)
		vmInterface.EXPECT().RemoveVolume(context.Background(), vmiName, gomock.Any()).DoAndReturn(func(_ context.Context, _, arg1 interface{}) interface{} {
			return errors.New("error removing")
		})
	}

	expectVMIEndpointRemoveVolume := func(dryRun bool) func() {
		return func() {
			kubecli.MockKubevirtClientInstance.
				EXPECT().
				VirtualMachineInstance(metav1.NamespaceDefault).
				Return(vmiInterface).
				Times(1)
			vmiInterface.EXPECT().RemoveVolume(context.Background(), vmiName, gomock.Any()).DoAndReturn(func(_ context.Context, _, arg1 interface{}) interface{} {
				volumeOptions, ok := arg1.(*v1.RemoveVolumeOptions)
				Expect(ok).To(BeTrue())
				Expect(volumeOptions.Name).To(Equal(volumeName))
				if dryRun {
					Expect(volumeOptions.DryRun).To(Equal([]string{metav1.DryRunAll}))
				} else {
					Expect(volumeOptions.DryRun).To(BeEmpty())
				}
				return nil
			})
		}
	}

	expectVMEndpointRemoveVolume := func(dryRun bool) func() {
		return func() {
			kubecli.MockKubevirtClientInstance.
				EXPECT().
				VirtualMachine(metav1.NamespaceDefault).
				Return(vmInterface).
				Times(1)
			vmInterface.EXPECT().RemoveVolume(context.Background(), vmiName, gomock.Any()).DoAndReturn(func(_ context.Context, _, arg1 interface{}) interface{} {
				volumeOptions, ok := arg1.(*v1.RemoveVolumeOptions)
				Expect(ok).To(BeTrue())
				Expect(volumeOptions.Name).To(Equal(volumeName))
				if dryRun {
					Expect(volumeOptions.DryRun).To(Equal([]string{metav1.DryRunAll}))
				} else {
					Expect(volumeOptions.DryRun).To(BeEmpty())
				}
				return nil
			})
		}
	}

	DescribeTable("should fail with missing required or invalid parameters", func(expected string, extraArgs ...string) {
		args := append([]string{"removevolume"}, extraArgs...)
		cmd := clientcmd.NewRepeatableVirtctlCommand(args...)
		Expect(cmd()).To(MatchError(ContainSubstring(expected)))
	},
		Entry("no args", "accepts 1 arg(s), received 0"),
		Entry("with name, missing required volume-name", "required flag(s)", vmiName),
		Entry("with name and volume-name but invalid extra parameter", "unknown flag", vmiName, "--volume-name=blah", "--invalid=test"),
	)

	DescribeTable("should report error if call returns error according to option", func(expectFn func(), extraArgs ...string) {
		expectFn()
		args := append([]string{"removevolume", vmiName, "--volume-name=" + volumeName}, extraArgs...)
		cmd := clientcmd.NewRepeatableVirtctlCommand(args...)
		Expect(cmd()).To(MatchError(ContainSubstring("error removing")))
	},
		Entry("no args", expectVMIEndpointRemoveVolumeError),
		Entry("with persist", expectVMEndpointRemoveVolumeError, "--persist"),
		Entry("with dry-run", expectVMIEndpointRemoveVolumeError, "--dry-run"),
		Entry("with persist and dry-run", expectVMEndpointRemoveVolumeError, "--persist", "--dry-run"),
	)

	DescribeTable("should call correct endpoint", func(expectFn func(), extraArgs ...string) {
		expectFn()
		args := append([]string{"removevolume", vmiName, "--volume-name=" + volumeName}, extraArgs...)
		cmd := clientcmd.NewRepeatableVirtctlCommand(args...)
		Expect(cmd()).To(Succeed())
	},
		Entry("no args should call VMI endpoint", expectVMIEndpointRemoveVolume(false)),
		Entry("with persist should call VM endpoint", expectVMEndpointRemoveVolume(false), "--persist"),
		Entry("no persist with dry-run should call VMI endpoint", expectVMIEndpointRemoveVolume(true), "--dry-run"),
		Entry("with persist with dry-run should call VM endpoint", expectVMEndpointRemoveVolume(true), "--persist", "--dry-run"),
	)
})
