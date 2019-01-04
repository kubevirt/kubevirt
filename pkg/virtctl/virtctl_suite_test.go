package virtctl

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestVirtctl(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Virtctl Suite")
}
