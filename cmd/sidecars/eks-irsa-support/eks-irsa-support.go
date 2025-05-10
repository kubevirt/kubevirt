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
	"path/filepath"
	"strings"

	"github.com/spf13/pflag"

	v1 "kubevirt.io/api/core/v1"

	cloudinit "kubevirt.io/kubevirt/pkg/cloud-init"
)

func generateWindowsScript(envVars map[string]string) string {
	var content strings.Builder
	content.WriteString("[System.Environment]::SetEnvironmentVariable('PATH', $env:PATH, [System.EnvironmentVariableTarget]::Machine)\n")
	for key, value := range envVars {
		content.WriteString(fmt.Sprintf("[System.Environment]::SetEnvironmentVariable('%s', '%s', [System.EnvironmentVariableTarget]::Machine)\n", key, value))
	}
	return content.String()
}

func generateLinuxScript(envVars map[string]string) string {
	var content strings.Builder
	for key, value := range envVars {
		content.WriteString(fmt.Sprintf("      %s=\"%s\"\n", key, value))
	}
	return content.String()
}

func preCloudInitIso(log *log.Logger, vmiJSON, cloudInitDataJSON []byte) (string, error) {
	log.Print("EKS IAM RSA hook's PreCloudInitIso callback method has been called")

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

	// Collect AWS environment variables
	awsEnvVars := make(map[string]string)
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "AWS_") {
			key := env[:strings.Index(env, "=")]
			value := env[strings.Index(env, "=")+1:]
			awsEnvVars[key] = value
		}
	}

	// Check if it's a Windows VM
	isWindows := false
	if vmi.Spec.Domain.Firmware != nil && vmi.Spec.Domain.Firmware.Bootloader != nil {
		if vmi.Spec.Domain.Firmware.Bootloader.EFI != nil {
			isWindows = true
		}
	}

	if isWindows {
		// For Windows VMs
		if cloudInitData.UserData == "" {
			cloudInitData.UserData = "#ps1_sysnative\n"
		}
		script := generateWindowsScript(awsEnvVars)
		cloudInitData.UserData += fmt.Sprintf("\n%s\n", script)
		log.Printf("Generated Windows PowerShell script for setting AWS environment variables")
	} else {
		// For Linux VMs
		if cloudInitData.UserData == "" {
			cloudInitData.UserData = "#cloud-config\n"
		}
		envContent := generateLinuxScript(awsEnvVars)
		cloudInitData.UserData += fmt.Sprintf("\nwrite_files:\n  - path: /etc/environment\n    content: |\n%s    append: true\n", envContent)
		log.Printf("Generated Linux cloud-config for setting AWS environment variables")
	}

	response, err := json.Marshal(cloudInitData)
	if err != nil {
		return "", fmt.Errorf("Failed to marshal CloudInitData: %s %+v", err, cloudInitData)
	}

	return string(response), nil
}

func onDefineDomain(log *log.Logger, vmiJSON, domainXML []byte) (string, error) {
	log.Print("EKS IRSA support hook's OnDefineDomain callback method has been called")

	vmi := v1.VirtualMachineInstance{}
	err := json.Unmarshal(vmiJSON, &vmi)
	if err != nil {
		return "", fmt.Errorf("Failed to unmarshal given VMI spec: %s %s", err, string(vmiJSON))
	}

	// Check if EKS token filesystem already exists
	domainStr := string(domainXML)
	if strings.Contains(domainStr, "target dir='aws-iam-token'") {
		log.Print("EKS token filesystem already exists in domain XML, skipping addition")
		return domainStr, nil
	}

	fsXML := `
    <filesystem type='mount' accessmode='passthrough'>
      <driver type='virtiofs' queue='1024'/>
      <source dir='' socket='/var/run/kubevirt/virtiofs-containers/aws-iam-token.sock'/>
      <target dir='aws-iam-token'/>
    </filesystem>`

	devicesEnd := strings.Index(domainStr, "</devices>")
	if devicesEnd == -1 {
		return "", fmt.Errorf("Could not find </devices> tag in domain XML")
	}

	modifiedDomain := domainStr[:devicesEnd] + fsXML + domainStr[devicesEnd:]
	return modifiedDomain, nil
}

func main() {
	var vmiJSON, cloudInitDataJSON, domainXML string

	pflag.StringVar(&vmiJSON, "vmi", "", "Current VMI, in JSON format")
	pflag.StringVar(&cloudInitDataJSON, "cloud-init", "", "The CloudInitData, in JSON format")
	pflag.StringVar(&domainXML, "domain", "", "The domain XML")
	pflag.Parse()

	logger := log.New(os.Stderr, "eks-irsa-support", log.Ldate)

	if vmiJSON == "" {
		logger.Printf("Bad input vmi=%d", len(vmiJSON))
		os.Exit(1)
	}

	var result string
	var err error

	executable, err := os.Executable()
	if err != nil {
		logger.Printf("Failed to get executable name: %s", err)
		os.Exit(1)
	}
	hookType := filepath.Base(executable)

	switch hookType {
	case "preCloudInitIso":
		if cloudInitDataJSON == "" {
			logger.Printf("Bad input cloud-init=%d", len(cloudInitDataJSON))
			os.Exit(1)
		}
		result, err = preCloudInitIso(logger, []byte(vmiJSON), []byte(cloudInitDataJSON))
	case "onDefineDomain":
		if domainXML == "" {
			logger.Printf("Bad input domain=%d", len(domainXML))
			os.Exit(1)
		}
		result, err = onDefineDomain(logger, []byte(vmiJSON), []byte(domainXML))
	default:
		logger.Printf("Unknown hook type: %s", hookType)
		os.Exit(1)
	}

	if err != nil {
		logger.Printf("%s failed: %s", hookType, err)
		panic(err)
	}
	fmt.Println(result)
}
