package portforward

import (
	"testing"

	"kubevirt.io/client-go/testutils"
)

func TestPortForward(t *testing.T) {
	testutils.KubeVirtTestSuiteSetup(t)
}
