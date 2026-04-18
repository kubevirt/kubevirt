/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package vmliveupdate_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestVMLiveUpdate(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "VMLiveUpdate Suite")
}
