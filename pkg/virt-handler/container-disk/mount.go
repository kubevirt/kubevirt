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

package container_disk

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/checkpoint"
	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	"kubevirt.io/kubevirt/pkg/safepath"
	containerdisk "kubevirt.io/kubevirt/pkg/storage/container-disk"
	"kubevirt.io/kubevirt/pkg/unsafepath"
	"kubevirt.io/kubevirt/pkg/util"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
	"kubevirt.io/kubevirt/pkg/virt-handler/mountrecord"
	virt_chroot "kubevirt.io/kubevirt/pkg/virt-handler/virt-chroot"

	v1 "kubevirt.io/api/core/v1"
)

const (
	failedCheckMountPointFmt = "failed to check mount point for containerDisk %v: %v"
	failedUnmountFmt         = "failed to unmount containerDisk %v: %v : %v"
)

var (
	ErrWaitingForDisks   = errors.New("waiting for containerdisks")
	ErrDiskContainerGone = errors.New("disk container is gone")
)

//go:generate mockgen -source $GOFILE -package=$GOPACKAGE -destination=generated_mock_$GOFILE

type mounter struct {
	podIsolationDetector       isolation.PodIsolationDetector
	mountRecords               *mountrecord.Store
	suppressWarningTimeout     time.Duration
	needsBindMountFunc         needsBindMountFunc
	socketPathGetter           containerdisk.SocketPathGetter
	kernelBootSocketPathGetter containerdisk.KernelBootSocketPathGetter
	clusterConfig              *virtconfig.ClusterConfig
	nodeIsolationResult        isolation.IsolationResult
}

type Mounter interface {
	ContainerDisksReady(vmi *v1.VirtualMachineInstance, notInitializedSince time.Time) (bool, error)
	MountAndVerify(vmi *v1.VirtualMachineInstance) error
	Unmount(vmi *v1.VirtualMachineInstance) error
}

type kernelArtifacts struct {
	kernel *safepath.Path
	initrd *safepath.Path
}

func NewMounter(isoDetector isolation.PodIsolationDetector, checkpointManager checkpoint.CheckpointManager, clusterConfig *virtconfig.ClusterConfig) Mounter {
	return &mounter{
		mountRecords:               mountrecord.NewStore(checkpointManager),
		podIsolationDetector:       isoDetector,
		suppressWarningTimeout:     1 * time.Minute,
		needsBindMountFunc:         newNeedsBindMountFunc(""),
		socketPathGetter:           containerdisk.NewSocketPathGetter(""),
		kernelBootSocketPathGetter: containerdisk.NewKernelBootSocketPathGetter(""),
		clusterConfig:              clusterConfig,
		nodeIsolationResult:        isolation.NodeIsolationResult(),
	}
}

// Mount takes a vmi and mounts all container disks of the VMI, so that they are visible for the qemu process.
// Additionally qcow2 images are validated if "verify" is true. The validation happens with rlimits set, to avoid DOS.
func (m *mounter) MountAndVerify(vmi *v1.VirtualMachineInstance) error {
	if m.clusterConfig.ImageVolumeEnabled() {
		bindMountNeeded, err := m.needsBindMountFunc(vmi)
		if err != nil {
			return fmt.Errorf("fail to detect if bind mount needed for vmi: %s in namespace: %v. err: %v", vmi.Name, vmi.Namespace, err)
		}
		if !bindMountNeeded {
			return nil
		}
	}

	var entries []mountrecord.Entry
	for i, volume := range vmi.Spec.Volumes {
		if volume.ContainerDisk != nil {
			diskTargetDir, err := containerdisk.GetDiskTargetDirFromHostView(vmi)
			if err != nil {
				return err
			}
			diskName := containerdisk.GetDiskTargetName(i)
			// If diskName is a symlink it will fail if the target exists.
			if err := safepath.TouchAtNoFollow(diskTargetDir, diskName, os.ModePerm); err != nil {
				if !os.IsExist(err) {
					return fmt.Errorf("failed to create mount point target: %v", err)
				}
			}
			targetFile, err := safepath.JoinNoFollow(diskTargetDir, diskName)
			if err != nil {
				return err
			}

			sock, err := m.socketPathGetter(vmi, i)
			if err != nil {
				return err
			}

			entries = append(entries, mountrecord.Entry{
				TargetFile: unsafepath.UnsafeAbsolute(targetFile.Raw()),
				SocketFile: sock,
			})
		}
	}

	if err := m.mountRecords.Add(vmi.UID, entries...); err != nil {
		return err
	}

	for i, volume := range vmi.Spec.Volumes {
		if volume.ContainerDisk != nil {
			diskTargetDir, err := containerdisk.GetDiskTargetDirFromHostView(vmi)
			if err != nil {
				return err
			}
			diskName := containerdisk.GetDiskTargetName(i)
			targetFile, err := safepath.JoinNoFollow(diskTargetDir, diskName)
			if err != nil {
				return err
			}

			if isMounted, err := isolation.IsMounted(targetFile); err != nil {
				return fmt.Errorf("failed to determine if %s is already mounted: %v", targetFile, err)
			} else if !isMounted {

				sourceFile, err := m.getContainerDiskPath(vmi, &volume, i)
				if err != nil {
					return fmt.Errorf("failed to find a sourceFile in containerDisk %v: %v", volume.Name, err)
				}

				log.DefaultLogger().Object(vmi).Infof("Bind mounting container disk at %s to %s", sourceFile, targetFile)
				out, err := virt_chroot.MountChroot(sourceFile, targetFile, true).CombinedOutput()
				if err != nil {
					return fmt.Errorf("failed to bindmount containerDisk %v: %v : %v", volume.Name, string(out), err)
				}
			}
		}
	}
	err := m.mountKernelArtifacts(vmi, true)
	if err != nil {
		return fmt.Errorf("error mounting kernel artifacts: %v", err)
	}

	return nil
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

	entries, err := m.mountRecords.Entries(vmi.UID)
	if err != nil {
		return err
	} else if len(entries) == 0 {
		log.DefaultLogger().Object(vmi).Infof("No container disk mount entries found to unmount")
		return nil
	}

	log.DefaultLogger().Object(vmi).Infof("Found container disk mount entries")
	for _, entry := range entries {
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
			// #nosec No risk for attacker injection. Parameters are predefined strings
			out, err := virt_chroot.UmountChroot(file.Path()).CombinedOutput()
			if err != nil {
				return fmt.Errorf(failedUnmountFmt, file, string(out), err)
			}
		}
	}
	for _, entry := range entries {
		_ = os.Remove(entry.TargetFile)
		_ = os.Remove(entry.SocketFile)
	}
	return m.mountRecords.Delete(vmi.UID)
}

