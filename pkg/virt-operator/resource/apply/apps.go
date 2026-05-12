package apply

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/openshift/library-go/pkg/operator/resource/resourcemerge"
	jsonpatch "gopkg.in/evanphx/json-patch.v4"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/pointer"
	kvtls "kubevirt.io/kubevirt/pkg/util/tls"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/placement"
	"kubevirt.io/kubevirt/pkg/virt-operator/util"
)

const (
	failedUpdateDaemonSetReason = "FailedUpdate"
	replaceOp                   = "replace"
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

func (r *Reconciler) syncDeployment(origDeployment *appsv1.Deployment) (*appsv1.Deployment, error) { //nolint:gocyclo,funlen
	kv := r.kv

	deployment := origDeployment.DeepCopy()

	apps := r.clientset.AppsV1()
	imageTag, imageRegistry, id := getTargetVersionRegistryID(kv)

	injectOperatorMetadata(kv, &deployment.ObjectMeta, imageTag, imageRegistry, id, true)
	injectOperatorMetadata(kv, &deployment.Spec.Template.ObjectMeta, imageTag, imageRegistry, id, false)
	placement.InjectPlacementMetadata(kv.Spec.Infra, &deployment.Spec.Template.Spec, placement.RequireControlPlanePreferNonWorker)

	if kv.Spec.Infra != nil && kv.Spec.Infra.Replicas != nil {
		replicas := int32(*kv.Spec.Infra.Replicas)
		if (deployment.Spec.Replicas == nil || *deployment.Spec.Replicas != replicas) &&
			deployment.Name != components.VirtTemplateApiserverDeploymentName &&
			deployment.Name != components.VirtTemplateControllerDeploymentName {
			deployment.Spec.Replicas = &replicas
			r.recorder.Eventf(deployment, corev1.EventTypeWarning,
				"AdvancedFeatureUse",
				"applying custom number of infra replica. "+
					"this is an advanced feature that prevents "+
					"auto-scaling for core kubevirt components. "+
					"Please use with caution!")
		}
	} else if deployment.Name == components.VirtAPIName &&
		!replicasAlreadyPatched(r.kv.Spec.CustomizeComponents.Patches, components.VirtAPIName) {
		replicas, err := getDesiredAPIReplicas(r.clientset)
		if err != nil {
			log.Log.Object(deployment).Warningf("%s", err.Error())
		} else {
			deployment.Spec.Replicas = pointer.P(replicas)
		}
	}

	switch deployment.Name {
	case components.VirtTemplateApiserverDeploymentName:
		if err := kvtls.InjectTLSConfigIntoDeployment(kv, deployment, components.VirtTemplateApiserverContainerName); err != nil {
			return nil, err
		}
	case components.VirtTemplateControllerDeploymentName:
		if err := kvtls.InjectTLSConfigIntoDeployment(kv, deployment, components.VirtTemplateControllerContainerName); err != nil {
			return nil, err
		}
	}

	obj, exists, _ := r.stores.DeploymentCache.Get(deployment)
	if !exists {
		r.expectations.Deployment.RaiseExpectations(r.kvKey, 1, 0)
		origDeployment := deployment
		var createErr error
		deployment, createErr = apps.Deployments(kv.Namespace).Create(context.Background(), deployment, metav1.CreateOptions{})
		if createErr != nil {
			r.expectations.Deployment.LowerExpectations(r.kvKey, 1, 0)
			log.Log.V(2).Infof("failed to create deployment %s: %+v", origDeployment.Name, origDeployment) //nolint:mnd
			return nil, fmt.Errorf("unable to create deployment %s: %v", origDeployment.Name, createErr)
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
		log.Log.V(4).Infof("deployment %v is up-to-date", deployment.GetName()) //nolint:mnd
		return deployment, nil
	}

	const revisionAnnotation = "deployment.kubernetes.io/revision"
	if val, ok := existingCopy.ObjectMeta.Annotations[revisionAnnotation]; ok {
		if deployment.ObjectMeta.Annotations == nil {
			deployment.ObjectMeta.Annotations = map[string]string{}
		}
		deployment.ObjectMeta.Annotations[revisionAnnotation] = val
	}

	ops, err := patch.New(getPatchWithObjectMetaAndSpec([]patch.PatchOption{
		patch.WithTest("/metadata/generation", cachedDeployment.ObjectMeta.Generation),
	},
		&deployment.ObjectMeta, deployment.Spec)...).GeneratePayload()
	if err != nil {
		return nil, err
	}

	prePatchDeployment := deployment
	deployment, err = apps.Deployments(kv.Namespace).Patch(
		context.Background(), deployment.Name, types.JSONPatchType,
		ops, metav1.PatchOptions{},
	)
	if err != nil {
		log.Log.V(2).Infof("failed to update deployment %s: %+v", prePatchDeployment.Name, prePatchDeployment) //nolint:mnd
		return nil, fmt.Errorf("unable to update deployment %s: %v", prePatchDeployment.Name, err)
	}

	SetGeneration(&kv.Status.Generations, deployment)
	log.Log.V(2).Infof("deployment %v updated", deployment.GetName()) //nolint:mnd

	return deployment, nil
}

func setMaxUnavailable(daemonSet *appsv1.DaemonSet, maxUnavailable intstr.IntOrString) {
	daemonSet.Spec.UpdateStrategy.RollingUpdate = &appsv1.RollingUpdateDaemonSet{
		MaxUnavailable: &maxUnavailable,
	}
}

func generateDaemonSetPatch(oldDs, newDs *appsv1.DaemonSet) ([]byte, error) {
	return patch.New(
		getPatchWithObjectMetaAndSpec([]patch.PatchOption{
			patch.WithTest("/metadata/generation", oldDs.ObjectMeta.Generation),
		},
			&newDs.ObjectMeta, newDs.Spec)...).GeneratePayload()
}

func (r *Reconciler) patchDaemonSet(oldDs, newDs *appsv1.DaemonSet) (*appsv1.DaemonSet, error) {
	patchBytes, err := generateDaemonSetPatch(oldDs, newDs)
	if err != nil {
		return nil, err
	}

	newDs, err = r.clientset.AppsV1().DaemonSets(r.kv.Namespace).Patch(
		context.Background(),
		newDs.Name,
		types.JSONPatchType,
		patchBytes,
		metav1.PatchOptions{})
	if err != nil {
		log.Log.V(2).Infof("failed to update daemonset %s: %+v", oldDs.Name, oldDs) //nolint:mnd
		return nil, fmt.Errorf("unable to update daemonset %s: %v", oldDs.Name, err)
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

//nolint:gocyclo,funlen,staticcheck
func (r *Reconciler) processCanaryUpgrade(
	cachedDaemonSet, newDS *appsv1.DaemonSet, forceUpdate bool,
) (bool, error, CanaryUpgradeStatus) {
	var updatedAndReadyPods int32
	var status CanaryUpgradeStatus
	done := false

	if hasTLS(cachedDaemonSet) && !hasTLS(newDS) {
		insertTLS(newDS)
	}
	if !hasCertificateSecret(&cachedDaemonSet.Spec.Template.Spec, components.VirtHandlerCertSecretName) &&
		hasCertificateSecret(&newDS.Spec.Template.Spec, components.VirtHandlerCertSecretName) {
		unattachCertificateSecret(&newDS.Spec.Template.Spec, components.VirtHandlerCertSecretName)
	}
	logger := log.Log.With("resource", fmt.Sprintf("ds/%s", cachedDaemonSet.Name))

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
				logger.V(2).Infof("failed to start canary upgrade for daemonset %s: %+v", newDS.Name, newDS) //nolint:mnd
				return false,
					fmt.Errorf("unable to start canary upgrade for daemonset %s: %v", newDS.Name, err),
					CanaryUpgradeStatusFailed
			}
			logger.V(2).Infof("daemonSet %v started upgrade", newDS.GetName()) //nolint:mnd
		} else {
			// check for a crashed canary pod
			canaryPods := r.getCanaryPods(cachedDaemonSet)
			for _, canary := range canaryPods {
				if canary != nil && util.PodIsCrashLooping(canary) {
					r.recorder.Eventf(cachedDaemonSet, corev1.EventTypeWarning,
						failedUpdateDaemonSetReason, "daemonSet %v rollout failed", cachedDaemonSet.Name)
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
				logger.V(2).Infof("failed to update daemonset %s: %+v", newDS.Name, newDS) //nolint:mnd
				return false, fmt.Errorf("unable to update daemonset %s: %v", newDS.Name, err), CanaryUpgradeStatusFailed
			}
			logger.V(2).Infof("daemonSet %v updated", newDS.GetName()) //nolint:mnd
			status = CanaryUpgradeStatusUpgradingDaemonSet
		} else {
			logger.V(4).Infof("waiting for all pods of daemonSet %v to be ready", newDS.GetName()) //nolint:mnd
			status = CanaryUpgradeStatusWaitingDaemonSetRollout
		}
		done = false
	case updatedAndReadyPods > 0 && updatedAndReadyPods == desiredReadyPods:

		// rollout has completed and all virt-handlers are ready
		if !daemonHasDefaultRolloutStrategy(cachedDaemonSet) {
			// revert maxUnavailable to default value
			setMaxUnavailable(newDS, daemonSetDefaultMaxUnavailable)
			var err error
			newDS, err = r.patchDaemonSet(cachedDaemonSet, newDS)
			if err != nil {
				return false, err, CanaryUpgradeStatusFailed
			}
			logger.V(2).Infof("daemonSet %v updated back to default", newDS.GetName()) //nolint:mnd
			SetGeneration(&r.kv.Status.Generations, newDS)
			return false, nil, CanaryUpgradeStatusWaitingDaemonSetRollout
		}

		if supportsTLS(cachedDaemonSet) {
			if !hasTLS(cachedDaemonSet) {
				insertTLS(newDS)
				_, err := r.patchDaemonSet(cachedDaemonSet, newDS)
				logger.V(2).Infof("daemonSet %v updated to default CN TLS", newDS.GetName()) //nolint:mnd
				SetGeneration(&r.kv.Status.Generations, newDS)
				return false, err, CanaryUpgradeStatusWaitingDaemonSetRollout
			}
			if hasCertificateSecret(&newDS.Spec.Template.Spec, components.VirtHandlerCertSecretName) {
				unattachCertificateSecret(&newDS.Spec.Template.Spec, components.VirtHandlerCertSecretName)
				var err error
				cachedDaemonSet, err = r.patchDaemonSet(cachedDaemonSet, newDS)
				if err != nil {
					return false, err, CanaryUpgradeStatusFailed
				}
				logger.V(2).Infof("daemonSet %v updated to secure certificates", newDS.GetName()) //nolint:mnd
			}
		}

		SetGeneration(&r.kv.Status.Generations, cachedDaemonSet)
		logger.V(2).Infof("daemonSet %v is ready", newDS.GetName()) //nolint:mnd
		done, status = true, CanaryUpgradeStatusSuccessful
	}
	return done, nil, status
}

func supportsTLS(daemonSet *appsv1.DaemonSet) bool {
	if daemonSet.Labels == nil {
		return false
	}
	value, ok := daemonSet.Labels[components.SupportsMigrationCNsValidation]
	return ok && value == trueString
}

func insertTLS(daemonSet *appsv1.DaemonSet) {
	daemonSet.Spec.Template.Spec.Containers[0].Args = append(
		daemonSet.Spec.Template.Spec.Containers[0].Args,
		"--migration-cn-types", "migration",
	)
}

func hasTLS(daemonSet *appsv1.DaemonSet) bool {
	container := &daemonSet.Spec.Template.Spec.Containers[0]
	for _, arg := range container.Args {
		if strings.Contains(arg, "migration-cn-types") {
			return true
		}
	}
	return false
}

func hasCertificateSecret(spec *corev1.PodSpec, secretName string) bool {
	for _, volume := range spec.Volumes {
		if volume.Name == secretName {
			return true
		}
	}
	return false
}

func unattachCertificateSecret(spec *corev1.PodSpec, secretName string) {
	newVolumes := []corev1.Volume{}
	for _, volume := range spec.Volumes {
		if volume.Name != secretName {
			newVolumes = append(newVolumes, volume)
		}
	}
	spec.Volumes = newVolumes
	newVolumeMounts := []corev1.VolumeMount{}
	for _, volumeMount := range spec.Containers[0].VolumeMounts {
		if volumeMount.Name != secretName {
			newVolumeMounts = append(newVolumeMounts, volumeMount)
		}
	}
	spec.Containers[0].VolumeMounts = newVolumeMounts
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
	placement.InjectPlacementMetadata(kv.Spec.Workloads, &daemonSet.Spec.Template.Spec, placement.AnyNode)

	if daemonSet.GetName() == "virt-handler" {
		setMaxDevices(r.kv, daemonSet)
	}

	var cachedDaemonSet *appsv1.DaemonSet
	obj, exists, _ := r.stores.DaemonSetCache.Get(daemonSet)

	if !exists {
		r.expectations.DaemonSet.RaiseExpectations(r.kvKey, 1, 0)
		if supportsTLS(daemonSet) && !hasTLS(daemonSet) {
			insertTLS(daemonSet)
			unattachCertificateSecret(&daemonSet.Spec.Template.Spec, components.VirtHandlerCertSecretName)
		}

		origDaemonSet := daemonSet
		var createErr error
		daemonSet, createErr = apps.DaemonSets(kv.Namespace).Create(context.Background(), daemonSet, metav1.CreateOptions{})
		if createErr != nil {
			r.expectations.DaemonSet.LowerExpectations(r.kvKey, 1, 0)
			log.Log.V(2).Infof("failed to create daemonset %s: %+v", origDaemonSet.Name, origDaemonSet) //nolint:mnd
			return false, fmt.Errorf("unable to create daemonset %s: %v", origDaemonSet.Name, createErr)
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
		log.Log.V(4).Infof("daemonset %v is up-to-date", daemonSet.GetName()) //nolint:mnd
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
		origPDB := podDisruptionBudget
		var createErr error
		podDisruptionBudget, createErr = pdbClient.Create(context.Background(), podDisruptionBudget, metav1.CreateOptions{})
		if createErr != nil {
			r.expectations.PodDisruptionBudget.LowerExpectations(r.kvKey, 1, 0)
			log.Log.V(2).Infof("failed to create poddisruptionbudget %s: %+v", origPDB.Name, origPDB) //nolint:mnd
			return fmt.Errorf("unable to create poddisruptionbudget %s: %v", origPDB.Name, createErr)
		}
		log.Log.V(2).Infof("poddisruptionbudget %v created", podDisruptionBudget.GetName()) //nolint:mnd
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
		log.Log.V(4).Infof("poddisruptionbudget %v is up-to-date", cachedPodDisruptionBudget.GetName()) //nolint:mnd
		return nil
	}

	patchBytes, err := patch.New(
		getPatchWithObjectMetaAndSpec(
			[]patch.PatchOption{},
			&podDisruptionBudget.ObjectMeta,
			podDisruptionBudget.Spec,
		)...,
	).GeneratePayload()
	if err != nil {
		return err
	}

	prePatchPDB := podDisruptionBudget
	podDisruptionBudget, err = pdbClient.Patch(
		context.Background(), podDisruptionBudget.Name,
		types.JSONPatchType, patchBytes, metav1.PatchOptions{},
	)
	if err != nil {
		log.Log.V(2).Infof("failed to patch/delete poddisruptionbudget %s: %+v", prePatchPDB.Name, prePatchPDB) //nolint:mnd
		return fmt.Errorf("unable to patch/delete poddisruptionbudget %s: %v", prePatchPDB.Name, err)
	}

	SetGeneration(&kv.Status.Generations, podDisruptionBudget)
	log.Log.V(2).Infof("poddisruptionbudget %v patched", podDisruptionBudget.GetName()) //nolint:mnd

	return nil
}

func getDesiredAPIReplicas(clientset kubecli.KubevirtClient) (replicas int32, err error) {
	nodeList, err := clientset.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", v1.NodeSchedulable, trueString),
	})
	if err != nil {
		return 0, fmt.Errorf("failed to get number of nodes to determine virt-api replicas: %v", err)
	}

	nodesCount := int32(min(len(nodeList.Items), math.MaxInt32)) // #nosec G115 -- clamped to max int32
	// This is a simple heuristic to achieve basic scalability so we could be running on large clusters.
	// From recent experiments we know that for a 100 nodes cluster, 9 virt-api replicas are enough.
	// This heuristic is not accurate. It could, and should, be replaced by something more sophisticated and refined
	// in the future.

	if nodesCount == 1 {
		return 1, nil
	}

	const minReplicas = 2

	replicas = nodesCount / 10
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
			log.Log.Warningf("%s", err.Error())
			continue
		}
		for _, operation := range decodedPatch {
			path, err := operation.Path()
			if err != nil {
				log.Log.Warningf("%s", err.Error())
				continue
			}
			op := operation.Kind()
			if path == "/spec/replicas" && op == replaceOp {
				return true
			}
		}
	}
	return false
}
