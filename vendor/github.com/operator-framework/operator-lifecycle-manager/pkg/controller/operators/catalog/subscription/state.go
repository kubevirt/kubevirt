package subscription

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	clientv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/client/clientset/versioned/typed/operators/v1alpha1"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/comparison"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/kubestate"
)

// SubscriptionState describes subscription states.
type SubscriptionState interface {
	kubestate.State

	isSubscriptionState()
	setSubscription(*v1alpha1.Subscription)

	Subscription() *v1alpha1.Subscription
	Add() SubscriptionExistsState
	Update() SubscriptionExistsState
	Delete() SubscriptionDeletedState
}

// SubscriptionExistsState describes subscription states in which the subscription exists on the cluster.
type SubscriptionExistsState interface {
	SubscriptionState

	isSubscriptionExistsState()
}

// SubscriptionAddedState describes subscription states in which the subscription was added to cluster.
type SubscriptionAddedState interface {
	SubscriptionExistsState

	isSubscriptionAddedState()
}

// SubscriptionUpdatedState describes subscription states in which the subscription was updated in the cluster.
type SubscriptionUpdatedState interface {
	SubscriptionExistsState

	isSubscriptionUpdatedState()
}

// SubscriptionDeletedState describes subscription states in which the subscription no longer exists and was deleted from the cluster.
type SubscriptionDeletedState interface {
	SubscriptionState

	isSubscriptionDeletedState()
}

// CatalogHealthState describes subscription states that represent a subscription with respect to catalog health.
type CatalogHealthState interface {
	SubscriptionExistsState

	isCatalogHealthState()

	// UpdateHealth transitions the CatalogHealthState to another CatalogHealthState based on the given subscription catalog health.
	// The state's underlying subscription may be updated on the cluster. If the subscription is updated, the resulting state will contain the updated version.
	UpdateHealth(now *metav1.Time, client clientv1alpha1.SubscriptionInterface, health ...v1alpha1.SubscriptionCatalogHealth) (CatalogHealthState, error)
}

// CatalogHealthKnownState describes subscription states in which all relevant catalog health is known.
type CatalogHealthKnownState interface {
	CatalogHealthState

	isCatalogHealthKnownState()
}

// CatalogHealthyState describes subscription states in which all relevant catalogs are known to be healthy.
type CatalogHealthyState interface {
	CatalogHealthKnownState

	isCatalogHealthyState()
}

// CatalogUnhealthyState describes subscription states in which at least one relevant catalog is known to be unhealthy.
type CatalogUnhealthyState interface {
	CatalogHealthKnownState

	isCatalogUnhealthyState()
}

// InstallPlanState describes Subscription states with respect to an InstallPlan.
type InstallPlanState interface {
	SubscriptionExistsState

	isInstallPlanState()

	CheckReference() InstallPlanState
}

type NoInstallPlanReferencedState interface {
	InstallPlanState

	isNoInstallPlanReferencedState()
}

type InstallPlanReferencedState interface {
	InstallPlanState

	isInstallPlanReferencedState()

	InstallPlanNotFound(now *metav1.Time, client clientv1alpha1.SubscriptionInterface) (InstallPlanReferencedState, error)

	CheckInstallPlanStatus(now *metav1.Time, client clientv1alpha1.SubscriptionInterface, status *v1alpha1.InstallPlanStatus) (InstallPlanReferencedState, error)
}

type InstallPlanKnownState interface {
	InstallPlanReferencedState

	isInstallPlanKnownState()
}

type InstallPlanMissingState interface {
	InstallPlanKnownState

	isInstallPlanMissingState()
}

type InstallPlanPendingState interface {
	InstallPlanKnownState

	isInstallPlanPendingState()
}

type InstallPlanFailedState interface {
	InstallPlanKnownState

	isInstallPlanFailedState()
}

type InstallPlanInstalledState interface {
	InstallPlanKnownState

	isInstallPlanInstalledState()
}

type subscriptionState struct {
	kubestate.State

	subscription *v1alpha1.Subscription
}

