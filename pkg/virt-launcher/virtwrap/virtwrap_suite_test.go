package virtwrap_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestVirtwrap(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Virtwrap Suite")
}
