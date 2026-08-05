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
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"time"

	"github.com/emicklei/go-restful/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	gomegatypes "github.com/onsi/gomega/types"
	"go.uber.org/mock/gomock"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/api"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/testutils"
)

const (
	Running     = true
	Paused      = true
	NotRunning  = false
	UnPaused    = false
	testVMName  = "testvm"
	testVMIName = "testvmi"
)

type readCloserWrapper struct {
	io.Reader
}

func (b *readCloserWrapper) Close() error { return nil }

func withDryRun() []string {
	return []string{k8smetav1.DryRunAll}
}

var _ = Describe("VirtualMachineInstance Subresources", func() {
	var backend *ghttp.Server
	var backendIP string
	var request *restful.Request
	var recorder *httptest.ResponseRecorder
	var response *restful.Response

	var ctrl *gomock.Controller
	var kubeClient *fake.Clientset
	var virtClient *kubecli.MockKubevirtClient
	var vmClient *kubecli.MockVirtualMachineInterface
	var vmiClient *kubecli.MockVirtualMachineInstanceInterface

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

	config, _, _ := testutils.NewFakeClusterConfigUsingKV(kv)

	app := SubresourceAPIApp{}
	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		kubeClient = fake.NewSimpleClientset()
		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		vmClient = kubecli.NewMockVirtualMachineInterface(ctrl)
		vmiClient = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)

		virtClient.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()
		virtClient.EXPECT().VirtualMachine(k8smetav1.NamespaceDefault).Return(vmClient).AnyTimes()
		virtClient.EXPECT().VirtualMachine("").Return(vmClient).AnyTimes()
		virtClient.EXPECT().VirtualMachineInstance(k8smetav1.NamespaceDefault).Return(vmiClient).AnyTimes()
		virtClient.EXPECT().VirtualMachineInstance("").Return(vmiClient).AnyTimes()

		backend = ghttp.NewTLSServer()
		backendAddr := strings.Split(backend.Addr(), ":")
		backendPort, err := strconv.Atoi(backendAddr[1])
		backendIP = backendAddr[0]
		Expect(err).ToNot(HaveOccurred())
		app.consoleServerPort = backendPort
		app.virtClient = virtClient
		app.k8sClient = kubeClient
		app.handlerTLSConfiguration = &tls.Config{InsecureSkipVerify: true}
		app.clusterConfig = config
		app.handlerHttpClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: app.handlerTLSConfiguration,
			},
			Timeout: 10 * time.Second,
		}

		request = restful.NewRequest(&http.Request{})
		recorder = httptest.NewRecorder()
		response = restful.NewResponse(recorder)
		// Make sure that any unexpected call to the client will fail
		kubeClient.Fake.PrependReactor("*", "*", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			Expect(action).To(BeNil())
			return true, nil, nil
		})
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

		kubeClient.Fake.PrependReactor("list", "pods", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			return true, &podList, nil
		})
	}

	guestAgentConnected := func(vmi *v1.VirtualMachineInstance) {
		if vmi.Status.Conditions == nil {
			vmi.Status.Conditions = []v1.VirtualMachineInstanceCondition{}
		}
		vmi.Status.Conditions = append(vmi.Status.Conditions, v1.VirtualMachineInstanceCondition{
			Type:   v1.VirtualMachineInstanceAgentConnected,
			Status: k8sv1.ConditionTrue,
		})
	}

	ACPIDisabled := func(vmi *v1.VirtualMachineInstance) {
		_false := false
		featureStateDisabled := v1.FeatureState{Enabled: &_false}
		if vmi.Spec.Domain.Features == nil {
			vmi.Spec.Domain.Features = &v1.Features{
				ACPI: featureStateDisabled,
			}
		} else {
			vmi.Spec.Domain.Features.ACPI = featureStateDisabled
		}
	}

	expectVMI := func(running, paused bool, vmiWarpFunctions ...func(vmi *v1.VirtualMachineInstance)) {
		request.PathParameters()["name"] = testVMIName
		request.PathParameters()["namespace"] = k8smetav1.NamespaceDefault

		phase := v1.Running
		if !running {
			phase = v1.Failed
		}

		vmi := v1.VirtualMachineInstance{
			ObjectMeta: k8smetav1.ObjectMeta{
				Name:      testVMIName,
				Namespace: k8smetav1.NamespaceDefault,
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

		for _, f := range vmiWarpFunctions {
			f(&vmi)
		}

		vmiClient.EXPECT().Get(context.Background(), vmi.Name, k8smetav1.GetOptions{}).Return(&vmi, nil)

		expectHandlerPod()
	}

	Context("Subresource api", func() {
		It("should find matching pod for running VirtualMachineInstance", func() {
			vmi := api.NewMinimalVMI(testVMIName)
			vmi.Status.Phase = v1.Running
			vmi.ObjectMeta.SetUID(uuid.NewUUID())

			expectHandlerPod()

			result, err := app.getVirtHandlerConnForVMI(vmi)

			Expect(err).ToNot(HaveOccurred())
			ip, _, _ := result.ConnectionDetails()
			Expect(ip).To(Equal(backendIP))
		})

		It("should fail if VirtualMachineInstance is not in running state", func() {
			vmi := api.NewMinimalVMI(testVMIName)
			vmi.Status.Phase = v1.Succeeded
			vmi.ObjectMeta.SetUID(uuid.NewUUID())

			_, err := app.getVirtHandlerConnForVMI(vmi)

			Expect(err).To(HaveOccurred())
		})

		It("should fail no matching pod is found", func() {
			vmi := api.NewMinimalVMI(testVMIName)
			vmi.Status.Phase = v1.Running
			vmi.ObjectMeta.SetUID(uuid.NewUUID())

			podList := k8sv1.PodList{}
			podList.Items = []k8sv1.Pod{}

			kubeClient.Fake.PrependReactor("list", "pods", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				return true, &podList, nil
			})

			conn, err := app.getVirtHandlerConnForVMI(vmi)
			Expect(err).ToNot(HaveOccurred())
			_, _, err = conn.ConnectionDetails()
			Expect(err).To(HaveOccurred())
		})

	})

	Context("Subresource api - Guest OS Info", func() {
		type subRes func(request *restful.Request, response *restful.Response)

		DescribeTable("should fail when the VMI does not exist", func(fn subRes) {
			request.PathParameters()["name"] = testVMName
			request.PathParameters()["namespace"] = k8smetav1.NamespaceDefault

			vmiClient.EXPECT().Get(context.Background(), testVMName, k8smetav1.GetOptions{}).Return(nil, errors.NewNotFound(v1.Resource("virtualmachineinstance"), testVMName))

			fn(request, response)

			Expect(response.Error()).To(HaveOccurred(), "Response should indicate VM not found")
			Expect(response.StatusCode()).To(Equal(http.StatusInternalServerError))

		},
			Entry("for GuestOSInfo", app.GuestOSInfo),
			Entry("for UserList", app.UserList),
			Entry("for Filesystem", app.FilesystemList),
		)

		DescribeTable("should fail when the VMI is not running", func(fn subRes) {
			request.PathParameters()["name"] = testVMName
			request.PathParameters()["namespace"] = k8smetav1.NamespaceDefault

			vmi := v1.VirtualMachineInstance{}

			vmiClient.EXPECT().Get(context.Background(), testVMName, k8smetav1.GetOptions{}).Return(&vmi, nil)

			fn(request, response)

			Expect(response.Error()).To(HaveOccurred())
			Expect(response.StatusCode()).To(Equal(http.StatusInternalServerError))
			Expect(response.Error().Error()).To(ContainSubstring("VMI is not running"))
		},
			Entry("for GuestOSInfo", app.GuestOSInfo),
			Entry("for UserList", app.UserList),
			Entry("for FilesystemList", app.FilesystemList),
		)

		DescribeTable("should fail when VMI does not have agent connected", func(fn subRes) {
			request.PathParameters()["name"] = testVMName
			request.PathParameters()["namespace"] = k8smetav1.NamespaceDefault

			vmi := v1.VirtualMachineInstance{
				Status: v1.VirtualMachineInstanceStatus{
					Phase:      v1.Running,
					Conditions: []v1.VirtualMachineInstanceCondition{},
				},
			}

			vmiClient.EXPECT().Get(context.Background(), testVMName, k8smetav1.GetOptions{}).Return(&vmi, nil)

			fn(request, response)

			Expect(response.Error()).To(HaveOccurred())
			Expect(response.StatusCode()).To(Equal(http.StatusInternalServerError))
			Expect(response.Error().Error()).To(ContainSubstring("VMI does not have guest agent connected"))
		},
			Entry("for GuestOSInfo", app.GuestOSInfo),
			Entry("for UserList", app.UserList),
			Entry("for FilesystemList", app.FilesystemList),
		)
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

	Context("Reset", func() {
		It("Should reset a running VMI", func() {
			backend.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("PUT", "/v1/namespaces/default/virtualmachineinstances/testvmi/reset"),
					ghttp.RespondWith(http.StatusOK, ""),
				),
			)

			expectVMI(Running, UnPaused)

			app.ResetVMIRequestHandler(request, response)

			Expect(response.StatusCode()).To(Equal(http.StatusOK))
		})

		It("Should fail reset on a not running VMI", func() {

			expectVMI(NotRunning, UnPaused)

			app.ResetVMIRequestHandler(request, response)

			ExpectStatusErrorWithCode(recorder, http.StatusBadRequest)
		})
	})

	Context("SoftReboot", func() {
		It("Should soft reboot a running VMI", func() {
			backend.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("PUT", "/v1/namespaces/default/virtualmachineinstances/testvmi/softreboot"),
					ghttp.RespondWith(http.StatusOK, ""),
				),
			)

			expectVMI(true, false, guestAgentConnected)

			app.SoftRebootVMIRequestHandler(request, response)

			Expect(response.StatusCode()).To(Equal(http.StatusOK))
		})

		It("Should fail soft reboot a not running VMI", func() {

			expectVMI(false, false, guestAgentConnected)

			app.SoftRebootVMIRequestHandler(request, response)

			ExpectStatusErrorWithCode(recorder, http.StatusConflict)
		})

		It("Should fail soft reboot a paused VMI", func() {

			expectVMI(true, true, guestAgentConnected)

			app.SoftRebootVMIRequestHandler(request, response)

			ExpectStatusErrorWithCode(recorder, http.StatusConflict)
		})

		It("Should fail soft reboot a guest agent disconnected and ACPI feature disabled VMI", func() {

			expectVMI(true, false, ACPIDisabled)

			app.SoftRebootVMIRequestHandler(request, response)

			ExpectStatusErrorWithCode(recorder, http.StatusConflict)
		})
	})

	Context("Pausing", func() {
		DescribeTable("Should pause a running, not paused VMI according to options", func(pauseOptions *v1.PauseOptions, matchExpectation gomegatypes.GomegaMatcher) {

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
			//In case of dry-run the request is not propagated to the handler
			Expect(backend.ReceivedRequests()).To(matchExpectation)
		},
			Entry("with default", &v1.PauseOptions{}, HaveLen(1)),
			Entry("with dry-run option", &v1.PauseOptions{DryRun: withDryRun()}, BeNil()),
		)

		withLivenessProbe := func(vmi *v1.VirtualMachineInstance) {
			vmi.Spec.LivenessProbe = &v1.Probe{
				Handler:             v1.Handler{},
				InitialDelaySeconds: 120,
				TimeoutSeconds:      120,
				PeriodSeconds:       120,
				SuccessThreshold:    1,
				FailureThreshold:    1,
			}
		}
		withGuestAgentPingLivenessProbe := func(vmi *v1.VirtualMachineInstance) {
			vmi.Spec.LivenessProbe = &v1.Probe{
				Handler:             v1.Handler{GuestAgentPing: &v1.GuestAgentPing{}},
				InitialDelaySeconds: 120,
				TimeoutSeconds:      120,
				PeriodSeconds:       120,
				SuccessThreshold:    1,
				FailureThreshold:    1,
			}
		}
		nilAdditionalOps := func(vmi *v1.VirtualMachineInstance) {
			return
		}

		DescribeTable("Should fail pausing", func(running bool, paused bool, additionalOpts func(vmi *v1.VirtualMachineInstance), pauseOptions *v1.PauseOptions, expectedCode int, expectedError string) {
			expectVMI(running, paused, additionalOpts)

			bytesRepresentation, _ := json.Marshal(pauseOptions)
			request.Request.Body = io.NopCloser(bytes.NewReader(bytesRepresentation))

			app.PauseVMIRequestHandler(request, response)

			ExpectStatusErrorWithCode(recorder, expectedCode)
			ExpectMessage(recorder, ContainSubstring(expectedError))
		},
			Entry("a not running VMI", NotRunning, UnPaused, nilAdditionalOps, &v1.PauseOptions{}, http.StatusConflict, "VM is not running"),
			Entry("a not running VMI with dry-run option", NotRunning, UnPaused, nilAdditionalOps, &v1.PauseOptions{DryRun: withDryRun()}, http.StatusConflict, "VM is not running"),

			Entry("a running but paused VMI", Running, Paused, nilAdditionalOps, &v1.PauseOptions{}, http.StatusConflict, "VMI is already paused"),
			Entry("a running but paused VMI with dry-run option", Running, Paused, nilAdditionalOps, &v1.PauseOptions{DryRun: withDryRun()}, http.StatusConflict, "VMI is already paused"),

			Entry("a running VMI with LivenessProbe", Running, UnPaused, withLivenessProbe, &v1.PauseOptions{}, http.StatusForbidden, "Pausing VMIs with a non-GuestAgentPing LivenessProbe is not supported"),
			Entry("a running VMI with LivenessProbe with dry-run option", Running, UnPaused, withLivenessProbe, &v1.PauseOptions{DryRun: withDryRun()}, http.StatusForbidden, "Pausing VMIs with a non-GuestAgentPing LivenessProbe is not supported"),
		)

		It("Should allow pausing a running VMI with a GuestAgentPing LivenessProbe", func() {
			backend.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("PUT", "/v1/namespaces/default/virtualmachineinstances/testvmi/pause"),
					ghttp.RespondWith(http.StatusOK, ""),
				),
			)
			expectVMI(Running, UnPaused, withGuestAgentPingLivenessProbe)

			bytesRepresentation, _ := json.Marshal(&v1.PauseOptions{})
			request.Request.Body = io.NopCloser(bytes.NewReader(bytesRepresentation))

			app.PauseVMIRequestHandler(request, response)

			Expect(response.StatusCode()).To(Equal(http.StatusOK))
			Expect(backend.ReceivedRequests()).To(HaveLen(1))
		})

		DescribeTable("Should fail unpausing due to VMI state", func(running bool, paused bool, unpauseOptions *v1.UnpauseOptions, expectedError string) {

			expectVMI(running, paused)

			vm := newVirtualMachineWithRunning(pointer.P(Running))
			vmClient.EXPECT().Get(context.Background(), testVMIName, k8smetav1.GetOptions{}).Return(vm, nil)

			bytesRepresentation, _ := json.Marshal(unpauseOptions)
			request.Request.Body = io.NopCloser(bytes.NewReader(bytesRepresentation))

			app.UnpauseVMIRequestHandler(request, response)

			ExpectStatusErrorWithCode(recorder, http.StatusConflict)
			ExpectMessage(recorder, ContainSubstring(expectedError))
		},
			Entry("a running, not paused VMI", Running, UnPaused, &v1.UnpauseOptions{}, "VMI is not paused"),
			Entry("a running, not paused VMI with dry-run option", Running, UnPaused, &v1.UnpauseOptions{DryRun: withDryRun()}, "VMI is not paused"),

			Entry("a not running VMI", NotRunning, UnPaused, &v1.UnpauseOptions{}, "VMI is not running"),
			Entry("a not running VMI with dry-run option", NotRunning, UnPaused, &v1.UnpauseOptions{DryRun: withDryRun()}, "VMI is not running"),
		)

		DescribeTable("Should fail unpausing when snapshot in progress", func(unpauseOptions *v1.UnpauseOptions) {

			request.PathParameters()["name"] = testVMIName
			request.PathParameters()["namespace"] = k8smetav1.NamespaceDefault

			vm := newVirtualMachineWithRunning(pointer.P(Running))
			vm.Status.SnapshotInProgress = &[]string{"test-snapshot"}[0]

			vmClient.EXPECT().Get(context.Background(), testVMIName, k8smetav1.GetOptions{}).Return(vm, nil)

			bytesRepresentation, _ := json.Marshal(unpauseOptions)
			request.Request.Body = io.NopCloser(bytes.NewReader(bytesRepresentation))

			app.UnpauseVMIRequestHandler(request, response)

			ExpectStatusErrorWithCode(recorder, http.StatusConflict)
			ExpectMessage(recorder, ContainSubstring(vmSnapshotInprogress))
		},
			Entry("with default options", &v1.UnpauseOptions{}),
			Entry("with dry-run option", &v1.UnpauseOptions{DryRun: withDryRun()}),
		)

		DescribeTable("Should unpause a running, paused VMI according to options", func(unpauseOptions *v1.UnpauseOptions, matchExpectation gomegatypes.GomegaMatcher) {

			backend.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("PUT", "/v1/namespaces/default/virtualmachineinstances/testvmi/unpause"),
					ghttp.RespondWith(http.StatusOK, ""),
				),
			)
			expectVMI(Running, Paused)
			vm := newVirtualMachineWithRunning(pointer.P(Running))
			vmClient.EXPECT().Get(context.Background(), testVMIName, k8smetav1.GetOptions{}).Return(vm, nil)

			bytesRepresentation, _ := json.Marshal(unpauseOptions)
			request.Request.Body = io.NopCloser(bytes.NewReader(bytesRepresentation))

			app.UnpauseVMIRequestHandler(request, response)

			Expect(response.StatusCode()).To(Equal(http.StatusOK))
			//In case of dry-run the request is not propagated to the handler
			Expect(backend.ReceivedRequests()).To(matchExpectation)
		},
			Entry("with default", &v1.UnpauseOptions{}, HaveLen(1)),
			Entry("with dry-run option", &v1.UnpauseOptions{DryRun: withDryRun()}, BeNil()),
		)
	})

	AfterEach(func() {
		backend.Close()
	})
})

func newVirtualMachineWithRunning(running *bool) *v1.VirtualMachine {
	return &v1.VirtualMachine{
		ObjectMeta: k8smetav1.ObjectMeta{
			Name:      testVMName,
			Namespace: k8smetav1.NamespaceDefault,
		},
		Spec: v1.VirtualMachineSpec{
			Running: running,
		},
	}
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

func ExpectMessage(recorder *httptest.ResponseRecorder, expected gomegatypes.GomegaMatcher) {
	status := k8smetav1.Status{}
	err := json.Unmarshal(recorder.Body.Bytes(), &status)

	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	ExpectWithOffset(1, status.Message).To(expected)
}
