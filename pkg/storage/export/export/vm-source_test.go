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
	"crypto/tls"
	"fmt"
	"os"
	"path/filepath"
	"time"

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
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	virtv1 "kubevirt.io/api/core/v1"
	exportv1 "kubevirt.io/api/export/v1alpha1"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"
	snapshotv1 "kubevirt.io/api/snapshot/v1alpha1"
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
	testVmName = "test-vm"
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
		fakeVolumeSnapshotProvider = &MockVolumeSnapshotProvider{
			volumeSnapshots: []*vsv1.VolumeSnapshot{},
		}

		config, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&virtv1.KubeVirtConfiguration{})
		k8sClient = k8sfake.NewSimpleClientset()
		vmExportClient = kubevirtfake.NewSimpleClientset()
		recorder = record.NewFakeRecorder(100)

		virtClient.EXPECT().CoreV1().Return(k8sClient.CoreV1()).AnyTimes()
		virtClient.EXPECT().VirtualMachineExport(testNamespace).
			Return(vmExportClient.ExportV1alpha1().VirtualMachineExports(testNamespace)).AnyTimes()

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
			TemplateService:             services.NewTemplateService("a", 240, "b", "c", "d", "e", "f", "g", pvcInformer.GetStore(), virtClient, config, qemuGid, "h", rqInformer.GetStore(), nsInformer.GetStore()),
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

	createExporterPod := func(name string, phase k8sv1.PodPhase) *k8sv1.Pod {
		return &k8sv1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: testNamespace,
			},
			Status: k8sv1.PodStatus{
				Phase: phase,
			},
		}
	}

	createVMWithoutVolumes := func() *virtv1.VirtualMachine {
		return &virtv1.VirtualMachine{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testVmName,
				Namespace: testNamespace,
			},
			Spec: virtv1.VirtualMachineSpec{
				Template: &virtv1.VirtualMachineInstanceTemplateSpec{
					Spec: virtv1.VirtualMachineInstanceSpec{
						Volumes: []virtv1.Volume{},
					},
				},
			},
		}
	}

	createVMWithDataVolumes := func() *virtv1.VirtualMachine {
		vm := createVMWithoutVolumes()
		vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, virtv1.Volume{
			Name: "volume1",
			VolumeSource: virtv1.VolumeSource{
				DataVolume: &virtv1.DataVolumeSource{
					Name: "volume1",
				},
			},
		})
		vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, virtv1.Volume{
			Name: "volume2",
			VolumeSource: virtv1.VolumeSource{
				DataVolume: &virtv1.DataVolumeSource{
					Name: "volume2",
				},
			},
		})
		return vm
	}

	createVMWithPVCs := func() *virtv1.VirtualMachine {
		vm := createVMWithoutVolumes()
		vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, virtv1.Volume{
			Name: "volume1",
			VolumeSource: virtv1.VolumeSource{
				PersistentVolumeClaim: &virtv1.PersistentVolumeClaimVolumeSource{
					PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
						ClaimName: "volume1",
					},
				},
			},
		})
		vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, virtv1.Volume{
			Name: "volume2",
			VolumeSource: virtv1.VolumeSource{
				PersistentVolumeClaim: &virtv1.PersistentVolumeClaimVolumeSource{
					PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
						ClaimName: "volume2",
					},
				},
			},
		})
		return vm
	}

	createVMWithPVCandMemoryDump := func() *virtv1.VirtualMachine {
		vm := createVMWithoutVolumes()
		vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, virtv1.Volume{
			Name: "volume1",
			VolumeSource: virtv1.VolumeSource{
				PersistentVolumeClaim: &virtv1.PersistentVolumeClaimVolumeSource{
					PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
						ClaimName: "volume1",
					},
				},
			},
		})
		vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, virtv1.Volume{
			Name: "volume2",
			VolumeSource: virtv1.VolumeSource{
				MemoryDump: &virtv1.MemoryDumpVolumeSource{
					PersistentVolumeClaimVolumeSource: virtv1.PersistentVolumeClaimVolumeSource{
						PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: "volume2",
						},
					},
				},
			},
		})
		return vm
	}

	createVMIWithDataVolumes := func() *virtv1.VirtualMachineInstance {
		return &virtv1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testVmName,
				Namespace: testNamespace,
			},
			Spec: virtv1.VirtualMachineInstanceSpec{
				Volumes: []virtv1.Volume{
					{
						Name: "volume1",
						VolumeSource: virtv1.VolumeSource{
							DataVolume: &virtv1.DataVolumeSource{
								Name: "volume1",
							},
						},
					},
					{
						Name: "volume2",
						VolumeSource: virtv1.VolumeSource{
							DataVolume: &virtv1.DataVolumeSource{
								Name: "volume2",
							},
						},
					},
				},
			},
		}
	}

	verifyMixedInternal := func(vmExport *exportv1.VirtualMachineExport, exportName, namespace string, volumeNames ...string) {
		exportVolumeFormats := make([]exportv1.VirtualMachineExportVolumeFormat, 0)
		exportVolumeFormats = append(exportVolumeFormats, exportv1.VirtualMachineExportVolumeFormat{
			Format: exportv1.KubeVirtRaw,
			Url:    fmt.Sprintf("https://%s.%s.svc/volumes/%s/disk.img", fmt.Sprintf("%s-%s", exportPrefix, exportName), namespace, volumeNames[0]),
		})
		exportVolumeFormats = append(exportVolumeFormats, exportv1.VirtualMachineExportVolumeFormat{
			Format: exportv1.KubeVirtGz,
			Url:    fmt.Sprintf("https://%s.%s.svc/volumes/%s/disk.img.gz", fmt.Sprintf("%s-%s", exportPrefix, exportName), namespace, volumeNames[0]),
		})
		exportVolumeFormats = append(exportVolumeFormats, exportv1.VirtualMachineExportVolumeFormat{
			Format: exportv1.Dir,
			Url:    fmt.Sprintf("https://%s.%s.svc/volumes/%s/dir", fmt.Sprintf("%s-%s", exportPrefix, exportName), namespace, volumeNames[1]),
		})
		exportVolumeFormats = append(exportVolumeFormats, exportv1.VirtualMachineExportVolumeFormat{
			Format: exportv1.ArchiveGz,
			Url:    fmt.Sprintf("https://%s.%s.svc/volumes/%s/disk.tar.gz", fmt.Sprintf("%s-%s", exportPrefix, exportName), namespace, volumeNames[1]),
		})
		verifyLinksInternal(vmExport, exportVolumeFormats...)
	}

	DescribeTable("Should create VM export, when VM is stopped", func(createVMFunc func() *virtv1.VirtualMachine, contentType1, contentType2 string, verifyFunc func(vmExport *exportv1.VirtualMachineExport, exportName, namespace string, volumeNames ...string)) {
		testVMExport := createVMVMExport()
		controller.VMInformer.GetStore().Add(createVMFunc())
		controller.PVCInformer.GetStore().Add(createPVC("volume1", contentType1))
		controller.PVCInformer.GetStore().Add(createPVC("volume2", contentType2))
		expectExporterCreate(k8sClient, k8sv1.PodRunning)
		vmExportClient.Fake.PrependReactor("update", "virtualmachineexports", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			update, ok := action.(testing.UpdateAction)
			Expect(ok).To(BeTrue())
			vmExport, ok := update.GetObject().(*exportv1.VirtualMachineExport)
			Expect(ok).To(BeTrue())
			verifyFunc(vmExport, vmExport.Name, testNamespace, "volume1", "volume2")
			for _, condition := range vmExport.Status.Conditions {
				if condition.Type == exportv1.ConditionReady {
					Expect(condition.Status).To(Equal(k8sv1.ConditionTrue))
					Expect(condition.Reason).To(Equal(podReadyReason))
				}
			}
			return true, vmExport, nil
		})
		retry, err := controller.updateVMExport(testVMExport)
		Expect(err).ToNot(HaveOccurred())
		Expect(retry).To(BeEquivalentTo(0))
		testutils.ExpectEvent(recorder, serviceCreatedEvent)
	},
		Entry("DataVolumes", createVMWithDataVolumes, "kubevirt", "kubevirt", verifyKubevirtInternal),
		Entry("PVCs", createVMWithPVCs, "kubevirt", "kubevirt", verifyKubevirtInternal),
		Entry("Memorydump and pvc", createVMWithPVCandMemoryDump, "kubevirt", "archive", verifyMixedInternal),
	)

	It("Should NOT create VM export, when VM is started", func() {
		testVMExport := createVMVMExport()
		controller.VMInformer.GetStore().Add(createVMWithDataVolumes())
		vmi := createVMIWithDataVolumes()
		controller.VMIInformer.GetStore().Add(vmi)
		controller.PVCInformer.GetStore().Add(createPVC("volume1", "kubevirt"))
		controller.PVCInformer.GetStore().Add(createPVC("volume2", "kubevirt"))
		vmExportClient.Fake.PrependReactor("update", "virtualmachineexports", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			update, ok := action.(testing.UpdateAction)
			Expect(ok).To(BeTrue())
			vmExport, ok := update.GetObject().(*exportv1.VirtualMachineExport)
			Expect(ok).To(BeTrue())
			verifyLinksEmpty(vmExport)
			for _, condition := range vmExport.Status.Conditions {
				if condition.Type == exportv1.ConditionReady {
					Expect(condition.Status).To(Equal(k8sv1.ConditionFalse))
					Expect(condition.Reason).To(Equal(inUseReason))
					Expect(condition.Message).To(Equal(fmt.Sprintf("Virtual Machine %s/%s is running", vmi.Namespace, vmi.Name)))
				}
			}
			return true, vmExport, nil
		})

		retry, err := controller.updateVMExport(testVMExport)
		Expect(err).ToNot(HaveOccurred())
		Expect(retry).To(BeEquivalentTo(0))
		testutils.ExpectEvent(recorder, serviceCreatedEvent)
	})

	createPopulatingDataVolume := func(name string) *cdiv1.DataVolume {
		return &cdiv1.DataVolume{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: testNamespace,
			},
			Status: cdiv1.DataVolumeStatus{
				Progress: "50%",
				Phase:    cdiv1.ImportInProgress,
			},
		}
	}

	It("Should NOT create VM export, when DV is not complete", func() {
		testVMExport := createVMVMExport()
		vm := createVMWithDataVolumes()
		controller.VMInformer.GetStore().Add(vm)
		dv := createPopulatingDataVolume("volume1")
		pvc1 := createPVC("volume1", "kubevirt")
		ownerRef := metav1.NewControllerRef(dv, datavolumeGVK)
		pvc1.GetObjectMeta().SetOwnerReferences([]metav1.OwnerReference{*ownerRef})
		controller.PVCInformer.GetStore().Add(pvc1)
		controller.PVCInformer.GetStore().Add(createPVC("volume2", "kubevirt"))
		controller.DataVolumeInformer.GetStore().Add(dv)
		vmExportClient.Fake.PrependReactor("update", "virtualmachineexports", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			update, ok := action.(testing.UpdateAction)
			Expect(ok).To(BeTrue())
			vmExport, ok := update.GetObject().(*exportv1.VirtualMachineExport)
			Expect(ok).To(BeTrue())
			verifyLinksEmpty(vmExport)
			for _, condition := range vmExport.Status.Conditions {
				if condition.Type == exportv1.ConditionReady {
					Expect(condition.Status).To(Equal(k8sv1.ConditionFalse))
					Expect(condition.Reason).To(Equal(inUseReason))
					Expect(condition.Message).To(Equal(fmt.Sprintf("Not all volumes in the Virtual Machine %s/%s are populated", vm.Namespace, vm.Name)))
				}
			}
			return true, vmExport, nil
		})

		retry, err := controller.updateVMExport(testVMExport)
		Expect(err).ToNot(HaveOccurred())
		Expect(retry).To(BeEquivalentTo(requeueTime))
		testutils.ExpectEvent(recorder, serviceCreatedEvent)
	})

	It("Should stop running VM export, when VM is started", func() {
		testVMExport := createVMVMExport()
		podName := controller.getExportPodName(testVMExport)
		controller.PodInformer.GetStore().Add(createExporterPod(podName, k8sv1.PodRunning))
		controller.VMInformer.GetStore().Add(createVMWithDataVolumes())
		vmi := createVMIWithDataVolumes()
		controller.VMIInformer.GetStore().Add(vmi)
		controller.PVCInformer.GetStore().Add(createPVC("volume1", "kubevirt"))
		controller.PVCInformer.GetStore().Add(createPVC("volume2", "kubevirt"))
		expectExporterDelete(k8sClient, podName)
		vmExportClient.Fake.PrependReactor("update", "virtualmachineexports", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			update, ok := action.(testing.UpdateAction)
			Expect(ok).To(BeTrue())
			vmExport, ok := update.GetObject().(*exportv1.VirtualMachineExport)
			Expect(ok).To(BeTrue())
			verifyLinksEmpty(vmExport)
			for _, condition := range vmExport.Status.Conditions {
				if condition.Type == exportv1.ConditionReady {
					Expect(condition.Status).To(Equal(k8sv1.ConditionFalse))
					Expect(condition.Reason).To(Equal(inUseReason))
					Expect(condition.Message).To(Equal(fmt.Sprintf("Virtual Machine %s/%s is running", vmi.Namespace, vmi.Name)), "%v", vmExport.Status.Conditions)
				}
			}
			return true, vmExport, nil
		})
		retry, err := controller.updateVMExport(testVMExport)
		Expect(err).ToNot(HaveOccurred())
		Expect(retry).To(BeEquivalentTo(0))
		testutils.ExpectEvent(recorder, serviceCreatedEvent)
		testutils.ExpectEvent(recorder, ExportPaused)
	})

	It("Should be in skipped phase when VM has no volumes", func() {
		testVMExport := createVMVMExport()
		controller.VMInformer.GetStore().Add(createVMWithoutVolumes())
		vmExportClient.Fake.PrependReactor("update", "virtualmachineexports", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			update, ok := action.(testing.UpdateAction)
			Expect(ok).To(BeTrue())
			vmExport, ok := update.GetObject().(*exportv1.VirtualMachineExport)
			Expect(ok).To(BeTrue())
			verifyLinksEmpty(vmExport)
			for _, condition := range vmExport.Status.Conditions {
				if condition.Type == exportv1.ConditionReady {
					Expect(condition.Status).To(Equal(k8sv1.ConditionFalse))
					Expect(condition.Reason).To(Equal(noVolumeVMReason))
				}
			}
			return true, vmExport, nil
		})
		retry, err := controller.updateVMExport(testVMExport)
		Expect(err).ToNot(HaveOccurred())
		Expect(retry).To(BeEquivalentTo(0))
	})

	It("Should handle failed exporter pod", func() {
		testVMExport := createVMVMExport()
		podName := controller.getExportPodName(testVMExport)
		controller.PodInformer.GetStore().Add(createExporterPod(podName, k8sv1.PodFailed))
		controller.VMInformer.GetStore().Add(createVMWithDataVolumes())
		controller.PVCInformer.GetStore().Add(createPVC("volume1", "kubevirt"))
		controller.PVCInformer.GetStore().Add(createPVC("volume2", "kubevirt"))
		vmExportClient.Fake.PrependReactor("update", "virtualmachineexports", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			update, ok := action.(testing.UpdateAction)
			Expect(ok).To(BeTrue())
			vmExport, ok := update.GetObject().(*exportv1.VirtualMachineExport)
			Expect(ok).To(BeTrue())
			verifyLinksEmpty(vmExport)
			for _, condition := range vmExport.Status.Conditions {
				if condition.Type == exportv1.ConditionReady {
					Expect(condition.Status).To(Equal(k8sv1.ConditionFalse))
					Expect(condition.Reason).To(Equal(unknownReason))
				}
			}
			return true, vmExport, nil
		})
		retry, err := controller.updateVMExport(testVMExport)
		Expect(err).ToNot(HaveOccurred())
		Expect(retry).To(BeEquivalentTo(0))
		testutils.ExpectEvent(recorder, serviceCreatedEvent)
		testutils.ExpectEvent(recorder, exporterPodFailedOrCompletedEvent)
	})
})
