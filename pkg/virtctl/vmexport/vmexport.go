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
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	pb "github.com/cheggaaa/pb/v3"
	"github.com/spf13/cobra"

	k8sv1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
	kubectlutil "k8s.io/kubectl/pkg/util"

	virtv1 "kubevirt.io/api/core/v1"
	exportv1 "kubevirt.io/api/export/v1alpha1"
	"kubevirt.io/client-go/kubecli"

	snapshotv1 "kubevirt.io/api/snapshot/v1alpha1"

	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

const (
	// Available vmexport functions
	CREATE   = "create"
	DELETE   = "delete"
	DOWNLOAD = "download"

	// Available vmexport flags
	OUTPUT_FLAG         = "--output"
	VOLUME_FLAG         = "--volume"
	VM_FLAG             = "--vm"
	SNAPSHOT_FLAG       = "--snapshot"
	INSECURE_FLAG       = "--insecure"
	KEEP_FLAG           = "--keep-vme"
	PVC_FLAG            = "--pvc"
	TTL_FLAG            = "--ttl"
	MANIFEST_FLAG       = "--manifest"
	OUTPUT_FORMAT_FLAG  = "--manifest-output-format"
	SERVICE_URL_FLAG    = "--service-url"
	INCLUDE_SECRET_FLAG = "--include-secret"
	PORT_FORWARD_FLAG   = "--port-forward"

	// Possible output format for manifests
	OUTPUT_FORMAT_JSON = "json"
	OUTPUT_FORMAT_YAML = "yaml"

	ACCEPT           = "Accept"
	APPLICATION_YAML = "application/yaml"
	APPLICATION_JSON = "application/json"

	// processingWaitInterval is the time interval used to wait for a virtualMachineExport to be ready
	processingWaitInterval = 2 * time.Second
	// processingWaitTotal is the maximum time used to wait for a virtualMachineExport to be ready
	processingWaitTotal = 2 * time.Minute

	// exportTokenHeader is the http header used to download the exported volume using the secret token
	exportTokenHeader = "x-kubevirt-export-token"
	// secretTokenKey is the entry used to store the token in the virtualMachineExport secret
	secretTokenKey = "token"
	// secretTokenLenght is the lenght of the randomly generated token
	secretTokenLenght = 20

	// ErrRequiredFlag serves as error message when a mandatory flag is missing
	ErrRequiredFlag = "Need to specify the '%s' flag when using '%s'"
	// ErrIncompatibleFlag serves as error message when an incompatible flag is used
	ErrIncompatibleFlag = "The '%s' flag is incompatible with '%s'"
	// ErrRequiredExportType serves as error message when no export kind is provided
	ErrRequiredExportType = "Need to specify export kind when attempting to create a VirtualMachineExport [--pvc|--vm|--snapshot]"
	// ErrIncompatibleExportType serves as error message when an export kind is provided with an incompatible argument
	ErrIncompatibleExportType = "Should not specify export kind"

	// progressBarCycle is a const used to store the cycle displayed in the progress bar when downloading the exported volume
	progressBarCycle = `"[___________________]" "[==>________________]" "[====>______________]" "[======>____________]" "[========>__________]" "[==========>________]" "[============>______]" "[==============>____]" "[================>__]" "[==================>]"`
)

var (
	// Flags
	vm                   string
	snapshot             string
	pvc                  string
	outputFile           string
	insecure             bool
	keepVme              bool
	shouldCreate         bool
	includeSecret        bool
	exportManifest       bool
	portForward          string
	serviceUrl           string
	volumeName           string
	ttl                  string
	manifestOutputFormat string
)

type exportFunc func(client kubecli.KubevirtClient, vmeInfo *VMExportInfo) error

type HTTPClientCreator func(*http.Transport, bool) *http.Client

type exportCompleteFunc func(kubecli.KubevirtClient, *VMExportInfo, time.Duration, time.Duration) error

// ExportProcessingComplete is used to store the function to wait for the export object to be ready.
// Useful for unit tests.
var ExportProcessingComplete exportCompleteFunc = waitForVirtualMachineExport

