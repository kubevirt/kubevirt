package controlplane_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestControlPlaneIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Control Plane Integration Tests")
}
