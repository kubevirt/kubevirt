package virtiofs

import (
	"fmt"
	"path/filepath"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/util"
)

// This is empty dir
var VirtioFSContainers = "virtiofs-containers"
var VirtioFSContainersMountBaseDir = filepath.Join(util.VirtShareDir, VirtioFSContainers)

func VirtioFSSocketPath(volumeName string) string {
	socketName := fmt.Sprintf("%s.sock", volumeName)
	return filepath.Join(VirtioFSContainersMountBaseDir, socketName)
}

// RequiresRootPrivileges Returns true if the volume requires the virtiofs
// container to run as user root
func RequiresRootPrivileges(volume *v1.Volume) bool {
	// only config volumes can be shared with an unprivileged container
	return volume.ConfigMap == nil && volume.Secret == nil &&
		volume.ServiceAccount == nil && volume.DownwardAPI == nil
}
