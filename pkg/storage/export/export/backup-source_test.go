/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
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

package export

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	vsv1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	routev1 "github.com/openshift/api/route/v1"
	appsv1 "k8s.io/api/apps/v1"
	k8sv1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	backupv1 "kubevirt.io/api/backup/v1alpha1"
	virtv1 "kubevirt.io/api/core/v1"
	exportv1 "kubevirt.io/api/export/v1beta1"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"
	snapshotv1 "kubevirt.io/api/snapshot/v1beta1"
	"kubevirt.io/client-go/kubecli"
	kubevirtfake "kubevirt.io/client-go/kubevirt/fake"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/certificates/bootstrap"
	virtcontroller "kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
)

const (
	testBackupName           = "test-backup"
	testBackupUID            = "123456"
	testBackupCheckpointName = "test-checkpoint"
	testBackupVolumeName     = "test-datavolume"
)

var _ = Describe("Backup source", func() {
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
		vmBackupInformer            cache.SharedIndexInformer
		controllerRevisionInformer  cache.SharedIndexInformer
		rqInformer                  cache.SharedIndexInformer
		nsInformer                  cache.SharedIndexInformer
		k8sClient                   *k8sfake.Clientset
		vmExportClient              *kubevirtfake.Clientset
		fakeVolumeSnapshotProvider  *MockVolumeSnapshotProvider
		mockVMExportQueue           *testutils.MockWorkQueue[string]
		routeCache                  cache.Store
		ingressCache                cache.Store
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
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
		vmBackupInformer, _ = testutils.NewFakeInformerFor(&backupv1.VirtualMachineBackup{})
		rqInformer, _ = testutils.NewFakeInformerFor(&k8sv1.ResourceQuota{})
		nsInformer, _ = testutils.NewFakeInformerFor(&k8sv1.Namespace{})
		fakeVolumeSnapshotProvider = &MockVolumeSnapshotProvider{
			volumeSnapshots: []*vsv1.VolumeSnapshot{},
		}
		fakeCertManager, err := bootstrap.NewMockCertificateManager()
		Expect(err).ToNot(HaveOccurred())

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
			ManifestRenderer:            services.NewTemplateService("a", 240, "b", "c", "d", "e", "f", pvcInformer.GetStore(), virtClient, config, qemuGid, "g", rqInformer.GetStore(), nsInformer.GetStore()),
			caCertManager:               fakeCertManager,
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
			VMBackupInformer:            vmBackupInformer,
			BackupCAConfigMapInformer:   cmInformer,
		}
		initCert = func(ctrl *VMExportController) {
			ctrl.caCertManager.Start()
			Expect(ctrl.caCertManager.Current()).ToNot(BeNil())
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

	createTestVMBackup := func(conditions []backupv1.Condition, includedVolumes []backupv1.BackupVolumeInfo, checkpointName *string) *backupv1.VirtualMachineBackup {
		return &backupv1.VirtualMachineBackup{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testBackupName,
				Namespace: testNamespace,
				UID:       testBackupUID,
			},
			Status: &backupv1.VirtualMachineBackupStatus{
				Type:            backupv1.Full,
				Conditions:      conditions,
				IncludedVolumes: includedVolumes,
				CheckpointName:  checkpointName,
			},
		}
	}

	createBackupVMExport := func() *exportv1.VirtualMachineExport {
		return &exportv1.VirtualMachineExport{
			ObjectMeta: createVMExportMeta(vmExportName),
			Spec: exportv1.VirtualMachineExportSpec{
				Source: k8sv1.TypedLocalObjectReference{
					APIGroup: &backupv1.SchemeGroupVersion.Group,
					Kind:     "VirtualMachineBackup",
					Name:     testBackupName,
				},
				TokenSecretRef: &tokenSecretName,
			},
		}
	}

	It("Should create VM export when backup is progressing", func() {
		testVMExport := createBackupVMExport()
		vmBackup := createTestVMBackup(
			[]backupv1.Condition{{Type: backupv1.ConditionProgressing, Status: k8sv1.ConditionTrue}},
			[]backupv1.BackupVolumeInfo{{VolumeName: testBackupVolumeName}},
			pointer.P(testBackupCheckpointName),
		)
		controller.VMBackupInformer.GetStore().Add(vmBackup)

		var pod *k8sv1.Pod
		k8sClient.Fake.PrependReactor("create", "pods", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			create, ok := action.(testing.CreateAction)
			Expect(ok).To(BeTrue())
			pod, ok = create.GetObject().(*k8sv1.Pod)
			Expect(ok).To(BeTrue())

			pod.Status = k8sv1.PodStatus{
				Phase: k8sv1.PodRunning,
				Conditions: []k8sv1.PodCondition{
					{Type: k8sv1.PodReady, Status: k8sv1.ConditionTrue},
				},
			}
			return true, pod, nil
		})

		vmExportClient.Fake.PrependReactor("update", "virtualmachineexports", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			update, ok := action.(testing.UpdateAction)
			Expect(ok).To(BeTrue())
			vmExport, ok := update.GetObject().(*exportv1.VirtualMachineExport)
			Expect(ok).To(BeTrue())
			Expect(vmExport.Status.Phase).To(Equal(exportv1.Ready))

			for _, condition := range vmExport.Status.Conditions {
				if condition.Type == exportv1.ConditionReady {
					Expect(condition.Status).To(Equal(k8sv1.ConditionTrue))
					Expect(condition.Reason).To(Equal(vmBackupReadyReason))
				}
			}

			Expect(vmExport.Status.Links).ToNot(BeNil())
			Expect(vmExport.Status.Links.Internal).ToNot(BeNil())
			Expect(vmExport.Status.Links.Internal.Backups).To(HaveLen(1))
			Expect(vmExport.Status.Links.Internal.Backups[0].Name).To(Equal(testBackupVolumeName))
			Expect(vmExport.Status.Links.Internal.Backups[0].Endpoints).To(HaveLen(2))

			return true, vmExport, nil
		})

		retry, err := controller.updateVMExport(testVMExport)
		Expect(err).ToNot(HaveOccurred())
		Expect(retry).To(BeEquivalentTo(0))
		Expect(pod).ToNot(BeNil())

		Expect(pod.Spec.Volumes).To(HaveLen(2), "Backup pods should only mount cert and token secrets")
		Expect(pod.Spec.Containers[0].VolumeDevices).To(BeEmpty())

		Expect(pod.Spec.Containers).To(HaveLen(1))
		cert, err := controller.backupCA()
		Expect(err).ToNot(HaveOccurred())

		Expect(pod.Spec.Containers[0].Env).To(ContainElements(
			k8sv1.EnvVar{Name: "BACKUP_CACERT", Value: cert},
			k8sv1.EnvVar{Name: "BACKUP_UID", Value: testBackupUID},
			k8sv1.EnvVar{Name: "BACKUP_TYPE", Value: string(backupv1.Full)},
			k8sv1.EnvVar{Name: "BACKUP_CHECKPOINT", Value: testBackupCheckpointName},
			k8sv1.EnvVar{Name: "BACKUP0_BACKUP_PATH", Value: testBackupVolumeName},
			k8sv1.EnvVar{Name: "BACKUP0_DATA_URI", Value: backupDataURI(testBackupVolumeName)},
			k8sv1.EnvVar{Name: "BACKUP0_MAP_URI", Value: backupMapURI(testBackupVolumeName)},
		))
		testutils.ExpectEvent(recorder, serviceCreatedEvent)
	})

	DescribeTable("Should update VM Export status according to backup source",
		func(hasContent bool, backupConditions []backupv1.Condition, expectedReadyStatus k8sv1.ConditionStatus, expectedMessage string) {
			testVMExport := createBackupVMExport()

			var volumes []backupv1.BackupVolumeInfo
			var checkpoint *string

			if hasContent {
				volumes = []backupv1.BackupVolumeInfo{{VolumeName: testBackupVolumeName}}
				checkpoint = pointer.P(testBackupCheckpointName)
			}

			vmBackup := createTestVMBackup(backupConditions, volumes, checkpoint)
			controller.VMBackupInformer.GetStore().Add(vmBackup)

			vmExportClient.Fake.PrependReactor("update", "virtualmachineexports", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				update, ok := action.(testing.UpdateAction)
				Expect(ok).To(BeTrue())
				vmExport, ok := update.GetObject().(*exportv1.VirtualMachineExport)
				Expect(ok).To(BeTrue())

				Expect(vmExport.Status.Conditions).To(ContainElement(SatisfyAll(
					HaveField("Type", exportv1.ConditionReady),
					HaveField("Status", expectedReadyStatus),
					HaveField("Reason", vmBackupReadyReason),
					HaveField("Message", expectedMessage),
				)), "Ready condition should be set with the correct status and message")
				return true, vmExport, nil
			})

			if expectedReadyStatus == k8sv1.ConditionFalse {
				k8sClient.Fake.PrependReactor("create", "pods", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
					Fail("Should not create pods when backup is not ready")
					return true, nil, nil
				})
			}

			retry, err := controller.updateVMExport(testVMExport)

			Expect(err).ToNot(HaveOccurred())
			Expect(retry).To(Equal(requeueTime))
		},
		Entry("when backup lacks content",
			false,
			[]backupv1.Condition{{Type: backupv1.ConditionProgressing, Status: k8sv1.ConditionFalse}},
			k8sv1.ConditionFalse,
			vmBackupNotReadyMessage,
		),
		Entry("when backup Progressing condition is missing entirely",
			true,
			[]backupv1.Condition{},
			k8sv1.ConditionFalse,
			"backup progressing condition not found",
		),
		Entry("when backup Progressing condition is false",
			true,
			[]backupv1.Condition{{Type: backupv1.ConditionProgressing, Status: k8sv1.ConditionFalse, Message: "Backup encountered a fatal error"}},
			k8sv1.ConditionFalse,
			"Backup encountered a fatal error",
		),
	)

	It("Should return error if VirtualMachineBackup is not found", func() {
		testVMExport := createBackupVMExport()

		retry, err := controller.updateVMExport(testVMExport)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("VirtualMachineBackup not found: %s/%s", testVMExport.Namespace, testVMExport.Spec.Source.Name))
		Expect(retry).To(BeEquivalentTo(0))
	})

	It("Should return error if VirtualMachineBackup status is empty", func() {
		testVMExport := createBackupVMExport()

		vmBackup := &backupv1.VirtualMachineBackup{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testBackupName,
				Namespace: testNamespace,
			},
		}
		controller.VMBackupInformer.GetStore().Add(vmBackup)

		retry, err := controller.updateVMExport(testVMExport)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("backup status empty"))
		Expect(retry).To(BeEquivalentTo(0))
	})

	It("Should add vmexport to queue if matching VMBackup is added/updated", func() {
		vmExport := createBackupVMExport()
		vmBackup := createTestVMBackup(nil, nil, nil)

		Expect(controller.VMExportInformer.GetStore().Add(vmExport)).To(Succeed())

		mockVMExportQueue.ExpectAdds(1)
		controller.handleVMBackup(vmBackup)
		mockVMExportQueue.Wait()
	})

	It("Should properly omit BACKUP_CHECKPOINT from pod when checkpoint is nil", func() {
		testVMExport := createBackupVMExport()
		vmBackup := createTestVMBackup(
			[]backupv1.Condition{{Type: backupv1.ConditionProgressing, Status: k8sv1.ConditionTrue}},
			[]backupv1.BackupVolumeInfo{{VolumeName: testBackupVolumeName}},
			nil,
		)
		controller.VMBackupInformer.GetStore().Add(vmBackup)

		var pod *k8sv1.Pod

		k8sClient.Fake.PrependReactor("create", "pods", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			create, ok := action.(testing.CreateAction)
			Expect(ok).To(BeTrue())
			pod, ok = create.GetObject().(*k8sv1.Pod)
			Expect(ok).To(BeTrue())
			return true, pod, nil
		})

		k8sClient.Fake.PrependReactor("create", "services", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			return true, &k8sv1.Service{ObjectMeta: metav1.ObjectMeta{Name: "test-svc"}}, nil
		})

		vmExportClient.Fake.PrependReactor("update", "virtualmachineexports", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			update, ok := action.(testing.UpdateAction)
			Expect(ok).To(BeTrue())
			vmExport, ok := update.GetObject().(*exportv1.VirtualMachineExport)
			Expect(ok).To(BeTrue())
			return true, vmExport, nil
		})

		_, err := controller.updateVMExport(testVMExport)
		Expect(err).ToNot(HaveOccurred())
		Expect(pod).ToNot(BeNil())
		Expect(pod.Spec.Containers[0].Env).ToNot(ContainElement(HaveField("Name", "BACKUP_CHECKPOINT")))
	})
})
