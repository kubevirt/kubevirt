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

const COMMAND_RESTART = "restart"

func NewRestartCommand() *cobra.Command {
	c := Command{command: COMMAND_RESTART}
	cmd := &cobra.Command{
		Use:     "restart (VM)",
		Short:   "Restart a virtual machine.",
		Example: usage(COMMAND_RESTART),
		Args:    cobra.ExactArgs(1),
		RunE:    c.restartRun,
	}
	cmd.Flags().BoolVar(&forceRestart, forceArg, false, "--force=false: Only used when grace-period=0. If true, immediately remove VMI pod from API and bypass graceful deletion. Note that immediate deletion of some resources may result in inconsistency or data loss and requires confirmation.")
	cmd.Flags().Int64Var(&gracePeriod, gracePeriodArg, -1, "--grace-period=-1: Period of time in seconds given to the VMI to terminate gracefully. Can only be set to 0 when --force is true (force deletion). Currently only setting 0 is supported.")
	cmd.Flags().BoolVar(&dryRun, dryRunArg, false, dryRunCommandUsage)
	return cmd
}

func (o *Command) restartRun(cmd *cobra.Command, args []string) error {
	vmiName := args[0]
	errorFmt := "error restarting VirtualMachine: %v"

	virtClient, namespace, _, err := clientconfig.ClientAndNamespaceFromContext(cmd.Context())
	if err != nil {
		return err
	}

	dryRunOption := setDryRunOption(dryRun)
	gracePeriodChanged := cmd.Flags().Changed(gracePeriodArg)

	if gracePeriodChanged != forceRestart {
		return fmt.Errorf("Must both use --force=true and set --grace-period.")
	}

	restartOpts := &v1.RestartOptions{DryRun: dryRunOption}
	if forceRestart {
		restartOpts.GracePeriodSeconds = &gracePeriod
		errorFmt = "error force restarting VirtualMachine: %v"
	}

	err = virtClient.VirtualMachine(namespace).Restart(context.Background(), vmiName, restartOpts)
	if err != nil {
		return fmt.Errorf(errorFmt, err)
	}

	fmt.Printf("VM %s was scheduled to %s\n", vmiName, o.command)

	return nil
}
