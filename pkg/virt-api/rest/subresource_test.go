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
		vmiClient = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)

		virtClient.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()
		virtClient.EXPECT().VirtualMachineInstance(k8smetav1.NamespaceDefault).Return(vmiClient).AnyTimes()
		virtClient.EXPECT().VirtualMachineInstance("").Return(vmiClient).AnyTimes()

		backend = ghttp.NewTLSServer()
		backendAddr := strings.Split(backend.Addr(), ":")
		backendPort, err := strconv.Atoi(backendAddr[1])
		backendIP = backendAddr[0]
		Expect(err).ToNot(HaveOccurred())
		app.consoleServerPort = backendPort
		app.virtCli = virtClient
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

	AfterEach(func() {
		backend.Close()
	})
})

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
