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
 * Copyright 2019 Red Hat, Inc.
 *
 */

package pause

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

const (
	COMMAND_PAUSE     = "pause"
	COMMAND_UNPAUSE   = "unpause"
	COMMAND_VM_SHORT  = "vm"
	COMMAND_VM_LONG   = "virtualmachine"
	COMMAND_VMI_SHORT = "vmi"
	COMMAND_VMI_LONG  = "virtualmachineinstance"
)

func NewPauseCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "pause vm|vmi (VM)|(VMI)",
		Short:   "Pause a virtual machine",
		Example: usageParent(COMMAND_PAUSE),
	}
	cmd.SetUsageTemplate(templates.UsageTemplate())
	cmd.AddCommand(NewChildCommands(COMMAND_PAUSE, clientConfig)...)
	return cmd
}

func NewUnpauseCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "unpause vm|vmi (VM)|(VMI)",
		Short:   "Unpause a virtual machine",
		Example: usageParent(COMMAND_UNPAUSE),
	}
	cmd.SetUsageTemplate(templates.UsageTemplate())
	cmd.AddCommand(NewChildCommands(COMMAND_UNPAUSE, clientConfig)...)
	return cmd
}

func NewChildCommands(parentCommand string, clientConfig clientcmd.ClientConfig) []*cobra.Command {
	commands := make([]*cobra.Command, 0)
	commands = append(commands, NewChildCommand(parentCommand, COMMAND_VM_SHORT, "(VM)", clientConfig))
	commands = append(commands, NewChildCommand(parentCommand, COMMAND_VM_LONG, "(VM)", clientConfig))
	commands = append(commands, NewChildCommand(parentCommand, COMMAND_VMI_SHORT, "(VMI)", clientConfig))
	commands = append(commands, NewChildCommand(parentCommand, COMMAND_VMI_LONG, "(VMI)", clientConfig))
	return commands
}

func NewChildCommand(parentCommand string, command string, arg string, clientConfig clientcmd.ClientConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:     fmt.Sprintf("%s %s", command, arg),
		Short:   fmt.Sprintf("%s a virtual machine.", strings.Title(parentCommand)),
		Args:    cobra.ExactArgs(1),
		Example: usageChild(parentCommand, command),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := VirtCommand{
				parentCommand: parentCommand,
				childCommand:  command,
				clientConfig:  clientConfig,
			}
			return c.Run(cmd, args)
		},
	}
	cmd.SetUsageTemplate(templates.UsageTemplate())
	return cmd
}

func usageParent(cmd string) string {
	usage := fmt.Sprintf("  # %s a virtual machine called 'myvm':\n", strings.Title(cmd))
	usage += fmt.Sprintf("  {{ProgramName}} %s vm|vmi myvm", cmd)
	return usage
}

func usageChild(parent string, cmd string) string {
	usage := fmt.Sprintf("  # %s a virtual machine called 'myvm':\n", strings.Title(parent))
	usage += fmt.Sprintf("  {{ProgramName}} %s %s myvm", parent, cmd)
	return usage
}

type VirtCommand struct {
	clientConfig  clientcmd.ClientConfig
	parentCommand string
	childCommand  string
}

func (vc *VirtCommand) Run(cmd *cobra.Command, args []string) error {

	resourceName := args[0]
	namespace, _, err := vc.clientConfig.Namespace()
	if err != nil {
		return err
	}

	virtClient, err := kubecli.GetKubevirtClientFromClientConfig(vc.clientConfig)
	if err != nil {
		return fmt.Errorf("Cannot obtain KubeVirt client: %v", err)
	}

	switch vc.parentCommand {
	case COMMAND_PAUSE:
		switch vc.childCommand {
		case COMMAND_VM_LONG, COMMAND_VM_SHORT:
			vm, err := virtClient.VirtualMachine(namespace).Get(resourceName, &v1.GetOptions{})
			if err != nil {
				return fmt.Errorf("Error getting VirtualMachine %s: %v", resourceName, err)
			}
			vmiName := vm.Name
			if vm.Spec.Template != nil && vm.Spec.Template.ObjectMeta.Name != "" {
				vmiName = vm.Spec.Template.ObjectMeta.Name
			}
			err = virtClient.VirtualMachineInstance(namespace).Pause(vmiName)
			if err != nil {
				return fmt.Errorf("Error pausing VirtualMachineInstance %s: %v", vmiName, err)
			}
			fmt.Printf("VMI %s was scheduled to %s\n", vmiName, vc.parentCommand)
		case COMMAND_VMI_LONG, COMMAND_VMI_SHORT:
			err = virtClient.VirtualMachineInstance(namespace).Pause(resourceName)
			if err != nil {
				return fmt.Errorf("Error pausing VirtualMachineInstance %s: %v", resourceName, err)
			}
			fmt.Printf("VMI %s was scheduled to %s\n", resourceName, vc.parentCommand)
		}
	case COMMAND_UNPAUSE:
		switch vc.childCommand {
		case COMMAND_VM_LONG, COMMAND_VM_SHORT:
			vm, err := virtClient.VirtualMachine(namespace).Get(resourceName, &v1.GetOptions{})
			if err != nil {
				return fmt.Errorf("Error getting VirtualMachine %s: %v", resourceName, err)
			}
			vmiName := vm.Name
			if vm.Spec.Template != nil && vm.Spec.Template.ObjectMeta.Name != "" {
				vmiName = vm.Spec.Template.ObjectMeta.Name
			}
			err = virtClient.VirtualMachineInstance(namespace).Unpause(vmiName)
			if err != nil {
				return fmt.Errorf("Error unpausing VirtualMachineInstance %s: %v", vmiName, err)
			}
			fmt.Printf("VMI %s was scheduled to %s\n", vmiName, vc.parentCommand)
		case COMMAND_VMI_LONG, COMMAND_VMI_SHORT:
			err = virtClient.VirtualMachineInstance(namespace).Unpause(resourceName)
			if err != nil {
				return fmt.Errorf("Error unpausing VirtualMachineInstance %s: %v", resourceName, err)
			}
			fmt.Printf("VMI %s was scheduled to %s\n", resourceName, vc.parentCommand)
		}
	}
	return nil
}
