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
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	routev1 "github.com/openshift/api/route/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"

	k8sv1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	virtv1 "kubevirt.io/api/core/v1"
	kubevirtfake "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/fake"

	exportv1 "kubevirt.io/api/export/v1alpha1"
	"kubevirt.io/client-go/kubecli"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/certificates/bootstrap"
	"kubevirt.io/kubevirt/pkg/certificates/triple"
	"kubevirt.io/kubevirt/pkg/certificates/triple/cert"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
)

const (
	testNamespace = "default"
)

var (
	qemuGid int64 = 107

	routeCerts = `-----BEGIN CERTIFICATE-----
MIIDDDCCAfSgAwIBAgIBATANBgkqhkiG9w0BAQsFADAmMSQwIgYDVQQDDBtpbmdy
ZXNzLW9wZXJhdG9yQDE2NTE4MDcyNDMwHhcNMjIwNTA2MDMyMDQzWhcNMjQwNTA1
MDMyMDQ0WjAmMSQwIgYDVQQDDBtpbmdyZXNzLW9wZXJhdG9yQDE2NTE4MDcyNDMw
ggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQCws7H+bplQfbuji0BmStSm
ZKis+zK559IGGJsiGeJixkJL5oDIy7fIDoo+Ixn/b+Zx1E12nkflRA/HnEiTXdwf
igJIqo52BRkgM0+yV6MXsCJdQHkqQ7cLv1b4lKqGyJC2qkyYIGpTgXam9xD5HRrs
PO2EuUzphOw/f3M+DuPiFMA2jpH5gjGV9tNY7STwGNoGP15S4GQjEKBCiwygx7ns
kJy5NuSXPsu8xtX39Nw3WYGnqMLmnH4i1FBOX7e0h8C47qirOasDCgGONXvod9hB
/NfqEozYOrHkDNTlGbElZY96jfb7Drbzg5o1OrOthPss+qNMOyoU3pECz9yw27F/
AgMBAAGjRTBDMA4GA1UdDwEB/wQEAwICpDASBgNVHRMBAf8ECDAGAQH/AgEAMB0G
A1UdDgQWBBSribjOYNouiKyuMoI5cK/ZrKj8ujANBgkqhkiG9w0BAQsFAAOCAQEA
Ax212uQxTDjIvjg7uYSyX6a3dveSYc9k6xWX7cLwT+2GYljWqk4Qo/bqj/tWl8jl
6vf9EbvCIMDFoOGAHU1ybYBS8CQr1b+ZoM2VaMGxW3LmgOXwNjF1Ck4AJ+dydeFE
Njjy17IOPEFkgI2sZUepD3czHXPr7PdwlAcT5eMXaZVCK9Q75p4mvquWXnirigY8
4Bz8ocoZ8JrjSXdOBkHm93coh/8AJd63pvHCBNxEesSDn92PdwdxJOGvV+cd7oZc
gVe5488uen1E79WyiXNsX8ehsIlvLVyrhLfnlMWqRk4xrE8ArcudnxZuqrYWwTKD
q2Ew9coarTm82g9SBsOw/w==
-----END CERTIFICATE-----
-----BEGIN CERTIFICATE-----
MIIDWzCCAkOgAwIBAgIIMd7lZVb9dm8wDQYJKoZIhvcNAQELBQAwJjEkMCIGA1UE
AwwbaW5ncmVzcy1vcGVyYXRvckAxNjUxODA3MjQzMB4XDTIyMDUwNjAzMjA0NloX
DTI0MDUwNTAzMjA0N1owHTEbMBkGA1UEAwwSKi5hcHBzLWNyYy50ZXN0aW5nMIIB
IjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAp8Wg/XTNbuabVhGHxWuWEhzj
tZJ7nhfn8BbiT49qr8GKuDIJIm5Iu1UD8MXKOlaBfSv7ymeQtO0hk1ZWSkRWgwqV
ZWYaGkPbAmFewB7OAnPsq5n0jdbKsThg+Tib0iZm9LXN758axJtgZD6oIRs9zB1i
37GJZ8EWnJUcvC72YhzXTw45N9h0WLoQ9/7iIoA0J1H1ykfWExp2KZ97FEyJE4Ks
eTdRqJD5HmfCC37I6OtdtZaAQ6aQQE+vKARcnwU/KUtaBm2Iv/JxEwWm9GxadUqX
+Zls+nilMFnqSDCG8e6462eK7e7A0nr+Gsj+g1ubT2MNIsv+JtdIT8qa5kGmNwID
AQABo4GVMIGSMA4GA1UdDwEB/wQEAwIFoDATBgNVHSUEDDAKBggrBgEFBQcDATAM
BgNVHRMBAf8EAjAAMB0GA1UdDgQWBBRWtuVIucqGVpZA5Ox/eu7cXUhKKzAfBgNV
HSMEGDAWgBSribjOYNouiKyuMoI5cK/ZrKj8ujAdBgNVHREEFjAUghIqLmFwcHMt
Y3JjLnRlc3RpbmcwDQYJKoZIhvcNAQELBQADggEBAAh/1Y3gQUCR0eXqER1PeFcS
g24L+wnmR0vUlMtTtcSl4KSLUrvfAw9N4jvFch0u6sESj16sHI7ZcjWGKMoh94zD
36G814tuNtiUEJsKoAxUtL+bm4c4r7by3ffUn1F/bmA+7JgqFO00sa7b+Rk2zR4j
aJ0s4Y6uNgX3ak5zWRRathdapNdkeXrvgwqlm3/+WWk1kOmevdwuojcLiiTjzBbC
Y4QrW7ja9qy1RP9eq950ixZ/sEHPsyiJieA/c/JwN7IeojOrxT69eOZk24Iwr0vr
NhEGG7KXC0rV3V08vIEezN0HWvu7Qkd4IUqlfTnvqSC4DQ8RTfbX0Y7yYKHtjOo=
-----END CERTIFICATE-----
`

	expectedPem = `-----BEGIN CERTIFICATE-----
MIIDWzCCAkOgAwIBAgIIMd7lZVb9dm8wDQYJKoZIhvcNAQELBQAwJjEkMCIGA1UE
AwwbaW5ncmVzcy1vcGVyYXRvckAxNjUxODA3MjQzMB4XDTIyMDUwNjAzMjA0NloX
DTI0MDUwNTAzMjA0N1owHTEbMBkGA1UEAwwSKi5hcHBzLWNyYy50ZXN0aW5nMIIB
IjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAp8Wg/XTNbuabVhGHxWuWEhzj
tZJ7nhfn8BbiT49qr8GKuDIJIm5Iu1UD8MXKOlaBfSv7ymeQtO0hk1ZWSkRWgwqV
ZWYaGkPbAmFewB7OAnPsq5n0jdbKsThg+Tib0iZm9LXN758axJtgZD6oIRs9zB1i
37GJZ8EWnJUcvC72YhzXTw45N9h0WLoQ9/7iIoA0J1H1ykfWExp2KZ97FEyJE4Ks
eTdRqJD5HmfCC37I6OtdtZaAQ6aQQE+vKARcnwU/KUtaBm2Iv/JxEwWm9GxadUqX
+Zls+nilMFnqSDCG8e6462eK7e7A0nr+Gsj+g1ubT2MNIsv+JtdIT8qa5kGmNwID
AQABo4GVMIGSMA4GA1UdDwEB/wQEAwIFoDATBgNVHSUEDDAKBggrBgEFBQcDATAM
BgNVHRMBAf8EAjAAMB0GA1UdDgQWBBRWtuVIucqGVpZA5Ox/eu7cXUhKKzAfBgNV
HSMEGDAWgBSribjOYNouiKyuMoI5cK/ZrKj8ujAdBgNVHREEFjAUghIqLmFwcHMt
Y3JjLnRlc3RpbmcwDQYJKoZIhvcNAQELBQADggEBAAh/1Y3gQUCR0eXqER1PeFcS
g24L+wnmR0vUlMtTtcSl4KSLUrvfAw9N4jvFch0u6sESj16sHI7ZcjWGKMoh94zD
36G814tuNtiUEJsKoAxUtL+bm4c4r7by3ffUn1F/bmA+7JgqFO00sa7b+Rk2zR4j
aJ0s4Y6uNgX3ak5zWRRathdapNdkeXrvgwqlm3/+WWk1kOmevdwuojcLiiTjzBbC
Y4QrW7ja9qy1RP9eq950ixZ/sEHPsyiJieA/c/JwN7IeojOrxT69eOZk24Iwr0vr
NhEGG7KXC0rV3V08vIEezN0HWvu7Qkd4IUqlfTnvqSC4DQ8RTfbX0Y7yYKHtjOo=
-----END CERTIFICATE-----`
)

