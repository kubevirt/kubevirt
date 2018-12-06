package utils

import (
	"time"

	"fmt"

	"github.com/onsi/ginkgo"
	k8sv1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

const (
	defaultPollInterval = 2 * time.Second
	defaultPollPeriod   = 60 * time.Second

	// DefaultPvcMountPath is the default mount path used
	DefaultPvcMountPath = "/pvc"

	// DefaultImagePath is the default destination for images created by CDI
	DefaultImagePath = DefaultPvcMountPath + "/disk.img"

	pvcPollInterval = defaultPollInterval
	pvcCreateTime   = defaultPollPeriod
	pvcDeleteTime   = defaultPollPeriod
	pvcPhaseTime    = defaultPollPeriod
)

// CreatePVCFromDefinition creates a PVC in the passed in namespace from the passed in PersistentVolumeClaim definition.
// An example of creating a PVC without annotations looks like this:
// CreatePVCFromDefinition(client, namespace, NewPVCDefinition(name, size, nil, nil))
func CreatePVCFromDefinition(clientSet *kubernetes.Clientset, namespace string, def *k8sv1.PersistentVolumeClaim) (*k8sv1.PersistentVolumeClaim, error) {
	var pvc *k8sv1.PersistentVolumeClaim
	err := wait.PollImmediate(pvcPollInterval, pvcCreateTime, func() (bool, error) {
		var err error
		pvc, err = clientSet.CoreV1().PersistentVolumeClaims(namespace).Create(def)
		if err == nil || apierrs.IsAlreadyExists(err) {
			return true, nil
		}
		return false, err
	})
	if err != nil {
		return nil, err
	}
	return pvc, nil
}

// DeletePVC deletes the passed in PVC
func DeletePVC(clientSet *kubernetes.Clientset, namespace string, pvc *k8sv1.PersistentVolumeClaim) error {
	return wait.PollImmediate(pvcPollInterval, pvcDeleteTime, func() (bool, error) {
		err := clientSet.CoreV1().PersistentVolumeClaims(namespace).Delete(pvc.GetName(), nil)
		if err == nil || apierrs.IsNotFound(err) {
			return true, nil
		}
		return false, err
	})
}

// FindPVC Finds the passed in PVC
func FindPVC(clientSet *kubernetes.Clientset, namespace, pvcName string) (*k8sv1.PersistentVolumeClaim, error) {
	return clientSet.CoreV1().PersistentVolumeClaims(namespace).Get(pvcName, metav1.GetOptions{})
}

// WaitForPVCAnnotation waits for an anotation to appear on a PVC
func WaitForPVCAnnotation(clientSet *kubernetes.Clientset, namespace string, pvc *k8sv1.PersistentVolumeClaim, annotation string) (string, bool, error) {
	var result string
	var called bool
	err := pollPVCAnnotation(clientSet, namespace, pvc, annotation, func(value string) bool {
		called = true
		result = value
		return true
	})
	return result, called, err
}

// WaitForPVCAnnotationWithValue waits  for an annotation with a specific value on a PVC
func WaitForPVCAnnotationWithValue(clientSet *kubernetes.Clientset, namespace string, pvc *k8sv1.PersistentVolumeClaim, annotation, expected string) (bool, error) {
	var result bool
	err := pollPVCAnnotation(clientSet, namespace, pvc, annotation, func(value string) bool {
		result = (expected == value)
		return result
	})
	return result, err
}

type pollPVCAnnotationFunc = func(string) bool

func pollPVCAnnotation(clientSet *kubernetes.Clientset, namespace string, pvc *k8sv1.PersistentVolumeClaim, annotation string, f pollPVCAnnotationFunc) error {
	err := wait.PollImmediate(pvcPollInterval, pvcCreateTime, func() (bool, error) {
		pvc, err := FindPVC(clientSet, namespace, pvc.Name)
		if err != nil {
			return false, err
		}
		fmt.Fprintf(ginkgo.GinkgoWriter, "INFO: PVC annotations: %v\n", pvc.ObjectMeta.Annotations)
		value, ok := pvc.ObjectMeta.Annotations[annotation]
		if ok {
			return f(value), nil
		}
		return false, err
	})
	return err
}

// NewPVCDefinition creates a PVC definition using the passed in name and requested size.
// You can use the following annotation keys to request an import or clone. The values are defined in the controller package
// AnnEndpoint
// AnnSecret
// AnnCloneRequest
// You can also pass in any label you want.
func NewPVCDefinition(pvcName string, size string, annotations, labels map[string]string) *k8sv1.PersistentVolumeClaim {
	return &k8sv1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:        pvcName,
			Annotations: annotations,
			Labels:      labels,
		},
		Spec: k8sv1.PersistentVolumeClaimSpec{
			AccessModes: []k8sv1.PersistentVolumeAccessMode{k8sv1.ReadWriteOnce},
			Resources: k8sv1.ResourceRequirements{
				Requests: k8sv1.ResourceList{
					k8sv1.ResourceName(k8sv1.ResourceStorage): resource.MustParse(size),
				},
			},
		},
	}
}

// WaitForPersistentVolumeClaimPhase waits for the PVC to be in a particular phase (Pending, Bound, or Lost)
func WaitForPersistentVolumeClaimPhase(clientSet *kubernetes.Clientset, namespace string, phase k8sv1.PersistentVolumeClaimPhase, pvcName string) error {
	err := wait.PollImmediate(pvcPollInterval, pvcPhaseTime, func() (bool, error) {
		pvc, err := clientSet.CoreV1().PersistentVolumeClaims(namespace).Get(pvcName, metav1.GetOptions{})
		fmt.Fprintf(ginkgo.GinkgoWriter, "INFO: Checking PVC phase: %s\n", string(pvc.Status.Phase))
		if err != nil || pvc.Status.Phase != phase {
			return false, err
		}
		return true, nil
	})
	if err != nil {
		return fmt.Errorf("PersistentVolumeClaim %s not in phase %s within %v", pvcName, phase, pvcPhaseTime)
	}
	return nil
}

// WaitPVCDeleted polls the specified PVC until timeout or it's not found, returns true if deleted in the specified timeout period, and any errors
func WaitPVCDeleted(clientSet *kubernetes.Clientset, pvcName, namespace string, timeout time.Duration) (bool, error) {
	var result bool
	err := wait.PollImmediate(2*time.Second, timeout, func() (bool, error) {
		_, err := clientSet.CoreV1().PersistentVolumeClaims(namespace).Get(pvcName, metav1.GetOptions{})
		if err != nil {
			if k8serrors.IsNotFound(err) {
				result = true
				return true, nil
			}
			return false, err
		}
		return false, nil
	})
	return result, err
}
