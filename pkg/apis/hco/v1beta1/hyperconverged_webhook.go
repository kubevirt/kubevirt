package v1beta1

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

var (
	whHandler ValidatorWebhookHandler

	_ webhook.Validator = &HyperConverged{}
)

type ValidatorWebhookHandler interface {
	ValidateCreate(hc *HyperConverged) error
	ValidateUpdate(requested *HyperConverged, exists *HyperConverged) error
	ValidateDelete(hc *HyperConverged) error
}

func SetValidatorWebhookHandler(handler ValidatorWebhookHandler) {
	whHandler = handler
}

func (r *HyperConverged) ValidateCreate() error {
	return whHandler.ValidateCreate(r)
}

func (r *HyperConverged) ValidateUpdate(old runtime.Object) error {
	oldR, ok := old.(*HyperConverged)
	if !ok {
		return fmt.Errorf("expect old object to be a %T instead of %T", oldR, old)
	}

	return whHandler.ValidateUpdate(r, oldR)
}

func (r *HyperConverged) ValidateDelete() error {
	return whHandler.ValidateDelete(r)
}
