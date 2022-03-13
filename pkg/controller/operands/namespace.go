package operands

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"

	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/common"
)

// Handles a specific resource (a CR, a configMap and so on), to be run during reconciliation
type namespaceHandler struct {
	// K8s client
	Client client.Client
	Scheme *runtime.Scheme
}

func newNamespaceHandler(Client client.Client, Scheme *runtime.Scheme) *namespaceHandler {
	return &namespaceHandler{
		Client: Client,
		Scheme: Scheme,
	}
}

func (h *namespaceHandler) ensure(req *common.HcoRequest) *EnsureResult {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: req.Instance.Namespace,
		},
	}
	key := client.ObjectKeyFromObject(namespace)
	found := &corev1.Namespace{}
	err := h.Client.Get(req.Ctx, key, found)
	if err != nil {
		req.Logger.Error(err, "failed fetching namespace")
		return &EnsureResult{
			Err: err,
		}
	}
	res := NewEnsureResult(found)
	res.SetName(key.Name)

	needUpdate := false
	if found.Annotations == nil {
		found.Annotations = make(map[string]string)
	}
	if found_v, hasKey := found.Annotations[hcoutil.OpenshiftNodeSelectorAnn]; !hasKey || found_v != "" {
		found.Annotations[hcoutil.OpenshiftNodeSelectorAnn] = ""
		needUpdate = true
	}

	if needUpdate {
		if req.HCOTriggered {
			req.Logger.Info("Updating existing namespace to new opinionated values")
		} else {
			req.Logger.Info("Reconciling an externally updated namespace to its opinionated values")
		}
		err := h.Client.Update(req.Ctx, found)
		if err != nil {
			if err != nil {
				req.Logger.Error(err, "failed updating the namespace")
				return &EnsureResult{
					Err: err,
				}
			}
		}
		res.SetUpdated()
		res.SetOverwritten(!req.HCOTriggered)
		return res
	}

	return res.SetUpgradeDone(req.ComponentUpgradeInProgress)
}

func (h namespaceHandler) reset( /* No implementation */ ) {}
