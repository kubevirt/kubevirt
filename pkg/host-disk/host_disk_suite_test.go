package hostdisk

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/log"
)

func TestHostDisk(t *testing.T) {
	log.Log.SetIOWriter(GinkgoWriter)
	RegisterFailHandler(Fail)
	RunSpecs(t, "HostDisk Suite")
}
