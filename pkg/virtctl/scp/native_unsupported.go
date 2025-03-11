//go:build excludenative

package scp

func (o *SCP) nativeSCP(_ *LocalArgument, _ *RemoteArgument, _ bool) error {
	panic("Native SCP is unsupported in this build!")
}
