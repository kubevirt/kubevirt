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

package memorydump

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	k8sv1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/clientcmd"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
	kutil "kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
	"kubevirt.io/kubevirt/pkg/virtctl/vmexport"
)

const (
	claimNameArg    = "claim-name"
	createClaimArg  = "create-claim"
	storageClassArg = "storage-class"
	accessModeArg   = "access-mode"
	portForwardArg  = "port-forward"
	formatArg       = "format"
	localPortArg    = "local-port"

	configName         = "config"
	filesystemOverhead = cdiv1.Percent("0.055")
	fsOverheadMsg      = "Using default 5.5%% filesystem overhead for pvc size"

	processingWaitInterval = 2 * time.Second
	processingWaitTotal    = 10 * time.Minute
)

var (
	claimName    string
	createClaim  bool
	portForward  bool
	format       string
	localPort    string
	storageClass string
	accessMode   string
	outputFile   string
)

type command struct {
	clientConfig clientcmd.ClientConfig
}

type memoryDumpCompleteFunc func(kubecli.KubevirtClient, string, string, time.Duration, time.Duration) (string, error)

// WaitMemoryDumpCompleted is used to store the function to wait for the memory dump to be complete.
// Useful for unit tests.
var WaitMemoryDumpComplete memoryDumpCompleteFunc = waitForMemoryDump

func usageMemoryDump() string {
	usage := `  #Dump memory of a virtual machine instance called 'myvm' to an existing pvc called 'memoryvolume'.
  {{ProgramName}} memory-dump get myvm --claim-name=memoryvolume

  #Create a PVC called 'memoryvolume' and dump the memory of a virtual machine instance called 'myvm' to it.
  {{ProgramName}} memory-dump get myvm --claim-name=memoryvolume --create-claim

  #Create and download memory dump to the given output file.
  {{ProgramName}} memory-dump get myvm --claim-name=memoryvolume --create-claim --output=memoryDump.dump.gz

  #Dump memory again to the same virtual machine with an already associated pvc(existing memory dump on vm status).
  {{ProgramName}} memory-dump get myvm

  #Download the last memory dump associated on the vm 'myvm' to the given output file.
  {{ProgramName}} memory-dump download myvm --output=memoryDump.dump.gz

  #Remove the association of the memory dump pvc (to be able to dump to another pvc).
  {{ProgramName}} memory-dump remove myvm
  `
	return usage
}

// NewMemoryDumpCommand returns a cobra.Command to handle the memory dump process
func NewMemoryDumpCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "memory-dump get/download/remove (VM)",
		Short:   "Dump the memory of a running VM to a pvc",
		Example: usageMemoryDump(),
		Args:    templates.ExactArgs("memory-dump", 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := command{clientConfig: clientConfig}
			return c.run(args)
		},
	}
	cmd.SetUsageTemplate(templates.UsageTemplate())
	cmd.Flags().StringVar(&claimName, claimNameArg, "", "pvc name to contain the memory dump")
	cmd.Flags().BoolVar(&createClaim, createClaimArg, false, "Create the pvc that will conatin the memory dump")
	cmd.Flags().BoolVar(&portForward, portForwardArg, false, "Configure and set port-forward in a random port to download the memory dump")
	cmd.Flags().StringVar(&format, formatArg, "", "Specifies the format of the memory dump download (gzipped or raw).")
	cmd.Flags().StringVar(&localPort, localPortArg, "0", "Specify port for port-forward")
	cmd.Flags().StringVar(&storageClass, storageClassArg, "", "The storage class for the PVC.")
	cmd.Flags().StringVar(&accessMode, accessModeArg, "", "The access mode for the PVC.")
	cmd.Flags().StringVar(&outputFile, "output", "", "Specifies the output path of the memory dump to be downloaded.")

	return cmd
}

func (c *command) run(args []string) error {
	namespace, _, err := c.clientConfig.Namespace()
	if err != nil {
		return err
	}
	virtClient, err := kubecli.GetKubevirtClientFromClientConfig(c.clientConfig)
	if err != nil {
		return fmt.Errorf("cannot obtain KubeVirt client: %v", err)
	}

	vmName := args[1]
	switch args[0] {
	case "get":
		return getMemoryDump(namespace, vmName, virtClient)
	case "download":
		return downloadMemoryDump(namespace, vmName, virtClient)
	case "remove":
		return removeMemoryDump(namespace, vmName, virtClient)
	default:
		return fmt.Errorf("invalid action type %s", args[0])
	}
}

