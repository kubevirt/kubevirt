package datastore

import (
	"errors"
	"fmt"
	"strings"

	"github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	"k8s.io/apimachinery/pkg/types"
)

var (
	Cache *memoryDatastore
)

func init() {
	// Cache is the global instance of datastore used by
	// the Marketplace operator.
	Cache = New()
}

// DatastoreLabel is the label used to indicate that the resulting CatalogSource
// acts as the datastore for the OperatorSource if it is set to "true".
const DatastoreLabel string = "opsrc-datastore"

// New returns an instance of memoryDatastore.
func New() *memoryDatastore {
	return &memoryDatastore{
		rows: newOperatorSourceRowMap(),
	}
}

// Reader is the interface that wraps the Read method.
type Reader interface {
	// Read takes a package identifier and returns metadata defined in the opsrc
	// spec to associate that package to a specific opsrc in the marketplace.
	// The OpsrcRef returned can be used to determine how to download manifests
	// for a specific operator package.
	Read(packageID string) (opsrcMeta *OpsrcRef, err error)

	// ReadVersion takes a package identifer and returns version metadata
	// to associate that package to a particular repository version.
	ReadRepositoryVersion(packageID string) (version string, err error)

	// CheckPackages returns an error if there are packages missing from the
	// datastore but listed in the spec.
	CheckPackages(packageIDs []string) error
}

// Writer is an interface that is used to manage the underlying datastore
// for operator manifest.
type Writer interface {
	// GetPackageIDs returns a comma separated list of operator ID(s). This list
	// includes operator(s) across all OperatorSource object(s). Each ID
	// returned can be used to retrieve the manifest associated with the
	// operator from underlying datastore.
	GetPackageIDs() string

	// GetPackageIDsByOperatorSource returns a comma separated list of operator
	// ID(s) associated with a given OperatorSource object.
	// Each ID returned can be used to retrieve the manifest associated with the
	// operator from underlying datastore.
	GetPackageIDsByOperatorSource(opsrcUID types.UID) string

	// Write saves the Spec associated with a given OperatorSource object and
	// the downloaded registry metadata into datastore.
	//
	// opsrc represents the given OperatorSource object.
	// rawManifests is the list of registry metadata associated with
	// a given operator source.
	//
	// On return, count is set to the number of registry metadata blobs
	// successfully processed and stored in datastore.
	// err will be set to nil if there was no error and all manifests were
	// processed and stored successfully.
	Write(opsrc *v1.OperatorSource, rawManifests []*RegistryMetadata) (count int, err error)

	// RemoveOperatorSource removes everything associated with a given operator
	// source from the underlying datastore.
	//
	// opsrcUID is the unique identifier associated with a given operator source.
	RemoveOperatorSource(opsrcUID types.UID)

	// AddOperatorSource registers a new OperatorSource object with the
	// the underlying datastore.
	AddOperatorSource(opsrc *v1.OperatorSource)

	// GetOperatorSource returns the Spec of the OperatorSource object
	// associated with the UID specified in opsrcUID.
	//
	// datastore uses the UID of the given OperatorSource object to check if
	// a Spec already exists. If no Spec is found then the function
	// returns (nil, false).
	GetOperatorSource(opsrcUID types.UID) (key *OperatorSourceKey, ok bool)

	// OperatorSourceHasUpdate returns true if the operator source in remote
	// registry specified in metadata has update(s) that need to be pulled.
	//
	// The function returns true if the remote registry has any update(s). The
	// following event(s) indicate that a remote registry has been updated.
	//   - New repositories have been added to the remote registry associated
	//     with the operator source.
	//   - Existing repositories have been removed from the remote registry
	//     associated with the operator source.
	//   - A new release for an existing repository has been pushed to
	//     the registry.
	//
	// Right now we consider remote and local operator source to be same only
	// when the following conditions are true:
	//
	// - Number of repositories in both local and remote are exactly the same.
	// - Each repository in remote has a corresponding local repository with
	//   exactly the same release.
	//
	// The current implementation does not return update information specific
	// to each repository. The lack of granular (per repository) information
	// will force us to reload the entire namespace.
	OperatorSourceHasUpdate(opsrcUID types.UID, metadata []*RegistryMetadata) (result *UpdateResult, err error)

	// GetAllOperatorSources returns a list of all OperatorSource objecs(s) that
	// datastore is aware of.
	GetAllOperatorSources() []*OperatorSourceKey
}

// memoryDatastore is an in-memory implementation of operator manifest datastore.
// TODO: In future, it will be replaced by an indexable persistent datastore.
type memoryDatastore struct {
	rows *operatorSourceRowMap
}

func (ds *memoryDatastore) Read(packageID string) (opsrcMeta *OpsrcRef, err error) {
	repository, err := ds.GetRepositoryByPackageName(packageID)
	if err != nil {
		return
	}

	opsrcMeta = repository.Opsrc

	return
}

