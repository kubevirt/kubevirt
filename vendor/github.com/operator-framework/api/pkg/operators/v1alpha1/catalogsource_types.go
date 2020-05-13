package v1alpha1

import (
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	CatalogSourceCRDAPIVersion = GroupName + "/" + GroupVersion
	CatalogSourceKind          = "CatalogSource"
)

// SourceType indicates the type of backing store for a CatalogSource
type SourceType string

const (
	// SourceTypeInternal (deprecated) specifies a CatalogSource of type SourceTypeConfigmap
	SourceTypeInternal SourceType = "internal"

	// SourceTypeConfigmap specifies a CatalogSource that generates a configmap-server registry
	SourceTypeConfigmap SourceType = "configmap"

	// SourceTypeGrpc specifies a CatalogSource that can use an operator registry image to generate a
	// registry-server or connect to a pre-existing registry at an address.
	SourceTypeGrpc SourceType = "grpc"
)

const (
	// CatalogSourceSpecInvalidError denotes when fields on the spec of the CatalogSource are not valid.
	CatalogSourceSpecInvalidError ConditionReason = "SpecInvalidError"
	// CatalogSourceConfigMapError denotes when there is an issue extracting manifests from the specified ConfigMap.
	CatalogSourceConfigMapError ConditionReason = "ConfigMapError"
	// CatalogSourceRegistryServerError denotes when there is an issue querying the specified registry server.
	CatalogSourceRegistryServerError ConditionReason = "RegistryServerError"
)

type CatalogSourceSpec struct {
	// SourceType is the type of source
	SourceType SourceType `json:"sourceType"`

	// ConfigMap is the name of the ConfigMap to be used to back a configmap-server registry.
	// Only used when SourceType = SourceTypeConfigmap or SourceTypeInternal.
	// +Optional
	ConfigMap string `json:"configMap,omitempty"`

	// Address is a host that OLM can use to connect to a pre-existing registry.
	// Format: <registry-host or ip>:<port>
	// Only used when SourceType = SourceTypeGrpc.
	// Ignored when the Image field is set.
	// +Optional
	Address string `json:"address,omitempty"`

	// Image is an operator-registry container image to instantiate a registry-server with.
	// Only used when SourceType = SourceTypeGrpc.
	// If present, the address field is ignored.
	// +Optional
	Image string `json:"image,omitempty"`

	// UpdateStrategy defines how updated catalog source images can be discovered
	// Consists of an interval that defines polling duration and an embedded strategy type
	// +Optional
	UpdateStrategy *UpdateStrategy `json:"updateStrategy,omitempty"`

	// Secrets represent set of secrets that can be used to access the contents of the catalog.
	// It is best to keep this list small, since each will need to be tried for every catalog entry.
	// +Optional
	Secrets []string `json:"secrets,omitempty"`

	// Metadata
	DisplayName string `json:"displayName,omitempty"`
	Description string `json:"description,omitempty"`
	Publisher   string `json:"publisher,omitempty"`
	Icon        Icon   `json:"icon,omitempty"`
}

// UpdateStrategy holds all the different types of catalog source update strategies
// Currently only registry polling strategy is implemented
type UpdateStrategy struct {
	*RegistryPoll `json:"registryPoll,omitempty"`
}

type RegistryPoll struct {
	// Interval is used to determine the time interval between checks of the latest catalog source version.
	// The catalog operator polls to see if a new version of the catalog source is available.
	// If available, the latest image is pulled and gRPC traffic is directed to the latest catalog source.
	Interval *metav1.Duration `json:"interval,omitempty"`
}

type RegistryServiceStatus struct {
	Protocol         string      `json:"protocol,omitempty"`
	ServiceName      string      `json:"serviceName,omitempty"`
	ServiceNamespace string      `json:"serviceNamespace,omitempty"`
	Port             string      `json:"port,omitempty"`
	CreatedAt        metav1.Time `json:"createdAt,omitempty"`
}

func (s *RegistryServiceStatus) Address() string {
	return fmt.Sprintf("%s.%s.svc:%s", s.ServiceName, s.ServiceNamespace, s.Port)
}

type GRPCConnectionState struct {
	Address           string      `json:"address,omitempty"`
	LastObservedState string      `json:"lastObservedState"`
	LastConnectTime   metav1.Time `json:"lastConnect,omitempty"`
}

