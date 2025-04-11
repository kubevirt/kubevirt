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
 * Copyright The KubeVirt Authors.
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

	"github.com/cheggaaa/pb/v3"
	"github.com/spf13/cobra"
	k8sv1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
	kubectlutil "k8s.io/kubectl/pkg/util"

	virtv1 "kubevirt.io/api/core/v1"
	exportv1 "kubevirt.io/api/export/v1beta1"
	snapshotv1 "kubevirt.io/api/snapshot/v1beta1"
	"kubevirt.io/client-go/kubecli"

	virtwait "kubevirt.io/kubevirt/pkg/apimachinery/wait"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/virtctl/clientconfig"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

const (
	// Available vmexport functions
	CREATE   = "create"
	DELETE   = "delete"
	DOWNLOAD = "download"

	// Available vmexport flags
	OUTPUT_FLAG            = "--output"
	VOLUME_FLAG            = "--volume"
	VM_FLAG                = "--vm"
	SNAPSHOT_FLAG          = "--snapshot"
	INSECURE_FLAG          = "--insecure"
	KEEP_FLAG              = "--keep-vme"
	DELETE_FLAG            = "--delete-vme"
	FORMAT_FLAG            = "--format"
	PVC_FLAG               = "--pvc"
	TTL_FLAG               = "--ttl"
	MANIFEST_FLAG          = "--manifest"
	OUTPUT_FORMAT_FLAG     = "--manifest-output-format"
	SERVICE_URL_FLAG       = "--service-url"
	INCLUDE_SECRET_FLAG    = "--include-secret"
	PORT_FORWARD_FLAG      = "--port-forward"
	LOCAL_PORT_FLAG        = "--local-port"
	RETRY_FLAG             = "--retry"
	LABELS_FLAG            = "--labels"
	ANNOTATIONS_FLAG       = "--annotations"
	READINESS_TIMEOUT_FLAG = "--readiness-timeout"

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
	// DefaultProcessingWaitTotal is the default maximum time used to wait for a virtualMachineExport to be ready
	DefaultProcessingWaitTotal = 2 * time.Minute

	// exportTokenHeader is the http header used to download the exported volume using the secret token
	exportTokenHeader = "x-kubevirt-export-token"
	// secretTokenKey is the entry used to store the token in the virtualMachineExport secret
	secretTokenKey = "token"

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
	deleteVme            bool
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
	downloadRetries      int
	resourceLabels       []string
	resourceAnnotations  []string
	readinessTimeout     string
)

type VMExportInfo struct {
	ShouldCreate     bool
	Insecure         bool
	KeepVme          bool
	DeleteVme        bool
	IncludeSecret    bool
	ExportManifest   bool
	Decompress       bool
	PortForward      bool
	LocalPort        string
	OutputFile       string
	OutputWriter     io.Writer
	VolumeName       string
	Namespace        string
	Name             string
	OutputFormat     string
	ServiceURL       string
	ExportSource     k8sv1.TypedLocalObjectReference
	TTL              metav1.Duration
	DownloadRetries  int
	ReadinessTimeout time.Duration
	Labels           map[string]string
	Annotations      map[string]string
}

type command struct {
	cmd *cobra.Command
}

// WaitForVirtualMachineExportFn allows overriding the function to wait for the export object to be ready (useful for unit testing)
var WaitForVirtualMachineExportFn = WaitForVirtualMachineExport

// GetHTTPClientFn allows overriding the default http client (useful for unit testing)
var GetHTTPClientFn = GetHTTPClient

// HandleHTTPGetRequestFn allows overriding the default http GET request handler (useful for unit testing)
var HandleHTTPGetRequestFn = HandleHTTPGetRequest

// RunPortForwardFn allows overriding the default port-forwarder (useful for unit testing)
var RunPortForwardFn = RunPortForward

var exportFunction func(client kubecli.KubevirtClient, vmeInfo *VMExportInfo) error

