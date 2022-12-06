package validator

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	openshiftconfigv1 "github.com/openshift/api/config/v1"

	"github.com/go-logr/logr"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	networkaddonsv1 "github.com/kubevirt/cluster-network-addons-operator/pkg/apis/networkaddonsoperator/v1"
	ttov1alpha1 "github.com/kubevirt/tekton-tasks-operator/api/v1alpha1"
	kubevirtcorev1 "kubevirt.io/api/core/v1"
	cdiv1beta1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"
	sspv1beta1 "kubevirt.io/ssp-operator/api/v1beta1"

	"github.com/openshift/library-go/pkg/crypto"

	"github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/operands"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"

	"github.com/samber/lo"
)

const (
	updateDryRunTimeOut = time.Second * 3
)

type WebhookHandler struct {
	logger      logr.Logger
	cli         client.Client
	namespace   string
	isOpenshift bool
	decoder     *admission.Decoder
}

var hcoTlsConfigCache *openshiftconfigv1.TLSSecurityProfile

func NewWebhookHandler(logger logr.Logger, cli client.Client, namespace string, isOpenshift bool, hcoTlsSecurityProfile *openshiftconfigv1.TLSSecurityProfile) *WebhookHandler {
	hcoTlsConfigCache = hcoTlsSecurityProfile
	return &WebhookHandler{
		logger:      logger,
		cli:         cli,
		namespace:   namespace,
		isOpenshift: isOpenshift,
	}
}

func (wh *WebhookHandler) Handle(ctx context.Context, req admission.Request) admission.Response {

	ctx = admission.NewContextWithRequest(ctx, req)

	// Get the object in the request
	obj := &v1beta1.HyperConverged{}

	dryRun := req.DryRun != nil && *req.DryRun

	var err error
	switch req.Operation {
	case admissionv1.Create:
		if err := wh.decoder.Decode(req, obj); err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}

		err = wh.ValidateCreate(ctx, dryRun, obj)
	case admissionv1.Update:
		oldObj := &v1beta1.HyperConverged{}
		if err := wh.decoder.DecodeRaw(req.Object, obj); err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}
		if err := wh.decoder.DecodeRaw(req.OldObject, oldObj); err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}

		err = wh.ValidateUpdate(ctx, dryRun, obj, oldObj)
	case admissionv1.Delete:
		// In reference to PR: https://github.com/kubernetes/kubernetes/pull/76346
		// OldObject contains the object being deleted
		if err := wh.decoder.DecodeRaw(req.OldObject, obj); err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}

		err = wh.ValidateDelete(ctx, dryRun, obj)
	default:
		return admission.Errored(http.StatusBadRequest, fmt.Errorf("unknown operation request %q", req.Operation))
	}

	// Check the error message first.
	if err != nil {
		var apiStatus apierrors.APIStatus
		if errors.As(err, &apiStatus) {
			return validationResponseFromStatus(false, apiStatus.Status())
		}
		return admission.Denied(err.Error())
	}

	// Return allowed if everything succeeded.
	return admission.Allowed("")
}

var _ admission.DecoderInjector = &WebhookHandler{}

// InjectDecoder injects the decoder.
// WebhookHandler implements admission.DecoderInjector so a decoder will be automatically injected.
func (wh *WebhookHandler) InjectDecoder(d *admission.Decoder) error {
	wh.decoder = d
	return nil
}

func (wh *WebhookHandler) ValidateCreate(ctx context.Context, dryrun bool, hc *v1beta1.HyperConverged) error {
	wh.logger.Info("Validating create", "name", hc.Name, "namespace:", hc.Namespace)

	if err := wh.validateCertConfig(hc); err != nil {
		return err
	}

	if hc.Namespace != wh.namespace {
		return fmt.Errorf("invalid namespace for v1beta1.HyperConverged - please use the %s namespace", wh.namespace)
	}

	if err := wh.validateDataImportCronTemplates(hc); err != nil {
		return err
	}

	if err := wh.validateTlsSecurityProfiles(hc); err != nil {
		return err
	}

	if _, err := operands.NewKubeVirt(hc); err != nil {
		return err
	}

	if _, err := operands.NewCDI(hc); err != nil {
		return err
	}

	if _, err := operands.NewNetworkAddons(hc); err != nil {
		return err
	}

	if !dryrun {
		hcoTlsConfigCache = hc.Spec.TLSSecurityProfile
	}

	return nil
}

func (wh *WebhookHandler) getOperands(requested *v1beta1.HyperConverged) (*kubevirtcorev1.KubeVirt, *cdiv1beta1.CDI, *networkaddonsv1.NetworkAddonsConfig, error) {
	if err := wh.validateCertConfig(requested); err != nil {
		return nil, nil, nil, err
	}

	kv, err := operands.NewKubeVirt(requested)
	if err != nil {
		return nil, nil, nil, err
	}

	cdi, err := operands.NewCDI(requested)
	if err != nil {
		return nil, nil, nil, err
	}

	cna, err := operands.NewNetworkAddons(requested)
	if err != nil {
		return nil, nil, nil, err
	}

	return kv, cdi, cna, nil
}

