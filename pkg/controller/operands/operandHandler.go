package operands

import (
	"context"
	"fmt"
	"sync"
	"time"

	log "github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	kubevirtcorev1 "kubevirt.io/api/core/v1"
	cdiv1beta1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/common"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/metrics"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
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

func NewOperandHandler(client client.Client, scheme *runtime.Scheme, isOpenshiftCluster bool, eventEmitter hcoutil.EventEmitter) *OperandHandler {
	operands := []Operand{
		(*genericOperand)(newKvPriorityClassHandler(client, scheme)),
		(*genericOperand)(newKubevirtHandler(client, scheme)),
		(*genericOperand)(newCdiHandler(client, scheme)),
		(*genericOperand)(newStorageConfigHandler(client, scheme)),
		(*genericOperand)(newCnaHandler(client, scheme)),
	}

	if isOpenshiftCluster {
		operands = append(operands, []Operand{
			newSspHandler(client, scheme),
			(*genericOperand)(newMetricsServiceHandler(client, scheme)),
			(*genericOperand)(newMetricsServiceMonitorHandler(client, scheme)),
			(*genericOperand)(newMonitoringPrometheusRuleHandler(client, scheme)),
			(*genericOperand)(newCliDownloadHandler(client, scheme)),
			(*genericOperand)(newCliDownloadsRouteHandler(client, scheme)),
			(*genericOperand)(newCliDownloadsServiceHandler(client, scheme)),
		}...)
	}

	return &OperandHandler{
		client:       client,
		operands:     operands,
		eventEmitter: eventEmitter,
	}
}

// The k8s client is not available when calling to NewOperandHandler.
// Initial operations that need to read/write from the cluster can only be done when the client is already working.
func (h *OperandHandler) FirstUseInitiation(scheme *runtime.Scheme, isOpenshiftCluster bool, hc *hcov1beta1.HyperConverged) {
	h.objects = make([]client.Object, 0)
	if isOpenshiftCluster {
		h.addOperands(scheme, hc, getQuickStartHandlers)
		h.addOperands(scheme, hc, getDashboardHandlers)
		h.addOperands(scheme, hc, getImageStreamHandlers)
		h.addOperands(scheme, hc, newVirtioWinCmHandler)
		h.addOperands(scheme, hc, newVirtioWinCmReaderRoleHandler)
		h.addOperands(scheme, hc, newVirtioWinCmReaderRoleBindingHandler)
	}
	// Role and RoleBinding for kvStorage Config Map should be created both on Openshift and plain k8s
	h.addOperands(scheme, hc, NewConfigReaderRoleHandler)
	h.addOperands(scheme, hc, newConfigReaderRoleBindingHandler)

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
			obj, err := handler.(*genericOperand).hooks.getFullCr(hc)
			if err != nil {
				logger.Error(err, "can't create object")
				continue
			}
			h.objects = append(h.objects, obj)
		}
		h.operands = append(h.operands, handlers...)
	}
}

func (h OperandHandler) Ensure(req *common.HcoRequest) error {
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
			if !res.Overwritten {
				h.eventEmitter.EmitEvent(req.Instance, corev1.EventTypeNormal, "Updated", fmt.Sprintf("Updated %s %s", res.Type, res.Name))
			} else {
				h.eventEmitter.EmitEvent(req.Instance, corev1.EventTypeWarning, "Overwritten", fmt.Sprintf("Overwritten %s %s", res.Type, res.Name))
				metrics.HcoMetrics.IncOverwrittenModifications(res.Type, res.Name)
			}
		}

		req.ComponentUpgradeInProgress = req.ComponentUpgradeInProgress && res.UpgradeDone
	}
	return nil

}

func (h OperandHandler) EnsureDeleted(req *common.HcoRequest) error {

	tCtx, cancel := context.WithTimeout(req.Ctx, deleteTimeOut)
	defer cancel()

	wg := sync.WaitGroup{}
	errorCh := make(chan error)
	done := make(chan bool)

	resources := []client.Object{
		NewKubeVirtWithNameOnly(req.Instance),
		NewCDIWithNameOnly(req.Instance),
		NewNetworkAddonsWithNameOnly(req.Instance),
		NewSSP(req.Instance),
		NewConsoleCLIDownload(req.Instance),
	}

	resources = append(resources, h.objects...)

	wg.Add(len(resources))

	go func() {
		wg.Wait()
		close(done)
	}()

	for _, res := range resources {
		go func(o client.Object, wgr *sync.WaitGroup) {
			defer wgr.Done()
			err := hcoutil.EnsureDeleted(tCtx, h.client, o, req.Instance.Name, req.Logger, false, true)
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
				errorCh <- err
			} else {
				key := client.ObjectKeyFromObject(o)
				h.eventEmitter.EmitEvent(req.Instance, corev1.EventTypeNormal, "Killing", fmt.Sprintf("Removed %s %s", o.GetObjectKind().GroupVersionKind().Kind, key.Name))
			}
		}(res, &wg)
	}

	select {
	case err := <-errorCh:
		return err
	case <-tCtx.Done():
		return tCtx.Err()
	case <-done:
		// just in case close(done) was selected while there is an error,
		// check the error channel again.
		if len(errorCh) != 0 {
			err := <-errorCh
			return err
		}

		return nil
	}
}

func (h *OperandHandler) Reset() {
	for _, op := range h.operands {
		op.reset()
	}
}
