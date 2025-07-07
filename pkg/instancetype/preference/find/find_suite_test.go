package find_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestFind(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Find Suite")
}
