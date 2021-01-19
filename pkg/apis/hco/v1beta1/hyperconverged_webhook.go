package v1beta1

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-logr/logr"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
	admissionv1 "k8s.io/api/admission/v1"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	webHookCertDirEnv     = "WEBHOOK_CERT_DIR"
	DefaultWebhookCertDir = "/apiserver.local.config/certificates"

	WebhookCertName = "apiserver.crt"
	WebhookKeyName  = "apiserver.key"
)

var (
	hcolog = logf.Log.WithName("hyperconverged-resource")
)

func GetWebhookCertDir() string {
	webhookCertDir := os.Getenv(webHookCertDirEnv)
	if webhookCertDir != "" {
		return webhookCertDir
	}

	return DefaultWebhookCertDir
}

type WebhookHandlerIfs interface {
	Init(logger logr.Logger, cli client.Client, namespace string, isOpenshift bool)
	ValidateCreate(hc *HyperConverged) error
	ValidateUpdate(requested *HyperConverged, exists *HyperConverged) error
	ValidateDelete(hc *HyperConverged) error
	HandleMutatingNsDelete(ns *corev1.Namespace, dryRun bool) (bool, error)
}

var whHandler WebhookHandlerIfs

func (r *HyperConverged) SetupWebhookWithManager(ctx context.Context, mgr ctrl.Manager, handler WebhookHandlerIfs, isOpenshift bool) error {
	operatorNsEnv, nserr := hcoutil.GetOperatorNamespaceFromEnv()
	if nserr != nil {
		hcolog.Error(nserr, "failed to get operator namespace from the environment")
		return nserr
	}

	// Make sure the certificates are mounted, this should be handled by the OLM
	whHandler = handler
	whHandler.Init(hcolog, mgr.GetClient(), operatorNsEnv, isOpenshift)

	webhookCertDir := GetWebhookCertDir()
	certs := []string{filepath.Join(webhookCertDir, WebhookCertName), filepath.Join(webhookCertDir, WebhookKeyName)}
	for _, fname := range certs {
		if _, err := os.Stat(fname); err != nil {
			hcolog.Error(err, "CSV certificates were not found, skipping webhook initialization")
			return err
		}
	}

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
	srv.CertDir = GetWebhookCertDir()
	srv.CertName = WebhookCertName
	srv.KeyName = WebhookKeyName
	srv.Port = hcoutil.WebhookPort
	srv.Register(hcoutil.HCONSWebhookPath, &webhook.Admission{Handler: &nsMutator{}})

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

// nsMutator mutates Ns requests
type nsMutator struct {
	decoder *admission.Decoder
}

// TODO: nsMutator should try to delete HyperConverged CR before deleting the namespace
// currently it simply blocks namespace deletion if HyperConverged CR is there
func (a *nsMutator) Handle(ctx context.Context, req admission.Request) admission.Response {
	hcolog.Info("reaching nsMutator.Handle")
	ns := &corev1.Namespace{}

	if req.Operation == admissionv1.Delete {

		// In reference to PR: https://github.com/kubernetes/kubernetes/pull/76346
		// OldObject contains the object being deleted
		err := a.decoder.DecodeRaw(req.OldObject, ns)
		if err != nil {
			hcolog.Error(err, "failed decoding namespace object")
			return admission.Errored(http.StatusBadRequest, err)
		}

		admitted, herr := whHandler.HandleMutatingNsDelete(ns, *req.DryRun)
		if err != nil {
			return admission.Errored(http.StatusInternalServerError, herr)
		}
		if admitted {
			return admission.Allowed("the namespace doesn't contain HyperConverged CR, admitting its deletion")
		}
		return admission.Denied("HyperConverged CR is still present, please remove it before deleting the containing namespace")
	}

	// ignoring other operations
	return admission.Allowed("ignoring other operations")

}

// nsMutator implements admission.DecoderInjector.
// A decoder will be automatically injected.

// InjectDecoder injects the decoder.
func (a *nsMutator) InjectDecoder(d *admission.Decoder) error {
	a.decoder = d
	return nil
}
