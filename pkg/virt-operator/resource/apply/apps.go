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

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
)

func (r *Reconciler) syncDeployment(origDeployment *appsv1.Deployment) (*appsv1.Deployment, error) {
	kv := r.kv

	deployment := origDeployment.DeepCopy()

	apps := r.clientset.AppsV1()
	imageTag, imageRegistry, id := getTargetVersionRegistryID(kv)

	injectOperatorMetadata(kv, &deployment.ObjectMeta, imageTag, imageRegistry, id, true)
	injectOperatorMetadata(kv, &deployment.Spec.Template.ObjectMeta, imageTag, imageRegistry, id, false)
	injectPlacementMetadata(kv.Spec.Infra, &deployment.Spec.Template.Spec)

	if kv.Spec.Infra != nil && kv.Spec.Infra.Replicas != nil {
		replicas := int32(*kv.Spec.Infra.Replicas)
		if deployment.Spec.Replicas == nil || *deployment.Spec.Replicas != replicas {
			deployment.Spec.Replicas = &replicas
		}
	}

	obj, exists, _ := r.stores.DeploymentCache.Get(deployment)
	if !exists {
		r.expectations.Deployment.RaiseExpectations(r.kvKey, 1, 0)
		deployment, err := apps.Deployments(kv.Namespace).Create(context.Background(), deployment, metav1.CreateOptions{})
		if err != nil {
			r.expectations.Deployment.LowerExpectations(r.kvKey, 1, 0)
			return nil, fmt.Errorf("unable to create deployment %+v: %v", deployment, err)
		}

		SetGeneration(&kv.Status.Generations, deployment)

		return deployment, nil
	}

	cachedDeployment := obj.(*appsv1.Deployment)
	modified := resourcemerge.BoolPtr(false)
	existingCopy := cachedDeployment.DeepCopy()
	expectedGeneration := GetExpectedGeneration(deployment, kv.Status.Generations)

	resourcemerge.EnsureObjectMeta(modified, &existingCopy.ObjectMeta, deployment.ObjectMeta)

	// there was no change to metadata, the generation matched
	if !*modified &&
		*deployment.Spec.Replicas == *origDeployment.Spec.Replicas &&
		existingCopy.GetGeneration() == expectedGeneration {
		log.Log.V(4).Infof("deployment %v is up-to-date", deployment.GetName())
		return deployment, nil
	}

	newSpec, err := json.Marshal(deployment.Spec)
	if err != nil {
		return nil, err
	}

	ops, err := getPatchWithObjectMetaAndSpec([]string{
		fmt.Sprintf(testGenerationJSONPatchTemplate, cachedDeployment.ObjectMeta.Generation),
	}, &deployment.ObjectMeta, newSpec)
	if err != nil {
		return nil, err
	}

	deployment, err = apps.Deployments(kv.Namespace).Patch(context.Background(), deployment.Name, types.JSONPatchType, generatePatchBytes(ops), metav1.PatchOptions{})
	if err != nil {
		return nil, fmt.Errorf("unable to update deployment %+v: %v", deployment, err)
	}

	SetGeneration(&kv.Status.Generations, deployment)
	log.Log.V(2).Infof("deployment %v updated", deployment.GetName())

	return deployment, nil
}

func (r *Reconciler) syncDaemonSet(daemonSet *appsv1.DaemonSet) error {
	kv := r.kv

	daemonSet = daemonSet.DeepCopy()

	apps := r.clientset.AppsV1()
	imageTag, imageRegistry, id := getTargetVersionRegistryID(kv)

	injectOperatorMetadata(kv, &daemonSet.ObjectMeta, imageTag, imageRegistry, id, true)
	injectOperatorMetadata(kv, &daemonSet.Spec.Template.ObjectMeta, imageTag, imageRegistry, id, false)
	injectPlacementMetadata(kv.Spec.Workloads, &daemonSet.Spec.Template.Spec)

	if daemonSet.GetName() == "virt-handler" {
		setMaxDevices(r.kv, daemonSet)
	}

	var cachedDaemonSet *appsv1.DaemonSet
	obj, exists, _ := r.stores.DaemonSetCache.Get(daemonSet)

	if !exists {
		r.expectations.DaemonSet.RaiseExpectations(r.kvKey, 1, 0)
		daemonSet, err := apps.DaemonSets(kv.Namespace).Create(context.Background(), daemonSet, metav1.CreateOptions{})
		if err != nil {
			r.expectations.DaemonSet.LowerExpectations(r.kvKey, 1, 0)
			return fmt.Errorf("unable to create daemonset %+v: %v", daemonSet, err)
		}

		SetGeneration(&kv.Status.Generations, daemonSet)

		return nil
	}

	cachedDaemonSet = obj.(*appsv1.DaemonSet)
	modified := resourcemerge.BoolPtr(false)
	existingCopy := cachedDaemonSet.DeepCopy()
	expectedGeneration := GetExpectedGeneration(daemonSet, kv.Status.Generations)

	resourcemerge.EnsureObjectMeta(modified, &existingCopy.ObjectMeta, daemonSet.ObjectMeta)
	// there was no change to metadata, the generation was right
	if !*modified && existingCopy.ObjectMeta.Generation == expectedGeneration {
		log.Log.V(4).Infof("daemonset %v is up-to-date", daemonSet.GetName())
		return nil
	}

	newSpec, err := json.Marshal(daemonSet.Spec)
	if err != nil {
		return err
	}

	ops, err := getPatchWithObjectMetaAndSpec([]string{
		fmt.Sprintf(testGenerationJSONPatchTemplate, cachedDaemonSet.ObjectMeta.Generation),
	}, &daemonSet.ObjectMeta, newSpec)
	if err != nil {
		return err
	}

	daemonSet, err = apps.DaemonSets(kv.Namespace).Patch(context.Background(), daemonSet.Name, types.JSONPatchType, generatePatchBytes(ops), metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("unable to update daemonset %+v: %v", daemonSet, err)
	}

	SetGeneration(&kv.Status.Generations, daemonSet)
	log.Log.V(2).Infof("daemonSet %v updated", daemonSet.GetName())

	return nil
}

