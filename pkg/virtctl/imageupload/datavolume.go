package imageupload

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"
)

func (c *command) createUploadDataVolume() (*cdiv1.DataVolume, error) {
	pvcSpec, err := c.createStorageSpec()
	if err != nil {
		return nil, err
	}

	// We check if the user-defined storageClass exists before attempting to create the dataVolume
	if c.storageClass != "" {
		_, err = c.client.StorageV1().StorageClasses().Get(context.Background(), c.storageClass, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
	}

	annotations := map[string]string{}
	if c.forceBind {
		annotations[forceImmediateBindingAnnotation] = ""
	}

	contentType := cdiv1.DataVolumeKubeVirt
	if c.archiveUpload {
		contentType = cdiv1.DataVolumeArchive
	}

	dv := &cdiv1.DataVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:        c.name,
			Namespace:   c.namespace,
			Annotations: annotations,
		},
		Spec: cdiv1.DataVolumeSpec{
			Source: &cdiv1.DataVolumeSource{
				Upload: &cdiv1.DataVolumeSourceUpload{},
			},
			ContentType: contentType,
			Storage:     pvcSpec,
		},
	}
	c.setDefaultInstancetypeLabels(&dv.ObjectMeta)

	dv, err = c.client.CdiClient().CdiV1beta1().DataVolumes(c.namespace).Create(context.Background(), dv, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	return dv, nil
}

func (c *command) validateUploadDataVolume(pvc *v1.PersistentVolumeClaim) (*v1.PersistentVolumeClaim, error) {
	dv, err := c.client.CdiClient().CdiV1beta1().DataVolumes(pvc.Namespace).Get(context.Background(), c.name, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			// If the DataVolume doesn't exist, the PVC was created independently of a DV.
			return nil, fmt.Errorf("No DataVolume is associated with the existing PVC %s/%s", pvc.Namespace, c.name)
		}
		return nil, err
	}

	// When using populators, the upload happens on the PVC Prime. We need to check it instead.
	if dv.Annotations[usePopulatorAnnotation] == "true" {
		// We can assume the PVC is populated once it's bound
		if pvc.Status.Phase == v1.ClaimBound {
			return nil, fmt.Errorf("PVC %s already successfully populated", c.name)
		}
		// Get the PVC Prime since the upload is happening there
		pvcPrimeName, ok := pvc.Annotations[pvcPrimeNameAnnotation]
		if !ok {
			return nil, fmt.Errorf("Unable to get PVC Prime name from PVC %s/%s", pvc.Namespace, c.name)
		}
		pvc, err = c.client.CoreV1().PersistentVolumeClaims(dv.Namespace).Get(context.Background(), pvcPrimeName, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("Unable to get PVC Prime %s/%s", dv.Namespace, c.name)
		}
	}

	return pvc, nil
}
