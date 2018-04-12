package emptydisk_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestEmptydisk(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Emptydisk Suite")
}
