package compute_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
)

func TestComputeIntegration(t *testing.T) {
	// Envtest uses this logger for control plane startup and CRD installation diagnostics.
	ctrllog.SetLogger(GinkgoLogr)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Compute Integration Tests")
}
