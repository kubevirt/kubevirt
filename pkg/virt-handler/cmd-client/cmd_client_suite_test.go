package cmdclient_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestCmdClient(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CmdClient Suite")
}
