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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package containerdisk

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	kubev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"kubevirt.io/kubevirt/pkg/safepath"

	ephemeraldisk "kubevirt.io/kubevirt/pkg/ephemeral-disk"
	virtpointer "kubevirt.io/kubevirt/pkg/pointer"

	k8sv1 "k8s.io/api/core/v1"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/util"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

const (
	mountBaseDir           = "/var/run/kubevirt/container-disks"
	KernelBootName         = "kernel-boot"
	KernelBootVolumeName   = KernelBootName + "-volume"
	DiskSourceFallbackPath = "/disk"
	pidFileDir             = "/var/run/containerdisk"
	Pidfile                = "pidfile"
)

type process interface {
	Signal(os.Signal) error
}

type ContainerDiskManager struct {
	pidFileDir   string
	mountBaseDir string
	procfs       string
	cdVolumes    []string
	findProcess  func(pid int) (process, error)
	getImageInfo func(img string) (*ImgInfo, error)
}

func NewContainerDiskManager() *ContainerDiskManager {
	return &ContainerDiskManager{
		pidFileDir:   pidFileDir,
		mountBaseDir: mountBaseDir,
		procfs:       "/proc",
		findProcess:  func(pid int) (process, error) { return os.FindProcess(pid) },
		getImageInfo: GetImageInfo,
	}
}

const ephemeralStorageOverheadSize = "50M"

var digestRegex = regexp.MustCompile(`sha256:([a-zA-Z0-9]+)`)

func GetDiskTargetName(volumeIndex int) string {
	return fmt.Sprintf("disk_%d.img", volumeIndex)
}

func (c *ContainerDiskManager) GetContainerDisksDirLauncherView() string {
	return c.mountBaseDir
}

func (c *ContainerDiskManager) GetDiskTargetPathFromLauncherView(volumeIndex int) string {
	return filepath.Join(c.mountBaseDir, GetDiskTargetName(volumeIndex))
}

func (c *ContainerDiskManager) GetKernelBootArtifactDirFromLauncherView() string {
	return filepath.Join(c.mountBaseDir, KernelBootVolumeName)
}

func (c *ContainerDiskManager) GetKernelBootArtifactPathFromLauncherView(artifact string) string {
	artifactBase := filepath.Base(artifact)
	dir := c.GetKernelBootArtifactDirFromLauncherView()
	return filepath.Join(dir, artifactBase)
}

func GetImage(root *safepath.Path, imagePath string) (*safepath.Path, error) {
	if imagePath != "" {
		var err error
		resolvedPath, err := root.AppendAndResolveWithRelativeRoot(imagePath)
		if err != nil {
			return nil, fmt.Errorf("failed to determine custom image path %s: %v", imagePath, err)
		}
		return resolvedPath, nil
	} else {
		fallbackPath, err := root.AppendAndResolveWithRelativeRoot(DiskSourceFallbackPath)
		if err != nil {
			return nil, fmt.Errorf("failed to determine default image path %v: %v", fallbackPath, err)
		}
		var files []os.DirEntry
		err = fallbackPath.ExecuteNoFollow(func(safePath string) (err error) {
			files, err = os.ReadDir(safePath)
			return err
		})
		if err != nil {
			return nil, fmt.Errorf("failed to check default image path %s: %v", fallbackPath, err)
		}
		if len(files) == 0 {
			return nil, fmt.Errorf("no file found in folder %s, no disk present", DiskSourceFallbackPath)
		} else if len(files) > 1 {
			return nil, fmt.Errorf("more than one file found in folder %s, only one disk is allowed", DiskSourceFallbackPath)
		}
		fileName := files[0].Name()
		resolvedPath, err := root.AppendAndResolveWithRelativeRoot(DiskSourceFallbackPath, fileName)
		if err != nil {
			return nil, fmt.Errorf("failed to check default image path %s: %v", imagePath, err)
		}
		return resolvedPath, nil
	}
}

func GenerateInitContainers(vmi *v1.VirtualMachineInstance, config *virtconfig.ClusterConfig, imageIDs map[string]string, podVolumeName string, binVolumeName string) []kubev1.Container {
	return generateContainersHelper(vmi, config, imageIDs, podVolumeName, binVolumeName, true)
}

