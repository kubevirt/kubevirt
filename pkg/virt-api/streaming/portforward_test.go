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

package streaming

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
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	kubevirtfake "kubevirt.io/client-go/kubevirt/fake"

	"kubevirt.io/kubevirt/pkg/libvmi"
	libvmistatus "kubevirt.io/kubevirt/pkg/libvmi/status"
)

var _ = Describe("PortForward streaming", func() {
	var (
		virtClient *kubevirtfake.Clientset
		streamer   *Streamer
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

		streamer = NewStreamer(mockVirtClient, backendPort, &tls.Config{InsecureSkipVerify: true})
	})

	streamPortForward := func(name, port, protocol string) *errors.StatusError {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		recorder := httptest.NewRecorder()
		return streamer.StreamPortForward(context.Background(), metav1.NamespaceDefault, name, port, protocol, recorder, req)
	}

	It("should fail if the vmi is paused", func() {
		vmi := libvmi.New(
			libvmi.WithName(testVMIName),
			libvmistatus.WithStatus(libvmistatus.New(
				libvmistatus.WithPhase(v1.Running),
				libvmistatus.WithCondition(v1.VirtualMachineInstanceCondition{
					Type:   v1.VirtualMachineInstancePaused,
					Status: k8sv1.ConditionTrue,
				}),
			)),
		)
		_, err := virtClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault).Create(context.TODO(), vmi, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		statusErr := streamPortForward(testVMIName, "8080", "tcp")
		Expect(statusErr).ToNot(BeNil())
		Expect(statusErr.Status().Code).To(Equal(int32(http.StatusConflict)))
	})

	It("should fail if no port is provided", func() {
		vmi := libvmi.New(
			libvmi.WithName(testVMIName),
			libvmistatus.WithStatus(libvmistatus.New(
				libvmistatus.WithPhase(v1.Running),
				libvmistatus.WithInterfaceStatus(v1.VirtualMachineInstanceNetworkInterface{IP: "10.0.0.1"}),
			)),
		)
		_, err := virtClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault).Create(context.TODO(), vmi, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		statusErr := streamPortForward(testVMIName, "", "tcp")
		Expect(statusErr).ToNot(BeNil())
		Expect(statusErr.Status().Code).To(Equal(int32(http.StatusBadRequest)))
	})
})
