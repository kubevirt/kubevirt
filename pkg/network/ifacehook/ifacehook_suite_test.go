package ifacehook_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestCpuhook(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Ifacehook Suite")
}
