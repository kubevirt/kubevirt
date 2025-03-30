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
 * Copyright The KubeVirt Authors
 *
 */

package rest

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"

	"github.com/emicklei/go-restful/v3"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"github.com/onsi/gomega/gstruct"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	kubevirtfake "kubevirt.io/client-go/kubevirt/fake"

	"kubevirt.io/kubevirt/pkg/libvmi"
	libvmistatus "kubevirt.io/kubevirt/pkg/libvmi/status"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
)

var _ = Describe("SEV Subresources", func() {
	const nodeName = "mynode"

	var (
		backend    *ghttp.Server
		request    *restful.Request
		response   *restful.Response
		virtClient *kubevirtfake.Clientset
		app        *SubresourceAPIApp

		kv = &v1.KubeVirt{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kubevirt",
				Namespace: "kubevirt",
			},
			Spec: v1.KubeVirtSpec{
				Configuration: v1.KubeVirtConfiguration{
					DeveloperConfiguration: &v1.DeveloperConfiguration{
						FeatureGates: []string{featuregate.WorkloadEncryptionSEV},
					},
				},
			},
			Status: v1.KubeVirtStatus{
				Phase: v1.KubeVirtPhaseDeploying,
			},
		}
	)

	config, _, _ := testutils.NewFakeClusterConfigUsingKV(kv)

	BeforeEach(func() {
		request = restful.NewRequest(&http.Request{})
		request.PathParameters()["name"] = testVMIName
		request.PathParameters()["namespace"] = metav1.NamespaceDefault
		recorder := httptest.NewRecorder()
		response = restful.NewResponse(recorder)

		backend = ghttp.NewTLSServer()
		backendAddr := strings.Split(backend.Addr(), ":")
		backendPort, err := strconv.Atoi(backendAddr[1])
		Expect(err).ToNot(HaveOccurred())
		backendIP := backendAddr[0]

		pod := &k8sv1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "madeup-name",
				Namespace: "kubevirt",
				Labels:    map[string]string{v1.AppLabel: "virt-handler"},
			},
			Spec: k8sv1.PodSpec{
				NodeName: nodeName,
			},
			Status: k8sv1.PodStatus{
				Phase: k8sv1.PodRunning,
				PodIP: backendIP,
			},
		}

		kubeClient := fake.NewSimpleClientset(pod)
		mockVirtClient := kubecli.NewMockKubevirtClient(gomock.NewController(GinkgoT()))
		virtClient = kubevirtfake.NewSimpleClientset()

		mockVirtClient.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()
		mockVirtClient.EXPECT().VirtualMachine(metav1.NamespaceDefault).Return(virtClient.KubevirtV1().VirtualMachines(metav1.NamespaceDefault)).AnyTimes()
		mockVirtClient.EXPECT().VirtualMachine("").Return(virtClient.KubevirtV1().VirtualMachines("")).AnyTimes()
		mockVirtClient.EXPECT().VirtualMachineInstance(metav1.NamespaceDefault).Return(virtClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault)).AnyTimes()
		mockVirtClient.EXPECT().VirtualMachineInstance("").Return(virtClient.KubevirtV1().VirtualMachineInstances("")).AnyTimes()
		mockVirtClient.EXPECT().VirtualMachineInstanceMigration(metav1.NamespaceDefault).Return(virtClient.KubevirtV1().VirtualMachineInstanceMigrations(metav1.NamespaceDefault)).AnyTimes()

		app = NewSubresourceAPIApp(mockVirtClient, backendPort, &tls.Config{InsecureSkipVerify: true}, config)
	})

	AfterEach(func() {
		backend.Close()
	})

	createVMI := func(running, paused bool, specOpts []libvmi.Option, statusOpts []libvmistatus.Option) {
		phase := v1.Running
		if !running {
			phase = v1.Failed
		}
		status := []libvmistatus.Option{
			libvmistatus.WithPhase(phase),
			libvmistatus.WithNodeName(nodeName),
		}
		if paused {
			status = append(status, libvmistatus.WithCondition(v1.VirtualMachineInstanceCondition{
				Type:   v1.VirtualMachineInstancePaused,
				Status: k8sv1.ConditionTrue,
			}))
		}

		status = append(status, statusOpts...)

		fullOpts := append([]libvmi.Option{
			libvmi.WithName(testVMIName),
			libvmi.WithNamespace(metav1.NamespaceDefault),
			libvmistatus.WithStatus(libvmistatus.New(status...)),
		},
			specOpts...,
		)
		vmi := libvmi.New(fullOpts...)

		_, err := virtClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault).Create(context.TODO(), vmi, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
	}

	It("Should allow to fetch certificates chain when VMI is running", func() {
		createVMI(Running, UnPaused, []libvmi.Option{libvmi.WithSEVAttestation()}, nil)
		backend.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest("GET", "/v1/namespaces/default/virtualmachineinstances/testvmi/sev/fetchcertchain"),
				ghttp.RespondWithJSONEncoded(http.StatusOK, v1.SEVPlatformInfo{}),
			),
		)
		response.SetRequestAccepts(restful.MIME_JSON)

		app.SEVFetchCertChainRequestHandler(request, response)
		Expect(response.Error()).ToNot(HaveOccurred())
		Expect(response.StatusCode()).To(Equal(http.StatusOK))
	})

	It("Should fail to fetch certificates chain when attestation is not requested", func() {
		createVMI(Running, UnPaused, nil, nil)
		app.SEVFetchCertChainRequestHandler(request, response)
		Expect(response.Error()).To(HaveOccurred())
		Expect(response.StatusCode()).To(Equal(http.StatusInternalServerError))
		Expect(response.Error().Error()).To(ContainSubstring("Attestation not requested for VMI"))
	})

	It("Should fail to fetch certificates chain when VMI is not running", func() {
		createVMI(NotRunning, UnPaused, nil, nil)
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

		createVMI(Running, Paused, []libvmi.Option{libvmi.WithSEVAttestation()}, nil)
		app.SEVQueryLaunchMeasurementHandler(request, response)
		Expect(response.Error()).ToNot(HaveOccurred())
		Expect(response.StatusCode()).To(Equal(http.StatusOK))
	})

	DescribeTable("Should fail to query launch measurement",
		func(running, paused bool, option ...libvmi.Option) {
			createVMI(running, paused, option, nil)
			app.SEVQueryLaunchMeasurementHandler(request, response)
			Expect(response.Error()).To(HaveOccurred())
			Expect(response.StatusCode()).To(Equal(http.StatusInternalServerError))
		},
		Entry("when VMI is not running", NotRunning, Paused, libvmi.WithSEVAttestation()),
		Entry("when VMI is not paused", Running, UnPaused, libvmi.WithSEVAttestation()),
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

		createVMI(NotRunning, UnPaused, []libvmi.Option{libvmi.WithSEVAttestation()}, []libvmistatus.Option{libvmistatus.WithPhase(v1.Scheduled)})

		app.SEVSetupSessionHandler(request, response)
		Expect(response.Error()).ToNot(HaveOccurred())
		Expect(response.StatusCode()).To(Equal(http.StatusAccepted))
		updatedVMI, err := virtClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault).Get(context.TODO(), testVMIName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(updatedVMI.Spec.Domain.LaunchSecurity.SEV).To(gstruct.PointTo(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
			"Session": Equal("AAABBB"),
			"DHCert":  Equal("CCCDDD"),
		})))
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

		createVMI(Running, Paused, []libvmi.Option{libvmi.WithSEVAttestation()}, nil)

		app.SEVInjectLaunchSecretHandler(request, response)
		Expect(response.Error()).ToNot(HaveOccurred())
		Expect(response.StatusCode()).To(Equal(http.StatusOK))
	})
})
