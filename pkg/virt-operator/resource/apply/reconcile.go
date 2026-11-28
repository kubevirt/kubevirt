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
 *
 */

package apply

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/coreos/go-semver/semver"
	secv1 "github.com/openshift/api/security/v1"
	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	apiregv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/certificates/triple"
	"kubevirt.io/kubevirt/pkg/certificates/triple/cert"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/install"
	"kubevirt.io/kubevirt/pkg/virt-operator/util"
)

const Duration7d = time.Hour * 24 * 7
const Duration1d = time.Hour * 24

func objectMatchesVersion(objectMeta *metav1.ObjectMeta, version, imageRegistry, id string, generation int64) bool {
	if objectMeta.Annotations == nil {
		return false
	}

	foundVersion, foundImageRegistry, foundID, _ := getInstallStrategyAnnotations(objectMeta)
	foundGeneration, generationExists := objectMeta.Annotations[v1.KubeVirtGenerationAnnotation]
	foundLabels := util.IsManagedByOperator(objectMeta.Labels)
	sGeneration := strconv.FormatInt(generation, 10)

	if generationExists && foundGeneration != sGeneration {
		return false
	}

	if foundVersion == version && foundImageRegistry == imageRegistry && foundID == id && foundLabels {
		return true
	}

	return false
}

func injectOperatorMetadata(kv *v1.KubeVirt, objectMeta *metav1.ObjectMeta, version string, imageRegistry string, id string, injectCustomizationMetadata bool) {
	if objectMeta.Labels == nil {
		objectMeta.Labels = make(map[string]string)
	}
	if kv.Spec.ProductVersion != "" && util.IsValidLabel(kv.Spec.ProductVersion) {
		objectMeta.Labels[v1.AppVersionLabel] = kv.Spec.ProductVersion
	}

	if kv.Spec.ProductName != "" && util.IsValidLabel(kv.Spec.ProductName) {
		objectMeta.Labels[v1.AppPartOfLabel] = kv.Spec.ProductName
	}

	objectMeta.Labels[v1.AppComponentLabel] = GetAppComponent(kv)

	objectMeta.Labels[v1.ManagedByLabel] = v1.ManagedByLabelOperatorValue

	if objectMeta.Annotations == nil {
		objectMeta.Annotations = make(map[string]string)
	}
	objectMeta.Annotations[v1.InstallStrategyVersionAnnotation] = version
	objectMeta.Annotations[v1.InstallStrategyRegistryAnnotation] = imageRegistry
	objectMeta.Annotations[v1.InstallStrategyIdentifierAnnotation] = id
	if injectCustomizationMetadata {
		objectMeta.Annotations[v1.KubeVirtGenerationAnnotation] = strconv.FormatInt(kv.ObjectMeta.GetGeneration(), 10)
	}
}

func GetAppComponent(kv *v1.KubeVirt) string {
	if kv.Spec.ProductComponent != "" && util.IsValidLabel(kv.Spec.ProductComponent) {
		return kv.Spec.ProductComponent
	}
	return v1.AppComponent
}

func createLabelsAndAnnotationsPatch(objectMeta *metav1.ObjectMeta) []patch.PatchOption {
	return []patch.PatchOption{patch.WithAdd("/metadata/labels", objectMeta.Labels),
		patch.WithAdd("/metadata/annotations", objectMeta.Annotations),
		patch.WithAdd("/metadata/ownerReferences", objectMeta.OwnerReferences)}
}

func getPatchWithObjectMetaAndSpec(ops []patch.PatchOption, meta *metav1.ObjectMeta, spec interface{}) []patch.PatchOption {
	// Add Labels and Annotations Patches
	ops = append(ops, createLabelsAndAnnotationsPatch(meta)...)
	// and spec replacement to patch
	return append(ops, patch.WithReplace("/spec", spec))
}

func shouldTakeUpdatePath(targetVersion, currentVersion string) bool {
	// if no current version, then this can't be an update
	if currentVersion == "" {
		return false
	}

	// semver doesn't like the 'v' prefix
	targetVersion = strings.TrimPrefix(targetVersion, "v")
	currentVersion = strings.TrimPrefix(currentVersion, "v")

	// our default position is that this is an update.
	// So if the target and current version do not
	// adhere to the semver spec, we assume by default the
	// update path is the correct path.
	shouldTakeUpdatePath := true
	target, err := semver.NewVersion(targetVersion)
	if err == nil {
		current, err := semver.NewVersion(currentVersion)
		if err == nil {
			if target.Compare(*current) <= 0 {
				shouldTakeUpdatePath = false
			}
		}
	}

	return shouldTakeUpdatePath
}

