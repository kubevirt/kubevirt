package apply

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
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

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/certificates/triple/cert"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/network/multus"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
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

func getValidCerts(certs []*x509.Certificate) []*tls.Certificate {
	externalCAList := make([]*tls.Certificate, 0)
	certMap := make(map[string]*x509.Certificate)
	for _, cert := range certs {
		hash := sha256.Sum256(cert.Raw)
		hashString := hex.EncodeToString(hash[:])
		if _, ok := certMap[hashString]; ok {
			continue
		}
		tlsCert := &tls.Certificate{
			Leaf: cert,
		}
		now := time.Now()
		if now.After(cert.NotBefore) && now.Before(cert.NotAfter) {
			// cert is still valid
			log.Log.V(2).Infof("adding CA from external CA list because it is still valid %s, %s, now %s, not before %s, not after %s", cert.Issuer, cert.Subject, now.Format(time.RFC3339), cert.NotBefore.Format(time.RFC3339), cert.NotAfter.Format(time.RFC3339))
			externalCAList = append(externalCAList, tlsCert)
			certMap[hashString] = cert
		} else if now.Before(cert.NotBefore) {
			log.Log.V(2).Infof("skipping CA from external CA because it is not yet valid %s, %s", cert.Issuer, cert.Subject)
		} else if now.After(cert.NotAfter) {
			log.Log.V(2).Infof("skipping CA from external CA because it is expired %s, %s", cert.Issuer, cert.Subject)
		}
	}
	return externalCAList
}

func (r *Reconciler) getRemotePublicCas() []*tls.Certificate {
	obj, exists, err := r.stores.ConfigMapCache.GetByKey(controller.NamespacedKey(r.kv.Namespace, components.ExternalKubeVirtCAConfigMapName))
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get external CA configmap")
		return nil
	} else if !exists {
		log.Log.V(3).Infof("external CA configmap does not exist, returning empty list")
		return nil
	}
	externalCaConfigMap := obj.(*corev1.ConfigMap)
	externalCAData := externalCaConfigMap.Data[components.CABundleKey]
	if externalCAData == "" {
		log.Log.V(3).Infof("external CA configmap is empty, returning empty list")
		return nil
	}
	log.Log.V(4).Infof("external CA data: %s", externalCAData)
	certs, err := cert.ParseCertsPEM([]byte(externalCAData))
	if err != nil {
		log.Log.Infof("failed to parse external CA certificates: %v", err)
		return nil
	}
	return getValidCerts(certs)
}

