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
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/virtctl/clientconfig"
)

func NewEvacuateCancelCommand() *cobra.Command {
	c := &evacuateCancelCommand{}

	cmd := &cobra.Command{
		Use:     "evacuate-cancel (vm <vm-name> | vmi <vmi-name>)",
		Short:   "Cancel evacuation for a VM, VMI, or all VMIs on a node",
		Example: usageEvacuateCancel(),
		Args:    cobra.ExactArgs(2),
		RunE:    c.Run,
	}

	cmd.Flags().BoolVar(&dryRun, dryRunArg, false, dryRunCommandUsage)

	return cmd
}

func usageEvacuateCancel() string {
	return `  # Cancel evacuation for a virtual machine
  {{ProgramName}} evacuate-cancel vm my-vm

  # Cancel evacuation for a virtual machine instance
  {{ProgramName}} evacuate-cancel vmi my-vmi

  # Cancel evacuation and also cancel an active migration for the VMI
  {{ProgramName}} evacuate-cancel vmi my-vmi --migrate-cancel
  `
}

type evacuateCancelCommand struct {
	cmd        *cobra.Command
	virtClient kubecli.KubevirtClient
}

func (c *evacuateCancelCommand) Run(cmd *cobra.Command, args []string) error {
	virtClient, namespace, _, err := clientconfig.ClientAndNamespaceFromContext(cmd.Context())
	if err != nil {
		return err
	}
	c.cmd = cmd
	c.virtClient = virtClient

	kind := args[0]
	name := args[1]

	handler, err := c.getHandler(kind)
	if err != nil {
		return err
	}

	opts := &virtv1.EvacuateCancelOptions{
		DryRun: setDryRunOption(dryRun),
	}

	return handler(name, namespace, opts)
}

func (c *evacuateCancelCommand) getHandler(kind string) (func(name, namespace string, opts *virtv1.EvacuateCancelOptions) error, error) {
	switch strings.ToLower(kind) {
	case "vm", "vms", "virtualmachine", "virtualmachines":
		return c.handleVM, nil
	case "vmi", "vmis", "virtualmachineinstance", "virtualmachineinstances":
		return c.handleVMI, nil
	}
	return nil, fmt.Errorf("unsupported resource type %q", kind)
}

func (c *evacuateCancelCommand) handleVM(name, namespace string, opts *virtv1.EvacuateCancelOptions) error {
	err := c.virtClient.VirtualMachine(namespace).EvacuateCancel(c.cmd.Context(), name, opts)
	if err != nil {
		return fmt.Errorf("error canceling evacuation for VM %s/%s: %w", namespace, name, err)
	}
	c.cmd.Printf("VM %s/%s was canceled evacuation\n", namespace, name)
	return nil
}

func (c *evacuateCancelCommand) handleVMI(name, namespace string, opts *virtv1.EvacuateCancelOptions) error {
	err := c.virtClient.VirtualMachineInstance(namespace).EvacuateCancel(c.cmd.Context(), name, opts)
	if err != nil {
		return fmt.Errorf("error canceling evacuation for VMI %s/%s: %w", namespace, name, err)
	}
	c.cmd.Printf("VMI %s/%s was canceled evacuation\n", namespace, name)
	return nil
}