func haveApiDeploymentsRolledOver(targetStrategy install.StrategyInterface, kv *v1.KubeVirt, stores util.Stores) bool {
	for _, deployment := range targetStrategy.ApiDeployments() {
		if !util.DeploymentIsReady(kv, deployment, stores) {
			log.Log.V(2).Infof("Waiting on deployment %v to roll over to latest version", deployment.GetName())
			// not rolled out yet
			return false
		}
	}

	return true
}

func haveControllerDeploymentsRolledOver(targetStrategy install.StrategyInterface, kv *v1.KubeVirt, stores util.Stores) bool {
	for _, deployment := range targetStrategy.ControllerDeployments() {
		if !util.DeploymentIsReady(kv, deployment, stores) {
			log.Log.V(2).Infof("Waiting on deployment %v to roll over to latest version", deployment.GetName())
			// not rolled out yet
			return false
		}
	}

	return true
}

func haveExportProxyDeploymentsRolledOver(targetStrategy install.StrategyInterface, kv *v1.KubeVirt, stores util.Stores) bool {
	for _, deployment := range targetStrategy.ExportProxyDeployments() {
		if !util.DeploymentIsReady(kv, deployment, stores) {
			log.Log.V(2).Infof("Waiting on deployment %v to roll over to latest version", deployment.GetName())
			// not rolled out yet
			return false
		}
	}

	return true
}

func haveSynchronizationControllerDeploymentsRolledOver(targetStrategy install.StrategyInterface, kv *v1.KubeVirt, stores util.Stores) bool {
	for _, deployment := range targetStrategy.SynchronizationControllerDeployments() {
		if !util.DeploymentIsReady(kv, deployment, stores) {
			log.Log.V(2).Infof("Waiting on deployment %v to roll over to latest version", deployment.GetName())
			// not rolled out yet
			return false
		}
	}

	return true
}

func haveVirtTemplateDeploymentsRolledOver(targetStrategy install.StrategyInterface, kv *v1.KubeVirt, stores util.Stores) bool {
	for _, deployment := range targetStrategy.VirtTemplateDeployments() {
		if !util.DeploymentIsReady(kv, deployment, stores) {
			log.Log.V(2).Infof("Waiting on deployment %v to roll over to latest version", deployment.GetName())
			// not rolled out yet
			return false
		}
	}

	return true
}

func haveDaemonSetsRolledOver(targetStrategy install.StrategyInterface, kv *v1.KubeVirt, stores util.Stores) bool {
	for _, daemonSet := range targetStrategy.DaemonSets() {
		if !util.DaemonsetIsReady(kv, daemonSet, stores) {
			log.Log.V(2).Infof("Waiting on daemonset %v to roll over to latest version", daemonSet.GetName())
			// not rolled out yet
			return false
		}
	}

	return true
}

func (r *Reconciler) createDummyWebhookValidator() error {

	var webhooks []admissionregistrationv1.ValidatingWebhook

	version, imageRegistry, id := getTargetVersionRegistryID(r.kv)

	// If webhook already exists in cache, then exit.
	objects := r.stores.ValidationWebhookCache.List()
	for _, obj := range objects {
		if webhook, ok := obj.(*admissionregistrationv1.ValidatingWebhookConfiguration); ok {

			if objectMatchesVersion(&webhook.ObjectMeta, version, imageRegistry, id, r.kv.GetGeneration()) {
				// already created blocking webhook for this version
				return nil
			}
		}
	}

	// generate a fake cert. this isn't actually used
	sideEffectNone := admissionregistrationv1.SideEffectClassNone
	failurePolicy := admissionregistrationv1.Fail

	for _, crd := range r.targetStrategy.CRDs() {
		_, exists, _ := r.stores.OperatorCrdCache.Get(crd)
		if exists {
			// this CRD isn't new, it already exists in cache so we don't
			// need a blocking admission webhook to wait until the new
			// apiserver is active
			continue
		}
		path := fmt.Sprintf("/fake-path/%s", crd.Name)
		webhooks = append(webhooks, admissionregistrationv1.ValidatingWebhook{
			Name:                    fmt.Sprintf("%s-tmp-validator", crd.Name),
			AdmissionReviewVersions: []string{"v1", "v1beta1"},
			SideEffects:             &sideEffectNone,
			FailurePolicy:           &failurePolicy,
			Rules: []admissionregistrationv1.RuleWithOperations{{
				Operations: []admissionregistrationv1.OperationType{
					admissionregistrationv1.Create,
				},
				Rule: admissionregistrationv1.Rule{
					APIGroups:   []string{crd.Spec.Group},
					APIVersions: v1.ApiSupportedWebhookVersions,
					Resources:   []string{crd.Spec.Names.Plural},
				},
			}},
			ClientConfig: admissionregistrationv1.WebhookClientConfig{
				Service: &admissionregistrationv1.ServiceReference{
					Namespace: r.kv.Namespace,
					Name:      "fake-validation-service",
					Path:      &path,
				},
			},
		})
	}

	// nothing to do here if we have no new CRDs to create webhooks for
	if len(webhooks) == 0 {
		return nil
	}

	// Set some fake signing cert bytes in for each rule so the k8s apiserver will
	// allow us to create the webhook.
	caKeyPair, _ := triple.NewCA("fake.kubevirt.io", time.Hour*24)
	signingCertBytes := cert.EncodeCertPEM(caKeyPair.Cert)
	for _, webhook := range webhooks {
		webhook.ClientConfig.CABundle = signingCertBytes
	}

	validationWebhook := &admissionregistrationv1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "virt-operator-tmp-webhook",
		},
		Webhooks: webhooks,
	}
	injectOperatorMetadata(r.kv, &validationWebhook.ObjectMeta, version, imageRegistry, id, true)

	r.expectations.ValidationWebhook.RaiseExpectations(r.kvKey, 1, 0)
	_, err := r.k8sClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().Create(context.Background(), validationWebhook, metav1.CreateOptions{})
	if err != nil {
		r.expectations.ValidationWebhook.LowerExpectations(r.kvKey, 1, 0)
		return fmt.Errorf("unable to create validation webhook: %v", err)
	}
	log.Log.V(2).Infof("Validation webhook created for image %s and registry %s", version, imageRegistry)

	return nil
}

