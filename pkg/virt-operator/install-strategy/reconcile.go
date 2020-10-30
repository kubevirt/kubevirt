package installstrategy

import (
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"kubevirt.io/controller-lifecycle-operator-sdk/pkg/sdk"
)

const LastAppliedConfigAnnotationKey = "kubevirt.io/last-applied-configuration"

func setLastAppliedConfiguration(obj metav1.Object) error {
	return sdk.SetLastAppliedConfiguration(obj, LastAppliedConfigAnnotationKey)
}

func resourceShouldUpdate(currentRuntimeObj, desiredRuntimeObj runtime.Object) (bool, runtime.Object, error) {
	currentMetaObj := currentRuntimeObj.(metav1.Object)
	desiredMetaObj := desiredRuntimeObj.(metav1.Object)

	currentRuntimeObj, err := sdk.StripStatusFromObject(currentRuntimeObj)
	if err != nil {
		return false, currentRuntimeObj, err
	}

	currentRuntimeObjCopy := currentRuntimeObj.DeepCopyObject()
	// allow users to add new annotations (but not change ours)
	sdk.MergeLabelsAndAnnotations(desiredMetaObj, currentMetaObj)

	if !sdk.IsMutable(currentRuntimeObj) {
		err = setLastAppliedConfiguration(desiredMetaObj)
		if err != nil {
			return false, currentRuntimeObj, err
		}
	}

	currentRuntimeObj, err = sdk.MergeObject(desiredRuntimeObj, currentRuntimeObj, LastAppliedConfigAnnotationKey)
	if err != nil {
		return false, currentRuntimeObj, err
	}

	return !reflect.DeepEqual(currentRuntimeObjCopy, currentRuntimeObj), currentRuntimeObj, nil
}
