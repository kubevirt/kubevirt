package nodelabeller_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestNodeLabeller(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "NodeLabeller Suite")
}
