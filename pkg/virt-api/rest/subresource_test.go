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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package rest

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	restful "github.com/emicklei/go-restful"
	"github.com/gorilla/websocket"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	k8sv1 "k8s.io/api/core/v1"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/httpstream"
	"k8s.io/apimachinery/pkg/util/httpstream/spdy"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
)

var _ = Describe("VirtualMachineInstance Subresources", func() {
	var server *ghttp.Server
	var backend *ghttp.Server
	var request *restful.Request
	var response *restful.Response
	var wsURL string

	running := true
	notRunning := false

	log.Log.SetIOWriter(GinkgoWriter)

	pvcCache := cache.NewIndexer(cache.DeletionHandlingMetaNamespaceKeyFunc, nil)
	app := SubresourceAPIApp{}
	BeforeEach(func() {
		server = ghttp.NewServer()
		backend = ghttp.NewServer()
		flag.Set("kubeconfig", "")
		flag.Set("master", server.URL())
		app.VirtCli, _ = kubecli.GetKubevirtClientFromFlags(server.URL(), "")

		request = restful.NewRequest(&http.Request{})
		response = restful.NewResponse(httptest.NewRecorder())
		wsURL = "ws" + strings.TrimPrefix(backend.URL(), "http")

		// To emulate rest server
		backend.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest("GET", "/"),
				func(w http.ResponseWriter, r *http.Request) {
					request.Request = r
					response.ResponseWriter = w
					app.VNCRequestHandler(request, response)
				},
			),
		)
	})

	Context("Subresource api", func() {
		It("should find matching pod for running VirtualMachineInstance", func(done Done) {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.Status.Phase = v1.Running
			vmi.ObjectMeta.SetUID(uuid.NewUUID())

			config, _ := testutils.NewFakeClusterConfig(&k8sv1.ConfigMap{})
			templateService := services.NewTemplateService("whatever", "whatever", "whatever", "whatever", pvcCache, app.VirtCli, config)

			pod, err := templateService.RenderLaunchManifest(vmi)
			Expect(err).ToNot(HaveOccurred())
			pod.ObjectMeta.Name = "madeup-name"

			pod.Spec.NodeName = "mynode"
			pod.Status.Phase = k8sv1.PodRunning

			podList := k8sv1.PodList{}
			podList.Items = []k8sv1.Pod{}
			podList.Items = append(podList.Items, *pod)

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/api/v1/namespaces/default/pods"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, podList),
				),
			)

			podName, httpStatusCode, err := app.remoteExecInfo(vmi)
			Expect(err).ToNot(HaveOccurred())

			Expect(podName).To(Equal("madeup-name"))
			Expect(httpStatusCode).To(Equal(http.StatusOK))
			close(done)
		}, 5)

		It("should fail if VirtualMachineInstance is not in running state", func(done Done) {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.Status.Phase = v1.Succeeded
			vmi.ObjectMeta.SetUID(uuid.NewUUID())

			_, httpStatusCode, err := app.remoteExecInfo(vmi)

			Expect(err).To(HaveOccurred())
			Expect(httpStatusCode).To(Equal(http.StatusBadRequest))
			close(done)
		}, 5)

		It("should fail no matching pod is found", func(done Done) {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.Status.Phase = v1.Running
			vmi.ObjectMeta.SetUID(uuid.NewUUID())

			podList := k8sv1.PodList{}
			podList.Items = []k8sv1.Pod{}

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/api/v1/namespaces/default/pods"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, podList),
				),
			)

			_, httpStatusCode, err := app.remoteExecInfo(vmi)

			Expect(err).To(HaveOccurred())
			Expect(httpStatusCode).To(Equal(http.StatusBadRequest))
			close(done)
		}, 5)

		It("should fail with no 'name' path param", func(done Done) {

			app.VNCRequestHandler(request, response)
			Expect(response.Error()).To(HaveOccurred())
			Expect(response.StatusCode()).To(Equal(http.StatusInternalServerError))
			close(done)
		}, 5)

		It("should fail with no 'namespace' path param", func(done Done) {

			request.PathParameters()["name"] = "testvmi"

			app.VNCRequestHandler(request, response)
			Expect(response.Error()).To(HaveOccurred())
			Expect(response.StatusCode()).To(Equal(http.StatusInternalServerError))
			close(done)
		}, 5)

		It("should fail if vmi is not found", func(done Done) {

			request.PathParameters()["name"] = "testvmi"
			request.PathParameters()["namespace"] = "default"

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/apis/kubevirt.io/v1alpha3/namespaces/default/virtualmachineinstances/testvmi"),
					ghttp.RespondWithJSONEncoded(http.StatusNotFound, nil),
				),
			)

			app.VNCRequestHandler(request, response)
			Expect(response.Error()).To(HaveOccurred())
			Expect(response.StatusCode()).To(Equal(http.StatusNotFound))
			close(done)
		}, 5)

		It("should fail with internal at fetching vmi errors", func(done Done) {

			request.PathParameters()["name"] = "testvmi"
			request.PathParameters()["namespace"] = "default"

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/apis/kubevirt.io/v1alpha3/namespaces/default/virtualmachineinstances/testvmi"),
					ghttp.RespondWithJSONEncoded(http.StatusServiceUnavailable, nil),
				),
			)

			app.VNCRequestHandler(request, response)
			Expect(response.Error()).To(HaveOccurred())
			Expect(response.StatusCode()).To(Equal(http.StatusInternalServerError))
			close(done)
		}, 5)

		It("should fail with no graphics device at VNC connections", func(done Done) {

			request.PathParameters()["name"] = "testvmi"
			request.PathParameters()["namespace"] = "default"

			flag := false
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.Status.Phase = v1.Running
			vmi.ObjectMeta.SetUID(uuid.NewUUID())
			vmi.Spec.Domain.Devices.AutoattachGraphicsDevice = &flag

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/apis/kubevirt.io/v1alpha3/namespaces/default/virtualmachineinstances/testvmi"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, vmi),
				),
			)
			app.VNCRequestHandler(request, response)
			Expect(response.Error()).To(HaveOccurred())
			Expect(response.StatusCode()).To(Equal(http.StatusBadRequest))
			close(done)
		}, 5)

		It("should fail with no graphics device at VNC connections", func(done Done) {

			request.PathParameters()["name"] = "testvmi"
			request.PathParameters()["namespace"] = "default"

			flag := false
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.Status.Phase = v1.Running
			vmi.ObjectMeta.SetUID(uuid.NewUUID())
			vmi.Spec.Domain.Devices.AutoattachGraphicsDevice = &flag

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/apis/kubevirt.io/v1alpha3/namespaces/default/virtualmachineinstances/testvmi"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, vmi),
				),
			)

			app.VNCRequestHandler(request, response)
			Expect(response.Error()).To(HaveOccurred())
			Expect(response.StatusCode()).To(Equal(http.StatusBadRequest))
			close(done)
		}, 5)

		It("Should pass client websocket io to server SPDY io", func(done Done) {

			request.PathParameters()["name"] = "testvmi"
			request.PathParameters()["namespace"] = "default"

			vmi := v1.NewMinimalVMI("testvmi")
			vmi.Status.Phase = v1.Running
			vmi.ObjectMeta.SetUID(uuid.NewUUID())

			config, _ := testutils.NewFakeClusterConfig(&k8sv1.ConfigMap{})
			templateService := services.NewTemplateService("whatever", "whatever", "whatever", "whatever", pvcCache, app.VirtCli, config)

			pod, err := templateService.RenderLaunchManifest(vmi)
			Expect(err).ToNot(HaveOccurred())
			pod.ObjectMeta.Name = "madeup-name"

			pod.Spec.NodeName = "mynode"
			pod.Status.Phase = k8sv1.PodRunning

			podList := k8sv1.PodList{}
			podList.Items = []k8sv1.Pod{}
			podList.Items = append(podList.Items, *pod)

			newStreamChannel := make(chan httpstream.Stream)

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/apis/kubevirt.io/v1alpha3/namespaces/default/virtualmachineinstances/testvmi"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, vmi),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/api/v1/namespaces/default/pods"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, podList),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/api/v1/namespaces/default/pods/madeup-name/exec"),
					func(w http.ResponseWriter, r *http.Request) {
						upgrader := spdy.NewResponseUpgrader()
						upgrader.UpgradeResponse(w, r,
							func(stream httpstream.Stream, replySent <-chan struct{}) error {
								newStreamChannel <- stream
								return nil
							})
					},
				),
			)

			ws, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusSwitchingProtocols))

			streamType := func(stream httpstream.Stream) string {
				return stream.Headers().Get("streamType")
			}

			// Receive accepted stream
			// FIXME: It's no good to depend on order, implementation
			// can change
			streamError := <-newStreamChannel
			Expect(streamType(streamError)).To(Equal("error"))
			streamStdin := <-newStreamChannel
			Expect(streamType(streamStdin)).To(Equal("stdin"))
			streamStdout := <-newStreamChannel
			Expect(streamType(streamStdout)).To(Equal("stdout"))
			streamStderror := <-newStreamChannel
			Expect(streamType(streamStderror)).To(Equal("stderr"))

			expected := []byte("Hello")
			err = ws.WriteMessage(websocket.BinaryMessage, expected)
			Expect(err).NotTo(HaveOccurred())

			obtained := make([]byte, len(expected))
			_, err = io.ReadFull(streamStdin, obtained)
			Expect(err).NotTo(HaveOccurred())
			Expect(obtained).To(Equal(expected))

			expected = []byte("World")
			_, err = streamStdout.Write(expected)
			Expect(err).NotTo(HaveOccurred())

			_, obtained, err = ws.ReadMessage()
			Expect(err).NotTo(HaveOccurred())
			Expect(obtained).To(Equal(expected))

			// TODO: Check error streams
			defer ws.Close()
			close(done)

		}, 5)

		It("should fail if VirtualMachine not exists", func(done Done) {
			request.PathParameters()["name"] = "testvm"
			request.PathParameters()["namespace"] = "default"

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/apis/kubevirt.io/v1alpha3/namespaces/default/virtualmachines/testvm"),
					ghttp.RespondWithJSONEncoded(http.StatusNotFound, nil),
				),
			)

			app.RestartVMRequestHandler(request, response)

			Expect(response.Error()).To(HaveOccurred())
			Expect(response.StatusCode()).To(Equal(http.StatusNotFound))
			close(done)
		}, 5)

		It("should fail if VirtualMachine is not in running state", func(done Done) {
			request.PathParameters()["name"] = "testvm"
			request.PathParameters()["namespace"] = "default"

			vm := v1.VirtualMachine{
				Spec: v1.VirtualMachineSpec{
					Running: &notRunning,
				},
			}

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/apis/kubevirt.io/v1alpha3/namespaces/default/virtualmachines/testvm"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, vm),
				),
			)

			app.RestartVMRequestHandler(request, response)

			Expect(response.Error()).To(HaveOccurred())
			Expect(response.StatusCode()).To(Equal(http.StatusBadRequest))
			close(done)
		})

		It("should restart VirtualMachine", func(done Done) {
			request.PathParameters()["name"] = "testvm"
			request.PathParameters()["namespace"] = "default"

			vm := v1.VirtualMachine{
				ObjectMeta: k8smetav1.ObjectMeta{
					Name: "testvm",
				},
				Spec: v1.VirtualMachineSpec{
					Running: &running,
				},
			}

			vmi := v1.VirtualMachineInstance{
				Spec: v1.VirtualMachineInstanceSpec{},
			}

			vmi.ObjectMeta.SetUID(uuid.NewUUID())

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/apis/kubevirt.io/v1alpha3/namespaces/default/virtualmachines/testvm"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, vm),
				),
			)

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/apis/kubevirt.io/v1alpha3/namespaces/default/virtualmachineinstances/testvm"),
					ghttp.RespondWithJSONEncoded(http.StatusNotFound, vmi),
				),
			)

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("PATCH", "/apis/kubevirt.io/v1alpha3/namespaces/default/virtualmachines/testvm"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, vm),
				),
			)

			app.RestartVMRequestHandler(request, response)

			Expect(response.Error()).ToNot(HaveOccurred())
			Expect(response.StatusCode()).To(Equal(http.StatusAccepted))
			close(done)
		})

		It("should start VirtualMachine if VMI doesn't exist", func(done Done) {
			request.PathParameters()["name"] = "testvm"
			request.PathParameters()["namespace"] = "default"

			vm := v1.VirtualMachine{
				ObjectMeta: k8smetav1.ObjectMeta{
					Name: "testvm",
				},
				Spec: v1.VirtualMachineSpec{
					Running: &running,
				},
			}

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/apis/kubevirt.io/v1alpha3/namespaces/default/virtualmachines/testvm"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, vm),
				),
			)

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/apis/kubevirt.io/v1alpha3/namespaces/default/virtualmachineinstances/testvm"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, nil),
				),
			)

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("PATCH", "/apis/kubevirt.io/v1alpha3/namespaces/default/virtualmachines/testvm"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, vm),
				),
			)

			app.RestartVMRequestHandler(request, response)

			Expect(response.Error()).NotTo(HaveOccurred())
			Expect(response.StatusCode()).To(Equal(http.StatusAccepted))
			close(done)
		})
	})

	Context("StateChange JSON", func() {
		It("should create a stop request if status exists", func() {
			uid := uuid.NewUUID()
			vm := newMinimalVM("testvm")
			vm.Status.Created = true
			stopRequest := v1.VirtualMachineStateChangeRequest{
				Action: v1.StopRequest,
				UID:    &uid,
			}

			res, err := getChangeRequestJson(vm, stopRequest)
			Expect(err).ToNot(HaveOccurred())

			ref := fmt.Sprintf(`[{ "op": "test", "path": "/status/stateChangeRequests", "value": null}, { "op": "add", "path": "/status/stateChangeRequests", "value": [{"action":"Stop","uid":"%s"}]}]`, uid)
			Expect(res).To(Equal(ref))
		})

		It("should create a stop request if status doesn't exist", func() {
			uid := uuid.NewUUID()
			vm := newMinimalVM("testvm")
			stopRequest := v1.VirtualMachineStateChangeRequest{
				Action: v1.StopRequest,
				UID:    &uid,
			}

			res, err := getChangeRequestJson(vm, stopRequest)
			Expect(err).ToNot(HaveOccurred())

			ref := fmt.Sprintf(`[{ "op": "add", "path": "/status", "value": {"stateChangeRequests":[{"action":"Stop","uid":"%s"}]}}]`, uid)
			Expect(res).To(Equal(ref))
		})

		It("should create a restart request if status exists", func() {
			uid := uuid.NewUUID()
			vm := newMinimalVM("testvm")
			vm.Status.Created = true
			stopRequest := v1.VirtualMachineStateChangeRequest{
				Action: v1.StopRequest,
				UID:    &uid,
			}
			startRequest := v1.VirtualMachineStateChangeRequest{
				Action: v1.StartRequest,
			}

			res, err := getChangeRequestJson(vm, stopRequest, startRequest)
			Expect(err).ToNot(HaveOccurred())

			ref := fmt.Sprintf(`[{ "op": "test", "path": "/status/stateChangeRequests", "value": null}, { "op": "add", "path": "/status/stateChangeRequests", "value": [{"action":"Stop","uid":"%s"},{"action":"Start"}]}]`, uid)
			Expect(res).To(Equal(ref))
		})

		It("should create a restart request if status doesn't exist", func() {
			uid := uuid.NewUUID()
			vm := newMinimalVM("testvm")
			stopRequest := v1.VirtualMachineStateChangeRequest{
				Action: v1.StopRequest,
				UID:    &uid,
			}
			startRequest := v1.VirtualMachineStateChangeRequest{
				Action: v1.StartRequest,
			}

			res, err := getChangeRequestJson(vm, stopRequest, startRequest)
			Expect(err).ToNot(HaveOccurred())

			ref := fmt.Sprintf(`[{ "op": "add", "path": "/status", "value": {"stateChangeRequests":[{"action":"Stop","uid":"%s"},{"action":"Start"}]}}]`, uid)
			Expect(res).To(Equal(ref))
		})

		It("should create a start request if status exists", func() {
			vm := newMinimalVM("testvm")
			vm.Status.Created = true

			startRequest := v1.VirtualMachineStateChangeRequest{
				Action: v1.StartRequest,
			}

			res, err := getChangeRequestJson(vm, startRequest)
			Expect(err).ToNot(HaveOccurred())

			ref := fmt.Sprintf(`[{ "op": "test", "path": "/status/stateChangeRequests", "value": null}, { "op": "add", "path": "/status/stateChangeRequests", "value": [{"action":"Start"}]}]`)
			Expect(res).To(Equal(ref))
		})

		It("should create a start request if status doesn't exist", func() {
			vm := newMinimalVM("testvm")

			startRequest := v1.VirtualMachineStateChangeRequest{
				Action: v1.StartRequest,
			}

			res, err := getChangeRequestJson(vm, startRequest)
			Expect(err).ToNot(HaveOccurred())

			ref := fmt.Sprintf(`[{ "op": "add", "path": "/status", "value": {"stateChangeRequests":[{"action":"Start"}]}}]`)
			Expect(res).To(Equal(ref))
		})

		It("should force a stop request to override", func() {
			uid := uuid.NewUUID()
			vm := newMinimalVM("testvm")
			stopRequest := v1.VirtualMachineStateChangeRequest{
				Action: v1.StopRequest,
				UID:    &uid,
			}
			startRequest := v1.VirtualMachineStateChangeRequest{
				Action: v1.StartRequest,
			}
			vm.Status.StateChangeRequests = append(vm.Status.StateChangeRequests, startRequest)

			res, err := getChangeRequestJson(vm, stopRequest)
			Expect(err).ToNot(HaveOccurred())

			ref := fmt.Sprintf(`[{ "op": "test", "path": "/status/stateChangeRequests", "value": [{"action":"Start"}]}, { "op": "replace", "path": "/status/stateChangeRequests", "value": [{"action":"Stop","uid":"%s"}]}]`, uid)
			Expect(res).To(Equal(ref))
		})

		It("should error on start request if other requests exist", func() {
			uid := uuid.NewUUID()
			vm := newMinimalVM("testvm")
			stopRequest := v1.VirtualMachineStateChangeRequest{
				Action: v1.StopRequest,
				UID:    &uid,
			}
			startRequest := v1.VirtualMachineStateChangeRequest{
				Action: v1.StartRequest,
			}
			vm.Status.StateChangeRequests = append(vm.Status.StateChangeRequests, stopRequest)

			_, err := getChangeRequestJson(vm, startRequest)
			Expect(err).To(HaveOccurred())
		})

		It("should error on restart request if other requests exist", func() {
			uid := uuid.NewUUID()
			vm := newMinimalVM("testvm")
			stopRequest := v1.VirtualMachineStateChangeRequest{
				Action: v1.StopRequest,
				UID:    &uid,
			}
			startRequest := v1.VirtualMachineStateChangeRequest{
				Action: v1.StartRequest,
			}
			vm.Status.StateChangeRequests = append(vm.Status.StateChangeRequests, startRequest)

			_, err := getChangeRequestJson(vm, stopRequest, startRequest)
			Expect(err).To(HaveOccurred())
		})
	})

	AfterEach(func() {
		server.Close()
		backend.Close()
	})
})

func newMinimalVM(name string) *v1.VirtualMachine {
	return &v1.VirtualMachine{TypeMeta: k8smetav1.TypeMeta{APIVersion: v1.GroupVersion.String(), Kind: "VirtualMachine"}, ObjectMeta: k8smetav1.ObjectMeta{Name: name}}
}
