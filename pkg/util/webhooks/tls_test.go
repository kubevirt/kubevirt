package webhooks_test

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	ocpv1 "github.com/openshift/api/config/v1"
	"k8s.io/client-go/tools/cache"

	v12 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/testutils"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/certificate"

	"kubevirt.io/kubevirt/pkg/certificates/triple/cert"
	"kubevirt.io/kubevirt/pkg/util/webhooks"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
)

type mockCAManager struct {
	caBundle []byte
}

type mockCertManager struct {
	crt *tls.Certificate
}

func (m *mockCertManager) Start() {
	panic("implement me")
}

func (m *mockCertManager) Stop() {
	panic("implement me")
}

func (m *mockCertManager) Current() *tls.Certificate {
	return m.crt
}

func (m *mockCertManager) ServerHealthy() bool {
	panic("implement me")
}

func (m *mockCAManager) GetCurrent() (*x509.CertPool, error) {
	pool := x509.NewCertPool()
	certs, _ := cert.ParseCertsPEM(m.caBundle)
	for _, crt := range certs {
		pool.AddCert(crt)
	}
	return pool, nil
}

var _ = Describe("TLS", func() {

	var caManager webhooks.ClientCAManager
	var certmanagers map[string]certificate.Manager
	var clusterConfig *virtconfig.ClusterConfig
	var kubeVirtInformer cache.SharedIndexInformer

	BeforeEach(func() {
		// Bootstrap TLS for kubevirt
		certmanagers = map[string]certificate.Manager{}
		caSecrets := components.NewCACertSecrets("whatever")
		var caSecret *k8sv1.Secret
		for _, ca := range caSecrets {
			if ca.Name == components.KubeVirtCASecretName {
				caSecret = ca
			}
		}

		secrets := components.NewCertSecrets("install_namespace", "operator_namespace")
		Expect(components.PopulateSecretWithCertificate(caSecret, nil, &v1.Duration{Duration: 1 * time.Hour})).To(Succeed())
		caCert, err := components.LoadCertificates(caSecret)
		Expect(err).ToNot(HaveOccurred())
		for _, secret := range secrets {
			Expect(components.PopulateSecretWithCertificate(secret, caCert, &v1.Duration{Duration: 1 * time.Hour})).To(Succeed())
			crt, err := components.LoadCertificates(secret)
			certmanagers[secret.Name] = &mockCertManager{crt: crt}
			Expect(err).ToNot(HaveOccurred())
		}
		caBundle := cert.EncodeCertPEM(caCert.Leaf)
		caManager = &mockCAManager{caBundle: caBundle}

		kv := &v12.KubeVirt{
			ObjectMeta: v1.ObjectMeta{
				Name:      "kubevirt",
				Namespace: "kubevirt",
			},
			Spec: v12.KubeVirtSpec{
				Configuration: v12.KubeVirtConfiguration{
					DeveloperConfiguration: &v12.DeveloperConfiguration{},
				},
			},
		}
		clusterConfig, _, kubeVirtInformer = testutils.NewFakeClusterConfigUsingKV(kv)
	})

	DescribeTable("on virt-handler with self-signed CA should", func(serverSecret, clientSecret string, errStr string) {
		serverTLSConfig := webhooks.SetupTLSForVirtHandlerServer(caManager, certmanagers[serverSecret], false, clusterConfig)
		clientTLSConfig := webhooks.SetupTLSForVirtHandlerClients(caManager, certmanagers[clientSecret], false)
		srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, "hello")
		}))
		srv.TLS = serverTLSConfig
		srv.StartTLS()
		defer srv.Close()
		srv.Client()
		client := &http.Client{Transport: &http.Transport{TLSClientConfig: clientTLSConfig}}
		resp, err := client.Get(srv.URL)
		if errStr == "" {
			Expect(err).ToNot(HaveOccurred())
		} else {
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(errStr))
			return
		}
		body, err := ioutil.ReadAll(resp.Body)
		Expect(err).ToNot(HaveOccurred())
		Expect(strings.TrimSpace(string(body))).To(Equal("hello"))
	},
		Entry(
			"connect with proper certificates",
			components.VirtHandlerServerCertSecretName,
			components.VirtHandlerCertSecretName,
			"",
		),
		Entry(
			"fail if client uses not a client certificate",
			components.VirtHandlerServerCertSecretName,
			components.VirtHandlerServerCertSecretName,
			"remote error: tls: bad certificate",
		),
		Entry(
			"fail if server uses not a server certificate",
			components.VirtHandlerCertSecretName,
			components.VirtHandlerCertSecretName,
			"x509: certificate specifies an incompatible key usage",
		),
	)

	DescribeTable("on virt-handler with externally-managed certificates should", func(serverSecret, clientSecret string, errStr string) {
		serverTLSConfig := webhooks.SetupTLSForVirtHandlerServer(caManager, certmanagers[serverSecret], true, clusterConfig)
		clientTLSConfig := webhooks.SetupTLSForVirtHandlerClients(caManager, certmanagers[clientSecret], true)
		srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, "hello")
		}))
		srv.TLS = serverTLSConfig
		srv.StartTLS()
		defer srv.Close()
		srv.Client()
		client := &http.Client{Transport: &http.Transport{TLSClientConfig: clientTLSConfig}}
		resp, err := client.Get(srv.URL)
		if errStr == "" {
			Expect(err).ToNot(HaveOccurred())
		} else {
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(errStr))
			return
		}
		body, err := ioutil.ReadAll(resp.Body)
		Expect(err).ToNot(HaveOccurred())
		Expect(strings.TrimSpace(string(body))).To(Equal("hello"))
	},
		Entry(
			"connect with proper certificates",
			components.VirtHandlerServerCertSecretName,
			components.VirtHandlerCertSecretName,
			"",
		),
		Entry(
			"fail if client uses not a client certificate",
			components.VirtHandlerServerCertSecretName,
			components.VirtHandlerServerCertSecretName,
			"remote error: tls: bad certificate",
		),
		Entry(
			"fail if server uses not a server certificate",
			components.VirtHandlerCertSecretName,
			components.VirtHandlerCertSecretName,
			"x509: certificate specifies an incompatible key usage",
		),
	)

	It("should allow anonymous TLS connections to prometheus endpoints", func() {
		serverTLSConfig := webhooks.SetupPromTLS(certmanagers[components.VirtHandlerServerCertSecretName], clusterConfig)
		srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, "hello")
		}))
		srv.TLS = serverTLSConfig
		srv.StartTLS()
		defer srv.Close()
		srv.Client()
		client := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}
		resp, err := client.Get(srv.URL)
		Expect(err).ToNot(HaveOccurred())
		body, err := ioutil.ReadAll(resp.Body)
		Expect(err).ToNot(HaveOccurred())
		Expect(strings.TrimSpace(string(body))).To(Equal("hello"))
	})

	DescribeTable("should verify self-signed client and server certificates", func(serverSecret, clientSecret string, errStr string) {
		serverTLSConfig := webhooks.SetupTLSWithCertManager(caManager, certmanagers[serverSecret], tls.RequireAndVerifyClientCert, clusterConfig)
		clientTLSConfig := webhooks.SetupTLSForVirtHandlerClients(caManager, certmanagers[clientSecret], false)
		srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, "hello")
		}))
		srv.TLS = serverTLSConfig
		srv.StartTLS()
		defer srv.Close()
		srv.Client()
		client := &http.Client{Transport: &http.Transport{TLSClientConfig: clientTLSConfig}}
		resp, err := client.Get(srv.URL)
		if errStr == "" {
			Expect(err).ToNot(HaveOccurred())
		} else {
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(errStr))
			return
		}
		body, err := ioutil.ReadAll(resp.Body)
		Expect(err).ToNot(HaveOccurred())
		Expect(strings.TrimSpace(string(body))).To(Equal("hello"))
	},
		Entry(
			"connect with proper certificates",
			components.VirtHandlerServerCertSecretName,
			components.VirtHandlerCertSecretName,
			"",
		),
		Entry(
			"fail if client uses an invalid certificate",
			components.VirtHandlerServerCertSecretName,
			components.VirtHandlerServerCertSecretName,
			"remote error: tls: bad certificate",
		),
		Entry(
			"fail if server uses an invalid certificate",
			components.VirtHandlerCertSecretName,
			components.VirtHandlerCertSecretName,
			"x509: certificate specifies an incompatible key usage",
		),
	)

	DescribeTable("should verify externally-managed client and server certificates", func(serverSecret, clientSecret string, errStr string) {
		serverTLSConfig := webhooks.SetupTLSWithCertManager(caManager, certmanagers[serverSecret], tls.RequireAndVerifyClientCert, clusterConfig)
		clientTLSConfig := webhooks.SetupTLSForVirtHandlerClients(caManager, certmanagers[clientSecret], true)
		srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, "hello")
		}))
		srv.TLS = serverTLSConfig
		srv.StartTLS()
		defer srv.Close()
		srv.Client()
		client := &http.Client{Transport: &http.Transport{TLSClientConfig: clientTLSConfig}}
		resp, err := client.Get(srv.URL)
		if errStr == "" {
			Expect(err).ToNot(HaveOccurred())
		} else {
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(errStr))
			return
		}
		body, err := ioutil.ReadAll(resp.Body)
		Expect(err).ToNot(HaveOccurred())
		Expect(strings.TrimSpace(string(body))).To(Equal("hello"))
	},
		Entry(
			"connect with proper certificates",
			components.VirtHandlerServerCertSecretName,
			components.VirtHandlerCertSecretName,
			"",
		),
		Entry(
			"fail if client uses an invalid certificate",
			components.VirtHandlerServerCertSecretName,
			components.VirtHandlerServerCertSecretName,
			"remote error: tls: bad certificate",
		),
		Entry(
			"fail if server uses an invalid certificate",
			components.VirtHandlerCertSecretName,
			components.VirtHandlerCertSecretName,
			"x509: certificate specifies an incompatible key usage",
		),
	)

	type configFunc func() *tls.Config

	DescribeTable("should use last kubevirt TLSSecurityProfile", func(serverTLSConfigFunc, clientTLSConfigFunc configFunc) {
		serverTLSConfig := serverTLSConfigFunc()
		clientTLSConfig := clientTLSConfigFunc()
		clientTLSConfig.MaxVersion = tls.VersionTLS12
		srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, "hello")
		}))
		srv.TLS = serverTLSConfig
		srv.StartTLS()
		defer srv.Close()
		srv.Client()
		client := &http.Client{Transport: &http.Transport{TLSClientConfig: clientTLSConfig}}
		resp, err := client.Get(srv.URL)
		Expect(err).ToNot(HaveOccurred())
		body, err := ioutil.ReadAll(resp.Body)
		Expect(err).ToNot(HaveOccurred())
		Expect(strings.TrimSpace(string(body))).To(Equal("hello"))

		kv := clusterConfig.GetConfigFromKubeVirtCR()
		kvConfig := kv.DeepCopy()
		kvConfig.Spec.TLSSecurityProfile = &ocpv1.TLSSecurityProfile{
			Type:   ocpv1.TLSProfileModernType,
			Modern: &ocpv1.ModernTLSProfile{},
		}
		testutils.UpdateFakeKubeVirtClusterConfig(kubeVirtInformer, kvConfig)
		client = &http.Client{Transport: &http.Transport{TLSClientConfig: clientTLSConfig}}
		resp, err = client.Get(srv.URL)
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(ContainSubstring("remote error: tls: protocol version not supported"))
	},
		Entry("on virt-handler",
			func() *tls.Config {
				return webhooks.SetupTLSForVirtHandlerServer(caManager, certmanagers[components.VirtHandlerServerCertSecretName], false, clusterConfig)
			},

			func() *tls.Config {
				return webhooks.SetupTLSForVirtHandlerClients(caManager, certmanagers[components.VirtHandlerCertSecretName], false)
			},
		),
		Entry("on prometheus endpoint",
			func() *tls.Config {
				return webhooks.SetupPromTLS(certmanagers[components.VirtHandlerServerCertSecretName], clusterConfig)
			},
			func() *tls.Config {
				return &tls.Config{InsecureSkipVerify: true}
			},
		),
	)
})
