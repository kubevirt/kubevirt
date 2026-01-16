package services

import (
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/tpm"

	k8sv1 "k8s.io/api/core/v1"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/util"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

const (
	restoreSourceUIDAnnotation = "restore.kubevirt.io/source-uid"
	pvcMountPath               = "/var/lib/tpm"
	swtpmBase                  = pvcMountPath + "/swtpm"
)

func generateTPMCloneHandlerInitContainer(
	vmi *v1.VirtualMachineInstance,
	image string,
	config *virtconfig.ClusterConfig,
) *k8sv1.Container {

	if !tpm.HasPersistentDevice(&vmi.Spec) {
		return nil
	}
	if vmi.Annotations == nil {
		return nil
	}

	oldUUID, ok := vmi.Annotations[restoreSourceUIDAnnotation]
	if !ok || oldUUID == "" {
		return nil
	}

	newUUID := string(vmi.Spec.Domain.Firmware.UUID)

	return &k8sv1.Container{
		Name:            "tpm-clone",
		Image:           image,
		ImagePullPolicy: config.GetImagePullPolicy(),
		Command:         []string{"tpm-clone-handler.sh"},
		Env: []k8sv1.EnvVar{
			{
				Name:  "OLD_UUID",
				Value: oldUUID,
			},
			{
				Name:  "NEW_UUID",
				Value: newUUID,
			},
			{
				Name:  "BASE",
				Value: swtpmBase,
			},
		},
		VolumeMounts: []k8sv1.VolumeMount{
			{
				Name:      "vm-state",
				MountPath: pvcMountPath,
				ReadOnly:  false,
			},
		},
		SecurityContext: &k8sv1.SecurityContext{
			RunAsUser:                pointer.P(int64(util.NonRootUID)),
			RunAsNonRoot:             pointer.P(true),
			AllowPrivilegeEscalation: pointer.P(false),
			Capabilities: &k8sv1.Capabilities{
				Drop: []k8sv1.Capability{"ALL"},
			},
		},
		RestartPolicy: pointer.P(k8sv1.ContainerRestartPolicyNever),
	}
}
