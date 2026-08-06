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

package backup

import (
	"context"
	"crypto/tls"
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

var _ = Describe("VirtualMachineInstance backup", func() {
	var (
		backend    *ghttp.Server
		backendIP  string
		kubeClient *fake.Clientset
		vmiClient  *kubecli.MockVirtualMachineInstanceInterface
		handler    *Handler
	)

	BeforeEach(func() {
		ctrl := gomock.NewController(GinkgoT())
		virtClient := kubecli.NewMockKubevirtClient(ctrl)
		kubeClient = fake.NewSimpleClientset()
		vmiClient = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)

		virtClient.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()
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

	expectVMI := func(vmi *v1.VirtualMachineInstance) {
		vmiClient.EXPECT().Get(context.Background(), testVMIName, metav1.GetOptions{}).Return(vmi, nil)
		expectHandlerPod()
	}

	expectStatusError := func(statusErr *errors.StatusError, code int, message string) {
		Expect(statusErr).ToNot(BeNil())
		Expect(statusErr.Status().Code).To(BeNumerically("==", code))
		Expect(statusErr.Error()).To(ContainSubstring(message))
	}

	Context("Backup", func() {
		It("backs up a running VMI", func() {
			backend.AppendHandlers(ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodPut, "/v1/namespaces/default/virtualmachineinstances/testvmi/backup"),
				ghttp.RespondWith(http.StatusOK, ""),
			))
			expectVMI(&v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{Name: testVMIName, Namespace: metav1.NamespaceDefault},
				Status:     v1.VirtualMachineInstanceStatus{Phase: v1.Running},
			})

			Expect(handler.BackupVMI(context.Background(), metav1.NamespaceDefault, testVMIName, nil)).To(BeNil())
		})

		It("fails when the VMI is not running", func() {
			expectVMI(&v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{Name: testVMIName, Namespace: metav1.NamespaceDefault},
				Status:     v1.VirtualMachineInstanceStatus{Phase: v1.Failed},
			})

			expectStatusError(
				handler.BackupVMI(context.Background(), metav1.NamespaceDefault, testVMIName, nil),
				http.StatusConflict,
				"VM is not running",
			)
		})
	})

	Context("RedefineCheckpoint", func() {
		It("redefines a checkpoint when ChangedBlockTracking is enabled", func() {
			backend.AppendHandlers(ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodPut, "/v1/namespaces/default/virtualmachineinstances/testvmi/redefine-checkpoint"),
				ghttp.RespondWith(http.StatusOK, ""),
			))
			expectVMI(&v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{Name: testVMIName, Namespace: metav1.NamespaceDefault},
				Status: v1.VirtualMachineInstanceStatus{
					Phase: v1.Running,
					ChangedBlockTracking: &v1.ChangedBlockTrackingStatus{
						State: v1.ChangedBlockTrackingEnabled,
					},
				},
			})

			Expect(handler.RedefineCheckpointVMI(context.Background(), metav1.NamespaceDefault, testVMIName, nil)).To(BeNil())
		})

		It("fails when ChangedBlockTracking is not enabled", func() {
			expectVMI(&v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{Name: testVMIName, Namespace: metav1.NamespaceDefault},
				Status:     v1.VirtualMachineInstanceStatus{Phase: v1.Running},
			})

			expectStatusError(
				handler.RedefineCheckpointVMI(context.Background(), metav1.NamespaceDefault, testVMIName, nil),
				http.StatusConflict,
				"ChangedBlockTracking is not enabled",
			)
		})
	})
})
