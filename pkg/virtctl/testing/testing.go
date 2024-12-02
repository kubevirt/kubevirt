package testing

import (
	"bytes"

	"kubevirt.io/kubevirt/pkg/virtctl"
)

func NewRepeatableVirtctlCommand(args ...string) func() error {
	return func() error {
		cmd, _ := virtctl.NewVirtctlCommand()
		cmd.SetArgs(args)
		return cmd.Execute()
	}
}

func NewRepeatableVirtctlCommandWithOut(args ...string) func() ([]byte, error) {
	return func() ([]byte, error) {
		out := &bytes.Buffer{}
		cmd, _ := virtctl.NewVirtctlCommand()
		cmd.SetArgs(args)
		cmd.SetOut(out)
		err := cmd.Execute()
		return out.Bytes(), err
	}
}
