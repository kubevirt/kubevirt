package utils

import (
	"fmt"
	"time"

	k8sv1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	cdiv1 "kubevirt.io/containerized-data-importer/pkg/apis/datavolumecontroller/v1alpha1"
	cdiclientset "kubevirt.io/containerized-data-importer/pkg/client/clientset/versioned"
)

const (
	dataVolumePollInterval = 2 * time.Second
	DataVolumeCreateTime   = 30 * time.Second
	DataVolumeDeleteTime   = 30 * time.Second
	DataVolumePhaseTime    = 30 * time.Second
)

const (
	TinyCoreIsoURL = "http://cdi-file-host.kube-system/tinyCore.iso"
)

func CreateDataVolumeFromDefinition(clientSet *cdiclientset.Clientset, namespace string, def *cdiv1.DataVolume) (*cdiv1.DataVolume, error) {
	var dataVolume *cdiv1.DataVolume
	err := wait.PollImmediate(dataVolumePollInterval, DataVolumeCreateTime, func() (bool, error) {
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

// Delete the passed in DataVolume
func DeleteDataVolume(clientSet *cdiclientset.Clientset, namespace string, dataVolume *cdiv1.DataVolume) error {
	return wait.PollImmediate(dataVolumePollInterval, DataVolumeDeleteTime, func() (bool, error) {
		err := clientSet.CdiV1alpha1().DataVolumes(namespace).Delete(dataVolume.GetName(), nil)
		if err == nil || apierrs.IsNotFound(err) {
			return true, nil
		}
		return false, err
	})
}

func NewDataVolumeWithPVCImport(dataVolumeName string, size string, targetPvc *k8sv1.PersistentVolumeClaim) *cdiv1.DataVolume {
	return &cdiv1.DataVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: dataVolumeName,
		},
		Spec: cdiv1.DataVolumeSpec{
			Source: cdiv1.DataVolumeSource{
				PVC: &cdiv1.DataVolumeSourcePVC{
					Name:      targetPvc.Name,
					Namespace: targetPvc.Namespace,
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

func NewDataVolumeWithHttpImport(dataVolumeName string, size string, httpUrl string) *cdiv1.DataVolume {
	return &cdiv1.DataVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: dataVolumeName,
		},
		Spec: cdiv1.DataVolumeSpec{
			Source: cdiv1.DataVolumeSource{
				HTTP: &cdiv1.DataVolumeSourceHTTP{
					URL: httpUrl,
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

// Wait for the DataVolume to be in a particular phase (Pending, Bound, or Lost)
func WaitForDataVolumePhase(clientSet *cdiclientset.Clientset, namespace string, phase cdiv1.DataVolumePhase, dataVolumeName string) error {
	err := wait.PollImmediate(dataVolumePollInterval, DataVolumePhaseTime, func() (bool, error) {
		dataVolume, err := clientSet.CdiV1alpha1().DataVolumes(namespace).Get(dataVolumeName, metav1.GetOptions{})
		if err != nil || dataVolume.Status.Phase != phase {
			return false, err
		}
		return true, nil
	})
	if err != nil {
		return fmt.Errorf("DataVolume %s not in phase %s within %v", dataVolumeName, phase, DataVolumePhaseTime)
	}
	return nil
}
