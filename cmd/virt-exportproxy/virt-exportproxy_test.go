package main

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/api/core/v1"
	exportv1 "kubevirt.io/api/export/v1"

	"kubevirt.io/kubevirt/pkg/certificates/triple"
	certutil "kubevirt.io/kubevirt/pkg/certificates/triple/cert"
	"kubevirt.io/kubevirt/pkg/testutils"
)

const (
	testExportService = "virt-export"
	testNamespace     = "default"
	testBackendHost   = testExportService + "." + testNamespace + ".svc"
)

type mockClientCAManager struct {
	pool *x509.CertPool
	err  error
}

func (m *mockClientCAManager) GetCurrent() (*x509.CertPool, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.pool, nil
}

func (m *mockClientCAManager) GetCurrentRaw() ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}

func newTestExportProxyApp(caManager *mockClientCAManager) *exportProxyApp {
	return &exportProxyApp{
		caManager: caManager,
	}
}

func newExportBackendCertChain(ca *triple.KeyPair, svcName, namespace string) (*x509.Certificate, *x509.Certificate) {
	intermediateKey, err := certutil.NewECDSAPrivateKey()
	Expect(err).NotTo(HaveOccurred())
	now := time.Now()
	intermediateTemplate := x509.Certificate{
		SerialNumber:          big.NewInt(now.UnixNano()),
		Subject:               pkix.Name{CommonName: fmt.Sprintf("intermediate@%d", now.Unix())},
		NotBefore:             now.UTC(),
		NotAfter:              now.Add(time.Hour).UTC(),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	intermediateDER, err := x509.CreateCertificate(rand.Reader, &intermediateTemplate, ca.Cert, intermediateKey.Public(), ca.Key)
	Expect(err).NotTo(HaveOccurred())
	intermediateCert, err := x509.ParseCertificate(intermediateDER)
	Expect(err).NotTo(HaveOccurred())

	namespacedName := fmt.Sprintf("%s.%s", svcName, namespace)
	leafKey, err := certutil.NewECDSAPrivateKey()
	Expect(err).NotTo(HaveOccurred())
	leafConfig := certutil.Config{
		CommonName: "export-server",
		AltNames: certutil.AltNames{
			DNSNames: []string{
				svcName,
				namespacedName,
				fmt.Sprintf("%s.svc", namespacedName),
			},
		},
		Usages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	leafCert, err := certutil.NewSignedCert(leafConfig, leafKey, intermediateCert, intermediateKey, time.Hour)
	Expect(err).NotTo(HaveOccurred())
	return leafCert, intermediateCert
}

func newExportBackendCertPair(ca *triple.KeyPair, svcName, namespace string) *triple.KeyPair {
	serverKP, err := triple.NewServerKeyPair(ca, "export-server", svcName, namespace, "cluster.local", nil, nil, time.Hour)
	Expect(err).NotTo(HaveOccurred())
	return serverKP
}

func certPoolFromCA(ca *triple.KeyPair) *x509.CertPool {
	pool := x509.NewCertPool()
	pool.AddCert(ca.Cert)
	return pool
}

func connectionState(serverName string, certs ...*x509.Certificate) tls.ConnectionState {
	return tls.ConnectionState{
		ServerName:       serverName,
		PeerCertificates: certs,
	}
}

var _ = Describe("verifyBackendConnection", func() {
	It("returns an error when no peer certificate is presented", func() {
		app := newTestExportProxyApp(&mockClientCAManager{pool: x509.NewCertPool()})

		err := app.verifyBackendConnection(tls.ConnectionState{})

		Expect(err).To(MatchError("backend presented no certificate"))
	})

	It("propagates errors from caManager.GetCurrent", func() {
		caErr := errors.New("ca unavailable")
		app := newTestExportProxyApp(&mockClientCAManager{err: caErr})

		err := app.verifyBackendConnection(connectionState(testBackendHost, &x509.Certificate{}))

		Expect(err).To(MatchError(caErr))
	})

	DescribeTable("should", func(
		setup func() (*exportProxyApp, tls.ConnectionState),
		expectErr bool,
		errMatcher types.GomegaMatcher,
	) {
		app, cs := setup()
		err := app.verifyBackendConnection(cs)
		if expectErr {
			Expect(err).To(HaveOccurred())
			Expect(err).To(errMatcher)
			return
		}
		Expect(err).NotTo(HaveOccurred())
	},
		Entry("reject a certificate that does not chain to the export CA",
			func() (*exportProxyApp, tls.ConnectionState) {
				trustedCA, err := triple.NewCA("trusted", time.Hour)
				Expect(err).NotTo(HaveOccurred())
				untrustedCA, err := triple.NewCA("untrusted", time.Hour)
				Expect(err).NotTo(HaveOccurred())
				serverKP := newExportBackendCertPair(untrustedCA, testExportService, testNamespace)
				app := newTestExportProxyApp(&mockClientCAManager{pool: certPoolFromCA(trustedCA)})
				return app, connectionState(testBackendHost, serverKP.Cert)
			},
			true,
			MatchError(ContainSubstring("could not verify backend certificate")),
		),
		Entry("reject verification when ServerName is empty",
			func() (*exportProxyApp, tls.ConnectionState) {
				ca, err := triple.NewCA("trusted", time.Hour)
				Expect(err).NotTo(HaveOccurred())
				serverKP := newExportBackendCertPair(ca, testExportService, testNamespace)
				app := newTestExportProxyApp(&mockClientCAManager{pool: certPoolFromCA(ca)})
				return app, connectionState("", serverKP.Cert)
			},
			true,
			MatchError("backend TLS ServerName is required"),
		),
		Entry("reject a valid chain when the hostname does not match",
			func() (*exportProxyApp, tls.ConnectionState) {
				ca, err := triple.NewCA("trusted", time.Hour)
				Expect(err).NotTo(HaveOccurred())
				serverKP := newExportBackendCertPair(ca, testExportService, testNamespace)
				app := newTestExportProxyApp(&mockClientCAManager{pool: certPoolFromCA(ca)})
				return app, connectionState("other."+testNamespace+".svc", serverKP.Cert)
			},
			true,
			MatchError(ContainSubstring("could not verify backend certificate")),
		),
		Entry("accept a valid certificate chain and hostname",
			func() (*exportProxyApp, tls.ConnectionState) {
				ca, err := triple.NewCA("trusted", time.Hour)
				Expect(err).NotTo(HaveOccurred())
				serverKP := newExportBackendCertPair(ca, testExportService, testNamespace)
				app := newTestExportProxyApp(&mockClientCAManager{pool: certPoolFromCA(ca)})
				return app, connectionState(testBackendHost, serverKP.Cert)
			},
			false,
			nil,
		),
		Entry("accept a valid certificate chain with an intermediate",
			func() (*exportProxyApp, tls.ConnectionState) {
				ca, err := triple.NewCA("trusted", time.Hour)
				Expect(err).NotTo(HaveOccurred())
				leafCert, intermediateCert := newExportBackendCertChain(ca, testExportService, testNamespace)
				app := newTestExportProxyApp(&mockClientCAManager{pool: certPoolFromCA(ca)})
				return app, connectionState(testBackendHost, leafCert, intermediateCert)
			},
			false,
			nil,
		),
	)
})

func tlsCertificateFromKeyPair(kp *triple.KeyPair) tls.Certificate {
	return tls.Certificate{
		Certificate: [][]byte{kp.Cert.Raw},
		PrivateKey:  kp.Key,
		Leaf:        kp.Cert,
	}
}

func newLocalBackendKeyPair(ca *triple.KeyPair) *triple.KeyPair {
	serverKP, err := triple.NewServerKeyPair(
		ca,
		"export-server",
		testExportService,
		testNamespace,
		"cluster.local",
		[]string{"127.0.0.1"},
		[]string{"localhost"},
		time.Hour,
	)
	Expect(err).NotTo(HaveOccurred())
	return serverKP
}

func startTLSBackend(serverCert tls.Certificate, minVersion uint16) (addr string) {
	ln, err := tls.Listen("tcp4", "127.0.0.1:0", &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		MinVersion:   minVersion,
		MaxVersion:   minVersion,
	})
	Expect(err).NotTo(HaveOccurred())
	DeferCleanup(func() {
		_ = ln.Close()
	})

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				_, _ = io.Copy(io.Discard, c)
			}(conn)
		}
	}()

	// Dial via localhost so TLS ServerName/SNI is a hostname (IPs are stripped from SNI).
	_, port, err := net.SplitHostPort(ln.Addr().String())
	Expect(err).NotTo(HaveOccurred())
	return net.JoinHostPort("localhost", port)
}

