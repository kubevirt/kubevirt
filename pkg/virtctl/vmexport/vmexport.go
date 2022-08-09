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
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	pb "gopkg.in/cheggaaa/pb.v1"
	k8sv1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/clientcmd"
	exportv1 "kubevirt.io/api/export/v1alpha1"

	"k8s.io/apimachinery/pkg/util/rand"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

const (
	// Available vmexport functions
	CREATE   = "create"
	DELETE   = "delete"
	DOWNLOAD = "download"

	// Available vmexport flags
	OUTPUT_FLAG   = "--output"
	CREATE_FLAG   = "--create"
	VOLUME_FLAG   = "--volume"
	VM_FLAG       = "--vm"
	SNAPSHOT_FLAG = "--snapshot"
	PVC_FLAG      = "--pvc"
	RAW_FLAG      = "--raw"

	// processingWaitInterval is the time interval used to wait for a virtualMachineExport to be ready
	processingWaitInterval = 2 * time.Second
	// processingWaitTotal is the maximum time used to wait for a virtualMachineExport to be ready
	processingWaitTotal = 24 * time.Hour

	// exportTokenHeader is the http header used to download the exported volume using the secret token
	exportTokenHeader = "x-kubevirt-export-token"
	// dirExportTokenHeader is the http header used to download a disk.img from a directory using the secret token
	dirExportTokenHeader = "/disk.img?x-kubevirt-export-token"

	// secretTokenEntry is the entry used to store the token in the virtualMachineExport secret
	secretTokenEntry = "token"

	// ErrRequiredFlag serves as error message when a mandatory flag is missing
	ErrRequiredFlag = "Need to specify the '%s' flag when using '%s'"
	// ErrIncompatibleFlag serves as error message when an incompatible flag is used
	ErrIncompatibleFlag = "The '%s' flag is incompatible with '%s'"
	// ErrRequiredExportType serves as error message when no export kind is provided
	ErrRequiredExportType = "Need to specify export kind when attempting to create a VirtualMachineExport [--pvc|--vm|--snapshot]"
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
	downloadRaw  bool
	volumeName   string

	// vmexport info
	vmexportName string
	resourceName string
	resourceKind string
	funcName     string
)

type exportFunc func(client kubecli.KubevirtClient, namespace string) error

type HTTPClientCreator func(*http.Transport) *http.Client

type command struct {
	clientConfig clientcmd.ClientConfig
}

var exportFunction exportFunc

var httpClientCreatorFunc HTTPClientCreator

// SetHTTPClientCreator allows overriding the default http client (useful for unit testing)
func SetHTTPClientCreator(f HTTPClientCreator) {
	httpClientCreatorFunc = f
}

// SetDefaultHTTPClientCreator sets the http client creator back to default
func SetDefaultHTTPClientCreator() {
	httpClientCreatorFunc = getHTTPClient
}

func init() {
	SetDefaultHTTPClientCreator()
}

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
			return v.run(args)
		},
	}

	cmd.Flags().StringVar(&vm, "vm", "", "Sets VirtualMachine as vmexport kind and specifies the vm name.")
	cmd.Flags().StringVar(&snapshot, "snapshot", "", "Sets VirtualMachineSnapshot as vmexport kind and specifies the snapshot name.")
	cmd.Flags().StringVar(&pvc, "pvc", "", "Sets PersistentVolumeClaim as vmexport kind and specifies the PVC name.")
	cmd.MarkFlagsMutuallyExclusive("vm", "snapshot", "pvc")
	cmd.Flags().StringVar(&outputFile, "output", "", "Specifies the output path of the volume to be downloaded.")
	cmd.Flags().StringVar(&volumeName, "volume", "", "Specifies the volume to be downloaded.")
	cmd.Flags().BoolVar(&shouldCreate, "create", shouldCreate, "When used with the 'download' option, specifies that a VirtualMachineExport should be created from scratch.")
	cmd.Flags().BoolVar(&downloadRaw, "raw", downloadRaw, "When used with the 'download' option, specifies that the file should be downloaded in a uncompressed format.")
	cmd.SetUsageTemplate(templates.UsageTemplate())

	return cmd
}

