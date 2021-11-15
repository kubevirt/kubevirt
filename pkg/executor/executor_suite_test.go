package executor_test

import (
	"fmt"
	"testing"

	"kubevirt.io/client-go/testutils"
)

func TestExecutor(t *testing.T) {
	testutils.KubeVirtTestSuiteSetup(t)
}

var testsExecError = fmt.Errorf("error occurred")

func failingCommandStub() func() error {
	return func() error {
		return testsExecError
	}
}
