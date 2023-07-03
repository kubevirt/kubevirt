package apply

import (
	"context"
	"encoding/json"
	"fmt"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/openshift/library-go/pkg/operator/resource/resourcemerge"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
	"kubevirt.io/kubevirt/pkg/virt-operator/util"
)

const (
	failedUpdateDaemonSetReason = "FailedUpdate"
)

var (
	daemonSetDefaultMaxUnavailable = intstr.FromInt(1)
	daemonSetFastMaxUnavailable    = intstr.FromString("10%")
)

type CanaryUpgradeStatus string

const (
	CanaryUpgradeStatusStarted                 CanaryUpgradeStatus = "started"
	CanaryUpgradeStatusUpgradingDaemonSet      CanaryUpgradeStatus = "upgrading daemonset"
	CanaryUpgradeStatusWaitingDaemonSetRollout CanaryUpgradeStatus = "waiting for daemonset rollout"
	CanaryUpgradeStatusSuccessful              CanaryUpgradeStatus = "successful"
	CanaryUpgradeStatusFailed                  CanaryUpgradeStatus = "failed"
)

func (r *Reconciler) syncDeployment(origDeployment *appsv1.Deployment) (*appsv1.Deployment, error) {
	kv := r.kv

	deployment := origDeployment.DeepCopy()

	apps := r.clientset.AppsV1()
	imageTag, imageRegistry, id := getTargetVersionRegistryID(kv)

	injectOperatorMetadata(kv, &deployment.ObjectMeta, imageTag, imageRegistry, id, true)
	injectOperatorMetadata(kv, &deployment.Spec.Template.ObjectMeta, imageTag, imageRegistry, id, false)
	InjectPlacementMetadata(kv.Spec.Infra, &deployment.Spec.Template.Spec)

	if kv.Spec.Infra != nil && kv.Spec.Infra.Replicas != nil {
		replicas := int32(*kv.Spec.Infra.Replicas)
		if deployment.Spec.Replicas == nil || *deployment.Spec.Replicas != replicas {
			deployment.Spec.Replicas = &replicas
			r.recorder.Eventf(deployment, corev1.EventTypeWarning, "AdvancedFeatureUse", "applying custom number of infra replica. this is an advanced feature that prevents "+
				"auto-scaling for core kubevirt components. Please use with caution!")
		}
	} else if deployment.Name == components.VirtAPIName && !replicasAlreadyPatched(r.kv.Spec.CustomizeComponents.Patches, components.VirtAPIName) {
		replicas, err := getDesiredApiReplicas(r.clientset)
		if err != nil {
			log.Log.Object(deployment).Warningf(err.Error())
		} else {
			deployment.Spec.Replicas = pointer.Int32(replicas)
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
		*existingCopy.Spec.Replicas == *deployment.Spec.Replicas &&
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

func setMaxUnavailable(daemonSet *appsv1.DaemonSet, maxUnavailable intstr.IntOrString) {
	daemonSet.Spec.UpdateStrategy.RollingUpdate = &appsv1.RollingUpdateDaemonSet{
		MaxUnavailable: &maxUnavailable,
	}
}

func generateDaemonSetPatch(oldDs, newDs *appsv1.DaemonSet) ([]byte, error) {
	newSpec, err := json.Marshal(newDs.Spec)
	if err != nil {
		return nil, err
	}

	ops, err := getPatchWithObjectMetaAndSpec([]string{
		fmt.Sprintf(testGenerationJSONPatchTemplate, oldDs.ObjectMeta.Generation),
	}, &newDs.ObjectMeta, newSpec)
	if err != nil {
		return nil, err
	}
	return generatePatchBytes(ops), nil
}

func (r *Reconciler) patchDaemonSet(oldDs, newDs *appsv1.DaemonSet) (*appsv1.DaemonSet, error) {
	patch, err := generateDaemonSetPatch(oldDs, newDs)
	if err != nil {
		return nil, err
	}

	newDs, err = r.clientset.AppsV1().DaemonSets(r.kv.Namespace).Patch(
		context.Background(),
		newDs.Name,
		types.JSONPatchType,
		patch,
		metav1.PatchOptions{})
	if err != nil {
		return nil, fmt.Errorf("unable to update daemonset %+v: %v", oldDs, err)
	}
	return newDs, nil
}

func (r *Reconciler) getCanaryPods(daemonSet *appsv1.DaemonSet) []*corev1.Pod {
	canaryPods := []*corev1.Pod{}

	for _, obj := range r.stores.InfrastructurePodCache.List() {
		pod := obj.(*corev1.Pod)
		owner := metav1.GetControllerOf(pod)

		if owner != nil && owner.Name == daemonSet.Name && util.PodIsUpToDate(pod, r.kv) {
			canaryPods = append(canaryPods, pod)
		}
	}
	return canaryPods
}

func (r *Reconciler) howManyUpdatedAndReadyPods(daemonSet *appsv1.DaemonSet) int32 {
	var updatedReadyPods int32

	for _, obj := range r.stores.InfrastructurePodCache.List() {
		pod := obj.(*corev1.Pod)
		owner := metav1.GetControllerOf(pod)

		if owner != nil && owner.Name == daemonSet.Name && util.PodIsUpToDate(pod, r.kv) && util.PodIsReady(pod) {
			updatedReadyPods++
		}
	}
	return updatedReadyPods
}

func daemonHasDefaultRolloutStrategy(daemonSet *appsv1.DaemonSet) bool {
	return getMaxUnavailable(daemonSet) == daemonSetDefaultMaxUnavailable.IntValue()
}

func (r *Reconciler) processCanaryUpgrade(cachedDaemonSet, newDS *appsv1.DaemonSet, forceUpdate bool) (bool, error, CanaryUpgradeStatus) {
	var updatedAndReadyPods int32
	var status CanaryUpgradeStatus
	done := false

	isDaemonSetUpdated := util.DaemonSetIsUpToDate(r.kv, cachedDaemonSet) && !forceUpdate
	desiredReadyPods := cachedDaemonSet.Status.DesiredNumberScheduled
	if isDaemonSetUpdated {
		updatedAndReadyPods = r.howManyUpdatedAndReadyPods(cachedDaemonSet)
	}

	switch {
	case updatedAndReadyPods == 0:
		if !isDaemonSetUpdated {
			// start canary upgrade
			setMaxUnavailable(newDS, daemonSetDefaultMaxUnavailable)
			_, err := r.patchDaemonSet(cachedDaemonSet, newDS)
			if err != nil {
				return false, fmt.Errorf("unable to start canary upgrade for daemonset %+v: %v", newDS, err), CanaryUpgradeStatusFailed
			}
		} else {
			// check for a crashed canary pod
			canaryPods := r.getCanaryPods(cachedDaemonSet)
			for _, canary := range canaryPods {
				if canary != nil && util.PodIsCrashLooping(canary) {
					r.recorder.Eventf(cachedDaemonSet, corev1.EventTypeWarning, failedUpdateDaemonSetReason, "daemonSet %v rollout failed", cachedDaemonSet.Name)
					return false, fmt.Errorf("daemonSet %s rollout failed", cachedDaemonSet.Name), CanaryUpgradeStatusFailed
				}
			}
		}
		done, status = false, CanaryUpgradeStatusStarted
	case updatedAndReadyPods > 0 && updatedAndReadyPods < desiredReadyPods:
		if daemonHasDefaultRolloutStrategy(cachedDaemonSet) {
			// canary was ok, start real rollout
			setMaxUnavailable(newDS, daemonSetFastMaxUnavailable)
			// start rollout again
			_, err := r.patchDaemonSet(cachedDaemonSet, newDS)
			if err != nil {
				return false, fmt.Errorf("unable to update daemonset %+v: %v", newDS, err), CanaryUpgradeStatusFailed
			}
			log.Log.V(2).Infof("daemonSet %v updated", newDS.GetName())
			status = CanaryUpgradeStatusUpgradingDaemonSet
		} else {
			log.Log.V(4).Infof("waiting for all pods of daemonSet %v to be ready", newDS.GetName())
			status = CanaryUpgradeStatusWaitingDaemonSetRollout
		}
		done = false
	case updatedAndReadyPods > 0 && updatedAndReadyPods == desiredReadyPods:
		// rollout has completed and all virt-handlers are ready
		// revert maxUnavailable to default value
		setMaxUnavailable(newDS, daemonSetDefaultMaxUnavailable)
		newDS, err := r.patchDaemonSet(cachedDaemonSet, newDS)
		if err != nil {
			return false, err, CanaryUpgradeStatusFailed
		}
		SetGeneration(&r.kv.Status.Generations, newDS)
		log.Log.V(2).Infof("daemonSet %v is ready", newDS.GetName())
		done, status = true, CanaryUpgradeStatusSuccessful
	}
	return done, nil, status
}

func getMaxUnavailable(daemonSet *appsv1.DaemonSet) int {
	update := daemonSet.Spec.UpdateStrategy.RollingUpdate

	if update == nil {
		return 0
	}
	if update.MaxUnavailable != nil {
		return update.MaxUnavailable.IntValue()
	}
	return daemonSetDefaultMaxUnavailable.IntValue()
}

func (r *Reconciler) syncDaemonSet(daemonSet *appsv1.DaemonSet) (bool, error) {
	kv := r.kv

	daemonSet = daemonSet.DeepCopy()

	apps := r.clientset.AppsV1()
	imageTag, imageRegistry, id := getTargetVersionRegistryID(kv)

	injectOperatorMetadata(kv, &daemonSet.ObjectMeta, imageTag, imageRegistry, id, true)
	injectOperatorMetadata(kv, &daemonSet.Spec.Template.ObjectMeta, imageTag, imageRegistry, id, false)
	InjectPlacementMetadata(kv.Spec.Workloads, &daemonSet.Spec.Template.Spec)

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
			return false, fmt.Errorf("unable to create daemonset %+v: %v", daemonSet, err)
		}

		SetGeneration(&kv.Status.Generations, daemonSet)
		return true, nil
	}

	cachedDaemonSet = obj.(*appsv1.DaemonSet)
	modified := resourcemerge.BoolPtr(false)
	existingCopy := cachedDaemonSet.DeepCopy()
	expectedGeneration := GetExpectedGeneration(daemonSet, kv.Status.Generations)

	resourcemerge.EnsureObjectMeta(modified, &existingCopy.ObjectMeta, daemonSet.ObjectMeta)
	// there was no change to metadata, the generation was right
	if !*modified && existingCopy.GetGeneration() == expectedGeneration {
		log.Log.V(4).Infof("daemonset %v is up-to-date", daemonSet.GetName())
		return true, nil
	}

	// canary pod upgrade
	// first update virt-handler with maxUnavailable=1
	// patch daemonSet with new version
	// wait for a new virt-handler to be ready
	// set maxUnavailable=10%
	// start the rollout of the new virt-handler again
	// wait for all nodes to complete the rollout
	// set maxUnavailable back to 1
	done, err, _ := r.processCanaryUpgrade(cachedDaemonSet, daemonSet, *modified)
	return done, err
}

func setMaxDevices(kv *v1.KubeVirt, vh *appsv1.DaemonSet) {
	if kv.Spec.Configuration.VirtualMachineInstancesPerNode == nil {
		return
	}

	vh.Spec.Template.Spec.Containers[0].Command = append(vh.Spec.Template.Spec.Containers[0].Command,
		"--max-devices",
		fmt.Sprintf("%d", *kv.Spec.Configuration.VirtualMachineInstancesPerNode))
}

func (r *Reconciler) syncPodDisruptionBudgetForDeployment(deployment *appsv1.Deployment) error {
	kv := r.kv
	podDisruptionBudget := components.NewPodDisruptionBudgetForDeployment(deployment)

	imageTag, imageRegistry, id := getTargetVersionRegistryID(kv)
	injectOperatorMetadata(kv, &podDisruptionBudget.ObjectMeta, imageTag, imageRegistry, id, true)

	pdbClient := r.clientset.PolicyV1().PodDisruptionBudgets(deployment.Namespace)

	var cachedPodDisruptionBudget *policyv1.PodDisruptionBudget
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

	cachedPodDisruptionBudget = obj.(*policyv1.PodDisruptionBudget)
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

func getDesiredApiReplicas(clientset kubecli.KubevirtClient) (replicas int32, err error) {
	nodeList, err := clientset.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return 0, fmt.Errorf("failed to get number of nodes to determine virt-api replicas: %v", err)
	}

	nodesCount := len(nodeList.Items)
	// This is a simple heuristic to achieve basic scalability so we could be running on large clusters.
	// From recent experiments we know that for a 100 nodes cluster, 9 virt-api replicas are enough.
	// This heuristic is not accurate. It could, and should, be replaced by something more sophisticated and refined
	// in the future.

	if nodesCount == 1 {
		return 1, nil
	}

	const minReplicas = 2

	replicas = int32(nodesCount) / 10
	if replicas < minReplicas {
		replicas = minReplicas
	}

	return replicas, nil
}

func replicasAlreadyPatched(patches []v1.CustomizeComponentsPatch, deploymentName string) bool {
	for _, patch := range patches {
		if patch.ResourceName != deploymentName {
			continue
		}
		decodedPatch, err := jsonpatch.DecodePatch([]byte(patch.Patch))
		if err != nil {
			log.Log.Warningf(err.Error())
			continue
		}
		for _, operation := range decodedPatch {
			path, err := operation.Path()
			if err != nil {
				log.Log.Warningf(err.Error())
				continue
			}
			op := operation.Kind()
			if path == "/spec/replicas" && op == "replace" {
				return true
			}
		}
	}
	return false
}
