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

// createUploadPVC validates and creates a new PVC for upload.
func (c *command) createUploadPVC() (*v1.PersistentVolumeClaim, error) {
	if c.accessMode == string(v1.ReadOnlyMany) {
		return nil, fmt.Errorf("cannot upload to a readonly volume, use either ReadWriteOnce or ReadWriteMany")
	}

	if err := c.validateStorageClass(); err != nil {
		return nil, err
	}

	quantity, err := resource.ParseQuantity(c.size)
	if err != nil {
		return nil, fmt.Errorf("invalid size=%s: %w", c.size, err)
	}

	pvc := storagetypes.RenderPVC(&quantity, c.name, c.namespace, c.storageClass, c.accessMode, c.volumeMode == "block")
	if c.volumeMode == "filesystem" {
		pvc.Spec.VolumeMode = pointer.P(v1.PersistentVolumeFilesystem)
	}

	pvc.ObjectMeta.Annotations = c.buildUploadAnnotations()
	c.setDefaultInstancetypeLabels(&pvc.ObjectMeta)

	return c.client.CoreV1().PersistentVolumeClaims(c.namespace).Create(context.Background(), pvc, metav1.CreateOptions{})
}

// ensurePVCSupportsUpload ensures the PVC has the correct upload annotation.
func (c *command) ensurePVCSupportsUpload(pvc *v1.PersistentVolumeClaim) (*v1.PersistentVolumeClaim, error) {
	if pvc.Annotations == nil {
		pvc.Annotations = make(map[string]string)
	}

	if _, exists := pvc.Annotations[uploadRequestAnnotation]; !exists {
		pvc.Annotations[uploadRequestAnnotation] = ""
		return c.client.CoreV1().PersistentVolumeClaims(pvc.Namespace).Update(context.Background(), pvc, metav1.UpdateOptions{})
	}

	return pvc, nil
}

// getAndValidateUploadPVC fetches the PVC and validates it for upload usage.
func (c *command) getAndValidateUploadPVC() (*v1.PersistentVolumeClaim, error) {
	pvc, err := c.client.CoreV1().PersistentVolumeClaims(c.namespace).Get(context.Background(), c.name, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			c.cmd.Printf("PVC %s/%s not found\n", c.namespace, c.name)
		}
		return nil, err
	}

	if !c.noCreate && c.size == "" {
		return nil, fmt.Errorf("PVC creation requires specifying a size")
	}

	if !c.createPVC {
		if pvc, err = c.validateUploadDataVolume(pvc); err != nil {
			return nil, err
		}
	}

	if pvc.Annotations[podPhaseAnnotation] == string(v1.PodSucceeded) {
		return nil, fmt.Errorf("PVC %s already successfully imported/cloned/updated", c.name)
	}

	if !c.noCreate && pvc.Annotations[uploadRequestAnnotation] == "" {
		return nil, fmt.Errorf("PVC %s not available for upload", c.name)
	}

	if c.archiveUpload {
		contentType := pvc.Annotations[contentTypeAnnotation]
		if contentType != string(cdiv1.DataVolumeArchive) {
			return nil, fmt.Errorf("PVC %s does not have archive contentType annotation", c.name)
		}
	}

	return pvc, nil
}

// validateStorageClass ensures the user-defined storage class exists.
func (c *command) validateStorageClass() error {
	if c.storageClass == "" {
		return nil
	}
	_, err := c.client.StorageV1().StorageClasses().Get(context.Background(), c.storageClass, metav1.GetOptions{})
	return err
}

// buildUploadAnnotations prepares the annotations for a new upload PVC.
func (c *command) buildUploadAnnotations() map[string]string {
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

	return annotations
}