func getTargetVersionRegistryID(kv *v1.KubeVirt) (version string, registry string, id string) {
	version = kv.Status.TargetKubeVirtVersion
	registry = kv.Status.TargetKubeVirtRegistry
	id = kv.Status.TargetDeploymentID

	return
}

func isServiceClusterIP(service *corev1.Service) bool {
	if service.Spec.Type == "" || service.Spec.Type == corev1.ServiceTypeClusterIP {
		return true
	}

	return false
}

type Reconciler struct {
	kv    *v1.KubeVirt
	kvKey string

	targetStrategy   install.StrategyInterface
	stores           util.Stores
	config           util.OperatorConfig
	virtClient       kubecli.KubevirtClient
	k8sClient        kubernetes.Interface
	aggregatorclient install.APIServiceInterface
	expectations     *util.Expectations
	recorder         record.EventRecorder

	// exportProxyHPAProfileResolver selects the HPA metrics profile for export-proxy.
	// Production uses autoDetectExportProxyHPAMetricsProfile; tests may replace this.
	exportProxyHPAProfileResolver func(namespace string) components.ExportProxyHPAMetricsProfile

	exportProxyHPAMetricsProfileCache *ExportProxyHPAMetricsProfileCache

	// Memoize profile resolution for the lifetime of this reconciler (one KubeVirt sync pass).
	exportProxyHPAMetricsProfileResolved components.ExportProxyHPAMetricsProfile
}

func NewReconciler(kv *v1.KubeVirt, targetStrategy install.StrategyInterface, stores util.Stores, config util.OperatorConfig, virtClient kubecli.KubevirtClient, k8sClient kubernetes.Interface, aggregatorclient install.APIServiceInterface, expectations *util.Expectations, recorder record.EventRecorder, exportProxyHPAMetricsProfileCache *ExportProxyHPAMetricsProfileCache) (*Reconciler, error) {
	kvKey, err := controller.KeyFunc(kv)
	if err != nil {
		return nil, err
	}

	customizer, err := NewCustomizer(kv.Spec.CustomizeComponents)
	if err != nil {
		return nil, err
	}

	err = customizer.Apply(targetStrategy)
	if err != nil {
		return nil, err
	}

	reconciler := &Reconciler{
		kv:                                kv,
		kvKey:                             kvKey,
		targetStrategy:                    targetStrategy,
		stores:                            stores,
		config:                            config,
		virtClient:                        virtClient,
		k8sClient:                         k8sClient,
		aggregatorclient:                  aggregatorclient,
		expectations:                      expectations,
		recorder:                          recorder,
		exportProxyHPAMetricsProfileCache: exportProxyHPAMetricsProfileCache,
	}
	reconciler.exportProxyHPAProfileResolver = reconciler.autoDetectExportProxyHPAMetricsProfile
	return reconciler, nil
}

