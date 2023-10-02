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
 * Copyright 2022 Red Hat, Inc.
 *
 */
package export

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"
	"path/filepath"
	"time"

	storagev1 "k8s.io/api/storage/v1"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	vsv1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	routev1 "github.com/openshift/api/route/v1"
	appsv1 "k8s.io/api/apps/v1"
	k8sv1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	virtv1 "kubevirt.io/api/core/v1"
	exportv1 "kubevirt.io/api/export/v1beta1"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"
	snapshotv1 "kubevirt.io/api/snapshot/v1beta1"
	kubevirtfake "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/fake"
	"kubevirt.io/client-go/kubecli"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/certificates/bootstrap"
	virtcontroller "kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
)

const (
	testPVCName = "test-pvc"
)

var _ = Describe("PVC source", func() {
	var (
		ctrl                        *gomock.Controller
		controller                  *VMExportController
		recorder                    *record.FakeRecorder
		pvcInformer                 cache.SharedIndexInformer
		podInformer                 cache.SharedIndexInformer
		cmInformer                  cache.SharedIndexInformer
		vmExportInformer            cache.SharedIndexInformer
		serviceInformer             cache.SharedIndexInformer
		dvInformer                  cache.SharedIndexInformer
		vmSnapshotInformer          cache.SharedIndexInformer
		vmSnapshotContentInformer   cache.SharedIndexInformer
		secretInformer              cache.SharedIndexInformer
		vmInformer                  cache.SharedIndexInformer
		vmiInformer                 cache.SharedIndexInformer
		kvInformer                  cache.SharedIndexInformer
		crdInformer                 cache.SharedIndexInformer
		instancetypeInformer        cache.SharedIndexInformer
		clusterInstancetypeInformer cache.SharedIndexInformer
		preferenceInformer          cache.SharedIndexInformer
		clusterPreferenceInformer   cache.SharedIndexInformer
		controllerRevisionInformer  cache.SharedIndexInformer
		rqInformer                  cache.SharedIndexInformer
		nsInformer                  cache.SharedIndexInformer
		storageClassInformer        cache.SharedIndexInformer
		storageProfileInformer      cache.SharedIndexInformer
		k8sClient                   *k8sfake.Clientset
		vmExportClient              *kubevirtfake.Clientset
		fakeVolumeSnapshotProvider  *MockVolumeSnapshotProvider
		mockVMExportQueue           *testutils.MockWorkQueue
		routeCache                  cache.Store
		ingressCache                cache.Store
		certDir                     string
		certFilePath                string
		keyFilePath                 string
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		var err error
		certDir, err = os.MkdirTemp("", "certs")
		Expect(err).ToNot(HaveOccurred())
		certFilePath = filepath.Join(certDir, "tls.crt")
		keyFilePath = filepath.Join(certDir, "tls.key")
		writeCertsToDir(certDir)
		virtClient := kubecli.NewMockKubevirtClient(ctrl)
		pvcInformer, _ = testutils.NewFakeInformerFor(&k8sv1.PersistentVolumeClaim{})
		podInformer, _ = testutils.NewFakeInformerFor(&k8sv1.Pod{})
		cmInformer, _ = testutils.NewFakeInformerFor(&k8sv1.ConfigMap{})
		serviceInformer, _ = testutils.NewFakeInformerFor(&k8sv1.Service{})
		vmExportInformer, _ = testutils.NewFakeInformerWithIndexersFor(&exportv1.VirtualMachineExport{}, virtcontroller.GetVirtualMachineExportInformerIndexers())
		dvInformer, _ = testutils.NewFakeInformerFor(&cdiv1.DataVolume{})
		vmSnapshotInformer, _ = testutils.NewFakeInformerFor(&snapshotv1.VirtualMachineSnapshot{})
		vmSnapshotContentInformer, _ = testutils.NewFakeInformerFor(&snapshotv1.VirtualMachineSnapshotContent{})
		vmInformer, _ = testutils.NewFakeInformerFor(&virtv1.VirtualMachine{})
		vmiInformer, _ = testutils.NewFakeInformerFor(&virtv1.VirtualMachineInstance{})
		routeInformer, _ := testutils.NewFakeInformerFor(&routev1.Route{})
		routeCache = routeInformer.GetStore()
		ingressInformer, _ := testutils.NewFakeInformerFor(&networkingv1.Ingress{})
		ingressCache = ingressInformer.GetStore()
		secretInformer, _ = testutils.NewFakeInformerFor(&k8sv1.Secret{})
		kvInformer, _ = testutils.NewFakeInformerFor(&virtv1.KubeVirt{})
		crdInformer, _ = testutils.NewFakeInformerFor(&extv1.CustomResourceDefinition{})
		instancetypeInformer, _ = testutils.NewFakeInformerFor(&instancetypev1beta1.VirtualMachineInstancetype{})
		clusterInstancetypeInformer, _ = testutils.NewFakeInformerFor(&instancetypev1beta1.VirtualMachineClusterInstancetype{})
		preferenceInformer, _ = testutils.NewFakeInformerFor(&instancetypev1beta1.VirtualMachinePreference{})
		clusterPreferenceInformer, _ = testutils.NewFakeInformerFor(&instancetypev1beta1.VirtualMachineClusterPreference{})
		controllerRevisionInformer, _ = testutils.NewFakeInformerFor(&appsv1.ControllerRevision{})
		rqInformer, _ = testutils.NewFakeInformerFor(&k8sv1.ResourceQuota{})
		nsInformer, _ = testutils.NewFakeInformerFor(&k8sv1.Namespace{})
		storageClassInformer, _ = testutils.NewFakeInformerFor(&storagev1.StorageClass{})
		storageProfileInformer, _ = testutils.NewFakeInformerFor(&cdiv1.StorageProfile{})
		fakeVolumeSnapshotProvider = &MockVolumeSnapshotProvider{
			volumeSnapshots: []*vsv1.VolumeSnapshot{},
		}

		config, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&virtv1.KubeVirtConfiguration{})
		k8sClient = k8sfake.NewSimpleClientset()
		vmExportClient = kubevirtfake.NewSimpleClientset()
		recorder = record.NewFakeRecorder(100)

		virtClient.EXPECT().CoreV1().Return(k8sClient.CoreV1()).AnyTimes()
		virtClient.EXPECT().VirtualMachineExport(testNamespace).
			Return(vmExportClient.ExportV1beta1().VirtualMachineExports(testNamespace)).AnyTimes()

		controller = &VMExportController{
			Client:                      virtClient,
			Recorder:                    recorder,
			PVCInformer:                 pvcInformer,
			PodInformer:                 podInformer,
			ConfigMapInformer:           cmInformer,
			VMExportInformer:            vmExportInformer,
			ServiceInformer:             serviceInformer,
			DataVolumeInformer:          dvInformer,
			KubevirtNamespace:           "kubevirt",
			TemplateService:             services.NewTemplateService("a", 240, "b", "c", "d", "e", "f", "g", pvcInformer.GetStore(), virtClient, config, qemuGid, "h", rqInformer.GetStore(), nsInformer.GetStore(), storageClassInformer.GetStore(), pvcInformer.GetIndexer(), storageProfileInformer.GetStore()),
			caCertManager:               bootstrap.NewFileCertificateManager(certFilePath, keyFilePath),
			RouteCache:                  routeCache,
			IngressCache:                ingressCache,
			RouteConfigMapInformer:      cmInformer,
			SecretInformer:              secretInformer,
			VMSnapshotInformer:          vmSnapshotInformer,
			VMSnapshotContentInformer:   vmSnapshotContentInformer,
			VolumeSnapshotProvider:      fakeVolumeSnapshotProvider,
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
		initCert = func(ctrl *VMExportController) {
			go controller.caCertManager.Start()
			// Give the thread time to read the certs.
			Eventually(func() *tls.Certificate {
				return controller.caCertManager.Current()
			}, time.Second, time.Millisecond).ShouldNot(BeNil())
		}

		controller.Init()
		mockVMExportQueue = testutils.NewMockWorkQueue(controller.vmExportQueue)
		controller.vmExportQueue = mockVMExportQueue

		Expect(
			cmInformer.GetStore().Add(&k8sv1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: controller.KubevirtNamespace,
					Name:      components.KubeVirtExportCASecretName,
				},
				Data: map[string]string{
					"ca-bundle": "replace me with ca cert",
				},
			}),
		).To(Succeed())

		Expect(
			kvInformer.GetStore().Add(&virtv1.KubeVirt{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: controller.KubevirtNamespace,
					Name:      "kv",
				},
				Spec: virtv1.KubeVirtSpec{
					CertificateRotationStrategy: virtv1.KubeVirtCertificateRotateStrategy{
						SelfSigned: &virtv1.KubeVirtSelfSignConfiguration{
							CA: &virtv1.CertConfig{
								Duration:    &metav1.Duration{Duration: 24 * time.Hour},
								RenewBefore: &metav1.Duration{Duration: 3 * time.Hour},
							},
							Server: &virtv1.CertConfig{
								Duration:    &metav1.Duration{Duration: 2 * time.Hour},
								RenewBefore: &metav1.Duration{Duration: 1 * time.Hour},
							},
						},
					},
				},
				Status: virtv1.KubeVirtStatus{
					Phase: virtv1.KubeVirtPhaseDeployed,
				},
			}),
		).To(Succeed())
	})

	AfterEach(func() {
		controller.caCertManager.Stop()
		os.RemoveAll(certDir)
	})

	It("Should properly update VMExport status with a valid token and no pvc", func() {
		testVMExport := createPVCVMExport()
		expectExporterCreate(k8sClient, k8sv1.PodRunning)
		vmExportClient.Fake.PrependReactor("update", "virtualmachineexports", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			update, ok := action.(testing.UpdateAction)
			Expect(ok).To(BeTrue())
			vmExport, ok := update.GetObject().(*exportv1.VirtualMachineExport)
			Expect(ok).To(BeTrue())
			verifyLinksEmpty(vmExport)
			return true, vmExport, nil
		})

		retry, err := controller.updateVMExport(testVMExport)
		Expect(err).ToNot(HaveOccurred())
		Expect(retry).To(BeEquivalentTo(0))
		service, err := k8sClient.CoreV1().Services(testNamespace).Get(context.Background(), fmt.Sprintf("%s-%s", exportPrefix, testVMExport.Name), metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(service.Name).To(Equal(fmt.Sprintf("%s-%s", exportPrefix, testVMExport.Name)))
	})

	It("Should properly update VMExport status with a valid token and archive pvc no route", func() {
		testVMExport := createPVCVMExport()
		pvcInformer.GetStore().Add(createPVC(testPVCName, "archive"))
		expectExporterCreate(k8sClient, k8sv1.PodRunning)
		vmExportClient.Fake.PrependReactor("update", "virtualmachineexports", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			update, ok := action.(testing.UpdateAction)
			Expect(ok).To(BeTrue())
			vmExport, ok := update.GetObject().(*exportv1.VirtualMachineExport)
			Expect(ok).To(BeTrue())
			Expect(vmExport.Status).ToNot(BeNil())
			Expect(vmExport.Status.Links).ToNot(BeNil())
			Expect(vmExport.Status.Links.External).To(BeNil())
			verifyArchiveInternal(vmExport, vmExport.Name, testNamespace, testVMExport.Spec.Source.Name)
			return true, vmExport, nil
		})
		retry, err := controller.updateVMExport(testVMExport)
		Expect(err).ToNot(HaveOccurred())
		Expect(retry).To(BeEquivalentTo(0))
		service, err := k8sClient.CoreV1().Services(testNamespace).Get(context.Background(), fmt.Sprintf("%s-%s", exportPrefix, testVMExport.Name), metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(service.Name).To(Equal(fmt.Sprintf("%s-%s", exportPrefix, testVMExport.Name)))
	})

	It("Should properly update VMExport status with a valid token and kubevirt pvc with route", func() {
		testVMExport := createPVCVMExport()
		pvcInformer.GetStore().Add(createPVC(testPVCName, "kubevirt"))
		expectExporterCreate(k8sClient, k8sv1.PodRunning)
		controller.RouteCache.Add(routeToHostAndService(components.VirtExportProxyServiceName))

		vmExportClient.Fake.PrependReactor("update", "virtualmachineexports", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			update, ok := action.(testing.UpdateAction)
			Expect(ok).To(BeTrue())
			vmExport, ok := update.GetObject().(*exportv1.VirtualMachineExport)
			Expect(ok).To(BeTrue())
			verifyKubevirtInternal(vmExport, vmExport.Name, testNamespace, testVMExport.Spec.Source.Name)
			verifyKubevirtExternal(vmExport, vmExport.Name, testNamespace, testVMExport.Spec.Source.Name)
			return true, vmExport, nil
		})
		retry, err := controller.updateVMExport(testVMExport)
		Expect(err).ToNot(HaveOccurred())
		Expect(retry).To(BeEquivalentTo(0))
		service, err := k8sClient.CoreV1().Services(testNamespace).Get(context.Background(), fmt.Sprintf("%s-%s", exportPrefix, testVMExport.Name), metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(service.Name).To(Equal(fmt.Sprintf("%s-%s", exportPrefix, testVMExport.Name)))
	})

	It("Should properly update VMExport status with a valid token and no pvc, pending pod", func() {
		testVMExport := createPVCVMExport()
		expectExporterCreate(k8sClient, k8sv1.PodPending)
		vmExportClient.Fake.PrependReactor("update", "virtualmachineexports", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			update, ok := action.(testing.UpdateAction)
			Expect(ok).To(BeTrue())
			vmExport, ok := update.GetObject().(*exportv1.VirtualMachineExport)
			Expect(ok).To(BeTrue())
			verifyLinksEmpty(vmExport)
			return true, vmExport, nil
		})
		retry, err := controller.updateVMExport(testVMExport)
		Expect(err).ToNot(HaveOccurred())
		Expect(retry).To(BeEquivalentTo(0))
		service, err := k8sClient.CoreV1().Services(testNamespace).Get(context.Background(), fmt.Sprintf("%s-%s", exportPrefix, testVMExport.Name), metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(service.Name).To(Equal(fmt.Sprintf("%s-%s", exportPrefix, testVMExport.Name)))
	})

	It("Should retry if PVC is in use by other pod", func() {
		testVMExport := createPVCVMExport()
		pod := &k8sv1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "inuse-pod",
				Namespace: testNamespace,
			},
			Spec: k8sv1.PodSpec{
				Volumes: []k8sv1.Volume{
					{
						VolumeSource: k8sv1.VolumeSource{
							PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
								ClaimName: testPVCName,
							},
						},
					},
				},
			},
		}
		pvc := &k8sv1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testPVCName,
				Namespace: testNamespace,
			},
			Status: k8sv1.PersistentVolumeClaimStatus{
				Phase: k8sv1.ClaimBound,
			},
		}
		vmExportClient.Fake.PrependReactor("update", "virtualmachineexports", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			update, ok := action.(testing.UpdateAction)
			Expect(ok).To(BeTrue())
			vmExport, ok := update.GetObject().(*exportv1.VirtualMachineExport)
			Expect(ok).To(BeTrue())
			verifyLinksEmpty(vmExport)
			return true, vmExport, nil
		})
		controller.PodInformer.GetStore().Add(pod)
		controller.PVCInformer.GetStore().Add(pvc)
		retry, err := controller.updateVMExport(testVMExport)
		Expect(err).ToNot(HaveOccurred())
		Expect(retry).To(BeEquivalentTo(requeueTime))
		service, err := k8sClient.CoreV1().Services(testNamespace).Get(context.Background(), fmt.Sprintf("%s-%s", exportPrefix, testVMExport.Name), metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(service.Name).To(Equal(fmt.Sprintf("%s-%s", exportPrefix, testVMExport.Name)))
	})

	DescribeTable("should detect content type properly", func(key, contentType string, expectedRes bool) {
		pvc := &k8sv1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					key: contentType,
				},
			},
		}
		res := controller.isKubevirtContentType(pvc)
		Expect(res).To(Equal(expectedRes))
	},
		Entry("missing content-type", "something", "something", false),
		Entry("blank content-type", annContentType, "", true),
		Entry("kubevirt content-type", annContentType, string(cdiv1.DataVolumeKubeVirt), true),
		Entry("archive content-type", annContentType, string(cdiv1.DataVolumeArchive), false),
	)

	DescribeTable("should detect kubevirt content type if a datavolume exists that is kubevirt", func(contentType cdiv1.DataVolumeContentType, expected bool) {
		dv := &cdiv1.DataVolume{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-dv",
				Namespace: testNamespace,
			},
			Spec: cdiv1.DataVolumeSpec{
				ContentType: contentType,
			},
		}
		controller.DataVolumeInformer.GetStore().Add(dv)
		pvc := &k8sv1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-dv",
				Namespace: testNamespace,
				OwnerReferences: []metav1.OwnerReference{
					*metav1.NewControllerRef(dv, schema.GroupVersionKind{
						Group:   cdiv1.SchemeGroupVersion.Group,
						Version: cdiv1.SchemeGroupVersion.Version,
						Kind:    "DataVolume",
					}),
				},
			},
		}
		res := controller.isKubevirtContentType(pvc)
		Expect(res).To(Equal(expected))
	},
		Entry("missing content-type", cdiv1.DataVolumeContentType(""), true),
		Entry("content-type kubevirt", cdiv1.DataVolumeKubeVirt, true),
		Entry("content-type archive", cdiv1.DataVolumeArchive, false),
	)

	DescribeTable("should create proper condition from PVC", func(phase k8sv1.PersistentVolumeClaimPhase, status k8sv1.ConditionStatus, reason, message string) {
		pvc := &k8sv1.PersistentVolumeClaim{
			Status: k8sv1.PersistentVolumeClaimStatus{
				Phase: phase,
			},
		}
		expectedCond := newPvcCondition(status, reason, message)
		condRes := controller.pvcConditionFromPVC([]*k8sv1.PersistentVolumeClaim{pvc})
		Expect(condRes.Type).To(Equal(expectedCond.Type))
		Expect(condRes.Status).To(Equal(expectedCond.Status))
		Expect(condRes.Reason).To(Equal(expectedCond.Reason))
		Expect(condRes.Message).To(Equal(message))
	},
		Entry("PVC bound", k8sv1.ClaimBound, k8sv1.ConditionTrue, pvcBoundReason, ""),
		Entry("PVC claim lost", k8sv1.ClaimLost, k8sv1.ConditionFalse, unknownReason, ""),
		Entry("PVC pending", k8sv1.ClaimPending, k8sv1.ConditionFalse, pvcPendingReason, ""),
	)
})
