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
	"gopkg.in/yaml.v2"

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
	replacer := strings.NewReplacer(
		"\\", "\\\\",
		"\"", "\\\"",
		"`", "\\`",
		"$", "\\$",
		"\n", "\\n",
	)

	content.WriteString("#!/bin/sh\n")
	content.WriteString("# AWS environment variables for IRSA\n\n")

	for key, value := range envVars {
		escapedValue := replacer.Replace(value)
		content.WriteString(fmt.Sprintf("export %s=\"%s\"\n", key, escapedValue))
	}
	return content.String()
}

type CloudInitFile struct {
	Path        string `yaml:"path"`
	Permissions string `yaml:"permissions"`
	Content     string `yaml:"content"`
}

type CloudInitConfig struct {
	WriteFiles []CloudInitFile `yaml:"write_files,omitempty"`
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

	awsEnvVars := make(map[string]string)
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "AWS_") {
			key := env[:strings.Index(env, "=")]
			value := env[strings.Index(env, "=")+1:]
			awsEnvVars[key] = value
		}
	}

	isWindows := false
	if vmi.Spec.Domain.Firmware != nil && vmi.Spec.Domain.Firmware.Bootloader != nil {
		if vmi.Spec.Domain.Firmware.Bootloader.EFI != nil {
			isWindows = true
		}
	}

	if isWindows {
		if cloudInitData.UserData == "" {
			cloudInitData.UserData = "#ps1_sysnative\n"
		}
		script := generateWindowsScript(awsEnvVars)
		cloudInitData.UserData += fmt.Sprintf("\n%s\n", script)
		log.Printf("Generated Windows PowerShell script for setting AWS environment variables")
	} else {
		if cloudInitData.UserData == "" {
			cloudInitData.UserData = "#cloud-config\n"
		}
		envContent := generateLinuxScript(awsEnvVars)

		config := CloudInitConfig{
			WriteFiles: []CloudInitFile{
				{
					Path:        "/etc/profile.d/aws-env.sh",
					Permissions: "0755",
					Content:     envContent,
				},
			},
		}

		yamlData, err := yaml.Marshal(config)
		if err != nil {
			return "", fmt.Errorf("Failed to marshal cloud-init config: %v", err)
		}

		cloudInitData.UserData += "\n" + string(yamlData)
		log.Printf("Generated Linux cloud-config for setting AWS environment variables in /etc/profile.d/aws-env.sh")
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

	domainStr := string(domainXML)
	if strings.Contains(domainStr, "aws-iam-token") {
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
