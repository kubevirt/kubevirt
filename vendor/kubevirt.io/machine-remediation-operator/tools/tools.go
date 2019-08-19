// Package tools contains code generation utilities
// This package imports things required by build scripts, to force `go mod` to see them as dependencies
package tools

import (
	_ "k8s.io/code-generator/cmd/client-gen"
	_ "k8s.io/code-generator/cmd/deepcopy-gen"
	_ "sigs.k8s.io/controller-tools/cmd/controller-gen"
)
