package instancetype_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestInstancetype(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Instancetype Suite")
}
