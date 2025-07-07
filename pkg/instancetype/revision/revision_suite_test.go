package revision_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestRevision(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Revision Suite")
}
