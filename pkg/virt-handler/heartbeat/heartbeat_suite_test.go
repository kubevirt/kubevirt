package heartbeat_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"kubevirt.io/client-go/log"
)

func TestHeartbeat(t *testing.T) {
	RegisterFailHandler(Fail)
	log.Log.SetIOWriter(GinkgoWriter)
	RunSpecs(t, "Heartbeat Suite")
}
