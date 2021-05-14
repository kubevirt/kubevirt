package dhcp_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestDhcpConfigurator(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "DHCP Suite")
}
