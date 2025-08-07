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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/pflag"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/libvmi"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const (
	gaNotAvailableError = "Guest agent not available for now"
	windowsOS           = "windows"
	freezeLimitReached  = "fsfreeze is limited"
)

type FreezerConfig struct {
	Freeze                 bool
	Unfreeze               bool
	Name                   string
	Namespace              string
	UnfreezeTimeoutSeconds int32
}

func getGrpcClient() (cmdclient.LauncherClient, error) {
	sockFile := "/run/kubevirt/sockets/launcher-sock"
	client, err := cmdclient.NewClient(sockFile)
	if err != nil {
		log.Log.Reason(err).Error("Failed to connect launcher")
		return nil, err
	}
	return client, nil
}

func shouldFreezeVirtualMachine(client cmdclient.LauncherClient) (bool, error) {
	domain, exists, err := client.GetDomain()
	if err != nil {
		log.Log.Reason(err).Error("Failed to get domain")
		return false, err
	}
	return exists && domain.Status.Status == api.Running, nil
}

func parseFlags() (*FreezerConfig, error) {
	freeze := pflag.Bool("freeze", false, "Freeze VM")
	unfreeze := pflag.Bool("unfreeze", false, "Unfreeze VM")
	name := pflag.String("name", "", "Name of the VirtualMachineInstance")
	namespace := pflag.String("namespace", "", "Namespace of the VirtualMachineInstance")
	unfreezeTimeoutSeconds := pflag.Int32("unfreezeTimeoutSeconds", 300, "Timeout in seconds to automatically unfreeze the VirtualMachineInstance")

	pflag.Parse()

	if !*freeze && !*unfreeze {
		return nil, fmt.Errorf("either --freeze or --unfreeze must be set")
	}
	if name == nil || namespace == nil || *name == "" || *namespace == "" {
		return nil, fmt.Errorf("both --name and --namespace must be provided")
	}

	return &FreezerConfig{
		Freeze:                 *freeze,
		Unfreeze:               *unfreeze,
		Name:                   *name,
		Namespace:              *namespace,
		UnfreezeTimeoutSeconds: *unfreezeTimeoutSeconds,
	}, nil
}

func run(config *FreezerConfig, client cmdclient.LauncherClient) error {
	vmi := libvmi.New(libvmi.WithName(config.Name), libvmi.WithNamespace(config.Namespace))

	info, err := client.GetGuestInfo(vmi, []string{})
	if err != nil {
		log.Log.Reason(err).Error("Failed to get guest info")
		return err
	}

	if info.GAVersion == "" {
		log.Log.Info("No guest agent, exiting")
		return nil
	}

	log.Log.Infof("Guest agent version is %s", info.GAVersion)

	shouldFreeze, err := shouldFreezeVirtualMachine(client)
	if err != nil {
		return err
	}
	if !shouldFreeze {
		log.Log.Info("VM domain not running, no need to freeze/unfreeze")
		return nil
	}

	if config.Freeze {
		err = client.FreezeVirtualMachine(vmi, config.UnfreezeTimeoutSeconds)
		if err != nil {
			if strings.Contains(err.Error(), gaNotAvailableError) {
				client.UnfreezeVirtualMachine(vmi)
				if strings.Contains(strings.ToLower(info.OS.Name), windowsOS) {
					log.Log.Reason(err).Error("Freezing VMI failed, please make sure guest agent and VSS are running and try again")
				} else {
					log.Log.Reason(err).Error("Freezing VMI failed, please make sure guest agent is running and try again")
				}
			} else {
				log.Log.Reason(err).Error("Freezing VMI failed")
			}
			return err
		}
	} else {
		err = client.UnfreezeVirtualMachine(vmi)
		if err != nil {
			if strings.Contains(err.Error(), freezeLimitReached) {
				log.Log.Reason(err).Error("Unfreezing VMI failed, please try again. If problem continues, stop the VM and backup while down")
			} else {
				log.Log.Reason(err).Error("Unfreezing VMI failed")
			}
			return err
		}
	}

	log.Log.Info("Operation completed successfully")
	return nil
}

func main() {
	log.InitializeLogging("freezer")

	config, err := parseFlags()
	if err != nil {
		log.Log.Reason(err).Error("Failed to parse flags")
		os.Exit(1)
	}

	client, err := getGrpcClient()
	if err != nil {
		os.Exit(1)
	}

	err = run(config, client)
	if err != nil {
		os.Exit(1)
	}
}