// TODO Should use cmd.Printf and cmd.SetOut
var printToOutput = fmt.Printf

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
func NewVirtualMachineExportCommand() *cobra.Command {
	c := command{}
	cmd := &cobra.Command{
		Use:     "vmexport",
		Short:   "Export a VM volume.",
		Example: usage(),
		Args:    cobra.ExactArgs(2),
		RunE:    c.run,
	}

	shouldCreate = false

	cmd.Flags().StringVar(&vm, "vm", "", "Sets VirtualMachine as vmexport kind and specifies the vm name.")
	cmd.Flags().StringVar(&snapshot, "snapshot", "", "Sets VirtualMachineSnapshot as vmexport kind and specifies the snapshot name.")
	cmd.Flags().StringVar(&pvc, "pvc", "", "Sets PersistentVolumeClaim as vmexport kind and specifies the PVC name.")
	cmd.MarkFlagsMutuallyExclusive("vm", "snapshot", "pvc")
	cmd.Flags().StringVar(&outputFile, "output", "", "Specifies the output path of the volume to be downloaded.")
	cmd.Flags().StringVar(&volumeName, "volume", "", "Specifies the volume to be downloaded.")
	cmd.Flags().StringVar(&format, "format", "", "Used to specify the format of the downloaded image. There's two options: gzip (default) and raw.")
	cmd.Flags().BoolVar(&insecure, "insecure", false, "When used with the 'download' option, specifies that the http request should be insecure.")
	cmd.Flags().BoolVar(&keepVme, "keep-vme", false, "When used with the 'download' option, specifies that the vmexport object should always be retained after the download finishes.")
	cmd.Flags().BoolVar(&deleteVme, "delete-vme", false, "When used with the 'download' option, specifies that the vmexport object should always be deleted after the download finishes.")
	cmd.MarkFlagsMutuallyExclusive("keep-vme", "delete-vme")
	cmd.Flags().StringVar(&ttl, "ttl", "", "The time after the export was created that it is eligible to be automatically deleted, defaults to 2 hours by the server side if not specified")
	cmd.Flags().StringVar(&manifestOutputFormat, "manifest-output-format", "", "Manifest output format, defaults to Yaml. Valid options are yaml or json")
	cmd.Flags().StringVar(&serviceUrl, "service-url", "", "Specify service url to use in the returned manifest, instead of the external URL in the Virtual Machine export status. This is useful for NodePorts or if you don't have an external URL configured")
	cmd.Flags().BoolVar(&portForward, "port-forward", false, "Configures port-forwarding on a random port. Useful to download without proper ingress/route configuration")
	cmd.Flags().StringVar(&localPort, "local-port", "0", "Defines the specific port to be used in port-forward.")
	cmd.Flags().IntVar(&downloadRetries, "retry", 0, "When export server returns a transient error, we retry this number of times before giving up")
	cmd.Flags().BoolVar(&includeSecret, "include-secret", false, "When used with manifest and set to true include a secret that contains proper headers for CDI to import using the manifest")
	cmd.Flags().BoolVar(&exportManifest, "manifest", false, "Instead of downloading a volume, retrieve the VM manifest")
	cmd.Flags().StringSliceVar(&resourceLabels, "labels", nil, "Specify custom labels to VM export object and its associated pod")
	cmd.Flags().StringSliceVar(&resourceAnnotations, "annotations", nil, "Specify custom annotations to VM export object and its associated pod")
	cmd.Flags().StringVar(&readinessTimeout, "readiness-timeout", "", "Specify maximum wait for VM export object to be ready")
	cmd.SetUsageTemplate(templates.UsageTemplate())

	return cmd
}

