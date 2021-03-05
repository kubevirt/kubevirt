package healthz

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestHealthz(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Healthz Suite")
}
