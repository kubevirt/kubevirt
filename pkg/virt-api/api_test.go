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
 * Copyright 2017 Red Hat, Inc.
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
	"k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/cert/triple"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
)

var _ = Describe("VM watcher", func() {
	var tmpDir string
	var server *ghttp.Server
	//var vmService services.VMService

	log.Log.SetIOWriter(GinkgoWriter)

	app := virtAPIApp{}
	BeforeEach(func() {
		tmpDir, err := ioutil.TempDir("", "api_tmp_dir")
		Expect(err).ToNot(HaveOccurred())
		server = ghttp.NewServer()
		app.virtCli, _ = kubecli.GetKubevirtClientFromFlags(server.URL(), "")
		app.certsDirectory = tmpDir
	})

	Context("Virt api server", func() {
		It("should generate certs the first time it is run", func(done Done) {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/api/v1/namespaces/kube-system/secrets/"+virtApiCertSecretName),
					ghttp.RespondWithJSONEncoded(http.StatusNotFound, nil),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/api/v1/namespaces/kube-system/secrets"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, nil),
				),
			)

			err := app.getSelfSignedCert()
			Expect(err).ToNot(HaveOccurred())

			Expect(len(app.signingCertBytes)).ToNot(Equal(0))
			Expect(len(app.certBytes)).ToNot(Equal(0))
			Expect(len(app.keyBytes)).ToNot(Equal(0))
			close(done)
		})

		It("should not generate certs if secret already exists", func(done Done) {
			caKeyPair, _ := triple.NewCA("kubevirt.io")
			keyPair, _ := triple.NewServerKeyPair(
				caKeyPair,
				"virt-api.kube-system.pod.cluster.local",
				"virt-api",
				"kube-system",
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
					Namespace: metav1.NamespaceSystem,
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
					ghttp.VerifyRequest("GET", "/api/v1/namespaces/kube-system/secrets/"+virtApiCertSecretName),
					ghttp.RespondWithJSONEncoded(http.StatusOK, secret),
				),
			)

			err := app.getSelfSignedCert()
			Expect(err).ToNot(HaveOccurred())
			Expect(app.signingCertBytes).To(Equal(signingCertBytes))
			Expect(app.certBytes).To(Equal(certBytes))
			Expect(app.keyBytes).To(Equal(keyBytes))
			close(done)
		})
	})

	AfterEach(func() {
		server.Close()
		os.RemoveAll(tmpDir)
	})
})
