package virtiofs

import (
	"path/filepath"

	"kubevirt.io/kubevirt/pkg/util"
)

// This is empty dir
var VirtioFSContainers = "virtiofs-containers"
var VirtioFSContainersMountBaseDir = filepath.Join(util.VirtShareDir, VirtioFSContainers)