func (s *subscriptionState) isSubscriptionState() {}

func (s *subscriptionState) setSubscription(sub *v1alpha1.Subscription) {
	s.subscription = sub
}

func (s *subscriptionState) Subscription() *v1alpha1.Subscription {
	return s.subscription
}

func (s *subscriptionState) Add() SubscriptionExistsState {
	return &subscriptionAddedState{
		SubscriptionExistsState: &subscriptionExistsState{
			SubscriptionState: s,
		},
	}
}

func (s *subscriptionState) Update() SubscriptionExistsState {
	return &subscriptionUpdatedState{
		SubscriptionExistsState: &subscriptionExistsState{
			SubscriptionState: s,
		},
	}
}

func (s *subscriptionState) Delete() SubscriptionDeletedState {
	return &subscriptionDeletedState{
		SubscriptionState: s,
	}
}

func NewSubscriptionState(sub *v1alpha1.Subscription) SubscriptionState {
	return &subscriptionState{
		State:        kubestate.NewState(),
		subscription: sub,
	}
}

type subscriptionExistsState struct {
	SubscriptionState
}

func (*subscriptionExistsState) isSubscriptionExistsState() {}

type subscriptionAddedState struct {
	SubscriptionExistsState
}

func (c *subscriptionAddedState) isSubscriptionAddedState() {}

type subscriptionUpdatedState struct {
	SubscriptionExistsState
}

func (c *subscriptionUpdatedState) isSubscriptionUpdatedState() {}

type subscriptionDeletedState struct {
	SubscriptionState
}

func (c *subscriptionDeletedState) isSubscriptionDeletedState() {}

type catalogHealthState struct {
	SubscriptionExistsState
}

func (c *catalogHealthState) isCatalogHealthState() {}

func (c *catalogHealthState) UpdateHealth(now *metav1.Time, client clientv1alpha1.SubscriptionInterface, catalogHealth ...v1alpha1.SubscriptionCatalogHealth) (CatalogHealthState, error) {
	in := c.Subscription()
	out := in.DeepCopy()

	healthSet := make(map[types.UID]v1alpha1.SubscriptionCatalogHealth, len(catalogHealth))
	healthy := true
	missingTargeted := true

	cond := out.Status.GetCondition(v1alpha1.SubscriptionCatalogSourcesUnhealthy)
	for _, h := range catalogHealth {
		ref := h.CatalogSourceRef
		healthSet[ref.UID] = h
		healthy = healthy && h.Healthy

		if ref.Namespace == in.Spec.CatalogSourceNamespace && ref.Name == in.Spec.CatalogSource {
			missingTargeted = false
			if !h.Healthy {
				cond.Message = fmt.Sprintf("targeted catalogsource %s/%s unhealthy", ref.Namespace, ref.Name)
			}
		}
	}

	var known CatalogHealthKnownState
	switch {
	case missingTargeted:
		healthy = false
		cond.Message = fmt.Sprintf("targeted catalogsource %s/%s missing", in.Spec.CatalogSourceNamespace, in.Spec.CatalogSource)
		fallthrough
	case !healthy:
		cond.Status = corev1.ConditionTrue
		cond.Reason = v1alpha1.UnhealthyCatalogSourceFound
		known = &catalogUnhealthyState{
			CatalogHealthKnownState: &catalogHealthKnownState{
				CatalogHealthState: c,
			},
		}
	default:
		cond.Status = corev1.ConditionFalse
		cond.Reason = v1alpha1.AllCatalogSourcesHealthy
		cond.Message = "all available catalogsources are healthy"
		known = &catalogHealthyState{
			CatalogHealthKnownState: &catalogHealthKnownState{
				CatalogHealthState: c,
			},
		}
	}

	// Check for changes in CatalogHealth
	update := true
	switch numNew, numOld := len(healthSet), len(in.Status.CatalogHealth); {
	case numNew > numOld:
		cond.Reason = v1alpha1.CatalogSourcesAdded
	case numNew < numOld:
		cond.Reason = v1alpha1.CatalogSourcesDeleted
	case numNew == 0 && numNew == numOld:
		healthy = false
		cond.Reason = v1alpha1.NoCatalogSourcesFound
		cond.Message = "dependency resolution requires at least one catalogsource"
	case numNew == numOld:
		// Check against existing subscription
		for _, oldHealth := range in.Status.CatalogHealth {
			uid := oldHealth.CatalogSourceRef.UID
			if newHealth, ok := healthSet[uid]; !ok || !newHealth.Equals(oldHealth) {
				cond.Reason = v1alpha1.CatalogSourcesUpdated
				break
			}
		}

		fallthrough
	default:
		update = false
	}

	if !update && cond.Equals(in.Status.GetCondition(v1alpha1.SubscriptionCatalogSourcesUnhealthy)) {
		// Nothing to do, transition to self
		return known, nil
	}

	cond.LastTransitionTime = now
	out.Status.LastUpdated = *now
	out.Status.SetCondition(cond)
	out.Status.CatalogHealth = catalogHealth

	updated, err := client.UpdateStatus(out)
	if err != nil {
		// Error occurred, transition to self
		return c, err
	}

	// Inject updated subscription into the state
	known.setSubscription(updated)

	return known, nil
}

