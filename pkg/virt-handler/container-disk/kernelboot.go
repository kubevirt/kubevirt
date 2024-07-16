package container_disk

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	containerdisk "kubevirt.io/kubevirt/pkg/container-disk"
	"kubevirt.io/kubevirt/pkg/safepath"
	"kubevirt.io/kubevirt/pkg/unsafepath"
	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
	virt_chroot "kubevirt.io/kubevirt/pkg/virt-handler/virt-chroot"
)

type KernelBootChecksum struct {
	Initrd *uint32
	Kernel *uint32
}

type kernelArtifacts struct {
	kernel *safepath.Path
	initrd *safepath.Path
}

// MountKernelArtifacts mounts artifacts defined by KernelBootName in VMI.
// This function is assumed to run after MountAndVerify.
func (m *mounter) mountKernelArtifacts(vmi *v1.VirtualMachineInstance, verify bool) error {
	const kernelBootName = containerdisk.KernelBootName

	log.Log.Object(vmi).Infof("mounting kernel artifacts")

	if !util.HasKernelBootContainerImage(vmi) {
		log.Log.Object(vmi).Infof("kernel boot not defined - nothing to mount")
		return nil
	}

	kb := vmi.Spec.Domain.Firmware.KernelBoot.Container

	targetDir, err := containerdisk.GetDiskTargetDirFromHostView(vmi)
	if err != nil {
		return fmt.Errorf("failed to get disk target dir: %v", err)
	}
	if err := safepath.MkdirAtNoFollow(targetDir, containerdisk.KernelBootName, 0755); err != nil {
		if !os.IsExist(err) {
			return err
		}
	}

	targetDir, err = safepath.JoinNoFollow(targetDir, containerdisk.KernelBootName)
	if err != nil {
		return err
	}
	if err := safepath.ChpermAtNoFollow(targetDir, 0, 0, 0755); err != nil {
		return err
	}

	socketFilePath, err := m.kernelBootSocketPathGetter(vmi)
	if err != nil {
		return fmt.Errorf("failed to find socket path for kernel artifacts: %v", err)
	}

	record := vmiMountTargetRecord{
		MountTargetEntries: []vmiMountTargetEntry{{
			TargetFile: unsafepath.UnsafeAbsolute(targetDir.Raw()),
			SocketFile: socketFilePath,
		}},
	}

	err = m.addMountTargetRecord(vmi, &record)
	if err != nil {
		return err
	}

	var targetInitrdPath *safepath.Path
	var targetKernelPath *safepath.Path

	if kb.InitrdPath != "" {
		if err := safepath.TouchAtNoFollow(targetDir, filepath.Base(kb.InitrdPath), 0655); err != nil && !os.IsExist(err) {
			return err
		}

		targetInitrdPath, err = safepath.JoinNoFollow(targetDir, filepath.Base(kb.InitrdPath))
		if err != nil {
			return err
		}
	}

	if kb.KernelPath != "" {
		if err := safepath.TouchAtNoFollow(targetDir, filepath.Base(kb.KernelPath), 0655); err != nil && !os.IsExist(err) {
			return err
		}

		targetKernelPath, err = safepath.JoinNoFollow(targetDir, filepath.Base(kb.KernelPath))
		if err != nil {
			return err
		}
	}

	areKernelArtifactsMounted := func(artifactsDir *safepath.Path, artifactFiles ...*safepath.Path) (bool, error) {
		if _, err = safepath.StatAtNoFollow(artifactsDir); errors.Is(err, os.ErrNotExist) {
			return false, nil
		} else if err != nil {
			return false, err
		}

		for _, mountPoint := range artifactFiles {
			if mountPoint != nil {
				isMounted, err := isolation.IsMounted(mountPoint)
				if !isMounted || err != nil {
					return isMounted, err
				}
			}
		}
		return true, nil
	}

	if isMounted, err := areKernelArtifactsMounted(targetDir, targetInitrdPath, targetKernelPath); err != nil {
		return fmt.Errorf("failed to determine if %s is already mounted: %v", targetDir, err)
	} else if !isMounted {
		log.Log.Object(vmi).Infof("kernel artifacts are not mounted - mounting...")

		kernelArtifacts, err := m.getKernelArtifactPaths(vmi)
		if err != nil {
			return err
		}

		if kernelArtifacts.kernel != nil {
			out, err := virt_chroot.MountChroot(kernelArtifacts.kernel, targetKernelPath, true).CombinedOutput()
			if err != nil {
				return fmt.Errorf("failed to bindmount %v: %v : %v", kernelBootName, string(out), err)
			}
		}

		if kernelArtifacts.initrd != nil {
			out, err := virt_chroot.MountChroot(kernelArtifacts.initrd, targetInitrdPath, true).CombinedOutput()
			if err != nil {
				return fmt.Errorf("failed to bindmount %v: %v : %v", kernelBootName, string(out), err)
			}
		}

	}

	if verify {
		mounted, err := areKernelArtifactsMounted(targetDir, targetInitrdPath, targetKernelPath)
		if err != nil {
			return fmt.Errorf("failed to check if kernel artifacts are mounted. error: %v", err)
		} else if !mounted {
			return fmt.Errorf("kernel artifacts verification failed")
		}
	}

	return nil
}
func (m *mounter) unmountKernelArtifacts(vmi *v1.VirtualMachineInstance) error {
	if !util.HasKernelBootContainerImage(vmi) {
		return nil
	}

	log.DefaultLogger().Object(vmi).Infof("unmounting kernel artifacts")

	kb := vmi.Spec.Domain.Firmware.KernelBoot.Container

	record, err := m.getMountTargetRecord(vmi)
	if err != nil {
		return fmt.Errorf("failed to get mount target record: %v", err)
	} else if record == nil {
		log.DefaultLogger().Object(vmi).Warning("Cannot find kernel-boot entries to unmount")
		return nil
	}

	unmount := func(targetDir *safepath.Path, artifactPaths ...string) error {
		for _, artifactPath := range artifactPaths {
			if artifactPath == "" {
				continue
			}

			targetPath, err := safepath.JoinNoFollow(targetDir, filepath.Base(artifactPath))
			if err != nil {
				return fmt.Errorf(failedCheckMountPointFmt, targetPath, err)
			}
			if mounted, err := isolation.IsMounted(targetPath); err != nil {
				return fmt.Errorf(failedCheckMountPointFmt, targetPath, err)
			} else if mounted {
				log.DefaultLogger().Object(vmi).Infof("unmounting container disk at targetDir %s", targetPath)

				out, err := virt_chroot.UmountChroot(targetPath).CombinedOutput()
				if err != nil {
					return fmt.Errorf(failedUnmountFmt, targetPath, string(out), err)
				}
			}
		}
		return nil
	}

	for idx, entry := range record.MountTargetEntries {
		if !strings.Contains(entry.TargetFile, containerdisk.KernelBootName) {
			continue
		}
		targetDir, err := safepath.NewFileNoFollow(entry.TargetFile)
		if err != nil {
			return fmt.Errorf("failed to obtaining a reference to the target directory %q: %v", targetDir, err)
		}
		_ = targetDir.Close()
		log.DefaultLogger().Object(vmi).Infof("unmounting kernel artifacts in path: %v", targetDir)

		if err = unmount(targetDir.Path(), kb.InitrdPath, kb.KernelPath); err != nil {
			// Not returning here since even if unmount wasn't successful it's better to keep
			// cleaning the mounted files.
			log.Log.Object(vmi).Reason(err).Error("unable to unmount kernel artifacts")
		}

		removeSliceElement := func(s []vmiMountTargetEntry, idxToRemove int) []vmiMountTargetEntry {
			// removes slice element efficiently
			s[idxToRemove] = s[len(s)-1]
			return s[:len(s)-1]
		}

		record.MountTargetEntries = removeSliceElement(record.MountTargetEntries, idx)
		return nil
	}

	return fmt.Errorf("kernel artifacts record wasn't found")
}

