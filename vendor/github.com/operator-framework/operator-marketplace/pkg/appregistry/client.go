package appregistry

import (
	"errors"
	"fmt"
	"strings"

	"github.com/operator-framework/operator-marketplace/pkg/datastore"
)

// Client exposes the functionality of app registry server
type Client interface {
	// RetrieveAll retrieves all visible packages from the given source
	// When namespace is specified, only package(s) associated with the given namespace are returned.
	// If namespace is empty then visible package(s) across all namespaces are returned.
	RetrieveAll(namespace string) ([]*datastore.OperatorMetadata, error)

	// RetrieveOne retrieves a given package from the source
	RetrieveOne(name, release string) (*datastore.OperatorMetadata, error)

	// ListPackages returns metadata associated with each package in the
	// specified namespace.
	ListPackages(namespace string) ([]*datastore.RegistryMetadata, error)
}

type client struct {
	adapter apprApiAdapter
	decoder blobDecoder
}

func (c *client) RetrieveAll(namespace string) ([]*datastore.OperatorMetadata, error) {
	packages, err := c.adapter.ListPackages(namespace)
	if err != nil {
		return nil, err
	}

	list := make([]*datastore.OperatorMetadata, len(packages))
	for i, pkg := range packages {
		manifest, err := c.RetrieveOne(pkg.Name, pkg.Default)
		if err != nil {
			return nil, err
		}

		list[i] = manifest
	}

	return list, nil
}

func (c *client) ListPackages(namespace string) ([]*datastore.RegistryMetadata, error) {
	packages, err := c.adapter.ListPackages(namespace)
	if err != nil {
		return nil, err
	}

	list := make([]*datastore.RegistryMetadata, len(packages))
	for i, pkg := range packages {
		namespace, repository, err := split(pkg.Name)
		if err != nil {
			return nil, err
		}

		metadata := &datastore.RegistryMetadata{
			Namespace:  namespace,
			Repository: repository,

			// 'Default' points to the latest release pushed.
			Release: pkg.Default,

			// Getting 'Digest' would require an additional call to the app
			// registry, so it is being defaulted.
		}

		list[i] = metadata
	}

	return list, nil
}

func (c *client) RetrieveOne(name, release string) (*datastore.OperatorMetadata, error) {
	namespace, repository, err := split(name)
	if err != nil {
		return nil, err
	}

	metadata, err := c.adapter.GetPackageMetadata(namespace, repository, release)
	if err != nil {
		return nil, err
	}

	digest := metadata.Content.Digest
	blob, err := c.adapter.DownloadOperatorManifest(namespace, repository, digest)
	if err != nil {
		return nil, err
	}

	decoded, err := c.decoder.Decode(blob)
	if err != nil {
		return nil, err
	}

	om := &datastore.OperatorMetadata{
		RegistryMetadata: datastore.RegistryMetadata{
			Namespace:  namespace,
			Repository: repository,
			Release:    release,
			Digest:     digest,
		},
		RawYAML: decoded,
	}

	return om, nil
}

func split(name string) (namespace string, repository string, err error) {
	// we expect package name to comply to this format - {namespace}/{repository}
	split := strings.Split(name, "/")
	if len(split) != 2 {
		return "", "", errors.New(fmt.Sprintf("package name should be specified in this format {namespace}/{repository}"))
	}

	namespace = split[0]
	repository = split[1]

	return namespace, repository, nil
}
