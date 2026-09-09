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

package lifecycle

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"go.uber.org/mock/gomock"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
)

const testVMIName = "testvmi"

func withDryRun() []string {
	return []string{metav1.DryRunAll}
}

var _ = Describe("VirtualMachineInstance lifecycle", func() {
	var (
		backend    *ghttp.Server
		backendIP  string
		kubeClient *fake.Clientset
		vmClient   *kubecli.MockVirtualMachineInterface
		vmiClient  *kubecli.MockVirtualMachineInstanceInterface
		handler    *Handler
	)

	BeforeEach(func() {
		ctrl := gomock.NewController(GinkgoT())
		virtClient := kubecli.NewMockKubevirtClient(ctrl)
		kubeClient = fake.NewSimpleClientset()
		vmClient = kubecli.NewMockVirtualMachineInterface(ctrl)
		vmiClient = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)

		virtClient.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()
		virtClient.EXPECT().VirtualMachine(metav1.NamespaceDefault).Return(vmClient).AnyTimes()
		virtClient.EXPECT().VirtualMachineInstance(metav1.NamespaceDefault).Return(vmiClient).AnyTimes()

		backend = ghttp.NewTLSServer()
		backendAddr := strings.Split(backend.Addr(), ":")
		backendPort, err := strconv.Atoi(backendAddr[1])
		Expect(err).ToNot(HaveOccurred())
		backendIP = backendAddr[0]
		handler = NewHandler(virtClient, backendPort, &tls.Config{InsecureSkipVerify: true})

		kubeClient.Fake.PrependReactor("*", "*", func(action testing.Action) (bool, runtime.Object, error) {
			Expect(action).To(BeNil())
			return true, nil, nil
		})
	})

	AfterEach(func() {
		backend.Close()
	})

	expectHandlerPod := func() {
		pod := k8sv1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "madeup-name",
				Labels: map[string]string{v1.AppLabel: "virt-handler"},
			},
			Spec:   k8sv1.PodSpec{NodeName: "mynode"},
			Status: k8sv1.PodStatus{Phase: k8sv1.PodRunning, PodIP: backendIP},
		}
		kubeClient.Fake.PrependReactor("list", "pods", func(testing.Action) (bool, runtime.Object, error) {
			return true, &k8sv1.PodList{Items: []k8sv1.Pod{pod}}, nil
		})
	}

	expectVMI := func(running, paused bool, modifiers ...func(*v1.VirtualMachineInstance)) {
		phase := v1.Running
		if !running {
			phase = v1.Failed
		}
		vmi := &v1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{Name: testVMIName, Namespace: metav1.NamespaceDefault},
			Status:     v1.VirtualMachineInstanceStatus{Phase: phase},
		}
		if paused {
			vmi.Status.Conditions = []v1.VirtualMachineInstanceCondition{{
				Type:   v1.VirtualMachineInstancePaused,
				Status: k8sv1.ConditionTrue,
			}}
		}
		for _, modify := range modifiers {
			modify(vmi)
		}
		vmiClient.EXPECT().Get(context.Background(), testVMIName, metav1.GetOptions{}).Return(vmi, nil)
		expectHandlerPod()
	}

	optionsBody := func(options interface{}) io.ReadCloser {
		data, err := json.Marshal(options)
		Expect(err).ToNot(HaveOccurred())
		return io.NopCloser(bytes.NewReader(data))
	}

	expectStatusError := func(statusErr *errors.StatusError, code int, message string) {
		Expect(statusErr).To(HaveOccurred())
		Expect(statusErr.Status().Code).To(BeNumerically("==", code))
		if message != "" {
			Expect(statusErr.Error()).To(ContainSubstring(message))
		}
	}

	Context("Freeze", func() {
		It("freezes a running VMI", func() {
			backend.AppendHandlers(ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodPut, "/v1/namespaces/default/virtualmachineinstances/testvmi/freeze"),
				ghttp.RespondWith(http.StatusOK, ""),
			))
			expectVMI(true, false)

			Expect(handler.FreezeVMI(context.Background(), metav1.NamespaceDefault, testVMIName, nil)).To(Succeed())
		})

		It("rejects a non-running VMI", func() {
			expectVMI(false, false)

			statusErr := handler.FreezeVMI(context.Background(), metav1.NamespaceDefault, testVMIName, nil)
			expectStatusError(statusErr, http.StatusConflict, "VM is not running")
		})
	})

	Context("Unfreeze", func() {
		It("unfreezes a running VMI", func() {
			backend.AppendHandlers(ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodPut, "/v1/namespaces/default/virtualmachineinstances/testvmi/unfreeze"),
				ghttp.RespondWith(http.StatusOK, ""),
			))
			expectVMI(true, false)

			Expect(handler.UnfreezeVMI(context.Background(), metav1.NamespaceDefault, testVMIName, nil)).To(Succeed())
		})

		It("rejects a non-running VMI", func() {
			expectVMI(false, false)

			statusErr := handler.UnfreezeVMI(context.Background(), metav1.NamespaceDefault, testVMIName, nil)
			expectStatusError(statusErr, http.StatusConflict, "VMI is not running")
		})
	})

	Context("Reset", func() {
		It("resets a running VMI", func() {
			backend.AppendHandlers(ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodPut, "/v1/namespaces/default/virtualmachineinstances/testvmi/reset"),
				ghttp.RespondWith(http.StatusOK, ""),
			))
			expectVMI(true, false)

			Expect(handler.ResetVMI(context.Background(), metav1.NamespaceDefault, testVMIName, nil)).To(Succeed())
		})

		It("adds context when resetting a non-running VMI fails", func() {
			expectVMI(false, false)

			statusErr := handler.ResetVMI(context.Background(), metav1.NamespaceDefault, testVMIName, nil)
			expectStatusError(statusErr, http.StatusBadRequest, "Failed to reset non-running VMI with phase Failed")
		})
	})

	Context("SoftReboot", func() {
		guestAgentConnected := func(vmi *v1.VirtualMachineInstance) {
			vmi.Status.Conditions = append(vmi.Status.Conditions, v1.VirtualMachineInstanceCondition{
				Type:   v1.VirtualMachineInstanceAgentConnected,
				Status: k8sv1.ConditionTrue,
			})
		}
		acpiDisabled := func(vmi *v1.VirtualMachineInstance) {
			disabled := false
			vmi.Spec.Domain.Features = &v1.Features{ACPI: v1.FeatureState{Enabled: &disabled}}
		}

		It("soft reboots a running VMI", func() {
			backend.AppendHandlers(ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodPut, "/v1/namespaces/default/virtualmachineinstances/testvmi/softreboot"),
				ghttp.RespondWith(http.StatusOK, ""),
			))
			expectVMI(true, false, guestAgentConnected)

			Expect(handler.SoftRebootVMI(context.Background(), metav1.NamespaceDefault, testVMIName, nil)).To(Succeed())
		})

		It("rejects a non-running VMI", func() {
			expectVMI(false, false, guestAgentConnected)

			statusErr := handler.SoftRebootVMI(context.Background(), metav1.NamespaceDefault, testVMIName, nil)
			expectStatusError(statusErr, http.StatusConflict, "VM is not running")
		})

		It("rejects a paused VMI", func() {
			expectVMI(true, true, guestAgentConnected)

			statusErr := handler.SoftRebootVMI(context.Background(), metav1.NamespaceDefault, testVMIName, nil)
			expectStatusError(statusErr, http.StatusConflict, "VMI is paused")
		})

		It("rejects a VMI without guest agent and with ACPI disabled", func() {
			expectVMI(true, false, acpiDisabled)

			statusErr := handler.SoftRebootVMI(context.Background(), metav1.NamespaceDefault, testVMIName, nil)
			expectStatusError(statusErr, http.StatusConflict, "VMI neither have the agent connected nor the ACPI feature enabled")
		})
	})

	Context("Pause", func() {
		withLivenessProbe := func(vmi *v1.VirtualMachineInstance) {
			vmi.Spec.LivenessProbe = &v1.Probe{Handler: v1.Handler{}}
		}
		withGuestAgentPingLivenessProbe := func(vmi *v1.VirtualMachineInstance) {
			vmi.Spec.LivenessProbe = &v1.Probe{Handler: v1.Handler{GuestAgentPing: &v1.GuestAgentPing{}}}
		}
		noModification := func(*v1.VirtualMachineInstance) {}

		DescribeTable("handles options", func(options *v1.PauseOptions, expectedRequests int) {
			if expectedRequests > 0 {
				backend.AppendHandlers(ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodPut, "/v1/namespaces/default/virtualmachineinstances/testvmi/pause"),
					ghttp.RespondWith(http.StatusOK, ""),
				))
			}
			expectVMI(true, false)

			statusErr := handler.PauseVMI(context.Background(), metav1.NamespaceDefault, testVMIName, optionsBody(options))
			Expect(statusErr).ToNot(HaveOccurred())
			Expect(backend.ReceivedRequests()).To(HaveLen(expectedRequests))
		},
			Entry("default", &v1.PauseOptions{}, 1),
			Entry("dry-run", &v1.PauseOptions{DryRun: withDryRun()}, 0),
		)

		DescribeTable("rejects invalid VMI state", func(running, paused bool, modify func(*v1.VirtualMachineInstance), options *v1.PauseOptions, code int, message string) {
			expectVMI(running, paused, modify)

			statusErr := handler.PauseVMI(context.Background(), metav1.NamespaceDefault, testVMIName, optionsBody(options))
			expectStatusError(statusErr, code, message)
		},
			Entry("not running", false, false, noModification, &v1.PauseOptions{}, http.StatusConflict, "VM is not running"),
			Entry("not running dry-run", false, false, noModification, &v1.PauseOptions{DryRun: withDryRun()}, http.StatusConflict, "VM is not running"),
			Entry("already paused", true, true, noModification, &v1.PauseOptions{}, http.StatusConflict, "VMI is already paused"),
			Entry("already paused dry-run", true, true, noModification, &v1.PauseOptions{DryRun: withDryRun()}, http.StatusConflict, "VMI is already paused"),
			Entry("unsupported liveness probe", true, false, withLivenessProbe, &v1.PauseOptions{}, http.StatusForbidden, "Pausing VMIs with a non-GuestAgentPing LivenessProbe is not supported"),
			Entry("unsupported liveness probe dry-run", true, false, withLivenessProbe, &v1.PauseOptions{DryRun: withDryRun()}, http.StatusForbidden, "Pausing VMIs with a non-GuestAgentPing LivenessProbe is not supported"),
		)

		It("allows a GuestAgentPing liveness probe", func() {
			backend.AppendHandlers(ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodPut, "/v1/namespaces/default/virtualmachineinstances/testvmi/pause"),
				ghttp.RespondWith(http.StatusOK, ""),
			))
			expectVMI(true, false, withGuestAgentPingLivenessProbe)

			Expect(handler.PauseVMI(context.Background(), metav1.NamespaceDefault, testVMIName, optionsBody(&v1.PauseOptions{}))).To(Succeed())
		})
	})

	Context("Unpause", func() {
		expectVM := func(snapshotInProgress bool) {
			vm := &v1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{Name: testVMIName, Namespace: metav1.NamespaceDefault},
			}
			if snapshotInProgress {
				snapshotName := "test-snapshot"
				vm.Status.SnapshotInProgress = &snapshotName
			}
			vmClient.EXPECT().Get(context.Background(), testVMIName, metav1.GetOptions{}).Return(vm, nil)
		}

		DescribeTable("rejects invalid VMI state", func(running, paused bool, options *v1.UnpauseOptions, message string) {
			expectVM(false)
			expectVMI(running, paused)

			statusErr := handler.UnpauseVMI(context.Background(), metav1.NamespaceDefault, testVMIName, optionsBody(options))
			expectStatusError(statusErr, http.StatusConflict, message)
		},
			Entry("not paused", true, false, &v1.UnpauseOptions{}, "VMI is not paused"),
			Entry("not paused dry-run", true, false, &v1.UnpauseOptions{DryRun: withDryRun()}, "VMI is not paused"),
			Entry("not running", false, false, &v1.UnpauseOptions{}, "VMI is not running"),
			Entry("not running dry-run", false, false, &v1.UnpauseOptions{DryRun: withDryRun()}, "VMI is not running"),
		)

		DescribeTable("rejects a VM snapshot in progress", func(options *v1.UnpauseOptions) {
			expectVM(true)

			statusErr := handler.UnpauseVMI(context.Background(), metav1.NamespaceDefault, testVMIName, optionsBody(options))
			expectStatusError(statusErr, http.StatusConflict, vmSnapshotInprogress)
		},
			Entry("default", &v1.UnpauseOptions{}),
			Entry("dry-run", &v1.UnpauseOptions{DryRun: withDryRun()}),
		)

		DescribeTable("handles options", func(options *v1.UnpauseOptions, expectedRequests int) {
			if expectedRequests > 0 {
				backend.AppendHandlers(ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodPut, "/v1/namespaces/default/virtualmachineinstances/testvmi/unpause"),
					ghttp.RespondWith(http.StatusOK, ""),
				))
			}
			expectVM(false)
			expectVMI(true, true)

			statusErr := handler.UnpauseVMI(context.Background(), metav1.NamespaceDefault, testVMIName, optionsBody(options))
			Expect(statusErr).ToNot(HaveOccurred())
			Expect(backend.ReceivedRequests()).To(HaveLen(expectedRequests))
		},
			Entry("default", &v1.UnpauseOptions{}, 1),
			Entry("dry-run", &v1.UnpauseOptions{DryRun: withDryRun()}, 0),
		)
	})

	Describe("request body validation", func() {
		It("rejects malformed PauseOptions", func() {
			statusErr := handler.PauseVMI(context.Background(), metav1.NamespaceDefault, testVMIName, io.NopCloser(strings.NewReader("{")))
			expectStatusError(statusErr, http.StatusBadRequest, "Can not unmarshal Request body")
		})

		It("rejects malformed UnpauseOptions", func() {
			vmClient.EXPECT().Get(context.Background(), testVMIName, metav1.GetOptions{}).
				Return(nil, errors.NewNotFound(v1.Resource("virtualmachine"), testVMIName))

			statusErr := handler.UnpauseVMI(context.Background(), metav1.NamespaceDefault, testVMIName, io.NopCloser(strings.NewReader("{")))
			expectStatusError(statusErr, http.StatusBadRequest, "Can not unmarshal Request body")
		})
	})
})
