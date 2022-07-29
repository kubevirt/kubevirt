//go:build excludenative

package scp

import (
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

func (o *SCP) nativeSCP(_ templates.LocalSCPArgument, _ templates.RemoteSCPArgument, _ bool) error {
	panic("Native SCP is unsupported in this build!")
}
