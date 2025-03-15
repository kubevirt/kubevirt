package imageupload

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/pointer"
	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
)

func (c *command) createUploadPVC() (*v1.PersistentVolumeClaim, error) {
	if c.accessMode == string(v1.ReadOnlyMany) {
		return nil, fmt.Errorf("cannot upload to a readonly volume, use either ReadWriteOnce or ReadWriteMany if supported")
	}

	// We check if the user-defined storageClass exists before attempting to create the PVC
	if c.storageClass != "" {
		_, err := c.client.StorageV1().StorageClasses().Get(context.Background(), c.storageClass, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
	}

	quantity, err := resource.ParseQuantity(c.size)
	if err != nil {
		return nil, fmt.Errorf("validation failed for size=%s: %s", c.size, err)
	}
	pvc := storagetypes.RenderPVC(&quantity, c.name, c.namespace, c.storageClass, c.accessMode, c.volumeMode == "block")
	if c.volumeMode == "filesystem" {
		pvc.Spec.VolumeMode = pointer.P(v1.PersistentVolumeFilesystem)
	}

	contentType := string(cdiv1.DataVolumeKubeVirt)
	if c.archiveUpload {
		contentType = string(cdiv1.DataVolumeArchive)
	}

	annotations := map[string]string{
		uploadRequestAnnotation: "",
		contentTypeAnnotation:   contentType,
	}

	if c.forceBind {
		annotations[forceImmediateBindingAnnotation] = ""
	}

	pvc.ObjectMeta.Annotations = annotations
	c.setDefaultInstancetypeLabels(&pvc.ObjectMeta)

	return c.client.CoreV1().PersistentVolumeClaims(c.namespace).Create(context.Background(), pvc, metav1.CreateOptions{})
}

func (c *command) ensurePVCSupportsUpload(pvc *v1.PersistentVolumeClaim) (*v1.PersistentVolumeClaim, error) {
	if pvc.GetAnnotations() == nil {
		pvc.SetAnnotations(make(map[string]string, 1))
	}

	if _, hasAnnotation := pvc.Annotations[uploadRequestAnnotation]; !hasAnnotation {
		pvc.Annotations[uploadRequestAnnotation] = ""
		var err error
		pvc, err = c.client.CoreV1().PersistentVolumeClaims(pvc.Namespace).Update(context.Background(), pvc, metav1.UpdateOptions{})
		if err != nil {
			return nil, err
		}
	}

	return pvc, nil
}

func (c *command) getAndValidateUploadPVC() (*v1.PersistentVolumeClaim, error) {
	pvc, err := c.client.CoreV1().PersistentVolumeClaims(c.namespace).Get(context.Background(), c.name, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			c.cmd.Printf("PVC %s/%s not found\n", c.namespace, c.name)
		}
		return nil, err
	}

	if !c.noCreate && len(c.size) == 0 {
		return nil, fmt.Errorf("when creating a resource, the size must be specified")
	}

	if !c.createPVC {
		pvc, err = c.validateUploadDataVolume(pvc)
		if err != nil {
			return nil, err
		}
	}

	if !c.createPVC {
		pvc, err = c.validateUploadDataVolume(pvc)
		if err != nil {
			return nil, err
		}
	}

	// for PVCs that exist, we ony want to use them if
	// 1. They have not already been used AND EITHER
	//   a. shouldExist is true
	//   b. shouldExist is false AND the upload annotation exists

	podPhase := pvc.Annotations[podPhaseAnnotation]

	if podPhase == string(v1.PodSucceeded) {
		return nil, fmt.Errorf("PVC %s already successfully imported/cloned/updated", c.name)
	}

	_, isUploadPVC := pvc.Annotations[uploadRequestAnnotation]
	if !c.noCreate && !isUploadPVC {
		return nil, fmt.Errorf("PVC %s not available for upload", c.name)
	}

	// for PVCs that exist and the user wants to upload archive
	// the pvc has to have the contentType archive annotation
	if c.archiveUpload {
		contentType, found := pvc.Annotations[contentTypeAnnotation]
		if !found || contentType != string(cdiv1.DataVolumeArchive) {
			return nil, fmt.Errorf("PVC %s doesn't have archive contentType annotation", c.name)
		}
	}

	return pvc, nil
}
