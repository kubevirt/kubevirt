package common

import (
	"context"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1beta1"
)

// hcoRequest - gather data for a specific request
type HcoRequest struct {
	reconcile.Request                                     // inheritance of operator request
	Logger                     logr.Logger                // request logger
	Conditions                 HcoConditions              // in-memory conditions
	Ctx                        context.Context            // context of this request, to be use for any other call
	Instance                   *hcov1beta1.HyperConverged // the current state of the CR, as read from K8s
	UpgradeMode                bool                       // copy of the reconciler upgrade mode
	ComponentUpgradeInProgress bool                       // if in upgrade mode, accumulate the component upgrade status
	Dirty                      bool                       // is something was changed in the CR
	StatusDirty                bool                       // is something was changed in the CR's Status
	HCOTriggered               bool                       // if the request got triggered by a direct modification on HCO CR
}

func NewHcoRequest(ctx context.Context, request reconcile.Request, log logr.Logger, upgradeMode, hcoTriggered bool) *HcoRequest {
	return &HcoRequest{
		Request:                    request,
		Logger:                     log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name),
		Conditions:                 NewHcoConditions(),
		Ctx:                        ctx,
		UpgradeMode:                upgradeMode,
		ComponentUpgradeInProgress: upgradeMode,
		Dirty:                      false,
		StatusDirty:                false,
		HCOTriggered:               hcoTriggered,
	}
}

func (req *HcoRequest) SetUpgradeMode(upgradeMode bool) {
	req.UpgradeMode = upgradeMode
	req.ComponentUpgradeInProgress = upgradeMode
}
