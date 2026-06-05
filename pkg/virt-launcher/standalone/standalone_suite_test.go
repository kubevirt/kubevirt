package standalone

import (
	"testing"

	"kubevirt.io/client-go/testutils"
)

func TestVirtLauncher(t *testing.T) {
	testutils.KubeVirtTestSuiteSetup(t)
}
