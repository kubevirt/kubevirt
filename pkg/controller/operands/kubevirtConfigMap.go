package operands

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/common"
	hcoutils "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
)

const kvCmName = "kubevirt-config"

// Make sure that the kubevirt-config configMap does not exist
type kubeVirtCmHandler struct {
	client  client.Client
	emitter hcoutils.EventEmitter
}

func newKubeVirtCmHandler(client client.Client, emitter hcoutils.EventEmitter) Operand {
	return &kubeVirtCmHandler{
		client:  client,
		emitter: emitter,
	}
}

func (handler kubeVirtCmHandler) ensure(req *common.HcoRequest) *EnsureResult {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kvCmName,
			Namespace: req.Namespace,
		},
	}

	res := NewEnsureResult(cm).SetUpgradeDone(true)
	if req.UpgradeMode {
		// There is different handling during upgrade mode, that also creates backup.
		// In case of error when creating the backup, we want to try again, so we don't
		// want to remove the CM before trying again.
		return res
	}

	err := hcoutils.GetRuntimeObject(req.Ctx, handler.client, cm, req.Logger)
	if apierrors.IsNotFound(err) {
		return res
	}

	if err != nil {
		return res.Error(fmt.Errorf("failed to read the %s ConfigMap; %w", kvCmName, err))
	}

	req.Logger.Info(fmt.Sprintf("removing the unused %s ConfigMap", kvCmName))

	unstructuredCm, err := hcoutils.ToUnstructured(cm)
	if err != nil {
		return res.Error(fmt.Errorf("failed to get Unstructured object from the %s ConfigMap; %w", kvCmName, err))
	}

	policy := metav1.DeletePropagationForeground
	wait := &client.DeleteOptions{
		PropagationPolicy: &policy,
	}

	err = handler.client.Delete(req.Ctx, unstructuredCm, wait)
	if err != nil {
		return res.Error(fmt.Errorf("failed to delete the %s ConfigMap; %w", kvCmName, err))
	}

	handler.emitter.EmitEvent(req.Instance, corev1.EventTypeNormal, "Killing", fmt.Sprintf("Removed ConfigMap %s", kvCmName))

	return res
}

func (kubeVirtCmHandler) reset() { /* Not Implemented */ }
