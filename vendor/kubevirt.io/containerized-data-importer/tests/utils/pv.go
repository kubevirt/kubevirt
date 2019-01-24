package utils

import (
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	//"time"
	"fmt"
	"github.com/onsi/ginkgo"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

const (
	pvPollInterval = defaultPollInterval
	pvCreateTime   = defaultPollPeriod
	pvDeleteTime   = defaultPollPeriod
)

// CreatePVFromDefinition creates a PV from the passed in PersistentVolume definition.
// An example of creating a PVC without annotations looks like this:
// CreatePVCFromDefinition(client, namespace, NewPVCDefinition(name, size, nil, nil))
func CreatePVFromDefinition(clientSet *kubernetes.Clientset, def *k8sv1.PersistentVolume) (*k8sv1.PersistentVolume, error) {
	var pv *k8sv1.PersistentVolume
	err := wait.PollImmediate(pvPollInterval, pvCreateTime, func() (bool, error) {
		var err error
		pv, err = clientSet.CoreV1().PersistentVolumes().Create(def)
		if err == nil || apierrs.IsAlreadyExists(err) {
			return true, nil
		}
		return false, err
	})
	if err != nil {
		return nil, err
	}
	return pv, nil
}

// NewPVDefinition creates a PV definition.
func NewPVDefinition(pvName string, size string, labels map[string]string, storageClassName string) *k8sv1.PersistentVolume {
	return &k8sv1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:   pvName,
			Labels: labels,
		},
		Spec: k8sv1.PersistentVolumeSpec{
			AccessModes:                   []k8sv1.PersistentVolumeAccessMode{k8sv1.ReadWriteOnce},
			PersistentVolumeReclaimPolicy: k8sv1.PersistentVolumeReclaimDelete,
			Capacity: k8sv1.ResourceList{
				k8sv1.ResourceName(k8sv1.ResourceStorage): resource.MustParse(size),
			},
			StorageClassName: storageClassName,
			PersistentVolumeSource: k8sv1.PersistentVolumeSource{
				Local: &k8sv1.LocalVolumeSource{
					Path: "/mnt/local-storage/local/disk2",
				},
			},
			//PersistentVolumeSource: "local",
			NodeAffinity: &k8sv1.VolumeNodeAffinity{
				Required: &k8sv1.NodeSelector{
					NodeSelectorTerms: []k8sv1.NodeSelectorTerm{
						{
							MatchExpressions: []k8sv1.NodeSelectorRequirement{
								{
									Key:      "kubernetes.io/hostname",
									Operator: k8sv1.NodeSelectorOpIn,
									Values:   []string{"node01"},
								},
							},
						},
					},
				},
			},
		},
	}
}

// WaitTimeoutForPVReady waits for the given pv to be created and ready
func WaitTimeoutForPVReady(clientSet *kubernetes.Clientset, pvName string, timeout time.Duration) error {
	return WaitTimeoutForPVStatus(clientSet, pvName, k8sv1.VolumeAvailable, timeout)
}

// WaitTimeoutForPVStatus waits for the given pv to be created and have a expected status
func WaitTimeoutForPVStatus(clientSet *kubernetes.Clientset, pvName string, status k8sv1.PersistentVolumePhase, timeout time.Duration) error {
	return wait.PollImmediate(2*time.Second, timeout, pvPhase(clientSet, pvName, status))
}

func pvPhase(clientSet *kubernetes.Clientset, pvName string, status k8sv1.PersistentVolumePhase) wait.ConditionFunc {
	return func() (bool, error) {
		pv, err := clientSet.CoreV1().PersistentVolumes().Get(pvName, metav1.GetOptions{})
		if err != nil {
			if k8serrors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		fmt.Fprintf(ginkgo.GinkgoWriter, "INFO: Checking PV phase: %s\n", string(pv.Status.Phase))
		switch pv.Status.Phase {
		case status:
			return true, nil
		}
		return false, nil
	}
}

// DeletePV deletes the passed in PV
func DeletePV(clientSet *kubernetes.Clientset, pv *k8sv1.PersistentVolume) error {
	return wait.PollImmediate(pvPollInterval, pvDeleteTime, func() (bool, error) {
		err := clientSet.CoreV1().PersistentVolumes().Delete(pv.GetName(), nil)
		if err == nil || apierrs.IsNotFound(err) {
			return true, nil
		}
		return false, err
	})
}
