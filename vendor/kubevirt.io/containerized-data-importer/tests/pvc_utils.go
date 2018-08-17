package tests

import (
	"time"

	. "github.com/onsi/ginkgo"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"k8s.io/client-go/kubernetes"
)

// Creates a PVC in the passed in namespace from the passed in PersistentVolumeClaim definition.
// An example of creating a PVC without annotations looks like this:
// CreatePVCFromDefinition(client, namespace, NewPVCDefinition(name, size, nil))
func CreatePVCFromDefinition(client *kubernetes.Clientset, namespace string, def *k8sv1.PersistentVolumeClaim) *k8sv1.PersistentVolumeClaim {
	var pvc *k8sv1.PersistentVolumeClaim
	err := wait.PollImmediate(2*time.Second, defaultTimeout, func() (bool, error) {
		var err error
		pvc, err = client.CoreV1().PersistentVolumeClaims(namespace).Create(def)
		if err != nil {
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		Fail("Unable to create PVC: " + def.GetName() + ", error: " + err.Error())
	}
	return pvc
}

// Delete the passed in PVC
func DeletePVC(client *kubernetes.Clientset, namespace string, pvc *k8sv1.PersistentVolumeClaim) {
	err := wait.PollImmediate(2*time.Second, defaultTimeout, func() (bool, error) {
		err := client.CoreV1().PersistentVolumeClaims(namespace).Delete(pvc.GetName(), nil)
		if err != nil {
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		Fail("Unable to delete PVC: " + pvc.GetName() + ", error: " + err.Error())
	}
}

// Creates a PVC definition using the passed in name and requested size.
// You can use the following annotation keys to request an import or clone. The values are defined in the controller package
// AnnEndpoint
// AnnSecret
// AnnCloneRequest
func NewPVCDefinition(name string, size string, annotations map[string]string) *k8sv1.PersistentVolumeClaim {
	return &k8sv1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Annotations: annotations,
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
