package services

import (
	"fmt"
	"path/filepath"

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
	resources.Limits[k8sv1.ResourceMemory] = resource.MustParse("40M")

	if dedicatedCPUs || guaranteedQOS {
		resources.Limits[k8sv1.ResourceCPU] = resource.MustParse("10m")
	} else {
		resources.Limits[k8sv1.ResourceCPU] = resource.MustParse("100m")
	}

	if guaranteedQOS {
		resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("40M")
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

func virtioFSMountPoint(volumeName string) string {
	return fmt.Sprintf("/%s", volumeName)
}

func VirtioFSSocketPath(volumeName string) string {
	socketName := fmt.Sprintf("%s.sock", volumeName)
	return filepath.Join(virtiofs.VirtioFSContainersMountBaseDir, socketName)
}

func generateContainerFromVolume(volume *v1.Volume, image string) k8sv1.Container {
	resources := resourcesForVirtioFSContainer(false, false)

	socketPathArg := fmt.Sprintf("--socket-path=%s", VirtioFSSocketPath(volume.Name))
	sourceArg := fmt.Sprintf("source=%s", virtioFSMountPoint(volume.Name))
	args := []string{socketPathArg, "-o", sourceArg, "-o", "sandbox=chroot", "-o", "xattr", "-o", "xattrmap=:map::user.virtiofsd.:"}

	return k8sv1.Container{
		Name:            fmt.Sprintf("virtiofs-%s", volume.Name),
		Image:           image,
		ImagePullPolicy: k8sv1.PullIfNotPresent,
		Command:         []string{"/usr/libexec/virtiofsd"},
		Args:            args,
		VolumeMounts: []k8sv1.VolumeMount{
			// This is required to pass socket to compute
			{
				Name:      virtiofs.VirtioFSContainers,
				MountPath: virtiofs.VirtioFSContainersMountBaseDir,
			},
			{
				Name:      volume.Name,
				MountPath: virtioFSMountPoint(volume.Name),
			},
		},
		Resources:       resources,
		SecurityContext: securityContextVirtioFS(),
	}
}
