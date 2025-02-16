package ingresscluster

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	configv1 "github.com/openshift/api/config/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/reqresolver"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/downloadhost"
	hcoutils "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
)

const (
	testDomain = "apps.my-domain.com"
	testNS     = "test-ns"
)

func TestIngressClusterController(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "IngressClusterController Suite")
}

var (
	scheme    *runtime.Scheme
	cert, key []byte
)

var _ = BeforeSuite(func() {
	scheme = runtime.NewScheme()
	Expect(corev1.AddToScheme(scheme)).To(Succeed())
	Expect(configv1.Install(scheme)).To(Succeed())
	Expect(hcov1beta1.AddToScheme(scheme)).To(Succeed())

	var err error
	cert, key, err = generateCert()
	Expect(err).ToNot(HaveOccurred())
})

var _ = Describe("IngressClusterController", func() {

	var (
		ingress                 *configv1.Ingress
		hostBeforeTest          downloadhost.CLIDownloadHost
		selfNamespaceBeforeTest string
		defaultDomain           configv1.Hostname
	)

	BeforeEach(func() {
		hostBeforeTest = downloadhost.Get()
		ingress = getIngress()
		selfNamespaceBeforeTest = hcoutils.GetOperatorNamespaceFromEnv()
		Expect(os.Setenv("OPERATOR_NAMESPACE", testNS)).To(Succeed())
		reqresolver.GeneratePlaceHolders()
		selfNamespace = testNS
		defaultDomain = getDefaultCLIIDownloadHost(testDomain)
	})

	AfterEach(func() {
		downloadhost.Set(hostBeforeTest)
		selfNamespace = selfNamespaceBeforeTest
		Expect(os.Setenv("OPERATOR_NAMESPACE", selfNamespace)).To(Succeed())
		reqresolver.GeneratePlaceHolders()
	})

	It("should not add virt-downloads component to the ingress status, if no HC CR", func(ctx context.Context) {
		ctx = logr.NewContext(ctx, GinkgoLogr)
		downloadhost.Set(downloadhost.CLIDownloadHost{
			CurrentHost: defaultDomain,
			DefaultHost: defaultDomain,
		})

		req := reconcile.Request{}
		ingressEventCh := make(chan event.TypedGenericEvent[client.Object], 1)
		defer close(ingressEventCh)

		cl := fake.NewClientBuilder().WithObjects(ingress).WithScheme(scheme).Build()
		r := newIngressClusterController(cl, ingressEventCh)

		_, err := r.Reconcile(ctx, req)
		Expect(err).ToNot(HaveOccurred())

		ingressAfter := &configv1.Ingress{}
		Expect(r.Get(ctx, client.ObjectKey{Name: "cluster"}, ingressAfter)).To(Succeed())
		Expect(ingressAfter.Status.ComponentRoutes).To(BeEmpty())
		Expect(ingressEventCh).To(BeEmpty())
	})

	It("should add virt-downloads component to the ingress status, if HC CR exists", func(ctx context.Context) {
		ctx = logr.NewContext(ctx, GinkgoLogr)
		downloadhost.Set(downloadhost.CLIDownloadHost{
			CurrentHost: defaultDomain,
			DefaultHost: defaultDomain,
		})

		req := reconcile.Request{}
		ingressEventCh := make(chan event.TypedGenericEvent[client.Object], 1)
		defer close(ingressEventCh)

		hc := &hcov1beta1.HyperConverged{
			ObjectMeta: metav1.ObjectMeta{
				Name:      hcoutils.HyperConvergedName,
				Namespace: testNS,
			},
		}

		cl := fake.NewClientBuilder().WithObjects(hc, ingress).WithStatusSubresource(ingress).WithScheme(scheme).Build()
		r := newIngressClusterController(cl, ingressEventCh)

		_, err := r.Reconcile(ctx, req)
		Expect(err).ToNot(HaveOccurred())

		ingressAfter := &configv1.Ingress{}
		Expect(r.Get(ctx, client.ObjectKey{Name: "cluster"}, ingressAfter)).To(Succeed())

		Expect(ingressAfter.Status.ComponentRoutes).To(HaveLen(1))

		component := ingressAfter.Status.ComponentRoutes[0]
		Expect(component.Name).To(Equal(componentName))
		Expect(component.Namespace).To(Equal(testNS))
		Expect(component.DefaultHostname).To(Equal(defaultDomain))
		Expect(component.CurrentHostnames).To(HaveLen(1))
		Expect(component.CurrentHostnames[0]).To(Equal(defaultDomain))
		Expect(meta.IsStatusConditionFalse(component.Conditions, "Degraded")).To(BeTrueBecause("Degraded condition should be False"))
		Expect(ingressEventCh).To(BeEmpty())
	})

	It("should notify for hostname changes, even if no HC CR", func(ctx context.Context) {
		ctx = logr.NewContext(ctx, GinkgoLogr)
		downloadhost.Set(downloadhost.CLIDownloadHost{
			CurrentHost: "something-else.com",
			DefaultHost: "something-else.com",
		})

		req := reconcile.Request{}
		ingressEventCh := make(chan event.TypedGenericEvent[client.Object], 1)
		defer close(ingressEventCh)

		cl := fake.NewClientBuilder().WithObjects(ingress).WithScheme(scheme).Build()
		r := newIngressClusterController(cl, ingressEventCh)

		_, err := r.Reconcile(ctx, req)
		Expect(err).ToNot(HaveOccurred())

		ingressAfter := &configv1.Ingress{}
		Expect(r.Get(ctx, client.ObjectKey{Name: "cluster"}, ingressAfter)).To(Succeed())
		Expect(ingressAfter.Status.ComponentRoutes).To(BeEmpty())
		Eventually(ingressEventCh).Should(Receive())
		Expect(downloadhost.Get().CurrentHost).To(Equal(defaultDomain))
	})

	It("should add virt-downloads component to the ingress status, if HC CR exists", func(ctx context.Context) {
		ctx = logr.NewContext(ctx, GinkgoLogr)
		downloadhost.Set(downloadhost.CLIDownloadHost{
			CurrentHost: "something-else.com",
			DefaultHost: "something-else.com",
		})

		req := reconcile.Request{}
		ingressEventCh := make(chan event.TypedGenericEvent[client.Object], 1)
		defer close(ingressEventCh)

		hc := &hcov1beta1.HyperConverged{
			ObjectMeta: metav1.ObjectMeta{
				Name:      hcoutils.HyperConvergedName,
				Namespace: testNS,
			},
		}

		cl := fake.NewClientBuilder().WithObjects(hc, ingress).WithStatusSubresource(ingress).WithScheme(scheme).Build()
		r := newIngressClusterController(cl, ingressEventCh)

		_, err := r.Reconcile(ctx, req)
		Expect(err).ToNot(HaveOccurred())

		ingressAfter := &configv1.Ingress{}
		Expect(r.Get(ctx, client.ObjectKey{Name: "cluster"}, ingressAfter)).To(Succeed())

		Expect(ingressAfter.Status.ComponentRoutes).To(HaveLen(1))

		Eventually(ingressEventCh).Should(Receive())
		Expect(downloadhost.Get().CurrentHost).To(Equal(defaultDomain))
	})

	It("should notify for hostname changes, even if domain was customized, and no HC CR", func(ctx context.Context) {
		ctx = logr.NewContext(ctx, GinkgoLogr)
		downloadhost.Set(downloadhost.CLIDownloadHost{
			CurrentHost: defaultDomain,
			DefaultHost: defaultDomain,
		})

		req := reconcile.Request{}
		ingressEventCh := make(chan event.TypedGenericEvent[client.Object], 1)
		defer close(ingressEventCh)

		const modifiedDomain = configv1.Hostname("my-dl-link." + testDomain)
		ingress.Spec.ComponentRoutes = []configv1.ComponentRouteSpec{
			{
				Hostname:  modifiedDomain,
				Name:      componentName,
				Namespace: testNS,
			},
		}

		cl := fake.NewClientBuilder().WithObjects(ingress).WithScheme(scheme).Build()
		r := newIngressClusterController(cl, ingressEventCh)

		_, err := r.Reconcile(ctx, req)
		Expect(err).ToNot(HaveOccurred())

		ingressAfter := &configv1.Ingress{}
		Expect(r.Get(ctx, client.ObjectKey{Name: "cluster"}, ingressAfter)).To(Succeed())
		Expect(ingressAfter.Status.ComponentRoutes).To(BeEmpty())
		Eventually(ingressEventCh).Should(Receive())
		Expect(downloadhost.Get().DefaultHost).To(Equal(defaultDomain))
		Expect(downloadhost.Get().CurrentHost).To(Equal(modifiedDomain))
	})

	It("should add virt-downloads component to the ingress status, if domain was customized and HC CR exists", func(ctx context.Context) {
		ctx = logr.NewContext(ctx, GinkgoLogr)
		downloadhost.Set(downloadhost.CLIDownloadHost{
			CurrentHost: defaultDomain,
			DefaultHost: defaultDomain,
		})

		req := reconcile.Request{}
		ingressEventCh := make(chan event.TypedGenericEvent[client.Object], 1)
		defer close(ingressEventCh)

		const modifiedDomain = configv1.Hostname("my-dl-link." + testDomain)
		ingress.Spec.ComponentRoutes = []configv1.ComponentRouteSpec{
			{
				Hostname:  modifiedDomain,
				Name:      componentName,
				Namespace: testNS,
			},
		}

		hc := &hcov1beta1.HyperConverged{
			ObjectMeta: metav1.ObjectMeta{
				Name:      hcoutils.HyperConvergedName,
				Namespace: testNS,
			},
		}

		cl := fake.NewClientBuilder().WithObjects(hc, ingress).WithStatusSubresource(ingress).WithScheme(scheme).Build()
		r := newIngressClusterController(cl, ingressEventCh)

		_, err := r.Reconcile(ctx, req)
		Expect(err).ToNot(HaveOccurred())

		ingressAfter := &configv1.Ingress{}
		Expect(r.Get(ctx, client.ObjectKey{Name: "cluster"}, ingressAfter)).To(Succeed())

		Expect(ingressAfter.Status.ComponentRoutes).To(HaveLen(1))

		component := ingressAfter.Status.ComponentRoutes[0]
		Expect(component.Name).To(Equal(componentName))
		Expect(component.Namespace).To(Equal(testNS))
		Expect(component.DefaultHostname).To(Equal(defaultDomain))
		Expect(component.CurrentHostnames).To(HaveLen(1))
		Expect(component.CurrentHostnames[0]).To(Equal(modifiedDomain))
		Eventually(ingressEventCh).Should(Receive())

		Expect(meta.IsStatusConditionFalse(component.Conditions, "Degraded")).To(BeTrueBecause("Degraded condition should be False"))

		Expect(downloadhost.Get().CurrentHost).To(Equal(modifiedDomain))
	})

	It("should not modify virt-downloads component, if the custom domain is not subdomain, and no secret", func(ctx context.Context) {
		ctx = logr.NewContext(ctx, GinkgoLogr)
		downloadhost.Set(downloadhost.CLIDownloadHost{
			CurrentHost: defaultDomain,
			DefaultHost: defaultDomain,
		})
		req := reconcile.Request{}
		ingressEventCh := make(chan event.TypedGenericEvent[client.Object], 1)
		defer close(ingressEventCh)

		hc := &hcov1beta1.HyperConverged{
			ObjectMeta: metav1.ObjectMeta{
				Name:      hcoutils.HyperConvergedName,
				Namespace: testNS,
			},
		}

		const modifiedDomain = configv1.Hostname("my-dl-link.something-else.com")
		ingress.Spec.ComponentRoutes = []configv1.ComponentRouteSpec{
			{
				Hostname:  modifiedDomain,
				Name:      componentName,
				Namespace: testNS,
			},
		}

		cl := fake.NewClientBuilder().WithObjects(hc, ingress).WithStatusSubresource(ingress).WithScheme(scheme).Build()
		r := newIngressClusterController(cl, ingressEventCh)

		_, err := r.Reconcile(ctx, req)
		Expect(err).ToNot(HaveOccurred())

		ingressAfter := &configv1.Ingress{}
		Expect(r.Get(ctx, client.ObjectKey{Name: "cluster"}, ingressAfter)).To(Succeed())

		Expect(ingressAfter.Status.ComponentRoutes).To(HaveLen(1))

		component := ingressAfter.Status.ComponentRoutes[0]
		Expect(component.Name).To(Equal(componentName))
		Expect(component.Namespace).To(Equal(testNS))
		Expect(component.DefaultHostname).To(Equal(defaultDomain))
		Expect(component.CurrentHostnames).To(HaveLen(1))
		Expect(component.CurrentHostnames[0]).To(Equal(defaultDomain))
		Expect(ingressEventCh).To(BeEmpty())
		Expect(downloadhost.Get().CurrentHost).To(Equal(defaultDomain))

		Expect(meta.IsStatusConditionTrue(component.Conditions, "Degraded")).To(BeTrueBecause("Degraded condition should be True"))
	})

	It("should grab the key, if defined", func(ctx context.Context) {
		ctx = logr.NewContext(ctx, GinkgoLogr)
		downloadhost.Set(downloadhost.CLIDownloadHost{
			CurrentHost: defaultDomain,
			DefaultHost: defaultDomain,
		})
		req := reconcile.Request{}
		ingressEventCh := make(chan event.TypedGenericEvent[client.Object], 1)
		defer close(ingressEventCh)

		hc := &hcov1beta1.HyperConverged{
			ObjectMeta: metav1.ObjectMeta{
				Name:      hcoutils.HyperConvergedName,
				Namespace: testNS,
			},
		}

		const modifiedDomain = configv1.Hostname("my-dl-link.something-else.com")
		ingress.Spec.ComponentRoutes = []configv1.ComponentRouteSpec{
			{
				Hostname:  modifiedDomain,
				Name:      componentName,
				Namespace: testNS,
				ServingCertKeyPairSecret: configv1.SecretNameReference{
					Name: "my-secret",
				},
			},
		}

		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-secret",
				Namespace: secretNamespace,
			},
			Type: corev1.SecretTypeTLS,
			Data: map[string][]byte{
				"tls.crt": cert,
				"tls.key": key,
			},
		}

		cl := fake.NewClientBuilder().WithObjects(hc, ingress, secret).WithStatusSubresource(ingress).WithScheme(scheme).Build()
		r := newIngressClusterController(cl, ingressEventCh)

		_, err := r.Reconcile(ctx, req)
		Expect(err).ToNot(HaveOccurred())

		ingressAfter := &configv1.Ingress{}
		Expect(r.Get(ctx, client.ObjectKey{Name: "cluster"}, ingressAfter)).To(Succeed())

		Expect(ingressAfter.Status.ComponentRoutes).To(HaveLen(1))

		component := ingressAfter.Status.ComponentRoutes[0]
		Expect(component.Name).To(Equal(componentName))
		Expect(component.Namespace).To(Equal(testNS))
		Expect(component.DefaultHostname).To(Equal(defaultDomain))
		Expect(component.CurrentHostnames).To(HaveLen(1))
		Expect(component.CurrentHostnames[0]).To(Equal(modifiedDomain))
		Expect(ingressEventCh).To(Receive())

		host := downloadhost.Get()
		Expect(host.DefaultHost).To(Equal(defaultDomain))
		Expect(host.CurrentHost).To(Equal(modifiedDomain))
		Expect(host.Cert).To(Equal(string(cert)))
		Expect(host.Key).To(Equal(string(key)))

		Expect(meta.IsStatusConditionFalse(component.Conditions, "Degraded")).To(BeTrueBecause("Degraded condition should be False"))
	})

	It("should not modify virt-downloads component, if the secret not found", func(ctx context.Context) {
		ctx = logr.NewContext(ctx, GinkgoLogr)
		downloadhost.Set(downloadhost.CLIDownloadHost{
			CurrentHost: defaultDomain,
			DefaultHost: defaultDomain,
		})
		req := reconcile.Request{}
		ingressEventCh := make(chan event.TypedGenericEvent[client.Object], 1)
		defer close(ingressEventCh)

		hc := &hcov1beta1.HyperConverged{
			ObjectMeta: metav1.ObjectMeta{
				Name:      hcoutils.HyperConvergedName,
				Namespace: testNS,
			},
		}

		const modifiedDomain = configv1.Hostname("my-dl-link." + testDomain)
		ingress.Spec.ComponentRoutes = []configv1.ComponentRouteSpec{
			{
				Hostname:  modifiedDomain,
				Name:      componentName,
				Namespace: testNS,
				ServingCertKeyPairSecret: configv1.SecretNameReference{
					Name: "my-secret",
				},
			},
		}

		cl := fake.NewClientBuilder().WithObjects(hc, ingress).WithStatusSubresource(ingress).WithScheme(scheme).Build()
		r := newIngressClusterController(cl, ingressEventCh)

		_, err := r.Reconcile(ctx, req)
		Expect(err).ToNot(HaveOccurred())

		ingressAfter := &configv1.Ingress{}
		Expect(r.Get(ctx, client.ObjectKey{Name: "cluster"}, ingressAfter)).To(Succeed())

		Expect(ingressAfter.Status.ComponentRoutes).To(HaveLen(1))

		component := ingressAfter.Status.ComponentRoutes[0]
		Expect(component.Name).To(Equal(componentName))
		Expect(component.Namespace).To(Equal(testNS))
		Expect(component.DefaultHostname).To(Equal(defaultDomain))
		Expect(component.CurrentHostnames).To(HaveLen(1))
		Expect(component.CurrentHostnames[0]).To(Equal(defaultDomain))
		Expect(ingressEventCh).To(BeEmpty())
		Expect(downloadhost.Get().CurrentHost).To(Equal(defaultDomain))

		Expect(meta.IsStatusConditionTrue(component.Conditions, "Degraded")).To(BeTrueBecause("Degraded condition should be True"))
	})

	It("should not modify virt-downloads component, if the secret is with a wrong type", func(ctx context.Context) {
		ctx = logr.NewContext(ctx, GinkgoLogr)
		downloadhost.Set(downloadhost.CLIDownloadHost{
			CurrentHost: defaultDomain,
			DefaultHost: defaultDomain,
		})
		req := reconcile.Request{}
		ingressEventCh := make(chan event.TypedGenericEvent[client.Object], 1)
		defer close(ingressEventCh)

		hc := &hcov1beta1.HyperConverged{
			ObjectMeta: metav1.ObjectMeta{
				Name:      hcoutils.HyperConvergedName,
				Namespace: testNS,
			},
		}

		const modifiedDomain = configv1.Hostname("my-dl-link." + testDomain)
		ingress.Spec.ComponentRoutes = []configv1.ComponentRouteSpec{
			{
				Hostname:  modifiedDomain,
				Name:      componentName,
				Namespace: testNS,
				ServingCertKeyPairSecret: configv1.SecretNameReference{
					Name: "my-secret",
				},
			},
		}

		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-secret",
				Namespace: secretNamespace,
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				"tls.crt": cert,
				"tls.key": key,
			},
		}

		cl := fake.NewClientBuilder().WithObjects(hc, ingress, secret).WithStatusSubresource(ingress).WithScheme(scheme).Build()
		r := newIngressClusterController(cl, ingressEventCh)

		_, err := r.Reconcile(ctx, req)
		Expect(err).ToNot(HaveOccurred())

		ingressAfter := &configv1.Ingress{}
		Expect(r.Get(ctx, client.ObjectKey{Name: "cluster"}, ingressAfter)).To(Succeed())

		Expect(ingressAfter.Status.ComponentRoutes).To(HaveLen(1))

		component := ingressAfter.Status.ComponentRoutes[0]
		Expect(component.Name).To(Equal(componentName))
		Expect(component.Namespace).To(Equal(testNS))
		Expect(component.DefaultHostname).To(Equal(defaultDomain))
		Expect(component.CurrentHostnames).To(HaveLen(1))
		Expect(component.CurrentHostnames[0]).To(Equal(defaultDomain))
		Expect(ingressEventCh).To(BeEmpty())
		Expect(downloadhost.Get().CurrentHost).To(Equal(defaultDomain))

		Expect(meta.IsStatusConditionTrue(component.Conditions, "Degraded")).To(BeTrueBecause("Degraded condition should be True"))
	})

	It("should not modify virt-downloads component, if the secret is wrong", func(ctx context.Context) {
		ctx = logr.NewContext(ctx, GinkgoLogr)
		downloadhost.Set(downloadhost.CLIDownloadHost{
			CurrentHost: defaultDomain,
			DefaultHost: defaultDomain,
		})
		req := reconcile.Request{}
		ingressEventCh := make(chan event.TypedGenericEvent[client.Object], 1)
		defer close(ingressEventCh)

		hc := &hcov1beta1.HyperConverged{
			ObjectMeta: metav1.ObjectMeta{
				Name:      hcoutils.HyperConvergedName,
				Namespace: testNS,
			},
		}

		const modifiedDomain = configv1.Hostname("my-dl-link." + testDomain)
		ingress.Spec.ComponentRoutes = []configv1.ComponentRouteSpec{
			{
				Hostname:  modifiedDomain,
				Name:      componentName,
				Namespace: testNS,
				ServingCertKeyPairSecret: configv1.SecretNameReference{
					Name: "my-secret",
				},
			},
		}

		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-secret",
				Namespace: secretNamespace,
			},
			Type: corev1.SecretTypeTLS,
			Data: map[string][]byte{
				"non-tls-key": []byte("some value"),
			},
		}

		cl := fake.NewClientBuilder().WithObjects(hc, ingress, secret).WithStatusSubresource(ingress).WithScheme(scheme).Build()
		r := newIngressClusterController(cl, ingressEventCh)

		_, err := r.Reconcile(ctx, req)
		Expect(err).ToNot(HaveOccurred())

		ingressAfter := &configv1.Ingress{}
		Expect(r.Get(ctx, client.ObjectKey{Name: "cluster"}, ingressAfter)).To(Succeed())

		Expect(ingressAfter.Status.ComponentRoutes).To(HaveLen(1))

		component := ingressAfter.Status.ComponentRoutes[0]
		Expect(component.Name).To(Equal(componentName))
		Expect(component.Namespace).To(Equal(testNS))
		Expect(component.DefaultHostname).To(Equal(defaultDomain))
		Expect(component.CurrentHostnames).To(HaveLen(1))
		Expect(component.CurrentHostnames[0]).To(Equal(defaultDomain))
		Expect(ingressEventCh).To(BeEmpty())
		Expect(downloadhost.Get().CurrentHost).To(Equal(defaultDomain))

		Expect(meta.IsStatusConditionTrue(component.Conditions, "Degraded")).To(BeTrueBecause("Degraded condition should be True"))
	})
})

func getIngress() *configv1.Ingress {
	return &configv1.Ingress{
		TypeMeta: metav1.TypeMeta{
			APIVersion: configv1.GroupVersion.String(),
			Kind:       "Ingress",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster",
		},
		Spec: configv1.IngressSpec{
			Domain: testDomain,
		},
		Status: configv1.IngressStatus{
			DefaultPlacement: "Workers",
		},
	}
}

func generateCert() ([]byte, []byte, error) {
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, nil, err
	}

	publicKey := privKey.PublicKey

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Acme Co"},
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(5 * time.Minute),

		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &publicKey, privKey)
	if err != nil {
		return nil, nil, err
	}

	certOut := &bytes.Buffer{}
	err = pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	if err != nil {
		return nil, nil, err
	}

	privBytes, err := x509.MarshalPKCS8PrivateKey(privKey)
	if err != nil {
		return nil, nil, err
	}

	keyOut := &bytes.Buffer{}
	err = pem.Encode(keyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: privBytes})
	if err != nil {
		return nil, nil, err
	}

	return certOut.Bytes(), keyOut.Bytes(), nil
}
