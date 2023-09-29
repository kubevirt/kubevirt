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
 * Copyright 2019 Red Hat, Inc.
 *
 */

package apply

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/blang/semver"
	promv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	secv1 "github.com/openshift/api/security/v1"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	apiregv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/certificates/triple"
	"kubevirt.io/kubevirt/pkg/certificates/triple/cert"
	"kubevirt.io/kubevirt/pkg/controller"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/install"
	"kubevirt.io/kubevirt/pkg/virt-operator/util"
)

const Duration7d = time.Hour * 24 * 7
const Duration1d = time.Hour * 24

const (
	replaceSpecPatchTemplate     = `{ "op": "replace", "path": "/spec", "value": %s }`
	replaceWebhooksValueTemplate = `{ "op": "replace", "path": "/webhooks", "value": %s }`

	testGenerationJSONPatchTemplate = `{ "op": "test", "path": "/metadata/generation", "value": %d }`
)

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
	objectMeta.Labels[v1.AppComponentLabel] = v1.AppComponent

	if kv.Spec.ProductComponent != "" && util.IsValidLabel(kv.Spec.ProductComponent) {
		objectMeta.Labels[v1.AppComponentLabel] = kv.Spec.ProductComponent
	}

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

const (
	kubernetesOSLabel = corev1.LabelOSStable
	kubernetesOSLinux = "linux"
)

// Merge all Tolerations, Affinity and NodeSelectos from NodePlacement into pod spec
func InjectPlacementMetadata(componentConfig *v1.ComponentConfig, podSpec *corev1.PodSpec) {
	if podSpec == nil {
		podSpec = &corev1.PodSpec{}
	}
	if componentConfig == nil || componentConfig.NodePlacement == nil {
		componentConfig = &v1.ComponentConfig{
			NodePlacement: &v1.NodePlacement{},
		}
	}
	nodePlacement := componentConfig.NodePlacement
	if len(nodePlacement.NodeSelector) == 0 {
		nodePlacement.NodeSelector = make(map[string]string)
	}
	if _, ok := nodePlacement.NodeSelector[kubernetesOSLabel]; !ok {
		nodePlacement.NodeSelector[kubernetesOSLabel] = kubernetesOSLinux
	}
	if len(podSpec.NodeSelector) == 0 {
		podSpec.NodeSelector = make(map[string]string, len(nodePlacement.NodeSelector))
	}
	// podSpec.NodeSelector
	for nsKey, nsVal := range nodePlacement.NodeSelector {
		// Favor podSpec over NodePlacement. This prevents cluster admin from clobbering
		// node selectors that KubeVirt intentionally set.
		if _, ok := podSpec.NodeSelector[nsKey]; !ok {
			podSpec.NodeSelector[nsKey] = nsVal
		}
	}

	// podSpec.Affinity
	if nodePlacement.Affinity != nil {
		if podSpec.Affinity == nil {
			podSpec.Affinity = nodePlacement.Affinity.DeepCopy()
		} else {
			// podSpec.Affinity.NodeAffinity
			if nodePlacement.Affinity.NodeAffinity != nil {
				if podSpec.Affinity.NodeAffinity == nil {
					podSpec.Affinity.NodeAffinity = nodePlacement.Affinity.NodeAffinity.DeepCopy()
				} else {
					// need to copy all affinity terms one by one
					if nodePlacement.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution != nil {
						if podSpec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution == nil {
							podSpec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution = nodePlacement.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.DeepCopy()
						} else {
							// merge the list of terms from NodePlacement into podSpec
							for _, term := range nodePlacement.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms {
								podSpec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms = append(podSpec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms, term)
							}
						}
					}

					//PreferredDuringSchedulingIgnoredDuringExecution
					for _, term := range nodePlacement.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution {
						podSpec.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution = append(podSpec.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution, term)
					}

				}
			}
			// podSpec.Affinity.PodAffinity
			if nodePlacement.Affinity.PodAffinity != nil {
				if podSpec.Affinity.PodAffinity == nil {
					podSpec.Affinity.PodAffinity = nodePlacement.Affinity.PodAffinity.DeepCopy()
				} else {
					//RequiredDuringSchedulingIgnoredDuringExecution
					for _, term := range nodePlacement.Affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution {
						podSpec.Affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution = append(podSpec.Affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution, term)
					}
					//PreferredDuringSchedulingIgnoredDuringExecution
					for _, term := range nodePlacement.Affinity.PodAffinity.PreferredDuringSchedulingIgnoredDuringExecution {
						podSpec.Affinity.PodAffinity.PreferredDuringSchedulingIgnoredDuringExecution = append(podSpec.Affinity.PodAffinity.PreferredDuringSchedulingIgnoredDuringExecution, term)
					}
				}
			}
			// podSpec.Affinity.PodAntiAffinity
			if nodePlacement.Affinity.PodAntiAffinity != nil {
				if podSpec.Affinity.PodAntiAffinity == nil {
					podSpec.Affinity.PodAntiAffinity = nodePlacement.Affinity.PodAntiAffinity.DeepCopy()
				} else {
					//RequiredDuringSchedulingIgnoredDuringExecution
					for _, term := range nodePlacement.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution {
						podSpec.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution = append(podSpec.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution, term)
					}
					//PreferredDuringSchedulingIgnoredDuringExecution
					for _, term := range nodePlacement.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution {
						podSpec.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution = append(podSpec.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution, term)
					}
				}
			}
		}
	}

	//podSpec.Tolerations
	if len(nodePlacement.Tolerations) != 0 {
		if len(podSpec.Tolerations) == 0 {
			podSpec.Tolerations = []corev1.Toleration{}
		}
		for _, toleration := range nodePlacement.Tolerations {
			podSpec.Tolerations = append(podSpec.Tolerations, toleration)
		}
	}
}

