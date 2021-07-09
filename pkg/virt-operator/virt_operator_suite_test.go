package virt_operator

import (
	"testing"

	"kubevirt.io/client-go/testutils"
)

func TestVirtOperator(t *testing.T) {
	testutils.KubeVirtTestSuiteSetup(t)
}
