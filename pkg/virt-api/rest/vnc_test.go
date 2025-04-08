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
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"

	"github.com/emicklei/go-restful/v3"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/testing"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	kubevirtfake "kubevirt.io/client-go/kubevirt/fake"

	"kubevirt.io/kubevirt/pkg/libvmi"
	libvmistatus "kubevirt.io/kubevirt/pkg/libvmi/status"
	"kubevirt.io/kubevirt/pkg/testutils"
)

var _ = Describe("VNC Subresource api", func() {
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

	It("should fail with no 'name' path param", func() {
		virtClient.Fake.PrependReactor("get", "virtualmachineinstances", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			return true, nil, fmt.Errorf("resource name may not be empty")
		})
		app.VNCRequestHandler(request, response)
		ExpectStatusErrorWithCode(recorder, http.StatusInternalServerError)
	})

	It("should fail with no 'namespace' path param", func() {
		virtClient.Fake.PrependReactor("get", "virtualmachineinstances", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			return true, nil, fmt.Errorf("an empty namespace may not be set when a resource name is provided")
		})
		request.PathParameters()["name"] = testVMIName

		app.VNCRequestHandler(request, response)
		ExpectStatusErrorWithCode(recorder, http.StatusInternalServerError)
	})

	It("should fail if vmi is not found", func() {
		request.PathParameters()["name"] = testVMIName
		request.PathParameters()["namespace"] = metav1.NamespaceDefault

		app.VNCRequestHandler(request, response)
		ExpectStatusErrorWithCode(recorder, http.StatusNotFound)
	})

	It("should fail with internal at fetching vmi errors", func() {

		request.PathParameters()["name"] = testVMIName
		request.PathParameters()["namespace"] = metav1.NamespaceDefault

		virtClient.Fake.PrependReactor("get", "virtualmachineinstances", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			return true, nil, fmt.Errorf("unable to retrieve vmi [%s]", testVMIName)
		})

		app.VNCRequestHandler(request, response)
		ExpectStatusErrorWithCode(recorder, http.StatusInternalServerError)
	})

	DescribeTable("request validation", func(autoattachGraphicsDevice bool, phase v1.VirtualMachineInstancePhase) {
		request.PathParameters()["name"] = testVMIName
		request.PathParameters()["namespace"] = metav1.NamespaceDefault

		vmi := libvmi.New(
			libvmi.WithName(testVMIName),
			libvmistatus.WithStatus(libvmistatus.New(libvmistatus.WithPhase(phase))),
		)
		vmi.Spec.Domain.Devices.AutoattachGraphicsDevice = &autoattachGraphicsDevice
		_, err := virtClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault).Create(context.TODO(), vmi, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		app.VNCRequestHandler(request, response)

		ExpectStatusErrorWithCode(recorder, http.StatusBadRequest)
		if !autoattachGraphicsDevice {
			ExpectMessage(recorder, Equal("No graphics devices are present."))
		}
	},
		Entry("should fail if there is no graphics device", false, v1.Running),
		Entry("should fail if vmi is not running", true, v1.Scheduling),
	)
})