func GenerateContainers(vmi *v1.VirtualMachineInstance, config *virtconfig.ClusterConfig, imageIDs map[string]string, podVolumeName string, binVolumeName string) []kubev1.Container {
	return generateContainersHelper(vmi, config, imageIDs, podVolumeName, binVolumeName, false)
}

func GenerateKernelBootContainer(vmi *v1.VirtualMachineInstance, config *virtconfig.ClusterConfig, imageIDs map[string]string, podVolumeName string, binVolumeName string) *kubev1.Container {
	return generateKernelBootContainerHelper(vmi, config, imageIDs, podVolumeName, binVolumeName, false)
}

func GenerateKernelBootInitContainer(vmi *v1.VirtualMachineInstance, config *virtconfig.ClusterConfig, imageIDs map[string]string, podVolumeName string, binVolumeName string) *kubev1.Container {
	return generateKernelBootContainerHelper(vmi, config, imageIDs, podVolumeName, binVolumeName, true)
}

func generateKernelBootContainerHelper(vmi *v1.VirtualMachineInstance, config *virtconfig.ClusterConfig, imageIDs map[string]string, podVolumeName string, binVolumeName string, isInit bool) *kubev1.Container {
	if !util.HasKernelBootContainerImage(vmi) {
		return nil
	}

	kernelBootContainer := vmi.Spec.Domain.Firmware.KernelBoot.Container

	kernelBootVolume := v1.Volume{
		Name: KernelBootVolumeName,
		VolumeSource: v1.VolumeSource{
			ContainerDisk: &v1.ContainerDiskSource{
				Image:           kernelBootContainer.Image,
				ImagePullSecret: kernelBootContainer.ImagePullSecret,
				Path:            "/",
				ImagePullPolicy: kernelBootContainer.ImagePullPolicy,
			},
		},
	}

	const fakeVolumeIdx = 0 // volume index makes no difference for kernel-boot container
	return generateContainerFromVolume(vmi, config, imageIDs, podVolumeName, binVolumeName, isInit, true, &kernelBootVolume, fakeVolumeIdx)
}

// The controller uses this function to generate the container
// specs for hosting the container registry disks.
func generateContainersHelper(vmi *v1.VirtualMachineInstance, config *virtconfig.ClusterConfig, imageIDs map[string]string, podVolumeName string, binVolumeName string, isInit bool) []kubev1.Container {
	var containers []kubev1.Container

	// Make VirtualMachineInstance Image Wrapper Containers
	for index, volume := range vmi.Spec.Volumes {
		if volume.Name == KernelBootVolumeName {
			continue
		}
		if container := generateContainerFromVolume(vmi, config, imageIDs, podVolumeName, binVolumeName, isInit, false, &volume, index); container != nil {
			containers = append(containers, *container)
		}
	}
	return containers
}

