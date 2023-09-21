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
	"compress/gzip"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
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
	FORMAT_FLAG         = "--format"
	PVC_FLAG            = "--pvc"
	TTL_FLAG            = "--ttl"
	MANIFEST_FLAG       = "--manifest"
	OUTPUT_FORMAT_FLAG  = "--manifest-output-format"
	SERVICE_URL_FLAG    = "--service-url"
	INCLUDE_SECRET_FLAG = "--include-secret"
	PORT_FORWARD_FLAG   = "--port-forward"
	LOCAL_PORT_FLAG     = "--local-port"

	// Possible output format for manifests
	OUTPUT_FORMAT_JSON = "json"
	OUTPUT_FORMAT_YAML = "yaml"

	// Possible output format for volumes
	GZIP_FORMAT = "gzip"
	RAW_FORMAT  = "raw"

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
	ErrRequiredFlag = "need to specify the '%s' flag when using '%s'"
	// ErrIncompatibleFlag serves as error message when an incompatible flag is used
	ErrIncompatibleFlag = "the '%s' flag is incompatible with '%s'"
	// ErrRequiredExportType serves as error message when no export kind is provided
	ErrRequiredExportType = "need to specify export kind when attempting to create a VirtualMachineExport [--pvc|--vm|--snapshot]"
	// ErrIncompatibleExportType serves as error message when an export kind is provided with an incompatible argument
	ErrIncompatibleExportType = "should not specify export kind"
	// ErrIncompatibleExportTypeManifest serves as error message when a PVC kind is defined when getting manifest
	ErrIncompatibleExportTypeManifest = "cannot get manifest for PVC export"
	// ErrInvalidValue ensures that the value provided in a flag is one of the acceptable values
	ErrInvalidValue = "%s is not a valid value, acceptable values are %s"

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
	portForward          bool
	format               string
	localPort            string
	serviceUrl           string
	volumeName           string
	ttl                  string
	manifestOutputFormat string
)

type exportFunc func(client kubecli.KubevirtClient, vmeInfo *VMExportInfo) error

type HTTPClientCreator func(*http.Transport, bool) *http.Client

type PortForwardFunc func(client kubecli.KubevirtClient, pod k8sv1.Pod, namespace string, ports []string, stopChan, readyChan chan struct{}, portChan chan uint16) error

type exportCompleteFunc func(kubecli.KubevirtClient, *VMExportInfo, time.Duration, time.Duration) error

// ExportProcessingComplete is used to store the function to wait for the export object to be ready.
// Useful for unit tests.
var ExportProcessingComplete exportCompleteFunc = waitForVirtualMachineExport

type VMExportInfo struct {
	ShouldCreate   bool
	Insecure       bool
	KeepVme        bool
	IncludeSecret  bool
	ExportManifest bool
	Decompress     bool
	PortForward    bool
	LocalPort      string
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
	cmd          *cobra.Command
}

var exportFunction exportFunc

var httpClientCreatorFunc HTTPClientCreator

var startPortForward PortForwardFunc

// SetHTTPClientCreator allows overriding the default http client (useful for unit testing)
func SetHTTPClientCreator(f HTTPClientCreator) {
	httpClientCreatorFunc = f
}

// SetDefaultHTTPClientCreator sets the http client creator back to default
func SetDefaultHTTPClientCreator() {
	httpClientCreatorFunc = getHTTPClient
}

// SetPortForwarder allows overriding the default port-forwarder (useful for unit testing)
func SetPortForwarder(f PortForwardFunc) {
	startPortForward = f
}

// SetDefaultPortForwarder sets the port forwarder back to default
func SetDefaultPortForwarder() {
	startPortForward = runPortForward
}

