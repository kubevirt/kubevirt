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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package virt_api

import (
	"crypto/tls"
	"errors"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"

	"github.com/emicklei/go-restful"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"github.com/prometheus/client_golang/prometheus"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/cert/triple"
	"k8s.io/client-go/util/certificate"
	aggregatorclient "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/virt-api/rest"
)

const namespaceKubevirt = "kubevirt"

var _ = Describe("Virt-api", func() {
	var app virtAPIApp
	var tmpDir string
	var certTmpDir string
	var goodPemCertificate1, badPemCertificate string
	var server *ghttp.Server
	var backend *httptest.Server
	var ctrl *gomock.Controller
	var authorizorMock *rest.MockVirtApiAuthorizor
	var expectedValidatingWebhooks *admissionregistrationv1beta1.ValidatingWebhookConfiguration
	var expectedMutatingWebhooks *admissionregistrationv1beta1.MutatingWebhookConfiguration
	var servedCert *tls.Certificate
	subresourceAggregatedApiName := v1.SubresourceGroupVersion.Version + "." + v1.SubresourceGroupName
	log.Log.SetIOWriter(GinkgoWriter)

	BeforeEach(func() {
		var err error
		app = virtAPIApp{namespace: namespaceKubevirt}
		server = ghttp.NewServer()

		backend = httptest.NewServer(nil)
		tmpDir, err = ioutil.TempDir("", "api_tmp_dir")
		certTmpDir, err = ioutil.TempDir("", "cert_tmp_dir")
		app.CertDir = tmpDir
		Expect(err).ToNot(HaveOccurred())
		app.virtCli, _ = kubecli.GetKubevirtClientFromFlags(server.URL(), "")
		prepareCAs(&app, certTmpDir)
		servedCert = prepareValidCert(&app, certTmpDir)

		config, err := clientcmd.BuildConfigFromFlags(server.URL(), "")
		Expect(err).ToNot(HaveOccurred())
		app.authorizor, err = rest.NewAuthorizorFromConfig(config)
		app.aggregatorClient = aggregatorclient.NewForConfigOrDie(config)
		Expect(err).ToNot(HaveOccurred())
		ctrl = gomock.NewController(GinkgoT())
		authorizorMock = rest.NewMockVirtApiAuthorizor(ctrl)

		// Reset go-restful
		http.DefaultServeMux = new(http.ServeMux)
		restful.DefaultContainer = restful.NewContainer()
		restful.DefaultContainer.ServeMux = http.DefaultServeMux

		expectedMutatingWebhooks = &admissionregistrationv1beta1.MutatingWebhookConfiguration{
			TypeMeta: metav1.TypeMeta{
				Kind:       "MutatingWebhookConfiguration",
				APIVersion: "admissionregistration.k8s.io/v1beta1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: virtWebhookMutator,
				Labels: map[string]string{
					v1.AppLabel: virtWebhookMutator,
				},
			},
			Webhooks: app.mutatingWebhooks(),
		}

		expectedValidatingWebhooks = &admissionregistrationv1beta1.ValidatingWebhookConfiguration{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ValidatingWebhookConfiguration",
				APIVersion: "admissionregistration.k8s.io/v1beta1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: virtWebhookValidator,
				Labels: map[string]string{
					v1.AppLabel: virtWebhookValidator,
				},
			},
			Webhooks: app.validatingWebhooks(),
		}
		badPemCertificate = "bad-pem"
		goodPemCertificate1 = `-----BEGIN RSA PRIVATE KEY-----
izfrNTmQLnfsLzi2Wb9xPz2Qj9fQYGgeug3N2MkDuVHwpPcgkhHkJgCQuuvT+qZI
MbS2U6wTS24SZk5RunJIUkitRKeWWMS28SLGfkDs1bBYlSPa5smAd3/q1OePi4ae
dU6YgWuDxzBAKEKVSUu6pA2HOdyQ9N4F1dI+F8w9J990zE93EgyNqZFBBa2L70h4
M7DrB0gJBWMdUMoxGnun5glLiCMo2JrHZ9RkMiallS1sHMhELx2UAlP8I1+0Mav8
iMlHGyUW8EJy0paVf09MPpceEcVwDBeX0+G4UQlO551GTFtOSRjcD8U+GkCzka9W
/SFQrSGe3Gh3SDaOw/4JEMAjWPDLiCglwh0rLIO4VwU6AxzTCuCw3d1ZxQsU6VFQ
PqHA8haOUATZIrp3886PBThVqALBk9p1Nqn51bXLh13Zy9DZIVx4Z5Ioz/EGuzgR
d68VW5wybLjYE2r6Q9nHpitSZ4ZderwjIZRes67HdxYFw8unm4Wo6kuGnb5jSSag
vwBxKzAf3Omn+J6IthTJKuDd13rKZGMcRpQQ6VstwihYt1TahQ/qfJUWPjPcU5ML
9LkgVwA8Ndi1wp1/sEPe+UlL16L6vO9jUHcueWN7+zSUOE/cDSJyMd9x/ZL8QASA
ETd5dujVIqlINL2vJKr1o4T+i0RsnpfFiqFmBKlFqww/SKzJeChdyEtpa/dJMrt2
8S86b6zEmkser+SDYgGketS2DZ4hB+vh2ujSXmS8Gkwrn+BfHMzkbtio8lWbGw0l
eM1tfdFZ6wMTLkxRhBkBK4JiMiUMvpERyPib6a2L6iXTfH+3RUDS6A==
-----END RSA PRIVATE KEY-----
-----BEGIN CERTIFICATE-----
MIICMzCCAZygAwIBAgIJALiPnVsvq8dsMA0GCSqGSIb3DQEBBQUAMFMxCzAJBgNV
BAYTAlVTMQwwCgYDVQQIEwNmb28xDDAKBgNVBAcTA2ZvbzEMMAoGA1UEChMDZm9v
MQwwCgYDVQQLEwNmb28xDDAKBgNVBAMTA2ZvbzAeFw0xMzAzMTkxNTQwMTlaFw0x
ODAzMTgxNTQwMTlaMFMxCzAJBgNVBAYTAlVTMQwwCgYDVQQIEwNmb28xDDAKBgNV
BAcTA2ZvbzEMMAoGA1UEChMDZm9vMQwwCgYDVQQLEwNmb28xDDAKBgNVBAMTA2Zv
bzCBnzANBgkqhkiG9w0BAQEFAAOBjQAwgYkCgYEAzdGfxi9CNbMf1UUcvDQh7MYB
OveIHyc0E0KIbhjK5FkCBU4CiZrbfHagaW7ZEcN0tt3EvpbOMxxc/ZQU2WN/s/wP
xph0pSfsfFsTKM4RhTWD2v4fgk+xZiKd1p0+L4hTtpwnEw0uXRVd0ki6muwV5y/P
+5FHUeldq+pgTcgzuK8CAwEAAaMPMA0wCwYDVR0PBAQDAgLkMA0GCSqGSIb3DQEB
BQUAA4GBAJiDAAtY0mQQeuxWdzLRzXmjvdSuL9GoyT3BF/jSnpxz5/58dba8pWen
v3pj4P3w5DoOso0rzkZy2jEsEitlVM2mLSbQpMM+MUVQCQoiG6W9xuCFuxSrwPIS
pAqEAuV4DNoxQKKWmhVv+J0ptMWD25Pnpxeq5sXzghfJnslJlQND
-----END CERTIFICATE-----`
	})

	Context("Virt api server", func() {

		It("should not request a certificate if it already exists", func() {

			Expect(app.PromTLSConfig).To(BeNil())
			Expect(app.setupTLS()).To(Succeed())
			Expect(app.PromTLSConfig).NotTo(BeNil())
			Eventually(func() *tls.Certificate {
				c, _ := app.PromTLSConfig.GetCertificate(nil)
				return c
			}).Should(Equal(servedCert))
		}, 5)

		It("should return error if client CA doesn't exist", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/api/v1/namespaces/kube-system/configmaps/extension-apiserver-authentication"),
					ghttp.RespondWithJSONEncoded(http.StatusNotFound, nil),
				),
			)

			err := app.getClientCert()
			Expect(err).To(HaveOccurred())

		}, 5)

		It("should retrieve client CA", func() {

			configMap := &k8sv1.ConfigMap{}
			configMap.Data = make(map[string]string)
			configMap.Data["client-ca-file"] = "fakedata"
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/api/v1/namespaces/kube-system/configmaps/extension-apiserver-authentication"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, configMap),
				),
			)

			err := app.getClientCert()
			Expect(err).ToNot(HaveOccurred())
			Expect(app.clientCABytes).To(Equal([]byte("fakedata")))
		}, 5)

		It("should auto detect correct request headers from cert configmap", func() {
			configMap := &k8sv1.ConfigMap{}
			configMap.Data = make(map[string]string)
			configMap.Data["client-ca-file"] = "fakedata"
			configMap.Data["requestheader-username-headers"] = "[\"fakeheader1\"]"
			configMap.Data["requestheader-group-headers"] = "[\"fakeheader2\"]"
			configMap.Data["requestheader-extra-headers-prefix"] = "[\"fakeheader3-\"]"
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/api/v1/namespaces/kube-system/configmaps/extension-apiserver-authentication"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, configMap),
				),
			)

			err := app.getClientCert()
			Expect(err).ToNot(HaveOccurred())
			Expect(app.authorizor.GetUserHeaders()).To(Equal([]string{"X-Remote-User", "fakeheader1"}))
			Expect(app.authorizor.GetGroupHeaders()).To(Equal([]string{"X-Remote-Group", "fakeheader2"}))
			Expect(app.authorizor.GetExtraPrefixHeaders()).To(Equal([]string{"X-Remote-Extra-", "fakeheader3-"}))
		}, 5)

		It("should create apiservice endpoint if one doesn't exist", func() {
			expectedApiService := app.subresourceApiservice()
			expectedApiService.Kind = "APIService"
			expectedApiService.APIVersion = "apiregistration.k8s.io/v1beta1"
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/apis/apiregistration.k8s.io/v1beta1/apiservices/"+subresourceAggregatedApiName),
					ghttp.RespondWithJSONEncoded(http.StatusNotFound, nil),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/apis/apiregistration.k8s.io/v1beta1/apiservices"),
					ghttp.VerifyJSONRepresenting(expectedApiService),
					ghttp.RespondWithJSONEncoded(http.StatusOK, nil),
				),
			)
			err := app.createSubresourceApiservice()
			Expect(err).ToNot(HaveOccurred())
		}, 5)

		It("should update apiservice endpoint if one does exist", func() {
			expectedApiService := app.subresourceApiservice()
			expectedApiService.Kind = "APIService"
			expectedApiService.APIVersion = "apiregistration.k8s.io/v1beta1"
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/apis/apiregistration.k8s.io/v1beta1/apiservices/"+subresourceAggregatedApiName),
					ghttp.RespondWithJSONEncoded(http.StatusOK, app.subresourceApiservice()),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("PUT", "/apis/apiregistration.k8s.io/v1beta1/apiservices/"+subresourceAggregatedApiName),
					ghttp.VerifyJSONRepresenting(expectedApiService),
					ghttp.RespondWithJSONEncoded(http.StatusOK, nil),
				),
			)
			err := app.createSubresourceApiservice()
			Expect(err).ToNot(HaveOccurred())
		}, 5)

		It("should fail if apiservice is at different namespace", func() {
			badApiService := app.subresourceApiservice()
			badApiService.Spec.Service.Namespace = "differentnamespace"
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/apis/apiregistration.k8s.io/v1beta1/apiservices/"+subresourceAggregatedApiName),
					ghttp.RespondWithJSONEncoded(http.StatusOK, badApiService),
				),
			)
			err := app.createSubresourceApiservice()
			Expect(err).To(HaveOccurred())
		}, 5)

		It("should return internal error on authorizor error", func() {
			app.authorizor = authorizorMock
			authorizorMock.EXPECT().
				Authorize(gomock.Not(gomock.Nil())).
				Return(false, "", errors.New("fake error at authorizor")).
				AnyTimes()
			app.Compose()
			resp, err := http.Get(backend.URL)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusInternalServerError))
		}, 5)

		It("should return unauthorized error if not allowed", func() {
			app.authorizor = authorizorMock
			authorizorMock.EXPECT().
				Authorize(gomock.Not(gomock.Nil())).
				Return(false, "", nil).
				AnyTimes()
			app.Compose()
			resp, err := http.Get(backend.URL)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
		}, 5)

		It("should return ok on root URL", func() {
			app.authorizor = authorizorMock
			authorizorMock.EXPECT().
				Authorize(gomock.Not(gomock.Nil())).
				Return(true, "", nil).
				AnyTimes()
			app.Compose()
			resp, err := http.Get(backend.URL)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
		}, 5)

		It("should have a test endpoint", func() {
			app.authorizor = authorizorMock
			authorizorMock.EXPECT().
				Authorize(gomock.Not(gomock.Nil())).
				Return(true, "", nil).
				AnyTimes()
			app.Compose()
			resp, err := http.Get(backend.URL + "/apis/subresources.kubevirt.io/v1alpha3/namespaces/default/virtualmachineinstances/vm1/test")
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
		}, 5)

		It("should have a version endpoint", func() {
			app.authorizor = authorizorMock
			authorizorMock.EXPECT().
				Authorize(gomock.Not(gomock.Nil())).
				Return(true, "", nil).
				AnyTimes()
			app.Compose()
			resp, err := http.Get(backend.URL + "/apis/subresources.kubevirt.io/v1alpha3/version")
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			// TODO: Check version
		}, 5)

		It("should return info on the api group version", func() {
			app.authorizor = authorizorMock
			authorizorMock.EXPECT().
				Authorize(gomock.Not(gomock.Nil())).
				Return(true, "", nil).
				AnyTimes()
			app.Compose()
			resp, err := http.Get(backend.URL + "/apis/subresources.kubevirt.io/v1alpha3/")
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			// TODO: Check list
		}, 5)

		It("should return info on the API group", func() {
			app.authorizor = authorizorMock
			authorizorMock.EXPECT().
				Authorize(gomock.Not(gomock.Nil())).
				Return(true, "", nil).
				AnyTimes()
			app.Compose()
			resp, err := http.Get(backend.URL + "/apis/subresources.kubevirt.io/")
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			// TODO: Check list
		}, 5)

		It("should return API group list on /apis", func() {
			app.authorizor = authorizorMock
			authorizorMock.EXPECT().
				Authorize(gomock.Not(gomock.Nil())).
				Return(true, "", nil).
				AnyTimes()
			app.Compose()
			resp, err := http.Get(backend.URL + "/apis/")
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			// TODO: Check list
		}, 5)

		It("should register validating webhook if not found", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/apis/admissionregistration.k8s.io/v1beta1/validatingwebhookconfigurations/virt-api-validator"),
					ghttp.RespondWithJSONEncoded(http.StatusNotFound, nil),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/apis/admissionregistration.k8s.io/v1beta1/validatingwebhookconfigurations"),
					ghttp.VerifyJSONRepresenting(expectedValidatingWebhooks),
					ghttp.RespondWithJSONEncoded(http.StatusOK, nil),
				),
			)

			err := app.createValidatingWebhook()
			Expect(err).ToNot(HaveOccurred())
		}, 5)

		It("should update validating webhook if found", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/apis/admissionregistration.k8s.io/v1beta1/validatingwebhookconfigurations/virt-api-validator"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, expectedValidatingWebhooks),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("PUT", "/apis/admissionregistration.k8s.io/v1beta1/validatingwebhookconfigurations/virt-api-validator"),
					ghttp.VerifyJSONRepresenting(expectedValidatingWebhooks),
					ghttp.RespondWithJSONEncoded(http.StatusOK, nil),
				),
			)

			err := app.createValidatingWebhook()
			Expect(err).ToNot(HaveOccurred())
		}, 5)

		It("should fail if validating webhook service at different namespace", func() {
			expectedValidatingWebhooks.Webhooks[0].ClientConfig.Service.Namespace = "WrongNamespace"

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/apis/admissionregistration.k8s.io/v1beta1/validatingwebhookconfigurations/virt-api-validator"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, expectedValidatingWebhooks),
				),
			)

			err := app.createValidatingWebhook()
			Expect(err).To(HaveOccurred())
		}, 5)

		It("should register mutating webhook if not found", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/apis/admissionregistration.k8s.io/v1beta1/mutatingwebhookconfigurations/virt-api-mutator"),
					ghttp.RespondWithJSONEncoded(http.StatusNotFound, nil),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/apis/admissionregistration.k8s.io/v1beta1/mutatingwebhookconfigurations"),
					ghttp.VerifyJSONRepresenting(expectedMutatingWebhooks),
					ghttp.RespondWithJSONEncoded(http.StatusOK, nil),
				),
			)

			err := app.createMutatingWebhook()
			Expect(err).ToNot(HaveOccurred())
		}, 5)

		It("should update mutating webhook if found", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/apis/admissionregistration.k8s.io/v1beta1/mutatingwebhookconfigurations/virt-api-mutator"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, expectedMutatingWebhooks),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("PUT", "/apis/admissionregistration.k8s.io/v1beta1/mutatingwebhookconfigurations/virt-api-mutator"),
					ghttp.VerifyJSONRepresenting(expectedMutatingWebhooks),
					ghttp.RespondWithJSONEncoded(http.StatusOK, nil),
				),
			)

			err := app.createMutatingWebhook()
			Expect(err).ToNot(HaveOccurred())
		}, 5)

		It("should fail if validating webhook service at different namespace", func() {

			expectedMutatingWebhooks.Webhooks[0].ClientConfig.Service.Namespace = "WrongNamespace"

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/apis/admissionregistration.k8s.io/v1beta1/mutatingwebhookconfigurations/virt-api-mutator"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, expectedMutatingWebhooks),
				),
			)

			err := app.createMutatingWebhook()
			Expect(err).To(HaveOccurred())
		}, 5)

		It("should pass setupTLS at good client CA from config", func() {
			app.requestHeaderClientCABytes = nil
			app.clientCABytes = []byte(goodPemCertificate1)
			err := app.setupTLS()
			Expect(err).ToNot(HaveOccurred())
		}, 5)

		It("should fail setupTLS at bad client CA from config", func() {
			app.requestHeaderClientCABytes = nil
			app.clientCABytes = []byte(badPemCertificate)
			err := app.setupTLS()
			Expect(err).To(HaveOccurred())
		}, 5)

		It("should pass setupTLS at good client CA from request", func() {
			app.requestHeaderClientCABytes = []byte(goodPemCertificate1)
			err := app.setupTLS()
			Expect(err).ToNot(HaveOccurred())
		}, 5)

		It("should fail setupTLS at bad client CA from request", func() {
			app.requestHeaderClientCABytes = []byte(badPemCertificate)
			err := app.setupTLS()
			Expect(err).To(HaveOccurred())
		}, 5)

		It("should have default values for flags", func() {
			app.AddFlags()
			Expect(app.SwaggerUI).To(Equal("third_party/swagger-ui"))
			Expect(app.SubresourcesOnly).To(Equal(false))
		}, 5)

	})

	AfterEach(func() {
		backend.Close()
		server.Close()
		os.RemoveAll(tmpDir)
		os.RemoveAll(certTmpDir)
		prometheus.DefaultRegisterer = prometheus.NewRegistry()
	})
})

