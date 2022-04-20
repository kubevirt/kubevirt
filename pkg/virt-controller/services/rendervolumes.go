package services

import (
	"fmt"
	"path/filepath"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/config"
	"kubevirt.io/kubevirt/pkg/hooks"
	hostdisk "kubevirt.io/kubevirt/pkg/host-disk"
	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/util/types"
)

type VolumeRendererOption func(renderer *VolumeRenderer) error

type VolumeRenderer struct {
	containerDiskDir string
	ephemeralDiskDir string
	virtShareDir     string
	namespace        string
	vmiVolumes       []v1.Volume
	podVolumes       []k8sv1.Volume
	podVolumeMounts  []k8sv1.VolumeMount
	volumeDevices    []k8sv1.VolumeDevice
}

func NewVolumeRenderer(namespace string, ephemeralDisk string, containerDiskDir string, virtShareDir string, volumeOptions ...VolumeRendererOption) (*VolumeRenderer, error) {
	volumeRenderer := &VolumeRenderer{
		containerDiskDir: containerDiskDir,
		ephemeralDiskDir: ephemeralDisk,
		namespace:        namespace,
		virtShareDir:     virtShareDir,
	}
	for _, volumeOption := range volumeOptions {
		if err := volumeOption(volumeRenderer); err != nil {
			return nil, err
		}
	}
	return volumeRenderer, nil
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
	return append(volumeMounts, vr.podVolumeMounts...)
}

func (vr *VolumeRenderer) Volumes() []k8sv1.Volume {
	volumes := []k8sv1.Volume{
		emptyDirVolume("private"),
		emptyDirVolume("public"),
		emptyDirVolume("sockets"),
	}
	return append(volumes, vr.podVolumes...)
}

