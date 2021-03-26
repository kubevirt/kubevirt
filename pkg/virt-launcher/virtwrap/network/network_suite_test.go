package network

import (
	"testing"

	"kubevirt.io/client-go/testutils"
)

func TestNetwork(t *testing.T) {
	testutils.KubeVirtTestSuiteSetup(t, "Network Suite")
}
