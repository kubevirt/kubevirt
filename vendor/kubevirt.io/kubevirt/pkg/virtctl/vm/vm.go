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
	"encoding/json"
	"fmt"
	"strings"

	v1 "kubevirt.io/client-go/api/v1"

	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

const (
	COMMAND_START       = "start"
	COMMAND_STOP        = "stop"
	COMMAND_RESTART     = "restart"
	COMMAND_MIGRATE     = "migrate"
	COMMAND_RENAME      = "rename"
	COMMAND_GUESTOSINFO = "guestosinfo"
	COMMAND_USERLIST    = "userlist"
	COMMAND_FSLIST      = "fslist"
)

var (
	forceRestart bool
	gracePeriod  int = -1
)

func NewStartCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "start (VM)",
		Short:   "Start a virtual machine.",
		Example: usage(COMMAND_START),
		Args:    templates.ExactArgs("start", 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := Command{command: COMMAND_START, clientConfig: clientConfig}
			return c.Run(args)
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
		Args:    templates.ExactArgs("stop", 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := Command{command: COMMAND_STOP, clientConfig: clientConfig}
			return c.Run(args)
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
		Args:    templates.ExactArgs("restart", 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := Command{command: COMMAND_RESTART, clientConfig: clientConfig}
			return c.Run(args)
		},
	}
	cmd.Flags().BoolVar(&forceRestart, "force", forceRestart, "--force=false: Only used when grace-period=0. If true, immediately remove VMI pod from API and bypass graceful deletion. Note that immediate deletion of some resources may result in inconsistency or data loss and requires confirmation.")
	cmd.Flags().IntVar(&gracePeriod, "grace-period", gracePeriod, "--grace-period=-1: Period of time in seconds given to the VMI to terminate gracefully. Can only be set to 0 when --force is true (force deletion). Currently only setting 0 is supported.")
	cmd.SetUsageTemplate(templates.UsageTemplate())
	return cmd
}

func NewMigrateCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "migrate (VM)",
		Short:   "Migrate a virtual machine.",
		Example: usage(COMMAND_MIGRATE),
		Args:    templates.ExactArgs("migrate", 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := Command{command: COMMAND_MIGRATE, clientConfig: clientConfig}
			return c.Run(args)
		},
	}
	cmd.SetUsageTemplate(templates.UsageTemplate())
	return cmd
}

func NewRenameCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "rename [vm_name] [new_vm_name]",
		Short:   "Rename a stopped virtual machine.",
		Example: usage(COMMAND_RENAME),
		Args:    templates.ExactArgs("rename", 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := Command{command: COMMAND_RENAME, clientConfig: clientConfig}
			return c.Run(args)
		},
	}
	cmd.SetUsageTemplate(templates.UsageTemplate())
	return cmd
}

func NewGuestOsInfoCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "guestosinfo (VMI)",
		Short:   "Return guest agent info about operating system.",
		Example: usage(COMMAND_GUESTOSINFO),
		Args:    templates.ExactArgs("guestosinfo", 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := Command{command: COMMAND_GUESTOSINFO, clientConfig: clientConfig}
			return c.Run(args)
		},
	}
	cmd.SetUsageTemplate(templates.UsageTemplate())
	return cmd
}

func NewUserListCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "userlist (VMI)",
		Short:   "Return full list of logged in users on the guest machine.",
		Example: usage(COMMAND_USERLIST),
		Args:    templates.ExactArgs("userlist", 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := Command{command: COMMAND_USERLIST, clientConfig: clientConfig}
			return c.Run(args)
		},
	}
	cmd.SetUsageTemplate(templates.UsageTemplate())
	return cmd
}

func NewFSListCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "fslist (VMI)",
		Short:   "Return full list of filesystems available on the guest machine.",
		Example: usage(COMMAND_FSLIST),
		Args:    templates.ExactArgs("fslist", 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := Command{command: COMMAND_FSLIST, clientConfig: clientConfig}
			return c.Run(args)
		},
	}
	cmd.SetUsageTemplate(templates.UsageTemplate())
	return cmd
}

type Command struct {
	clientConfig clientcmd.ClientConfig
	command      string
}

