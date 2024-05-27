package apply

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/openshift/library-go/pkg/operator/resource/resourcemerge"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	"k8s.io/apimachinery/pkg/api/equality"
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

func convertV1ValidatingWebhookToV1beta1(from *admissionregistrationv1.ValidatingWebhookConfiguration) (*admissionregistrationv1beta1.ValidatingWebhookConfiguration, error) {
	var b []byte
	b, err := from.Marshal()
	if err != nil {
		return nil, err
	}
	webhookv1beta1 := &admissionregistrationv1beta1.ValidatingWebhookConfiguration{}
	if err = webhookv1beta1.Unmarshal(b); err != nil {
		return nil, err
	}
	return webhookv1beta1, nil
}

func convertV1beta1ValidatingWebhookToV1(from *admissionregistrationv1beta1.ValidatingWebhookConfiguration) (*admissionregistrationv1.ValidatingWebhookConfiguration, error) {
	var b []byte
	b, err := from.Marshal()
	if err != nil {
		return nil, err
	}
	webhookv1 := &admissionregistrationv1.ValidatingWebhookConfiguration{}
	if err = webhookv1.Unmarshal(b); err != nil {
		return nil, err
	}
	return webhookv1, nil
}

func convertV1MutatingWebhookToV1beta1(from *admissionregistrationv1.MutatingWebhookConfiguration) (*admissionregistrationv1beta1.MutatingWebhookConfiguration, error) {
	var b []byte
	b, err := from.Marshal()
	if err != nil {
		return nil, err
	}
	webhookv1beta1 := &admissionregistrationv1beta1.MutatingWebhookConfiguration{}
	if err = webhookv1beta1.Unmarshal(b); err != nil {
		return nil, err
	}
	return webhookv1beta1, nil
}

func convertV1beta1MutatingWebhookToV1(from *admissionregistrationv1beta1.MutatingWebhookConfiguration) (*admissionregistrationv1.MutatingWebhookConfiguration, error) {
	var b []byte
	b, err := from.Marshal()
	if err != nil {
		return nil, err
	}
	webhookv1 := &admissionregistrationv1.MutatingWebhookConfiguration{}
	if err = webhookv1.Unmarshal(b); err != nil {
		return nil, err
	}
	return webhookv1, nil
}

func (r *Reconciler) patchValidatingWebhookConfiguration(webhook *admissionregistrationv1.ValidatingWebhookConfiguration, ops []string) (patchedWebhook *admissionregistrationv1.ValidatingWebhookConfiguration, err error) {
	switch webhook.APIVersion {
	case admissionregistrationv1.SchemeGroupVersion.Version, admissionregistrationv1.SchemeGroupVersion.String():
		patchedWebhook, err = r.clientset.AdmissionregistrationV1().ValidatingWebhookConfigurations().Patch(context.Background(), webhook.Name, types.JSONPatchType, generatePatchBytes(ops), metav1.PatchOptions{})
	case admissionregistrationv1beta1.SchemeGroupVersion.Version, admissionregistrationv1beta1.SchemeGroupVersion.String():
		var out *admissionregistrationv1beta1.ValidatingWebhookConfiguration
		out, err = r.clientset.AdmissionregistrationV1beta1().ValidatingWebhookConfigurations().Patch(context.Background(), webhook.Name, types.JSONPatchType, generatePatchBytes(ops), metav1.PatchOptions{})
		if err != nil {
			return
		}
		patchedWebhook, err = convertV1beta1ValidatingWebhookToV1(out)
	default:
		err = fmt.Errorf("ValidatingWebhookConfiguration APIVersion %s not supported", webhook.APIVersion)
	}
	return
}

