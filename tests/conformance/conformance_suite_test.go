package main

import (
	"testing"

	"kubevirt.io/client-go/testutils"
)

func TestConformance(t *testing.T) {
	testutils.KubeVirtTestSuiteSetup(t)
}
