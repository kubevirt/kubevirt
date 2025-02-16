//go:build excludenative

package scp

import "kubevirt.io/client-go/kubecli"

func (o *SCP) nativeSCP(_ *LocalArgument, _ *RemoteArgument, _ bool, _ kubecli.KubevirtClient) error {
	panic("Native SCP is unsupported in this build!")
}
