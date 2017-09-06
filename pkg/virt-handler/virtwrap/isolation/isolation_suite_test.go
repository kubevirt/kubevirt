package isolation

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestIsolation(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Isolation Suite")
}