func (vr *VolumeRenderer) VolumeDevices() []k8sv1.VolumeDevice {
	return vr.volumeDevices
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

func withVMIVolumes(pvcStore cache.Store, vmiSpecVolumes []v1.Volume, vmiVolumeStatus []v1.VolumeStatus) VolumeRendererOption {
	return func(renderer *VolumeRenderer) error {
		hotplugVolumes := make(map[string]bool)
		for _, volumeStatus := range vmiVolumeStatus {
			if volumeStatus.HotplugVolume != nil {
				hotplugVolumes[volumeStatus.Name] = true
			}
		}
		// This detects hotplug volumes for a started but not ready VMI
		for _, volume := range vmiSpecVolumes {
			if (volume.DataVolume != nil && volume.DataVolume.Hotpluggable) || (volume.PersistentVolumeClaim != nil && volume.PersistentVolumeClaim.Hotpluggable) {
				hotplugVolumes[volume.Name] = true
			}
		}

		for _, volume := range vmiSpecVolumes {
			if hotplugVolumes[volume.Name] {
				continue
			}
			if volume.PersistentVolumeClaim != nil {
				claimName := volume.PersistentVolumeClaim.ClaimName
				if err := addPVCToLaunchManifest(pvcStore, volume, claimName, renderer.namespace, &renderer.podVolumeMounts, &renderer.volumeDevices); err != nil {
					return err
				}
				renderer.podVolumes = append(renderer.podVolumes, k8sv1.Volume{
					Name: volume.Name,
					VolumeSource: k8sv1.VolumeSource{
						PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: volume.PersistentVolumeClaim.ClaimName,
							ReadOnly:  volume.PersistentVolumeClaim.ReadOnly,
						},
					},
				})
			}
			if volume.Ephemeral != nil {
				claimName := volume.Ephemeral.PersistentVolumeClaim.ClaimName
				if err := addPVCToLaunchManifest(pvcStore, volume, claimName, renderer.namespace, &renderer.podVolumeMounts, &renderer.volumeDevices); err != nil {
					return err
				}
				renderer.podVolumes = append(renderer.podVolumes, k8sv1.Volume{
					Name: volume.Name,
					VolumeSource: k8sv1.VolumeSource{
						PersistentVolumeClaim: volume.Ephemeral.PersistentVolumeClaim,
					},
				})
			}
			imgPullSecrets(volume)
			if volume.HostDisk != nil {
				var hostPathType k8sv1.HostPathType

				switch hostType := volume.HostDisk.Type; hostType {
				case v1.HostDiskExists:
					hostPathType = k8sv1.HostPathDirectory
				case v1.HostDiskExistsOrCreate:
					hostPathType = k8sv1.HostPathDirectoryOrCreate
				}

				renderer.podVolumeMounts = append(renderer.podVolumeMounts, k8sv1.VolumeMount{
					Name:      volume.Name,
					MountPath: hostdisk.GetMountedHostDiskDir(volume.Name),
				})
				renderer.podVolumes = append(renderer.podVolumes, k8sv1.Volume{
					Name: volume.Name,
					VolumeSource: k8sv1.VolumeSource{
						HostPath: &k8sv1.HostPathVolumeSource{
							Path: filepath.Dir(volume.HostDisk.Path),
							Type: &hostPathType,
						},
					},
				})
			}
			if volume.DataVolume != nil {
				claimName := volume.DataVolume.Name
				if err := addPVCToLaunchManifest(pvcStore, volume, claimName, renderer.namespace, &renderer.podVolumeMounts, &renderer.volumeDevices); err != nil {
					return err
				}
				renderer.podVolumes = append(renderer.podVolumes, k8sv1.Volume{
					Name: volume.Name,
					VolumeSource: k8sv1.VolumeSource{
						PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: claimName,
						},
					},
				})
			}
			if volume.ConfigMap != nil {
				// attach a ConfigMap to the pod
				renderer.podVolumeMounts = append(renderer.podVolumeMounts, k8sv1.VolumeMount{
					Name:      volume.Name,
					MountPath: filepath.Join(config.ConfigMapSourceDir, volume.Name),
					ReadOnly:  true,
				})
				renderer.podVolumes = append(renderer.podVolumes, k8sv1.Volume{
					Name: volume.Name,
					VolumeSource: k8sv1.VolumeSource{
						ConfigMap: &k8sv1.ConfigMapVolumeSource{
							LocalObjectReference: volume.ConfigMap.LocalObjectReference,
							Optional:             volume.ConfigMap.Optional,
						},
					},
				})
			}

			if volume.Secret != nil {
				// attach a Secret to the pod
				renderer.podVolumeMounts = append(renderer.podVolumeMounts, k8sv1.VolumeMount{
					Name:      volume.Name,
					MountPath: filepath.Join(config.SecretSourceDir, volume.Name),
					ReadOnly:  true,
				})
				renderer.podVolumes = append(renderer.podVolumes, k8sv1.Volume{
					Name: volume.Name,
					VolumeSource: k8sv1.VolumeSource{
						Secret: &k8sv1.SecretVolumeSource{
							SecretName: volume.Secret.SecretName,
							Optional:   volume.Secret.Optional,
						},
					},
				})
			}

			if volume.DownwardAPI != nil {
				// attach a Secret to the pod
				renderer.podVolumeMounts = append(renderer.podVolumeMounts, k8sv1.VolumeMount{
					Name:      volume.Name,
					MountPath: filepath.Join(config.DownwardAPISourceDir, volume.Name),
					ReadOnly:  true,
				})
				renderer.podVolumes = append(renderer.podVolumes, k8sv1.Volume{
					Name: volume.Name,
					VolumeSource: k8sv1.VolumeSource{
						DownwardAPI: &k8sv1.DownwardAPIVolumeSource{
							Items: volume.DownwardAPI.Fields,
						},
					},
				})
			}

			if volume.DownwardMetrics != nil {
				sizeLimit := resource.MustParse("1Mi")
				renderer.podVolumes = append(renderer.podVolumes, k8sv1.Volume{
					Name: volume.Name,
					VolumeSource: k8sv1.VolumeSource{
						EmptyDir: &k8sv1.EmptyDirVolumeSource{
							Medium:    "Memory",
							SizeLimit: &sizeLimit,
						},
					},
				})
				renderer.podVolumeMounts = append(renderer.podVolumeMounts, k8sv1.VolumeMount{
					Name:      volume.Name,
					MountPath: config.DownwardMetricDisksDir,
				})
			}

			if volume.CloudInitNoCloud != nil {
				if volume.CloudInitNoCloud.UserDataSecretRef != nil {
					// attach a secret referenced by the user
					volumeName := volume.Name + "-udata"
					renderer.podVolumes = append(renderer.podVolumes, k8sv1.Volume{
						Name: volumeName,
						VolumeSource: k8sv1.VolumeSource{
							Secret: &k8sv1.SecretVolumeSource{
								SecretName: volume.CloudInitNoCloud.UserDataSecretRef.Name,
							},
						},
					})
					renderer.podVolumeMounts = append(renderer.podVolumeMounts, k8sv1.VolumeMount{
						Name:      volumeName,
						MountPath: filepath.Join(config.SecretSourceDir, volume.Name, "userdata"),
						SubPath:   "userdata",
						ReadOnly:  true,
					})
					renderer.podVolumeMounts = append(renderer.podVolumeMounts, k8sv1.VolumeMount{
						Name:      volumeName,
						MountPath: filepath.Join(config.SecretSourceDir, volume.Name, "userData"),
						SubPath:   "userData",
						ReadOnly:  true,
					})
				}
				if volume.CloudInitNoCloud.NetworkDataSecretRef != nil {
					// attach a secret referenced by the networkdata
					volumeName := volume.Name + "-ndata"
					renderer.podVolumes = append(renderer.podVolumes, k8sv1.Volume{
						Name: volumeName,
						VolumeSource: k8sv1.VolumeSource{
							Secret: &k8sv1.SecretVolumeSource{
								SecretName: volume.CloudInitNoCloud.NetworkDataSecretRef.Name,
							},
						},
					})
					renderer.podVolumeMounts = append(renderer.podVolumeMounts, k8sv1.VolumeMount{
						Name:      volumeName,
						MountPath: filepath.Join(config.SecretSourceDir, volume.Name, "networkdata"),
						SubPath:   "networkdata",
						ReadOnly:  true,
					})
					renderer.podVolumeMounts = append(renderer.podVolumeMounts, k8sv1.VolumeMount{
						Name:      volumeName,
						MountPath: filepath.Join(config.SecretSourceDir, volume.Name, "networkData"),
						SubPath:   "networkData",
						ReadOnly:  true,
					})
				}
			}

			if volume.Sysprep != nil {
				var volumeSource k8sv1.VolumeSource
				// attach a Secret or ConfigMap referenced by the user
				volumeSource, err := sysprepVolumeSource(*volume.Sysprep)
				if err != nil {
					//return nil, err
				}
				renderer.podVolumes = append(renderer.podVolumes, k8sv1.Volume{
					Name:         volume.Name,
					VolumeSource: volumeSource,
				})
				renderer.podVolumeMounts = append(renderer.podVolumeMounts, k8sv1.VolumeMount{
					Name:      volume.Name,
					MountPath: filepath.Join(config.SysprepSourceDir, volume.Name),
					ReadOnly:  true,
				})
			}

			if volume.CloudInitConfigDrive != nil {
				if volume.CloudInitConfigDrive.UserDataSecretRef != nil {
					// attach a secret referenced by the user
					volumeName := volume.Name + "-udata"
					renderer.podVolumes = append(renderer.podVolumes, k8sv1.Volume{
						Name: volumeName,
						VolumeSource: k8sv1.VolumeSource{
							Secret: &k8sv1.SecretVolumeSource{
								SecretName: volume.CloudInitConfigDrive.UserDataSecretRef.Name,
							},
						},
					})
					renderer.podVolumeMounts = append(renderer.podVolumeMounts, k8sv1.VolumeMount{
						Name:      volumeName,
						MountPath: filepath.Join(config.SecretSourceDir, volume.Name, "userdata"),
						SubPath:   "userdata",
						ReadOnly:  true,
					})
					renderer.podVolumeMounts = append(renderer.podVolumeMounts, k8sv1.VolumeMount{
						Name:      volumeName,
						MountPath: filepath.Join(config.SecretSourceDir, volume.Name, "userData"),
						SubPath:   "userData",
						ReadOnly:  true,
					})
				}
				if volume.CloudInitConfigDrive.NetworkDataSecretRef != nil {
					// attach a secret referenced by the networkdata
					volumeName := volume.Name + "-ndata"
					renderer.podVolumes = append(renderer.podVolumes, k8sv1.Volume{
						Name: volumeName,
						VolumeSource: k8sv1.VolumeSource{
							Secret: &k8sv1.SecretVolumeSource{
								SecretName: volume.CloudInitConfigDrive.NetworkDataSecretRef.Name,
							},
						},
					})
					renderer.podVolumeMounts = append(renderer.podVolumeMounts, k8sv1.VolumeMount{
						Name:      volumeName,
						MountPath: filepath.Join(config.SecretSourceDir, volume.Name, "networkdata"),
						SubPath:   "networkdata",
						ReadOnly:  true,
					})
					renderer.podVolumeMounts = append(renderer.podVolumeMounts, k8sv1.VolumeMount{
						Name:      volumeName,
						MountPath: filepath.Join(config.SecretSourceDir, volume.Name, "networkData"),
						SubPath:   "networkData",
						ReadOnly:  true,
					})
				}
			}
		}
		return nil
	}
}

