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

package template

import (
	"github.com/spf13/cobra"

	"kubevirt.io/kubevirt/pkg/virtctl/template/convert"
	"kubevirt.io/kubevirt/pkg/virtctl/template/create"
	"kubevirt.io/kubevirt/pkg/virtctl/template/process"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

const (
	TEMPLATE = "template"
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   TEMPLATE,
		Short: "Work with VirtualMachineTemplates",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Printf("%s", cmd.UsageString())
		},
	}

	cmd.AddCommand(process.NewCommand())
	cmd.SetUsageTemplate(templates.UsageTemplate())

	return cmd
}
