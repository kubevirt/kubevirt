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
 * Copyright 2022 Red Hat, Inc.
 *
 */

package vmexport

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	k8sv1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/clientcmd"
	exportv1 "kubevirt.io/api/export/v1alpha1"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

const (
	// processingWaitInterval is the time interval used to wait for a virtualMachineExport to be ready
	processingWaitInterval = 2 * time.Second
	// processingWaitTotal is the maximum time used to wait for a virtualMachineExport to be ready
	processingWaitTotal = 24 * time.Hour

	// ErrRequiredFlag serves as error message when a mandatory flag is missing
	ErrRequiredFlag = "Need to specify the '%s' flag when using '%s'"
	// ErrIncompatibleFlag serves as error message when an incompatible flag is used
	ErrIncompatibleFlag = "The '%s' flag is incompatible with '%s'"
	// ErrRequiredExportType serves as error message when no export kind is provided
	ErrRequiredExportType = "Need to specify export kind when attempting to create a VirtualMachineExport [pvc|vm|snapshot]"
	// ErrIncompatibleExportType serves as error message when an export kind is provided with an incompatible argument
	ErrIncompatibleExportType = "Should not specify export kind"
	// ErrBadExportArguments serves as error message when vmexport is used with bad arguments
	ErrBadExportArguments = "Expecting two args: vmexport function [create|delete|download] and VirtualMachineExport name"
)

var (
	// Flags
	vm           string
	snapshot     string
	pvc          string
	outputFile   string
	shouldCreate bool
	volumeName   string

	// vmexport info
	vmexportName string
	resourceName string
	resourceKind string
	funcName     string
)

type exportFunc func(client kubecli.KubevirtClient, namespace string) error

type command struct {
	clientConfig clientcmd.ClientConfig
}

var exportFunction exportFunc

// usage provides several valid usage examples of vmexport
func usage() string {
	usage := `# Create a VirtualMachineExport to export a volume from a virtual machine:
	{{ProgramName}} vmexport create vm1-export --vm=vm1
  
	# Create a VirtualMachineExport to export a volume from a virtual machine snapshot
	{{ProgramName}} vmexport create snap1-export --snapshot=snap1
  
	# Create a VirtualMachineExport to export a volume from a PVC
	{{ProgramName}} vmexport create pvc1-export --pvc=pvc1
  
	# Delete a VirtualMachineExport resource
	{{ProgramName}} vmexport delete snap1-export
  
	# Download a volume from an already existing VirtualMachineExport
	{{ProgramName}} vmexport download vm1-export --volume=volume1 --output=disk.img.gz
  
	# Create a VirtualMachineExport and download the requested volume from it
	{{ProgramName}} vmexport download vm1-export --create --volume=volume1 --output=disk.img.gz`

	return usage
}

// NewVirtualMachineExportCommand returns a cobra.Command to handle the export process
func NewVirtualMachineExportCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "vmexport",
		Short:   "Export a VM volume.",
		Example: usage(),
		Args:    cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			v := command{clientConfig: clientConfig}
			return v.runVMExport(args)
		},
	}

	cmd.Flags().StringVar(&vm, "vm", "", "Sets VirtualMachine as vmexport kind and specifies the vm name.")
	cmd.Flags().StringVar(&snapshot, "snapshot", "", "Sets VirtualMachineSnapshot as vmexport kind and specifies the snapshot name.")
	cmd.Flags().StringVar(&pvc, "pvc", "", "Sets PersistentVolumeClaim as vmexport kind and specifies the PVC name.")
	cmd.MarkFlagsMutuallyExclusive("vm", "snapshot", "pvc")
	cmd.Flags().StringVar(&outputFile, "output", "", "Specifies the output path of the volume to be downloaded.")
	cmd.Flags().StringVar(&volumeName, "volume", "", "Specifies the volume to be downloaded.")
	cmd.Flags().BoolVar(&shouldCreate, "create", shouldCreate, "When used with the 'download' option, specifies that a VirtualMachineExport should be created from scratch.")
	cmd.SetUsageTemplate(templates.UsageTemplate())

	return cmd
}