func setMaxDevices(kv *v1.KubeVirt, vh *appsv1.DaemonSet) {
	if kv.Spec.Configuration.VirtualMachineInstancesPerNode == nil {
		return
	}

	vh.Spec.Template.Spec.Containers[0].Command = append(vh.Spec.Template.Spec.Containers[0].Command,
		"--maxDevices",
		fmt.Sprintf("%d", *kv.Spec.Configuration.VirtualMachineInstancesPerNode))
}

func (r *Reconciler) syncPodDisruptionBudgetForDeployment(deployment *appsv1.Deployment) error {
	kv := r.kv
	podDisruptionBudget := components.NewPodDisruptionBudgetForDeployment(deployment)

	imageTag, imageRegistry, id := getTargetVersionRegistryID(kv)
	injectOperatorMetadata(kv, &podDisruptionBudget.ObjectMeta, imageTag, imageRegistry, id, true)

	pdbClient := r.clientset.PolicyV1beta1().PodDisruptionBudgets(deployment.Namespace)

	var cachedPodDisruptionBudget *policyv1beta1.PodDisruptionBudget
	obj, exists, _ := r.stores.PodDisruptionBudgetCache.Get(podDisruptionBudget)

	if podDisruptionBudget.Spec.MinAvailable.IntValue() == 0 {
		var err error
		if exists {
			err = pdbClient.Delete(context.Background(), podDisruptionBudget.Name, metav1.DeleteOptions{})
		}
		return err
	}

	if !exists {
		r.expectations.PodDisruptionBudget.RaiseExpectations(r.kvKey, 1, 0)
		podDisruptionBudget, err := pdbClient.Create(context.Background(), podDisruptionBudget, metav1.CreateOptions{})
		if err != nil {
			r.expectations.PodDisruptionBudget.LowerExpectations(r.kvKey, 1, 0)
			return fmt.Errorf("unable to create poddisruptionbudget %+v: %v", podDisruptionBudget, err)
		}
		log.Log.V(2).Infof("poddisruptionbudget %v created", podDisruptionBudget.GetName())
		SetGeneration(&kv.Status.Generations, podDisruptionBudget)

		return nil
	}

	cachedPodDisruptionBudget = obj.(*policyv1beta1.PodDisruptionBudget)
	modified := resourcemerge.BoolPtr(false)
	existingCopy := cachedPodDisruptionBudget.DeepCopy()
	expectedGeneration := GetExpectedGeneration(podDisruptionBudget, kv.Status.Generations)

	resourcemerge.EnsureObjectMeta(modified, &existingCopy.ObjectMeta, podDisruptionBudget.ObjectMeta)
	// there was no change to metadata or minAvailable, the generation was right
	if !*modified &&
		existingCopy.Spec.MinAvailable.IntValue() == podDisruptionBudget.Spec.MinAvailable.IntValue() &&
		existingCopy.ObjectMeta.Generation == expectedGeneration {
		log.Log.V(4).Infof("poddisruptionbudget %v is up-to-date", cachedPodDisruptionBudget.GetName())
		return nil
	}

	// Add Spec Patch
	newSpec, err := json.Marshal(podDisruptionBudget.Spec)
	if err != nil {
		return err
	}

	ops, err := getPatchWithObjectMetaAndSpec([]string{}, &podDisruptionBudget.ObjectMeta, newSpec)
	if err != nil {
		return err
	}

	podDisruptionBudget, err = pdbClient.Patch(context.Background(), podDisruptionBudget.Name, types.JSONPatchType, generatePatchBytes(ops), metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("unable to patch/delete poddisruptionbudget %+v: %v", podDisruptionBudget, err)
	}

	SetGeneration(&kv.Status.Generations, podDisruptionBudget)
	log.Log.V(2).Infof("poddisruptionbudget %v patched", podDisruptionBudget.GetName())

	return nil
}
