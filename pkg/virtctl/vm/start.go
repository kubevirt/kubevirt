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

package vm

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virtctl/clientconfig"
)

const (
	COMMAND_START = "start"
	pausedArg     = "paused"
)

var (
	startPaused bool
)

func NewStartCommand() *cobra.Command {
	c := Command{command: COMMAND_START}
	cmd := &cobra.Command{
		Use:     "start (VM)",
		Short:   "Start a virtual machine.",
		Example: usage(COMMAND_START),
		Args:    cobra.ExactArgs(1),
		RunE:    c.startRun,
	}
	cmd.Flags().BoolVar(&startPaused, pausedArg, false, "--paused=false: If set to true, start virtual machine in paused state")
	cmd.Flags().BoolVar(&dryRun, dryRunArg, false, dryRunCommandUsage)
	return cmd
}

func (o *Command) startRun(cmd *cobra.Command, args []string) error {
	vmiName := args[0]

	virtClient, namespace, _, err := clientconfig.ClientAndNamespaceFromContext(cmd.Context())
	if err != nil {
		return err
	}

	dryRunOption := setDryRunOption(dryRun)

	err = virtClient.VirtualMachine(namespace).Start(context.Background(), vmiName, &v1.StartOptions{Paused: startPaused, DryRun: dryRunOption})
	if err != nil {
		return fmt.Errorf("Error starting VirtualMachine %v", err)
	}

	fmt.Printf("VM %s was scheduled to %s\n", vmiName, o.command)

	return nil
}
