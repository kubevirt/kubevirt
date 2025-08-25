package defaults_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestDefaults(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Defaults Suite")
}
