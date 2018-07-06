package common

import (
	"time"

	"k8s.io/api/core/v1"
)

// Common types and constants used by the importer and controller.
// TODO: maybe the vm cloner can use these common values

const (
	CDI_VERSION = "v1.0.0"

	IMPORTER_DEFAULT_IMAGE = "docker.io/kubevirt/cdi-importer:" + CDI_VERSION
	CDI_LABEL_KEY          = "app"
	CDI_LABEL_VALUE        = "containerized-data-importer"
	CDI_LABEL_SELECTOR     = CDI_LABEL_KEY + "=" + CDI_LABEL_VALUE

	// host file constants:
	IMPORTER_WRITE_DIR  = "/data"
	IMPORTER_WRITE_FILE = "disk.img"
	IMPORTER_WRITE_PATH = IMPORTER_WRITE_DIR + "/" + IMPORTER_WRITE_FILE
	// importer container constants:
	IMPORTER_PODNAME             = "importer"
	IMPORTER_DATA_DIR            = "/data"
	IMPORTER_S3_HOST             = "s3.amazonaws.com"
	IMPORTER_DEFAULT_PULL_POLICY = string(v1.PullIfNotPresent)
	// env var names
	IMPORTER_PULL_POLICY   = "IMPORTER_PULL_POLICY"
	IMPORTER_ENDPOINT      = "IMPORTER_ENDPOINT"
	IMPORTER_ACCESS_KEY_ID = "IMPORTER_ACCESS_KEY_ID"
	IMPORTER_SECRET_KEY    = "IMPORTER_SECRET_KEY"
	// key names expected in credential secret
	KeyAccess = "accessKeyId"
	KeySecret = "secretKey"

	// Shared informer resync period.
	DEFAULT_RESYNC_PERIOD = 10 * time.Minute

	// logging verbosity
	Vuser                    = 1
	Vadmin                   = 2
	Vdebug                   = 3
	IMPORTER_DEFAULT_VERBOSE = Vuser
)
