package operands

import (
	"context"
	"fmt"
	"time"

	log "github.com/go-logr/logr"
	"golang.org/x/sync/errgroup"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/common"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/monitoring/metrics"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
	kubevirtcorev1 "kubevirt.io/api/core/v1"
	cdiv1beta1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"
)

const (
	reconcileFailed       = "ReconcileFailed"
	ErrCDIUninstall       = "ErrCDIUninstall"
	uninstallCDIErrorMsg  = "The uninstall request failed on CDI component: "
	ErrVirtUninstall      = "ErrVirtUninstall"
	uninstallVirtErrorMsg = "The uninstall request failed on virt component: "
	ErrHCOUninstall       = "ErrHCOUninstall"
	uninstallHCOErrorMsg  = "The uninstall request failed on dependent components, please check their logs."
	deleteTimeOut         = 30 * time.Second
)

// common constants
const (
	kvPriorityClass = "kubevirt-cluster-critical"
)

var (
	logger = logf.Log.WithName("operandHandlerInit")
)

type OperandHandler struct {
	client   client.Client
	operands []Operand
	// save for deletions
	objects      []client.Object
	eventEmitter hcoutil.EventEmitter
}

func NewOperandHandler(client client.Client, scheme *runtime.Scheme, ci hcoutil.ClusterInfo, eventEmitter hcoutil.EventEmitter) *OperandHandler {
	operands := []Operand{
		(*genericOperand)(newKvPriorityClassHandler(client, scheme)),
		(*genericOperand)(newKubevirtHandler(client, scheme)),
		(*genericOperand)(newCdiHandler(client, scheme)),
		(*genericOperand)(newCnaHandler(client, scheme)),
		newMtqHandler(client, scheme),
	}

	if ci.IsOpenshift() {
		operands = append(operands, []Operand{
			(*genericOperand)(newSspHandler(client, scheme)),
			(*genericOperand)(newCliDownloadHandler(client, scheme)),
			(*genericOperand)(newCliDownloadsRouteHandler(client, scheme)),
			(*genericOperand)(newServiceHandler(client, scheme, NewCliDownloadsService)),
		}...)
	}

	if ci.IsOpenshift() && ci.IsConsolePluginImageProvided() {
		operands = append(operands, newConsoleHandler(client))
		operands = append(operands, (*genericOperand)(newServiceHandler(client, scheme, NewKvUIPluginSvc)))
		operands = append(operands, (*genericOperand)(newServiceHandler(client, scheme, NewKvUIProxySvc)))
	}

	if ci.IsManagedByOLM() {
		operands = append(operands, newCsvHandler(client, ci))
	}

	return &OperandHandler{
		client:       client,
		operands:     operands,
		eventEmitter: eventEmitter,
	}
}

// FirstUseInitiation is a lazy init function
// The k8s client is not available when calling to NewOperandHandler.
// Initial operations that need to read/write from the cluster can only be done when the client is already working.
func (h *OperandHandler) FirstUseInitiation(scheme *runtime.Scheme, ci hcoutil.ClusterInfo, hc *hcov1beta1.HyperConverged) {
	h.objects = make([]client.Object, 0)
	if ci.IsOpenshift() {
		h.addOperands(scheme, hc, getQuickStartHandlers)
		h.addOperands(scheme, hc, getDashboardHandlers)
		h.addOperands(scheme, hc, getImageStreamHandlers)
		h.addOperands(scheme, hc, newVirtioWinCmHandler)
		h.addOperands(scheme, hc, newVirtioWinCmReaderRoleHandler)
		h.addOperands(scheme, hc, newVirtioWinCmReaderRoleBindingHandler)
	}

	if ci.IsOpenshift() && ci.IsConsolePluginImageProvided() {
		h.addOperands(scheme, hc, newKvUIPluginDeploymentHandler)
		h.addOperands(scheme, hc, newKvUIProxyDeploymentHandler)
		h.addOperands(scheme, hc, newKvUINginxCMHandler)
		h.addOperands(scheme, hc, newKvUIPluginCRHandler)
	}
}

func (h *OperandHandler) GetQuickStartNames() []string {
	return quickstartNames
}

type GetHandler func(logger log.Logger, Client client.Client, Scheme *runtime.Scheme, hc *hcov1beta1.HyperConverged) ([]Operand, error)

