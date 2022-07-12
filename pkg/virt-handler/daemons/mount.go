package daemons

import (
	"fmt"
	"os"
	"path/filepath"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/daemons"
	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	"kubevirt.io/kubevirt/pkg/safepath"
	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
	"kubevirt.io/kubevirt/pkg/virt-handler/selinux"
	virtchroot "kubevirt.io/kubevirt/pkg/virt-handler/virt-chroot"
)

type Mounter interface {
	MountDaemonsSockets(vmi *v1.VirtualMachineInstance) error
	UmountDaemonsSockets(vmi *v1.VirtualMachineInstance) error
}

func MountDaemonsSockets(vmi *v1.VirtualMachineInstance) error {
	var lastError error
	safeSourceDaemonsPath, err := daemonSourcePRSocketPath()
	if err != nil {
		return fmt.Errorf("failed parsing source path:%v", err)
	}
	// Check if VMI requires persistent reservation
	if !daemons.IsPRHelperNeeded(vmi) {
		return nil
	}
	if len(vmi.Status.ActivePods) < 1 {
		return fmt.Errorf("failed bindmount daemons socket dir: no active pods for the vmi %s", vmi.Name)
	}
	for uid, _ := range vmi.Status.ActivePods {
		targetDir, err := daemonSocketDirPath(string(uid))
		if err != nil {
			lastError = wrapError(lastError, fmt.Errorf("failed creating the path for the target dir for pod uid%s:%v", string(uid), err))
			continue
		}
		if err := safepath.MkdirAtNoFollow(targetDir, daemons.PrHelperDir, 0755); err != nil {
			if !os.IsExist(err) {
				lastError = wrapError(lastError, err)
				continue
			}
		}
		socketDir, err := safepath.JoinNoFollow(targetDir, daemons.PrHelperDir)
		if err != nil {
			lastError = wrapError(lastError, fmt.Errorf("failed creating the path for the socket dir for pod uid%s:%v", string(uid), err))
			continue
		}

		mounted, err := isolation.IsMounted(socketDir)
		if err != nil {
			lastError = wrapError(lastError, fmt.Errorf("failed checking if daemon socket dir %s is mounted: %v", socketDir.String(), err))
			continue
		}
		if mounted {
			continue
		}

		out, err := virtchroot.MountChroot(safeSourceDaemonsPath, socketDir, false).CombinedOutput()
		if err != nil {
			lastError = wrapError(lastError, fmt.Errorf("failed bindmount daemons socket dir: %v: %v", string(out), err))
		}
		// Change ownership to the directory and relabel
		err = changeOwnershipAndRelabel(socketDir)
		if err != nil {
			lastError = wrapError(lastError, fmt.Errorf("failed relabeling pr socket dir: %s: %v", socketDir.String(), err))
			continue
		}
		// Change ownership to the socket and relabel
		socket, err := safepath.JoinNoFollow(socketDir, daemons.PrHelperSocket)
		if err != nil {
			lastError = wrapError(lastError, fmt.Errorf("failed creating socket path: %v", err))
			continue
		}

		err = changeOwnershipAndRelabel(socket)
		if err != nil {
			lastError = wrapError(lastError, fmt.Errorf("failed relabeling socket: %s: %v", socket.String(), err))
			lastError = wrapError(lastError, err)
			continue
		}
		log.Log.V(1).Infof("mounted daemon socket: %s", socket.String())
	}
	return lastError
}

func UmountDaemonsSocket(vmi *v1.VirtualMachineInstance) error {
	var lastError error
	// Check if VMI requires persistent reservation
	if !daemons.IsPRHelperNeeded(vmi) {
		return nil
	}
	for uid, _ := range vmi.Status.ActivePods {
		socketDir, err := daemonPRSocketDirPath(string(uid))
		if err != nil {
			lastError = wrapError(lastError, fmt.Errorf("failed creating the path for the socket dir for pod uid%s:%v", string(uid), err))
			continue
		}
		mounted, err := isolation.IsMounted(socketDir)
		if err != nil {
			lastError = wrapError(lastError, err)
			continue
		}
		if mounted {
			continue
		}
		out, err := virtchroot.UmountChroot(socketDir).CombinedOutput()
		if err != nil {
			lastError = wrapError(lastError, fmt.Errorf("failed unmount daemons socket dir: %v: %v", string(out), err))
		}
		if err := safepath.UnlinkAtNoFollow(socketDir); err != nil {
			lastError = wrapError(lastError, err)
		}
	}
	return lastError
}

func daemonSocketDirPath(podUID string) (*safepath.Path, error) {
	path, err := safepath.NewPathNoFollow(filepath.Join(util.KubeletPodsDir, podUID, daemons.SuffixDaemonPath))
	if err != nil {
		return nil, err
	}
	return path, nil
}

func daemonPRSocketDirPath(podUID string) (*safepath.Path, error) {
	path, err := safepath.NewPathNoFollow(filepath.Join(util.KubeletPodsDir, podUID, daemons.SuffixDaemonPath, daemons.PrHelperDir))
	if err != nil {
		return nil, err
	}
	return path, nil
}

func daemonSourcePRSocketPath() (*safepath.Path, error) {
	path, err := safepath.NewPathNoFollow(daemons.GetPrHelperSocketDir())
	if err != nil {
		return nil, err
	}
	return path, nil
}

func wrapError(lastError, err error) error {
	if lastError == nil {
		return err
	}
	return fmt.Errorf("%w, %s", lastError, err.Error())
}

func changeOwnershipAndRelabel(path *safepath.Path) error {
	err := diskutils.DefaultOwnershipManager.SetFileOwnership(path)
	if err != nil {
		return err
	}

	seLinux, selinuxEnabled, err := selinux.NewSELinux()
	if err == nil && selinuxEnabled {
		unprivilegedContainerSELinuxLabel := "system_u:object_r:container_file_t:s0"
		err = selinux.RelabelFiles(unprivilegedContainerSELinuxLabel, seLinux.IsPermissive(), path)
		if err != nil {
			return (fmt.Errorf("error relabeling %s: %v", path, err))
		}

	}
	return err
}
