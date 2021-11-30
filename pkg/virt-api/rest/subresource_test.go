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

	"kubevirt.io/client-go/api"

	"kubevirt.io/kubevirt/pkg/util/status"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/pkg/testutils"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

const (
	vmPathFormat = "/apis/kubevirt.io/%s/namespaces/%s/virtualmachines/%s"
	Running      = true
	Paused       = true
	NotRunning   = false
	UnPaused     = false
)

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

	kv := &v1.KubeVirt{
		ObjectMeta: k8smetav1.ObjectMeta{
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

	config, _, kvInformer := testutils.NewFakeClusterConfigUsingKV(kv)

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
		app.clusterConfig = config

		request = restful.NewRequest(&http.Request{})
		recorder = httptest.NewRecorder()
		response = restful.NewResponse(recorder)
	})

	enableFeatureGate := func(featureGate string) {
		kvConfig := kv.DeepCopy()
		kvConfig.Spec.Configuration.DeveloperConfiguration.FeatureGates = []string{featureGate}
		testutils.UpdateFakeKubeVirtClusterConfig(kvInformer, kvConfig)
	}
	disableFeatureGates := func() {
		testutils.UpdateFakeKubeVirtClusterConfig(kvInformer, kv)
	}

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

	Context("Subresource api", func() {
		It("should find matching pod for running VirtualMachineInstance", func(done Done) {
			vmi := api.NewMinimalVMI("testvmi")
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
			vmi := api.NewMinimalVMI("testvmi")
			vmi.Status.Phase = v1.Succeeded
			vmi.ObjectMeta.SetUID(uuid.NewUUID())

			_, err := app.getVirtHandlerConnForVMI(vmi)

			Expect(err).To(HaveOccurred())
			close(done)
		}, 5)

		It("should fail no matching pod is found", func(done Done) {
			vmi := api.NewMinimalVMI("testvmi")
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

		Context("VNC", func() {
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
				vmi := api.NewMinimalVMI("testvmi")
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

		})

		Context("PortForward", func() {
			It("should fail with no 'name' path param", func(done Done) {

				app.PortForwardRequestHandler(app.FetchVirtualMachineInstance)(request, response)
				ExpectStatusErrorWithCode(recorder, http.StatusInternalServerError)
				close(done)
			}, 5)

			It("should fail with no 'namespace' path param", func(done Done) {

				request.PathParameters()["name"] = "testvmi"

				app.PortForwardRequestHandler(app.FetchVirtualMachineInstance)(request, response)
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

				app.PortForwardRequestHandler(app.FetchVirtualMachineInstance)(request, response)
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

				app.PortForwardRequestHandler(app.FetchVirtualMachineInstance)(request, response)
				ExpectStatusErrorWithCode(recorder, http.StatusInternalServerError)
				close(done)
			}, 5)

		})

		Context("console", func() {
			It("should fail with no serial console at console connections", func(done Done) {

				request.PathParameters()["name"] = "testvmi"
				request.PathParameters()["namespace"] = "default"

				flag := false
				vmi := api.NewMinimalVMI("testvmi")
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

			It("should fail to connect to the serial console if the VMI is Failed", func(done Done) {

				request.PathParameters()["name"] = "testvmi"
				request.PathParameters()["namespace"] = "default"

				expectVMI(NotRunning, UnPaused)

				app.ConsoleRequestHandler(request, response)
				ExpectStatusErrorWithCode(recorder, http.StatusConflict)
				close(done)
			}, 5)

		})

		Context("restart", func() {
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
				request.Request.Body = io.NopCloser(bytes.NewReader(bytesRepresentation))

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
				request.Request.Body = io.NopCloser(bytes.NewReader(bytesRepresentation))

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
		})

		Context("stop", func() {
			table.DescribeTable("should ForceStop VirtualMachine", func(statusPhase v1.VirtualMachineInstancePhase) {
				request.PathParameters()["name"] = "testvm"
				request.PathParameters()["namespace"] = "default"
				var terminationGracePeriodSeconds int64 = 1800

				body := map[string]int64{
					"gracePeriod": 0,
				}
				bytesRepresentation, _ := json.Marshal(body)
				request.Request.Body = io.NopCloser(bytes.NewReader(bytesRepresentation))

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
					ObjectMeta: k8smetav1.ObjectMeta{
						Name:      "testvm",
						Namespace: "default",
					},
					Spec: v1.VirtualMachineInstanceSpec{
						TerminationGracePeriodSeconds: &terminationGracePeriodSeconds,
					},
					Status: v1.VirtualMachineInstanceStatus{
						Phase: statusPhase,
					},
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

				if statusPhase != v1.Failed {
					server.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("PATCH", "/apis/kubevirt.io/v1alpha3/namespaces/default/virtualmachineinstances/testvm"),
							ghttp.RespondWithJSONEncoded(http.StatusOK, vmi),
						),
					)
				}

				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("PATCH", "/apis/kubevirt.io/v1alpha3/namespaces/default/virtualmachines/testvm"),
						ghttp.RespondWithJSONEncoded(http.StatusOK, vm),
					),
				)

				app.StopVMRequestHandler(request, response)

				Expect(response.Error()).ToNot(HaveOccurred())
				Expect(response.StatusCode()).To(Equal(http.StatusAccepted))
			},
				table.Entry("in status Running", v1.Running),
				table.Entry("in status Failed", v1.Failed),
			)
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

		table.DescribeTable("should fail anytime without VMI and RunStrategy", func(runStrategy v1.VirtualMachineRunStrategy, msg string, restartOptions *v1.RestartOptions) {
			vm := newVirtualMachineWithRunStrategy(runStrategy)

			bytesRepresentation, _ := json.Marshal(restartOptions)
			request.Request.Body = io.NopCloser(bytes.NewReader(bytesRepresentation))

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
			table.Entry("Always", v1.RunStrategyAlways, "VM is not running", &v1.RestartOptions{}),
			table.Entry("Manual", v1.RunStrategyManual, "VM is not running", &v1.RestartOptions{}),
			table.Entry("RerunOnFailure", v1.RunStrategyRerunOnFailure, "VM is not running", &v1.RestartOptions{}),
			table.Entry("Halted", v1.RunStrategyHalted, "Halted does not support manual restart requests", &v1.RestartOptions{}),

			table.Entry("Always with dry-run option", v1.RunStrategyAlways, "VM is not running", &v1.RestartOptions{DryRun: []string{k8smetav1.DryRunAll}}),
			table.Entry("Manual with dry-run option", v1.RunStrategyManual, "VM is not running", &v1.RestartOptions{DryRun: []string{k8smetav1.DryRunAll}}),
			table.Entry("RerunOnFailure with dry-run option", v1.RunStrategyRerunOnFailure, "VM is not running", &v1.RestartOptions{DryRun: []string{k8smetav1.DryRunAll}}),
			table.Entry("Halted with dry-run option", v1.RunStrategyHalted, "Halted does not support manual restart requests", &v1.RestartOptions{DryRun: []string{k8smetav1.DryRunAll}}),
		)
	})

	Context("Add/Remove Volume Subresource api", func() {

		newAddVolumeBody := func(opts *v1.AddVolumeOptions) io.ReadCloser {
			optsJson, _ := json.Marshal(opts)
			return &readCloserWrapper{bytes.NewReader(optsJson)}
		}
		newRemoveVolumeBody := func(opts *v1.RemoveVolumeOptions) io.ReadCloser {
			optsJson, _ := json.Marshal(opts)
			return &readCloserWrapper{bytes.NewReader(optsJson)}
		}

		BeforeEach(func() {
			request.PathParameters()["name"] = "testvm"
			request.PathParameters()["namespace"] = "default"
		})

		table.DescribeTable("Should succeed with add volume request", func(addOpts *v1.AddVolumeOptions, removeOpts *v1.RemoveVolumeOptions, isVM bool, code int, enableGate bool) {

			if enableGate {
				enableFeatureGate(virtconfig.HotplugVolumesGate)
			}
			if addOpts != nil {
				request.Request.Body = newAddVolumeBody(addOpts)
			} else {
				request.Request.Body = newRemoveVolumeBody(removeOpts)
			}

			if isVM {
				vm := newMinimalVM(request.PathParameter("name"))
				vm.Namespace = "default"

				patchedVM := vm.DeepCopy()
				patchedVM.Status.VolumeRequests = append(patchedVM.Status.VolumeRequests, v1.VirtualMachineVolumeRequest{AddVolumeOptions: addOpts, RemoveVolumeOptions: removeOpts})
				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/apis/kubevirt.io/v1alpha3/namespaces/default/virtualmachines/testvm"),
						ghttp.RespondWithJSONEncoded(http.StatusOK, vm),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("PATCH", "/apis/kubevirt.io/v1alpha3/namespaces/default/virtualmachines/testvm/status"),
						ghttp.RespondWithJSONEncoded(http.StatusOK, patchedVM),
					),
				)
				if addOpts != nil {
					app.VMAddVolumeRequestHandler(request, response)
				} else {
					app.VMRemoveVolumeRequestHandler(request, response)
				}
			} else {
				vmi := api.NewMinimalVMI(request.PathParameter("name"))
				vmi.Namespace = "default"
				vmi.Status.Phase = v1.Running
				vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
					Name: "existingvol",
				})
				vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
					Name: "existingvol",
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: "testpvcdiskclaim",
						}},
					},
				})
				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/apis/kubevirt.io/v1alpha3/namespaces/default/virtualmachineinstances/testvm"),
						ghttp.RespondWithJSONEncoded(http.StatusOK, vmi),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("PATCH", "/apis/kubevirt.io/v1alpha3/namespaces/default/virtualmachineinstances/testvm"),
						ghttp.RespondWithJSONEncoded(http.StatusOK, vmi),
					),
				)

				if addOpts != nil {
					app.VMIAddVolumeRequestHandler(request, response)
				} else {
					app.VMIRemoveVolumeRequestHandler(request, response)
				}
			}

			Expect(response.StatusCode()).To(Equal(code))
		},
			table.Entry("VM with a valid add volume request", &v1.AddVolumeOptions{
				Name:         "vol1",
				Disk:         &v1.Disk{},
				VolumeSource: &v1.HotplugVolumeSource{},
			}, nil, true, http.StatusAccepted, true),
			table.Entry("VMI with a valid add volume request", &v1.AddVolumeOptions{
				Name:         "vol1",
				Disk:         &v1.Disk{},
				VolumeSource: &v1.HotplugVolumeSource{},
			}, nil, false, http.StatusAccepted, true),
			table.Entry("VMI with an invalid add volume request that's missing a name", &v1.AddVolumeOptions{
				VolumeSource: &v1.HotplugVolumeSource{},
				Disk:         &v1.Disk{},
			}, nil, false, http.StatusBadRequest, true),
			table.Entry("VMI with an invalid add volume request that's missing a disk", &v1.AddVolumeOptions{
				Name:         "vol1",
				VolumeSource: &v1.HotplugVolumeSource{},
			}, nil, false, http.StatusBadRequest, true),
			table.Entry("VMI with an invalid add volume request that's missing a volume", &v1.AddVolumeOptions{
				Name: "vol1",
				Disk: &v1.Disk{},
			}, nil, false, http.StatusBadRequest, true),
			table.Entry("VM with a valid remove volume request", nil, &v1.RemoveVolumeOptions{
				Name: "vol1",
			}, true, http.StatusAccepted, true),
			table.Entry("VMI with a valid remove volume request", nil, &v1.RemoveVolumeOptions{
				Name: "existingvol",
			}, false, http.StatusAccepted, true),
			table.Entry("VMI with a invalid remove volume request missing a name", nil, &v1.RemoveVolumeOptions{}, false, http.StatusBadRequest, true),
			table.Entry("VMI with a valid remove volume request but no feature gate", nil, &v1.RemoveVolumeOptions{
				Name: "existingvol",
			}, false, http.StatusBadRequest, false),
			table.Entry("VM with a valid add volume request but no feature gate", &v1.AddVolumeOptions{
				Name:         "vol1",
				Disk:         &v1.Disk{},
				VolumeSource: &v1.HotplugVolumeSource{},
			}, nil, true, http.StatusBadRequest, false),
		)

		table.DescribeTable("Should generate expected vmi patch", func(volumeRequest *v1.VirtualMachineVolumeRequest, expectedPatch string, expectError bool) {

			vmi := api.NewMinimalVMI(request.PathParameter("name"))
			vmi.Namespace = "default"
			vmi.Status.Phase = v1.Running
			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "existingvol",
			})
			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				Name: "existingvol",
				VolumeSource: v1.VolumeSource{
					PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
						ClaimName: "testpvcdiskclaim",
					}},
				},
			})

			patch, err := generateVMIVolumeRequestPatch(vmi, volumeRequest)
			if expectError {
				Expect(err).ToNot(BeNil())
			} else {
				Expect(err).To(BeNil())
			}

			Expect(patch).To(Equal(expectedPatch))
		},
			table.Entry("add volume request",
				&v1.VirtualMachineVolumeRequest{
					AddVolumeOptions: &v1.AddVolumeOptions{
						Name:         "vol1",
						Disk:         &v1.Disk{},
						VolumeSource: &v1.HotplugVolumeSource{},
					},
				},
				"[{ \"op\": \"test\", \"path\": \"/spec/volumes\", \"value\": [{\"name\":\"existingvol\",\"persistentVolumeClaim\":{\"claimName\":\"testpvcdiskclaim\"}}]}, { \"op\": \"test\", \"path\": \"/spec/domain/devices/disks\", \"value\": [{\"name\":\"existingvol\"}]}, { \"op\": \"replace\", \"path\": \"/spec/volumes\", \"value\": [{\"name\":\"existingvol\",\"persistentVolumeClaim\":{\"claimName\":\"testpvcdiskclaim\"}},{\"name\":\"vol1\"}]}, { \"op\": \"replace\", \"path\": \"/spec/domain/devices/disks\", \"value\": [{\"name\":\"existingvol\"},{\"name\":\"vol1\"}]}]",
				false),
			table.Entry("remove volume request",
				&v1.VirtualMachineVolumeRequest{
					RemoveVolumeOptions: &v1.RemoveVolumeOptions{
						Name: "existingvol",
					},
				},
				"[{ \"op\": \"test\", \"path\": \"/spec/volumes\", \"value\": [{\"name\":\"existingvol\",\"persistentVolumeClaim\":{\"claimName\":\"testpvcdiskclaim\"}}]}, { \"op\": \"test\", \"path\": \"/spec/domain/devices/disks\", \"value\": [{\"name\":\"existingvol\"}]}, { \"op\": \"replace\", \"path\": \"/spec/volumes\", \"value\": []}, { \"op\": \"replace\", \"path\": \"/spec/domain/devices/disks\", \"value\": []}]",
				false),
			table.Entry("remove volume that doesn't exist",
				&v1.VirtualMachineVolumeRequest{
					RemoveVolumeOptions: &v1.RemoveVolumeOptions{
						Name: "non-existent",
					},
				},
				"",
				true),
			table.Entry("add a volume that already exists",
				&v1.VirtualMachineVolumeRequest{
					AddVolumeOptions: &v1.AddVolumeOptions{
						Name:         "existingvol",
						Disk:         &v1.Disk{},
						VolumeSource: &v1.HotplugVolumeSource{},
					},
				},
				"",
				true),
		)
		table.DescribeTable("Should generate expected vm patch", func(volumeRequest *v1.VirtualMachineVolumeRequest, existingVolumeRequests []v1.VirtualMachineVolumeRequest, expectedPatch string, expectError bool) {

			vm := newMinimalVM(request.PathParameter("name"))
			vm.Namespace = "default"

			if len(existingVolumeRequests) > 0 {
				vm.Status.VolumeRequests = existingVolumeRequests
			}

			patch, err := generateVMVolumeRequestPatch(vm, volumeRequest)
			if expectError {
				Expect(err).ToNot(BeNil())
			} else {
				Expect(err).To(BeNil())
			}

			Expect(patch).To(Equal(expectedPatch))
		},
			table.Entry("add volume request with no existing volumes",
				&v1.VirtualMachineVolumeRequest{
					AddVolumeOptions: &v1.AddVolumeOptions{
						Name:         "vol1",
						Disk:         &v1.Disk{},
						VolumeSource: &v1.HotplugVolumeSource{},
					},
				},
				nil,
				"[{ \"op\": \"test\", \"path\": \"/status/volumeRequests\", \"value\": null}, { \"op\": \"add\", \"path\": \"/status/volumeRequests\", \"value\": [{\"addVolumeOptions\":{\"name\":\"vol1\",\"disk\":{\"name\":\"\"},\"volumeSource\":{}}}]}]",
				false),
			table.Entry("add volume request that already exists should fail",
				&v1.VirtualMachineVolumeRequest{
					AddVolumeOptions: &v1.AddVolumeOptions{
						Name:         "vol1",
						Disk:         &v1.Disk{},
						VolumeSource: &v1.HotplugVolumeSource{},
					},
				},
				[]v1.VirtualMachineVolumeRequest{
					{
						AddVolumeOptions: &v1.AddVolumeOptions{
							Name:         "vol1",
							Disk:         &v1.Disk{},
							VolumeSource: &v1.HotplugVolumeSource{},
						},
					},
				},
				"",
				true),
			table.Entry("add volume request when volume requests alread exist",
				&v1.VirtualMachineVolumeRequest{
					AddVolumeOptions: &v1.AddVolumeOptions{
						Name:         "vol1",
						Disk:         &v1.Disk{},
						VolumeSource: &v1.HotplugVolumeSource{},
					},
				},
				[]v1.VirtualMachineVolumeRequest{
					{
						AddVolumeOptions: &v1.AddVolumeOptions{
							Name:         "vol2",
							Disk:         &v1.Disk{},
							VolumeSource: &v1.HotplugVolumeSource{},
						},
					},
				},
				"[{ \"op\": \"test\", \"path\": \"/status/volumeRequests\", \"value\": [{\"addVolumeOptions\":{\"name\":\"vol2\",\"disk\":{\"name\":\"\"},\"volumeSource\":{}}}]}, { \"op\": \"replace\", \"path\": \"/status/volumeRequests\", \"value\": [{\"addVolumeOptions\":{\"name\":\"vol2\",\"disk\":{\"name\":\"\"},\"volumeSource\":{}}},{\"addVolumeOptions\":{\"name\":\"vol1\",\"disk\":{\"name\":\"\"},\"volumeSource\":{}}}]}]",
				false),
			table.Entry("remove volume request with no existing volume request", &v1.VirtualMachineVolumeRequest{
				RemoveVolumeOptions: &v1.RemoveVolumeOptions{
					Name: "vol1",
				},
			},
				nil,
				"[{ \"op\": \"test\", \"path\": \"/status/volumeRequests\", \"value\": null}, { \"op\": \"add\", \"path\": \"/status/volumeRequests\", \"value\": [{\"removeVolumeOptions\":{\"name\":\"vol1\"}}]}]",
				false),
			table.Entry("remove volume request should replace add volume request",
				&v1.VirtualMachineVolumeRequest{
					RemoveVolumeOptions: &v1.RemoveVolumeOptions{
						Name: "vol2",
					},
				},
				[]v1.VirtualMachineVolumeRequest{
					{
						AddVolumeOptions: &v1.AddVolumeOptions{
							Name:         "vol2",
							Disk:         &v1.Disk{},
							VolumeSource: &v1.HotplugVolumeSource{},
						},
					},
				},
				"[{ \"op\": \"test\", \"path\": \"/status/volumeRequests\", \"value\": [{\"addVolumeOptions\":{\"name\":\"vol2\",\"disk\":{\"name\":\"\"},\"volumeSource\":{}}}]}, { \"op\": \"replace\", \"path\": \"/status/volumeRequests\", \"value\": [{\"removeVolumeOptions\":{\"name\":\"vol2\"}}]}]",
				false),
			table.Entry("remove volume request that already exists should fail",
				&v1.VirtualMachineVolumeRequest{
					RemoveVolumeOptions: &v1.RemoveVolumeOptions{
						Name: "vol2",
					},
				},
				[]v1.VirtualMachineVolumeRequest{
					{
						RemoveVolumeOptions: &v1.RemoveVolumeOptions{
							Name: "vol2",
						},
					},
				},
				"",
				true),
		)
	})

	Context("Subresource api - error handling for StartVMRequestHandler", func() {
		BeforeEach(func() {
			request.PathParameters()["name"] = "testvm"
			request.PathParameters()["namespace"] = "default"
		})

		table.DescribeTable("should fail on VM with RunStrategy",
			func(runStrategy v1.VirtualMachineRunStrategy, phase v1.VirtualMachineInstancePhase, status int, msg string, startOptions *v1.StartOptions) {
				vm := newVirtualMachineWithRunStrategy(runStrategy)
				var vmi *v1.VirtualMachineInstance
				if phase != v1.VmPhaseUnset {
					vmi = newVirtualMachineInstanceInPhase(phase)
				}

				bytesRepresentation, _ := json.Marshal(startOptions)
				request.Request.Body = io.NopCloser(bytes.NewReader(bytesRepresentation))

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
			table.Entry("Always without VMI", v1.RunStrategyAlways, v1.VmPhaseUnset, http.StatusNotFound, "Always does not support manual start requests", &v1.StartOptions{}),
			table.Entry("Always with VMI in phase Running", v1.RunStrategyAlways, v1.Running, http.StatusOK, "VM is already running", &v1.StartOptions{}),
			table.Entry("RerunOnFailure with VMI in phase Failed", v1.RunStrategyRerunOnFailure, v1.Failed, http.StatusOK, "RerunOnFailure does not support starting VM from failed state", &v1.StartOptions{}),

			table.Entry("Always without VMI and with dry-run option", v1.RunStrategyAlways, v1.VmPhaseUnset, http.StatusNotFound, "Always does not support manual start requests", &v1.StartOptions{DryRun: []string{k8smetav1.DryRunAll}}),
			table.Entry("Always with VMI in phase Running and with dry-run option", v1.RunStrategyAlways, v1.Running, http.StatusOK, "VM is already running", &v1.StartOptions{DryRun: []string{k8smetav1.DryRunAll}}),
			table.Entry("RerunOnFailure with VMI in phase Failed and with dry-run option", v1.RunStrategyRerunOnFailure, v1.Failed, http.StatusOK, "RerunOnFailure does not support starting VM from failed state", &v1.StartOptions{DryRun: []string{k8smetav1.DryRunAll}}),
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
	})

	Context("Subresource api - error handling for StopVMRequestHandler", func() {
		BeforeEach(func() {
			request.PathParameters()["name"] = "testvm"
			request.PathParameters()["namespace"] = "default"
		})

		table.DescribeTable("should handle VMI does not exist per run strategy", func(runStrategy v1.VirtualMachineRunStrategy, msg string, expectError bool, stopOptions *v1.StopOptions) {
			vm := newVirtualMachineWithRunStrategy(runStrategy)

			bytesRepresentation, _ := json.Marshal(stopOptions)
			request.Request.Body = io.NopCloser(bytes.NewReader(bytesRepresentation))

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

			if !expectError {
				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("PATCH", "/apis/kubevirt.io/v1alpha3/namespaces/default/virtualmachines/testvm"),
						ghttp.RespondWithJSONEncoded(http.StatusOK, vm),
					),
				)
			}

			app.StopVMRequestHandler(request, response)

			if expectError {
				statusErr := ExpectStatusErrorWithCode(recorder, http.StatusConflict)
				// check the msg string that would be presented to virtctl output
				Expect(statusErr.Error()).To(ContainSubstring(msg))
			} else {

				Expect(response.StatusCode()).To(Equal(http.StatusAccepted))
			}

		},
			table.Entry("RunStrategyAlways", v1.RunStrategyAlways, "", false, &v1.StopOptions{}),
			table.Entry("RunStrategyRerunOnFailure", v1.RunStrategyRerunOnFailure, "", false, &v1.StopOptions{}),
			table.Entry("RunStrategyManual", v1.RunStrategyManual, "VM is not running", true, &v1.StopOptions{}),
			table.Entry("RunStrategyHalted", v1.RunStrategyHalted, "VM is not running", true, &v1.StopOptions{}),

			table.Entry("RunStrategyAlways with dry-run option", v1.RunStrategyAlways, "", false, &v1.StopOptions{DryRun: []string{k8smetav1.DryRunAll}}),
			table.Entry("RunStrategyRerunOnFailure with dry-run option", v1.RunStrategyRerunOnFailure, "", false, &v1.StopOptions{DryRun: []string{k8smetav1.DryRunAll}}),
			table.Entry("RunStrategyManual with dry-run option", v1.RunStrategyManual, "VM is not running", true, &v1.StopOptions{DryRun: []string{k8smetav1.DryRunAll}}),
			table.Entry("RunStrategyHalted with dry-run option", v1.RunStrategyHalted, "VM is not running", true, &v1.StopOptions{DryRun: []string{k8smetav1.DryRunAll}}),
		)

		It("should fail on VM with VMI in Unknown Phase", func() {
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
			Expect(statusErr.Error()).To(ContainSubstring("Halted does not support manual stop requests"))
		})

		It("should fail on VM with RunStrategyHalted", func() {
			vm := newVirtualMachineWithRunStrategy(v1.RunStrategyHalted)
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

			app.StopVMRequestHandler(request, response)

			statusErr := ExpectStatusErrorWithCode(recorder, http.StatusConflict)
			// check the msg string that would be presented to virtctl output
			Expect(statusErr.Error()).To(ContainSubstring("Halted does not support manual stop requests"))
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
			vmi := v1.VirtualMachineInstance{}

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

			app.MigrateVMRequestHandler(request, response)

			status := ExpectStatusErrorWithCode(recorder, http.StatusConflict)
			Expect(status.Error()).To(ContainSubstring("VM is not running"))
			close(done)
		})

		It("should fail if migration is not posted", func(done Done) {
			request.PathParameters()["name"] = "testvm"
			request.PathParameters()["namespace"] = "default"

			vm := v1.VirtualMachine{}

			vmi := v1.VirtualMachineInstance{
				Status: v1.VirtualMachineInstanceStatus{
					Phase: v1.Running,
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
					ghttp.RespondWithJSONEncoded(http.StatusOK, vmi),
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

			vm := v1.VirtualMachine{}

			vmi := v1.VirtualMachineInstance{
				Status: v1.VirtualMachineInstanceStatus{
					Phase: v1.Running,
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
					ghttp.RespondWithJSONEncoded(http.StatusOK, vmi),
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

	Context("Freezing", func() {
		It("Should freeze a running VMI", func() {

			backend.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("PUT", "/v1/namespaces/default/virtualmachineinstances/testvmi/freeze"),
					ghttp.RespondWith(http.StatusOK, ""),
				),
			)
			expectVMI(Running, UnPaused)

			app.FreezeVMIRequestHandler(request, response)

			Expect(response.StatusCode()).To(Equal(http.StatusOK))
		})

		It("Should fail freezing a not running VMI", func() {

			expectVMI(NotRunning, UnPaused)

			app.FreezeVMIRequestHandler(request, response)

			ExpectStatusErrorWithCode(recorder, http.StatusConflict)
		})

		It("Should fail unfreezing a not running VMI", func() {

			expectVMI(NotRunning, UnPaused)

			app.UnfreezeVMIRequestHandler(request, response)

			ExpectStatusErrorWithCode(recorder, http.StatusConflict)
		})

		It("Should unfreeze a running VMI", func() {
			backend.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("PUT", "/v1/namespaces/default/virtualmachineinstances/testvmi/unfreeze"),
					ghttp.RespondWith(http.StatusOK, ""),
				),
			)
			expectVMI(Running, UnPaused)

			app.UnfreezeVMIRequestHandler(request, response)

			Expect(response.StatusCode()).To(Equal(http.StatusOK))
		})
	})

	Context("Pausing", func() {
		table.DescribeTable("Should pause a running, not paused VMI according to options", func(pauseOptions *v1.PauseOptions) {

			backend.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("PUT", "/v1/namespaces/default/virtualmachineinstances/testvmi/pause"),
					ghttp.RespondWith(http.StatusOK, ""),
				),
			)
			expectVMI(Running, UnPaused)

			bytesRepresentation, _ := json.Marshal(pauseOptions)
			request.Request.Body = io.NopCloser(bytes.NewReader(bytesRepresentation))

			app.PauseVMIRequestHandler(request, response)

			Expect(response.StatusCode()).To(Equal(http.StatusOK))

		},
			table.Entry("with default", &v1.PauseOptions{}),
			table.Entry("with dry-run option", &v1.PauseOptions{DryRun: []string{k8smetav1.DryRunAll}}),
		)

		table.DescribeTable("Should fail pausing", func(running bool, paused bool, pauseOptions *v1.PauseOptions) {

			expectVMI(running, paused)

			bytesRepresentation, _ := json.Marshal(pauseOptions)
			request.Request.Body = io.NopCloser(bytes.NewReader(bytesRepresentation))

			app.PauseVMIRequestHandler(request, response)

			ExpectStatusErrorWithCode(recorder, http.StatusConflict)
		},
			table.Entry("a not running VMI", NotRunning, UnPaused, &v1.PauseOptions{}),
			table.Entry("a not running VMI with dry-run option", NotRunning, UnPaused, &v1.PauseOptions{DryRun: []string{k8smetav1.DryRunAll}}),

			table.Entry("a running but paused VMI", Running, Paused, &v1.PauseOptions{}),
			table.Entry("a running but paused VMI with dry-run option", Running, Paused, &v1.PauseOptions{DryRun: []string{k8smetav1.DryRunAll}}),
		)

		table.DescribeTable("Should fail unpausing", func(running bool, paused bool, unpauseOptions *v1.UnpauseOptions) {

			expectVMI(running, paused)

			bytesRepresentation, _ := json.Marshal(unpauseOptions)
			request.Request.Body = io.NopCloser(bytes.NewReader(bytesRepresentation))

			app.UnpauseVMIRequestHandler(request, response)

			ExpectStatusErrorWithCode(recorder, http.StatusConflict)
		},
			table.Entry("a running, not paused VMI", Running, UnPaused, &v1.UnpauseOptions{}),
			table.Entry("a running, not paused VMI with dry-run option", Running, UnPaused, &v1.UnpauseOptions{DryRun: []string{k8smetav1.DryRunAll}}),

			table.Entry("a not running VMI", NotRunning, UnPaused, &v1.UnpauseOptions{}),
			table.Entry("a not running VMI with dry-run option", NotRunning, UnPaused, &v1.UnpauseOptions{DryRun: []string{k8smetav1.DryRunAll}}),
		)

		table.DescribeTable("Should unpause a running, paused VMI according to options", func(unpauseOptions *v1.UnpauseOptions) {

			backend.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("PUT", "/v1/namespaces/default/virtualmachineinstances/testvmi/unpause"),
					ghttp.RespondWith(http.StatusOK, ""),
				),
			)
			expectVMI(Running, Paused)

			bytesRepresentation, _ := json.Marshal(unpauseOptions)
			request.Request.Body = io.NopCloser(bytes.NewReader(bytesRepresentation))

			app.UnpauseVMIRequestHandler(request, response)

			Expect(response.StatusCode()).To(Equal(http.StatusOK))

		},
			table.Entry("with default", &v1.UnpauseOptions{}),
			table.Entry("with dry-run option", &v1.UnpauseOptions{DryRun: []string{k8smetav1.DryRunAll}}),
		)
	})

	Context("Subresource api - start paused", func() {
		BeforeEach(func() {
			request.PathParameters()["name"] = "testvm"
			request.PathParameters()["namespace"] = "default"
		})
		It("should patch status on start", func(done Done) {
			body := map[string]bool{
				"paused": true,
			}
			bytesRepresentation, _ := json.Marshal(body)
			request.Request.Body = io.NopCloser(bytes.NewReader(bytesRepresentation))

			vm := v1.VirtualMachine{
				ObjectMeta: k8smetav1.ObjectMeta{
					Name:      "testvm",
					Namespace: "default",
				},
				Spec: v1.VirtualMachineSpec{
					Running:  &notRunning,
					Template: &v1.VirtualMachineInstanceTemplateSpec{},
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

			app.StartVMRequestHandler(request, response)

			Expect(response.Error()).ToNot(HaveOccurred())
			Expect(response.StatusCode()).To(Equal(http.StatusAccepted))
			close(done)
		})

		table.DescribeTable("should patch status on start for VM with RunStrategy",
			func(runStrategy v1.VirtualMachineRunStrategy) {
				vm := newVirtualMachineWithRunStrategy(runStrategy)
				vm.Spec.Template = &v1.VirtualMachineInstanceTemplateSpec{}
				body := map[string]bool{
					"paused": true,
				}
				bytesRepresentation, _ := json.Marshal(body)
				request.Request.Body = io.NopCloser(bytes.NewReader(bytesRepresentation))

				vmi := newVirtualMachineInstanceInPhase(v1.Succeeded)

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
			table.Entry("Manual RunStrategy", v1.RunStrategyManual),
			table.Entry("RerunOnFailure RunStrategy", v1.RunStrategyRerunOnFailure),
		)
	})

	AfterEach(func() {
		server.Close()
		backend.Close()
		disableFeatureGates()
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
