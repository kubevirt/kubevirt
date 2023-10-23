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
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

const (
	COMMAND_ADDVOLUME = "addvolume"
	serialArg         = "serial"
	cacheArg          = "cache"
	diskTypeArg       = "disk-type"
)

var (
	serial   string
	cache    string
	diskType string
)

func NewAddVolumeCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "addvolume VMI",
		Short:   "add a volume to a running VM",
		Example: usageAddVolume(),
		Args:    templates.ExactArgs("addvolume", 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := Command{command: COMMAND_ADDVOLUME, clientConfig: clientConfig}
			return c.addVolumeRun(args)
		},
	}
	cmd.SetUsageTemplate(templates.UsageTemplate())
	cmd.Flags().StringVar(&volumeName, volumeNameArg, "", "name used in volumes section of spec")
	cmd.MarkFlagRequired(volumeNameArg)
	cmd.Flags().StringVar(&serial, serialArg, "", "serial number you want to assign to the disk")
	cmd.Flags().StringVar(&cache, cacheArg, "", "caching options attribute control the cache mechanism")
	cmd.Flags().BoolVar(&persist, persistArg, false, "if set, the added volume will be persisted in the VM spec (if it exists)")
	cmd.Flags().BoolVar(&dryRun, dryRunArg, false, dryRunCommandUsage)
	cmd.Flags().StringVar(&diskType, diskTypeArg, "disk", "specifies disk type to be hotplugged (disk/lun). Disk by default.")

	return cmd
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

func (o *Command) addVolumeRun(args []string) error {
	virtClient, namespace, err := GetNamespaceAndClient(o.clientConfig)
	if err != nil {
		return err
	}

	dryRunOption := setDryRunOption(dryRun)

	return addVolume(args[0], volumeName, namespace, virtClient, &dryRunOption)
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

func addVolume(vmiName, volumeName, namespace string, virtClient kubecli.KubevirtClient, dryRunOption *[]string) error {
	volumeSource, err := getVolumeSourceFromVolume(volumeName, namespace, virtClient)
	if err != nil {
		return fmt.Errorf("error adding volume, %v", err)
	}
	hotplugRequest := &v1.AddVolumeOptions{
		Name: volumeName,
		Disk: &v1.Disk{
			DiskDevice: v1.DiskDevice{},
		},
		VolumeSource: volumeSource,
		DryRun:       *dryRunOption,
	}

	switch diskType {
	case "disk":
		hotplugRequest.Disk.DiskDevice.Disk = &v1.DiskTarget{
			Bus: "scsi",
		}
	case "lun":
		hotplugRequest.Disk.DiskDevice.LUN = &v1.LunTarget{
			Bus: "scsi",
		}
	default:
		return fmt.Errorf("Invalid disk type '%s'. Only LUN and Disk are supported.", diskType)
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
