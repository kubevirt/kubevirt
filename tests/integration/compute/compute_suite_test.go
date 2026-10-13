package compute_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"

	"kubevirt.io/kubevirt/tests/integration/framework"
)

var f *framework.Framework

var _ = BeforeSuite(func() {
	f = framework.New()
	f.Start(context.Background())
})

func TestComputeIntegration(t *testing.T) {
	// Envtest uses this logger for control plane startup and CRD installation diagnostics.
	ctrllog.SetLogger(GinkgoLogr)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Compute Integration Tests")
}
