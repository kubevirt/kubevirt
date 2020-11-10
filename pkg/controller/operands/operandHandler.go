package operands

import (
	"fmt"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/common"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kubevirtv1 "kubevirt.io/client-go/api/v1"
	cdiv1beta1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	reconcileFailed       = "ReconcileFailed"
	ErrCDIUninstall       = "ErrCDIUninstall"
	uninstallCDIErrorMsg  = "The uninstall request failed on CDI component: "
	ErrVirtUninstall      = "ErrVirtUninstall"
	uninstallVirtErrorMsg = "The uninstall request failed on virt component: "
	ErrHCOUninstall       = "ErrHCOUninstall"
	uninstallHCOErrorMsg  = "The uninstall request failed on dependent components, please check their logs."
)

type OperandHandler struct {
	client       client.Client
	operands     []Operand
	eventEmitter hcoutil.EventEmitter
}

func NewOperandHandler(client client.Client, scheme *runtime.Scheme, isOpenshiftCluster bool, eventEmitter hcoutil.EventEmitter) *OperandHandler {
	operands := []Operand{
		&kvConfigHandler{Client: client, Scheme: scheme},
		&kvPriorityClassHandler{Client: client, Scheme: scheme},
		newKubevirtHandler(client, scheme),
		newCdiHandler(client, scheme),
		newCnaHandler(client, scheme),
		&vmImportHandler{Client: client, Scheme: scheme},
		&imsConfigHandler{Client: client, Scheme: scheme},
	}

	if isOpenshiftCluster {
		operands = append(operands, []Operand{
			newCommonTemplateBundleHandler(client, scheme),
			newNodeLabellerBundleHandler(client, scheme),
			newTemplateValidatorHandler(client, scheme),
			newMetricsAggregationHandler(client, scheme),
		}...)
	}

	return &OperandHandler{
		client:       client,
		operands:     operands,
		eventEmitter: eventEmitter,
	}
}

func (h OperandHandler) Ensure(req *common.HcoRequest) error {
	for _, handler := range h.operands {
		res := handler.Ensure(req)
		if res.Err != nil {
			req.ComponentUpgradeInProgress = false
			req.Conditions.SetStatusCondition(conditionsv1.Condition{
				Type:    hcov1beta1.ConditionReconcileComplete,
				Status:  corev1.ConditionFalse,
				Reason:  reconcileFailed,
				Message: fmt.Sprintf("Error while reconciling: %v", res.Err),
			})
			return res.Err
		}

		if res.Created {
			h.eventEmitter.EmitEvent(req.Instance, corev1.EventTypeNormal, "Created", fmt.Sprintf("Created %s %s", res.Type, res.Name))
		} else if res.Updated {
			if !res.Overwritten {
				h.eventEmitter.EmitEvent(req.Instance, corev1.EventTypeNormal, "Updated", fmt.Sprintf("Updated %s %s", res.Type, res.Name))
			} else {
				h.eventEmitter.EmitEvent(req.Instance, corev1.EventTypeWarning, "Overwritten", fmt.Sprintf("Overwritten %s %s", res.Type, res.Name))
				// TODO: raise an alert, count them with a CR specific metric and so on...
			}
		}

		req.ComponentUpgradeInProgress = req.ComponentUpgradeInProgress && res.UpgradeDone
	}
	return nil

}

func (h OperandHandler) EnsureDeleted(req *common.HcoRequest) error {
	for _, obj := range []runtime.Object{
		NewKubeVirt(req.Instance),
		NewCDI(req.Instance),
		NewNetworkAddons(req.Instance),
		NewKubeVirtCommonTemplateBundle(req.Instance),
		NewConsoleCLIDownload(req.Instance),
		NewVMImportForCR(req.Instance),
	} {
		err := hcoutil.EnsureDeleted(req.Ctx, h.client, obj, req.Instance.Name, req.Logger, false)
		if err != nil {
			req.Logger.Error(err, "Failed to manually delete objects")

			// TODO: ask to other components to expose something like
			// func IsDeleteRefused(err error) bool
			// to be able to clearly distinguish between an explicit
			// refuse from other operator and any other kind of error that
			// could potentially happen in the process

			errT := ErrHCOUninstall
			errMsg := uninstallHCOErrorMsg
			switch obj.(type) {
			case *kubevirtv1.KubeVirt:
				errT = ErrVirtUninstall
				errMsg = uninstallVirtErrorMsg + err.Error()
			case *cdiv1beta1.CDI:
				errT = ErrCDIUninstall
				errMsg = uninstallCDIErrorMsg + err.Error()
			}

			h.eventEmitter.EmitEvent(req.Instance, corev1.EventTypeWarning, errT, errMsg)

			return err
		}

		if key, err := client.ObjectKeyFromObject(obj); err == nil {
			h.eventEmitter.EmitEvent(req.Instance, corev1.EventTypeNormal, "Killing", fmt.Sprintf("Removed %s %s", obj.GetObjectKind().GroupVersionKind().Kind, key.Name))
		}
	}
	return nil
}
