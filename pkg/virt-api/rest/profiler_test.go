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
	"io/ioutil"
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
		Expect(err).To(BeNil())
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
		table.DescribeTable("should return error when feature gate is not enabled", func(fn func(*restful.Request, *restful.Response)) {

			fn(request, response)
			Expect(recorder.Code).To(Equal(http.StatusForbidden))
		},
			table.Entry("start function", app.StartClusterProfilerHandler),
			table.Entry("stop function", app.StopClusterProfilerHandler),
			table.Entry("dump function", app.DumpClusterProfilerHandler),
		)
		table.DescribeTable("start/stop should return success when feature gate is enabled", func(fn func(*restful.Request, *restful.Response), cmd string) {

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
			table.Entry("start function", app.StartClusterProfilerHandler, "start"),
			table.Entry("stop function", app.StopClusterProfilerHandler, "stop"),
		)

		table.DescribeTable("dump should return success when feature gate is enabled", func(fn func(*restful.Request, *restful.Response), cmd string) {

			results := v1.ClusterProfilerResults{
				ComponentResults: make(map[string]v1.ProfilerResult),
			}

			b, err := json.Marshal(&v1.ClusterProfilerRequest{})
			Expect(err).To(BeNil())
			request.Request.Body = ioutil.NopCloser(bytes.NewBuffer(b))

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
			table.Entry("dump function", app.DumpClusterProfilerHandler, "dump"),
		)
	})

	AfterEach(func() {
		server.Close()
		disableFeatureGates()
		backend.Close()
	})
})
