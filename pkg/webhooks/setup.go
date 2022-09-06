package webhooks

import (
	"context"
	"crypto/tls"
	"os"
	"path/filepath"

	openshiftconfigv1 "github.com/openshift/api/config/v1"

	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/webhooks/mutator"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/webhooks/validator"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
)

const (
	webHookCertDirEnv = "WEBHOOK_CERT_DIR"
)

var (
	logger = logf.Log.WithName("webhook-setup")
)

func GetWebhookCertDir() string {
	webhookCertDir := os.Getenv(webHookCertDirEnv)
	if webhookCertDir != "" {
		return webhookCertDir
	}

	return hcoutil.DefaultWebhookCertDir
}

func SetupWebhookWithManager(ctx context.Context, mgr ctrl.Manager, isOpenshift bool, tlsSecurityProfile *openshiftconfigv1.TLSSecurityProfile) error {
	operatorNsEnv, nserr := hcoutil.GetOperatorNamespaceFromEnv()
	if nserr != nil {
		logger.Error(nserr, "failed to get operator namespace from the environment")
		return nserr
	}

	whHandler := validator.NewWebhookHandler(logger, mgr.GetClient(), operatorNsEnv, isOpenshift, tlsSecurityProfile)

	nsMutator := mutator.NewNsMutator(mgr.GetClient(), operatorNsEnv)

	// Make sure the certificates are mounted, this should be handled by the OLM
	webhookCertDir := GetWebhookCertDir()
	certs := []string{filepath.Join(webhookCertDir, hcoutil.WebhookCertName), filepath.Join(webhookCertDir, hcoutil.WebhookKeyName)}
	for _, fname := range certs {
		if _, err := os.Stat(fname); err != nil {
			logger.Error(err, "CSV certificates were not found, skipping webhook initialization")
			return err
		}
	}

	if err := allowWatchAllNamespaces(ctx, mgr); err != nil {
		return err
	}

	srv := mgr.GetWebhookServer()
	srv.CertDir = GetWebhookCertDir()
	srv.CertName = hcoutil.WebhookCertName
	srv.KeyName = hcoutil.WebhookKeyName
	srv.Port = hcoutil.WebhookPort

	srv.TLSOpts = []func(*tls.Config){whHandler.MutateTLSConfig}

	srv.Register(hcoutil.HCONSWebhookPath, &webhook.Admission{Handler: nsMutator})
	srv.Register(hcoutil.HCOWebhookPath, &webhook.Admission{Handler: whHandler})

	return nil
}

// The OLM limits the webhook scope to the namespaces that are defined in the OperatorGroup
// by setting namespaceSelector in the ValidatingWebhookConfiguration. We would like our webhook to intercept
// requests from all namespaces, and fail them if they're not in the correct namespace for HCO (for CREATE).
// Luckily the OLM does not watch and reconcile the ValidatingWebhookConfiguration so we can simply reset the
// namespaceSelector
func allowWatchAllNamespaces(ctx context.Context, mgr ctrl.Manager) error {
	vwcList := &admissionregistrationv1.ValidatingWebhookConfigurationList{}
	err := mgr.GetAPIReader().List(ctx, vwcList, client.MatchingLabels{"olm.webhook-description-generate-name": hcoutil.HcoValidatingWebhook})
	if err != nil {
		logger.Error(err, "A validating webhook for the HCO was not found")
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
			logger.Info("Removing namespace scope from webhook", "webhook", vwc.Name)
			err = mgr.GetClient().Update(ctx, &vwc)
			if err != nil {
				logger.Error(err, "Failed updating webhook", "webhook", vwc.Name)
				return err
			}
		}
	}
	return nil
}
