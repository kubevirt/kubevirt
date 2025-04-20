/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright The KubeVirt Authors.
 */

package apply

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"time"

	"github.com/openshift/library-go/pkg/operator/resource/resourcemerge"

	corev1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	typedv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/util/workqueue"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/certificates/triple/cert"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
	"kubevirt.io/kubevirt/pkg/virt-operator/util"
)

func (r *Reconciler) syncKubevirtNamespaceLabels() error {

	targetNamespace := r.kv.ObjectMeta.Namespace
	obj, exists, err := r.stores.NamespaceCache.GetByKey(targetNamespace)
	if err != nil {
		log.Log.Errorf("Failed to retrieve kubevirt namespace from store. Error: %s", err.Error())
		return err
	}

	if !exists {
		return fmt.Errorf("Could not find namespace in store. Namespace key: %s", targetNamespace)
	}

	cachedNamespace := obj.(*corev1.Namespace)
	// Prepare namespace metadata patch
	targetLabels := map[string]string{
		"openshift.io/cluster-monitoring": "true",
	}
	cachedLabels := cachedNamespace.ObjectMeta.Labels
	labelsToPatch := make(map[string]string)
	for targetLabelKey, targetLabelValue := range targetLabels {
		cachedLabelValue, ok := cachedLabels[targetLabelKey]
		if ok && cachedLabelValue == targetLabelValue {
			continue
		}
		labelsToPatch[targetLabelKey] = targetLabelValue
	}

	if len(labelsToPatch) == 0 {
		log.Log.Infof("Kubevirt namespace (%s) labels are in sync", targetNamespace)
		return nil
	}

	labelsPatch, err := json.Marshal(labelsToPatch)
	if err != nil {
		log.Log.Errorf("Failed to marshal namespace labels: %s", err.Error())
		return err
	}

	log.Log.Infof("Patching namespace %s with %s", targetNamespace, labelsPatch)
	_, err = r.clientset.CoreV1().Namespaces().Patch(context.Background(),
		targetNamespace,
		types.MergePatchType,
		[]byte(fmt.Sprintf(`{"metadata":{"labels": %s}}`, labelsPatch)),
		metav1.PatchOptions{},
	)
	if err != nil {
		log.Log.Errorf("Could not patch kubevirt namespace labels: %s", err.Error())
		return err
	}
	log.Log.Infof("kubevirt namespace labels patched")
	return nil
}

func (r *Reconciler) createOrUpdateServices() (bool, error) {

	for _, service := range r.targetStrategy.Services() {
		pending, err := r.createOrUpdateService(service.DeepCopy())
		if pending || err != nil {
			return pending, err
		}
	}

	return false, nil
}

func (r *Reconciler) createOrUpdateService(service *corev1.Service) (bool, error) {
	core := r.clientset.CoreV1()
	version, imageRegistry, id := getTargetVersionRegistryID(r.kv)

	injectOperatorMetadata(r.kv, &service.ObjectMeta, version, imageRegistry, id, true)

	obj, exists, _ := r.stores.ServiceCache.Get(service)

	if !exists {
		r.expectations.Service.RaiseExpectations(r.kvKey, 1, 0)
		_, err := core.Services(service.Namespace).Create(context.Background(), service, metav1.CreateOptions{})
		if err != nil {
			r.expectations.Service.LowerExpectations(r.kvKey, 1, 0)
			return false, fmt.Errorf("unable to create service %+v: %v", service, err)
		}

		return false, nil
	}

	cachedService := obj.(*corev1.Service)

	deleteAndReplace := hasImmutableFieldChanged(service, cachedService)
	if deleteAndReplace {
		err := deleteService(cachedService, r.kvKey, r.expectations, core)
		if err != nil {
			return false, err
		}
		// waiting for old service to be deleted,
		// after which the operator will recreate using new spec
		return true, nil
	}

	patchBytes, err := generateServicePatch(cachedService, service)
	if err != nil {
		return false, fmt.Errorf("unable to generate service endpoint patch operations for %+v: %v", service, err)
	}

	if len(patchBytes) == 0 {
		log.Log.V(4).Infof("service %v is up-to-date", service.GetName())
		return false, nil
	}

	_, err = core.Services(service.Namespace).Patch(context.Background(), service.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
	if err != nil {
		return false, fmt.Errorf("unable to patch service %+v: %v", service, err)
	}

	log.Log.V(2).Infof("service %v patched", service.GetName())
	return false, nil
}

func (r *Reconciler) getSecret(secret *corev1.Secret) (*corev1.Secret, bool, error) {
	obj, exists, _ := r.stores.SecretCache.Get(secret)
	if exists {
		return obj.(*corev1.Secret), exists, nil
	}

	cachedSecret, err := r.clientset.CoreV1().Secrets(secret.Namespace).Get(context.Background(), secret.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, false, nil
		}

		return nil, false, err
	}

	return cachedSecret, true, nil
}

