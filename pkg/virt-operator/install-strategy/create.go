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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	extv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

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

func injectOperatorLabelAndAnnotations(objectMeta *metav1.ObjectMeta, imageTag string, imageRegistry string) {
	if objectMeta.Labels == nil {
		objectMeta.Labels = make(map[string]string)
	}
	objectMeta.Labels[v1.ManagedByLabel] = v1.ManagedByLabelOperatorValue

	if objectMeta.Annotations == nil {
		objectMeta.Annotations = make(map[string]string)
	}
	objectMeta.Annotations[v1.InstallStrategyVersionAnnotation] = imageTag
	objectMeta.Annotations[v1.InstallStrategyRegistryAnnotation] = imageRegistry
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

	injectOperatorLabelAndAnnotations(&daemonSet.ObjectMeta, imageTag, imageRegistry)
	injectOperatorLabelAndAnnotations(&daemonSet.Spec.Template.ObjectMeta, imageTag, imageRegistry)

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

	injectOperatorLabelAndAnnotations(&deployment.ObjectMeta, imageTag, imageRegistry)
	injectOperatorLabelAndAnnotations(&deployment.Spec.Template.ObjectMeta, imageTag, imageRegistry)

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

	ext := clientset.ExtensionsClient()
	core := clientset.CoreV1()
	rbac := clientset.RbacV1()
	scc := clientset.SecClient()

	imageTag := kv.Status.TargetKubeVirtVersion
	imageRegistry := kv.Status.TargetKubeVirtRegistry

	takeUpdatePath := shouldTakeUpdatePath(kv.Status.TargetKubeVirtVersion, kv.Status.ObservedKubeVirtVersion)

	// -------- CREATE AND ROLE OUT UPDATED OBJECTS --------

	// create/update CRDs
	for _, crd := range targetStrategy.crds {
		var cachedCrd *extv1beta1.CustomResourceDefinition

		crd := crd.DeepCopy()
		obj, exists, _ := stores.CrdCache.Get(crd)
		if exists {
			cachedCrd = obj.(*extv1beta1.CustomResourceDefinition)
		}

		injectOperatorLabelAndAnnotations(&crd.ObjectMeta, imageTag, imageRegistry)
		if !exists {
			// Create non existent
			expectations.Crd.RaiseExpectations(kvkey, 1, 0)
			_, err := ext.ApiextensionsV1beta1().CustomResourceDefinitions().Create(crd)
			if err != nil {
				expectations.Crd.LowerExpectations(kvkey, 1, 0)
				return false, fmt.Errorf("unable to create crd %+v: %v", crd, err)
			}
			log.Log.V(2).Infof("crd %v created", crd.GetName())

		} else if !objectMatchesVersion(&cachedCrd.ObjectMeta, imageTag, imageRegistry) {
			// Patch if old version
			var ops []string

			if crd.Spec.Version != cachedCrd.Spec.Version {
				// We can't support transitioning between versions until
				// the conversion webhook is supported.
				return false, fmt.Errorf("No supported update path from crd %s version %s to version %s", crd.Name, cachedCrd.Spec.Version, crd.Spec.Version)
			}

			// Add Labels and Annotations Patches
			labelAnnotationPatch, err := createLabelsAndAnnotationsPatch(&crd.ObjectMeta)
			if err != nil {
				return false, err
			}
			ops = append(ops, labelAnnotationPatch...)

			// Add Spec Patch
			newSpec, err := json.Marshal(crd.Spec)
			if err != nil {
				return false, err
			}
			ops = append(ops, fmt.Sprintf(`{ "op": "replace", "path": "/spec", "value": %s }`, string(newSpec)))

			_, err = ext.ApiextensionsV1beta1().CustomResourceDefinitions().Patch(crd.Name, types.JSONPatchType, generatePatchBytes(ops))
			if err != nil {
				return false, fmt.Errorf("unable to patch crd %+v: %v", crd, err)
			}
			log.Log.V(2).Infof("crd %v updated", crd.GetName())

		} else {
			log.Log.V(4).Infof("crd %v is up-to-date", crd.GetName())
		}
	}

	// create/update ServiceAccounts
	for _, sa := range targetStrategy.serviceAccounts {
		var cachedSa *corev1.ServiceAccount

		sa := sa.DeepCopy()
		obj, exists, _ := stores.ServiceAccountCache.Get(sa)
		if exists {
			cachedSa = obj.(*corev1.ServiceAccount)
		}

		injectOperatorLabelAndAnnotations(&sa.ObjectMeta, imageTag, imageRegistry)
		if !exists {
			// Create non existent
			expectations.ServiceAccount.RaiseExpectations(kvkey, 1, 0)
			_, err := core.ServiceAccounts(kv.Namespace).Create(sa)
			if err != nil {
				expectations.ServiceAccount.LowerExpectations(kvkey, 1, 0)
				return false, fmt.Errorf("unable to create serviceaccount %+v: %v", sa, err)
			}
			log.Log.V(2).Infof("serviceaccount %v created", sa.GetName())
		} else if !objectMatchesVersion(&cachedSa.ObjectMeta, imageTag, imageRegistry) {
			// Patch if old version
			var ops []string

			// Patch Labels and Annotations
			labelAnnotationPatch, err := createLabelsAndAnnotationsPatch(&sa.ObjectMeta)
			if err != nil {
				return false, err
			}
			ops = append(ops, labelAnnotationPatch...)

			_, err = core.ServiceAccounts(kv.Namespace).Patch(sa.Name, types.JSONPatchType, generatePatchBytes(ops))
			if err != nil {
				return false, fmt.Errorf("unable to patch serviceaccount %+v: %v", sa, err)
			}
			log.Log.V(2).Infof("serviceaccount %v updated", sa.GetName())

		} else {
			// Up to date
			log.Log.V(2).Infof("serviceaccount %v already exists and is up-to-date", sa.GetName())
		}
	}

	// create/update ClusterRoles
	for _, cr := range targetStrategy.clusterRoles {
		var cachedCr *rbacv1.ClusterRole

		cr := cr.DeepCopy()
		obj, exists, _ := stores.ClusterRoleCache.Get(cr)

		if exists {
			cachedCr = obj.(*rbacv1.ClusterRole)
		}

		injectOperatorLabelAndAnnotations(&cr.ObjectMeta, imageTag, imageRegistry)
		if !exists {
			// Create non existent
			expectations.ClusterRole.RaiseExpectations(kvkey, 1, 0)
			_, err := rbac.ClusterRoles().Create(cr)
			if err != nil {
				expectations.ClusterRole.LowerExpectations(kvkey, 1, 0)
				return false, fmt.Errorf("unable to create clusterrole %+v: %v", cr, err)
			}
			log.Log.V(2).Infof("clusterrole %v created", cr.GetName())
		} else if !objectMatchesVersion(&cachedCr.ObjectMeta, imageTag, imageRegistry) {
			// Update existing, we don't need to patch for rbac rules.
			_, err = rbac.ClusterRoles().Update(cr)
			if err != nil {
				return false, fmt.Errorf("unable to update clusterrole %+v: %v", cr, err)
			}
			log.Log.V(2).Infof("clusterrole %v updated", cr.GetName())

		} else {
			log.Log.V(4).Infof("clusterrole %v already exists", cr.GetName())
		}
	}

	// create/update ClusterRoleBindings
	for _, crb := range targetStrategy.clusterRoleBindings {

		var cachedCrb *rbacv1.ClusterRoleBinding

		crb := crb.DeepCopy()
		obj, exists, _ := stores.ClusterRoleBindingCache.Get(crb)
		if exists {
			cachedCrb = obj.(*rbacv1.ClusterRoleBinding)
		}

		injectOperatorLabelAndAnnotations(&crb.ObjectMeta, imageTag, imageRegistry)
		if !exists {
			// Create non existent
			expectations.ClusterRoleBinding.RaiseExpectations(kvkey, 1, 0)
			_, err := rbac.ClusterRoleBindings().Create(crb)
			if err != nil {
				expectations.ClusterRoleBinding.LowerExpectations(kvkey, 1, 0)
				return false, fmt.Errorf("unable to create clusterrolebinding %+v: %v", crb, err)
			}
			log.Log.V(2).Infof("clusterrolebinding %v created", crb.GetName())
		} else if !objectMatchesVersion(&cachedCrb.ObjectMeta, imageTag, imageRegistry) {
			// Update existing, we don't need to patch for rbac rules.
			_, err = rbac.ClusterRoleBindings().Update(crb)
			if err != nil {
				return false, fmt.Errorf("unable to update clusterrolebinding %+v: %v", crb, err)
			}
			log.Log.V(2).Infof("clusterrolebinding %v updated", crb.GetName())

		} else {
			log.Log.V(4).Infof("clusterrolebinding %v already exists", crb.GetName())
		}
	}

	// create/update Roles
	for _, r := range targetStrategy.roles {
		var cachedR *rbacv1.Role

		r := r.DeepCopy()
		obj, exists, _ := stores.RoleCache.Get(r)
		if exists {
			cachedR = obj.(*rbacv1.Role)
		}

		injectOperatorLabelAndAnnotations(&r.ObjectMeta, imageTag, imageRegistry)
		if !exists {
			// Create non existent
			expectations.Role.RaiseExpectations(kvkey, 1, 0)
			_, err := rbac.Roles(kv.Namespace).Create(r)
			if err != nil {
				expectations.Role.LowerExpectations(kvkey, 1, 0)
				return false, fmt.Errorf("unable to create role %+v: %v", r, err)
			}
			log.Log.V(2).Infof("role %v created", r.GetName())
		} else if !objectMatchesVersion(&cachedR.ObjectMeta, imageTag, imageRegistry) {
			// Update existing, we don't need to patch for rbac rules.
			_, err = rbac.Roles(kv.Namespace).Update(r)
			if err != nil {
				return false, fmt.Errorf("unable to update role %+v: %v", r, err)
			}
			log.Log.V(2).Infof("role %v updated", r.GetName())

		} else {
			log.Log.V(4).Infof("role %v already exists", r.GetName())
		}
	}

	// create/update RoleBindings
	for _, rb := range targetStrategy.roleBindings {
		var cachedRb *rbacv1.RoleBinding

		rb := rb.DeepCopy()
		obj, exists, _ := stores.RoleBindingCache.Get(rb)

		if exists {
			cachedRb = obj.(*rbacv1.RoleBinding)
		}

		injectOperatorLabelAndAnnotations(&rb.ObjectMeta, imageTag, imageRegistry)
		if !exists {
			// Create non existent
			expectations.RoleBinding.RaiseExpectations(kvkey, 1, 0)
			_, err := rbac.RoleBindings(kv.Namespace).Create(rb)
			if err != nil {
				expectations.RoleBinding.LowerExpectations(kvkey, 1, 0)
				return false, fmt.Errorf("unable to create rolebinding %+v: %v", rb, err)
			}
			log.Log.V(2).Infof("rolebinding %v created", rb.GetName())
		} else if !objectMatchesVersion(&cachedRb.ObjectMeta, imageTag, imageRegistry) {
			// Update existing, we don't need to patch for rbac rules.
			_, err = rbac.RoleBindings(kv.Namespace).Update(rb)
			if err != nil {
				return false, fmt.Errorf("unable to update rolebinding %+v: %v", rb, err)
			}
			log.Log.V(2).Infof("rolebinding %v updated", rb.GetName())

		} else {
			log.Log.V(4).Infof("rolebinding %v already exists", rb.GetName())
		}
	}

	// create/update Services
	for _, service := range targetStrategy.services {
		var cachedService *corev1.Service
		service = service.DeepCopy()

		obj, exists, _ := stores.ServiceCache.Get(service)
		if exists {
			cachedService = obj.(*corev1.Service)
		}

		injectOperatorLabelAndAnnotations(&service.ObjectMeta, imageTag, imageRegistry)
		if !exists {
			expectations.Service.RaiseExpectations(kvkey, 1, 0)
			_, err := core.Services(kv.Namespace).Create(service)
			if err != nil {
				expectations.Service.LowerExpectations(kvkey, 1, 0)
				return false, fmt.Errorf("unable to create service %+v: %v", service, err)
			}
		} else if !objectMatchesVersion(&cachedService.ObjectMeta, imageTag, imageRegistry) {
			if !reflect.DeepEqual(cachedService.Spec, service.Spec) {

				// The spec of a service is immutable. If the specs
				// are not equal, we have to delete and recreate them.
				if cachedService.DeletionTimestamp == nil {
					if key, err := controller.KeyFunc(cachedService); err == nil {
						expectations.Service.AddExpectedDeletion(kvkey, key)
						err := clientset.CoreV1().Services(kv.Namespace).Delete(cachedService.Name, deleteOptions)
						if err != nil {
							expectations.Service.DeletionObserved(kvkey, key)
							log.Log.Errorf("Failed to delete service %+v: %v", cachedService, err)
							return false, err
						}

						log.Log.V(2).Infof("service %v deleted. It must be re-created", cachedService.GetName())
						return false, nil
					}
				}
			} else {
				// Patch if old version
				var ops []string

				// Add Labels and Annotations Patches
				labelAnnotationPatch, err := createLabelsAndAnnotationsPatch(&service.ObjectMeta)
				if err != nil {
					return false, err
				}
				ops = append(ops, labelAnnotationPatch...)

				_, err = core.Services(kv.Namespace).Patch(service.Name, types.JSONPatchType, generatePatchBytes(ops))
				if err != nil {
					return false, fmt.Errorf("unable to patch service %+v: %v", service, err)
				}
				log.Log.V(2).Infof("service %v updated", service.GetName())
			}
		} else {
			log.Log.V(4).Infof("service %v is up-to-date", service.GetName())
		}
	}

	// Add new SCC Privileges and remove unsed SCC Privileges
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
			return false, err
		}

		privSCC, ok := privSCCObj.(*secv1.SecurityContextConstraints)
		if !ok {
			return false, fmt.Errorf("couldn't cast object to SecurityContextConstraints: %+v", privSCCObj)
		}
		privSCCCopy := privSCC.DeepCopy()

		modified := false
		users := privSCCCopy.Users

		// remove users from previous
		if curSccPriv != nil {
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
				return false, err
			}

			data := []byte(fmt.Sprintf(`{"users": %s}`, userBytes))
			_, err = scc.SecurityContextConstraints().Patch(sccPriv.TargetSCC, types.StrategicMergePatchType, data)
			if err != nil {
				return false, fmt.Errorf("unable to patch scc: %v", err)
			}
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

		// wait for daemonsets
		for _, daemonSet := range targetStrategy.daemonSets {
			if !util.DaemonsetIsReady(kv, daemonSet, stores) {
				log.Log.V(2).Infof("Waiting on daemonset %v to roll over to latest version", daemonSet.GetName())
				// not rolled out yet
				return false, nil
			}
		}
		// wait for controller deployments
		for _, deployment := range controllerDeployments(targetStrategy) {
			if !util.DeploymentIsReady(kv, deployment, stores) {
				log.Log.V(2).Infof("Waiting on deployment %v to roll over to latest version", deployment.GetName())
				// not rolled out yet
				return false, nil
			}
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
		for _, deployment := range apiDeployments(targetStrategy) {
			if !util.DeploymentIsReady(kv, deployment, stores) {
				log.Log.V(2).Infof("Waiting on deployment %v to roll over to latest version", deployment.GetName())
				// not rolled out yet
				return false, nil
			}
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
