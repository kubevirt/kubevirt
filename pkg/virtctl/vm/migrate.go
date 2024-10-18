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
 * Copyright 2023 Red Hat, Inc.
 *
 */

package vm

import (
	"context"
	"fmt"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

const COMMAND_MIGRATE = "migrate"

var nodeName string

func NewMigrateCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {

	cmd := &cobra.Command{
		Use:     "migrate (VM)",
		Short:   "Migrate a virtual machine.",
		Example: usage(COMMAND_MIGRATE),
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := Command{command: COMMAND_MIGRATE, clientConfig: clientConfig}
			return c.migrateRun(args)
		},
	}

	cmd.Flags().StringVar(&nodeName, "nodeName", nodeName, "--nodeName=<nodeName>: Flag to migrate this VM to a specific node regardless of its affinity rules. If it's omitted, recommended, the scheduler becomes responsible for finding the best Node to migrate the VM to.")
	cmd.Flags().BoolVar(&dryRun, dryRunArg, false, dryRunCommandUsage)
	cmd.SetUsageTemplate(templates.UsageTemplate())
	return cmd
}

func (o *Command) migrateRun(args []string) error {
	vmiName := args[0]

	virtClient, namespace, err := GetNamespaceAndClient(o.clientConfig)
	if err != nil {
		return err
	}

	dryRunOption := setDryRunOption(dryRun)

	var nodeSelectorTerm *k8sv1.NodeSelectorTerm
	if nodeName != "" {
		nodeSelectorTerm = &k8sv1.NodeSelectorTerm{
			MatchFields: []k8sv1.NodeSelectorRequirement{
				{
					Key:      metav1.ObjectNameField,
					Operator: k8sv1.NodeSelectorOpIn,
					Values:   []string{nodeName},
				},
			},
		}
	}

	err = virtClient.VirtualMachine(namespace).Migrate(context.Background(), vmiName, &v1.MigrateOptions{DryRun: dryRunOption, AddedNodeSelectorTerm: nodeSelectorTerm})
	if err != nil {
		return fmt.Errorf("Error migrating VirtualMachine %v", err)
	}

	fmt.Printf("VM %s was scheduled to %s\n", vmiName, o.command)

	return nil
}
