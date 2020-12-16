package operands_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestOperators(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Operands Suite")
}