func prepareValidCert(app *virtAPIApp, certTmpDir string) *tls.Certificate {
	caKeyPair, _ := triple.NewCA("kubevirt.io")
	keyPair, _ := triple.NewServerKeyPair(
		caKeyPair,
		"virt-api.kubevirt.pod.cluster.local",
		"virt-api.kubevirt.svc",
		namespaceKubevirt,
		"cluster.local",
		nil,
		nil,
	)

	keyBytes := cert.EncodePrivateKeyPEM(keyPair.Key)
	certBytes := cert.EncodeCertPEM(keyPair.Cert)
	keyFile := filepath.Join(certTmpDir, "key.pem")
	certFile := filepath.Join(certTmpDir, "cert.pem")
	Expect(ioutil.WriteFile(keyFile, keyBytes, 777)).To(Succeed())
	Expect(ioutil.WriteFile(certFile, certBytes, 777)).To(Succeed())

	store, err := certificate.NewFileStore("kubevirt-client", app.CertDir, app.CertDir, "", "")
	Expect(err).ToNot(HaveOccurred())
	_, err = store.Update(certBytes, keyBytes)
	Expect(err).ToNot(HaveOccurred())

	cert, err := store.Current()
	Expect(err).ToNot(HaveOccurred())
	Expect(cert).ToNot(BeNil())
	Expect(err).ToNot(HaveOccurred())
	return cert
}

func prepareCAs(app *virtAPIApp, certTmpDir string) {
	caKeyPair, _ := triple.NewCA("kubevirt.io")
	caBytes := cert.EncodeCertPEM(caKeyPair.Cert)
	app.PodIpAddress = net.ParseIP("127.0.0.1")
	caFile := filepath.Join(certTmpDir, "ca.pem")
	Expect(ioutil.WriteFile(caFile, caBytes, 777)).To(Succeed())
	app.RootCAFile = caFile
	app.requestHeaderClientCABytes = caBytes
}
