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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package version

import (
	"fmt"

	"kubevirt.io/kubevirt/pkg/virtctl/util"

	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"

	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/version"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

var clientOnly bool

func VersionCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "version",
		Short:   "Print the client and server version information.",
		Example: usage(),
		Args:    cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			util.LoadEnvVariables(cmd)
			v := Version{clientConfig: clientConfig}
			return v.Run(cmd, args)
		},
	}
	cmd.Flags().BoolVarP(&clientOnly, "client", "c", clientOnly, "Client version only (no server required).")
	cmd.SetUsageTemplate(templates.UsageTemplate())
	return cmd
}

func usage() string {
	usage := "  # Print the client and server versions for the current context:\n"
	usage += "  virtctl version"
	return usage
}

type Version struct {
	clientConfig clientcmd.ClientConfig
}

func (v *Version) Run(cmd *cobra.Command, args []string) error {
	fmt.Printf("Client Version: %s\n", fmt.Sprintf("%#v", version.Get()))

	if !clientOnly {
		virCli, err := kubecli.GetKubevirtClientFromClientConfig(v.clientConfig)
		if err != nil {
			return err
		}

		test, _ := v.clientConfig.ClientConfig()
		fmt.Printf("%s\n", test.Host)

		serverInfo, err := virCli.ServerVersion().Get()
		if err != nil {
			return err
		}

		fmt.Printf("Server Version: %s\n", fmt.Sprintf("%#v", *serverInfo))
	}

	return nil
}
