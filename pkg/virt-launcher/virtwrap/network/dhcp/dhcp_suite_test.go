package dhcp

import (
	"testing"

	"kubevirt.io/client-go/testutils"
)

func TestNetwork(t *testing.T) {
	testutils.KubeVirtTestSuiteSetup(t, "DHCP test Suite")
}