func certificationNeedsRotation(secret *corev1.Secret, duration *metav1.Duration, ca *tls.Certificate, renewBefore *metav1.Duration, caRenewBefore *metav1.Duration) bool {
	crt, err := components.LoadCertificates(secret)
	if err != nil {
		log.DefaultLogger().Reason(err).Infof("Failed to load certificate from secret %s, will rotate it.", secret.Name)
		return true
	}

	if secret.Annotations["kubevirt.io/duration"] != duration.String() {
		return true
	}

	rotationTime := components.NextRotationDeadline(crt, ca, renewBefore, caRenewBefore)
	// We update the certificate if it has passed its renewal timeout
	if rotationTime.Before(time.Now()) {
		return true
	}

	return false
}

func deleteService(service *corev1.Service, kvKey string, expectations *util.Expectations, core typedv1.CoreV1Interface) error {
	if service.DeletionTimestamp != nil {
		return nil
	}

	key, err := controller.KeyFunc(service)
	if err != nil {
		return err
	}

	gracePeriod := int64(0)
	deleteOptions := metav1.DeleteOptions{
		GracePeriodSeconds: &gracePeriod,
	}

	expectations.Service.AddExpectedDeletion(kvKey, key)
	err = core.Services(service.Namespace).Delete(context.Background(), service.Name, deleteOptions)
	if err != nil {
		expectations.Service.DeletionObserved(kvKey, key)
		log.Log.Errorf("Failed to delete service %+v: %v", service, err)
		return err
	}

	log.Log.V(2).Infof("service %v deleted. It must be re-created", service.GetName())
	return nil
}

