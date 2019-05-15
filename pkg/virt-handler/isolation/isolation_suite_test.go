package isolation

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/kunalkushwaha/go-sdk/log"
)

func TestIsolation(t *testing.T) {
	RegisterFailHandler(Fail)
	log.Log.SetIOWriter(GinkgoWriter)
	RunSpecs(t, "Isolation Suite")
}