func (r *Reconciler) Sync(queue workqueue.TypedRateLimitingInterface[string]) (bool, error) {
	// Avoid log spam by logging this issue once early instead of for once each object created
	if !util.IsValidLabel(r.kv.Spec.ProductVersion) {
		log.Log.Errorf("invalid kubevirt.spec.productVersion: labels must be 63 characters or less, begin and end with alphanumeric characters, and contain only dot, hyphen or underscore")
	}
	if !util.IsValidLabel(r.kv.Spec.ProductName) {
		log.Log.Errorf("invalid kubevirt.spec.productName: labels must be 63 characters or less, begin and end with alphanumeric characters, and contain only dot, hyphen or underscore")
	}
	if !util.IsValidLabel(r.kv.Spec.ProductComponent) {
		log.Log.Errorf("invalid kubevirt.spec.productComponent: labels must be 63 characters or less, begin and end with alphanumeric characters, and contain only dot, hyphen or underscore")
	}

	targetVersion := r.kv.Status.TargetKubeVirtVersion
	targetImageRegistry := r.kv.Status.TargetKubeVirtRegistry

	observedVersion := r.kv.Status.ObservedKubeVirtVersion
	observedImageRegistry := r.kv.Status.ObservedKubeVirtRegistry

	apiDeploymentsRolledOver := haveApiDeploymentsRolledOver(r.targetStrategy, r.kv, r.stores)
	controllerDeploymentsRolledOver := haveControllerDeploymentsRolledOver(r.targetStrategy, r.kv, r.stores)

	exportProxyDeploymentsRolledOver := haveExportProxyDeploymentsRolledOver(r.targetStrategy, r.kv, r.stores)
	synchronizationControllerEnabled := r.isFeatureGateEnabled(featuregate.DecentralizedLiveMigration)
	synchronizationControllerDeploymentRolledOver := !synchronizationControllerEnabled || haveSynchronizationControllerDeploymentsRolledOver(r.targetStrategy, r.kv, r.stores)
	virtTemplateDeploymentEnabled := r.virtTemplateDeploymentEnabled()
	virtTemplateDeploymentsRolledOver := !virtTemplateDeploymentEnabled || haveVirtTemplateDeploymentsRolledOver(r.targetStrategy, r.kv, r.stores)

	daemonSetsRolledOver := haveDaemonSetsRolledOver(r.targetStrategy, r.kv, r.stores)

	infrastructureRolledOver := false
	if apiDeploymentsRolledOver && controllerDeploymentsRolledOver &&
		exportProxyDeploymentsRolledOver && daemonSetsRolledOver &&
		synchronizationControllerDeploymentRolledOver && virtTemplateDeploymentsRolledOver {
		// infrastructure has rolled over and is available
		infrastructureRolledOver = true
	} else if (targetVersion == observedVersion) && (targetImageRegistry == observedImageRegistry) &&
		// virt-template TLS config is injected as container args, so config-only changes trigger a rollout
		virtTemplateDeploymentsRolledOver {
		// infrastructure was observed to have rolled over successfully
		// in the past
		infrastructureRolledOver = true
	}

	// -------- CREATE AND ROLL OUT UPDATED OBJECTS --------

	// creates a blocking webhook for any new CRDs that don't exist previously.
	// this webhook is removed once the new apiserver is online.
	if !apiDeploymentsRolledOver {
		err := r.createDummyWebhookValidator()
		if err != nil {
			return false, err
		}
	} else {
		err := deleteDummyWebhookValidators(r.kv, r.k8sClient, r.stores, r.expectations)
		if err != nil {
			return false, err
		}
	}

	// create/update CRDs
	err := r.createOrUpdateCrds()
	if err != nil {
		return false, err
	}

	// create/update serviceMonitor
	err = r.createOrUpdateServiceMonitors()
	if err != nil {
		return false, err
	}

	// create/update PrometheusRules
	err = r.createOrUpdatePrometheusRules()
	if err != nil {
		return false, err
	}

	// backup any old RBAC rules that don't match current version
	if !infrastructureRolledOver {
		err = r.backupRBACs()
		if err != nil {
			return false, err
		}
	}

	// create/update all RBAC rules
	err = r.createOrUpdateRbac()
	if err != nil {
		return false, err
	}

	// create/update SCCs
	err = r.createOrUpdateSCC()
	if err != nil {
		return false, err
	}

	// create/update Services
	pending, err := r.createOrUpdateServices()
	if err != nil {
		return false, err
	} else if pending {
		// waiting on multi step service change.
		// During an update, if the 'type' of the service changes then
		// we have to delete the service, wait for the deletion to be observed,
		// then create the new service. This is because a service's "type" is
		// not mutatable.
		return false, nil
	}

	// A binding with paramRef is rejected unless its policy already exists.
	// Without the policy, the apiserver cannot tell which resource the
	// operator must be allowed to read, and requires list on every resource.
	err = r.createOrUpdateValidatingAdmissionPolicies()
	if err != nil {
		return false, err
	}

	err = r.createOrUpdateValidatingAdmissionPolicyBindings()
	if err != nil {
		return false, err
	}

	err = r.createOrUpdateComponentsWithCertificates(queue)
	if err != nil {
		return false, err
	}

	if infrastructureRolledOver {
		err = r.removeKvServiceAccountsFromDefaultSCC(r.kv.Namespace)
		if err != nil {
			return false, err
		}
	}

	if shouldTakeUpdatePath(targetVersion, observedVersion) {
		finished, err := r.updateKubeVirtSystem(controllerDeploymentsRolledOver)
		if !finished || err != nil {
			return false, err
		}
	} else {
		finished, err := r.createOrRollBackSystem(apiDeploymentsRolledOver)
		if !finished || err != nil {
			return false, err
		}
	}

	err = r.syncKubevirtNamespaceLabels()
	if err != nil {
		return false, err
	}

	if !infrastructureRolledOver {
		// still waiting on roll out before cleaning up.
		return false, nil
	}

	// -------- ROLLOUT INCOMPATIBLE CHANGES WHICH REQUIRE A FULL CONTROL PLANE ROLL OVER --------
	// some changes can only be done after the control plane rolled over
	err = r.rolloutNonCompatibleCRDChanges()
	if err != nil {
		return false, err
	}

	// -------- CLEAN UP OLD UNUSED OBJECTS --------
	// outdated webhooks can potentially block deletes of other objects during the cleanup and need to be removed first
	err = r.deleteObjectsNotInInstallStrategy()
	if err != nil {
		return false, err
	}

	if err := r.updateSynchronizationAddress(); err != nil {
		return false, err
	}
	if r.commonInstancetypesDeploymentEnabled() {
		if err := r.createOrUpdateInstancetypes(); err != nil {
			return false, err
		}
		if err := r.createOrUpdatePreferences(); err != nil {
			return false, err
		}
	} else {
		if err := r.deleteInstancetypes(); err != nil {
			return false, err
		}
		if err := r.deletePreferences(); err != nil {
			return false, err
		}
	}

	return true, nil
}