// ValidateUpdate is the ValidateUpdate webhook implementation. It calls all the resources in parallel, to dry-run the
// upgrade.
func (wh *WebhookHandler) ValidateUpdate(extCtx context.Context, dryrun bool, requested *v1beta1.HyperConverged, exists *v1beta1.HyperConverged) error {
	if err := wh.validateDataImportCronTemplates(requested); err != nil {
		return err
	}

	if err := wh.validateTlsSecurityProfiles(requested); err != nil {
		return err
	}

	wh.logger.Info("Validating update", "name", requested.Name)
	ctx, cancel := context.WithTimeout(context.Background(), updateDryRunTimeOut)
	defer cancel()

	// If no change is detected in the spec nor the annotations - nothing to validate
	if reflect.DeepEqual(exists.Spec, requested.Spec) &&
		reflect.DeepEqual(exists.Annotations, requested.Annotations) {
		return nil
	}

	kv, cdi, cna, err := wh.getOperands(requested)
	if err != nil {
		return err
	}

	wg := sync.WaitGroup{}
	errorCh := make(chan error)
	done := make(chan bool)

	opts := &client.UpdateOptions{DryRun: []string{metav1.DryRunAll}}

	resources := []client.Object{
		kv,
		cdi,
		cna,
	}

	if wh.isOpenshift {
		ssp, _, err := operands.NewSSP(requested)
		if err != nil {
			return err
		}
		resources = append(resources, ssp)
	}

	wg.Add(len(resources))

	go func() {
		wg.Wait()
		close(done)
	}()

	for _, obj := range resources {
		go func(o client.Object, wgr *sync.WaitGroup) {
			defer wgr.Done()
			if err := wh.updateOperatorCr(ctx, requested, o, opts); err != nil {
				errorCh <- err
			}
		}(obj, &wg)
	}

	select {
	case err := <-errorCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		// just in case close(done) was selected while there is an error,
		// check the error channel again.
		if len(errorCh) != 0 {
			err := <-errorCh
			return err
		}

		if !dryrun {
			hcoTlsConfigCache = requested.Spec.TLSSecurityProfile
		}
		return nil
	}
}

func (wh WebhookHandler) updateOperatorCr(ctx context.Context, hc *v1beta1.HyperConverged, exists client.Object, opts *client.UpdateOptions) error {
	err := hcoutil.GetRuntimeObject(ctx, wh.cli, exists, wh.logger)
	if err != nil {
		wh.logger.Error(err, "failed to get object from kubernetes", "kind", exists.GetObjectKind())
		return err
	}

	switch existing := exists.(type) {
	case *kubevirtcorev1.KubeVirt:
		required, err := operands.NewKubeVirt(hc)
		if err != nil {
			return err
		}
		required.Spec.DeepCopyInto(&existing.Spec)

	case *cdiv1beta1.CDI:
		required, err := operands.NewCDI(hc)
		if err != nil {
			return err
		}
		required.Spec.DeepCopyInto(&existing.Spec)

	case *networkaddonsv1.NetworkAddonsConfig:
		required, err := operands.NewNetworkAddons(hc)
		if err != nil {
			return err
		}
		required.Spec.DeepCopyInto(&existing.Spec)

	case *sspv1beta1.SSP:
		required, _, err := operands.NewSSP(hc)
		if err != nil {
			return err
		}
		required.Spec.DeepCopyInto(&existing.Spec)

	case *ttov1alpha1.TektonTasks:
		required := operands.NewTTO(hc)
		if err != nil {
			return err
		}
		required.Spec.DeepCopyInto(&existing.Spec)

	}

	if err = wh.cli.Update(ctx, exists, opts); err != nil {
		wh.logger.Error(err, "failed to dry-run update the object", "kind", exists.GetObjectKind())
		return err
	}

	wh.logger.Info("dry-run update the object passed", "kind", exists.GetObjectKind())
	return nil
}

func (wh WebhookHandler) ValidateDelete(ctx context.Context, dryrun bool, hc *v1beta1.HyperConverged) error {
	wh.logger.Info("Validating delete", "name", hc.Name, "namespace", hc.Namespace)

	kv := operands.NewKubeVirtWithNameOnly(hc)
	cdi := operands.NewCDIWithNameOnly(hc)

	for _, obj := range []client.Object{
		kv,
		cdi,
	} {
		_, err := hcoutil.EnsureDeleted(ctx, wh.cli, obj, hc.Name, wh.logger, true, false, true)
		if err != nil {
			wh.logger.Error(err, "Delete validation failed", "GVK", obj.GetObjectKind().GroupVersionKind())
			return err
		}
	}
	if !dryrun {
		hcoTlsConfigCache = nil
	}
	return nil
}