func NewCatalogHealthState(s SubscriptionExistsState) CatalogHealthState {
	return &catalogHealthState{
		SubscriptionExistsState: s,
	}
}

type catalogHealthKnownState struct {
	CatalogHealthState
}

func (c *catalogHealthKnownState) isCatalogHealthKnownState() {}

func (c *catalogHealthKnownState) CatalogHealth() []v1alpha1.SubscriptionCatalogHealth {
	return c.Subscription().Status.CatalogHealth
}

type catalogHealthyState struct {
	CatalogHealthKnownState
}

func (c *catalogHealthyState) isCatalogHealthyState() {}

type catalogUnhealthyState struct {
	CatalogHealthKnownState
}

func (c *catalogUnhealthyState) isCatalogUnhealthyState() {}

type installPlanState struct {
	SubscriptionExistsState
}

func (i *installPlanState) isInstallPlanState() {}

func (i *installPlanState) CheckReference() InstallPlanState {
	if i.Subscription().Status.InstallPlanRef != nil {
		return &installPlanReferencedState{
			InstallPlanState: i,
		}
	}

	return &noInstallPlanReferencedState{
		InstallPlanState: i,
	}
}

func newInstallPlanState(s SubscriptionExistsState) InstallPlanState {
	return &installPlanState{
		SubscriptionExistsState: s,
	}
}

type noInstallPlanReferencedState struct {
	InstallPlanState
}

func (n *noInstallPlanReferencedState) isNoInstallPlanReferencedState() {}

type installPlanReferencedState struct {
	InstallPlanState
}

func (i *installPlanReferencedState) isInstallPlanReferencedState() {}

var hashEqual = comparison.NewHashEqualitor()

func (i *installPlanReferencedState) InstallPlanNotFound(now *metav1.Time, client clientv1alpha1.SubscriptionInterface) (InstallPlanReferencedState, error) {
	in := i.Subscription()
	out := in.DeepCopy()

	// Remove pending and failed conditions
	out.Status.RemoveConditions(v1alpha1.SubscriptionInstallPlanPending, v1alpha1.SubscriptionInstallPlanFailed)

	// Set missing condition to true
	missingCond := out.Status.GetCondition(v1alpha1.SubscriptionInstallPlanMissing)
	missingCond.Status = corev1.ConditionTrue
	missingCond.Reason = v1alpha1.ReferencedInstallPlanNotFound
	missingCond.LastTransitionTime = now
	out.Status.SetCondition(missingCond)

	// Build missing state
	missingState := &installPlanMissingState{
		InstallPlanKnownState: &installPlanKnownState{
			InstallPlanReferencedState: i,
		},
	}

	// Bail out if the conditions haven't changed (using select fields included in a hash)
	if hashEqual(out.Status.Conditions, in.Status.Conditions) {
		return missingState, nil
	}

	// Update the Subscription
	out.Status.LastUpdated = *now
	updated, err := client.UpdateStatus(out)
	if err != nil {
		return i, err
	}

	// Stuff updated Subscription into state
	missingState.setSubscription(updated)

	return missingState, nil
}

