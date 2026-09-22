package integration_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/tests/integration/framework"
)

var f *framework.Framework

var _ = BeforeSuite(func() {
	f = framework.New()
	f.Start(context.Background())
})

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration Tests")
}
