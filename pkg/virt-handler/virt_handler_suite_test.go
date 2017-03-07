package virthandler_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestVirtHandler(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "VirtHandler Suite")
}
