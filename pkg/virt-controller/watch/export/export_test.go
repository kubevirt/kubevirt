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
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	routev1 "github.com/openshift/api/route/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/pointer"

	k8sv1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	vsv1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	virtv1 "kubevirt.io/api/core/v1"
	kubevirtfake "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/fake"

	exportv1 "kubevirt.io/api/export/v1alpha1"
	"kubevirt.io/client-go/kubecli"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	snapshotv1 "kubevirt.io/api/snapshot/v1alpha1"

	"kubevirt.io/kubevirt/pkg/certificates/bootstrap"
	"kubevirt.io/kubevirt/pkg/certificates/triple"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"

	certutil "kubevirt.io/kubevirt/pkg/certificates/triple/cert"
)

const (
	testNamespace          = "default"
	testVmsnapshotName     = "test-vmsnapshot"
	testVolumesnapshotName = "test-snapshot"
)

var (
	qemuGid int64 = 107
)

var _ = Describe("Export controlleer", func() {
	var (
		ctrl                       *gomock.Controller
		controller                 *VMExportController
		recorder                   *record.FakeRecorder
		pvcInformer                cache.SharedIndexInformer
		podInformer                cache.SharedIndexInformer
		cmInformer                 cache.SharedIndexInformer
		vmExportInformer           cache.SharedIndexInformer
		serviceInformer            cache.SharedIndexInformer
		dvInformer                 cache.SharedIndexInformer
		vmSnapshotInformer         cache.SharedIndexInformer
		vmSnapshotContentInformer  cache.SharedIndexInformer
		k8sClient                  *k8sfake.Clientset
		vmExportClient             *kubevirtfake.Clientset
		fakeVolumeSnapshotProvider *MockVolumeSnapshotProvider
		routeCache                 cache.Store
		ingressCache               cache.Store
		certDir                    string
		certFilePath               string
		keyFilePath                string
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		var err error
		certDir, err = ioutil.TempDir("", "certs")
		Expect(err).ToNot(HaveOccurred())
		certFilePath = filepath.Join(certDir, "tls.crt")
		keyFilePath = filepath.Join(certDir, "tls.key")
		writeCertsToDir(certDir)
		virtClient := kubecli.NewMockKubevirtClient(ctrl)
		pvcInformer, _ = testutils.NewFakeInformerFor(&k8sv1.PersistentVolumeClaim{})
		podInformer, _ = testutils.NewFakeInformerFor(&k8sv1.Pod{})
		cmInformer, _ = testutils.NewFakeInformerFor(&k8sv1.ConfigMap{})
		serviceInformer, _ = testutils.NewFakeInformerFor(&k8sv1.Service{})
		vmExportInformer, _ = testutils.NewFakeInformerFor(&exportv1.VirtualMachineExport{})
		dvInformer, _ = testutils.NewFakeInformerFor(&cdiv1.DataVolume{})
		vmSnapshotInformer, _ = testutils.NewFakeInformerFor(&snapshotv1.VirtualMachineSnapshot{})
		vmSnapshotContentInformer, _ = testutils.NewFakeInformerFor(&snapshotv1.VirtualMachineSnapshotContent{})
		routeInformer, _ := testutils.NewFakeInformerFor(&routev1.Route{})
		routeCache = routeInformer.GetStore()
		ingressInformer, _ := testutils.NewFakeInformerFor(&networkingv1.Ingress{})
		ingressCache = ingressInformer.GetStore()
		secretInformer, _ := testutils.NewFakeInformerFor(&k8sv1.Secret{})
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
			Client:                    virtClient,
			Recorder:                  recorder,
			PVCInformer:               pvcInformer,
			PodInformer:               podInformer,
			ConfigMapInformer:         cmInformer,
			VMExportInformer:          vmExportInformer,
			ServiceInformer:           serviceInformer,
			DataVolumeInformer:        dvInformer,
			KubevirtNamespace:         "kubevirt",
			TemplateService:           services.NewTemplateService("a", 240, "b", "c", "d", "e", "f", "g", pvcInformer.GetStore(), virtClient, config, qemuGid, "h"),
			caCertManager:             bootstrap.NewFileCertificateManager(certFilePath, keyFilePath),
			RouteCache:                routeCache,
			IngressCache:              ingressCache,
			RouteConfigMapInformer:    cmInformer,
			SecretInformer:            secretInformer,
			VMSnapshotInformer:        vmSnapshotInformer,
			VMSnapshotContentInformer: vmSnapshotContentInformer,
			VolumeSnapshotProvider:    fakeVolumeSnapshotProvider,
		}
		// Wrap our workqueue to have a way to detect when we are done processing updates
		mockVMExportQueue := testutils.NewMockWorkQueue(controller.vmExportQueue)
		controller.vmExportQueue = mockVMExportQueue

		go controller.caCertManager.Start()
		// Give the thread time to read the certs.
		Eventually(func() *tls.Certificate {
			return controller.caCertManager.Current()
		}, time.Second, time.Millisecond).ShouldNot(BeNil())
		cmInformer.GetStore().Add(&k8sv1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: controller.KubevirtNamespace,
				Name:      components.KubeVirtExportCASecretName,
			},
			Data: map[string]string{
				"ca-bundle": "replace me with ca cert",
			},
		})
	})

	AfterEach(func() {
		controller.caCertManager.Stop()
		os.RemoveAll(certDir)
	})

	generateExpectedCert := func() string {
		key, err := certutil.NewPrivateKey()
		Expect(err).ToNot(HaveOccurred())

		config := certutil.Config{
			CommonName: "blah blah",
		}

		cert, err := certutil.NewSelfSignedCACertWithAltNames(config, key, time.Hour, "hahaha.wwoo", "*.apps-crc.testing", "fgdgd.dfsgdf")
		Expect(err).ToNot(HaveOccurred())
		pemOut := strings.Builder{}
		pem.Encode(&pemOut, &pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})
		return strings.TrimSpace(pemOut.String())
	}

	var expectedPem = generateExpectedCert()

	generateRouteCert := func() string {
		key, err := certutil.NewPrivateKey()
		Expect(err).ToNot(HaveOccurred())

		config := certutil.Config{
			CommonName: "something else",
		}

		cert, err := certutil.NewSelfSignedCACert(config, key, time.Hour)
		Expect(err).ToNot(HaveOccurred())
		pemOut := strings.Builder{}
		pem.Encode(&pemOut, &pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})
		return strings.TrimSpace(pemOut.String()) + "\n" + expectedPem
	}

	createRouteConfigMap := func() *k8sv1.ConfigMap {
		return &k8sv1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      routeCAConfigMapName,
				Namespace: controller.KubevirtNamespace,
			},
			Data: map[string]string{
				routeCaKey: generateRouteCert(),
			},
		}
	}

	validIngressDefaultBackend := func(serviceName string) *networkingv1.Ingress {
		return &networkingv1.Ingress{
			Spec: networkingv1.IngressSpec{
				DefaultBackend: &networkingv1.IngressBackend{
					Service: &networkingv1.IngressServiceBackend{
						Name: serviceName,
					},
				},
				Rules: []networkingv1.IngressRule{
					{
						Host: "backend-host",
					},
				},
			},
		}
	}

	ingressDefaultBackendNoService := func() *networkingv1.Ingress {
		return &networkingv1.Ingress{
			Spec: networkingv1.IngressSpec{
				DefaultBackend: &networkingv1.IngressBackend{},
				Rules: []networkingv1.IngressRule{
					{
						Host: "backend-host",
					},
				},
			},
		}
	}

	validIngressRules := func(serviceName string) *networkingv1.Ingress {
		return &networkingv1.Ingress{
			Spec: networkingv1.IngressSpec{
				Rules: []networkingv1.IngressRule{
					{
						IngressRuleValue: networkingv1.IngressRuleValue{
							HTTP: &networkingv1.HTTPIngressRuleValue{
								Paths: []networkingv1.HTTPIngressPath{
									{
										Backend: networkingv1.IngressBackend{
											Service: &networkingv1.IngressServiceBackend{
												Name: serviceName,
											},
										},
									},
								},
							},
						},
						Host: "rule-host",
					},
				},
			},
		}
	}

	ingressRulesNoBackend := func() *networkingv1.Ingress {
		return &networkingv1.Ingress{
			Spec: networkingv1.IngressSpec{
				Rules: []networkingv1.IngressRule{
					{
						IngressRuleValue: networkingv1.IngressRuleValue{
							HTTP: &networkingv1.HTTPIngressRuleValue{
								Paths: []networkingv1.HTTPIngressPath{
									{
										Backend: networkingv1.IngressBackend{},
									},
								},
							},
						},
						Host: "rule-host",
					},
				},
			},
		}
	}

	routeToHostAndService := func(serviceName string) *routev1.Route {
		return &routev1.Route{
			Spec: routev1.RouteSpec{
				To: routev1.RouteTargetReference{
					Name: serviceName,
				},
			},
			Status: routev1.RouteStatus{
				Ingress: []routev1.RouteIngress{
					{
						Host: fmt.Sprintf("%s-kubevirt.apps-crc.testing", components.VirtExportProxyServiceName),
					},
				},
			},
		}
	}

	routeToHostAndNoIngress := func() *routev1.Route {
		return &routev1.Route{
			Spec: routev1.RouteSpec{
				To: routev1.RouteTargetReference{
					Name: components.VirtExportProxyServiceName,
				},
			},
			Status: routev1.RouteStatus{
				Ingress: []routev1.RouteIngress{},
			},
		}
	}

	verifyLinksInternal := func(vmExport *exportv1.VirtualMachineExport, link1Format exportv1.ExportVolumeFormat, link1Url string, link2Format exportv1.ExportVolumeFormat, link2Url string) {
		Expect(vmExport.Status).ToNot(BeNil())
		Expect(vmExport.Status.Links).ToNot(BeNil())
		Expect(vmExport.Status.Links.Internal).NotTo(BeNil())
		Expect(vmExport.Status.Links.Internal.Cert).NotTo(BeEmpty())
		Expect(vmExport.Status.Links.Internal.Volumes).To(HaveLen(1))
		Expect(vmExport.Status.Links.Internal.Volumes[0].Formats).To(HaveLen(2))
		Expect(vmExport.Status.Links.Internal.Volumes[0].Formats).To(ContainElements(exportv1.VirtualMachineExportVolumeFormat{
			Format: link1Format,
			Url:    link1Url,
		}, exportv1.VirtualMachineExportVolumeFormat{
			Format: link2Format,
			Url:    link2Url,
		}))
	}

	verifyLinksExternal := func(vmExport *exportv1.VirtualMachineExport, link1Format exportv1.ExportVolumeFormat, link1Url string, link2Format exportv1.ExportVolumeFormat, link2Url string) {
		Expect(vmExport.Status.Links.External).ToNot(BeNil())
		Expect(vmExport.Status.Links.External.Cert).To(BeEmpty())
		Expect(vmExport.Status.Links.External.Volumes).To(HaveLen(1))
		Expect(vmExport.Status.Links.External.Volumes[0].Formats).To(HaveLen(2))
		Expect(vmExport.Status.Links.External.Volumes[0].Formats).To(ContainElements(exportv1.VirtualMachineExportVolumeFormat{
			Format: link1Format,
			Url:    link1Url,
		}, exportv1.VirtualMachineExportVolumeFormat{
			Format: link2Format,
			Url:    link2Url,
		}))
	}

	verifyLinksEmpty := func(vmExport *exportv1.VirtualMachineExport) {
		Expect(vmExport.Status).ToNot(BeNil())
		Expect(vmExport.Status.Links).ToNot(BeNil())
		Expect(vmExport.Status.Links.Internal).NotTo(BeNil())
		Expect(vmExport.Status.Links.Internal.Cert).NotTo(BeEmpty())
		Expect(vmExport.Status.Links.Internal.Volumes).To(HaveLen(0))
		Expect(vmExport.Status.Links.External).To(BeNil())
	}

	verifyKubevirtInternal := func(vmExport *exportv1.VirtualMachineExport, exportName, namespace, volumeName string) {
		verifyLinksInternal(vmExport,
			exportv1.KubeVirtRaw,
			fmt.Sprintf("https://%s.%s.svc/volumes/%s/disk.img", fmt.Sprintf("%s-%s", exportPrefix, exportName), namespace, volumeName),
			exportv1.KubeVirtGz,
			fmt.Sprintf("https://%s.%s.svc/volumes/%s/disk.img.gz", fmt.Sprintf("%s-%s", exportPrefix, exportName), namespace, volumeName))
	}

	verifyKubevirtExternal := func(vmExport *exportv1.VirtualMachineExport, exportName, namespace, volumeName string) {
		verifyLinksExternal(vmExport,
			exportv1.KubeVirtRaw,
			fmt.Sprintf("https://virt-exportproxy-kubevirt.apps-crc.testing/api/export.kubevirt.io/v1alpha1/namespaces/%s/virtualmachineexports/%s/volumes/%s/disk.img", namespace, exportName, volumeName),
			exportv1.KubeVirtGz,
			fmt.Sprintf("https://virt-exportproxy-kubevirt.apps-crc.testing/api/export.kubevirt.io/v1alpha1/namespaces/%s/virtualmachineexports/%s/volumes/%s/disk.img.gz", namespace, exportName, volumeName))
	}

	verifyArchiveInternal := func(vmExport *exportv1.VirtualMachineExport, exportName, namespace, volumeName string) {
		verifyLinksInternal(vmExport,
			exportv1.Dir,
			fmt.Sprintf("https://%s.%s.svc/volumes/%s/dir", fmt.Sprintf("%s-%s", exportPrefix, exportName), namespace, volumeName),
			exportv1.ArchiveGz,
			fmt.Sprintf("https://%s.%s.svc/volumes/%s/disk.tar.gz", fmt.Sprintf("%s-%s", exportPrefix, exportName), namespace, volumeName))
	}

	verifyArchiveExternal := func(vmExport *exportv1.VirtualMachineExport, exportName, namespace, volumeName string) {
		verifyLinksExternal(vmExport,
			exportv1.Dir,
			fmt.Sprintf("https://virt-exportproxy-kubevirt.apps-crc.testing/api/export.kubevirt.io/v1alpha1/namespaces/%s/virtualmachineexports/%s/volumes/%s/dir", namespace, exportName, volumeName),
			exportv1.ArchiveGz,
			fmt.Sprintf("https://virt-exportproxy-kubevirt.apps-crc.testing/api/export.kubevirt.io/v1alpha1/namespaces/%s/virtualmachineexports/%s/volumes/%s/disk.tar.gz", namespace, exportName, volumeName))
	}

	It("Should create a service based on the name of the VMExport", func() {
		var service *k8sv1.Service
		testVMExport := &exportv1.VirtualMachineExport{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: testNamespace,
			},
			Spec: exportv1.VirtualMachineExportSpec{
				Source: k8sv1.TypedLocalObjectReference{
					APIGroup: &k8sv1.SchemeGroupVersion.Group,
					Kind:     "PersistentVolumeClaim",
					Name:     "test-pvc",
				},
			},
		}
		k8sClient.Fake.PrependReactor("create", "services", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			create, ok := action.(testing.CreateAction)
			Expect(ok).To(BeTrue())
			service, ok = create.GetObject().(*k8sv1.Service)
			Expect(ok).To(BeTrue())
			service.Status.Conditions = make([]metav1.Condition, 1)
			service.Status.Conditions[0].Type = "test"
			Expect(service.GetName()).To(Equal(controller.getExportServiceName(testVMExport)))
			Expect(service.GetNamespace()).To(Equal(testNamespace))
			return true, service, nil
		})

		service, err := controller.getOrCreateExportService(testVMExport)
		Expect(err).ToNot(HaveOccurred())
		Expect(service).ToNot(BeNil())
		Expect(service.Status.Conditions[0].Type).To(Equal("test"))

		serviceInformer.GetStore().Add(&k8sv1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      controller.getExportServiceName(testVMExport),
				Namespace: testVMExport.Namespace,
			},
			Spec: k8sv1.ServiceSpec{},
			Status: k8sv1.ServiceStatus{
				Conditions: []metav1.Condition{
					{
						Type: "test2",
					},
				},
			},
		})
		service, err = controller.getOrCreateExportService(testVMExport)
		Expect(err).ToNot(HaveOccurred())
		Expect(service).ToNot(BeNil())
		Expect(service.Status.Conditions[0].Type).To(Equal("test2"))
	})

	It("Should create a pod based on the name of the VMExport", func() {
		testPVC := &k8sv1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pvc",
				Namespace: testNamespace,
			},
		}
		testVMExport := createPVCVMExport()
		k8sClient.Fake.PrependReactor("create", "pods", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			create, ok := action.(testing.CreateAction)
			Expect(ok).To(BeTrue())
			pod, ok := create.GetObject().(*k8sv1.Pod)
			Expect(ok).To(BeTrue())
			Expect(pod.GetName()).To(Equal(fmt.Sprintf("%s-%s", exportPrefix, testVMExport.Name)))
			Expect(pod.GetNamespace()).To(Equal(testNamespace))
			return true, pod, nil
		})
		pod, err := controller.createExporterPod(testVMExport, []*k8sv1.PersistentVolumeClaim{testPVC})
		Expect(err).ToNot(HaveOccurred())
		Expect(pod).ToNot(BeNil())
		Expect(pod.Name).To(Equal(fmt.Sprintf("%s-%s", exportPrefix, testVMExport.Name)))
		Expect(pod.Spec.Volumes).To(HaveLen(3), "There should be 3 volumes, one pvc, and two secrets (token and certs)")
		certSecretName := ""
		for _, volume := range pod.Spec.Volumes {
			if volume.Name == certificates {
				certSecretName = volume.Secret.SecretName
			}
		}
		Expect(pod.Spec.Volumes).To(ContainElement(k8sv1.Volume{
			Name: "test-pvc",
			VolumeSource: k8sv1.VolumeSource{
				PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
					ClaimName: "test-pvc",
				},
			},
		}))
		Expect(pod.Spec.Volumes).To(ContainElement(k8sv1.Volume{
			Name: certificates,
			VolumeSource: k8sv1.VolumeSource{
				Secret: &k8sv1.SecretVolumeSource{
					SecretName: certSecretName,
				},
			},
		}))
		Expect(pod.Spec.Volumes).To(ContainElement(k8sv1.Volume{
			Name: "token",
			VolumeSource: k8sv1.VolumeSource{
				Secret: &k8sv1.SecretVolumeSource{
					SecretName: "token",
				},
			},
		}))
		Expect(pod.Spec.Containers).To(HaveLen(1))
		Expect(pod.Spec.Containers[0].VolumeMounts).To(HaveLen(3))
		Expect(pod.Spec.Containers[0].VolumeMounts).To(ContainElement(k8sv1.VolumeMount{
			Name:      "test-pvc",
			ReadOnly:  true,
			MountPath: fmt.Sprintf("%s/%s", fileSystemMountPath, testPVC.Name),
		}))
		Expect(pod.Spec.Containers[0].VolumeMounts).To(ContainElement(k8sv1.VolumeMount{
			Name:      certificates,
			MountPath: "/cert",
		}))
		Expect(pod.Spec.Containers[0].VolumeMounts).To(ContainElement(k8sv1.VolumeMount{
			Name:      testVMExport.Spec.TokenSecretRef,
			MountPath: "/token",
		}))
	})

	It("Should create a secret based on the vm export", func() {
		testVMExport := createPVCVMExport()
		testExportPod := &k8sv1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-export-pod",
			},
			Spec: k8sv1.PodSpec{
				Volumes: []k8sv1.Volume{
					{
						Name: certificates,
						VolumeSource: k8sv1.VolumeSource{
							Secret: &k8sv1.SecretVolumeSource{
								SecretName: "test-secret",
							},
						},
					},
				},
			},
		}
		k8sClient.Fake.PrependReactor("create", "secrets", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			create, ok := action.(testing.CreateAction)
			Expect(ok).To(BeTrue())
			secret, ok := create.GetObject().(*k8sv1.Secret)
			Expect(ok).To(BeTrue())
			Expect(secret.GetName()).To(Equal(controller.getExportSecretName(testExportPod)))
			Expect(secret.GetNamespace()).To(Equal(testNamespace))
			return true, secret, nil
		})
		err := controller.getOrCreateCertSecret(testVMExport, testExportPod)
		Expect(err).ToNot(HaveOccurred())
		By("Creating again, and returning exists")
		k8sClient.Fake.PrependReactor("create", "secrets", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			create, ok := action.(testing.CreateAction)
			Expect(ok).To(BeTrue())
			secret, ok := create.GetObject().(*k8sv1.Secret)
			Expect(ok).To(BeTrue())
			Expect(secret.GetName()).To(Equal(controller.getExportSecretName(testExportPod)))
			Expect(secret.GetNamespace()).To(Equal(testNamespace))
			secret.Name = "something"
			return true, secret, errors.NewAlreadyExists(schema.GroupResource{}, "already exists")
		})
		err = controller.getOrCreateCertSecret(testVMExport, testExportPod)
		Expect(err).ToNot(HaveOccurred())
		k8sClient.Fake.PrependReactor("create", "secrets", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			create, ok := action.(testing.CreateAction)
			Expect(ok).To(BeTrue())
			secret, ok := create.GetObject().(*k8sv1.Secret)
			Expect(ok).To(BeTrue())
			Expect(secret.GetName()).To(Equal(controller.getExportSecretName(testExportPod)))
			Expect(secret.GetNamespace()).To(Equal(testNamespace))
			return true, nil, fmt.Errorf("failure")
		})
		err = controller.getOrCreateCertSecret(testVMExport, testExportPod)
		Expect(err).To(HaveOccurred())
	})

	DescribeTable("Should ignore invalid VMExports kind/api combinations", func(kind, apigroup string) {
		testVMExport := createPVCVMExport()
		testVMExport.Spec.Source.Kind = kind
		testVMExport.Spec.Source.APIGroup = &apigroup
		retry, err := controller.updateVMExport(testVMExport)
		Expect(err).ToNot(HaveOccurred())
		Expect(retry).To(Equal(time.Duration(0)))
	},
		Entry("VirtualMachineSnapshot kind blank apigroup", "VirtualMachineSnapshot", ""),
		Entry("VirtualMachineSnapshot kind invalid apigroup", "VirtualMachineSnapshot", "invalid"),
		Entry("PersistentVolumeClaim kind invalid apigroup", "PersistentVolumeClaim", "invalid"),
		Entry("PersistentVolumeClaim kind VMSnapshot apigroup", "PersistentVolumeClaim", snapshotv1.SchemeGroupVersion.Group),
	)

	Context("PersistentVolumeClaim export", func() {
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
			Expect(retry).To(BeEquivalentTo(time.Second))
			service, err := k8sClient.CoreV1().Services(testNamespace).Get(context.Background(), fmt.Sprintf("%s-%s", exportPrefix, testVMExport.Name), metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(service.Name).To(Equal(fmt.Sprintf("%s-%s", exportPrefix, testVMExport.Name)))
		})

		It("Should properly update VMExport status with a valid token and archive pvc no route", func() {
			testVMExport := createPVCVMExport()
			pvcInformer.GetStore().Add(&k8sv1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pvc",
					Namespace: testNamespace,
					Annotations: map[string]string{
						annContentType: "archive",
					},
				},
			})
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
			Expect(retry).To(BeEquivalentTo(time.Duration(0)))
			service, err := k8sClient.CoreV1().Services(testNamespace).Get(context.Background(), fmt.Sprintf("%s-%s", exportPrefix, testVMExport.Name), metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(service.Name).To(Equal(fmt.Sprintf("%s-%s", exportPrefix, testVMExport.Name)))
		})

		It("Should properly update VMExport status with a valid token and kubevirt pvc with route", func() {
			testVMExport := createPVCVMExport()
			pvcInformer.GetStore().Add(&k8sv1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pvc",
					Namespace: testNamespace,
					Annotations: map[string]string{
						annContentType: "kubevirt",
					},
				},
			})
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
			Expect(retry).To(BeEquivalentTo(time.Duration(0)))
			service, err := k8sClient.CoreV1().Services(testNamespace).Get(context.Background(), fmt.Sprintf("%s-%s", exportPrefix, testVMExport.Name), metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(service.Name).To(Equal(fmt.Sprintf("%s-%s", exportPrefix, testVMExport.Name)))
		})

		It("Should properly update VMExport status with a valid token and pvc, pending pod", func() {
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
			Expect(retry).To(BeEquivalentTo(time.Second))
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
			Entry("blank content-type", annContentType, "", false),
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

		DescribeTable("should create proper condition from PVC", func(phase k8sv1.PersistentVolumeClaimPhase, status k8sv1.ConditionStatus, reason string) {
			pvc := &k8sv1.PersistentVolumeClaim{
				Status: k8sv1.PersistentVolumeClaimStatus{
					Phase: phase,
				},
			}
			expectedCond := newPvcCondition(status, reason)
			condRes := controller.pvcConditionFromPVC([]*k8sv1.PersistentVolumeClaim{pvc})
			Expect(condRes.Type).To(Equal(expectedCond.Type))
			Expect(condRes.Status).To(Equal(expectedCond.Status))
			Expect(condRes.Reason).To(Equal(expectedCond.Reason))
		},
			Entry("PVC bound", k8sv1.ClaimBound, k8sv1.ConditionTrue, pvcBoundReason),
			Entry("PVC claim lost", k8sv1.ClaimLost, k8sv1.ConditionFalse, unknownReason),
			Entry("PVC pending", k8sv1.ClaimPending, k8sv1.ConditionFalse, pvcPendingReason),
		)
	})

	Context("VMSnapshot export", func() {
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
									Resources: k8sv1.ResourceRequirements{
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
					Resources: k8sv1.ResourceRequirements{
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
					}
				}
				return true, vmExport, nil
			})

			retry, err := controller.updateVMExport(testVMExport)
			Expect(err).ToNot(HaveOccurred())
			Expect(retry).To(BeEquivalentTo(time.Second))
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
				for _, condition := range vmExport.Status.Conditions {
					if condition.Type == exportv1.ConditionReady {
						Expect(condition.Status).To(Equal(k8sv1.ConditionFalse))
						Expect(condition.Reason).To(Equal(podPendingReason))
					}
				}
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
					APIVersion:         apiVersion,
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
			Expect(retry).To(BeEquivalentTo(time.Second))
		})

		It("Should not re-create restored PVCs from VMSnapshot if pvc already exists", func() {
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
						Expect(condition.Reason).To(Equal(podPendingReason))
					}
				}
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
			Expect(retry).To(BeEquivalentTo(time.Second))
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
					APIVersion:         apiVersion,
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
			Expect(retry).To(BeEquivalentTo(time.Duration(0)))
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
					APIVersion:         apiVersion,
					Kind:               "VirtualMachineExport",
					Name:               testVMExport.Name,
					UID:                testVMExport.UID,
					Controller:         pointer.BoolPtr(true),
					BlockOwnerDeletion: pointer.BoolPtr(true),
				}))
				Expect(pvc.GetAnnotations()).ToNot(BeEmpty())
				Expect(pvc.GetAnnotations()[annContentType]).To(BeEmpty())
				return true, pvc, nil
			})
			expectExporterCreate(k8sClient, k8sv1.PodRunning)
			controller.RouteCache.Add(routeToHostAndService(components.VirtExportProxyServiceName))
			vmSnapshotInformer.GetStore().Add(createTestVMSnapshot(false))
			content := createTestVMSnapshotContent("snapshot-content")
			content.Spec.Source.VirtualMachine.Spec.Template.Spec.Volumes[0].DataVolume = nil
			content.Spec.Source.VirtualMachine.Spec.Template.Spec.Volumes[0].MemoryDump = &virtv1.MemoryDumpVolumeSource{}
			vmSnapshotContentInformer.GetStore().Add(content)
			fakeVolumeSnapshotProvider.Add(createTestVolumeSnapshot(testVolumesnapshotName))
			retry, err := controller.updateVMExport(testVMExport)
			Expect(err).ToNot(HaveOccurred())
			Expect(retry).To(BeEquivalentTo(time.Second))
		})
	})

	DescribeTable("should find host when Ingress is defined", func(ingress *networkingv1.Ingress, hostname string) {
		controller.IngressCache.Add(ingress)
		host, _ := controller.getExternalLinkHostAndCert()
		Expect(hostname).To(Equal(host))
	},
		Entry("ingress with default backend host", validIngressDefaultBackend(components.VirtExportProxyServiceName), "backend-host"),
		Entry("ingress with default backend host different service", validIngressDefaultBackend("other-service"), ""),
		Entry("ingress with rules host", validIngressRules(components.VirtExportProxyServiceName), "rule-host"),
		Entry("ingress with rules host different service", validIngressRules("other-service"), ""),
		Entry("ingress with no default backend service", ingressDefaultBackendNoService(), ""),
		Entry("ingress with rules no backend service", ingressRulesNoBackend(), ""),
	)

	DescribeTable("should find host when route is defined", func(route *routev1.Route, hostname, expectedCert string) {
		controller.RouteCache.Add(route)
		controller.RouteConfigMapInformer.GetStore().Add(createRouteConfigMap())
		host, cert := controller.getExternalLinkHostAndCert()
		Expect(hostname).To(Equal(host))
		Expect(expectedCert).To(Equal(cert))
	},
		Entry("route with service and host", routeToHostAndService(components.VirtExportProxyServiceName), "virt-exportproxy-kubevirt.apps-crc.testing", expectedPem),
		Entry("route with different service and host", routeToHostAndService("other-service"), "", ""),
		Entry("route with service and no ingress", routeToHostAndNoIngress(), "", ""),
	)

	It("should pick ingress over route if both exist", func() {
		controller.IngressCache.Add(validIngressDefaultBackend(components.VirtExportProxyServiceName))
		controller.RouteCache.Add(routeToHostAndService(components.VirtExportProxyServiceName))
		host, _ := controller.getExternalLinkHostAndCert()
		Expect("backend-host").To(Equal(host))
	})
})

