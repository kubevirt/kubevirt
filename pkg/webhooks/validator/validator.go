package validator

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/go-logr/logr"
	openshiftconfigv1 "github.com/openshift/api/config/v1"
	"github.com/samber/lo"
	xsync "golang.org/x/sync/errgroup"
	admissionv1 "k8s.io/api/admission/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/strings/slices"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	networkaddonsv1 "github.com/kubevirt/cluster-network-addons-operator/pkg/apis/networkaddonsoperator/v1"
	kubevirtcorev1 "kubevirt.io/api/core/v1"
	cdiv1beta1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"
	sspv1beta2 "kubevirt.io/ssp-operator/api/v1beta2"

	"github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/operands"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
)

const (
	updateDryRunTimeOut = time.Second * 3
)

type ValidationWarning struct {
	warnings []string
}

func newValidationWarning(warnings []string) *ValidationWarning {
	return &ValidationWarning{
		warnings: warnings,
	}
}

func (v *ValidationWarning) Error() string {
	return ""
}

func (v *ValidationWarning) Warnings() []string {
	return v.warnings
}

type WebhookHandler struct {
	logger      logr.Logger
	cli         client.Client
	namespace   string
	isOpenshift bool
	decoder     admission.Decoder
}

var hcoTLSConfigCache *openshiftconfigv1.TLSSecurityProfile

func NewWebhookHandler(logger logr.Logger, cli client.Client, decoder admission.Decoder, namespace string, isOpenshift bool, hcoTLSSecurityProfile *openshiftconfigv1.TLSSecurityProfile) *WebhookHandler {
	hcoTLSConfigCache = hcoTLSSecurityProfile
	return &WebhookHandler{
		logger:      logger,
		cli:         cli,
		namespace:   namespace,
		isOpenshift: isOpenshift,
		decoder:     decoder,
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

		var vw *ValidationWarning
		if errors.As(err, &vw) {
			return admission.Allowed("").WithWarnings(vw.Warnings()...)
		}

		return admission.Denied(err.Error())
	}

	// Return allowed if everything succeeded.
	return admission.Allowed("")
}

