package common

import (
	"time"

	"k8s.io/api/core/v1"
)

// Common types and constants used by the importer and controller.
// TODO: maybe the vm cloner can use these common values

const (
	// CDILabelKey provides a constant for CDI PVC labels
	CDILabelKey = "app"
	// CDILabelValue provides a constant  for CDI PVC label values
	CDILabelValue = "containerized-data-importer"
	// CDILabelSelector provides a constant to use for the selector to identify CDI objects in list
	CDILabelSelector = CDILabelKey + "=" + CDILabelValue

	// CDIComponentLabel can be added to all CDI resources
	CDIComponentLabel = "cdi.kubevirt.io"

	// PrometheusLabel provides the label to indicate prometheus metrics are available in the pods.
	PrometheusLabel = "prometheus.kubevirt.io"

	// ImporterVolumePath provides a constant for the directory where the PV is mounted.
	ImporterVolumePath = "/data"
	// DiskImageName provides a constant for our importer/datastream_ginkgo_test and to build ImporterWritePath
	DiskImageName = "disk.img"
	// ImporterWritePath provides a constant for the cmd/cdi-importer/importer.go executable
	ImporterWritePath = ImporterVolumePath + "/" + DiskImageName

	// ImporterPodName provides a constant to use as a prefix for Pods created by CDI (controller only)
	ImporterPodName = "importer"
	// ImporterDataDir provides a constant for the controller pkg to use as a hardcoded path to where content is transferred to/from (controller only)
	ImporterDataDir = "/data"
	// ImporterS3Host provides an S3 string used by importer/dataStream.go only
	ImporterS3Host = "s3.amazonaws.com"
	// DefaultPullPolicy imports k8s "IfNotPresent" string for the import_controller_gingko_test and the cdi-controller executable
	DefaultPullPolicy = string(v1.PullIfNotPresent)

	// PullPolicy provides a constant to capture our env variable "PULL_POLICY" (only used by cmd/cdi-controller/controller.go)
	PullPolicy = "PULL_POLICY"
	// ImporterSource provides a constant to capture our env variable "IMPORTER_SOURCE"
	ImporterSource = "IMPORTER_SOURCE"
	// ImporterContentType provides a constant to capture our env variable "IMPORTER_CONTENTTYPE"
	ImporterContentType = "IMPORTER_CONTENTTYPE"
	// ImporterEndpoint provides a constant to capture our env variable "IMPORTER_ENDPOINT"
	ImporterEndpoint = "IMPORTER_ENDPOINT"
	// ImporterAccessKeyID provides a constant to capture our env variable "IMPORTER_ACCES_KEY_ID"
	ImporterAccessKeyID = "IMPORTER_ACCESS_KEY_ID"
	// ImporterSecretKey provides a constant to capture our env variable "IMPORTER_SECRET_KEY"
	ImporterSecretKey = "IMPORTER_SECRET_KEY"
	// ImporterImageSize provides a constant to capture our env variable "IMPORTER_IMAGE_SIZE"
	ImporterImageSize = "IMPORTER_IMAGE_SIZE"

	// CloningLabelKey provides a constant to use as a label name for pod affinity (controller pkg only)
	CloningLabelKey = "cloning"
	// CloningLabelValue provides a constant to use as a label value for pod affinity (controller pkg only)
	CloningLabelValue = "host-assisted-cloning"
	// CloningTopologyKey  (controller pkg only)
	CloningTopologyKey = "kubernetes.io/hostname"
	// ClonerSourcePodName (controller pkg only)
	ClonerSourcePodName = "clone-source-pod"
	// ClonerTargetPodName (controller pkg only)
	ClonerTargetPodName = "clone-target-pod"
	// ClonerImagePath (controller pkg only)
	ClonerImagePath = "/tmp/clone/image"
	// ClonerSocketPath (controller pkg only)
	ClonerSocketPath = "/tmp/clone/socket"

	// UploadServerCDILabel is the label applied to upload server resources
	UploadServerCDILabel = "cdi-upload-server"

	// UploadServerPodname is name of the upload server pod container
	UploadServerPodname = UploadServerCDILabel

	// UploadServerDataDir is the destination directoryfor uploads
	UploadServerDataDir = ImporterDataDir
	// UploadServerServiceLabel is the label selector for upload server services
	UploadServerServiceLabel = "service"

	// OwnerUID provides the UID of the owner entity (either PVC or DV)
	OwnerUID = "OWNER_UID"

	// KeyAccess provides a constant to the accessKeyId label using in controller pkg and transport_test.go
	KeyAccess = "accessKeyId"
	// KeySecret provides a constant to the secretKey label using in controller pkg and transport_test.go
	KeySecret = "secretKey"

	// DefaultResyncPeriod sets a 10 minute resync period, used in the controller pkg and the controller cmd executable
	DefaultResyncPeriod = 10 * time.Minute
)
