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