func init() {
	SetDefaultHTTPClientCreator()
	SetDefaultPortForwarder()
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
	{{ProgramName}} vmexport download vm1-export --volume=volume1 --output=disk.img.gz --port-forward --local-port=5410
  
	# Create a VirtualMachineExport and download the requested volume from it
	{{ProgramName}} vmexport download vm1-export --vm=vm1 --volume=volume1 --output=disk.img.gz

	# Create a VirtualMachineExport and get the VirtualMachine manifest in Yaml format
	{{ProgramName}} vmexport download vm1-export --vm=vm1 --manifest

	# Get the VirtualMachine manifest in Yaml format from an existing VirtualMachineExport including CDI header secret
	{{ProgramName}} vmexport download existing-export --include-secret --manifest`
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
			v := command{clientConfig: clientConfig, cmd: cmd}
			return v.run(args)
		},
	}

	cmd.Flags().StringVar(&vm, "vm", "", "Sets VirtualMachine as vmexport kind and specifies the vm name.")
	cmd.Flags().StringVar(&snapshot, "snapshot", "", "Sets VirtualMachineSnapshot as vmexport kind and specifies the snapshot name.")
	cmd.Flags().StringVar(&pvc, "pvc", "", "Sets PersistentVolumeClaim as vmexport kind and specifies the PVC name.")
	cmd.MarkFlagsMutuallyExclusive("vm", "snapshot", "pvc")
	cmd.Flags().StringVar(&outputFile, "output", "", "Specifies the output path of the volume to be downloaded.")
	cmd.Flags().StringVar(&volumeName, "volume", "", "Specifies the volume to be downloaded.")
	cmd.Flags().StringVar(&format, "format", "", "Used to specify the format of the downloaded image. There's two options: gzip (default) and raw.")
	cmd.Flags().BoolVar(&insecure, "insecure", false, "When used with the 'download' option, specifies that the http request should be insecure.")
	cmd.Flags().BoolVar(&keepVme, "keep-vme", false, "When used with the 'download' option, specifies that the vmexport object should not be deleted after the download finishes.")
	cmd.Flags().StringVar(&ttl, "ttl", "", "The time after the export was created that it is eligible to be automatically deleted, defaults to 2 hours by the server side if not specified")
	cmd.Flags().StringVar(&manifestOutputFormat, "manifest-output-format", "", "Manifest output format, defaults to Yaml. Valid options are yaml or json")
	cmd.Flags().StringVar(&serviceUrl, "service-url", "", "Specify service url to use in the returned manifest, instead of the external URL in the Virtual Machine export status. This is useful for NodePorts or if you don't have an external URL configured")
	cmd.Flags().BoolVar(&portForward, "port-forward", false, "Configures port-forwarding on a random port. Useful to download without proper ingress/route configuration")
	cmd.Flags().StringVar(&localPort, "local-port", "0", "Defines the specific port to be used in port-forward.")
	cmd.Flags().BoolVar(&includeSecret, "include-secret", false, "When used with manifest and set to true include a secret that contains proper headers for CDI to import using the manifest")
	cmd.Flags().BoolVar(&exportManifest, "manifest", false, "Instead of downloading a volume, retrieve the VM manifest")
	cmd.SetUsageTemplate(templates.UsageTemplate())

	return cmd
}

// run serves as entrypoint for the vmexport command
func (c *command) run(args []string) error {
	var vmeInfo VMExportInfo
	if err := c.parseExportArguments(args, &vmeInfo); err != nil {
		return err
	}
	// If writing to a file, the OutputWriter will also be a Closer
	if closer, ok := vmeInfo.OutputWriter.(io.Closer); ok && vmeInfo.OutputFile != "" {
		defer util.CloseIOAndCheckErr(closer, nil)
	}

	namespace, _, err := c.clientConfig.Namespace()
	if err != nil {
		return err
	}
	vmeInfo.Namespace = namespace

	virtClient, err := kubecli.GetKubevirtClientFromClientConfig(c.clientConfig)
	if err != nil {
		return fmt.Errorf("cannot obtain KubeVirt client: %v", err)
	}

	// Finally, run the vmexport function (create|delete|download)
	if err := exportFunction(virtClient, &vmeInfo); err != nil {
		return err
	}

	return nil
}

// parseExportArguments parses and validates vmexport arguments and flags. These arguments should always be:
//  1. The vmexport function (create|delete|download)
//  2. The VirtualMachineExport name
func (c *command) parseExportArguments(args []string, vmeInfo *VMExportInfo) error {
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
		return fmt.Errorf("invalid function '%s'", funcName)
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
	// User wants the output in a file, create
	if outputFile != "" {
		output, err := os.Create(vmeInfo.OutputFile)
		if err != nil {
			return err
		}
		vmeInfo.OutputWriter = output
	} else {
		vmeInfo.OutputWriter = c.cmd.OutOrStdout()
	}
	// If raw format is specified, we'll attempt to download and decompress a gzipped volume
	if format == RAW_FORMAT {
		vmeInfo.Decompress = true
	}
	vmeInfo.ShouldCreate = shouldCreate
	vmeInfo.Insecure = insecure
	vmeInfo.KeepVme = keepVme
	vmeInfo.VolumeName = volumeName
	vmeInfo.ServiceURL = serviceUrl
	vmeInfo.OutputFormat = manifestOutputFormat
	vmeInfo.IncludeSecret = includeSecret
	vmeInfo.ExportManifest = exportManifest
	if portForward {
		vmeInfo.PortForward = portForward
		vmeInfo.Insecure = true
		// Defaults to 0, which will be replaced by a random available port
		vmeInfo.LocalPort = localPort
		if vmeInfo.ServiceURL == "" {
			vmeInfo.ServiceURL = fmt.Sprintf("127.0.0.1:%s", vmeInfo.LocalPort)
		}
	}
	vmeInfo.TTL = metav1.Duration{}
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
	vmexport, err := client.VirtualMachineExport(vmeInfo.Namespace).Get(context.TODO(), vmeInfo.Name, metav1.GetOptions{})
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
		fmt.Printf("VirtualMachineExport '%s/%s' does not exist\n", vmeInfo.Namespace, vmeInfo.Name)
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

	if !vmeInfo.KeepVme && !vmeInfo.ExportManifest {
		defer DeleteVirtualMachineExport(client, vmeInfo)
	}

	if vmeInfo.PortForward {
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
		return fmt.Errorf("unable to get '%s/%s' VirtualMachineExport", vmeInfo.Namespace, vmeInfo.Name)
	}

	if vmeInfo.ExportManifest {
		// Grab the VM Manifest and display it.
		if err := getVirtualMachineManifest(client, vmexport, vmeInfo); err != nil {
			return err
		}
	} else {
		// Download the exported volume
		if err := downloadVolume(client, vmexport, vmeInfo); err != nil {
			return err
		}
	}
	return nil
}

func printRequestBody(client kubecli.KubevirtClient, vmexport *exportv1.VirtualMachineExport, vmeInfo *VMExportInfo, manifestUrl string, headers map[string]string) error {
	resp, err := HandleHTTPRequest(client, vmexport, manifestUrl, vmeInfo.Insecure, vmeInfo.ServiceURL, headers)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}
	bodyAll, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	fmt.Fprintf(vmeInfo.OutputWriter, "%s", bodyAll)
	return nil
}

func getVirtualMachineManifest(client kubecli.KubevirtClient, vmexport *exportv1.VirtualMachineExport, vmeInfo *VMExportInfo) error {
	manifestMap, err := GetManifestUrlsFromVirtualMachineExport(vmexport, vmeInfo)
	if err != nil {
		return err
	}
	headers := make(map[string]string)
	headers[ACCEPT] = APPLICATION_YAML
	if strings.ToLower(vmeInfo.OutputFormat) == OUTPUT_FORMAT_JSON {
		headers[ACCEPT] = APPLICATION_JSON
	}
	if err := printRequestBody(client, vmexport, vmeInfo, manifestMap[exportv1.AllManifests], headers); err != nil {
		return err
	}
	if vmeInfo.IncludeSecret {
		if err := printRequestBody(client, vmexport, vmeInfo, manifestMap[exportv1.AuthHeader], headers); err != nil {
			return err
		}
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

	resp, err := HandleHTTPRequest(client, vmexport, downloadUrl, vmeInfo.Insecure, vmeInfo.ServiceURL, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// Lastly, copy the file to the expected output
	if err := copyFileWithProgressBar(vmeInfo.OutputWriter, resp, vmeInfo.Decompress); err != nil {
		return err
	}

	// Prevent this output ending up in the stdout
	if vmeInfo.OutputFile != "" {
		fmt.Println("Download finished succesfully")
	}
	return nil
}

func replaceUrlWithServiceUrl(manifestUrl string, vmeInfo *VMExportInfo) (string, error) {
	// Replace internal URL with specified URL
	manUrl, err := url.Parse(manifestUrl)
	if err != nil {
		return "", err
	}
	if vmeInfo.ServiceURL != "" {
		manUrl.Host = vmeInfo.ServiceURL
	}
	return manUrl.String(), nil
}

// GetUrlFromVirtualMachineExport inspects the VirtualMachineExport status to fetch the extected URL
func GetUrlFromVirtualMachineExport(vmexport *exportv1.VirtualMachineExport, vmeInfo *VMExportInfo) (string, error) {
	var (
		downloadUrl string
		err         error
		format      exportv1.VirtualMachineExportVolumeFormat
		links       *exportv1.VirtualMachineExportLink
	)

	if vmeInfo.ServiceURL == "" && vmexport.Status.Links != nil && vmexport.Status.Links.External != nil {
		links = vmexport.Status.Links.External
	} else if vmexport.Status.Links != nil && vmexport.Status.Links.Internal != nil {
		links = vmexport.Status.Links.Internal
	}
	if links == nil || len(links.Volumes) <= 0 {
		return "", fmt.Errorf("unable to access the volume info from '%s/%s' VirtualMachineExport", vmexport.Namespace, vmexport.Name)
	}
	volumeNumber := len(links.Volumes)
	if volumeNumber > 1 && vmeInfo.VolumeName == "" {
		return "", fmt.Errorf("detected more than one downloadable volume in '%s/%s' VirtualMachineExport: Select the expected volume using the --volume flag", vmexport.Namespace, vmexport.Name)
	}
	for _, exportVolume := range links.Volumes {
		// Access the requested volume
		if volumeNumber == 1 || exportVolume.Name == vmeInfo.VolumeName {
			for _, format = range exportVolume.Formats {
				if format.Format == exportv1.KubeVirtGz || format.Format == exportv1.ArchiveGz || format.Format == exportv1.KubeVirtRaw {
					downloadUrl, err = replaceUrlWithServiceUrl(format.Url, vmeInfo)
					if err != nil {
						return "", err
					}
				}
				// By default, we always attempt to find and get the compressed file URL,
				// so we only break the loop when one is found.
				if format.Format == exportv1.KubeVirtGz || format.Format == exportv1.ArchiveGz {
					break
				}
			}
		}
	}

	// No need to decompress file if format is not gzip
	if format.Format == exportv1.KubeVirtRaw {
		vmeInfo.Decompress = false
	}

	if downloadUrl == "" {
		return "", fmt.Errorf("unable to get a valid URL from '%s/%s' VirtualMachineExport", vmexport.Namespace, vmexport.Name)
	}

	return downloadUrl, nil
}

// GetManifestUrlsFromVirtualMachineExport retrieves the manifest URLs from VirtualMachineExport status
func GetManifestUrlsFromVirtualMachineExport(vmexport *exportv1.VirtualMachineExport, vmeInfo *VMExportInfo) (map[exportv1.ExportManifestType]string, error) {
	res := make(map[exportv1.ExportManifestType]string, 0)
	if vmeInfo.ServiceURL == "" {
		if vmexport.Status.Links == nil || vmexport.Status.Links.External == nil || len(vmexport.Status.Links.External.Manifests) == 0 {
			return nil, fmt.Errorf("unable to access the manifest info from '%s/%s' VirtualMachineExport", vmexport.Namespace, vmexport.Name)
		}

		for _, manifest := range vmexport.Status.Links.External.Manifests {
			res[manifest.Type] = manifest.Url
		}
	} else {
		if vmexport.Status.Links == nil || vmexport.Status.Links.Internal == nil || len(vmexport.Status.Links.Internal.Manifests) == 0 {
			return nil, fmt.Errorf("unable to access the manifest info from '%s/%s' VirtualMachineExport", vmexport.Namespace, vmexport.Name)
		}

		for _, manifest := range vmexport.Status.Links.Internal.Manifests {
			// Replace internal URL with specified URL
			manUrl, err := url.Parse(manifest.Url)
			if err != nil {
				return nil, err
			}
			manUrl.Host = vmeInfo.ServiceURL
			res[manifest.Type] = manUrl.String()
		}
	}
	return res, nil
}

// waitForVirtualMachineExport waits for the VirtualMachineExport status and external links to be ready
func waitForVirtualMachineExport(client kubecli.KubevirtClient, vmeInfo *VMExportInfo, interval, timeout time.Duration) error {
	err := wait.PollImmediate(interval, timeout, func() (bool, error) {
		vmexport, err := getVirtualMachineExport(client, vmeInfo)
		if err != nil {
			return false, err
		}

		if vmexport == nil {
			fmt.Printf("couldn't get VM Export %s, waiting for it to be created...\n", vmeInfo.Name)
			return false, nil
		}

		if vmexport.Status == nil {
			return false, nil
		}

		if vmexport.Status.Phase != exportv1.Ready {
			fmt.Printf("waiting for VM Export %s status to be ready...\n", vmeInfo.Name)
			return false, nil
		}

		if vmeInfo.ServiceURL == "" {
			if vmexport.Status.Links == nil || vmexport.Status.Links.External == nil {
				fmt.Printf("waiting for VM Export %s external links to be available...\n", vmeInfo.Name)
				return false, nil
			}
		} else {
			if vmexport.Status.Links == nil || vmexport.Status.Links.Internal == nil {
				fmt.Printf("waiting for VM Export %s internal links to be available...\n", vmeInfo.Name)
				return false, nil
			}
		}
		return true, nil
	})

	return err
}

// HandleHTTPRequestFunc function used to handle http requests
type HandleHTTPRequestFunc func(client kubecli.KubevirtClient, vmexport *exportv1.VirtualMachineExport, downloadUrl string, insecure bool, exportURL string, headers map[string]string) (*http.Response, error)

// instance of function used to handle http requests
var HandleHTTPRequest HandleHTTPRequestFunc = handleHTTPGetRequest

// handleHTTPGetRequest generates the GET request with proper certificate handling
func handleHTTPGetRequest(client kubecli.KubevirtClient, vmexport *exportv1.VirtualMachineExport, downloadUrl string, insecure bool, exportURL string, headers map[string]string) (*http.Response, error) {
	token, err := getTokenFromSecret(client, vmexport)
	if err != nil {
		return nil, err
	}

	var cert string
	// Create new certPool and append our external SSL certificate
	if exportURL == "" {
		cert = vmexport.Status.Links.External.Cert
	} else {
		cert = vmexport.Status.Links.Internal.Cert
	}
	roots := x509.NewCertPool()
	roots.AppendCertsFromPEM([]byte(cert))
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{RootCAs: roots},
	}
	httpClient := httpClientCreatorFunc(transport, insecure)

	// Generate and do the request
	req, _ := http.NewRequest("GET", downloadUrl, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	req.Header.Set(exportTokenHeader, token)

	return httpClient.Do(req)
}

// getHTTPClient assigns the default, non-mocked HTTP client
func getHTTPClient(transport *http.Transport, insecure bool) *http.Client {
	if insecure {
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
func copyFileWithProgressBar(output io.Writer, resp *http.Response, decompress bool) error {
	var rd io.Reader
	barTemplate := fmt.Sprintf(`{{ "Downloading file:" }} {{counters . }} {{ cycle . %s }} {{speed . }}`, progressBarCycle)

	// start bar based on our template
	bar := pb.ProgressBarTemplate(barTemplate).Start(0)
	defer bar.Finish()
	barRd := bar.NewProxyReader(resp.Body)
	rd = barRd
	bar.Start()

	if decompress {
		// Create a new gzip reader
		gzipReader, err := gzip.NewReader(barRd)
		if err != nil {
			return err
		}
		defer gzipReader.Close()
		rd = gzipReader
		fmt.Println("Decompressing image:")
	}

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
	if portForward {
		return fmt.Errorf(ErrIncompatibleFlag, PORT_FORWARD_FLAG, CREATE)
	}
	if localPort != "0" {
		return fmt.Errorf(ErrIncompatibleFlag, LOCAL_PORT_FLAG, CREATE)
	}
	if format != "" {
		return fmt.Errorf(ErrIncompatibleFlag, FORMAT_FLAG, CREATE)
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
	if portForward {
		return fmt.Errorf(ErrIncompatibleFlag, PORT_FORWARD_FLAG, DELETE)
	}
	if localPort != "0" {
		return fmt.Errorf(ErrIncompatibleFlag, LOCAL_PORT_FLAG, DELETE)
	}
	if format != "" {
		return fmt.Errorf(ErrIncompatibleFlag, FORMAT_FLAG, DELETE)
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

	if portForward {
		port, err := strconv.Atoi(localPort)
		if err != nil || port < 0 || port > 65535 {
			return fmt.Errorf(ErrInvalidValue, LOCAL_PORT_FLAG, "valid port numbers")
		}
	}

	if format != "" && format != GZIP_FORMAT && format != RAW_FORMAT {
		return fmt.Errorf(ErrInvalidValue, FORMAT_FLAG, "gzip/raw")
	}

	if exportManifest {
		if volumeName != "" {
			return fmt.Errorf(ErrIncompatibleFlag, VOLUME_FLAG, MANIFEST_FLAG)
		}

		manifestOutputFormat = strings.ToLower(manifestOutputFormat)
		if manifestOutputFormat != OUTPUT_FORMAT_JSON && manifestOutputFormat != OUTPUT_FORMAT_YAML && manifestOutputFormat != "" {
			return fmt.Errorf(ErrInvalidValue, OUTPUT_FORMAT_FLAG, "json/yaml")
		}

		if pvc != "" {
			return fmt.Errorf(ErrIncompatibleFlag, PVC_FLAG, MANIFEST_FLAG)
		}
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
	ports, err := translateServicePortToTargetPort(vmeInfo.LocalPort, "443", *service, podList.Items[0])
	if err != nil {
		return nil, err
	}

	stopChan := make(chan struct{}, 1)
	readyChan := make(chan struct{})
	portChan := make(chan uint16)
	go startPortForward(client, podList.Items[0], vmeInfo.Namespace, ports, stopChan, readyChan, portChan)

	// Wait for the port forwarding to be ready
	select {
	case <-readyChan:
		fmt.Println("Port forwarding is ready.")
		// Using 0 allows listening on a random available port.
		// Now we need to find out which port was used
		if vmeInfo.LocalPort == "0" {
			localPort := <-portChan
			close(portChan)
			vmeInfo.ServiceURL = fmt.Sprintf("127.0.0.1:%d", localPort)
		}
	case <-time.After(30 * time.Second):
		return nil, fmt.Errorf("Timeout waiting for port forwarding to be ready.")
	}
	return stopChan, nil
}

// runPortForward is the actual function that runs the port-forward. Meant to be run concurrently
func runPortForward(client kubecli.KubevirtClient, pod k8sv1.Pod, namespace string, ports []string, stopChan, readyChan chan struct{}, portChan chan uint16) error {
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
	slicedPorts := strings.Split(ports[0], ":")
	if len(slicedPorts) == 2 && slicedPorts[0] == "0" {
		// If the local port is 0, then the port-forwarder will pick a random available port.
		// We need to send this port number back to the caller.
		go func() {
			<-readyChan
			forwardedPorts, err := fw.GetPorts()
			if err != nil {
				log.Fatalf("Failed to get forwarded ports: %v", err)
			}
			portChan <- forwardedPorts[0].Local
		}()
	}
	return fw.ForwardPorts()
}
