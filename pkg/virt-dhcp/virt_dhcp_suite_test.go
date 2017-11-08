package virtdhcp

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestVirtDHCP(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "VirtDHCP Suite")
}