func (r *Reconciler) createOrUpdateCertificateSecret(queue workqueue.TypedRateLimitingInterface[string], ca *tls.Certificate, secret *corev1.Secret, duration *metav1.Duration, renewBefore *metav1.Duration, caRenewBefore *metav1.Duration) (*tls.Certificate, error) {
	var cachedSecret *corev1.Secret
	var err error

	secret = secret.DeepCopy()
	version, imageRegistry, id := getTargetVersionRegistryID(r.kv)
	injectOperatorMetadata(r.kv, &secret.ObjectMeta, version, imageRegistry, id, true)

	log.DefaultLogger().V(4).Infof("checking certificate %v", secret.Name)

	cachedSecret, exists, err := r.getSecret(secret)
	if err != nil {
		return nil, err
	}

	rotateCertificate := false
	if exists {
		rotateCertificate = certificationNeedsRotation(cachedSecret, duration, ca, renewBefore, caRenewBefore)
	}

	// populate the secret with correct certificate
	if !exists || rotateCertificate {
		if err := components.PopulateSecretWithCertificate(secret, ca, duration); err != nil {
			return nil, err
		}
	} else {
		secret.Data = cachedSecret.Data
	}

	crt, err := components.LoadCertificates(secret)
	if err != nil {
		log.DefaultLogger().Reason(err).Infof("Failed to load certificate from secret %s.", secret.Name)
		return nil, err
	}
	// we need to ensure that we revisit certificates before they expire
	wakeupDeadline := components.NextRotationDeadline(crt, ca, renewBefore, caRenewBefore).Sub(time.Now())
	queue.AddAfter(r.kvKey, wakeupDeadline)

	if !exists {
		r.expectations.Secrets.RaiseExpectations(r.kvKey, 1, 0)
		_, err := r.clientset.CoreV1().Secrets(secret.Namespace).Create(context.Background(), secret, metav1.CreateOptions{})
		if err != nil {
			r.expectations.Secrets.LowerExpectations(r.kvKey, 1, 0)
			return nil, fmt.Errorf("unable to create secret %+v: %v", secret, err)
		}

		return crt, nil
	}

	modified := resourcemerge.BoolPtr(false)
	resourcemerge.EnsureObjectMeta(modified, &cachedSecret.ObjectMeta, secret.ObjectMeta)

	if !*modified && !rotateCertificate {
		log.Log.V(4).Infof("secret %v is up-to-date", secret.GetName())
		return crt, nil
	}

	patchBytes, err := createSecretPatch(secret)
	if err != nil {
		return nil, err
	}

	_, err = r.clientset.CoreV1().Secrets(secret.Namespace).Patch(context.Background(), secret.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
	if err != nil {
		return nil, fmt.Errorf("unable to patch secret %+v: %v", secret, err)
	}

	log.Log.V(2).Infof("secret %v updated", secret.GetName())

	return crt, nil
}

func createSecretPatch(secret *corev1.Secret) ([]byte, error) {
	// Add Labels and Annotations Patches
	ops := createLabelsAndAnnotationsPatch(&secret.ObjectMeta)
	ops = append(ops, patch.WithReplace("/data", secret.Data))

	return patch.New(ops...).GeneratePayload()
}

func (r *Reconciler) createOrUpdateCertificateSecrets(queue workqueue.TypedRateLimitingInterface[string], caCert *tls.Certificate, duration *metav1.Duration, renewBefore *metav1.Duration, caRenewBefore *metav1.Duration) error {

	for _, secret := range r.targetStrategy.CertificateSecrets() {

		// The CA certificate needs to be handled separately and before other secrets, and ignore export CA
		if secret.Name == components.KubeVirtCASecretName || secret.Name == components.KubeVirtExportCASecretName {
			continue
		}

		_, err := r.createOrUpdateCertificateSecret(queue, caCert, secret, duration, renewBefore, caRenewBefore)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *Reconciler) createOrUpdateComponentsWithCertificates(queue workqueue.TypedRateLimitingInterface[string]) error {
	caDuration := GetCADuration(r.kv.Spec.CertificateRotationStrategy.SelfSigned)
	caExportDuration := GetCADuration(r.kv.Spec.CertificateRotationStrategy.SelfSigned)
	caRenewBefore := GetCARenewBefore(r.kv.Spec.CertificateRotationStrategy.SelfSigned)
	certDuration := GetCertDuration(r.kv.Spec.CertificateRotationStrategy.SelfSigned)
	certRenewBefore := GetCertRenewBefore(r.kv.Spec.CertificateRotationStrategy.SelfSigned)
	caExportRenewBefore := GetCertRenewBefore(r.kv.Spec.CertificateRotationStrategy.SelfSigned)

	// create/update CA Certificate secret
	caCert, err := r.createOrUpdateCACertificateSecret(queue, components.KubeVirtCASecretName, caDuration, caRenewBefore)
	if err != nil {
		return err
	}

	// create/update CA Certificate secret
	caExportCert, err := r.createOrUpdateCACertificateSecret(queue, components.KubeVirtExportCASecretName, caExportDuration, caExportRenewBefore)
	if err != nil {
		return err
	}

	// create/update CA config map
	caBundle, err := r.createOrUpdateKubeVirtCAConfigMap(queue, caCert, caRenewBefore, findRequiredCAConfigMap(components.KubeVirtCASecretName, r.targetStrategy.ConfigMaps()))
	if err != nil {
		return err
	}

	// create/update export CA config map
	_, err = r.createOrUpdateKubeVirtCAConfigMap(queue, caExportCert, caExportRenewBefore, findRequiredCAConfigMap(components.KubeVirtExportCASecretName, r.targetStrategy.ConfigMaps()))
	if err != nil {
		return err
	}

	// create/update ValidatingWebhookConfiguration
	err = r.createOrUpdateValidatingWebhookConfigurations(caBundle)
	if err != nil {
		return err
	}

	// create/update MutatingWebhookConfiguration
	err = r.createOrUpdateMutatingWebhookConfigurations(caBundle)
	if err != nil {
		return err
	}

	// create/update APIServices
	err = r.createOrUpdateAPIServices(caBundle)
	if err != nil {
		return err
	}

	// create/update Routes
	err = r.createOrUpdateRoutes(caBundle)
	if err != nil {
		return err
	}

	// create/update Certificate secrets
	err = r.createOrUpdateCertificateSecrets(queue, caCert, certDuration, certRenewBefore, caRenewBefore)
	if err != nil {
		return err
	}

	return nil
}

func shouldEnforceClusterIP(desired, current string) bool {
	if desired == "" {
		return false
	}

	return desired != current
}

func getObjectMetaPatch(desired, current metav1.ObjectMeta) []patch.PatchOption {
	modified := resourcemerge.BoolPtr(false)
	existingCopy := current.DeepCopy()
	resourcemerge.EnsureObjectMeta(modified, existingCopy, desired)
	if *modified {
		// labels and/or annotations modified add patch
		return createLabelsAndAnnotationsPatch(&desired)
	}

	return nil
}

func hasImmutableFieldChanged(service, cachedService *corev1.Service) bool {
	deleteAndReplace := false

	typeSame := isServiceClusterIP(cachedService) && isServiceClusterIP(service)
	if !typeSame || shouldEnforceClusterIP(service.Spec.ClusterIP, cachedService.Spec.ClusterIP) {
		deleteAndReplace = true
	}

	return deleteAndReplace
}

func generateServicePatch(
	cachedService *corev1.Service,
	service *corev1.Service) ([]byte, error) {

	patchOps := getObjectMetaPatch(service.ObjectMeta, cachedService.ObjectMeta)

	// set these values in the case they are empty
	service.Spec.ClusterIP = cachedService.Spec.ClusterIP
	service.Spec.Type = cachedService.Spec.Type
	if service.Spec.SessionAffinity == "" {
		service.Spec.SessionAffinity = cachedService.Spec.SessionAffinity
	}

	// If the Specs don't equal each other, replace it
	if !equality.Semantic.DeepEqual(cachedService.Spec, service.Spec) {
		patchOps = append(patchOps, patch.WithReplace("/spec", service.Spec))
	}
	patchset := patch.New(patchOps...)
	if patchset.IsEmpty() {
		return nil, nil
	}
	return patchset.GeneratePayload()
}

func (r *Reconciler) createOrUpdateServiceAccount(sa *corev1.ServiceAccount) error {
	core := r.clientset.CoreV1()
	version, imageRegistry, id := getTargetVersionRegistryID(r.kv)
	injectOperatorMetadata(r.kv, &sa.ObjectMeta, version, imageRegistry, id, true)

	obj, exists, _ := r.stores.ServiceAccountCache.Get(sa)
	if !exists {
		// Create non existent
		r.expectations.ServiceAccount.RaiseExpectations(r.kvKey, 1, 0)
		_, err := core.ServiceAccounts(r.kv.Namespace).Create(context.Background(), sa, metav1.CreateOptions{})
		if err != nil {
			r.expectations.ServiceAccount.LowerExpectations(r.kvKey, 1, 0)
			return fmt.Errorf("unable to create serviceaccount %+v: %v", sa, err)
		}
		log.Log.V(2).Infof("serviceaccount %v created", sa.GetName())
		return nil
	}

	cachedSa := obj.(*corev1.ServiceAccount)
	modified := resourcemerge.BoolPtr(false)
	resourcemerge.EnsureObjectMeta(modified, &cachedSa.ObjectMeta, sa.ObjectMeta)
	// there was no change to metadata
	if !*modified {
		// Up to date
		log.Log.V(4).Infof("serviceaccount %v already exists and is up-to-date", sa.GetName())
		return nil
	}

	// Patch Labels and Annotations
	labelAnnotationPatch, err := patch.New(createLabelsAndAnnotationsPatch(&sa.ObjectMeta)...).GeneratePayload()
	if err != nil {
		return err
	}

	_, err = core.ServiceAccounts(r.kv.Namespace).Patch(context.Background(), sa.Name, types.JSONPatchType, labelAnnotationPatch, metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("unable to patch serviceaccount %+v: %v", sa, err)
	}

	log.Log.V(2).Infof("serviceaccount %v updated", sa.GetName())

	return nil
}

func (r *Reconciler) createOrUpdateRbac() error {

	version, imageRegistry, id := getTargetVersionRegistryID(r.kv)

	// create/update ServiceAccounts
	for _, sa := range r.targetStrategy.ServiceAccounts() {
		if err := r.createOrUpdateServiceAccount(sa.DeepCopy()); err != nil {
			return err
		}
	}

	// create/update ClusterRoles
	for _, cr := range r.targetStrategy.ClusterRoles() {
		err := r.createOrUpdateClusterRole(cr, version, imageRegistry, id)
		if err != nil {
			return err
		}
	}

	// create/update ClusterRoleBindings
	for _, crb := range r.targetStrategy.ClusterRoleBindings() {
		err := r.createOrUpdateClusterRoleBinding(crb, version, imageRegistry, id)
		if err != nil {
			return err
		}

	}

	// create/update Roles
	for _, role := range r.targetStrategy.Roles() {
		err := r.createOrUpdateRole(role, version, imageRegistry, id)
		if err != nil {
			return err
		}
	}

	// create/update RoleBindings
	for _, rb := range r.targetStrategy.RoleBindings() {
		err := r.createOrUpdateRoleBinding(rb, version, imageRegistry, id)
		if err != nil {
			return err
		}
	}

	return nil
}

func findRequiredCAConfigMap(name string, configmaps []*corev1.ConfigMap) *corev1.ConfigMap {
	for _, cm := range configmaps {
		if cm.Name != name {
			continue
		}

		return cm.DeepCopy()
	}

	return nil
}

func shouldUpdateBundle(required, existing *corev1.ConfigMap, key string, queue workqueue.TypedRateLimitingInterface[string], caCert *tls.Certificate, overlapInterval *metav1.Duration) (bool, error) {
	bundle, certCount, err := components.MergeCABundle(caCert, []byte(existing.Data[components.CABundleKey]), overlapInterval.Duration)
	if err != nil {
		// the only error that can be returned form MergeCABundle is if the CA caBundle
		// is unable to be parsed. If we can not parse it we should update it
		return true, err
	}

	// ensure that we remove the old CA after the overlap period
	if certCount > 1 {
		queue.AddAfter(key, overlapInterval.Duration)
	}

	updateBundle := false
	required.Data = map[string]string{components.CABundleKey: string(bundle)}
	if !equality.Semantic.DeepEqual(required.Data, existing.Data) {
		updateBundle = true
	}

	return updateBundle, nil
}

func (r *Reconciler) createOrUpdateKubeVirtCAConfigMap(queue workqueue.TypedRateLimitingInterface[string], caCert *tls.Certificate, overlapInterval *metav1.Duration, configMap *corev1.ConfigMap) (caBundle []byte, err error) {
	if configMap == nil {
		return nil, nil
	}

	log.DefaultLogger().V(4).Infof("checking ca config map %v", configMap.Name)

	version, imageRegistry, id := getTargetVersionRegistryID(r.kv)
	injectOperatorMetadata(r.kv, &configMap.ObjectMeta, version, imageRegistry, id, true)

	obj, exists, _ := r.stores.ConfigMapCache.Get(configMap)

	if !exists {
		configMap.Data = map[string]string{components.CABundleKey: string(cert.EncodeCertPEM(caCert.Leaf))}

		r.expectations.ConfigMap.RaiseExpectations(r.kvKey, 1, 0)
		_, err := r.clientset.CoreV1().ConfigMaps(configMap.Namespace).Create(context.Background(), configMap, metav1.CreateOptions{})
		if err != nil {
			r.expectations.ConfigMap.LowerExpectations(r.kvKey, 1, 0)
			return nil, fmt.Errorf("unable to create configMap %+v: %v", configMap, err)
		}

		return []byte(configMap.Data[components.CABundleKey]), nil
	}

	existing := obj.(*corev1.ConfigMap)
	updateBundle, err := shouldUpdateBundle(configMap, existing, r.kvKey, queue, caCert, overlapInterval)
	if err != nil {
		if !updateBundle {
			return nil, err
		}

		configMap.Data = map[string]string{components.CABundleKey: string(cert.EncodeCertPEM(caCert.Leaf))}
		log.Log.Reason(err).V(2).Infof("There was an error validating the CA bundle stored in configmap %s. We are updating the bundle.", configMap.GetName())
	}

	modified := resourcemerge.BoolPtr(false)
	resourcemerge.EnsureObjectMeta(modified, &existing.DeepCopy().ObjectMeta, configMap.ObjectMeta)

	if !*modified && !updateBundle {
		log.Log.V(4).Infof("configMap %v is up-to-date", configMap.GetName())
		return []byte(configMap.Data[components.CABundleKey]), nil
	}

	patchBytes, err := createConfigMapPatch(configMap)
	if err != nil {
		return nil, err
	}

	_, err = r.clientset.CoreV1().ConfigMaps(configMap.Namespace).Patch(context.Background(), configMap.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
	if err != nil {
		return nil, fmt.Errorf("unable to patch configMap %+v: %v", configMap, err)
	}

	log.Log.V(2).Infof("configMap %v updated", configMap.GetName())

	return []byte(configMap.Data[components.CABundleKey]), nil
}

func createConfigMapPatch(configMap *corev1.ConfigMap) ([]byte, error) {
	// Add Labels and Annotations Patches
	ops := createLabelsAndAnnotationsPatch(&configMap.ObjectMeta)
	ops = append(ops, patch.WithReplace("/data", configMap.Data))

	return patch.New(ops...).GeneratePayload()
}

func (r *Reconciler) createOrUpdateCACertificateSecret(queue workqueue.TypedRateLimitingInterface[string], name string, duration *metav1.Duration, renewBefore *metav1.Duration) (caCert *tls.Certificate, err error) {
	for _, secret := range r.targetStrategy.CertificateSecrets() {
		// Only work on the ca secrets
		if secret.Name != name {
			continue
		}
		cert, err := r.createOrUpdateCertificateSecret(queue, nil, secret, duration, renewBefore, nil)
		if err != nil {
			return nil, err
		}
		caCert = cert
	}
	return caCert, nil
}
