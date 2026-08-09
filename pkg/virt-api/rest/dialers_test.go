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
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"

	"github.com/emicklei/go-restful/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	"k8s.io/client-go/kubernetes/fake"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libvmi"
	libvmistatus "kubevirt.io/kubevirt/pkg/libvmi/status"
	"kubevirt.io/kubevirt/pkg/testutils"
)

var _ = Describe("NetDialer", func() {
	const (
		vmName      = "test-vm"
		vmNamespace = "test-namespace"
	)

	var (
		request *restful.Request
	)

	BeforeEach(func() {
		httpReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/apis/subresources.kubevirt.io/v1alpha3/namespaces/%s/virtualmachineinstances/%s/ssh/22", vmNamespace, vmName), nil)
		request = restful.NewRequest(httpReq)
	})

	newVMIWithPodNetwork := func(ip string) *v1.VirtualMachineInstance {
		return libvmi.New(
			libvmi.WithNamespace(vmNamespace),
			libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
			libvmistatus.WithStatus(libvmistatus.New(
				libvmistatus.WithInterfaceStatus(v1.VirtualMachineInstanceNetworkInterface{
					Name: v1.DefaultPodNetwork().Name,
					IP:   ip,
				}),
			)),
		)
	}

	It("Should fail if VMI has no pod network", func() {
		vmi := libvmi.New(
			libvmi.WithNamespace(vmNamespace),
			libvmistatus.WithStatus(libvmistatus.New(
				libvmistatus.WithInterfaceStatus(v1.VirtualMachineInstanceNetworkInterface{
					IP: "10.0.0.1",
				}),
			)),
		)
		dialer := netDial{request: request}
		_, statusErr := dialer.DialUnderlying(vmi)
		Expect(statusErr.Status().Message).To(ContainSubstring("no pod network"))
	})

	It("Should fail if pod network interface has no IP", func() {
		dialer := netDial{request: request}
		_, statusErr := dialer.DialUnderlying(newVMIWithPodNetwork(""))
		Expect(statusErr.Status().Message).To(ContainSubstring("no IP"))
	})

	It("Should fail if request has no port", func() {
		request.PathParameters()["port"] = ""
		dialer := netDial{request: request}
		_, statusErr := dialer.DialUnderlying(newVMIWithPodNetwork("192.168.0.1"))
		Expect(statusErr.Status().Message).To(Equal("port must not be empty"))
	})

	It("Should fail with unsupported protocol", func() {
		request.PathParameters()["port"] = "22"
		request.PathParameters()["protocol"] = "unix"
		dialer := netDial{request: request}
		_, statusErr := dialer.DialUnderlying(newVMIWithPodNetwork("192.168.0.1"))
		Expect(statusErr.Status().Message).To(ContainSubstring("unsupported protocol"))
	})

	DescribeTable("Should reject dangerous target IPs", func(ip, expectedMsg string) {
		dialer := netDial{request: request}
		_, statusErr := dialer.DialUnderlying(newVMIWithPodNetwork(ip))
		Expect(statusErr.Status().Message).To(ContainSubstring(expectedMsg))
	},
		Entry("loopback IPv4", "127.0.0.1", "loopback"),
		Entry("loopback IPv6", "::1", "loopback"),
		Entry("link-local IPv4", "169.254.1.1", "link-local"),
		Entry("link-local IPv6", "fe80::1", "link-local"),
		Entry("multicast IPv4", "224.0.0.1", "multicast"),
		Entry("multicast IPv6", "ff02::1", "multicast"),
		Entry("unspecified IPv4", "0.0.0.0", "unspecified"),
		Entry("unspecified IPv6", "::", "unspecified"),
	)

	It("Should forward error from Request's Body", func() {
		const errMsg = "foo bar from the App handler!"
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
			response := restful.NewResponse(rw)
			response.WriteHeader(http.StatusBadRequest)
			nbytes, err := response.Write([]byte(errMsg))
			Expect(nbytes).To(Equal(len(errMsg)))
			Expect(err).ToNot(HaveOccurred())
			response.Flush()
		}))
		defer server.Close()

		config, _, _ := testutils.NewFakeClusterConfigUsingKV(&v1.KubeVirt{})

		u, err := url.Parse(server.URL)
		Expect(err).NotTo(HaveOccurred())

		fullURL := "ws://" + u.Host + request.Request.URL.RequestURI()
		port, err := strconv.ParseInt(u.Port(), 10, 32)
		Expect(err).NotTo(HaveOccurred())

		ctrl := gomock.NewController(GinkgoT())
		virtClient := kubecli.NewMockKubevirtClient(ctrl)
		k8sfakeClient := fake.NewSimpleClientset()
		virtClient.EXPECT().CoreV1().Return(k8sfakeClient.CoreV1()).AnyTimes()

		runningStatus := libvmistatus.WithStatus(libvmistatus.New(libvmistatus.WithPhase(v1.Running)))
		vmi := libvmi.New(runningStatus)
		app := NewSubresourceAPIApp(virtClient, int(port), nil, config)
		dialer := app.virtHandlerDialer(func(_ *v1.VirtualMachineInstance, _ kubecli.VirtHandlerConn) (string, error) {
			return fullURL, nil
		})

		conn, statusErr := dialer.DialUnderlying(vmi)
		Expect(statusErr).To(MatchError(ContainSubstring(errMsg)))
		Expect(conn).To(BeNil())
	})
})