// run serves as entrypoint for the vmexport command
func (c *command) run(cmd *cobra.Command, args []string) error {
	c.cmd = cmd

	var vmeInfo VMExportInfo
	if err := c.parseExportArguments(args, &vmeInfo); err != nil {
		return err
	}
	// If writing to a file, the OutputWriter will also be a Closer
	if closer, ok := vmeInfo.OutputWriter.(io.Closer); ok && vmeInfo.OutputFile != "" {
		defer util.CloseIOAndCheckErr(closer, nil)
	}

	virtClient, namespace, _, err := clientconfig.ClientAndNamespaceFromContext(cmd.Context())
	if err != nil {
		return err
	}
	vmeInfo.Namespace = namespace

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
	// User wants the output in a file, create
	if outputFile != "" && outputFile != "-" {
		vmeInfo.OutputFile = outputFile
		output, err := os.Create(vmeInfo.OutputFile)
		if err != nil {
			return err
		}
		vmeInfo.OutputWriter = output
	} else {
		vmeInfo.OutputWriter = c.cmd.OutOrStdout()
		vmeInfo.OutputFile = ""
		// Setting standard printer to output into stderr. We can then output
		// the volume into stdout without any interfering prints.
		printToOutput = func(format string, a ...interface{}) (int, error) {
			return fmt.Fprintf(os.Stderr, format, a...)
		}
	}
	// If raw format is specified, we'll attempt to download and decompress a gzipped volume
	if format == RAW_FORMAT {
		vmeInfo.Decompress = true
	}
	vmeInfo.DownloadRetries = downloadRetries
	vmeInfo.ShouldCreate = shouldCreate
	vmeInfo.Insecure = insecure
	vmeInfo.KeepVme = keepVme
	vmeInfo.DeleteVme = deleteVme
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
	vmeInfo.ReadinessTimeout = DefaultProcessingWaitTotal
	if readinessTimeout != "" {
		duration, err := time.ParseDuration(readinessTimeout)
		if err != nil {
			return err
		}
		vmeInfo.ReadinessTimeout = duration
	}

	vmeInfo.Labels = convertSliceToMap(resourceLabels)
	vmeInfo.Annotations = convertSliceToMap(resourceAnnotations)

	return nil
}

// Convert a slice of "key=value" strings to a map
func convertSliceToMap(slice []string) map[string]string {
	mapResult := make(map[string]string)
	for _, item := range slice {
		parts := strings.SplitN(item, "=", 2)
		if len(parts) == 2 {
			mapResult[parts[0]] = parts[1]
		}
	}
	return mapResult
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
			Name:        vmeInfo.Name,
			Namespace:   vmeInfo.Namespace,
			Labels:      vmeInfo.Labels,
			Annotations: vmeInfo.Annotations,
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

	printToOutput("VirtualMachineExport '%s/%s' created succesfully\n", vmeInfo.Namespace, vmeInfo.Name)
	return nil
}

// DeleteVirtualMachineExport serves as a wrapper to delete the virtualMachineExport object
func DeleteVirtualMachineExport(client kubecli.KubevirtClient, vmeInfo *VMExportInfo) error {
	if err := client.VirtualMachineExport(vmeInfo.Namespace).Delete(context.TODO(), vmeInfo.Name, metav1.DeleteOptions{}); err != nil {
		if !k8serrors.IsNotFound(err) {
			return err
		}
		printToOutput("VirtualMachineExport '%s/%s' does not exist\n", vmeInfo.Namespace, vmeInfo.Name)
		return nil
	}

	printToOutput("VirtualMachineExport '%s/%s' deleted succesfully\n", vmeInfo.Namespace, vmeInfo.Name)
	return nil
}

// DownloadVirtualMachineExport handles the process of downloading the requested volume from a VirtualMachineExport object
func DownloadVirtualMachineExport(client kubecli.KubevirtClient, vmeInfo *VMExportInfo) error {
	for attempt := 0; attempt <= vmeInfo.DownloadRetries; attempt++ {
		succeeded, err := downloadVirtualMachineExport(client, vmeInfo)
		if err != nil {
			return err
		}
		if succeeded {
			return nil
		}
		if attempt < vmeInfo.DownloadRetries {
			printToOutput("Retrying...\n")
			time.Sleep(2 * time.Second)
		}
	}
	return fmt.Errorf("retry count reached, exiting unsuccesfully")
}

