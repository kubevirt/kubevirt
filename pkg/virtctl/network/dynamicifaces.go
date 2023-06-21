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

package network

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"k8s.io/client-go/tools/clientcmd"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

const (
	HotplugCmdName   = "addinterface"
	HotUnplugCmdName = "removeinterface"

	ifaceNameArg                       = "name"
	networkAttachmentDefinitionNameArg = "network-attachment-definition-name"
)

var (
	ifaceName                       string
	networkAttachmentDefinitionName string
)

type dynamicIfacesCmd struct {
	kvClient  kubecli.KubevirtClient
	namespace string
}

func NewAddInterfaceCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "addinterface VM",
		Short:   "add a network interface to a running VM",
		Example: usageAddInterface(),
		Args:    templates.ExactArgs(HotplugCmdName, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newDynamicIfaceCmd(clientConfig)
			if err != nil {
				return fmt.Errorf("error creating the `AddInterface` command: %w", err)
			}
			return c.addInterface(args[0], networkAttachmentDefinitionName, ifaceName)
		},
	}
	cmd.SetUsageTemplate(templates.UsageTemplate())
	cmd.Flags().StringVar(&networkAttachmentDefinitionName, networkAttachmentDefinitionNameArg, "", "The referenced network-attachment-definition name. Format:\n<networkAttachmentDefinitionName>, <ns>/<networkAttachmentDefinitionName>")
	_ = cmd.MarkFlagRequired(networkAttachmentDefinitionNameArg)
	cmd.Flags().StringVar(&ifaceName, ifaceNameArg, "", "Logical name of the interface to be plugged")
	_ = cmd.MarkFlagRequired(ifaceNameArg)

	return cmd
}

func NewRemoveInterfaceCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "removeinterface VM",
		Short:   "remove a network interface from a running VM",
		Example: usageRemoveInterface(),
		Args:    templates.ExactArgs(HotUnplugCmdName, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newDynamicIfaceCmd(clientConfig)
			if err != nil {
				return fmt.Errorf("error creating the `RemoveInterface` command: %w", err)
			}
			return c.removeInterface(args[0], ifaceName)
		},
	}
	cmd.SetUsageTemplate(templates.UsageTemplate())
	cmd.Flags().StringVar(&ifaceName, ifaceNameArg, "", "Logical name of the interface to be plugged")
	_ = cmd.MarkFlagRequired(ifaceNameArg)

	return cmd
}

func usageAddInterface() string {
	usage := `  #Dynamically attach a network interface to a running VM and persisting it in the VM spec. At next VM restart the network interface will be attached like any other network interface.
  {{ProgramName}} addinterface <vm-name> --network-attachment-definition-name <network-attachment-definition name> --name <logical interface name>
  `
	return usage
}

func usageRemoveInterface() string {
	usage := `  #Dynamically detach a network interface from a running VM and persisting it in the VM spec. At next VM restart the network interface won't be attached to the VM.
  {{ProgramName}} removeinterface <vmi-name> --name <logical interface name>
  `
	return usage
}

func newDynamicIfaceCmd(clientCfg clientcmd.ClientConfig) (*dynamicIfacesCmd, error) {
	virtClient, err := kubecli.GetKubevirtClientFromClientConfig(clientCfg)
	if err != nil {
		return nil, fmt.Errorf("cannot obtain KubeVirt client: %v", err)
	}
	namespace, _, err := clientCfg.Namespace()
	if err != nil {
		return nil, err
	}
	return &dynamicIfacesCmd{kvClient: virtClient, namespace: namespace}, nil
}

func (dic *dynamicIfacesCmd) addInterface(vmName string, networkAttachmentDefinitionName string, name string) error {
	return dic.kvClient.VirtualMachine(dic.namespace).AddInterface(
		context.Background(),
		vmName,
		&v1.AddInterfaceOptions{
			NetworkAttachmentDefinitionName: networkAttachmentDefinitionName,
			Name:                            name,
		},
	)
}

func (dic *dynamicIfacesCmd) removeInterface(vmName string, name string) error {
	return dic.kvClient.VirtualMachine(dic.namespace).RemoveInterface(
		context.Background(),
		vmName,
		&v1.RemoveInterfaceOptions{
			Name: name,
		},
	)
}