func (wh WebhookHandler) validateCertConfig(hc *v1beta1.HyperConverged) error {
	minimalDuration := metav1.Duration{Duration: 10 * time.Minute}

	ccValues := make(map[string]time.Duration)
	ccValues["spec.certConfig.ca.duration"] = hc.Spec.CertConfig.CA.Duration.Duration
	ccValues["spec.certConfig.ca.renewBefore"] = hc.Spec.CertConfig.CA.RenewBefore.Duration
	ccValues["spec.certConfig.server.duration"] = hc.Spec.CertConfig.Server.Duration.Duration
	ccValues["spec.certConfig.server.renewBefore"] = hc.Spec.CertConfig.Server.RenewBefore.Duration

	for key, value := range ccValues {
		if value < minimalDuration.Duration {
			return fmt.Errorf("%v: value is too small", key)
		}
	}

	if hc.Spec.CertConfig.CA.Duration.Duration < hc.Spec.CertConfig.CA.RenewBefore.Duration {
		return errors.New("spec.certConfig.ca: duration is smaller than renewBefore")
	}

	if hc.Spec.CertConfig.Server.Duration.Duration < hc.Spec.CertConfig.Server.RenewBefore.Duration {
		return errors.New("spec.certConfig.server: duration is smaller than renewBefore")
	}

	if hc.Spec.CertConfig.CA.Duration.Duration < hc.Spec.CertConfig.Server.Duration.Duration {
		return errors.New("spec.certConfig: ca.duration is smaller than server.duration")
	}

	return nil
}

func (wh WebhookHandler) validateDataImportCronTemplates(hc *v1beta1.HyperConverged) error {

	for _, dict := range hc.Spec.DataImportCronTemplates {
		val, ok := dict.Annotations[hcoutil.DataImportCronEnabledAnnotation]
		val = strings.ToLower(val)
		if ok && !(val == "false" || val == "true") {
			return fmt.Errorf(`the %s annotation of a dataImportCronTemplate must be either "true" or "false"`, hcoutil.DataImportCronEnabledAnnotation)
		}

		enabled := !ok || val == "true"

		if enabled && dict.Spec == nil {
			return fmt.Errorf("dataImportCronTemplate spec is empty for an enabled DataImportCronTemplate")
		}
	}

	return nil
}

func (wh WebhookHandler) validateTlsSecurityProfiles(hc *v1beta1.HyperConverged) error {
	tlsSP := hc.Spec.TLSSecurityProfile

	if tlsSP == nil || tlsSP.Custom == nil {
		return nil
	}

	if tlsSP.Custom.MinTLSVersion < openshiftconfigv1.VersionTLS13 && !hasRequiredHTTP2Ciphers(tlsSP.Custom.Ciphers) {
		return fmt.Errorf("http2: TLSConfig.CipherSuites is missing an HTTP/2-required AES_128_GCM_SHA256 cipher (need at least one of ECDHE-RSA-AES128-GCM-SHA256 or ECDHE-ECDSA-AES128-GCM-SHA256)")
	}

	return nil
}

func hasRequiredHTTP2Ciphers(ciphers []string) bool {
	var requiredHTTP2Ciphers = []string{
		"ECDHE-RSA-AES128-GCM-SHA256",
		"ECDHE-ECDSA-AES128-GCM-SHA256",
	}

	// lo.Some returns true if at least 1 element of a subset is contained into a collection
	return lo.Some[string](requiredHTTP2Ciphers, ciphers)
}

func (wh WebhookHandler) MutateTLSConfig(cfg *tls.Config) {
	// This callback executes on each client call returning a new config to be used
	// please be aware that the APIServer is using http keepalive so this is going to
	// be executed only after a while for fresh connections and not on existing ones
	cfg.GetConfigForClient = func(_ *tls.ClientHelloInfo) (*tls.Config, error) {
		cipherNames, minTypedTLSVersion := wh.selectCipherSuitesAndMinTLSVersion()
		cfg.CipherSuites = crypto.CipherSuitesOrDie(crypto.OpenSSLToIANACipherSuites(cipherNames))
		cfg.MinVersion = crypto.TLSVersionOrDie(string(minTypedTLSVersion))
		return cfg, nil
	}
}

func (wh WebhookHandler) selectCipherSuitesAndMinTLSVersion() ([]string, openshiftconfigv1.TLSProtocolVersion) {
	ci := hcoutil.GetClusterInfo()
	profile := ci.GetTLSSecurityProfile(hcoTlsConfigCache)

	if profile.Custom != nil {
		return profile.Custom.TLSProfileSpec.Ciphers, profile.Custom.TLSProfileSpec.MinTLSVersion
	}

	return openshiftconfigv1.TLSProfiles[profile.Type].Ciphers, openshiftconfigv1.TLSProfiles[profile.Type].MinTLSVersion
}

// validationResponseFromStatus returns a response for admitting a request with provided Status object.
func validationResponseFromStatus(allowed bool, status metav1.Status) admission.Response {
	resp := admission.Response{
		AdmissionResponse: admissionv1.AdmissionResponse{
			Allowed: allowed,
			Result:  &status,
		},
	}
	return resp
}
