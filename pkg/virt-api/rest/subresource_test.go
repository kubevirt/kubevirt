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
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/emicklei/go-restful/v3"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	gomegatypes "github.com/onsi/gomega/types"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/api"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/libvmi"
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
	var migrateClient *kubecli.MockVirtualMachineInstanceMigrationInterface

	gracePeriodZero := pointer.P(int64(0))

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
		migrateClient = kubecli.NewMockVirtualMachineInstanceMigrationInterface(ctrl)

		virtClient.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()
		virtClient.EXPECT().VirtualMachine(k8smetav1.NamespaceDefault).Return(vmClient).AnyTimes()
		virtClient.EXPECT().VirtualMachine("").Return(vmClient).AnyTimes()
		virtClient.EXPECT().VirtualMachineInstance(k8smetav1.NamespaceDefault).Return(vmiClient).AnyTimes()
		virtClient.EXPECT().VirtualMachineInstance("").Return(vmiClient).AnyTimes()
		virtClient.EXPECT().VirtualMachineInstanceMigration(k8smetav1.NamespaceDefault).Return(migrateClient).AnyTimes()

		backend = ghttp.NewTLSServer()
		backendAddr := strings.Split(backend.Addr(), ":")
		backendPort, err := strconv.Atoi(backendAddr[1])
		backendIP = backendAddr[0]
		Expect(err).ToNot(HaveOccurred())
		app.consoleServerPort = backendPort
		app.virtCli = virtClient
		app.credentialsLock = &sync.Mutex{}
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

		Context("restart", func() {
			It("should fail if VirtualMachine not exists", func() {
				request.PathParameters()["name"] = testVMName
				request.PathParameters()["namespace"] = k8smetav1.NamespaceDefault

				vmClient.EXPECT().Get(context.Background(), testVMName, k8smetav1.GetOptions{}).Return(nil, errors.NewNotFound(v1.Resource("virtualmachine"), testVMName))

				app.RestartVMRequestHandler(request, response)

				ExpectStatusErrorWithCode(recorder, http.StatusNotFound)
			})

			DescribeTable("should return an error when VM is not running", func(errMsg string, running *bool, runStrategy *v1.VirtualMachineRunStrategy) {
				request.PathParameters()["name"] = testVMName
				request.PathParameters()["namespace"] = k8smetav1.NamespaceDefault

				vm := v1.VirtualMachine{
					Spec: v1.VirtualMachineSpec{
						Running:     running,
						RunStrategy: runStrategy,
					},
				}

				vmClient.EXPECT().Get(context.Background(), testVMName, k8smetav1.GetOptions{}).Return(&vm, nil)

				if runStrategy != nil && *runStrategy == v1.RunStrategyManual {
					vmiClient.EXPECT().Get(context.Background(), testVMName, k8smetav1.GetOptions{}).Return(nil, errors.NewNotFound(v1.Resource("virtualmachineinstance"), vm.Name))
				}
				app.RestartVMRequestHandler(request, response)

				status := ExpectStatusErrorWithCode(recorder, http.StatusConflict)
				Expect(status.Error()).To(ContainSubstring(errMsg))
			},
				Entry("with Running field", "RunStategy Halted does not support manual restart requests", pointer.P(NotRunning), nil),
				Entry("with RunStrategyHalted", "RunStategy Halted does not support manual restart requests", nil, pointer.P(v1.RunStrategyHalted)),
				Entry("with RunStrategyManual", "VM is not running: Halted", nil, pointer.P(v1.RunStrategyManual)),
			)

			DescribeTable("should ForceRestart VirtualMachine according to options", func(restartOptions *v1.RestartOptions) {
				request.PathParameters()["name"] = testVMName
				request.PathParameters()["namespace"] = k8smetav1.NamespaceDefault

				bytesRepresentation, _ := json.Marshal(restartOptions)
				request.Request.Body = io.NopCloser(bytes.NewReader(bytesRepresentation))

				vm := newVirtualMachineWithRunning(pointer.P(Running))
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

				vmClient.EXPECT().Get(context.Background(), vm.Name, k8smetav1.GetOptions{}).Return(vm, nil)
				vmiClient.EXPECT().Get(context.Background(), vm.Name, k8smetav1.GetOptions{}).Return(&vmi, nil)
				vmClient.EXPECT().PatchStatus(context.Background(), vm.Name, types.JSONPatchType, gomock.Any(), gomock.Any()).DoAndReturn(
					func(ctx context.Context, name string, patchType types.PatchType, body interface{}, opts k8smetav1.PatchOptions) (interface{}, interface{}) {
						// check that dryRun option has been propagated to patch request
						Expect(opts.DryRun).To(BeEquivalentTo(restartOptions.DryRun))
						return vm, nil
					})
				kubeClient.Fake.PrependReactor("list", "pods", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
					return true, &podList, nil
				})
				kubeClient.Fake.PrependReactor("delete", "pods", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
					_, ok := action.(testing.DeleteAction)
					Expect(ok).To(BeTrue())
					return true, nil, nil
				})

				app.RestartVMRequestHandler(request, response)

				Expect(response.Error()).ToNot(HaveOccurred())
				Expect(response.StatusCode()).To(Equal(http.StatusAccepted))
			},
				Entry("with default", &v1.RestartOptions{GracePeriodSeconds: gracePeriodZero}),
				Entry("with dry-run option", &v1.RestartOptions{GracePeriodSeconds: gracePeriodZero, DryRun: withDryRun()}),
			)

			It("should not ForceRestart VirtualMachine if no Pods found for the VMI", func() {
				request.PathParameters()["name"] = testVMName
				request.PathParameters()["namespace"] = k8smetav1.NamespaceDefault

				body := map[string]int64{
					"gracePeriodSeconds": 0,
				}
				bytesRepresentation, _ := json.Marshal(body)
				request.Request.Body = io.NopCloser(bytes.NewReader(bytesRepresentation))

				vm := newVirtualMachineWithRunning(pointer.P(Running))
				vmi := v1.VirtualMachineInstance{
					Spec: v1.VirtualMachineInstanceSpec{},
				}
				vmi.ObjectMeta.SetUID(uuid.NewUUID())

				podList := k8sv1.PodList{}
				podList.Items = []k8sv1.Pod{}

				kubeClient.Fake.PrependReactor("list", "pods", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
					return true, &podList, nil
				})
				vmClient.EXPECT().Get(context.Background(), vm.Name, k8smetav1.GetOptions{}).Return(vm, nil)
				vmiClient.EXPECT().Get(context.Background(), vm.Name, k8smetav1.GetOptions{}).Return(&vmi, nil)
				vmClient.EXPECT().PatchStatus(context.Background(), vm.Name, types.JSONPatchType, gomock.Any(), k8smetav1.PatchOptions{}).Return(vm, nil)

				app.RestartVMRequestHandler(request, response)

				Expect(response.Error()).ToNot(HaveOccurred())
				Expect(response.StatusCode()).To(Equal(http.StatusAccepted))
			})

			It("should restart VirtualMachine", func() {
				request.PathParameters()["name"] = testVMName
				request.PathParameters()["namespace"] = k8smetav1.NamespaceDefault

				vm := newVirtualMachineWithRunning(pointer.P(Running))

				vmi := v1.VirtualMachineInstance{
					Spec: v1.VirtualMachineInstanceSpec{},
				}

				vmi.ObjectMeta.SetUID(uuid.NewUUID())
				vmClient.EXPECT().Get(context.Background(), vm.Name, k8smetav1.GetOptions{}).Return(vm, nil)
				vmiClient.EXPECT().Get(context.Background(), vm.Name, k8smetav1.GetOptions{}).Return(&vmi, nil)
				vmClient.EXPECT().PatchStatus(context.Background(), vm.Name, types.JSONPatchType, gomock.Any(), k8smetav1.PatchOptions{}).Return(vm, nil)

				app.RestartVMRequestHandler(request, response)

				Expect(response.Error()).ToNot(HaveOccurred())
				Expect(response.StatusCode()).To(Equal(http.StatusAccepted))
			})

			It("should start VirtualMachine if VMI doesn't exist", func() {
				request.PathParameters()["name"] = testVMName
				request.PathParameters()["namespace"] = k8smetav1.NamespaceDefault

				vm := newVirtualMachineWithRunning(pointer.P(Running))
				vmi := newVirtualMachineInstanceInPhase(v1.Running)

				vmClient.EXPECT().Get(context.Background(), vm.Name, k8smetav1.GetOptions{}).Return(vm, nil)
				vmiClient.EXPECT().Get(context.Background(), vm.Name, k8smetav1.GetOptions{}).Return(vmi, nil)
				vmClient.EXPECT().PatchStatus(context.Background(), vm.Name, types.JSONPatchType, gomock.Any(), k8smetav1.PatchOptions{}).Return(vm, nil)

				app.RestartVMRequestHandler(request, response)
				Expect(response.Error()).NotTo(HaveOccurred())
				Expect(response.StatusCode()).To(Equal(http.StatusAccepted))
			})

			It("should fail when the volume migration in ongoing", func() {
				vmi := libvmi.New()
				vm := libvmi.NewVirtualMachine(vmi)
				controller.NewVirtualMachineConditionManager().UpdateCondition(vm, &v1.VirtualMachineCondition{
					Type:   v1.VirtualMachineConditionType(v1.VirtualMachineInstanceVolumesChange),
					Status: k8sv1.ConditionTrue,
				})
				request.PathParameters()["name"] = vm.Name
				vmClient.EXPECT().Get(context.Background(), vm.Name, k8smetav1.GetOptions{}).Return(vm, nil)

				app.RestartVMRequestHandler(request, response)

				statusErr := ExpectStatusErrorWithCode(recorder, http.StatusConflict)
				Expect(statusErr.Error()).To(ContainSubstring("VM recovery required"))
			})
		})

		Context("stop", func() {
			DescribeTable("should ForceStop VirtualMachine according to options", func(statusPhase v1.VirtualMachineInstancePhase, stopOptions *v1.StopOptions) {
				request.PathParameters()["name"] = testVMName
				request.PathParameters()["namespace"] = k8smetav1.NamespaceDefault
				var terminationGracePeriodSeconds int64 = 1800

				bytesRepresentation, _ := json.Marshal(stopOptions)
				request.Request.Body = io.NopCloser(bytes.NewReader(bytesRepresentation))

				vm := newVirtualMachineWithRunning(pointer.P(Running))

				vmi := v1.VirtualMachineInstance{
					ObjectMeta: k8smetav1.ObjectMeta{
						Name:      testVMName,
						Namespace: k8smetav1.NamespaceDefault,
					},
					Spec: v1.VirtualMachineInstanceSpec{
						TerminationGracePeriodSeconds: &terminationGracePeriodSeconds,
					},
					Status: v1.VirtualMachineInstanceStatus{
						Phase: statusPhase,
					},
				}
				vmi.ObjectMeta.SetUID(uuid.NewUUID())

				vmClient.EXPECT().Get(context.Background(), vm.Name, k8smetav1.GetOptions{}).Return(vm, nil)
				vmiClient.EXPECT().Get(context.Background(), vmi.Name, k8smetav1.GetOptions{}).Return(&vmi, nil)
				vmiClient.EXPECT().Patch(context.Background(), vmi.Name, types.JSONPatchType, gomock.Any(), gomock.Any()).DoAndReturn(
					func(ctx context.Context, name string, patchType types.PatchType, body interface{}, opts k8smetav1.PatchOptions, _ ...string) (interface{}, interface{}) {
						// check that dryRun option has been propagated to patch request
						Expect(opts.DryRun).To(BeEquivalentTo(stopOptions.DryRun))
						return &vmi, nil
					}).AnyTimes()
				vmClient.EXPECT().Patch(context.Background(), vm.Name, types.JSONPatchType, gomock.Any(), gomock.Any()).DoAndReturn(
					func(ctx context.Context, name string, patchType types.PatchType, body interface{}, opts k8smetav1.PatchOptions, _ ...string) (interface{}, interface{}) {
						// check that dryRun option has been propagated to patch request
						Expect(opts.DryRun).To(BeEquivalentTo(stopOptions.DryRun))
						return vm, nil
					})

				app.StopVMRequestHandler(request, response)
				Expect(response.Error()).ToNot(HaveOccurred())
				Expect(response.StatusCode()).To(Equal(http.StatusAccepted))
			},
				Entry("in status Running with default", v1.Running, &v1.StopOptions{GracePeriod: gracePeriodZero}),
				Entry("in status Failed with default", v1.Failed, &v1.StopOptions{GracePeriod: gracePeriodZero}),
				Entry("in status Running with dry-run", v1.Running, &v1.StopOptions{GracePeriod: gracePeriodZero, DryRun: withDryRun()}),
				Entry("in status Failed with dry-run", v1.Failed, &v1.StopOptions{GracePeriod: gracePeriodZero, DryRun: withDryRun()}),
			)
		})
	})

	Context("Subresource api - error handling for RestartVMRequestHandler", func() {
		BeforeEach(func() {
			request.PathParameters()["name"] = testVMName
			request.PathParameters()["namespace"] = k8smetav1.NamespaceDefault
		})

		It("should fail on VM with RunStrategyHalted", func() {
			vm := newVirtualMachineWithRunStrategy(v1.RunStrategyHalted)

			vmClient.EXPECT().Get(context.Background(), vm.Name, k8smetav1.GetOptions{}).Return(vm, nil)

			app.RestartVMRequestHandler(request, response)

			status := ExpectStatusErrorWithCode(recorder, http.StatusConflict)
			// check the msg string that would be presented to virtctl output
			Expect(status.Error()).To(ContainSubstring("Halted does not support manual restart requests"))
		})

		DescribeTable("should not fail with VMI and RunStrategy", func(runStrategy v1.VirtualMachineRunStrategy) {
			vm := newVirtualMachineWithRunStrategy(runStrategy)
			vmi := newVirtualMachineInstanceInPhase(v1.Failed)

			vmClient.EXPECT().Get(context.Background(), vm.Name, k8smetav1.GetOptions{}).Return(vm, nil)
			vmiClient.EXPECT().Get(context.Background(), vm.Name, k8smetav1.GetOptions{}).Return(vmi, nil)
			vmClient.EXPECT().PatchStatus(context.Background(), vm.Name, types.JSONPatchType, gomock.Any(), k8smetav1.PatchOptions{}).Return(vm, nil)

			app.RestartVMRequestHandler(request, response)

			Expect(response.StatusCode()).To(Equal(http.StatusAccepted))
		},
			Entry("Always", v1.RunStrategyAlways),
			Entry("Manual", v1.RunStrategyManual),
			Entry("RerunOnFailure", v1.RunStrategyRerunOnFailure),
		)

		DescribeTable("should fail anytime without VMI and RunStrategy", func(runStrategy v1.VirtualMachineRunStrategy, msg string, restartOptions *v1.RestartOptions) {
			vm := newVirtualMachineWithRunStrategy(runStrategy)

			bytesRepresentation, _ := json.Marshal(restartOptions)
			request.Request.Body = io.NopCloser(bytes.NewReader(bytesRepresentation))

			vmClient.EXPECT().Get(context.Background(), vm.Name, k8smetav1.GetOptions{}).Return(vm, nil)
			vmiClient.EXPECT().Get(context.Background(), vm.Name, k8smetav1.GetOptions{}).Return(nil, errors.NewNotFound(v1.Resource("virtualmachineinstance"), vm.Name)).AnyTimes()

			app.RestartVMRequestHandler(request, response)

			status := ExpectStatusErrorWithCode(recorder, http.StatusConflict)
			// check the msg string that would be presented to virtctl output
			Expect(status.Error()).To(ContainSubstring(msg))
		},
			Entry("Always", v1.RunStrategyAlways, "VM is not running", &v1.RestartOptions{}),
			Entry("Manual", v1.RunStrategyManual, "VM is not running", &v1.RestartOptions{}),
			Entry("RerunOnFailure", v1.RunStrategyRerunOnFailure, "VM is not running", &v1.RestartOptions{}),
			Entry("Once", v1.RunStrategyOnce, "Once does not support manual restart requests", &v1.RestartOptions{}),
			Entry("Halted", v1.RunStrategyHalted, "Halted does not support manual restart requests", &v1.RestartOptions{}),

			Entry("Always with dry-run option", v1.RunStrategyAlways, "VM is not running", &v1.RestartOptions{DryRun: withDryRun()}),
			Entry("Manual with dry-run option", v1.RunStrategyManual, "VM is not running", &v1.RestartOptions{DryRun: withDryRun()}),
			Entry("RerunOnFailure with dry-run option", v1.RunStrategyRerunOnFailure, "VM is not running", &v1.RestartOptions{DryRun: withDryRun()}),
			Entry("Once with dry-run option", v1.RunStrategyOnce, "Once does not support manual restart requests", &v1.RestartOptions{DryRun: withDryRun()}),
			Entry("Halted with dry-run option", v1.RunStrategyHalted, "Halted does not support manual restart requests", &v1.RestartOptions{DryRun: withDryRun()}),
		)
	})

	Context("Subresource api - error handling for StartVMRequestHandler", func() {
		BeforeEach(func() {
			request.PathParameters()["name"] = testVMName
			request.PathParameters()["namespace"] = k8smetav1.NamespaceDefault
		})

		DescribeTable("should fail on VM with RunStrategy",
			func(runStrategy v1.VirtualMachineRunStrategy, phase v1.VirtualMachineInstancePhase, status int, msg string, startOptions *v1.StartOptions) {
				vm := newVirtualMachineWithRunStrategy(runStrategy)
				var vmi *v1.VirtualMachineInstance
				if phase != v1.VmPhaseUnset {
					vmi = newVirtualMachineInstanceInPhase(phase)
				}

				bytesRepresentation, _ := json.Marshal(startOptions)
				request.Request.Body = io.NopCloser(bytes.NewReader(bytesRepresentation))

				vmClient.EXPECT().Get(context.Background(), vm.Name, k8smetav1.GetOptions{}).Return(vm, nil)
				vmiClient.EXPECT().Get(context.Background(), testVMName, k8smetav1.GetOptions{}).DoAndReturn(func(ctx context.Context, name string, opts k8smetav1.GetOptions) (interface{}, interface{}) {
					if status == http.StatusNotFound {
						return vmi, errors.NewNotFound(v1.Resource("virtualmachineinstance"), testVMName)
					}
					return vmi, nil
				})

				app.StartVMRequestHandler(request, response)

				statusErr := ExpectStatusErrorWithCode(recorder, http.StatusConflict)
				// check the msg string that would be presented to virtctl output
				Expect(statusErr.Error()).To(ContainSubstring(msg))
			},
			Entry("Always without VMI", v1.RunStrategyAlways, v1.VmPhaseUnset, http.StatusNotFound, "Always does not support manual start requests", &v1.StartOptions{}),
			Entry("Always with VMI in phase Running", v1.RunStrategyAlways, v1.Running, http.StatusOK, "VM is already running", &v1.StartOptions{}),
			Entry("Once", v1.RunStrategyOnce, v1.VmPhaseUnset, http.StatusNotFound, "Once does not support manual start requests", &v1.StartOptions{}),
			Entry("RerunOnFailure with VMI in phase Failed", v1.RunStrategyRerunOnFailure, v1.Failed, http.StatusOK, "RerunOnFailure does not support starting VM from failed state", &v1.StartOptions{}),

			Entry("Always without VMI and with dry-run option", v1.RunStrategyAlways, v1.VmPhaseUnset, http.StatusNotFound, "Always does not support manual start requests", &v1.StartOptions{DryRun: withDryRun()}),
			Entry("Always with VMI in phase Running and with dry-run option", v1.RunStrategyAlways, v1.Running, http.StatusOK, "VM is already running", &v1.StartOptions{DryRun: withDryRun()}),
			Entry("Once with dry-run option", v1.RunStrategyOnce, v1.VmPhaseUnset, http.StatusNotFound, "Once does not support manual start requests", &v1.StartOptions{DryRun: withDryRun()}),
			Entry("RerunOnFailure with VMI in phase Failed and with dry-run option", v1.RunStrategyRerunOnFailure, v1.Failed, http.StatusOK, "RerunOnFailure does not support starting VM from failed state", &v1.StartOptions{DryRun: withDryRun()}),
		)

		DescribeTable("should not fail on VM with RunStrategy ",
			func(runStrategy v1.VirtualMachineRunStrategy, phase v1.VirtualMachineInstancePhase, status int) {
				vm := newVirtualMachineWithRunStrategy(runStrategy)
				var vmi *v1.VirtualMachineInstance
				if phase != v1.VmPhaseUnset {
					vmi = newVirtualMachineInstanceInPhase(phase)
				}

				vmClient.EXPECT().Get(context.Background(), vm.Name, k8smetav1.GetOptions{}).Return(vm, nil)
				vmiClient.EXPECT().Get(context.Background(), vm.Name, k8smetav1.GetOptions{}).Return(vmi, nil)
				vmClient.EXPECT().PatchStatus(context.Background(), vm.Name, types.JSONPatchType, gomock.Any(), k8smetav1.PatchOptions{}).DoAndReturn(
					func(ctx context.Context, name string, patchType types.PatchType, body interface{}, opts k8smetav1.PatchOptions) (interface{}, interface{}) {
						Expect(opts.DryRun).To(BeNil())
						return vm, nil
					})

				app.StartVMRequestHandler(request, response)

				Expect(response.StatusCode()).To(Equal(http.StatusAccepted))
			},
			Entry("RerunOnFailure with VMI in state Succeeded", v1.RunStrategyRerunOnFailure, v1.Succeeded, http.StatusOK),
			Entry("Manual with VMI in state Succeeded", v1.RunStrategyManual, v1.Succeeded, http.StatusOK),
			Entry("Manual with VMI in state Failed", v1.RunStrategyManual, v1.Failed, http.StatusOK),
		)

		It("should fail when the volume migration in ongoing", func() {
			vmi := libvmi.New()
			vm := libvmi.NewVirtualMachine(vmi)
			controller.NewVirtualMachineConditionManager().UpdateCondition(vm, &v1.VirtualMachineCondition{
				Type:   v1.VirtualMachineManualRecoveryRequired,
				Status: k8sv1.ConditionTrue,
			})
			request.PathParameters()["name"] = vm.Name
			vmClient.EXPECT().Get(context.Background(), vm.Name, k8smetav1.GetOptions{}).Return(vm, nil)
			vmiClient.EXPECT().Get(context.Background(), vm.Name, k8smetav1.GetOptions{}).Return(vmi, nil)

			app.StartVMRequestHandler(request, response)

			statusErr := ExpectStatusErrorWithCode(recorder, http.StatusConflict)
			Expect(statusErr.Error()).To(ContainSubstring("VM recovery required"))
		})
	})

	Context("Subresource api - error handling for StopVMRequestHandler", func() {
		BeforeEach(func() {
			request.PathParameters()["name"] = testVMName
			request.PathParameters()["namespace"] = k8smetav1.NamespaceDefault
		})

		DescribeTable("should handle VMI does not exist per run strategy", func(runStrategy v1.VirtualMachineRunStrategy, msg string, expectError bool, stopOptions *v1.StopOptions) {
			vm := newVirtualMachineWithRunStrategy(runStrategy)

			bytesRepresentation, _ := json.Marshal(stopOptions)
			request.Request.Body = io.NopCloser(bytes.NewReader(bytesRepresentation))

			vmClient.EXPECT().Get(context.Background(), vm.Name, k8smetav1.GetOptions{}).Return(vm, nil)
			vmiClient.EXPECT().Get(context.Background(), vm.Name, k8smetav1.GetOptions{}).Return(nil, errors.NewNotFound(v1.Resource("virtualmachineinstance"), testVMName))
			if !expectError {
				vmClient.EXPECT().Patch(context.Background(), vm.Name, types.JSONPatchType, gomock.Any(), gomock.Any()).DoAndReturn(
					func(ctx context.Context, name string, patchType types.PatchType, body interface{}, opts k8smetav1.PatchOptions, _ ...string) (interface{}, interface{}) {
						// check that dryRun option has been propagated to patch request
						Expect(opts.DryRun).To(BeEquivalentTo(stopOptions.DryRun))
						return vm, nil
					})
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
			Entry("RunStrategyAlways", v1.RunStrategyAlways, "", false, &v1.StopOptions{}),
			Entry("RunStrategyOnce", v1.RunStrategyOnce, "", false, &v1.StopOptions{}),
			Entry("RunStrategyRerunOnFailure", v1.RunStrategyRerunOnFailure, "", true, &v1.StopOptions{}),
			Entry("RunStrategyManual", v1.RunStrategyManual, "VM is not running", true, &v1.StopOptions{}),
			Entry("RunStrategyHalted", v1.RunStrategyHalted, "VM is not running", true, &v1.StopOptions{}),

			Entry("RunStrategyAlways with dry-run option", v1.RunStrategyAlways, "", false, &v1.StopOptions{DryRun: withDryRun()}),
			Entry("RunStrategyOnce with dry-run option", v1.RunStrategyOnce, "", false, &v1.StopOptions{DryRun: withDryRun()}),
			Entry("RunStrategyRerunOnFailure with dry-run option", v1.RunStrategyRerunOnFailure, "", true, &v1.StopOptions{DryRun: withDryRun()}),
			Entry("RunStrategyManual with dry-run option", v1.RunStrategyManual, "VM is not running", true, &v1.StopOptions{DryRun: withDryRun()}),
			Entry("RunStrategyHalted with dry-run option", v1.RunStrategyHalted, "VM is not running", true, &v1.StopOptions{DryRun: withDryRun()}),
		)

		It("should fail on VM with VMI in Unknown Phase", func() {
			vm := newVirtualMachineWithRunStrategy(v1.RunStrategyHalted)
			vmi := newVirtualMachineInstanceInPhase(v1.Unknown)

			vmClient.EXPECT().Get(context.Background(), vm.Name, k8smetav1.GetOptions{}).Return(vm, nil)
			vmiClient.EXPECT().Get(context.Background(), vm.Name, k8smetav1.GetOptions{}).Return(vmi, nil)

			app.StopVMRequestHandler(request, response)

			statusErr := ExpectStatusErrorWithCode(recorder, http.StatusConflict)
			// check the msg string that would be presented to virtctl output
			Expect(statusErr.Error()).To(ContainSubstring("Halted only supports manual stop requests with a shorter graceperiod"))
		})

		DescribeTable("for VM with RunStrategyHalted, should", func(terminationGracePeriod *int64, graceperiod *int64, shouldFail bool) {
			vm := newVirtualMachineWithRunStrategy(v1.RunStrategyHalted)
			vmi := newVirtualMachineInstanceInPhase(v1.Running)

			vmi.Spec.TerminationGracePeriodSeconds = terminationGracePeriod

			stopOptions := &v1.StopOptions{GracePeriod: graceperiod}

			bytesRepresentation, err := json.Marshal(stopOptions)
			Expect(err).ToNot(HaveOccurred())
			request.Request.Body = io.NopCloser(bytes.NewReader(bytesRepresentation))

			vmClient.EXPECT().Get(context.Background(), vm.Name, k8smetav1.GetOptions{}).Return(vm, nil)
			vmiClient.EXPECT().Get(context.Background(), vm.Name, k8smetav1.GetOptions{}).Return(vmi, nil)

			if graceperiod != nil {
				vmiClient.EXPECT().Patch(context.Background(), vmi.Name, types.JSONPatchType, gomock.Any(), gomock.Any()).DoAndReturn(
					func(ctx context.Context, name string, patchType types.PatchType, data []byte, opts k8smetav1.PatchOptions, _ ...string) (interface{}, interface{}) {
						// check that dryRun option has been propagated to patch request
						Expect(opts.DryRun).To(BeEquivalentTo(stopOptions.DryRun))

						patchSet := patch.New()
						// used for stopping a VM with RunStrategyHalted
						if vmi.Spec.TerminationGracePeriodSeconds != nil {
							patchSet.AddOption(patch.WithTest("/spec/terminationGracePeriodSeconds", *vmi.Spec.TerminationGracePeriodSeconds))
						} else {
							patchSet.AddOption(patch.WithTest("/spec/terminationGracePeriodSeconds", nil))
						}
						patchSet.AddOption(patch.WithReplace("/spec/terminationGracePeriodSeconds", *graceperiod))
						patchBytes, err := patchSet.GeneratePayload()
						Expect(err).ToNot(HaveOccurred())
						Expect(string(data)).To(Equal(string(patchBytes)))
						return vm, nil
					})
			}
			if !shouldFail {
				vmClient.EXPECT().PatchStatus(context.Background(), vm.Name, types.JSONPatchType, gomock.Any(), k8smetav1.PatchOptions{}).Return(vm, nil)
			}
			app.StopVMRequestHandler(request, response)

			if shouldFail {
				statusErr := ExpectStatusErrorWithCode(recorder, http.StatusConflict)
				// check the msg string that would be presented to virtctl output
				Expect(statusErr.Error()).To(ContainSubstring("Halted only supports manual stop requests with a shorter graceperiod"))
			} else {
				Expect(response.Error()).ToNot(HaveOccurred())
				Expect(response.StatusCode()).To(Equal(http.StatusAccepted))
			}
		},
			Entry("fail with nil graceperiod", pointer.P(int64(1800)), nil, true),
			Entry("fail with equal graceperiod", pointer.P(int64(1800)), pointer.P(int64(1800)), true),
			Entry("fail with greater graceperiod", pointer.P(int64(1800)), pointer.P(int64(2400)), true),
			Entry("not fail with non-nil graceperiod and nil termination graceperiod", nil, pointer.P(int64(1800)), false),
			Entry("not fail with shorter graceperiod and non-nil termination graceperiod", pointer.P(int64(1800)), pointer.P(int64(800)), false),
		)

		DescribeTable("should not fail on VM with RunStrategy", func(runStrategy v1.VirtualMachineRunStrategy) {
			vm := newVirtualMachineWithRunStrategy(runStrategy)
			vmi := newVirtualMachineInstanceInPhase(v1.Running)

			vmClient.EXPECT().Get(context.Background(), vm.Name, k8smetav1.GetOptions{}).Return(vm, nil)
			vmiClient.EXPECT().Get(context.Background(), vm.Name, k8smetav1.GetOptions{}).Return(vmi, nil)

			if runStrategy == v1.RunStrategyManual || runStrategy == v1.RunStrategyRerunOnFailure {
				vmClient.EXPECT().PatchStatus(context.Background(), vm.Name, types.JSONPatchType, gomock.Any(), k8smetav1.PatchOptions{}).Return(vm, nil)
			} else {
				vmClient.EXPECT().Patch(context.Background(), vm.Name, types.JSONPatchType, gomock.Any(), k8smetav1.PatchOptions{}).Return(vm, nil)
			}

			app.StopVMRequestHandler(request, response)

			Expect(response.StatusCode()).To(Equal(http.StatusAccepted))
		},
			Entry("Always", v1.RunStrategyAlways),
			Entry("RerunOnFailure", v1.RunStrategyRerunOnFailure),
			Entry("Once", v1.RunStrategyOnce),
			Entry("Manual", v1.RunStrategyManual),
		)
	})

	Context("Subresource api - MigrateVMRequestHandler", func() {
		DescribeTable("should fail if VirtualMachine not exists according to options", func(migrateOptions *v1.MigrateOptions) {
			request.PathParameters()["name"] = testVMName
			request.PathParameters()["namespace"] = k8smetav1.NamespaceDefault

			bytesRepresentation, _ := json.Marshal(migrateOptions)
			request.Request.Body = io.NopCloser(bytes.NewReader(bytesRepresentation))

			vmClient.EXPECT().Get(context.Background(), testVMName, k8smetav1.GetOptions{}).Return(nil, errors.NewNotFound(v1.Resource("virtualmachine"), testVMName))
			app.MigrateVMRequestHandler(request, response)

			ExpectStatusErrorWithCode(recorder, http.StatusNotFound)
		},
			Entry("with default", &v1.MigrateOptions{}),
			Entry("with dry-run option", &v1.MigrateOptions{DryRun: withDryRun()}),
		)

		DescribeTable("should fail if VirtualMachine is not running according to options", func(migrateOptions *v1.MigrateOptions) {
			request.PathParameters()["name"] = testVMName
			request.PathParameters()["namespace"] = k8smetav1.NamespaceDefault

			vm := v1.VirtualMachine{}
			vmi := v1.VirtualMachineInstance{}

			bytesRepresentation, _ := json.Marshal(migrateOptions)
			request.Request.Body = io.NopCloser(bytes.NewReader(bytesRepresentation))

			vmClient.EXPECT().Get(context.Background(), testVMName, k8smetav1.GetOptions{}).Return(&vm, nil)
			vmiClient.EXPECT().Get(context.Background(), testVMName, k8smetav1.GetOptions{}).Return(&vmi, nil)

			app.MigrateVMRequestHandler(request, response)

			status := ExpectStatusErrorWithCode(recorder, http.StatusConflict)
			Expect(status.Error()).To(ContainSubstring("VM is not running"))
		},
			Entry("with default", &v1.MigrateOptions{}),
			Entry("with dry-run option", &v1.MigrateOptions{DryRun: withDryRun()}),
		)

		DescribeTable("should fail if migration is not posted according to options", func(migrateOptions *v1.MigrateOptions) {
			request.PathParameters()["name"] = testVMName
			request.PathParameters()["namespace"] = k8smetav1.NamespaceDefault

			vm := v1.VirtualMachine{}

			vmi := v1.VirtualMachineInstance{
				Status: v1.VirtualMachineInstanceStatus{
					Phase: v1.Running,
				},
			}

			bytesRepresentation, _ := json.Marshal(migrateOptions)
			request.Request.Body = io.NopCloser(bytes.NewReader(bytesRepresentation))

			vmClient.EXPECT().Get(context.Background(), testVMName, k8smetav1.GetOptions{}).Return(&vm, nil)
			vmiClient.EXPECT().Get(context.Background(), testVMName, k8smetav1.GetOptions{}).Return(&vmi, nil)
			migrateClient.EXPECT().Create(context.Background(), gomock.Any(), gomock.Any()).Return(nil, errors.NewInternalError(fmt.Errorf("error creating object")))
			app.MigrateVMRequestHandler(request, response)

			ExpectStatusErrorWithCode(recorder, http.StatusInternalServerError)
		},
			Entry("with default", &v1.MigrateOptions{}),
			Entry("with dry-run option", &v1.MigrateOptions{DryRun: withDryRun()}),
		)

		DescribeTable("should migrate VirtualMachine according to options", func(migrateOptions *v1.MigrateOptions) {
			request.PathParameters()["name"] = testVMName
			request.PathParameters()["namespace"] = k8smetav1.NamespaceDefault

			vm := v1.VirtualMachine{}

			vmi := v1.VirtualMachineInstance{
				Status: v1.VirtualMachineInstanceStatus{
					Phase: v1.Running,
				},
			}

			bytesRepresentation, _ := json.Marshal(migrateOptions)
			request.Request.Body = io.NopCloser(bytes.NewReader(bytesRepresentation))
			migration := v1.VirtualMachineInstanceMigration{}

			vmClient.EXPECT().Get(context.Background(), testVMName, k8smetav1.GetOptions{}).Return(&vm, nil)
			vmiClient.EXPECT().Get(context.Background(), testVMName, k8smetav1.GetOptions{}).Return(&vmi, nil)

			migrateClient.EXPECT().Create(context.Background(), gomock.Any(), gomock.Any()).Do(
				func(ctx context.Context, obj interface{}, opts k8smetav1.CreateOptions) {
					Expect(opts.DryRun).To(BeEquivalentTo(migrateOptions.DryRun))
				}).Return(&migration, nil)
			app.MigrateVMRequestHandler(request, response)

			Expect(response.Error()).ToNot(HaveOccurred())
			Expect(response.StatusCode()).To(Equal(http.StatusAccepted))
		},
			Entry("with default", &v1.MigrateOptions{}),
			Entry("with dry-run option", &v1.MigrateOptions{DryRun: withDryRun()}),
		)
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

	Context("StateChange JSON", func() {
		It("should create a stop request if status exists", func() {
			uid := uuid.NewUUID()
			vm := newMinimalVM(testVMName)
			vm.Status.Created = true
			stopRequest := v1.VirtualMachineStateChangeRequest{
				Action: v1.StopRequest,
				UID:    &uid,
			}

			res, err := getChangeRequestJson(vm, stopRequest)
			Expect(err).ToNot(HaveOccurred())

			ref, err := patch.New(
				patch.WithTest("/status/stateChangeRequests", nil),
				patch.WithAdd("/status/stateChangeRequests", []v1.VirtualMachineStateChangeRequest{stopRequest}),
			).GeneratePayload()
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(Equal(ref))
		})

		It("should create a stop request if status doesn't exist", func() {
			uid := uuid.NewUUID()
			vm := newMinimalVM(testVMName)
			stopRequest := v1.VirtualMachineStateChangeRequest{
				Action: v1.StopRequest,
				UID:    &uid,
			}

			res, err := getChangeRequestJson(vm, stopRequest)
			Expect(err).ToNot(HaveOccurred())

			ref, err := patch.New(
				patch.WithAdd("/status", v1.VirtualMachineStatus{StateChangeRequests: []v1.VirtualMachineStateChangeRequest{stopRequest}}),
			).GeneratePayload()
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(Equal(ref))
		})

		It("should create a restart request if status exists", func() {
			uid := uuid.NewUUID()
			vm := newMinimalVM(testVMName)
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

			ref, err := patch.New(
				patch.WithTest("/status/stateChangeRequests", nil),
				patch.WithAdd("/status/stateChangeRequests", []v1.VirtualMachineStateChangeRequest{stopRequest, startRequest}),
			).GeneratePayload()
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(Equal(ref))
		})

		It("should create a restart request if status doesn't exist", func() {
			uid := uuid.NewUUID()
			vm := newMinimalVM(testVMName)
			stopRequest := v1.VirtualMachineStateChangeRequest{
				Action: v1.StopRequest,
				UID:    &uid,
			}
			startRequest := v1.VirtualMachineStateChangeRequest{
				Action: v1.StartRequest,
			}

			res, err := getChangeRequestJson(vm, stopRequest, startRequest)
			Expect(err).ToNot(HaveOccurred())

			ref, err := patch.New(
				patch.WithAdd("/status", v1.VirtualMachineStatus{StateChangeRequests: []v1.VirtualMachineStateChangeRequest{stopRequest, startRequest}}),
			).GeneratePayload()
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(Equal(ref))
		})

		It("should create a start request if status exists", func() {
			vm := newMinimalVM(testVMName)
			vm.Status.Created = true

			startRequest := v1.VirtualMachineStateChangeRequest{
				Action: v1.StartRequest,
			}

			res, err := getChangeRequestJson(vm, startRequest)
			Expect(err).ToNot(HaveOccurred())

			ref, err := patch.New(
				patch.WithTest("/status/stateChangeRequests", nil),
				patch.WithAdd("/status/stateChangeRequests", []v1.VirtualMachineStateChangeRequest{startRequest}),
			).GeneratePayload()
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(Equal(ref))
		})

		It("should create a start request if status doesn't exist", func() {
			vm := newMinimalVM(testVMName)

			startRequest := v1.VirtualMachineStateChangeRequest{
				Action: v1.StartRequest,
			}

			res, err := getChangeRequestJson(vm, startRequest)
			Expect(err).ToNot(HaveOccurred())

			ref, err := patch.New(
				patch.WithAdd("/status", v1.VirtualMachineStatus{StateChangeRequests: []v1.VirtualMachineStateChangeRequest{startRequest}}),
			).GeneratePayload()
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(Equal(ref))
		})

		It("should force a stop request to override", func() {
			uid := uuid.NewUUID()
			vm := newMinimalVM(testVMName)
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

			ref, err := patch.New(
				patch.WithTest("/status/stateChangeRequests", []v1.VirtualMachineStateChangeRequest{startRequest}),
				patch.WithReplace("/status/stateChangeRequests", []v1.VirtualMachineStateChangeRequest{stopRequest}),
			).GeneratePayload()
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(Equal(ref))
		})

		It("should error on start request if other requests exist", func() {
			uid := uuid.NewUUID()
			vm := newMinimalVM(testVMName)
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
			vm := newMinimalVM(testVMName)
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
			// In case of dry-run the request is not propagated to the handler
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

			Entry("a running VMI with LivenessProbe", Running, UnPaused, withLivenessProbe, &v1.PauseOptions{}, http.StatusForbidden, "Pausing VMIs with LivenessProbe is currently not supported"),
			Entry("a running VMI with LivenessProbe with dry-run option", Running, UnPaused, withLivenessProbe, &v1.PauseOptions{DryRun: withDryRun()}, http.StatusForbidden, "Pausing VMIs with LivenessProbe is currently not supported"),
		)

		DescribeTable("Should fail unpausing", func(running bool, paused bool, unpauseOptions *v1.UnpauseOptions, expectedError string) {
			expectVMI(running, paused)

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

		DescribeTable("Should unpause a running, paused VMI according to options", func(unpauseOptions *v1.UnpauseOptions, matchExpectation gomegatypes.GomegaMatcher) {
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
			// In case of dry-run the request is not propagated to the handler
			Expect(backend.ReceivedRequests()).To(matchExpectation)
		},
			Entry("with default", &v1.UnpauseOptions{}, HaveLen(1)),
			Entry("with dry-run option", &v1.UnpauseOptions{DryRun: withDryRun()}, BeNil()),
		)
	})

	Context("Subresource api - start paused", func() {
		BeforeEach(func() {
			request.PathParameters()["name"] = testVMName
			request.PathParameters()["namespace"] = k8smetav1.NamespaceDefault
		})
		DescribeTable("should patch status on start according to options", func(startOptions *v1.StartOptions) {
			bytesRepresentation, _ := json.Marshal(startOptions)
			request.Request.Body = io.NopCloser(bytes.NewReader(bytesRepresentation))

			vm := v1.VirtualMachine{
				ObjectMeta: k8smetav1.ObjectMeta{
					Name:      testVMName,
					Namespace: k8smetav1.NamespaceDefault,
				},
				Spec: v1.VirtualMachineSpec{
					Running:  pointer.P(NotRunning),
					Template: &v1.VirtualMachineInstanceTemplateSpec{},
				},
			}
			vmi := v1.VirtualMachineInstance{
				Spec: v1.VirtualMachineInstanceSpec{},
			}
			vmi.ObjectMeta.SetUID(uuid.NewUUID())

			vmClient.EXPECT().Get(context.Background(), vm.Name, k8smetav1.GetOptions{}).Return(&vm, nil)
			vmiClient.EXPECT().Get(context.Background(), vm.Name, k8smetav1.GetOptions{}).Return(&vmi, nil)
			vmClient.EXPECT().PatchStatus(context.Background(), vm.Name, types.JSONPatchType, gomock.Any(), gomock.Any()).DoAndReturn(
				func(ctx context.Context, name string, patchType types.PatchType, body interface{}, opts k8smetav1.PatchOptions) (interface{}, interface{}) {
					// check that dryRun option has been propagated to patch request
					Expect(opts.DryRun).To(BeEquivalentTo(startOptions.DryRun))
					return &vm, nil
				})

			app.StartVMRequestHandler(request, response)

			Expect(response.Error()).ToNot(HaveOccurred())
			Expect(response.StatusCode()).To(Equal(http.StatusAccepted))
		},
			Entry("with default", &v1.StartOptions{Paused: Paused}),
			Entry("with dry-run option", &v1.StartOptions{Paused: Paused, DryRun: withDryRun()}),
		)

		DescribeTable("should patch status on start for VM with RunStrategy",
			func(runStrategy v1.VirtualMachineRunStrategy) {
				vm := newVirtualMachineWithRunStrategy(runStrategy)
				vm.Spec.Template = &v1.VirtualMachineInstanceTemplateSpec{}
				body := map[string]bool{
					"paused": true,
				}
				bytesRepresentation, _ := json.Marshal(body)
				request.Request.Body = io.NopCloser(bytes.NewReader(bytesRepresentation))

				vmi := newVirtualMachineInstanceInPhase(v1.Succeeded)

				vmClient.EXPECT().Get(context.Background(), vm.Name, k8smetav1.GetOptions{}).Return(vm, nil)
				vmiClient.EXPECT().Get(context.Background(), vm.Name, k8smetav1.GetOptions{}).Return(vmi, nil)
				vmClient.EXPECT().PatchStatus(context.Background(), vm.Name, types.JSONPatchType, gomock.Any(), k8smetav1.PatchOptions{}).Return(vm, nil)

				app.StartVMRequestHandler(request, response)

				Expect(response.StatusCode()).To(Equal(http.StatusAccepted))
			},
			Entry("Manual RunStrategy", v1.RunStrategyManual),
			Entry("RerunOnFailure RunStrategy", v1.RunStrategyRerunOnFailure),
		)
	})

	AfterEach(func() {
		backend.Close()
	})
})

func newVirtualMachineWithRunStrategy(runStrategy v1.VirtualMachineRunStrategy) *v1.VirtualMachine {
	return &v1.VirtualMachine{
		ObjectMeta: k8smetav1.ObjectMeta{
			Name:      testVMName,
			Namespace: k8smetav1.NamespaceDefault,
		},
		Spec: v1.VirtualMachineSpec{
			RunStrategy: &runStrategy,
		},
	}
}

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

func ExpectMessage(recorder *httptest.ResponseRecorder, expected gomegatypes.GomegaMatcher) {
	status := k8smetav1.Status{}
	err := json.Unmarshal(recorder.Body.Bytes(), &status)

	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	ExpectWithOffset(1, status.Message).To(expected)
}
