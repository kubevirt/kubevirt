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
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang/mock/gomock"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	"k8s.io/utils/pointer"

	"github.com/emicklei/go-restful/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"

	"kubevirt.io/client-go/api"
	"kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/util/status"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"

	v1 "kubevirt.io/api/core/v1"
	cdifake "kubevirt.io/client-go/generated/containerized-data-importer/clientset/versioned/fake"
	"kubevirt.io/client-go/kubecli"

	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"

	"kubevirt.io/kubevirt/pkg/testutils"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
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

func getDryRunOption() []string {
	return []string{k8smetav1.DryRunAll}
}

var _ = Describe("VirtualMachineInstance Subresources", func() {
	kubecli.Init()

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

	gracePeriodZero := pointer.Int64(0)

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
		flag.Set("kubeconfig", "")
		app.virtCli = virtClient
		app.statusUpdater = status.NewVMStatusUpdater(app.virtCli)
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

		vmiClient.EXPECT().Get(context.Background(), vmi.Name, &k8smetav1.GetOptions{}).Return(&vmi, nil)

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

		Context("VNC", func() {
			It("should fail with no 'name' path param", func() {

				vmiClient.EXPECT().Get(context.Background(), "", &k8smetav1.GetOptions{}).Return(nil, errors.NewInternalError(fmt.Errorf("no name defined")))

				app.VNCRequestHandler(request, response)
				ExpectStatusErrorWithCode(recorder, http.StatusInternalServerError)
			})

			It("should fail with no 'namespace' path param", func() {

				request.PathParameters()["name"] = testVMIName

				vmiClient.EXPECT().Get(context.Background(), testVMIName, &k8smetav1.GetOptions{}).Return(nil, errors.NewInternalError(fmt.Errorf("no namespace defined")))

				app.VNCRequestHandler(request, response)
				ExpectStatusErrorWithCode(recorder, http.StatusInternalServerError)
			})

			It("should fail if vmi is not found", func() {

				request.PathParameters()["name"] = testVMIName
				request.PathParameters()["namespace"] = k8smetav1.NamespaceDefault

				vmiClient.EXPECT().Get(context.Background(), testVMIName, &k8smetav1.GetOptions{}).Return(nil, errors.NewNotFound(v1.Resource("virtualmachine"), testVMIName))

				app.VNCRequestHandler(request, response)
				ExpectStatusErrorWithCode(recorder, http.StatusNotFound)
			})

			It("should fail with internal at fetching vmi errors", func() {

				request.PathParameters()["name"] = testVMIName
				request.PathParameters()["namespace"] = k8smetav1.NamespaceDefault

				vmiClient.EXPECT().Get(context.Background(), testVMIName, &k8smetav1.GetOptions{}).Return(nil, errors.NewInternalError(fmt.Errorf("unable to retrieve vmi [%s]", testVMIName)))

				app.VNCRequestHandler(request, response)
				ExpectStatusErrorWithCode(recorder, http.StatusInternalServerError)
			})

			DescribeTable("request validation", func(autoattachGraphicsDevice bool, phase v1.VirtualMachineInstancePhase) {
				request.PathParameters()["name"] = testVMIName
				request.PathParameters()["namespace"] = k8smetav1.NamespaceDefault

				vmi := api.NewMinimalVMI(testVMIName)
				vmi.Status.Phase = phase
				vmi.ObjectMeta.SetUID(uuid.NewUUID())
				vmi.Spec.Domain.Devices.AutoattachGraphicsDevice = &autoattachGraphicsDevice

				vmiClient.EXPECT().Get(context.Background(), testVMIName, &k8smetav1.GetOptions{}).Return(vmi, nil)

				app.VNCRequestHandler(request, response)

				ExpectStatusErrorWithCode(recorder, http.StatusBadRequest)
			},
				Entry("should fail if there is no graphics device", false, v1.Running),
				Entry("should fail if vmi is not running", true, v1.Scheduling),
			)
		})

		Context("PortForward", func() {
			It("should fail with no 'name' path param", func() {

				vmiClient.EXPECT().Get(context.Background(), "", &k8smetav1.GetOptions{}).Return(nil, errors.NewInternalError(fmt.Errorf("no name defined")))

				app.PortForwardRequestHandler(app.FetchVirtualMachineInstance)(request, response)
				ExpectStatusErrorWithCode(recorder, http.StatusInternalServerError)
			})

			It("should fail with no 'namespace' path param", func() {

				request.PathParameters()["name"] = testVMIName

				vmiClient.EXPECT().Get(context.Background(), testVMIName, &k8smetav1.GetOptions{}).Return(nil, errors.NewInternalError(fmt.Errorf("no namespace defined")))

				app.PortForwardRequestHandler(app.FetchVirtualMachineInstance)(request, response)
				ExpectStatusErrorWithCode(recorder, http.StatusInternalServerError)
			})

			It("should fail if vmi is not found", func() {

				request.PathParameters()["name"] = testVMIName
				request.PathParameters()["namespace"] = k8smetav1.NamespaceDefault

				vmiClient.EXPECT().Get(context.Background(), testVMIName, &k8smetav1.GetOptions{}).Return(nil, errors.NewNotFound(v1.Resource("virtualmachine"), testVMIName))

				app.PortForwardRequestHandler(app.FetchVirtualMachineInstance)(request, response)
				ExpectStatusErrorWithCode(recorder, http.StatusNotFound)
			})

			It("should fail with internal at fetching vmi errors", func() {

				request.PathParameters()["name"] = testVMIName
				request.PathParameters()["namespace"] = k8smetav1.NamespaceDefault

				vmiClient.EXPECT().Get(context.Background(), testVMIName, &k8smetav1.GetOptions{}).Return(nil, errors.NewInternalError(fmt.Errorf("unable to retrieve vmi [%s]", testVMIName)))

				app.PortForwardRequestHandler(app.FetchVirtualMachineInstance)(request, response)
				ExpectStatusErrorWithCode(recorder, http.StatusInternalServerError)
			})

		})

		Context("console", func() {
			DescribeTable("request validation", func(autoattachSerialConsole bool, phase v1.VirtualMachineInstancePhase) {
				request.PathParameters()["name"] = testVMIName
				request.PathParameters()["namespace"] = k8smetav1.NamespaceDefault

				vmi := api.NewMinimalVMI(testVMIName)
				vmi.Status.Phase = phase
				vmi.ObjectMeta.SetUID(uuid.NewUUID())
				vmi.Spec.Domain.Devices.AutoattachSerialConsole = &autoattachSerialConsole

				vmiClient.EXPECT().Get(context.Background(), vmi.Name, &k8smetav1.GetOptions{}).Return(vmi, nil)

				app.ConsoleRequestHandler(request, response)
				ExpectStatusErrorWithCode(recorder, http.StatusBadRequest)
			},
				Entry("should fail if there is no serial console", false, v1.Running),
				Entry("should fail if vmi is not running", true, v1.Scheduling),
			)

			It("should fail to connect to the serial console if the VMI is Failed", func() {

				request.PathParameters()["name"] = testVMIName
				request.PathParameters()["namespace"] = k8smetav1.NamespaceDefault

				expectVMI(NotRunning, UnPaused)

				app.ConsoleRequestHandler(request, response)
				ExpectStatusErrorWithCode(recorder, http.StatusConflict)
			})

		})

		Context("restart", func() {
			It("should fail if VirtualMachine not exists", func() {
				request.PathParameters()["name"] = testVMName
				request.PathParameters()["namespace"] = k8smetav1.NamespaceDefault

				vmClient.EXPECT().Get(context.Background(), testVMName, &k8smetav1.GetOptions{}).Return(nil, errors.NewNotFound(v1.Resource("virtualmachine"), testVMName))

				app.RestartVMRequestHandler(request, response)

				ExpectStatusErrorWithCode(recorder, http.StatusNotFound)
			})

			It("should fail if VirtualMachine is not in running state", func() {
				request.PathParameters()["name"] = testVMName
				request.PathParameters()["namespace"] = k8smetav1.NamespaceDefault

				vm := v1.VirtualMachine{
					Spec: v1.VirtualMachineSpec{
						Running: pointer.Bool(NotRunning),
					},
				}

				vmClient.EXPECT().Get(context.Background(), testVMName, &k8smetav1.GetOptions{}).Return(&vm, nil)

				app.RestartVMRequestHandler(request, response)

				status := ExpectStatusErrorWithCode(recorder, http.StatusConflict)
				// check the msg string that would be presented to virtctl output
				Expect(status.Error()).To(ContainSubstring("Halted does not support manual restart requests"))
			})

			DescribeTable("should ForceRestart VirtualMachine according to options", func(restartOptions *v1.RestartOptions) {
				request.PathParameters()["name"] = testVMName
				request.PathParameters()["namespace"] = k8smetav1.NamespaceDefault

				bytesRepresentation, _ := json.Marshal(restartOptions)
				request.Request.Body = io.NopCloser(bytes.NewReader(bytesRepresentation))

				vm := newVirtualMachineWithRunning(pointer.Bool(Running))
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

				vmClient.EXPECT().Get(context.Background(), vm.Name, &k8smetav1.GetOptions{}).Return(vm, nil)
				vmiClient.EXPECT().Get(context.Background(), vm.Name, &k8smetav1.GetOptions{}).Return(&vmi, nil)
				vmClient.EXPECT().PatchStatus(context.Background(), vm.Name, types.JSONPatchType, gomock.Any(), gomock.Any()).DoAndReturn(
					func(ctx context.Context, name string, patchType types.PatchType, body interface{}, opts *k8smetav1.PatchOptions) (interface{}, interface{}) {
						//check that dryRun option has been propagated to patch request
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
				Entry("with dry-run option", &v1.RestartOptions{GracePeriodSeconds: gracePeriodZero, DryRun: getDryRunOption()}),
			)

			It("should not ForceRestart VirtualMachine if no Pods found for the VMI", func() {
				request.PathParameters()["name"] = testVMName
				request.PathParameters()["namespace"] = k8smetav1.NamespaceDefault

				body := map[string]int64{
					"gracePeriodSeconds": 0,
				}
				bytesRepresentation, _ := json.Marshal(body)
				request.Request.Body = io.NopCloser(bytes.NewReader(bytesRepresentation))

				vm := newVirtualMachineWithRunning(pointer.Bool(Running))
				vmi := v1.VirtualMachineInstance{
					Spec: v1.VirtualMachineInstanceSpec{},
				}
				vmi.ObjectMeta.SetUID(uuid.NewUUID())

				podList := k8sv1.PodList{}
				podList.Items = []k8sv1.Pod{}

				kubeClient.Fake.PrependReactor("list", "pods", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
					return true, &podList, nil
				})
				vmClient.EXPECT().Get(context.Background(), vm.Name, &k8smetav1.GetOptions{}).Return(vm, nil)
				vmiClient.EXPECT().Get(context.Background(), vm.Name, &k8smetav1.GetOptions{}).Return(&vmi, nil)
				vmClient.EXPECT().PatchStatus(context.Background(), vm.Name, types.JSONPatchType, gomock.Any(), &k8smetav1.PatchOptions{}).Return(vm, nil)

				app.RestartVMRequestHandler(request, response)

				Expect(response.Error()).ToNot(HaveOccurred())
				Expect(response.StatusCode()).To(Equal(http.StatusAccepted))
			})

			It("should restart VirtualMachine", func() {
				request.PathParameters()["name"] = testVMName
				request.PathParameters()["namespace"] = k8smetav1.NamespaceDefault

				vm := newVirtualMachineWithRunning(pointer.Bool(Running))

				vmi := v1.VirtualMachineInstance{
					Spec: v1.VirtualMachineInstanceSpec{},
				}

				vmi.ObjectMeta.SetUID(uuid.NewUUID())
				vmClient.EXPECT().Get(context.Background(), vm.Name, &k8smetav1.GetOptions{}).Return(vm, nil)
				vmiClient.EXPECT().Get(context.Background(), vm.Name, &k8smetav1.GetOptions{}).Return(&vmi, nil)
				vmClient.EXPECT().PatchStatus(context.Background(), vm.Name, types.JSONPatchType, gomock.Any(), &k8smetav1.PatchOptions{}).Return(vm, nil)

				app.RestartVMRequestHandler(request, response)

				Expect(response.Error()).ToNot(HaveOccurred())
				Expect(response.StatusCode()).To(Equal(http.StatusAccepted))
			})

			It("should start VirtualMachine if VMI doesn't exist", func() {
				request.PathParameters()["name"] = testVMName
				request.PathParameters()["namespace"] = k8smetav1.NamespaceDefault

				vm := newVirtualMachineWithRunning(pointer.Bool(Running))
				vmi := newVirtualMachineInstanceInPhase(v1.Running)

				vmClient.EXPECT().Get(context.Background(), vm.Name, &k8smetav1.GetOptions{}).Return(vm, nil)
				vmiClient.EXPECT().Get(context.Background(), vm.Name, &k8smetav1.GetOptions{}).Return(vmi, nil)
				vmClient.EXPECT().PatchStatus(context.Background(), vm.Name, types.JSONPatchType, gomock.Any(), &k8smetav1.PatchOptions{}).Return(vm, nil)

				app.RestartVMRequestHandler(request, response)
				Expect(response.Error()).NotTo(HaveOccurred())
				Expect(response.StatusCode()).To(Equal(http.StatusAccepted))
			})
		})

		Context("stop", func() {
			DescribeTable("should ForceStop VirtualMachine according to options", func(statusPhase v1.VirtualMachineInstancePhase, stopOptions *v1.StopOptions) {
				request.PathParameters()["name"] = testVMName
				request.PathParameters()["namespace"] = k8smetav1.NamespaceDefault
				var terminationGracePeriodSeconds int64 = 1800

				bytesRepresentation, _ := json.Marshal(stopOptions)
				request.Request.Body = io.NopCloser(bytes.NewReader(bytesRepresentation))

				vm := newVirtualMachineWithRunning(pointer.Bool(Running))

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

				vmClient.EXPECT().Get(context.Background(), vm.Name, &k8smetav1.GetOptions{}).Return(vm, nil)
				vmiClient.EXPECT().Get(context.Background(), vmi.Name, &k8smetav1.GetOptions{}).Return(&vmi, nil)
				vmiClient.EXPECT().Patch(context.Background(), vmi.Name, types.MergePatchType, gomock.Any(), gomock.Any()).DoAndReturn(
					func(ctx context.Context, name string, patchType types.PatchType, body interface{}, opts *k8smetav1.PatchOptions, _ ...string) (interface{}, interface{}) {
						//check that dryRun option has been propagated to patch request
						Expect(opts.DryRun).To(BeEquivalentTo(stopOptions.DryRun))
						return &vmi, nil
					}).AnyTimes()
				vmClient.EXPECT().Patch(context.Background(), vm.Name, types.MergePatchType, gomock.Any(), gomock.Any()).DoAndReturn(
					func(ctx context.Context, name string, patchType types.PatchType, body interface{}, opts *k8smetav1.PatchOptions, _ ...string) (interface{}, interface{}) {
						//check that dryRun option has been propagated to patch request
						Expect(opts.DryRun).To(BeEquivalentTo(stopOptions.DryRun))
						return vm, nil
					})

				app.StopVMRequestHandler(request, response)
				Expect(response.Error()).ToNot(HaveOccurred())
				Expect(response.StatusCode()).To(Equal(http.StatusAccepted))
			},
				Entry("in status Running with default", v1.Running, &v1.StopOptions{GracePeriod: gracePeriodZero}),
				Entry("in status Failed with default", v1.Failed, &v1.StopOptions{GracePeriod: gracePeriodZero}),
				Entry("in status Running with dry-run", v1.Running, &v1.StopOptions{GracePeriod: gracePeriodZero, DryRun: getDryRunOption()}),
				Entry("in status Failed with dry-run", v1.Failed, &v1.StopOptions{GracePeriod: gracePeriodZero, DryRun: getDryRunOption()}),
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

			vmClient.EXPECT().Get(context.Background(), vm.Name, &k8smetav1.GetOptions{}).Return(vm, nil)

			app.RestartVMRequestHandler(request, response)

			status := ExpectStatusErrorWithCode(recorder, http.StatusConflict)
			// check the msg string that would be presented to virtctl output
			Expect(status.Error()).To(ContainSubstring("Halted does not support manual restart requests"))
		})

		DescribeTable("should not fail with VMI and RunStrategy", func(runStrategy v1.VirtualMachineRunStrategy) {
			vm := newVirtualMachineWithRunStrategy(runStrategy)
			vmi := newVirtualMachineInstanceInPhase(v1.Failed)

			vmClient.EXPECT().Get(context.Background(), vm.Name, &k8smetav1.GetOptions{}).Return(vm, nil)
			vmiClient.EXPECT().Get(context.Background(), vm.Name, &k8smetav1.GetOptions{}).Return(vmi, nil)
			vmClient.EXPECT().PatchStatus(context.Background(), vm.Name, types.JSONPatchType, gomock.Any(), &k8smetav1.PatchOptions{}).Return(vm, nil)

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

			vmClient.EXPECT().Get(context.Background(), vm.Name, &k8smetav1.GetOptions{}).Return(vm, nil)
			vmiClient.EXPECT().Get(context.Background(), vm.Name, &k8smetav1.GetOptions{}).Return(nil, errors.NewNotFound(v1.Resource("virtualmachineinstance"), vm.Name)).AnyTimes()

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

			Entry("Always with dry-run option", v1.RunStrategyAlways, "VM is not running", &v1.RestartOptions{DryRun: getDryRunOption()}),
			Entry("Manual with dry-run option", v1.RunStrategyManual, "VM is not running", &v1.RestartOptions{DryRun: getDryRunOption()}),
			Entry("RerunOnFailure with dry-run option", v1.RunStrategyRerunOnFailure, "VM is not running", &v1.RestartOptions{DryRun: getDryRunOption()}),
			Entry("Once with dry-run option", v1.RunStrategyOnce, "Once does not support manual restart requests", &v1.RestartOptions{DryRun: getDryRunOption()}),
			Entry("Halted with dry-run option", v1.RunStrategyHalted, "Halted does not support manual restart requests", &v1.RestartOptions{DryRun: getDryRunOption()}),
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
			request.PathParameters()["name"] = testVMName
			request.PathParameters()["namespace"] = k8smetav1.NamespaceDefault
		})

		DescribeTable("Should succeed with add volume request", func(addOpts *v1.AddVolumeOptions, removeOpts *v1.RemoveVolumeOptions, isVM bool, code int, enableGate bool) {

			if enableGate {
				enableFeatureGate(virtconfig.HotplugVolumesGate)
			}
			if addOpts != nil {
				request.Request.Body = newAddVolumeBody(addOpts)
			} else {
				request.Request.Body = newRemoveVolumeBody(removeOpts)
			}

			vmi := api.NewMinimalVMI(request.PathParameter("name"))
			vmi.Namespace = k8smetav1.NamespaceDefault
			vmi.Status.Phase = v1.Running
			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "existingvol",
			})
			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "hotpluggedPVC",
			})
			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				Name: "existingvol",
				VolumeSource: v1.VolumeSource{
					PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
						ClaimName: "testpvcdiskclaim",
					}},
				},
			})
			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				Name: "hotpluggedPVC",
				VolumeSource: v1.VolumeSource{
					PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
						PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: "hotpluggedPVC",
						},
						Hotpluggable: true,
					},
				},
			})

			if isVM {
				vm := newMinimalVM(request.PathParameter("name"))
				vm.Namespace = k8smetav1.NamespaceDefault
				vm.Spec.Template = &v1.VirtualMachineInstanceTemplateSpec{
					Spec: vmi.Spec,
				}

				patchedVM := vm.DeepCopy()
				patchedVM.Status.VolumeRequests = append(patchedVM.Status.VolumeRequests, v1.VirtualMachineVolumeRequest{AddVolumeOptions: addOpts, RemoveVolumeOptions: removeOpts})

				vmClient.EXPECT().Get(context.Background(), vm.Name, &k8smetav1.GetOptions{}).Return(vm, nil).AnyTimes()

				if addOpts != nil {
					vmClient.EXPECT().PatchStatus(context.Background(), vm.Name, types.JSONPatchType, gomock.Any(), gomock.Any()).DoAndReturn(
						func(ctx context.Context, name string, patchType types.PatchType, body interface{}, opts *k8smetav1.PatchOptions) (interface{}, interface{}) {
							//check that dryRun option has been propagated to patch request
							Expect(opts.DryRun).To(BeEquivalentTo(addOpts.DryRun))
							return patchedVM, nil
						}).AnyTimes()
					app.VMAddVolumeRequestHandler(request, response)
				} else {
					vmClient.EXPECT().PatchStatus(context.Background(), vm.Name, types.JSONPatchType, gomock.Any(), gomock.Any()).DoAndReturn(
						func(ctx context.Context, name string, patchType types.PatchType, body interface{}, opts *k8smetav1.PatchOptions) (interface{}, interface{}) {
							//check that dryRun option has been propagated to patch request
							Expect(opts.DryRun).To(BeEquivalentTo(removeOpts.DryRun))
							return patchedVM, nil
						})
					app.VMRemoveVolumeRequestHandler(request, response)
				}
			} else {
				vmiClient.EXPECT().Get(context.Background(), vmi.Name, &k8smetav1.GetOptions{}).Return(vmi, nil).AnyTimes()

				if addOpts != nil {
					vmiClient.EXPECT().Patch(context.Background(), vmi.Name, types.JSONPatchType, gomock.Any(), gomock.Any()).DoAndReturn(
						func(ctx context.Context, name string, patchType types.PatchType, body interface{}, opts *k8smetav1.PatchOptions, _ ...string) (interface{}, interface{}) {
							//check that dryRun option has been propagated to patch request
							Expect(opts.DryRun).To(BeEquivalentTo(addOpts.DryRun))
							return vmi, nil
						}).AnyTimes()
					app.VMIAddVolumeRequestHandler(request, response)
				} else {
					vmiClient.EXPECT().Patch(context.Background(), vmi.Name, types.JSONPatchType, gomock.Any(), gomock.Any()).DoAndReturn(
						func(ctx context.Context, name string, patchType types.PatchType, body interface{}, opts *k8smetav1.PatchOptions, _ ...string) (interface{}, interface{}) {
							//check that dryRun option has been propagated to patch request
							Expect(opts.DryRun).To(BeEquivalentTo(removeOpts.DryRun))
							return vmi, nil
						}).AnyTimes()
					app.VMIRemoveVolumeRequestHandler(request, response)
				}
			}

			Expect(response.StatusCode()).To(Equal(code))
		},
			Entry("VM with a valid add volume request", &v1.AddVolumeOptions{
				Name:         "vol1",
				Disk:         &v1.Disk{},
				VolumeSource: &v1.HotplugVolumeSource{},
			}, nil, true, http.StatusAccepted, true),
			Entry("VMI with a valid add volume request", &v1.AddVolumeOptions{
				Name:         "vol1",
				Disk:         &v1.Disk{},
				VolumeSource: &v1.HotplugVolumeSource{},
			}, nil, false, http.StatusAccepted, true),
			Entry("VMI with an invalid add volume request that's missing a name", &v1.AddVolumeOptions{
				VolumeSource: &v1.HotplugVolumeSource{},
				Disk:         &v1.Disk{},
			}, nil, false, http.StatusBadRequest, true),
			Entry("VMI with an invalid add volume request that's missing a disk", &v1.AddVolumeOptions{
				Name:         "vol1",
				VolumeSource: &v1.HotplugVolumeSource{},
			}, nil, false, http.StatusBadRequest, true),
			Entry("VMI with an invalid add volume request that's missing a volume", &v1.AddVolumeOptions{
				Name: "vol1",
				Disk: &v1.Disk{},
			}, nil, false, http.StatusBadRequest, true),
			Entry("VM with a valid remove volume request", nil, &v1.RemoveVolumeOptions{
				Name: "hotpluggedPVC",
			}, true, http.StatusAccepted, true),
			Entry("VMI with a valid remove volume request", nil, &v1.RemoveVolumeOptions{
				Name: "hotpluggedPVC",
			}, false, http.StatusAccepted, true),
			Entry("VMI with a invalid remove volume request missing a name", nil, &v1.RemoveVolumeOptions{}, false, http.StatusBadRequest, true),
			Entry("VMI with a valid remove volume request but no feature gate", nil, &v1.RemoveVolumeOptions{
				Name: "existingvol",
			}, false, http.StatusBadRequest, false),
			Entry("VM with a valid add volume request but no feature gate", &v1.AddVolumeOptions{
				Name:         "vol1",
				Disk:         &v1.Disk{},
				VolumeSource: &v1.HotplugVolumeSource{},
			}, nil, true, http.StatusBadRequest, false),
			Entry("VM with a valid add volume request with DryRun", &v1.AddVolumeOptions{
				Name:         "vol1",
				Disk:         &v1.Disk{},
				VolumeSource: &v1.HotplugVolumeSource{},
				DryRun:       getDryRunOption(),
			}, nil, true, http.StatusAccepted, true),
			Entry("VMI with a valid add volume request with DryRun", &v1.AddVolumeOptions{
				Name:         "vol1",
				Disk:         &v1.Disk{},
				VolumeSource: &v1.HotplugVolumeSource{},
				DryRun:       getDryRunOption(),
			}, nil, false, http.StatusAccepted, true),
			Entry("VMI with an invalid add volume request that's missing a name with DryRun", &v1.AddVolumeOptions{
				VolumeSource: &v1.HotplugVolumeSource{},
				Disk:         &v1.Disk{},
				DryRun:       getDryRunOption(),
			}, nil, false, http.StatusBadRequest, true),
			Entry("VMI with an invalid add volume request that's missing a disk with DryRun", &v1.AddVolumeOptions{
				Name:         "vol1",
				VolumeSource: &v1.HotplugVolumeSource{},
				DryRun:       getDryRunOption(),
			}, nil, false, http.StatusBadRequest, true),
			Entry("VMI with an invalid add volume request that's missing a volume with DryRun", &v1.AddVolumeOptions{
				Name:   "vol1",
				Disk:   &v1.Disk{},
				DryRun: getDryRunOption(),
			}, nil, false, http.StatusBadRequest, true),
			Entry("VM with a valid remove volume request with DryRun", nil, &v1.RemoveVolumeOptions{
				Name:   "hotpluggedPVC",
				DryRun: getDryRunOption(),
			}, true, http.StatusAccepted, true),
			Entry("VMI with a valid remove volume request with DryRun", nil, &v1.RemoveVolumeOptions{
				Name:   "hotpluggedPVC",
				DryRun: getDryRunOption(),
			}, false, http.StatusAccepted, true),
			Entry("VMI with a invalid remove volume request missing a name with DryRun", nil, &v1.RemoveVolumeOptions{
				DryRun: getDryRunOption(),
			}, false, http.StatusBadRequest, true),
			Entry("VMI with a valid remove volume request but no feature gate with DryRun", nil, &v1.RemoveVolumeOptions{
				Name:   "existingvol",
				DryRun: getDryRunOption(),
			}, false, http.StatusBadRequest, false),
			Entry("VM with a valid add volume request but no feature gate with DryRun", &v1.AddVolumeOptions{
				Name:         "vol1",
				Disk:         &v1.Disk{},
				VolumeSource: &v1.HotplugVolumeSource{},
				DryRun:       getDryRunOption(),
			}, nil, true, http.StatusBadRequest, false),
		)

		DescribeTable("Should generate expected vmi patch", func(volumeRequest *v1.VirtualMachineVolumeRequest, expectedPatch string, expectError bool) {

			vmi := api.NewMinimalVMI(request.PathParameter("name"))
			vmi.Namespace = k8smetav1.NamespaceDefault
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
				Expect(err).To(HaveOccurred())
			} else {
				Expect(err).ToNot(HaveOccurred())
			}

			Expect(patch).To(Equal(expectedPatch))
		},
			Entry("add volume request",
				&v1.VirtualMachineVolumeRequest{
					AddVolumeOptions: &v1.AddVolumeOptions{
						Name:         "vol1",
						Disk:         &v1.Disk{},
						VolumeSource: &v1.HotplugVolumeSource{},
					},
				},
				"[{ \"op\": \"test\", \"path\": \"/spec/volumes\", \"value\": [{\"name\":\"existingvol\",\"persistentVolumeClaim\":{\"claimName\":\"testpvcdiskclaim\"}}]}, { \"op\": \"test\", \"path\": \"/spec/domain/devices/disks\", \"value\": [{\"name\":\"existingvol\"}]}, { \"op\": \"replace\", \"path\": \"/spec/volumes\", \"value\": [{\"name\":\"existingvol\",\"persistentVolumeClaim\":{\"claimName\":\"testpvcdiskclaim\"}},{\"name\":\"vol1\"}]}, { \"op\": \"replace\", \"path\": \"/spec/domain/devices/disks\", \"value\": [{\"name\":\"existingvol\"},{\"name\":\"vol1\"}]}]",
				false),
			Entry("remove volume request",
				&v1.VirtualMachineVolumeRequest{
					RemoveVolumeOptions: &v1.RemoveVolumeOptions{
						Name: "existingvol",
					},
				},
				"[{ \"op\": \"test\", \"path\": \"/spec/volumes\", \"value\": [{\"name\":\"existingvol\",\"persistentVolumeClaim\":{\"claimName\":\"testpvcdiskclaim\"}}]}, { \"op\": \"test\", \"path\": \"/spec/domain/devices/disks\", \"value\": [{\"name\":\"existingvol\"}]}, { \"op\": \"replace\", \"path\": \"/spec/volumes\", \"value\": []}, { \"op\": \"replace\", \"path\": \"/spec/domain/devices/disks\", \"value\": []}]",
				false),
		)
		DescribeTable("Should generate expected vm patch", func(volumeRequest *v1.VirtualMachineVolumeRequest, existingVolumeRequests []v1.VirtualMachineVolumeRequest, expectedPatch string, expectError bool) {

			vm := newMinimalVM(request.PathParameter("name"))
			vm.Namespace = k8smetav1.NamespaceDefault

			if len(existingVolumeRequests) > 0 {
				vm.Status.VolumeRequests = existingVolumeRequests
			}

			patch, err := generateVMVolumeRequestPatch(vm, volumeRequest)
			if expectError {
				Expect(err).To(HaveOccurred())
			} else {
				Expect(err).ToNot(HaveOccurred())
			}

			Expect(patch).To(Equal(expectedPatch))
		},
			Entry("add volume request with no existing volumes",
				&v1.VirtualMachineVolumeRequest{
					AddVolumeOptions: &v1.AddVolumeOptions{
						Name:         "vol1",
						Disk:         &v1.Disk{},
						VolumeSource: &v1.HotplugVolumeSource{},
					},
				},
				nil,
				`[{"op":"test","path":"/status/volumeRequests","value":null},{"op":"add","path":"/status/volumeRequests","value":[{"addVolumeOptions":{"name":"vol1","disk":{"name":""},"volumeSource":{}}}]}]`,
				false),
			Entry("add volume request that already exists should fail",
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
			Entry("add volume request when volume requests alread exist",
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
				`[{"op":"test","path":"/status/volumeRequests","value":[{"addVolumeOptions":{"name":"vol2","disk":{"name":""},"volumeSource":{}}}]},{"op":"replace","path":"/status/volumeRequests","value":[{"addVolumeOptions":{"name":"vol2","disk":{"name":""},"volumeSource":{}}},{"addVolumeOptions":{"name":"vol1","disk":{"name":""},"volumeSource":{}}}]}]`,
				false),
			Entry("remove volume request with no existing volume request", &v1.VirtualMachineVolumeRequest{
				RemoveVolumeOptions: &v1.RemoveVolumeOptions{
					Name: "vol1",
				},
			},
				nil,
				`[{"op":"test","path":"/status/volumeRequests","value":null},{"op":"add","path":"/status/volumeRequests","value":[{"removeVolumeOptions":{"name":"vol1"}}]}]`,
				false),
			Entry("remove volume request should replace add volume request",
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
				`[{"op":"test","path":"/status/volumeRequests","value":[{"addVolumeOptions":{"name":"vol2","disk":{"name":""},"volumeSource":{}}}]},{"op":"replace","path":"/status/volumeRequests","value":[{"removeVolumeOptions":{"name":"vol2"}}]}]`,
				false),
			Entry("remove volume request that already exists should fail",
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

		DescribeTable("Should verify volume option", func(volumeRequest *v1.VirtualMachineVolumeRequest, existingVolumes []v1.Volume, expectedError string) {
			err := verifyVolumeOption(existingVolumes, volumeRequest)
			if expectedError != "" {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal(expectedError))
			} else {
				Expect(err).ToNot(HaveOccurred())
			}
		},
			Entry("add volume name which already exists should fail",
				&v1.VirtualMachineVolumeRequest{
					AddVolumeOptions: &v1.AddVolumeOptions{
						Name:         "vol1",
						Disk:         &v1.Disk{},
						VolumeSource: &v1.HotplugVolumeSource{},
					},
				},
				[]v1.Volume{
					{
						Name: "vol1",
						VolumeSource: v1.VolumeSource{
							DataVolume: &v1.DataVolumeSource{
								Name: "dv1",
							},
						},
					},
				},
				"Unable to add volume [vol1] because volume with that name already exists"),
			Entry("add volume source which already exists should fail(existing dv)",
				&v1.VirtualMachineVolumeRequest{
					AddVolumeOptions: &v1.AddVolumeOptions{
						Name: "dv1",
						Disk: &v1.Disk{},
						VolumeSource: &v1.HotplugVolumeSource{
							DataVolume: &v1.DataVolumeSource{
								Name: "dv1",
							},
						},
					},
				},
				[]v1.Volume{
					{
						Name: "vol1",
						VolumeSource: v1.VolumeSource{
							DataVolume: &v1.DataVolumeSource{
								Name: "dv1",
							},
						},
					},
				},
				"Unable to add volume source [dv1] because it already exists"),
			Entry("add volume which source already exists should fail(existing pvc)",
				&v1.VirtualMachineVolumeRequest{
					AddVolumeOptions: &v1.AddVolumeOptions{
						Name: "pvc1",
						Disk: &v1.Disk{},
						VolumeSource: &v1.HotplugVolumeSource{
							PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
								ClaimName: "pvc1",
							}},
						},
					},
				},
				[]v1.Volume{
					{
						Name: "vol1",
						VolumeSource: v1.VolumeSource{
							PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
								ClaimName: "pvc1",
							}},
						},
					},
				},
				"Unable to add volume source [pvc1] because it already exists"),
			Entry("remove volume which doesnt exist should fail",
				&v1.VirtualMachineVolumeRequest{
					RemoveVolumeOptions: &v1.RemoveVolumeOptions{
						Name: "vol1",
					},
				},
				[]v1.Volume{},
				"Unable to remove volume [vol1] because it does not exist"),
			Entry("remove volume which wasnt hotplugged should fail(existing dv)",
				&v1.VirtualMachineVolumeRequest{
					RemoveVolumeOptions: &v1.RemoveVolumeOptions{
						Name: "dv1",
					},
				},
				[]v1.Volume{
					{
						Name: "vol1",
						VolumeSource: v1.VolumeSource{
							DataVolume: &v1.DataVolumeSource{
								Name: "dv1",
							},
						},
					},
				},
				"Unable to remove volume [vol1] because it is not hotpluggable"),
			Entry("remove volume which wasnt hotplugged should fail(existing cloudInit)",
				&v1.VirtualMachineVolumeRequest{
					RemoveVolumeOptions: &v1.RemoveVolumeOptions{
						Name: "cloudinitdisk",
					},
				},
				[]v1.Volume{
					{
						Name: "cloudinitdisk",
						VolumeSource: v1.VolumeSource{
							CloudInitNoCloud: &v1.CloudInitNoCloudSource{},
						},
					},
				},
				"Unable to remove volume [cloudinitdisk] because it is not hotpluggable"),
		)
	})

	Context("Memory dump Subresource api", func() {
		const (
			fs          = false
			block       = true
			notReadOnly = false
			readOnly    = true
			testPVCName = "testPVC"
		)
		var cdiClient *cdifake.Clientset

		newMemoryDumpBody := func(req *v1.VirtualMachineMemoryDumpRequest) io.ReadCloser {
			reqJson, _ := json.Marshal(req)
			return &readCloserWrapper{bytes.NewReader(reqJson)}
		}

		createTestPVC := func(size string, blockMode bool, readOnlyMode bool) *k8sv1.PersistentVolumeClaim {
			quantity, _ := resource.ParseQuantity(size)
			pvc := &k8sv1.PersistentVolumeClaim{
				ObjectMeta: k8smetav1.ObjectMeta{
					Name:      testPVCName,
					Namespace: k8smetav1.NamespaceDefault,
				},
				Spec: k8sv1.PersistentVolumeClaimSpec{
					Resources: k8sv1.ResourceRequirements{
						Requests: k8sv1.ResourceList{
							k8sv1.ResourceStorage: quantity,
						},
					},
				},
			}
			if blockMode {
				volumeMode := k8sv1.PersistentVolumeBlock
				pvc.Spec.VolumeMode = &volumeMode
			}
			if readOnlyMode {
				pvc.Spec.AccessModes = []k8sv1.PersistentVolumeAccessMode{k8sv1.ReadOnlyMany}
			}
			return pvc
		}

		cdiConfigInit := func() (cdiConfig *v1beta1.CDIConfig) {
			cdiConfig = &v1beta1.CDIConfig{
				ObjectMeta: k8smetav1.ObjectMeta{
					Name: storagetypes.ConfigName,
				},
				Spec: v1beta1.CDIConfigSpec{
					UploadProxyURLOverride: nil,
				},
				Status: v1beta1.CDIConfigStatus{
					FilesystemOverhead: &v1beta1.FilesystemOverhead{
						Global: storagetypes.DefaultFSOverhead,
					},
				},
			}
			return
		}

		BeforeEach(func() {
			request.PathParameters()["name"] = testVMName
			request.PathParameters()["namespace"] = k8smetav1.NamespaceDefault
			cdiConfig := cdiConfigInit()
			cdiClient = cdifake.NewSimpleClientset(cdiConfig)
		})

		DescribeTable("With memory dump request", func(memDumpReq *v1.VirtualMachineMemoryDumpRequest, statusCode int, enableGate bool, vmiRunning bool, pvc *k8sv1.PersistentVolumeClaim) {

			if enableGate {
				enableFeatureGate(virtconfig.HotplugVolumesGate)
			}
			request.Request.Body = newMemoryDumpBody(memDumpReq)

			vm := newMinimalVM(request.PathParameter("name"))
			vm.Namespace = k8smetav1.NamespaceDefault

			patchedVM := vm.DeepCopy()
			patchedVM.Status.MemoryDumpRequest = memDumpReq
			patchedVM.Status.MemoryDumpRequest.Phase = v1.MemoryDumpAssociating

			vmClient.EXPECT().Get(context.Background(), vm.Name, &k8smetav1.GetOptions{}).Return(vm, nil).AnyTimes()
			vmi := &v1.VirtualMachineInstance{}
			if vmiRunning {
				vmi = api.NewMinimalVMI(testVMIName)
				vmi.Status.Phase = v1.Running
				vmi.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
					k8sv1.ResourceMemory: resource.MustParse("1Gi"),
				}
				kubeClient.Fake.PrependReactor("get", "persistentvolumeclaims", func(action testing.Action) (bool, runtime.Object, error) {
					get, ok := action.(testing.GetAction)
					Expect(ok).To(BeTrue())
					Expect(get.GetNamespace()).To(Equal(k8smetav1.NamespaceDefault))
					Expect(get.GetName()).To(Equal(testPVCName))
					if pvc == nil {
						return true, nil, errors.NewNotFound(v1.Resource("persistentvolumeclaim"), testPVCName)
					}
					return true, pvc, nil
				})
			}
			if statusCode == http.StatusAccepted || (pvc != nil && pvc.Spec.Resources.Requests[k8sv1.ResourceStorage] == resource.MustParse("1Gi")) {
				virtClient.EXPECT().CdiClient().Return(cdiClient).AnyTimes()
			}
			vmiClient.EXPECT().Get(context.Background(), vm.Name, &k8smetav1.GetOptions{}).Return(vmi, nil).AnyTimes()
			vmClient.EXPECT().PatchStatus(context.Background(), vm.Name, types.JSONPatchType, gomock.Any(), gomock.Any()).DoAndReturn(
				func(ctx context.Context, name string, patchType types.PatchType, body interface{}, opts *k8smetav1.PatchOptions) (interface{}, interface{}) {
					return patchedVM, nil
				}).AnyTimes()
			app.MemoryDumpVMRequestHandler(request, response)

			Expect(response.StatusCode()).To(Equal(statusCode))
		},
			Entry("VM with a valid memory dump request should succeed", &v1.VirtualMachineMemoryDumpRequest{
				ClaimName: testPVCName,
			}, http.StatusAccepted, true, true, createTestPVC("2Gi", fs, notReadOnly)),
			Entry("VM with a valid memory dump request but no feature gate should fail", &v1.VirtualMachineMemoryDumpRequest{
				ClaimName: testPVCName,
			}, http.StatusBadRequest, false, true, createTestPVC("2Gi", fs, notReadOnly)),
			Entry("VM with a valid memory dump request vmi not running should fail", &v1.VirtualMachineMemoryDumpRequest{
				ClaimName: testPVCName,
			}, http.StatusConflict, true, false, createTestPVC("2Gi", fs, notReadOnly)),
			Entry("VM with a memory dump request with a non existing PVC", &v1.VirtualMachineMemoryDumpRequest{
				ClaimName: testPVCName,
			}, http.StatusNotFound, true, true, nil),
			Entry("VM with a memory dump request pvc block mode should fail", &v1.VirtualMachineMemoryDumpRequest{
				ClaimName: testPVCName,
			}, http.StatusConflict, true, true, createTestPVC("2Gi", block, notReadOnly)),
			Entry("VM with a memory dump request pvc read only mode should fail", &v1.VirtualMachineMemoryDumpRequest{
				ClaimName: testPVCName,
			}, http.StatusConflict, true, true, createTestPVC("2Gi", fs, readOnly)),
			Entry("VM with a memory dump request pvc size too small should fail", &v1.VirtualMachineMemoryDumpRequest{
				ClaimName: testPVCName,
			}, http.StatusConflict, true, true, createTestPVC("1Gi", fs, notReadOnly)),
		)

		DescribeTable("With memory dump request", func(memDumpReq, prevMemDumpReq *v1.VirtualMachineMemoryDumpRequest, statusCode int) {
			enableFeatureGate(virtconfig.HotplugVolumesGate)
			request.Request.Body = newMemoryDumpBody(memDumpReq)
			vm := newMinimalVM(request.PathParameter("name"))
			vm.Namespace = k8smetav1.NamespaceDefault
			if prevMemDumpReq != nil {
				vm.Status.MemoryDumpRequest = prevMemDumpReq
			}

			patchedVM := vm.DeepCopy()
			patchedVM.Status.MemoryDumpRequest = memDumpReq
			patchedVM.Status.MemoryDumpRequest.Phase = v1.MemoryDumpAssociating

			vmClient.EXPECT().Get(context.Background(), vm.Name, &k8smetav1.GetOptions{}).Return(vm, nil).AnyTimes()
			vmi := api.NewMinimalVMI(testVMIName)
			vmi.Status.Phase = v1.Running
			vmi.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
				k8sv1.ResourceMemory: resource.MustParse("1Gi"),
			}
			kubeClient.Fake.PrependReactor("get", "persistentvolumeclaims", func(action testing.Action) (bool, runtime.Object, error) {
				_, ok := action.(testing.GetAction)
				Expect(ok).To(BeTrue())
				return true, createTestPVC("2Gi", fs, notReadOnly), nil
			})
			vmiClient.EXPECT().Get(context.Background(), vm.Name, &k8smetav1.GetOptions{}).Return(vmi, nil).AnyTimes()
			vmClient.EXPECT().PatchStatus(context.Background(), vm.Name, types.JSONPatchType, gomock.Any(), gomock.Any()).DoAndReturn(
				func(ctx context.Context, name string, patchType types.PatchType, body interface{}, opts *k8smetav1.PatchOptions) (interface{}, interface{}) {
					return patchedVM, nil
				}).AnyTimes()
			if statusCode == http.StatusAccepted {
				virtClient.EXPECT().CdiClient().Return(cdiClient).AnyTimes()
			}
			app.MemoryDumpVMRequestHandler(request, response)

			Expect(response.StatusCode()).To(Equal(statusCode))
		},
			Entry("VM with a memory dump request without claim name with assocaited memory dump should succeed",
				&v1.VirtualMachineMemoryDumpRequest{},
				&v1.VirtualMachineMemoryDumpRequest{
					ClaimName: testPVCName,
					Phase:     v1.MemoryDumpCompleted,
				}, http.StatusAccepted),
			Entry("VM with a memory dump request missing claim name without previous memory dump should fail",
				&v1.VirtualMachineMemoryDumpRequest{}, nil, http.StatusBadRequest),
			Entry("VM with a memory dump request with claim name different then assocaited memory dump should fail",
				&v1.VirtualMachineMemoryDumpRequest{
					ClaimName: "diffPVCName",
				},
				&v1.VirtualMachineMemoryDumpRequest{
					ClaimName: testPVCName,
					Phase:     v1.MemoryDumpCompleted,
				}, http.StatusConflict),
		)

		DescribeTable("Should generate expected vm patch", func(memDumpReq *v1.VirtualMachineMemoryDumpRequest, existingMemDumpReq *v1.VirtualMachineMemoryDumpRequest, expectedPatch string, expectError bool, removeReq bool) {

			vm := newMinimalVM(request.PathParameter("name"))
			vm.Namespace = k8smetav1.NamespaceDefault

			if existingMemDumpReq != nil {
				vm.Status.MemoryDumpRequest = existingMemDumpReq
			}

			patch, err := generateVMMemoryDumpRequestPatch(vm, memDumpReq, removeReq)
			if expectError {
				Expect(err).To(HaveOccurred())
			} else {
				Expect(err).ToNot(HaveOccurred())
			}

			fmt.Println(patch)
			Expect(patch).To(Equal(expectedPatch))
		},
			Entry("add memory dump request with no existing request",
				&v1.VirtualMachineMemoryDumpRequest{
					ClaimName: "vol1",
					Phase:     v1.MemoryDumpAssociating,
				},
				nil,
				"[{ \"op\": \"test\", \"path\": \"/status/memoryDumpRequest\", \"value\": null}, { \"op\": \"add\", \"path\": \"/status/memoryDumpRequest\", \"value\": {\"claimName\":\"vol1\",\"phase\":\"Associating\"}}]",
				false, false),
			Entry("add memory dump request to the same vol after completed",
				&v1.VirtualMachineMemoryDumpRequest{
					ClaimName: "vol1",
					Phase:     v1.MemoryDumpAssociating,
				},
				&v1.VirtualMachineMemoryDumpRequest{
					ClaimName: "vol1",
					Phase:     v1.MemoryDumpCompleted,
				},
				"[{ \"op\": \"test\", \"path\": \"/status/memoryDumpRequest\", \"value\": {\"claimName\":\"vol1\",\"phase\":\"Completed\"}}, { \"op\": \"replace\", \"path\": \"/status/memoryDumpRequest\", \"value\": {\"claimName\":\"vol1\",\"phase\":\"Associating\"}}]",
				false, false),
			Entry("add memory dump request to the same vol after previous failed",
				&v1.VirtualMachineMemoryDumpRequest{
					ClaimName: "vol1",
					Phase:     v1.MemoryDumpAssociating,
				},
				&v1.VirtualMachineMemoryDumpRequest{
					ClaimName: "vol1",
					Phase:     v1.MemoryDumpFailed,
				},
				"[{ \"op\": \"test\", \"path\": \"/status/memoryDumpRequest\", \"value\": {\"claimName\":\"vol1\",\"phase\":\"Failed\"}}, { \"op\": \"replace\", \"path\": \"/status/memoryDumpRequest\", \"value\": {\"claimName\":\"vol1\",\"phase\":\"Associating\"}}]",
				false, false),
			Entry("add memory dump request to the same vol while memory dump in progress should fail",
				&v1.VirtualMachineMemoryDumpRequest{
					ClaimName: "vol1",
					Phase:     v1.MemoryDumpAssociating,
				},
				&v1.VirtualMachineMemoryDumpRequest{
					ClaimName: "vol1",
					Phase:     v1.MemoryDumpInProgress,
				},
				"",
				true, false),
			Entry("add memory dump request to the same vol while it is being dissociated should fail",
				&v1.VirtualMachineMemoryDumpRequest{
					ClaimName: "vol1",
					Phase:     v1.MemoryDumpAssociating,
				},
				&v1.VirtualMachineMemoryDumpRequest{
					ClaimName: "vol1",
					Phase:     v1.MemoryDumpDissociating,
				},
				"",
				true, false),
			Entry("remove memory dump request to already removed memory dump should fail",
				&v1.VirtualMachineMemoryDumpRequest{
					Phase:  v1.MemoryDumpDissociating,
					Remove: true,
				},
				nil,
				"",
				true, true),
			Entry("remove memory dump request to memory dump in progress should succeed",
				&v1.VirtualMachineMemoryDumpRequest{
					Phase:  v1.MemoryDumpDissociating,
					Remove: true,
				},
				&v1.VirtualMachineMemoryDumpRequest{
					ClaimName: "vol1",
					Phase:     v1.MemoryDumpInProgress,
				},
				"[{ \"op\": \"test\", \"path\": \"/status/memoryDumpRequest\", \"value\": {\"claimName\":\"vol1\",\"phase\":\"InProgress\"}}, { \"op\": \"replace\", \"path\": \"/status/memoryDumpRequest\", \"value\": {\"claimName\":\"vol1\",\"phase\":\"Dissociating\",\"remove\":true}}]",
				false, true),
			Entry("remove memory dump request with Remove request should fail",
				&v1.VirtualMachineMemoryDumpRequest{
					Phase:  v1.MemoryDumpDissociating,
					Remove: true,
				},
				&v1.VirtualMachineMemoryDumpRequest{
					ClaimName: "vol1",
					Phase:     v1.MemoryDumpDissociating,
					Remove:    true,
				},
				"",
				true, true),
			Entry("remove memory dump request to completed memory dump should succeed",
				&v1.VirtualMachineMemoryDumpRequest{
					Phase:  v1.MemoryDumpDissociating,
					Remove: true,
				},
				&v1.VirtualMachineMemoryDumpRequest{
					ClaimName: "vol1",
					Phase:     v1.MemoryDumpCompleted,
				},
				"[{ \"op\": \"test\", \"path\": \"/status/memoryDumpRequest\", \"value\": {\"claimName\":\"vol1\",\"phase\":\"Completed\"}}, { \"op\": \"replace\", \"path\": \"/status/memoryDumpRequest\", \"value\": {\"claimName\":\"vol1\",\"phase\":\"Dissociating\",\"remove\":true}}]",
				false, true),
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

				vmClient.EXPECT().Get(context.Background(), vm.Name, &k8smetav1.GetOptions{}).Return(vm, nil)
				vmiClient.EXPECT().Get(context.Background(), testVMName, &k8smetav1.GetOptions{}).DoAndReturn(func(ctx context.Context, name string, opts *k8smetav1.GetOptions) (interface{}, interface{}) {
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

			Entry("Always without VMI and with dry-run option", v1.RunStrategyAlways, v1.VmPhaseUnset, http.StatusNotFound, "Always does not support manual start requests", &v1.StartOptions{DryRun: getDryRunOption()}),
			Entry("Always with VMI in phase Running and with dry-run option", v1.RunStrategyAlways, v1.Running, http.StatusOK, "VM is already running", &v1.StartOptions{DryRun: getDryRunOption()}),
			Entry("Once with dry-run option", v1.RunStrategyOnce, v1.VmPhaseUnset, http.StatusNotFound, "Once does not support manual start requests", &v1.StartOptions{DryRun: getDryRunOption()}),
			Entry("RerunOnFailure with VMI in phase Failed and with dry-run option", v1.RunStrategyRerunOnFailure, v1.Failed, http.StatusOK, "RerunOnFailure does not support starting VM from failed state", &v1.StartOptions{DryRun: getDryRunOption()}),
		)

		DescribeTable("should not fail on VM with RunStrategy ",
			func(runStrategy v1.VirtualMachineRunStrategy, phase v1.VirtualMachineInstancePhase, status int) {
				vm := newVirtualMachineWithRunStrategy(runStrategy)
				var vmi *v1.VirtualMachineInstance
				if phase != v1.VmPhaseUnset {
					vmi = newVirtualMachineInstanceInPhase(phase)
				}

				vmClient.EXPECT().Get(context.Background(), vm.Name, &k8smetav1.GetOptions{}).Return(vm, nil)
				vmiClient.EXPECT().Get(context.Background(), vm.Name, &k8smetav1.GetOptions{}).Return(vmi, nil)
				vmClient.EXPECT().PatchStatus(context.Background(), vm.Name, types.JSONPatchType, gomock.Any(), &k8smetav1.PatchOptions{}).DoAndReturn(
					func(ctx context.Context, name string, patchType types.PatchType, body interface{}, opts *k8smetav1.PatchOptions) (interface{}, interface{}) {
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

			vmClient.EXPECT().Get(context.Background(), vm.Name, &k8smetav1.GetOptions{}).Return(vm, nil)
			vmiClient.EXPECT().Get(context.Background(), vm.Name, &k8smetav1.GetOptions{}).Return(nil, errors.NewNotFound(v1.Resource("virtualmachineinstance"), testVMName))
			if !expectError {
				vmClient.EXPECT().Patch(context.Background(), vm.Name, types.MergePatchType, gomock.Any(), gomock.Any()).DoAndReturn(
					func(ctx context.Context, name string, patchType types.PatchType, body interface{}, opts *k8smetav1.PatchOptions, _ ...string) (interface{}, interface{}) {
						//check that dryRun option has been propagated to patch request
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
			Entry("RunStrategyRerunOnFailure", v1.RunStrategyRerunOnFailure, "", false, &v1.StopOptions{}),
			Entry("RunStrategyManual", v1.RunStrategyManual, "VM is not running", true, &v1.StopOptions{}),
			Entry("RunStrategyHalted", v1.RunStrategyHalted, "VM is not running", true, &v1.StopOptions{}),

			Entry("RunStrategyAlways with dry-run option", v1.RunStrategyAlways, "", false, &v1.StopOptions{DryRun: getDryRunOption()}),
			Entry("RunStrategyOnce with dry-run option", v1.RunStrategyOnce, "", false, &v1.StopOptions{DryRun: getDryRunOption()}),
			Entry("RunStrategyRerunOnFailure with dry-run option", v1.RunStrategyRerunOnFailure, "", false, &v1.StopOptions{DryRun: getDryRunOption()}),
			Entry("RunStrategyManual with dry-run option", v1.RunStrategyManual, "VM is not running", true, &v1.StopOptions{DryRun: getDryRunOption()}),
			Entry("RunStrategyHalted with dry-run option", v1.RunStrategyHalted, "VM is not running", true, &v1.StopOptions{DryRun: getDryRunOption()}),
		)

		It("should fail on VM with VMI in Unknown Phase", func() {
			vm := newVirtualMachineWithRunStrategy(v1.RunStrategyHalted)
			vmi := newVirtualMachineInstanceInPhase(v1.Unknown)

			vmClient.EXPECT().Get(context.Background(), vm.Name, &k8smetav1.GetOptions{}).Return(vm, nil)
			vmiClient.EXPECT().Get(context.Background(), vm.Name, &k8smetav1.GetOptions{}).Return(vmi, nil)

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

			vmClient.EXPECT().Get(context.Background(), vm.Name, &k8smetav1.GetOptions{}).Return(vm, nil)
			vmiClient.EXPECT().Get(context.Background(), vm.Name, &k8smetav1.GetOptions{}).Return(vmi, nil)

			if graceperiod != nil {
				vmiClient.EXPECT().Patch(context.Background(), vmi.Name, types.MergePatchType, gomock.Any(), gomock.Any()).DoAndReturn(
					func(ctx context.Context, name string, patchType types.PatchType, body interface{}, opts *k8smetav1.PatchOptions, _ ...string) (interface{}, interface{}) {
						//check that dryRun option has been propagated to patch request
						Expect(opts.DryRun).To(BeEquivalentTo(stopOptions.DryRun))
						return vm, nil
					})
			}
			if !shouldFail {
				vmClient.EXPECT().PatchStatus(context.Background(), vm.Name, types.JSONPatchType, gomock.Any(), &k8smetav1.PatchOptions{}).Return(vm, nil)
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
			Entry("fail with nil graceperiod", pointer.Int64(int64(1800)), nil, true),
			Entry("fail with equal graceperiod", pointer.Int64(int64(1800)), pointer.Int64(int64(1800)), true),
			Entry("fail with greater graceperiod", pointer.Int64(int64(1800)), pointer.Int64(int64(2400)), true),
			Entry("not fail with non-nil graceperiod and nil termination graceperiod", nil, pointer.Int64(int64(1800)), false),
			Entry("not fail with shorter graceperiod and non-nil termination graceperiod", pointer.Int64(int64(1800)), pointer.Int64(int64(800)), false),
		)

		DescribeTable("should not fail on VM with RunStrategy", func(runStrategy v1.VirtualMachineRunStrategy) {
			vm := newVirtualMachineWithRunStrategy(runStrategy)
			vmi := newVirtualMachineInstanceInPhase(v1.Running)

			vmClient.EXPECT().Get(context.Background(), vm.Name, &k8smetav1.GetOptions{}).Return(vm, nil)
			vmiClient.EXPECT().Get(context.Background(), vm.Name, &k8smetav1.GetOptions{}).Return(vmi, nil)

			if runStrategy == v1.RunStrategyManual {
				vmClient.EXPECT().PatchStatus(context.Background(), vm.Name, types.JSONPatchType, gomock.Any(), &k8smetav1.PatchOptions{}).Return(vm, nil)
			} else {
				vmClient.EXPECT().Patch(context.Background(), vm.Name, types.MergePatchType, gomock.Any(), &k8smetav1.PatchOptions{}).Return(vm, nil)
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

			vmClient.EXPECT().Get(context.Background(), testVMName, &k8smetav1.GetOptions{}).Return(nil, errors.NewNotFound(v1.Resource("virtualmachine"), testVMName))
			app.MigrateVMRequestHandler(request, response)

			ExpectStatusErrorWithCode(recorder, http.StatusNotFound)
		},
			Entry("with default", &v1.MigrateOptions{}),
			Entry("with dry-run option", &v1.MigrateOptions{DryRun: getDryRunOption()}),
		)

		DescribeTable("should fail if VirtualMachine is not running according to options", func(migrateOptions *v1.MigrateOptions) {
			request.PathParameters()["name"] = testVMName
			request.PathParameters()["namespace"] = k8smetav1.NamespaceDefault

			vm := v1.VirtualMachine{}
			vmi := v1.VirtualMachineInstance{}

			bytesRepresentation, _ := json.Marshal(migrateOptions)
			request.Request.Body = io.NopCloser(bytes.NewReader(bytesRepresentation))

			vmClient.EXPECT().Get(context.Background(), testVMName, &k8smetav1.GetOptions{}).Return(&vm, nil)
			vmiClient.EXPECT().Get(context.Background(), testVMName, &k8smetav1.GetOptions{}).Return(&vmi, nil)

			app.MigrateVMRequestHandler(request, response)

			status := ExpectStatusErrorWithCode(recorder, http.StatusConflict)
			Expect(status.Error()).To(ContainSubstring("VM is not running"))
		},
			Entry("with default", &v1.MigrateOptions{}),
			Entry("with dry-run option", &v1.MigrateOptions{DryRun: getDryRunOption()}),
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

			vmClient.EXPECT().Get(context.Background(), testVMName, &k8smetav1.GetOptions{}).Return(&vm, nil)
			vmiClient.EXPECT().Get(context.Background(), testVMName, &k8smetav1.GetOptions{}).Return(&vmi, nil)
			migrateClient.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil, errors.NewInternalError(fmt.Errorf("error creating object")))
			app.MigrateVMRequestHandler(request, response)

			ExpectStatusErrorWithCode(recorder, http.StatusInternalServerError)
		},
			Entry("with default", &v1.MigrateOptions{}),
			Entry("with dry-run option", &v1.MigrateOptions{DryRun: getDryRunOption()}),
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

			vmClient.EXPECT().Get(context.Background(), testVMName, &k8smetav1.GetOptions{}).Return(&vm, nil)
			vmiClient.EXPECT().Get(context.Background(), testVMName, &k8smetav1.GetOptions{}).Return(&vmi, nil)

			migrateClient.EXPECT().Create(gomock.Any(), gomock.Any()).Do(
				func(obj interface{}, opts *k8smetav1.CreateOptions) {
					Expect(opts.DryRun).To(BeEquivalentTo(migrateOptions.DryRun))
				}).Return(&migration, nil)
			app.MigrateVMRequestHandler(request, response)

			Expect(response.Error()).ToNot(HaveOccurred())
			Expect(response.StatusCode()).To(Equal(http.StatusAccepted))
		},
			Entry("with default", &v1.MigrateOptions{}),
			Entry("with dry-run option", &v1.MigrateOptions{DryRun: getDryRunOption()}),
		)
	})

	Context("Subresource api - Guest OS Info", func() {
		type subRes func(request *restful.Request, response *restful.Response)

		DescribeTable("should fail when the VMI does not exist", func(fn subRes) {
			request.PathParameters()["name"] = testVMName
			request.PathParameters()["namespace"] = k8smetav1.NamespaceDefault

			vmiClient.EXPECT().Get(context.Background(), testVMName, &k8smetav1.GetOptions{}).Return(nil, errors.NewNotFound(v1.Resource("virtualmachineinstance"), testVMName))

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

			vmiClient.EXPECT().Get(context.Background(), testVMName, &k8smetav1.GetOptions{}).Return(&vmi, nil)

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

			vmiClient.EXPECT().Get(context.Background(), testVMName, &k8smetav1.GetOptions{}).Return(&vmi, nil)

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

			ref := fmt.Sprintf(`[{ "op": "test", "path": "/status/stateChangeRequests", "value": null}, { "op": "add", "path": "/status/stateChangeRequests", "value": [{"action":"Stop","uid":"%s"}]}]`, uid)
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

			ref := fmt.Sprintf(`[{ "op": "add", "path": "/status", "value": {"stateChangeRequests":[{"action":"Stop","uid":"%s"}]}}]`, uid)
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

			ref := fmt.Sprintf(`[{ "op": "test", "path": "/status/stateChangeRequests", "value": null}, { "op": "add", "path": "/status/stateChangeRequests", "value": [{"action":"Stop","uid":"%s"},{"action":"Start"}]}]`, uid)
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

			ref := fmt.Sprintf(`[{ "op": "add", "path": "/status", "value": {"stateChangeRequests":[{"action":"Stop","uid":"%s"},{"action":"Start"}]}}]`, uid)
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

			ref := fmt.Sprintf(`[{ "op": "test", "path": "/status/stateChangeRequests", "value": null}, { "op": "add", "path": "/status/stateChangeRequests", "value": [{"action":"Start"}]}]`)
			Expect(res).To(Equal(ref))
		})

		It("should create a start request if status doesn't exist", func() {
			vm := newMinimalVM(testVMName)

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

			ref := fmt.Sprintf(`[{ "op": "test", "path": "/status/stateChangeRequests", "value": [{"action":"Start"}]}, { "op": "replace", "path": "/status/stateChangeRequests", "value": [{"action":"Stop","uid":"%s"}]}]`, uid)
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
		DescribeTable("Should pause a running, not paused VMI according to options", func(pauseOptions *v1.PauseOptions) {

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
			Entry("with default", &v1.PauseOptions{}),
			Entry("with dry-run option", &v1.PauseOptions{DryRun: getDryRunOption()}),
		)

		DescribeTable("Should fail pausing", func(running bool, paused bool, pauseOptions *v1.PauseOptions) {

			expectVMI(running, paused)

			bytesRepresentation, _ := json.Marshal(pauseOptions)
			request.Request.Body = io.NopCloser(bytes.NewReader(bytesRepresentation))

			app.PauseVMIRequestHandler(request, response)

			ExpectStatusErrorWithCode(recorder, http.StatusConflict)
		},
			Entry("a not running VMI", NotRunning, UnPaused, &v1.PauseOptions{}),
			Entry("a not running VMI with dry-run option", NotRunning, UnPaused, &v1.PauseOptions{DryRun: getDryRunOption()}),

			Entry("a running but paused VMI", Running, Paused, &v1.PauseOptions{}),
			Entry("a running but paused VMI with dry-run option", Running, Paused, &v1.PauseOptions{DryRun: getDryRunOption()}),
		)

		DescribeTable("Should fail unpausing", func(running bool, paused bool, unpauseOptions *v1.UnpauseOptions) {

			expectVMI(running, paused)

			bytesRepresentation, _ := json.Marshal(unpauseOptions)
			request.Request.Body = io.NopCloser(bytes.NewReader(bytesRepresentation))

			app.UnpauseVMIRequestHandler(request, response)

			ExpectStatusErrorWithCode(recorder, http.StatusConflict)
		},
			Entry("a running, not paused VMI", Running, UnPaused, &v1.UnpauseOptions{}),
			Entry("a running, not paused VMI with dry-run option", Running, UnPaused, &v1.UnpauseOptions{DryRun: getDryRunOption()}),

			Entry("a not running VMI", NotRunning, UnPaused, &v1.UnpauseOptions{}),
			Entry("a not running VMI with dry-run option", NotRunning, UnPaused, &v1.UnpauseOptions{DryRun: getDryRunOption()}),
		)

		DescribeTable("Should unpause a running, paused VMI according to options", func(unpauseOptions *v1.UnpauseOptions) {

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
			Entry("with default", &v1.UnpauseOptions{}),
			Entry("with dry-run option", &v1.UnpauseOptions{DryRun: getDryRunOption()}),
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
					Running:  pointer.Bool(NotRunning),
					Template: &v1.VirtualMachineInstanceTemplateSpec{},
				},
			}
			vmi := v1.VirtualMachineInstance{
				Spec: v1.VirtualMachineInstanceSpec{},
			}
			vmi.ObjectMeta.SetUID(uuid.NewUUID())

			vmClient.EXPECT().Get(context.Background(), vm.Name, &k8smetav1.GetOptions{}).Return(&vm, nil)
			vmiClient.EXPECT().Get(context.Background(), vm.Name, &k8smetav1.GetOptions{}).Return(&vmi, nil)
			vmClient.EXPECT().PatchStatus(context.Background(), vm.Name, types.JSONPatchType, gomock.Any(), gomock.Any()).DoAndReturn(
				func(ctx context.Context, name string, patchType types.PatchType, body interface{}, opts *k8smetav1.PatchOptions) (interface{}, interface{}) {
					//check that dryRun option has been propagated to patch request
					Expect(opts.DryRun).To(BeEquivalentTo(startOptions.DryRun))
					return &vm, nil
				})

			app.StartVMRequestHandler(request, response)

			Expect(response.Error()).ToNot(HaveOccurred())
			Expect(response.StatusCode()).To(Equal(http.StatusAccepted))
		},
			Entry("with default", &v1.StartOptions{Paused: Paused}),
			Entry("with dry-run option", &v1.StartOptions{Paused: Paused, DryRun: getDryRunOption()}),
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

				vmClient.EXPECT().Get(context.Background(), vm.Name, &k8smetav1.GetOptions{}).Return(vm, nil)
				vmiClient.EXPECT().Get(context.Background(), vm.Name, &k8smetav1.GetOptions{}).Return(vmi, nil)
				vmClient.EXPECT().PatchStatus(context.Background(), vm.Name, types.JSONPatchType, gomock.Any(), &k8smetav1.PatchOptions{}).Return(vm, nil)

				app.StartVMRequestHandler(request, response)

				Expect(response.StatusCode()).To(Equal(http.StatusAccepted))
			},
			Entry("Manual RunStrategy", v1.RunStrategyManual),
			Entry("RerunOnFailure RunStrategy", v1.RunStrategyRerunOnFailure),
		)
	})

	Context("Subresource api - AMD SEV attestation", func() {
		withSEVAttestation := func(vmi *v1.VirtualMachineInstance) {
			vmi.Spec.Domain.LaunchSecurity = &v1.LaunchSecurity{
				SEV: &v1.SEV{
					Attestation: &v1.SEVAttestation{},
				},
			}
		}

		withScheduledPhase := func(vmi *v1.VirtualMachineInstance) {
			vmi.Status.Phase = v1.Scheduled
		}

		BeforeEach(func() {
			enableFeatureGate(virtconfig.WorkloadEncryptionSEV)
		})

		It("Should allow to fetch certificates chain when VMI is running", func() {
			backend.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/v1/namespaces/default/virtualmachineinstances/testvmi/sev/fetchcertchain"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, v1.SEVPlatformInfo{}),
				),
			)
			response.SetRequestAccepts(restful.MIME_JSON)

			expectVMI(Running, UnPaused, withSEVAttestation)
			app.SEVFetchCertChainRequestHandler(request, response)
			Expect(response.Error()).ToNot(HaveOccurred())
			Expect(response.StatusCode()).To(Equal(http.StatusOK))
		})

		It("Should fail to fetch certificates chain when attestation is not requested", func() {
			expectVMI(Running, UnPaused)
			app.SEVFetchCertChainRequestHandler(request, response)
			Expect(response.Error()).To(HaveOccurred())
			Expect(response.StatusCode()).To(Equal(http.StatusInternalServerError))
		})

		It("Should fail to fetch certificates chain when VMI is not running", func() {
			expectVMI(NotRunning, UnPaused)
			app.SEVFetchCertChainRequestHandler(request, response)
			Expect(response.Error()).To(HaveOccurred())
			Expect(response.StatusCode()).To(Equal(http.StatusInternalServerError))
		})

		It("Should allow to query launch measurement when VMI is paused", func() {
			backend.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/v1/namespaces/default/virtualmachineinstances/testvmi/sev/querylaunchmeasurement"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, v1.SEVMeasurementInfo{}),
				),
			)
			response.SetRequestAccepts(restful.MIME_JSON)

			expectVMI(Running, Paused, withSEVAttestation)
			app.SEVQueryLaunchMeasurementHandler(request, response)
			Expect(response.Error()).ToNot(HaveOccurred())
			Expect(response.StatusCode()).To(Equal(http.StatusOK))
		})

		DescribeTable("Should fail to query launch measurement",
			func(running, paused bool, vmiWarpFunctions ...func(vmi *v1.VirtualMachineInstance)) {
				expectVMI(running, paused, vmiWarpFunctions...)
				app.SEVQueryLaunchMeasurementHandler(request, response)
				Expect(response.Error()).To(HaveOccurred())
				Expect(response.StatusCode()).To(Equal(http.StatusInternalServerError))
			},
			Entry("when VMI is not running", NotRunning, Paused, withSEVAttestation),
			Entry("when VMI is not paused", Running, UnPaused, withSEVAttestation),
			Entry("when attestation is not requested ", Running, Paused),
		)

		It("Should allow to setup SEV session parameters for a paused VMI", func() {
			sevSessionOptions := &v1.SEVSessionOptions{
				Session: "AAABBB",
				DHCert:  "CCCDDD",
			}
			body, err := json.Marshal(sevSessionOptions)
			Expect(err).ToNot(HaveOccurred())
			request.Request.Body = &readCloserWrapper{bytes.NewReader(body)}

			expectVMI(NotRunning, UnPaused, withSEVAttestation, withScheduledPhase)
			vmiClient.EXPECT().Patch(context.Background(), testVMIName, types.JSONPatchType, gomock.Any(), gomock.Any()).DoAndReturn(
				func(ctx context.Context, name string, patchType types.PatchType, body interface{}, opts *k8smetav1.PatchOptions, _ ...string) (interface{}, interface{}) {
					patch := []byte(`[{"op":"test","path":"/spec/domain/launchSecurity/sev","value":{"attestation":{}}},{"op":"replace","path":"/spec/domain/launchSecurity/sev","value":{"attestation":{},"session":"AAABBB","dhCert":"CCCDDD"}}]`)
					Expect(body).To(Equal(patch))
					return nil, nil
				},
			)

			app.SEVSetupSessionHandler(request, response)
			Expect(response.Error()).ToNot(HaveOccurred())
			Expect(response.StatusCode()).To(Equal(http.StatusAccepted))
		})

		It("Should allow to inject SEV launch secret into a paused VMI", func() {
			backend.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("PUT", "/v1/namespaces/default/virtualmachineinstances/testvmi/sev/injectlaunchsecret"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, ""),
				),
			)

			sevSecretOptions := &v1.SEVSecretOptions{}
			body, err := json.Marshal(sevSecretOptions)
			Expect(err).ToNot(HaveOccurred())
			request.Request.Body = &readCloserWrapper{bytes.NewReader(body)}

			expectVMI(Running, Paused, withSEVAttestation)

			app.SEVInjectLaunchSecretHandler(request, response)
			Expect(response.Error()).ToNot(HaveOccurred())
			Expect(response.StatusCode()).To(Equal(http.StatusOK))
		})
	})

	AfterEach(func() {
		backend.Close()
		disableFeatureGates()
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