func downloadVirtualMachineExport(client kubecli.KubevirtClient, vmeInfo *VMExportInfo) (bool, error) {
	if vmeInfo.ShouldCreate {
		if err := CreateVirtualMachineExport(client, vmeInfo); err != nil {
			if errExportAlreadyExists(err) {
				// Don't delete VMExports that already exist unless specified explicitely
				vmeInfo.KeepVme = true
			} else {
				return false, err
			}
		}
	}

	if shouldDeleteVMExport(vmeInfo) {
		defer DeleteVirtualMachineExport(client, vmeInfo)
	}

	if vmeInfo.PortForward {
		stopChan, err := setupPortForward(client, vmeInfo)
		if err != nil {
			return false, err
		}
		defer close(stopChan)
	}

	// Wait for the vmexport object to be ready
	if err := WaitForVirtualMachineExportFn(client, vmeInfo, processingWaitInterval, vmeInfo.ReadinessTimeout); err != nil {
		return false, err
	}

	vmexport, err := getVirtualMachineExport(client, vmeInfo)
	if err != nil {
		return false, err
	}
	if vmexport == nil {
		return false, fmt.Errorf("unable to get '%s/%s' VirtualMachineExport", vmeInfo.Namespace, vmeInfo.Name)
	}

	// Grab the VM Manifest and display it.
	if vmeInfo.ExportManifest {
		return getVirtualMachineManifest(client, vmexport, vmeInfo)
	}

	// Download the exported volume
	return downloadVolume(client, vmexport, vmeInfo)
}

func printRequestBody(client kubecli.KubevirtClient, vmexport *exportv1.VirtualMachineExport, vmeInfo *VMExportInfo, manifestUrl string, headers map[string]string) (bool, error) {
	resp, err := HandleHTTPGetRequestFn(client, vmexport, manifestUrl, vmeInfo.Insecure, vmeInfo.ServiceURL, headers)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		printToOutput("Bad status: %s\n", resp.Status)
		return false, nil
	}
	bodyAll, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}
	fmt.Fprintf(vmeInfo.OutputWriter, "%s", bodyAll)
	return true, nil
}

func getVirtualMachineManifest(client kubecli.KubevirtClient, vmexport *exportv1.VirtualMachineExport, vmeInfo *VMExportInfo) (bool, error) {
	manifestMap, err := GetManifestUrlsFromVirtualMachineExport(vmexport, vmeInfo)
	if err != nil {
		return false, err
	}
	headers := make(map[string]string)
	headers[ACCEPT] = APPLICATION_YAML
	if strings.ToLower(vmeInfo.OutputFormat) == OUTPUT_FORMAT_JSON {
		headers[ACCEPT] = APPLICATION_JSON
	}
	succeeded, err := printRequestBody(client, vmexport, vmeInfo, manifestMap[exportv1.AllManifests], headers)
	if err != nil || !succeeded {
		return false, err
	}
	if vmeInfo.IncludeSecret {
		return printRequestBody(client, vmexport, vmeInfo, manifestMap[exportv1.AuthHeader], headers)
	}
	return true, nil
}

// downloadVolume handles the process of downloading the requested volume from a VirtualMachineExport
func downloadVolume(client kubecli.KubevirtClient, vmexport *exportv1.VirtualMachineExport, vmeInfo *VMExportInfo) (bool, error) {
	// Extract the URL from the vmexport
	downloadUrl, err := GetUrlFromVirtualMachineExport(vmexport, vmeInfo)
	if err != nil {
		return false, err
	}

	resp, err := HandleHTTPGetRequestFn(client, vmexport, downloadUrl, vmeInfo.Insecure, vmeInfo.ServiceURL, nil)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		printToOutput("Bad status: %s\n", resp.Status)
		return false, nil
	}

	// Lastly, copy the file to the expected output
	if err := copyFileWithProgressBar(vmeInfo.OutputWriter, resp, vmeInfo.Decompress); err != nil {
		return false, err
	}

	printToOutput("Download finished succesfully\n")

	return true, nil
}

