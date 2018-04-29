package expose_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// TODO: not tied to actual unit tests
func TestExpose(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Expose Suite")
}