// runVMExport serves as entrypoint for the vmexport command
func (c *command) runVMExport(args []string) error {
	if err := parseExportArguments(args); err != nil {
		return err
	}

	namespace, _, err := c.clientConfig.Namespace()
	if err != nil {
		return err
	}
	virtClient, err := kubecli.GetKubevirtClientFromClientConfig(c.clientConfig)
	if err != nil {
		return fmt.Errorf("Cannot obtain KubeVirt client: %v", err)
	}

	// Finally, run the vmexport function (create|delete|download)
	if err := exportFunction(virtClient, namespace); err != nil {
		return err
	}

	return nil
}

// parseExportArguments parses and validates vmexport arguments and flags. These arguments should always be:
// 	1. The vmexport function (create|delete|download)
// 	2. The VirtualMachineExport name
func parseExportArguments(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf(ErrBadExportArguments)
	}

	funcName = strings.ToLower(args[0])

	// Assign the appropiate vmexport function and make sure the used flags are compatible
	switch funcName {
	case "create":
		exportFunction = createVirtualMachineExport
		if err := handleCreateFlags(args); err != nil {
			return err
		}
	case "delete":
		exportFunction = deleteVirtualMachineExport
		if err := handleDeleteFlags(args); err != nil {
			return err
		}
	case "download":
		exportFunction = downloadVirtualMachineExport
		if err := handleDownloadFlags(args); err != nil {
			return err
		}
	default:
		return fmt.Errorf("Invalid function '%s'", funcName)
	}

	// VirtualMachineExport name
	vmexportName = args[1]

	return nil
}

