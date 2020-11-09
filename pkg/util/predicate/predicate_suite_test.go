package predicate

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestPredicate(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Predicate Suite")
}
