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

package installstrategy

//go:generate mockgen -source $GOFILE -imports "libvirt=libvirt.org/libvirt-go" -package=$GOPACKAGE -destination=generated_mock_$GOFILE

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	promv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	secv1 "github.com/openshift/api/security/v1"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	extv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/kube-aggregator/pkg/apis/apiregistration/v1beta1"

	"github.com/blang/semver"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/certificates/triple"
	"kubevirt.io/kubevirt/pkg/certificates/triple/cert"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/virt-operator/creation/components"
	"kubevirt.io/kubevirt/pkg/virt-operator/creation/rbac"
	"kubevirt.io/kubevirt/pkg/virt-operator/util"
)

const Duration7d = time.Hour * 24 * 7
const Duration1d = time.Hour * 24

type APIServiceInterface interface {
	Get(name string, options metav1.GetOptions) (*v1beta1.APIService, error)
	Create(*v1beta1.APIService) (*v1beta1.APIService, error)
	Delete(name string, options *metav1.DeleteOptions) error
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1beta1.APIService, err error)
}

func objectMatchesVersion(objectMeta *metav1.ObjectMeta, version, imageRegistry, id string, generation int64) bool {

	if objectMeta.Annotations == nil {
		return false
	}

	foundVersion := objectMeta.Annotations[v1.InstallStrategyVersionAnnotation]
	foundImageRegistry := objectMeta.Annotations[v1.InstallStrategyRegistryAnnotation]
	foundID := objectMeta.Annotations[v1.InstallStrategyIdentifierAnnotation]
	foundGeneration := objectMeta.Annotations[v1.KubeVirtGenerationAnnotation]
	foundLabels := objectMeta.Labels[v1.ManagedByLabel] == v1.ManagedByLabelOperatorValue
	sGeneration := strconv.FormatInt(generation, 10)

	if foundVersion == version && foundImageRegistry == imageRegistry && foundID == id && foundGeneration == sGeneration && foundLabels {
		return true
	}

	return false
}

func apiDeployments(strategy *InstallStrategy) []*appsv1.Deployment {
	var deployments []*appsv1.Deployment

	for _, deployment := range strategy.deployments {
		if !strings.Contains(deployment.Name, "virt-api") {
			continue
		}
		deployments = append(deployments, deployment)

	}
	return deployments
}

func controllerDeployments(strategy *InstallStrategy) []*appsv1.Deployment {
	var deployments []*appsv1.Deployment

	for _, deployment := range strategy.deployments {
		if strings.Contains(deployment.Name, "virt-api") {
			continue
		}
		deployments = append(deployments, deployment)

	}
	return deployments
}

func isValidLabel(label string) bool {
	// First and last character must be alphanumeric
	// middle chars can be alphanumeric, or dot hyphen or dash
	// entire string must not exceed 63 chars
	r := regexp.MustCompile(`^([a-z0-9A-Z]([a-z0-9A-Z\-\_\.]{0,61}[a-z0-9A-Z])?)?$`)
	return r.Match([]byte(label))
}

