package imageupload

import (
	"fmt"
	"net/url"
	"os"
	"reflect"

	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/virtctl/clientconfig"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

const (
	VirtualMachineInstancetype        = "VirtualMachineInstancetype"
	VirtualMachineClusterInstancetype = "VirtualMachineClusterInstancetype"
	VirtualMachinePreference          = "VirtualMachinePreference"
	VirtualMachineClusterPreference   = "VirtualMachineClusterPreference"
)

// NewImageUploadCommand returns a cobra.Command for handling the uploading of VM images
func NewImageUploadCommand() *cobra.Command {
	c := command{}
	cmd := &cobra.Command{
		Use:     "image-upload",
		Short:   "Upload a VM image to a DataVolume/PersistentVolumeClaim.",
		Example: usage(),
		Args:    cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, namespace, _, err := clientconfig.ClientAndNamespaceFromContext(cmd.Context())
			if err != nil {
				return err
			}

			c.cmd = cmd
			c.client = client
			c.namespace = namespace
			return c.run(args)
		},
	}
	// Connection and proxy settings
	cmd.Flags().BoolVar(&c.insecure, "insecure", false, "Allow insecure server connections when using HTTPS.")
	cmd.Flags().StringVar(&c.uploadProxyURL, "uploadproxy-url", "", "The URL of the cdi-upload proxy service.")

	// Resource creation settings
	cmd.Flags().StringVar(&c.name, "pvc-name", "", "The destination DataVolume/PVC name.")
	cmd.Flags().StringVar(&c.size, "size", "", "The size of the DataVolume/PVC to create (ex. 10Gi, 500Mi).")
	cmd.Flags().StringVar(&c.storageClass, "storage-class", "", "The storage class for the PVC.")
	cmd.Flags().StringVar(&c.accessMode, "access-mode", "", "The access mode for the PVC (e.g., ReadWriteOnce).")
	cmd.Flags().StringVar(&c.volumeMode, "volume-mode", "", "Specify the VolumeMode (block/filesystem) used to create the PVC.")
	cmd.Flags().BoolVar(&c.noCreate, "no-create", false, "Don't attempt to create a new DataVolume/PVC.")
	cmd.Flags().BoolVar(&c.forceBind, "force-bind", false, "Force bind the PVC, ignoring the WaitForFirstConsumer logic.")

	// Image/archive settings
	cmd.Flags().StringVar(&c.imagePath, "image-path", "", "Path to the local VM image.")
	cmd.Flags().StringVar(&c.archivePath, "archive-path", "", "Path to the local archive.")

	// Upload settings
	cmd.Flags().UintVar(&c.uploadPodWaitSecs, "wait-secs", 300, "Seconds to wait for upload pod to start.")
	cmd.Flags().UintVar(&c.uploadRetries, "retry", 5, "When upload server returns a transient error, we retry this number of times before giving up.")

	// DataSource settings
	cmd.Flags().BoolVar(&c.dataSource, "datasource", false, "Create a DataSource pointing to the created DataVolume/PVC.")

	// Default instance type and preference settings
	cmd.Flags().StringVar(&c.defaultInstancetype, "default-instancetype", "", "The default instance type to associate with the image.")
	cmd.Flags().StringVar(&c.defaultInstancetypeKind, "default-instancetype-kind", VirtualMachineClusterInstancetype, fmt.Sprintf("The default instance type kind to associate with the image. Allowed values: %v", []string{VirtualMachineInstancetype, VirtualMachineClusterInstancetype}))
	cmd.Flags().StringVar(&c.defaultPreference, "default-preference", "", "The default preference to associate with the image.")
	cmd.Flags().StringVar(&c.defaultPreferenceKind, "default-preference-kind", VirtualMachineClusterPreference, fmt.Sprintf("The default preference kind to associate with the image. Allowed values: %v", []string{VirtualMachinePreference, VirtualMachineClusterPreference}))
	// Mark mutually exclusive flags
	cmd.MarkFlagsMutuallyExclusive("image-path", "archive-path")
	cmd.MarkFlagsMutuallyExclusive("pvc-name", "size")

	// Set usage template
	cmd.SetUsageTemplate(templates.UsageTemplate())

	return cmd
}

func usage() string {
	usage := `  # Upload a local disk image to a newly created DataVolume:
  {{ProgramName}} image-upload dv fedora-dv --size=10Gi --image-path=/images/fedora30.qcow2

  # Upload a local disk image to an existing DataVolume
  {{ProgramName}} image-upload dv fedora-dv --no-create --image-path=/images/fedora30.qcow2

  # Upload a local disk image to a newly created PersistentVolumeClaim
  {{ProgramName}} image-upload pvc fedora-pvc --size=10Gi --image-path=/images/fedora30.qcow2

  # Upload a local disk image to a newly created PersistentVolumeClaim and label it with a default instance type and preference
  {{ProgramName}} image-upload pvc fedora-pvc --size=10Gi --image-path=/images/fedora30.qcow2 --default-instancetype=n1.medium --default-preference=fedora

  # Upload a local disk image to an existing PersistentVolumeClaim
  {{ProgramName}} image-upload pvc fedora-pvc --no-create --image-path=/images/fedora30.qcow2

  # Upload to a DataVolume with explicit URL to CDI Upload Proxy
  {{ProgramName}} image-upload dv fedora-dv --uploadproxy-url=https://cdi-uploadproxy.mycluster.com --image-path=/images/fedora30.qcow2

  # Upload a local disk archive to a newly created DataVolume:
  {{ProgramName}} image-upload dv fedora-dv --size=10Gi --archive-path=/images/fedora30.tar`
	return usage
}

