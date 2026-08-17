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

package vnc

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"go.uber.org/mock/gomock"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	kubevirtfake "kubevirt.io/client-go/kubevirt/fake"

	"kubevirt.io/kubevirt/pkg/libvmi"
	libvmistatus "kubevirt.io/kubevirt/pkg/libvmi/status"
	"kubevirt.io/kubevirt/pkg/virt-api/streaming"
)

const testVMIName = "testvmi"

var _ = Describe("VNC streaming", func() {
	var (
		virtClient *kubevirtfake.Clientset
		handler    *Handler
	)

	BeforeEach(func() {
		backend := ghttp.NewTLSServer()
		backendAddr := strings.Split(backend.Addr(), ":")
		backendPort, err := strconv.Atoi(backendAddr[1])
		Expect(err).ToNot(HaveOccurred())
		ctrl := gomock.NewController(GinkgoT())

		mockVirtClient := kubecli.NewMockKubevirtClient(ctrl)
		virtClient = kubevirtfake.NewSimpleClientset()

		mockVirtClient.EXPECT().VirtualMachineInstance(metav1.NamespaceDefault).Return(virtClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault)).AnyTimes()

		streamer := streaming.NewStreamer(mockVirtClient, backendPort, &tls.Config{InsecureSkipVerify: true})
		handler = NewHandler(streamer)
	})

	streamVNC := func(name string) *errors.StatusError {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		recorder := httptest.NewRecorder()
		return handler.StreamVNC(context.Background(), metav1.NamespaceDefault, name, false, recorder, req)
	}

	DescribeTable("request validation", func(autoattachGraphicsDevice bool, phase v1.VirtualMachineInstancePhase) {
		vmi := libvmi.New(
			libvmi.WithName(testVMIName),
			libvmistatus.WithStatus(libvmistatus.New(libvmistatus.WithPhase(phase))),
		)
		vmi.Spec.Domain.Devices.AutoattachGraphicsDevice = &autoattachGraphicsDevice
		_, err := virtClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault).Create(context.TODO(), vmi, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		statusErr := streamVNC(testVMIName)

		Expect(statusErr).To(HaveOccurred())
		Expect(statusErr.Status().Code).To(Equal(int32(http.StatusBadRequest)))
	},
		Entry("should fail if there is no graphics device", false, v1.Running),
		Entry("should fail if vmi is not running", true, v1.Scheduling),
	)

	It("should fail if the vmi is not found", func() {
		statusErr := streamVNC(testVMIName)
		Expect(statusErr).To(HaveOccurred())
		Expect(statusErr.Status().Code).To(Equal(int32(http.StatusNotFound)))
	})

	DescribeTable("screenshot validation", func(autoattachGraphicsDevice bool, phase v1.VirtualMachineInstancePhase) {
		vmi := libvmi.New(
			libvmi.WithName(testVMIName),
			libvmistatus.WithStatus(libvmistatus.New(libvmistatus.WithPhase(phase))),
		)
		vmi.Spec.Domain.Devices.AutoattachGraphicsDevice = &autoattachGraphicsDevice
		_, err := virtClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault).Create(context.TODO(), vmi, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		_, statusErr := handler.Screenshot(context.Background(), metav1.NamespaceDefault, testVMIName)

		Expect(statusErr).To(HaveOccurred())
		Expect(statusErr.Status().Code).To(Equal(int32(http.StatusBadRequest)))
	},
		Entry("should fail if there is no graphics device", false, v1.Running),
		Entry("should fail if vmi is not running", true, v1.Scheduling),
	)

	It("should fail to take a screenshot if the vmi is not found", func() {
		_, statusErr := handler.Screenshot(context.Background(), metav1.NamespaceDefault, testVMIName)
		Expect(statusErr).To(HaveOccurred())
		Expect(statusErr.Status().Code).To(Equal(int32(http.StatusNotFound)))
	})
})
