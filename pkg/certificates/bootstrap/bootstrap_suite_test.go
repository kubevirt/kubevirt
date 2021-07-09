package bootstrap_test

import (
	"testing"

	"kubevirt.io/client-go/testutils"
)

func TestBootstrap(t *testing.T) {
	testutils.KubeVirtTestSuiteSetup(t)
}