func generateContainerFromVolume(vmi *v1.VirtualMachineInstance, config *virtconfig.ClusterConfig, imageIDs map[string]string, podVolumeName, binVolumeName string, isInit, isKernelBoot bool, volume *v1.Volume, volumeIdx int) *kubev1.Container {
	if volume.ContainerDisk == nil {
		return nil
	}

	diskContainerImage := volume.ContainerDisk.Image
	if img, exists := imageIDs[volume.Name]; exists {
		diskContainerImage = img
	}

	resources := kubev1.ResourceRequirements{}
	resources.Requests = make(kubev1.ResourceList)
	resources.Limits = make(kubev1.ResourceList)

	resources.Requests[kubev1.ResourceCPU] = resource.MustParse("1m")
	if cpuRequest := config.GetSupportContainerRequest(v1.ContainerDisk, kubev1.ResourceCPU); cpuRequest != nil {
		resources.Requests[kubev1.ResourceCPU] = *cpuRequest
	}
	resources.Requests[kubev1.ResourceMemory] = resource.MustParse("1M")
	if memRequest := config.GetSupportContainerRequest(v1.ContainerDisk, kubev1.ResourceMemory); memRequest != nil {
		resources.Requests[kubev1.ResourceMemory] = *memRequest
	}
	resources.Requests[kubev1.ResourceEphemeralStorage] = resource.MustParse(ephemeralStorageOverheadSize)

	resources.Limits[kubev1.ResourceCPU] = resource.MustParse("10m")
	if cpuLimit := config.GetSupportContainerLimit(v1.ContainerDisk, kubev1.ResourceCPU); cpuLimit != nil {
		resources.Limits[kubev1.ResourceCPU] = *cpuLimit
	}
	resources.Limits[kubev1.ResourceMemory] = resource.MustParse("40M")
	if memLimit := config.GetSupportContainerLimit(v1.ContainerDisk, kubev1.ResourceMemory); memLimit != nil {
		resources.Limits[kubev1.ResourceMemory] = *memLimit
	}

	if vmi.IsCPUDedicated() || vmi.WantsToHaveQOSGuaranteed() {
		resources.Requests[kubev1.ResourceCPU] = resources.Limits[kubev1.ResourceCPU]
		resources.Requests[kubev1.ResourceMemory] = resources.Limits[kubev1.ResourceMemory]
	}
	var userId int64 = util.NonRootUID
	container := &kubev1.Container{
		Image:           diskContainerImage,
		ImagePullPolicy: volume.ContainerDisk.ImagePullPolicy,
		Command:         []string{"/usr/bin/container-disk"},
		VolumeMounts: []kubev1.VolumeMount{
			{
				Name:      binVolumeName,
				MountPath: "/usr/bin",
			},
		},
		Resources: resources,
		SecurityContext: &kubev1.SecurityContext{
			RunAsUser:                &userId,
			RunAsNonRoot:             virtpointer.P(true),
			AllowPrivilegeEscalation: virtpointer.P(false),
			Capabilities: &kubev1.Capabilities{
				Drop: []kubev1.Capability{"ALL"},
			},
		},
	}
	switch {
	case isInit:
		container.Name = toContainerName(volume.Name) + "-init"
		container.Args = []string{"--no-op"}
	case isKernelBoot:
		container.Name = toContainerName(KernelBootVolumeName)
		container.VolumeMounts = append(container.VolumeMounts, kubev1.VolumeMount{
			Name:      GetPidfileVolumeName(KernelBootVolumeName),
			MountPath: pidFileDir,
		})
	default:
		container.Name = toContainerName(volume.Name)
		container.VolumeMounts = append(container.VolumeMounts, kubev1.VolumeMount{
			Name:      GetPidfileVolumeName(volume.Name),
			MountPath: pidFileDir,
		})
	}

	return container
}

func GetPidfileVolumeName(name string) string {
	return fmt.Sprintf("pidfile-%s", name)
}

func (c *ContainerDiskManager) GetPidfileDir(name string) string {
	return filepath.Join(c.pidFileDir, name)
}

func (c *ContainerDiskManager) GetPidfilePath(name string) string {
	return filepath.Join(c.GetPidfileDir(name), Pidfile)
}

func (c *ContainerDiskManager) GetVolumeMountPidfileContainerDisk(name string) k8sv1.VolumeMount {
	return k8sv1.VolumeMount{
		Name:      GetPidfileVolumeName(name),
		MountPath: c.GetPidfileDir(name),
	}
}

func (c *ContainerDiskManager) readPidfile(volName string) (int, error) {
	t, err := os.ReadFile(c.GetPidfilePath(volName))
	if err != nil {
		return -1, err
	}
	pid, err := strconv.Atoi(string(t))
	if err != nil {
		return -1, err
	}
	return pid, nil
}

func (c *ContainerDiskManager) GetContainerDiksPath(volume *v1.Volume) (string, error) {
	if volume.VolumeSource.ContainerDisk == nil {
		return "", fmt.Errorf("not a container disk")
	}
	pid, err := c.readPidfile(volume.Name)
	if err != nil {
		return "", err
	}
	if volume.VolumeSource.ContainerDisk.Path != "" {
		file := filepath.Join(c.procfs, strconv.Itoa(pid), "/root", volume.VolumeSource.ContainerDisk.Path)
		if _, err := os.Stat(file); err != nil {
			return "", fmt.Errorf("failed to check the file %s: %v", volume.VolumeSource.ContainerDisk.Path, err)
		}
		return file, nil
	}
	fullPath := filepath.Join(c.procfs, strconv.Itoa(pid), "/root", DiskSourceFallbackPath)
	files, err := os.ReadDir(fullPath)
	if err != nil {
		return "", err
	}
	if len(files) == 0 {
		return "", fmt.Errorf("no file found in folder %s, no disk present", DiskSourceFallbackPath)
	} else if len(files) > 1 {
		return "", fmt.Errorf("more than one file found in folder %s, only one disk is allowed", DiskSourceFallbackPath)
	}

	return filepath.Join(fullPath, files[0].Name()), nil
}