func generatePatchBytes(ops []string) []byte {
	return controller.GeneratePatchBytes(ops)
}

func createLabelsAndAnnotationsPatch(objectMeta *metav1.ObjectMeta) ([]string, error) {
	var ops []string
	labelBytes, err := json.Marshal(objectMeta.Labels)
	if err != nil {
		return ops, err
	}
	annotationBytes, err := json.Marshal(objectMeta.Annotations)
	if err != nil {
		return ops, err
	}
	ownerRefBytes, err := json.Marshal(objectMeta.OwnerReferences)
	if err != nil {
		return ops, err
	}
	ops = append(ops, fmt.Sprintf(`{ "op": "add", "path": "/metadata/labels", "value": %s }`, string(labelBytes)))
	ops = append(ops, fmt.Sprintf(`{ "op": "add", "path": "/metadata/annotations", "value": %s }`, string(annotationBytes)))
	ops = append(ops, fmt.Sprintf(`{ "op": "add", "path": "/metadata/ownerReferences", "value": %s }`, ownerRefBytes))

	return ops, nil
}

func getPatchWithObjectMetaAndSpec(ops []string, meta *metav1.ObjectMeta, spec []byte) ([]string, error) {
	// Add Labels and Annotations Patches
	labelAnnotationPatch, err := createLabelsAndAnnotationsPatch(meta)
	if err != nil {
		return ops, err
	}

	ops = append(ops, labelAnnotationPatch...)

	// and spec replacement to patch
	ops = append(ops, fmt.Sprintf(replaceSpecPatchTemplate, string(spec)))

	return ops, nil
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
	target, err := semver.Make(targetVersion)
	if err == nil {
		current, err := semver.Make(currentVersion)
		if err == nil {
			if target.Compare(current) <= 0 {
				shouldTakeUpdatePath = false
			}
		}
	}

	return shouldTakeUpdatePath
}

func haveApiDeploymentsRolledOver(targetStrategy *install.Strategy, kv *v1.KubeVirt, stores util.Stores) bool {
	for _, deployment := range targetStrategy.ApiDeployments() {
		if !util.DeploymentIsReady(kv, deployment, stores) {
			log.Log.V(2).Infof("Waiting on deployment %v to roll over to latest version", deployment.GetName())
			// not rolled out yet
			return false
		}
	}

	return true
}

