package container_disk

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"kubevirt.io/kubevirt/pkg/unsafepath"

	"kubevirt.io/kubevirt/pkg/safepath"
	"kubevirt.io/kubevirt/pkg/util"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	virt_chroot "kubevirt.io/kubevirt/pkg/virt-handler/virt-chroot"

	"kubevirt.io/client-go/log"

	containerdisk "kubevirt.io/kubevirt/pkg/container-disk"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
	"kubevirt.io/kubevirt/pkg/virt-handler/mount-manager/recorder"

	v1 "kubevirt.io/api/core/v1"
)

const (
	failedCheckMountPointFmt = "failed to check mount point for containerDisk %v: %v"
	failedUnmountFmt         = "failed to unmount containerDisk %v: %v : %v"
)

//go:generate mockgen -source $GOFILE -package=$GOPACKAGE -destination=generated_mock_$GOFILE

type mounter struct {
	podIsolationDetector       isolation.PodIsolationDetector
	mountRecorder              recorder.MountRecorder
	suppressWarningTimeout     time.Duration
	socketPathGetter           containerdisk.SocketPathGetter
	kernelBootSocketPathGetter containerdisk.KernelBootSocketPathGetter
	clusterConfig              *virtconfig.ClusterConfig
}

type Mounter interface {
	ContainerDisksReady(vmi *v1.VirtualMachineInstance, notInitializedSince time.Time) (bool, error)
	MountAndVerify(vmi *v1.VirtualMachineInstance) (map[string]*containerdisk.DiskInfo, error)
	Unmount(vmi *v1.VirtualMachineInstance) error
}

func NewMounter(isoDetector isolation.PodIsolationDetector, clusterConfig *virtconfig.ClusterConfig, mountRecorder recorder.MountRecorder) Mounter {
	return &mounter{
		podIsolationDetector:       isoDetector,
		mountRecorder:              mountRecorder,
		suppressWarningTimeout:     1 * time.Minute,
		socketPathGetter:           containerdisk.NewSocketPathGetter(""),
		kernelBootSocketPathGetter: containerdisk.NewKernelBootSocketPathGetter(""),
		clusterConfig:              clusterConfig,
	}
}

// Mount takes a vmi and mounts all container disks of the VMI, so that they are visible for the qemu process.
// Additionally qcow2 images are validated if "verify" is true. The validation happens with rlimits set, to avoid DOS.
func (m *mounter) MountAndVerify(vmi *v1.VirtualMachineInstance) (map[string]*containerdisk.DiskInfo, error) {
	record := []recorder.MountTargetEntry{}
	disksInfo := map[string]*containerdisk.DiskInfo{}

	for i, volume := range vmi.Spec.Volumes {
		if volume.ContainerDisk != nil {
			diskTargetDir, err := containerdisk.GetDiskTargetDirFromHostView(vmi)
			if err != nil {
				return nil, err
			}
			diskName := containerdisk.GetDiskTargetName(i)
			// If diskName is a symlink it will fail if the target exists.
			if err := safepath.TouchAtNoFollow(diskTargetDir, diskName, os.ModePerm); err != nil {
				if err != nil && !os.IsExist(err) {
					return nil, fmt.Errorf("failed to create mount point target: %v", err)
				}
			}
			targetFile, err := safepath.JoinNoFollow(diskTargetDir, diskName)
			if err != nil {
				return nil, err
			}

			sock, err := m.socketPathGetter(vmi, i)
			if err != nil {
				return nil, err
			}

			record = append(record, recorder.MountTargetEntry{
				TargetFile: unsafepath.UnsafeAbsolute(targetFile.Raw()),
				SocketFile: sock,
			})
		}
	}

	if len(record) > 0 {
		err := m.mountRecorder.SetMountRecord(vmi, record)
		if err != nil {
			return nil, err
		}
	}

	vmiRes, err := m.podIsolationDetector.Detect(vmi)
	if err != nil {
		return nil, fmt.Errorf("failed to detect VMI pod: %v", err)
	}

	for i, volume := range vmi.Spec.Volumes {
		if volume.ContainerDisk != nil {
			diskTargetDir, err := containerdisk.GetDiskTargetDirFromHostView(vmi)
			if err != nil {
				return nil, err
			}
			diskName := containerdisk.GetDiskTargetName(i)
			targetFile, err := safepath.JoinNoFollow(diskTargetDir, diskName)
			if err != nil {
				return nil, err
			}

			nodeRes := isolation.NodeIsolationResult()

			if isMounted, err := isolation.IsMounted(targetFile); err != nil {
				return nil, fmt.Errorf("failed to determine if %s is already mounted: %v", targetFile, err)
			} else if !isMounted {
				sock, err := m.socketPathGetter(vmi, i)
				if err != nil {
					return nil, err
				}

				res, err := m.podIsolationDetector.DetectForSocket(vmi, sock)
				if err != nil {
					return nil, fmt.Errorf("failed to detect socket for containerDisk %v: %v", volume.Name, err)
				}
				mountPoint, err := isolation.ParentPathForRootMount(nodeRes, res)
				if err != nil {
					return nil, fmt.Errorf("failed to detect root mount point of containerDisk %v on the node: %v", volume.Name, err)
				}
				sourceFile, err := containerdisk.GetImage(mountPoint, volume.ContainerDisk.Path)
				if err != nil {
					return nil, fmt.Errorf("failed to find a sourceFile in containerDisk %v: %v", volume.Name, err)
				}

				log.DefaultLogger().Object(vmi).Infof("Bind mounting container disk at %s to %s", sourceFile, targetFile)
				out, err := virt_chroot.MountChroot(sourceFile, targetFile, true).CombinedOutput()
				if err != nil {
					return nil, fmt.Errorf("failed to bindmount containerDisk %v: %v : %v", volume.Name, string(out), err)
				}
			}

			imageInfo, err := isolation.GetImageInfo(containerdisk.GetDiskTargetPathFromLauncherView(i), vmiRes, m.clusterConfig.GetDiskVerification())
			if err != nil {
				return nil, fmt.Errorf("failed to get image info: %v", err)
			}
			if err := containerdisk.VerifyImage(imageInfo); err != nil {
				return nil, fmt.Errorf("invalid image in containerDisk %v: %v", volume.Name, err)
			}
			disksInfo[volume.Name] = imageInfo
		}
	}
	err = m.mountKernelArtifacts(vmi, true)
	if err != nil {
		return nil, fmt.Errorf("error mounting kernel artifacts: %v", err)
	}

	return disksInfo, nil
}

