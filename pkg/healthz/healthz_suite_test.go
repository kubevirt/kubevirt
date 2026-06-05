package healthz

import (
	"kubevirt.io/client-go/testutils"

	"testing"
)

func TestHealthz(t *testing.T) {
	testutils.KubeVirtTestSuiteSetup(t)
}
