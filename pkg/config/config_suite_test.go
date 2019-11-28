package config

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	ephemeraldiskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"

	"kubevirt.io/client-go/log"
)

func TestConfig(t *testing.T) {
	log.Log.SetIOWriter(GinkgoWriter)
	ephemeraldiskutils.MockDefaultOwnershipManager()
	RegisterFailHandler(Fail)
	RunSpecs(t, "Config Suite")
}
