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
 * Copyright 2024 The Kubevirt Authors
 *
 */

package update

import (
	"github.com/spf13/cobra"

	"k8s.io/client-go/tools/clientcmd"

	machinetype "kubevirt.io/kubevirt/pkg/virtctl/update/machine-type"
)

const (
	UPDATE = "update"
)

func NewCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   UPDATE,
		Short: "Update an attribute of one or many VMs.",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Printf(cmd.UsageString())
		},
	}

	cmd.AddCommand(machinetype.NewMachineTypeCommand(clientConfig))

	return cmd
}