func (r *Reconciler) createOrRollBackSystem(apiDeploymentsRolledOver bool) (bool, error) {
	// CREATE/ROLLBACK PATH IS
	// 1. apiserver - ensures validation of objects occur before allowing any control plane to act on them.
	// 2. wait for apiservers to roll over
	// 3. controllers and daemonsets

	// create/update API Deployments
	for _, deployment := range r.targetStrategy.ApiDeployments() {
		deployment, err := r.syncDeployment(deployment)
		if err != nil {
			return false, err
		}
		err = r.syncPodDisruptionBudgetForDeployment(deployment)
		if err != nil {
			return false, err
		}
	}

	// wait on api servers to roll over
	if !apiDeploymentsRolledOver {
		// not rolled out yet
		return false, nil
	}

	// create/update Controller Deployments
	for _, deployment := range r.targetStrategy.ControllerDeployments() {
		deployment, err := r.syncDeployment(deployment)
		if err != nil {
			return false, err
		}
		err = r.syncPodDisruptionBudgetForDeployment(deployment)
		if err != nil {
			return false, err
		}
	}

	// create/update ExportProxy Deployments
	desiredExportProxyReplicas, err := getDesiredApiReplicas(r.k8sClient)
	if err != nil {
		return false, fmt.Errorf("failed to determine export-proxy replicas: %w", err)
	}
	for _, deployment := range r.targetStrategy.ExportProxyDeployments() {
		deployment = deployment.DeepCopy()
		r.applyExportProxyReplicaPolicy(deployment, desiredExportProxyReplicas)
		deployment, err := r.syncDeployment(deployment)
		if err != nil {
			return false, err
		}
		err = r.syncExportProxyPodDisruptionBudget(deployment, desiredExportProxyReplicas)
		if err != nil {
			return false, err
		}
		err = r.syncExportProxyHorizontalPodAutoscaler(deployment, desiredExportProxyReplicas)
		if err != nil {
			return false, err
		}
	}

	// create/update Synchronization controller Deployments
	for _, deployment := range r.targetStrategy.SynchronizationControllerDeployments() {
		if r.isFeatureGateEnabled(featuregate.DecentralizedLiveMigration) {
			deployment, err := r.syncDeployment(deployment)
			if err != nil {
				return false, err
			}
			err = r.syncPodDisruptionBudgetForDeployment(deployment)
			if err != nil {
				return false, err
			}
		} else if err := r.deleteDeployment(deployment); err != nil {
			return false, err
		}
	}

	// create/update virt-template Deployments
	for _, deployment := range r.targetStrategy.VirtTemplateDeployments() {
		if r.virtTemplateDeploymentEnabled() {
			deployment, err := r.syncDeployment(deployment)
			if err != nil {
				return false, err
			}
			err = r.syncPodDisruptionBudgetForDeployment(deployment)
			if err != nil {
				return false, err
			}
		} else if err := r.deleteDeployment(deployment); err != nil {
			return false, err
		}
	}

	// create/update Daemonsets
	for _, daemonSet := range r.targetStrategy.DaemonSets() {
		finished, err := r.syncDaemonSet(daemonSet)
		if !finished || err != nil {
			return false, err
		}
	}

	return true, nil
}

