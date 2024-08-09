package libstorage

import (
	"context"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
)

func isLocalPV(pv k8sv1.PersistentVolume) bool {
	return pv.Spec.NodeAffinity != nil &&
		pv.Spec.NodeAffinity.Required != nil &&
		len(pv.Spec.NodeAffinity.Required.NodeSelectorTerms) > 0 &&
		(pv.Spec.VolumeMode == nil || *pv.Spec.VolumeMode != k8sv1.PersistentVolumeBlock)
}

func isPVAvailable(pv k8sv1.PersistentVolume) bool {
	return pv.Spec.ClaimRef == nil
}

func MakePVAvailable(ctx context.Context, pv *k8sv1.PersistentVolume) error {
	if pv.Status.Phase != k8sv1.VolumeReleased {
		return nil
	}
	virtClient := kubevirt.Client()
	patchPayload, err := patch.New(patch.WithRemove("/spec/claimRef")).GeneratePayload()
	if err != nil {
		return err
	}

	_, err = virtClient.CoreV1().PersistentVolumes().Patch(
		ctx,
		pv.Name,
		types.JSONPatchType,
		patchPayload,
		metav1.PatchOptions{},
	)

	return err
}

func countLocalStoragePVAvailableForUse(pvList *k8sv1.PersistentVolumeList, storageClassName string) int {
	count := 0
	for _, pv := range pvList.Items {
		if pv.Spec.StorageClassName == storageClassName && isLocalPV(pv) && isPVAvailable(pv) {
			count++
		}
	}
	return count
}
