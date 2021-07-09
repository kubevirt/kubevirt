package tools

// Keep a reference to tool binaries in vendor, so that go mod keeps them
import (
	_ "github.com/onsi/ginkgo/ginkgo"
	_ "github.com/wadey/gocovmerge"
	_ "mvdan.cc/sh/v3/cmd/shfmt"
)
