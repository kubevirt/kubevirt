package scp

import "strings"

func (o *SCP) buildSCPTarget(local *LocalArgument, remote *RemoteArgument, toRemote bool) (opts []string) {
	if o.recursive {
		opts = append(opts, "-r")
	}
	if o.preserve {
		opts = append(opts, "-p")
	}

	target := strings.Builder{}
	if len(o.options.SSHUsername) > 0 {
		target.WriteString(o.options.SSHUsername)
		target.WriteRune('@')
	}
	target.WriteString(remote.Name)
	target.WriteRune('.')
	target.WriteString(remote.Namespace)
	target.WriteRune(':')
	target.WriteString(remote.Path)

	if toRemote {
		opts = append(opts, local.Path, target.String())
	} else {
		opts = append(opts, target.String(), local.Path)
	}
	return
}