// run serves as entrypoint for the vmexport command
func (c *command) run(args []string) error {
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
	case CREATE:
		exportFunction = createVirtualMachineExport
		if err := handleCreateFlags(args); err != nil {
			return err
		}
	case DELETE:
		exportFunction = deleteVirtualMachineExport
		if err := handleDeleteFlags(args); err != nil {
			return err
		}
	case DOWNLOAD:
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
		return fmt.Errorf("VirtualMachineExport '%s/%s' already exists", namespace, vmexportName)
	}

	// Generate/get secret to be used with the vmexport
	secret, err := getOrCreateTokenSecret(client, namespace)
	if err != nil {
		return err
	}

	vmexport = &exportv1.VirtualMachineExport{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vmexportName,
			Namespace: namespace,
		},
		Spec: exportv1.VirtualMachineExportSpec{
			TokenSecretRef: secret.Name,
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

	// If it exists, attempt to delete the related secret
	if err := deleteExportSecret(client, namespace); err != nil {
		return err
	}

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
	if vmexport == nil {
		return fmt.Errorf("Unable to get '%s/%s' VirtualMachineExport", namespace, vmexportName)
	}
	// Wait for the vmexport object to be ready
	if err := waitForVirtualMachineExport(client, namespace, processingWaitInterval, processingWaitTotal); err != nil {
		return err
	}
	// Download the exported volume
	if err := downloadVolume(client, vmexport); err != nil {
		return err
	}

	fmt.Println("Cleaning resources...")
	if err := deleteVirtualMachineExport(client, namespace); err != nil {
		return err
	}

	return nil
}

// downloadVolume handles the process of downloading the requested volume from a VirtualMachineExport
func downloadVolume(client kubecli.KubevirtClient, vmexport *exportv1.VirtualMachineExport) error {
	output, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer util.CloseIOAndCheckErr(output, nil)

	// Extract the URL from the vmexport
	downloadUrl, httpHeader, err := getUrlFromVirtualMachineExport(client, vmexport)
	if err != nil {
		return err
	}

	resp, err := handleHTTPRequest(client, vmexport, downloadUrl, httpHeader)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Bad status: %s", resp.Status)
	}

	// Lastly, copy the file to the expected output
	if err := copyFileWithProgressBar(output, resp); err != nil {
		return err
	}

	fmt.Println("Download finished succesfully")
	return nil
}

// getUrlFromVirtualMachineExport inspects the VirtualMachineExport status to fetch the extected URL
func getUrlFromVirtualMachineExport(client kubecli.KubevirtClient, vmexport *exportv1.VirtualMachineExport) (string, string, error) {
	var downloadUrl string
	var httpHeader string

	if vmexport.Status.Links.External.Volumes == nil {
		return "", "", fmt.Errorf("Unable to access the volume info from '%s/%s' VirtualMachineExport", vmexport.Namespace, vmexport.Name)
	}

	for _, exportVolume := range vmexport.Status.Links.External.Volumes {
		// Access the requested volume
		if exportVolume.Name == volumeName {
			for _, format := range exportVolume.Formats {
				// If the --raw flag is used, we download the uncompressed file
				if downloadRaw {
					if format.Format == exportv1.KubeVirtRaw {
						downloadUrl = format.Url
						httpHeader = exportTokenHeader
						break
					} else if format.Format == exportv1.Dir {
						downloadUrl = format.Url
						httpHeader = dirExportTokenHeader
						break
					}
				} else {
					if format.Format == exportv1.KubeVirtGz || format.Format == exportv1.ArchiveGz {
						downloadUrl = format.Url
						httpHeader = exportTokenHeader
						break
					}
				}
			}
		}
	}

	if downloadUrl == "" {
		return "", "", fmt.Errorf("Unable to get a valid URL from '%s/%s' VirtualMachineExport", vmexport.Namespace, vmexport.Name)
	}

	return downloadUrl, httpHeader, nil
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
			return false, nil
		}

		if vmexport.Status.Links == nil || vmexport.Status.Links.External == nil {
			fmt.Printf("Waiting for VM Export %s external links to be available...\n", vmexportName)
			return false, nil
		}

		fmt.Printf("Processing completed successfully\n")
		return done, nil
	})

	return err
}