// Unmount unmounts all container disks of a given VMI.
func (m *mounter) Unmount(vmi *v1.VirtualMachineInstance) error {
	if vmi.UID == "" {
		return nil
	}

	err := m.unmountKernelArtifacts(vmi)
	if err != nil {
		return fmt.Errorf("error unmounting kernel artifacts: %v", err)
	}

	record, err := m.mountRecorder.GetMountRecord(vmi)
	if err != nil {
		return err
	}
	if len(record) < 1 {
		// no entries to unmount

		log.DefaultLogger().Object(vmi).Infof("No container disk mount entries found to unmount")
		return nil
	}

	log.DefaultLogger().Object(vmi).Infof("Found container disk mount entries")
	for _, entry := range record {
		log.DefaultLogger().Object(vmi).Infof("Looking to see if containerdisk is mounted at path %s", entry.TargetFile)
		file, err := safepath.NewFileNoFollow(entry.TargetFile)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return fmt.Errorf(failedCheckMountPointFmt, entry.TargetFile, err)
		}
		_ = file.Close()
		if mounted, err := isolation.IsMounted(file.Path()); err != nil {
			return fmt.Errorf(failedCheckMountPointFmt, file, err)
		} else if mounted {
			log.DefaultLogger().Object(vmi).Infof("unmounting container disk at path %s", file)
			// #nosec No risk for attacket injection. Parameters are predefined strings
			out, err := virt_chroot.UmountChroot(file.Path()).CombinedOutput()
			if err != nil {
				return fmt.Errorf(failedUnmountFmt, file, string(out), err)
			}
		}
	}
	err = m.mountRecorder.DeleteMountRecord(vmi)
	if err != nil {
		return err
	}

	return nil
}

func (m *mounter) ContainerDisksReady(vmi *v1.VirtualMachineInstance, notInitializedSince time.Time) (bool, error) {
	for i, volume := range vmi.Spec.Volumes {
		if volume.ContainerDisk != nil {
			_, err := m.socketPathGetter(vmi, i)
			if err != nil {
				log.DefaultLogger().Object(vmi).Reason(err).Infof("containerdisk %s not yet ready", volume.Name)
				if time.Now().After(notInitializedSince.Add(m.suppressWarningTimeout)) {
					return false, fmt.Errorf("containerdisk %s still not ready after one minute", volume.Name)
				}
				return false, nil
			}
		}
	}
	log.DefaultLogger().Object(vmi).V(4).Info("all containerdisks are ready")
	return true, nil
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
		return fmt.Errorf("failed to find socker path for kernel artifacts: %v", err)
	}

	record := []recorder.MountTargetEntry{
		{
			TargetFile: unsafepath.UnsafeAbsolute(targetDir.Raw()),
			SocketFile: socketFilePath,
		},
	}

	err = m.mountRecorder.AddMountRecord(vmi, record)
	if err != nil {
		return err
	}

	nodeRes := isolation.NodeIsolationResult()

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

		mounted, err := nodeRes.AreMounted(artifactFiles...)
		return mounted, err
	}

	if isMounted, err := areKernelArtifactsMounted(targetDir, targetInitrdPath, targetKernelPath); err != nil {
		return fmt.Errorf("failed to determine if %s is already mounted: %v", targetDir, err)
	} else if !isMounted {
		log.Log.Object(vmi).Infof("kernel artifacts are not mounted - mounting...")

		res, err := m.podIsolationDetector.DetectForSocket(vmi, socketFilePath)
		if err != nil {
			return fmt.Errorf("failed to detect socket for containerDisk %v: %v", kernelBootName, err)
		}
		mountRootPath, err := isolation.ParentPathForRootMount(nodeRes, res)
		if err != nil {
			return fmt.Errorf("failed to detect root mount point of %v on the node: %v", kernelBootName, err)
		}

		mount := func(artifactPath string, targetPath *safepath.Path) error {

			sourcePath, err := containerdisk.GetImage(mountRootPath, artifactPath)
			if err != nil {
				return err
			}

			out, err := virt_chroot.MountChroot(sourcePath, targetPath, true).CombinedOutput()
			if err != nil {
				return fmt.Errorf("failed to bindmount %v: %v : %v", kernelBootName, string(out), err)
			}

			return nil
		}

		if kb.InitrdPath != "" {
			if err = mount(kb.InitrdPath, targetInitrdPath); err != nil {
				return err
			}
		}

		if kb.KernelPath != "" {
			if err = mount(kb.KernelPath, targetKernelPath); err != nil {
				return err
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

	record, err := m.mountRecorder.GetMountRecord(vmi)
	if err != nil {
		return fmt.Errorf("failed to get mount target record: %v", err)
	}
	if len(record) == 0 {
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

	for _, entry := range record {
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
		return nil
	}

	return fmt.Errorf("kernel artifacts record wasn't found")
}
