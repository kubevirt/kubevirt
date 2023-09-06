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
 * Copyright 2022 Red Hat, Inc.
 *
 */

package create

import (
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"

	"kubevirt.io/kubevirt/pkg/virtctl/create/clone"
	"kubevirt.io/kubevirt/pkg/virtctl/create/instancetype"
	"kubevirt.io/kubevirt/pkg/virtctl/create/preference"
	"kubevirt.io/kubevirt/pkg/virtctl/create/vm"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

const (
	CREATE = "create"
)

func NewCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   CREATE,
		Short: "Create a manifest for the specified Kind.",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Printf(cmd.UsageString())
		},
	}

	cmd.AddCommand(vm.NewCommand(clientConfig))
	cmd.AddCommand(preference.NewCommand(clientConfig))
	cmd.AddCommand(instancetype.NewCommand(clientConfig))
	cmd.AddCommand(clone.NewCommand(clientConfig))
	cmd.SetUsageTemplate(templates.UsageTemplate())

	return cmd
}
