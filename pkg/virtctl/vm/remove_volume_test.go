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
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/golang/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/testing"
	kubevirtfake "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/fake"
	kvtesting "kubevirt.io/client-go/testing"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests/clientcmd"
)

var _ = Describe("Remove volume command", func() {
	const (
		vmiName    = "testvmi"
		volumeName = "testvolume"
	)

	var virtClient *kubevirtfake.Clientset

	BeforeEach(func() {
		ctrl := gomock.NewController(GinkgoT())
		kubecli.GetKubevirtClientFromClientConfig = kubecli.GetMockKubevirtClientFromClientConfig
		kubecli.MockKubevirtClientInstance = kubecli.NewMockKubevirtClient(ctrl)
		virtClient = kubevirtfake.NewSimpleClientset()
	})

	expectVMIEndpointRemoveVolumeError := func() {
		kubecli.MockKubevirtClientInstance.
			EXPECT().
			VirtualMachineInstance(metav1.NamespaceDefault).
			Return(virtClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault)).
			Times(1)
		virtClient.PrependReactor("put", "virtualmachineinstances/removevolume", func(_ testing.Action) (handled bool, ret runtime.Object, err error) {
			return true, nil, errors.New("error removing")
		})
	}

	expectVMEndpointRemoveVolumeError := func() {
		kubecli.MockKubevirtClientInstance.
			EXPECT().
			VirtualMachine(metav1.NamespaceDefault).
			Return(virtClient.KubevirtV1().VirtualMachines(metav1.NamespaceDefault)).
			Times(1)
		virtClient.PrependReactor("put", "virtualmachines/removevolume", func(_ testing.Action) (handled bool, ret runtime.Object, err error) {
			return true, nil, errors.New("error removing")
		})
	}

	expectVMIEndpointRemoveVolume := func(dryRun bool) func() {
		return func() {
			kubecli.MockKubevirtClientInstance.
				EXPECT().
				VirtualMachineInstance(metav1.NamespaceDefault).
				Return(virtClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault)).
				Times(1)
			virtClient.PrependReactor("put", "virtualmachineinstances/removevolume", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
				switch action := action.(type) {
				case kvtesting.PutAction[*v1.RemoveVolumeOptions]:
					volumeOptions := action.GetOptions()
					Expect(volumeOptions.Name).To(Equal(volumeName))
					if dryRun {
						Expect(volumeOptions.DryRun).To(Equal([]string{metav1.DryRunAll}))
					} else {
						Expect(volumeOptions.DryRun).To(BeEmpty())
					}
					return true, nil, nil
				default:
					Fail("unexpected action type on removevolume")
					return false, nil, nil
				}
			})
		}
	}

	expectVMEndpointRemoveVolume := func(dryRun bool) func() {
		return func() {
			kubecli.MockKubevirtClientInstance.
				EXPECT().
				VirtualMachine(metav1.NamespaceDefault).
				Return(virtClient.KubevirtV1().VirtualMachines(metav1.NamespaceDefault)).
				Times(1)
			virtClient.PrependReactor("put", "virtualmachines/removevolume", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
				switch action := action.(type) {
				case kvtesting.PutAction[*v1.RemoveVolumeOptions]:
					volumeOptions := action.GetOptions()
					Expect(volumeOptions.Name).To(Equal(volumeName))
					if dryRun {
						Expect(volumeOptions.DryRun).To(Equal([]string{metav1.DryRunAll}))
					} else {
						Expect(volumeOptions.DryRun).To(BeEmpty())
					}
					return true, nil, nil
				default:
					Fail("unexpected action type on removevolume")
					return false, nil, nil
				}
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

	DescribeTable("should report error if call returns error according to option", func(expectFn func(), resourceName string, extraArgs ...string) {
		expectFn()
		args := append([]string{"removevolume", vmiName, "--volume-name=" + volumeName}, extraArgs...)
		cmd := clientcmd.NewRepeatableVirtctlCommand(args...)
		Expect(cmd()).To(MatchError(ContainSubstring("error removing")))
		Expect(kvtesting.FilterActions(&virtClient.Fake, "put", resourceName, "removevolume")).To(HaveLen(1))
	},
		Entry("no args", expectVMIEndpointRemoveVolumeError, "virtualmachineinstances"),
		Entry("with persist", expectVMEndpointRemoveVolumeError, "virtualmachines", "--persist"),
		Entry("with dry-run", expectVMIEndpointRemoveVolumeError, "virtualmachineinstances", "--dry-run"),
		Entry("with persist and dry-run", expectVMEndpointRemoveVolumeError, "virtualmachines", "--persist", "--dry-run"),
	)

	DescribeTable("should call correct endpoint", func(expectFn func(), resourceName string, extraArgs ...string) {
		expectFn()
		args := append([]string{"removevolume", vmiName, "--volume-name=" + volumeName}, extraArgs...)
		cmd := clientcmd.NewRepeatableVirtctlCommand(args...)
		Expect(cmd()).To(Succeed())
		Expect(kvtesting.FilterActions(&virtClient.Fake, "put", resourceName, "removevolume")).To(HaveLen(1))
	},
		Entry("no args should call VMI endpoint", expectVMIEndpointRemoveVolume(false), "virtualmachineinstances"),
		Entry("with persist should call VM endpoint", expectVMEndpointRemoveVolume(false), "virtualmachines", "--persist"),
		Entry("no persist with dry-run should call VMI endpoint", expectVMIEndpointRemoveVolume(true), "virtualmachineinstances", "--dry-run"),
		Entry("with persist with dry-run should call VM endpoint", expectVMEndpointRemoveVolume(true), "virtualmachines", "--persist", "--dry-run"),
	)
})