// handleHTTPRequest generates the GET request with proper certificate handling
func handleHTTPRequest(client kubecli.KubevirtClient, vmexport *exportv1.VirtualMachineExport, downloadUrl, httpHeader string) (*http.Response, error) {
	token, err := getTokenFromSecret(client, vmexport.Namespace)
	if err != nil {
		return nil, err
	}

	// Create new certPool and append our external SSL certificate
	cert := vmexport.Status.Links.External.Cert
	roots := x509.NewCertPool()
	roots.AppendCertsFromPEM([]byte(cert))
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{RootCAs: roots},
	}
	httpClient := httpClientCreatorFunc(transport)

	// Generate and do the request
	req, _ := http.NewRequest("GET", downloadUrl, nil)
	req.Header.Set(httpHeader, token)

	return httpClient.Do(req)
}

// getHTTPClient assigns the default, non-mocked HTTP client
func getHTTPClient(transport *http.Transport) *http.Client {
	if transport == nil {
		transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
	}

	client := &http.Client{Transport: transport}
	return client
}

// copyFileWithProgressBar serves as a wrapper to copy the file with a progress bar
func copyFileWithProgressBar(output *os.File, resp *http.Response) error {
	// TODO: Unable to get file size when downloading using the compressed URL
	filesize, _ := strconv.Atoi(resp.Header.Get("Content-Length"))

	// Initiate progress bar
	bar := pb.New(filesize).SetUnits(pb.U_BYTES)
	rd := bar.NewProxyReader(resp.Body)
	fmt.Println("Downloading file...")
	bar.Start()

	_, err := io.Copy(output, rd)
	bar.Finish()
	return err
}

// getOrCreateTokenSecret obtains a token secret to be used along with the virtualMachineExport
func getOrCreateTokenSecret(client kubecli.KubevirtClient, namespace string) (*k8sv1.Secret, error) {
	// Generate a random, 20 char string to be used as a token
	token := rand.String(20)

	secret := &k8sv1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getExportSecretName(),
			Namespace: namespace,
		},
		Type: k8sv1.SecretTypeOpaque,
		Data: map[string][]byte{
			secretTokenEntry: []byte(token),
		},
	}

	_, err := client.CoreV1().Secrets(namespace).Create(context.Background(), secret, metav1.CreateOptions{})
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return nil, err
	}

	return secret, nil
}

// deleteExportSecret deletes the secret assigned to the virtualMachineExport
func deleteExportSecret(client kubecli.KubevirtClient, namespace string) error {
	secretName := getExportSecretName()
	if err := client.CoreV1().Secrets(namespace).Delete(context.TODO(), secretName, metav1.DeleteOptions{}); err != nil {
		if !k8serrors.IsNotFound(err) {
			return err
		}
		return nil
	}

	fmt.Printf("Secret '%s/%s' deleted succesfully\n", namespace, secretName)
	return nil
}

// getTokenFromSecret extracts the token from the secret specified on the virtualMachineExport
func getTokenFromSecret(client kubecli.KubevirtClient, namespace string) (string, error) {
	secretName := getExportSecretName()
	secret, err := client.CoreV1().Secrets(namespace).Get(context.Background(), secretName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	token := secret.Data[secretTokenEntry]
	return string(token), nil
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
	if funcName == CREATE {
		if outputFile != "" {
			return fmt.Errorf(ErrIncompatibleFlag, OUTPUT_FLAG, funcName)
		}
		if volumeName != "" {
			return fmt.Errorf(ErrIncompatibleFlag, VOLUME_FLAG, funcName)
		}
		if shouldCreate {
			return fmt.Errorf(ErrIncompatibleFlag, CREATE_FLAG, funcName)
		}
		if downloadRaw {
			return fmt.Errorf(ErrIncompatibleFlag, RAW_FLAG, funcName)
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
		return fmt.Errorf(ErrIncompatibleFlag, OUTPUT_FLAG, funcName)
	}
	if volumeName != "" {
		return fmt.Errorf(ErrIncompatibleFlag, VOLUME_FLAG, funcName)
	}
	if shouldCreate {
		return fmt.Errorf(ErrIncompatibleFlag, CREATE_FLAG, funcName)
	}
	if downloadRaw {
		return fmt.Errorf(ErrIncompatibleFlag, RAW_FLAG, funcName)
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
		return fmt.Errorf(ErrRequiredFlag, OUTPUT_FLAG, funcName)
	}
	if volumeName == "" {
		return fmt.Errorf(ErrRequiredFlag, VOLUME_FLAG, funcName)
	}

	return nil
}

// getExportSecretName builds the name of the token secret based on the virtualMachineExport object
func getExportSecretName() string {
	return fmt.Sprintf("secret-%s", vmexportName)
}
