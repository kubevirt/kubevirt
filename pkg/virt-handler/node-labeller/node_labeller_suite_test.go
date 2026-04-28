//go:build amd64 || s390x

/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package nodelabeller_test

import (
	"testing"

	"kubevirt.io/client-go/testutils"
)

func TestNodeLabeller(t *testing.T) {
	testutils.KubeVirtTestSuiteSetup(t)
}
