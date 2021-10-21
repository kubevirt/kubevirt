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
	"context"
	"encoding/json"
	"fmt"
	"strings"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/client-go/apis/core/v1"

	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

const (
	COMMAND_START        = "start"
	COMMAND_STOP         = "stop"
	COMMAND_RESTART      = "restart"
	COMMAND_MIGRATE      = "migrate"
	COMMAND_GUESTOSINFO  = "guestosinfo"
	COMMAND_USERLIST     = "userlist"
	COMMAND_FSLIST       = "fslist"
	COMMAND_ADDVOLUME    = "addvolume"
	COMMAND_REMOVEVOLUME = "removevolume"

	volumeNameArg         = "volume-name"
	notDefinedGracePeriod = -1
)

var (
	forceRestart bool
	gracePeriod  int = -1
	volumeName   string
	serial       string
	persist      bool
	startPaused  bool
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
	cmd.Flags().BoolVar(&startPaused, "paused", false, "--paused=false: If set to true, start virtual machine in paused state")
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

	cmd.Flags().BoolVar(&forceRestart, "force", forceRestart, "--force=false: Only used when grace-period=0. If true, immediately remove VMI pod from API and bypass graceful deletion. Note that immediate deletion of some resources may result in inconsistency or data loss and requires confirmation.")
	cmd.Flags().IntVar(&gracePeriod, "grace-period", gracePeriod, "--grace-period=-1: Period of time in seconds given to the VMI to terminate gracefully. Can only be set to 0 when --force is true (force deletion). Currently only setting 0 is supported.")
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

func NewAddVolumeCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "addvolume VMI",
		Short:   "add a volume to a running VM",
		Example: usageAddVolume(),
		Args:    templates.ExactArgs("addvolume", 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := Command{command: COMMAND_ADDVOLUME, clientConfig: clientConfig}
			return c.Run(args)
		},
	}
	cmd.SetUsageTemplate(templates.UsageTemplate())
	cmd.Flags().StringVar(&volumeName, volumeNameArg, "", "name used in volumes section of spec")
	cmd.MarkFlagRequired(volumeNameArg)
	cmd.Flags().StringVar(&serial, "serial", "", "serial number you want to assign to the disk")
	cmd.Flags().BoolVar(&persist, "persist", false, "if set, the added volume will be persisted in the VM spec (if it exists)")

	return cmd
}

func NewRemoveVolumeCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "removevolume VMI",
		Short:   "remove a volume from a running VM",
		Example: usageRemoveVolume(),
		Args:    templates.ExactArgs("removevolume", 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := Command{command: COMMAND_REMOVEVOLUME, clientConfig: clientConfig}
			return c.Run(args)
		},
	}
	cmd.SetUsageTemplate(templates.UsageTemplate())
	cmd.Flags().StringVar(&volumeName, volumeNameArg, "", "name used in volumes section of spec")
	cmd.MarkFlagRequired(volumeNameArg)
	cmd.Flags().BoolVar(&persist, "persist", false, "if set, the added volume will be persisted in the VM spec (if it exists)")
	return cmd
}

func getVolumeSourceFromVolume(volumeName, namespace string, virtClient kubecli.KubevirtClient) (*v1.HotplugVolumeSource, error) {
	//Check if data volume exists.
	_, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(namespace).Get(context.TODO(), volumeName, metav1.GetOptions{})
	if err == nil {
		return &v1.HotplugVolumeSource{
			DataVolume: &v1.DataVolumeSource{
				Name:         volumeName,
				Hotpluggable: true,
			},
		}, nil
	}
	// DataVolume not found, try PVC
	_, err = virtClient.CoreV1().PersistentVolumeClaims(namespace).Get(context.TODO(), volumeName, metav1.GetOptions{})
	if err == nil {
		return &v1.HotplugVolumeSource{
			PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
				PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
					ClaimName: volumeName,
				},
				Hotpluggable: true,
			},
		}, nil
	}
	// Neither return error
	return nil, fmt.Errorf("Volume %s is not a DataVolume or PersistentVolumeClaim", volumeName)
}

type Command struct {
	clientConfig clientcmd.ClientConfig
	command      string
}

