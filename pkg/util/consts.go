package util

// HCO common constants
const (
	OperatorNamespaceEnv = "OPERATOR_NAMESPACE"
	HcoKvIoVersionName   = "HCO_KV_IO_VERSION"
	KubevirtVersionEnvV  = "KUBEVIRT_VERSION"
	CdiVersionEnvV       = "CDI_VERSION"
	CnaoVersionEnvV      = "NETWORK_ADDONS_VERSION"
	SspVersionEnvV       = "SSP_VERSION"
	NmoVersionEnvV       = "NMO_VERSION"
	HppoVersionEnvV      = "HPPO_VERSION"
	VMImportEnvV         = "VM_IMPORT_VERSION"
	HcoValidatingWebhook = "validate-hco.kubevirt.io"
	AppLabel             = "app"
	UndefinedNamespace   = ""
	OpenshiftNamespace   = "openshift"
	APIVersionAlpha      = "v1alpha1"
	APIVersionBeta       = "v1beta1"
	APIVersionGroup      = "hco.kubevirt.io"
	APIVersion           = APIVersionGroup + "/" + APIVersionBeta
)
