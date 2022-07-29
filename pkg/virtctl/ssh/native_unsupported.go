//go:build excludenative

package ssh

import (
	"fmt"

	"github.com/spf13/pflag"
)

const (
	wrapLocalSSHDefault = true
)

func additionalUsage() string {
	return ""
}

func addAdditionalCommandlineArgs(flagset *pflag.FlagSet, opts *SSHOptions) {
	flagset.StringArrayVarP(&opts.AdditionalSSHLocalOptions, additionalOpts, additionalOptsShort, opts.AdditionalSSHLocalOptions,
		fmt.Sprintf(`--%s="-o StrictHostKeyChecking=no" : Additional options to be passed to the local ssh`, additionalOpts))
}

func (o *SSH) nativeSSH(_, _, _ string) error {
	panic("Native SSH is unsupported in this build!")
}
