package dhcp_test

import (
	"testing"

	"kubevirt.io/client-go/testutils"
)

func TestDhcpConfigurator(t *testing.T) {
	testutils.KubeVirtTestSuiteSetup(t, "DHCP Suite")
}