func haveControllerDeploymentsRolledOver(targetStrategy *install.Strategy, kv *v1.KubeVirt, stores util.Stores) bool {
	for _, deployment := range targetStrategy.ControllerDeployments() {
		if !util.DeploymentIsReady(kv, deployment, stores) {
			log.Log.V(2).Infof("Waiting on deployment %v to roll over to latest version", deployment.GetName())
			// not rolled out yet
			return false
		}
	}

	return true
}

func haveExportProxyDeploymentsRolledOver(targetStrategy *install.Strategy, kv *v1.KubeVirt, stores util.Stores) bool {
	for _, deployment := range targetStrategy.ExportProxyDeployments() {
		if !util.DeploymentIsReady(kv, deployment, stores) {
			log.Log.V(2).Infof("Waiting on deployment %v to roll over to latest version", deployment.GetName())
			// not rolled out yet
			return false
		}
	}

	return true
}

func haveDaemonSetsRolledOver(targetStrategy *install.Strategy, kv *v1.KubeVirt, stores util.Stores) bool {
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
		_, exists, _ := r.stores.CrdCache.Get(crd)
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
	_, err := r.clientset.AdmissionregistrationV1().ValidatingWebhookConfigurations().Create(context.Background(), validationWebhook, metav1.CreateOptions{})
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

	targetStrategy   *install.Strategy
	stores           util.Stores
	clientset        kubecli.KubevirtClient
	aggregatorclient install.APIServiceInterface
	expectations     *util.Expectations
	recorder         record.EventRecorder
}

func NewReconciler(kv *v1.KubeVirt, targetStrategy *install.Strategy, stores util.Stores, clientset kubecli.KubevirtClient, aggregatorclient install.APIServiceInterface, expectations *util.Expectations, recorder record.EventRecorder) (*Reconciler, error) {
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

	return &Reconciler{
		kv:               kv,
		kvKey:            kvKey,
		targetStrategy:   targetStrategy,
		stores:           stores,
		clientset:        clientset,
		aggregatorclient: aggregatorclient,
		expectations:     expectations,
		recorder:         recorder,
	}, nil
}

