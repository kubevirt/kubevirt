package utils

import (
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	k8sv1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	cdiv1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1alpha1"
	cdiclientset "kubevirt.io/containerized-data-importer/pkg/client/clientset/versioned"
)

const (
	dataVolumePollInterval = 3 * time.Second
	dataVolumeCreateTime   = 60 * time.Second
	dataVolumeDeleteTime   = 60 * time.Second
	dataVolumePhaseTime    = 60 * time.Second
)

const (
	// TinyCoreIsoURL provides a test url for the tineyCore iso image
	TinyCoreIsoURL = "http://cdi-file-host.cdi/tinyCore.iso"
	//TinyCoreIsoRegistryURL provides a test url for the tinycore.qcow2 image wrapped in docker container
	TinyCoreIsoRegistryURL = "docker://cdi-docker-registry-host.cdi/tinycoreqcow2"
	// HTTPSTinyCoreIsoURL provides a test (https) url for the tineyCore iso image
	HTTPSTinyCoreIsoURL = "https://cdi-file-host.cdi/tinyCore.iso"
	// TinyCoreQcow2URLRateLimit provides a test url for the tineyCore iso image
	TinyCoreQcow2URLRateLimit = "http://cdi-file-host.cdi:82/tinyCore.qcow2"
)

// CreateDataVolumeFromDefinition is used by tests to create a testable Data Volume
func CreateDataVolumeFromDefinition(clientSet *cdiclientset.Clientset, namespace string, def *cdiv1.DataVolume) (*cdiv1.DataVolume, error) {
	var dataVolume *cdiv1.DataVolume
	err := wait.PollImmediate(dataVolumePollInterval, dataVolumeCreateTime, func() (bool, error) {
		var err error
		dataVolume, err = clientSet.CdiV1alpha1().DataVolumes(namespace).Create(def)
		if err == nil || apierrs.IsAlreadyExists(err) {
			return true, nil
		}
		return false, err
	})
	if err != nil {
		return nil, err
	}
	return dataVolume, nil
}

// DeleteDataVolume deletes the DataVolume with the given name
func DeleteDataVolume(clientSet *cdiclientset.Clientset, namespace, name string) error {
	return wait.PollImmediate(dataVolumePollInterval, dataVolumeDeleteTime, func() (bool, error) {
		err := clientSet.CdiV1alpha1().DataVolumes(namespace).Delete(name, nil)
		if err == nil || apierrs.IsNotFound(err) {
			return true, nil
		}
		return false, err
	})
}

// NewCloningDataVolume initializes a DataVolume struct with PVC annotations
func NewCloningDataVolume(dataVolumeName string, size string, sourcePvc *k8sv1.PersistentVolumeClaim) *cdiv1.DataVolume {
	return &cdiv1.DataVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: dataVolumeName,
		},
		Spec: cdiv1.DataVolumeSpec{
			Source: cdiv1.DataVolumeSource{
				PVC: &cdiv1.DataVolumeSourcePVC{
					Name:      sourcePvc.Name,
					Namespace: sourcePvc.Namespace,
				},
			},
			PVC: &k8sv1.PersistentVolumeClaimSpec{
				AccessModes: []k8sv1.PersistentVolumeAccessMode{k8sv1.ReadWriteOnce},
				Resources: k8sv1.ResourceRequirements{
					Requests: k8sv1.ResourceList{
						k8sv1.ResourceName(k8sv1.ResourceStorage): resource.MustParse(size),
					},
				},
			},
		},
	}
}

// NewDataVolumeWithHTTPImport initializes a DataVolume struct with HTTP annotations
func NewDataVolumeWithHTTPImport(dataVolumeName string, size string, httpURL string) *cdiv1.DataVolume {
	return &cdiv1.DataVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: dataVolumeName,
		},
		Spec: cdiv1.DataVolumeSpec{
			Source: cdiv1.DataVolumeSource{
				HTTP: &cdiv1.DataVolumeSourceHTTP{
					URL: httpURL,
				},
			},
			PVC: &k8sv1.PersistentVolumeClaimSpec{
				AccessModes: []k8sv1.PersistentVolumeAccessMode{k8sv1.ReadWriteOnce},
				Resources: k8sv1.ResourceRequirements{
					Requests: k8sv1.ResourceList{
						k8sv1.ResourceName(k8sv1.ResourceStorage): resource.MustParse(size),
					},
				},
			},
		},
	}
}

