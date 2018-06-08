package expose_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestExpose(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Expose Suite")
}
