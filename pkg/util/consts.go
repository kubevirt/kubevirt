package util

// HCO common constants
const (
	OperatorNamespaceEnv   = "OPERATOR_NAMESPACE"
	OperatorWebhookModeEnv = "WEBHOOK_MODE"
	HcoKvIoVersionName     = "HCO_KV_IO_VERSION"
	KubevirtVersionEnvV    = "KUBEVIRT_VERSION"
	CdiVersionEnvV         = "CDI_VERSION"
	CnaoVersionEnvV        = "NETWORK_ADDONS_VERSION"
	SspVersionEnvV         = "SSP_VERSION"
	NmoVersionEnvV         = "NMO_VERSION"
	HppoVersionEnvV        = "HPPO_VERSION"
	VMImportEnvV           = "VM_IMPORT_VERSION"
	HcoValidatingWebhook   = "validate-hco.kubevirt.io"
	HcoMutatingWebhookNS   = "mutate-ns-hco.kubevirt.io"
	AppLabel               = "app"
	UndefinedNamespace     = ""
	OpenshiftNamespace     = "openshift"
	APIVersionAlpha        = "v1alpha1"
	APIVersionBeta         = "v1beta1"
	CurrentAPIVersion      = APIVersionBeta
	APIVersionGroup        = "hco.kubevirt.io"
	APIVersion             = APIVersionGroup + "/" + APIVersionBeta
	// HyperConvergedName is the name of the HyperConverged resource that will be reconciled
	HyperConvergedName          = "kubevirt-hyperconverged"
	MetricsHost                 = "0.0.0.0"
	MetricsPort           int32 = 8383
	OperatorMetricsPort   int32 = 8686
	HealthProbeHost             = "0.0.0.0"
	HealthProbePort       int32 = 6060
	ReadinessEndpointName       = "/readyz"
	LivenessEndpointName        = "/livez"
	HCOWebhookPath              = "/validate-hco-kubevirt-io-v1beta1-hyperconverged"
	HCONSWebhookPath            = "/mutate-ns-hco-kubevirt-io"
	WebhookPort                 = 4343
)
