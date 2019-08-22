package install

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	rbac "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/util/diff"

	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/wrappers"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/ownerutil"
)

const (
	InstallStrategyNameDeployment = "deployment"
)

// StrategyDeploymentPermissions describe the rbac rules and service account needed by the install strategy
type StrategyDeploymentPermissions struct {
	ServiceAccountName string            `json:"serviceAccountName"`
	Rules              []rbac.PolicyRule `json:"rules"`
}

// StrategyDeploymentSpec contains the name and spec for the deployment ALM should create
type StrategyDeploymentSpec struct {
	Name string                `json:"name"`
	Spec appsv1.DeploymentSpec `json:"spec"`
}

// StrategyDetailsDeployment represents the parsed details of a Deployment
// InstallStrategy.
type StrategyDetailsDeployment struct {
	DeploymentSpecs    []StrategyDeploymentSpec        `json:"deployments"`
	Permissions        []StrategyDeploymentPermissions `json:"permissions,omitempty"`
	ClusterPermissions []StrategyDeploymentPermissions `json:"clusterPermissions,omitempty"`
}

type StrategyDeploymentInstaller struct {
	strategyClient      wrappers.InstallStrategyDeploymentInterface
	owner               ownerutil.Owner
	previousStrategy    Strategy
	templateAnnotations map[string]string
	initializers        DeploymentInitializerFuncChain
}

func (d *StrategyDetailsDeployment) GetStrategyName() string {
	return InstallStrategyNameDeployment
}

var _ Strategy = &StrategyDetailsDeployment{}
var _ StrategyInstaller = &StrategyDeploymentInstaller{}

// DeploymentInitializerFunc takes a deployment object and appropriately
// initializes it for install.
//
// Before a deployment is created on the cluster, we can run a series of
// initializer functions that will properly initialize the deployment object.
type DeploymentInitializerFunc func(deployment *appsv1.Deployment) error

// DeploymentInitializerFuncChain defines a chain of DeploymentInitializerFunc.
type DeploymentInitializerFuncChain []DeploymentInitializerFunc

// Apply runs series of initializer functions that will properly initialize
// the deployment object.
func (c DeploymentInitializerFuncChain) Apply(deployment *appsv1.Deployment) (err error) {
	for _, initializer := range c {
		if initializer == nil {
			continue
		}

		if initializationErr := initializer(deployment); initializationErr != nil {
			err = initializationErr
			break
		}
	}

	return
}

type DeploymentInitializerFuncBuilder func(owner ownerutil.Owner) DeploymentInitializerFunc

func NewStrategyDeploymentInstaller(strategyClient wrappers.InstallStrategyDeploymentInterface, templateAnnotations map[string]string, owner ownerutil.Owner, previousStrategy Strategy, initializers DeploymentInitializerFuncChain) StrategyInstaller {
	return &StrategyDeploymentInstaller{
		strategyClient:      strategyClient,
		owner:               owner,
		previousStrategy:    previousStrategy,
		templateAnnotations: templateAnnotations,
		initializers:        initializers,
	}
}

func (i *StrategyDeploymentInstaller) installDeployments(deps []StrategyDeploymentSpec) error {
	for _, d := range deps {
		deployment, err := i.deploymentForSpec(d.Name, d.Spec)
		if err != nil {
			return err
		}

		if _, err := i.strategyClient.CreateOrUpdateDeployment(deployment); err != nil {
			return err
		}
	}

	return nil
}

func (i *StrategyDeploymentInstaller) deploymentForSpec(name string, spec appsv1.DeploymentSpec) (deployment *appsv1.Deployment, err error) {
	dep := &appsv1.Deployment{Spec: spec}
	dep.SetName(name)
	dep.SetNamespace(i.owner.GetNamespace())

	// Merge annotations (to avoid losing info from pod template)
	annotations := map[string]string{}
	for k, v := range i.templateAnnotations {
		annotations[k] = v
	}
	for k, v := range dep.Spec.Template.GetAnnotations() {
		annotations[k] = v
	}
	dep.Spec.Template.SetAnnotations(annotations)

	ownerutil.AddNonBlockingOwner(dep, i.owner)
	ownerutil.AddOwnerLabelsForKind(dep, i.owner, v1alpha1.ClusterServiceVersionKind)

	if applyErr := i.initializers.Apply(dep); applyErr != nil {
		err = applyErr
		return
	}

	deployment = dep
	return
}