func writeCertsToDir(dir string) {
	caKeyPair, _ := triple.NewCA("kubevirt.io", time.Hour*24*7)
	crt := certutil.EncodeCertPEM(caKeyPair.Cert)
	key := certutil.EncodePrivateKeyPEM(caKeyPair.Key)
	Expect(ioutil.WriteFile(filepath.Join(dir, bootstrap.CertBytesValue), crt, 0777)).To(Succeed())
	Expect(ioutil.WriteFile(filepath.Join(dir, bootstrap.KeyBytesValue), key, 0777)).To(Succeed())
}

func createPVCVMExport() *exportv1.VirtualMachineExport {
	return &exportv1.VirtualMachineExport{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: testNamespace,
		},
		Spec: exportv1.VirtualMachineExportSpec{
			Source: k8sv1.TypedLocalObjectReference{
				APIGroup: &k8sv1.SchemeGroupVersion.Group,
				Kind:     "PersistentVolumeClaim",
				Name:     "test-pvc",
			},
			TokenSecretRef: "token",
		},
	}
}

func createSnapshotVMExport() *exportv1.VirtualMachineExport {
	return &exportv1.VirtualMachineExport{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: testNamespace,
			UID:       "11111-22222-33333",
		},
		Spec: exportv1.VirtualMachineExportSpec{
			Source: k8sv1.TypedLocalObjectReference{
				APIGroup: &snapshotv1.SchemeGroupVersion.Group,
				Kind:     "VirtualMachineSnapshot",
				Name:     testVmsnapshotName,
			},
			TokenSecretRef: "token",
		},
	}
}

func expectExporterCreate(k8sClient *k8sfake.Clientset, phase k8sv1.PodPhase) {
	k8sClient.Fake.PrependReactor("create", "pods", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		create, ok := action.(testing.CreateAction)
		Expect(ok).To(BeTrue())
		exportPod, ok := create.GetObject().(*k8sv1.Pod)
		Expect(ok).To(BeTrue())
		exportPod.Status = k8sv1.PodStatus{
			Phase: phase,
		}
		return true, exportPod, nil
	})
}

// A mock to implement volumeSnapshotProvider interface
type MockVolumeSnapshotProvider struct {
	volumeSnapshots []*vsv1.VolumeSnapshot
}

func (v *MockVolumeSnapshotProvider) GetVolumeSnapshot(namespace, name string) (*vsv1.VolumeSnapshot, error) {
	if len(v.volumeSnapshots) == 0 {
		return nil, nil
	}
	vs := v.volumeSnapshots[0]
	v.volumeSnapshots = v.volumeSnapshots[1:]
	return vs, nil
}

func (v *MockVolumeSnapshotProvider) Add(s *vsv1.VolumeSnapshot) {
	v.volumeSnapshots = append(v.volumeSnapshots, s)
}
