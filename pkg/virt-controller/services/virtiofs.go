package services

import (
	"fmt"

	k8sv1 "k8s.io/api/core/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/config"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/util"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virtiofs"
)

func generateVirtioFSContainers(vmi *v1.VirtualMachineInstance, image string, config *virtconfig.ClusterConfig) []k8sv1.Container {
	passthroughFSVolumes := make(map[string]v1.FilesystemVirtiofs)
	for i := range vmi.Spec.Domain.Devices.Filesystems {
		fs := vmi.Spec.Domain.Devices.Filesystems[i]
		// Virtiofs presence is the type discriminator; align with
		// VirtiofsConfigurator and util.IsVMIVirtiofsEnabled.
		if fs.Virtiofs == nil {
			continue
		}
		passthroughFSVolumes[fs.Name] = *fs.Virtiofs
	}
	if len(passthroughFSVolumes) == 0 {
		return nil
	}

	containers := []k8sv1.Container{}
	for _, volume := range vmi.Spec.Volumes {
		if virtioFS, isPassthroughFSVolume := passthroughFSVolumes[volume.Name]; isPassthroughFSVolume {
			// Skip ContainerPath volumes - they are handled by the pod mutating webhook
			// because external mutators inject the actual volumes after pod creation
			if volume.ContainerPath != nil {
				continue
			}
			resources := virtiofs.ResourcesForVirtioFSContainer(vmi.IsCPUDedicated(), vmi.IsCPUDedicated() || vmi.WantsToHaveQOSGuaranteed(), config)
			container := generateContainerFromVolume(&volume, virtioFS, image, resources)
			containers = append(containers, container)

		}
	}

	return containers
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

func generateContainerFromVolume(volume *v1.Volume, virtioFS v1.FilesystemVirtiofs, image string, resources k8sv1.ResourceRequirements) k8sv1.Container {

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

	if volume.ServiceAccount != nil {
		// In K8s 1.36+, the Service Account symlink token file owner matches
		// the pod's `RunAsUser` UID
		// (see https://github.com/kubernetes/kubernetes/pull/137332).
		//
		// Because `/var/run/secrets/kubernetes.io/serviceaccount/` has the
		// sticky bit (`+t`) set, the guest kernel's protected symlinks feature
		// blocks the `root` user from following this symlink since the symlink
		// owner no longer matches the directory owner.
		// (see: https://github.com/kubevirt/kubevirt/issues/17792)
		//
		// Workaround: Force the symlink's owner to show up as UID 0.
		// This is safe because the `serviceaccount` mount point is read-only.
		mapUidToGuestRoot := fmt.Sprintf("--translate-uid=host:%d:0:1", util.NonRootUID)
		args = append(args, mapUidToGuestRoot)
	}

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
			SubPath:   virtioFS.SubPath,
			ReadOnly:  virtioFS.ReadOnly,
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
