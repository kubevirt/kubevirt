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

	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

const COMMAND_STOP = "stop"

func NewStopCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "stop (VM)",
		Short:   "Stop a virtual machine.",
		Example: usage(COMMAND_STOP),
		Args:    templates.ExactArgs("stop", 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := Command{command: COMMAND_STOP, clientConfig: clientConfig}
			return c.stopRun(args, cmd)
		},
	}

	cmd.Flags().BoolVar(&forceRestart, forceArg, false, "--force=false: Only used when grace-period=0. If true, immediately remove VMI pod from API and bypass graceful deletion. Note that immediate deletion of some resources may result in inconsistency or data loss and requires confirmation.")
	cmd.Flags().Int64Var(&gracePeriod, gracePeriodArg, -1, "--grace-period=-1: Period of time in seconds given to the VMI to terminate gracefully. Can only be set to 0 when --force is true (force deletion). Currently only setting 0 is supported.")
	cmd.Flags().BoolVar(&dryRun, dryRunArg, false, dryRunCommandUsage)
	cmd.SetUsageTemplate(templates.UsageTemplate())
	return cmd
}

func (o *Command) stopRun(args []string, cmd *cobra.Command) error {
	vmiName := args[0]

	virtClient, namespace, err := GetNamespaceAndClient(o.clientConfig)
	if err != nil {
		return err
	}

	dryRunOption := setDryRunOption(dryRun)
	gracePeriodChanged := cmd.Flags().Changed(gracePeriodArg)

	if gracePeriodChanged != forceRestart {
		return fmt.Errorf("Must both use --force=true and set --grace-period.")
	}

	if forceRestart {
		err = virtClient.VirtualMachine(namespace).ForceStop(context.Background(), vmiName, &v1.StopOptions{GracePeriod: &gracePeriod, DryRun: dryRunOption})
		if err != nil {
			return fmt.Errorf("Error force stopping VirtualMachine, %v", err)
		}
	} else {
		err = virtClient.VirtualMachine(namespace).Stop(context.Background(), vmiName, &v1.StopOptions{DryRun: dryRunOption})
		if err != nil {
			return fmt.Errorf("Error stopping VirtualMachine %v", err)
		}
	}

	fmt.Printf("VM %s was scheduled to %s\n", vmiName, o.command)

	return nil
}
