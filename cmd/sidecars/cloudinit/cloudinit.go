/*
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
 * Copyright 2019 StackPath, LLC
 *
 */

// Inspired by cmd/example-hook-sidecar

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/spf13/pflag"

	v1 "kubevirt.io/api/core/v1"

	cloudinit "kubevirt.io/kubevirt/pkg/cloud-init"
)

func preCloudInitIso(log *log.Logger, vmiJSON, cloudInitDataJSON []byte) (string, error) {
	log.Print("Hook's PreCloudInitIso callback method has been called")

	vmi := v1.VirtualMachineInstance{}
	err := json.Unmarshal(vmiJSON, &vmi)
	if err != nil {
		return "", fmt.Errorf("Failed to unmarshal given VMI spec: %s %s", err, string(vmiJSON))
	}

	cloudInitData := cloudinit.CloudInitData{}
	err = json.Unmarshal(cloudInitDataJSON, &cloudInitData)
	if err != nil {
		return "", fmt.Errorf("Failed to unmarshal given CloudInitData: %s %s", err, string(cloudInitDataJSON))
	}

	cloudInitData.UserData = "#cloud-config\n"

	response, err := json.Marshal(cloudInitData)
	if err != nil {
		return "", fmt.Errorf("Failed to marshal CloudInitData: %s %+v", err, cloudInitData)
	}

	return string(response), nil
}

func main() {
	var vmiJSON, cloudInitDataJSON string
	pflag.StringVar(&vmiJSON, "vmi", "", "Current VMI, in JSON format")
	pflag.StringVar(&cloudInitDataJSON, "cloud-init", "", "The CloudInitData, in JSON format")
	pflag.Parse()

	logger := log.New(os.Stderr, "cloudinit", log.Ldate)
	if vmiJSON == "" || cloudInitDataJSON == "" {
		logger.Printf("Bad input vmi=%d, cloud-init=%d", len(vmiJSON), len(cloudInitDataJSON))
		os.Exit(1)
	}

	cloudInitData, err := preCloudInitIso(logger, []byte(vmiJSON), []byte(cloudInitDataJSON))
	if err != nil {
		logger.Printf("preCloudInitIso failed: %s", err)
		panic(err)
	}
	fmt.Println(cloudInitData)
}
