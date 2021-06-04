package vmistats_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestVmistats(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Vmistats Suite")
}
