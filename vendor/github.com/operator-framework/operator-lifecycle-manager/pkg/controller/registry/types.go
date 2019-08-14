//go:generate counterfeiter -o ../../fakes/fake_registry_source.go types.go Source

package registry

import (
	"fmt"

	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"

	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
)

// Catalog Source
//    - Map name to ClusterServiceVersion
//    - Map CRD to CRD definition
//    - Map CRD to ClusterServiceVersion that manages it

type Source interface {
	FindCSVForPackageNameUnderChannel(packageName string, channelName string) (*v1alpha1.ClusterServiceVersion, error)
	FindReplacementCSVForPackageNameUnderChannel(packageName string, channelName string, csvName string) (*v1alpha1.ClusterServiceVersion, error)
	AllPackages() map[string]PackageManifest

	// Deprecated: Switch to FindReplacementCSVForPackageNameUnderChannel when the caller has package and channel
	// information.
	FindReplacementCSVForName(name string) (*v1alpha1.ClusterServiceVersion, error)

	FindCSVByName(name string) (*v1alpha1.ClusterServiceVersion, error)
	ListServices() ([]v1alpha1.ClusterServiceVersion, error)

	FindCRDByKey(key CRDKey) (*v1beta1.CustomResourceDefinition, error)
	ListLatestCSVsForCRD(key CRDKey) ([]CSVAndChannelInfo, error)
}

// ResourceKey contains metadata to uniquely identify a resource
type ResourceKey struct {
	Name      string
	Kind      string
	Namespace string
}

// SourceRef associates a Source with it's SourceKey
type SourceRef struct {
	SourceKey ResourceKey
	Source    Source
}

// CRDKey contains metadata needed to uniquely identify a CRD
type CRDKey struct {
	Kind    string
	Name    string
	Version string
}

func (k CRDKey) String() string {
	return fmt.Sprintf("%s/%s/%s", k.Kind, k.Name, k.Version)
}

// CSVAndChannelInfo holds information about a CSV and the channel in which it lives.
type CSVAndChannelInfo struct {
	// CSV is the CSV found.
	CSV *v1alpha1.ClusterServiceVersion

	// Channel is the channel that "contains" this CSV, as it is declared as part of the channel.
	Channel PackageChannel

	// IsDefaultChannel returns true iff the channel is the default channel for the package.
	IsDefaultChannel bool
}

// CSVMetadata holds the necessary information to locate a particular CSV in the catalog
type CSVMetadata struct {
	Name    string
	Version string
}

// PackageManifest holds information about a package, which is a reference to one (or more)
// channels under a single package.
type PackageManifest struct {
	// PackageName is the name of the overall package, ala `etcd`.
	PackageName string `json:"packageName"`

	// Channels are the declared channels for the package, ala `stable` or `alpha`.
	Channels []PackageChannel `json:"channels"`

	// DefaultChannelName is, if specified, the name of the default channel for the package. The
	// default channel will be installed if no other channel is explicitly given. If the package
	// has a single channel, then that channel is implicitly the default.
	DefaultChannelName string `json:"defaultChannel"`
}

// GetDefaultChannel gets the default channel or returns the only one if there's only one. returns empty string if it
// can't determine the default
func (m PackageManifest) GetDefaultChannel() string {
	if m.DefaultChannelName != "" {
		return m.DefaultChannelName
	}
	if len(m.Channels) == 1 {
		return m.Channels[0].Name
	}
	return ""
}

// PackageChannel defines a single channel under a package, pointing to a version of that
// package.
type PackageChannel struct {
	// Name is the name of the channel, e.g. `alpha` or `stable`
	Name string `json:"name"`

	// CurrentCSVName defines a reference to the CSV holding the version of this package currently
	// for the channel.
	CurrentCSVName string `json:"currentCSV"`
}

// IsDefaultChannel returns true if the PackageChennel is the default for the PackageManifest
func (pc PackageChannel) IsDefaultChannel(pm PackageManifest) bool {
	return pc.Name == pm.DefaultChannelName || len(pm.Channels) == 1
}

type SubscriptionKey struct {
	Name      string
	Namespace string
}
