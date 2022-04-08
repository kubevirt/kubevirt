package services

import (
	k8sv1 "k8s.io/api/core/v1"
	"kubevirt.io/kubevirt/pkg/util"
	"path/filepath"
)

type VolumeRendererOption func(renderer *VolumeRenderer)

type VolumeRenderer struct {
	containerDiskDir string
	ephemeralDiskDir string
	virtShareDir     string
}

func NewVolumeRenderer(ephemeralDisk string, containerDiskDir string, virtShareDir string) *VolumeRenderer {
	return &VolumeRenderer{
		containerDiskDir: containerDiskDir,
		ephemeralDiskDir: ephemeralDisk,
		virtShareDir:     virtShareDir,
	}
}

func (vr *VolumeRenderer) Mounts() []k8sv1.VolumeMount {
	volumeMounts := []k8sv1.VolumeMount{
		mountPath("private", util.VirtPrivateDir),
		mountPath("public", util.VirtShareDir),
		mountPath("ephemeral-disks", vr.ephemeralDiskDir),
		mountPathWithPropagation(containerDisks, vr.containerDiskDir, k8sv1.MountPropagationHostToContainer),
		mountPath("libvirt-runtime", "/var/run/libvirt"),
		mountPath("sockets", filepath.Join(vr.virtShareDir, "sockets")),
	}
	return volumeMounts
}

func (vr *VolumeRenderer) Volumes() []k8sv1.Volume {
	volumes := []k8sv1.Volume{
		emptyDirVolume("private"),
		emptyDirVolume("public"),
		emptyDirVolume("sockets"),
	}
	return volumes
}

func mountPath(name string, path string) k8sv1.VolumeMount {
	return k8sv1.VolumeMount{
		Name:      name,
		MountPath: path,
	}
}

func mountPathWithPropagation(name string, path string, propagation k8sv1.MountPropagationMode) k8sv1.VolumeMount {
	return k8sv1.VolumeMount{
		Name:             name,
		MountPath:        path,
		MountPropagation: &propagation,
	}
}

func emptyDirVolume(name string) k8sv1.Volume {
	return k8sv1.Volume{
		Name: name,
		VolumeSource: k8sv1.VolumeSource{
			EmptyDir: &k8sv1.EmptyDirVolumeSource{}},
	}
}