func calcMemoryDumpExpectedSize(vmName, namespace string, virtClient kubecli.KubevirtClient) (*resource.Quantity, error) {
	vmi, err := virtClient.VirtualMachineInstance(namespace).Get(context.Background(), vmName, &metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return kutil.CalcExpectedMemoryDumpSize(vmi), nil
}

func calcPVCNeededSize(memoryDumpExpectedSize *resource.Quantity, storageClass *string, virtClient kubecli.KubevirtClient) (*resource.Quantity, error) {
	cdiConfig, err := virtClient.CdiClient().CdiV1beta1().CDIConfigs().Get(context.Background(), configName, metav1.GetOptions{})
	if k8serrors.IsNotFound(err) {
		// can't properly determine the overhead - continue with default overhead of 5.5%
		fmt.Println(fsOverheadMsg)
		return storagetypes.GetSizeIncludingGivenOverhead(memoryDumpExpectedSize, filesystemOverhead)
	}
	if err != nil {
		return nil, err
	}

	fsVolumeMode := k8sv1.PersistentVolumeFilesystem
	if *storageClass == "" {
		storageClass = nil
	}

	return storagetypes.GetSizeIncludingFSOverhead(memoryDumpExpectedSize, storageClass, &fsVolumeMode, cdiConfig)
}

func generatePVC(size *resource.Quantity, claimName, namespace, storageClass, accessMode string) (*k8sv1.PersistentVolumeClaim, error) {
	pvc := &k8sv1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      claimName,
			Namespace: namespace,
		},
		Spec: k8sv1.PersistentVolumeClaimSpec{
			Resources: k8sv1.ResourceRequirements{
				Requests: k8sv1.ResourceList{
					k8sv1.ResourceStorage: *size,
				},
			},
		},
	}

	if storageClass != "" {
		pvc.Spec.StorageClassName = &storageClass
	}

	if accessMode != "" {
		if accessMode == string(k8sv1.ReadOnlyMany) {
			return nil, fmt.Errorf("cannot dump memory to a readonly pvc, use either ReadWriteOnce or ReadWriteMany if supported")
		}
		// TODO: fix when issue: https://github.com/kubevirt/containerized-data-importer/issues/2365 is done
		if accessMode != string(k8sv1.ReadWriteOnce) && accessMode != string(k8sv1.ReadWriteMany) {
			return nil, fmt.Errorf("invalid access mode, use either ReadWriteOnce or ReadWriteMany if supported")
		}

		pvc.Spec.AccessModes = []k8sv1.PersistentVolumeAccessMode{k8sv1.PersistentVolumeAccessMode(accessMode)}
	} else {
		pvc.Spec.AccessModes = []k8sv1.PersistentVolumeAccessMode{k8sv1.ReadWriteOnce}
	}

	return pvc, nil
}

func checkNoExistingPVC(namespace, claimName string, virtClient kubecli.KubevirtClient) error {
	_, err := virtClient.CoreV1().PersistentVolumeClaims(namespace).Get(context.Background(), claimName, metav1.GetOptions{})
	if err == nil {
		return fmt.Errorf("PVC %s/%s already exists, check if it should be created if not remove create flag", namespace, claimName)
	}
	if !k8serrors.IsNotFound(err) {
		return err
	}
	return nil
}

func checkNoAssociatedMemoryDump(namespace, vmName string, virtClient kubecli.KubevirtClient) error {
	vm, err := virtClient.VirtualMachine(namespace).Get(context.Background(), vmName, &metav1.GetOptions{})
	if err != nil {
		return err
	}
	if vm.Status.MemoryDumpRequest != nil {
		return fmt.Errorf("please remove current memory dump association before creating a new claim for a new memory dump")
	}
	return nil
}

