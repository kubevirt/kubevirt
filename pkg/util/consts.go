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
	NmoVersionEnvV         = "NMO_VERSION"
	HppoVersionEnvV        = "HPPO_VERSION"
	VMImportEnvV           = "VM_IMPORT_VERSION"
	HcoValidatingWebhook   = "validate-hco.kubevirt.io"
	HcoMutatingWebhookNS   = "mutate-ns-hco.kubevirt.io"
	AppLabel               = "app"
	UndefinedNamespace     = ""
	OpenshiftNamespace     = "openshift"
	OperatorTestNamespace  = "test-operators"
	APIVersionAlpha        = "v1alpha1"
	APIVersionBeta         = "v1beta1"
	CurrentAPIVersion      = APIVersionBeta
	APIVersionGroup        = "hco.kubevirt.io"
	APIVersion             = APIVersionGroup + "/" + APIVersionBeta
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
	HyperConvergedCluster = "hyperconverged-cluster"

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

type AppComponent string

const (
	AppComponentCompute    AppComponent = "compute"
	AppComponentStorage                 = "storage"
	AppComponentImport                  = "import"
	AppComponentNetwork                 = "network"
	AppComponentMonitoring              = "monitoring"
	AppComponentSchedule                = "schedule"
	AppComponentDeployment              = "deployment"
)
