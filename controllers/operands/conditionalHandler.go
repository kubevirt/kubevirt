package operands

import (
	objectreferencesv1 "github.com/openshift/custom-resource-status/objectreferences/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/reference"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/common"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
)

// conditionalHandler is an operand handler that only deploy the operand CR if a given shouldDeploy function returns true.
// If not, it makes sure the CR is deleted.
type conditionalHandler struct {
	operand       *genericOperand
	shouldDeploy  func(hc *v1beta1.HyperConverged) bool
	getCRWithName func(hc *v1beta1.HyperConverged) client.Object
}

func (ch *conditionalHandler) ensure(req *common.HcoRequest) *EnsureResult {
	if ch.shouldDeploy(req.Instance) {
		return ch.operand.ensure(req)
	}
	return ch.ensureDeleted(req)
}

func (ch *conditionalHandler) reset() {
	ch.operand.reset()
}

func (ch *conditionalHandler) ensureDeleted(req *common.HcoRequest) *EnsureResult {
	cr := ch.getCRWithName(req.Instance)
	res := NewEnsureResult(req.Instance)
	res.SetName(cr.GetName())

	// hcoutil.EnsureDeleted does check that the CR exists before removing it. But it also writes a log message each
	// time it happens, i.e. for every reconcile loop. Assuming the client cache is up-to-date, we can safely get it here
	// with no meaningful performance cost.
	err := ch.operand.Client.Get(req.Ctx, client.ObjectKeyFromObject(cr), cr)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return res.Error(err)
		}
	} else {
		deleted, err := hcoutil.EnsureDeleted(req.Ctx, ch.operand.Client, cr, req.Instance.Name, req.Logger, false, false, true)
		if err != nil {
			return res.Error(err)
		}

		if deleted {
			res.SetDeleted()
			objectRef, err := reference.GetReference(ch.operand.Scheme, cr)
			if err != nil {
				return res.Error(err)
			}

			if err = objectreferencesv1.RemoveObjectReference(&req.Instance.Status.RelatedObjects, *objectRef); err != nil {
				return res.Error(err)
			}
			req.StatusDirty = true
		}
	}

	return res.SetUpgradeDone(req.ComponentUpgradeInProgress)
}