func createSymlink(oldpath, newpath string) error {
	if _, err := os.Stat(newpath); err == nil {
		return fmt.Errorf("new path %s for the symlink already exists", newpath)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return os.Symlink(oldpath, newpath)
}

func checkExistingSymlink(symlink, backingFile string) (bool, error) {
	fileInfo, err := os.Lstat(symlink)
	if err == nil {
		if fileInfo.Mode()&fs.ModeSymlink == 0 {
			return true, fmt.Errorf("the file %s isn't a symlink to the disk image", symlink)
		}
		link, err := os.Readlink(symlink)
		if err != nil {
			return true, fmt.Errorf("failed checking the symlink for %s: %v", link, err)
		}
		if link != backingFile {
			return true, fmt.Errorf("failed checking the symlink for %s, doesn't match with %s", link, backingFile)
		}
		return true, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return false, err
	}
	return false, nil
}

func (c *ContainerDiskManager) AccessKernelBoot(vmi *v1.VirtualMachineInstance) error {
	if !util.HasKernelBootContainerImage(vmi) {
		return nil
	}
	pid, err := c.readPidfile(KernelBootVolumeName)
	if err != nil {
		return err
	}

	kbc := vmi.Spec.Domain.Firmware.KernelBoot.Container
	if kbc.KernelPath != "" {
		// Create symlink for kernel path
		kernelPath := c.GetKernelBootArtifactPathFromLauncherView(kbc.KernelPath)
		contkernelPath := filepath.Join(c.procfs, strconv.Itoa(pid), "/root", kbc.KernelPath)
		exist, err := checkExistingSymlink(kernelPath, contkernelPath)
		if err != nil {
			return err
		}
		if !exist {
			if err := createSymlink(contkernelPath, kernelPath); err != nil {
				return err
			}
		}
	}

	if kbc.InitrdPath != "" {
		// Create symlink for the initrd path
		initrdPath := c.GetKernelBootArtifactPathFromLauncherView(kbc.InitrdPath)
		contInitrdPath := filepath.Join(c.procfs, strconv.Itoa(pid), "/root", kbc.InitrdPath)
		exist, err := checkExistingSymlink(initrdPath, contInitrdPath)
		if err != nil {
			return err
		}
		if !exist {
			if err := createSymlink(contInitrdPath, initrdPath); err != nil {
				return err
			}
		}

	}

	return nil
}

// TODO: should move this
type ImgInfo struct {
	// Format contains the format of the image
	Format string `json:"format"`
	// BackingFile is the file name of the backing file
	BackingFile string `json:"backing-filename"`
	// VirtualSize is the disk size of the image which will be read by vm
	VirtualSize int64 `json:"virtual-size"`
	// ActualSize is the size of the qcow2 image
	ActualSize int64 `json:"actual-size"`
}

func GetImageInfo(img string) (*ImgInfo, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("qemu-img", "info", "--output=json", img)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("qemu-img failed stdout:%s stderr:%s err:%v", string(stdout.Bytes()),
			string(stderr.Bytes()), err)
	}
	var info ImgInfo
	if err := json.Unmarshal(stdout.Bytes(), &info); err != nil {
		return nil, err
	}
	return &info, nil
}

func (c *ContainerDiskManager) CreateEphemeralImages(
	vmi *v1.VirtualMachineInstance,
	diskCreator ephemeraldisk.EphemeralDiskCreatorInterface,
) error {
	for i, volume := range vmi.Spec.Volumes {
		if volume.VolumeSource.ContainerDisk == nil {
			continue
		}
		backingFile, err := c.GetContainerDiksPath(&volume)
		if err != nil {
			return err
		}
		// Create symlink to the old location for containerdisk alpha2 where the backing image was created.
		// Using a constant path will faciliate upgraded and migrations
		symlink := c.GetDiskTargetPathFromLauncherView(i)
		exist, err := checkExistingSymlink(symlink, backingFile)
		if err != nil {
			return err
		}
		if !exist {
			if err := createSymlink(backingFile, symlink); err != nil {
				return err
			}
		}
		info, err := c.getImageInfo(backingFile)
		if err != nil {
			return err
		}
		if err := diskCreator.CreateBackedImageForVolume(volume, backingFile, info.Format); err != nil {
			return err
		}
	}

	return nil
}