// shouldDeleteVMExport decides wether we should retain or delete a VMExport after a download. If delete/retain are not explicitly specified,
// the vmexport will be deleted when is created in the same instance as the download, retained otherwise.
func shouldDeleteVMExport(vmeInfo *VMExportInfo) bool {
	return !vmeInfo.ExportManifest && (vmeInfo.DeleteVme || (vmeInfo.ShouldCreate && !vmeInfo.KeepVme))
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

// WaitForVirtualMachineExport waits for the VirtualMachineExport status and external links to be ready
func WaitForVirtualMachineExport(client kubecli.KubevirtClient, vmeInfo *VMExportInfo, interval, timeout time.Duration) error {
	err := virtwait.PollImmediately(interval, timeout, func(_ context.Context) (bool, error) {
		vmexport, err := getVirtualMachineExport(client, vmeInfo)
		if err != nil {
			return false, err
		}

		if vmexport == nil {
			printToOutput("couldn't get VM Export %s, waiting for it to be created...\n", vmeInfo.Name)
			return false, nil
		}

		if vmexport.Status == nil {
			return false, nil
		}

		if vmexport.Status.Phase != exportv1.Ready {
			printToOutput("waiting for VM Export %s status to be ready...\n", vmeInfo.Name)
			return false, nil
		}

		if vmeInfo.ServiceURL == "" {
			if vmexport.Status.Links == nil || vmexport.Status.Links.External == nil {
				printToOutput("waiting for VM Export %s external links to be available...\n", vmeInfo.Name)
				return false, nil
			}
		} else {
			if vmexport.Status.Links == nil || vmexport.Status.Links.Internal == nil {
				printToOutput("waiting for VM Export %s internal links to be available...\n", vmeInfo.Name)
				return false, nil
			}
		}
		return true, nil
	})

	return err
}

// HandleHTTPGetRequest generates the GET request with proper certificate handling
func HandleHTTPGetRequest(client kubecli.KubevirtClient, vmexport *exportv1.VirtualMachineExport, downloadUrl string, insecure bool, exportURL string, headers map[string]string) (*http.Response, error) {
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
	httpClient := GetHTTPClientFn(transport, insecure)

	// Generate and do the request
	req, _ := http.NewRequest("GET", downloadUrl, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	req.Header.Set(exportTokenHeader, token)

	return httpClient.Do(req)
}

// GetHTTPClient assigns the default, non-mocked HTTP client
func GetHTTPClient(transport *http.Transport, insecure bool) *http.Client {
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
		printToOutput("Decompressing image:\n")
	}

	_, err := io.Copy(output, rd)
	return err
}

// getOrCreateTokenSecret obtains a token secret to be used along with the virtualMachineExport
func getOrCreateTokenSecret(client kubecli.KubevirtClient, vmexport *exportv1.VirtualMachineExport) (*k8sv1.Secret, error) {
	// Securely randomize a 20 char string to be used as a token
	token, err := util.GenerateVMExportToken()
	if err != nil {
		return nil, err
	}

	ownerRef := metav1.NewControllerRef(vmexport, schema.GroupVersionKind{
		Group:   exportv1.SchemeGroupVersion.Group,
		Version: exportv1.SchemeGroupVersion.Version,
		Kind:    "VirtualMachineExport",
	})
	// This requires more RBAC on certain k8s distributions and isn't really needed
	ownerRef.BlockOwnerDeletion = pointer.P(false)
	secret := &k8sv1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getExportSecretName(vmexport.Name),
			Namespace: vmexport.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*ownerRef,
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
	if deleteVme {
		return fmt.Errorf(ErrIncompatibleFlag, DELETE_FLAG, CREATE)
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
	if downloadRetries != 0 {
		return fmt.Errorf(ErrIncompatibleFlag, RETRY_FLAG, CREATE)
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
	if deleteVme {
		return fmt.Errorf(ErrIncompatibleFlag, DELETE_FLAG, DELETE)
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
	if downloadRetries != 0 {
		return fmt.Errorf(ErrIncompatibleFlag, RETRY_FLAG, DELETE)
	}
	if readinessTimeout != "" {
		return fmt.Errorf(ErrIncompatibleFlag, READINESS_TIMEOUT_FLAG, DELETE)
	}
	if len(resourceLabels) > 0 {
		return fmt.Errorf(ErrIncompatibleFlag, LABELS_FLAG, DELETE)
	}
	if len(resourceAnnotations) > 0 {
		return fmt.Errorf(ErrIncompatibleFlag, ANNOTATIONS_FLAG, DELETE)
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

	if downloadRetries < 0 {
		return fmt.Errorf(ErrInvalidValue, RETRY_FLAG, "positive integers")
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
	if !exportManifest && outputFile == "" {
		return fmt.Errorf("warning: Binary output can mess up your terminal. Use '%s -' to output into stdout anyway or consider '%s <FILE>' to save to a file", OUTPUT_FLAG, OUTPUT_FLAG)
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
	err := virtwait.PollImmediately(interval, timeout, func(ctx context.Context) (bool, error) {
		vmexport, err := getVirtualMachineExport(client, vmeInfo)
		if err != nil || vmexport == nil {
			return false, err
		}

		if vmexport.Status == nil || vmexport.Status.Phase != exportv1.Ready {
			printToOutput("waiting for VM Export %s status to be ready...\n", vmeInfo.Name)
			return false, nil
		}

		service, err = client.CoreV1().Services(vmeInfo.Namespace).Get(ctx, serviceName, metav1.GetOptions{})
		if err != nil {
			if k8serrors.IsNotFound(err) {
				printToOutput("waiting for service %s to be ready before port-forwarding...\n", serviceName)
				return false, nil
			}
			return false, err
		}
		printToOutput("service %s is ready for port-forwarding\n", service.Name)
		return true, nil
	})
	return service, err
}

// setupPortForward runs a port-forward after initializing all required arguments
func setupPortForward(client kubecli.KubevirtClient, vmeInfo *VMExportInfo) (chan struct{}, error) {
	// Wait for the vmexport object to be ready
	service, err := waitForExportServiceToBeReady(client, vmeInfo, processingWaitInterval, vmeInfo.ReadinessTimeout)
	if err != nil {
		return nil, err
	}

	// Extract the target pod selector from the service
	podSelector := labels.SelectorFromSet(service.Spec.Selector)

	// List the pods matching the selector
	podList, err := client.CoreV1().Pods(vmeInfo.Namespace).List(context.Background(), metav1.ListOptions{LabelSelector: podSelector.String()})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %v", err)
	}

	// Pick the first pod to forward the port
	if len(podList.Items) == 0 {
		return nil, fmt.Errorf("no pods found for the service %s", service.Name)
	}

	// Set up the port forwarding ports
	ports, err := translateServicePortToTargetPort(vmeInfo.LocalPort, "443", *service, podList.Items[0])
	if err != nil {
		return nil, err
	}

	stopChan := make(chan struct{}, 1)
	readyChan := make(chan struct{})
	portChan := make(chan uint16)
	go RunPortForwardFn(client, podList.Items[0], vmeInfo.Namespace, ports, stopChan, readyChan, portChan)

	// Wait for the port forwarding to be ready
	select {
	case <-readyChan:
		printToOutput("Port forwarding is ready.\n")
		// Using 0 allows listening on a random available port.
		// Now we need to find out which port was used
		if vmeInfo.LocalPort == "0" {
			localPort := <-portChan
			close(portChan)
			vmeInfo.ServiceURL = fmt.Sprintf("127.0.0.1:%d", localPort)
		}
	case <-time.After(30 * time.Second):
		return nil, fmt.Errorf("timeout waiting for port forwarding to be ready")
	}
	return stopChan, nil
}

// RunPortForward is the actual function that runs the port-forward. Meant to be run concurrently
func RunPortForward(client kubecli.KubevirtClient, pod k8sv1.Pod, namespace string, ports []string, stopChan, readyChan chan struct{}, portChan chan uint16) error {
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
	fw, err := portforward.New(dialer, ports, stopChan, readyChan, os.Stderr, os.Stderr)
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
