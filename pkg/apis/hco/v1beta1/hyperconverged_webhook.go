package v1beta1

import (
	"context"
	"fmt"
	kubevirtv1 "kubevirt.io/client-go/api/v1"
	cdiv1beta1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1beta1"
	"os"
	"path/filepath"
	"reflect"

	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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

func (r *HyperConverged) SetupWebhookWithManager(ctx context.Context, mgr ctrl.Manager) error {
	// Make sure the certificates are mounted, this should be handled by the OLM
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
	hcolog.Info("Validating create", "name", r.Name, "namespace:", r.Namespace)

	operatorNsEnv, err := hcoutil.GetOperatorNamespaceFromEnv()
	if err != nil {
		hcolog.Error(err, "Failed to get operator namespace from the environment")
		return err
	}

	if r.Namespace != operatorNsEnv {
		return fmt.Errorf("Invalid namespace for HyperConverged - please use the %s namespace", operatorNsEnv)
	}

	return nil
}

func (r *HyperConverged) ValidateUpdate(old runtime.Object) error {
	hcolog.Info("Validating update", "name", r.Name)

	ctx := context.TODO()

	oldR, ok := old.(*HyperConverged)
	if !ok {
		return fmt.Errorf("expect old object to be a %T instead of %T", oldR, old)
	}

	if !reflect.DeepEqual(
		oldR.Spec.Workloads,
		r.Spec.Workloads) {

		opts := &client.UpdateOptions{DryRun: []string{metav1.DryRunAll}}
		for _, obj := range []runtime.Object{
			r.NewKubeVirt(),
			r.NewCDI(),
		} {
			if err := r.UpdateOperatorCr(ctx, obj, opts); err != nil {
				return err
			}
		}
	}

	return nil
}

// currently only supports KV and CDI
func (r *HyperConverged) UpdateOperatorCr(ctx context.Context, exists runtime.Object, opts *client.UpdateOptions) error {
	err := hcoutil.GetRuntimeObject(ctx, cli, exists, hcolog)
	if err != nil {
		hcolog.Error(err, "failed to get object from kubernetes", "kind", exists.GetObjectKind())
		return err
	}

	switch exists.(type) {
	case *kubevirtv1.KubeVirt:
		existingKv := exists.(*kubevirtv1.KubeVirt)
		required := r.NewKubeVirt()
		existingKv.Spec = required.Spec

	case *cdiv1beta1.CDI:
		existingCdi := exists.(*cdiv1beta1.CDI)
		required := r.NewCDI()
		existingCdi.Spec = required.Spec
	}

	if err = cli.Update(ctx, exists, opts); err != nil {
		hcolog.Error(err, "failed to dry-run update the object", "kind", exists.GetObjectKind())
		return err
	}

	hcolog.Info("dry-run update the object passed", "kind", exists.GetObjectKind())
	return nil
}

func (r *HyperConverged) ValidateDelete() error {
	hcolog.Info("Validating delete", "name", r.Name, "namespace", r.Namespace)

	ctx := context.TODO()

	for _, obj := range []runtime.Object{
		r.NewKubeVirt(),
		r.NewCDI(),
	} {
		err := hcoutil.EnsureDeleted(ctx, cli, obj, r.Name, hcolog, true)
		if err != nil {
			hcolog.Error(err, "Delete validation failed", "GVK", obj.GetObjectKind().GroupVersionKind())
			return err
		}
	}

	return nil
}
