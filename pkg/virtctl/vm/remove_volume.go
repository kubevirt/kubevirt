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

const COMMAND_REMOVEVOLUME = "removevolume"

func NewRemoveVolumeCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "removevolume VMI",
		Short:   "remove a volume from a running VM",
		Example: usageRemoveVolume(),
		Args:    templates.ExactArgs("removevolume", 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := Command{clientConfig: clientConfig}
			return c.removeVolumeRun(args)
		},
	}
	cmd.SetUsageTemplate(templates.UsageTemplate())
	cmd.Flags().StringVar(&volumeName, volumeNameArg, "", "name used in volumes section of spec")
	cmd.MarkFlagRequired(volumeNameArg)
	cmd.Flags().BoolVar(&persist, persistArg, false, "if set, the added volume will be persisted in the VM spec (if it exists)")
	cmd.Flags().BoolVar(&dryRun, dryRunArg, false, dryRunCommandUsage)
	return cmd
}

func usageRemoveVolume() string {
	return `  #Remove volume that was dynamically attached to a running VM.
  {{ProgramName}} removevolume fedora-dv --volume-name=example-dv

  #Remove volume dynamically attached to a running VM and persisting it in the VM spec.
  {{ProgramName}} removevolume fedora-dv --volume-name=example-dv --persist
  `
}

func (o *Command) removeVolumeRun(args []string) error {
	var err error
	vmiName := args[0]

	virtClient, namespace, err := GetNamespaceAndClient(o.clientConfig)
	if err != nil {
		return err
	}

	dryRunOption := setDryRunOption(dryRun)

	if !persist {
		err = virtClient.VirtualMachineInstance(namespace).RemoveVolume(context.Background(), vmiName, &v1.RemoveVolumeOptions{
			Name:   volumeName,
			DryRun: dryRunOption,
		})
	} else {
		err = virtClient.VirtualMachine(namespace).RemoveVolume(context.Background(), vmiName, &v1.RemoveVolumeOptions{
			Name:   volumeName,
			DryRun: dryRunOption,
		})
	}
	if err != nil {
		return fmt.Errorf("error removing volume, %v", err)
	}
	fmt.Printf("Successfully submitted remove volume request to VM %s for volume %s\n", vmiName, volumeName)
	return nil
}
