package isolation

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"kubevirt.io/client-go/log"
)

func TestIsolation(t *testing.T) {
	RegisterFailHandler(Fail)
	log.Log.SetIOWriter(GinkgoWriter)
	RunSpecs(t, "Isolation Suite")
}