func withAccessCredentials(accessCredentials []v1.AccessCredential) VolumeRendererOption {
	return func(renderer *VolumeRenderer) error {
		for _, accessCred := range accessCredentials {
			secretName := ""
			if accessCred.SSHPublicKey != nil && accessCred.SSHPublicKey.Source.Secret != nil {
				secretName = accessCred.SSHPublicKey.Source.Secret.SecretName
			} else if accessCred.UserPassword != nil && accessCred.UserPassword.Source.Secret != nil {
				secretName = accessCred.UserPassword.Source.Secret.SecretName
			}

			if secretName == "" {
				continue
			}
			volumeName := secretName + "-access-cred"
			renderer.podVolumes = append(renderer.podVolumes, k8sv1.Volume{
				Name: volumeName,
				VolumeSource: k8sv1.VolumeSource{
					Secret: &k8sv1.SecretVolumeSource{
						SecretName: secretName,
					},
				},
			})
			renderer.podVolumeMounts = append(renderer.podVolumeMounts, k8sv1.VolumeMount{
				Name:      volumeName,
				MountPath: filepath.Join(config.SecretSourceDir, volumeName),
				ReadOnly:  true,
			})
		}
		return nil
	}
}

func withSidecarVolumes(hookSidecars hooks.HookSidecarList) VolumeRendererOption {
	return func(renderer *VolumeRenderer) error {
		if len(hookSidecars) != 0 {
			renderer.podVolumes = append(renderer.podVolumes, k8sv1.Volume{
				Name: hookSidecarSocks,
				VolumeSource: k8sv1.VolumeSource{
					EmptyDir: &k8sv1.EmptyDirVolumeSource{},
				},
			})
			renderer.podVolumeMounts = append(renderer.podVolumeMounts, k8sv1.VolumeMount{
				Name:      hookSidecarSocks,
				MountPath: hooks.HookSocketsSharedDirectory,
			})
		}
		return nil
	}
}