func createPVCforMemoryDump(namespace, vmName, claimName string, virtClient kubecli.KubevirtClient) error {
	// Before creating a new pvc check that there is not already
	// assocaited memory dump pvc
	if err := checkNoAssociatedMemoryDump(namespace, vmName, virtClient); err != nil {
		return err
	}

	if err := checkNoExistingPVC(namespace, claimName, virtClient); err != nil {
		return err
	}

	memoryDumpExpectedSize, err := calcMemoryDumpExpectedSize(vmName, namespace, virtClient)
	if err != nil {
		return err
	}

	neededSize, err := calcPVCNeededSize(memoryDumpExpectedSize, &storageClass, virtClient)
	if err != nil {
		return err
	}

	pvc, err := generatePVC(neededSize, claimName, namespace, storageClass, accessMode)
	if err != nil {
		return err
	}

	_, err = virtClient.CoreV1().PersistentVolumeClaims(namespace).Create(context.Background(), pvc, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	fmt.Printf("PVC %s/%s created\n", namespace, claimName)

	return nil
}

func createMemoryDump(namespace, vmName, claimName string, virtClient kubecli.KubevirtClient) error {
	memoryDumpRequest := &v1.VirtualMachineMemoryDumpRequest{
		ClaimName: claimName,
	}

	err := virtClient.VirtualMachine(namespace).MemoryDump(context.Background(), vmName, memoryDumpRequest)
	if err != nil {
		return fmt.Errorf("error dumping vm memory, %v", err)
	}
	fmt.Printf("Successfully submitted memory dump request of VM %s\n", vmName)
	return nil
}

func getMemoryDump(namespace, vmName string, virtClient kubecli.KubevirtClient) error {
	if createClaim {
		if claimName == "" {
			return fmt.Errorf("missing claim name")
		}
		if err := createPVCforMemoryDump(namespace, vmName, claimName, virtClient); err != nil {
			return err
		}
	}

	if err := createMemoryDump(namespace, vmName, claimName, virtClient); err != nil {
		return err
	}

	if outputFile != "" {
		return downloadMemoryDump(namespace, vmName, virtClient)
	}

	return nil
}

func downloadMemoryDump(namespace, vmName string, virtClient kubecli.KubevirtClient) error {
	if outputFile == "" {
		return fmt.Errorf("missing outputFile to download the memory dump")
	}

	// Wait for the memorydump to complete
	claimName, err := WaitMemoryDumpComplete(virtClient, namespace, vmName, processingWaitInterval, processingWaitTotal)
	if err != nil {
		return err
	}
	if claimName == "" {
		return fmt.Errorf("claim name not on vm memory dump request")
	}
	exportSource := k8sv1.TypedLocalObjectReference{
		APIGroup: &k8sv1.SchemeGroupVersion.Group,
		Kind:     "PersistentVolumeClaim",
		Name:     claimName,
	}
	vmexportName := getVMExportName(vmName, claimName)
	vmExportInfo := &vmexport.VMExportInfo{

		ShouldCreate: true,
		Insecure:     true,
		KeepVme:      false,
		OutputFile:   outputFile,
		Namespace:    namespace,
		Name:         vmexportName,
		ExportSource: exportSource,
		PortForward:  portForward,
		LocalPort:    localPort,
	}

	if portForward {
		vmExportInfo.ServiceURL = fmt.Sprintf("127.0.0.1:%s", localPort)
	}
	// User wants a raw download, will decompress gzipped file if necessary
	if format == "raw" {
		vmExportInfo.Decompress = true
	}

	// User wants the output in a file, create
	output, err := os.Create(vmExportInfo.OutputFile)
	if err != nil {
		return err
	}
	vmExportInfo.OutputWriter = output
	return vmexport.DownloadVirtualMachineExport(virtClient, vmExportInfo)
}

func waitForMemoryDump(virtClient kubecli.KubevirtClient, namespace, vmName string, interval, timeout time.Duration) (string, error) {
	var claimName string
	err := wait.PollImmediate(interval, timeout, func() (bool, error) {
		vm, err := virtClient.VirtualMachine(namespace).Get(context.Background(), vmName, &metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		if vm.Status.MemoryDumpRequest == nil {
			return false, nil
		}

		if vm.Status.MemoryDumpRequest.Phase != v1.MemoryDumpCompleted {
			fmt.Printf("Waiting for memorydump %s to complete, current phase: %s...\n", claimName, vm.Status.MemoryDumpRequest.Phase)
			return false, nil
		}

		claimName = vm.Status.MemoryDumpRequest.ClaimName
		fmt.Println("Memory dump completed successfully")
		return true, nil
	})

	return claimName, err
}

func removeMemoryDump(namespace, vmName string, virtClient kubecli.KubevirtClient) error {
	err := virtClient.VirtualMachine(namespace).RemoveMemoryDump(context.Background(), vmName)
	if err != nil {
		return fmt.Errorf("error removing memory dump association, %v", err)
	}
	fmt.Printf("Successfully submitted remove memory dump association of VM %s\n", vmName)
	return nil
}

func getVMExportName(vmName, claimName string) string {
	return fmt.Sprintf("export-%s-%s", vmName, claimName)
}