func (wh *WebhookHandler) ValidateCreate(_ context.Context, dryrun bool, hc *v1beta1.HyperConverged) error {
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

	if err := wh.validateTLSSecurityProfiles(hc); err != nil {
		return err
	}

	if err := wh.validateMediatedDeviceTypes(hc); err != nil {
		return err
	}

	if err := wh.validateFeatureGatesOnCreate(hc); err != nil {
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

	if _, _, err := operands.NewSSP(hc); err != nil {
		return err
	}

	if !dryrun {
		hcoTLSConfigCache = hc.Spec.TLSSecurityProfile
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
func (wh *WebhookHandler) ValidateUpdate(ctx context.Context, dryrun bool, requested *v1beta1.HyperConverged, exists *v1beta1.HyperConverged) error {
	wh.logger.Info("Validating update", "name", requested.Name)

	if err := wh.validateDataImportCronTemplates(requested); err != nil {
		return err
	}

	if err := wh.validateTLSSecurityProfiles(requested); err != nil {
		return err
	}

	if err := wh.validateMediatedDeviceTypes(requested); err != nil {
		return err
	}

	if err := wh.validateFeatureGatesOnUpdate(requested, exists); err != nil {
		return err
	}

	// If no change is detected in the spec nor the annotations - nothing to validate
	if reflect.DeepEqual(exists.Spec, requested.Spec) &&
		reflect.DeepEqual(exists.Annotations, requested.Annotations) {
		return nil
	}

	kv, cdi, cna, err := wh.getOperands(requested)
	if err != nil {
		return err
	}

	toCtx, cancel := context.WithTimeout(ctx, updateDryRunTimeOut)
	defer cancel()

	eg, egCtx := xsync.WithContext(toCtx)
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

	for _, obj := range resources {
		func(o client.Object) {
			eg.Go(func() error {
				return wh.updateOperatorCr(egCtx, requested, o, opts)
			})
		}(obj)
	}

	err = eg.Wait()
	if err != nil {
		return err
	}

	if !dryrun {
		hcoTLSConfigCache = requested.Spec.TLSSecurityProfile
	}

	return nil
}

func (wh *WebhookHandler) updateOperatorCr(ctx context.Context, hc *v1beta1.HyperConverged, exists client.Object, opts *client.UpdateOptions) error {
	err := hcoutil.GetRuntimeObject(ctx, wh.cli, exists)
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

	case *sspv1beta2.SSP:
		required, _, err := operands.NewSSP(hc)
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

func (wh *WebhookHandler) ValidateDelete(ctx context.Context, dryrun bool, hc *v1beta1.HyperConverged) error {
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
		hcoTLSConfigCache = nil
	}
	return nil
}

func (wh *WebhookHandler) validateCertConfig(hc *v1beta1.HyperConverged) error {
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

func (wh *WebhookHandler) validateDataImportCronTemplates(hc *v1beta1.HyperConverged) error {

	for _, dict := range hc.Spec.DataImportCronTemplates {
		val, ok := dict.Annotations[hcoutil.DataImportCronEnabledAnnotation]
		val = strings.ToLower(val)
		if ok && val != "false" && val != "true" {
			return fmt.Errorf(`the %s annotation of a dataImportCronTemplate must be either "true" or "false"`, hcoutil.DataImportCronEnabledAnnotation)
		}

		enabled := !ok || val == "true"

		if enabled && dict.Spec == nil {
			return fmt.Errorf("dataImportCronTemplate spec is empty for an enabled DataImportCronTemplate")
		}
	}

	return nil
}

func (wh *WebhookHandler) validateTLSSecurityProfiles(hc *v1beta1.HyperConverged) error {
	tlsSP := hc.Spec.TLSSecurityProfile

	if tlsSP == nil || tlsSP.Custom == nil {
		return nil
	}

	if !isValidTLSProtocolVersion(tlsSP.Custom.MinTLSVersion) {
		return fmt.Errorf("invalid value for spec.tlsSecurityProfile.custom.minTLSVersion")
	}

	if tlsSP.Custom.MinTLSVersion < openshiftconfigv1.VersionTLS13 && !hasRequiredHTTP2Ciphers(tlsSP.Custom.Ciphers) {
		return fmt.Errorf("http2: TLSConfig.CipherSuites is missing an HTTP/2-required AES_128_GCM_SHA256 cipher (need at least one of ECDHE-RSA-AES128-GCM-SHA256 or ECDHE-ECDSA-AES128-GCM-SHA256)")
	} else if tlsSP.Custom.MinTLSVersion == openshiftconfigv1.VersionTLS13 && len(tlsSP.Custom.Ciphers) > 0 {
		return fmt.Errorf("custom ciphers cannot be selected when minTLSVersion is VersionTLS13")
	}

	return nil
}

func (wh *WebhookHandler) validateMediatedDeviceTypes(hc *v1beta1.HyperConverged) error {
	mdc := hc.Spec.MediatedDevicesConfiguration
	if mdc != nil {
		if len(mdc.MediatedDevicesTypes) > 0 && len(mdc.MediatedDeviceTypes) > 0 && !slices.Equal(mdc.MediatedDevicesTypes, mdc.MediatedDeviceTypes) { //nolint SA1019
			return fmt.Errorf("mediatedDevicesTypes is deprecated, please use mediatedDeviceTypes instead")
		}
		for _, nmdc := range mdc.NodeMediatedDeviceTypes {
			if len(nmdc.MediatedDevicesTypes) > 0 && len(nmdc.MediatedDeviceTypes) > 0 && !slices.Equal(nmdc.MediatedDevicesTypes, nmdc.MediatedDeviceTypes) { //nolint SA1019
				return fmt.Errorf("mediatedDevicesTypes is deprecated, please use mediatedDeviceTypes instead")
			}
		}
	}
	return nil
}

const (
	fgMovedWarning       = "spec.featureGates.%[1]s is deprecated and ignored. It will removed in a future version; use spec.%[1]s instead"
	fgDeprecationWarning = "spec.featureGates.%s is deprecated and ignored. It will be removed in a future version;"
)

func (wh *WebhookHandler) validateFeatureGatesOnCreate(hc *v1beta1.HyperConverged) error {
	warnings := wh.validateDeprecatedFeatureGates(hc)
	warnings = validateOldFGOnCreate(warnings, hc)

	if len(warnings) > 0 {
		return newValidationWarning(warnings)
	}

	return nil
}

func (wh *WebhookHandler) validateFeatureGatesOnUpdate(requested, exists *v1beta1.HyperConverged) error {
	warnings := wh.validateDeprecatedFeatureGates(requested)
	warnings = validateOldFGOnUpdate(warnings, requested, exists)

	if len(warnings) > 0 {
		return newValidationWarning(warnings)
	}

	return nil
}

func (wh *WebhookHandler) validateDeprecatedFeatureGates(hc *v1beta1.HyperConverged) []string {
	var warnings []string

	//nolint:staticcheck
	if hc.Spec.FeatureGates.WithHostPassthroughCPU != nil {
		warnings = append(warnings, fmt.Sprintf(fgDeprecationWarning, "withHostPassthroughCPU"))
	}

	//nolint:staticcheck
	if hc.Spec.FeatureGates.DeployTektonTaskResources != nil {
		warnings = append(warnings, fmt.Sprintf(fgDeprecationWarning, "deployTektonTaskResources"))
	}

	//nolint:staticcheck
	if hc.Spec.FeatureGates.NonRoot != nil {
		warnings = append(warnings, fmt.Sprintf(fgDeprecationWarning, "nonRoot"))
	}

	//nolint:staticcheck
	if hc.Spec.FeatureGates.EnableManagedTenantQuota != nil {
		warnings = append(warnings, fmt.Sprintf(fgDeprecationWarning, "enableManagedTenantQuota"))
	}

	return warnings
}

func validateOldFGOnCreate(warnings []string, hc *v1beta1.HyperConverged) []string {
	//nolint:staticcheck
	if hc.Spec.FeatureGates.EnableApplicationAwareQuota != nil {
		warnings = append(warnings, fmt.Sprintf(fgMovedWarning, "enableApplicationAwareQuota"))
	}

	//nolint:staticcheck
	if hc.Spec.FeatureGates.EnableCommonBootImageImport != nil {
		warnings = append(warnings, fmt.Sprintf(fgMovedWarning, "enableCommonBootImageImport"))
	}

	//nolint:staticcheck
	if hc.Spec.FeatureGates.DeployVMConsoleProxy != nil {
		warnings = append(warnings, fmt.Sprintf(fgMovedWarning, "deployVmConsoleProxy"))
	}

	//nolint:staticcheck
	if hc.Spec.FeatureGates.DeployKubeSecondaryDNS != nil {
		warnings = append(warnings, fmt.Sprintf(fgMovedWarning, "deployKubeSecondaryDNS"))
	}

	return warnings
}

func validateOldFGOnUpdate(warnings []string, hc, prevHC *v1beta1.HyperConverged) []string {
	//nolint:staticcheck
	if oldFGChanged(hc.Spec.FeatureGates.EnableApplicationAwareQuota, prevHC.Spec.FeatureGates.EnableApplicationAwareQuota) {
		warnings = append(warnings, fmt.Sprintf(fgMovedWarning, "enableApplicationAwareQuota"))
	}

	//nolint:staticcheck
	if oldFGChanged(hc.Spec.FeatureGates.EnableCommonBootImageImport, prevHC.Spec.FeatureGates.EnableCommonBootImageImport) {
		warnings = append(warnings, fmt.Sprintf(fgMovedWarning, "enableCommonBootImageImport"))
	}

	//nolint:staticcheck
	if oldFGChanged(hc.Spec.FeatureGates.DeployVMConsoleProxy, prevHC.Spec.FeatureGates.DeployVMConsoleProxy) {
		warnings = append(warnings, fmt.Sprintf(fgMovedWarning, "deployVmConsoleProxy"))
	}

	//nolint:staticcheck
	if oldFGChanged(hc.Spec.FeatureGates.DeployKubeSecondaryDNS, prevHC.Spec.FeatureGates.DeployKubeSecondaryDNS) {
		warnings = append(warnings, fmt.Sprintf(fgMovedWarning, "deployKubeSecondaryDNS"))
	}

	return warnings
}

func oldFGChanged(newFG, prevFG *bool) bool {
	return newFG != nil && (prevFG == nil || *newFG != *prevFG)
}

func hasRequiredHTTP2Ciphers(ciphers []string) bool {
	var requiredHTTP2Ciphers = []string{
		"ECDHE-RSA-AES128-GCM-SHA256",
		"ECDHE-ECDSA-AES128-GCM-SHA256",
	}

	// lo.Some returns true if at least 1 element of a subset is contained into a collection
	return lo.Some[string](requiredHTTP2Ciphers, ciphers)
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

func SelectCipherSuitesAndMinTLSVersion() ([]string, openshiftconfigv1.TLSProtocolVersion) {
	ci := hcoutil.GetClusterInfo()
	profile := ci.GetTLSSecurityProfile(hcoTLSConfigCache)

	if profile.Custom != nil {
		return profile.Custom.Ciphers, profile.Custom.MinTLSVersion
	}

	return openshiftconfigv1.TLSProfiles[profile.Type].Ciphers, openshiftconfigv1.TLSProfiles[profile.Type].MinTLSVersion
}

func isValidTLSProtocolVersion(pv openshiftconfigv1.TLSProtocolVersion) bool {
	switch pv {
	case
		openshiftconfigv1.VersionTLS10,
		openshiftconfigv1.VersionTLS11,
		openshiftconfigv1.VersionTLS12,
		openshiftconfigv1.VersionTLS13:
		return true
	}
	return false
}
