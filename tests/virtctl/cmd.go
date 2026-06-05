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
 *
 */

package virtctl

import (
	"bytes"
	"flag"

	"github.com/spf13/cobra"

	"kubevirt.io/kubevirt/pkg/virtctl"
)

func newVirtctlCommand(extraArgs ...string) *cobra.Command {
	var args []string
	if server := flag.Lookup("server"); server != nil && server.Value.String() != "" {
		args = append(args, "--server="+server.Value.String())
	}
	if kubeconfig := flag.Lookup("kubeconfig"); kubeconfig != nil && kubeconfig.Value.String() != "" {
		args = append(args, "--kubeconfig="+kubeconfig.Value.String())
	}
	cmd := virtctl.NewVirtctlCommand()
	cmd.SetArgs(append(args, extraArgs...))
	return cmd
}

func newRepeatableVirtctlCommand(args ...string) func() error {
	return func() error {
		return newVirtctlCommand(args...).Execute()
	}
}

func newRepeatableVirtctlCommandWithOut(args ...string) func() ([]byte, error) {
	return func() ([]byte, error) {
		out := &bytes.Buffer{}
		cmd := newVirtctlCommand(args...)
		cmd.SetOut(out)
		err := cmd.Execute()
		return out.Bytes(), err
	}
}
