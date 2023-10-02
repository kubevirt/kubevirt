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
	vsv1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	routev1 "github.com/openshift/api/route/v1"

	appsv1 "k8s.io/api/apps/v1"
	k8sv1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/pointer"

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
	testVmsnapshotName     = "test-vmsnapshot"
	testVolumesnapshotName = "test-snapshot"
)

var _ = Describe("VMSnapshot source", func() {
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
		initCert = func(_ *VMExportController) {
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

	createTestVMSnapshot := func(ready bool) *snapshotv1.VirtualMachineSnapshot {
		return &snapshotv1.VirtualMachineSnapshot{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testVmsnapshotName,
				Namespace: testNamespace,
			},
			Spec: snapshotv1.VirtualMachineSnapshotSpec{},
			Status: &snapshotv1.VirtualMachineSnapshotStatus{
				VirtualMachineSnapshotContentName: pointer.StringPtr("snapshot-content"),
				ReadyToUse:                        pointer.BoolPtr(ready),
			},
		}
	}

	createTestVMSnapshotContent := func(name string) *snapshotv1.VirtualMachineSnapshotContent {
		return &snapshotv1.VirtualMachineSnapshotContent{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: testNamespace,
			},
			Spec: snapshotv1.VirtualMachineSnapshotContentSpec{
				VolumeBackups: []snapshotv1.VolumeBackup{
					{
						VolumeName: "test-volume",
						PersistentVolumeClaim: snapshotv1.PersistentVolumeClaim{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test-snapshot",
							},
							Spec: k8sv1.PersistentVolumeClaimSpec{
								Resources: k8sv1.VolumeResourceRequirements{
									Requests: k8sv1.ResourceList{},
								},
							},
						},
						VolumeSnapshotName: pointer.StringPtr(testVolumesnapshotName),
					},
				},
				Source: snapshotv1.SourceSpec{
					VirtualMachine: &snapshotv1.VirtualMachine{
						Spec: virtv1.VirtualMachineSpec{
							Template: &virtv1.VirtualMachineInstanceTemplateSpec{
								Spec: virtv1.VirtualMachineInstanceSpec{
									Volumes: []virtv1.Volume{
										{
											Name: "test-volume",
											VolumeSource: virtv1.VolumeSource{
												DataVolume: &virtv1.DataVolumeSource{},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			Status: &snapshotv1.VirtualMachineSnapshotContentStatus{
				VolumeSnapshotStatus: []snapshotv1.VolumeSnapshotStatus{
					{
						VolumeSnapshotName: testVolumesnapshotName,
						ReadyToUse:         pointer.BoolPtr(true),
					},
				},
			},
		}
	}

	createTestVMSnapshotContentNoVolumes := func(name string) *snapshotv1.VirtualMachineSnapshotContent {
		return &snapshotv1.VirtualMachineSnapshotContent{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: testNamespace,
			},
			Spec: snapshotv1.VirtualMachineSnapshotContentSpec{
				VolumeBackups: []snapshotv1.VolumeBackup{},
				Source: snapshotv1.SourceSpec{
					VirtualMachine: &snapshotv1.VirtualMachine{
						Spec: virtv1.VirtualMachineSpec{
							Template: &virtv1.VirtualMachineInstanceTemplateSpec{
								Spec: virtv1.VirtualMachineInstanceSpec{
									Volumes: []virtv1.Volume{
										{
											Name: "test-volume",
											VolumeSource: virtv1.VolumeSource{
												DataVolume: &virtv1.DataVolumeSource{},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			Status: &snapshotv1.VirtualMachineSnapshotContentStatus{
				VolumeSnapshotStatus: []snapshotv1.VolumeSnapshotStatus{},
			},
		}
	}

	createTestVolumeSnapshot := func(name string) *vsv1.VolumeSnapshot {
		size := resource.MustParse("1Gi")
		return &vsv1.VolumeSnapshot{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: testNamespace,
			},
			Spec: vsv1.VolumeSnapshotSpec{},
			Status: &vsv1.VolumeSnapshotStatus{
				RestoreSize: &size,
			},
		}
	}

	createRestoredPVC := func(name string) *k8sv1.PersistentVolumeClaim {
		return &k8sv1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: testNamespace,
			},
			Spec: k8sv1.PersistentVolumeClaimSpec{
				Resources: k8sv1.VolumeResourceRequirements{
					Requests: k8sv1.ResourceList{
						k8sv1.ResourceStorage: resource.MustParse("1Gi"),
					},
				},
				DataSource: &k8sv1.TypedLocalObjectReference{
					APIGroup: pointer.StringPtr(vsv1.GroupName),
					Kind:     "VolumeSnapshot",
					Name:     testVolumesnapshotName,
				},
			},
			Status: k8sv1.PersistentVolumeClaimStatus{
				Phase: k8sv1.ClaimBound,
			},
		}
	}

	It("Should properly update VMExport status with a valid token and no VMSnapshot", func() {
		testVMExport := createSnapshotVMExport()
		vmExportClient.Fake.PrependReactor("update", "virtualmachineexports", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			update, ok := action.(testing.UpdateAction)
			Expect(ok).To(BeTrue())
			vmExport, ok := update.GetObject().(*exportv1.VirtualMachineExport)
			Expect(ok).To(BeTrue())
			verifyLinksEmpty(vmExport)
			for _, condition := range vmExport.Status.Conditions {
				if condition.Type == exportv1.ConditionReady {
					Expect(condition.Status).To(Equal(k8sv1.ConditionFalse))
					Expect(condition.Reason).To(Equal(initializingReason))
					Expect(condition.Message).To(Equal(""))
				}
				if condition.Type == exportv1.ConditionVolumesCreated {
					Expect(condition.Status).To(Equal(k8sv1.ConditionFalse))
					Expect(condition.Reason).To(Equal(noVolumeSnapshotReason))
					Expect(condition.Message).To(Equal(fmt.Sprintf("VirtualMachineSnapshot %s/%s does not contain any volume snapshots", vmExport.Namespace, vmExport.Spec.Source.Name)))
				}
			}
			return true, vmExport, nil
		})

		retry, err := controller.updateVMExport(testVMExport)
		Expect(err).ToNot(HaveOccurred())
		Expect(retry).To(BeEquivalentTo(0))
		service, err := k8sClient.CoreV1().Services(testNamespace).Get(context.Background(), fmt.Sprintf("%s-%s", exportPrefix, testVMExport.Name), metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(service.Name).To(Equal(fmt.Sprintf("%s-%s", exportPrefix, testVMExport.Name)))
	})

	It("Should properly update VMExport status with a valid token with VMSnapshot without volumes", func() {
		testVMExport := createSnapshotVMExport()
		vmExportClient.Fake.PrependReactor("update", "virtualmachineexports", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			update, ok := action.(testing.UpdateAction)
			Expect(ok).To(BeTrue())
			vmExport, ok := update.GetObject().(*exportv1.VirtualMachineExport)
			Expect(ok).To(BeTrue())
			verifyLinksEmpty(vmExport)
			volumeCreateConditionSet := false
			for _, condition := range vmExport.Status.Conditions {
				if condition.Type == exportv1.ConditionReady {
					Expect(condition.Status).To(Equal(k8sv1.ConditionFalse))
					Expect(condition.Reason).To(Equal(initializingReason))
					Expect(condition.Message).To(Equal(""))
				}
				if condition.Type == exportv1.ConditionVolumesCreated {
					volumeCreateConditionSet = true
					Expect(condition.Status).To(Equal(k8sv1.ConditionFalse))
					Expect(condition.Reason).To(Equal(noVolumeSnapshotReason))
					Expect(condition.Message).To(Equal(fmt.Sprintf("VirtualMachineSnapshot %s/%s does not contain any volume snapshots", vmExport.Namespace, vmExport.Spec.Source.Name)))
				}
			}
			Expect(volumeCreateConditionSet).To(BeTrue())
			Expect(vmExport.Status.Phase).To(Equal(exportv1.Skipped))
			return true, vmExport, nil
		})
		vmSnapshotInformer.GetStore().Add(createTestVMSnapshot(true))
		vmSnapshotContentInformer.GetStore().Add(createTestVMSnapshotContentNoVolumes("snapshot-content"))
		retry, err := controller.updateVMExport(testVMExport)
		Expect(err).ToNot(HaveOccurred())
		Expect(retry).To(BeEquivalentTo(0))
		service, err := k8sClient.CoreV1().Services(testNamespace).Get(context.Background(), fmt.Sprintf("%s-%s", exportPrefix, testVMExport.Name), metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(service.Name).To(Equal(fmt.Sprintf("%s-%s", exportPrefix, testVMExport.Name)))
	})

	It("Should create restored PVCs from VMSnapshot", func() {
		testVMExport := createSnapshotVMExport()
		vmExportClient.Fake.PrependReactor("update", "virtualmachineexports", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			update, ok := action.(testing.UpdateAction)
			Expect(ok).To(BeTrue())
			vmExport, ok := update.GetObject().(*exportv1.VirtualMachineExport)
			Expect(ok).To(BeTrue())
			verifyLinksEmpty(vmExport)
			volumeCreateConditionSet := false
			for _, condition := range vmExport.Status.Conditions {
				if condition.Type == exportv1.ConditionReady {
					Expect(condition.Status).To(Equal(k8sv1.ConditionFalse))
					Expect(condition.Reason).To(Equal(podPendingReason))
					Expect(condition.Message).To(Equal(""))
				}
				if condition.Type == exportv1.ConditionVolumesCreated {
					volumeCreateConditionSet = true
					Expect(condition.Status).To(Equal(k8sv1.ConditionFalse))
					Expect(condition.Reason).To(Equal(notAllPVCsReady))
					Expect(condition.Message).To(Equal("Not all PVCs are ready"))
				}
			}
			Expect(volumeCreateConditionSet).To(BeTrue())
			Expect(vmExport.Status.Phase).To(Equal(exportv1.Pending))
			return true, vmExport, nil
		})

		k8sClient.Fake.PrependReactor("create", "persistentvolumeclaims", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			create, ok := action.(testing.CreateAction)
			Expect(ok).To(BeTrue())
			pvc, ok := create.GetObject().(*k8sv1.PersistentVolumeClaim)
			Expect(ok).To(BeTrue())
			Expect(pvc.Name).To(Equal("test-test-snapshot"))
			Expect(pvc.Spec.DataSource).ToNot(BeNil())
			Expect(pvc.Spec.Resources.Requests).ToNot(BeEmpty())
			Expect(pvc.Spec.Resources.Requests[k8sv1.ResourceStorage]).To(Equal(resource.MustParse("1Gi")))
			Expect(pvc.Spec.DataSource).To(Equal(&k8sv1.TypedLocalObjectReference{
				APIGroup: pointer.StringPtr(vsv1.GroupName),
				Kind:     "VolumeSnapshot",
				Name:     testVolumesnapshotName,
			}))
			By("Ensuring the PVC is owned by the vmExport")
			Expect(pvc.OwnerReferences).To(HaveLen(1))
			Expect(pvc.OwnerReferences[0]).To(Equal(metav1.OwnerReference{
				APIVersion:         exportGVK.GroupVersion().String(),
				Kind:               "VirtualMachineExport",
				Name:               testVMExport.Name,
				UID:                testVMExport.UID,
				Controller:         pointer.BoolPtr(true),
				BlockOwnerDeletion: pointer.BoolPtr(true),
			}))
			return true, pvc, nil
		})
		expectExporterCreate(k8sClient, k8sv1.PodPending)

		vmSnapshotInformer.GetStore().Add(createTestVMSnapshot(true))
		vmSnapshotContentInformer.GetStore().Add(createTestVMSnapshotContent("snapshot-content"))
		fakeVolumeSnapshotProvider.Add(createTestVolumeSnapshot(testVolumesnapshotName))
		retry, err := controller.updateVMExport(testVMExport)
		Expect(err).ToNot(HaveOccurred())
		Expect(retry).To(BeEquivalentTo(0))
	})

	It("Should not re-create restored PVCs from VMSnapshot if pvc already exists", func() {
		testVMExport := createSnapshotVMExport()
		vmExportClient.Fake.PrependReactor("update", "virtualmachineexports", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			update, ok := action.(testing.UpdateAction)
			Expect(ok).To(BeTrue())
			vmExport, ok := update.GetObject().(*exportv1.VirtualMachineExport)
			Expect(ok).To(BeTrue())
			verifyLinksEmpty(vmExport)
			volumeCreateConditionSet := false
			for _, condition := range vmExport.Status.Conditions {
				if condition.Type == exportv1.ConditionReady {
					Expect(condition.Status).To(Equal(k8sv1.ConditionFalse))
					Expect(condition.Reason).To(Equal(podPendingReason))
					Expect(condition.Message).To(Equal(""))
				}
				if condition.Type == exportv1.ConditionVolumesCreated {
					volumeCreateConditionSet = true
					Expect(condition.Status).To(Equal(k8sv1.ConditionTrue))
					Expect(condition.Reason).To(Equal(allPVCsReady))
					Expect(condition.Message).To(Equal(""))
				}
			}
			Expect(volumeCreateConditionSet).To(BeTrue())
			Expect(vmExport.Status.Phase).To(Equal(exportv1.Pending))
			return true, vmExport, nil
		})

		k8sClient.Fake.PrependReactor("create", "persistentvolumeclaims", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			_, ok := action.(testing.CreateAction)
			Expect(ok).To(BeTrue())
			Fail("unexpected create persistentvolumeclaims called")
			return true, nil, nil
		})
		expectExporterCreate(k8sClient, k8sv1.PodPending)
		pvcInformer.GetStore().Add(createRestoredPVC("test-test-snapshot"))
		vmSnapshotInformer.GetStore().Add(createTestVMSnapshot(true))
		vmSnapshotContentInformer.GetStore().Add(createTestVMSnapshotContent("snapshot-content"))
		fakeVolumeSnapshotProvider.Add(createTestVolumeSnapshot(testVolumesnapshotName))
		retry, err := controller.updateVMExport(testVMExport)
		Expect(err).ToNot(HaveOccurred())
		Expect(retry).To(BeEquivalentTo(0))
	})

	It("Should update status with correct links from snapshot with kubevirt content type", func() {
		testVMExport := createSnapshotVMExport()
		restoreName := fmt.Sprintf("%s-%s", testVMExport.Name, testVolumesnapshotName)
		vmExportClient.Fake.PrependReactor("update", "virtualmachineexports", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			update, ok := action.(testing.UpdateAction)
			Expect(ok).To(BeTrue())
			vmExport, ok := update.GetObject().(*exportv1.VirtualMachineExport)
			Expect(ok).To(BeTrue())
			verifyKubevirtInternal(vmExport, vmExport.Name, testNamespace, restoreName)
			verifyKubevirtExternal(vmExport, vmExport.Name, testNamespace, restoreName)
			return true, vmExport, nil
		})

		k8sClient.Fake.PrependReactor("create", "persistentvolumeclaims", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			create, ok := action.(testing.CreateAction)
			Expect(ok).To(BeTrue())
			pvc, ok := create.GetObject().(*k8sv1.PersistentVolumeClaim)
			Expect(ok).To(BeTrue())
			Expect(pvc.Name).To(Equal("test-test-snapshot"))
			Expect(pvc.Spec.DataSource).ToNot(BeNil())
			Expect(pvc.Spec.Resources.Requests).ToNot(BeEmpty())
			Expect(pvc.Spec.Resources.Requests[k8sv1.ResourceStorage]).To(Equal(resource.MustParse("1Gi")))
			Expect(pvc.Spec.DataSource).To(Equal(&k8sv1.TypedLocalObjectReference{
				APIGroup: pointer.StringPtr(vsv1.GroupName),
				Kind:     "VolumeSnapshot",
				Name:     testVolumesnapshotName,
			}))
			By("Ensuring the PVC is owned by the vmExport")
			Expect(pvc.OwnerReferences).To(HaveLen(1))
			Expect(pvc.OwnerReferences[0]).To(Equal(metav1.OwnerReference{
				APIVersion:         exportGVK.GroupVersion().String(),
				Kind:               "VirtualMachineExport",
				Name:               testVMExport.Name,
				UID:                testVMExport.UID,
				Controller:         pointer.BoolPtr(true),
				BlockOwnerDeletion: pointer.BoolPtr(true),
			}))
			Expect(pvc.GetAnnotations()).ToNot(BeEmpty())
			Expect(pvc.GetAnnotations()[annContentType]).To(BeEquivalentTo(cdiv1.DataVolumeKubeVirt))
			return true, pvc, nil
		})
		expectExporterCreate(k8sClient, k8sv1.PodRunning)
		controller.RouteCache.Add(routeToHostAndService(components.VirtExportProxyServiceName))
		vmSnapshotInformer.GetStore().Add(createTestVMSnapshot(true))
		vmSnapshotContentInformer.GetStore().Add(createTestVMSnapshotContent("snapshot-content"))
		fakeVolumeSnapshotProvider.Add(createTestVolumeSnapshot(testVolumesnapshotName))
		retry, err := controller.updateVMExport(testVMExport)
		Expect(err).ToNot(HaveOccurred())
		Expect(retry).To(BeEquivalentTo(0))
	})

	It("Should update status with correct links from snapshot with other content type", func() {
		testVMExport := createSnapshotVMExport()
		restoreName := fmt.Sprintf("%s-%s", testVMExport.Name, testVolumesnapshotName)
		vmExportClient.Fake.PrependReactor("update", "virtualmachineexports", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			update, ok := action.(testing.UpdateAction)
			Expect(ok).To(BeTrue())
			vmExport, ok := update.GetObject().(*exportv1.VirtualMachineExport)
			Expect(ok).To(BeTrue())
			verifyArchiveInternal(vmExport, vmExport.Name, testNamespace, restoreName)
			verifyArchiveExternal(vmExport, vmExport.Name, testNamespace, restoreName)
			return true, vmExport, nil
		})

		k8sClient.Fake.PrependReactor("create", "persistentvolumeclaims", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			create, ok := action.(testing.CreateAction)
			Expect(ok).To(BeTrue())
			pvc, ok := create.GetObject().(*k8sv1.PersistentVolumeClaim)
			Expect(ok).To(BeTrue())
			Expect(pvc.Name).To(Equal("test-test-snapshot"))
			Expect(pvc.Spec.DataSource).ToNot(BeNil())
			Expect(pvc.Spec.Resources.Requests).ToNot(BeEmpty())
			Expect(pvc.Spec.Resources.Requests[k8sv1.ResourceStorage]).To(Equal(resource.MustParse("1Gi")))
			Expect(pvc.Spec.DataSource).To(Equal(&k8sv1.TypedLocalObjectReference{
				APIGroup: pointer.StringPtr(vsv1.GroupName),
				Kind:     "VolumeSnapshot",
				Name:     testVolumesnapshotName,
			}))
			By("Ensuring the PVC is owned by the vmExport")
			Expect(pvc.OwnerReferences).To(HaveLen(1))
			Expect(pvc.OwnerReferences[0]).To(Equal(metav1.OwnerReference{
				APIVersion:         exportGVK.GroupVersion().String(),
				Kind:               "VirtualMachineExport",
				Name:               testVMExport.Name,
				UID:                testVMExport.UID,
				Controller:         pointer.BoolPtr(true),
				BlockOwnerDeletion: pointer.BoolPtr(true),
			}))
			Expect(pvc.GetAnnotations()[annContentType]).To(BeEmpty())
			return true, pvc, nil
		})
		expectExporterCreate(k8sClient, k8sv1.PodRunning)
		controller.RouteCache.Add(routeToHostAndService(components.VirtExportProxyServiceName))
		vmSnapshotInformer.GetStore().Add(createTestVMSnapshot(true))
		content := createTestVMSnapshotContent("snapshot-content")
		content.Spec.Source.VirtualMachine.Spec.Template.Spec.Volumes[0].DataVolume = nil
		content.Spec.Source.VirtualMachine.Spec.Template.Spec.Volumes[0].MemoryDump = &virtv1.MemoryDumpVolumeSource{}
		vmSnapshotContentInformer.GetStore().Add(content)
		fakeVolumeSnapshotProvider.Add(createTestVolumeSnapshot(testVolumesnapshotName))
		retry, err := controller.updateVMExport(testVMExport)
		Expect(err).ToNot(HaveOccurred())
		Expect(retry).To(BeEquivalentTo(0))
	})

	It("Should update status with no links and not ready if snapshot is not ready", func() {
		testVMExport := createSnapshotVMExport()
		vmExportClient.Fake.PrependReactor("update", "virtualmachineexports", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			update, ok := action.(testing.UpdateAction)
			Expect(ok).To(BeTrue())
			vmExport, ok := update.GetObject().(*exportv1.VirtualMachineExport)
			Expect(ok).To(BeTrue())
			verifyLinksEmpty(vmExport)
			volumeCreateConditionSet := false
			for _, condition := range vmExport.Status.Conditions {
				if condition.Type == exportv1.ConditionReady {
					Expect(condition.Status).To(Equal(k8sv1.ConditionFalse))
					Expect(condition.Reason).To(Equal(inUseReason))
					Expect(condition.Message).To(Equal(fmt.Sprintf("VirtualMachineSnapshot %s/%s is not ready to use", vmExport.Namespace, vmExport.Spec.Source.Name)))
				}
				if condition.Type == exportv1.ConditionVolumesCreated {
					volumeCreateConditionSet = true
					Expect(condition.Status).To(Equal(k8sv1.ConditionFalse))
					Expect(condition.Reason).To(Equal(notAllPVCsCreated))
					Expect(condition.Message).To(Equal(fmt.Sprintf("VirtualMachineSnapshot %s/%s is not ready to use", vmExport.Namespace, vmExport.Spec.Source.Name)))
				}
			}
			Expect(volumeCreateConditionSet).To(BeTrue())
			Expect(vmExport.Status.Phase).To(Equal(exportv1.Pending))
			return true, vmExport, nil
		})

		k8sClient.Fake.PrependReactor("create", "persistentvolumeclaims", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			_, ok := action.(testing.CreateAction)
			Expect(ok).To(BeTrue())
			Fail("Should not create PVCs")
			return true, nil, nil
		})
		vmSnapshotInformer.GetStore().Add(createTestVMSnapshot(false))
		content := createTestVMSnapshotContent("snapshot-content")
		vmSnapshotContentInformer.GetStore().Add(content)
		fakeVolumeSnapshotProvider.Add(createTestVolumeSnapshot(testVolumesnapshotName))
		retry, err := controller.updateVMExport(testVMExport)
		Expect(err).ToNot(HaveOccurred())
		Expect(retry).To(BeEquivalentTo(0))
	})

})
