package util

// HCO common constants
const (
	OperatorNamespaceEnv   = "OPERATOR_NAMESPACE"
	OperatorWebhookModeEnv = "WEBHOOK_MODE"
	ContainerAppName       = "APP"
	ContainerOperatorApp   = "OPERATOR"
	ContainerWebhookApp    = "WEBHOOK"
	HcoKvIoVersionName     = "HCO_KV_IO_VERSION"
	KubevirtVersionEnvV    = "KUBEVIRT_VERSION"
	CdiVersionEnvV         = "CDI_VERSION"
	CnaoVersionEnvV        = "NETWORK_ADDONS_VERSION"
	SspVersionEnvV         = "SSP_VERSION"
	TtoVersionEnvV         = "TTO_VERSION"
	NmoVersionEnvV         = "NMO_VERSION"
	HppoVersionEnvV        = "HPPO_VERSION"
	KvUiPluginImageEnvV    = "KV_CONSOLE_PLUGIN_IMAGE"
	HcoValidatingWebhook   = "validate-hco.kubevirt.io"
	HcoMutatingWebhookNS   = "mutate-ns-hco.kubevirt.io"
	AppLabel               = "app"
	UndefinedNamespace     = ""
	OpenshiftNamespace     = "openshift"
	OperatorTestNamespace  = "test-operators"
	OperatorHubNamespace   = "operators"
	APIVersionAlpha        = "v1alpha1"
	APIVersionBeta         = "v1beta1"
	CurrentAPIVersion      = APIVersionBeta
	APIVersionGroup        = "hco.kubevirt.io"
	APIVersion             = APIVersionGroup + "/" + CurrentAPIVersion
	HyperConvergedKind     = "HyperConverged"
	// Recommended labels by Kubernetes. See
	// https://kubernetes.io/docs/concepts/overview/working-with-objects/common-labels/
	AppLabelPrefix    = "app.kubernetes.io"
	AppLabelVersion   = AppLabelPrefix + "/version"
	AppLabelManagedBy = AppLabelPrefix + "/managed-by"
	AppLabelPartOf    = AppLabelPrefix + "/part-of"
	AppLabelComponent = AppLabelPrefix + "/component"
	// Operator name for managed-by label
	OperatorName = "hco-operator"
	// Value for "part-of" label
	HyperConvergedCluster    = "hyperconverged-cluster"
	OpenshiftNodeSelectorAnn = "openshift.io/node-selector"
	KubernetesMetadataName   = "kubernetes.io/metadata.name"

	// HyperConvergedName is the name of the HyperConverged resource that will be reconciled
	HyperConvergedName          = "kubevirt-hyperconverged"
	MetricsHost                 = "0.0.0.0"
	MetricsPort           int32 = 8383
	HealthProbeHost             = "0.0.0.0"
	HealthProbePort       int32 = 6060
	ReadinessEndpointName       = "/readyz"
	LivenessEndpointName        = "/livez"
	HCOWebhookPath              = "/validate-hco-kubevirt-io-v1beta1-hyperconverged"
	HCONSWebhookPath            = "/mutate-ns-hco-kubevirt-io"
	DefaulterWebhookPath        = "/mutate-hco-kubevirt-io-v1beta1-hyperconverged"
	WebhookPort                 = 4343

	WebhookCertName       = "apiserver.crt"
	WebhookKeyName        = "apiserver.key"
	DefaultWebhookCertDir = "/apiserver.local.config/certificates"

	CliDownloadsServerPort       = 8080
	UiPluginServerPort     int32 = 9443
)

type AppComponent string

const (
	AppComponentCompute    AppComponent = "compute"
	AppComponentStorage    AppComponent = "storage"
	AppComponentImport     AppComponent = "import"
	AppComponentNetwork    AppComponent = "network"
	AppComponentMonitoring AppComponent = "monitoring"
	AppComponentSchedule   AppComponent = "schedule"
	AppComponentDeployment AppComponent = "deployment"
	AppComponentTekton     AppComponent = "tekton"
)
