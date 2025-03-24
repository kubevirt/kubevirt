package multipath_monitor_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestMultipathMonitor(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "MultipathMonitor Suite")
}
