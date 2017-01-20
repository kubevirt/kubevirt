package virtwrap

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestLibvirt(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Libvirt Suite")
}