// ExtractImageIDsFromSourcePod takes the VMI and its source pod to determine the exact image used by containerdisks and boot container images,
// which is recorded in the status section of a started pod; if the status section does not contain this info the tag is used.
// It returns a map where the key is the vlume name and the value is the imageID
func ExtractImageIDsFromSourcePod(vmi *v1.VirtualMachineInstance, sourcePod *kubev1.Pod) (imageIDs map[string]string) {
	imageIDs = map[string]string{}
	for _, volume := range vmi.Spec.Volumes {
		if volume.ContainerDisk == nil {
			continue
		}
		imageIDs[volume.Name] = volume.ContainerDisk.Image
	}

	if util.HasKernelBootContainerImage(vmi) {
		imageIDs[KernelBootVolumeName] = vmi.Spec.Domain.Firmware.KernelBoot.Container.Image
	}

	for _, status := range sourcePod.Status.ContainerStatuses {
		if !isImageVolume(status.Name) {
			continue
		}
		key := toVolumeName(status.Name)
		image, exists := imageIDs[key]
		if !exists {
			continue
		}
		imageIDs[key] = toPullableImageReference(image, status.ImageID)
	}
	return
}

func toPullableImageReference(image string, imageID string) string {
	baseImage := image
	if strings.LastIndex(image, "@sha256:") != -1 {
		baseImage = strings.Split(image, "@sha256:")[0]
	} else if colonIndex := strings.LastIndex(image, ":"); colonIndex > strings.LastIndex(image, "/") {
		baseImage = image[:colonIndex]
	}

	digestMatches := digestRegex.FindStringSubmatch(imageID)
	if len(digestMatches) < 2 {
		// failed to identify image digest for container, will use the image tag
		// as virt-handler will anyway check the checksum of the root disk image
		return image
	}
	return fmt.Sprintf("%s@sha256:%s", baseImage, digestMatches[1])
}

func isImageVolume(containerName string) bool {
	return strings.HasPrefix(containerName, "volume")
}

func toContainerName(volumeName string) string {
	return fmt.Sprintf("volume%s", volumeName)
}

func toVolumeName(containerName string) string {
	return strings.TrimPrefix(containerName, "volume")
}

func (c *ContainerDiskManager) WaitContainerDisksToBecomeReady(vmi *v1.VirtualMachineInstance, timeout time.Duration) error {
	errChan := make(chan error, 1)
	cds := make(map[string]bool)
	for _, v := range vmi.Spec.Volumes {
		if v.ContainerDisk != nil {
			cds[v.Name] = true
			c.cdVolumes = append(c.cdVolumes, v.Name)
		}

	}
	if util.HasKernelBootContainerImage(vmi) {
		cds[KernelBootVolumeName] = true
		c.cdVolumes = append(c.cdVolumes, KernelBootVolumeName)
	}
	go func() {
		for {
			for v, _ := range cds {
				path := c.GetPidfilePath(v)
				_, err := os.Stat(path)
				switch {
				case errors.Is(err, os.ErrNotExist):
					break
				case err != nil:
					errChan <- err
					return
				default:
					delete(cds, v)
				}
			}
			if len(cds) == 0 {
				errChan <- nil
				break
			}
			time.Sleep(1 * time.Second)
		}
	}()
	select {
	case err := <-errChan:
		return err
	case <-time.After(timeout * time.Second):
		return fmt.Errorf("timeout waiting for container disks to become ready")
	}
}

type errorWrapper struct {
	msg string
}

func (w *errorWrapper) wrapError(e1, e2 error) error {
	var errRes error
	if e1 == nil {
		errRes = fmt.Errorf("%s: %w", w.msg, e2)
	} else {
		errRes = fmt.Errorf("%w, %w", e1, e2)
	}
	return errRes
}

func (c *ContainerDiskManager) killContainerdisk(name string) error {
	pid, err := c.readPidfile(name)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	proc, err := c.findProcess(pid)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	if err := proc.Signal(os.Interrupt); err != nil {
		return err
	}
	return nil
}

func (c *ContainerDiskManager) StopContainerDiskContainers() error {
	w := &errorWrapper{msg: "failed to stop container disks"}
	var retErr error
	for _, v := range c.cdVolumes {
		if err := c.killContainerdisk(v); err != nil {
			retErr = w.wrapError(retErr, err)
		}
	}
	return retErr
}