// getVirtualMachineExport serves as a wrapper to get the VirtualMachineExport object
func getVirtualMachineExport(client kubecli.KubevirtClient, namespace string) (*exportv1.VirtualMachineExport, error) {
	vmexport, err := client.VirtualMachineExport(namespace).Get(context.TODO(), vmexportName, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	return vmexport, nil
}

// createVirtualMachineExport serves as a wrapper to create the virtualMachineExport object and, if needed, do error handling
func createVirtualMachineExport(client kubecli.KubevirtClient, namespace string) error {
	vmexport, err := getVirtualMachineExport(client, namespace)
	if err != nil {
		return err
	}
	if vmexport != nil {
		return fmt.Errorf("VirtualMachineExport '%s/%s' already exists", vmexportName, namespace)
	}

	vmexport = &exportv1.VirtualMachineExport{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vmexportName,
			Namespace: namespace,
		},
		Spec: exportv1.VirtualMachineExportSpec{
			Source: k8sv1.TypedLocalObjectReference{
				APIGroup: &k8sv1.SchemeGroupVersion.Group,
				Kind:     resourceKind,
				Name:     resourceName,
			},
		},
	}

	_, err = client.VirtualMachineExport(namespace).Create(context.TODO(), vmexport, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	fmt.Printf("VirtualMachineExport '%s/%s' created succesfully\n", namespace, vmexportName)
	return nil
}

// deleteVirtualMachineExport serves as a wrapper to delete the virtualMachineExport object
func deleteVirtualMachineExport(client kubecli.KubevirtClient, namespace string) error {
	if err := client.VirtualMachineExport(namespace).Delete(context.TODO(), vmexportName, metav1.DeleteOptions{}); err != nil {
		if !k8serrors.IsNotFound(err) {
			return err
		}
		return fmt.Errorf("VirtualMachineExport '%s/%s' does not exist", namespace, vmexportName)
	}

	fmt.Printf("VirtualMachineExport '%s/%s' deleted succesfully\n", namespace, vmexportName)
	return nil
}

// downloadVirtualMachineExport handles the process of downloading the requested volume from a VirtualMachineExport object
func downloadVirtualMachineExport(client kubecli.KubevirtClient, namespace string) error {
	if shouldCreate {
		if err := createVirtualMachineExport(client, namespace); err != nil {
			return err
		}
	}

	vmexport, err := getVirtualMachineExport(client, namespace)
	if err != nil {
		return err
	}

	if err := waitForVirtualMachineExport(client, namespace, processingWaitInterval, processingWaitTotal); err != nil {
		return err
	}

	if err := downloadVolumeInOutput(client, vmexport); err != nil {
		return err
	}

	return nil
}

// downloadVolumeInOutput handles the process of downloading the requested volume from a VirtualMachineExport object
func downloadVolumeInOutput(client kubecli.KubevirtClient, vmexport *exportv1.VirtualMachineExport) error {
	output, err := os.Open(outputFile)
	if err != nil {
		return err
	}

	defer util.CloseIOAndCheckErr(output, nil)

	resp, err := getUrlFromVirtualMachineExport(vmexport)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Bad status: %s", resp.Status)
	}

	_, err = io.Copy(output, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

// getUrlFromVirtualMachineExport inspects the VirtualMachineExport status to fetch the extected URL
func getUrlFromVirtualMachineExport(vmexport *exportv1.VirtualMachineExport) (*http.Response, error) {
	var downloadUrl string
	outputFileExtension := filepath.Ext(outputFile)

	if vmexport == nil || vmexport.Status == nil || vmexport.Status.Links == nil ||
		vmexport.Status.Links.External == nil || vmexport.Status.Links.External.Volumes == nil {
		return nil, fmt.Errorf("Unable to get URL from virtualMachineExport")
	}

	for _, exportVolume := range vmexport.Status.Links.External.Volumes {
		// Access the requested volume
		if exportVolume.Name == volumeName {
			for _, format := range exportVolume.Formats {
				// Get the appropiate URL according to the output's extension
				if string(format.Format) == outputFileExtension {
					downloadUrl = format.Url
				}
			}
		}
	}

	if downloadUrl == "" {
		return nil, fmt.Errorf("Couldn't get an appropiate URL")
	}

	// get the URL
	resp, err := http.Get(downloadUrl)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// waitForVirtualMachineExport waits for the VirtualMachineExport status to be ready
func waitForVirtualMachineExport(client kubecli.KubevirtClient, namespace string, interval, timeout time.Duration) error {
	err := wait.PollImmediate(interval, timeout, func() (bool, error) {
		vmexport, err := getVirtualMachineExport(client, namespace)
		if err != nil {
			return false, err
		}

		if vmexport == nil || vmexport.Status == nil {
			return false, nil
		}

		done := vmexport.Status.Phase == "Ready"
		if !done {
			fmt.Printf("Waiting for VM Export %s status to be ready...\n", vmexportName)
		} else {
			fmt.Printf("Processing completed successfully\n")
		}

		return done, nil
	})

	return err
}

// handleCreateFlags ensures that only compatible flag combinations are used with 'create' or with 'download' with the --create flag
func handleCreateFlags(args []string) error {
	if vm == "" && snapshot == "" && pvc == "" {
		return fmt.Errorf(ErrRequiredExportType)
	}

	if vm != "" {
		resourceKind = "VirtualMachine"
		resourceName = vm
	}
	if snapshot != "" {
		resourceKind = "VirtualMachineSnapshot"
		resourceName = snapshot
	}
	if pvc != "" {
		resourceKind = "PersistentVolumeClaim"
		resourceName = pvc
	}

	// These flags should only be checked when using 'create'
	if funcName == "create" {
		if outputFile != "" {
			return fmt.Errorf(ErrIncompatibleFlag, "--output", funcName)
		}
		if volumeName != "" {
			return fmt.Errorf(ErrIncompatibleFlag, "--volume", funcName)
		}
		if shouldCreate {
			return fmt.Errorf(ErrIncompatibleFlag, "--create", funcName)
		}
	}

	return nil
}

// handleDeleteFlags ensures that only compatible flag combinations are used with 'delete'
func handleDeleteFlags(args []string) error {
	if vm != "" || snapshot != "" || pvc != "" {
		return fmt.Errorf(ErrIncompatibleExportType)
	}

	if outputFile != "" {
		return fmt.Errorf(ErrIncompatibleFlag, "--output", funcName)
	}
	if volumeName != "" {
		return fmt.Errorf(ErrIncompatibleFlag, "--volume", funcName)
	}
	if shouldCreate {
		return fmt.Errorf(ErrIncompatibleFlag, "--create", funcName)
	}

	return nil
}

// handleDownloadFlags ensures that only compatible flag combinations are used with 'download'
func handleDownloadFlags(args []string) error {
	// We also need to handle some 'create' flags when using --create
	err := handleCreateFlags(args)
	if shouldCreate && err != nil {
		return err
	}
	if !shouldCreate && err == nil {
		return fmt.Errorf(ErrIncompatibleExportType)
	}

	if outputFile == "" {
		return fmt.Errorf(ErrRequiredFlag, "--output", funcName)
	}
	if volumeName == "" {
		return fmt.Errorf(ErrRequiredFlag, "--volume", funcName)
	}

	return nil
}
