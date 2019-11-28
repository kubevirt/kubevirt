package emptydisk

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	ephemeraldiskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"

	"kubevirt.io/client-go/log"
)

func TestEmptydisk(t *testing.T) {
	log.Log.SetIOWriter(GinkgoWriter)
	RegisterFailHandler(Fail)
	ephemeraldiskutils.MockDefaultOwnershipManager()
	RunSpecs(t, "Emptydisk Suite")
}