func (i *StrategyDeploymentInstaller) cleanupPrevious(current *StrategyDetailsDeployment, previous *StrategyDetailsDeployment) error {
	previousDeploymentsMap := map[string]struct{}{}
	for _, d := range previous.DeploymentSpecs {
		previousDeploymentsMap[d.Name] = struct{}{}
	}
	for _, d := range current.DeploymentSpecs {
		delete(previousDeploymentsMap, d.Name)
	}
	log.Debugf("preparing to cleanup: %s", previousDeploymentsMap)
	// delete deployments in old strategy but not new
	var err error = nil
	for name := range previousDeploymentsMap {
		err = i.strategyClient.DeleteDeployment(name)
	}
	return err
}

func (i *StrategyDeploymentInstaller) Install(s Strategy) error {
	strategy, ok := s.(*StrategyDetailsDeployment)
	if !ok {
		return fmt.Errorf("attempted to install %s strategy with deployment installer", strategy.GetStrategyName())
	}

	if err := i.installDeployments(strategy.DeploymentSpecs); err != nil {
		return err
	}

	// Clean up orphaned deployments
	return i.cleanupOrphanedDeployments(strategy.DeploymentSpecs)
}

// CheckInstalled can return nil (installed), or errors
// Errors can indicate: some component missing (keep installing), unable to query (check again later), or unrecoverable (failed in a way we know we can't recover from)
func (i *StrategyDeploymentInstaller) CheckInstalled(s Strategy) (installed bool, err error) {
	strategy, ok := s.(*StrategyDetailsDeployment)
	if !ok {
		return false, StrategyError{Reason: StrategyErrReasonInvalidStrategy, Message: fmt.Sprintf("attempted to check %s strategy with deployment installer", strategy.GetStrategyName())}
	}

	// Check deployments
	if err := i.checkForDeployments(strategy.DeploymentSpecs); err != nil {
		return false, err
	}
	return true, nil
}

func (i *StrategyDeploymentInstaller) checkForDeployments(deploymentSpecs []StrategyDeploymentSpec) error {
	var depNames []string
	for _, dep := range deploymentSpecs {
		depNames = append(depNames, dep.Name)
	}

	// Check the owner is a CSV
	csv, ok := i.owner.(*v1alpha1.ClusterServiceVersion)
	if !ok {
		return StrategyError{Reason: StrategyErrReasonComponentMissing, Message: fmt.Sprintf("owner %s is not a CSV", i.owner.GetName())}
	}

	existingDeployments, err := i.strategyClient.FindAnyDeploymentsMatchingLabels(ownerutil.CSVOwnerSelector(csv))
	if err != nil {
		return StrategyError{Reason: StrategyErrReasonComponentMissing, Message: fmt.Sprintf("error querying existing deployments for CSV %s: %s", csv.GetName(), err)}
	}

	// compare deployments to see if any need to be created/updated
	existingMap := map[string]*appsv1.Deployment{}
	for _, d := range existingDeployments {
		existingMap[d.GetName()] = d
	}
	for _, spec := range deploymentSpecs {
		dep, exists := existingMap[spec.Name]
		if !exists {
			log.Debugf("missing deployment with name=%s", spec.Name)
			return StrategyError{Reason: StrategyErrReasonComponentMissing, Message: fmt.Sprintf("missing deployment with name=%s", spec.Name)}
		}
		reason, ready, err := DeploymentStatus(dep)
		if err != nil {
			log.Debugf("deployment %s not ready before timeout: %s", dep.Name, err.Error())
			return StrategyError{Reason: StrategyErrReasonTimeout, Message: fmt.Sprintf("deployment %s not ready before timeout: %s", dep.Name, err.Error())}
		}
		if !ready {
			return StrategyError{Reason: StrategyErrReasonWaiting, Message: fmt.Sprintf("waiting for deployment %s to become ready: %s", dep.Name, reason)}
		}

		// check annotations
		if len(i.templateAnnotations) > 0 && dep.Spec.Template.Annotations == nil {
			return StrategyError{Reason: StrategyErrReasonAnnotationsMissing, Message: fmt.Sprintf("no annotations found on deployment")}
		}
		for key, value := range i.templateAnnotations {
			if dep.Spec.Template.Annotations[key] != value {
				return StrategyError{Reason: StrategyErrReasonAnnotationsMissing, Message: fmt.Sprintf("annotations on deployment don't match. couldn't find %s: %s", key, value)}
			}
		}

		// check equality
		calculated, err := i.deploymentForSpec(spec.Name, spec.Spec)
		if err != nil {
			return err
		}

		if !i.equalDeployments(&calculated.Spec, &dep.Spec) {
			return StrategyError{Reason: StrategyErrDeploymentUpdated, Message: fmt.Sprintf("deployment changed, rolling update with patch: %s\n%#v\n%#v", diff.ObjectDiff(dep.Spec.Template.Spec, calculated.Spec.Template.Spec), calculated.Spec.Template.Spec, dep.Spec.Template.Spec)}
		}
	}
	return nil
}

