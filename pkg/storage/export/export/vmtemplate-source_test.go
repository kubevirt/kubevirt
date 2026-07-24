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

package export

import (
	"context"
	"encoding/json"
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
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	backupv1 "kubevirt.io/api/backup/v1alpha1"
	virtv1 "kubevirt.io/api/core/v1"
	exportv1 "kubevirt.io/api/export/v1"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"
	snapshotv1 "kubevirt.io/api/snapshot/v1beta1"
	"kubevirt.io/client-go/kubecli"
	kubevirtfake "kubevirt.io/client-go/kubevirt/fake"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"
	templateapi "kubevirt.io/virt-template-api/core"
	"kubevirt.io/virt-template-api/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/certificates/bootstrap"
	virtcontroller "kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate/compute"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
)

const (
	testTemplateName = "test-template"
	sourcePVCName    = "source-pvc"
	dvtName          = "my-dvt"
)

var _ = Describe("VMTemplate source", func() {
	var (
		ctrl               *gomock.Controller
		controller         *VMExportController
		recorder           *record.FakeRecorder
		pvcInformer        cache.SharedIndexInformer
		vmExportInformer   cache.SharedIndexInformer
		dvInformer         cache.SharedIndexInformer
		vmTemplateInformer cache.SharedIndexInformer
		k8sClient          *k8sfake.Clientset
		vmExportClient     *kubevirtfake.Clientset
		mockVMExportQueue  *testutils.MockWorkQueue[string]
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		virtClient := kubecli.NewMockKubevirtClient(ctrl)
		pvcInformer, _ = testutils.NewFakeInformerFor(&k8sv1.PersistentVolumeClaim{})
		podInformer, _ := testutils.NewFakeInformerFor(&k8sv1.Pod{})
		cmInformer, _ := testutils.NewFakeInformerFor(&k8sv1.ConfigMap{})
		serviceInformer, _ := testutils.NewFakeInformerFor(&k8sv1.Service{})
		vmExportInformer, _ = testutils.NewFakeInformerWithIndexersFor(&exportv1.VirtualMachineExport{}, virtcontroller.GetVirtualMachineExportInformerIndexers())
		dvInformer, _ = testutils.NewFakeInformerFor(&cdiv1.DataVolume{})
		vmSnapshotInformer, _ := testutils.NewFakeInformerFor(&snapshotv1.VirtualMachineSnapshot{})
		vmSnapshotContentInformer, _ := testutils.NewFakeInformerFor(&snapshotv1.VirtualMachineSnapshotContent{})
		vmInformer, _ := testutils.NewFakeInformerFor(&virtv1.VirtualMachine{})
		vmiInformer, _ := testutils.NewFakeInformerFor(&virtv1.VirtualMachineInstance{})
		routeInformer, _ := testutils.NewFakeInformerFor(&routev1.Route{})
		ingressInformer, _ := testutils.NewFakeInformerFor(&networkingv1.Ingress{})
		secretInformer, _ := testutils.NewFakeInformerFor(&k8sv1.Secret{})
		kvInformer, _ := testutils.NewFakeInformerFor(&virtv1.KubeVirt{})
		crdInformer, _ := testutils.NewFakeInformerFor(&extv1.CustomResourceDefinition{})
		instancetypeInformer, _ := testutils.NewFakeInformerFor(&instancetypev1beta1.VirtualMachineInstancetype{})
		clusterInstancetypeInformer, _ := testutils.NewFakeInformerFor(&instancetypev1beta1.VirtualMachineClusterInstancetype{})
		preferenceInformer, _ := testutils.NewFakeInformerFor(&instancetypev1beta1.VirtualMachinePreference{})
		clusterPreferenceInformer, _ := testutils.NewFakeInformerFor(&instancetypev1beta1.VirtualMachineClusterPreference{})
		controllerRevisionInformer, _ := testutils.NewFakeInformerFor(&appsv1.ControllerRevision{})
		vmBackupInformer, _ := testutils.NewFakeInformerFor(&backupv1.VirtualMachineBackup{})
		vmTemplateInformer, _ = testutils.NewFakeInformerFor(&v1beta1.VirtualMachineTemplate{})
		rqInformer, _ := testutils.NewFakeInformerFor(&k8sv1.ResourceQuota{})
		nsInformer, _ := testutils.NewFakeInformerFor(&k8sv1.Namespace{})
		fakeVolumeSnapshotProvider := &MockVolumeSnapshotProvider{
			volumeSnapshots: []*vsv1.VolumeSnapshot{},
		}
		fakeCertManager, err := bootstrap.NewMockCertificateManager()
		Expect(err).ToNot(HaveOccurred())

		config, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&virtv1.KubeVirtConfiguration{
			DeveloperConfiguration: &virtv1.DeveloperConfiguration{
				FeatureGates: []string{compute.Template},
			},
		})
		k8sClient = k8sfake.NewSimpleClientset()
		vmExportClient = kubevirtfake.NewSimpleClientset()
		recorder = record.NewFakeRecorder(100)

		virtClient.EXPECT().CoreV1().Return(k8sClient.CoreV1()).AnyTimes()
		virtClient.EXPECT().VirtualMachineExport(testNamespace).
			Return(vmExportClient.ExportV1().VirtualMachineExports(testNamespace)).AnyTimes()

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
			RouteCache:                  routeInformer.GetStore(),
			IngressCache:                ingressInformer.GetStore(),
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
			VMTemplateInformer:          vmTemplateInformer,
		}
		initCert = func(ctrl *VMExportController) {
			ctrl.caCertManager.Start()
			Expect(ctrl.caCertManager.Current()).ToNot(BeNil())
		}

		Expect(controller.Init()).To(Succeed())
		controller.clusterConfig = config
		mockVMExportQueue = testutils.NewMockWorkQueue(controller.vmExportQueue)
		controller.vmExportQueue = mockVMExportQueue

		Expect(
			controller.ConfigMapInformer.GetStore().Add(&k8sv1.ConfigMap{
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
			controller.KubeVirtInformer.GetStore().Add(&virtv1.KubeVirt{
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

	newVMExport := func() *exportv1.VirtualMachineExport {
		apiGroup := templateapi.GroupName
		return &exportv1.VirtualMachineExport{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-export",
				Namespace: testNamespace,
			},
			Spec: exportv1.VirtualMachineExportSpec{
				Source: k8sv1.TypedLocalObjectReference{
					APIGroup: &apiGroup,
					Kind:     vmTemplateKind,
					Name:     testTemplateName,
				},
			},
			Status: &exportv1.VirtualMachineExportStatus{},
		}
	}

	newTemplate := func(vm *virtv1.VirtualMachine) *v1beta1.VirtualMachineTemplate {
		vmJSON, err := json.Marshal(vm)
		Expect(err).ToNot(HaveOccurred())
		return &v1beta1.VirtualMachineTemplate{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testTemplateName,
				Namespace: testNamespace,
			},
			Spec: v1beta1.VirtualMachineTemplateSpec{
				VirtualMachine: &runtime.RawExtension{Raw: vmJSON},
			},
			Status: v1beta1.VirtualMachineTemplateStatus{
				Conditions: []metav1.Condition{
					{Type: v1beta1.ConditionReady, Status: metav1.ConditionTrue},
				},
			},
		}
	}

	DescribeTable("Should NOT create VMTemplate export", func(setup func(), expectedReason string, expectedRetry time.Duration) {
		setup()
		testVMExport := newVMExport()
		vmExportClient.Fake.PrependReactor("update", "virtualmachineexports", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			update, ok := action.(testing.UpdateAction)
			Expect(ok).To(BeTrue())
			vmExport, ok := update.GetObject().(*exportv1.VirtualMachineExport)
			Expect(ok).To(BeTrue())
			for _, condition := range vmExport.Status.Conditions {
				if condition.Type == exportv1.ConditionReady {
					Expect(condition.Status).To(Equal(k8sv1.ConditionFalse))
					Expect(condition.Reason).To(Equal(expectedReason))
				}
			}
			return true, vmExport, nil
		})
		retry, err := controller.updateVMExport(testVMExport)
		Expect(err).ToNot(HaveOccurred())
		Expect(retry).To(BeEquivalentTo(expectedRetry))
	},
		Entry("when template does not exist", func() {}, vmTemplateNotFoundReason, time.Duration(0)),
		Entry("when template is not ready", func() {
			tpl := newTemplate(&virtv1.VirtualMachine{})
			tpl.Status.Conditions = nil
			Expect(vmTemplateInformer.GetStore().Add(tpl)).To(Succeed())
		}, vmTemplateNotReadyReason, requeueTime),
		Entry("when template has no volumes", func() {
			Expect(vmTemplateInformer.GetStore().Add(newTemplate(&virtv1.VirtualMachine{}))).To(Succeed())
		}, noVolumeVMReason, time.Duration(0)),
		Entry("when DVT source PVC does not exist", func() {
			Expect(vmTemplateInformer.GetStore().Add(newTemplate(&virtv1.VirtualMachine{
				Spec: virtv1.VirtualMachineSpec{
					DataVolumeTemplates: []virtv1.DataVolumeTemplateSpec{
						{
							ObjectMeta: metav1.ObjectMeta{Name: dvtName},
							Spec: cdiv1.DataVolumeSpec{
								Source: &cdiv1.DataVolumeSource{
									PVC: &cdiv1.DataVolumeSourcePVC{Name: "missing-pvc"},
								},
							},
						},
					},
				},
			}))).To(Succeed())
		}, volumesNotPopulatedReason, requeueTime),
	)

	It("Should create VMTemplate export with DVT source PVC", func() {
		testVMExport := newVMExport()
		_, err := vmExportClient.ExportV1().VirtualMachineExports(testNamespace).Create(
			context.Background(), testVMExport, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		Expect(vmTemplateInformer.GetStore().Add(newTemplate(&virtv1.VirtualMachine{
			Spec: virtv1.VirtualMachineSpec{
				DataVolumeTemplates: []virtv1.DataVolumeTemplateSpec{
					{
						ObjectMeta: metav1.ObjectMeta{Name: dvtName},
						Spec: cdiv1.DataVolumeSpec{
							Source: &cdiv1.DataVolumeSource{
								PVC: &cdiv1.DataVolumeSourcePVC{Name: sourcePVCName},
							},
						},
					},
				},
			},
		}))).To(Succeed())
		Expect(pvcInformer.GetStore().Add(createPVC(sourcePVCName, "kubevirt"))).To(Succeed())
		expectExporterCreate(k8sClient, k8sv1.PodRunning)

		vmExportClient.Fake.PrependReactor("update", "virtualmachineexports", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			update, ok := action.(testing.UpdateAction)
			Expect(ok).To(BeTrue())
			vmExport, ok := update.GetObject().(*exportv1.VirtualMachineExport)
			Expect(ok).To(BeTrue())
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
	})

	It("Should return false for isSourceVMTemplate when VMTemplateInformer is nil", func() {
		saved := controller.VMTemplateInformer
		controller.VMTemplateInformer = nil
		defer func() { controller.VMTemplateInformer = saved }()

		apiGroup := templateapi.GroupName
		spec := &exportv1.VirtualMachineExportSpec{
			Source: k8sv1.TypedLocalObjectReference{
				APIGroup: &apiGroup,
				Kind:     vmTemplateKind,
				Name:     "test",
			},
		}
		Expect(controller.isSourceVMTemplate(spec)).To(BeFalse())
	})
})

var _ = Describe("VMTemplate ManifestData", func() {
	It("should strip cluster fields", func() {
		tpl := &v1beta1.VirtualMachineTemplate{
			ObjectMeta: metav1.ObjectMeta{
				Name:            "my-template",
				Namespace:       "some-ns",
				UID:             "uid-123",
				ResourceVersion: "99",
				Generation:      5,
				ManagedFields:   []metav1.ManagedFieldsEntry{{Manager: "m"}},
				Labels:          map[string]string{"app": "test"},
				Annotations:     map[string]string{"note": "val"},
			},
			Spec: v1beta1.VirtualMachineTemplateSpec{
				VirtualMachine: &runtime.RawExtension{Raw: []byte(`{"spec":{}}`)},
			},
			Status: v1beta1.VirtualMachineTemplateStatus{
				Conditions: []metav1.Condition{{Type: v1beta1.ConditionReady, Status: metav1.ConditionTrue}},
			},
		}

		tplSource := NewVMTemplateSource(tpl, &sourceVolumes{})
		key, result, extra, err := tplSource.ManifestData()
		Expect(err).ToNot(HaveOccurred())
		Expect(key).To(Equal(vmTemplateManifest))
		Expect(extra).To(BeNil())

		var out v1beta1.VirtualMachineTemplate
		Expect(json.Unmarshal(result, &out)).To(Succeed())
		Expect(out.Name).To(Equal("my-template"))
		Expect(out.Namespace).To(BeEmpty())
		Expect(out.Labels).To(HaveKeyWithValue("app", "test"))
		Expect(out.Annotations).To(HaveKeyWithValue("note", "val"))
		Expect(string(out.UID)).To(BeEmpty())
		Expect(out.ResourceVersion).To(BeEmpty())
		Expect(out.Generation).To(BeZero())
		Expect(out.ManagedFields).To(BeNil())
		Expect(out.Status).To(Equal(v1beta1.VirtualMachineTemplateStatus{}))
	})

	It("should preserve parameters and message", func() {
		vmJSON, err := json.Marshal(&virtv1.VirtualMachine{
			Spec: virtv1.VirtualMachineSpec{
				Template: &virtv1.VirtualMachineInstanceTemplateSpec{
					Spec: virtv1.VirtualMachineInstanceSpec{Architecture: "amd64"},
				},
			},
		})
		Expect(err).ToNot(HaveOccurred())

		tpl := &v1beta1.VirtualMachineTemplate{
			ObjectMeta: metav1.ObjectMeta{Name: "test"},
			Spec: v1beta1.VirtualMachineTemplateSpec{
				VirtualMachine: &runtime.RawExtension{Raw: vmJSON},
				Parameters: []v1beta1.Parameter{
					{Name: "VM_NAME", Value: "my-vm"},
				},
				Message: "test message",
			},
		}

		tplSource := NewVMTemplateSource(tpl, &sourceVolumes{})
		_, result, _, err := tplSource.ManifestData()
		Expect(err).ToNot(HaveOccurred())

		var out v1beta1.VirtualMachineTemplate
		Expect(json.Unmarshal(result, &out)).To(Succeed())
		Expect(out.Spec.Parameters).To(HaveLen(1))
		Expect(out.Spec.Parameters[0].Name).To(Equal("VM_NAME"))
		Expect(out.Spec.Message).To(Equal("test message"))
	})

	It("should preserve the embedded VirtualMachine RawExtension", func() {
		vmJSON, err := json.Marshal(&virtv1.VirtualMachine{
			Spec: virtv1.VirtualMachineSpec{
				Template: &virtv1.VirtualMachineInstanceTemplateSpec{
					Spec: virtv1.VirtualMachineInstanceSpec{Architecture: "amd64"},
				},
			},
		})
		Expect(err).ToNot(HaveOccurred())

		tpl := &v1beta1.VirtualMachineTemplate{
			ObjectMeta: metav1.ObjectMeta{Name: "test"},
			Spec: v1beta1.VirtualMachineTemplateSpec{
				VirtualMachine: &runtime.RawExtension{Raw: vmJSON},
			},
		}

		tplSource := NewVMTemplateSource(tpl, &sourceVolumes{})
		_, result, _, err := tplSource.ManifestData()
		Expect(err).ToNot(HaveOccurred())

		var out v1beta1.VirtualMachineTemplate
		Expect(json.Unmarshal(result, &out)).To(Succeed())
		Expect(out.Spec.VirtualMachine).ToNot(BeNil())
		Expect(out.Spec.VirtualMachine.Raw).ToNot(BeEmpty())
	})

	It("should handle nil VirtualMachine in template", func() {
		tpl := &v1beta1.VirtualMachineTemplate{
			ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: testNamespace},
			Spec:       v1beta1.VirtualMachineTemplateSpec{},
		}

		tplSource := NewVMTemplateSource(tpl, &sourceVolumes{})
		_, result, _, err := tplSource.ManifestData()
		Expect(err).ToNot(HaveOccurred())

		var out v1beta1.VirtualMachineTemplate
		Expect(json.Unmarshal(result, &out)).To(Succeed())
		Expect(out.Spec.VirtualMachine).To(BeNil())
	})

	It("should return empty for nil template", func() {
		tplSource := NewVMTemplateSource(nil, &sourceVolumes{})
		key, result, _, err := tplSource.ManifestData()
		Expect(err).ToNot(HaveOccurred())
		Expect(key).To(BeEmpty())
		Expect(result).To(BeNil())
	})
})

var _ = Describe("VMTemplate source helpers", func() {
	marshalVM := func(vm *virtv1.VirtualMachine) map[string]any {
		data, err := json.Marshal(vm)
		Expect(err).ToNot(HaveOccurred())
		var obj map[string]any
		Expect(json.Unmarshal(data, &obj)).To(Succeed())
		return obj
	}

	Context("ResolveParameterValue", func() {
		It("should resolve parameter placeholders", func() {
			params := []v1beta1.Parameter{
				{Name: "VM_NAME", Value: "my-vm"},
			}
			resolved, ok := ResolveParameterValue("${VM_NAME}-rootdisk", params)
			Expect(ok).To(BeTrue())
			Expect(resolved).To(Equal("my-vm-rootdisk"))
		})

		It("should skip parameters with empty values", func() {
			params := []v1beta1.Parameter{
				{Name: "VM_NAME", Value: ""},
			}
			resolved, ok := ResolveParameterValue("${VM_NAME}-rootdisk", params)
			Expect(ok).To(BeFalse())
			Expect(resolved).To(Equal("${VM_NAME}-rootdisk"))
		})

		It("should return false for unresolvable placeholders", func() {
			_, ok := ResolveParameterValue("${UNKNOWN}-disk", nil)
			Expect(ok).To(BeFalse())
		})

		It("should return true when no placeholders present", func() {
			resolved, ok := ResolveParameterValue("plain-name", nil)
			Expect(ok).To(BeTrue())
			Expect(resolved).To(Equal("plain-name"))
		})
	})

	Context("extractLocalDVTPVCNames", func() {
		DescribeTable("should handle PVC source namespace", func(srcNamespace string, expectFound bool) {
			obj := marshalVM(&virtv1.VirtualMachine{
				Spec: virtv1.VirtualMachineSpec{
					DataVolumeTemplates: []virtv1.DataVolumeTemplateSpec{
						{
							ObjectMeta: metav1.ObjectMeta{Name: dvtName},
							Spec: cdiv1.DataVolumeSpec{
								Source: &cdiv1.DataVolumeSource{
									PVC: &cdiv1.DataVolumeSourcePVC{Name: sourcePVCName, Namespace: srcNamespace},
								},
							},
						},
					},
				},
			})
			pvcNames, dvtNames := extractLocalDVTPVCNames(obj, nil, testNamespace)
			if expectFound {
				Expect(pvcNames).To(ConsistOf(sourcePVCName))
				Expect(dvtNames).To(ConsistOf(dvtName))
			} else {
				Expect(pvcNames).To(BeEmpty())
				Expect(dvtNames).To(BeEmpty())
			}
		},
			Entry("same namespace", testNamespace, true),
			Entry("empty namespace (local)", "", true),
			Entry("cross-namespace", "other-ns", false),
		)

		It("should skip DVTs without PVC source", func() {
			obj := marshalVM(&virtv1.VirtualMachine{
				Spec: virtv1.VirtualMachineSpec{
					DataVolumeTemplates: []virtv1.DataVolumeTemplateSpec{
						{
							ObjectMeta: metav1.ObjectMeta{Name: dvtName},
							Spec: cdiv1.DataVolumeSpec{
								Source: &cdiv1.DataVolumeSource{
									HTTP: &cdiv1.DataVolumeSourceHTTP{URL: "https://example.com/disk.img"},
								},
							},
						},
					},
				},
			})
			pvcNames, _ := extractLocalDVTPVCNames(obj, nil, testNamespace)
			Expect(pvcNames).To(BeEmpty())
		})

		It("should skip DVTs with empty metadata name", func() {
			obj := marshalVM(&virtv1.VirtualMachine{
				Spec: virtv1.VirtualMachineSpec{
					DataVolumeTemplates: []virtv1.DataVolumeTemplateSpec{
						{
							Spec: cdiv1.DataVolumeSpec{
								Source: &cdiv1.DataVolumeSource{
									PVC: &cdiv1.DataVolumeSourcePVC{Name: sourcePVCName},
								},
							},
						},
					},
				},
			})
			pvcNames, dvtNames := extractLocalDVTPVCNames(obj, nil, testNamespace)
			Expect(pvcNames).To(BeEmpty())
			Expect(dvtNames).To(BeEmpty())
		})
	})

	Context("rewriteEmbeddedVM", func() {
		It("should rewrite DataVolume volume sources to PVC sources", func() {
			vm := &virtv1.VirtualMachine{
				Spec: virtv1.VirtualMachineSpec{
					Template: &virtv1.VirtualMachineInstanceTemplateSpec{
						Spec: virtv1.VirtualMachineInstanceSpec{
							Volumes: []virtv1.Volume{
								{
									Name: "rootdisk",
									VolumeSource: virtv1.VolumeSource{
										DataVolume: &virtv1.DataVolumeSource{Name: "rootdisk-dv"},
									},
								},
							},
						},
					},
				},
			}
			vmJSON, err := json.Marshal(vm)
			Expect(err).ToNot(HaveOccurred())

			spec := &v1beta1.VirtualMachineTemplateSpec{
				VirtualMachine: &runtime.RawExtension{Raw: vmJSON},
			}

			rewritten, err := rewriteEmbeddedVM(spec, testNamespace)
			Expect(err).ToNot(HaveOccurred())

			var outVM virtv1.VirtualMachine
			Expect(json.Unmarshal(rewritten.Raw, &outVM)).To(Succeed())
			Expect(outVM.Spec.Template.Spec.Volumes).To(HaveLen(1))
			vol := outVM.Spec.Template.Spec.Volumes[0]
			Expect(vol.DataVolume).To(BeNil())
			Expect(vol.PersistentVolumeClaim).ToNot(BeNil())
			Expect(vol.PersistentVolumeClaim.ClaimName).To(Equal("rootdisk-dv"))
		})

		It("should not rewrite DataVolume volume that references a DVT", func() {
			vm := &virtv1.VirtualMachine{
				Spec: virtv1.VirtualMachineSpec{
					DataVolumeTemplates: []virtv1.DataVolumeTemplateSpec{
						{
							ObjectMeta: metav1.ObjectMeta{Name: dvtName},
							Spec: cdiv1.DataVolumeSpec{
								Source: &cdiv1.DataVolumeSource{
									PVC: &cdiv1.DataVolumeSourcePVC{
										Name: sourcePVCName,
									},
								},
							},
						},
					},
					Template: &virtv1.VirtualMachineInstanceTemplateSpec{
						Spec: virtv1.VirtualMachineInstanceSpec{
							Volumes: []virtv1.Volume{
								{
									Name: "rootdisk",
									VolumeSource: virtv1.VolumeSource{
										DataVolume: &virtv1.DataVolumeSource{Name: dvtName},
									},
								},
							},
						},
					},
				},
			}
			vmJSON, err := json.Marshal(vm)
			Expect(err).ToNot(HaveOccurred())

			spec := &v1beta1.VirtualMachineTemplateSpec{
				VirtualMachine: &runtime.RawExtension{Raw: vmJSON},
			}

			rewritten, err := rewriteEmbeddedVM(spec, testNamespace)
			Expect(err).ToNot(HaveOccurred())

			var outVM virtv1.VirtualMachine
			Expect(json.Unmarshal(rewritten.Raw, &outVM)).To(Succeed())
			Expect(outVM.Spec.Template.Spec.Volumes).To(HaveLen(1))
			vol := outVM.Spec.Template.Spec.Volumes[0]
			Expect(vol.DataVolume).ToNot(BeNil(), "DataVolume volume referencing a DVT should not be rewritten")
			Expect(vol.DataVolume.Name).To(Equal(dvtName))
			Expect(vol.PersistentVolumeClaim).To(BeNil())
		})

		It("should rewrite DVT with local PVC source to reference exported PVC", func() {
			vm := &virtv1.VirtualMachine{
				Spec: virtv1.VirtualMachineSpec{
					DataVolumeTemplates: []virtv1.DataVolumeTemplateSpec{
						{
							ObjectMeta: metav1.ObjectMeta{Name: "my-dv"},
							Spec: cdiv1.DataVolumeSpec{
								Source: &cdiv1.DataVolumeSource{
									PVC: &cdiv1.DataVolumeSourcePVC{
										Name:      sourcePVCName,
										Namespace: testNamespace,
									},
								},
							},
						},
					},
				},
			}
			vmJSON, err := json.Marshal(vm)
			Expect(err).ToNot(HaveOccurred())

			spec := &v1beta1.VirtualMachineTemplateSpec{
				VirtualMachine: &runtime.RawExtension{Raw: vmJSON},
			}

			rewritten, err := rewriteEmbeddedVM(spec, testNamespace)
			Expect(err).ToNot(HaveOccurred())

			var outVM virtv1.VirtualMachine
			Expect(json.Unmarshal(rewritten.Raw, &outVM)).To(Succeed())
			Expect(outVM.Spec.DataVolumeTemplates).To(HaveLen(1))
			dvt := outVM.Spec.DataVolumeTemplates[0]
			Expect(dvt.Spec.Source.PVC).ToNot(BeNil())
			Expect(dvt.Spec.Source.PVC.Name).To(Equal(sourcePVCName))
			Expect(dvt.Spec.SourceRef).To(BeNil())
		})

		It("should not rewrite DVT with cross-namespace PVC source", func() {
			vm := &virtv1.VirtualMachine{
				Spec: virtv1.VirtualMachineSpec{
					DataVolumeTemplates: []virtv1.DataVolumeTemplateSpec{
						{
							ObjectMeta: metav1.ObjectMeta{Name: "my-dv"},
							Spec: cdiv1.DataVolumeSpec{
								Source: &cdiv1.DataVolumeSource{
									PVC: &cdiv1.DataVolumeSourcePVC{
										Name:      sourcePVCName,
										Namespace: "other-ns",
									},
								},
							},
						},
					},
				},
			}
			vmJSON, err := json.Marshal(vm)
			Expect(err).ToNot(HaveOccurred())

			spec := &v1beta1.VirtualMachineTemplateSpec{
				VirtualMachine: &runtime.RawExtension{Raw: vmJSON},
			}

			rewritten, err := rewriteEmbeddedVM(spec, testNamespace)
			Expect(err).ToNot(HaveOccurred())

			var outVM virtv1.VirtualMachine
			Expect(json.Unmarshal(rewritten.Raw, &outVM)).To(Succeed())
			Expect(outVM.Spec.DataVolumeTemplates).To(HaveLen(1))
			dvt := outVM.Spec.DataVolumeTemplates[0]
			Expect(dvt.Spec.Source.PVC.Name).To(Equal(sourcePVCName))
			Expect(dvt.Spec.Source.PVC.Namespace).To(Equal("other-ns"))
		})

		It("should not rewrite DVT without PVC source", func() {
			vm := &virtv1.VirtualMachine{
				Spec: virtv1.VirtualMachineSpec{
					DataVolumeTemplates: []virtv1.DataVolumeTemplateSpec{
						{
							ObjectMeta: metav1.ObjectMeta{Name: "http-dv"},
							Spec: cdiv1.DataVolumeSpec{
								Source: &cdiv1.DataVolumeSource{
									HTTP: &cdiv1.DataVolumeSourceHTTP{
										URL: "https://example.com/disk.img",
									},
								},
							},
						},
					},
				},
			}
			vmJSON, err := json.Marshal(vm)
			Expect(err).ToNot(HaveOccurred())

			spec := &v1beta1.VirtualMachineTemplateSpec{
				VirtualMachine: &runtime.RawExtension{Raw: vmJSON},
			}

			rewritten, err := rewriteEmbeddedVM(spec, testNamespace)
			Expect(err).ToNot(HaveOccurred())

			var outVM virtv1.VirtualMachine
			Expect(json.Unmarshal(rewritten.Raw, &outVM)).To(Succeed())
			Expect(outVM.Spec.DataVolumeTemplates).To(HaveLen(1))
			Expect(outVM.Spec.DataVolumeTemplates[0].Spec.Source.HTTP).ToNot(BeNil())
			Expect(outVM.Spec.DataVolumeTemplates[0].Spec.Source.HTTP.URL).To(Equal("https://example.com/disk.img"))
		})

		It("should keep elements with unresolvable placeholders as-is", func() {
			vm := &virtv1.VirtualMachine{
				Spec: virtv1.VirtualMachineSpec{
					DataVolumeTemplates: []virtv1.DataVolumeTemplateSpec{
						{
							ObjectMeta: metav1.ObjectMeta{Name: "${PARAM}"},
							Spec: cdiv1.DataVolumeSpec{
								Source: &cdiv1.DataVolumeSource{
									PVC: &cdiv1.DataVolumeSourcePVC{Name: "src"},
								},
							},
						},
					},
					Template: &virtv1.VirtualMachineInstanceTemplateSpec{
						Spec: virtv1.VirtualMachineInstanceSpec{
							Volumes: []virtv1.Volume{
								{
									Name: "${VOL}",
									VolumeSource: virtv1.VolumeSource{
										DataVolume: &virtv1.DataVolumeSource{Name: "${DV}"},
									},
								},
							},
						},
					},
				},
			}
			vmJSON, err := json.Marshal(vm)
			Expect(err).ToNot(HaveOccurred())

			spec := &v1beta1.VirtualMachineTemplateSpec{
				VirtualMachine: &runtime.RawExtension{Raw: vmJSON},
			}

			rewritten, err := rewriteEmbeddedVM(spec, testNamespace)
			Expect(err).ToNot(HaveOccurred())

			var outObj map[string]any
			Expect(json.Unmarshal(rewritten.Raw, &outObj)).To(Succeed())

			dvts, found, err := unstructured.NestedSlice(outObj, "spec", "dataVolumeTemplates")
			Expect(err).ToNot(HaveOccurred())
			Expect(found).To(BeTrue())
			Expect(dvts).To(HaveLen(1))
			dvtMap := dvts[0].(map[string]any)
			srcName, found, err := unstructured.NestedString(dvtMap, "spec", "source", "pvc", "name")
			Expect(err).ToNot(HaveOccurred())
			Expect(found).To(BeTrue())
			Expect(srcName).To(Equal("src"), "DVT source PVC should be unchanged when DVT name has unresolvable placeholder")

			volumes, found, err := unstructured.NestedSlice(outObj, "spec", "template", "spec", "volumes")
			Expect(err).ToNot(HaveOccurred())
			Expect(found).To(BeTrue())
			Expect(volumes).To(HaveLen(1))
			volMap := volumes[0].(map[string]any)
			dvName, found, err := unstructured.NestedString(volMap, "dataVolume", "name")
			Expect(err).ToNot(HaveOccurred())
			Expect(found).To(BeTrue())
			Expect(dvName).To(Equal("${DV}"), "DataVolume volume should be unchanged when name has unresolvable placeholder")
			_, hasPVC, err := unstructured.NestedMap(volMap, "persistentVolumeClaim")
			Expect(err).ToNot(HaveOccurred())
			Expect(hasPVC).To(BeFalse(), "volume should not be rewritten to PVC")
		})

		It("should resolve parameter placeholders in DVT and volume rewriting", func() {
			vm := &virtv1.VirtualMachine{
				Spec: virtv1.VirtualMachineSpec{
					DataVolumeTemplates: []virtv1.DataVolumeTemplateSpec{
						{
							ObjectMeta: metav1.ObjectMeta{Name: dvtName},
							Spec: cdiv1.DataVolumeSpec{
								Source: &cdiv1.DataVolumeSource{
									PVC: &cdiv1.DataVolumeSourcePVC{Name: "${SRC_PVC}"},
								},
							},
						},
					},
					Template: &virtv1.VirtualMachineInstanceTemplateSpec{
						Spec: virtv1.VirtualMachineInstanceSpec{
							Volumes: []virtv1.Volume{
								{
									Name: "standalone",
									VolumeSource: virtv1.VolumeSource{
										DataVolume: &virtv1.DataVolumeSource{Name: "${DV_NAME}"},
									},
								},
							},
						},
					},
				},
			}
			vmJSON, err := json.Marshal(vm)
			Expect(err).ToNot(HaveOccurred())

			spec := &v1beta1.VirtualMachineTemplateSpec{
				VirtualMachine: &runtime.RawExtension{Raw: vmJSON},
				Parameters: []v1beta1.Parameter{
					{Name: "SRC_PVC", Value: "resolved-src"},
					{Name: "DV_NAME", Value: "resolved-dv"},
				},
			}

			rewritten, err := rewriteEmbeddedVM(spec, testNamespace)
			Expect(err).ToNot(HaveOccurred())

			var outVM virtv1.VirtualMachine
			Expect(json.Unmarshal(rewritten.Raw, &outVM)).To(Succeed())

			Expect(outVM.Spec.DataVolumeTemplates).To(HaveLen(1))
			Expect(outVM.Spec.DataVolumeTemplates[0].Spec.Source.PVC.Name).To(Equal("resolved-src"))

			Expect(outVM.Spec.Template.Spec.Volumes).To(HaveLen(1))
			vol := outVM.Spec.Template.Spec.Volumes[0]
			Expect(vol.PersistentVolumeClaim).ToNot(BeNil())
			Expect(vol.PersistentVolumeClaim.ClaimName).To(Equal("resolved-dv"))
		})
	})

	Context("extractVolumePVCNames", func() {
		It("should extract PVC claim names", func() {
			obj := marshalVM(&virtv1.VirtualMachine{
				Spec: virtv1.VirtualMachineSpec{
					Template: &virtv1.VirtualMachineInstanceTemplateSpec{
						Spec: virtv1.VirtualMachineInstanceSpec{
							Volumes: []virtv1.Volume{
								{
									Name: "vol1",
									VolumeSource: virtv1.VolumeSource{
										PersistentVolumeClaim: &virtv1.PersistentVolumeClaimVolumeSource{
											PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
												ClaimName: "my-pvc",
											},
										},
									},
								},
							},
						},
					},
				},
			})
			names := extractVolumePVCNames(obj, nil)
			Expect(names).To(HaveKeyWithValue("my-pvc", "my-pvc"))
		})

		It("should extract DataVolume names as PVC names", func() {
			obj := marshalVM(&virtv1.VirtualMachine{
				Spec: virtv1.VirtualMachineSpec{
					Template: &virtv1.VirtualMachineInstanceTemplateSpec{
						Spec: virtv1.VirtualMachineInstanceSpec{
							Volumes: []virtv1.Volume{
								{
									Name: "vol1",
									VolumeSource: virtv1.VolumeSource{
										DataVolume: &virtv1.DataVolumeSource{Name: "my-dv"},
									},
								},
							},
						},
					},
				},
			})
			names := extractVolumePVCNames(obj, nil)
			Expect(names).To(HaveKeyWithValue("my-dv", "my-dv"))
		})

		It("should skip volumes without PVC or DataVolume", func() {
			obj := marshalVM(&virtv1.VirtualMachine{
				Spec: virtv1.VirtualMachineSpec{
					Template: &virtv1.VirtualMachineInstanceTemplateSpec{
						Spec: virtv1.VirtualMachineInstanceSpec{
							Volumes: []virtv1.Volume{
								{
									Name: "vol1",
									VolumeSource: virtv1.VolumeSource{
										CloudInitNoCloud: &virtv1.CloudInitNoCloudSource{UserData: "#cloud-config"},
									},
								},
							},
						},
					},
				},
			})
			names := extractVolumePVCNames(obj, nil)
			Expect(names).To(BeEmpty())
		})
	})
})
