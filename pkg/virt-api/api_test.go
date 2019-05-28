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
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/emicklei/go-restful"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/cert"
	aggregatorclient "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/certificates/triple"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/virt-api/rest"
)

const namespaceKubevirt = "kubevirt"

var _ = Describe("Virt-api", func() {
	var app virtAPIApp
	var tmpDir string
	var server *ghttp.Server
	var backend *httptest.Server
	var ctrl *gomock.Controller
	var authorizorMock *rest.MockVirtApiAuthorizor
	var expectedValidatingWebhooks *admissionregistrationv1beta1.ValidatingWebhookConfiguration
	var expectedMutatingWebhooks *admissionregistrationv1beta1.MutatingWebhookConfiguration
	subresourceAggregatedApiName := v1.SubresourceGroupVersion.Version + "." + v1.SubresourceGroupName
	log.Log.SetIOWriter(GinkgoWriter)

	BeforeEach(func() {
		app = virtAPIApp{namespace: namespaceKubevirt}
		server = ghttp.NewServer()

		backend = httptest.NewServer(nil)
		tmpDir, err := ioutil.TempDir("", "api_tmp_dir")
		Expect(err).ToNot(HaveOccurred())
		app.virtCli, _ = kubecli.GetKubevirtClientFromFlags(server.URL(), "")
		app.certsDirectory = tmpDir

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
	})

	Context("Virt api server", func() {
		It("should generate certs the first time it is run", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/api/v1/namespaces/kubevirt/secrets/"+virtApiCertSecretName),
					ghttp.RespondWithJSONEncoded(http.StatusNotFound, nil),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/api/v1/namespaces/kubevirt/secrets"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, nil),
				),
			)

			err := app.getSelfSignedCert()
			Expect(err).ToNot(HaveOccurred())
			Expect(len(app.signingCertBytes)).ToNot(Equal(0))
			Expect(len(app.certBytes)).ToNot(Equal(0))
			Expect(len(app.keyBytes)).ToNot(Equal(0))
		}, 5)

		It("should not generate certs if secret already exists", func() {
			caKeyPair, _ := triple.NewCA("kubevirt.io")
			keyPair, _ := triple.NewServerKeyPair(
				caKeyPair,
				"virt-api.kubevirt.pod.cluster.local",
				"virt-api",
				namespaceKubevirt,
				"cluster.local",
				nil,
				nil,
			)
			keyBytes := cert.EncodePrivateKeyPEM(keyPair.Key)
			certBytes := cert.EncodeCertPEM(keyPair.Cert)
			signingCertBytes := cert.EncodeCertPEM(caKeyPair.Cert)
			secret := k8sv1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      virtApiCertSecretName,
					Namespace: namespaceKubevirt,
					Labels: map[string]string{
						v1.AppLabel: "virt-api-aggregator",
					},
				},
				Type: "Opaque",
				Data: map[string][]byte{
					certBytesValue:        certBytes,
					keyBytesValue:         keyBytes,
					signingCertBytesValue: signingCertBytes,
				},
			}

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/api/v1/namespaces/kubevirt/secrets/"+virtApiCertSecretName),
					ghttp.RespondWithJSONEncoded(http.StatusOK, secret),
				),
			)

			err := app.getSelfSignedCert()
			Expect(err).ToNot(HaveOccurred())
			Expect(app.signingCertBytes).To(Equal(signingCertBytes))
			Expect(app.certBytes).To(Equal(certBytes))
			Expect(app.keyBytes).To(Equal(keyBytes))
		}, 5)

		It("should return error if extension-apiserver-authentication ConfigMap doesn't exist", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/api/v1/namespaces/kube-system/configmaps/extension-apiserver-authentication"),
					ghttp.RespondWithJSONEncoded(http.StatusNotFound, nil),
				),
			)

			err := app.readRequestHeader()
			Expect(err).To(HaveOccurred())

		}, 5)

		It("should fail without requestheader CA", func() {

			configMap := &k8sv1.ConfigMap{}
			configMap.Data = make(map[string]string)
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/api/v1/namespaces/kube-system/configmaps/extension-apiserver-authentication"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, configMap),
				),
			)

			err := app.readRequestHeader()
			Expect(err).To(HaveOccurred())
		}, 5)

		It("should create a tls config which uses the CA Manager", func() {
			ca, err := triple.NewCA("first")
			// Just provide any cert
			app.certBytes = cert.EncodeCertPEM(ca.Cert)
			app.keyBytes = cert.EncodePrivateKeyPEM(ca.Key)
			Expect(err).ToNot(HaveOccurred())
			configMap := &k8sv1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:            util.ExtensionAPIServerAuthenticationConfigMap,
					Namespace:       metav1.NamespaceSystem,
					ResourceVersion: "1",
				},
				Data: map[string]string{
					util.RequestHeaderClientCAFileKey: string(cert.EncodeCertPEM(ca.Cert)),
				},
			}
			store := cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)
			Expect(store.Add(configMap)).To(Succeed())
			manager := NewClientCAManager(store)
			Expect(app.setupTLS(manager)).To(Succeed())

			By("checking if the initial certificate is used in the tlsConfig")
			config, err := app.tlsConfig.GetConfigForClient(nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(config.ClientCAs.Subjects()[0]).To(ContainSubstring("first"))

			By("checking if the new certificate is used in the tlsConfig")
			newCA, err := triple.NewCA("new")
			Expect(err).ToNot(HaveOccurred())
			configMap.Data[util.RequestHeaderClientCAFileKey] = string(cert.EncodeCertPEM(newCA.Cert))
			configMap.ObjectMeta.ResourceVersion = "2"
			config, err = app.tlsConfig.GetConfigForClient(nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(config.ClientCAs.Subjects()[0]).To(ContainSubstring("new"))
		})

		It("should auto detect correct request headers from cert configmap", func() {
			configMap := &k8sv1.ConfigMap{}
			configMap.Data = make(map[string]string)
			configMap.Data[util.RequestHeaderClientCAFileKey] = "morefakedata"
			configMap.Data["requestheader-username-headers"] = "[\"fakeheader1\"]"
			configMap.Data["requestheader-group-headers"] = "[\"fakeheader2\"]"
			configMap.Data["requestheader-extra-headers-prefix"] = "[\"fakeheader3-\"]"
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/api/v1/namespaces/kube-system/configmaps/extension-apiserver-authentication"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, configMap),
				),
			)

			err := app.readRequestHeader()
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
	})
})