func (i *installPlanReferencedState) CheckInstallPlanStatus(now *metav1.Time, client clientv1alpha1.SubscriptionInterface, status *v1alpha1.InstallPlanStatus) (InstallPlanReferencedState, error) {
	in := i.Subscription()
	out := in.DeepCopy()

	// Remove missing, pending, and failed conditions
	out.Status.RemoveConditions(v1alpha1.SubscriptionInstallPlanMissing, v1alpha1.SubscriptionInstallPlanPending, v1alpha1.SubscriptionInstallPlanFailed)

	// Build and set the InstallPlan condition, if any
	cond := v1alpha1.SubscriptionCondition{
		Status:             corev1.ConditionUnknown,
		LastTransitionTime: now,
	}

	// TODO: Use InstallPlan conditions instead of phases
	// Get the status of the InstallPlan and create the appropriate condition and state
	var known InstallPlanKnownState
	switch phase := status.Phase; phase {
	case v1alpha1.InstallPlanPhaseNone:
		// Set reason and let the following case fill out the pending condition
		cond.Reason = v1alpha1.InstallPlanNotYetReconciled
		fallthrough
	case v1alpha1.InstallPlanPhasePlanning, v1alpha1.InstallPlanPhaseInstalling, v1alpha1.InstallPlanPhaseRequiresApproval:
		if cond.Reason == "" {
			cond.Reason = string(phase)
		}

		cond.Type = v1alpha1.SubscriptionInstallPlanPending
		cond.Status = corev1.ConditionTrue
		out.Status.SetCondition(cond)

		// Build pending state
		known = &installPlanPendingState{
			InstallPlanKnownState: &installPlanKnownState{
				InstallPlanReferencedState: i,
			},
		}
	case v1alpha1.InstallPlanPhaseFailed:
		// Attempt to project reason from failed InstallPlan condition
		if installedCond := status.GetCondition(v1alpha1.InstallPlanInstalled); installedCond.Status == corev1.ConditionFalse {
			cond.Reason = string(installedCond.Reason)
		} else {
			cond.Reason = v1alpha1.InstallPlanFailed
		}

		cond.Type = v1alpha1.SubscriptionInstallPlanFailed
		cond.Status = corev1.ConditionTrue
		out.Status.SetCondition(cond)

		// Build failed state
		known = &installPlanFailedState{
			InstallPlanKnownState: &installPlanKnownState{
				InstallPlanReferencedState: i,
			},
		}
	default:
		// Build installed state
		known = &installPlanInstalledState{
			InstallPlanKnownState: &installPlanKnownState{
				InstallPlanReferencedState: i,
			},
		}
	}

	// Bail out if the conditions haven't changed (using select fields included in a hash)
	if hashEqual(out.Status.Conditions, in.Status.Conditions) {
		return known, nil
	}

	// Update the Subscription
	out.Status.LastUpdated = *now
	updated, err := client.UpdateStatus(out)
	if err != nil {
		return i, err
	}

	// Stuff updated Subscription into state
	known.setSubscription(updated)

	return known, nil
}

type installPlanKnownState struct {
	InstallPlanReferencedState
}

func (i *installPlanKnownState) isInstallPlanKnownState() {}

type installPlanMissingState struct {
	InstallPlanKnownState
}

func (i *installPlanMissingState) isInstallPlanMissingState() {}

type installPlanPendingState struct {
	InstallPlanKnownState
}

func (i *installPlanPendingState) isInstallPlanPendingState() {}

type installPlanFailedState struct {
	InstallPlanKnownState
}

func (i *installPlanFailedState) isInstallPlanFailedState() {}

type installPlanInstalledState struct {
	InstallPlanKnownState
}

func (i *installPlanInstalledState) isInstallPlanInstalledState() {}
