package statefulvm_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestStatefulvm(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Statefulvm Suite")
}
