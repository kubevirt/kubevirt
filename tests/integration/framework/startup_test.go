package framework

import (
	"context"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var _ = Describe("Framework startup", func() {
	It("cleans up a partially started control plane when a binary is missing", func() {
		f := New()
		// An explicit path overrides the builder's KUBEBUILDER_ASSETS setting.
		f.env.ControlPlane.GetAPIServer().Path = filepath.Join(GinkgoT().TempDir(), "missing-apiserver")
		DeferCleanup(f.Stop)
		err := InterceptGomegaFailure(func() { f.Start(context.Background()) })
		Expect(err).To(MatchError(ContainSubstring("failed to start envtest")))
	})

	Context("when CRD installation fails", Ordered, func() {
		var f *Framework

		// Keep the regression test from leaking processes if automatic cleanup fails.
		AfterAll(func() { f.Stop() })

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