func (h *OperandHandler) addOperands(scheme *runtime.Scheme, hc *hcov1beta1.HyperConverged, getHandler GetHandler) {
	handlers, err := getHandler(logger, h.client, scheme, hc)
	if err != nil {
		logger.Error(err, "can't create ConsoleQuickStarts objects")
	} else if len(handlers) > 0 {
		for _, handler := range handlers {
			var (
				obj client.Object
				err error
			)

			switch h := handler.(type) {
			case *genericOperand:
				obj, err = h.hooks.getFullCr(hc)
			case *imageStreamOperand:
				obj, err = h.operand.hooks.getFullCr(hc)
			default:
				err = fmt.Errorf("unknown handler with type %v", h)
			}

			if err != nil {
				logger.Error(err, "can't create object")
				continue
			}
			h.objects = append(h.objects, obj)
		}
		h.operands = append(h.operands, handlers...)
	}
}

func (h *OperandHandler) Ensure(req *common.HcoRequest) error {
	for _, handler := range h.operands {
		res := handler.ensure(req)
		if res.Err != nil {
			req.Logger.Error(res.Err, "failed to ensure an operand")

			req.ComponentUpgradeInProgress = false
			req.Conditions.SetStatusCondition(metav1.Condition{
				Type:               hcov1beta1.ConditionReconcileComplete,
				Status:             metav1.ConditionFalse,
				Reason:             reconcileFailed,
				Message:            fmt.Sprintf("Error while reconciling: %v", res.Err),
				ObservedGeneration: req.Instance.Generation,
			})
			return res.Err
		}

		if res.Created {
			h.eventEmitter.EmitEvent(req.Instance, corev1.EventTypeNormal, "Created", fmt.Sprintf("Created %s %s", res.Type, res.Name))
		} else if res.Updated {
			h.handleUpdatedOperand(req, res)
		} else if res.Deleted {
			h.eventEmitter.EmitEvent(req.Instance, corev1.EventTypeNormal, "Killing", fmt.Sprintf("Removed %s %s", res.Type, res.Name))
		}

		req.ComponentUpgradeInProgress = req.ComponentUpgradeInProgress && res.UpgradeDone
	}
	return nil

}

func (h *OperandHandler) handleUpdatedOperand(req *common.HcoRequest, res *EnsureResult) {
	if !res.Overwritten {
		h.eventEmitter.EmitEvent(req.Instance, corev1.EventTypeNormal, "Updated", fmt.Sprintf("Updated %s %s", res.Type, res.Name))
	} else {
		h.eventEmitter.EmitEvent(req.Instance, corev1.EventTypeWarning, "Overwritten", fmt.Sprintf("Overwritten %s %s", res.Type, res.Name))
		if !req.UpgradeMode {
			metrics.IncOverwrittenModifications(res.Type, res.Name)
		}
	}
}

func (h *OperandHandler) EnsureDeleted(req *common.HcoRequest) error {

	tCtx, cancel := context.WithTimeout(req.Ctx, deleteTimeOut)
	defer cancel()

	resources := []client.Object{
		NewKubeVirtWithNameOnly(req.Instance),
		NewCDIWithNameOnly(req.Instance),
		NewNetworkAddonsWithNameOnly(req.Instance),
		NewSSPWithNameOnly(req.Instance),
		NewConsoleCLIDownload(req.Instance),
		NewMTQWithNameOnly(req.Instance),
	}

	resources = append(resources, h.objects...)

	eg, egCtx := errgroup.WithContext(tCtx)

	for _, res := range resources {
		func(o client.Object) {
			eg.Go(func() error {
				deleted, err := hcoutil.EnsureDeleted(egCtx, h.client, o, req.Instance.Name, req.Logger, false, true, true)
				if err != nil {
					req.Logger.Error(err, "Failed to manually delete objects")
					errT := ErrHCOUninstall
					errMsg := uninstallHCOErrorMsg
					switch o.(type) {
					case *kubevirtcorev1.KubeVirt:
						errT = ErrVirtUninstall
						errMsg = uninstallVirtErrorMsg + err.Error()
					case *cdiv1beta1.CDI:
						errT = ErrCDIUninstall
						errMsg = uninstallCDIErrorMsg + err.Error()
					}

					h.eventEmitter.EmitEvent(req.Instance, corev1.EventTypeWarning, errT, errMsg)
					return err
				} else if deleted {
					key := client.ObjectKeyFromObject(o)
					h.eventEmitter.EmitEvent(req.Instance, corev1.EventTypeNormal, "Killing", fmt.Sprintf("Removed %s %s", o.GetObjectKind().GroupVersionKind().Kind, key.Name))
				}
				return nil
			})
		}(res)
	}

	return eg.Wait()
}

func (h *OperandHandler) Reset() {
	for _, op := range h.operands {
		op.reset()
	}
}
