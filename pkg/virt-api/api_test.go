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
	"path/filepath"

	restful "github.com/emicklei/go-restful"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/cert/triple"
	aggregatorclient "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/virt-api/rest"
)

const namespaceKubevirt = "kubevirt"

var _ = Describe("Virt-api", func() {
	var app virtAPIApp
	var tmpDir, keyFile, certFile, signingCertFile, clientCAFile string
	var goodPemCertificate1, goodPemCertificate2, badPemCertificate string
	var server *ghttp.Server
	var backend *httptest.Server
	var ctrl *gomock.Controller
	var authorizorMock *rest.MockVirtApiAuthorizor
	var filesystemMock *MockFilesystem
	var fileMock *MockFile
	var expectedValidatingWebhooks *admissionregistrationv1beta1.ValidatingWebhookConfiguration
	var expectedMutatingWebhooks *admissionregistrationv1beta1.MutatingWebhookConfiguration
	var restrictiveMode os.FileMode
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
		keyFile = filepath.Join(app.certsDirectory, "/key.pem")
		certFile = filepath.Join(app.certsDirectory, "/cert.pem")
		clientCAFile = filepath.Join(app.certsDirectory, "/clientCA.crt")
		signingCertFile = filepath.Join(app.certsDirectory, "/signingCert.pem")
		restrictiveMode = 0600

		config, err := clientcmd.BuildConfigFromFlags(server.URL(), "")
		Expect(err).ToNot(HaveOccurred())
		app.authorizor, err = rest.NewAuthorizorFromConfig(config)
		app.aggregatorClient = aggregatorclient.NewForConfigOrDie(config)
		Expect(err).ToNot(HaveOccurred())
		ctrl = gomock.NewController(GinkgoT())
		authorizorMock = rest.NewMockVirtApiAuthorizor(ctrl)
		filesystemMock = NewMockFilesystem(ctrl)
		fileMock = NewMockFile(ctrl)

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
		goodPemCertificate2 = `-----BEGIN PRIVATE KEY-----
MIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQC+Z2mi8shZ3T0c
5ItI4KwLfYFXxNr3dmIY+2DS9boD18T46Ccow3wW/15SCcI1BdD/pPBmOTpUWP0b
B22l4gAOUIWfk/CAHuaD+pHAGMlolAMdscwDZPdaM3XEdu839y2gy6RL4Pxls8No
sI5h2BTGl3YgjSxA+vKJE+/IXzjajfiKfGHuywPjwPpGl1juOgSaU6zqLf4MlUnq
Daq7r5V9KtJh8dz46PB6c8ALGNM+dxMJSLMyDvT1/7d9aYBkBvpb3mxOu9agIDNn
2AZ3AxYwA0ykBLmc5R6V0toqRIjfruvBHqQfcjsFaoKS6O+QjtA/eBTIzGajnyis
1TfeJxURAgMBAAECggEARANIlq5GpuMCW3m/zy6CBjC0rRdiaBbff7D7qx+fbJP8
hjTXGBaMEuLxXDikKLCFMWxHexxiG5MWBjunDSQnhPV6ZcBAnmNrUCWHPqkb+ME2
Q7so9uVv/cZ4AM/DL6iZoeBcNcaOIf4OhSzcD1NSSIX96i7Dagq57AE1G8v30Qlb
CDybxrkbW8D9TkPh57oH/VNuhGLsFp62BjleYtNqo+aknlnHCj09mFQ+N5cA8DuK
0CcNFCy4C8oZvg9kVsfdypBr4IR0kXTArqMyjUgXe9KqOzf/GdHR9anWhOzHsy/b
T1Nb+vF6fDm0o6WHWhfODoF4iklrdwRibAnme+zegQKBgQDzvt4KQLnG5jWxeCLn
P+QR9q63H98oL3kKToyXPaJVL+I71GtZm4yhYP8KB+bTgrMHPhR5se63cySX2lMJ
RRKkieeEFuDVKVHulRH39g9fMvvl/f97qwv2mAJhdNuIaIjLSVFjQ8WZ6UWvJczp
sTAyhIxDGiOV00HaUp3BFFnEOQKBgQDH+gLGIsLlPAwuMpbSyuDJucMk0Jzo3+at
6h19pu5JpfTWn71Zs0RL45x9BLwbx8oi+vjECjMiaE2OyKC6uObxpXRl5okWQ63E
XBpbONB+fx2v2h1cuB7iJCJxJ6DPTL70torWtwCp+I7CcIT+J/2SqPhiKipWo0Sk
R2dxeb1HmQKBgEU8mmXfLOZKzkWzEncNtwNDRy3NZ95KXd+HoHf1kf8Qsvq7xCKY
BMJygv+ebvr1zVTpVXecC2sg0ewwoBWqATmr0o+6z/K84gEbZxdAVe181gDmvYOr
eqJ5W3PDdfixeOoF0ZCY17B4isrNuf9HzaEL9au56RHOCI6zmQwXc8hBAoGBAMaI
h0h+Kk+7FbynrOUJVbHwIrTiB2WLJFF1JGIi4F9ty21omWv8dcmB51KW6MoLx7qC
v4ahObLnKlifBjNabq1pPe4MufzIpDNV3TTDavqq6KY1PQFYKhEJHsiINzaXUt1Q
fPY+KQKWKeUQIHjS6wQ3jKCoi/AHl5Yg7anS2v/BAoGBAIi309nwDFJG/2UFSObA
WC+V6T7qy62UlwFlBwsFCbxf9FmFQfoP6wwbQef35Wx2aDnZaSzoXKn/1jvG5e1e
TFW7K9oC8JkeA//mnTAVrgkvkaHGZmd27zQYB1U3DsO3fLvEt62PZn8fyEwaczeM
vOOkHciP4pIhAObg/uiO0V9I
-----END PRIVATE KEY-----
-----BEGIN CERTIFICATE-----
MIIDXTCCAkWgAwIBAgIJAM9HYUREwVxFMA0GCSqGSIb3DQEBCwUAMEUxCzAJBgNV
BAYTAkNOMRMwEQYDVQQIDApTb21lLVN0YXRlMSEwHwYDVQQKDBhJbnRlcm5ldCBX
aWRnaXRzIFB0eSBMdGQwHhcNMTQxMTAxMDY1OTE2WhcNMjQxMDI5MDY1OTE2WjBF
MQswCQYDVQQGEwJDTjETMBEGA1UECAwKU29tZS1TdGF0ZTEhMB8GA1UECgwYSW50
ZXJuZXQgV2lkZ2l0cyBQdHkgTHRkMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIB
CgKCAQEAvmdpovLIWd09HOSLSOCsC32BV8Ta93ZiGPtg0vW6A9fE+OgnKMN8Fv9e
UgnCNQXQ/6TwZjk6VFj9GwdtpeIADlCFn5PwgB7mg/qRwBjJaJQDHbHMA2T3WjN1
xHbvN/ctoMukS+D8ZbPDaLCOYdgUxpd2II0sQPryiRPvyF842o34inxh7ssD48D6
RpdY7joEmlOs6i3+DJVJ6g2qu6+VfSrSYfHc+OjwenPACxjTPncTCUizMg709f+3
fWmAZAb6W95sTrvWoCAzZ9gGdwMWMANMpAS5nOUeldLaKkSI367rwR6kH3I7BWqC
kujvkI7QP3gUyMxmo58orNU33icVEQIDAQABo1AwTjAdBgNVHQ4EFgQUJS1vm+Z3
dm8z29qqdzeI94ZmoqwwHwYDVR0jBBgwFoAUJS1vm+Z3dm8z29qqdzeI94Zmoqww
DAYDVR0TBAUwAwEB/zANBgkqhkiG9w0BAQsFAAOCAQEAl/6TWgvtFCcxue+YmpLz
gcPckabL2dbgC7uxbIMgFEJUjtmHRpY1Tih8pKqqbdkPWhK2IBvyCqp7L1P5A4ib
FTKRGogJSaWMjnh/w644yrmsjjo5uoAueqygwha+OAC3gtt6p844hb9KJTjaoMHC
caaZ6jCAnfjAp2O/3bBpgXCy69UNlWizx8aXajn5a9ah/DrY8wZfI+ESRH3oMd/f
hecgZLhdTPSkUJi/l6WK9wBuI8mVl+/Gesi8zgz8u+/BRZsxQoP9tBWUjOG396fm
PCpapHzlchV1N1s0k+poxmoO/GI0GTPcIY3RhU6QJIQ0dtGCLZFVWchJms5u9GBg
xw==
-----END CERTIFICATE-----`

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

		It("should fail setupTLS at clientCAFile write error", func() {
			clientCAFile := filepath.Join(app.certsDirectory, "/clientCA.crt")
			filesystemMock.EXPECT().
				WriteFile(clientCAFile, app.clientCABytes, restrictiveMode).
				Return(errors.New("fake error writing " + clientCAFile))
			err := app.setupTLS(filesystemMock)
			Expect(err).To(HaveOccurred())
		}, 5)

		It("should fail setupTLS at keyBytes write error", func() {
			filesystemMock.EXPECT().
				WriteFile(clientCAFile, app.clientCABytes, restrictiveMode)
			filesystemMock.EXPECT().
				WriteFile(keyFile, app.keyBytes, restrictiveMode).
				Return(errors.New("fake error writing " + keyFile))
			err := app.setupTLS(filesystemMock)
			Expect(err).To(HaveOccurred())
		}, 5)

		It("should fail setupTLS at certFile write error", func() {
			filesystemMock.EXPECT().
				WriteFile(clientCAFile, app.clientCABytes, restrictiveMode)
			filesystemMock.EXPECT().
				WriteFile(keyFile, app.keyBytes, restrictiveMode)
			filesystemMock.EXPECT().
				WriteFile(certFile, app.certBytes, restrictiveMode).
				Return(errors.New("fake error writing " + certFile))
			err := app.setupTLS(filesystemMock)
			Expect(err).To(HaveOccurred())
		}, 5)

		It("should fail setupTLS at signingCertBytes write error", func() {
			filesystemMock.EXPECT().
				WriteFile(clientCAFile, app.clientCABytes, restrictiveMode)
			filesystemMock.EXPECT().
				WriteFile(keyFile, app.keyBytes, restrictiveMode)
			filesystemMock.EXPECT().
				WriteFile(certFile, app.certBytes, restrictiveMode)
			filesystemMock.EXPECT().
				WriteFile(signingCertFile, app.signingCertBytes, restrictiveMode).
				Return(errors.New("fake error writing " + signingCertFile))
			err := app.setupTLS(filesystemMock)
			Expect(err).To(HaveOccurred())
		}, 5)

		It("should fail setupTLS at new pool error", func() {
			filesystemMock.EXPECT().
				WriteFile(clientCAFile, app.clientCABytes, restrictiveMode)
			filesystemMock.EXPECT().
				WriteFile(keyFile, app.keyBytes, restrictiveMode)
			filesystemMock.EXPECT().
				WriteFile(certFile, app.certBytes, restrictiveMode)
			filesystemMock.EXPECT().
				WriteFile(signingCertFile, app.signingCertBytes, restrictiveMode)
			err := app.setupTLS(filesystemMock)
			Expect(err).To(HaveOccurred())
		}, 5)

		It("should fail when client CA from request at open error", func() {
			app.requestHeaderClientCABytes = []byte(goodPemCertificate1)
			filesystemMock.EXPECT().
				WriteFile(clientCAFile, app.clientCABytes, restrictiveMode)
			filesystemMock.EXPECT().
				OpenFile(clientCAFile, os.O_APPEND|os.O_WRONLY, restrictiveMode).
				Return(nil, errors.New("fake error opening "+clientCAFile))
			err := app.setupTLS(filesystemMock)
			Expect(err).To(HaveOccurred())
		}, 5)

		It("should fail when client CA from request at write error", func() {
			app.requestHeaderClientCABytes = []byte(goodPemCertificate1)
			filesystemMock.EXPECT().
				WriteFile(clientCAFile, app.clientCABytes, restrictiveMode)
			filesystemMock.EXPECT().
				OpenFile(clientCAFile, os.O_APPEND|os.O_WRONLY, restrictiveMode).
				Return(fileMock, nil)
			fileMock.EXPECT().
				Write(app.requestHeaderClientCABytes).
				Return(0, errors.New("fake error writing request client CA"))
			fileMock.EXPECT().Close()
			err := app.setupTLS(filesystemMock)
			Expect(err).To(HaveOccurred())
		}, 5)

		It("should pass setupTLS at good client CA from config", func() {
			app.clientCABytes = []byte(goodPemCertificate1)
			err := app.setupTLS(IOUtil{})
			Expect(err).ToNot(HaveOccurred())
		}, 5)

		It("should fail setupTLS at bad client CA from config", func() {
			app.clientCABytes = []byte(badPemCertificate)
			err := app.setupTLS(IOUtil{})
			Expect(len(app.requestHeaderClientCABytes)).To(Equal(0))
			Expect(err).To(HaveOccurred())
		}, 5)

		It("should pass setupTLS at good client CA from request", func() {
			app.requestHeaderClientCABytes = []byte(goodPemCertificate1)
			err := app.setupTLS(IOUtil{})
			Expect(err).ToNot(HaveOccurred())
		}, 5)

		It("should fail setupTLS at bad client CA from request", func() {
			app.requestHeaderClientCABytes = []byte(badPemCertificate)
			err := app.setupTLS(IOUtil{})
			Expect(err).To(HaveOccurred())
		}, 5)

		It("should concatenate at setupTLS client CA from request and config", func() {
			app.clientCABytes = []byte(goodPemCertificate1)
			app.requestHeaderClientCABytes = []byte(goodPemCertificate2)
			err := app.setupTLS(IOUtil{})
			Expect(err).ToNot(HaveOccurred())
			clientCABytes, err := ioutil.ReadFile(clientCAFile)
			Expect(err).ToNot(HaveOccurred())
			Expect(clientCABytes).To(Equal([]byte(goodPemCertificate1 + goodPemCertificate2)))
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