func (r *Reconciler) Sync(queue workqueue.RateLimitingInterface) (bool, error) {
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

	exportProxyEnabled := r.exportProxyEnabled()
	exportProxyDeploymentsRolledOver := !exportProxyEnabled || haveExportProxyDeploymentsRolledOver(r.targetStrategy, r.kv, r.stores)

	daemonSetsRolledOver := haveDaemonSetsRolledOver(r.targetStrategy, r.kv, r.stores)

	infrastructureRolledOver := false
	if apiDeploymentsRolledOver && controllerDeploymentsRolledOver && exportProxyDeploymentsRolledOver && daemonSetsRolledOver {

		// infrastructure has rolled over and is available
		infrastructureRolledOver = true
	} else if (targetVersion == observedVersion) && (targetImageRegistry == observedImageRegistry) {
		// infrastructure was observed to have rolled over successfully
		// in the past
		infrastructureRolledOver = true
	}

	// -------- CREATE AND ROLE OUT UPDATED OBJECTS --------

	// creates a blocking webhook for any new CRDs that don't exist previously.
	// this webhook is removed once the new apiserver is online.
	if !apiDeploymentsRolledOver {
		err := r.createDummyWebhookValidator()
		if err != nil {
			return false, err
		}
	} else {
		err := deleteDummyWebhookValidators(r.kv, r.clientset, r.stores, r.expectations)
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
	for _, deployment := range r.targetStrategy.ExportProxyDeployments() {
		if r.exportProxyEnabled() {
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
	if err := r.clientset.AppsV1().Deployments(deployment.Namespace).Delete(context.Background(), deployment.Name, metav1.DeleteOptions{}); err != nil {
		r.expectations.Deployment.DeletionObserved(r.kvKey, key)
		return err
	}

	return nil
}

func (r *Reconciler) deleteObjectsNotInInstallStrategy() error {
	gracePeriod := int64(0)
	deleteOptions := metav1.DeleteOptions{
		GracePeriodSeconds: &gracePeriod,
	}

	client := r.clientset.ExtensionsClient()
	// -------- CLEAN UP OLD UNUSED OBJECTS --------
	// outdated webhooks can potentially block deletes of other objects during the cleanup and need to be removed first

	// remove unused validating webhooks
	objects := r.stores.ValidationWebhookCache.List()
	for _, obj := range objects {
		if webhook, ok := obj.(*admissionregistrationv1.ValidatingWebhookConfiguration); ok && webhook.DeletionTimestamp == nil {
			found := false
			if strings.HasPrefix(webhook.Name, "virt-operator-tmp-webhook") {
				continue
			}

			for _, targetWebhook := range r.targetStrategy.ValidatingWebhookConfigurations() {

				if targetWebhook.Name == webhook.Name {
					found = true
					break
				}
			}
			if !found {
				if key, err := controller.KeyFunc(webhook); err == nil {
					r.expectations.ValidationWebhook.AddExpectedDeletion(r.kvKey, key)
					err := r.clientset.AdmissionregistrationV1().ValidatingWebhookConfigurations().Delete(context.Background(), webhook.Name, deleteOptions)
					if err != nil {
						r.expectations.ValidationWebhook.DeletionObserved(r.kvKey, key)
						log.Log.Errorf("Failed to delete webhook %+v: %v", webhook, err)
						return err
					}
				}
			}
		}
	}

	// remove unused mutating webhooks
	objects = r.stores.MutatingWebhookCache.List()
	for _, obj := range objects {
		if webhook, ok := obj.(*admissionregistrationv1.MutatingWebhookConfiguration); ok && webhook.DeletionTimestamp == nil {
			found := false
			for _, targetWebhook := range r.targetStrategy.MutatingWebhookConfigurations() {
				if targetWebhook.Name == webhook.Name {
					found = true
					break
				}
			}
			if !found {
				if key, err := controller.KeyFunc(webhook); err == nil {
					r.expectations.MutatingWebhook.AddExpectedDeletion(r.kvKey, key)
					err := r.clientset.AdmissionregistrationV1().MutatingWebhookConfigurations().Delete(context.Background(), webhook.Name, deleteOptions)
					if err != nil {
						r.expectations.MutatingWebhook.DeletionObserved(r.kvKey, key)
						log.Log.Errorf("Failed to delete webhook %+v: %v", webhook, err)
						return err
					}
				}
			}
		}
	}

	// remove unused APIServices
	objects = r.stores.APIServiceCache.List()
	for _, obj := range objects {
		if apiService, ok := obj.(*apiregv1.APIService); ok && apiService.DeletionTimestamp == nil {
			found := false
			for _, targetAPIService := range r.targetStrategy.APIServices() {
				if targetAPIService.Name == apiService.Name {
					found = true
					break
				}
			}
			if !found {
				if key, err := controller.KeyFunc(apiService); err == nil {
					r.expectations.APIService.AddExpectedDeletion(r.kvKey, key)
					err := r.aggregatorclient.Delete(context.Background(), apiService.Name, deleteOptions)
					if err != nil {
						r.expectations.APIService.DeletionObserved(r.kvKey, key)
						log.Log.Errorf("Failed to delete apiService %+v: %v", apiService, err)
						return err
					}
				}
			}
		}
	}

	// remove unused Secrets
	objects = r.stores.SecretCache.List()
	for _, obj := range objects {
		if secret, ok := obj.(*corev1.Secret); ok && secret.DeletionTimestamp == nil {
			found := false
			for _, targetSecret := range r.targetStrategy.CertificateSecrets() {
				if targetSecret.Name == secret.Name {
					found = true
					break
				}
			}
			if !found {
				if key, err := controller.KeyFunc(secret); err == nil {
					r.expectations.Secrets.AddExpectedDeletion(r.kvKey, key)
					err := r.clientset.CoreV1().Secrets(secret.Namespace).Delete(context.Background(), secret.Name, deleteOptions)
					if err != nil {
						r.expectations.Secrets.DeletionObserved(r.kvKey, key)
						log.Log.Errorf("Failed to delete secret %+v: %v", secret, err)
						return err
					}
				}
			}
		}
	}

	// remove unused ConfigMaps
	objects = r.stores.ConfigMapCache.List()
	for _, obj := range objects {
		if configMap, ok := obj.(*corev1.ConfigMap); ok && configMap.DeletionTimestamp == nil {
			found := false
			for _, targetConfigMap := range r.targetStrategy.ConfigMaps() {
				if targetConfigMap.Name == configMap.Name {
					found = true
					break
				}
			}
			if !found {
				if key, err := controller.KeyFunc(configMap); err == nil {
					r.expectations.ConfigMap.AddExpectedDeletion(r.kvKey, key)
					err := r.clientset.CoreV1().ConfigMaps(configMap.Namespace).Delete(context.Background(), configMap.Name, deleteOptions)
					if err != nil {
						r.expectations.ConfigMap.DeletionObserved(r.kvKey, key)
						log.Log.Errorf("Failed to delete configmap %+v: %v", configMap, err)
						return err
					}
				}
			}
		}
	}

	// remove unused crds
	objects = r.stores.CrdCache.List()
	for _, obj := range objects {
		if crd, ok := obj.(*extv1.CustomResourceDefinition); ok && crd.DeletionTimestamp == nil {
			found := false
			for _, targetCrd := range r.targetStrategy.CRDs() {
				if targetCrd.Name == crd.Name {
					found = true
					break
				}
			}
			if !found {
				if key, err := controller.KeyFunc(crd); err == nil {
					r.expectations.Crd.AddExpectedDeletion(r.kvKey, key)
					err := client.ApiextensionsV1().CustomResourceDefinitions().Delete(context.Background(), crd.Name, deleteOptions)
					if err != nil {
						r.expectations.Crd.DeletionObserved(r.kvKey, key)
						log.Log.Errorf("Failed to delete crd %+v: %v", crd, err)
						return err
					}
				}
			}
		}
	}

	// remove unused daemonsets
	objects = r.stores.DaemonSetCache.List()
	for _, obj := range objects {
		if ds, ok := obj.(*appsv1.DaemonSet); ok && ds.DeletionTimestamp == nil {
			found := false
			for _, targetDs := range r.targetStrategy.DaemonSets() {
				if targetDs.Name == ds.Name && targetDs.Namespace == ds.Namespace {
					found = true
					break
				}
			}
			if !found {
				if key, err := controller.KeyFunc(ds); err == nil {
					r.expectations.DaemonSet.AddExpectedDeletion(r.kvKey, key)
					err := r.clientset.AppsV1().DaemonSets(ds.Namespace).Delete(context.Background(), ds.Name, deleteOptions)
					if err != nil {
						r.expectations.DaemonSet.DeletionObserved(r.kvKey, key)
						log.Log.Errorf("Failed to delete daemonset: %v", err)
						return err
					}
				}
			}
		}
	}

	// remove unused deployments
	objects = r.stores.DeploymentCache.List()
	for _, obj := range objects {
		if deployment, ok := obj.(*appsv1.Deployment); ok && deployment.DeletionTimestamp == nil {
			found := false
			for _, targetDeployment := range r.targetStrategy.Deployments() {
				if targetDeployment.Name == deployment.Name && targetDeployment.Namespace == deployment.Namespace {
					found = true
					break
				}
			}
			if !found {
				if key, err := controller.KeyFunc(deployment); err == nil {
					r.expectations.Deployment.AddExpectedDeletion(r.kvKey, key)
					err := r.clientset.AppsV1().Deployments(deployment.Namespace).Delete(context.Background(), deployment.Name, deleteOptions)
					if err != nil {
						r.expectations.Deployment.DeletionObserved(r.kvKey, key)
						log.Log.Errorf("Failed to delete deployment: %v", err)
						return err
					}
				}
			}
		}
	}

	// remove unused services
	objects = r.stores.ServiceCache.List()
	for _, obj := range objects {
		if svc, ok := obj.(*corev1.Service); ok && svc.DeletionTimestamp == nil {
			found := false
			for _, targetSvc := range r.targetStrategy.Services() {
				if targetSvc.Name == svc.Name && targetSvc.Namespace == svc.Namespace {
					found = true
					break
				}
			}
			if !found {
				if key, err := controller.KeyFunc(svc); err == nil {
					r.expectations.Service.AddExpectedDeletion(r.kvKey, key)
					err := r.clientset.CoreV1().Services(svc.Namespace).Delete(context.Background(), svc.Name, deleteOptions)
					if err != nil {
						r.expectations.Service.DeletionObserved(r.kvKey, key)
						log.Log.Errorf("Failed to delete service %+v: %v", svc, err)
						return err
					}
				}
			}
		}
	}

	// remove unused clusterrolebindings
	objects = r.stores.ClusterRoleBindingCache.List()
	for _, obj := range objects {
		if crb, ok := obj.(*rbacv1.ClusterRoleBinding); ok && crb.DeletionTimestamp == nil {
			found := false
			for _, targetCrb := range r.targetStrategy.ClusterRoleBindings() {
				if targetCrb.Name == crb.Name && targetCrb.Namespace == crb.Namespace {
					found = true
					break
				}
			}
			if !found {
				if key, err := controller.KeyFunc(crb); err == nil {
					r.expectations.ClusterRoleBinding.AddExpectedDeletion(r.kvKey, key)
					err := r.clientset.RbacV1().ClusterRoleBindings().Delete(context.Background(), crb.Name, deleteOptions)
					if err != nil {
						r.expectations.ClusterRoleBinding.DeletionObserved(r.kvKey, key)
						log.Log.Errorf("Failed to delete crb %+v: %v", crb, err)
						return err
					}
				}
			}
		}
	}

	// remove unused clusterroles
	objects = r.stores.ClusterRoleCache.List()
	for _, obj := range objects {
		if cr, ok := obj.(*rbacv1.ClusterRole); ok && cr.DeletionTimestamp == nil {
			found := false
			for _, targetCr := range r.targetStrategy.ClusterRoles() {
				if targetCr.Name == cr.Name && targetCr.Namespace == cr.Namespace {
					found = true
					break
				}
			}
			if !found {
				if key, err := controller.KeyFunc(cr); err == nil {
					r.expectations.ClusterRole.AddExpectedDeletion(r.kvKey, key)
					err := r.clientset.RbacV1().ClusterRoles().Delete(context.Background(), cr.Name, deleteOptions)
					if err != nil {
						r.expectations.ClusterRole.DeletionObserved(r.kvKey, key)
						log.Log.Errorf("Failed to delete cr %+v: %v", cr, err)
						return err
					}
				}
			}
		}
	}

	// remove unused rolebindings
	objects = r.stores.RoleBindingCache.List()
	for _, obj := range objects {
		if rb, ok := obj.(*rbacv1.RoleBinding); ok && rb.DeletionTimestamp == nil {
			found := false
			for _, targetRb := range r.targetStrategy.RoleBindings() {
				if targetRb.Name == rb.Name && targetRb.Namespace == rb.Namespace {
					found = true
					break
				}
			}
			if !found {
				if key, err := controller.KeyFunc(rb); err == nil {
					r.expectations.RoleBinding.AddExpectedDeletion(r.kvKey, key)
					err := r.clientset.RbacV1().RoleBindings(rb.Namespace).Delete(context.Background(), rb.Name, deleteOptions)
					if err != nil {
						r.expectations.RoleBinding.DeletionObserved(r.kvKey, key)
						log.Log.Errorf("Failed to delete rb %+v: %v", rb, err)
						return err
					}
				}
			}
		}
	}

	// remove unused roles
	objects = r.stores.RoleCache.List()
	for _, obj := range objects {
		if role, ok := obj.(*rbacv1.Role); ok && role.DeletionTimestamp == nil {
			found := false
			for _, targetR := range r.targetStrategy.Roles() {
				if targetR.Name == role.Name && targetR.Namespace == role.Namespace {
					found = true
					break
				}
			}
			if !found {
				if key, err := controller.KeyFunc(role); err == nil {
					r.expectations.Role.AddExpectedDeletion(r.kvKey, key)
					err := r.clientset.RbacV1().Roles(role.Namespace).Delete(context.Background(), role.Name, deleteOptions)
					if err != nil {
						r.expectations.Role.DeletionObserved(r.kvKey, key)
						log.Log.Errorf("Failed to delete role %+v: %v", role, err)
						return err
					}
				}
			}
		}
	}

	// remove unused serviceaccounts
	objects = r.stores.ServiceAccountCache.List()
	for _, obj := range objects {
		if sa, ok := obj.(*corev1.ServiceAccount); ok && sa.DeletionTimestamp == nil {
			found := false
			for _, targetSa := range r.targetStrategy.ServiceAccounts() {
				if targetSa.Name == sa.Name && targetSa.Namespace == sa.Namespace {
					found = true
					break
				}
			}
			if !found {
				if key, err := controller.KeyFunc(sa); err == nil {
					r.expectations.ServiceAccount.AddExpectedDeletion(r.kvKey, key)
					err := r.clientset.CoreV1().ServiceAccounts(sa.Namespace).Delete(context.Background(), sa.Name, deleteOptions)
					if err != nil {
						r.expectations.ServiceAccount.DeletionObserved(r.kvKey, key)
						log.Log.Errorf("Failed to delete serviceaccount %+v: %v", sa, err)
						return err
					}
				}
			}
		}
	}

	// remove unused sccs
	objects = r.stores.SCCCache.List()
	for _, obj := range objects {
		if scc, ok := obj.(*secv1.SecurityContextConstraints); ok && scc.DeletionTimestamp == nil {

			// informer watches all SCC objects, it cannot be changed because of kubevirt updates
			if !util.IsManagedByOperator(scc.GetLabels()) {
				continue
			}

			found := false
			for _, targetScc := range r.targetStrategy.SCCs() {
				if targetScc.Name == scc.Name {
					found = true
					break
				}
			}
			if !found {
				if key, err := controller.KeyFunc(scc); err == nil {
					r.expectations.SCC.AddExpectedDeletion(r.kvKey, key)
					err := r.clientset.SecClient().SecurityContextConstraints().Delete(context.Background(), scc.Name, deleteOptions)
					if err != nil {
						r.expectations.SCC.DeletionObserved(r.kvKey, key)
						log.Log.Errorf("Failed to delete SecurityContextConstraints %+v: %v", scc, err)
						return err
					}
				}
			}
		}
	}

	// remove unused prometheus rules
	objects = r.stores.PrometheusRuleCache.List()
	for _, obj := range objects {
		if cachePromRule, ok := obj.(*promv1.PrometheusRule); ok && cachePromRule.DeletionTimestamp == nil {
			found := false
			for _, targetPromRule := range r.targetStrategy.PrometheusRules() {
				if targetPromRule.Name == cachePromRule.Name && targetPromRule.Namespace == cachePromRule.Namespace {
					found = true
					break
				}
			}
			if !found {
				if key, err := controller.KeyFunc(cachePromRule); err == nil {
					r.expectations.PrometheusRule.AddExpectedDeletion(r.kvKey, key)
					err := r.clientset.PrometheusClient().
						MonitoringV1().
						PrometheusRules(cachePromRule.Namespace).
						Delete(context.Background(), cachePromRule.Name, deleteOptions)
					if err != nil {
						r.expectations.PrometheusRule.DeletionObserved(r.kvKey, key)
						log.Log.Errorf("Failed to delete prometheusrule %+v: %v", cachePromRule, err)
						return err
					}
				}
			}
		}
	}

	// remove unused prometheus serviceMonitor obejcts
	objects = r.stores.ServiceMonitorCache.List()
	for _, obj := range objects {
		if cacheServiceMonitor, ok := obj.(*promv1.ServiceMonitor); ok && cacheServiceMonitor.DeletionTimestamp == nil {
			found := false
			for _, targetServiceMonitor := range r.targetStrategy.ServiceMonitors() {
				if targetServiceMonitor.Name == cacheServiceMonitor.Name && targetServiceMonitor.Namespace == cacheServiceMonitor.Namespace {
					found = true
					break
				}
			}
			if !found {
				if key, err := controller.KeyFunc(cacheServiceMonitor); err == nil {
					r.expectations.ServiceMonitor.AddExpectedDeletion(r.kvKey, key)
					err := r.clientset.PrometheusClient().
						MonitoringV1().
						ServiceMonitors(cacheServiceMonitor.Namespace).
						Delete(context.Background(), cacheServiceMonitor.Name, deleteOptions)
					if err != nil {
						r.expectations.ServiceMonitor.DeletionObserved(r.kvKey, key)
						log.Log.Errorf("Failed to delete prometheusServiceMonitor %+v: %v", cacheServiceMonitor, err)
						return err
					}
				}
			}
		}
	}

	managedByVirtOperatorLabelSet := labels.Set{
		v1.AppComponentLabel: v1.AppComponent,
		v1.ManagedByLabel:    v1.ManagedByLabelOperatorValue,
	}

	// remove unused instancetype objects
	instancetypes, err := r.clientset.VirtualMachineClusterInstancetype().List(context.Background(), metav1.ListOptions{LabelSelector: managedByVirtOperatorLabelSet.String()})
	if err != nil {
		log.Log.Errorf("Failed to get instancetypes: %v", err)
	}
	for _, instancetype := range instancetypes.Items {
		if instancetype.DeletionTimestamp == nil {
			found := false
			for _, targetInstancetype := range r.targetStrategy.Instancetypes() {
				if targetInstancetype.Name == instancetype.Name {
					found = true
					break
				}
			}
			if !found {
				if err := r.clientset.VirtualMachineClusterInstancetype().Delete(context.Background(), instancetype.Name, metav1.DeleteOptions{}); err != nil {
					log.Log.Errorf("Failed to delete instancetype %+v: %v", instancetype, err)
					return err
				}
			}
		}
	}

	// remove unused preference objects
	preferences, err := r.clientset.VirtualMachineClusterPreference().List(context.Background(), metav1.ListOptions{LabelSelector: managedByVirtOperatorLabelSet.String()})
	if err != nil {
		log.Log.Errorf("Failed to get preferences: %v", err)
	}
	for _, preference := range preferences.Items {
		if preference.DeletionTimestamp == nil {
			found := false
			for _, targetPreference := range r.targetStrategy.Preferences() {
				if targetPreference.Name == preference.Name {
					found = true
					break
				}
			}
			if !found {
				if err := r.clientset.VirtualMachineClusterPreference().Delete(context.Background(), preference.Name, metav1.DeleteOptions{}); err != nil {
					log.Log.Errorf("Failed to delete preference %+v: %v", preference, err)
					return err
				}
			}
		}
	}

	return nil
}

func (r *Reconciler) isFeatureGateEnabled(featureGate string) bool {
	if r.kv.Spec.Configuration.DeveloperConfiguration == nil {
		return false
	}

	for _, fg := range r.kv.Spec.Configuration.DeveloperConfiguration.FeatureGates {
		if fg == featureGate {
			return true
		}
	}

	return false
}

func (r *Reconciler) exportProxyEnabled() bool {
	return r.isFeatureGateEnabled(virtconfig.VMExportGate)
}

func (r *Reconciler) commonInstancetypesDeploymentEnabled() bool {
	return r.isFeatureGateEnabled(virtconfig.CommonInstancetypesDeploymentGate)
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
