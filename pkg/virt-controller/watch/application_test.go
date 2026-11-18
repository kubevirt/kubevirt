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

package watch

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/emicklei/go-restful/v3"
	k8sv1 "k8s.io/api/core/v1"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/controller"
	metrics "kubevirt.io/kubevirt/pkg/monitoring/metrics/virt-controller"
	"kubevirt.io/kubevirt/pkg/rest"
	"kubevirt.io/kubevirt/pkg/testutils"
)

func newValidGetRequest() *http.Request {
	request, _ := http.NewRequest("GET", "/leader", nil)
	return request
}

var _ = Describe("Application", func() {
	var app = VirtControllerApp{}

	It("Reports leader prometheus metric when onStartedLeading is called ", func() {
		originalStartControllers := startControllers
		DeferCleanup(func() { startControllers = originalStartControllers })
		startControllers = func(_ *VirtControllerApp, _ <-chan struct{}) {}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		vmiInformer, _ := testutils.NewFakeInformerFor(&v1.VirtualMachineInstance{})
		migrationInformer, _ := testutils.NewFakeInformerFor(&v1.VirtualMachineInstanceMigration{})
		resourceQuotaInformer, _ := testutils.NewFakeInformerFor(&k8sv1.ResourceQuota{})
		pvcInformer, _ := testutils.NewFakeInformerFor(&k8sv1.PersistentVolumeClaim{})
		namespaceInformer, _ := testutils.NewFakeInformerFor(&k8sv1.Namespace{})

		app.vmiInformer = vmiInformer
		app.migrationInformer = migrationInformer
		app.informerFactory = controller.NewKubeInformerFactory(nil, nil, nil, nil, "test")
		app.persistentVolumeClaimInformer = pvcInformer
		app.resourceQuotaInformer = resourceQuotaInformer
		app.namespaceInformer = namespaceInformer
		app.readyChan = make(chan bool)

		By("Invoking callback")
		go app.onStartedLeading()(ctx)

		By("Checking prometheus metric before sync")
		dto, err := metrics.GetVirtControllerMetric()
		Expect(err).ToNot(HaveOccurred())

		zero := 0.0
		Expect(dto.GetGauge().Value).To(Equal(&zero), "Leader should be reported after virt-controller is fully operational")

		// for sync
		go pvcInformer.Run(ctx.Done())
		go resourceQuotaInformer.Run(ctx.Done())
		go namespaceInformer.Run(ctx.Done())
		time.Sleep(time.Second)

		By("Checking prometheus metric")
		dto, err = metrics.GetVirtControllerMetric()
		Expect(err).ToNot(HaveOccurred())

		one := 1.0
		Expect(dto.GetGauge().Value).To(Equal(&one))

	})

	Describe("Reinitialization conditions", func() {
		DescribeTable("Re-trigger initialization", func(hasCDIAtInit bool, addCrd bool, removeCrd bool, expectReInit bool) {
			var reInitTriggered bool

			app := VirtControllerApp{}

			clusterConfig, crdInformer, _ := testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{})
			app.clusterConfig = clusterConfig
			app.reInitChan = make(chan string, 10)
			app.hasCDI = hasCDIAtInit
			app.isVirtTemplateDeploymentEnabled = clusterConfig.VirtTemplateDeploymentEnabled()
			app.isDRAEnabled = app.clusterConfig.AnyDeviceDRAGateEnabled()

			app.clusterConfig.SetConfigModifiedCallback(app.configModificationCallback)

			if addCrd {
				testutils.AddDataVolumeAPI(crdInformer)
			} else if removeCrd {
				testutils.RemoveDataVolumeAPI(crdInformer)
			}

			select {
			case <-app.reInitChan:
				reInitTriggered = true
			case <-time.After(1 * time.Second):
				reInitTriggered = false
			}

			Expect(reInitTriggered).To(Equal(expectReInit))
		},
			Entry("when CDI is introduced", false, true, false, true),
			Entry("when CDI is removed", true, false, true, true),
			Entry("not when nothing changed and cdi exists", true, true, false, false),
			Entry("not when nothing changed and does not exist", false, false, true, false),
		)
	})

	Describe("Readiness probe", func() {
		var recorder *httptest.ResponseRecorder
		var request *http.Request
		var handler http.Handler

		BeforeEach(func() {
			app.readyChan = make(chan bool, 1)

			ws := new(restful.WebService)
			ws.Produces(restful.MIME_JSON)
			handler = http.Handler(restful.NewContainer().Add(ws))
			ws.Route(ws.GET("/leader").Produces(rest.MIME_JSON).To(app.leaderProbe))

			request = newValidGetRequest()
			recorder = httptest.NewRecorder()
		})

		Context("with closed channel", func() {
			It("should return 200 and that it is the leader", func() {

				close(app.readyChan)
				request.URL, _ = url.Parse("/leader")
				handler.ServeHTTP(recorder, request)
				var x map[string]interface{}
				Expect(json.Unmarshal(recorder.Body.Bytes(), &x)).To(Succeed())
				Expect(recorder.Code).To(Equal(http.StatusOK))
				Expect(x["apiserver"].(map[string]interface{})["leader"]).To(Equal("true"))
			})
		})
		Context("with opened channel", func() {
			It("should return 200 and that it is not the leader", func() {
				request.URL, _ = url.Parse("/leader")
				handler.ServeHTTP(recorder, request)
				var x map[string]interface{}
				Expect(json.Unmarshal(recorder.Body.Bytes(), &x)).To(Succeed())
				Expect(recorder.Code).To(Equal(http.StatusOK))
				Expect(x["apiserver"].(map[string]interface{})["leader"]).To(Equal("false"))
			})
		})
	})
})
