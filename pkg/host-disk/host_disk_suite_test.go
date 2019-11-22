package hostdisk

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	ephemeraldiskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"

	"kubevirt.io/client-go/log"
)

func TestHostDisk(t *testing.T) {
	log.Log.SetIOWriter(GinkgoWriter)
	RegisterFailHandler(Fail)
	ephemeraldiskutils.MockDefaultOwnershipManager()
	RunSpecs(t, "HostDisk Suite")
}
