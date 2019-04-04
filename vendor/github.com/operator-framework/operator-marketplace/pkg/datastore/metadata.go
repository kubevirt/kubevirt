package datastore

import (
	"fmt"
)

// OperatorMetadata encapsulates registry metadata and blob associated with
// an operator manifest.
//
// When an operator manifest is downloaded from a remote registry, it should be
// serialized into this type so that it can be further processed by datastore
// package.
type OperatorMetadata struct {
	// Metadata that uniquely identifies the given operator manifest in registry.
	RegistryMetadata RegistryMetadata

	// Operator manifest(s) in raw YAML format that contains a set of CRD(s),
	// CSV(s) and package(s).
	RawYAML []byte
}

// Repository holds metadata associated with a repository in remote registry and
// the operator package name associated with the repository.
//
// We need this object to relate the operator package that user subscribes to a
// given repository in remote registry.
type Repository struct {
	// Metadata that uniquely identifies the given operator manifest in registry.
	Metadata RegistryMetadata

	// Package is the operator package name associated with the
	// given repository.
	Package string

	// Since we enforce that each package repository can only be in one datasource, we can link opsrc here
	// This is the metadata that uniquely identifies the Operator Source for this repository
	Opsrc *OpsrcRef
}

// OpsrcRef defines the endpoint, registry namespace and secret for a given
// OperatorSource.
type OpsrcRef struct {
	// Endpoint points to the remote app registry server from
	// where operator manifests can be fetched.
	Endpoint string

	// RegistryNamespace refers to the namespace in app registry. Only operator
	// manifests under this namespace will be visible.
	// Please note that this is not a k8s namespace.
	RegistryNamespace string

	// SecretNamespacedName is the name of the kubernetes Secret object.
	SecretNamespacedName string
}

// RegistryMetadata encapsulates metadata that uniquely describes the source of
// the given operator manifest in registry.
type RegistryMetadata struct {
	// Namespace is the namespace in application registry server
	// under which the given operator manifest is hosted.
	Namespace string

	// Repository is the repository that contains the given operator manifest.
	// The repository is located under the given namespace in application
	// registry.
	Repository string

	// Release represents the version number of the given operator manifest.
	Release string

	// Digest is the sha256 hash value that uniquely corresponds to the blob
	// associated with this particular release of the operator manifest.
	Digest string
}

// ID returns the unique identifier associated with this operator manifest.
func (rm *RegistryMetadata) ID() string {
	return fmt.Sprintf("%s/%s", rm.Namespace, rm.Repository)
}