func (ds *memoryDatastore) ReadRepositoryVersion(packageID string) (version string, err error) {
	repository, err := ds.GetRepositoryByPackageName(packageID)
	if err != nil {
		return
	}

	version = repository.Metadata.Release

	return
}

func (ds *memoryDatastore) Write(opsrc *v1.OperatorSource, registryMetas []*RegistryMetadata) (count int, err error) {
	if opsrc == nil || registryMetas == nil {
		err = errors.New("invalid argument")
		return
	}

	repositories := map[string]*Repository{}

	for _, metadata := range registryMetas {
		// For each repository store the associated registry metadata.
		secretNamespacedName := ""
		if opsrc.Spec.AuthorizationToken.SecretName != "" {
			secretNamespacedName = opsrc.Namespace + "/" + opsrc.Spec.AuthorizationToken.SecretName
		}

		repository := &Repository{
			Metadata: *metadata,
			Package:  metadata.Repository,
			Opsrc: &OpsrcRef{
				Endpoint:             opsrc.Spec.Endpoint,
				RegistryNamespace:    opsrc.Spec.RegistryNamespace,
				SecretNamespacedName: secretNamespacedName,
			},
		}
		repositories[metadata.Repository] = repository
	}

	ds.rows.Add(opsrc, repositories)

	count = len(registryMetas)
	return
}

func (ds *memoryDatastore) GetRepositoryByPackageName(pkg string) (repository *Repository, err error) {
	repositories := ds.rows.GetAllRepositories()

	if len(repositories) == 0 {
		err = fmt.Errorf("Datastore is empty. No package metadata to return.")
		return
	}

	for _, repo := range repositories {
		if repo.Package == pkg {
			repository = repo
		}
	}
	if repository == nil {
		err = fmt.Errorf("datastore has no record of the specified package [%s]", pkg)
	}
	return
}

func (ds *memoryDatastore) GetPackageIDs() string {
	keys := ds.rows.GetAllPackages()
	return strings.Join(keys, ",")
}

func (ds *memoryDatastore) GetPackageIDsByOperatorSource(opsrcUID types.UID) string {
	row, exists := ds.rows.GetRow(opsrcUID)
	if !exists {
		return ""
	}

	packages := row.GetPackages()
	return strings.Join(packages, ",")
}

func (ds *memoryDatastore) CheckPackages(packageIDs []string) error {
	missingPackages := []string{}
	for _, packageID := range packageIDs {
		if _, err := ds.Read(packageID); err != nil {
			missingPackages = append(missingPackages, packageID)
			continue
		}
	}

	if len(missingPackages) > 0 {
		return fmt.Errorf(
			"Still resolving package(s) - %s. Please make sure these are valid packages.",
			strings.Join(missingPackages, ","),
		)
	}
	return nil
}

func (ds *memoryDatastore) AddOperatorSource(opsrc *v1.OperatorSource) {
	ds.rows.AddEmpty(opsrc)
}

func (ds *memoryDatastore) RemoveOperatorSource(uid types.UID) {
	ds.rows.Remove(uid)
}

func (ds *memoryDatastore) GetOperatorSource(opsrcUID types.UID) (*OperatorSourceKey, bool) {
	row, exists := ds.rows.GetRow(opsrcUID)
	if !exists {
		return nil, false
	}

	return &row.OperatorSourceKey, true
}

func (ds *memoryDatastore) OperatorSourceHasUpdate(opsrcUID types.UID, metadata []*RegistryMetadata) (result *UpdateResult, err error) {
	row, exists := ds.rows.GetRow(opsrcUID)
	if !exists {
		err = fmt.Errorf("datastore has no record of the specified OperatorSource [%s]", opsrcUID)
		return
	}

	result = newUpdateResult()

	for _, remote := range metadata {
		if remote.Release == "" {
			err = fmt.Errorf("Release not specified for repository [%s]", remote.ID())
			return
		}

		local, exists := row.Repositories[remote.Repository]
		if !exists {
			// This is a new repository that has been pushed. We do not know the
			// package(s) associated with it yet. The best we can do is indicate
			// that we have an update for the operator source.
			result.RegistryHasUpdate = true
			continue
		}

		if local.Metadata.Release != remote.Release {
			// Since the repository has gone through an update we consider that
			// all package(s) associated with it have a new version.
			// The version/release value specified here refers to the version of
			// the repository, not the actual version of an operator.
			result.Updated = append(result.Updated, row.GetPackages()...)
		}
	}

	remoteMap := map[string]*RegistryMetadata{}
	for _, remote := range metadata {
		remoteMap[remote.Repository] = remote
	}

	for repositoryName, local := range row.Repositories {
		_, exists := remoteMap[repositoryName]
		if !exists {
			// This repository has been removed from the remote registry.
			result.Removed = append(result.Removed, local.Package)
		}
	}

	if len(result.Removed) > 0 || len(result.Updated) > 0 {
		result.RegistryHasUpdate = true
	}

	return
}

func (ds *memoryDatastore) GetAllOperatorSources() []*OperatorSourceKey {
	return ds.rows.GetAllRows()
}
