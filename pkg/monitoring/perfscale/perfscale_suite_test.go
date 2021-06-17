package perfscale_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestPerfscale(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Perfscale Suite")
}
