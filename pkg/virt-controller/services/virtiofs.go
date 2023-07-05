package services

import (
	"fmt"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/pointer"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/config"

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

var privilegedId = int64(util.RootUser)
var restrictedId = int64(util.NonRootUID)

type securityProfile uint8

const (
	restricted securityProfile = iota
	privileged
)

func isRestricted(profile securityProfile) bool {
	return profile == restricted
}

func isPrivileged(profile securityProfile) bool {
	return profile == privileged
}

func virtiofsCredential(profile securityProfile) *int64 {
	credential := &restrictedId
	if isPrivileged(profile) {
		credential = &privilegedId
	}
	return credential
}

func virtiofsCapabilities(profile securityProfile) *k8sv1.Capabilities {
	if isPrivileged(profile) {
		return &k8sv1.Capabilities{
			Add: []k8sv1.Capability{
				"CHOWN",
				"DAC_OVERRIDE",
				"FOWNER",
				"FSETID",
				"SETGID",
				"SETUID",
				"MKNOD",
				"SETFCAP",
				"SYS_CHROOT",
			},
		}
	}

	return &k8sv1.Capabilities{
		Drop: []k8sv1.Capability{
			"ALL",
		},
	}
}

func securityContextVirtioFS(profile securityProfile) *k8sv1.SecurityContext {
	credential := virtiofsCredential(profile)

	return &k8sv1.SecurityContext{
		RunAsUser:                credential,
		RunAsGroup:               credential,
		RunAsNonRoot:             pointer.Bool(isRestricted(profile)),
		AllowPrivilegeEscalation: pointer.Bool(isPrivileged(profile)),
		Capabilities:             virtiofsCapabilities(profile),
	}
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
	args := []string{socketPathArg, sourceArg, "--cache=auto"}

	securityProfile := restricted
	sandbox := "none"
	if virtiofs.RequiresRootPrivileges(volume) {
		securityProfile = privileged
		sandbox = "chroot"
		args = append(args, "--xattr")
	}

	sandboxArg := fmt.Sprintf("--sandbox=%s", sandbox)
	args = append(args, sandboxArg)

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
		SecurityContext: securityContextVirtioFS(securityProfile),
	}
}
