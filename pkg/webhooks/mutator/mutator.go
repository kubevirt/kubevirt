package mutator

import (
	"context"
	"net/http"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
)

const (
	ignoreOperationMessage   = "ignoring other operations"
	admittingDeletionMessage = "the namespace doesn't contain HyperConverged CR, admitting its deletion"
	deniedDeletionMessage    = "HyperConverged CR is still present, please remove it before deleting the containing namespace"
)

var (
	logger = logf.Log.WithName("mutator")

	_ admission.Handler = &NsMutator{}
)

// NsMutator mutates Ns requests
type NsMutator struct {
	decoder   *admission.Decoder
	cli       client.Client
	namespace string
}

func NewNsMutator(cli client.Client, namespace string) *NsMutator {
	return &NsMutator{
		cli:       cli,
		namespace: namespace,
	}
}

func (nm *NsMutator) Handle(_ context.Context, req admission.Request) admission.Response {
	logger.Info("reaching NsMutator.Handle")

	if req.Operation == admissionv1.Delete {
		return nm.handleNsDelete(req)
	}

	// ignoring other operations
	return admission.Allowed(ignoreOperationMessage)

}

func (nm *NsMutator) handleNsDelete(req admission.Request) admission.Response {
	ns := &corev1.Namespace{}

	// In reference to PR: https://github.com/kubernetes/kubernetes/pull/76346
	// OldObject contains the object being deleted
	err := nm.decoder.DecodeRaw(req.OldObject, ns)
	if err != nil {
		logger.Error(err, "failed decoding namespace object")
		return admission.Errored(http.StatusBadRequest, err)
	}

	admitted, err := nm.handleMutatingNsDelete(ns)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	if admitted {
		return admission.Allowed(admittingDeletionMessage)
	}

	return admission.Denied(deniedDeletionMessage)
}

// NsMutator implements admission.DecoderInjector.
// A decoder will be automatically injected.

// InjectDecoder injects the decoder.
func (nm *NsMutator) InjectDecoder(d *admission.Decoder) error {
	nm.decoder = d
	return nil
}

func (nm *NsMutator) handleMutatingNsDelete(ns *corev1.Namespace) (bool, error) {
	logger.Info("validating namespace deletion", "name", ns.Name)

	if ns.Name != nm.namespace {
		logger.Info("ignoring request for a different namespace")
		return true, nil
	}

	ctx := context.TODO()
	hco := &v1beta1.HyperConverged{
		ObjectMeta: metav1.ObjectMeta{
			Name:      hcoutil.HyperConvergedName,
			Namespace: nm.namespace,
		},
	}

	// Block the deletion if the namespace with a clear error message
	// if HCO CR is still there

	found := &v1beta1.HyperConverged{}
	err := nm.cli.Get(ctx, client.ObjectKeyFromObject(hco), found)
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("HCO CR doesn't not exist, allow namespace deletion")
			return true, nil
		}
		logger.Error(err, "failed getting HyperConverged CR")
		return false, err
	}
	logger.Info("HCO CR still exists, forbid namespace deletion")
	return false, nil
}
