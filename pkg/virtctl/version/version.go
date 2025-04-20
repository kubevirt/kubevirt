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

package version

import (
	"fmt"

	"github.com/spf13/cobra"

	client_version "kubevirt.io/client-go/version"

	"kubevirt.io/kubevirt/pkg/virtctl/clientconfig"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

type version struct {
	clientOnly bool
}

func VersionCommand() *cobra.Command {
	v := &version{}
	cmd := &cobra.Command{
		Use:     "version",
		Short:   "Print the client and server version information.",
		Example: usage(),
		Args:    cobra.NoArgs,
		RunE:    v.Run,
	}
	cmd.Flags().BoolVarP(&v.clientOnly, "client", "c", v.clientOnly, "Client version only (no server required).")
	cmd.SetUsageTemplate(templates.UsageTemplate())

	return cmd
}

func usage() string {
	return `  # Print the client and server versions for the current context:
  {{ProgramName}} version`
}

func (v *version) Run(cmd *cobra.Command, _ []string) error {
	cmd.Printf("Client Version: %#v\n", client_version.Get())

	if v.clientOnly {
		return nil
	}

	virtClient, _, _, err := clientconfig.ClientAndNamespaceFromContext(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to get virtClient config: %w", err)
	}

	serverInfo, err := virtClient.ServerVersion().Get()
	if err != nil {
		return fmt.Errorf("failed to get server version: %w", err)
	}

	cmd.Printf("Server Version: %#v\n", *serverInfo)
	return nil
}