func (i *StrategyDeploymentInstaller) equalDeployments(calculated, onCluster *appsv1.DeploymentSpec) bool {
	// ignore template annotations, OLM injects these elsewhere
	calculated.Template.Annotations = nil

	// DeepDerivative doesn't treat `0` ints as unset. Stripping them here means we miss changes to these values,
	// but we don't end up getting bitten by the defaulter for deployments.
	for i, c := range onCluster.Template.Spec.Containers {
		o := calculated.Template.Spec.Containers[i]
		if o.ReadinessProbe != nil {
			o.ReadinessProbe.InitialDelaySeconds = c.ReadinessProbe.InitialDelaySeconds
			o.ReadinessProbe.TimeoutSeconds = c.ReadinessProbe.TimeoutSeconds
			o.ReadinessProbe.PeriodSeconds = c.ReadinessProbe.PeriodSeconds
			o.ReadinessProbe.SuccessThreshold = c.ReadinessProbe.SuccessThreshold
			o.ReadinessProbe.FailureThreshold = c.ReadinessProbe.FailureThreshold
		}
		if o.LivenessProbe != nil {
			o.LivenessProbe.InitialDelaySeconds = c.LivenessProbe.InitialDelaySeconds
			o.LivenessProbe.TimeoutSeconds = c.LivenessProbe.TimeoutSeconds
			o.LivenessProbe.PeriodSeconds = c.LivenessProbe.PeriodSeconds
			o.LivenessProbe.SuccessThreshold = c.LivenessProbe.SuccessThreshold
			o.LivenessProbe.FailureThreshold = c.LivenessProbe.FailureThreshold
		}
	}

	// DeepDerivative ensures that, for any non-nil, non-empty value in A, the corresponding value is set in B
	return equality.Semantic.DeepDerivative(calculated, onCluster)
}

// Clean up orphaned deployments after reinstalling deployments process
func (i *StrategyDeploymentInstaller) cleanupOrphanedDeployments(deploymentSpecs []StrategyDeploymentSpec) error {
	// Map of deployments
	depNames := map[string]string{}
	for _, dep := range deploymentSpecs {
		depNames[dep.Name] = dep.Name
	}

	// Check the owner is a CSV
	csv, ok := i.owner.(*v1alpha1.ClusterServiceVersion)
	if !ok {
		return fmt.Errorf("owner %s is not a CSV", i.owner.GetName())
	}

	// Get existing deployments in CSV's namespace and owned by CSV
	existingDeployments, err := i.strategyClient.FindAnyDeploymentsMatchingLabels(ownerutil.CSVOwnerSelector(csv))
	if err != nil {
		return err
	}

	// compare existing deployments to deployments in CSV's spec to see if any need to be deleted
	for _, d := range existingDeployments {
		if _, exists := depNames[d.GetName()]; !exists {
			if ownerutil.IsOwnedBy(d, i.owner) {
				log.Infof("found an orphaned deployment %s in namespace %s", d.GetName(), i.owner.GetNamespace())
				if err := i.strategyClient.DeleteDeployment(d.GetName()); err != nil {
					log.Warnf("error cleaning up deployment %s", d.GetName())
					return err
				}
			}
		}
	}

	return nil
}
