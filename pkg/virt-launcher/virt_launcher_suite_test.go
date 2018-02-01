package virtlauncher_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestVirtLauncher(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "VirtLauncher Suite")
}
