package vm_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestVm(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Vm Suite")
}
