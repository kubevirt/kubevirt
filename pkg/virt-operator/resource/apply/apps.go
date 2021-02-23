package apply

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/openshift/library-go/pkg/operator/resource/resourcemerge"
	appsv1 "k8s.io/api/apps/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
)

func (r *Reconciler) syncDeployment(deployment *appsv1.Deployment) error {
	kv := r.kv

	deployment = deployment.DeepCopy()

	apps := r.clientset.AppsV1()
	imageTag, imageRegistry, id := getTargetVersionRegistryID(kv)

	injectOperatorMetadata(kv, &deployment.ObjectMeta, imageTag, imageRegistry, id, true)
	injectOperatorMetadata(kv, &deployment.Spec.Template.ObjectMeta, imageTag, imageRegistry, id, false)
	injectPlacementMetadata(kv.Spec.Infra, &deployment.Spec.Template.Spec)

	obj, exists, _ := r.stores.DeploymentCache.Get(deployment)
	if !exists {
		r.expectations.Deployment.RaiseExpectations(r.kvKey, 1, 0)
		deployment, err := apps.Deployments(kv.Namespace).Create(context.Background(), deployment, metav1.CreateOptions{})
		if err != nil {
			r.expectations.Deployment.LowerExpectations(r.kvKey, 1, 0)
			return fmt.Errorf("unable to create deployment %+v: %v", deployment, err)
		}

		resourcemerge.SetDeploymentGeneration(&kv.Status.Generations, deployment)

		return nil
	}

	cachedDeployment := obj.(*appsv1.Deployment)
	modified := resourcemerge.BoolPtr(false)
	existingCopy := cachedDeployment.DeepCopy()
	expectedGeneration := resourcemerge.ExpectedDeploymentGeneration(deployment, kv.Status.Generations)

	resourcemerge.EnsureObjectMeta(modified, &existingCopy.ObjectMeta, deployment.ObjectMeta)
	// there was no change to metadata, the generation matched
	if !*modified && existingCopy.ObjectMeta.Generation == expectedGeneration {
		log.Log.V(4).Infof("deployment %v is up-to-date", deployment.GetName())
		return nil
	}

	// Patch if old version
	ops := []string{
		fmt.Sprintf(testGenerationJSONPatchTemplate, cachedDeployment.ObjectMeta.Generation),
	}

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
	ops = append(ops, fmt.Sprintf(replaceSpecPatchTemplate, string(newSpec)))

	deployment, err = apps.Deployments(kv.Namespace).Patch(context.Background(), deployment.Name, types.JSONPatchType, generatePatchBytes(ops), metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("unable to update deployment %+v: %v", deployment, err)
	}

	resourcemerge.SetDeploymentGeneration(&kv.Status.Generations, deployment)
	log.Log.V(2).Infof("deployment %v updated", deployment.GetName())

	return nil
}

func (r *Reconciler) syncDaemonSet(daemonSet *appsv1.DaemonSet) error {
	kv := r.kv

	daemonSet = daemonSet.DeepCopy()

	apps := r.clientset.AppsV1()
	imageTag, imageRegistry, id := getTargetVersionRegistryID(kv)

	injectOperatorMetadata(kv, &daemonSet.ObjectMeta, imageTag, imageRegistry, id, true)
	injectOperatorMetadata(kv, &daemonSet.Spec.Template.ObjectMeta, imageTag, imageRegistry, id, false)
	injectPlacementMetadata(kv.Spec.Workloads, &daemonSet.Spec.Template.Spec)

	var cachedDaemonSet *appsv1.DaemonSet
	obj, exists, _ := r.stores.DaemonSetCache.Get(daemonSet)

	if !exists {
		r.expectations.DaemonSet.RaiseExpectations(r.kvKey, 1, 0)
		daemonSet, err := apps.DaemonSets(kv.Namespace).Create(context.Background(), daemonSet, metav1.CreateOptions{})
		if err != nil {
			r.expectations.DaemonSet.LowerExpectations(r.kvKey, 1, 0)
			return fmt.Errorf("unable to create daemonset %+v: %v", daemonSet, err)
		}

		resourcemerge.SetDaemonSetGeneration(&kv.Status.Generations, daemonSet)

		return nil
	}

	cachedDaemonSet = obj.(*appsv1.DaemonSet)
	modified := resourcemerge.BoolPtr(false)
	existingCopy := cachedDaemonSet.DeepCopy()
	expectedGeneration := resourcemerge.ExpectedDaemonSetGeneration(daemonSet, kv.Status.Generations)

	resourcemerge.EnsureObjectMeta(modified, &existingCopy.ObjectMeta, daemonSet.ObjectMeta)
	// there was no change to metadata, the generation was right
	if !*modified && existingCopy.ObjectMeta.Generation == expectedGeneration {
		log.Log.V(4).Infof("daemonset %v is up-to-date", daemonSet.GetName())
		return nil
	}

	// Patch if old version
	ops := []string{
		fmt.Sprintf(testGenerationJSONPatchTemplate, cachedDaemonSet.ObjectMeta.Generation),
	}

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
	ops = append(ops, fmt.Sprintf(replaceSpecPatchTemplate, string(newSpec)))

	daemonSet, err = apps.DaemonSets(kv.Namespace).Patch(context.Background(), daemonSet.Name, types.JSONPatchType, generatePatchBytes(ops), metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("unable to update daemonset %+v: %v", daemonSet, err)
	}

	resourcemerge.SetDaemonSetGeneration(&kv.Status.Generations, daemonSet)
	log.Log.V(2).Infof("daemonSet %v updated", daemonSet.GetName())

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
		_, err := pdbClient.Create(context.Background(), podDisruptionBudget, metav1.CreateOptions{})
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

	_, err = pdbClient.Patch(context.Background(), podDisruptionBudget.Name, types.JSONPatchType, generatePatchBytes(ops), metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("unable to patch poddisruptionbudget %+v: %v", podDisruptionBudget, err)
	}
	log.Log.V(2).Infof("poddisruptionbudget %v patched", podDisruptionBudget.GetName())

	return nil
}