func usage(cmd string) string {
	if cmd == COMMAND_RENAME {
		usage := "	# rename a virtual machine called 'myvm' to 'notmyvm'\n"
		usage += fmt.Sprintf("	{{ProgramName}} %s myvm notmyvm", cmd)
		return usage
	}

	if cmd == COMMAND_USERLIST || cmd == COMMAND_FSLIST || cmd == COMMAND_GUESTOSINFO {
		usage := fmt.Sprintf("  # %s a virtual machine instance called 'myvm':\n", strings.Title(cmd))
		usage += fmt.Sprintf("  {{ProgramName}} %s myvm", cmd)
		return usage
	}

	usage := fmt.Sprintf("  # %s a virtual machine called 'myvm':\n", strings.Title(cmd))
	usage += fmt.Sprintf("  {{ProgramName}} %s myvm", cmd)
	return usage
}

func (o *Command) Run(args []string) error {

	vmiName := args[0]

	namespace, _, err := o.clientConfig.Namespace()
	if err != nil {
		return err
	}

	virtClient, err := kubecli.GetKubevirtClientFromClientConfig(o.clientConfig)
	if err != nil {
		return fmt.Errorf("Cannot obtain KubeVirt client: %v", err)
	}

	switch o.command {
	case COMMAND_START:
		err = virtClient.VirtualMachine(namespace).Start(vmiName)
		if err != nil {
			return fmt.Errorf("Error starting VirtualMachine %v", err)
		}
	case COMMAND_STOP:
		err = virtClient.VirtualMachine(namespace).Stop(vmiName)
		if err != nil {
			return fmt.Errorf("Error stopping VirtualMachine %v", err)
		}
	case COMMAND_RESTART:
		if gracePeriod != -1 && forceRestart == false {
			return fmt.Errorf("Can not set gracePeriod without --force=true")
		}
		if forceRestart {
			if gracePeriod != -1 {
				err = virtClient.VirtualMachine(namespace).ForceRestart(vmiName, gracePeriod)
				if err != nil {
					return fmt.Errorf("Error restarting VirtualMachine, %v", err)
				}
			} else if gracePeriod == -1 {
				return fmt.Errorf("Can not force restart without gracePeriod")
			}
			break
		}
		err = virtClient.VirtualMachine(namespace).Restart(vmiName)
		if err != nil {
			return fmt.Errorf("Error restarting VirtualMachine %v", err)
		}
	case COMMAND_MIGRATE:
		err = virtClient.VirtualMachine(namespace).Migrate(vmiName)
		if err != nil {
			return fmt.Errorf("Error migrating VirtualMachine %v", err)
		}
	case COMMAND_RENAME:
		err = virtClient.VirtualMachine(namespace).Rename(vmiName, &v1.RenameOptions{NewName: args[1]})
		if err != nil {
			return fmt.Errorf("Error renaming VirtualMachine %v", err)
		}
	case COMMAND_GUESTOSINFO:
		guestosinfo, err := virtClient.VirtualMachineInstance(namespace).GuestOsInfo(vmiName)
		if err != nil {
			return fmt.Errorf("Error getting guestosinfo of VirtualMachineInstance %s, %v", vmiName, err)
		}

		data, err := json.MarshalIndent(guestosinfo, "", "  ")
		if err != nil {
			return fmt.Errorf("Cannot marshal guestosinfo %v", err)
		}

		fmt.Printf("%s\n", string(data))
		return nil
	case COMMAND_USERLIST:
		userlist, err := virtClient.VirtualMachineInstance(namespace).UserList(vmiName)
		if err != nil {
			return fmt.Errorf("Error listing users of VirtualMachineInstance %s, %v", vmiName, err)
		}

		data, err := json.MarshalIndent(userlist, "", "  ")
		if err != nil {
			return fmt.Errorf("Cannot marshal userlist %v", err)
		}

		fmt.Printf("%s\n", string(data))
		return nil
	case COMMAND_FSLIST:
		fslist, err := virtClient.VirtualMachineInstance(namespace).FilesystemList(vmiName)
		if err != nil {
			return fmt.Errorf("Error listing filesystems of VirtualMachineInstance %s, %v", vmiName, err)
		}

		data, err := json.MarshalIndent(fslist, "", "  ")
		if err != nil {
			return fmt.Errorf("Cannot marshal filesystem list %v", err)
		}

		fmt.Printf("%s\n", string(data))
		return nil
	}

	fmt.Printf("VM %s was scheduled to %s\n", vmiName, o.command)
	return nil
}
