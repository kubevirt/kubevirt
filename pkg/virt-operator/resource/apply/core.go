package apply

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/openshift/library-go/pkg/operator/resource/resourcemerge"
	corev1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"

	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/certificates/triple/cert"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
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

func (r *Reconciler) createOrUpdateService() (bool, error) {

	core := r.clientset.CoreV1()
	version, imageRegistry, id := getTargetVersionRegistryID(r.kv)

	gracePeriod := int64(0)
	deleteOptions := metav1.DeleteOptions{
		GracePeriodSeconds: &gracePeriod,
	}

	for _, service := range r.targetStrategy.Services() {
		var cachedService *corev1.Service
		service = service.DeepCopy()

		obj, exists, _ := r.stores.ServiceCache.Get(service)
		if exists {
			cachedService = obj.(*corev1.Service)
		}

		injectOperatorMetadata(r.kv, &service.ObjectMeta, version, imageRegistry, id, true)
		if !exists {
			r.expectations.Service.RaiseExpectations(r.kvKey, 1, 0)
			_, err := core.Services(service.Namespace).Create(context.Background(), service, metav1.CreateOptions{})
			if err != nil {
				r.expectations.Service.LowerExpectations(r.kvKey, 1, 0)
				return false, fmt.Errorf("unable to create service %+v: %v", service, err)
			}
		} else {

			patchOps, deleteAndReplace, err := r.generateServicePatch(cachedService, service)
			if err != nil {
				return false, fmt.Errorf("unable to generate service endpoint patch operations for %+v: %v", service, err)
			}

			if deleteAndReplace {
				if cachedService.DeletionTimestamp == nil {
					if key, err := controller.KeyFunc(cachedService); err == nil {
						r.expectations.Service.AddExpectedDeletion(r.kvKey, key)
						err := core.Services(service.Namespace).Delete(context.Background(), cachedService.Name, deleteOptions)
						if err != nil {
							r.expectations.Service.DeletionObserved(r.kvKey, key)
							log.Log.Errorf("Failed to delete service %+v: %v", cachedService, err)
							return false, err
						}

						log.Log.V(2).Infof("service %v deleted. It must be re-created", cachedService.GetName())
					}
				}
				// waiting for old service to be deleted,
				// after which the operator will recreate using new spec
				return true, nil
			} else if len(patchOps) != 0 {
				_, err = core.Services(service.Namespace).Patch(context.Background(), service.Name, types.JSONPatchType, generatePatchBytes(patchOps), metav1.PatchOptions{})
				if err != nil {
					return false, fmt.Errorf("unable to patch service %+v: %v", service, err)
				}
				log.Log.V(2).Infof("service %v patched", service.GetName())
			} else {
				log.Log.V(4).Infof("service %v is up-to-date", service.GetName())
			}
		}
	}
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

func (r *Reconciler) createOrUpdateCertificateSecret(queue workqueue.RateLimitingInterface, ca *tls.Certificate, secret *corev1.Secret, duration *metav1.Duration, renewBefore *metav1.Duration, caRenewBefore *metav1.Duration) (*tls.Certificate, error) {
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
	} else if exists {
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

	ops, err := createSecretPatch(secret)
	if err != nil {
		return nil, err
	}

	_, err = r.clientset.CoreV1().Secrets(secret.Namespace).Patch(context.Background(), secret.Name, types.JSONPatchType, generatePatchBytes(ops), metav1.PatchOptions{})
	if err != nil {
		return nil, fmt.Errorf("unable to patch secret %+v: %v", secret, err)
	}

	log.Log.V(2).Infof("secret %v updated", secret.GetName())

	return crt, nil
}

func createSecretPatch(secret *corev1.Secret) ([]string, error) {
	// Add Spec Patch
	data, err := json.Marshal(secret.Data)
	if err != nil {
		return nil, err
	}

	// Patch if old version
	var ops []string

	// Add Labels and Annotations Patches
	labelAnnotationPatch, err := createLabelsAndAnnotationsPatch(&secret.ObjectMeta)
	if err != nil {
		return nil, err
	}
	ops = append(ops, labelAnnotationPatch...)

	ops = append(ops, fmt.Sprintf(`{ "op": "replace", "path": "/data", "value": %s }`, string(data)))

	return ops, nil
}

func (r *Reconciler) createOrUpdateCertificateSecrets(queue workqueue.RateLimitingInterface, caCert *tls.Certificate, duration *metav1.Duration, renewBefore *metav1.Duration, caRenewBefore *metav1.Duration) error {

	for _, secret := range r.targetStrategy.CertificateSecrets() {

		// The CA certificate needs to be handled separately and before other secrets
		if secret.Name == components.KubeVirtCASecretName {
			continue
		}

		_, err := r.createOrUpdateCertificateSecret(queue, caCert, secret, duration, renewBefore, caRenewBefore)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *Reconciler) createOrUpdateComponentsWithCertificates(queue workqueue.RateLimitingInterface) error {
	caDuration := GetCADuration(r.kv.Spec.CertificateRotationStrategy.SelfSigned)
	caRenewBefore := GetCARenewBefore(r.kv.Spec.CertificateRotationStrategy.SelfSigned)
	certDuration := GetCertDuration(r.kv.Spec.CertificateRotationStrategy.SelfSigned)
	certRenewBefore := GetCertRenewBefore(r.kv.Spec.CertificateRotationStrategy.SelfSigned)

	// create/update CA Certificate secret
	caCert, err := r.createOrUpdateCACertificateSecret(queue, caDuration, caRenewBefore)
	if err != nil {
		return err
	}

	// create/update CA config map
	caBundle, err := r.createOrUpdateKubeVirtCAConfigMap(queue, caCert, caRenewBefore)
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

	// create/update Certificate secrets
	err = r.createOrUpdateCertificateSecrets(queue, caCert, certDuration, certRenewBefore, caRenewBefore)
	if err != nil {
		return err
	}
	return nil
}

// This function determines how to process updating a service endpoint.
//
// If the update involves fields that can't be mutated, then the deleteAndReplace bool will be set.
// If the update involves patching fields, then a list of patch operations will be returned.
// If neither patchOps nor deleteAndReplace are set, then the service is already up-to-date
//
// NOTE. see the unit test that exercises this function to further learn about the expected behavior.
func (r *Reconciler) generateServicePatch(
	cachedService *corev1.Service,
	service *corev1.Service) ([]string, bool, error) {

	var patchOps []string
	var deleteAndReplace bool

	version, imageRegistry, id := getTargetVersionRegistryID(r.kv)

	// First check if there's anything to do.
	if objectMatchesVersion(&cachedService.ObjectMeta, version, imageRegistry, id, r.kv.GetGeneration()) {
		// spec and annotations are already up to date. Nothing to do
		return patchOps, false, nil
	}

	if !isServiceClusterIP(cachedService) || !isServiceClusterIP(service) {
		// we're only going to attempt to mutate Type ==ClusterIPs right now because
		// that's the only logic we have tested.

		// This means both the matching cached service and the new target service must be of
		// type "ClusterIP" for us to attempt any Patch operation
		deleteAndReplace = true
	} else if cachedService.Spec.ClusterIP != service.Spec.ClusterIP {

		if service.Spec.ClusterIP == "" {
			// clusterIP is not mutable. A ClusterIP == "" will mean one is dynamically assigned.
			// It is okay if the cached service has a ClusterIP and the target one is empty. We just
			// ensure the cached ClusterIP is carried over in the Patch.
			service.Spec.ClusterIP = cachedService.Spec.ClusterIP
		} else {

			// If both the cached service and the desired service have ClusterIPs set
			// that are not equal, our only option is to delete/replace because ClusterIPs
			// are not mutable
			deleteAndReplace = true
		}
	}

	if deleteAndReplace {
		// spec can not be merged or mutated. The only way to update is to replace.
		return patchOps, deleteAndReplace, nil
	}

	// Add Labels and Annotations Patches
	labelAnnotationPatch, err := createLabelsAndAnnotationsPatch(&service.ObjectMeta)
	if err != nil {
		return patchOps, deleteAndReplace, err
	}
	patchOps = append(patchOps, labelAnnotationPatch...)

	// Before creating the SPEC patch...
	// Ensure that we always preserve the ClusterIP value from the cached service.
	// This value is dynamically set when it is equal to "" and can't be mutated once it is set.
	if isServiceClusterIP(service) && isServiceClusterIP(cachedService) && service.Spec.ClusterIP == "" {
		service.Spec.ClusterIP = cachedService.Spec.ClusterIP
	}

	// If the Specs don't equal each other, replace it
	if !reflect.DeepEqual(cachedService.Spec, service.Spec) {
		// Add Spec Patch
		newSpec, err := json.Marshal(service.Spec)
		if err != nil {
			return patchOps, deleteAndReplace, err
		}
		patchOps = append(patchOps, fmt.Sprintf(`{ "op": "replace", "path": "/spec", "value": %s }`, string(newSpec)))
	}

	return patchOps, deleteAndReplace, nil
}

func (r *Reconciler) createOrUpdateRbac() error {

	core := r.clientset.CoreV1()

	version, imageRegistry, id := getTargetVersionRegistryID(r.kv)

	// create/update ServiceAccounts
	for _, sa := range r.targetStrategy.ServiceAccounts() {
		var cachedSa *corev1.ServiceAccount

		sa := sa.DeepCopy()
		obj, exists, _ := r.stores.ServiceAccountCache.Get(sa)
		if exists {
			cachedSa = obj.(*corev1.ServiceAccount)
		}

		injectOperatorMetadata(r.kv, &sa.ObjectMeta, version, imageRegistry, id, true)
		if !exists {
			// Create non existent
			r.expectations.ServiceAccount.RaiseExpectations(r.kvKey, 1, 0)
			_, err := core.ServiceAccounts(r.kv.Namespace).Create(context.Background(), sa, metav1.CreateOptions{})
			if err != nil {
				r.expectations.ServiceAccount.LowerExpectations(r.kvKey, 1, 0)
				return fmt.Errorf("unable to create serviceaccount %+v: %v", sa, err)
			}
			log.Log.V(2).Infof("serviceaccount %v created", sa.GetName())
		} else if !objectMatchesVersion(&cachedSa.ObjectMeta, version, imageRegistry, id, r.kv.GetGeneration()) {
			// Patch if old version
			var ops []string

			// Patch Labels and Annotations
			labelAnnotationPatch, err := createLabelsAndAnnotationsPatch(&sa.ObjectMeta)
			if err != nil {
				return err
			}
			ops = append(ops, labelAnnotationPatch...)

			_, err = core.ServiceAccounts(r.kv.Namespace).Patch(context.Background(), sa.Name, types.JSONPatchType, generatePatchBytes(ops), metav1.PatchOptions{})
			if err != nil {
				return fmt.Errorf("unable to patch serviceaccount %+v: %v", sa, err)
			}
			log.Log.V(2).Infof("serviceaccount %v updated", sa.GetName())

		} else {
			// Up to date
			log.Log.V(4).Infof("serviceaccount %v already exists and is up-to-date", sa.GetName())
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

func (r *Reconciler) createOrUpdateConfigMaps() error {
	core := r.clientset.CoreV1()
	version, imageRegistry, id := getTargetVersionRegistryID(r.kv)

	for _, cm := range r.targetStrategy.ConfigMaps() {

		if cm.Name == components.KubeVirtCASecretName {
			continue
		}

		var cachedCM *corev1.ConfigMap

		cm := cm.DeepCopy()
		obj, exists, _ := r.stores.ConfigMapCache.Get(cm)
		if exists {
			cachedCM = obj.(*corev1.ConfigMap)
		}

		injectOperatorMetadata(r.kv, &cm.ObjectMeta, version, imageRegistry, id, true)
		if !exists {
			// Create non existent
			r.expectations.ConfigMap.RaiseExpectations(r.kvKey, 1, 0)
			_, err := core.ConfigMaps(cm.Namespace).Create(context.Background(), cm, metav1.CreateOptions{})
			if err != nil {
				r.expectations.ConfigMap.LowerExpectations(r.kvKey, 1, 0)
				return fmt.Errorf("unable to create config map %+v: %v", cm, err)
			}
			log.Log.V(2).Infof("config map %v created", cm.GetName())

		} else if !objectMatchesVersion(&cachedCM.ObjectMeta, version, imageRegistry, id, r.kv.GetGeneration()) {
			// Patch if old version
			var ops []string

			// Add Labels and Annotations Patches
			labelAnnotationPatch, err := createLabelsAndAnnotationsPatch(&cm.ObjectMeta)
			if err != nil {
				return err
			}
			ops = append(ops, labelAnnotationPatch...)

			// Add Spec Patch
			newSpec, err := json.Marshal(cm.Data)
			if err != nil {
				return err
			}
			ops = append(ops, fmt.Sprintf(`{ "op": "replace", "path": "/spec", "value": %s }`, string(newSpec)))

			_, err = core.ConfigMaps(cm.Namespace).Patch(context.Background(), cm.Name, types.JSONPatchType, generatePatchBytes(ops), metav1.PatchOptions{})
			if err != nil {
				return fmt.Errorf("unable to patch config map %+v: %v", cm, err)
			}
			log.Log.V(2).Infof("config map %v updated", cm.GetName())

		} else {
			log.Log.V(4).Infof("config map %v is up-to-date", cm.GetName())
		}
	}

	return nil
}

func (r *Reconciler) createOrUpdateKubeVirtCAConfigMap(queue workqueue.RateLimitingInterface, caCert *tls.Certificate, overlapInterval *metav1.Duration) (caBundle []byte, err error) {

	for _, configMap := range r.targetStrategy.ConfigMaps() {

		if configMap.Name != components.KubeVirtCASecretName {
			continue
		}

		var cachedConfigMap *corev1.ConfigMap
		configMap = configMap.DeepCopy()

		log.DefaultLogger().V(4).Infof("checking ca config map %v", configMap.Name)

		version, imageRegistry, id := getTargetVersionRegistryID(r.kv)
		obj, exists, _ := r.stores.ConfigMapCache.Get(configMap)

		updateBundle := false
		if exists {
			cachedConfigMap = obj.(*corev1.ConfigMap)

			bundle, certCount, err := components.MergeCABundle(caCert, []byte(cachedConfigMap.Data[components.CABundleKey]), overlapInterval.Duration)
			if err != nil {
				return nil, err
			}

			// ensure that we remove the old CA after the overlap period
			if certCount > 1 {
				queue.AddAfter(r.kvKey, overlapInterval.Duration)
			}

			configMap.Data = map[string]string{components.CABundleKey: string(bundle)}

			if !reflect.DeepEqual(configMap.Data, cachedConfigMap.Data) {
				updateBundle = true
			}
		} else {
			configMap.Data = map[string]string{components.CABundleKey: string(cert.EncodeCertPEM(caCert.Leaf))}
		}

		injectOperatorMetadata(r.kv, &configMap.ObjectMeta, version, imageRegistry, id, true)
		if !exists {
			r.expectations.ConfigMap.RaiseExpectations(r.kvKey, 1, 0)
			_, err := r.clientset.CoreV1().ConfigMaps(configMap.Namespace).Create(context.Background(), configMap, metav1.CreateOptions{})
			if err != nil {
				r.expectations.ConfigMap.LowerExpectations(r.kvKey, 1, 0)
				return nil, fmt.Errorf("unable to create configMap %+v: %v", configMap, err)
			}
		} else {
			if !objectMatchesVersion(&cachedConfigMap.ObjectMeta, version, imageRegistry, id, r.kv.GetGeneration()) || updateBundle {
				// Patch if old version
				var ops []string

				// Add Labels and Annotations Patches
				labelAnnotationPatch, err := createLabelsAndAnnotationsPatch(&configMap.ObjectMeta)
				if err != nil {
					return nil, err
				}
				ops = append(ops, labelAnnotationPatch...)

				// Add Spec Patch
				data, err := json.Marshal(configMap.Data)
				if err != nil {
					return nil, err
				}
				ops = append(ops, fmt.Sprintf(`{ "op": "replace", "path": "/data", "value": %s }`, string(data)))

				_, err = r.clientset.CoreV1().ConfigMaps(configMap.Namespace).Patch(context.Background(), configMap.Name, types.JSONPatchType, generatePatchBytes(ops), metav1.PatchOptions{})
				if err != nil {
					return nil, fmt.Errorf("unable to patch configMap %+v: %v", configMap, err)
				}
				log.Log.V(2).Infof("configMap %v updated", configMap.GetName())
			} else {
				log.Log.V(4).Infof("configMap %v is up-to-date", configMap.GetName())
			}
		}
		return []byte(configMap.Data[components.CABundleKey]), nil
	}
	return nil, nil
}

func (r *Reconciler) createOrUpdateCACertificateSecret(queue workqueue.RateLimitingInterface, duration *metav1.Duration, renewBefore *metav1.Duration) (caCert *tls.Certificate, err error) {

	for _, secret := range r.targetStrategy.CertificateSecrets() {
		// Only work on the ca secret
		if secret.Name != components.KubeVirtCASecretName {
			continue
		}

		return r.createOrUpdateCertificateSecret(queue, nil, secret, duration, renewBefore, nil)
	}

	return nil, nil
}
