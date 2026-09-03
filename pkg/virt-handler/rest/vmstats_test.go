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
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/emicklei/go-restful/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/api/core/v1"

	cmdv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"
	"kubevirt.io/kubevirt/pkg/monitoring/metrics/virt-handler/collector"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/stats"
)

var _ = Describe("VMStats handler", func() {
	var (
		vmiStore cache.Store
		kvStore  cache.Store
		handler  *VMStatsHandler
	)

	makeRequest := func(body string) *httptest.ResponseRecorder {
		httpReq, _ := http.NewRequest("POST", "/v1/vmstats", strings.NewReader(body))
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Accept", "application/json")

		recorder := httptest.NewRecorder()

		ws := new(restful.WebService)
		ws.Route(ws.POST("/v1/vmstats").To(handler.GetVMStats).
			Consumes(restful.MIME_JSON).Produces(restful.MIME_JSON))

		container := restful.NewContainer()
		container.Add(ws)
		container.ServeHTTP(recorder, httpReq)
		return recorder
	}

	enableFeatureGate := func() {
		testutils.UpdateFakeKubeVirtClusterConfig(kvStore, &v1.KubeVirt{
			Spec: v1.KubeVirtSpec{
				Configuration: v1.KubeVirtConfiguration{
					DeveloperConfiguration: &v1.DeveloperConfiguration{
						FeatureGates: []string{featuregate.VMStatsCollector},
					},
				},
			},
		})
	}

	disableFeatureGate := func() {
		testutils.UpdateFakeKubeVirtClusterConfig(kvStore, &v1.KubeVirt{
			Spec: v1.KubeVirtSpec{
				Configuration: v1.KubeVirtConfiguration{
					DeveloperConfiguration: &v1.DeveloperConfiguration{
						FeatureGates: []string{},
					},
				},
			},
		})
	}

	BeforeEach(func() {
		vmiStore = cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)

		kv := &v1.KubeVirt{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kubevirt",
				Namespace: "kubevirt",
			},
		}
		clusterConfig, _, store := testutils.NewFakeClusterConfigUsingKV(kv)
		kvStore = store

		handler = NewVMStatsHandler(vmiStore, clusterConfig, collector.NewConcurrentCollector(1))
	})

	It("should return 403 when feature gate is disabled", func() {
		disableFeatureGate()
		recorder := makeRequest(`{"vmis":{"default/test-vm":{"domainStats":{}}}}`)
		Expect(recorder.Code).To(Equal(http.StatusForbidden))
	})

	It("should return 400 when the body has no VMIs", func() {
		enableFeatureGate()
		recorder := makeRequest(`{"vmis":{}}`)
		Expect(recorder.Code).To(Equal(http.StatusBadRequest))
	})

	It("should return 400 when a VMI requests no stats categories", func() {
		enableFeatureGate()
		recorder := makeRequest(`{"vmis":{"default/test-vm":{}}}`)
		Expect(recorder.Code).To(Equal(http.StatusBadRequest))
	})

	It("should report VMI not found on node for keys absent from the store", func() {
		enableFeatureGate()
		recorder := makeRequest(`{"vmis":{"default/missing-vm":{"domainStats":{}}}}`)
		Expect(recorder.Code).To(Equal(http.StatusOK))
		Expect(recorder.Body.String()).To(MatchJSON(
			`{"default/missing-vm":{"error":"VMI not found on node"}}`))
	})

	It("should report stats not available for an on-node VMI with no socket", func() {
		enableFeatureGate()
		Expect(vmiStore.Add(&v1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "test-vm"},
		})).To(Succeed())

		recorder := makeRequest(`{"vmis":{"default/test-vm":{"domainStats":{}}}}`)
		Expect(recorder.Code).To(Equal(http.StatusOK))
		Expect(recorder.Body.String()).To(MatchJSON(
			`{"default/test-vm":{"error":"stats not available: VMI socket not found or busy"}}`))
	})

	It("should only report the VMI named in the body when others are in the store", func() {
		enableFeatureGate()
		Expect(vmiStore.Add(&v1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "vm1"},
		})).To(Succeed())
		Expect(vmiStore.Add(&v1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "vm2"},
		})).To(Succeed())

		recorder := makeRequest(`{"vmis":{"default/vm1":{"domainStats":{}}}}`)
		Expect(recorder.Code).To(Equal(http.StatusOK))
		Expect(recorder.Body.String()).To(MatchJSON(
			`{"default/vm1":{"error":"stats not available: VMI socket not found or busy"}}`))
	})

	It("should report per-VMI results when one requested VMI is on-node and another is not", func() {
		enableFeatureGate()
		Expect(vmiStore.Add(&v1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "on-node-vm"},
		})).To(Succeed())

		recorder := makeRequest(`{"vmis":{"default/on-node-vm":{"domainStats":{}},"default/missing-vm":{"domainStats":{}}}}`)
		Expect(recorder.Code).To(Equal(http.StatusOK))
		Expect(recorder.Body.String()).To(MatchJSON(
			`{"default/on-node-vm":{"error":"stats not available: VMI socket not found or busy"},"default/missing-vm":{"error":"VMI not found on node"}}`))
	})
})