// NewDataVolumeWithHTTPImportToBlockPV initializes a DataVolume struct with HTTP annotations to import to block PV
func NewDataVolumeWithHTTPImportToBlockPV(dataVolumeName string, size string, httpURL string) *cdiv1.DataVolume {
	volumeMode := corev1.PersistentVolumeMode(corev1.PersistentVolumeBlock)
	storageClassName := "manual"
	dataVolume := &cdiv1.DataVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: dataVolumeName,
		},
		Spec: cdiv1.DataVolumeSpec{
			Source: cdiv1.DataVolumeSource{
				HTTP: &cdiv1.DataVolumeSourceHTTP{
					URL: httpURL,
				},
			},
			PVC: &k8sv1.PersistentVolumeClaimSpec{
				VolumeMode:       &volumeMode,
				StorageClassName: &storageClassName,
				AccessModes:      []k8sv1.PersistentVolumeAccessMode{k8sv1.ReadWriteOnce},
				Resources: k8sv1.ResourceRequirements{
					Requests: k8sv1.ResourceList{
						k8sv1.ResourceName(k8sv1.ResourceStorage): resource.MustParse(size),
					},
				},
			},
		},
	}
	return dataVolume
}

// NewDataVolumeForUpload initializes a DataVolume struct with Upload annotations
func NewDataVolumeForUpload(dataVolumeName string, size string) *cdiv1.DataVolume {
	return &cdiv1.DataVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: dataVolumeName,
		},
		Spec: cdiv1.DataVolumeSpec{
			Source: cdiv1.DataVolumeSource{
				Upload: &cdiv1.DataVolumeSourceUpload{},
			},
			PVC: &k8sv1.PersistentVolumeClaimSpec{
				AccessModes: []k8sv1.PersistentVolumeAccessMode{k8sv1.ReadWriteOnce},
				Resources: k8sv1.ResourceRequirements{
					Requests: k8sv1.ResourceList{
						k8sv1.ResourceName(k8sv1.ResourceStorage): resource.MustParse(size),
					},
				},
			},
		},
	}
}

// NewDataVolumeForBlankRawImage initializes a DataVolume struct for creating blank raw image
func NewDataVolumeForBlankRawImage(dataVolumeName, size string) *cdiv1.DataVolume {
	return &cdiv1.DataVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: dataVolumeName,
		},
		Spec: cdiv1.DataVolumeSpec{
			Source: cdiv1.DataVolumeSource{
				Blank: &cdiv1.DataVolumeBlankImage{},
			},
			PVC: &k8sv1.PersistentVolumeClaimSpec{
				AccessModes: []k8sv1.PersistentVolumeAccessMode{k8sv1.ReadWriteOnce},
				Resources: k8sv1.ResourceRequirements{
					Requests: k8sv1.ResourceList{
						k8sv1.ResourceName(k8sv1.ResourceStorage): resource.MustParse(size),
					},
				},
			},
		},
	}
}

// NewDataVolumeForImageCloning initializes a DataVolume struct for cloning disk image
func NewDataVolumeForImageCloning(dataVolumeName, size string, namespace, pvcName string) *cdiv1.DataVolume {
	return &cdiv1.DataVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: dataVolumeName,
		},
		Spec: cdiv1.DataVolumeSpec{
			Source: cdiv1.DataVolumeSource{
				PVC: &cdiv1.DataVolumeSourcePVC{
					Namespace: namespace,
					Name:      pvcName,
				},
			},
			PVC: &k8sv1.PersistentVolumeClaimSpec{
				AccessModes: []k8sv1.PersistentVolumeAccessMode{k8sv1.ReadWriteOnce},
				Resources: k8sv1.ResourceRequirements{
					Requests: k8sv1.ResourceList{
						k8sv1.ResourceName(k8sv1.ResourceStorage): resource.MustParse(size),
					},
				},
			},
		},
	}
}

// NewDataVolumeWithRegistryImport initializes a DataVolume struct with registry annotations
func NewDataVolumeWithRegistryImport(dataVolumeName string, size string, registryURL string) *cdiv1.DataVolume {
	return &cdiv1.DataVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: dataVolumeName,
		},
		Spec: cdiv1.DataVolumeSpec{
			Source: cdiv1.DataVolumeSource{
				Registry: &cdiv1.DataVolumeSourceRegistry{
					URL: registryURL,
				},
			},
			PVC: &k8sv1.PersistentVolumeClaimSpec{
				AccessModes: []k8sv1.PersistentVolumeAccessMode{k8sv1.ReadWriteOnce},
				Resources: k8sv1.ResourceRequirements{
					Requests: k8sv1.ResourceList{
						k8sv1.ResourceName(k8sv1.ResourceStorage): resource.MustParse(size),
					},
				},
			},
		},
	}
}

// WaitForDataVolumePhase waits for DV's phase to be in a particular phase (Pending, Bound, or Lost)
func WaitForDataVolumePhase(clientSet *cdiclientset.Clientset, namespace string, phase cdiv1.DataVolumePhase, dataVolumeName string) error {
	err := wait.PollImmediate(dataVolumePollInterval, dataVolumePhaseTime, func() (bool, error) {
		dataVolume, err := clientSet.CdiV1alpha1().DataVolumes(namespace).Get(dataVolumeName, metav1.GetOptions{})
		if err != nil || dataVolume.Status.Phase != phase {
			return false, err
		}
		return true, nil
	})
	if err != nil {
		return fmt.Errorf("DataVolume %s not in phase %s within %v", dataVolumeName, phase, dataVolumePhaseTime)
	}
	return nil
}