func (r *Reconciler) deleteDeployment(deployment *appsv1.Deployment) error {
	if err := r.deletePodDisruptionBudgetForDeployment(deployment); err != nil {
		return err
	}

	obj, exists, err := r.stores.DeploymentCache.Get(deployment)
	if err != nil {
		return err
	}

	if !exists || obj.(*appsv1.Deployment).DeletionTimestamp != nil {
		return nil
	}

	key, err := controller.KeyFunc(deployment)
	if err != nil {
		return err
	}
	r.expectations.Deployment.AddExpectedDeletion(r.kvKey, key)
	if err := r.k8sClient.AppsV1().Deployments(deployment.Namespace).Delete(context.Background(), deployment.Name, metav1.DeleteOptions{}); err != nil {
		r.expectations.Deployment.DeletionObserved(r.kvKey, key)
		return err
	}

	return nil
}

func (r *Reconciler) deleteObjectsNotInInstallStrategy() error {
	gracePeriod := int64(0)
	deleteOptions := metav1.DeleteOptions{GracePeriodSeconds: &gracePeriod}
	client := r.virtClient.ExtensionsClient()

	// Outdated webhooks can block deletion of other resources, so remove them first.
	cleanups := []func() error{
		obsoleteObjectsCleanup(
			r.stores.ValidationWebhookCache, r.targetStrategy.ValidatingWebhookConfigurations(), false,
			func(webhook *admissionregistrationv1.ValidatingWebhookConfiguration) bool {
				return strings.HasPrefix(webhook.Name, "virt-operator-tmp-webhook")
			},
			r.expectations.ValidationWebhook, r.kvKey, func(webhook *admissionregistrationv1.ValidatingWebhookConfiguration) error {
				return r.k8sClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().Delete(context.Background(), webhook.Name, deleteOptions)
			},
		),
		obsoleteObjectsCleanup(r.stores.MutatingWebhookCache, r.targetStrategy.MutatingWebhookConfigurations(), false, nil,
			r.expectations.MutatingWebhook, r.kvKey, func(webhook *admissionregistrationv1.MutatingWebhookConfiguration) error {
				return r.k8sClient.AdmissionregistrationV1().MutatingWebhookConfigurations().Delete(context.Background(), webhook.Name, deleteOptions)
			}),
		obsoleteObjectsCleanup(r.stores.APIServiceCache, r.targetStrategy.APIServices(), false, nil,
			r.expectations.APIService, r.kvKey, func(apiService *apiregv1.APIService) error {
				return r.aggregatorclient.Delete(context.Background(), apiService.Name, deleteOptions)
			}),
		obsoleteObjectsCleanup(r.stores.SecretCache, r.targetStrategy.CertificateSecrets(), false, nil,
			r.expectations.Secrets, r.kvKey, func(secret *corev1.Secret) error {
				return r.k8sClient.CoreV1().Secrets(secret.Namespace).Delete(context.Background(), secret.Name, deleteOptions)
			}),
		obsoleteObjectsCleanup(r.stores.ConfigMapCache, r.targetStrategy.ConfigMaps(), false, nil,
			r.expectations.ConfigMap, r.kvKey, func(configMap *corev1.ConfigMap) error {
				return r.k8sClient.CoreV1().ConfigMaps(configMap.Namespace).Delete(context.Background(), configMap.Name, deleteOptions)
			}),
		obsoleteObjectsCleanup(
			r.stores.ValidatingAdmissionPolicyBindingCache, r.targetStrategy.ValidatingAdmissionPolicyBindings(), false, nil,
			r.expectations.ValidatingAdmissionPolicyBinding, r.kvKey,
			func(validatingAdmissionPolicyBinding *admissionregistrationv1.ValidatingAdmissionPolicyBinding) error {
				return r.k8sClient.AdmissionregistrationV1().ValidatingAdmissionPolicyBindings().
					Delete(context.Background(), validatingAdmissionPolicyBinding.Name, deleteOptions)
			},
		),
		obsoleteObjectsCleanup(
			r.stores.ValidatingAdmissionPolicyCache, r.targetStrategy.ValidatingAdmissionPolicies(), false, nil,
			r.expectations.ValidatingAdmissionPolicy, r.kvKey,
			func(validatingAdmissionPolicy *admissionregistrationv1.ValidatingAdmissionPolicy) error {
				return r.k8sClient.AdmissionregistrationV1().ValidatingAdmissionPolicies().
					Delete(context.Background(), validatingAdmissionPolicy.Name, deleteOptions)
			},
		),
		obsoleteObjectsCleanup(r.stores.OperatorCrdCache, r.targetStrategy.CRDs(), false, nil,
			r.expectations.OperatorCrd, r.kvKey, func(crd *extv1.CustomResourceDefinition) error {
				return client.ApiextensionsV1().CustomResourceDefinitions().Delete(context.Background(), crd.Name, deleteOptions)
			}),
		obsoleteObjectsCleanup(r.stores.DaemonSetCache, r.targetStrategy.DaemonSets(), true, nil,
			r.expectations.DaemonSet, r.kvKey, func(ds *appsv1.DaemonSet) error {
				return r.k8sClient.AppsV1().DaemonSets(ds.Namespace).Delete(context.Background(), ds.Name, deleteOptions)
			}),
		obsoleteObjectsCleanup(r.stores.DeploymentCache, r.targetStrategy.Deployments(), true, nil,
			r.expectations.Deployment, r.kvKey, func(deployment *appsv1.Deployment) error {
				return r.k8sClient.AppsV1().Deployments(deployment.Namespace).Delete(context.Background(), deployment.Name, deleteOptions)
			}),
		obsoleteObjectsCleanup(r.stores.ServiceCache, r.targetStrategy.Services(), true, nil,
			r.expectations.Service, r.kvKey, func(svc *corev1.Service) error {
				return r.k8sClient.CoreV1().Services(svc.Namespace).Delete(context.Background(), svc.Name, deleteOptions)
			}),
		obsoleteObjectsCleanup(r.stores.ClusterRoleBindingCache, r.targetStrategy.ClusterRoleBindings(), true, nil,
			r.expectations.ClusterRoleBinding, r.kvKey, func(crb *rbacv1.ClusterRoleBinding) error {
				return r.k8sClient.RbacV1().ClusterRoleBindings().Delete(context.Background(), crb.Name, deleteOptions)
			}),
		obsoleteObjectsCleanup(r.stores.ClusterRoleCache, r.targetStrategy.ClusterRoles(), true, nil,
			r.expectations.ClusterRole, r.kvKey, func(cr *rbacv1.ClusterRole) error {
				return r.k8sClient.RbacV1().ClusterRoles().Delete(context.Background(), cr.Name, deleteOptions)
			}),
		obsoleteObjectsCleanup(r.stores.RoleBindingCache, r.targetStrategy.RoleBindings(), true, nil,
			r.expectations.RoleBinding, r.kvKey, func(rb *rbacv1.RoleBinding) error {
				return r.k8sClient.RbacV1().RoleBindings(rb.Namespace).Delete(context.Background(), rb.Name, deleteOptions)
			}),
		obsoleteObjectsCleanup(r.stores.RoleCache, r.targetStrategy.Roles(), true, nil,
			r.expectations.Role, r.kvKey, func(role *rbacv1.Role) error {
				return r.k8sClient.RbacV1().Roles(role.Namespace).Delete(context.Background(), role.Name, deleteOptions)
			}),
		obsoleteObjectsCleanup(r.stores.ServiceAccountCache, r.targetStrategy.ServiceAccounts(), true, nil,
			r.expectations.ServiceAccount, r.kvKey, func(sa *corev1.ServiceAccount) error {
				return r.k8sClient.CoreV1().ServiceAccounts(sa.Namespace).Delete(context.Background(), sa.Name, deleteOptions)
			}),
		obsoleteObjectsCleanup(
			r.stores.SCCCache, r.targetStrategy.SCCs(), false, func(scc *secv1.SecurityContextConstraints) bool {
				return !util.IsManagedByOperator(scc.GetLabels())
			},
			r.expectations.SCC, r.kvKey, func(scc *secv1.SecurityContextConstraints) error {
				return r.virtClient.SecClient().SecurityContextConstraints().Delete(context.Background(), scc.Name, deleteOptions)
			},
		),
		obsoleteObjectsCleanup(r.stores.PrometheusRuleCache, r.targetStrategy.PrometheusRules(), true, nil,
			r.expectations.PrometheusRule, r.kvKey, func(cachePromRule *promv1.PrometheusRule) error {
				return r.virtClient.PrometheusClient().
					MonitoringV1().
					PrometheusRules(cachePromRule.Namespace).
					Delete(context.Background(), cachePromRule.Name, deleteOptions)
			}),
		obsoleteObjectsCleanup(r.stores.ServiceMonitorCache, r.targetStrategy.ServiceMonitors(), true, nil,
			r.expectations.ServiceMonitor, r.kvKey, func(cacheServiceMonitor *promv1.ServiceMonitor) error {
				return r.virtClient.PrometheusClient().
					MonitoringV1().
					ServiceMonitors(cacheServiceMonitor.Namespace).
					Delete(context.Background(), cacheServiceMonitor.Name, deleteOptions)
			}),
		r.deleteObsoleteClusterTypes,
	}
	for _, cleanup := range cleanups {
		if err := cleanup(); err != nil {
			return err
		}
	}
	return nil
}

