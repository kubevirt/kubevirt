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
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"

	"github.com/emicklei/go-restful/v3"
	"github.com/gorilla/websocket"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"go.uber.org/mock/gomock"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	kubevirtfake "kubevirt.io/client-go/kubevirt/fake"

	"kubevirt.io/kubevirt/pkg/testutils"
)

var _ = Describe("PortForward Subresource api", func() {
	var (
		recorder       *httptest.ResponseRecorder
		request        *restful.Request
		response       *restful.Response
		mockVirtClient *kubecli.MockKubevirtClient
		virtClient     *kubevirtfake.Clientset
		app            *SubresourceAPIApp

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

		mockVirtClient = kubecli.NewMockKubevirtClient(ctrl)
		virtClient = kubevirtfake.NewSimpleClientset()

		mockVirtClient.EXPECT().VirtualMachineInstance(metav1.NamespaceDefault).Return(virtClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault)).AnyTimes()
		mockVirtClient.EXPECT().VirtualMachineInstance("").Return(virtClient.KubevirtV1().VirtualMachineInstances("")).AnyTimes()

		app = NewSubresourceAPIApp(mockVirtClient, nil, nil, backendPort, &tls.Config{InsecureSkipVerify: true}, config)
	})

	It("should fail with no 'name' path param", func() {
		virtClient.Fake.PrependReactor("get", "virtualmachineinstances", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			return true, nil, fmt.Errorf("resource name may not be empty")
		})
		app.PortForwardRequestHandler(app.FetchVirtualMachineInstance)(request, response)
		ExpectStatusErrorWithCode(recorder, http.StatusInternalServerError)
	})

	It("should fail with no 'namespace' path param", func() {
		virtClient.Fake.PrependReactor("get", "virtualmachineinstances", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			return true, nil, fmt.Errorf("an empty namespace may not be set when a resource name is provided")
		})
		request.PathParameters()["name"] = testVMIName

		app.PortForwardRequestHandler(app.FetchVirtualMachineInstance)(request, response)
		ExpectStatusErrorWithCode(recorder, http.StatusInternalServerError)
	})

	It("should fail if vmi is not found", func() {
		request.PathParameters()["name"] = testVMIName
		request.PathParameters()["namespace"] = metav1.NamespaceDefault

		app.PortForwardRequestHandler(app.FetchVirtualMachineInstance)(request, response)
		ExpectStatusErrorWithCode(recorder, http.StatusNotFound)
	})

	It("should fail with internal at fetching vmi errors", func() {

		request.PathParameters()["name"] = testVMIName
		request.PathParameters()["namespace"] = metav1.NamespaceDefault

		virtClient.Fake.PrependReactor("get", "virtualmachineinstances", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			return true, nil, fmt.Errorf("unable to retrieve vmi [%s]", testVMIName)
		})

		app.PortForwardRequestHandler(app.FetchVirtualMachineInstance)(request, response)
		ExpectStatusErrorWithCode(recorder, http.StatusInternalServerError)
	})

	It("should dial the launcher pod IP resolved from the indexer", func() {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		Expect(err).ToNot(HaveOccurred())
		defer ln.Close()
		tcpAddr := ln.Addr().(*net.TCPAddr)

		// Nothing listens on the interface IP; the listener is only reachable via the pod IP.
		vmi := &v1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{Name: testVMIName, Namespace: metav1.NamespaceDefault, UID: "1234"},
			Spec:       v1.VirtualMachineInstanceSpec{Networks: []v1.Network{*v1.DefaultPodNetwork()}},
			Status: v1.VirtualMachineInstanceStatus{
				Phase:      v1.Running,
				NodeName:   "node1",
				Interfaces: []v1.VirtualMachineInstanceNetworkInterface{{Name: v1.DefaultPodNetwork().Name, IP: "::1"}},
			},
		}
		_, err = virtClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault).Create(context.Background(), vmi, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		pod := &k8sv1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:            "virt-launcher-" + testVMIName,
				Namespace:       metav1.NamespaceDefault,
				OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(vmi, v1.VirtualMachineInstanceGroupVersionKind)},
			},
			Spec:   k8sv1.PodSpec{NodeName: "node1"},
			Status: k8sv1.PodStatus{PodIP: tcpAddr.IP.String()},
		}
		indexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
		Expect(indexer.Add(pod)).To(Succeed())

		app = NewSubresourceAPIApp(mockVirtClient, nil, indexer, 0, nil, config)

		ws := new(restful.WebService)
		ws.Route(ws.GET("/namespaces/{namespace}/virtualmachineinstances/{name}/portforward/{port}").
			To(app.PortForwardRequestHandler(app.FetchVirtualMachineInstance)))
		container := restful.NewContainer()
		container.Add(ws)
		srv := httptest.NewServer(container)
		defer srv.Close()

		accepted := make(chan struct{})
		go func() {
			defer GinkgoRecover()
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			conn.Close()
			close(accepted)
		}()

		url := fmt.Sprintf("ws%s/namespaces/%s/virtualmachineinstances/%s/portforward/%d",
			strings.TrimPrefix(srv.URL, "http"), metav1.NamespaceDefault, testVMIName, tcpAddr.Port)
		clientConn, _, err := websocket.DefaultDialer.Dial(url, nil)
		Expect(err).ToNot(HaveOccurred())
		defer clientConn.Close()
		Eventually(accepted).Should(BeClosed())
	})
})
