package services

import (
	"fmt"
	"path/filepath"

	"kubevirt.io/kubevirt/pkg/config"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/pointer"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/virtiofs"
)

func generateVirtioFSContainers(vmi *v1.VirtualMachineInstance, image string) []k8sv1.Container {
	passthroughFSVolumes := make(map[string]struct{})
	for i := range vmi.Spec.Domain.Devices.Filesystems {
		passthroughFSVolumes[vmi.Spec.Domain.Devices.Filesystems[i].Name] = struct{}{}
	}
	if len(passthroughFSVolumes) == 0 {
		return nil
	}

	containers := []k8sv1.Container{}
	for _, volume := range vmi.Spec.Volumes {
		if _, isPassthroughFSVolume := passthroughFSVolumes[volume.Name]; isPassthroughFSVolume {

			container := generateContainerFromVolume(&volume, image)
			containers = append(containers, container)

		}
	}

	return containers
}

func resourcesForVirtioFSContainer(dedicatedCPUs bool, guaranteedQOS bool) k8sv1.ResourceRequirements {
	resources := k8sv1.ResourceRequirements{Requests: k8sv1.ResourceList{}, Limits: k8sv1.ResourceList{}}

	// TODO: Find out correct values
	resources.Requests[k8sv1.ResourceCPU] = resource.MustParse("10m")
	resources.Limits[k8sv1.ResourceMemory] = resource.MustParse("80M")

	if dedicatedCPUs || guaranteedQOS {
		resources.Limits[k8sv1.ResourceCPU] = resource.MustParse("10m")
	} else {
		resources.Limits[k8sv1.ResourceCPU] = resource.MustParse("100m")
	}

	if guaranteedQOS {
		resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("80M")
	} else {
		resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("1M")
	}

	return resources

}

var userAndGroup = int64(util.RootUser)

func getVirtiofsCapabilities() []k8sv1.Capability {
	return []k8sv1.Capability{
		"CHOWN",
		"DAC_OVERRIDE",
		"FOWNER",
		"FSETID",
		"SETGID",
		"SETUID",
		"MKNOD",
		"SETFCAP",
		"SYS_CHROOT",
	}
}

func securityContextVirtioFS() *k8sv1.SecurityContext {

	return &k8sv1.SecurityContext{
		RunAsUser:    &userAndGroup,
		RunAsGroup:   &userAndGroup,
		RunAsNonRoot: pointer.Bool(false),
		Capabilities: &k8sv1.Capabilities{
			Add: getVirtiofsCapabilities(),
		},
	}
}

func isConfig(volume *v1.Volume) bool {
	return volume.ConfigMap != nil || volume.Secret != nil ||
		volume.ServiceAccount != nil || volume.DownwardAPI != nil
}

func isAutomount(volume *v1.Volume) bool {
	// The template service sets pod.Spec.AutomountServiceAccountToken as true
	return volume.ServiceAccount != nil
}

func virtioFSMountPoint(volume *v1.Volume) string {
	volumeMountPoint := fmt.Sprintf("/%s", volume.Name)

	if volume.ConfigMap != nil {
		volumeMountPoint = config.GetConfigMapSourcePath(volume.Name)
	} else if volume.Secret != nil {
		volumeMountPoint = config.GetSecretSourcePath(volume.Name)
	} else if volume.ServiceAccount != nil {
		volumeMountPoint = config.ServiceAccountSourceDir
	} else if volume.DownwardAPI != nil {
		volumeMountPoint = config.GetDownwardAPISourcePath(volume.Name)
	}

	return volumeMountPoint
}

func VirtioFSSocketPath(volumeName string) string {
	socketName := fmt.Sprintf("%s.sock", volumeName)
	return filepath.Join(virtiofs.VirtioFSContainersMountBaseDir, socketName)
}

func generateContainerFromVolume(volume *v1.Volume, image string) k8sv1.Container {
	resources := resourcesForVirtioFSContainer(false, false)

	socketPathArg := fmt.Sprintf("--socket-path=%s", VirtioFSSocketPath(volume.Name))
	sourceArg := fmt.Sprintf("--shared-dir=%s", virtioFSMountPoint(volume))
	args := []string{socketPathArg, sourceArg, "--cache=auto", "--sandbox=chroot"}

	if !isConfig(volume) {
		args = append(args, "--xattr")
	}

	volumeMounts := []k8sv1.VolumeMount{
		// This is required to pass socket to compute
		{
			Name:      virtiofs.VirtioFSContainers,
			MountPath: virtiofs.VirtioFSContainersMountBaseDir,
		},
	}

	if !isAutomount(volume) {
		volumeMounts = append(volumeMounts, k8sv1.VolumeMount{
			Name:      volume.Name,
			MountPath: virtioFSMountPoint(volume),
		})
	}

	return k8sv1.Container{
		Name:            fmt.Sprintf("virtiofs-%s", volume.Name),
		Image:           image,
		ImagePullPolicy: k8sv1.PullIfNotPresent,
		Command:         []string{"/usr/libexec/virtiofsd"},
		Args:            args,
		VolumeMounts:    volumeMounts,
		Resources:       resources,
		SecurityContext: securityContextVirtioFS(),
	}
}