func usage(cmd string) string {
	if cmd == COMMAND_USERLIST || cmd == COMMAND_FSLIST || cmd == COMMAND_GUESTOSINFO {
		usage := fmt.Sprintf("  # %s a virtual machine instance called 'myvm':\n", strings.Title(cmd))
		usage += fmt.Sprintf("  {{ProgramName}} %s myvm", cmd)
		return usage
	}

	usage := fmt.Sprintf("  # %s a virtual machine called 'myvm':\n", strings.Title(cmd))
	usage += fmt.Sprintf("  {{ProgramName}} %s myvm", cmd)
	return usage
}

func usageAddVolume() string {
	usage := `  #Dynamically attach a volume to a running VM.
  {{ProgramName}} addvolume fedora-dv --volume-name=example-dv

  #Dynamically attach a volume to a running VM giving it a serial number to identify the volume inside the guest.
  {{ProgramName}} addvolume fedora-dv --volume-name=example-dv --serial=1234567890

  #Dynamically attach a volume to a running VM and persisting it in the VM spec. At next VM restart the volume will be attached like any other volume.
  {{ProgramName}} addvolume fedora-dv --volume-name=example-dv --persist
  `
	return usage
}

func usageRemoveVolume() string {
	usage := `  #Remove volume that was dynamically attached to a running VM.
  {{ProgramName}} removevolume fedora-dv --volume-name=example-dv

  #Remove volume dynamically attached to a running VM and persisting it in the VM spec.
  {{ProgramName}} removevolume fedora-dv --volume-name=example-dv --persist
  `
	return usage
}

func addVolume(vmiName, volumeName, namespace string, virtClient kubecli.KubevirtClient) error {
	volumeSource, err := getVolumeSourceFromVolume(volumeName, namespace, virtClient)
	if err != nil {
		return fmt.Errorf("error adding volume, %v", err)
	}
	hotplugRequest := &v1.AddVolumeOptions{
		Name: volumeName,
		Disk: &v1.Disk{
			DiskDevice: v1.DiskDevice{
				Disk: &v1.DiskTarget{
					Bus: "scsi",
				},
			},
		},
		VolumeSource: volumeSource,
	}
	if serial != "" {
		hotplugRequest.Disk.Serial = serial
	} else {
		hotplugRequest.Disk.Serial = volumeName
	}
	if !persist {
		err = virtClient.VirtualMachineInstance(namespace).AddVolume(vmiName, hotplugRequest)
	} else {
		err = virtClient.VirtualMachine(namespace).AddVolume(vmiName, hotplugRequest)
	}
	if err != nil {
		return fmt.Errorf("error adding volume, %v", err)
	}
	fmt.Printf("Successfully submitted add volume request to VM %s for volume %s\n", vmiName, volumeName)
	return nil
}

func removeVolume(vmiName, volumeName, namespace string, virtClient kubecli.KubevirtClient) error {
	var err error
	if !persist {
		err = virtClient.VirtualMachineInstance(namespace).RemoveVolume(vmiName, &v1.RemoveVolumeOptions{
			Name: volumeName,
		})
	} else {
		err = virtClient.VirtualMachine(namespace).RemoveVolume(vmiName, &v1.RemoveVolumeOptions{
			Name: volumeName,
		})
	}
	if err != nil {
		return fmt.Errorf("error removing volume, %v", err)
	}
	fmt.Printf("Successfully submitted remove volume request to VM %s for volume %s\n", vmiName, volumeName)
	return nil
}

func gracePeriodIsSet(period int) bool {
	return period != notDefinedGracePeriod
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
		err = virtClient.VirtualMachine(namespace).Start(vmiName, &v1.StartOptions{Paused: startPaused})
		if err != nil {
			return fmt.Errorf("Error starting VirtualMachine %v", err)
		}
	case COMMAND_STOP:
		if gracePeriodIsSet(gracePeriod) && forceRestart == false {
			return fmt.Errorf("Can not set gracePeriod without --force=true")
		}
		if forceRestart {
			if gracePeriodIsSet(gracePeriod) {
				err = virtClient.VirtualMachine(namespace).ForceStop(vmiName, gracePeriod)
				if err != nil {
					return fmt.Errorf("Error force stoping VirtualMachine, %v", err)
				}
			} else if !gracePeriodIsSet(gracePeriod) {
				return fmt.Errorf("Can not force stop without gracePeriod")
			}
			break
		}
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
	case COMMAND_ADDVOLUME:
		return addVolume(args[0], volumeName, namespace, virtClient)
	case COMMAND_REMOVEVOLUME:
		return removeVolume(args[0], volumeName, namespace, virtClient)
	}

	fmt.Printf("VM %s was scheduled to %s\n", vmiName, o.command)
	return nil
}
