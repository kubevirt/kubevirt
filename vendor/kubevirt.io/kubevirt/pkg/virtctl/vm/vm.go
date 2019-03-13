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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package vm

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"

	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

const (
	COMMAND_START   = "start"
	COMMAND_STOP    = "stop"
	COMMAND_RESTART = "restart"
)

func NewStartCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "start (VM)",
		Short:   "Start a virtual machine.",
		Example: usage(COMMAND_START),
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := Command{command: COMMAND_START, clientConfig: clientConfig}
			return c.Run(cmd, args)
		},
	}
	cmd.SetUsageTemplate(templates.UsageTemplate())
	return cmd
}

func NewStopCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "stop (VM)",
		Short:   "Stop a virtual machine.",
		Example: usage(COMMAND_STOP),
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := Command{command: COMMAND_STOP, clientConfig: clientConfig}
			return c.Run(cmd, args)
		},
	}
	cmd.SetUsageTemplate(templates.UsageTemplate())
	return cmd
}

func NewRestartCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "restart (VM)",
		Short:   "Restart a virtual machine.",
		Example: usage(COMMAND_RESTART),
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := Command{command: COMMAND_RESTART, clientConfig: clientConfig}
			return c.Run(cmd, args)
		},
	}
	cmd.SetUsageTemplate(templates.UsageTemplate())
	return cmd
}

type Command struct {
	clientConfig clientcmd.ClientConfig
	command      string
}

func NewCommand(command string) *Command {
	return &Command{command: command}
}

func usage(cmd string) string {
	usage := fmt.Sprintf("  # %s a virtual machine called 'myvm':\n", strings.Title(cmd))
	usage += fmt.Sprintf("  virtctl %s myvm", cmd)
	return usage
}

func (o *Command) Run(cmd *cobra.Command, args []string) error {

	vmiName := args[0]

	namespace, _, err := o.clientConfig.Namespace()
	if err != nil {
		return err
	}

	virtClient, err := kubecli.GetKubevirtClientFromClientConfig(o.clientConfig)
	if err != nil {
		return fmt.Errorf("Cannot obtain KubeVirt client: %v", err)
	}

	options := k8smetav1.GetOptions{}
	vm, err := virtClient.VirtualMachine(namespace).Get(vmiName, &options)
	if err != nil {
		return fmt.Errorf("Error fetching VirtualMachine: %v", err)
	}

	switch o.command {
	case COMMAND_STOP, COMMAND_START:
		running := false
		if o.command == COMMAND_START {
			running = true
		}

		if vm.Spec.Running != running {
			bodyStr := fmt.Sprintf("{\"spec\":{\"running\":%t}}", running)

			vm, err := virtClient.VirtualMachine(namespace).Patch(vm.Name, types.MergePatchType,
				[]byte(bodyStr))

			if err != nil {
				return fmt.Errorf("Error updating VirtualMachine: %v", err)
			}

			if vm.Spec.Running != running {
				return fmt.Errorf("Error VirtualMachine running state should be %t but returned: %t", running, vm.Spec.Running)
			}

		} else {
			stateMsg := "stopped"
			if running {
				stateMsg = "running"
			}
			return fmt.Errorf("Error: VirtualMachineInstance '%s' is already %s", vmiName, stateMsg)
		}
	case COMMAND_RESTART:
		err = virtClient.VirtualMachine(namespace).Restart(vmiName)
		if err != nil {
			return fmt.Errorf("Error restarting VirtualMachine %v", err)
		}
	}

	fmt.Printf("VM %s was scheduled to %s\n", vmiName, o.command)
	return nil
}
