package main

import (
	"testing"

	"kubevirt.io/client-go/testutils"
)

func TestVirtTail(t *testing.T) {
	testutils.KubeVirtTestSuiteSetup(t)
}
