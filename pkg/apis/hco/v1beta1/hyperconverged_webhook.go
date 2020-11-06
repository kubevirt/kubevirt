package v1beta1

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"os"
	"path/filepath"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

const (
	WebhookPort     = 4343
	WebhookCertDir  = "/apiserver.local.config/certificates"
	WebhookCertName = "apiserver.crt"
	WebhookKeyName  = "apiserver.key"
)

var (
	hcolog = logf.Log.WithName("hyperconverged-resource")
	cli    client.Client
)

type WebhookHandlerIfs interface {
	Init(logger logr.Logger, cli client.Client)
	ValidateCreate(hc *HyperConverged) error
	ValidateUpdate(requested *HyperConverged, exists *HyperConverged) error
	ValidateDelete(hc *HyperConverged) error
}

var whHandler WebhookHandlerIfs

func (r *HyperConverged) SetupWebhookWithManager(ctx context.Context, mgr ctrl.Manager, handler WebhookHandlerIfs) error {
	// Make sure the certificates are mounted, this should be handled by the OLM
	whHandler = handler
	whHandler.Init(hcolog, cli)

	certs := []string{filepath.Join(WebhookCertDir, WebhookCertName), filepath.Join(WebhookCertDir, WebhookKeyName)}
	for _, fname := range certs {
		if _, err := os.Stat(fname); err != nil {
			hcolog.Error(err, "CSV certificates were not found, skipping webhook initialization")
			return err
		}
	}

	// Use the client from the manager in the validating functions
	cli = mgr.GetClient()

	// The OLM limits the webhook scope to the namespaces that are defined in the OperatorGroup
	// by setting namespaceSelector in the ValidatingWebhookConfiguration.  We would like our webhook to intercept
	// requests from all namespaces, and fail them if they're not in the correct namespace for HCO (for CREATE).
	// Lucikly the OLM does not watch and reconcile the ValidatingWebhookConfiguration so we can simply reset the
	// namespaceSelector

	vwcList := &admissionregistrationv1.ValidatingWebhookConfigurationList{}
	err := mgr.GetAPIReader().List(ctx, vwcList, client.MatchingLabels{"olm.webhook-description-generate-name": hcoutil.HcoValidatingWebhook})
	if err != nil {
		hcolog.Error(err, "A validating webhook for the HCO was not found")
		return err
	}

	for _, vwc := range vwcList.Items {
		update := false

		for i, wh := range vwc.Webhooks {
			if wh.Name == hcoutil.HcoValidatingWebhook {
				vwc.Webhooks[i].NamespaceSelector = &metav1.LabelSelector{MatchLabels: map[string]string{}}
				update = true
			}
		}

		if update {
			hcolog.Info("Removing namespace scope from webhook", "webhook", vwc.Name)
			err = mgr.GetClient().Update(ctx, &vwc)
			if err != nil {
				hcolog.Error(err, "Failed updating webhook", "webhook", vwc.Name)
				return err
			}
		}
	}

	bldr := ctrl.NewWebhookManagedBy(mgr).For(r)
	srv := mgr.GetWebhookServer()
	srv.CertDir = WebhookCertDir
	srv.CertName = WebhookCertName
	srv.KeyName = WebhookKeyName
	srv.Port = WebhookPort
	return bldr.Complete()
}

var _ webhook.Validator = &HyperConverged{}

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