type CatalogSourceStatus struct {
	// A human readable message indicating details about why the CatalogSource is in this condition.
	// +optional
	Message string `json:"message,omitempty"`
	// Reason is the reason the CatalogSource was transitioned to its current state.
	// +optional
	Reason ConditionReason `json:"reason,omitempty"`

	// The last time the CatalogSource image registry has been polled to ensure the image is up-to-date
	LatestImageRegistryPoll *metav1.Time `json:"latestImageRegistryPoll,omitempty"`

	ConfigMapResource     *ConfigMapResourceReference `json:"configMapReference,omitempty"`
	RegistryServiceStatus *RegistryServiceStatus      `json:"registryService,omitempty"`
	GRPCConnectionState   *GRPCConnectionState        `json:"connectionState,omitempty"`
}

type ConfigMapResourceReference struct {
	Name            string      `json:"name"`
	Namespace       string      `json:"namespace"`
	UID             types.UID   `json:"uid,omitempty"`
	ResourceVersion string      `json:"resourceVersion,omitempty"`
	LastUpdateTime  metav1.Time `json:"lastUpdateTime,omitempty"`
}

func (r *ConfigMapResourceReference) IsAMatch(object *metav1.ObjectMeta) bool {
	return r.UID == object.GetUID() && r.ResourceVersion == object.GetResourceVersion()
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient
// +kubebuilder:resource:shortName=catsrc,categories=olm
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Display",type=string,JSONPath=`.spec.displayName`,description="The pretty name of the catalog"
// +kubebuilder:printcolumn:name="Type",type=string,JSONPath=`.spec.sourceType`,description="The type of the catalog"
// +kubebuilder:printcolumn:name="Publisher",type=string,JSONPath=`.spec.publisher`,description="The publisher of the catalog"
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// CatalogSource is a repository of CSVs, CRDs, and operator packages.
type CatalogSource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec CatalogSourceSpec `json:"spec"`
	// +optional
	Status CatalogSourceStatus `json:"status"`
}

func (c *CatalogSource) Address() string {
	if c.Spec.Address != "" {
		return c.Spec.Address
	}
	return c.Status.RegistryServiceStatus.Address()
}

func (c *CatalogSource) SetError(reason ConditionReason, err error) {
	c.Status.Reason = reason
	c.Status.Message = ""
	if err != nil {
		c.Status.Message = err.Error()
	}
}

func (c *CatalogSource) SetLastUpdateTime() {
	now := metav1.Now()
	c.Status.LatestImageRegistryPoll = &now
}

// Check if it is time to update based on polling setting
func (c *CatalogSource) Update() bool {
	if !c.Poll() {
		return false
	}
	interval := c.Spec.UpdateStrategy.Interval.Duration
	latest := c.Status.LatestImageRegistryPoll
	if latest == nil {
		logrus.WithField("CatalogSource", c.Name).Debugf("latest poll %v", latest)
	} else {
		logrus.WithField("CatalogSource", c.Name).Debugf("latest poll %v", *c.Status.LatestImageRegistryPoll)
	}

	if c.Status.LatestImageRegistryPoll.IsZero() {
		logrus.WithField("CatalogSource", c.Name).Debugf("creation timestamp plus interval before now %t", c.CreationTimestamp.Add(interval).Before(time.Now()))
		if c.CreationTimestamp.Add(interval).Before(time.Now()) {
			return true
		}
	} else {
		logrus.WithField("CatalogSource", c.Name).Debugf("latest poll plus interval before now %t", c.Status.LatestImageRegistryPoll.Add(interval).Before(time.Now()))
		if c.Status.LatestImageRegistryPoll.Add(interval).Before(time.Now()) {
			return true
		}
	}

	return false
}

// Poll determines whether the polling feature is enabled on the particular catalog source
func (c *CatalogSource) Poll() bool {
	if c.Spec.UpdateStrategy == nil {
		return false
	}
	// if polling interval is zero polling will not be done
	if c.Spec.UpdateStrategy.RegistryPoll == nil {
		return false
	}
	// if catalog source is not backed by an image polling will not be done
	if c.Spec.Image == "" {
		return false
	}
	// if image is not type gRPC polling will not be done
	if c.Spec.SourceType != SourceTypeGrpc {
		return false
	}
	return true
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CatalogSourceList is a repository of CSVs, CRDs, and operator packages.
type CatalogSourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []CatalogSource `json:"items"`
}
