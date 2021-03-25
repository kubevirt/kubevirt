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

	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"

	kubevirtV1 "kubevirt.io/client-go/api/v1"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

const (
	COMMAND_PAUSE   = "pause"
	COMMAND_UNPAUSE = "unpause"
	ARG_VM_SHORT    = "vm"
	ARG_VM_LONG     = "virtualmachine"
	ARG_VMI_SHORT   = "vmi"
	ARG_VMI_LONG    = "virtualmachineinstance"
)

func NewPauseCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pause vm|vmi (VM)|(VMI)",
		Short: "Pause a virtual machine",
		Long: `Pauses a virtual machine by freezing it. Machine state is kept in memory.
First argument is the resource type, possible types are (case insensitive, both singular and plural forms) virtualmachineinstance (vmi) or virtualmachine (vm).
Second argument is the name of the resource.`,
		Args:    templates.ExactArgs(COMMAND_PAUSE, 2),
		Example: usage(COMMAND_PAUSE),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := VirtCommand{
				command:      COMMAND_PAUSE,
				clientConfig: clientConfig,
			}
			return c.Run(args)
		},
	}
	cmd.SetUsageTemplate(templates.UsageTemplate())
	return cmd
}

func NewUnpauseCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unpause vm|vmi (VM)|(VMI)",
		Short: "Unpause a virtual machine",
		Long: `Unpauses a virtual machine.
First argument is the resource type, possible types are (case insensitive, both singular and plural forms) virtualmachineinstance (vmi) or virtualmachine (vm).
Second argument is the name of the resource.`,
		Args:    templates.ExactArgs("unpause", 2),
		Example: usage(COMMAND_UNPAUSE),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := VirtCommand{
				command:      COMMAND_UNPAUSE,
				clientConfig: clientConfig,
			}
			return c.Run(args)
		},
	}
	cmd.SetUsageTemplate(templates.UsageTemplate())
	return cmd
}

func usage(cmd string) string {
	usage := fmt.Sprintf("  # %s a virtualmachine called 'myvm':\n", strings.Title(cmd))
	usage += fmt.Sprintf("  {{ProgramName}} %s vm myvm", cmd)
	return usage
}

type VirtCommand struct {
	clientConfig clientcmd.ClientConfig
	command      string
}

func (vc *VirtCommand) Run(args []string) error {
	resourceType := strings.ToLower(args[0])
	resourceName := args[1]
	namespace, _, err := vc.clientConfig.Namespace()
	if err != nil {
		return err
	}

	virtClient, err := kubecli.GetKubevirtClientFromClientConfig(vc.clientConfig)
	if err != nil {
		return fmt.Errorf("Cannot obtain KubeVirt client: %v", err)
	}

	switch vc.command {
	case COMMAND_PAUSE:
		switch resourceType {
		case ARG_VM_LONG, ARG_VM_SHORT:
			vm, err := virtClient.VirtualMachine(namespace).Get(resourceName, &v1.GetOptions{})
			if err != nil {
				return fmt.Errorf("Error getting VirtualMachine %s: %v", resourceName, err)
			}
			vmiName := vm.Name
			err = virtClient.VirtualMachineInstance(namespace).Pause(vmiName)
			if err != nil {
				if errors.IsNotFound(err) {
					runningStrategy, err := vm.RunStrategy()
					if err != nil {
						return fmt.Errorf("Error pausing VirutalMachineInstance %s: %v", vmiName, err)
					}
					if runningStrategy == kubevirtV1.RunStrategyHalted {
						return fmt.Errorf("Error pausing VirtualMachineInstance %s. VirtualMachine %s is not set to run", vmiName, vm.Name)
					}
					return fmt.Errorf("Error pausing VirtualMachineInstance %s, it was not found", vmiName)

				}
				return fmt.Errorf("Error pausing VirutalMachineInstance %s: %v", vmiName, err)
			}
			fmt.Printf("VMI %s was scheduled to %s\n", vmiName, vc.command)
		case ARG_VMI_LONG, ARG_VMI_SHORT:
			err = virtClient.VirtualMachineInstance(namespace).Pause(resourceName)
			if err != nil {
				return fmt.Errorf("Error pausing VirtualMachineInstance %s: %v", resourceName, err)
			}
			fmt.Printf("VMI %s was scheduled to %s\n", resourceName, vc.command)
		}
	case COMMAND_UNPAUSE:
		switch resourceType {
		case ARG_VM_LONG, ARG_VM_SHORT:
			vm, err := virtClient.VirtualMachine(namespace).Get(resourceName, &v1.GetOptions{})
			if err != nil {
				return fmt.Errorf("Error getting VirtualMachine %s: %v", resourceName, err)
			}
			vmiName := vm.Name
			err = virtClient.VirtualMachineInstance(namespace).Unpause(vmiName)
			if err != nil {
				return fmt.Errorf("Error unpausing VirtualMachineInstance %s: %v", vmiName, err)
			}
			fmt.Printf("VMI %s was scheduled to %s\n", vmiName, vc.command)
		case ARG_VMI_LONG, ARG_VMI_SHORT:
			err = virtClient.VirtualMachineInstance(namespace).Unpause(resourceName)
			if err != nil {
				return fmt.Errorf("Error unpausing VirtualMachineInstance %s: %v", resourceName, err)
			}
			fmt.Printf("VMI %s was scheduled to %s\n", resourceName, vc.command)
		}
	}
	return nil
}
