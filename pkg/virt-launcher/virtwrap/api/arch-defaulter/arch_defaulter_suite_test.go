package archdefaulter_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestArchDefaulter(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ArchDefaulter Suite")
}