func (r *Reconciler) deleteObsoleteClusterTypes() error {
	managedByVirtOperatorLabelSet := labels.Set{
		v1.AppComponentLabel: GetAppComponent(r.kv),
		v1.ManagedByLabel:    v1.ManagedByLabelOperatorValue,
	}

	// remove unused instancetype objects
	instancetypes, err := r.virtClient.VirtualMachineClusterInstancetype().
		List(context.Background(), metav1.ListOptions{LabelSelector: managedByVirtOperatorLabelSet.String()})
	if err != nil {
		log.Log.Errorf("Failed to get instancetypes: %v", err)
	}
	for _, instancetype := range instancetypes.Items {
		if instancetype.DeletionTimestamp != nil || strategyContainsName(instancetype.Name, r.targetStrategy.Instancetypes()) {
			continue
		}
		err = r.virtClient.VirtualMachineClusterInstancetype().Delete(context.Background(), instancetype.Name, metav1.DeleteOptions{})
		if err != nil {
			log.Log.Errorf("Failed to delete instancetype %+v: %v", instancetype, err)
			return err
		}
	}

	// remove unused preference objects
	preferences, err := r.virtClient.VirtualMachineClusterPreference().
		List(context.Background(), metav1.ListOptions{LabelSelector: managedByVirtOperatorLabelSet.String()})
	if err != nil {
		log.Log.Errorf("Failed to get preferences: %v", err)
	}
	for _, preference := range preferences.Items {
		if preference.DeletionTimestamp != nil || strategyContainsName(preference.Name, r.targetStrategy.Preferences()) {
			continue
		}
		err = r.virtClient.VirtualMachineClusterPreference().Delete(context.Background(), preference.Name, metav1.DeleteOptions{})
		if err != nil {
			log.Log.Errorf("Failed to delete preference %+v: %v", preference, err)
			return err
		}
	}

	return nil
}