func newDialTestApp(caManager *mockClientCAManager, kv *v1.KubeVirt) *exportProxyApp {
	if kv == nil {
		kv = &v1.KubeVirt{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kubevirt",
				Namespace: "kubevirt",
			},
			Spec: v1.KubeVirtSpec{
				Configuration: v1.KubeVirtConfiguration{},
			},
		}
	}
	_, _, store := testutils.NewFakeClusterConfigUsingKV(kv)
	return &exportProxyApp{
		caManager:     caManager,
		kubeVirtStore: store,
	}
}

var _ = Describe("dialBackendTLS", func() {
	It("establishes a TLS connection to a trusted backend", func() {
		ca, err := triple.NewCA("trusted", time.Hour)
		Expect(err).NotTo(HaveOccurred())
		serverKP := newLocalBackendKeyPair(ca)
		addr := startTLSBackend(tlsCertificateFromKeyPair(serverKP), tls.VersionTLS12)

		app := newDialTestApp(&mockClientCAManager{pool: certPoolFromCA(ca)}, nil)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		DeferCleanup(cancel)

		conn, err := app.dialBackendTLS(ctx, "tcp4", addr)
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() {
			_ = conn.Close()
		})

		tlsConn, ok := conn.(*tls.Conn)
		Expect(ok).To(BeTrue())
		Expect(tlsConn.ConnectionState().HandshakeComplete).To(BeTrue())
		Expect(tlsConn.ConnectionState().PeerCertificates).NotTo(BeEmpty())
	})

	It("fails when the backend certificate is not trusted", func() {
		trustedCA, err := triple.NewCA("trusted", time.Hour)
		Expect(err).NotTo(HaveOccurred())
		untrustedCA, err := triple.NewCA("untrusted", time.Hour)
		Expect(err).NotTo(HaveOccurred())
		serverKP := newLocalBackendKeyPair(untrustedCA)
		addr := startTLSBackend(tlsCertificateFromKeyPair(serverKP), tls.VersionTLS12)

		app := newDialTestApp(&mockClientCAManager{pool: certPoolFromCA(trustedCA)}, nil)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		DeferCleanup(cancel)

		conn, err := app.dialBackendTLS(ctx, "tcp4", addr)
		Expect(err).To(HaveOccurred())
		Expect(conn).To(BeNil())
	})

	It("fails when the dial cannot connect", func() {
		app := newDialTestApp(&mockClientCAManager{pool: x509.NewCertPool()}, nil)
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		DeferCleanup(cancel)

		conn, err := app.dialBackendTLS(ctx, "tcp4", "127.0.0.1:1")
		Expect(err).To(HaveOccurred())
		Expect(conn).To(BeNil())
	})

})

