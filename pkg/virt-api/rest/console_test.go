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
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"

	"github.com/emicklei/go-restful/v3"
	"github.com/gorilla/websocket"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"go.uber.org/mock/gomock"
	k8scorev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	kubevirtfake "kubevirt.io/client-go/kubevirt/fake"
	corev1 "kubevirt.io/client-go/kubevirt/typed/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	libvmistatus "kubevirt.io/kubevirt/pkg/libvmi/status"
	"kubevirt.io/kubevirt/pkg/testutils"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

var _ = Describe("Console Subresource api", func() {
	var (
		virtClient *kubevirtfake.Clientset
		app        *SubresourceAPIApp
		config     *virtconfig.ClusterConfig
	)

	BeforeEach(func() {
		mockVirtClient := kubecli.NewMockKubevirtClient(gomock.NewController(GinkgoT()))
		virtClient = kubevirtfake.NewSimpleClientset()

		mockVirtClient.EXPECT().VirtualMachineInstance(metav1.NamespaceDefault).Return(virtClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault)).AnyTimes()
		mockVirtClient.EXPECT().VirtualMachineInstance("").Return(virtClient.KubevirtV1().VirtualMachineInstances("")).AnyTimes()

		config, _, _ = testutils.NewFakeClusterConfigUsingKV(&v1.KubeVirt{
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
		})
		app = NewSubresourceAPIApp(mockVirtClient, 8080, &tls.Config{InsecureSkipVerify: true}, config)
	})

	Context("validation", func() {
		var (
			recorder *httptest.ResponseRecorder
			request  *restful.Request
			response *restful.Response
		)

		BeforeEach(func() {
			recorder = httptest.NewRecorder()
			request = restful.NewRequest(&http.Request{})
			request.PathParameters()["name"] = testVMIName
			request.PathParameters()["namespace"] = metav1.NamespaceDefault

			response = restful.NewResponse(recorder)

		})

		DescribeTable("request validation", func(autoattachSerialConsole bool, phase v1.VirtualMachineInstancePhase) {
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

	It("should proxy websocket data from client to handler", func() {
		const vmiName = "testvmi"
		// Create a fake handler server that echoes back messages
		handlerServer := ghttp.NewTLSServer()
		defer handlerServer.Close()

		handlerServer.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest("GET", "/v1/namespaces/default/virtualmachineinstances/"+vmiName+"/console"),
				func(w http.ResponseWriter, r *http.Request) {
					upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
					conn, err := upgrader.Upgrade(w, r, nil)
					Expect(err).ToNot(HaveOccurred())
					defer conn.Close()

					for {
						msgType, data, err := conn.ReadMessage()
						if err != nil {
							return
						}
						conn.WriteMessage(msgType, data)
					}
				},
			),
		)

		// Create a running VMI with serial console
		vmi := libvmi.New(
			libvmi.WithName(vmiName),
			libvmi.WithNamespace(metav1.NamespaceDefault),
		)
		vmi.Status.Phase = v1.Running
		vmi.Status.NodeName = "test-node"
		autoattach := true
		vmi.Spec.Domain.Devices.AutoattachSerialConsole = &autoattach
		_, err := virtClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault).Create(context.TODO(), vmi, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		// Extract the handler server's address and port
		handlerAddr := handlerServer.Addr()
		handlerParts := strings.Split(handlerAddr, ":")
		handlerPort := handlerParts[len(handlerParts)-1]

		// Set up mock handler pod for node lookup
		kubeClient := k8sfake.NewClientset(
			&k8scorev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "virt-handler-test",
					Namespace: "kubevirt",
					Labels: map[string]string{
						"kubevirt.io": "virt-handler",
					},
				},
				Spec: k8scorev1.PodSpec{
					NodeName: "test-node",
				},
				Status: k8scorev1.PodStatus{
					PodIP: "127.0.0.1",
				},
			},
		)

		// Replace the mock client's CoreV1 to return our fake kubeClient
		mockVirtClient := kubecli.NewMockKubevirtClient(gomock.NewController(GinkgoT()))
		mockVirtClient.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()
		mockVirtClient.EXPECT().VirtualMachineInstance(metav1.NamespaceDefault).Return(virtClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault)).AnyTimes()
		mockVirtClient.EXPECT().VirtualMachineInstance("").Return(virtClient.KubevirtV1().VirtualMachineInstances("")).AnyTimes()

		// Create app with updated mock that has CoreV1 setup
		handlerPortInt, err := strconv.Atoi(handlerPort)
		Expect(err).ToNot(HaveOccurred())
		app = NewSubresourceAPIApp(mockVirtClient, handlerPortInt, &tls.Config{InsecureSkipVerify: true}, config)

		// Create API server that wraps ConsoleRequestHandler
		apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			restReq := restful.NewRequest(r)
			// restReq.
			restReq.PathParameters()["namespace"] = metav1.NamespaceDefault
			restReq.PathParameters()["name"] = vmiName

			app.ConsoleRequestHandler(restReq, restful.NewResponse(w))
		}))
		defer apiServer.Close()

		// Use AsyncSubresourceHelper (from client-go) to connect as a real client would
		clientConfig := rest.Config{
			Host:            apiServer.URL,
			TLSClientConfig: rest.TLSClientConfig{Insecure: true},
		}

		stream, err := corev1.AsyncSubresourceHelper(
			&clientConfig,
			"virtualmachineinstances",
			metav1.NamespaceDefault,
			vmiName,
			"console",
			url.Values{},
		)
		Expect(err).ToNot(HaveOccurred())

		testData := []byte("hello from console client")
		var out bytes.Buffer
		readerChan := make(chan struct{})

		streamDone := make(chan error, 1)
		go func() {
			streamDone <- stream.Stream(corev1.StreamOptions{
				In:  io.MultiReader(bytes.NewReader(testData), blockReader(readerChan)),
				Out: &out,
			})
		}()

		Eventually(out.Bytes).Should(Equal(testData))
		close(readerChan)
		Expect(<-streamDone).ToNot(HaveOccurred())

	})

	It("should proxy websocket data from handler to client", func() {
		const vmiName = "testvmi"
		// Create a fake handler server that echoes back messages
		handlerServer := ghttp.NewTLSServer()
		defer handlerServer.Close()

		// More than our buffer
		randomData := make([]byte, 10340)
		_, err := rand.Read(randomData)
		Expect(err).ToNot(HaveOccurred())

		handlerServer.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest("GET", "/v1/namespaces/default/virtualmachineinstances/"+vmiName+"/console"),
				func(w http.ResponseWriter, r *http.Request) {
					upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
					conn, err := upgrader.Upgrade(w, r, nil)
					Expect(err).ToNot(HaveOccurred())
					defer conn.Close()

					conn.WriteMessage(websocket.BinaryMessage, randomData)
					conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				},
			),
		)

		// Create a running VMI with serial console
		vmi := libvmi.New(
			libvmi.WithName(vmiName),
			libvmi.WithNamespace(metav1.NamespaceDefault),
		)
		vmi.Status.Phase = v1.Running
		vmi.Status.NodeName = "test-node"
		autoattach := true
		vmi.Spec.Domain.Devices.AutoattachSerialConsole = &autoattach
		_, err = virtClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault).Create(context.TODO(), vmi, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		// Extract the handler server's address and port
		handlerAddr := handlerServer.Addr()
		handlerParts := strings.Split(handlerAddr, ":")
		handlerPort := handlerParts[len(handlerParts)-1]

		// Set up mock handler pod for node lookup
		kubeClient := k8sfake.NewClientset(
			&k8scorev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "virt-handler-test",
					Namespace: "kubevirt",
					Labels: map[string]string{
						"kubevirt.io": "virt-handler",
					},
				},
				Spec: k8scorev1.PodSpec{
					NodeName: "test-node",
				},
				Status: k8scorev1.PodStatus{
					PodIP: "127.0.0.1",
				},
			},
		)

		// Replace the mock client's CoreV1 to return our fake kubeClient
		mockVirtClient := kubecli.NewMockKubevirtClient(gomock.NewController(GinkgoT()))
		mockVirtClient.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()
		mockVirtClient.EXPECT().VirtualMachineInstance(metav1.NamespaceDefault).Return(virtClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault)).AnyTimes()
		mockVirtClient.EXPECT().VirtualMachineInstance("").Return(virtClient.KubevirtV1().VirtualMachineInstances("")).AnyTimes()

		// Create app with updated mock that has CoreV1 setup
		handlerPortInt, err := strconv.Atoi(handlerPort)
		Expect(err).ToNot(HaveOccurred())
		app = NewSubresourceAPIApp(mockVirtClient, handlerPortInt, &tls.Config{InsecureSkipVerify: true}, config)

		// Create API server that wraps ConsoleRequestHandler
		apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			restReq := restful.NewRequest(r)
			// restReq.
			restReq.PathParameters()["namespace"] = metav1.NamespaceDefault
			restReq.PathParameters()["name"] = vmiName

			app.ConsoleRequestHandler(restReq, restful.NewResponse(w))
		}))
		defer apiServer.Close()

		// Use AsyncSubresourceHelper (from client-go) to connect as a real client would
		clientConfig := rest.Config{
			Host:            apiServer.URL,
			TLSClientConfig: rest.TLSClientConfig{Insecure: true},
		}

		stream, err := corev1.AsyncSubresourceHelper(
			&clientConfig,
			"virtualmachineinstances",
			metav1.NamespaceDefault,
			vmiName,
			"console",
			url.Values{},
		)
		Expect(err).ToNot(HaveOccurred())

		var out bytes.Buffer
		readerChan := make(chan struct{})
		defer close(readerChan)

		streamDone := make(chan error, 1)
		go func() {
			streamDone <- stream.Stream(corev1.StreamOptions{
				In:  blockReader(readerChan),
				Out: &out,
			})
		}()

		Eventually(out.Bytes).Should(Equal(randomData))
		Expect(<-streamDone).ToNot(HaveOccurred())
	})

})

type blockReader <-chan struct{}

func (b blockReader) Read([]byte) (int, error) {
	<-b
	return 0, io.EOF
}
