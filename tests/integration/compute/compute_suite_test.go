package compute_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestComputeIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Compute Integration Tests")
}
