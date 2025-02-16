package services

import (
	"fmt"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/config"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/util"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virtiofs"
)

func generateVirtioFSContainers(vmi *v1.VirtualMachineInstance, image string, config *virtconfig.ClusterConfig) []k8sv1.Container {
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
			resources := resourcesForVirtioFSContainer(vmi.IsCPUDedicated(), vmi.IsCPUDedicated() || vmi.WantsToHaveQOSGuaranteed(), config)
			container := generateContainerFromVolume(&volume, image, resources)
			containers = append(containers, container)

		}
	}

	return containers
}

func resourcesForVirtioFSContainer(dedicatedCPUs bool, guaranteedQOS bool, config *virtconfig.ClusterConfig) k8sv1.ResourceRequirements {
	resources := k8sv1.ResourceRequirements{Requests: k8sv1.ResourceList{}, Limits: k8sv1.ResourceList{}}

	resources.Requests[k8sv1.ResourceCPU] = resource.MustParse("10m")
	if reqCpu := config.GetSupportContainerRequest(v1.VirtioFS, k8sv1.ResourceCPU); reqCpu != nil {
		resources.Requests[k8sv1.ResourceCPU] = *reqCpu
	}
	resources.Limits[k8sv1.ResourceMemory] = resource.MustParse("80M")
	if limMem := config.GetSupportContainerLimit(v1.VirtioFS, k8sv1.ResourceMemory); limMem != nil {
		resources.Limits[k8sv1.ResourceMemory] = *limMem
	}

	resources.Limits[k8sv1.ResourceCPU] = resource.MustParse("100m")
	if limCpu := config.GetSupportContainerLimit(v1.VirtioFS, k8sv1.ResourceCPU); limCpu != nil {
		resources.Limits[k8sv1.ResourceCPU] = *limCpu
	}
	if dedicatedCPUs || guaranteedQOS {
		resources.Requests[k8sv1.ResourceCPU] = resources.Limits[k8sv1.ResourceCPU]
	}

	if guaranteedQOS {
		resources.Requests[k8sv1.ResourceMemory] = resources.Limits[k8sv1.ResourceMemory]
	} else {
		resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("1M")
		if reqMem := config.GetSupportContainerRequest(v1.VirtioFS, k8sv1.ResourceMemory); reqMem != nil {
			resources.Requests[k8sv1.ResourceMemory] = *reqMem
		}
	}

	return resources

}

func isAutoMount(volume *v1.Volume) bool {
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

func generateContainerFromVolume(volume *v1.Volume, image string, resources k8sv1.ResourceRequirements) k8sv1.Container {

	socketPathArg := fmt.Sprintf("--socket-path=%s", virtiofs.VirtioFSSocketPath(volume.Name))
	sourceArg := fmt.Sprintf("--shared-dir=%s", virtioFSMountPoint(volume))

	args := []string{socketPathArg, sourceArg, "--sandbox=none", "--cache=auto"}

	// If some files cannot be migrated, let's allow the migration to finish.
	// Mark these files as invalid, the guest will not be able to access any such files,
	// receiving only errors
	args = append(args, "--migration-on-error=guest-error")

	// This mode look up its file references paths by reading the symlinks in /proc/self/fd,
	// falling back to iterating through the shared directory (exhaustive search) to find those paths.
	// This migration mode doesn't require any privileges.
	args = append(args, "--migration-mode=find-paths")

	volumeMounts := []k8sv1.VolumeMount{
		// This is required to pass socket to compute
		{
			Name:      virtiofs.VirtioFSContainers,
			MountPath: virtiofs.VirtioFSContainersMountBaseDir,
		},
	}

	if !isAutoMount(volume) {
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
		SecurityContext: &k8sv1.SecurityContext{
			RunAsUser:                pointer.P(int64(util.NonRootUID)),
			RunAsGroup:               pointer.P(int64(util.NonRootUID)),
			RunAsNonRoot:             pointer.P(true),
			AllowPrivilegeEscalation: pointer.P(false),
			Capabilities: &k8sv1.Capabilities{
				Drop: []k8sv1.Capability{
					"ALL",
				},
			},
		},
	}
}
