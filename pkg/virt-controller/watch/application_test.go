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
 * Copyright 2017 Red Hat, Inc.
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

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/emicklei/go-restful/v3"
	appsv1 "k8s.io/api/apps/v1"
	k8sv1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	storagev1 "k8s.io/api/storage/v1"
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	clonev1alpha1 "kubevirt.io/api/clone/v1alpha1"
	v1 "kubevirt.io/api/core/v1"
	exportv1 "kubevirt.io/api/export/v1alpha1"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"
	migrationsv1 "kubevirt.io/api/migrations/v1alpha1"
	snapshotv1 "kubevirt.io/api/snapshot/v1alpha1"
	"kubevirt.io/client-go/kubecli"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/monitoring/virt-controller/metrics"
	"kubevirt.io/kubevirt/pkg/rest"
	"kubevirt.io/kubevirt/pkg/storage/export/export"
	"kubevirt.io/kubevirt/pkg/storage/snapshot"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
	"kubevirt.io/kubevirt/pkg/virt-controller/watch/clone"
	"kubevirt.io/kubevirt/pkg/virt-controller/watch/drain/disruptionbudget"
	"kubevirt.io/kubevirt/pkg/virt-controller/watch/drain/evacuation"
	"kubevirt.io/kubevirt/pkg/virt-controller/watch/topology"
)

func newValidGetRequest() *http.Request {
	request, _ := http.NewRequest("GET", "/leader", nil)
	return request
}

