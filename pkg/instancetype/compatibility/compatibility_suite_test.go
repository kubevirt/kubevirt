/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package compatibility_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestCompatibility(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Compatibility Suite")
}
