package healthz

import (
	"testing"

	"kubevirt.io/client-go/testutils"
)

func TestHealthz(t *testing.T) {
	testutils.KubeVirtTestSuiteSetup(t)
}
