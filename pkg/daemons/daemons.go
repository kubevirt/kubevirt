package daemons

import (
	"fmt"
	"path/filepath"

	k8sv1 "k8s.io/api/core/v1"
	v1 "kubevirt.io/api/core/v1"
)

const (
	SourceDaemonsPath = "/run/kubevirt/daemons"
	SuffixDaemonPath  = "volumes/kubernetes.io~empty-dir/daemons"
)

const (
	PrHelperContainerName = "pr-helper"
	PrHelperImageName     = "pr-helper"
	PrHelperDir           = "pr"
	PrHelperSocket        = "pr-helper.sock"
	PrVolumeName          = "pr-socket-volume"
)

func GetPrHelperSocketDir() string {
	return fmt.Sprintf(filepath.Join(SourceDaemonsPath, PrHelperDir))
}

func GetPrHelperSocket() string {
	return fmt.Sprintf(filepath.Join(GetPrHelperSocketDir(), PrHelperSocket))
}

func RenderPrHelperContainer() k8sv1.Container {
	boolTrue := true
	bidi := k8sv1.MountPropagationBidirectional
	return k8sv1.Container{
		Name: PrHelperContainerName,
		// TODO afrosi make the image name configurable
		Image:   "registry:5000/kubevirt/pr-helper:devel",
		Command: []string{"/usr/bin/qemu-pr-helper"},
		Args: []string{
			"-k", GetPrHelperSocket(),
		},
		VolumeMounts: []k8sv1.VolumeMount{
			{
				Name:             PrVolumeName,
				MountPath:        GetPrHelperSocketDir(),
				MountPropagation: &bidi,
			},
		},
		SecurityContext: &k8sv1.SecurityContext{
			Privileged: &boolTrue,
		},
		Stdin: true,
	}
}

func IsPRHelperNeeded(vmi *v1.VirtualMachineInstance) bool {
	for _, disk := range vmi.Spec.Domain.Devices.Disks {
		if disk.DiskDevice.LUN != nil && disk.DiskDevice.LUN.Reservation {
			return true
		}
	}
	return false
}
