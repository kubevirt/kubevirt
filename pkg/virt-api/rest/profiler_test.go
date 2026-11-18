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
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
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
	"github.com/onsi/gomega/types"

	k8sv1 "k8s.io/api/core/v1"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/testutils"
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

	config, _, kvStore := testutils.NewFakeClusterConfigUsingKV(kv)

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
		app.virtClient, _ = kubecli.GetKubevirtClientFromFlags(server.URL(), "")
		app.k8sClient, _ = kubecli.GetK8sClientFromFlags(server.URL(), "")
		app.handlerTLSConfiguration = &tls.Config{InsecureSkipVerify: true}
		app.clusterConfig = config
		app.profilerComponentPort = backendPort

		request = restful.NewRequest(&http.Request{})
		recorder = httptest.NewRecorder()
		response = restful.NewResponse(recorder)
	})

	enableClusterProfiler := func() {
		kvConfig := kv.DeepCopy()
		kvConfig.Spec.Configuration.DeveloperConfiguration.ClusterProfiler = true
		testutils.UpdateFakeKubeVirtClusterConfig(kvStore, kvConfig)
	}
	disableClusterProfiler := func() {
		testutils.UpdateFakeKubeVirtClusterConfig(kvStore, kv)
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

		selector, err := clusterProfilerSelector("")
		Expect(err).ToNot(HaveOccurred())

		server.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest("GET", "/api/v1/namespaces/kubevirt/pods"),
				func(_ http.ResponseWriter, req *http.Request) {
					Expect(req.URL.Query().Get("labelSelector")).To(Equal(selector.String()))
				},
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

			enableClusterProfiler()
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

			enableClusterProfiler()
			expectPodList()
			fn(request, response)
			Expect(recorder.Code).To(Equal(http.StatusOK))
		},
			Entry("dump function", app.DumpClusterProfilerHandler, "dump"),
		)
	})

	Context("dump pagination", func() {
		dumpWithRequest := func(cpRequest *v1.ClusterProfilerRequest) {
			b, err := json.Marshal(cpRequest)
			Expect(err).ToNot(HaveOccurred())
			request.Request.Body = io.NopCloser(bytes.NewBuffer(b))
		}

		expectPodListResponse := func(podList k8sv1.PodList, verify func(*http.Request)) {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/api/v1/namespaces/kubevirt/pods"),
					func(_ http.ResponseWriter, req *http.Request) {
						if verify != nil {
							verify(req)
						}
					},
					ghttp.RespondWithJSONEncoded(http.StatusOK, podList),
				),
			)
		}

		It("should list pods with the default control-plane selector when none is provided", func() {
			enableClusterProfiler()
			dumpWithRequest(&v1.ClusterProfilerRequest{})
			selector, err := clusterProfilerSelector("")
			Expect(err).ToNot(HaveOccurred())

			expectPodListResponse(k8sv1.PodList{}, func(req *http.Request) {
				Expect(req.URL.Query().Get("labelSelector")).To(Equal(selector.String()))
			})

			app.DumpClusterProfilerHandler(request, response)
			Expect(recorder.Code).To(Equal(http.StatusNoContent))
		})

		It("should intersect an explicit label selector with the default control-plane selector", func() {
			enableClusterProfiler()
			dumpWithRequest(&v1.ClusterProfilerRequest{LabelSelector: "kubevirt.io=virt-api"})
			selector, err := clusterProfilerSelector("kubevirt.io=virt-api")
			Expect(err).ToNot(HaveOccurred())

			expectPodListResponse(k8sv1.PodList{}, func(req *http.Request) {
				Expect(req.URL.Query().Get("labelSelector")).To(Equal(selector.String()))
			})

			app.DumpClusterProfilerHandler(request, response)
			Expect(recorder.Code).To(Equal(http.StatusNoContent))
		})

		It("should return continue when a page has no ready component pods", func() {
			enableClusterProfiler()
			dumpWithRequest(&v1.ClusterProfilerRequest{LabelSelector: "kubevirt.io"})
			selector, err := clusterProfilerSelector("kubevirt.io")
			Expect(err).ToNot(HaveOccurred())

			podList := k8sv1.PodList{
				ListMeta: k8smetav1.ListMeta{Continue: "next-page"},
				Items: []k8sv1.Pod{
					{
						ObjectMeta: k8smetav1.ObjectMeta{Name: "virt-api-8xxfgt"},
						Status: k8sv1.PodStatus{
							Phase: k8sv1.PodRunning,
							Conditions: []k8sv1.PodCondition{
								{Type: k8sv1.PodReady, Status: k8sv1.ConditionFalse},
							},
						},
					},
				},
			}
			expectPodListResponse(podList, func(req *http.Request) {
				Expect(req.URL.Query().Get("labelSelector")).To(Equal(selector.String()))
			})

			app.DumpClusterProfilerHandler(request, response)
			Expect(recorder.Code).To(Equal(http.StatusOK))

			var results v1.ClusterProfilerResults
			Expect(json.Unmarshal(recorder.Body.Bytes(), &results)).To(Succeed())
			Expect(results.Continue).To(Equal("next-page"))
			Expect(results.ComponentResults).To(BeEmpty())
		})
	})

	DescribeTable("clusterProfilerSelector", func(requested string, expectErr bool, expectedSelector string) {
		selector, err := clusterProfilerSelector(requested)
		if expectErr {
			Expect(err).To(HaveOccurred())
			return
		}
		Expect(err).NotTo(HaveOccurred())
		if expectedSelector == "" {
			defaultSelector, parseErr := labels.Parse(v1.DefaultClusterProfilerLabelSelector)
			Expect(parseErr).NotTo(HaveOccurred())
			expectedSelector = defaultSelector.String()
		}
		Expect(selector.String()).To(Equal(expectedSelector))
	},
		Entry("returns the default control-plane selector when requested is empty", "", false, ""),
		Entry(
			"intersects an explicit selector with the default control-plane selector",
			"kubevirt.io=virt-api",
			false,
			"kubevirt.io in (virt-api,virt-controller,virt-handler,virt-operator),kubevirt.io=virt-api",
		),
		Entry("returns an error for an invalid requested selector", "!!!not-a-selector!!!", true, ""),
		Entry(
			"returns without error for a non-control-plane selector that matches no pods",
			"kubevirt.io=virt-exportproxy",
			false,
			"kubevirt.io in (virt-api,virt-controller,virt-handler,virt-operator),kubevirt.io=virt-exportproxy",
		),
	)

	DescribeTable("podIsReady function should return", func(deletionTimestamp *k8smetav1.Time, phase k8sv1.PodPhase, isReady k8sv1.ConditionStatus, matcher types.GomegaMatcher) {
		pod := &k8sv1.Pod{
			ObjectMeta: k8smetav1.ObjectMeta{
				Name:              "test-pod",
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
		Expect(podIsReady(pod)).To(matcher)
	},
		Entry("true with running and ready", nil, k8sv1.PodRunning, k8sv1.ConditionTrue, BeTrue()),
		Entry("false with non running", nil, k8sv1.PodPending, k8sv1.ConditionTrue, BeFalse()),
		Entry("false with running but non-ready", nil, k8sv1.PodRunning, k8sv1.ConditionFalse, BeFalse()),
		Entry("false with deletionTimestamp valued", &k8smetav1.Time{Time: time.Now()}, k8sv1.PodRunning, k8sv1.ConditionTrue, BeFalse()),
	)

	DescribeTable("clusterProfilerSelector matches pod labels", func(requested string, podLabels map[string]string, expectMatch bool) {
		selector, err := clusterProfilerSelector(requested)
		Expect(err).NotTo(HaveOccurred())
		Expect(selector.Matches(labels.Set(podLabels))).To(Equal(expectMatch))
	},
		Entry("default selector matches a control-plane pod", "", map[string]string{v1.AppLabel: "virt-api"}, true),
		Entry("default selector does not match a non-control-plane label value", "", map[string]string{v1.AppLabel: "virt-exportproxy"}, false),
		Entry("intersected selector matches the requested control-plane value", "kubevirt.io=virt-handler", map[string]string{v1.AppLabel: "virt-handler"}, true),
		Entry("user cannot widen past control plane with a non-control-plane value", "kubevirt.io=virt-exportproxy", map[string]string{v1.AppLabel: "virt-exportproxy"}, false),
	)

	AfterEach(func() {
		server.Close()
		disableClusterProfiler()
		backend.Close()
	})
})
