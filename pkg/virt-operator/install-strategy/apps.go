package installstrategy

import (
	"encoding/json"
	"fmt"

	"kubevirt.io/kubevirt/pkg/controller"

	appsv1 "k8s.io/api/apps/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	"k8s.io/apimachinery/pkg/types"

	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/virt-operator/creation/components"
)

func (r *Reconciler) syncDeployment(deployment *appsv1.Deployment) error {
	deployment = deployment.DeepCopy()

	apps := r.clientset.AppsV1()
	imageTag, imageRegistry, id := getTargetVersionRegistryID(r.kv)

	injectOperatorMetadata(r.kv, &deployment.ObjectMeta, imageTag, imageRegistry, id, true)
	injectOperatorMetadata(r.kv, &deployment.Spec.Template.ObjectMeta, imageTag, imageRegistry, id, false)
	injectPlacementMetadata(r.kv.Spec.Infra, &deployment.Spec.Template.Spec)

	kvkey, err := controller.KeyFunc(r.kv)
	if err != nil {
		return err
	}

	var cachedDeployment *appsv1.Deployment

	obj, exists, _ := r.stores.DeploymentCache.Get(deployment)
	if exists {
		cachedDeployment = obj.(*appsv1.Deployment)
	}

	if !exists {
		r.expectations.Deployment.RaiseExpectations(kvkey, 1, 0)
		_, err = apps.Deployments(r.kv.Namespace).Create(deployment)
		if err != nil {
			r.expectations.Deployment.LowerExpectations(kvkey, 1, 0)
			return fmt.Errorf("unable to create deployment %+v: %v", deployment, err)
		}
	} else if !objectMatchesVersion(&cachedDeployment.ObjectMeta, imageTag, imageRegistry, id, r.kv.GetGeneration()) {
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

		_, err = apps.Deployments(r.kv.Namespace).Patch(deployment.Name, types.JSONPatchType, generatePatchBytes(ops))
		if err != nil {
			return fmt.Errorf("unable to patch deployment %+v: %v", deployment, err)
		}
		log.Log.V(2).Infof("deployment %v updated", deployment.GetName())

	} else {
		log.Log.V(4).Infof("deployment %v is up-to-date", deployment.GetName())
	}

	return nil
}

func (r *Reconciler) syncDaemonSet(daemonSet *appsv1.DaemonSet) error {

	daemonSet = daemonSet.DeepCopy()
	kv := r.kv

	apps := r.clientset.AppsV1()
	imageTag, imageRegistry, id := getTargetVersionRegistryID(r.kv)

	injectOperatorMetadata(kv, &daemonSet.ObjectMeta, imageTag, imageRegistry, id, true)
	injectOperatorMetadata(kv, &daemonSet.Spec.Template.ObjectMeta, imageTag, imageRegistry, id, false)
	injectPlacementMetadata(kv.Spec.Workloads, &daemonSet.Spec.Template.Spec)

	kvkey, err := controller.KeyFunc(kv)
	if err != nil {
		return err
	}

	var cachedDaemonSet *appsv1.DaemonSet
	obj, exists, _ := r.stores.DaemonSetCache.Get(daemonSet)
	if exists {
		cachedDaemonSet = obj.(*appsv1.DaemonSet)
	}
	if !exists {
		r.expectations.DaemonSet.RaiseExpectations(kvkey, 1, 0)
		_, err = apps.DaemonSets(kv.Namespace).Create(daemonSet)
		if err != nil {
			r.expectations.DaemonSet.LowerExpectations(kvkey, 1, 0)
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

func (r *Reconciler) syncPodDisruptionBudgetForDeployment(deployment *appsv1.Deployment) error {
	podDisruptionBudget := components.NewPodDisruptionBudgetForDeployment(deployment)

	imageTag, imageRegistry, id := getTargetVersionRegistryID(r.kv)
	injectOperatorMetadata(r.kv, &podDisruptionBudget.ObjectMeta, imageTag, imageRegistry, id, true)

	pdbClient := r.clientset.PolicyV1beta1().PodDisruptionBudgets(deployment.Namespace)

	var cachedPodDisruptionBudget *policyv1beta1.PodDisruptionBudget
	obj, exists, _ := r.stores.PodDisruptionBudgetCache.Get(podDisruptionBudget)
	if exists {
		cachedPodDisruptionBudget = obj.(*policyv1beta1.PodDisruptionBudget)
	}

	if !exists {
		r.expectations.PodDisruptionBudget.RaiseExpectations(r.kvKey, 1, 0)
		_, err := pdbClient.Create(podDisruptionBudget)
		if err != nil {
			r.expectations.PodDisruptionBudget.LowerExpectations(r.kvKey, 1, 0)
			return fmt.Errorf("unable to create poddisruptionbudget %+v: %v", podDisruptionBudget, err)
		}
		log.Log.V(2).Infof("poddisruptionbudget %v created", podDisruptionBudget.GetName())

		return nil
	}

	if objectMatchesVersion(&cachedPodDisruptionBudget.ObjectMeta, imageTag, imageRegistry, id, r.kv.GetGeneration()) {
		log.Log.V(4).Infof("poddisruptionbudget %v is up-to-date", cachedPodDisruptionBudget.GetName())
		return nil
	}
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

	return nil
}
