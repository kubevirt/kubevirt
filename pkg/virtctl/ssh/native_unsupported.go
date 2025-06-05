//go:build excludenative

package ssh

import (
	"fmt"
	"io"

	"github.com/spf13/pflag"
	"kubevirt.io/client-go/kubecli"
)

func additionalUsage() string {
	return ""
}

func addAdditionalCommandlineArgs(flagset *pflag.FlagSet, opts *SSHOptions) {
	flagset.StringArrayVarP(&opts.AdditionalSSHLocalOptions, additionalOpts, additionalOptsShort, opts.AdditionalSSHLocalOptions,
		fmt.Sprintf(`--%s="-o StrictHostKeyChecking=no" : Additional options to be passed to the local ssh`, additionalOpts))
}

func (o *SSH) nativeSSH(_, _, _ string, _ kubecli.KubevirtClient, _ io.Reader, _, _ io.Writer) error {
	panic("Native SSH is unsupported in this build!")
}
