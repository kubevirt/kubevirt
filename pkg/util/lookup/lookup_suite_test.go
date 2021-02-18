package lookup

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestOpenapi(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Lookup Suite")
}
