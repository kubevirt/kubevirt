package dra

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/client-go/log"
)

func TestDRA(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "DRA Suite")
}

var _ = BeforeSuite(func() {
	log.Log.SetIOWriter(GinkgoWriter)
})
