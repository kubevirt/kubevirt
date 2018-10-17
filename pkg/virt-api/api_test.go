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
	"io/ioutil"
	"net/http"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/cert/triple"
	apiregistrationv1beta1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1beta1"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/virt-api/rest"
)

const namespaceKubevirt = "kubevirt"

var _ = Describe("Virt-api", func() {
	var tmpDir string
	var server *ghttp.Server
	subresourceAggregatedApiName := v1.SubresourceGroupVersion.Version + "." + v1.SubresourceGroupName

	log.Log.SetIOWriter(GinkgoWriter)

	app := virtAPIApp{namespace: namespaceKubevirt}
	BeforeEach(func() {
		server = ghttp.NewServer()
		tmpDir, err := ioutil.TempDir("", "api_tmp_dir")
		Expect(err).ToNot(HaveOccurred())
		app.virtCli, _ = kubecli.GetKubevirtClientFromFlags(server.URL(), "")
		app.certsDirectory = tmpDir
		config, err := clientcmd.BuildConfigFromFlags(server.URL(), "")
		Expect(err).ToNot(HaveOccurred())
		app.authorizor, err = rest.NewAuthorizorFromConfig(config)
		Expect(err).ToNot(HaveOccurred())
	})

	Context("Virt api server", func() {
		It("should generate certs the first time it is run", func(done Done) {
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
			close(done)
		}, 5)

		It("should not generate certs if secret already exists", func(done Done) {
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
			close(done)
		}, 5)

		It("should return error if client CA doesn't exist", func(done Done) {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/api/v1/namespaces/kube-system/configmaps/extension-apiserver-authentication"),
					ghttp.RespondWithJSONEncoded(http.StatusNotFound, nil),
				),
			)

			err := app.getClientCert()
			Expect(err).To(HaveOccurred())

			close(done)
		}, 5)

		It("should retrieve client CA", func(done Done) {

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
			close(done)
		}, 5)

		It("should auto detect correct request headers from cert configmap", func(done Done) {
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

			close(done)
		}, 5)

		It("should create apiservice endpoint if one doesn't exist", func(done Done) {
			app.signingCertBytes = []byte("fake")
			apiService := &apiregistrationv1beta1.APIService{
				ObjectMeta: metav1.ObjectMeta{
					Name: subresourceAggregatedApiName,
					Labels: map[string]string{
						v1.AppLabel: "virt-api-aggregator",
					},
				},
				Spec: apiregistrationv1beta1.APIServiceSpec{
					Service: &apiregistrationv1beta1.ServiceReference{
						Namespace: namespaceKubevirt,
						Name:      "virt-api",
					},
					Group:                v1.SubresourceGroupName,
					Version:              v1.SubresourceGroupVersion.Version,
					CABundle:             app.signingCertBytes,
					GroupPriorityMinimum: 1000,
					VersionPriority:      15,
				},
			}
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/api/v1/namespaces/kubevirt/apiservices/"+subresourceAggregatedApiName),
					ghttp.RespondWithJSONEncoded(http.StatusNotFound, nil),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/api/v1/namespaces/kubevirt/apiservices"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, apiService),
				),
			)
			app.createSubresourceApiservice()
			close(done)
		}, 5)

		It("should not create apiservice endpoint if one does exist", func(done Done) {
			app.signingCertBytes = []byte("fake")
			apiService := &apiregistrationv1beta1.APIService{
				ObjectMeta: metav1.ObjectMeta{
					Name: subresourceAggregatedApiName,
					Labels: map[string]string{
						v1.AppLabel: "virt-api-aggregator",
					},
				},
				Spec: apiregistrationv1beta1.APIServiceSpec{
					Service: &apiregistrationv1beta1.ServiceReference{
						Namespace: namespaceKubevirt,
						Name:      "virt-api",
					},
					Group:                v1.SubresourceGroupName,
					Version:              v1.SubresourceGroupVersion.Version,
					CABundle:             app.signingCertBytes,
					GroupPriorityMinimum: 1000,
					VersionPriority:      15,
				},
			}
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/api/v1/namespaces/kubevirt/apiservices/"+subresourceAggregatedApiName),
					ghttp.RespondWithJSONEncoded(http.StatusNotFound, apiService),
				),
			)
			app.createSubresourceApiservice()

			close(done)
		}, 5)

		It("should update apiservice endpoint if CA bundle changes", func(done Done) {
			app.signingCertBytes = []byte("fake")
			apiServiceDifferent := &apiregistrationv1beta1.APIService{
				ObjectMeta: metav1.ObjectMeta{
					Name: subresourceAggregatedApiName,
					Labels: map[string]string{
						v1.AppLabel: "virt-api-aggregator",
					},
				},
				Spec: apiregistrationv1beta1.APIServiceSpec{
					Service: &apiregistrationv1beta1.ServiceReference{
						Namespace: namespaceKubevirt,
						Name:      "virt-api",
					},
					Group:                v1.SubresourceGroupName,
					Version:              v1.SubresourceGroupVersion.Version,
					CABundle:             []byte("different"),
					GroupPriorityMinimum: 1000,
					VersionPriority:      15,
				},
			}
			apiServiceFixed := &apiregistrationv1beta1.APIService{
				ObjectMeta: metav1.ObjectMeta{
					Name: subresourceAggregatedApiName,
					Labels: map[string]string{
						v1.AppLabel: "virt-api-aggregator",
					},
				},
				Spec: apiregistrationv1beta1.APIServiceSpec{
					Service: &apiregistrationv1beta1.ServiceReference{
						Namespace: namespaceKubevirt,
						Name:      "virt-api",
					},
					Group:                v1.SubresourceGroupName,
					Version:              v1.SubresourceGroupVersion.Version,
					CABundle:             []byte("fake"),
					GroupPriorityMinimum: 1000,
					VersionPriority:      15,
				},
			}
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/api/v1/namespaces/kubevirt/apiservices/"+subresourceAggregatedApiName),
					ghttp.RespondWithJSONEncoded(http.StatusNotFound, apiServiceDifferent),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("PUT", "/api/v1/namespaces/kubevirt/apiservices"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, apiServiceFixed),
				),
			)
			app.createSubresourceApiservice()
			close(done)
		}, 5)
	})

	AfterEach(func() {
		server.Close()
		os.RemoveAll(tmpDir)
	})
})
