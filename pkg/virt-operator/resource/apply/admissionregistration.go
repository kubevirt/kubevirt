package apply

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/openshift/library-go/pkg/operator/resource/resourcemerge"
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

	version, imageRegistry, id := getTargetVersionRegistryID(r.kv)

	webhook = webhook.DeepCopy()

	for i := range webhook.Webhooks {
		webhook.Webhooks[i].ClientConfig.CABundle = caBundle
	}
	injectOperatorMetadata(r.kv, &webhook.ObjectMeta, version, imageRegistry, id, true)

	var cachedWebhook *admissionregistrationv1beta1.ValidatingWebhookConfiguration
	var err error
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
	} else {
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

	if !exists {
		r.expectations.ValidationWebhook.RaiseExpectations(r.kvKey, 1, 0)
		webhook, err := r.clientset.AdmissionregistrationV1beta1().ValidatingWebhookConfigurations().Create(context.Background(), webhook, metav1.CreateOptions{})
		if err != nil {
			r.expectations.ValidationWebhook.LowerExpectations(r.kvKey, 1, 0)
			return fmt.Errorf("unable to create validatingwebhook %+v: %v", webhook, err)
		}
		SetValidatingWebhookConfigurationGeneration(&r.kv.Status.Generations, webhook)

		return nil
	}

	modified := resourcemerge.BoolPtr(false)
	existingCopy := cachedWebhook.DeepCopy()
	expectedGeneration := ExpectedValidatingWebhookConfigurationGeneration(webhook, r.kv.Status.Generations)

	resourcemerge.EnsureObjectMeta(modified, &existingCopy.ObjectMeta, webhook.ObjectMeta)
	// there was no change to metadata, the generation was right
	if !*modified && existingCopy.ObjectMeta.Generation == expectedGeneration && certsMatch {
		log.Log.V(4).Infof("validatingwebhookconfiguration %v is up-to-date", webhook.GetName())
		return nil
	}

	// Patch if old version
	ops := []string{
		fmt.Sprintf(testGenerationJSONPatchTemplate, cachedWebhook.ObjectMeta.Generation),
	}

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

	ops = append(ops, fmt.Sprintf(replaceWebhooksValueTemplate, string(webhooks)))

	webhook, err = r.clientset.AdmissionregistrationV1beta1().ValidatingWebhookConfigurations().Patch(context.Background(), webhook.Name, types.JSONPatchType, generatePatchBytes(ops), metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("unable to update validatingwebhookconfiguration %+v: %v", webhook, err)
	}

	SetValidatingWebhookConfigurationGeneration(&r.kv.Status.Generations, webhook)
	log.Log.V(2).Infof("validatingwebhoookconfiguration %v updated", webhook.Name)

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
	version, imageRegistry, id := getTargetVersionRegistryID(r.kv)

	webhook = webhook.DeepCopy()

	for i := range webhook.Webhooks {
		webhook.Webhooks[i].ClientConfig.CABundle = caBundle
	}

	injectOperatorMetadata(r.kv, &webhook.ObjectMeta, version, imageRegistry, id, true)

	var cachedWebhook *admissionregistrationv1beta1.MutatingWebhookConfiguration
	var err error
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
	} else {
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

	if !exists {
		r.expectations.MutatingWebhook.RaiseExpectations(r.kvKey, 1, 0)
		webhook, err = r.clientset.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Create(context.Background(), webhook, metav1.CreateOptions{})
		if err != nil {
			r.expectations.MutatingWebhook.LowerExpectations(r.kvKey, 1, 0)
			return fmt.Errorf("unable to create mutatingwebhook %+v: %v", webhook, err)
		}

		SetMutatingWebhookConfigurationGeneration(&r.kv.Status.Generations, webhook)
		log.Log.V(2).Infof("mutatingwebhoookconfiguration %v created", webhook.Name)
		return nil
	}

	modified := resourcemerge.BoolPtr(false)
	existingCopy := cachedWebhook.DeepCopy()
	expectedGeneration := ExpectedMutatingWebhookConfigurationGeneration(webhook, r.kv.Status.Generations)

	resourcemerge.EnsureObjectMeta(modified, &existingCopy.ObjectMeta, webhook.ObjectMeta)
	// there was no change to metadata, the generation was right
	if !*modified && existingCopy.ObjectMeta.Generation == expectedGeneration && certsMatch {
		log.Log.V(4).Infof("mutating webhook configuration %v is up-to-date", webhook.GetName())
		return nil
	}

	// Patch if old version
	ops := []string{
		fmt.Sprintf(testGenerationJSONPatchTemplate, cachedWebhook.ObjectMeta.Generation),
	}

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

	ops = append(ops, fmt.Sprintf(replaceWebhooksValueTemplate, string(webhooks)))

	webhook, err = r.clientset.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Patch(context.Background(), webhook.Name, types.JSONPatchType, generatePatchBytes(ops), metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("unable to update mutatingwebhookconfiguration %+v: %v", webhook, err)
	}

	SetMutatingWebhookConfigurationGeneration(&r.kv.Status.Generations, webhook)
	log.Log.V(2).Infof("mutatingwebhoookconfiguration %v updated", webhook.Name)

	return nil
}