var _ = Describe("Application", func() {
	var app = VirtControllerApp{}

	It("Reports leader prometheus metric when onStartedLeading is called ", func() {
		ctrl := gomock.NewController(GinkgoT())
		virtClient := kubecli.NewMockKubevirtClient(ctrl)
		topologyUpdater := topology.NewMockNodeTopologyUpdater(ctrl)
		topologyUpdater.EXPECT().Run(gomock.Any(), gomock.Any())

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		kvInformer, _ := testutils.NewFakeInformerFor(&v1.KubeVirt{})
		vmInformer, _ := testutils.NewFakeInformerFor(&v1.VirtualMachine{})
		vmiInformer, _ := testutils.NewFakeInformerFor(&v1.VirtualMachineInstance{})
		vmSnapshotInformer, _ := testutils.NewFakeInformerFor(&snapshotv1.VirtualMachineSnapshot{})
		vmSnapshotContentInformer, _ := testutils.NewFakeInformerFor(&snapshotv1.VirtualMachineSnapshotContent{})
		migrationInformer, _ := testutils.NewFakeInformerFor(&v1.VirtualMachineInstanceMigration{})
		nodeInformer, _ := testutils.NewFakeInformerFor(&k8sv1.Node{})
		recorder := record.NewFakeRecorder(100)
		recorder.IncludeObject = true
		config, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{})

		pdbInformer, _ := testutils.NewFakeInformerFor(&policyv1.PodDisruptionBudget{})
		migrationPolicyInformer, _ := testutils.NewFakeInformerFor(&migrationsv1.MigrationPolicy{})
		podInformer, _ := testutils.NewFakeInformerFor(&k8sv1.Pod{})
		resourceQuotaInformer, _ := testutils.NewFakeInformerFor(&k8sv1.ResourceQuota{})
		pvcInformer, _ := testutils.NewFakeInformerFor(&k8sv1.PersistentVolumeClaim{})
		namespaceInformer, _ := testutils.NewFakeInformerFor(&k8sv1.Namespace{})
		crInformer, _ := testutils.NewFakeInformerFor(&appsv1.ControllerRevision{})
		dataVolumeInformer, _ := testutils.NewFakeInformerFor(&cdiv1.DataVolume{})
		dataSourceInformer, _ := testutils.NewFakeInformerFor(&cdiv1.DataSource{})
		cdiInformer, _ := testutils.NewFakeInformerFor(&cdiv1.DataVolume{})
		cdiConfigInformer, _ := testutils.NewFakeInformerFor(&cdiv1.DataVolume{})
		rsInformer, _ := testutils.NewFakeInformerFor(&v1.VirtualMachineInstanceReplicaSet{})
		storageClassInformer, _ := testutils.NewFakeInformerFor(&storagev1.StorageClass{})
		crdInformer, _ := testutils.NewFakeInformerFor(&extv1.CustomResourceDefinition{})
		vmRestoreInformer, _ := testutils.NewFakeInformerFor(&snapshotv1.VirtualMachineRestore{})
		vmExportInformer, _ := testutils.NewFakeInformerFor(&exportv1.VirtualMachineExport{})
		configMapInformer, _ := testutils.NewFakeInformerFor(&k8sv1.ConfigMap{})
		routeConfigMapInformer, _ := testutils.NewFakeInformerFor(&k8sv1.ConfigMap{})
		dvInformer, _ := testutils.NewFakeInformerFor(&cdiv1.DataVolume{})
		instancetypeMethods := testutils.NewMockInstancetypeMethods()
		exportServiceInformer, _ := testutils.NewFakeInformerFor(&k8sv1.Service{})
		cloneInformer, _ := testutils.NewFakeInformerFor(&clonev1alpha1.VirtualMachineClone{})
		secretInformer, _ := testutils.NewFakeInformerFor(&k8sv1.Secret{})
		instancetypeInformer, _ := testutils.NewFakeInformerFor(&instancetypev1beta1.VirtualMachineInstancetype{})
		clusterInstancetypeInformer, _ := testutils.NewFakeInformerFor(&instancetypev1beta1.VirtualMachineClusterInstancetype{})
		preferenceInformer, _ := testutils.NewFakeInformerFor(&instancetypev1beta1.VirtualMachinePreference{})
		clusterPreferenceInformer, _ := testutils.NewFakeInformerFor(&instancetypev1beta1.VirtualMachineClusterPreference{})
		controllerRevisionInformer, _ := testutils.NewFakeInformerFor(&appsv1.ControllerRevision{})

		var qemuGid int64 = 107

		app.vmiInformer = vmiInformer
		app.nodeTopologyUpdater = topologyUpdater
		app.informerFactory = controller.NewKubeInformerFactory(nil, nil, nil, "test")
		app.evacuationController, _ = evacuation.NewEvacuationController(vmiInformer, migrationInformer, nodeInformer, podInformer, recorder, virtClient, config)
		app.disruptionBudgetController, _ = disruptionbudget.NewDisruptionBudgetController(vmiInformer, pdbInformer, podInformer, migrationInformer, recorder, virtClient, config)
		app.nodeController, _ = NewNodeController(virtClient, nodeInformer, vmiInformer, recorder)
		app.vmiController, _ = NewVMIController(services.NewTemplateService("a", 240, "b", "c", "d", "e", "f", "g", pvcInformer.GetStore(), virtClient, config, qemuGid, "h", resourceQuotaInformer.GetStore(), namespaceInformer.GetStore()),
			vmiInformer,
			vmInformer,
			podInformer,
			pvcInformer,
			recorder,
			virtClient,
			dataVolumeInformer,
			cdiInformer,
			cdiConfigInformer,
			config,
			topology.NewTopologyHinter(&cache.FakeCustomStore{}, &cache.FakeCustomStore{}, nil),
		)
		app.rsController, _ = NewVMIReplicaSet(vmiInformer, rsInformer, recorder, virtClient, uint(10))
		app.vmController, _ = NewVMController(vmiInformer,
			vmInformer,
			dataVolumeInformer,
			dataSourceInformer,
			namespaceInformer.GetStore(),
			pvcInformer,
			crInformer,
			podInformer,
			instancetypeMethods,
			recorder,
			virtClient,
			config)
		app.migrationController, _ = NewMigrationController(services.NewTemplateService("a", 240, "b", "c", "d", "e", "f", "g", pvcInformer.GetStore(), virtClient, config, qemuGid, "h", resourceQuotaInformer.GetStore(), namespaceInformer.GetStore()),
			vmiInformer,
			podInformer,
			migrationInformer,
			nodeInformer,
			pvcInformer,
			pdbInformer,
			migrationPolicyInformer,
			resourceQuotaInformer,
			recorder,
			virtClient,
			config,
		)
		app.snapshotController = &snapshot.VMSnapshotController{
			Client:                    virtClient,
			VMSnapshotInformer:        vmSnapshotInformer,
			VMSnapshotContentInformer: vmSnapshotContentInformer,
			VMInformer:                vmInformer,
			VMIInformer:               vmiInformer,
			PodInformer:               podInformer,
			StorageClassInformer:      storageClassInformer,
			PVCInformer:               pvcInformer,
			CRDInformer:               crdInformer,
			DVInformer:                dvInformer,
			Recorder:                  recorder,
			ResyncPeriod:              60 * time.Second,
		}
		_ = app.snapshotController.Init()
		app.restoreController = &snapshot.VMRestoreController{
			Client:                    virtClient,
			VMRestoreInformer:         vmRestoreInformer,
			VMSnapshotInformer:        vmSnapshotInformer,
			VMSnapshotContentInformer: vmSnapshotContentInformer,
			VMInformer:                vmInformer,
			VMIInformer:               vmiInformer,
			PVCInformer:               pvcInformer,
			StorageClassInformer:      storageClassInformer,
			DataVolumeInformer:        dataVolumeInformer,
			Recorder:                  recorder,
		}
		_ = app.restoreController.Init()
		app.exportController = &export.VMExportController{
			Client:                      virtClient,
			TemplateService:             services.NewTemplateService("a", 240, "b", "c", "d", "e", "f", "g", pvcInformer.GetStore(), virtClient, config, qemuGid, "h", resourceQuotaInformer.GetStore(), namespaceInformer.GetStore()),
			VMExportInformer:            vmExportInformer,
			PVCInformer:                 pvcInformer,
			PodInformer:                 podInformer,
			DataVolumeInformer:          dataVolumeInformer,
			ServiceInformer:             exportServiceInformer,
			ConfigMapInformer:           configMapInformer,
			RouteConfigMapInformer:      routeConfigMapInformer,
			Recorder:                    recorder,
			SecretInformer:              secretInformer,
			VMSnapshotInformer:          vmSnapshotInformer,
			VMSnapshotContentInformer:   vmSnapshotContentInformer,
			VMInformer:                  vmInformer,
			VMIInformer:                 vmiInformer,
			CRDInformer:                 crdInformer,
			KubeVirtInformer:            kvInformer,
			InstancetypeInformer:        instancetypeInformer,
			ClusterInstancetypeInformer: clusterInstancetypeInformer,
			PreferenceInformer:          preferenceInformer,
			ClusterPreferenceInformer:   clusterPreferenceInformer,
			ControllerRevisionInformer:  controllerRevisionInformer,
		}
		_ = app.exportController.Init()
		app.persistentVolumeClaimInformer = pvcInformer
		app.nodeInformer = nodeInformer
		app.resourceQuotaInformer = resourceQuotaInformer
		app.namespaceInformer = namespaceInformer
		app.vmCloneController, _ = clone.NewVmCloneController(
			virtClient,
			cloneInformer,
			vmSnapshotInformer,
			vmRestoreInformer,
			vmInformer,
			vmSnapshotContentInformer,
			recorder,
		)

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
		go nodeInformer.Run(ctx.Done())
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
