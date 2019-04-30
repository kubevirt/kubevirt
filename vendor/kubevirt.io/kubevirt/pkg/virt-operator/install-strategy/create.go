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

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	secv1 "github.com/openshift/api/security/v1"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	extv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/cert/triple"

	"github.com/blang/semver"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/virt-operator/util"
)

func objectMatchesVersion(objectMeta *metav1.ObjectMeta, imageTag string, imageRegistry string) bool {

	if objectMeta.Annotations == nil {
		return false
	}

	foundImageTag := objectMeta.Annotations[v1.InstallStrategyVersionAnnotation]
	foundImageRegistry := objectMeta.Annotations[v1.InstallStrategyRegistryAnnotation]

	if foundImageTag == imageTag && foundImageRegistry == imageRegistry {
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

func injectOperatorMetadata(kv *v1.KubeVirt, objectMeta *metav1.ObjectMeta, imageTag string, imageRegistry string) {
	if objectMeta.Labels == nil {
		objectMeta.Labels = make(map[string]string)
	}
	objectMeta.Labels[v1.ManagedByLabel] = v1.ManagedByLabelOperatorValue

	if objectMeta.Annotations == nil {
		objectMeta.Annotations = make(map[string]string)
	}
	objectMeta.Annotations[v1.InstallStrategyVersionAnnotation] = imageTag
	objectMeta.Annotations[v1.InstallStrategyRegistryAnnotation] = imageRegistry

	objectMeta.OwnerReferences = []metav1.OwnerReference{*metav1.NewControllerRef(kv, v1.KubeVirtGroupVersionKind)}

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
	ops = append(ops, fmt.Sprintf(`{ "op": "add", "path": "/metadata/labels", "value": %s }`, string(labelBytes)))
	ops = append(ops, fmt.Sprintf(`{ "op": "add", "path": "/metadata/annotations", "value": %s }`, string(annotationBytes)))

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

	injectOperatorMetadata(kv, &daemonSet.ObjectMeta, imageTag, imageRegistry)
	injectOperatorMetadata(kv, &daemonSet.Spec.Template.ObjectMeta, imageTag, imageRegistry)

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
	} else if !objectMatchesVersion(&cachedDaemonSet.ObjectMeta, imageTag, imageRegistry) {
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

	injectOperatorMetadata(kv, &deployment.ObjectMeta, imageTag, imageRegistry)
	injectOperatorMetadata(kv, &deployment.Spec.Template.ObjectMeta, imageTag, imageRegistry)

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
	} else if !objectMatchesVersion(&cachedDeployment.ObjectMeta, imageTag, imageRegistry) {
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

func verifyCRDUpdatePath(targetStrategy *InstallStrategy, stores util.Stores, kv *v1.KubeVirt) error {
	imageTag := kv.Status.TargetKubeVirtVersion
	imageRegistry := kv.Status.TargetKubeVirtRegistry

	for _, crd := range targetStrategy.crds {
		var cachedCrd *extv1beta1.CustomResourceDefinition

		obj, exists, _ := stores.CrdCache.Get(crd)
		if exists {
			cachedCrd = obj.(*extv1beta1.CustomResourceDefinition)
		}

		if !exists {
			// we can always update to a "new" crd that doesn't exist yet.
			continue
		} else if objectMatchesVersion(&cachedCrd.ObjectMeta, imageTag, imageRegistry) {
			// already up-to-date
			continue
		}

		if crd.Spec.Version != cachedCrd.Spec.Version {
			// We don't currently allow transitioning between versions until
			// the conversion webhook is supported.
			return fmt.Errorf("No supported update path from crd %s version %s to version %s", crd.Name, cachedCrd.Spec.Version, crd.Spec.Version)
		}
	}

	return nil
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

	var webhooks []admissionregistrationv1beta1.Webhook

	kvkey, err := controller.KeyFunc(kv)
	if err != nil {
		return err
	}
	imageTag := kv.Status.TargetKubeVirtVersion
	imageRegistry := kv.Status.TargetKubeVirtRegistry

	// If webhook already exists in cache, then exit.
	objects := stores.ValidationWebhookCache.List()
	for _, obj := range objects {
		if webhook, ok := obj.(*admissionregistrationv1beta1.ValidatingWebhookConfiguration); ok {

			if objectMatchesVersion(&webhook.ObjectMeta, imageTag, imageRegistry) {
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
		webhooks = append(webhooks, admissionregistrationv1beta1.Webhook{
			Name:          fmt.Sprintf("%s-tmp-validator", crd.Name),
			FailurePolicy: &failurePolicy,
			Rules: []admissionregistrationv1beta1.RuleWithOperations{{
				Operations: []admissionregistrationv1beta1.OperationType{
					admissionregistrationv1beta1.Create,
				},
				Rule: admissionregistrationv1beta1.Rule{
					APIGroups:   []string{crd.Spec.Group},
					APIVersions: []string{crd.Spec.Version},
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
	caKeyPair, _ := triple.NewCA("fake.kubevirt.io")
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
	injectOperatorMetadata(kv, &validationWebhook.ObjectMeta, imageTag, imageRegistry)

	expectations.ValidationWebhook.RaiseExpectations(kvkey, 1, 0)
	_, err = clientset.AdmissionregistrationV1beta1().ValidatingWebhookConfigurations().Create(validationWebhook)
	if err != nil {
		expectations.ValidationWebhook.LowerExpectations(kvkey, 1, 0)
		return fmt.Errorf("unable to create validation webhook: %v", err)
	}
	log.Log.V(2).Infof("Validation webhook created for image %s and registry %s", imageTag, imageRegistry)

	return nil
}

func createOrUpdateRoleBinding(rb *rbacv1.RoleBinding,
	kv *v1.KubeVirt,
	imageTag string,
	imageRegistry string,
	namespace string,
	stores util.Stores,
	clientset kubecli.KubevirtClient,
	expectations *util.Expectations) error {

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

	injectOperatorMetadata(kv, &rb.ObjectMeta, imageTag, imageRegistry)
	if !exists {
		// Create non existent
		expectations.RoleBinding.RaiseExpectations(kvkey, 1, 0)
		_, err := rbac.RoleBindings(namespace).Create(rb)
		if err != nil {
			expectations.RoleBinding.LowerExpectations(kvkey, 1, 0)
			return fmt.Errorf("unable to create rolebinding %+v: %v", rb, err)
		}
		log.Log.V(2).Infof("rolebinding %v created", rb.GetName())
	} else if !objectMatchesVersion(&cachedRb.ObjectMeta, imageTag, imageRegistry) {
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
	namespace string,
	stores util.Stores,
	clientset kubecli.KubevirtClient,
	expectations *util.Expectations) error {

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

	injectOperatorMetadata(kv, &r.ObjectMeta, imageTag, imageRegistry)
	if !exists {
		// Create non existent
		expectations.Role.RaiseExpectations(kvkey, 1, 0)
		_, err := rbac.Roles(namespace).Create(r)
		if err != nil {
			expectations.Role.LowerExpectations(kvkey, 1, 0)
			return fmt.Errorf("unable to create role %+v: %v", r, err)
		}
		log.Log.V(2).Infof("role %v created", r.GetName())
	} else if !objectMatchesVersion(&cachedR.ObjectMeta, imageTag, imageRegistry) {
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

	injectOperatorMetadata(kv, &crb.ObjectMeta, imageTag, imageRegistry)
	if !exists {
		// Create non existent
		expectations.ClusterRoleBinding.RaiseExpectations(kvkey, 1, 0)
		_, err := rbac.ClusterRoleBindings().Create(crb)
		if err != nil {
			expectations.ClusterRoleBinding.LowerExpectations(kvkey, 1, 0)
			return fmt.Errorf("unable to create clusterrolebinding %+v: %v", crb, err)
		}
		log.Log.V(2).Infof("clusterrolebinding %v created", crb.GetName())
	} else if !objectMatchesVersion(&cachedCrb.ObjectMeta, imageTag, imageRegistry) {
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

	injectOperatorMetadata(kv, &cr.ObjectMeta, imageTag, imageRegistry)
	if !exists {
		// Create non existent
		expectations.ClusterRole.RaiseExpectations(kvkey, 1, 0)
		_, err := rbac.ClusterRoles().Create(cr)
		if err != nil {
			expectations.ClusterRole.LowerExpectations(kvkey, 1, 0)
			return fmt.Errorf("unable to create clusterrole %+v: %v", cr, err)
		}
		log.Log.V(2).Infof("clusterrole %v created", cr.GetName())
	} else if !objectMatchesVersion(&cachedCr.ObjectMeta, imageTag, imageRegistry) {
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

	imageTag := kv.Status.TargetKubeVirtVersion
	imageRegistry := kv.Status.TargetKubeVirtRegistry

	for _, crd := range targetStrategy.crds {
		var cachedCrd *extv1beta1.CustomResourceDefinition

		crd := crd.DeepCopy()
		obj, exists, _ := stores.CrdCache.Get(crd)
		if exists {
			cachedCrd = obj.(*extv1beta1.CustomResourceDefinition)
		}

		injectOperatorMetadata(kv, &crd.ObjectMeta, imageTag, imageRegistry)
		if !exists {
			// Create non existent
			expectations.Crd.RaiseExpectations(kvkey, 1, 0)
			_, err := ext.ApiextensionsV1beta1().CustomResourceDefinitions().Create(crd)
			if err != nil {
				expectations.Crd.LowerExpectations(kvkey, 1, 0)
				return fmt.Errorf("unable to create crd %+v: %v", crd, err)
			}
			log.Log.V(2).Infof("crd %v created", crd.GetName())

		} else if !objectMatchesVersion(&cachedCrd.ObjectMeta, imageTag, imageRegistry) {
			// Patch if old version
			var ops []string

			// Add Labels and Annotations Patches
			labelAnnotationPatch, err := createLabelsAndAnnotationsPatch(&crd.ObjectMeta)
			if err != nil {
				return err
			}
			ops = append(ops, labelAnnotationPatch...)

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

func shouldBackupRBACObject(kv *v1.KubeVirt, objectMeta *metav1.ObjectMeta) (bool, string, string) {
	curImageTag := kv.Status.TargetKubeVirtVersion
	curImageRegistry := kv.Status.TargetKubeVirtRegistry

	if objectMatchesVersion(objectMeta, curImageTag, curImageRegistry) {
		// matches current target version already, so doesn't need backup
		return false, "", ""
	}

	if objectMeta.Annotations == nil {
		return false, "", ""
	}

	_, ok := objectMeta.Annotations[v1.EphemeralBackupObject]
	if ok {
		// ephemeral backup objects don't need to be backed up because
		// they are the backup
		return false, "", ""
	}

	imageTag, ok := objectMeta.Annotations[v1.InstallStrategyVersionAnnotation]
	if !ok {
		return false, "", ""
	}

	imageRegistry, ok := objectMeta.Annotations[v1.InstallStrategyRegistryAnnotation]
	if !ok {
		return false, "", ""
	}

	return true, imageTag, imageRegistry

}

func needsClusterRoleBindingBackup(kv *v1.KubeVirt, stores util.Stores, crb *rbacv1.ClusterRoleBinding) bool {

	backup, imageTag, imageRegistry := shouldBackupRBACObject(kv, &crb.ObjectMeta)
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

		if uid == string(crb.UID) && objectMatchesVersion(&cachedCrb.ObjectMeta, imageTag, imageRegistry) {
			// found backup. UID matches and versions match
			// note, it's possible for a single UID to have multiple backups with
			// different versions
			return false
		}
	}

	return true
}

func needsClusterRoleBackup(kv *v1.KubeVirt, stores util.Stores, cr *rbacv1.ClusterRole) bool {
	backup, imageTag, imageRegistry := shouldBackupRBACObject(kv, &cr.ObjectMeta)
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

		if uid == string(cr.UID) && objectMatchesVersion(&cachedCr.ObjectMeta, imageTag, imageRegistry) {
			// found backup. UID matches and versions match
			// note, it's possible for a single UID to have multiple backups with
			// different versions
			return false
		}
	}

	return true
}

func needsRoleBindingBackup(kv *v1.KubeVirt, stores util.Stores, rb *rbacv1.RoleBinding) bool {

	backup, imageTag, imageRegistry := shouldBackupRBACObject(kv, &rb.ObjectMeta)
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

		if uid == string(rb.UID) && objectMatchesVersion(&cachedRb.ObjectMeta, imageTag, imageRegistry) {
			// found backup. UID matches and versions match
			// note, it's possible for a single UID to have multiple backups with
			// different versions
			return false
		}
	}

	return true
}

func needsRoleBackup(kv *v1.KubeVirt, stores util.Stores, r *rbacv1.Role) bool {

	backup, imageTag, imageRegistry := shouldBackupRBACObject(kv, &r.ObjectMeta)
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

		if uid == string(r.UID) && objectMatchesVersion(&cachedR.ObjectMeta, imageTag, imageRegistry) {
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

		// needs backup, so create a new object that will temporarily
		// backup this object while the update is in progress.
		cr := cachedCr.DeepCopy()
		cr.ObjectMeta = metav1.ObjectMeta{
			GenerateName: cachedCr.Name,
		}
		injectOperatorMetadata(kv, &cr.ObjectMeta, imageTag, imageRegistry)
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

		// needs backup, so create a new object that will temporarily
		// backup this object while the update is in progress.
		crb := cachedCrb.DeepCopy()
		crb.ObjectMeta = metav1.ObjectMeta{
			GenerateName: cachedCrb.Name,
		}
		injectOperatorMetadata(kv, &crb.ObjectMeta, imageTag, imageRegistry)
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

		// needs backup, so create a new object that will temporarily
		// backup this object while the update is in progress.
		r := cachedCr.DeepCopy()
		r.ObjectMeta = metav1.ObjectMeta{
			GenerateName: cachedCr.Name,
		}
		injectOperatorMetadata(kv, &r.ObjectMeta, imageTag, imageRegistry)
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

		// needs backup, so create a new object that will temporarily
		// backup this object while the update is in progress.
		rb := cachedRb.DeepCopy()
		rb.ObjectMeta = metav1.ObjectMeta{
			GenerateName: cachedRb.Name,
		}
		injectOperatorMetadata(kv, &rb.ObjectMeta, imageTag, imageRegistry)
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

	imageTag := kv.Status.TargetKubeVirtVersion
	imageRegistry := kv.Status.TargetKubeVirtRegistry

	// create/update ServiceAccounts
	for _, sa := range targetStrategy.serviceAccounts {
		var cachedSa *corev1.ServiceAccount

		sa := sa.DeepCopy()
		obj, exists, _ := stores.ServiceAccountCache.Get(sa)
		if exists {
			cachedSa = obj.(*corev1.ServiceAccount)
		}

		injectOperatorMetadata(kv, &sa.ObjectMeta, imageTag, imageRegistry)
		if !exists {
			// Create non existent
			expectations.ServiceAccount.RaiseExpectations(kvkey, 1, 0)
			_, err := core.ServiceAccounts(kv.Namespace).Create(sa)
			if err != nil {
				expectations.ServiceAccount.LowerExpectations(kvkey, 1, 0)
				return fmt.Errorf("unable to create serviceaccount %+v: %v", sa, err)
			}
			log.Log.V(2).Infof("serviceaccount %v created", sa.GetName())
		} else if !objectMatchesVersion(&cachedSa.ObjectMeta, imageTag, imageRegistry) {
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
			log.Log.V(2).Infof("serviceaccount %v already exists and is up-to-date", sa.GetName())
		}
	}

	// create/update ClusterRoles
	for _, cr := range targetStrategy.clusterRoles {
		err := createOrUpdateClusterRole(cr,
			kv,
			imageTag,
			imageRegistry,
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
			imageTag,
			imageRegistry,
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
			imageTag,
			imageRegistry,
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
			imageTag,
			imageRegistry,
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

	imageTag := kv.Status.TargetKubeVirtVersion
	imageRegistry := kv.Status.TargetKubeVirtRegistry

	// First check if there's anything to do.
	if objectMatchesVersion(&cachedService.ObjectMeta, imageTag, imageRegistry) {
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

func createOrUpdateService(kv *v1.KubeVirt,
	targetStrategy *InstallStrategy,
	stores util.Stores,
	clientset kubecli.KubevirtClient,
	expectations *util.Expectations) (bool, error) {

	core := clientset.CoreV1()
	imageTag := kv.Status.TargetKubeVirtVersion
	imageRegistry := kv.Status.TargetKubeVirtRegistry

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

		injectOperatorMetadata(kv, &service.ObjectMeta, imageTag, imageRegistry)
		if !exists {
			expectations.Service.RaiseExpectations(kvkey, 1, 0)
			_, err := core.Services(kv.Namespace).Create(service)
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
						err := core.Services(kv.Namespace).Delete(cachedService.Name, deleteOptions)
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
				_, err = core.Services(kv.Namespace).Patch(service.Name, types.JSONPatchType, generatePatchBytes(patchOps))
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
		privSCCCopy := privSCC.DeepCopy()

		modified := false
		users := privSCCCopy.Users

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
			userBytes, err := json.Marshal(users)
			if err != nil {
				return err
			}

			data := []byte(fmt.Sprintf(`{"users": %s}`, userBytes))
			_, err = scc.SecurityContextConstraints().Patch(sccPriv.TargetSCC, types.StrategicMergePatchType, data)
			if err != nil {
				return fmt.Errorf("unable to patch scc: %v", err)
			}
		}
	}
	return nil
}

func SyncAll(kv *v1.KubeVirt,
	prevStrategy *InstallStrategy,
	targetStrategy *InstallStrategy,
	stores util.Stores,
	clientset kubecli.KubevirtClient,
	expectations *util.Expectations) (bool, error) {

	kvkey, err := controller.KeyFunc(kv)
	if err != nil {
		return false, err
	}

	gracePeriod := int64(0)
	deleteOptions := &metav1.DeleteOptions{
		GracePeriodSeconds: &gracePeriod,
	}

	targetImageTag := kv.Status.TargetKubeVirtVersion
	targetImageRegistry := kv.Status.TargetKubeVirtRegistry
	observedImageTag := kv.Status.ObservedKubeVirtVersion
	observedImageRegistry := kv.Status.ObservedKubeVirtRegistry

	apiDeploymentsRolledOver := haveApiDeploymentsRolledOver(targetStrategy, kv, stores)
	controllerDeploymentsRolledOver := haveControllerDeploymentsRolledOver(targetStrategy, kv, stores)
	daemonSetsRolledOver := haveDaemonSetsRolledOver(targetStrategy, kv, stores)

	infrastructureRolledOver := false
	if apiDeploymentsRolledOver && controllerDeploymentsRolledOver && daemonSetsRolledOver {

		// infrastructure has rolled over and is available
		infrastructureRolledOver = true
	} else if (targetImageTag == observedImageTag) && (targetImageRegistry == observedImageRegistry) {

		// infrastructure was observed to have rolled over successfully
		// in the past
		infrastructureRolledOver = true
	}

	ext := clientset.ExtensionsClient()

	takeUpdatePath := shouldTakeUpdatePath(kv.Status.TargetKubeVirtVersion, kv.Status.ObservedKubeVirtVersion)

	// -------- CREATE AND ROLE OUT UPDATED OBJECTS --------

	// create/update CRDs

	// Verify we can transition to the target API version
	err = verifyCRDUpdatePath(targetStrategy, stores, kv)
	if err != nil {
		return false, err
	}

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

	// create/update CRDs
	err = createOrUpdateCrds(kv, targetStrategy, stores, clientset, expectations)
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

		}
		// create/update Daemonsets
		for _, daemonSet := range targetStrategy.daemonSets {
			err := syncDaemonSet(kv, daemonSet, stores, clientset, expectations)
			if err != nil {
				return false, err
			}
		}

	}

	// -------- CLEAN UP OLD UNUSED OBJECTS --------
	if !infrastructureRolledOver {
		// still waiting on roll out before cleaning up.
		return false, nil
	}

	// remove unused crds
	objects := stores.CrdCache.List()
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
					err := clientset.CoreV1().Services(kv.Namespace).Delete(svc.Name, deleteOptions)
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

	return true, nil
}