func (r *Reconciler) cleanupExternalCACerts(configMap *corev1.ConfigMap) error {
	if configMap == nil {
		return nil
	}
	version, imageRegistry, id := getTargetVersionRegistryID(r.kv)
	injectOperatorMetadata(r.kv, &configMap.ObjectMeta, version, imageRegistry, id, true)

	configMap.Data = map[string]string{components.CABundleKey: ""}
	_, exists, _ := r.stores.ConfigMapCache.Get(configMap)
	if !exists {
		_, err := r.clientset.CoreV1().ConfigMaps(configMap.Namespace).Create(context.Background(), configMap, metav1.CreateOptions{})
		if err != nil && !errors.IsAlreadyExists(err) {
			return fmt.Errorf("unable to create configMap %+v: %v", configMap, err)
		}
	} else {
		_, err := r.clientset.CoreV1().ConfigMaps(configMap.Namespace).Update(context.Background(), configMap, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("unable to update configMap %+v: %v", configMap, err)
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

	// create/update export CA Certificate secret
	caExportCert, err := r.createOrUpdateCACertificateSecret(queue, components.KubeVirtExportCASecretName, caExportDuration, caExportRenewBefore)
	if err != nil {
		return err
	}

	err = r.createExternalKubeVirtCAConfigMap(findRequiredCAConfigMap(components.ExternalKubeVirtCAConfigMapName, r.targetStrategy.ConfigMaps()))
	if err != nil {
		return err
	}

	log.Log.V(3).Info("reading external CA configmap")
	externalCACerts := r.getRemotePublicCas()
	log.Log.V(3).Infof("found %d external CA certificates", len(externalCACerts))
	// create/update CA config map
	caBundle, err := r.createOrUpdateKubeVirtCAConfigMap(queue, caCert, externalCACerts, caRenewBefore, findRequiredCAConfigMap(components.KubeVirtCASecretName, r.targetStrategy.ConfigMaps()))
	if err != nil {
		return err
	}

	err = r.cleanupExternalCACerts(findRequiredCAConfigMap(components.ExternalKubeVirtCAConfigMapName, r.targetStrategy.ConfigMaps()))
	if err != nil {
		return err
	}

	// create/update export CA config map
	_, err = r.createOrUpdateKubeVirtCAConfigMap(queue, caExportCert, nil, caExportRenewBefore, findRequiredCAConfigMap(components.KubeVirtExportCASecretName, r.targetStrategy.ConfigMaps()))
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

func buildCertMap(certs []*x509.Certificate) map[string]*x509.Certificate {
	certMap := make(map[string]*x509.Certificate)
	for _, cert := range certs {
		hash := sha256.Sum256(cert.Raw)
		hashString := hex.EncodeToString(hash[:])
		certMap[hashString] = cert
	}
	return certMap
}

func shouldUpdateBundle(required, existing *corev1.ConfigMap, key string, queue workqueue.TypedRateLimitingInterface[string], caCert *tls.Certificate, externalCACerts []*tls.Certificate, overlapInterval *metav1.Duration) (bool, error) {
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

	bundleCerts, err := cert.ParseCertsPEM(bundle)
	if err != nil {
		return true, err
	}
	bundleCertMap := buildCertMap(bundleCerts)

	bundleString := string(bundle)
	for _, externalCACert := range externalCACerts {
		pem := string(cert.EncodeCertPEM(externalCACert.Leaf))
		hash := sha256.Sum256(externalCACert.Leaf.Raw)
		hashString := hex.EncodeToString(hash[:])
		if _, ok := bundleCertMap[hashString]; ok {
			continue
		}
		bundleCertMap[hashString] = externalCACert.Leaf
		bundleString += pem
	}
	required.Data = map[string]string{components.CABundleKey: bundleString}
	return !equality.Semantic.DeepEqual(required.Data, existing.Data), nil
}

func (r *Reconciler) createExternalKubeVirtCAConfigMap(configMap *corev1.ConfigMap) error {
	if configMap == nil {
		log.Log.V(2).Infof("cannot create external CA configmap because it is nil")
		return nil
	}
	_, exists, _ := r.stores.ConfigMapCache.Get(configMap)
	if !exists {
		log.Log.V(4).Infof("checking external ca config map %v", configMap.Name)

		version, imageRegistry, id := getTargetVersionRegistryID(r.kv)
		injectOperatorMetadata(r.kv, &configMap.ObjectMeta, version, imageRegistry, id, true)

		configMap.Data = map[string]string{components.CABundleKey: ""}
		_, err := r.clientset.CoreV1().ConfigMaps(configMap.Namespace).Create(context.Background(), configMap, metav1.CreateOptions{})
		if err != nil && !errors.IsAlreadyExists(err) {
			return fmt.Errorf("unable to create configMap %+v: %v", configMap, err)
		}
	}
	return nil
}

func (r *Reconciler) createOrUpdateKubeVirtCAConfigMap(queue workqueue.TypedRateLimitingInterface[string], caCert *tls.Certificate, externalCACerts []*tls.Certificate, overlapInterval *metav1.Duration, configMap *corev1.ConfigMap) (caBundle []byte, err error) {
	if configMap == nil {
		return nil, nil
	}

	log.DefaultLogger().V(4).Infof("checking ca config map %v", configMap.Name)

	version, imageRegistry, id := getTargetVersionRegistryID(r.kv)
	injectOperatorMetadata(r.kv, &configMap.ObjectMeta, version, imageRegistry, id, true)

	data := ""
	obj, exists, _ := r.stores.ConfigMapCache.Get(configMap)
	if !exists {
		for _, externalCert := range externalCACerts {
			data = data + string(cert.EncodeCertPEM(externalCert.Leaf))
		}
		data = data + string(cert.EncodeCertPEM(caCert.Leaf))
		configMap.Data = map[string]string{components.CABundleKey: data}
		r.expectations.ConfigMap.RaiseExpectations(r.kvKey, 1, 0)
		_, err := r.clientset.CoreV1().ConfigMaps(configMap.Namespace).Create(context.Background(), configMap, metav1.CreateOptions{})
		if err != nil {
			r.expectations.ConfigMap.LowerExpectations(r.kvKey, 1, 0)
			return nil, fmt.Errorf("unable to create configMap %+v: %v", configMap, err)
		}

		return []byte(configMap.Data[components.CABundleKey]), nil
	}

	existing := obj.(*corev1.ConfigMap)
	updateBundle, err := shouldUpdateBundle(configMap, existing, r.kvKey, queue, caCert, externalCACerts, overlapInterval)
	if err != nil {
		if !updateBundle {
			return nil, err
		}
		data = data + string(cert.EncodeCertPEM(caCert.Leaf))
		configMap.Data = map[string]string{components.CABundleKey: data}
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

func (r *Reconciler) updateSynchronizationAddress() (err error) {
	if !r.isFeatureGateEnabled(featuregate.DecentralizedLiveMigration) {
		r.kv.Status.SynchronizationAddresses = nil
		return nil
	}
	// Find the lease associated with the virt-synchronization controller
	lease, err := r.clientset.CoordinationV1().Leases(r.kv.Namespace).Get(context.Background(), components.VirtSynchronizationControllerName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	podName := ""
	if lease.Spec.HolderIdentity != nil {
		podName = *lease.Spec.HolderIdentity
	}
	if podName == "" {
		return nil
	}
	pod, err := r.clientset.CoreV1().Pods(r.kv.Namespace).Get(context.Background(), podName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}

	// Check for the migration network address in the pod annotations.
	ips := r.getIpsFromAnnotations(pod)
	if len(ips) == 0 && pod.Status.PodIPs != nil {
		// Did not find annotations, use the pod ip address instead
		ips = make([]string, len(pod.Status.PodIPs))
		for i, podIP := range pod.Status.PodIPs {
			ips[i] = podIP.IP
		}
	}
	if len(ips) == 0 {
		return nil
	}
	port := util.DefaultSynchronizationPort
	if r.kv.Spec.SynchronizationPort != "" {
		p, err := strconv.Atoi(r.kv.Spec.SynchronizationPort)
		if err != nil {
			return err
		}
		port = int32(p)
	}
	addresses := make([]string, len(ips))
	for i, ip := range ips {
		addresses[i] = fmt.Sprintf("%s:%d", ip, port)
	}
	r.kv.Status.SynchronizationAddresses = addresses
	return nil
}

func (r *Reconciler) getIpsFromAnnotations(pod *corev1.Pod) []string {
	networkStatuses := multus.NetworkStatusesFromPod(pod)
	for _, networkStatus := range networkStatuses {
		if networkStatus.Interface == v1.MigrationInterfaceName {
			if len(networkStatus.IPs) == 0 {
				break
			}
			log.Log.Object(pod).V(4).Infof("found migration network ip addresses %v", networkStatus.IPs)
			return networkStatus.IPs
		}
	}
	log.Log.Object(pod).V(4).Infof("didn't find migration network ip in annotations %v", pod.Annotations)
	return nil
}