func (c *command) run(args []string) error {
	if err := c.parseArgs(args); err != nil {
		return err
	}

	if err := c.validateDefaultInstancetypeArgs(); err != nil {
		return err
	}

	file, err := c.openImageFile()
	if err != nil {
		return err
	}
	defer util.CloseIOAndCheckErr(file, nil)

	pvc, err := c.getOrCreateUploadResource()
	if err != nil {
		return err
	}

	if err := c.waitForUploadReadiness(pvc); err != nil {
		return err
	}

	if err := c.setUploadProxyURL(); err != nil {
		return err
	}

	token, err := c.getUploadToken()
	if err != nil {
		return err
	}

	if err := c.uploadData(token, file); err != nil {
		return err
	}

	if c.dataSource {
		if err := c.handleDataSource(); err != nil {
			return err
		}
	}

	return c.waitForProcessingCompletion()
}

// Opens the image file for reading.
func (c *command) openImageFile() (*os.File, error) {
	// #nosec G304 No risk for path injection as this function executes with
	// the same privileges as those of virtctl user who supplies imagePath
	file, err := os.Open(c.imagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open image file: %w", err)
	}
	return file, nil
}

// Gets or creates the PVC or DataVolume for the upload.
func (c *command) getOrCreateUploadResource() (*v1.PersistentVolumeClaim, error) {
	pvc, err := c.getAndValidateUploadPVC()
	if err != nil {
		if !(k8serrors.IsNotFound(err) && !c.noCreate) {
			return nil, err
		}

		var obj metav1.Object
		if c.createPVC {
			obj, err = c.createUploadPVC()
		} else {
			obj, err = c.createUploadDataVolume()
		}
		if err != nil {
			return nil, err
		}

		c.cmd.Printf("%s %s/%s created\n", reflect.TypeOf(obj).Elem().Name(), obj.GetNamespace(), obj.GetName())
		return nil, nil
	}

	pvc, err = c.ensurePVCSupportsUpload(pvc)
	if err != nil {
		return nil, err
	}

	c.cmd.Printf("Using existing PVC %s/%s\n", c.namespace, pvc.Name)
	return pvc, nil
}

// waitForUploadReadiness
func (c *command) waitForUploadReadiness(pvc *v1.PersistentVolumeClaim) error {
	if c.createPVC {
		return c.waitUploadServerReady()
	}
	return c.waitDvUploadScheduled()
}

// Sets the upload proxy URL, ensuring it has a valid scheme
func (c *command) setUploadProxyURL() error {
	if c.uploadProxyURL == "" {
		var err error
		c.uploadProxyURL, err = c.getUploadProxyURL()
		if err != nil {
			return err
		}
		if c.uploadProxyURL == "" {
			return fmt.Errorf("uploadproxy URL not found")
		}
	}

	u, err := url.Parse(c.uploadProxyURL)
	if err != nil {
		return err
	}

	if u.Scheme == "" {
		c.uploadProxyURL = fmt.Sprintf("https://%s", c.uploadProxyURL)
	}

	c.cmd.Printf("Uploading data to %s\n", c.uploadProxyURL)
	return nil
}

// Waits for post-upload processing to complete
func (c *command) waitForProcessingCompletion() error {
	c.cmd.Println("Uploading data completed successfully, waiting for processing to complete, you can hit ctrl-c without interrupting the progress")
	err := UploadProcessingCompleteFunc(c.client, c.cmd, c.namespace, c.name, processingWaitInterval, processingWaitTotal)
	if err != nil {
		c.cmd.Printf("Timed out waiting for post upload processing to complete, please check upload pod status for progress\n")
	} else {
		c.cmd.Printf("Uploading %s completed successfully\n", c.imagePath)
	}
	return err
}

func validateKind(kind string, allowedValues []string) error {
	for _, allowed := range allowedValues {
		if kind == allowed {
			return nil
		}
	}
	return fmt.Errorf("invalid kind: %s. Allowed values are: %v", kind, allowedValues)
}

func (c *command) validateDefaultInstancetypeArgs() error {
	// Validate defaultInstancetypeKind
	allowedInstanceTypeKinds := []string{VirtualMachineInstancetype, VirtualMachineClusterInstancetype}
	if c.defaultInstancetypeKind != "" {
		if err := validateKind(c.defaultInstancetypeKind, allowedInstanceTypeKinds); err != nil {
			return fmt.Errorf("invalid default-instancetype-kind: %w", err)
		}
	}

	// Validate defaultPreferenceKind
	allowedPreferenceKinds := []string{VirtualMachinePreference, VirtualMachineClusterPreference}
	if c.defaultPreferenceKind != "" {
		if err := validateKind(c.defaultPreferenceKind, allowedPreferenceKinds); err != nil {
			return fmt.Errorf("invalid default-preference-kind: %w", err)
		}
	}

	// Ensure defaultInstancetype is provided if defaultInstancetypeKind is set
	if c.defaultInstancetype == "" && c.defaultInstancetypeKind != "" {
		return fmt.Errorf("--default-instancetype must be provided with --default-instancetype-kind")
	}

	// Ensure defaultPreference is provided if defaultPreferenceKind is set
	if c.defaultPreference == "" && c.defaultPreferenceKind != "" {
		return fmt.Errorf("--default-preference must be provided with --default-preference-kind")
	}

	// Ensure defaultInstancetype and defaultPreference are not used with --no-create
	if (c.defaultInstancetype != "" || c.defaultPreference != "") && c.noCreate {
		return fmt.Errorf("--default-instancetype and --default-preference cannot be used with --no-create")
	}

	return nil
}
