// SPDX-License-Identifier: Apache-2.0

package virtiofs_placeholder_test

import (
	"flag"
	"testing"

	"kubevirt.io/client-go/testutils"
)

var placeholderBinary string

func init() {
	flag.StringVar(&placeholderBinary, "placeholder-binary", "_out/cmd/virtiofs-placeholder", "path to virtiofs placeholder binary")
}

func TestVirtiofsPlaceholder(t *testing.T) {
	testutils.KubeVirtTestSuiteSetup(t)
}
