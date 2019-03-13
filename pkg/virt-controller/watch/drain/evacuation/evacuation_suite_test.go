package evacuation_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestEvacuation(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Evacuation Suite")
}
