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
	"k8s.io/client-go/tools/clientcmd"

	"github.com/spf13/cobra"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

const (
	COMMAND_MIGRATE_CANCEL = "migrate-cancel"
	COMMAND_GUESTOSINFO    = "guestosinfo"
	COMMAND_USERLIST       = "userlist"
	COMMAND_FSLIST         = "fslist"
	COMMAND_ADDVOLUME      = "addvolume"

	volumeNameArg         = "volume-name"
	notDefinedGracePeriod = -1
	dryRunCommandUsage    = "--dry-run=false: Flag used to set whether to perform a dry run or not. If true the command will be executed without performing any changes."

	dryRunArg      = "dry-run"
	forceArg       = "force"
	gracePeriodArg = "grace-period"
	serialArg      = "serial"
	persistArg     = "persist"
	cacheArg       = "cache"

	YAML = "yaml"
	JSON = "json"
)

var (
	vmiName      string
	forceRestart bool
	gracePeriod  int64
	volumeName   string
	serial       string
	persist      bool
	dryRun       bool
	cache        string
)

func NewMigrateCancelCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "migrate-cancel (VM)",
		Short:   "Cancel migration of a virtual machine.",
		Example: usage(COMMAND_MIGRATE_CANCEL),
		Args:    templates.ExactArgs("migrate-cancel", 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := Command{command: COMMAND_MIGRATE_CANCEL, clientConfig: clientConfig}
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
	cmd.Flags().StringVar(&serial, serialArg, "", "serial number you want to assign to the disk")
	cmd.Flags().StringVar(&cache, cacheArg, "", "caching options attribute control the cache mechanism")
	cmd.Flags().BoolVar(&persist, persistArg, false, "if set, the added volume will be persisted in the VM spec (if it exists)")
	cmd.Flags().BoolVar(&dryRun, dryRunArg, false, dryRunCommandUsage)

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
	cmd          *cobra.Command
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
	return `  #Dynamically attach a volume to a running VM.
  {{ProgramName}} addvolume fedora-dv --volume-name=example-dv

  #Dynamically attach a volume to a running VM giving it a serial number to identify the volume inside the guest.
  {{ProgramName}} addvolume fedora-dv --volume-name=example-dv --serial=1234567890

  #Dynamically attach a volume to a running VM and persisting it in the VM spec. At next VM restart the volume will be attached like any other volume.
  {{ProgramName}} addvolume fedora-dv --volume-name=example-dv --persist

  #Dynamically attach a volume with 'none' cache attribute to a running VM.
  {{ProgramName}} addvolume fedora-dv --volume-name=example-dv --cache=none
  `
}

func addVolume(vmiName, volumeName, namespace string, virtClient kubecli.KubevirtClient, dryRunOption *[]string) error {
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
		DryRun:       *dryRunOption,
	}
	if serial != "" {
		hotplugRequest.Disk.Serial = serial
	} else {
		hotplugRequest.Disk.Serial = volumeName
	}
	if cache != "" {
		hotplugRequest.Disk.Cache = v1.DriverCache(cache)
		// Verify if cache mode is valid
		if hotplugRequest.Disk.Cache != v1.CacheNone &&
			hotplugRequest.Disk.Cache != v1.CacheWriteThrough &&
			hotplugRequest.Disk.Cache != v1.CacheWriteBack {
			return fmt.Errorf("error adding volume, invalid cache value %s", cache)
		}
	}
	if !persist {
		err = virtClient.VirtualMachineInstance(namespace).AddVolume(context.Background(), vmiName, hotplugRequest)
	} else {
		err = virtClient.VirtualMachine(namespace).AddVolume(context.Background(), vmiName, hotplugRequest)
	}
	if err != nil {
		return fmt.Errorf("error adding volume, %v", err)
	}
	fmt.Printf("Successfully submitted add volume request to VM %s for volume %s\n", vmiName, volumeName)
	return nil
}

func gracePeriodIsSet(period int64) bool {
	return period != notDefinedGracePeriod
}

func getDryRunOption(dryRun bool) []string {
	if dryRun {
		return []string{metav1.DryRunAll}
	}
	return nil
}

func (o *Command) Run(args []string) error {
	if len(args) != 0 {
		vmiName = args[0]
	}

	namespace, _, err := o.clientConfig.Namespace()
	if err != nil {
		return err
	}

	virtClient, err := kubecli.GetKubevirtClientFromClientConfig(o.clientConfig)
	if err != nil {
		return fmt.Errorf("Cannot obtain KubeVirt client: %v", err)
	}

	dryRunOption := getDryRunOption(dryRun)
	if len(dryRunOption) > 0 && dryRunOption[0] == metav1.DryRunAll {
		fmt.Printf("Dry Run execution\n")
	}
	switch o.command {
	case COMMAND_MIGRATE_CANCEL:
		// get a list of migrations for vmiName (use LabelSelector filter)
		labelselector := fmt.Sprintf("%s==%s", v1.MigrationSelectorLabel, vmiName)
		migrations, err := virtClient.VirtualMachineInstanceMigration(namespace).List(&metav1.ListOptions{
			LabelSelector: labelselector})
		if err != nil {
			return fmt.Errorf("Error fetching virtual machine instance migration list  %v", err)
		}

		// There may be a single active migrations but several completed/failed ones
		// go over the migrations list and find the active one
		done := false
		for _, mig := range migrations.Items {
			migname := mig.ObjectMeta.Name

			if !mig.IsFinal() {
				// Cancel the active migration by calling Delete
				err = virtClient.VirtualMachineInstanceMigration(namespace).Delete(migname, &metav1.DeleteOptions{})
				if err != nil {
					return fmt.Errorf("Error canceling migration %s of a VirtualMachine %s: %v", migname, vmiName, err)
				}
				done = true
				break
			}
		}

		if !done {
			return fmt.Errorf("Found no migration to cancel for %s", vmiName)
		}
	case COMMAND_GUESTOSINFO:
		guestosinfo, err := virtClient.VirtualMachineInstance(namespace).GuestOsInfo(context.Background(), vmiName)
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
		userlist, err := virtClient.VirtualMachineInstance(namespace).UserList(context.Background(), vmiName)
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
		fslist, err := virtClient.VirtualMachineInstance(namespace).FilesystemList(context.Background(), vmiName)
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
		return addVolume(args[0], volumeName, namespace, virtClient, &dryRunOption)
	}

	fmt.Printf("VM %s was scheduled to %s\n", vmiName, o.command)

	return nil
}