type VMExportInfo struct {
	ShouldCreate bool
	Insecure     bool
	KeepVme      bool
	OutputFile   string
	VolumeName   string
	Namespace    string
	Name         string
	ExportSource k8sv1.TypedLocalObjectReference
	TTL          metav1.Duration
	ShouldCreate   bool
	Insecure       bool
	KeepVme        bool
	IncludeSecret  bool
	ExportManifest bool
	PortForward    string
	OutputFile     string
	OutputWriter   io.Writer
	VolumeName     string
	Namespace      string
	Name           string
	OutputFormat   string
	ServiceURL     string
	ExportSource   k8sv1.TypedLocalObjectReference
	TTL            metav1.Duration
}

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
  
	# Download a volume from an already existing VirtualMachineExport (--volume is optional when only one volume is available)
	{{ProgramName}} vmexport download vm1-export --volume=volume1 --output=disk.img.gz

	# Download a volume as before but through local port 5410
	{{ProgramName}} vmexport download vm1-export --volume=volume1 --output=disk.img.gz --port-forward=5410
  
	# Create a VirtualMachineExport and download the requested volume from it
	{{ProgramName}} vmexport download vm1-export --vm=vm1 --volume=volume1 --output=disk.img.gz`

	return usage
}

// NewVirtualMachineExportCommand returns a cobra.Command to handle the export process
func NewVirtualMachineExportCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "vmexport",
		Short:   "Export a VM volume.",
		Example: usage(),
		Args:    templates.ExactArgs("vmexport", 2),
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
	cmd.Flags().BoolVar(&insecure, "insecure", false, "When used with the 'download' option, specifies that the http request should be insecure.")
	cmd.Flags().BoolVar(&keepVme, "keep-vme", false, "When used with the 'download' option, specifies that the vmexport object should not be deleted after the download finishes.")
	cmd.Flags().StringVar(&ttl, "ttl", "", "The time after the export was created that it is eligible to be automatically deleted, defaults to 2 hours by the server side if not specified")
	cmd.Flags().StringVar(&manifestOutputFormat, "manifest-output-format", "", "Manifest output format, defaults to Yaml. Valid options are yaml or json")
	cmd.Flags().StringVar(&serviceUrl, "service-url", "", "Specify service url to use in the returned manifest, instead of the external URL in the Virtual Machine export status. This is useful for NodePorts or if you don't have an external URL configured")
	cmd.Flags().StringVar(&portForward, "port-forward", "", "Configures port-forwarding on the specified port. Useful to download without proper ingress/route configuration")
	cmd.Flags().BoolVar(&includeSecret, "include-secret", false, "When used with manifest and set to true include a secret that contains proper headers for CDI to import using the manifest")
	cmd.Flags().BoolVar(&exportManifest, "manifest", false, "Instead of downloading a volume, retrieve the VM manifest")
	cmd.SetUsageTemplate(templates.UsageTemplate())

	return cmd
// run serves as entrypoint for the vmexport command
func (c *command) run(args []string) error {
	var vmeInfo VMExportInfo
	if err := parseExportArguments(args, &vmeInfo); err != nil {
		return err

	namespace, _, err := c.clientConfig.Namespace()
	if err != nil {
		return err
	}
	vmeInfo.Namespace = namespace

	virtClient, err := kubecli.GetKubevirtClientFromClientConfig(c.clientConfig)
	if err != nil {
		return fmt.Errorf("Cannot obtain KubeVirt client: %v", err)
	}

	// Finally, run the vmexport function (create|delete|download)
	if err := exportFunction(virtClient, &vmeInfo); err != nil {
		return err
	}

	return nil
}

// parseExportArguments parses and validates vmexport arguments and flags. These arguments should always be:
// 	1. The vmexport function (create|delete|download)
// 	2. The VirtualMachineExport name
func parseExportArguments(args []string, vmeInfo *VMExportInfo) error {
	funcName := strings.ToLower(args[0])

	// Assign the appropiate vmexport function and make sure the used flags are compatible
	switch funcName {
	case CREATE:
		exportFunction = CreateVirtualMachineExport
		if err := handleCreateFlags(); err != nil {
			return err
		}
	case DELETE:
		exportFunction = DeleteVirtualMachineExport
		if err := handleDeleteFlags(); err != nil {
			return err
		}
	case DOWNLOAD:
		exportFunction = DownloadVirtualMachineExport
		if err := handleDownloadFlags(); err != nil {
			return err
		}
	default:
		return fmt.Errorf("Invalid function '%s'", funcName)
	}

	// VirtualMachineExport name
	vmeInfo.Name = args[1]

	// We store the flags in a struct to avoid relying on global variables
	if err := c.initVMExportInfo(vmeInfo); err != nil {
		return err
	}

	return nil
}

func (c *command) initVMExportInfo(vmeInfo *VMExportInfo) error {
	vmeInfo.ExportSource = getExportSource()
	vmeInfo.OutputFile = outputFile
	vmeInfo.ShouldCreate = shouldCreate
	vmeInfo.Insecure = insecure
	vmeInfo.KeepVme = keepVme
	vmeInfo.VolumeName = volumeName
	vmeInfo.ServiceURL = serviceUrl
	vmeInfo.OutputFormat = manifestOutputFormat
	vmeInfo.IncludeSecret = includeSecret
	vmeInfo.ExportManifest = exportManifest
	if portForward != "" {
		vmeInfo.PortForward = portForward
		vmeInfo.Insecure = true
		if vmeInfo.ServiceURL == "" {
			// Defaulting to localhost
			vmeInfo.ServiceURL = fmt.Sprintf("127.0.0.1:%s", portForward)
		}
	}
	if ttl != "" {
		duration, err := time.ParseDuration(ttl)
		if err != nil {
			return err
		}
		vmeInfo.TTL = metav1.Duration{Duration: duration}
	}
	return nil
}

// getVirtualMachineExport serves as a wrapper to get the VirtualMachineExport object
func getVirtualMachineExport(client kubecli.KubevirtClient, vmeInfo *VMExportInfo) (*exportv1.VirtualMachineExport, error) {
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	return vmexport, nil
}

// CreateVirtualMachineExport serves as a wrapper to create the virtualMachineExport object and, if needed, do error handling
func CreateVirtualMachineExport(client kubecli.KubevirtClient, vmeInfo *VMExportInfo) error {
	vmexport, err := getVirtualMachineExport(client, vmeInfo)
	if err != nil {
		return err
	}
	if vmexport != nil {
		return fmt.Errorf("VirtualMachineExport '%s/%s' already exists", vmeInfo.Namespace, vmeInfo.Name)
	}

	secretRef := getExportSecretName(vmeInfo.Name)
	vmexport = &exportv1.VirtualMachineExport{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vmeInfo.Name,
			Namespace: vmeInfo.Namespace,
		},
		Spec: exportv1.VirtualMachineExportSpec{
			TokenSecretRef: &secretRef,
			Source:         vmeInfo.ExportSource,
		},
	}
	if vmeInfo.TTL.Duration > 0 {
		vmexport.Spec.TTLDuration = &vmeInfo.TTL
	}

	vmexport, err = client.VirtualMachineExport(vmeInfo.Namespace).Create(context.TODO(), vmexport, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	// Generate/get secret to be used with the vmexport
	_, err = getOrCreateTokenSecret(client, vmexport)
	if err != nil {
		return err
	}

	fmt.Printf("VirtualMachineExport '%s/%s' created succesfully\n", vmeInfo.Namespace, vmeInfo.Name)
	return nil
}

// DeleteVirtualMachineExport serves as a wrapper to delete the virtualMachineExport object
func DeleteVirtualMachineExport(client kubecli.KubevirtClient, vmeInfo *VMExportInfo) error {
	if err := client.VirtualMachineExport(vmeInfo.Namespace).Delete(context.TODO(), vmeInfo.Name, metav1.DeleteOptions{}); err != nil {
		if !k8serrors.IsNotFound(err) {
			return err
		}
		fmt.Printf("VirtualMachineExport '%s/%s' does not exist", vmeInfo.Namespace, vmeInfo.Name)
		return nil
	}

	fmt.Printf("VirtualMachineExport '%s/%s' deleted succesfully\n", vmeInfo.Namespace, vmeInfo.Name)
	return nil
}

// DownloadVirtualMachineExport handles the process of downloading the requested volume from a VirtualMachineExport object
func DownloadVirtualMachineExport(client kubecli.KubevirtClient, vmeInfo *VMExportInfo) error {
	if vmeInfo.ShouldCreate {
		if err := CreateVirtualMachineExport(client, vmeInfo); err != nil {
			if !errExportAlreadyExists(err) {
				return err
			}
		}
	}

	if !vmeInfo.KeepVme {
		defer DeleteVirtualMachineExport(client, vmeInfo)
	}

	if vmeInfo.PortForward != "" {
		stopChan, err := setupPortForward(client, vmeInfo)
		if err != nil {
			return err
		}
		defer close(stopChan)
	}

	// Wait for the vmexport object to be ready
	if err := ExportProcessingComplete(client, vmeInfo, processingWaitInterval, processingWaitTotal); err != nil {
		return err
	}

	vmexport, err := getVirtualMachineExport(client, vmeInfo)
	if err != nil {
		return err
	}
	if vmexport == nil {
		return fmt.Errorf("Unable to get '%s/%s' VirtualMachineExport", vmeInfo.Namespace, vmeInfo.Name)
	}

	// Download the exported volume
	if err := downloadVolume(client, vmexport, vmeInfo); err != nil {
		return err
	}

	return nil
}

// downloadVolume handles the process of downloading the requested volume from a VirtualMachineExport
func downloadVolume(client kubecli.KubevirtClient, vmexport *exportv1.VirtualMachineExport, vmeInfo *VMExportInfo) error {
	// Extract the URL from the vmexport
	downloadUrl, err := GetUrlFromVirtualMachineExport(vmexport, vmeInfo)
	if err != nil {
		return err
	}

	resp, err := handleHTTPRequest(client, vmexport, downloadUrl, vmeInfo.Insecure)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Bad status: %s", resp.Status)
	}

	output, err := os.Create(vmeInfo.OutputFile)
	if err != nil {
		return err
	}
	defer util.CloseIOAndCheckErr(output, nil)

	// Lastly, copy the file to the expected output
	if err := copyFileWithProgressBar(output, resp); err != nil {
		return err
	}

	fmt.Println("Download finished succesfully")
	return nil
}

// GetUrlFromVirtualMachineExport inspects the VirtualMachineExport status to fetch the extected URL
func GetUrlFromVirtualMachineExport(vmexport *exportv1.VirtualMachineExport, vmeInfo *VMExportInfo) (string, error) {
	var downloadUrl string

	if vmexport.Status.Links == nil || vmexport.Status.Links.External == nil || len(vmexport.Status.Links.External.Volumes) <= 0 {
		return "", fmt.Errorf("Unable to access the volume info from '%s/%s' VirtualMachineExport", vmexport.Namespace, vmexport.Name)
	}

	volumeNumber := len(vmexport.Status.Links.External.Volumes)
	if volumeNumber > 1 && vmeInfo.VolumeName == "" {
		return "", fmt.Errorf("Detected more than one downloadable volume in '%s/%s' VirtualMachineExport: Select the expected volume using the --volume flag", vmexport.Namespace, vmexport.Name)
	}

	for _, exportVolume := range vmexport.Status.Links.External.Volumes {
		// Access the requested volume
		if volumeNumber == 1 || exportVolume.Name == vmeInfo.VolumeName {
			for _, format := range exportVolume.Formats {
				// We always attempt to find and get the compressed file URL, so we only break the loop when one is found
				if format.Format == exportv1.KubeVirtGz || format.Format == exportv1.ArchiveGz {
					downloadUrl = format.Url
					break
				} else if format.Format == exportv1.KubeVirtRaw {
					downloadUrl = format.Url
				}
			}
		}
	}

	if downloadUrl == "" {
		return "", fmt.Errorf("Unable to get a valid URL from '%s/%s' VirtualMachineExport", vmexport.Namespace, vmexport.Name)
	}

	return downloadUrl, nil
}

// waitForVirtualMachineExport waits for the VirtualMachineExport status and external links to be ready
func waitForVirtualMachineExport(client kubecli.KubevirtClient, vmeInfo *VMExportInfo, interval, timeout time.Duration) error {
	err := wait.PollImmediate(interval, timeout, func() (bool, error) {
		vmexport, err := getVirtualMachineExport(client, vmeInfo)
		if err != nil || vmexport == nil {
			return false, err
		}

		if vmexport.Status == nil {
			return false, nil
		}

		if vmexport.Status.Phase != exportv1.Ready {
			fmt.Printf("Waiting for VM Export %s status to be ready...\n", vmeInfo.Name)
			return false, nil
		}

		if vmexport.Status.Links == nil || vmexport.Status.Links.External == nil {
			fmt.Printf("Waiting for VM Export %s external links to be available...\n", vmeInfo.Name)
			return false, nil
		}

		fmt.Printf("Processing completed successfully\n")
		return true, nil
	})

	return err
}

// handleHTTPRequest generates the GET request with proper certificate handling
func handleHTTPRequest(client kubecli.KubevirtClient, vmexport *exportv1.VirtualMachineExport, downloadUrl string, insecure bool) (*http.Response, error) {
	token, err := getTokenFromSecret(client, vmexport)
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
	httpClient := httpClientCreatorFunc(transport, insecure)

	// Generate and do the request
	req, _ := http.NewRequest("GET", downloadUrl, nil)
	req.Header.Set(exportTokenHeader, token)

	return httpClient.Do(req)
}

// getHTTPClient assigns the default, non-mocked HTTP client
func getHTTPClient(transport *http.Transport, insecure bool) *http.Client {
	if insecure == true {
		transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: insecure,
			},
		}
	}

	client := &http.Client{Transport: transport}
	return client
}

// copyFileWithProgressBar serves as a wrapper to copy the file with a progress bar
func copyFileWithProgressBar(output *os.File, resp *http.Response) error {
	barTemplate := fmt.Sprintf(`{{ "Downloading file:" }} {{counters . }} {{ cycle . %s }} {{speed . }}`, progressBarCycle)

	// start bar based on our template
	bar := pb.ProgressBarTemplate(barTemplate).Start(0)
	defer bar.Finish()
	rd := bar.NewProxyReader(resp.Body)
	bar.Start()

	_, err := io.Copy(output, rd)
	return err
}

// getOrCreateTokenSecret obtains a token secret to be used along with the virtualMachineExport
func getOrCreateTokenSecret(client kubecli.KubevirtClient, vmexport *exportv1.VirtualMachineExport) (*k8sv1.Secret, error) {
	// Securely randomize a 20 char string to be used as a token
	token, err := util.GenerateSecureRandomString(secretTokenLenght)
	if err != nil {
		return nil, err
	}

	secret := &k8sv1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getExportSecretName(vmexport.Name),
			Namespace: vmexport.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(vmexport, schema.GroupVersionKind{
					Group:   exportv1.SchemeGroupVersion.Group,
					Version: exportv1.SchemeGroupVersion.Version,
					Kind:    "VirtualMachineExport",
				}),
			},
		},
		Type: k8sv1.SecretTypeOpaque,
		Data: map[string][]byte{
			secretTokenKey: []byte(token),
		},
	}

	secret, err = client.CoreV1().Secrets(vmexport.Namespace).Create(context.Background(), secret, metav1.CreateOptions{})
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return nil, err
	}

	return secret, nil
}

// getTokenFromSecret extracts the token from the secret specified on the virtualMachineExport
func getTokenFromSecret(client kubecli.KubevirtClient, vmexport *exportv1.VirtualMachineExport) (string, error) {
	secretName := ""
	if vmexport.Status != nil && vmexport.Status.TokenSecretRef != nil {
		secretName = *vmexport.Status.TokenSecretRef
	}

	secret, err := client.CoreV1().Secrets(vmexport.Namespace).Get(context.Background(), secretName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	token := secret.Data[secretTokenKey]
	return string(token), nil
}

// getExportSource creates and assembles the object that'll be used as a source in the virtualMachineExport
func getExportSource() k8sv1.TypedLocalObjectReference {
	var exportSource k8sv1.TypedLocalObjectReference
	if vm != "" {
		exportSource = k8sv1.TypedLocalObjectReference{
			APIGroup: &virtv1.SchemeGroupVersion.Group,
			Kind:     "VirtualMachine",
			Name:     vm,
		}
	}
	if snapshot != "" {
		exportSource = k8sv1.TypedLocalObjectReference{
			APIGroup: &snapshotv1.SchemeGroupVersion.Group,
			Kind:     "VirtualMachineSnapshot",
			Name:     snapshot,
		}
	}
	if pvc != "" {
		exportSource = k8sv1.TypedLocalObjectReference{
			APIGroup: &k8sv1.SchemeGroupVersion.Group,
			Kind:     "PersistentVolumeClaim",
			Name:     pvc,
		}
	}

	return exportSource
}

// handleCreateFlags ensures that only compatible flag combinations are used with 'create'
func handleCreateFlags() error {
	if vm == "" && snapshot == "" && pvc == "" {
		return fmt.Errorf(ErrRequiredExportType)
	}

	if outputFile != "" {
		return fmt.Errorf(ErrIncompatibleFlag, OUTPUT_FLAG, CREATE)
	}
	if volumeName != "" {
		return fmt.Errorf(ErrIncompatibleFlag, VOLUME_FLAG, CREATE)
	}
	if insecure {
		return fmt.Errorf(ErrIncompatibleFlag, INSECURE_FLAG, CREATE)
	}
	if keepVme {
		return fmt.Errorf(ErrIncompatibleFlag, KEEP_FLAG, CREATE)
	}
	if portForward != "" {
		return fmt.Errorf(ErrIncompatibleFlag, PORT_FORWARD_FLAG, CREATE)
	}
	if serviceUrl != "" {
		return fmt.Errorf(ErrIncompatibleFlag, SERVICE_URL_FLAG, CREATE)
	}

	return nil
}

// handleDeleteFlags ensures that only compatible flag combinations are used with 'delete'
func handleDeleteFlags() error {
	if vm != "" || snapshot != "" || pvc != "" {
		return fmt.Errorf(ErrIncompatibleExportType)
	}

	if outputFile != "" {
		return fmt.Errorf(ErrIncompatibleFlag, OUTPUT_FLAG, DELETE)
	}
	if volumeName != "" {
		return fmt.Errorf(ErrIncompatibleFlag, VOLUME_FLAG, DELETE)
	}
	if insecure {
		return fmt.Errorf(ErrIncompatibleFlag, INSECURE_FLAG, DELETE)
	}
	if keepVme {
		return fmt.Errorf(ErrIncompatibleFlag, KEEP_FLAG, DELETE)
	}
	if portForward != "" {
		return fmt.Errorf(ErrIncompatibleFlag, PORT_FORWARD_FLAG, DELETE)
	}
	if serviceUrl != "" {
		return fmt.Errorf(ErrIncompatibleFlag, SERVICE_URL_FLAG, DELETE)
	}
	return nil
}

// handleDownloadFlags ensures that only compatible flag combinations are used with 'download'
func handleDownloadFlags() error {
	// We assume that the vmexport should be created if a source has been specified
	if hasSource := vm != "" || snapshot != "" || pvc != ""; hasSource {
		shouldCreate = true
	}

	if outputFile == "" {
		return fmt.Errorf(ErrRequiredFlag, OUTPUT_FLAG, DOWNLOAD)
	}

	return nil
}

// getExportSecretName builds the name of the token secret based on the virtualMachineExport object
func getExportSecretName(vmexportName string) string {
	return fmt.Sprintf("secret-%s", vmexportName)
}

// errExportAlreadyExists is used to the determine if an error happened when creating an already existing vmexport
func errExportAlreadyExists(err error) bool {
	return strings.Contains(err.Error(), "VirtualMachineExport") && strings.Contains(err.Error(), "already exists")
}

// Port-forward functions

// translateServicePortToTargetPort tranlates the specified port to be used with the service's pod
func translateServicePortToTargetPort(localPort string, remotePort string, svc k8sv1.Service, pod k8sv1.Pod) ([]string, error) {
	ports := []string{}
	portnum, err := strconv.Atoi(remotePort)
	if err != nil {
		return ports, err
	}
	containerPort, err := kubectlutil.LookupContainerPortNumberByServicePort(svc, pod, int32(portnum))
	if err != nil {
		// can't resolve a named port, or Service did not declare this port, return an error
		return ports, err
	}

	// convert the resolved target port back to a string
	remotePort = strconv.Itoa(int(containerPort))
	if localPort != remotePort {
		return append(ports, fmt.Sprintf("%s:%s", localPort, remotePort)), nil
	}

	return append(ports, remotePort), nil
}

// waitForExportServiceToBeReady waits until the vmexport service is ready for port-forwarding
func waitForExportServiceToBeReady(client kubecli.KubevirtClient, vmeInfo *VMExportInfo, interval, timeout time.Duration) (*k8sv1.Service, error) {
	service := &k8sv1.Service{}
	serviceName := fmt.Sprintf("virt-export-%s", vmeInfo.Name)
	err := wait.PollImmediate(interval, timeout, func() (bool, error) {
		vmexport, err := getVirtualMachineExport(client, vmeInfo)
		if err != nil || vmexport == nil {
			return false, err
		}

		if vmexport.Status == nil || vmexport.Status.Phase != exportv1.Ready {
			fmt.Printf("waiting for VM Export %s status to be ready...\n", vmeInfo.Name)
			return false, nil
		}

		service, err = client.CoreV1().Services(vmeInfo.Namespace).Get(context.TODO(), serviceName, metav1.GetOptions{})
		if err != nil {
			if k8serrors.IsNotFound(err) {
				fmt.Printf("waiting for service %s to be ready before port-forwarding...\n", serviceName)
				return false, nil
			}
			return false, err
		}
		fmt.Printf("service %s is ready for port-forwarding\n", service.Name)
		return true, nil
	})
	return service, err
}

// runPortForward is the actual function that runs the port-forward. Meant to be run concurrently
func runPortForward(client kubecli.KubevirtClient, pod k8sv1.Pod, namespace string, ports []string, stopChan, readyChan chan struct{}) error {
	// Create a port forwarding request
	req := client.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(pod.Name).
		Namespace(namespace).
		SubResource("portforward")

	// Set up the port forwarding options
	transport, upgrader, err := spdy.RoundTripperFor(client.Config())
	if err != nil {
		log.Fatalf("Failed to set up transport: %v", err)
	}
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, "POST", req.URL())

	// Start port-forwarding
	fw, err := portforward.New(dialer, ports, stopChan, readyChan, os.Stdout, os.Stderr)
	if err != nil {
		log.Fatalf("Failed to setup port forward: %v", err)
	}
	return fw.ForwardPorts()
}

// setupPortForward runs a port-forward after initializing all required arguments
func setupPortForward(client kubecli.KubevirtClient, vmeInfo *VMExportInfo) (chan struct{}, error) {
	// Wait for the vmexport object to be ready
	service, err := waitForExportServiceToBeReady(client, vmeInfo, processingWaitInterval, processingWaitTotal)
	if err != nil {
		return nil, err
	}

	// Extract the target pod selector from the service
	podSelector := labels.SelectorFromSet(service.Spec.Selector)

	// List the pods matching the selector
	podList, err := client.CoreV1().Pods(vmeInfo.Namespace).List(context.Background(), metav1.ListOptions{LabelSelector: podSelector.String()})
	if err != nil {
		return nil, fmt.Errorf("Failed to list pods: %v", err)
	}

	// Pick the first pod to forward the port
	if len(podList.Items) == 0 {
		return nil, fmt.Errorf("No pods found for the service %s", service.Name)
	}

	// Set up the port forwarding ports
	ports, err := translateServicePortToTargetPort(vmeInfo.PortForward, "443", *service, podList.Items[0])
	if err != nil {
		return nil, err
	}

	stopChan := make(chan struct{}, 1)
	readyChan := make(chan struct{})
	go runPortForward(client, podList.Items[0], vmeInfo.Namespace, ports, stopChan, readyChan)

	// Wait for the port forwarding to be ready
	select {
	case <-readyChan:
		fmt.Println("Port forwarding is ready.")
	case <-time.After(30 * time.Second):
		return nil, fmt.Errorf("Timeout waiting for port forwarding to be ready.")
	}
	return stopChan, nil
}