func imgPullSecrets(volumes ...v1.Volume) []k8sv1.LocalObjectReference {
	var imagePullSecrets []k8sv1.LocalObjectReference
	for _, volume := range volumes {
		if volume.ContainerDisk != nil && volume.ContainerDisk.ImagePullSecret != "" {
			imagePullSecrets = appendUniqueImagePullSecret(imagePullSecrets, k8sv1.LocalObjectReference{
				Name: volume.ContainerDisk.ImagePullSecret,
			})
		}
	}
	return imagePullSecrets
}

func serviceAccount(volumes ...v1.Volume) string {
	for _, volume := range volumes {
		if volume.ServiceAccount != nil {
			return volume.ServiceAccount.ServiceAccountName
		}
	}
	return ""
}

func addPVCToLaunchManifest(pvcStore cache.Store, volume v1.Volume, claimName string, namespace string, volumeMounts *[]k8sv1.VolumeMount, volumeDevices *[]k8sv1.VolumeDevice) error {
	logger := log.DefaultLogger()
	_, exists, isBlock, err := types.IsPVCBlockFromStore(pvcStore, namespace, claimName)
	if err != nil {
		logger.Errorf("error getting PVC: %v", claimName)
		return err
	} else if !exists {
		logger.Errorf("didn't find PVC %v", claimName)
		return PvcNotFoundError{Reason: fmt.Sprintf("didn't find PVC %v", claimName)}
	} else if isBlock {
		devicePath := filepath.Join(string(filepath.Separator), "dev", volume.Name)
		device := k8sv1.VolumeDevice{
			Name:       volume.Name,
			DevicePath: devicePath,
		}
		*volumeDevices = append(*volumeDevices, device)
	} else {
		volumeMount := k8sv1.VolumeMount{
			Name:      volume.Name,
			MountPath: hostdisk.GetMountedHostDiskDir(volume.Name),
		}
		*volumeMounts = append(*volumeMounts, volumeMount)
	}
	return nil
}