func (m *mounter) ContainerDisksReady(vmi *v1.VirtualMachineInstance, notInitializedSince time.Time) (bool, error) {
	if m.clusterConfig.ImageVolumeEnabled() {
		bindMountNeeded, err := m.needsBindMountFunc(vmi)
		if err != nil {
			return false, fmt.Errorf("fail to detect if bind mount needed for vmi: %s in namespace: %v. err: %v", vmi.Name, vmi.Namespace, err)
		}
		if !bindMountNeeded {
			return true, nil
		}
	}
	for i, volume := range vmi.Spec.Volumes {
		if volume.ContainerDisk != nil {
			sock, err := m.socketPathGetter(vmi, i)
			if err == nil {
				_, err = m.podIsolationDetector.DetectForSocket(sock)
			}

			if err != nil {
				log.DefaultLogger().Object(vmi).Reason(err).Infof("containerdisk %s not yet ready", volume.Name)
				if time.Now().After(notInitializedSince.Add(m.suppressWarningTimeout)) {
					return false, fmt.Errorf("containerdisk %s still not ready after one minute", volume.Name)
				}
				return false, nil
			}

		}
	}

	if util.HasKernelBootContainerImage(vmi) {
		sock, err := m.kernelBootSocketPathGetter(vmi)
		if err == nil {
			_, err = m.podIsolationDetector.DetectForSocket(sock)
		}
		if err != nil {
			log.DefaultLogger().Object(vmi).Reason(err).Info("kernelboot container not yet ready")
			if time.Now().After(notInitializedSince.Add(m.suppressWarningTimeout)) {
				return false, fmt.Errorf("kernelboot container still not ready after one minute")
			}
			return false, nil
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
		return fmt.Errorf("failed to find socket path for kernel artifacts: %v", err)
	}

	err = m.mountRecords.Add(vmi.UID, mountrecord.Entry{
		TargetFile: unsafepath.UnsafeAbsolute(targetDir.Raw()),
		SocketFile: socketFilePath,
	})
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

	entries, err := m.mountRecords.Entries(vmi.UID)
	if err != nil {
		return fmt.Errorf("failed to get mount target record: %v", err)
	} else if len(entries) == 0 {
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

	idx := slices.IndexFunc(entries, func(e mountrecord.Entry) bool {
		return strings.Contains(e.TargetFile, containerdisk.KernelBootName)
	})
	if idx < 0 {
		return fmt.Errorf("kernel artifacts record wasn't found")
	}
	targetDir, err := safepath.NewFileNoFollow(entries[idx].TargetFile)
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

func (m *mounter) getContainerDiskPath(vmi *v1.VirtualMachineInstance, volume *v1.Volume, volumeIndex int) (*safepath.Path, error) {
	sock, err := m.socketPathGetter(vmi, volumeIndex)
	if err != nil {
		return nil, ErrDiskContainerGone
	}

	res, err := m.podIsolationDetector.DetectForSocket(sock)
	if err != nil {
		return nil, fmt.Errorf("failed to detect socket for containerDisk %v: %v", volume.Name, err)
	}

	mountPoint, err := isolation.ParentPathForRootMount(m.nodeIsolationResult, res)
	if err != nil {
		return nil, fmt.Errorf("failed to detect root mount point of containerDisk %v on the node: %v", volume.Name, err)
	}

	return containerdisk.GetImage(mountPoint, volume.ContainerDisk.Path)
}

func (m *mounter) getKernelArtifactPaths(vmi *v1.VirtualMachineInstance) (*kernelArtifacts, error) {
	sock, err := m.kernelBootSocketPathGetter(vmi)
	if err != nil {
		return nil, ErrDiskContainerGone
	}

	res, err := m.podIsolationDetector.DetectForSocket(sock)
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

type needsBindMountFunc func(vmi *v1.VirtualMachineInstance) (bool, error)

func newNeedsBindMountFunc(baseDir string) needsBindMountFunc {
	return func(vmi *v1.VirtualMachineInstance) (bool, error) {
		for podUID := range vmi.Status.ActivePods {
			virtLauncherSocketPath := cmdclient.SocketDirectoryOnHost(string(podUID))
			launcherSocketExists, err := diskutils.FileExists(virtLauncherSocketPath)
			if err != nil {
				return false, err
			}
			basePath := fmt.Sprintf("%s/pods/%s/containers", baseDir, string(podUID))
			containerDiskPath := filepath.Join(basePath, "container-disk-binary")
			containerDiskInitContainerExists, err := diskutils.FileExists(containerDiskPath)
			if err != nil {
				return false, err
			}
			// we must check for launcherSocket to make sure this isn't an old launcher that is already completed
			if launcherSocketExists && containerDiskInitContainerExists {
				return true, nil
			}
		}
		return false, nil
	}
}
