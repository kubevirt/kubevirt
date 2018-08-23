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

package install

import (
	"github.com/spf13/cobra"

	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

func InstallCommand(rootCommand *cobra.Command) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "install",
		Short:   "Install virtctl as a kubectl plugin",
		Example: usage(),
		Args:    cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			i := Install{rootCommand: rootCommand}
			return i.Run(cmd, args)
		},
	}
	cmd.SetUsageTemplate(templates.UsageTemplate())
	return cmd
}

func usage() string {
	usage := "# Install virtctl as a kubectl plugin \n"
	usage += "virtctl install"
	return usage
}

type Install struct {
	rootCommand *cobra.Command
}

func (I *Install) Run(cmd *cobra.Command, args []string) error {
	return kubecli.InstallVirtPlugin(I.rootCommand)
}