type captureTransport struct {
	lastReq *http.Request
}

func (t *captureTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.lastReq = req
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader("ok")),
		Header:     make(http.Header),
		Request:    req,
	}, nil
}

func newReadyExport(namespace, name, serviceName string) *exportv1.VirtualMachineExport {
	return &exportv1.VirtualMachineExport{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Status: &exportv1.VirtualMachineExportStatus{
			Phase:       exportv1.Ready,
			ServiceName: serviceName,
		},
	}
}

var _ = Describe("proxyHandler", func() {
	var (
		kvStore cache.Store
		app     *exportProxyApp
		capture *captureTransport
	)

	BeforeEach(func() {
		kv := &v1.KubeVirt{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kubevirt",
				Namespace: "kubevirt",
			},
			Spec: v1.KubeVirtSpec{
				Configuration: v1.KubeVirtConfiguration{},
			},
		}
		_, _, kvStore = testutils.NewFakeClusterConfigUsingKV(kv)
		capture = &captureTransport{}
		app = &exportProxyApp{
			kubeVirtStore: kvStore,
			exportStore:   cache.NewStore(cache.MetaNamespaceKeyFunc),
		}
		app.initReverseProxy()
		app.reverseProxy.Transport = capture
	})

	It("rewrites the outbound request URL and clears Host for the backend", func() {
		Expect(app.exportStore.Add(newReadyExport(testNamespace, "my-export", testExportService))).To(Succeed())

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/export.kubevirt.io/v1/namespaces/default/virtualmachineexports/my-export/volumes/disk.img", nil)
		req.Host = "virt-exportproxy.kubevirt.svc:443"
		app.proxyHandler(rec, req)

		Expect(rec.Code).To(Equal(http.StatusOK))
		Expect(capture.lastReq.URL.Scheme).To(Equal("https"))
		Expect(capture.lastReq.URL.Host).To(Equal(testBackendHost + ":443"))
		Expect(capture.lastReq.URL.Path).To(Equal("/volumes/disk.img"))
		Expect(capture.lastReq.Host).To(Equal(""))
	})

	It("returns 404 when the export does not exist", func() {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/export.kubevirt.io/v1/namespaces/default/virtualmachineexports/missing/volumes/disk.img", nil)
		app.proxyHandler(rec, req)

		Expect(rec.Code).To(Equal(http.StatusNotFound))
		Expect(capture.lastReq).To(BeNil())
	})

	It("returns 503 when the export is not Ready", func() {
		export := newReadyExport(testNamespace, "my-export", testExportService)
		export.Status.Phase = exportv1.Pending
		Expect(app.exportStore.Add(export)).To(Succeed())

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/export.kubevirt.io/v1/namespaces/default/virtualmachineexports/my-export/volumes/disk.img", nil)
		app.proxyHandler(rec, req)

		Expect(rec.Code).To(Equal(http.StatusServiceUnavailable))
		Expect(capture.lastReq).To(BeNil())
	})
})
