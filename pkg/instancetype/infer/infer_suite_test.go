/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package infer_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestInfer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Infer Suite")
}
