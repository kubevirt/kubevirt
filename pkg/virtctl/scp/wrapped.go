/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright The KubeVirt Authors.
 */

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
	target.WriteString(remote.Kind)
	target.WriteString(".")
	target.WriteString(remote.Name)
	target.WriteString(".")
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