func strategyContainsName[T metav1.Object](name string, targets []T) bool {
	for _, target := range targets {
		if target.GetName() == name {
			return true
		}
	}
	return false
}

// obsoleteObjectsCleanup preserves the matching rules of each resource kind. Some
// namespaced resources historically match by name alone; callers select whether
// the namespace is part of the identity.
func obsoleteObjectsCleanup[T metav1.Object](store cache.Store, targets []T, namespaced bool, skip func(T) bool,
	expectations *controller.UIDTrackingControllerExpectations, kvKey string, deleteObject func(T) error,
) func() error {
	return func() error {
		identity := func(obj T) string {
			if namespaced {
				return obj.GetNamespace() + "/" + obj.GetName()
			}
			return obj.GetName()
		}
		wanted := make(map[string]struct{}, len(targets))
		for _, target := range targets {
			wanted[identity(target)] = struct{}{}
		}
		for _, object := range store.List() {
			obj, ok := object.(T)
			if !ok || obj.GetDeletionTimestamp() != nil {
				continue
			}
			if skip != nil && skip(obj) {
				continue
			}
			if _, exists := wanted[identity(obj)]; exists {
				continue
			}
			key, err := controller.KeyFunc(obj)
			if err != nil {
				continue
			}
			expectations.AddExpectedDeletion(kvKey, key)
			if err := deleteObject(obj); err != nil {
				expectations.DeletionObserved(kvKey, key)
				log.Log.Errorf("Failed to delete %T %s: %v", obj, key, err)
				return err
			}
		}
		return nil
	}
}

func (r *Reconciler) isFeatureGateEnabled(fg string) bool {
	return featuregate.IsEnabled(fg, r.kv.Spec.Configuration.DeveloperConfiguration)
}

func (r *Reconciler) commonInstancetypesDeploymentEnabled() bool {
	config := r.kv.Spec.Configuration.CommonInstancetypesDeployment
	if config != nil && config.Enabled != nil {
		return *config.Enabled
	}
	return true
}

func (r *Reconciler) virtTemplateDeploymentEnabled() bool {
	if !r.isFeatureGateEnabled(featuregate.Template) {
		return false
	}
	virtTemplateDeployment := r.kv.Spec.Configuration.VirtTemplateDeployment
	return virtTemplateDeployment == nil || virtTemplateDeployment.Enabled == nil || *virtTemplateDeployment.Enabled
}

func getInstallStrategyAnnotations(meta *metav1.ObjectMeta) (imageTag, imageRegistry, id string, ok bool) {
	var exists bool

	imageTag, exists = meta.Annotations[v1.InstallStrategyVersionAnnotation]
	if !exists {
		ok = false
	}
	imageRegistry, exists = meta.Annotations[v1.InstallStrategyRegistryAnnotation]
	if !exists {
		ok = false
	}
	id, exists = meta.Annotations[v1.InstallStrategyIdentifierAnnotation]
	if !exists {
		ok = false
	}

	return
}