func (r *Reconciler) createValidatingWebhookConfiguration(webhook *admissionregistrationv1.ValidatingWebhookConfiguration) (createdWebhook *admissionregistrationv1.ValidatingWebhookConfiguration, err error) {
	switch webhook.APIVersion {
	case admissionregistrationv1.SchemeGroupVersion.Version, admissionregistrationv1.SchemeGroupVersion.String():
		createdWebhook, err = r.clientset.AdmissionregistrationV1().ValidatingWebhookConfigurations().Create(context.Background(), webhook, metav1.CreateOptions{})
	case admissionregistrationv1beta1.SchemeGroupVersion.Version, admissionregistrationv1beta1.SchemeGroupVersion.String():
		var webhookv1beta1 *admissionregistrationv1beta1.ValidatingWebhookConfiguration
		webhookv1beta1, err = convertV1ValidatingWebhookToV1beta1(webhook)
		if err != nil {
			return nil, err
		}
		webhookv1beta1, err = r.clientset.AdmissionregistrationV1beta1().ValidatingWebhookConfigurations().Create(context.Background(), webhookv1beta1, metav1.CreateOptions{})
		if err != nil {
			return nil, err
		}
		createdWebhook, err = convertV1beta1ValidatingWebhookToV1(webhookv1beta1)
	default:
		err = fmt.Errorf("ValidatingWebhookConfiguration APIVersion %s not supported", webhook.APIVersion)
	}
	return
}

func (r *Reconciler) patchMutatingWebhookConfiguration(webhook *admissionregistrationv1.MutatingWebhookConfiguration, ops []string) (patchedWebhook *admissionregistrationv1.MutatingWebhookConfiguration, err error) {
	switch webhook.APIVersion {
	case admissionregistrationv1.SchemeGroupVersion.Version, admissionregistrationv1.SchemeGroupVersion.String():
		patchedWebhook, err = r.clientset.AdmissionregistrationV1().MutatingWebhookConfigurations().Patch(context.Background(), webhook.Name, types.JSONPatchType, generatePatchBytes(ops), metav1.PatchOptions{})
	case admissionregistrationv1beta1.SchemeGroupVersion.Version, admissionregistrationv1beta1.SchemeGroupVersion.String():
		var out *admissionregistrationv1beta1.MutatingWebhookConfiguration
		out, err = r.clientset.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Patch(context.Background(), webhook.Name, types.JSONPatchType, generatePatchBytes(ops), metav1.PatchOptions{})
		if err != nil {
			return
		}
		patchedWebhook, err = convertV1beta1MutatingWebhookToV1(out)
	default:
		err = fmt.Errorf("MutatingWebhookConfiguration APIVersion %s not supported", webhook.APIVersion)
	}
	return
}

func (r *Reconciler) createMutatingWebhookConfiguration(webhook *admissionregistrationv1.MutatingWebhookConfiguration) (createdWebhook *admissionregistrationv1.MutatingWebhookConfiguration, err error) {
	switch webhook.APIVersion {
	case admissionregistrationv1.SchemeGroupVersion.Version, admissionregistrationv1.SchemeGroupVersion.String():
		createdWebhook, err = r.clientset.AdmissionregistrationV1().MutatingWebhookConfigurations().Create(context.Background(), webhook, metav1.CreateOptions{})
	case admissionregistrationv1beta1.SchemeGroupVersion.Version, admissionregistrationv1beta1.SchemeGroupVersion.String():
		var webhookv1beta1 *admissionregistrationv1beta1.MutatingWebhookConfiguration
		webhookv1beta1, err = convertV1MutatingWebhookToV1beta1(webhook)
		if err != nil {
			return nil, err
		}
		webhookv1beta1, err = r.clientset.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Create(context.Background(), webhookv1beta1, metav1.CreateOptions{})
		if err != nil {
			return nil, err
		}
		createdWebhook, err = convertV1beta1MutatingWebhookToV1(webhookv1beta1)
	default:
		err = fmt.Errorf("MutatingWebhookConfiguration APIVersion %s not supported", webhook.APIVersion)
	}
	return
}