func injectOperatorMetadata(kv *v1.KubeVirt, objectMeta *metav1.ObjectMeta, version string, imageRegistry string, id string, injectCustomizationMetadata bool) {
	if objectMeta.Labels == nil {
		objectMeta.Labels = make(map[string]string)
	}
	if kv.Spec.ProductVersion != "" && isValidLabel(kv.Spec.ProductVersion) {
		objectMeta.Labels[v1.AppVersionLabel] = kv.Spec.ProductVersion
	}
	if kv.Spec.ProductName != "" && isValidLabel(kv.Spec.ProductName) {
		objectMeta.Labels[v1.AppPartOfLabel] = kv.Spec.ProductName
	}
	objectMeta.Labels[v1.AppComponentLabel] = v1.AppComponent

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

func generatePatchBytes(ops []string) []byte {
	opsStr := "["
	for idx, entry := range ops {
		sep := ", "
		if len(ops)-1 == idx {
			sep = "]"
		}
		opsStr = fmt.Sprintf("%s%s%s", opsStr, entry, sep)
	}
	return []byte(opsStr)
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

func syncDaemonSet(kv *v1.KubeVirt,
	daemonSet *appsv1.DaemonSet,
	stores util.Stores,
	clientset kubecli.KubevirtClient,
	expectations *util.Expectations) error {

	daemonSet = daemonSet.DeepCopy()

	apps := clientset.AppsV1()
	imageTag := kv.Status.TargetKubeVirtVersion
	imageRegistry := kv.Status.TargetKubeVirtRegistry
	id := kv.Status.TargetDeploymentID

	injectOperatorMetadata(kv, &daemonSet.ObjectMeta, imageTag, imageRegistry, id, true)
	injectOperatorMetadata(kv, &daemonSet.Spec.Template.ObjectMeta, imageTag, imageRegistry, id, false)

	kvkey, err := controller.KeyFunc(kv)
	if err != nil {
		return err
	}

	var cachedDaemonSet *appsv1.DaemonSet
	obj, exists, _ := stores.DaemonSetCache.Get(daemonSet)
	if exists {
		cachedDaemonSet = obj.(*appsv1.DaemonSet)
	}
	if !exists {
		expectations.DaemonSet.RaiseExpectations(kvkey, 1, 0)
		_, err = apps.DaemonSets(kv.Namespace).Create(daemonSet)
		if err != nil {
			expectations.DaemonSet.LowerExpectations(kvkey, 1, 0)
			return fmt.Errorf("unable to create daemonset %+v: %v", daemonSet, err)
		}
	} else if !objectMatchesVersion(&cachedDaemonSet.ObjectMeta, imageTag, imageRegistry, id, kv.GetGeneration()) {
		// Patch if old version
		var ops []string

		// Add Labels and Annotations Patches
		labelAnnotationPatch, err := createLabelsAndAnnotationsPatch(&daemonSet.ObjectMeta)
		if err != nil {
			return err
		}
		ops = append(ops, labelAnnotationPatch...)

		// Add Spec Patch
		newSpec, err := json.Marshal(daemonSet.Spec)
		if err != nil {
			return err
		}
		ops = append(ops, fmt.Sprintf(`{ "op": "replace", "path": "/spec", "value": %s }`, string(newSpec)))

		_, err = apps.DaemonSets(kv.Namespace).Patch(daemonSet.Name, types.JSONPatchType, generatePatchBytes(ops))
		if err != nil {
			return fmt.Errorf("unable to patch daemonset %+v: %v", daemonSet, err)
		}
		log.Log.V(2).Infof("daemonset %v updated", daemonSet.GetName())

	} else {
		log.Log.V(4).Infof("daemonset %v is up-to-date", daemonSet.GetName())
	}
	return nil
}

func syncDeployment(kv *v1.KubeVirt,
	deployment *appsv1.Deployment,
	stores util.Stores,
	clientset kubecli.KubevirtClient,
	expectations *util.Expectations) error {

	deployment = deployment.DeepCopy()

	apps := clientset.AppsV1()
	imageTag := kv.Status.TargetKubeVirtVersion
	imageRegistry := kv.Status.TargetKubeVirtRegistry
	id := kv.Status.TargetDeploymentID

	injectOperatorMetadata(kv, &deployment.ObjectMeta, imageTag, imageRegistry, id, true)
	injectOperatorMetadata(kv, &deployment.Spec.Template.ObjectMeta, imageTag, imageRegistry, id, false)

	kvkey, err := controller.KeyFunc(kv)
	if err != nil {
		return err
	}

	var cachedDeployment *appsv1.Deployment

	obj, exists, _ := stores.DeploymentCache.Get(deployment)
	if exists {
		cachedDeployment = obj.(*appsv1.Deployment)
	}

	if !exists {
		expectations.Deployment.RaiseExpectations(kvkey, 1, 0)
		_, err = apps.Deployments(kv.Namespace).Create(deployment)
		if err != nil {
			expectations.Deployment.LowerExpectations(kvkey, 1, 0)
			return fmt.Errorf("unable to create deployment %+v: %v", deployment, err)
		}
	} else if !objectMatchesVersion(&cachedDeployment.ObjectMeta, imageTag, imageRegistry, id, kv.GetGeneration()) {
		// Patch if old version
		var ops []string

		// Add Labels and Annotations Patches
		labelAnnotationPatch, err := createLabelsAndAnnotationsPatch(&deployment.ObjectMeta)
		if err != nil {
			return err
		}
		ops = append(ops, labelAnnotationPatch...)

		// Add Spec Patch
		newSpec, err := json.Marshal(deployment.Spec)
		if err != nil {
			return err
		}
		ops = append(ops, fmt.Sprintf(`{ "op": "replace", "path": "/spec", "value": %s }`, string(newSpec)))

		_, err = apps.Deployments(kv.Namespace).Patch(deployment.Name, types.JSONPatchType, generatePatchBytes(ops))
		if err != nil {
			return fmt.Errorf("unable to patch deployment %+v: %v", deployment, err)
		}
		log.Log.V(2).Infof("deployment %v updated", deployment.GetName())

	} else {
		log.Log.V(4).Infof("deployment %v is up-to-date", deployment.GetName())
	}

	return nil
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

func haveApiDeploymentsRolledOver(targetStrategy *InstallStrategy, kv *v1.KubeVirt, stores util.Stores) bool {
	for _, deployment := range apiDeployments(targetStrategy) {
		if !util.DeploymentIsReady(kv, deployment, stores) {
			log.Log.V(2).Infof("Waiting on deployment %v to roll over to latest version", deployment.GetName())
			// not rolled out yet
			return false
		}
	}

	return true
}

func haveControllerDeploymentsRolledOver(targetStrategy *InstallStrategy, kv *v1.KubeVirt, stores util.Stores) bool {

	for _, deployment := range controllerDeployments(targetStrategy) {
		if !util.DeploymentIsReady(kv, deployment, stores) {
			log.Log.V(2).Infof("Waiting on deployment %v to roll over to latest version", deployment.GetName())
			// not rolled out yet
			return false
		}
	}
	return true
}

func haveDaemonSetsRolledOver(targetStrategy *InstallStrategy, kv *v1.KubeVirt, stores util.Stores) bool {

	for _, daemonSet := range targetStrategy.daemonSets {
		if !util.DaemonsetIsReady(kv, daemonSet, stores) {
			log.Log.V(2).Infof("Waiting on daemonset %v to roll over to latest version", daemonSet.GetName())
			// not rolled out yet
			return false
		}
	}

	return true
}

func createDummyWebhookValidator(targetStrategy *InstallStrategy,
	kv *v1.KubeVirt,
	clientset kubecli.KubevirtClient,
	stores util.Stores,
	expectations *util.Expectations) error {

	var webhooks []admissionregistrationv1beta1.ValidatingWebhook

	kvkey, err := controller.KeyFunc(kv)
	if err != nil {
		return err
	}
	version := kv.Status.TargetKubeVirtVersion
	imageRegistry := kv.Status.TargetKubeVirtRegistry
	id := kv.Status.TargetDeploymentID

	// If webhook already exists in cache, then exit.
	objects := stores.ValidationWebhookCache.List()
	for _, obj := range objects {
		if webhook, ok := obj.(*admissionregistrationv1beta1.ValidatingWebhookConfiguration); ok {

			if objectMatchesVersion(&webhook.ObjectMeta, version, imageRegistry, id, kv.GetGeneration()) {
				// already created blocking webhook for this version
				return nil
			}
		}
	}

	// generate a fake cert. this isn't actually used
	failurePolicy := admissionregistrationv1beta1.Fail

	for _, crd := range targetStrategy.crds {
		_, exists, _ := stores.CrdCache.Get(crd)
		if exists {
			// this CRD isn't new, it already exists in cache so we don't
			// need a blocking admission webhook to wait until the new
			// apiserver is active
			continue
		}
		path := fmt.Sprintf("/fake-path/%s", crd.Name)
		webhooks = append(webhooks, admissionregistrationv1beta1.ValidatingWebhook{
			Name:          fmt.Sprintf("%s-tmp-validator", crd.Name),
			FailurePolicy: &failurePolicy,
			Rules: []admissionregistrationv1beta1.RuleWithOperations{{
				Operations: []admissionregistrationv1beta1.OperationType{
					admissionregistrationv1beta1.Create,
				},
				Rule: admissionregistrationv1beta1.Rule{
					APIGroups:   []string{crd.Spec.Group},
					APIVersions: v1.ApiSupportedWebhookVersions,
					Resources:   []string{crd.Spec.Names.Plural},
				},
			}},
			ClientConfig: admissionregistrationv1beta1.WebhookClientConfig{
				Service: &admissionregistrationv1beta1.ServiceReference{
					Namespace: kv.Namespace,
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

	validationWebhook := &admissionregistrationv1beta1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "virt-operator-tmp-webhook",
		},
		Webhooks: webhooks,
	}
	injectOperatorMetadata(kv, &validationWebhook.ObjectMeta, version, imageRegistry, id, true)

	expectations.ValidationWebhook.RaiseExpectations(kvkey, 1, 0)
	_, err = clientset.AdmissionregistrationV1beta1().ValidatingWebhookConfigurations().Create(validationWebhook)
	if err != nil {
		expectations.ValidationWebhook.LowerExpectations(kvkey, 1, 0)
		return fmt.Errorf("unable to create validation webhook: %v", err)
	}
	log.Log.V(2).Infof("Validation webhook created for image %s and registry %s", version, imageRegistry)

	return nil
}

func createOrUpdateRoleBinding(rb *rbacv1.RoleBinding,
	kv *v1.KubeVirt,
	imageTag string,
	imageRegistry string,
	id string,
	namespace string,
	stores util.Stores,
	clientset kubecli.KubevirtClient,
	expectations *util.Expectations) error {

	if !stores.ServiceMonitorEnabled && (rb.Name == rbac.MONITOR_SERVICEACCOUNT_NAME) {
		return nil
	}

	var err error
	rbac := clientset.RbacV1()

	var cachedRb *rbacv1.RoleBinding

	kvkey, err := controller.KeyFunc(kv)
	if err != nil {
		return err
	}

	rb = rb.DeepCopy()
	obj, exists, _ := stores.RoleBindingCache.Get(rb)

	if exists {
		cachedRb = obj.(*rbacv1.RoleBinding)
	}

	injectOperatorMetadata(kv, &rb.ObjectMeta, imageTag, imageRegistry, id, true)
	if !exists {
		// Create non existent
		expectations.RoleBinding.RaiseExpectations(kvkey, 1, 0)
		_, err := rbac.RoleBindings(namespace).Create(rb)
		if err != nil {
			expectations.RoleBinding.LowerExpectations(kvkey, 1, 0)
			return fmt.Errorf("unable to create rolebinding %+v: %v", rb, err)
		}
		log.Log.V(2).Infof("rolebinding %v created", rb.GetName())
	} else if !objectMatchesVersion(&cachedRb.ObjectMeta, imageTag, imageRegistry, id, kv.GetGeneration()) {
		// Update existing, we don't need to patch for rbac rules.
		_, err = rbac.RoleBindings(namespace).Update(rb)
		if err != nil {
			return fmt.Errorf("unable to update rolebinding %+v: %v", rb, err)
		}
		log.Log.V(2).Infof("rolebinding %v updated", rb.GetName())

	} else {
		log.Log.V(4).Infof("rolebinding %v already exists", rb.GetName())
	}

	return nil
}

func createOrUpdateRole(r *rbacv1.Role,
	kv *v1.KubeVirt,
	imageTag string,
	imageRegistry string,
	id string,
	namespace string,
	stores util.Stores,
	clientset kubecli.KubevirtClient,
	expectations *util.Expectations) error {

	if !stores.ServiceMonitorEnabled && (r.Name == rbac.MONITOR_SERVICEACCOUNT_NAME) {
		return nil
	}

	var err error
	rbac := clientset.RbacV1()

	var cachedR *rbacv1.Role

	kvkey, err := controller.KeyFunc(kv)
	if err != nil {
		return err
	}

	r = r.DeepCopy()
	obj, exists, _ := stores.RoleCache.Get(r)
	if exists {
		cachedR = obj.(*rbacv1.Role)
	}

	injectOperatorMetadata(kv, &r.ObjectMeta, imageTag, imageRegistry, id, true)
	if !exists {
		// Create non existent
		expectations.Role.RaiseExpectations(kvkey, 1, 0)
		_, err := rbac.Roles(namespace).Create(r)
		if err != nil {
			expectations.Role.LowerExpectations(kvkey, 1, 0)
			return fmt.Errorf("unable to create role %+v: %v", r, err)
		}
		log.Log.V(2).Infof("role %v created", r.GetName())
	} else if !objectMatchesVersion(&cachedR.ObjectMeta, imageTag, imageRegistry, id, kv.GetGeneration()) {
		// Update existing, we don't need to patch for rbac rules.
		_, err = rbac.Roles(namespace).Update(r)
		if err != nil {
			return fmt.Errorf("unable to update role %+v: %v", r, err)
		}
		log.Log.V(2).Infof("role %v updated", r.GetName())

	} else {
		log.Log.V(4).Infof("role %v already exists", r.GetName())
	}
	return nil
}

func createOrUpdateClusterRoleBinding(crb *rbacv1.ClusterRoleBinding,
	kv *v1.KubeVirt,
	imageTag string,
	imageRegistry string,
	id string,
	stores util.Stores,
	clientset kubecli.KubevirtClient,
	expectations *util.Expectations) error {

	var err error
	rbac := clientset.RbacV1()

	var cachedCrb *rbacv1.ClusterRoleBinding

	kvkey, err := controller.KeyFunc(kv)
	if err != nil {
		return err
	}

	crb = crb.DeepCopy()
	obj, exists, _ := stores.ClusterRoleBindingCache.Get(crb)
	if exists {
		cachedCrb = obj.(*rbacv1.ClusterRoleBinding)
	}

	injectOperatorMetadata(kv, &crb.ObjectMeta, imageTag, imageRegistry, id, true)
	if !exists {
		// Create non existent
		expectations.ClusterRoleBinding.RaiseExpectations(kvkey, 1, 0)
		_, err := rbac.ClusterRoleBindings().Create(crb)
		if err != nil {
			expectations.ClusterRoleBinding.LowerExpectations(kvkey, 1, 0)
			return fmt.Errorf("unable to create clusterrolebinding %+v: %v", crb, err)
		}
		log.Log.V(2).Infof("clusterrolebinding %v created", crb.GetName())
	} else if !objectMatchesVersion(&cachedCrb.ObjectMeta, imageTag, imageRegistry, id, kv.GetGeneration()) {
		// Update existing, we don't need to patch for rbac rules.
		_, err = rbac.ClusterRoleBindings().Update(crb)
		if err != nil {
			return fmt.Errorf("unable to update clusterrolebinding %+v: %v", crb, err)
		}
		log.Log.V(2).Infof("clusterrolebinding %v updated", crb.GetName())

	} else {
		log.Log.V(4).Infof("clusterrolebinding %v already exists", crb.GetName())
	}

	return nil
}

func createOrUpdateClusterRole(cr *rbacv1.ClusterRole,
	kv *v1.KubeVirt,
	imageTag string,
	imageRegistry string,
	id string,
	stores util.Stores,
	clientset kubecli.KubevirtClient,
	expectations *util.Expectations) error {

	var err error
	rbac := clientset.RbacV1()

	var cachedCr *rbacv1.ClusterRole

	kvkey, err := controller.KeyFunc(kv)
	if err != nil {
		return err
	}

	cr = cr.DeepCopy()
	obj, exists, _ := stores.ClusterRoleCache.Get(cr)

	if exists {
		cachedCr = obj.(*rbacv1.ClusterRole)
	}

	injectOperatorMetadata(kv, &cr.ObjectMeta, imageTag, imageRegistry, id, true)
	if !exists {
		// Create non existent
		expectations.ClusterRole.RaiseExpectations(kvkey, 1, 0)
		_, err := rbac.ClusterRoles().Create(cr)
		if err != nil {
			expectations.ClusterRole.LowerExpectations(kvkey, 1, 0)
			return fmt.Errorf("unable to create clusterrole %+v: %v", cr, err)
		}
		log.Log.V(2).Infof("clusterrole %v created", cr.GetName())
	} else if !objectMatchesVersion(&cachedCr.ObjectMeta, imageTag, imageRegistry, id, kv.GetGeneration()) {
		// Update existing, we don't need to patch for rbac rules.
		_, err = rbac.ClusterRoles().Update(cr)
		if err != nil {
			return fmt.Errorf("unable to update clusterrole %+v: %v", cr, err)
		}
		log.Log.V(2).Infof("clusterrole %v updated", cr.GetName())

	} else {
		log.Log.V(4).Infof("clusterrole %v already exists", cr.GetName())
	}

	return nil
}

func createOrUpdateConfigMaps(kv *v1.KubeVirt,
	targetStrategy *InstallStrategy,
	stores util.Stores,
	clientset kubecli.KubevirtClient,
	expectations *util.Expectations) error {

	core := clientset.CoreV1()

	kvkey, err := controller.KeyFunc(kv)
	if err != nil {
		return err
	}

	version := kv.Status.TargetKubeVirtVersion
	imageRegistry := kv.Status.TargetKubeVirtRegistry
	id := kv.Status.TargetDeploymentID

	for _, cm := range targetStrategy.configMaps {

		if cm.Name == components.KubeVirtCASecretName {
			continue
		}

		var cachedCM *corev1.ConfigMap

		cm := cm.DeepCopy()
		obj, exists, _ := stores.ConfigMapCache.Get(cm)
		if exists {
			cachedCM = obj.(*corev1.ConfigMap)
		}

		injectOperatorMetadata(kv, &cm.ObjectMeta, version, imageRegistry, id, true)
		if !exists {
			// Create non existent
			expectations.ConfigMap.RaiseExpectations(kvkey, 1, 0)
			_, err := core.ConfigMaps(cm.Namespace).Create(cm)
			if err != nil {
				expectations.ConfigMap.LowerExpectations(kvkey, 1, 0)
				return fmt.Errorf("unable to create config map %+v: %v", cm, err)
			}
			log.Log.V(2).Infof("config map %v created", cm.GetName())

		} else if !objectMatchesVersion(&cachedCM.ObjectMeta, version, imageRegistry, id, kv.GetGeneration()) {
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

			_, err = core.ConfigMaps(cm.Namespace).Patch(cm.Name, types.JSONPatchType, generatePatchBytes(ops))
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
func rolloutNonCompatibleCRDChanges(kv *v1.KubeVirt,
	targetStrategy *InstallStrategy,
	stores util.Stores,
	clientset kubecli.KubevirtClient,
	expectations *util.Expectations) error {

	ext := clientset.ExtensionsClient()

	version := kv.Status.TargetKubeVirtVersion
	imageRegistry := kv.Status.TargetKubeVirtRegistry
	id := kv.Status.TargetDeploymentID

	for _, crd := range targetStrategy.crds {
		var cachedCrd *extv1beta1.CustomResourceDefinition

		crd := crd.DeepCopy()
		obj, exists, _ := stores.CrdCache.Get(crd)
		if exists {
			cachedCrd = obj.(*extv1beta1.CustomResourceDefinition)
		}

		injectOperatorMetadata(kv, &crd.ObjectMeta, version, imageRegistry, id, true)
		if exists && objectMatchesVersion(&cachedCrd.ObjectMeta, version, imageRegistry, id, kv.GetGeneration()) {
			// Patch if in the deployed version the subresource is not enabled
			var ops []string

			// enable the status subresources now, in case that they were disabled before
			if crd.Spec.Subresources == nil || crd.Spec.Subresources.Status == nil {
				continue
			} else if crd.Spec.Subresources != nil && crd.Spec.Subresources.Status != nil {
				if cachedCrd.Spec.Subresources != nil && cachedCrd.Spec.Subresources.Status != nil {
					continue
				}
			}

			// Add Spec Patch
			newSpec, err := json.Marshal(crd.Spec)
			if err != nil {
				return err
			}
			ops = append(ops, fmt.Sprintf(`{ "op": "replace", "path": "/spec", "value": %s }`, string(newSpec)))

			_, err = ext.ApiextensionsV1beta1().CustomResourceDefinitions().Patch(crd.Name, types.JSONPatchType, generatePatchBytes(ops))
			if err != nil {
				return fmt.Errorf("unable to patch crd %+v: %v", crd, err)
			}
			log.Log.V(2).Infof("crd %v updated", crd.GetName())

		} else {
			log.Log.V(4).Infof("crd %v is up-to-date", crd.GetName())
		}
	}

	return nil
}

func createOrUpdateCrds(kv *v1.KubeVirt,
	targetStrategy *InstallStrategy,
	stores util.Stores,
	clientset kubecli.KubevirtClient,
	expectations *util.Expectations) error {

	ext := clientset.ExtensionsClient()

	kvkey, err := controller.KeyFunc(kv)
	if err != nil {
		return err
	}

	version := kv.Status.TargetKubeVirtVersion
	imageRegistry := kv.Status.TargetKubeVirtRegistry
	id := kv.Status.TargetDeploymentID

	for _, crd := range targetStrategy.crds {
		var cachedCrd *extv1beta1.CustomResourceDefinition

		crd := crd.DeepCopy()
		obj, exists, _ := stores.CrdCache.Get(crd)
		if exists {
			cachedCrd = obj.(*extv1beta1.CustomResourceDefinition)
		}

		injectOperatorMetadata(kv, &crd.ObjectMeta, version, imageRegistry, id, true)
		if !exists {
			// Create non existent
			expectations.Crd.RaiseExpectations(kvkey, 1, 0)
			_, err := ext.ApiextensionsV1beta1().CustomResourceDefinitions().Create(crd)
			if err != nil {
				expectations.Crd.LowerExpectations(kvkey, 1, 0)
				return fmt.Errorf("unable to create crd %+v: %v", crd, err)
			}
			log.Log.V(2).Infof("crd %v created", crd.GetName())

		} else if !objectMatchesVersion(&cachedCrd.ObjectMeta, version, imageRegistry, id, kv.GetGeneration()) {
			// Patch if old version
			var ops []string

			// Add Labels and Annotations Patches
			labelAnnotationPatch, err := createLabelsAndAnnotationsPatch(&crd.ObjectMeta)
			if err != nil {
				return err
			}
			ops = append(ops, labelAnnotationPatch...)

			// subresource support needs to be introduced carefully after the controll plane roll-over
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

			_, err = ext.ApiextensionsV1beta1().CustomResourceDefinitions().Patch(crd.Name, types.JSONPatchType, generatePatchBytes(ops))
			if err != nil {
				return fmt.Errorf("unable to patch crd %+v: %v", crd, err)
			}
			log.Log.V(2).Infof("crd %v updated", crd.GetName())

		} else {
			log.Log.V(4).Infof("crd %v is up-to-date", crd.GetName())
		}
	}

	return nil
}

func shouldBackupRBACObject(kv *v1.KubeVirt, objectMeta *metav1.ObjectMeta) (bool, string, string, string) {
	curVersion := kv.Status.TargetKubeVirtVersion
	curImageRegistry := kv.Status.TargetKubeVirtRegistry
	curID := kv.Status.TargetDeploymentID

	if objectMatchesVersion(objectMeta, curVersion, curImageRegistry, curID, kv.GetGeneration()) {
		// matches current target version already, so doesn't need backup
		return false, "", "", ""
	}

	if objectMeta.Annotations == nil {
		return false, "", "", ""
	}

	_, ok := objectMeta.Annotations[v1.EphemeralBackupObject]
	if ok {
		// ephemeral backup objects don't need to be backed up because
		// they are the backup
		return false, "", "", ""
	}

	version, ok := objectMeta.Annotations[v1.InstallStrategyVersionAnnotation]
	if !ok {
		return false, "", "", ""
	}

	imageRegistry, ok := objectMeta.Annotations[v1.InstallStrategyRegistryAnnotation]
	if !ok {
		return false, "", "", ""
	}

	id, ok := objectMeta.Annotations[v1.InstallStrategyIdentifierAnnotation]
	if !ok {
		return false, "", "", ""
	}

	return true, version, imageRegistry, id

}

func needsClusterRoleBindingBackup(kv *v1.KubeVirt, stores util.Stores, crb *rbacv1.ClusterRoleBinding) bool {

	backup, imageTag, imageRegistry, id := shouldBackupRBACObject(kv, &crb.ObjectMeta)
	if !backup {
		return false
	}

	// loop through cache and determine if there's an ephemeral backup
	// for this object already
	objects := stores.ClusterRoleBindingCache.List()
	for _, obj := range objects {
		cachedCrb, ok := obj.(*rbacv1.ClusterRoleBinding)

		if !ok ||
			cachedCrb.DeletionTimestamp != nil ||
			crb.Annotations == nil {
			continue
		}

		uid, ok := cachedCrb.Annotations[v1.EphemeralBackupObject]
		if !ok {
			// this is not an ephemeral backup object
			continue
		}

		if uid == string(crb.UID) && objectMatchesVersion(&cachedCrb.ObjectMeta, imageTag, imageRegistry, id, kv.GetGeneration()) {
			// found backup. UID matches and versions match
			// note, it's possible for a single UID to have multiple backups with
			// different versions
			return false
		}
	}

	return true
}

func needsClusterRoleBackup(kv *v1.KubeVirt, stores util.Stores, cr *rbacv1.ClusterRole) bool {
	backup, imageTag, imageRegistry, id := shouldBackupRBACObject(kv, &cr.ObjectMeta)
	if !backup {
		return false
	}

	// loop through cache and determine if there's an ephemeral backup
	// for this object already
	objects := stores.ClusterRoleCache.List()
	for _, obj := range objects {
		cachedCr, ok := obj.(*rbacv1.ClusterRole)

		if !ok ||
			cachedCr.DeletionTimestamp != nil ||
			cr.Annotations == nil {
			continue
		}

		uid, ok := cachedCr.Annotations[v1.EphemeralBackupObject]
		if !ok {
			// this is not an ephemeral backup object
			continue
		}

		if uid == string(cr.UID) && objectMatchesVersion(&cachedCr.ObjectMeta, imageTag, imageRegistry, id, kv.GetGeneration()) {
			// found backup. UID matches and versions match
			// note, it's possible for a single UID to have multiple backups with
			// different versions
			return false
		}
	}

	return true
}

func needsRoleBindingBackup(kv *v1.KubeVirt, stores util.Stores, rb *rbacv1.RoleBinding) bool {

	backup, imageTag, imageRegistry, id := shouldBackupRBACObject(kv, &rb.ObjectMeta)
	if !backup {
		return false
	}

	// loop through cache and determine if there's an ephemeral backup
	// for this object already
	objects := stores.RoleBindingCache.List()
	for _, obj := range objects {
		cachedRb, ok := obj.(*rbacv1.RoleBinding)

		if !ok ||
			cachedRb.DeletionTimestamp != nil ||
			rb.Annotations == nil {
			continue
		}

		uid, ok := cachedRb.Annotations[v1.EphemeralBackupObject]
		if !ok {
			// this is not an ephemeral backup object
			continue
		}

		if uid == string(rb.UID) && objectMatchesVersion(&cachedRb.ObjectMeta, imageTag, imageRegistry, id, kv.GetGeneration()) {
			// found backup. UID matches and versions match
			// note, it's possible for a single UID to have multiple backups with
			// different versions
			return false
		}
	}

	return true
}

func needsRoleBackup(kv *v1.KubeVirt, stores util.Stores, r *rbacv1.Role) bool {

	backup, imageTag, imageRegistry, id := shouldBackupRBACObject(kv, &r.ObjectMeta)
	if !backup {
		return false
	}

	// loop through cache and determine if there's an ephemeral backup
	// for this object already
	objects := stores.RoleCache.List()
	for _, obj := range objects {
		cachedR, ok := obj.(*rbacv1.Role)

		if !ok ||
			cachedR.DeletionTimestamp != nil ||
			r.Annotations == nil {
			continue
		}

		uid, ok := cachedR.Annotations[v1.EphemeralBackupObject]
		if !ok {
			// this is not an ephemeral backup object
			continue
		}

		if uid == string(r.UID) && objectMatchesVersion(&cachedR.ObjectMeta, imageTag, imageRegistry, id, kv.GetGeneration()) {
			// found backup. UID matches and versions match
			// note, it's possible for a single UID to have multiple backups with
			// different versions
			return false
		}
	}

	return true
}

func backupRbac(kv *v1.KubeVirt,
	stores util.Stores,
	clientset kubecli.KubevirtClient,
	expectations *util.Expectations) error {

	kvkey, err := controller.KeyFunc(kv)
	if err != nil {
		return err
	}

	rbac := clientset.RbacV1()

	// Backup existing ClusterRoles
	objects := stores.ClusterRoleCache.List()
	for _, obj := range objects {
		cachedCr, ok := obj.(*rbacv1.ClusterRole)
		if !ok || !needsClusterRoleBackup(kv, stores, cachedCr) {
			continue
		}
		imageTag, ok := cachedCr.Annotations[v1.InstallStrategyVersionAnnotation]
		if !ok {
			continue
		}
		imageRegistry, ok := cachedCr.Annotations[v1.InstallStrategyRegistryAnnotation]
		if !ok {
			continue
		}
		id, ok := cachedCr.Annotations[v1.InstallStrategyIdentifierAnnotation]
		if !ok {
			continue
		}

		// needs backup, so create a new object that will temporarily
		// backup this object while the update is in progress.
		cr := cachedCr.DeepCopy()
		cr.ObjectMeta = metav1.ObjectMeta{
			GenerateName: cachedCr.Name,
		}
		injectOperatorMetadata(kv, &cr.ObjectMeta, imageTag, imageRegistry, id, true)
		cr.Annotations[v1.EphemeralBackupObject] = string(cachedCr.UID)

		// Create backup
		expectations.ClusterRole.RaiseExpectations(kvkey, 1, 0)
		_, err := rbac.ClusterRoles().Create(cr)
		if err != nil {
			expectations.ClusterRole.LowerExpectations(kvkey, 1, 0)
			return fmt.Errorf("unable to create backup clusterrole %+v: %v", cr, err)
		}
		log.Log.V(2).Infof("backup clusterrole %v created", cr.GetName())
	}

	// Backup existing ClusterRoleBindings
	objects = stores.ClusterRoleBindingCache.List()
	for _, obj := range objects {
		cachedCrb, ok := obj.(*rbacv1.ClusterRoleBinding)
		if !ok || !needsClusterRoleBindingBackup(kv, stores, cachedCrb) {
			continue
		}
		imageTag, ok := cachedCrb.Annotations[v1.InstallStrategyVersionAnnotation]
		if !ok {
			continue
		}
		imageRegistry, ok := cachedCrb.Annotations[v1.InstallStrategyRegistryAnnotation]
		if !ok {
			continue
		}
		id, ok := cachedCrb.Annotations[v1.InstallStrategyIdentifierAnnotation]
		if !ok {
			continue
		}

		// needs backup, so create a new object that will temporarily
		// backup this object while the update is in progress.
		crb := cachedCrb.DeepCopy()
		crb.ObjectMeta = metav1.ObjectMeta{
			GenerateName: cachedCrb.Name,
		}
		injectOperatorMetadata(kv, &crb.ObjectMeta, imageTag, imageRegistry, id, true)
		crb.Annotations[v1.EphemeralBackupObject] = string(cachedCrb.UID)

		// Create backup
		expectations.ClusterRoleBinding.RaiseExpectations(kvkey, 1, 0)
		_, err := rbac.ClusterRoleBindings().Create(crb)
		if err != nil {
			expectations.ClusterRoleBinding.LowerExpectations(kvkey, 1, 0)
			return fmt.Errorf("unable to create backup clusterrolebinding %+v: %v", crb, err)
		}
		log.Log.V(2).Infof("backup clusterrolebinding %v created", crb.GetName())
	}

	// Backup existing Roles
	objects = stores.RoleCache.List()
	for _, obj := range objects {
		cachedCr, ok := obj.(*rbacv1.Role)
		if !ok || !needsRoleBackup(kv, stores, cachedCr) {
			continue
		}
		imageTag, ok := cachedCr.Annotations[v1.InstallStrategyVersionAnnotation]
		if !ok {
			continue
		}
		imageRegistry, ok := cachedCr.Annotations[v1.InstallStrategyRegistryAnnotation]
		if !ok {
			continue
		}
		id, ok := cachedCr.Annotations[v1.InstallStrategyIdentifierAnnotation]
		if !ok {
			continue
		}

		// needs backup, so create a new object that will temporarily
		// backup this object while the update is in progress.
		r := cachedCr.DeepCopy()
		r.ObjectMeta = metav1.ObjectMeta{
			GenerateName: cachedCr.Name,
		}
		injectOperatorMetadata(kv, &r.ObjectMeta, imageTag, imageRegistry, id, true)
		r.Annotations[v1.EphemeralBackupObject] = string(cachedCr.UID)

		// Create backup
		expectations.Role.RaiseExpectations(kvkey, 1, 0)
		_, err := rbac.Roles(cachedCr.Namespace).Create(r)
		if err != nil {
			expectations.Role.LowerExpectations(kvkey, 1, 0)
			return fmt.Errorf("unable to create backup role %+v: %v", r, err)
		}
		log.Log.V(2).Infof("backup role %v created", r.GetName())
	}

	// Backup existing RoleBindings
	objects = stores.RoleBindingCache.List()
	for _, obj := range objects {
		cachedRb, ok := obj.(*rbacv1.RoleBinding)
		if !ok || !needsRoleBindingBackup(kv, stores, cachedRb) {
			continue
		}
		imageTag, ok := cachedRb.Annotations[v1.InstallStrategyVersionAnnotation]
		if !ok {
			continue
		}
		imageRegistry, ok := cachedRb.Annotations[v1.InstallStrategyRegistryAnnotation]
		if !ok {
			continue
		}
		id, ok := cachedRb.Annotations[v1.InstallStrategyIdentifierAnnotation]
		if !ok {
			continue
		}

		// needs backup, so create a new object that will temporarily
		// backup this object while the update is in progress.
		rb := cachedRb.DeepCopy()
		rb.ObjectMeta = metav1.ObjectMeta{
			GenerateName: cachedRb.Name,
		}
		injectOperatorMetadata(kv, &rb.ObjectMeta, imageTag, imageRegistry, id, true)
		rb.Annotations[v1.EphemeralBackupObject] = string(cachedRb.UID)

		// Create backup
		expectations.RoleBinding.RaiseExpectations(kvkey, 1, 0)
		_, err := rbac.RoleBindings(cachedRb.Namespace).Create(rb)
		if err != nil {
			expectations.RoleBinding.LowerExpectations(kvkey, 1, 0)
			return fmt.Errorf("unable to create backup rolebinding %+v: %v", rb, err)
		}
		log.Log.V(2).Infof("backup rolebinding %v created", rb.GetName())
	}

	return nil
}

func createOrUpdateRbac(kv *v1.KubeVirt,
	targetStrategy *InstallStrategy,
	stores util.Stores,
	clientset kubecli.KubevirtClient,
	expectations *util.Expectations) error {

	kvkey, err := controller.KeyFunc(kv)
	if err != nil {
		return err
	}

	core := clientset.CoreV1()

	version := kv.Status.TargetKubeVirtVersion
	imageRegistry := kv.Status.TargetKubeVirtRegistry
	id := kv.Status.TargetDeploymentID

	// create/update ServiceAccounts
	for _, sa := range targetStrategy.serviceAccounts {
		var cachedSa *corev1.ServiceAccount

		sa := sa.DeepCopy()
		obj, exists, _ := stores.ServiceAccountCache.Get(sa)
		if exists {
			cachedSa = obj.(*corev1.ServiceAccount)
		}

		injectOperatorMetadata(kv, &sa.ObjectMeta, version, imageRegistry, id, true)
		if !exists {
			// Create non existent
			expectations.ServiceAccount.RaiseExpectations(kvkey, 1, 0)
			_, err := core.ServiceAccounts(kv.Namespace).Create(sa)
			if err != nil {
				expectations.ServiceAccount.LowerExpectations(kvkey, 1, 0)
				return fmt.Errorf("unable to create serviceaccount %+v: %v", sa, err)
			}
			log.Log.V(2).Infof("serviceaccount %v created", sa.GetName())
		} else if !objectMatchesVersion(&cachedSa.ObjectMeta, version, imageRegistry, id, kv.GetGeneration()) {
			// Patch if old version
			var ops []string

			// Patch Labels and Annotations
			labelAnnotationPatch, err := createLabelsAndAnnotationsPatch(&sa.ObjectMeta)
			if err != nil {
				return err
			}
			ops = append(ops, labelAnnotationPatch...)

			_, err = core.ServiceAccounts(kv.Namespace).Patch(sa.Name, types.JSONPatchType, generatePatchBytes(ops))
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
	for _, cr := range targetStrategy.clusterRoles {
		err := createOrUpdateClusterRole(cr,
			kv,
			version,
			imageRegistry,
			id,
			stores,
			clientset,
			expectations)
		if err != nil {
			return err
		}
	}

	// create/update ClusterRoleBindings
	for _, crb := range targetStrategy.clusterRoleBindings {
		err := createOrUpdateClusterRoleBinding(crb,
			kv,
			version,
			imageRegistry,
			id,
			stores,
			clientset,
			expectations)
		if err != nil {
			return err
		}

	}

	// create/update Roles
	for _, r := range targetStrategy.roles {
		err := createOrUpdateRole(r,
			kv,
			version,
			imageRegistry,
			id,
			kv.Namespace,
			stores,
			clientset,
			expectations)
		if err != nil {
			return err
		}
	}

	// create/update RoleBindings
	for _, rb := range targetStrategy.roleBindings {
		err := createOrUpdateRoleBinding(rb,
			kv,
			version,
			imageRegistry,
			id,
			kv.Namespace,
			stores,
			clientset,
			expectations)
		if err != nil {
			return err
		}
	}

	return nil
}

func getPortProtocol(port *corev1.ServicePort) corev1.Protocol {
	if port.Protocol == "" {
		return corev1.ProtocolTCP
	}

	return port.Protocol
}

func isServiceClusterIP(service *corev1.Service) bool {
	if service.Spec.Type == "" || service.Spec.Type == corev1.ServiceTypeClusterIP {
		return true
	}
	return false
}

// This function determines how to process updating a service endpoint.
//
// If the update involves fields that can't be mutated, then the deleteAndReplace bool will be set.
// If the update involves patching fields, then a list of patch operations will be returned.
// If neither patchOps nor deleteAndReplace are set, then the service is already up-to-date
//
// NOTE. see the unit test that exercises this function to further learn about the expected behavior.
func generateServicePatch(kv *v1.KubeVirt,
	cachedService *corev1.Service,
	service *corev1.Service) ([]string, bool, error) {

	var patchOps []string
	var deleteAndReplace bool

	version := kv.Status.TargetKubeVirtVersion
	imageRegistry := kv.Status.TargetKubeVirtRegistry
	id := kv.Status.TargetDeploymentID

	// First check if there's anything to do.
	if objectMatchesVersion(&cachedService.ObjectMeta, version, imageRegistry, id, kv.GetGeneration()) {
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

		// clusterIP is not mutable. A ClusterIP == "" will mean one is dynamically assigned.
		// It is okay if the cached service has a ClusterIP and the target one is empty. We just
		// ensure the cached ClusterIP is carried over in the Patch.
		deleteAndReplace = true
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

func createOrUpdateAPIServices(kv *v1.KubeVirt,
	targetStrategy *InstallStrategy,
	stores util.Stores,
	aggregatorClient APIServiceInterface,
	expectations *util.Expectations,
	caBundle []byte,
) error {

	version := kv.Status.TargetKubeVirtVersion
	imageRegistry := kv.Status.TargetKubeVirtRegistry
	id := kv.Status.TargetDeploymentID

	kvkey, err := controller.KeyFunc(kv)
	if err != nil {
		return err
	}

	for _, apiService := range targetStrategy.apiServices {
		var cachedAPIService *v1beta1.APIService
		apiService = apiService.DeepCopy()

		apiService.Spec.CABundle = caBundle

		obj, exists, _ := stores.APIServiceCache.Get(apiService)
		// since these objects was in the past unmanaged, reconcile and pick it up if it exists
		if !exists {
			cachedAPIService, err = aggregatorClient.Get(apiService.Name, metav1.GetOptions{})
			if errors.IsNotFound(err) {
				exists = false
			} else if err != nil {
				return err
			} else {
				exists = true
			}
		} else if exists {
			cachedAPIService = obj.(*v1beta1.APIService)
		}

		certsMatch := true
		if exists {
			if !reflect.DeepEqual(apiService.Spec.CABundle, cachedAPIService.Spec.CABundle) {
				certsMatch = false
			}
		}

		injectOperatorMetadata(kv, &apiService.ObjectMeta, version, imageRegistry, id, true)
		if !exists {
			expectations.APIService.RaiseExpectations(kvkey, 1, 0)
			_, err := aggregatorClient.Create(apiService)
			if err != nil {
				expectations.APIService.LowerExpectations(kvkey, 1, 0)
				return fmt.Errorf("unable to create apiservice %+v: %v", apiService, err)
			}
		} else {
			if !objectMatchesVersion(&cachedAPIService.ObjectMeta, version, imageRegistry, id, kv.GetGeneration()) || !certsMatch {
				// Patch if old version
				var ops []string

				// Add Labels and Annotations Patches
				labelAnnotationPatch, err := createLabelsAndAnnotationsPatch(&apiService.ObjectMeta)
				if err != nil {
					return err
				}
				ops = append(ops, labelAnnotationPatch...)

				// Add Spec Patch
				spec, err := json.Marshal(apiService.Spec)
				if err != nil {
					return err
				}
				ops = append(ops, fmt.Sprintf(`{ "op": "replace", "path": "/spec", "value": %s }`, string(spec)))

				_, err = aggregatorClient.Patch(apiService.Name, types.JSONPatchType, generatePatchBytes(ops))
				if err != nil {
					return fmt.Errorf("unable to patch apiservice %+v: %v", apiService, err)
				}
				log.Log.V(2).Infof("apiservice %v updated", apiService.GetName())

			} else {
				log.Log.V(4).Infof("apiservice %v is up-to-date", apiService.GetName())
			}
		}
	}
	return nil
}

func createOrUpdateMutatingWebhookConfigurations(kv *v1.KubeVirt,
	targetStrategy *InstallStrategy,
	stores util.Stores,
	clientset kubecli.KubevirtClient,
	expectations *util.Expectations,
	caBundle []byte,
) error {

	version := kv.Status.TargetKubeVirtVersion
	imageRegistry := kv.Status.TargetKubeVirtRegistry
	id := kv.Status.TargetDeploymentID

	kvkey, err := controller.KeyFunc(kv)
	if err != nil {
		return err
	}

	for _, webhook := range targetStrategy.mutatingWebhookConfigurations {
		var cachedWebhook *admissionregistrationv1beta1.MutatingWebhookConfiguration
		webhook = webhook.DeepCopy()

		for i, _ := range webhook.Webhooks {
			webhook.Webhooks[i].ClientConfig.CABundle = caBundle
		}

		obj, exists, _ := stores.MutatingWebhookCache.Get(webhook)
		// since these objects was in the past unmanaged, reconcile and pick it up if it exists
		if !exists {
			cachedWebhook, err = clientset.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Get(webhook.Name, metav1.GetOptions{})
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
			expectations.MutatingWebhook.RaiseExpectations(kvkey, 1, 0)
			_, err := clientset.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Create(webhook)
			if err != nil {
				expectations.MutatingWebhook.LowerExpectations(kvkey, 1, 0)
				return fmt.Errorf("unable to create mutatingwebhook %+v: %v", webhook, err)
			}
		} else {
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

				_, err = clientset.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Patch(webhook.Name, types.JSONPatchType, generatePatchBytes(ops))
				if err != nil {
					return fmt.Errorf("unable to patch mutatingwebhookconfiguration %+v: %v", webhook, err)
				}
				log.Log.V(2).Infof("mutatingwebhoookconfiguration %v updated", webhook.GetName())

			} else {
				log.Log.V(4).Infof("mutatingwebhookconfiguration %v is up-to-date", webhook.GetName())
			}
		}
	}
	return nil
}

func createOrUpdateValidatingWebhookConfigurations(kv *v1.KubeVirt,
	targetStrategy *InstallStrategy,
	stores util.Stores,
	clientset kubecli.KubevirtClient,
	expectations *util.Expectations,
	caBundle []byte,
) error {

	version := kv.Status.TargetKubeVirtVersion
	imageRegistry := kv.Status.TargetKubeVirtRegistry
	id := kv.Status.TargetDeploymentID

	kvkey, err := controller.KeyFunc(kv)
	if err != nil {
		return err
	}

	for _, webhook := range targetStrategy.validatingWebhookConfigurations {
		var cachedWebhook *admissionregistrationv1beta1.ValidatingWebhookConfiguration
		webhook = webhook.DeepCopy()

		for i, _ := range webhook.Webhooks {
			webhook.Webhooks[i].ClientConfig.CABundle = caBundle
		}

		obj, exists, _ := stores.ValidationWebhookCache.Get(webhook)
		// since these objects was in the past unmanaged, reconcile and pick it up if it exists
		if !exists {
			cachedWebhook, err = clientset.AdmissionregistrationV1beta1().ValidatingWebhookConfigurations().Get(webhook.Name, metav1.GetOptions{})
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
			expectations.ValidationWebhook.RaiseExpectations(kvkey, 1, 0)
			_, err := clientset.AdmissionregistrationV1beta1().ValidatingWebhookConfigurations().Create(webhook)
			if err != nil {
				expectations.ValidationWebhook.LowerExpectations(kvkey, 1, 0)
				return fmt.Errorf("unable to create validatingwebhook %+v: %v", webhook, err)
			}
		} else {
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

				_, err = clientset.AdmissionregistrationV1beta1().ValidatingWebhookConfigurations().Patch(webhook.Name, types.JSONPatchType, generatePatchBytes(ops))
				if err != nil {
					return fmt.Errorf("unable to patch validatingwebhookconfiguration %+v: %v", webhook, err)
				}
				log.Log.V(2).Infof("validatingwebhoookconfiguration %v updated", webhook.GetName())

			} else {
				log.Log.V(4).Infof("validatingwebhookconfiguration %v is up-to-date", webhook.GetName())
			}
		}
	}
	return nil
}

func createOrUpdateCACertificateSecret(
	queue workqueue.RateLimitingInterface,
	kv *v1.KubeVirt,
	targetStrategy *InstallStrategy,
	stores util.Stores,
	clientset kubecli.KubevirtClient,
	expectations *util.Expectations,
	duration *metav1.Duration,
) (caCert *tls.Certificate, err error) {

	for _, secret := range targetStrategy.certificateSecrets {

		// Only work on the ca secret
		if secret.Name != components.KubeVirtCASecretName {
			continue
		}
		caCert, err := createOrUpdateCertificateSecret(queue, kv, stores, clientset, expectations, nil, secret, duration)
		if err != nil {
			return nil, err
		}
		return caCert, nil
	}
	return nil, nil
}

func createOrUpdateCertificateSecret(
	queue workqueue.RateLimitingInterface,
	kv *v1.KubeVirt,
	stores util.Stores,
	clientset kubecli.KubevirtClient,
	expectations *util.Expectations,
	ca *tls.Certificate,
	secret *corev1.Secret,
	duration *metav1.Duration,
) (*tls.Certificate, error) {
	var cachedSecret *corev1.Secret
	secret = secret.DeepCopy()

	log.DefaultLogger().V(4).Infof("checking certificate %v", secret.Name)

	version := kv.Status.TargetKubeVirtVersion
	imageRegistry := kv.Status.TargetKubeVirtRegistry
	id := kv.Status.TargetDeploymentID

	kvkey, err := controller.KeyFunc(kv)
	if err != nil {
		return nil, err
	}

	obj, exists, _ := stores.SecretCache.Get(secret)

	// since these objects was in the past unmanaged, reconcile and pick it up if it exists
	if !exists {
		cachedSecret, err = clientset.CoreV1().Secrets(secret.Namespace).Get(secret.Name, metav1.GetOptions{})
		if errors.IsNotFound(err) {
			exists = false
		} else if err != nil {
			return nil, err
		} else {
			exists = true
		}
	} else if exists {
		cachedSecret = obj.(*corev1.Secret)
	}

	rotateCertificate := false
	if exists {

		crt, err := components.LoadCertificates(cachedSecret)
		if err != nil {
			log.DefaultLogger().Reason(err).Infof("Failed to load certificate from secret %s, will rotate it.", secret.Name)
			rotateCertificate = true
		} else if cachedSecret.Annotations["kubevirt.io/duration"] != duration.String() {
			rotateCertificate = true
		} else {
			rotationTime := components.NextRotationDeadline(crt, ca, duration)
			// We update the certificate if it has passed 80 percent of its lifetime
			if rotationTime.Before(time.Now()) {
				rotateCertificate = true
			}
		}
	}

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
	wakeupDeadline := components.NextRotationDeadline(crt, ca, duration).Sub(time.Now())
	queue.AddAfter(kvkey, wakeupDeadline)

	injectOperatorMetadata(kv, &secret.ObjectMeta, version, imageRegistry, id, true)
	if !exists {
		expectations.Secrets.RaiseExpectations(kvkey, 1, 0)
		_, err := clientset.CoreV1().Secrets(secret.Namespace).Create(secret)
		if err != nil {
			expectations.Secrets.LowerExpectations(kvkey, 1, 0)
			return nil, fmt.Errorf("unable to create secret %+v: %v", secret, err)
		}
	} else {
		if !objectMatchesVersion(&cachedSecret.ObjectMeta, version, imageRegistry, id, kv.GetGeneration()) || rotateCertificate {
			// Patch if old version
			var ops []string

			// Add Labels and Annotations Patches
			labelAnnotationPatch, err := createLabelsAndAnnotationsPatch(&secret.ObjectMeta)
			if err != nil {
				return nil, err
			}
			ops = append(ops, labelAnnotationPatch...)

			// Add Spec Patch
			data, err := json.Marshal(secret.Data)
			if err != nil {
				return nil, err
			}
			ops = append(ops, fmt.Sprintf(`{ "op": "replace", "path": "/data", "value": %s }`, string(data)))

			_, err = clientset.CoreV1().Secrets(secret.Namespace).Patch(secret.Name, types.JSONPatchType, generatePatchBytes(ops))
			if err != nil {
				return nil, fmt.Errorf("unable to patch secret %+v: %v", secret, err)
			}
			log.Log.V(2).Infof("secret %v updated", secret.GetName())

		} else {
			log.Log.V(4).Infof("secret %v is up-to-date", secret.GetName())
		}
	}
	return crt, nil
}

func createOrUpdateCertificateSecrets(
	queue workqueue.RateLimitingInterface,
	kv *v1.KubeVirt,
	targetStrategy *InstallStrategy,
	stores util.Stores,
	clientset kubecli.KubevirtClient,
	expectations *util.Expectations,
	caCert *tls.Certificate,
	duration *metav1.Duration,
) error {

	for _, secret := range targetStrategy.certificateSecrets {

		// The CA certificate needs to be handled separetely and before other secrets
		if secret.Name == components.KubeVirtCASecretName {
			continue
		}

		_, err := createOrUpdateCertificateSecret(queue, kv, stores, clientset, expectations, caCert, secret, duration)
		if err != nil {
			return err
		}
	}
	return nil
}

func createOrUpdateService(kv *v1.KubeVirt,
	targetStrategy *InstallStrategy,
	stores util.Stores,
	clientset kubecli.KubevirtClient,
	expectations *util.Expectations) (bool, error) {

	core := clientset.CoreV1()
	version := kv.Status.TargetKubeVirtVersion
	imageRegistry := kv.Status.TargetKubeVirtRegistry
	id := kv.Status.TargetDeploymentID

	kvkey, err := controller.KeyFunc(kv)
	if err != nil {
		return false, err
	}

	gracePeriod := int64(0)
	deleteOptions := &metav1.DeleteOptions{
		GracePeriodSeconds: &gracePeriod,
	}

	for _, service := range targetStrategy.services {
		var cachedService *corev1.Service
		service = service.DeepCopy()

		obj, exists, _ := stores.ServiceCache.Get(service)
		if exists {
			cachedService = obj.(*corev1.Service)
		}

		injectOperatorMetadata(kv, &service.ObjectMeta, version, imageRegistry, id, true)
		if !exists {
			expectations.Service.RaiseExpectations(kvkey, 1, 0)
			_, err := core.Services(service.Namespace).Create(service)
			if err != nil {
				expectations.Service.LowerExpectations(kvkey, 1, 0)
				return false, fmt.Errorf("unable to create service %+v: %v", service, err)
			}
		} else {

			patchOps, deleteAndReplace, err := generateServicePatch(kv, cachedService, service)
			if err != nil {
				return false, fmt.Errorf("unable to generate service endpoint patch operations for %+v: %v", service, err)
			}

			if deleteAndReplace {
				if cachedService.DeletionTimestamp == nil {
					if key, err := controller.KeyFunc(cachedService); err == nil {
						expectations.Service.AddExpectedDeletion(kvkey, key)
						err := core.Services(service.Namespace).Delete(cachedService.Name, deleteOptions)
						if err != nil {
							expectations.Service.DeletionObserved(kvkey, key)
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
				_, err = core.Services(service.Namespace).Patch(service.Name, types.JSONPatchType, generatePatchBytes(patchOps))
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

func createOrUpdateServiceMonitors(kv *v1.KubeVirt,
	targetStrategy *InstallStrategy,
	stores util.Stores,
	clientset kubecli.KubevirtClient,
	expectations *util.Expectations) error {

	if !stores.ServiceMonitorEnabled {
		return nil
	}

	prometheusClient := clientset.PrometheusClient()

	kvkey, err := controller.KeyFunc(kv)
	if err != nil {
		return err
	}

	version := kv.Status.TargetKubeVirtVersion
	imageRegistry := kv.Status.TargetKubeVirtRegistry
	id := kv.Status.TargetDeploymentID

	for _, serviceMonitor := range targetStrategy.serviceMonitors {
		var cachedServiceMonitor *promv1.ServiceMonitor

		serviceMonitor := serviceMonitor.DeepCopy()
		obj, exists, _ := stores.ServiceMonitorCache.Get(serviceMonitor)
		if exists {
			cachedServiceMonitor = obj.(*promv1.ServiceMonitor)
		}

		injectOperatorMetadata(kv, &serviceMonitor.ObjectMeta, version, imageRegistry, id, true)
		if !exists {
			// Create non existent
			expectations.ServiceMonitor.RaiseExpectations(kvkey, 1, 0)
			_, err := prometheusClient.MonitoringV1().ServiceMonitors(serviceMonitor.Namespace).Create(serviceMonitor)
			if err != nil {
				expectations.ServiceMonitor.LowerExpectations(kvkey, 1, 0)
				return fmt.Errorf("unable to create serviceMonitor %+v: %v", serviceMonitor, err)
			}
			log.Log.V(2).Infof("serviceMonitor %v created", serviceMonitor.GetName())

		} else if !objectMatchesVersion(&cachedServiceMonitor.ObjectMeta, version, imageRegistry, id, kv.GetGeneration()) {
			// Patch if old version
			var ops []string

			// Add Labels and Annotations Patches
			labelAnnotationPatch, err := createLabelsAndAnnotationsPatch(&serviceMonitor.ObjectMeta)
			if err != nil {
				return err
			}
			ops = append(ops, labelAnnotationPatch...)

			// Add Spec Patch
			newSpec, err := json.Marshal(serviceMonitor.Spec)
			if err != nil {
				return err
			}
			ops = append(ops, fmt.Sprintf(`{ "op": "replace", "path": "/spec", "value": %s }`, string(newSpec)))

			_, err = prometheusClient.MonitoringV1().ServiceMonitors(serviceMonitor.Namespace).Patch(serviceMonitor.Name, types.JSONPatchType, generatePatchBytes(ops))
			if err != nil {
				return fmt.Errorf("unable to patch serviceMonitor %+v: %v", serviceMonitor, err)
			}
			log.Log.V(2).Infof("serviceMonitor %v updated", serviceMonitor.GetName())

		} else {
			log.Log.V(4).Infof("serviceMonitor %v is up-to-date", serviceMonitor.GetName())
		}
	}

	return nil
}

func createOrUpdatePrometheusRules(kv *v1.KubeVirt,
	targetStrategy *InstallStrategy,
	stores util.Stores,
	clientset kubecli.KubevirtClient,
	expectations *util.Expectations) error {

	if !stores.PrometheusRulesEnabled {
		return nil
	}

	prometheusClient := clientset.PrometheusClient()

	kvkey, err := controller.KeyFunc(kv)
	if err != nil {
		return err
	}

	version := kv.Status.TargetKubeVirtVersion
	imageRegistry := kv.Status.TargetKubeVirtRegistry
	id := kv.Status.TargetDeploymentID

	for _, prometheusRule := range targetStrategy.prometheusRules {
		var cachedPrometheusRule *promv1.PrometheusRule

		prometheusRule := prometheusRule.DeepCopy()
		obj, exists, _ := stores.PrometheusRuleCache.Get(prometheusRule)
		if exists {
			cachedPrometheusRule = obj.(*promv1.PrometheusRule)
		}

		injectOperatorMetadata(kv, &prometheusRule.ObjectMeta, version, imageRegistry, id, true)
		if !exists {
			// Create non existent
			expectations.PrometheusRule.RaiseExpectations(kvkey, 1, 0)
			_, err := prometheusClient.MonitoringV1().PrometheusRules(prometheusRule.Namespace).Create(prometheusRule)
			if err != nil {
				expectations.PrometheusRule.LowerExpectations(kvkey, 1, 0)
				return fmt.Errorf("unable to create PrometheusRule %+v: %v", prometheusRule, err)
			}
			log.Log.V(2).Infof("PrometheusRule %v created", prometheusRule.GetName())

		} else if !objectMatchesVersion(&cachedPrometheusRule.ObjectMeta, version, imageRegistry, id, kv.GetGeneration()) {
			// Patch if old version
			var ops []string

			// Add Labels and Annotations Patches
			labelAnnotationPatch, err := createLabelsAndAnnotationsPatch(&prometheusRule.ObjectMeta)
			if err != nil {
				return err
			}
			ops = append(ops, labelAnnotationPatch...)

			// Add Spec Patch
			newSpec, err := json.Marshal(prometheusRule.Spec)
			if err != nil {
				return err
			}
			ops = append(ops, fmt.Sprintf(`{ "op": "replace", "path": "/spec", "value": %s }`, string(newSpec)))

			_, err = prometheusClient.MonitoringV1().PrometheusRules(prometheusRule.Namespace).Patch(prometheusRule.Name, types.JSONPatchType, generatePatchBytes(ops))
			if err != nil {
				return fmt.Errorf("unable to patch PrometheusRule %+v: %v", prometheusRule, err)
			}
			log.Log.V(2).Infof("PrometheusRule %v updated", prometheusRule.GetName())

		} else {
			log.Log.V(4).Infof("PrometheusRule %v is up-to-date", prometheusRule.GetName())
		}
	}

	return nil
}

// deprecated, keep it for backwards compatibility
func addOrRemoveSSC(targetStrategy *InstallStrategy,
	prevStrategy *InstallStrategy,
	clientset kubecli.KubevirtClient,
	stores util.Stores,
	addOnly bool) error {

	scc := clientset.SecClient()
	for _, sccPriv := range targetStrategy.customSCCPrivileges {
		var curSccPriv *customSCCPrivilegedAccounts
		if prevStrategy != nil {
			for _, entry := range prevStrategy.customSCCPrivileges {
				if sccPriv.TargetSCC == entry.TargetSCC {
					curSccPriv = entry
					break
				}
			}
		}

		privSCCObj, exists, err := stores.SCCCache.GetByKey(sccPriv.TargetSCC)
		if !exists {
			continue
		} else if err != nil {
			return err
		}

		privSCC, ok := privSCCObj.(*secv1.SecurityContextConstraints)
		if !ok {
			return fmt.Errorf("couldn't cast object to SecurityContextConstraints: %+v", privSCCObj)
		}

		oldUsers := privSCC.Users
		privSCCCopy := privSCC.DeepCopy()
		users := privSCCCopy.Users

		modified := false

		// remove users from previous
		if curSccPriv != nil && !addOnly {
			for _, acc := range curSccPriv.ServiceAccounts {
				shouldRemove := true
				// only remove if the target doesn't contain the same
				// rule, otherwise leave as is.
				for _, targetAcc := range sccPriv.ServiceAccounts {
					if acc == targetAcc {
						shouldRemove = false
						break
					}
				}
				if shouldRemove {
					removed := false
					users, removed = remove(users, acc)
					modified = modified || removed
				}
			}
		}

		// add any users from target that don't already exist
		for _, acc := range sccPriv.ServiceAccounts {
			if !contains(users, acc) {
				users = append(users, acc)
				modified = true
			}
		}

		if modified {
			oldUserBytes, err := json.Marshal(oldUsers)
			if err != nil {
				return err
			}
			userBytes, err := json.Marshal(users)
			if err != nil {
				return err
			}

			test := fmt.Sprintf(`{ "op": "test", "path": "/users", "value": %s }`, string(oldUserBytes))
			patch := fmt.Sprintf(`{ "op": "replace", "path": "/users", "value": %s }`, string(userBytes))

			_, err = scc.SecurityContextConstraints().Patch(sccPriv.TargetSCC, types.JSONPatchType, []byte(fmt.Sprintf("[ %s, %s ]", test, patch)))
			if err != nil {
				return fmt.Errorf("unable to patch scc: %v", err)
			}
		}
	}
	return nil
}

func syncKubevirtNamespaceLabels(kv *v1.KubeVirt, stores util.Stores, clientset kubecli.KubevirtClient) error {

	targetNamespace := kv.ObjectMeta.Namespace
	obj, exists, err := stores.NamespaceCache.GetByKey(targetNamespace)
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
	_, err = clientset.CoreV1().Namespaces().Patch(
		targetNamespace,
		types.MergePatchType,
		[]byte(fmt.Sprintf(`{"metadata":{"labels": %s}}`, labelsPatch)),
	)
	if err != nil {
		log.Log.Errorf("Could not patch kubevirt namespace labels: %s", err.Error())
		return err
	}
	log.Log.Infof("kubevirt namespace labels patched")
	return nil
}

func createOrUpdateSCC(kv *v1.KubeVirt,
	targetStrategy *InstallStrategy,
	stores util.Stores,
	clientset kubecli.KubevirtClient,
	expectations *util.Expectations) error {
	sec := clientset.SecClient()

	if !stores.IsOnOpenshift {
		return nil
	}

	version := kv.Status.TargetKubeVirtVersion
	imageRegistry := kv.Status.TargetKubeVirtRegistry
	id := kv.Status.TargetDeploymentID
	kvkey, err := controller.KeyFunc(kv)
	if err != nil {
		return err
	}

	for _, scc := range targetStrategy.sccs {
		var cachedSCC *secv1.SecurityContextConstraints
		scc := scc.DeepCopy()
		obj, exists, _ := stores.SCCCache.GetByKey(scc.Name)
		if exists {
			cachedSCC = obj.(*secv1.SecurityContextConstraints)
		}

		injectOperatorMetadata(kv, &scc.ObjectMeta, version, imageRegistry, id, true)
		if !exists {
			expectations.SCC.RaiseExpectations(kvkey, 1, 0)
			_, err := sec.SecurityContextConstraints().Create(scc)
			if err != nil {
				expectations.SCC.LowerExpectations(kvkey, 1, 0)
				return fmt.Errorf("unable to create SCC %+v: %v", scc, err)
			}
			log.Log.V(2).Infof("SCC %v created", scc.Name)
		} else if !objectMatchesVersion(&cachedSCC.ObjectMeta, version, imageRegistry, id, kv.GetGeneration()) {
			scc.ObjectMeta = *cachedSCC.ObjectMeta.DeepCopy()
			injectOperatorMetadata(kv, &scc.ObjectMeta, version, imageRegistry, id, true)
			_, err = sec.SecurityContextConstraints().Update(scc)
			if err != nil {
				return fmt.Errorf("Unable to update %s SecurityContextConstraints", scc.Name)
			}
			log.Log.V(2).Infof("SecurityContextConstraints %s updated", scc.Name)
		} else {
			log.Log.V(4).Infof("SCC %s is up to date", scc.Name)
		}

	}

	return nil
}

func syncPodDisruptionBudgetForDeployment(deployment *appsv1.Deployment, clientset kubecli.KubevirtClient, kv *v1.KubeVirt, expectations *util.Expectations, stores util.Stores) error {
	podDisruptionBudget := components.NewPodDisruptionBudgetForDeployment(deployment)

	imageTag := kv.Status.TargetKubeVirtVersion
	imageRegistry := kv.Status.TargetKubeVirtRegistry
	id := kv.Status.TargetDeploymentID

	injectOperatorMetadata(kv, &podDisruptionBudget.ObjectMeta, imageTag, imageRegistry, id, true)

	pdbClient := clientset.PolicyV1beta1().PodDisruptionBudgets(deployment.Namespace)

	var cachedPodDisruptionBudget *policyv1beta1.PodDisruptionBudget

	obj, exists, _ := stores.PodDisruptionBudgetCache.Get(podDisruptionBudget)
	if exists {
		cachedPodDisruptionBudget = obj.(*policyv1beta1.PodDisruptionBudget)
	}

	if !exists {
		kvkey, err := controller.KeyFunc(kv)
		if err != nil {
			return err
		}

		expectations.PodDisruptionBudget.RaiseExpectations(kvkey, 1, 0)
		_, err = pdbClient.Create(podDisruptionBudget)
		if err != nil {
			expectations.PodDisruptionBudget.LowerExpectations(kvkey, 1, 0)
			return fmt.Errorf("unable to create poddisruptionbudget %+v: %v", podDisruptionBudget, err)
		}
		log.Log.V(2).Infof("poddisruptionbudget %v created", podDisruptionBudget.GetName())
	} else if !objectMatchesVersion(&cachedPodDisruptionBudget.ObjectMeta, imageTag, imageRegistry, id, kv.GetGeneration()) {
		// Patch if old version
		var ops []string

		// Add Labels and Annotations Patches
		labelAnnotationPatch, err := createLabelsAndAnnotationsPatch(&podDisruptionBudget.ObjectMeta)
		if err != nil {
			return err
		}
		ops = append(ops, labelAnnotationPatch...)

		// Add Spec Patch
		newSpec, err := json.Marshal(podDisruptionBudget.Spec)
		if err != nil {
			return err
		}
		ops = append(ops, fmt.Sprintf(`{ "op": "replace", "path": "/spec", "value": %s }`, string(newSpec)))

		_, err = pdbClient.Patch(podDisruptionBudget.Name, types.JSONPatchType, generatePatchBytes(ops))
		if err != nil {
			return fmt.Errorf("unable to patch poddisruptionbudget %+v: %v", podDisruptionBudget, err)
		}
		log.Log.V(2).Infof("poddisruptionbudget %v patched", podDisruptionBudget.GetName())
	} else {
		log.Log.V(4).Infof("poddisruptionbudget %v is up-to-date", cachedPodDisruptionBudget.GetName())
	}

	return nil
}

func SyncAll(queue workqueue.RateLimitingInterface, kv *v1.KubeVirt, prevStrategy *InstallStrategy, targetStrategy *InstallStrategy, stores util.Stores, clientset kubecli.KubevirtClient, aggregatorclient APIServiceInterface, expectations *util.Expectations) (bool, error) {
	kvkey, err := controller.KeyFunc(kv)
	if err != nil {
		return false, err
	}

	gracePeriod := int64(0)
	deleteOptions := &metav1.DeleteOptions{
		GracePeriodSeconds: &gracePeriod,
	}

	// Avoid log spam by logging this issue once early instead of for once each object created
	if !isValidLabel(kv.Spec.ProductVersion) {
		log.Log.Errorf("invalid kubevirt.spec.productVersion: labels must be 63 characters or less, begin and end with alphanumeric characters, and contain only dot, hyphen or underscore")
	}
	if !isValidLabel(kv.Spec.ProductName) {
		log.Log.Errorf("invalid kubevirt.spec.productName: labels must be 63 characters or less, begin and end with alphanumeric characters, and contain only dot, hyphen or underscore")
	}

	targetVersion := kv.Status.TargetKubeVirtVersion
	targetImageRegistry := kv.Status.TargetKubeVirtRegistry
	observedVersion := kv.Status.ObservedKubeVirtVersion
	observedImageRegistry := kv.Status.ObservedKubeVirtRegistry

	apiDeploymentsRolledOver := haveApiDeploymentsRolledOver(targetStrategy, kv, stores)
	controllerDeploymentsRolledOver := haveControllerDeploymentsRolledOver(targetStrategy, kv, stores)
	daemonSetsRolledOver := haveDaemonSetsRolledOver(targetStrategy, kv, stores)

	infrastructureRolledOver := false
	if apiDeploymentsRolledOver && controllerDeploymentsRolledOver && daemonSetsRolledOver {

		// infrastructure has rolled over and is available
		infrastructureRolledOver = true
	} else if (targetVersion == observedVersion) && (targetImageRegistry == observedImageRegistry) {

		// infrastructure was observed to have rolled over successfully
		// in the past
		infrastructureRolledOver = true
	}

	ext := clientset.ExtensionsClient()

	takeUpdatePath := shouldTakeUpdatePath(kv.Status.TargetKubeVirtVersion, kv.Status.ObservedKubeVirtVersion)

	// -------- CREATE AND ROLE OUT UPDATED OBJECTS --------

	// creates a blocking webhook for any new CRDs that don't exist previously.
	// this webhook is removed once the new apiserver is online.
	if !apiDeploymentsRolledOver {
		err := createDummyWebhookValidator(targetStrategy, kv, clientset, stores, expectations)
		if err != nil {
			return false, err
		}
	} else {
		err := deleteDummyWebhookValidators(kv, clientset, stores, expectations)
		if err != nil {
			return false, err
		}
	}

	customize, err := NewCustomizer(kv.Spec.CustomizeComponents)
	if err != nil {
		return false, err
	}

	err = customize.GenericApplyPatches(targetStrategy.deployments)
	if err != nil {
		return false, err
	}
	err = customize.GenericApplyPatches(targetStrategy.services)
	if err != nil {
		return false, err
	}
	err = customize.GenericApplyPatches(targetStrategy.daemonSets)
	if err != nil {
		return false, err
	}
	err = customize.GenericApplyPatches(targetStrategy.validatingWebhookConfigurations)
	if err != nil {
		return false, err
	}
	err = customize.GenericApplyPatches(targetStrategy.mutatingWebhookConfigurations)
	if err != nil {
		return false, err
	}
	err = customize.GenericApplyPatches(targetStrategy.apiServices)
	if err != nil {
		return false, err
	}
	err = customize.GenericApplyPatches(targetStrategy.certificateSecrets)
	if err != nil {
		return false, err
	}

	// create/update CRDs
	err = createOrUpdateCrds(kv, targetStrategy, stores, clientset, expectations)
	if err != nil {
		return false, err
	}

	// create/update config maps
	err = createOrUpdateConfigMaps(kv, targetStrategy, stores, clientset, expectations)
	if err != nil {
		return false, err
	}

	// create/update serviceMonitor
	err = createOrUpdateServiceMonitors(kv, targetStrategy, stores, clientset, expectations)
	if err != nil {
		return false, err
	}

	// create/update PrometheusRules
	err = createOrUpdatePrometheusRules(kv, targetStrategy, stores, clientset, expectations)
	if err != nil {
		return false, err
	}

	// backup any old RBAC rules that don't match current version
	if !infrastructureRolledOver {
		err = backupRbac(kv,
			stores,
			clientset,
			expectations)
		if err != nil {
			return false, err
		}
	}

	// create/update all RBAC rules
	err = createOrUpdateRbac(kv,
		targetStrategy,
		stores,
		clientset,
		expectations)
	if err != nil {
		return false, err
	}

	// create/update SCCs
	err = createOrUpdateSCC(kv, targetStrategy, stores, clientset, expectations)
	if err != nil {
		return false, err
	}

	caDuration := &metav1.Duration{Duration: Duration7d}
	caOverlapTime := &metav1.Duration{Duration: Duration1d}
	certDuration := &metav1.Duration{Duration: Duration1d}
	if kv.Spec.CertificateRotationStrategy.SelfSigned != nil {
		if kv.Spec.CertificateRotationStrategy.SelfSigned.CARotateInterval != nil {
			caDuration = kv.Spec.CertificateRotationStrategy.SelfSigned.CARotateInterval
		}
		if kv.Spec.CertificateRotationStrategy.SelfSigned.CAOverlapInterval != nil {
			caOverlapTime = kv.Spec.CertificateRotationStrategy.SelfSigned.CAOverlapInterval
		}
		if kv.Spec.CertificateRotationStrategy.SelfSigned.CertRotateInterval != nil {
			certDuration = kv.Spec.CertificateRotationStrategy.SelfSigned.CertRotateInterval
		}
	}

	// create/update CA Certificate secret
	caCert, err := createOrUpdateCACertificateSecret(queue, kv, targetStrategy, stores, clientset, expectations, caDuration)
	if err != nil {
		return false, err
	}

	// create/update CA config map
	caBundle, err := createOrUpdateKubeVirtCAConfigMap(queue, kv, targetStrategy, stores, clientset, expectations, caCert, caOverlapTime)
	if err != nil {
		return false, err
	}

	// create/update Certificate secrets
	err = createOrUpdateCertificateSecrets(queue, kv, targetStrategy, stores, clientset, expectations, caCert, certDuration)
	if err != nil {
		return false, err
	}

	// create/update ValidatingWebhookConfiguration
	err = createOrUpdateValidatingWebhookConfigurations(kv, targetStrategy, stores, clientset, expectations, caBundle)
	if err != nil {
		return false, err
	}

	// create/update MutatingWebhookConfiguration
	err = createOrUpdateMutatingWebhookConfigurations(kv, targetStrategy, stores, clientset, expectations, caBundle)
	if err != nil {
		return false, err
	}

	// create/update APIServices
	err = createOrUpdateAPIServices(kv, targetStrategy, stores, aggregatorclient, expectations, caBundle)
	if err != nil {
		return false, err
	}

	// create/update Services
	pending, err := createOrUpdateService(kv,
		targetStrategy,
		stores,
		clientset,
		expectations)
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

	// Add new SCC Privileges and remove unsed SCC Privileges
	if infrastructureRolledOver {
		err := addOrRemoveSSC(targetStrategy, prevStrategy, clientset, stores, false)
		if err != nil {
			return false, err
		}

	} else {
		err := addOrRemoveSSC(targetStrategy, prevStrategy, clientset, stores, true)
		if err != nil {
			return false, err
		}
	}

	if takeUpdatePath {
		// UPDATE PATH IS
		// 1. daemonsets - ensures all compute nodes are updated to handle new features
		// 2. controllers - ensures controll plane is ready for new features
		// 3. wait for daemonsets and controllers to roll over
		// 4. apiserver - toggles on new features.

		// create/update Daemonsets
		for _, daemonSet := range targetStrategy.daemonSets {
			err := syncDaemonSet(kv, daemonSet, stores, clientset, expectations)
			if err != nil {
				return false, err
			}
		}

		// create/update Controller Deployments
		for _, deployment := range controllerDeployments(targetStrategy) {
			err := syncDeployment(kv, deployment, stores, clientset, expectations)
			if err != nil {
				return false, err
			}
			err = syncPodDisruptionBudgetForDeployment(deployment, clientset, kv, expectations, stores)
			if err != nil {
				return false, err
			}
		}

		// wait for daemonsets and controllers
		if !daemonSetsRolledOver || !controllerDeploymentsRolledOver {
			// not rolled out yet
			return false, nil
		}

		// create/update API Deployments
		for _, deployment := range apiDeployments(targetStrategy) {
			deployment := deployment.DeepCopy()
			err := syncDeployment(kv, deployment, stores, clientset, expectations)
			if err != nil {
				return false, err
			}
			err = syncPodDisruptionBudgetForDeployment(deployment, clientset, kv, expectations, stores)
			if err != nil {
				return false, err
			}
		}
	} else {
		// CREATE/ROLLBACK PATH IS
		// 1. apiserver - ensures validation of objects occur before allowing any control plane to act on them.
		// 2. wait for apiservers to roll over
		// 3. controllers and daemonsets

		// create/update API Deployments
		for _, deployment := range apiDeployments(targetStrategy) {
			deployment := deployment.DeepCopy()
			err := syncDeployment(kv, deployment, stores, clientset, expectations)
			if err != nil {
				return false, err
			}
			err = syncPodDisruptionBudgetForDeployment(deployment, clientset, kv, expectations, stores)
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
		for _, deployment := range controllerDeployments(targetStrategy) {
			err := syncDeployment(kv, deployment, stores, clientset, expectations)
			if err != nil {
				return false, err
			}
			err = syncPodDisruptionBudgetForDeployment(deployment, clientset, kv, expectations, stores)
			if err != nil {
				return false, err
			}
		}

		// create/update Daemonsets
		for _, daemonSet := range targetStrategy.daemonSets {
			err := syncDaemonSet(kv, daemonSet, stores, clientset, expectations)
			if err != nil {
				return false, err
			}
		}

	}

	err = syncKubevirtNamespaceLabels(kv, stores, clientset)
	if err != nil {
		return false, err
	}

	if !infrastructureRolledOver {
		// still waiting on roll out before cleaning up.
		return false, nil
	}

	// -------- ROLLOUT INCOMPATIBLE CHANGES WHICH REQUIRE A FULL CONTROLL PLANE ROLL OVER --------
	// some changes can only be done after the control plane rolled over
	err = rolloutNonCompatibleCRDChanges(kv, targetStrategy, stores, clientset, expectations)
	if err != nil {
		return false, err
	}

	// -------- CLEAN UP OLD UNUSED OBJECTS --------
	// outdated webhooks can potentially block deletes of other objects during the cleanup and need to be removed first

	// remove unused validating webhooks
	objects := stores.ValidationWebhookCache.List()
	for _, obj := range objects {
		if webhook, ok := obj.(*admissionregistrationv1beta1.ValidatingWebhookConfiguration); ok && webhook.DeletionTimestamp == nil {
			found := false
			if strings.HasPrefix(webhook.Name, "virt-operator-tmp-webhook") {
				continue
			}

			for _, targetWebhook := range targetStrategy.validatingWebhookConfigurations {

				if targetWebhook.Name == webhook.Name {
					found = true
					break
				}
			}
			// This is for backward compatibility where virt-api managed the entity itself.
			// If someone upgrades from such an old version and then has to roll back, we want avoid deleting a resource
			// which was not explicitly created by an operator, but is still visible to it.
			// TODO: Remove this once we don't support upgrading from such old kubevirt installations.
			if _, ok := webhook.Annotations[v1.InstallStrategyVersionAnnotation]; !ok {
				found = true
			}
			if !found {
				if key, err := controller.KeyFunc(webhook); err == nil {
					expectations.ValidationWebhook.AddExpectedDeletion(kvkey, key)
					err := clientset.AdmissionregistrationV1beta1().ValidatingWebhookConfigurations().Delete(webhook.Name, deleteOptions)
					if err != nil {
						expectations.ValidationWebhook.DeletionObserved(kvkey, key)
						log.Log.Errorf("Failed to delete webhook %+v: %v", webhook, err)
						return false, err
					}
				}
			}
		}
	}

	// remove unused mutating webhooks
	objects = stores.MutatingWebhookCache.List()
	for _, obj := range objects {
		if webhook, ok := obj.(*admissionregistrationv1beta1.MutatingWebhookConfiguration); ok && webhook.DeletionTimestamp == nil {
			found := false
			for _, targetWebhook := range targetStrategy.mutatingWebhookConfigurations {
				if targetWebhook.Name == webhook.Name {
					found = true
					break
				}
			}
			// This is for backward compatibility where virt-api managed the entity itself.
			// If someone upgrades from such an old version and then has to roll back, we want avoid deleting a resource
			// which was not explicitly created by an operator, but is still visible to it.
			// TODO: Remove this once we don't support upgrading from such old kubevirt installations.
			if _, ok := webhook.Annotations[v1.InstallStrategyVersionAnnotation]; !ok {
				found = true
			}
			if !found {
				if key, err := controller.KeyFunc(webhook); err == nil {
					expectations.MutatingWebhook.AddExpectedDeletion(kvkey, key)
					err := clientset.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Delete(webhook.Name, deleteOptions)
					if err != nil {
						expectations.MutatingWebhook.DeletionObserved(kvkey, key)
						log.Log.Errorf("Failed to delete webhook %+v: %v", webhook, err)
						return false, err
					}
				}
			}
		}
	}

	// remove unused APIServices
	objects = stores.APIServiceCache.List()
	for _, obj := range objects {
		if apiService, ok := obj.(*v1beta1.APIService); ok && apiService.DeletionTimestamp == nil {
			found := false
			for _, targetAPIService := range targetStrategy.apiServices {
				if targetAPIService.Name == apiService.Name {
					found = true
					break
				}
			}
			// This is for backward compatibility where virt-api managed the entity itself.
			// If someone upgrades from such an old version and then has to roll back, we want avoid deleting a resource
			// which was not explicitly created by an operator, but is still visible to it.
			// TODO: Remove this once we don't support upgrading from such old kubevirt installations.
			if _, ok := apiService.Annotations[v1.InstallStrategyVersionAnnotation]; !ok {
				found = true
			}
			if !found {
				if key, err := controller.KeyFunc(apiService); err == nil {
					expectations.APIService.AddExpectedDeletion(kvkey, key)
					err := aggregatorclient.Delete(apiService.Name, deleteOptions)
					if err != nil {
						expectations.APIService.DeletionObserved(kvkey, key)
						log.Log.Errorf("Failed to delete apiService %+v: %v", apiService, err)
						return false, err
					}
				}
			}
		}
	}

	// remove unused Secrets
	objects = stores.SecretCache.List()
	for _, obj := range objects {
		if secret, ok := obj.(*corev1.Secret); ok && secret.DeletionTimestamp == nil {
			found := false
			for _, targetSecret := range targetStrategy.certificateSecrets {
				if targetSecret.Name == secret.Name {
					found = true
					break
				}
			}
			// This is for backward compatibility where virt-api managed the entity itself.
			// If someone upgrades from such an old version and then has to roll back, we want avoid deleting a resource
			// which was not explicitly created by an operator, but is still visible to it.
			// TODO: Remove this once we don't support upgrading from such old kubevirt installations.
			if _, ok := secret.Annotations[v1.InstallStrategyVersionAnnotation]; !ok {
				found = true
			}
			if !found {
				if key, err := controller.KeyFunc(secret); err == nil {
					expectations.Secrets.AddExpectedDeletion(kvkey, key)
					err := clientset.CoreV1().Secrets(secret.Namespace).Delete(secret.Name, deleteOptions)
					if err != nil {
						expectations.Secrets.DeletionObserved(kvkey, key)
						log.Log.Errorf("Failed to delete secret %+v: %v", secret, err)
						return false, err
					}
				}
			}
		}
	}

	// remove unused ConfigMaps
	objects = stores.ConfigMapCache.List()
	for _, obj := range objects {
		if configMap, ok := obj.(*corev1.ConfigMap); ok && configMap.DeletionTimestamp == nil {
			found := false
			for _, targetConfigMap := range targetStrategy.configMaps {
				if targetConfigMap.Name == configMap.Name {
					found = true
					break
				}
			}
			if !found {
				if key, err := controller.KeyFunc(configMap); err == nil {
					expectations.ConfigMap.AddExpectedDeletion(kvkey, key)
					err := clientset.CoreV1().ConfigMaps(configMap.Namespace).Delete(configMap.Name, deleteOptions)
					if err != nil {
						expectations.ConfigMap.DeletionObserved(kvkey, key)
						log.Log.Errorf("Failed to delete configmap %+v: %v", configMap, err)
						return false, err
					}
				}
			}
		}
	}

	// remove unused crds
	objects = stores.CrdCache.List()
	for _, obj := range objects {
		if crd, ok := obj.(*extv1beta1.CustomResourceDefinition); ok && crd.DeletionTimestamp == nil {
			found := false
			for _, targetCrd := range targetStrategy.crds {
				if targetCrd.Name == crd.Name {
					found = true
					break
				}
			}
			if !found {
				if key, err := controller.KeyFunc(crd); err == nil {
					expectations.Crd.AddExpectedDeletion(kvkey, key)
					err := ext.ApiextensionsV1beta1().CustomResourceDefinitions().Delete(crd.Name, deleteOptions)
					if err != nil {
						expectations.Crd.DeletionObserved(kvkey, key)
						log.Log.Errorf("Failed to delete crd %+v: %v", crd, err)
						return false, err
					}
				}
			}
		}
	}

	// remove unused daemonsets
	objects = stores.DaemonSetCache.List()
	for _, obj := range objects {
		if ds, ok := obj.(*appsv1.DaemonSet); ok && ds.DeletionTimestamp == nil {
			found := false
			for _, targetDs := range targetStrategy.daemonSets {
				if targetDs.Name == ds.Name && targetDs.Namespace == ds.Namespace {
					found = true
					break
				}
			}
			if !found {
				if key, err := controller.KeyFunc(ds); err == nil {
					expectations.DaemonSet.AddExpectedDeletion(kvkey, key)
					err := clientset.AppsV1().DaemonSets(ds.Namespace).Delete(ds.Name, deleteOptions)
					if err != nil {
						expectations.DaemonSet.DeletionObserved(kvkey, key)
						log.Log.Errorf("Failed to delete daemonset: %v", err)
						return false, err
					}
				}
			}
		}
	}

	// remove unused deployments
	objects = stores.DeploymentCache.List()
	for _, obj := range objects {
		if deployment, ok := obj.(*appsv1.Deployment); ok && deployment.DeletionTimestamp == nil {
			found := false
			for _, targetDeployment := range targetStrategy.deployments {
				if targetDeployment.Name == deployment.Name && targetDeployment.Namespace == deployment.Namespace {
					found = true
					break
				}
			}
			if !found {
				if key, err := controller.KeyFunc(deployment); err == nil {
					expectations.Deployment.AddExpectedDeletion(kvkey, key)
					err := clientset.AppsV1().Deployments(deployment.Namespace).Delete(deployment.Name, deleteOptions)
					if err != nil {
						expectations.Deployment.DeletionObserved(kvkey, key)
						log.Log.Errorf("Failed to delete deployment: %v", err)
						return false, err
					}
				}
			}
		}
	}

	// remove unused services
	objects = stores.ServiceCache.List()
	for _, obj := range objects {
		if svc, ok := obj.(*corev1.Service); ok && svc.DeletionTimestamp == nil {
			found := false
			for _, targetSvc := range targetStrategy.services {
				if targetSvc.Name == svc.Name && targetSvc.Namespace == svc.Namespace {
					found = true
					break
				}
			}
			if !found {
				if key, err := controller.KeyFunc(svc); err == nil {
					expectations.Service.AddExpectedDeletion(kvkey, key)
					err := clientset.CoreV1().Services(svc.Namespace).Delete(svc.Name, deleteOptions)
					if err != nil {
						expectations.Service.DeletionObserved(kvkey, key)
						log.Log.Errorf("Failed to delete service %+v: %v", svc, err)
						return false, err
					}
				}
			}
		}
	}

	// remove unused clusterrolebindings
	objects = stores.ClusterRoleBindingCache.List()
	for _, obj := range objects {
		if crb, ok := obj.(*rbacv1.ClusterRoleBinding); ok && crb.DeletionTimestamp == nil {
			found := false
			for _, targetCrb := range targetStrategy.clusterRoleBindings {
				if targetCrb.Name == crb.Name && targetCrb.Namespace == crb.Namespace {
					found = true
					break
				}
			}
			if !found {
				if key, err := controller.KeyFunc(crb); err == nil {
					expectations.ClusterRoleBinding.AddExpectedDeletion(kvkey, key)
					err := clientset.RbacV1().ClusterRoleBindings().Delete(crb.Name, deleteOptions)
					if err != nil {
						expectations.ClusterRoleBinding.DeletionObserved(kvkey, key)
						log.Log.Errorf("Failed to delete crb %+v: %v", crb, err)
						return false, err
					}
				}
			}
		}
	}

	// remove unused clusterroles
	objects = stores.ClusterRoleCache.List()
	for _, obj := range objects {
		if cr, ok := obj.(*rbacv1.ClusterRole); ok && cr.DeletionTimestamp == nil {
			found := false
			for _, targetCr := range targetStrategy.clusterRoles {
				if targetCr.Name == cr.Name && targetCr.Namespace == cr.Namespace {
					found = true
					break
				}
			}
			if !found {
				if key, err := controller.KeyFunc(cr); err == nil {
					expectations.ClusterRole.AddExpectedDeletion(kvkey, key)
					err := clientset.RbacV1().ClusterRoles().Delete(cr.Name, deleteOptions)
					if err != nil {
						expectations.ClusterRole.DeletionObserved(kvkey, key)
						log.Log.Errorf("Failed to delete cr %+v: %v", cr, err)
						return false, err
					}
				}
			}
		}
	}

	// remove unused rolebindings
	objects = stores.RoleBindingCache.List()
	for _, obj := range objects {
		if rb, ok := obj.(*rbacv1.RoleBinding); ok && rb.DeletionTimestamp == nil {
			found := false
			for _, targetRb := range targetStrategy.roleBindings {
				if targetRb.Name == rb.Name && targetRb.Namespace == rb.Namespace {
					found = true
					break
				}
			}
			if !found {
				if key, err := controller.KeyFunc(rb); err == nil {
					expectations.RoleBinding.AddExpectedDeletion(kvkey, key)
					err := clientset.RbacV1().RoleBindings(kv.Namespace).Delete(rb.Name, deleteOptions)
					if err != nil {
						expectations.RoleBinding.DeletionObserved(kvkey, key)
						log.Log.Errorf("Failed to delete rb %+v: %v", rb, err)
						return false, err
					}
				}
			}
		}
	}

	// remove unused roles
	objects = stores.RoleCache.List()
	for _, obj := range objects {
		if role, ok := obj.(*rbacv1.Role); ok && role.DeletionTimestamp == nil {
			found := false
			for _, targetR := range targetStrategy.roles {
				if targetR.Name == role.Name && targetR.Namespace == role.Namespace {
					found = true
					break
				}
			}
			if !found {
				if key, err := controller.KeyFunc(role); err == nil {
					expectations.Role.AddExpectedDeletion(kvkey, key)
					err := clientset.RbacV1().Roles(kv.Namespace).Delete(role.Name, deleteOptions)
					if err != nil {
						expectations.Role.DeletionObserved(kvkey, key)
						log.Log.Errorf("Failed to delete role %+v: %v", role, err)
						return false, err
					}
				}
			}
		}
	}

	// remove unused serviceaccounts
	objects = stores.ServiceAccountCache.List()
	for _, obj := range objects {
		if sa, ok := obj.(*corev1.ServiceAccount); ok && sa.DeletionTimestamp == nil {
			found := false
			for _, targetSa := range targetStrategy.serviceAccounts {
				if targetSa.Name == sa.Name && targetSa.Namespace == sa.Namespace {
					found = true
					break
				}
			}
			if !found {
				if key, err := controller.KeyFunc(sa); err == nil {
					expectations.ServiceAccount.AddExpectedDeletion(kvkey, key)
					err := clientset.CoreV1().ServiceAccounts(kv.Namespace).Delete(sa.Name, deleteOptions)
					if err != nil {
						expectations.ServiceAccount.DeletionObserved(kvkey, key)
						log.Log.Errorf("Failed to delete serviceaccount %+v: %v", sa, err)
						return false, err
					}
				}
			}
		}
	}

	// remove unused sccs
	objects = stores.SCCCache.List()
	for _, obj := range objects {
		if scc, ok := obj.(*secv1.SecurityContextConstraints); ok && scc.DeletionTimestamp == nil {

			// informer watches all SCC objects, it cannot be changed because of kubevirt updates
			if !util.IsManagedByOperator(scc.GetLabels()) {
				continue
			}

			found := false
			for _, targetScc := range targetStrategy.sccs {
				if targetScc.Name == scc.Name {
					found = true
					break
				}
			}
			if !found {
				if key, err := controller.KeyFunc(scc); err == nil {
					expectations.SCC.AddExpectedDeletion(kvkey, key)
					err := clientset.SecClient().SecurityContextConstraints().Delete(scc.Name, deleteOptions)
					if err != nil {
						expectations.SCC.DeletionObserved(kvkey, key)
						log.Log.Errorf("Failed to delete SecurityContextConstraints %+v: %v", scc, err)
						return false, err
					}
				}
			}
		}
	}

	// remove unused prometheus rules
	objects = stores.PrometheusRuleCache.List()
	for _, obj := range objects {
		if cachePromRule, ok := obj.(*promv1.PrometheusRule); ok && cachePromRule.DeletionTimestamp == nil {
			found := false
			for _, targetPromRule := range targetStrategy.prometheusRules {
				if targetPromRule.Name == cachePromRule.Name && targetPromRule.Namespace == cachePromRule.Namespace {
					found = true
					break
				}
			}
			if !found {
				if key, err := controller.KeyFunc(cachePromRule); err == nil {
					expectations.PrometheusRule.AddExpectedDeletion(kvkey, key)
					err := clientset.PrometheusClient().
						MonitoringV1().
						PrometheusRules(kv.Namespace).
						Delete(cachePromRule.Name, deleteOptions)
					if err != nil {
						expectations.PrometheusRule.DeletionObserved(kvkey, key)
						log.Log.Errorf("Failed to delete prometheusrule %+v: %v", cachePromRule, err)
						return false, err
					}
				}
			}
		}
	}

	return true, nil
}

func createOrUpdateKubeVirtCAConfigMap(
	queue workqueue.RateLimitingInterface,
	kv *v1.KubeVirt,
	strategy *InstallStrategy,
	stores util.Stores,
	clientset kubecli.KubevirtClient,
	expectations *util.Expectations,
	caCert *tls.Certificate,
	overlapInterval *metav1.Duration,
) (caBundle []byte, err error) {

	for _, configMap := range strategy.configMaps {

		if configMap.Name != components.KubeVirtCASecretName {
			continue
		}
		var cachedConfigMap *corev1.ConfigMap
		configMap = configMap.DeepCopy()

		log.DefaultLogger().V(4).Infof("checking ca config map %v", configMap.Name)

		version := kv.Status.TargetKubeVirtVersion
		imageRegistry := kv.Status.TargetKubeVirtRegistry
		id := kv.Status.TargetDeploymentID

		kvkey, err := controller.KeyFunc(kv)
		if err != nil {
			return nil, err
		}

		obj, exists, _ := stores.ConfigMapCache.Get(configMap)

		updateBundle := false
		if exists {
			cachedConfigMap = obj.(*corev1.ConfigMap)

			bundle, certCount, err := components.MergeCABundle(caCert, []byte(cachedConfigMap.Data[components.CABundleKey]), overlapInterval.Duration)
			if err != nil {
				return nil, err
			}

			// ensure that we remove the old CA after the overlap period
			if certCount > 1 {
				queue.AddAfter(kvkey, overlapInterval.Duration)
			}

			configMap.Data = map[string]string{components.CABundleKey: string(bundle)}

			if !reflect.DeepEqual(configMap.Data, cachedConfigMap.Data) {
				updateBundle = true
			}
		} else {
			configMap.Data = map[string]string{components.CABundleKey: string(cert.EncodeCertPEM(caCert.Leaf))}
		}

		injectOperatorMetadata(kv, &configMap.ObjectMeta, version, imageRegistry, id, true)
		if !exists {
			expectations.ConfigMap.RaiseExpectations(kvkey, 1, 0)
			_, err := clientset.CoreV1().ConfigMaps(configMap.Namespace).Create(configMap)
			if err != nil {
				expectations.ConfigMap.LowerExpectations(kvkey, 1, 0)
				return nil, fmt.Errorf("unable to create configMap %+v: %v", configMap, err)
			}
		} else {
			if !objectMatchesVersion(&cachedConfigMap.ObjectMeta, version, imageRegistry, id, kv.GetGeneration()) || updateBundle {
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

				_, err = clientset.CoreV1().ConfigMaps(configMap.Namespace).Patch(configMap.Name, types.JSONPatchType, generatePatchBytes(ops))
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