var _ = Describe("parseVMStatsRequests", func() {
	parse := func(body string) (map[string]*cmdv1.VMStatsRequest, error) {
		httpReq, _ := http.NewRequest("POST", "/v1/vmstats", strings.NewReader(body))
		httpReq.Header.Set("Content-Type", "application/json")
		return parseVMStatsRequests(restful.NewRequest(httpReq))
	}

	It("should decode a per-VMI request map", func() {
		requests, err := parse(`{"vmis":{"default/vm1":{"domainStats":{}},"default/vm2":{"dirtyRate":{}}}}`)
		Expect(err).ToNot(HaveOccurred())
		Expect(requests).To(HaveLen(2))
		Expect(requests["default/vm1"].DomainStats).ToNot(BeNil())
		Expect(requests["default/vm1"].DirtyRate).To(BeNil())
		Expect(requests["default/vm2"].DirtyRate).ToNot(BeNil())
		Expect(requests["default/vm2"].DomainStats).To(BeNil())
	})

	It("should return an empty map when no vmis are provided", func() {
		requests, err := parse(`{"vmis":{}}`)
		Expect(err).ToNot(HaveOccurred())
		Expect(requests).To(BeEmpty())
	})

	It("should return an error for malformed JSON", func() {
		_, err := parse(`{"vmis":`)
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("VMStatsScraper", func() {
	var (
		ctrl       *gomock.Controller
		mockClient *cmdclient.MockLauncherClient
	)

	newVMI := func(namespace, name string) *v1.VirtualMachineInstance {
		return &v1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      name,
				UID:       types.UID(fmt.Sprintf("uid-%s-%s", namespace, name)),
			},
		}
	}

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockClient = cmdclient.NewMockLauncherClient(ctrl)
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	It("should collect stats using the request configured for the VMI", func() {
		expectedStats := &stats.VMStats{
			DomainStats: stats.DomainStats{
				Name: "default_test-vm",
				UUID: "test-uuid",
			},
		}

		mockClient.EXPECT().GetVMStats(gomock.Any()).DoAndReturn(
			func(req *cmdv1.VMStatsRequest) (*stats.VMStats, error) {
				Expect(req.DomainStats).ToNot(BeNil())
				Expect(req.DirtyRate).To(BeNil())
				Expect(req.GuestGetLoad).To(BeNil())
				return expectedStats, nil
			},
		)
		mockClient.EXPECT().Close()

		requests := map[string]*cmdv1.VMStatsRequest{
			"default/test-vm": {DomainStats: &cmdv1.DomainStatsRequest{}},
		}
		scraper := NewVMStatsScraper(1, func(socketFile string) (cmdclient.LauncherClient, error) {
			return mockClient, nil
		}, requests)

		scraper.Scrape("/some/socket", newVMI("default", "test-vm"))
		scraper.Complete()

		results := scraper.GetValues()
		Expect(results).To(HaveLen(1))
		Expect(results).To(HaveKey("default/test-vm"))
		Expect(results["default/test-vm"].Stats.DomainStats.Name).To(Equal("default_test-vm"))
		Expect(results["default/test-vm"].Error).To(BeEmpty())
	})

	It("should send each VMI its own request", func() {
		ctrl2 := gomock.NewController(GinkgoT())
		mockClient2 := cmdclient.NewMockLauncherClient(ctrl2)

		mockClient.EXPECT().GetVMStats(gomock.Any()).DoAndReturn(
			func(req *cmdv1.VMStatsRequest) (*stats.VMStats, error) {
				Expect(req.DomainStats).ToNot(BeNil())
				Expect(req.DirtyRate).To(BeNil())
				return &stats.VMStats{DomainStats: stats.DomainStats{Name: "default_vm1"}}, nil
			},
		)
		mockClient.EXPECT().Close()
		mockClient2.EXPECT().GetVMStats(gomock.Any()).DoAndReturn(
			func(req *cmdv1.VMStatsRequest) (*stats.VMStats, error) {
				Expect(req.DirtyRate).ToNot(BeNil())
				Expect(req.DomainStats).To(BeNil())
				return &stats.VMStats{DomainStats: stats.DomainStats{Name: "default_vm2"}}, nil
			},
		)
		mockClient2.EXPECT().Close()

		requests := map[string]*cmdv1.VMStatsRequest{
			"default/vm1": {DomainStats: &cmdv1.DomainStatsRequest{}},
			"default/vm2": {DirtyRate: &cmdv1.DirtyRateRequest{}},
		}
		callCount := 0
		scraper := NewVMStatsScraper(2, func(socketFile string) (cmdclient.LauncherClient, error) {
			callCount++
			if socketFile == "/socket/1" {
				return mockClient, nil
			}
			return mockClient2, nil
		}, requests)

		scraper.Scrape("/socket/1", newVMI("default", "vm1"))
		scraper.Scrape("/socket/2", newVMI("default", "vm2"))
		scraper.Complete()

		results := scraper.GetValues()
		Expect(results).To(HaveLen(2))
		Expect(results["default/vm1"].Error).To(BeEmpty())
		Expect(results["default/vm2"].Error).To(BeEmpty())

		ctrl2.Finish()
	})

	It("should report an error when no request is configured for the VMI", func() {
		scraper := NewVMStatsScraper(1, func(socketFile string) (cmdclient.LauncherClient, error) {
			Fail("newClient should not be called when no request is configured")
			return nil, nil
		}, map[string]*cmdv1.VMStatsRequest{})

		scraper.Scrape("/some/socket", newVMI("default", "test-vm"))
		scraper.Complete()

		results := scraper.GetValues()
		Expect(results).To(HaveLen(1))
		Expect(results["default/test-vm"].Stats).To(BeNil())
		Expect(results["default/test-vm"].Error).To(Equal("no stats request configured for VMI"))
	})

	It("should report error when gRPC call fails", func() {
		mockClient.EXPECT().GetVMStats(gomock.Any()).Return(nil, fmt.Errorf("gRPC connection failed"))
		mockClient.EXPECT().Close()

		requests := map[string]*cmdv1.VMStatsRequest{
			"default/test-vm": {DomainStats: &cmdv1.DomainStatsRequest{}},
		}
		scraper := NewVMStatsScraper(1, func(socketFile string) (cmdclient.LauncherClient, error) {
			return mockClient, nil
		}, requests)

		scraper.Scrape("/some/socket", newVMI("default", "test-vm"))
		scraper.Complete()

		results := scraper.GetValues()
		Expect(results).To(HaveLen(1))
		Expect(results["default/test-vm"].Stats).To(BeNil())
		Expect(results["default/test-vm"].Error).To(Equal("gRPC connection failed"))
	})

	It("should report error when client creation fails", func() {
		requests := map[string]*cmdv1.VMStatsRequest{
			"default/test-vm": {DomainStats: &cmdv1.DomainStatsRequest{}},
		}
		scraper := NewVMStatsScraper(1, func(socketFile string) (cmdclient.LauncherClient, error) {
			return nil, fmt.Errorf("socket not found")
		}, requests)

		scraper.Scrape("/some/socket", newVMI("default", "test-vm"))
		scraper.Complete()

		results := scraper.GetValues()
		Expect(results).To(HaveLen(1))
		Expect(results["default/test-vm"].Stats).To(BeNil())
		Expect(results["default/test-vm"].Error).To(Equal("socket not found"))
	})
})
