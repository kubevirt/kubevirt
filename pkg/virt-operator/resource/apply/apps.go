package apply

import (
	"context"
	"fmt"
	"slices"
	"strings"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/openshift/library-go/pkg/operator/resource/resourcemerge"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/placement"
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
	placement.InjectPlacementMetadata(kv.Spec.Infra, &deployment.Spec.Template.Spec, placement.RequireControlPlanePreferNonWorker)

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
			log.Log.Object(deployment).Warningf("%s", err.Error())
		} else {
			deployment.Spec.Replicas = pointer.P(replicas)
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

	deployment, err = apps.Deployments(kv.Namespace).Patch(context.Background(), deployment.Name, types.JSONPatchType, ops, metav1.PatchOptions{})
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
	return patch.New(
		getPatchWithObjectMetaAndSpec([]patch.PatchOption{
			patch.WithTest("/metadata/generation", oldDs.ObjectMeta.Generation),
		},
			&newDs.ObjectMeta, newDs.Spec)...).GeneratePayload()
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

func findContainerByName(containers []corev1.Container, name string) *corev1.Container {
	for i := range containers {
		if containers[i].Name == name {
			return &containers[i]
		}
	}
	return nil
}

// sortedVolumeMounts returns a copy of the volume mounts slice sorted by name.
// This ensures ordering differences don't cause false positive mismatches.
func sortedVolumeMounts(mounts []corev1.VolumeMount) []corev1.VolumeMount {
	if len(mounts) == 0 {
		return nil
	}
	sorted := slices.Clone(mounts)
	slices.SortFunc(sorted, func(a, b corev1.VolumeMount) int {
		if a.Name < b.Name {
			return -1
		}
		if a.Name > b.Name {
			return 1
		}
		return 0
	})
	return sorted
}

// sortedEnvVars returns a copy of the env vars slice sorted by name.
// This ensures ordering differences don't cause false positive mismatches.
func sortedEnvVars(envs []corev1.EnvVar) []corev1.EnvVar {
	if len(envs) == 0 {
		return nil
	}
	sorted := slices.Clone(envs)
	slices.SortFunc(sorted, func(a, b corev1.EnvVar) int {
		if a.Name < b.Name {
			return -1
		}
		if a.Name > b.Name {
			return 1
		}
		return 0
	})
	return sorted
}

// containerSpecMismatch compares security-relevant fields of two containers.
// Returns an empty string if containers match, or the name of the first mismatched field.
// Excludes Args which may be modified by TLS migration logic.
// Pod-level Volumes are not checked here; only container-level VolumeMounts are compared.
func containerSpecMismatch(cached, desired *corev1.Container) string {
	if cached.Image != desired.Image {
		return "Image"
	}
	if !equality.Semantic.DeepEqual(cached.Command, desired.Command) {
		return "Command"
	}
	if !equality.Semantic.DeepEqual(cached.Resources, desired.Resources) {
		return "Resources"
	}
	// Sort volume mounts by name before comparison to avoid false positives from ordering differences
	if !equality.Semantic.DeepEqual(sortedVolumeMounts(cached.VolumeMounts), sortedVolumeMounts(desired.VolumeMounts)) {
		return "VolumeMounts"
	}
	if !equality.Semantic.DeepEqual(cached.SecurityContext, desired.SecurityContext) {
		return "SecurityContext"
	}
	// Sort env vars by name before comparison to avoid false positives from ordering differences
	if !equality.Semantic.DeepEqual(sortedEnvVars(cached.Env), sortedEnvVars(desired.Env)) {
		return "Env"
	}
	return ""
}

// containerNames returns a sorted list of container names
func containerNames(containers []corev1.Container) []string {
	names := make([]string, len(containers))
	for i, c := range containers {
		names[i] = c.Name
	}
	slices.Sort(names)
	return names
}

// containersSpecMismatch compares containers by name and checks security-relevant fields.
// Returns an empty string if all containers match, or a description of the mismatch.
func containersSpecMismatch(cached, desired []corev1.Container) string {
	cachedNames := containerNames(cached)
	desiredNames := containerNames(desired)

	// Find extra containers in cached (present in cached but not in desired)
	var extraInCached []string
	for _, name := range cachedNames {
		if findContainerByName(desired, name) == nil {
			extraInCached = append(extraInCached, name)
		}
	}

	// Find missing containers (present in desired but not in cached)
	var missingInCached []string
	for _, name := range desiredNames {
		if findContainerByName(cached, name) == nil {
			missingInCached = append(missingInCached, name)
		}
	}

	// Report container set differences
	if len(extraInCached) > 0 || len(missingInCached) > 0 {
		var parts []string
		if len(extraInCached) > 0 {
			parts = append(parts, fmt.Sprintf("extra: %v", extraInCached))
		}
		if len(missingInCached) > 0 {
			parts = append(parts, fmt.Sprintf("missing: %v", missingInCached))
		}
		return fmt.Sprintf("container set mismatch (%s)", strings.Join(parts, ", "))
	}

	// All containers present, check for field-level differences
	for _, desiredContainer := range desired {
		cachedContainer := findContainerByName(cached, desiredContainer.Name)
		if mismatch := containerSpecMismatch(cachedContainer, &desiredContainer); mismatch != "" {
			return fmt.Sprintf("container %q field %s", desiredContainer.Name, mismatch)
		}
	}
	return ""
}

// daemonSetCoreSpecMismatch compares the core pod spec fields of two DaemonSets
// to detect unauthorized modifications. Returns an empty string if specs match,
// or a description of the mismatch.
// Excludes Args which are intentionally modified during TLS migration.
// Pod-level Volumes are not checked as they may also be modified during TLS migration.
func daemonSetCoreSpecMismatch(cached, desired *appsv1.DaemonSet) (bool, string) {
	cachedSpec := &cached.Spec.Template.Spec
	desiredSpec := &desired.Spec.Template.Spec

	if mismatch := containersSpecMismatch(cachedSpec.Containers, desiredSpec.Containers); mismatch != "" {
		return true, mismatch
	}
	if mismatch := containersSpecMismatch(cachedSpec.InitContainers, desiredSpec.InitContainers); mismatch != "" {
		return true, fmt.Sprintf("initContainer: %s", mismatch)
	}
	return false, ""
}

func (r *Reconciler) processCanaryUpgrade(cachedDaemonSet, newDS *appsv1.DaemonSet, forceUpdate bool) (bool, error, CanaryUpgradeStatus) {
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
	log := log.Log.With("resource", fmt.Sprintf("ds/%s", cachedDaemonSet.Name))

	isVersionUpToDate := util.DaemonSetIsUpToDate(r.kv, cachedDaemonSet)

	// Compare the actual DaemonSet spec to detect unexpected modifications.
	// Only log a warning when the version is up-to-date but the spec differs,
	// indicating someone has manually modified the DaemonSet. During normal
	// version upgrades, spec differences are expected and should not be logged.
	specMismatch, specMismatchField := daemonSetCoreSpecMismatch(cachedDaemonSet, newDS)
	if specMismatch && isVersionUpToDate {
		log.Warningf("detected spec modification in daemonset %s (%s), will revert to expected configuration", cachedDaemonSet.Name, specMismatchField)
	}

	isDaemonSetUpdated := isVersionUpToDate && !specMismatch && !forceUpdate
	desiredReadyPods := cachedDaemonSet.Status.DesiredNumberScheduled

	// Use the DaemonSet's UpdatedNumberScheduled status to determine how many pods
	// have actually been rolled out with the new template. This is more accurate than
	// howManyUpdatedAndReadyPods which only checks version annotations and would
	// incorrectly count all pods as "updated" for CustomizeComponents changes that
	// don't change the version.
	//
	// We take the minimum of UpdatedNumberScheduled and the pods we count as ready
	// to get an approximation of "updated AND ready" pods.
	if isDaemonSetUpdated {
		// All conditions met, count pods that have correct version AND are ready
		updatedAndReadyPods = r.howManyUpdatedAndReadyPods(cachedDaemonSet)
		// But cap this by how many pods the DaemonSet controller reports as updated.
		// This handles CustomizeComponents changes where version doesn't change but
		// pods still need to be rolled out with the new template.
		updatedAndReadyPods = min(updatedAndReadyPods, cachedDaemonSet.Status.UpdatedNumberScheduled)
	}

	switch {
	case updatedAndReadyPods == 0:
		if !isDaemonSetUpdated {
			// start canary upgrade (or revert unauthorized modification)
			setMaxUnavailable(newDS, daemonSetDefaultMaxUnavailable)
			_, err := r.patchDaemonSet(cachedDaemonSet, newDS)
			if err != nil {
				return false, fmt.Errorf("unable to start canary upgrade for daemonset %+v: %v", newDS, err), CanaryUpgradeStatusFailed
			}
			log.V(2).Infof("daemonSet %v started upgrade", newDS.GetName())
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
			log.V(2).Infof("daemonSet %v updated", newDS.GetName())
			status = CanaryUpgradeStatusUpgradingDaemonSet
		} else {
			log.V(4).Infof("waiting for all pods of daemonSet %v to be ready", newDS.GetName())
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
			log.V(2).Infof("daemonSet %v updated back to default", newDS.GetName())
			SetGeneration(&r.kv.Status.Generations, newDS)
			return false, nil, CanaryUpgradeStatusWaitingDaemonSetRollout
		}

		if supportsTLS(cachedDaemonSet) {
			if !hasTLS(cachedDaemonSet) {
				insertTLS(newDS)
				_, err := r.patchDaemonSet(cachedDaemonSet, newDS)
				log.V(2).Infof("daemonSet %v updated to default CN TLS", newDS.GetName())
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
				log.V(2).Infof("daemonSet %v updated to secure certificates", newDS.GetName())
			}
		}

		SetGeneration(&r.kv.Status.Generations, cachedDaemonSet)
		log.V(2).Infof("daemonSet %v is ready", newDS.GetName())
		done, status = true, CanaryUpgradeStatusSuccessful
	}
	return done, nil, status
}

func supportsTLS(daemonSet *appsv1.DaemonSet) bool {
	if daemonSet.Labels == nil {
		return false
	}
	value, ok := daemonSet.Labels[components.SupportsMigrationCNsValidation]
	return ok && value == "true"
}

func insertTLS(daemonSet *appsv1.DaemonSet) {
	daemonSet.Spec.Template.Spec.Containers[0].Args = append(daemonSet.Spec.Template.Spec.Containers[0].Args, "--migration-cn-types", "migration")
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

	patchBytes, err := patch.New(getPatchWithObjectMetaAndSpec([]patch.PatchOption{}, &podDisruptionBudget.ObjectMeta, podDisruptionBudget.Spec)...).GeneratePayload()
	if err != nil {
		return err
	}

	podDisruptionBudget, err = pdbClient.Patch(context.Background(), podDisruptionBudget.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("unable to patch/delete poddisruptionbudget %+v: %v", podDisruptionBudget, err)
	}

	SetGeneration(&kv.Status.Generations, podDisruptionBudget)
	log.Log.V(2).Infof("poddisruptionbudget %v patched", podDisruptionBudget.GetName())

	return nil
}

func getDesiredApiReplicas(clientset kubecli.KubevirtClient) (replicas int32, err error) {
	nodeList, err := clientset.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", v1.NodeSchedulable, "true"),
	})
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
			if path == "/spec/replicas" && op == "replace" {
				return true
			}
		}
	}
	return false
}
