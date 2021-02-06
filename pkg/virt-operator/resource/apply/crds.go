package apply

import (
	"context"
	"encoding/json"
	"fmt"

	extv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"kubevirt.io/client-go/log"
)

func (r *Reconciler) createOrUpdateCrds() error {
	for _, crd := range r.targetStrategy.CRDs() {
		err := r.createOrUpdateCrd(crd)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *Reconciler) createOrUpdateCrd(crd *extv1beta1.CustomResourceDefinition) error {
	ext := r.clientset.ExtensionsClient()
	version, imageRegistry, id := getTargetVersionRegistryID(r.kv)
	var cachedCrd *extv1beta1.CustomResourceDefinition

	crd = crd.DeepCopy()
	obj, exists, _ := r.stores.CrdCache.Get(crd)
	if exists {
		cachedCrd = obj.(*extv1beta1.CustomResourceDefinition)
	}

	injectOperatorMetadata(r.kv, &crd.ObjectMeta, version, imageRegistry, id, true)
	if !exists {
		// Create non existent
		r.expectations.Crd.RaiseExpectations(r.kvKey, 1, 0)
		_, err := ext.ApiextensionsV1beta1().CustomResourceDefinitions().Create(context.Background(), crd, metav1.CreateOptions{})
		if err != nil {
			r.expectations.Crd.LowerExpectations(r.kvKey, 1, 0)
			return fmt.Errorf("unable to create crd %+v: %v", crd, err)
		}
		log.Log.V(2).Infof("crd %v created", crd.GetName())
		return nil
	}

	if !objectMatchesVersion(&cachedCrd.ObjectMeta, version, imageRegistry, id, r.kv.GetGeneration()) {
		// Patch if old version
		var ops []string

		// Add Labels and Annotations Patches
		labelAnnotationPatch, err := createLabelsAndAnnotationsPatch(&crd.ObjectMeta)
		if err != nil {
			return err
		}
		ops = append(ops, labelAnnotationPatch...)

		// subresource support needs to be introduced carefully after the control plane roll-over
		// to avoid creating zombie entities which don't get processed du to ignored status updates
		if cachedCrd.Spec.Subresources == nil || cachedCrd.Spec.Subresources.Status == nil {
			if crd.Spec.Subresources != nil && crd.Spec.Subresources.Status != nil {
				crd.Spec.Subresources.Status = nil
			}
		}

		// Add Spec Patch
		newSpec, err := json.Marshal(crd.Spec)
		if err != nil {
			return err
		}
		ops = append(ops, fmt.Sprintf(`{ "op": "replace", "path": "/spec", "value": %s }`, string(newSpec)))

		_, err = ext.ApiextensionsV1beta1().CustomResourceDefinitions().Patch(context.Background(), crd.Name, types.JSONPatchType, generatePatchBytes(ops), metav1.PatchOptions{})
		if err != nil {
			return fmt.Errorf("unable to patch crd %+v: %v", crd, err)
		}

		log.Log.V(2).Infof("crd %v updated", crd.GetName())
		return nil
	}

	log.Log.V(4).Infof("crd %v is up-to-date", crd.GetName())
	return nil
}

func (r *Reconciler) rolloutNonCompatibleCRDChanges() error {
	for _, crd := range r.targetStrategy.CRDs() {
		err := r.rolloutNonCompatibleCRDChange(crd)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *Reconciler) rolloutNonCompatibleCRDChange(crd *extv1beta1.CustomResourceDefinition) error {

	ext := r.clientset.ExtensionsClient()
	version, imageRegistry, id := getTargetVersionRegistryID(r.kv)
	var cachedCrd *extv1beta1.CustomResourceDefinition

	crd = crd.DeepCopy()
	obj, exists, _ := r.stores.CrdCache.Get(crd)
	if exists {
		cachedCrd = obj.(*extv1beta1.CustomResourceDefinition)
	}

	injectOperatorMetadata(r.kv, &crd.ObjectMeta, version, imageRegistry, id, true)
	if exists && objectMatchesVersion(&cachedCrd.ObjectMeta, version, imageRegistry, id, r.kv.GetGeneration()) {
		// Patch if in the deployed version the subresource is not enabled
		var ops []string

		// enable the status subresources now, in case that they were disabled before
		if crd.Spec.Subresources == nil || crd.Spec.Subresources.Status == nil {
			return nil
		}

		if crd.Spec.Subresources != nil && crd.Spec.Subresources.Status != nil {
			if cachedCrd.Spec.Subresources != nil && cachedCrd.Spec.Subresources.Status != nil {
				return nil
			}
		}

		// Add Spec Patch
		newSpec, err := json.Marshal(crd.Spec)
		if err != nil {
			return err
		}
		ops = append(ops, fmt.Sprintf(`{ "op": "replace", "path": "/spec", "value": %s }`, string(newSpec)))

		_, err = ext.ApiextensionsV1beta1().CustomResourceDefinitions().Patch(context.Background(), crd.Name, types.JSONPatchType, generatePatchBytes(ops), metav1.PatchOptions{})
		if err != nil {
			return fmt.Errorf("unable to patch crd %+v: %v", crd, err)
		}

		log.Log.V(2).Infof("crd %v updated", crd.GetName())
		return nil
	}

	log.Log.V(4).Infof("crd %v is up-to-date", crd.GetName())
	return nil
}