func (m *mounter) getKernelArtifactPaths(vmi *v1.VirtualMachineInstance) (*kernelArtifacts, error) {
	sock, err := m.kernelBootSocketPathGetter(vmi)
	if err != nil {
		return nil, ErrDiskContainerGone
	}

	res, err := m.podIsolationDetector.DetectForSocket(vmi, sock)
	if err != nil {
		return nil, fmt.Errorf("failed to detect socket for kernelboot container: %v", err)
	}

	mountPoint, err := isolation.ParentPathForRootMount(m.nodeIsolationResult, res)
	if err != nil {
		return nil, fmt.Errorf("failed to detect root mount point of kernel/initrd container on the node: %v", err)
	}

	kernelContainer := vmi.Spec.Domain.Firmware.KernelBoot.Container
	kernelArtifacts := &kernelArtifacts{}

	if kernelContainer.KernelPath != "" {
		kernelPath, err := containerdisk.GetImage(mountPoint, kernelContainer.KernelPath)
		if err != nil {
			return nil, err
		}
		kernelArtifacts.kernel = kernelPath
	}
	if kernelContainer.InitrdPath != "" {
		initrdPath, err := containerdisk.GetImage(mountPoint, kernelContainer.InitrdPath)
		if err != nil {
			return nil, err
		}
		kernelArtifacts.initrd = initrdPath
	}

	return kernelArtifacts, nil
}
