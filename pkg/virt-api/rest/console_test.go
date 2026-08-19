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

package rest

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"

	"github.com/emicklei/go-restful/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	kubevirtfake "kubevirt.io/client-go/kubevirt/fake"

	"kubevirt.io/kubevirt/pkg/libvmi"
	libvmistatus "kubevirt.io/kubevirt/pkg/libvmi/status"
	"kubevirt.io/kubevirt/pkg/testutils"
)

var _ = Describe("Console Subresource api", func() {
	var (
		recorder   *httptest.ResponseRecorder
		request    *restful.Request
		response   *restful.Response
		virtClient *kubevirtfake.Clientset
		app        *SubresourceAPIApp

		kv = &v1.KubeVirt{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kubevirt",
				Namespace: "kubevirt",
			},
			Spec: v1.KubeVirtSpec{
				Configuration: v1.KubeVirtConfiguration{
					DeveloperConfiguration: &v1.DeveloperConfiguration{},
				},
			},
			Status: v1.KubeVirtStatus{
				Phase: v1.KubeVirtPhaseDeploying,
			},
		}
	)

	config, _, _ := testutils.NewFakeClusterConfigUsingKV(kv)

	BeforeEach(func() {
		recorder = httptest.NewRecorder()
		request = restful.NewRequest(&http.Request{})
		response = restful.NewResponse(recorder)

		backend := ghttp.NewTLSServer()
		backendAddr := strings.Split(backend.Addr(), ":")
		backendPort, err := strconv.Atoi(backendAddr[1])
		Expect(err).ToNot(HaveOccurred())
		ctrl := gomock.NewController(GinkgoT())

		mockVirtClient := kubecli.NewMockKubevirtClient(ctrl)
		virtClient = kubevirtfake.NewSimpleClientset()

		mockVirtClient.EXPECT().VirtualMachineInstance(metav1.NamespaceDefault).Return(virtClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault)).AnyTimes()
		mockVirtClient.EXPECT().VirtualMachineInstance("").Return(virtClient.KubevirtV1().VirtualMachineInstances("")).AnyTimes()

		app = NewSubresourceAPIApp(mockVirtClient, backendPort, &tls.Config{InsecureSkipVerify: true}, config)
	})

	DescribeTable("request validation", func(autoattachSerialConsole bool, phase v1.VirtualMachineInstancePhase) {
		request.PathParameters()["name"] = testVMIName
		request.PathParameters()["namespace"] = metav1.NamespaceDefault

		vmi := libvmi.New(
			libvmi.WithName(testVMIName),
			libvmistatus.WithStatus(libvmistatus.New(libvmistatus.WithPhase(phase))),
		)
		vmi.Spec.Domain.Devices.AutoattachSerialConsole = &autoattachSerialConsole
		_, err := virtClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault).Create(context.TODO(), vmi, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		app.ConsoleRequestHandler(request, response)

		ExpectStatusErrorWithCode(recorder, http.StatusBadRequest)
	},
		Entry("should fail if there is no serial console", false, v1.Running),
		Entry("should fail if vmi is not running", true, v1.Scheduling),
	)

	It("should fail to connect to the serial console if the VMI is Failed", func() {
		request.PathParameters()["name"] = testVMIName
		request.PathParameters()["namespace"] = metav1.NamespaceDefault

		vmi := libvmi.New(libvmi.WithName(testVMIName),
			libvmi.WithNamespace(metav1.NamespaceDefault),
			libvmistatus.WithStatus(libvmistatus.New(
				libvmistatus.WithPhase(v1.Failed),
			)),
		)

		_, err := virtClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault).Create(context.TODO(), vmi, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		app.ConsoleRequestHandler(request, response)
		ExpectStatusErrorWithCode(recorder, http.StatusConflict)
	})
})
