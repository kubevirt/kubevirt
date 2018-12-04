package stats_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestStats(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Stats Suite")
}
