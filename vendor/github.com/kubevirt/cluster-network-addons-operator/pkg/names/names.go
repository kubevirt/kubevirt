package names

// OperatorConfig is the name of the CRD that defines the complete
// operator configuration
const OPERATOR_CONFIG = "cluster"

// APPLIED_PREFIX is the prefix applied to the config maps
// where we store previously applied configuration
const APPLIED_PREFIX = "cluster-networks-addons-operator-applied-"

// REJECT_OWNER_ANNOTATION can be set on objects under data/ that should not be
// assigned with NetworkAddonsConfig as their owner. This can be used to prevent
// garbage collection deletion upon NetworkAddonsConfig removal.
const REJECT_OWNER_ANNOTATION = "networkaddonsoperator.network.kubevirt.io/rejectOwner"

const PROMETHEUS_LABEL_KEY = "prometheus.cnao.io"

// Relationship labels
const COMPONENT_LABEL_KEY = "app.kubernetes.io/component"
const PART_OF_LABEL_KEY = "app.kubernetes.io/part-of"
const VERSION_LABEL_KEY = "app.kubernetes.io/version"
const MANAGED_BY_LABEL_KEY = "app.kubernetes.io/managed-by"
const COMPONENT_LABEL_DEFAULT_VALUE = "network"
const MANAGED_BY_LABEL_DEFAULT_VALUE = "cnao-operator"