var _ = Describe("Export controlleer", func() {
	var (
		ctrl             *gomock.Controller
		controller       *VMExportController
		recorder         *record.FakeRecorder
		pvcInformer      cache.SharedIndexInformer
		podInformer      cache.SharedIndexInformer
		cmInformer       cache.SharedIndexInformer
		vmExportInformer cache.SharedIndexInformer
		serviceInformer  cache.SharedIndexInformer
		dvInformer       cache.SharedIndexInformer
		k8sClient        *k8sfake.Clientset
		vmExportClient   *kubevirtfake.Clientset
		routeCache       cache.Store
		ingressCache     cache.Store
		certDir          string
		certFilePath     string
		keyFilePath      string
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
		routeInformer, _ := testutils.NewFakeInformerFor(&routev1.Route{})
		routeCache = routeInformer.GetStore()
		ingressInformer, _ := testutils.NewFakeInformerFor(&networkingv1.Ingress{})
		ingressCache = ingressInformer.GetStore()
		secretInformer, _ := testutils.NewFakeInformerFor(&k8sv1.Secret{})

		config, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&virtv1.KubeVirtConfiguration{})
		k8sClient = k8sfake.NewSimpleClientset()
		vmExportClient = kubevirtfake.NewSimpleClientset()
		recorder = record.NewFakeRecorder(100)

		virtClient.EXPECT().CoreV1().Return(k8sClient.CoreV1()).AnyTimes()
		virtClient.EXPECT().VirtualMachineExport(testNamespace).
			Return(vmExportClient.ExportV1alpha1().VirtualMachineExports(testNamespace)).AnyTimes()

		controller = &VMExportController{
			Client:                 virtClient,
			Recorder:               recorder,
			PVCInformer:            pvcInformer,
			PodInformer:            podInformer,
			ConfigMapInformer:      cmInformer,
			VMExportInformer:       vmExportInformer,
			ServiceInformer:        serviceInformer,
			DataVolumeInformer:     dvInformer,
			KubevirtNamespace:      "kubevirt",
			TemplateService:        services.NewTemplateService("a", 240, "b", "c", "d", "e", "f", "g", pvcInformer.GetStore(), virtClient, config, qemuGid, "h"),
			caCertManager:          bootstrap.NewFileCertificateManager(certFilePath, keyFilePath),
			RouteCache:             routeCache,
			IngressCache:           ingressCache,
			RouteConfigMapInformer: cmInformer,
			SecretInformer:         secretInformer,
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

	createRouteConfigMap := func() *k8sv1.ConfigMap {
		return &k8sv1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      routeCAConfigMapName,
				Namespace: controller.KubevirtNamespace,
			},
			Data: map[string]string{
				routeCaKey: routeCerts,
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
		testVMExport := createVMExport()
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
		testVMExport := createVMExport()
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

	It("Should ignore non pvc VMExports", func() {
		testVMExport := createVMExport()
		testVMExport.Spec.Source.Kind = "invalid"
		retry, err := controller.updateVMExport(testVMExport)
		Expect(err).ToNot(HaveOccurred())
		Expect(retry).To(Equal(time.Duration(0)))
	})

	It("Should properly update VMExport status with a valid token and no pvc", func() {
		testVMExport := createVMExport()
		expectExporterCreate(k8sClient, k8sv1.PodRunning)
		vmExportClient.Fake.PrependReactor("update", "virtualmachineexports", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			update, ok := action.(testing.UpdateAction)
			Expect(ok).To(BeTrue())
			vmExport, ok := update.GetObject().(*exportv1.VirtualMachineExport)
			Expect(ok).To(BeTrue())
			Expect(vmExport.Status).ToNot(BeNil())
			Expect(vmExport.Status.Links).ToNot(BeNil())
			Expect(vmExport.Status.Links.External).To(BeNil())
			Expect(vmExport.Status.Links.Internal).NotTo(BeNil())
			Expect(vmExport.Status.Links.Internal.Cert).NotTo(BeEmpty())
			Expect(vmExport.Status.Links.Internal.Volumes).To(HaveLen(0))
			return true, vmExport, nil
		})

		retry, err := controller.updateVMExport(testVMExport)
		Expect(err).ToNot(HaveOccurred())
		Expect(retry).To(BeEquivalentTo(time.Second))
		service, err := k8sClient.CoreV1().Services(testNamespace).Get(context.Background(), fmt.Sprintf("%s-%s", exportPrefix, testVMExport.Name), metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(service.Name).To(Equal(fmt.Sprintf("%s-%s", exportPrefix, testVMExport.Name)))
	})

	It("Should properly update VMExport status with a valid token and archive pvc", func() {
		testVMExport := createVMExport()
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
			Expect(vmExport.Status.Links.Internal).NotTo(BeNil())
			Expect(vmExport.Status.Links.Internal.Cert).NotTo(BeEmpty())
			Expect(vmExport.Status.Links.Internal.Volumes).To(HaveLen(1))
			Expect(vmExport.Status.Links.Internal.Volumes[0].Formats).To(HaveLen(2))
			Expect(vmExport.Status.Links.Internal.Volumes[0].Formats).To(ContainElements(exportv1.VirtualMachineExportVolumeFormat{
				Format: exportv1.Dir,
				Url:    fmt.Sprintf("https://%s.%s.svc/volumes/%s/dir", fmt.Sprintf("%s-%s", exportPrefix, testVMExport.Name), testNamespace, testVMExport.Spec.Source.Name),
			}, exportv1.VirtualMachineExportVolumeFormat{
				Format: exportv1.ArchiveGz,
				Url:    fmt.Sprintf("https://%s.%s.svc/volumes/%s/disk.tar.gz", fmt.Sprintf("%s-%s", exportPrefix, testVMExport.Name), testNamespace, testVMExport.Spec.Source.Name),
			}))
			return true, vmExport, nil
		})
		retry, err := controller.updateVMExport(testVMExport)
		Expect(err).ToNot(HaveOccurred())
		Expect(retry).To(BeEquivalentTo(time.Duration(0)))
		service, err := k8sClient.CoreV1().Services(testNamespace).Get(context.Background(), fmt.Sprintf("%s-%s", exportPrefix, testVMExport.Name), metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(service.Name).To(Equal(fmt.Sprintf("%s-%s", exportPrefix, testVMExport.Name)))
	})

	It("Should properly update VMExport status with a valid token and kubevirt pvc", func() {
		testVMExport := createVMExport()
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
			Expect(vmExport.Status).ToNot(BeNil())
			Expect(vmExport.Status.Links).ToNot(BeNil())
			Expect(vmExport.Status.Links.Internal).NotTo(BeNil())
			Expect(vmExport.Status.Links.Internal.Cert).NotTo(BeEmpty())
			Expect(vmExport.Status.Links.Internal.Volumes).To(HaveLen(1))
			Expect(vmExport.Status.Links.Internal.Volumes[0].Formats).To(HaveLen(2))
			Expect(vmExport.Status.Links.Internal.Volumes[0].Formats).To(ContainElements(exportv1.VirtualMachineExportVolumeFormat{
				Format: exportv1.KubeVirtRaw,
				Url:    fmt.Sprintf("https://%s.%s.svc/volumes/%s/disk.img", fmt.Sprintf("%s-%s", exportPrefix, testVMExport.Name), testNamespace, testVMExport.Spec.Source.Name),
			}, exportv1.VirtualMachineExportVolumeFormat{
				Format: exportv1.KubeVirtGz,
				Url:    fmt.Sprintf("https://%s.%s.svc/volumes/%s/disk.img.gz", fmt.Sprintf("%s-%s", exportPrefix, testVMExport.Name), testNamespace, testVMExport.Spec.Source.Name),
			}))
			Expect(vmExport.Status.Links.External).ToNot(BeNil())
			Expect(vmExport.Status.Links.External.Cert).To(BeEmpty())
			Expect(vmExport.Status.Links.External.Volumes).To(HaveLen(1))
			Expect(vmExport.Status.Links.External.Volumes[0].Formats).To(HaveLen(2))
			Expect(vmExport.Status.Links.External.Volumes[0].Formats).To(ContainElements(exportv1.VirtualMachineExportVolumeFormat{
				Format: exportv1.KubeVirtRaw,
				Url:    fmt.Sprintf("https://virt-exportproxy-kubevirt.apps-crc.testing/api/export.kubevirt.io/v1alpha1/namespaces/%s/virtualmachineexports/%s/volumes/%s/disk.img", testNamespace, testVMExport.Name, testVMExport.Spec.Source.Name),
			}, exportv1.VirtualMachineExportVolumeFormat{
				Format: exportv1.KubeVirtGz,
				Url:    fmt.Sprintf("https://virt-exportproxy-kubevirt.apps-crc.testing/api/export.kubevirt.io/v1alpha1/namespaces/%s/virtualmachineexports/%s/volumes/%s/disk.img.gz", testNamespace, testVMExport.Name, testVMExport.Spec.Source.Name),
			}))
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
		testVMExport := createVMExport()
		expectExporterCreate(k8sClient, k8sv1.PodPending)
		vmExportClient.Fake.PrependReactor("update", "virtualmachineexports", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			update, ok := action.(testing.UpdateAction)
			Expect(ok).To(BeTrue())
			vmExport, ok := update.GetObject().(*exportv1.VirtualMachineExport)
			Expect(ok).To(BeTrue())
			Expect(vmExport.Status).ToNot(BeNil())
			Expect(vmExport.Status.Links).ToNot(BeNil())
			Expect(vmExport.Status.Links.Internal).NotTo(BeNil())
			Expect(vmExport.Status.Links.Internal.Cert).NotTo(BeEmpty())
			Expect(vmExport.Status.Links.Internal.Volumes).To(HaveLen(0))
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
	crt := cert.EncodeCertPEM(caKeyPair.Cert)
	key := cert.EncodePrivateKeyPEM(caKeyPair.Key)
	Expect(ioutil.WriteFile(filepath.Join(dir, bootstrap.CertBytesValue), crt, 0777)).To(Succeed())
	Expect(ioutil.WriteFile(filepath.Join(dir, bootstrap.KeyBytesValue), key, 0777)).To(Succeed())
}

func createVMExport() *exportv1.VirtualMachineExport {
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
