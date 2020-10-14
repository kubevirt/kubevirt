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
	"bytes"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"

	"github.com/emicklei/go-restful"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"

	"kubevirt.io/kubevirt/pkg/util/status"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
)

const vmPathFormat = "/apis/kubevirt.io/%s/namespaces/%s/virtualmachines/%s"

type readCloserWrapper struct {
	io.Reader
}

func (b *readCloserWrapper) Close() error { return nil }

var _ = Describe("VirtualMachineInstance Subresources", func() {
	kubecli.Init()

	var server *ghttp.Server
	var backend *ghttp.Server
	var backendIP string
	var request *restful.Request
	var recorder *httptest.ResponseRecorder
	var response *restful.Response

	running := true
	notRunning := false

	log.Log.SetIOWriter(GinkgoWriter)

	app := SubresourceAPIApp{}
	BeforeEach(func() {
		server = ghttp.NewServer()
		backend = ghttp.NewTLSServer()
		backendAddr := strings.Split(backend.Addr(), ":")
		backendPort, err := strconv.Atoi(backendAddr[1])
		backendIP = backendAddr[0]
		Expect(err).ToNot(HaveOccurred())
		app.consoleServerPort = backendPort
		flag.Set("kubeconfig", "")
		flag.Set("master", server.URL())
		app.virtCli, _ = kubecli.GetKubevirtClientFromFlags(server.URL(), "")
		app.statusUpdater = status.NewVMStatusUpdater(app.virtCli)
		app.credentialsLock = &sync.Mutex{}
		app.handlerTLSConfiguration = &tls.Config{InsecureSkipVerify: true}

		request = restful.NewRequest(&http.Request{})
		recorder = httptest.NewRecorder()
		response = restful.NewResponse(recorder)
	})

	expectHandlerPod := func() {
		pod := &k8sv1.Pod{}
		pod.Labels = map[string]string{}
		pod.Labels[v1.AppLabel] = "virt-handler"
		pod.ObjectMeta.Name = "madeup-name"

		pod.Spec.NodeName = "mynode"
		pod.Status.Phase = k8sv1.PodRunning
		pod.Status.PodIP = backendIP

		podList := k8sv1.PodList{}
		podList.Items = []k8sv1.Pod{}
		podList.Items = append(podList.Items, *pod)

		server.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest("GET", "/api/v1/namespaces/kubevirt/pods"),
				ghttp.RespondWithJSONEncoded(http.StatusOK, podList),
			),
		)
	}

	expectVMI := func(running, paused bool) {
		request.PathParameters()["name"] = "testvmi"
		request.PathParameters()["namespace"] = "default"

		phase := v1.Running
		if !running {
			phase = v1.Failed
		}

		vmi := v1.VirtualMachineInstance{
			ObjectMeta: k8smetav1.ObjectMeta{
				Name:      "testvmi",
				Namespace: "default",
			},
			Status: v1.VirtualMachineInstanceStatus{
				Phase: phase,
			},
		}

		if paused {
			vmi.Status.Conditions = []v1.VirtualMachineInstanceCondition{
				{
					Type:   v1.VirtualMachineInstancePaused,
					Status: k8sv1.ConditionTrue,
				},
			}
		}

		server.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest("GET", "/apis/kubevirt.io/v1alpha3/namespaces/default/virtualmachineinstances/testvmi"),
				ghttp.RespondWithJSONEncoded(http.StatusOK, vmi),
			),
		)

		expectHandlerPod()
	}

	getVMPath := func(version, namespace, vmName string) string {
		return fmt.Sprintf(vmPathFormat, version, namespace, vmName)
	}

	Context("Subresource api", func() {
		It("should find matching pod for running VirtualMachineInstance", func(done Done) {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.Status.Phase = v1.Running
			vmi.ObjectMeta.SetUID(uuid.NewUUID())

			expectHandlerPod()

			result, err := app.getVirtHandlerConnForVMI(vmi)

			Expect(err).ToNot(HaveOccurred())
			ip, _, _ := result.ConnectionDetails()
			Expect(ip).To(Equal(backendIP))
			close(done)
		}, 5)

		It("should fail if VirtualMachineInstance is not in running state", func(done Done) {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.Status.Phase = v1.Succeeded
			vmi.ObjectMeta.SetUID(uuid.NewUUID())

			_, err := app.getVirtHandlerConnForVMI(vmi)

			Expect(err).To(HaveOccurred())
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
					ghttp.VerifyRequest("GET", "/api/v1/namespaces/kubevirt/pods"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, podList),
				),
			)

			conn, err := app.getVirtHandlerConnForVMI(vmi)
			Expect(err).ToNot(HaveOccurred())
			_, _, err = conn.ConnectionDetails()
			Expect(err).To(HaveOccurred())
			close(done)
		}, 5)

		It("should fail with no 'name' path param", func(done Done) {

			app.VNCRequestHandler(request, response)
			ExpectStatusErrorWithCode(recorder, http.StatusInternalServerError)
			close(done)
		}, 5)

		It("should fail with no 'namespace' path param", func(done Done) {

			request.PathParameters()["name"] = "testvmi"

			app.VNCRequestHandler(request, response)
			ExpectStatusErrorWithCode(recorder, http.StatusInternalServerError)
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
			ExpectStatusErrorWithCode(recorder, http.StatusNotFound)
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
			ExpectStatusErrorWithCode(recorder, http.StatusInternalServerError)
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
			ExpectStatusErrorWithCode(recorder, http.StatusBadRequest)
			close(done)
		}, 5)

		It("should fail to connect to VNC if the VMI is paused", func(done Done) {

			request.PathParameters()["name"] = "testvmi"
			request.PathParameters()["namespace"] = "default"

			expectVMI(true, true)

			app.VNCRequestHandler(request, response)
			ExpectStatusErrorWithCode(recorder, http.StatusConflict)
			close(done)
		}, 5)

		It("should fail with no serial console at console connections", func(done Done) {

			request.PathParameters()["name"] = "testvmi"
			request.PathParameters()["namespace"] = "default"

			flag := false
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.Status.Phase = v1.Running
			vmi.ObjectMeta.SetUID(uuid.NewUUID())
			vmi.Spec.Domain.Devices.AutoattachSerialConsole = &flag

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/apis/kubevirt.io/v1alpha3/namespaces/default/virtualmachineinstances/testvmi"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, vmi),
				),
			)

			app.ConsoleRequestHandler(request, response)
			ExpectStatusErrorWithCode(recorder, http.StatusBadRequest)
			close(done)
		}, 5)

		It("should fail to connect to the serial console if the VMI is paused", func(done Done) {

			request.PathParameters()["name"] = "testvmi"
			request.PathParameters()["namespace"] = "default"

			expectVMI(true, true)

			app.ConsoleRequestHandler(request, response)
			ExpectStatusErrorWithCode(recorder, http.StatusConflict)
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

			ExpectStatusErrorWithCode(recorder, http.StatusNotFound)
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

			status := ExpectStatusErrorWithCode(recorder, http.StatusConflict)
			// check the msg string that would be presented to virtctl output
			Expect(status.Error()).To(ContainSubstring("Halted does not support manual restart requests"))
			close(done)
		})

		It("should ForceRestart VirtualMachine", func(done Done) {
			request.PathParameters()["name"] = "testvm"
			request.PathParameters()["namespace"] = "default"

			body := map[string]int64{
				"gracePeriodSeconds": 0,
			}
			bytesRepresentation, _ := json.Marshal(body)
			request.Request.Body = ioutil.NopCloser(bytes.NewReader(bytesRepresentation))

			vm := v1.VirtualMachine{
				ObjectMeta: k8smetav1.ObjectMeta{
					Name:      "testvm",
					Namespace: "default",
				},
				Spec: v1.VirtualMachineSpec{
					Running: &running,
				},
			}
			vmi := v1.VirtualMachineInstance{
				Spec: v1.VirtualMachineInstanceSpec{},
			}
			vmi.ObjectMeta.SetUID(uuid.NewUUID())

			pod := &k8sv1.Pod{}
			pod.Labels = map[string]string{}
			pod.Annotations = map[string]string{}
			pod.Labels[v1.AppLabel] = "virt-launcher"
			pod.ObjectMeta.Name = "virt-launcher-testvm"
			pod.Spec.NodeName = "mynode"
			pod.Status.Phase = k8sv1.PodRunning
			pod.Status.PodIP = "10.35.1.1"
			pod.Labels[v1.CreatedByLabel] = string(vmi.UID)
			pod.Annotations[v1.DomainAnnotation] = vm.Name

			podList := k8sv1.PodList{}
			podList.Items = []k8sv1.Pod{}
			podList.Items = append(podList.Items, *pod)

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/apis/kubevirt.io/v1alpha3/namespaces/default/virtualmachines/testvm"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, vm),
				),
			)

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/apis/kubevirt.io/v1alpha3/namespaces/default/virtualmachineinstances/testvm"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, vmi),
				),
			)

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("PATCH", "/apis/kubevirt.io/v1alpha3/namespaces/default/virtualmachines/testvm/status"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, vm),
				),
			)

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/api/v1/namespaces/default/pods"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, podList),
				),
			)

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("DELETE", "/api/v1/namespaces/default/pods/virt-launcher-testvm"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, vm),
				),
			)

			app.RestartVMRequestHandler(request, response)

			Expect(response.Error()).ToNot(HaveOccurred())
			Expect(response.StatusCode()).To(Equal(http.StatusAccepted))
			close(done)
		})

		It("should not ForceRestart VirtualMachine if no Pods found for the VMI", func(done Done) {
			request.PathParameters()["name"] = "testvm"
			request.PathParameters()["namespace"] = "default"

			body := map[string]int64{
				"gracePeriodSeconds": 0,
			}
			bytesRepresentation, _ := json.Marshal(body)
			request.Request.Body = ioutil.NopCloser(bytes.NewReader(bytesRepresentation))

			vm := v1.VirtualMachine{
				ObjectMeta: k8smetav1.ObjectMeta{
					Name:      "testvm",
					Namespace: "default",
				},
				Spec: v1.VirtualMachineSpec{
					Running: &running,
				},
			}
			vmi := v1.VirtualMachineInstance{
				Spec: v1.VirtualMachineInstanceSpec{},
			}
			vmi.ObjectMeta.SetUID(uuid.NewUUID())

			podList := k8sv1.PodList{}
			podList.Items = []k8sv1.Pod{}

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/apis/kubevirt.io/v1alpha3/namespaces/default/virtualmachines/testvm"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, vm),
				),
			)

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/apis/kubevirt.io/v1alpha3/namespaces/default/virtualmachineinstances/testvm"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, vmi),
				),
			)

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("PATCH", "/apis/kubevirt.io/v1alpha3/namespaces/default/virtualmachines/testvm/status"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, vm),
				),
			)

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/api/v1/namespaces/default/pods"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, podList),
				),
			)

			app.RestartVMRequestHandler(request, response)

			Expect(response.Error()).ToNot(HaveOccurred())
			Expect(response.StatusCode()).To(Equal(http.StatusAccepted))
			close(done)
		})

		It("should restart VirtualMachine", func(done Done) {
			request.PathParameters()["name"] = "testvm"
			request.PathParameters()["namespace"] = "default"

			vm := v1.VirtualMachine{
				ObjectMeta: k8smetav1.ObjectMeta{
					Name:      "testvm",
					Namespace: "default",
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
					ghttp.RespondWithJSONEncoded(http.StatusOK, vmi),
				),
			)

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("PATCH", "/apis/kubevirt.io/v1alpha3/namespaces/default/virtualmachines/testvm/status"),
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
					Name:      "testvm",
					Namespace: "default",
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
					ghttp.VerifyRequest("PATCH", "/apis/kubevirt.io/v1alpha3/namespaces/default/virtualmachines/testvm/status"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, vm),
				),
			)

			app.RestartVMRequestHandler(request, response)

			Expect(response.Error()).NotTo(HaveOccurred())
			Expect(response.StatusCode()).To(Equal(http.StatusAccepted))
			close(done)
		})

		Context("Subresource api - rename", func() {
			var (
				vm      *v1.VirtualMachine
				newName string
			)

			newRenameBody := func(newName string) io.ReadCloser {
				renameOpts := &v1.RenameOptions{NewName: newName}

				optsJson, _ := json.Marshal(renameOpts)

				return &readCloserWrapper{bytes.NewReader(optsJson)}
			}

			BeforeEach(func() {
				vm = newMinimalVM("renametest")
				vm.Namespace = "renametestns"

				request.PathParameters()["name"] = vm.Name
				request.PathParameters()["namespace"] = vm.Namespace

				newName = vm.Name + "new"
			})

			Context("Without VM mocking", func() {
				It("should fail if no name is provided in the request body", func(done Done) {
					app.RenameVMRequestHandler(request, response)

					status := ExpectStatusErrorWithCode(recorder, http.StatusBadRequest)
					Expect(status.Error()).To(ContainSubstring("no body"))

					close(done)
				})

				It("should fail if the new name is empty", func(done Done) {
					request.Request.Body = newRenameBody("")

					app.RenameVMRequestHandler(request, response)

					status := ExpectStatusErrorWithCode(recorder, http.StatusBadRequest)
					Expect(status.Error()).To(ContainSubstring("Please provide a new name for the VM"))
					close(done)
				})

				It("should fail if the new name is identical to the current name", func(done Done) {
					request.Request.Body = newRenameBody(vm.Name)

					app.RenameVMRequestHandler(request, response)

					status := ExpectStatusErrorWithCode(recorder, http.StatusBadRequest)
					Expect(status.Error()).To(ContainSubstring("identical"))

					close(done)
				})
			})

			Context("With source VM mocking", func() {
				BeforeEach(func() {
					vmGetStatus := http.StatusOK
					request.Request.Body = newRenameBody(newName)

					server.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("GET", getVMPath("v1alpha3", vm.Namespace, vm.Name)),
							ghttp.RespondWithJSONEncodedPtr(&vmGetStatus, vm),
						),
					)
				})

				Context("With invalid source VM running status", func() {
					BeforeEach(func() {
						running := true
						vm.Spec.Running = &running
					})

					It("should fail if the VM is running", func(done Done) {
						app.RenameVMRequestHandler(request, response)

						status := ExpectStatusErrorWithCode(recorder, http.StatusBadRequest)
						Expect(status.Error()).To(ContainSubstring("running VM is not allowed"))

						close(done)
					})
				})

				Context("With invalid source VM runStrategy applied", func() {
					BeforeEach(func() {
						runStrategy := v1.RunStrategyManual
						vm.Spec.RunStrategy = &runStrategy
					})

					It("should fail if non-halted run strategy is applied", func(done Done) {
						app.RenameVMRequestHandler(request, response)

						status := ExpectStatusErrorWithCode(recorder, http.StatusBadRequest)
						Expect(status.Error()).To(ContainSubstring("Renaming a running VM is not allowed"))

						close(done)
					})
				})

				Context("With valid running status or runStrategy", func() {
					BeforeEach(func() {
						running := false
						vm.Spec.Running = &running
					})

					Context("With destination VM mocking", func() {
						var (
							newVM *v1.VirtualMachine
						)

						BeforeEach(func() {
							newVM = newMinimalVM(newName)
							newVM.Namespace = vm.Namespace

							server.AppendHandlers(
								ghttp.CombineHandlers(
									ghttp.VerifyRequest("GET", getVMPath("v1alpha3", newVM.Namespace, newVM.Name)),
									ghttp.RespondWithJSONEncoded(http.StatusOK, newVM),
								),
							)
						})

						It("should fail if the new name is already taken", func(done Done) {
							request.Request.Body = newRenameBody(newVM.Name)

							app.RenameVMRequestHandler(request, response)

							status := ExpectStatusErrorWithCode(recorder, http.StatusBadRequest)
							Expect(status.Error()).To(ContainSubstring("already exists"))

							close(done)
						})
					})

					Context("When a VM with the new name does not exist", func() {
						BeforeEach(func() {
							server.AppendHandlers(
								ghttp.CombineHandlers(
									ghttp.VerifyRequest("GET", getVMPath("v1alpha3", vm.Namespace, newName)),
									ghttp.RespondWith(http.StatusNotFound, nil),
								),
							)
						})

						Context("With vm patch mocking", func() {
							Context("With failing patch", func() {
								BeforeEach(func() {
									server.AppendHandlers(
										ghttp.CombineHandlers(
											ghttp.VerifyRequest("PATCH", getVMPath("v1alpha3", vm.Namespace, vm.Name)+"/status"),
											ghttp.RespondWith(http.StatusInternalServerError, nil),
										),
									)
								})

								It("should fail due to failed patch", func(done Done) {
									app.RenameVMRequestHandler(request, response)

									ExpectStatusErrorWithCode(recorder, http.StatusInternalServerError)

									close(done)
								})
							})

							Context("With successfull patch", func() {
								BeforeEach(func() {
									vmPatchStatus := http.StatusOK

									patchedVM := vm.DeepCopy()
									patchedVM.Status = v1.VirtualMachineStatus{
										StateChangeRequests: []v1.VirtualMachineStateChangeRequest{
											{
												Action: v1.RenameRequest,
												Data: map[string]string{
													"newName": newName,
												},
											},
										},
									}

									server.AppendHandlers(
										ghttp.CombineHandlers(
											ghttp.VerifyRequest("PATCH", getVMPath("v1alpha3", vm.Namespace, vm.Name)+"/status"),
											ghttp.RespondWithJSONEncodedPtr(&vmPatchStatus, patchedVM),
										),
									)
								})

								It("should succeed", func(done Done) {
									request.Request.Body = newRenameBody(newName)

									app.RenameVMRequestHandler(request, response)

									Expect(response.StatusCode()).To(Equal(http.StatusAccepted))

									close(done)
								})
							})
						})
					})
				})
			})
		})
	})

	Context("Subresource api - error handling for RestartVMRequestHandler", func() {
		BeforeEach(func() {
			request.PathParameters()["name"] = "testvm"
			request.PathParameters()["namespace"] = "default"
		})

		It("should fail on VM with RunStrategyHalted", func() {
			vm := newVirtualMachineWithRunStrategy(v1.RunStrategyHalted)

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/apis/kubevirt.io/v1alpha3/namespaces/default/virtualmachines/testvm"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, vm),
				),
			)

			app.RestartVMRequestHandler(request, response)

			status := ExpectStatusErrorWithCode(recorder, http.StatusConflict)
			// check the msg string that would be presented to virtctl output
			Expect(status.Error()).To(ContainSubstring("Halted does not support manual restart requests"))
		})

		table.DescribeTable("should not fail with VMI and RunStrategy", func(runStrategy v1.VirtualMachineRunStrategy) {
			vm := newVirtualMachineWithRunStrategy(runStrategy)
			vmi := newVirtualMachineInstanceInPhase(v1.Failed)

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/apis/kubevirt.io/v1alpha3/namespaces/default/virtualmachines/testvm"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, vm),
				),
			)

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/apis/kubevirt.io/v1alpha3/namespaces/default/virtualmachineinstances/testvm"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, vmi),
				),
			)

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("PATCH", "/apis/kubevirt.io/v1alpha3/namespaces/default/virtualmachines/testvm/status"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, vm),
				),
			)

			app.RestartVMRequestHandler(request, response)

			Expect(response.StatusCode()).To(Equal(http.StatusAccepted))
		},
			table.Entry("Always", v1.RunStrategyAlways),
			table.Entry("Manual", v1.RunStrategyManual),
			table.Entry("RerunOnFailure", v1.RunStrategyRerunOnFailure),
		)

		table.DescribeTable("should fail anytime without VMI and RunStrategy", func(runStrategy v1.VirtualMachineRunStrategy, msg string) {
			vm := newVirtualMachineWithRunStrategy(runStrategy)

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/apis/kubevirt.io/v1alpha3/namespaces/default/virtualmachines/testvm"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, vm),
				),
			)

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/apis/kubevirt.io/v1alpha3/namespaces/default/virtualmachineinstances/testvm"),
					ghttp.RespondWithJSONEncoded(http.StatusNotFound, nil),
				),
			)

			app.RestartVMRequestHandler(request, response)

			status := ExpectStatusErrorWithCode(recorder, http.StatusConflict)
			// check the msg string that would be presented to virtctl output
			Expect(status.Error()).To(ContainSubstring(msg))
		},
			table.Entry("Always", v1.RunStrategyAlways, "VM is not running"),
			table.Entry("Manual", v1.RunStrategyManual, "VM is not running"),
			table.Entry("RerunOnFailure", v1.RunStrategyRerunOnFailure, "VM is not running"),
			table.Entry("Halted", v1.RunStrategyHalted, "Halted does not support manual restart requests"),
		)

		It("should fail on a VM that is scheduled to be renamed", func() {
			vm := newMinimalVM(request.PathParameter("name"))
			vm.Status.StateChangeRequests = []v1.VirtualMachineStateChangeRequest{
				{
					Action: v1.RenameRequest,
					Data: map[string]string{
						"newName": "newvm",
					},
				},
			}

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", getVMPath("v1alpha3", k8sv1.NamespaceDefault, vm.Name)),
					ghttp.RespondWithJSONEncoded(http.StatusOK, vm),
				),
			)

			app.RestartVMRequestHandler(request, response)
			status := ExpectStatusErrorWithCode(recorder, http.StatusBadRequest)
			// check the msg string that would be presented to virtctl output
			Expect(status.Error()).To(ContainSubstring("rename"))
		})
	})

	Context("Subresource api - error handling for StartVMRequestHandler", func() {
		BeforeEach(func() {
			request.PathParameters()["name"] = "testvm"
			request.PathParameters()["namespace"] = "default"
		})

		table.DescribeTable("should fail on VM with RunStrategy",
			func(runStrategy v1.VirtualMachineRunStrategy, phase v1.VirtualMachineInstancePhase, status int, msg string) {
				vm := newVirtualMachineWithRunStrategy(runStrategy)
				var vmi *v1.VirtualMachineInstance
				if phase != v1.VmPhaseUnset {
					vmi = newVirtualMachineInstanceInPhase(phase)
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
						ghttp.RespondWithJSONEncoded(status, vmi),
					),
				)

				app.StartVMRequestHandler(request, response)

				statusErr := ExpectStatusErrorWithCode(recorder, http.StatusConflict)
				// check the msg string that would be presented to virtctl output
				Expect(statusErr.Error()).To(ContainSubstring(msg))
			},
			table.Entry("Always without VMI", v1.RunStrategyAlways, v1.VmPhaseUnset, http.StatusNotFound, "Always does not support manual start requests"),
			table.Entry("Always with VMI in phase Running", v1.RunStrategyAlways, v1.Running, http.StatusOK, "VM is already running"),
			table.Entry("RerunOnFailure with VMI in phase Failed", v1.RunStrategyRerunOnFailure, v1.Failed, http.StatusOK, "RerunOnFailure does not support starting VM from failed state"),
		)

		table.DescribeTable("should not fail on VM with RunStrategy ",
			func(runStrategy v1.VirtualMachineRunStrategy, phase v1.VirtualMachineInstancePhase, status int) {
				vm := newVirtualMachineWithRunStrategy(runStrategy)
				var vmi *v1.VirtualMachineInstance
				if phase != v1.VmPhaseUnset {
					vmi = newVirtualMachineInstanceInPhase(phase)
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
						ghttp.RespondWithJSONEncoded(http.StatusOK, vmi),
					),
				)

				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("PATCH", "/apis/kubevirt.io/v1alpha3/namespaces/default/virtualmachines/testvm/status"),
						ghttp.RespondWithJSONEncoded(http.StatusOK, vm),
					),
				)

				app.StartVMRequestHandler(request, response)

				Expect(response.StatusCode()).To(Equal(http.StatusAccepted))
			},
			table.Entry("RerunOnFailure with VMI in state Succeeded", v1.RunStrategyRerunOnFailure, v1.Succeeded, http.StatusOK),
			table.Entry("Manual with VMI in state Succeeded", v1.RunStrategyManual, v1.Succeeded, http.StatusOK),
			table.Entry("Manual with VMI in state Failed", v1.RunStrategyManual, v1.Failed, http.StatusOK),
		)

		It("should fail on a VM that is scheduled to be renamed", func() {
			vm := newMinimalVM(request.PathParameter("name"))
			vm.Status.StateChangeRequests = []v1.VirtualMachineStateChangeRequest{
				{
					Action: v1.RenameRequest,
					Data: map[string]string{
						"newName": "newvm",
					},
				},
			}

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", getVMPath("v1alpha3", k8sv1.NamespaceDefault, vm.Name)),
					ghttp.RespondWithJSONEncoded(http.StatusOK, vm),
				),
			)

			app.StartVMRequestHandler(request, response)
			status := ExpectStatusErrorWithCode(recorder, http.StatusBadRequest)
			// check the msg string that would be presented to virtctl output
			Expect(status.Error()).To(ContainSubstring("rename"))
		})
	})

	Context("Subresource api - error handling for StopVMRequestHandler", func() {
		BeforeEach(func() {
			request.PathParameters()["name"] = "testvm"
			request.PathParameters()["namespace"] = "default"
		})

		table.DescribeTable("should fail with any strategy if VMI does not exist", func(runStrategy v1.VirtualMachineRunStrategy, msg string) {
			vm := newVirtualMachineWithRunStrategy(runStrategy)

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/apis/kubevirt.io/v1alpha3/namespaces/default/virtualmachines/testvm"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, vm),
				),
			)

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/apis/kubevirt.io/v1alpha3/namespaces/default/virtualmachineinstances/testvm"),
					ghttp.RespondWithJSONEncoded(http.StatusNotFound, nil),
				),
			)

			app.StopVMRequestHandler(request, response)

			statusErr := ExpectStatusErrorWithCode(recorder, http.StatusConflict)
			// check the msg string that would be presented to virtctl output
			Expect(statusErr.Error()).To(ContainSubstring(msg))
		},
			table.Entry("RunStrategyAlways", v1.RunStrategyAlways, "VM is not running"),
			table.Entry("RunStrategyManual", v1.RunStrategyManual, "VM is not running"),
			table.Entry("RunStrategyRerunOnFailure", v1.RunStrategyRerunOnFailure, "VM is not running"),
			table.Entry("RunStrategyHalted", v1.RunStrategyHalted, "VM is not running"),
		)

		It("should fail on VM with RunStrategyHalted", func() {
			vm := newVirtualMachineWithRunStrategy(v1.RunStrategyHalted)
			vmi := newVirtualMachineInstanceInPhase(v1.Unknown)

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/apis/kubevirt.io/v1alpha3/namespaces/default/virtualmachines/testvm"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, vm),
				),
			)

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/apis/kubevirt.io/v1alpha3/namespaces/default/virtualmachineinstances/testvm"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, vmi),
				),
			)

			app.StopVMRequestHandler(request, response)

			statusErr := ExpectStatusErrorWithCode(recorder, http.StatusConflict)
			// check the msg string that would be presented to virtctl output
			Expect(statusErr.Error()).To(ContainSubstring("VM is not running"))
		})

		table.DescribeTable("should not fail on VM with RunStrategy", func(runStrategy v1.VirtualMachineRunStrategy) {
			vm := newVirtualMachineWithRunStrategy(runStrategy)
			vmi := newVirtualMachineInstanceInPhase(v1.Running)

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/apis/kubevirt.io/v1alpha3/namespaces/default/virtualmachines/testvm"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, vm),
				),
			)

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/apis/kubevirt.io/v1alpha3/namespaces/default/virtualmachineinstances/testvm"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, vmi),
				),
			)

			if runStrategy == v1.RunStrategyManual {
				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("PATCH", "/apis/kubevirt.io/v1alpha3/namespaces/default/virtualmachines/testvm/status"),
						ghttp.RespondWithJSONEncoded(http.StatusOK, vm),
					),
				)
			} else {
				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("PATCH", "/apis/kubevirt.io/v1alpha3/namespaces/default/virtualmachines/testvm"),
						ghttp.RespondWithJSONEncoded(http.StatusOK, vm),
					),
				)
			}

			app.StopVMRequestHandler(request, response)

			Expect(response.StatusCode()).To(Equal(http.StatusAccepted))
		},
			table.Entry("Always", v1.RunStrategyAlways),
			table.Entry("RerunOnFailure", v1.RunStrategyRerunOnFailure),
			table.Entry("Manual", v1.RunStrategyManual),
		)
	})

	Context("Subresource api - MigrateVMRequestHandler", func() {
		It("should fail if VirtualMachine not exists", func(done Done) {
			request.PathParameters()["name"] = "testvm"
			request.PathParameters()["namespace"] = "default"

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/apis/kubevirt.io/v1alpha3/namespaces/default/virtualmachines/testvm"),
					ghttp.RespondWithJSONEncoded(http.StatusNotFound, nil),
				),
			)

			app.MigrateVMRequestHandler(request, response)

			ExpectStatusErrorWithCode(recorder, http.StatusNotFound)
			close(done)
		}, 5)

		It("should fail if VirtualMachine is not running", func(done Done) {
			request.PathParameters()["name"] = "testvm"
			request.PathParameters()["namespace"] = "default"

			vm := v1.VirtualMachine{}

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/apis/kubevirt.io/v1alpha3/namespaces/default/virtualmachines/testvm"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, vm),
				),
			)

			app.MigrateVMRequestHandler(request, response)

			status := ExpectStatusErrorWithCode(recorder, http.StatusConflict)
			Expect(status.Error()).To(ContainSubstring("VM is not running"))
			close(done)
		})

		It("should fail if migration is not posted", func(done Done) {
			request.PathParameters()["name"] = "testvm"
			request.PathParameters()["namespace"] = "default"

			vm := v1.VirtualMachine{
				Status: v1.VirtualMachineStatus{
					Ready: true,
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
					ghttp.VerifyRequest("POST", "/apis/kubevirt.io/v1alpha3/namespaces/default/virtualmachineinstancemigrations"),
					ghttp.RespondWithJSONEncoded(http.StatusInternalServerError, nil),
				),
			)

			app.MigrateVMRequestHandler(request, response)

			ExpectStatusErrorWithCode(recorder, http.StatusInternalServerError)
			close(done)
		})

		It("should migrate VirtualMachine", func(done Done) {
			request.PathParameters()["name"] = "testvm"
			request.PathParameters()["namespace"] = "default"

			vm := v1.VirtualMachine{
				Status: v1.VirtualMachineStatus{
					Ready: true,
				},
			}

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/apis/kubevirt.io/v1alpha3/namespaces/default/virtualmachines/testvm"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, vm),
				),
			)

			migration := v1.VirtualMachineInstanceMigration{}

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/apis/kubevirt.io/v1alpha3/namespaces/default/virtualmachineinstancemigrations"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, migration),
				),
			)

			app.MigrateVMRequestHandler(request, response)

			Expect(response.Error()).ToNot(HaveOccurred())
			Expect(response.StatusCode()).To(Equal(http.StatusAccepted))
			close(done)
		})
	})

	Context("Subresource api - Guest OS Info", func() {
		type subRes func(request *restful.Request, response *restful.Response)

		table.DescribeTable("should fail when the VMI does not exist", func(fn subRes) {
			request.PathParameters()["name"] = "testvm"
			request.PathParameters()["namespace"] = "default"

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/apis/kubevirt.io/v1alpha3/namespaces/default/virtualmachineinstances/testvm"),
					ghttp.RespondWithJSONEncoded(http.StatusNotFound, nil),
				),
			)

			fn(request, response)

			Expect(response.Error()).To(HaveOccurred(), "Response should indicate VM not found")
			Expect(response.StatusCode()).To(Equal(http.StatusInternalServerError))

		},
			table.Entry("for GuestOSInfo", app.GuestOSInfo),
			table.Entry("for UserList", app.UserList),
			table.Entry("for Filesystem", app.FilesystemList),
		)

		table.DescribeTable("should fail when the VMI is not running", func(fn subRes) {
			request.PathParameters()["name"] = "testvm"
			request.PathParameters()["namespace"] = "default"

			vmi := v1.VirtualMachineInstance{}

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/apis/kubevirt.io/v1alpha3/namespaces/default/virtualmachineinstances/testvm"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, vmi),
				),
			)

			fn(request, response)

			Expect(response.Error()).To(HaveOccurred())
			Expect(response.StatusCode()).To(Equal(http.StatusInternalServerError))
			Expect(response.Error().Error()).To(ContainSubstring("VMI is not running"))
		},
			table.Entry("for GuestOSInfo", app.GuestOSInfo),
			table.Entry("for UserList", app.UserList),
			table.Entry("for FilesystemList", app.FilesystemList),
		)

		table.DescribeTable("should fail when VMI does not have agent connected", func(fn subRes) {
			request.PathParameters()["name"] = "testvm"
			request.PathParameters()["namespace"] = "default"

			vmi := v1.VirtualMachineInstance{
				Status: v1.VirtualMachineInstanceStatus{
					Phase:      v1.Running,
					Conditions: []v1.VirtualMachineInstanceCondition{},
				},
			}

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/apis/kubevirt.io/v1alpha3/namespaces/default/virtualmachineinstances/testvm"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, vmi),
				),
			)

			fn(request, response)

			Expect(response.Error()).To(HaveOccurred())
			Expect(response.StatusCode()).To(Equal(http.StatusInternalServerError))
			Expect(response.Error().Error()).To(ContainSubstring("VMI does not have guest agent connected"))
		},
			table.Entry("for GuestOSInfo", app.GuestOSInfo),
			table.Entry("for UserList", app.UserList),
			table.Entry("for FilesystemList", app.FilesystemList),
		)
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

	Context("Pausing", func() {
		It("Should pause a running, not paused VMI", func() {

			backend.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("PUT", "/v1/namespaces/default/virtualmachineinstances/testvmi/pause"),
					ghttp.RespondWith(http.StatusOK, ""),
				),
			)
			expectVMI(true, false)

			app.PauseVMIRequestHandler(request, response)

			Expect(response.StatusCode()).To(Equal(http.StatusOK))
		})

		It("Should fail pausing a not running VMI", func() {

			expectVMI(false, false)

			app.PauseVMIRequestHandler(request, response)

			ExpectStatusErrorWithCode(recorder, http.StatusConflict)
		})

		It("Should fail pausing a running but paused VMI", func() {

			expectVMI(true, true)

			app.PauseVMIRequestHandler(request, response)

			ExpectStatusErrorWithCode(recorder, http.StatusConflict)
		})

		It("Should fail unpausing a running, not paused VMI", func() {

			expectVMI(true, false)

			app.UnpauseVMIRequestHandler(request, response)

			ExpectStatusErrorWithCode(recorder, http.StatusConflict)
		})

		It("Should fail unpausing a not running VMI", func() {

			expectVMI(false, false)

			app.UnpauseVMIRequestHandler(request, response)

			ExpectStatusErrorWithCode(recorder, http.StatusConflict)
		})

		It("Should unpause a running, paused VMI", func() {
			backend.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("PUT", "/v1/namespaces/default/virtualmachineinstances/testvmi/unpause"),
					ghttp.RespondWith(http.StatusOK, ""),
				),
			)
			expectVMI(true, true)

			app.UnpauseVMIRequestHandler(request, response)

			Expect(response.StatusCode()).To(Equal(http.StatusOK))
		})
	})

	AfterEach(func() {
		server.Close()
		backend.Close()
	})
})

