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
	"os"
	"strings"

	"github.com/spf13/pflag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"

	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
)

const (
	gaNotAvailableError = "Guest agent not available for now"
	windowsOS           = "windows"
	freezeLimitReached  = "fsfreeze is limited"
)

func getGrpcClient() (cmdclient.LauncherClient, error) {
	sockFile := "/run/kubevirt/sockets/launcher-sock"
	client, err := cmdclient.NewClient(sockFile)
	if err != nil {
		log.Log.Reason(err).Error("Failed to connect launcher")
		os.Exit(1)
	}

	return client, err
}

func shouldFreezeVirtualMachine(client cmdclient.LauncherClient) bool {
	domain, exists, err := client.GetDomain()
	if err != nil {
		log.Log.Reason(err).Error("Failed to get domain")
		os.Exit(1)
	}

	return exists && domain.Status.Status == api.Running
}

func main() {
	log.InitializeLogging("freezer")
	log.Log.Info("Starting...")

	freeze := pflag.Bool("freeze", false, "Freeze VM")
	unfreeze := pflag.Bool("unfreeze", false, "Unfreeze VM")
	name := pflag.String("name", "", "Name of the VirtualMachineInstance")
	namespace := pflag.String("namespace", "", "Namespace of the VirtualMachineInstance")
	unfreezeTimeoutSeconds := pflag.Int32("unfreezeTimeoutSeconds", 300, "Timeout in seconds to automatically unfreeze the VirtualMachineInstance")

	pflag.Parse()

	if !*freeze && !*unfreeze {
		log.Log.Errorf("Use either --freeze or --unfreeze")
		os.Exit(1)
	}
	if name == nil || namespace == nil {
		log.Log.Errorf("Both name and namespace flags must be provided")
		os.Exit(1)
	}

	vmi := &v1.VirtualMachineInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      *name,
			Namespace: *namespace,
		},
	}

	client, err := getGrpcClient()
	if err != nil {
		log.Log.Reason(err).Error("Failed to connect launcher")
		os.Exit(1)
	}

	info, err := client.GetGuestInfo()
	if err != nil {
		log.Log.Reason(err).Error("Failed to get guest info")
		os.Exit(1)
	}

	if info.GAVersion == "" {
		log.Log.Info("No guest agent, exiting")
		os.Exit(0)
	}

	log.Log.Infof("Guest agent version is %s", info.GAVersion)

	if !shouldFreezeVirtualMachine(client) {
		log.Log.Info("VM domain not running, no need to freeze/unfreeze")
		os.Exit(0)
	}

	if *freeze {
		err = client.FreezeVirtualMachine(vmi, *unfreezeTimeoutSeconds)
		if err != nil {
			if strings.Contains(err.Error(), gaNotAvailableError) {
				// make best effort of make sure fsstatus is not stuck on frozen
				// due to bug https://issues.redhat.com/browse/RHEL-24046
				client.UnfreezeVirtualMachine(vmi)
				if strings.Contains(strings.ToLower(info.OS.Name), windowsOS) {
					log.Log.Reason(err).Error("Freezeing VMI failed, please make sure guest agent and VSS are runnning and try again")
				} else {
					log.Log.Reason(err).Error("Freezeing VMI failed, please make sure guest agent is runnning and try again")
				}
			} else {
				log.Log.Reason(err).Error("Freezeing VMI failed")
			}
			os.Exit(1)
		}
	} else {
		err = client.UnfreezeVirtualMachine(vmi)
		if err != nil {
			if strings.Contains(err.Error(), freezeLimitReached) {
				log.Log.Reason(err).Error("Unfreezeing VMI failed, please try again. If problem continues, stop the vm and backup while down")
			} else {
				log.Log.Reason(err).Error("Unfreezeing VMI failed")
			}
			os.Exit(1)
		}
	}

	log.Log.Info("Exiting...")
}
