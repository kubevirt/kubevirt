package components_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestComponents(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Components Suite")
}
