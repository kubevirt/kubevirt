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
	"time"

	"github.com/emicklei/go-restful/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"github.com/onsi/gomega/types"

	"kubevirt.io/kubevirt/pkg/util/status"

	k8sv1 "k8s.io/api/core/v1"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/testutils"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

var _ = Describe("Cluster Profiler Subresources", func() {
	kubecli.Init()

	var server *ghttp.Server
	var request *restful.Request
	var recorder *httptest.ResponseRecorder
	var response *restful.Response
	var backend *ghttp.Server
	var backendIP string

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
		backend = ghttp.NewTLSServer()
		backendAddr := strings.Split(backend.Addr(), ":")
		backendPort, err := strconv.Atoi(backendAddr[1])
		Expect(err).ToNot(HaveOccurred())
		backendIP = backendAddr[0]
		server = ghttp.NewServer()
		flag.Set("kubeconfig", "")
		flag.Set("master", server.URL())
		app.virtCli, _ = kubecli.GetKubevirtClientFromFlags(server.URL(), "")
		app.statusUpdater = status.NewVMStatusUpdater(app.virtCli)
		app.credentialsLock = &sync.Mutex{}
		app.handlerTLSConfiguration = &tls.Config{InsecureSkipVerify: true}
		app.clusterConfig = config
		app.profilerComponentPort = backendPort

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

	expectPodList := func() {
		pod := &k8sv1.Pod{}
		pod.Labels = map[string]string{}
		pod.Labels[v1.AppLabel] = "virt-handler"
		pod.ObjectMeta.Name = "virt-handler-123"

		pod.Spec.NodeName = "mynode"
		pod.Status.Phase = k8sv1.PodRunning
		pod.Status.PodIP = backendIP

		pod.Status.Conditions = append(pod.Status.Conditions, k8sv1.PodCondition{
			Type:   k8sv1.PodReady,
			Status: k8sv1.ConditionTrue,
		})

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

	Context("handler functions", func() {
		DescribeTable("should return error when feature gate is not enabled", func(fn func(*restful.Request, *restful.Response)) {

			fn(request, response)
			Expect(recorder.Code).To(Equal(http.StatusForbidden))
		},
			Entry("start function", app.StartClusterProfilerHandler),
			Entry("stop function", app.StopClusterProfilerHandler),
			Entry("dump function", app.DumpClusterProfilerHandler),
		)
		DescribeTable("start/stop should return success when feature gate is enabled", func(fn func(*restful.Request, *restful.Response), cmd string) {

			results := v1.ClusterProfilerResults{
				ComponentResults: make(map[string]v1.ProfilerResult),
			}

			backend.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", fmt.Sprintf("/%s-profiler", cmd)),
					ghttp.RespondWithJSONEncoded(http.StatusOK, results),
				),
			)

			enableFeatureGate(virtconfig.ClusterProfiler)
			expectPodList()
			fn(request, response)
			Expect(recorder.Code).To(Equal(http.StatusOK))
		},
			Entry("start function", app.StartClusterProfilerHandler, "start"),
			Entry("stop function", app.StopClusterProfilerHandler, "stop"),
		)

		DescribeTable("dump should return success when feature gate is enabled", func(fn func(*restful.Request, *restful.Response), cmd string) {

			results := v1.ClusterProfilerResults{
				ComponentResults: make(map[string]v1.ProfilerResult),
			}

			b, err := json.Marshal(&v1.ClusterProfilerRequest{})
			Expect(err).ToNot(HaveOccurred())
			request.Request.Body = io.NopCloser(bytes.NewBuffer(b))

			backend.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", fmt.Sprintf("/%s-profiler", cmd)),
					ghttp.RespondWithJSONEncoded(http.StatusOK, results),
				),
			)

			enableFeatureGate(virtconfig.ClusterProfiler)
			expectPodList()
			fn(request, response)
			Expect(recorder.Code).To(Equal(http.StatusOK))
		},
			Entry("dump function", app.DumpClusterProfilerHandler, "dump"),
		)
	})

	DescribeTable(", podIsReadyComponent function should return", func(name string, deletionTimestamp *k8smetav1.Time, phase k8sv1.PodPhase, isReady k8sv1.ConditionStatus, matcher types.GomegaMatcher) {
		pod := &k8sv1.Pod{
			ObjectMeta: k8smetav1.ObjectMeta{
				Name:              name,
				DeletionTimestamp: deletionTimestamp,
			},
			Status: k8sv1.PodStatus{
				Phase: phase,
				Conditions: []k8sv1.PodCondition{
					{
						Type:   k8sv1.PodReady,
						Status: isReady,
					},
				},
			},
		}
		isReadyComponentPod := podIsReadyComponent(pod)
		Expect(isReadyComponentPod).To(matcher)
	},
		Entry("true with running and ready virt-handler", "virt-handler-8xxfgt", nil, k8sv1.PodRunning, k8sv1.ConditionTrue, BeTrue()),
		Entry("true with running and ready virt-controller", "virt-controller-8xxfgt", nil, k8sv1.PodRunning, k8sv1.ConditionTrue, BeTrue()),
		Entry("true with running and ready virt-operator", "virt-operator-8xxfgt", nil, k8sv1.PodRunning, k8sv1.ConditionTrue, BeTrue()),
		Entry("true with running and ready virt-api", "virt-api-8xxfgt", nil, k8sv1.PodRunning, k8sv1.ConditionTrue, BeTrue()),
		Entry("false with non running virt-handler", "virt-handler-8xxfgt", nil, k8sv1.PodPending, k8sv1.ConditionTrue, BeFalse()),
		Entry("false with non running virt-controller", "virt-controller-8xxfgt", nil, k8sv1.PodPending, k8sv1.ConditionTrue, BeFalse()),
		Entry("false with non running virt-operator", "virt-operator-8xxfgt", nil, k8sv1.PodPending, k8sv1.ConditionTrue, BeFalse()),
		Entry("false with non running virt-api", "virt-api-8xxfgt", nil, k8sv1.PodPending, k8sv1.ConditionTrue, BeFalse()),
		Entry("false with running but non-ready virt-handler", "virt-handler-8xxfgt", nil, k8sv1.PodRunning, k8sv1.ConditionFalse, BeFalse()),
		Entry("false with running but non-ready virt-controller", "virt-controller-8xxfgt", nil, k8sv1.PodRunning, k8sv1.ConditionFalse, BeFalse()),
		Entry("false with running but non-ready virt-operator", "virt-operator-8xxfgt", nil, k8sv1.PodRunning, k8sv1.ConditionFalse, BeFalse()),
		Entry("false with running but non-ready virt-api", "virt-api-8xxfgt", nil, k8sv1.PodRunning, k8sv1.ConditionFalse, BeFalse()),
		Entry("false with deletionTimestamp valued", "virt-handler-8xxfgt", &k8smetav1.Time{Time: time.Now()}, k8sv1.PodRunning, k8sv1.ConditionTrue, BeFalse()),
		Entry("false with other component", "kubevirt-apiproxy-8xxfgt", nil, k8sv1.PodRunning, k8sv1.ConditionTrue, BeFalse()),
	)

	AfterEach(func() {
		server.Close()
		disableFeatureGates()
		backend.Close()
	})
})