func newVirtualMachineWithRunStrategy(runStrategy v1.VirtualMachineRunStrategy) *v1.VirtualMachine {
	return &v1.VirtualMachine{
		ObjectMeta: k8smetav1.ObjectMeta{
			Name:      "testvm",
			Namespace: "default",
		},
		Spec: v1.VirtualMachineSpec{
			RunStrategy: &runStrategy,
		},
	}
}

func newVirtualMachineInstanceInPhase(phase v1.VirtualMachineInstancePhase) *v1.VirtualMachineInstance {
	virtualMachineInstance := v1.VirtualMachineInstance{
		Spec:   v1.VirtualMachineInstanceSpec{},
		Status: v1.VirtualMachineInstanceStatus{Phase: phase},
	}
	virtualMachineInstance.ObjectMeta.SetUID(uuid.NewUUID())
	return &virtualMachineInstance
}

func newMinimalVM(name string) *v1.VirtualMachine {
	return &v1.VirtualMachine{TypeMeta: k8smetav1.TypeMeta{APIVersion: v1.GroupVersion.String(), Kind: "VirtualMachine"}, ObjectMeta: k8smetav1.ObjectMeta{Name: name}}
}

func ExpectStatusErrorWithCode(recorder *httptest.ResponseRecorder, code int) *errors.StatusError {
	status := k8smetav1.Status{}
	err := json.Unmarshal(recorder.Body.Bytes(), &status)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	ExpectWithOffset(1, status.Kind).To(Equal("Status"))
	ExpectWithOffset(1, status.Code).To(BeNumerically("==", code))
	ExpectWithOffset(1, recorder.Code).To(BeNumerically("==", code))
	return &errors.StatusError{ErrStatus: status}
}
