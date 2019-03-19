package virt_operator

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestVirtOperator(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "VirtOperator Suite")
}
