package operands

import (
	"errors"
	"reflect"
	"sync"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/common"
)

type newDeploymentFunc func(hc *hcov1beta1.HyperConverged) *appsv1.Deployment

func newDeploymentHandler(Client client.Client, Scheme *runtime.Scheme, deploymentGenerator newDeploymentFunc, hc *hcov1beta1.HyperConverged) Operand {
	return &genericOperand{
		Client: Client,
		Scheme: Scheme,
		crType: "Deployment",
		hooks:  newDeploymentHooks(deploymentGenerator, hc),
	}
}

type deploymentHooks struct {
	sync.Mutex
	deploymentGenerator newDeploymentFunc
	cache               *appsv1.Deployment
}

func newDeploymentHooks(deploymentGenerator newDeploymentFunc, hc *hcov1beta1.HyperConverged) *deploymentHooks {
	return &deploymentHooks{
		cache:               deploymentGenerator(hc),
		deploymentGenerator: deploymentGenerator,
	}
}

func (h *deploymentHooks) reset() {
	h.Lock()
	defer h.Unlock()

	h.cache = nil
}

func (h *deploymentHooks) getFullCr(hc *hcov1beta1.HyperConverged) (client.Object, error) {
	h.Lock()
	defer h.Unlock()

	if h.cache == nil {
		h.cache = h.deploymentGenerator(hc)
	}

	return h.cache, nil
}

func (h *deploymentHooks) getEmptyCr() client.Object {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: h.cache.Name,
		},
	}
}

func (h *deploymentHooks) justBeforeComplete(_ *common.HcoRequest) { /* no implementation */ }

func (h *deploymentHooks) updateCr(req *common.HcoRequest, Client client.Client, exists runtime.Object, required runtime.Object) (bool, bool, error) {
	deployment, ok1 := required.(*appsv1.Deployment)
	found, ok2 := exists.(*appsv1.Deployment)

	if !ok1 || !ok2 {
		return false, false, errors.New("can't convert to Deployment")
	}
	if !hasCorrectDeploymentFields(found, deployment) {
		if req.HCOTriggered {
			req.Logger.Info("Updating existing Deployment to new opinionated values", "name", deployment.Name)
		} else {
			req.Logger.Info("Reconciling an externally updated Deployment to its opinionated values", "name", deployment.Name)
		}
		if shouldRecreate(found, deployment) {
			err := Client.Delete(req.Ctx, found, &client.DeleteOptions{})
			if err != nil {
				return false, false, err
			}
			err = Client.Create(req.Ctx, deployment, &client.CreateOptions{})
			if err != nil {
				return false, false, err
			}
			return true, !req.HCOTriggered, nil
		}
		util.MergeLabels(&deployment.ObjectMeta, &found.ObjectMeta)
		deployment.Spec.DeepCopyInto(&found.Spec)
		err := Client.Update(req.Ctx, found)
		if err != nil {
			return false, false, err
		}
		return true, !req.HCOTriggered, nil
	}

	return false, false, nil
}

// We need to check only certain fields in the deployment resource, since some of the fields
// are being set by k8s.
func hasCorrectDeploymentFields(found *appsv1.Deployment, required *appsv1.Deployment) bool {
	return util.CompareLabels(required, found) &&
		reflect.DeepEqual(found.Spec.Selector, required.Spec.Selector) &&
		reflect.DeepEqual(found.Spec.Replicas, required.Spec.Replicas) &&
		reflect.DeepEqual(found.Spec.Template.Spec.Containers, required.Spec.Template.Spec.Containers) &&
		reflect.DeepEqual(found.Spec.Template.Spec.ServiceAccountName, required.Spec.Template.Spec.ServiceAccountName) &&
		reflect.DeepEqual(found.Spec.Template.Spec.PriorityClassName, required.Spec.Template.Spec.PriorityClassName) &&
		reflect.DeepEqual(found.Spec.Template.Spec.Affinity, required.Spec.Template.Spec.Affinity) &&
		reflect.DeepEqual(found.Spec.Template.Spec.NodeSelector, required.Spec.Template.Spec.NodeSelector) &&
		reflect.DeepEqual(found.Spec.Template.Spec.Tolerations, required.Spec.Template.Spec.Tolerations)
}

func shouldRecreate(found, required *appsv1.Deployment) bool {
	// updating LabelSelector (it's immutable) would be rejected by API server; create new Deployment instead
	return !reflect.DeepEqual(found.Spec.Selector, required.Spec.Selector)
}