func (r *Reconciler) createOrUpdateValidatingWebhookConfiguration(webhook *admissionregistrationv1.ValidatingWebhookConfiguration, caBundle []byte) error {
	version, imageRegistry, id := getTargetVersionRegistryID(r.kv)

	webhook = webhook.DeepCopy()

	for i := range webhook.Webhooks {
		webhook.Webhooks[i].ClientConfig.CABundle = caBundle
	}
	injectOperatorMetadata(r.kv, &webhook.ObjectMeta, version, imageRegistry, id, true)

	var cachedWebhook *admissionregistrationv1.ValidatingWebhookConfiguration
	var err error
	obj, exists, _ := r.stores.ValidationWebhookCache.Get(webhook)
	// since these objects was in the past unmanaged, reconcile and pick it up if it exists

	if !exists {
		cachedWebhook, err = r.clientset.AdmissionregistrationV1().ValidatingWebhookConfigurations().Get(context.Background(), webhook.Name, metav1.GetOptions{})
		if errors.IsNotFound(err) {
			exists = false
		} else if err != nil {
			return err
		} else {
			exists = true
		}
	} else {
		cachedWebhook = obj.(*admissionregistrationv1.ValidatingWebhookConfiguration)
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
		webhook, err := r.createValidatingWebhookConfiguration(webhook)
		if err != nil {
			r.expectations.ValidationWebhook.LowerExpectations(r.kvKey, 1, 0)
			return fmt.Errorf("unable to create validatingwebhook %+v: %v", webhook, err)
		}

		SetGeneration(&r.kv.Status.Generations, webhook)

		return nil
	}

	modified := resourcemerge.BoolPtr(false)
	existingCopy := cachedWebhook.DeepCopy()
	expectedGeneration := GetExpectedGeneration(webhook, r.kv.Status.Generations)

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

	webhook, err = r.patchValidatingWebhookConfiguration(webhook, ops)
	if err != nil {
		return fmt.Errorf("unable to update validatingwebhookconfiguration %+v: %v", webhook, err)
	}

	SetGeneration(&r.kv.Status.Generations, webhook)
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

func (r *Reconciler) createOrUpdateMutatingWebhookConfiguration(webhook *admissionregistrationv1.MutatingWebhookConfiguration, caBundle []byte) error {
	version, imageRegistry, id := getTargetVersionRegistryID(r.kv)

	webhook = webhook.DeepCopy()

	for i := range webhook.Webhooks {
		webhook.Webhooks[i].ClientConfig.CABundle = caBundle
	}

	injectOperatorMetadata(r.kv, &webhook.ObjectMeta, version, imageRegistry, id, true)

	var cachedWebhook *admissionregistrationv1.MutatingWebhookConfiguration
	var err error
	obj, exists, _ := r.stores.MutatingWebhookCache.Get(webhook)
	// since these objects was in the past unmanaged, reconcile and pick it up if it exists
	if !exists {
		cachedWebhook, err = r.clientset.AdmissionregistrationV1().MutatingWebhookConfigurations().Get(context.Background(), webhook.Name, metav1.GetOptions{})
		if errors.IsNotFound(err) {
			exists = false
		} else if err != nil {
			return err
		} else {
			exists = true
		}
	} else {
		cachedWebhook = obj.(*admissionregistrationv1.MutatingWebhookConfiguration)
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
		webhook, err := r.createMutatingWebhookConfiguration(webhook)
		if err != nil {
			r.expectations.MutatingWebhook.LowerExpectations(r.kvKey, 1, 0)
			return fmt.Errorf("unable to create mutatingwebhook %+v: %v", webhook, err)
		}

		SetGeneration(&r.kv.Status.Generations, webhook)
		log.Log.V(2).Infof("mutatingwebhoookconfiguration %v created", webhook.Name)
		return nil
	}

	modified := resourcemerge.BoolPtr(false)
	existingCopy := cachedWebhook.DeepCopy()
	expectedGeneration := GetExpectedGeneration(webhook, r.kv.Status.Generations)

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

	webhook, err = r.patchMutatingWebhookConfiguration(webhook, ops)
	if err != nil {
		return fmt.Errorf("unable to update mutatingwebhookconfiguration %+v: %v", webhook, err)
	}

	SetGeneration(&r.kv.Status.Generations, webhook)
	log.Log.V(2).Infof("mutatingwebhoookconfiguration %v updated", webhook.Name)

	return nil
}

func generateValidatingAdmissionPolicyBindingPatch(
	cachedValidatingAdmissionPolicyBinding *admissionregistrationv1.ValidatingAdmissionPolicyBinding,
	validatingAdmissionPolicyBinding *admissionregistrationv1.ValidatingAdmissionPolicyBinding) ([]string, error) {

	patchOps, err := getObjectMetaPatch(validatingAdmissionPolicyBinding.ObjectMeta, cachedValidatingAdmissionPolicyBinding.ObjectMeta)
	if err != nil {
		return nil, err
	}

	// If the Specs don't equal each other, replace it
	if !equality.Semantic.DeepEqual(cachedValidatingAdmissionPolicyBinding.Spec, validatingAdmissionPolicyBinding.Spec) {
		newSpec, err := json.Marshal(validatingAdmissionPolicyBinding.Spec)
		if err != nil {
			return patchOps, err
		}

		patchOps = append(patchOps, fmt.Sprintf(`{ "op": "replace", "path": "/spec", "value": %s }`, string(newSpec)))
	}

	return patchOps, nil
}

func (r *Reconciler) createOrUpdateValidatingAdmissionPolicyBindings() error {
	if !r.stores.ValidatingAdmissionPolicyBindingEnabled {
		return nil
	}

	for _, validatingAdmissionPolicyBinding := range r.targetStrategy.ValidatingAdmissionPolicyBindings() {
		err := r.createOrUpdateValidatingAdmissionPolicyBinding(validatingAdmissionPolicyBinding.DeepCopy())
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *Reconciler) createOrUpdateValidatingAdmissionPolicyBinding(validatingAdmissionPolicyBinding *admissionregistrationv1.ValidatingAdmissionPolicyBinding) error {
	admissionRegistrationV1 := r.clientset.AdmissionregistrationV1()
	version, imageRegistry, id := getTargetVersionRegistryID(r.kv)

	injectOperatorMetadata(r.kv, &validatingAdmissionPolicyBinding.ObjectMeta, version, imageRegistry, id, true)

	obj, exists, _ := r.stores.ValidatingAdmissionPolicyBindingCache.Get(validatingAdmissionPolicyBinding)

	if !exists {
		r.expectations.ValidatingAdmissionPolicyBinding.RaiseExpectations(r.kvKey, 1, 0)
		_, err := admissionRegistrationV1.ValidatingAdmissionPolicyBindings().Create(context.Background(), validatingAdmissionPolicyBinding, metav1.CreateOptions{})
		if err != nil {
			r.expectations.ValidatingAdmissionPolicyBinding.LowerExpectations(r.kvKey, 1, 0)
			return fmt.Errorf("unable to create validatingAdmissionPolicyBinding %+v: %v", validatingAdmissionPolicyBinding, err)
		}

		return nil
	}

	cachedValidatingAdmissionPolicyBinding := obj.(*admissionregistrationv1.ValidatingAdmissionPolicyBinding)

	patchOps, err := generateValidatingAdmissionPolicyBindingPatch(cachedValidatingAdmissionPolicyBinding, validatingAdmissionPolicyBinding)
	if err != nil {
		return fmt.Errorf("unable to generate validatingAdmissionPolicyBinding patch operations for %+v: %v", validatingAdmissionPolicyBinding, err)
	}

	if len(patchOps) == 0 {
		log.Log.V(4).Infof("validatingAdmissionPolicyBinding %v is up-to-date", validatingAdmissionPolicyBinding.GetName())
		return nil
	}

	_, err = admissionRegistrationV1.ValidatingAdmissionPolicyBindings().Patch(context.Background(),
		validatingAdmissionPolicyBinding.Name,
		types.JSONPatchType,
		generatePatchBytes(patchOps),
		metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("unable to patch validatingAdmissionPolicyBinding %+v: %v", validatingAdmissionPolicyBinding, err)
	}

	log.Log.V(2).Infof("validatingAdmissionPolicyBinding %v patched", validatingAdmissionPolicyBinding.GetName())
	return nil
}

func generateValidatingAdmissionPolicyPatch(
	cachedValidatingAdmissionPolicy *admissionregistrationv1.ValidatingAdmissionPolicy,
	validatingAdmissionPolicy *admissionregistrationv1.ValidatingAdmissionPolicy) ([]string, error) {

	patchOps, err := getObjectMetaPatch(validatingAdmissionPolicy.ObjectMeta, cachedValidatingAdmissionPolicy.ObjectMeta)
	if err != nil {
		return nil, err
	}

	// If the Specs don't equal each other, replace it
	if !equality.Semantic.DeepEqual(cachedValidatingAdmissionPolicy.Spec, validatingAdmissionPolicy.Spec) {
		newSpec, err := json.Marshal(validatingAdmissionPolicy.Spec)
		if err != nil {
			return patchOps, err
		}

		patchOps = append(patchOps, fmt.Sprintf(`{ "op": "replace", "path": "/spec", "value": %s }`, string(newSpec)))
	}

	return patchOps, nil
}

func (r *Reconciler) createOrUpdateValidatingAdmissionPolicies() error {
	if !r.stores.ValidatingAdmissionPolicyEnabled {
		return nil
	}

	for _, validatingAdmissionPolicy := range r.targetStrategy.ValidatingAdmissionPolicies() {
		err := r.createOrUpdateValidatingAdmissionPolicy(validatingAdmissionPolicy.DeepCopy())
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *Reconciler) createOrUpdateValidatingAdmissionPolicy(validatingAdmissionPolicy *admissionregistrationv1.ValidatingAdmissionPolicy) error {
	admissionRegistrationV1 := r.clientset.AdmissionregistrationV1()
	version, imageRegistry, id := getTargetVersionRegistryID(r.kv)

	injectOperatorMetadata(r.kv, &validatingAdmissionPolicy.ObjectMeta, version, imageRegistry, id, true)

	obj, exists, _ := r.stores.ValidatingAdmissionPolicyCache.Get(validatingAdmissionPolicy)

	if !exists {
		r.expectations.ValidatingAdmissionPolicy.RaiseExpectations(r.kvKey, 1, 0)
		_, err := admissionRegistrationV1.ValidatingAdmissionPolicies().Create(context.Background(), validatingAdmissionPolicy, metav1.CreateOptions{})
		if err != nil {
			r.expectations.ValidatingAdmissionPolicy.LowerExpectations(r.kvKey, 1, 0)
			return fmt.Errorf("unable to create validatingAdmissionPolicy %+v: %v", validatingAdmissionPolicy, err)
		}

		return nil
	}

	cachedValidatingAdmissionPolicy := obj.(*admissionregistrationv1.ValidatingAdmissionPolicy)

	patchOps, err := generateValidatingAdmissionPolicyPatch(cachedValidatingAdmissionPolicy, validatingAdmissionPolicy)
	if err != nil {
		return fmt.Errorf("unable to generate validatingAdmissionPolicy patch operations for %+v: %v", validatingAdmissionPolicy, err)
	}

	if len(patchOps) == 0 {
		log.Log.V(4).Infof("validatingAdmissionPolicy %v is up-to-date", validatingAdmissionPolicy.GetName())
		return nil
	}

	_, err = admissionRegistrationV1.ValidatingAdmissionPolicies().Patch(context.Background(), validatingAdmissionPolicy.Name, types.JSONPatchType, generatePatchBytes(patchOps), metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("unable to patch validatingAdmissionPolicy %+v: %v", validatingAdmissionPolicy, err)
	}

	log.Log.V(2).Infof("validatingAdmissionPolicy %v patched", validatingAdmissionPolicy.GetName())
	return nil
}
