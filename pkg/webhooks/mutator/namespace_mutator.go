package mutator

import (
	"context"
	"net/http"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	ignoreOperationMessage   = "ignoring other operations"
	admittingDeletionMessage = "the hcoNamespace doesn't contain HyperConverged CR, admitting its deletion"
	deniedDeletionMessage    = "HyperConverged CR is still present, please remove it before deleting the containing hcoNamespace"
)

var (
	logger = logf.Log.WithName("hcoNamespace mutator")

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

func (nm *NsMutator) Handle(ctx context.Context, req admission.Request) admission.Response {
	logger.Info("reaching NsMutator.Handle")

	if req.Operation == admissionv1.Delete {
		return nm.handleNsDelete(ctx, req)
	}

	// ignoring other operations
	return admission.Allowed(ignoreOperationMessage)

}

func (nm *NsMutator) handleNsDelete(ctx context.Context, req admission.Request) admission.Response {
	ns := &corev1.Namespace{}

	// In reference to PR: https://github.com/kubernetes/kubernetes/pull/76346
	// OldObject contains the object being deleted
	err := nm.decoder.DecodeRaw(req.OldObject, ns)
	if err != nil {
		logger.Error(err, "failed decoding hcoNamespace object")
		return admission.Errored(http.StatusBadRequest, err)
	}

	admitted, err := nm.handleMutatingNsDelete(ctx, ns)
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

func (nm *NsMutator) handleMutatingNsDelete(ctx context.Context, ns *corev1.Namespace) (bool, error) {
	logger.Info("validating hcoNamespace deletion", "name", ns.Name)

	if ns.Name != nm.namespace {
		logger.Info("ignoring request for a different hcoNamespace")
		return true, nil
	}

	// Block the deletion if the hcoNamespace with a clear error message
	// if HCO CR is still there
	_, err := getHcoObject(ctx, nm.cli, nm.namespace)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return true, nil
		}
		return false, err
	}
	logger.Info("HCO CR still exists, forbid hcoNamespace deletion")
	return false, nil
}
