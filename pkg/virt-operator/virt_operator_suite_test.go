package virt_operator_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestVirtOperator(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "VirtOperator Suite")
}
