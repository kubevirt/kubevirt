package dmetrics_manager

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestDmetricsManager(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "DownwardMetricsManager Suite")
}
