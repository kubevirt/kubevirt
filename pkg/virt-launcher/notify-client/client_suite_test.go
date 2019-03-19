package eventsclient

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/log"
)

func TestClient(t *testing.T) {
	log.Log.SetIOWriter(GinkgoWriter)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Client Suite")
}
