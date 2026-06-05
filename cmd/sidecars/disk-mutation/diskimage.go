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
 * Copyright 2022 Nvidia
 *
 */

// Inspired by cmd/example-hook-sidecar

package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/spf13/pflag"

	vmSchema "kubevirt.io/api/core/v1"

	domainSchema "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const bootDiskImageNameAnnotation = "diskimage.vm.kubevirt.io/bootImageName"

func OnDefineDomain(log *log.Logger, vmiJSON, domainXML []byte) (string, error) {
	vmiSpec := vmSchema.VirtualMachineInstance{}
	err := json.Unmarshal(vmiJSON, &vmiSpec)
	if err != nil {
		return "", fmt.Errorf("Failed to unmarshal given VMI spec: %s %s", err, string(vmiJSON))
	}

	annotations := vmiSpec.GetAnnotations()
	if _, found := annotations[bootDiskImageNameAnnotation]; !found {
		log.Print("Boot disk hook sidecar was requested, but no attributes provided. Returning original domain spec")
		return string(domainXML), nil
	}

	domainSpec := domainSchema.DomainSpec{}
	err = xml.Unmarshal(domainXML, &domainSpec)
	if err != nil {
		return "", fmt.Errorf("Failed to unmarshal given domain spec: %s %s", err, string(domainXML))
	}

	// Get Boot disk
	bootDiskLocation := domainSpec.Devices.Disks[0].Source.File
	dir, diskName := filepath.Split(bootDiskLocation)
	newDiskName := annotations[bootDiskImageNameAnnotation]
	if newDiskName == diskName {
		log.Printf("Boot disk image name is already %q. Returning original domain spec", newDiskName)
		return string(domainXML), nil
	}

	bootDiskLocation = filepath.Join(dir, newDiskName)
	domainSpec.Devices.Disks[0].Source.File = bootDiskLocation

	newDomainXML, err := xml.Marshal(domainSpec)
	if err != nil {
		return "", fmt.Errorf("Failed to marshal updated domain spec: %s %+v", err, domainSpec)
	}

	return string(newDomainXML), nil
}

func main() {
	var vmiJSON, domainXML string
	pflag.StringVar(&vmiJSON, "vmi", "", "VMI to change in JSON format")
	pflag.StringVar(&domainXML, "domain", "", "Domain spec in XML format")
	pflag.Parse()

	logger := log.New(os.Stderr, "diskimage", log.Ldate)
	if vmiJSON == "" || domainXML == "" {
		logger.Printf("Bad input vmi=%d, domain=%d", len(vmiJSON), len(domainXML))
		os.Exit(1)
	}

	domainXML, err := OnDefineDomain(logger, []byte(vmiJSON), []byte(domainXML))
	if err != nil {
		logger.Printf("OnDefineDomain failed: %s", err)
		panic(err)
	}
	fmt.Println(domainXML)
}
