package apply

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"kubevirt.io/kubevirt/pkg/controller"

	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"kubevirt.io/client-go/log"
)

func (r *Reconciler) createOrUpdateValidatingWebhookConfigurations(caBundle []byte) error {

	for _, webhook := range r.targetStrategy.ValidatingWebhookConfigurations() {
		err := r.createOrUpdateValidatingWebhookConfiguration(webhook, caBundle)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *Reconciler) createOrUpdateValidatingWebhookConfiguration(webhook *admissionregistrationv1beta1.ValidatingWebhookConfiguration, caBundle []byte) error {
	kv := r.kv
	version, imageRegistry, id := getTargetVersionRegistryID(r.kv)

	kvkey, err := controller.KeyFunc(kv)
	if err != nil {
		return err
	}

	var cachedWebhook *admissionregistrationv1beta1.ValidatingWebhookConfiguration
	webhook = webhook.DeepCopy()

	for i, _ := range webhook.Webhooks {
		webhook.Webhooks[i].ClientConfig.CABundle = caBundle
	}

	obj, exists, _ := r.stores.ValidationWebhookCache.Get(webhook)
	// since these objects was in the past unmanaged, reconcile and pick it up if it exists
	if !exists {
		cachedWebhook, err = r.clientset.AdmissionregistrationV1beta1().ValidatingWebhookConfigurations().Get(context.Background(), webhook.Name, metav1.GetOptions{})
		if errors.IsNotFound(err) {
			exists = false
		} else if err != nil {
			return err
		} else {
			exists = true
		}
	} else if exists {
		cachedWebhook = obj.(*admissionregistrationv1beta1.ValidatingWebhookConfiguration)
	}

	certsMatch := true
	if exists {
		for _, wh := range cachedWebhook.Webhooks {
			if !reflect.DeepEqual(wh.ClientConfig.CABundle, caBundle) {
				certsMatch = false
				break
			}
		}
	}

	injectOperatorMetadata(kv, &webhook.ObjectMeta, version, imageRegistry, id, true)
	if !exists {
		r.expectations.ValidationWebhook.RaiseExpectations(kvkey, 1, 0)
		_, err := r.clientset.AdmissionregistrationV1beta1().ValidatingWebhookConfigurations().Create(context.Background(), webhook, metav1.CreateOptions{})
		if err != nil {
			r.expectations.ValidationWebhook.LowerExpectations(kvkey, 1, 0)
			return fmt.Errorf("unable to create validatingwebhook %+v: %v", webhook, err)
		}

		return nil
	}

	if !objectMatchesVersion(&cachedWebhook.ObjectMeta, version, imageRegistry, id, kv.GetGeneration()) || !certsMatch {
		// Patch if old version
		var ops []string

		// Add Labels and Annotations Patches
		labelAnnotationPatch, err := createLabelsAndAnnotationsPatch(&webhook.ObjectMeta)
		if err != nil {
			return err
		}
		ops = append(ops, labelAnnotationPatch...)

		// Add Spec Patch
		webhooks, err := json.Marshal(webhook.Webhooks)
		if err != nil {
			return err
		}
		ops = append(ops, fmt.Sprintf(`{ "op": "replace", "path": "/webhooks", "value": %s }`, string(webhooks)))

		_, err = r.clientset.AdmissionregistrationV1beta1().ValidatingWebhookConfigurations().Patch(context.Background(), webhook.Name, types.JSONPatchType, generatePatchBytes(ops), metav1.PatchOptions{})
		if err != nil {
			return fmt.Errorf("unable to patch validatingwebhookconfiguration %+v: %v", webhook, err)
		}

		log.Log.V(2).Infof("validatingwebhoookconfiguration %v updated", webhook.GetName())
		return nil

	}

	log.Log.V(4).Infof("validatingwebhookconfiguration %v is up-to-date", webhook.GetName())
	return nil
}

func (r *Reconciler) createOrUpdateMutatingWebhookConfigurations(caBundle []byte) error {
	for _, webhook := range r.targetStrategy.MutatingWebhookConfigurations() {
		err := r.createOrUpdateMutatingWebhookConfiguration(webhook, caBundle)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *Reconciler) createOrUpdateMutatingWebhookConfiguration(webhook *admissionregistrationv1beta1.MutatingWebhookConfiguration, caBundle []byte) error {

	kv := r.kv
	version, imageRegistry, id := getTargetVersionRegistryID(r.kv)

	kvkey, err := controller.KeyFunc(kv)
	if err != nil {
		return err
	}

	var cachedWebhook *admissionregistrationv1beta1.MutatingWebhookConfiguration
	webhook = webhook.DeepCopy()

	for i, _ := range webhook.Webhooks {
		webhook.Webhooks[i].ClientConfig.CABundle = caBundle
	}

	obj, exists, _ := r.stores.MutatingWebhookCache.Get(webhook)
	// since these objects was in the past unmanaged, reconcile and pick it up if it exists
	if !exists {
		cachedWebhook, err = r.clientset.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Get(context.Background(), webhook.Name, metav1.GetOptions{})
		if errors.IsNotFound(err) {
			exists = false
		} else if err != nil {
			return err
		} else {
			exists = true
		}
	} else if exists {
		cachedWebhook = obj.(*admissionregistrationv1beta1.MutatingWebhookConfiguration)
	}

	certsMatch := true
	if exists {
		for _, wh := range cachedWebhook.Webhooks {
			if !reflect.DeepEqual(wh.ClientConfig.CABundle, caBundle) {
				certsMatch = false
				break
			}
		}
	}

	injectOperatorMetadata(kv, &webhook.ObjectMeta, version, imageRegistry, id, true)
	if !exists {
		r.expectations.MutatingWebhook.RaiseExpectations(kvkey, 1, 0)
		_, err := r.clientset.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Create(context.Background(), webhook, metav1.CreateOptions{})
		if err != nil {
			r.expectations.MutatingWebhook.LowerExpectations(kvkey, 1, 0)
			return fmt.Errorf("unable to create mutatingwebhook %+v: %v", webhook, err)
		}

		return nil
	}

	if !objectMatchesVersion(&cachedWebhook.ObjectMeta, version, imageRegistry, id, kv.GetGeneration()) || !certsMatch {
		// Patch if old version
		var ops []string

		// Add Labels and Annotations Patches
		labelAnnotationPatch, err := createLabelsAndAnnotationsPatch(&webhook.ObjectMeta)
		if err != nil {
			return err
		}
		ops = append(ops, labelAnnotationPatch...)

		// Add Spec Patch
		webhooks, err := json.Marshal(webhook.Webhooks)
		if err != nil {
			return err
		}
		ops = append(ops, fmt.Sprintf(`{ "op": "replace", "path": "/webhooks", "value": %s }`, string(webhooks)))

		_, err = r.clientset.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Patch(context.Background(), webhook.Name, types.JSONPatchType, generatePatchBytes(ops), metav1.PatchOptions{})
		if err != nil {
			return fmt.Errorf("unable to patch mutatingwebhookconfiguration %+v: %v", webhook, err)
		}

		log.Log.V(2).Infof("mutatingwebhoookconfiguration %v updated", webhook.GetName())
		return nil

	}

	log.Log.V(4).Infof("mutatingwebhookconfiguration %v is up-to-date", webhook.GetName())
	return nil
}
