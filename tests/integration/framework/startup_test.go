package framework

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	virtv1 "kubevirt.io/api/core/v1"
)

var _ = Describe("Framework startup", func() {
	DescribeTable("creates the initial namespace and supplied KubeVirt CR",
		func(ctx SpecContext, namespace string) {
			kv := &virtv1.KubeVirt{
				ObjectMeta: metav1.ObjectMeta{Name: "kubevirt", Namespace: namespace},
				Spec: virtv1.KubeVirtSpec{Configuration: virtv1.KubeVirtConfiguration{
					PermittedHostDevices: &virtv1.PermittedHostDevices{
						USB: []virtv1.USBHostDevice{{ResourceName: "devices.kubevirt.io/usb"}},
					},
				}},
			}
			f := New(WithKubeVirtCR(kv))
			Expect(f.Namespace()).NotTo(BeEmpty())
			Expect(kv.Namespace).To(Equal(namespace), "the option must not mutate the caller's object")
			kv.Spec.Configuration.PermittedHostDevices.USB[0].ResourceName = "changed"
			f.Start(context.Background())
			_, err := f.K8sClient().CoreV1().Namespaces().Get(ctx, f.Namespace(), metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			if namespace == "" {
				namespace = f.Namespace()
			}
			created, err := f.VirtClient().GeneratedKubeVirtClient().KubevirtV1().KubeVirts(namespace).Get(ctx, "kubevirt", metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(created.Spec.Configuration.PermittedHostDevices.USB[0].ResourceName).To(Equal("devices.kubevirt.io/usb"))
		},
		Entry("using the framework namespace by default", ""),
		Entry("preserving an explicit namespace", "custom-install"),
	)

	It("starts an isolated control plane despite inherited existing-cluster mode", func(ctx SpecContext) {
		GinkgoT().Setenv("USE_EXISTING_CLUSTER", "true")
		var requests atomic.Int64
		existingCluster := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			requests.Add(1)
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		DeferCleanup(existingCluster.Close)

		f := New()
		// Redirect any attempted existing-cluster connection to this local endpoint.
		f.env.Config = &rest.Config{Host: existingCluster.URL, Timeout: time.Second}
		DeferCleanup(f.Stop)
		cfg, err := f.env.Start()
		Expect(requests.Load()).To(BeZero(), "startup contacted the existing-cluster endpoint")
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg.Host).NotTo(Equal(existingCluster.URL))

		client, err := kubernetes.NewForConfig(cfg)
		Expect(err).NotTo(HaveOccurred())
		_, err = client.CoreV1().Namespaces().Get(ctx, "default", metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
	}, SpecTimeout(time.Minute))

	It("cleans up a partially started control plane when a binary is missing", func() {
		f := New()
		// An explicit path overrides the builder's KUBEBUILDER_ASSETS setting.
		f.env.ControlPlane.GetAPIServer().Path = filepath.Join(GinkgoT().TempDir(), "missing-apiserver")
		err := InterceptGomegaFailure(func() { f.Start(context.Background()) })
		Expect(err).To(MatchError(ContainSubstring("failed to start envtest")))
	})

	Context("when CRD installation fails", Ordered, func() {
		var f *Framework

		It("reports the failed installation after starting the control plane", func() {
			f = New()
			f.env.CRDs[0].Name = "invalid"
			err := InterceptGomegaFailure(func() { f.Start(context.Background()) })
			Expect(err).To(MatchError(ContainSubstring("failed to start envtest")))
			Expect(f.env.Config).NotTo(BeNil())
		})

		It("stops the API server before the next spec starts", func(ctx SpecContext) {
			cfg := *f.env.Config
			cfg.Timeout = time.Second
			client, err := kubernetes.NewForConfig(&cfg)
			Expect(err).NotTo(HaveOccurred())
			_, err = client.CoreV1().Namespaces().Get(ctx, "default", metav1.GetOptions{})
			Expect(err).To(HaveOccurred(), "API server is still running after failed startup")
		}, SpecTimeout(5*time.Second))
	})
})
