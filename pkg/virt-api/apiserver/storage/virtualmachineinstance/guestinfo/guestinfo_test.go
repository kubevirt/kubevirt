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

package guestinfo

import (
	"context"
	"crypto/tls"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
)

const testVMIName = "testvm"

var _ = Describe("VirtualMachineInstance guest info", func() {
	var (
		vmiClient *kubecli.MockVirtualMachineInstanceInterface
		handler   *Handler
	)

	BeforeEach(func() {
		ctrl := gomock.NewController(GinkgoT())
		virtClient := kubecli.NewMockKubevirtClient(ctrl)
		kubeClient := fake.NewSimpleClientset()
		vmiClient = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)

		virtClient.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()
		virtClient.EXPECT().VirtualMachineInstance(metav1.NamespaceDefault).Return(vmiClient).AnyTimes()

		handler = NewHandler(virtClient, 0, &tls.Config{InsecureSkipVerify: true})

		kubeClient.Fake.PrependReactor("*", "*", func(action testing.Action) (bool, runtime.Object, error) {
			Expect(action).To(BeNil())
			return true, nil, nil
		})
	})

	type getter func(context.Context, string, string) (interface{}, error)

	DescribeTable("fails when the VMI does not exist", func(call getter) {
		vmiClient.EXPECT().Get(context.Background(), testVMIName, metav1.GetOptions{}).
			Return(nil, errors.NewNotFound(v1.Resource("virtualmachineinstance"), testVMIName))

		_, err := call(context.Background(), metav1.NamespaceDefault, testVMIName)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("not found"))
	},
		Entry("GuestOSInfo", func(ctx context.Context, ns, name string) (interface{}, error) {
			return handler.GetGuestOSInfo(ctx, ns, name)
		}),
		Entry("UserList", func(ctx context.Context, ns, name string) (interface{}, error) {
			return handler.GetUserList(ctx, ns, name)
		}),
		Entry("FilesystemList", func(ctx context.Context, ns, name string) (interface{}, error) {
			return handler.GetFilesystemList(ctx, ns, name)
		}),
	)

	DescribeTable("fails when the VMI is not running", func(call getter) {
		vmiClient.EXPECT().Get(context.Background(), testVMIName, metav1.GetOptions{}).
			Return(&v1.VirtualMachineInstance{}, nil)

		_, err := call(context.Background(), metav1.NamespaceDefault, testVMIName)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("VMI is not running"))
	},
		Entry("GuestOSInfo", func(ctx context.Context, ns, name string) (interface{}, error) {
			return handler.GetGuestOSInfo(ctx, ns, name)
		}),
		Entry("UserList", func(ctx context.Context, ns, name string) (interface{}, error) {
			return handler.GetUserList(ctx, ns, name)
		}),
		Entry("FilesystemList", func(ctx context.Context, ns, name string) (interface{}, error) {
			return handler.GetFilesystemList(ctx, ns, name)
		}),
	)

	DescribeTable("fails when the guest agent is not connected", func(call getter) {
		vmi := &v1.VirtualMachineInstance{
			Status: v1.VirtualMachineInstanceStatus{
				Phase:      v1.Running,
				Conditions: []v1.VirtualMachineInstanceCondition{},
			},
		}
		vmiClient.EXPECT().Get(context.Background(), testVMIName, metav1.GetOptions{}).Return(vmi, nil)

		_, err := call(context.Background(), metav1.NamespaceDefault, testVMIName)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("VMI does not have guest agent connected"))
	},
		Entry("GuestOSInfo", func(ctx context.Context, ns, name string) (interface{}, error) {
			return handler.GetGuestOSInfo(ctx, ns, name)
		}),
		Entry("UserList", func(ctx context.Context, ns, name string) (interface{}, error) {
			return handler.GetUserList(ctx, ns, name)
		}),
		Entry("FilesystemList", func(ctx context.Context, ns, name string) (interface{}, error) {
			return handler.GetFilesystemList(ctx, ns, name)
		}),
	)
})
