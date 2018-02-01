package virtwrap_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestVirtwrap(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Virtwrap Suite")
}
