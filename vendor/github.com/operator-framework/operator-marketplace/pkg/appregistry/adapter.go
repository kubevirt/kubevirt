package appregistry

import (
	"bytes"

	appr "github.com/operator-framework/go-appr/appregistry"
	appr_blobs "github.com/operator-framework/go-appr/appregistry/blobs"
	appr_package "github.com/operator-framework/go-appr/appregistry/package_appr"
	appr_models "github.com/operator-framework/go-appr/models"
)

const (
	mediaType string = "helm"
)

// This interface (internal to this package) encapsulates nitty gritty details of go-appr client bindings
type apprApiAdapter interface {
	// ListPackages returns a list of package(s) available to the user.
	// When namespace is specified, only package(s) associated with the given namespace are returned.
	// If namespace is empty then visible package(s) across all namespaces are returned.
	ListPackages(namespace string) (appr_models.Packages, error)

	// GetPackageMetadata returns metadata associated with a given package
	GetPackageMetadata(namespace string, repository string, release string) (*appr_models.Package, error)

	// DownloadOperatorManifest downloads the blob associated with a given digest that directly corresponds to a package release
	DownloadOperatorManifest(namespace string, repository string, digest string) ([]byte, error)
}

type apprApiAdapterImpl struct {
	client *appr.Appregistry
}

func (a *apprApiAdapterImpl) ListPackages(namespace string) (appr_models.Packages, error) {
	params := appr_package.NewListPackagesParams()

	if namespace != "" {
		params.SetNamespace(&namespace)
	}

	packages, err := a.client.PackageAppr.ListPackages(params)
	if err != nil {
		return nil, err
	}

	return packages.Payload, nil
}

func (a *apprApiAdapterImpl) GetPackageMetadata(namespace string, repository string, release string) (*appr_models.Package, error) {
	params := appr_package.NewShowPackageParams().
		WithNamespace(namespace).
		WithPackage(repository).
		WithRelease(release).
		WithMediaType(mediaType)

	pkg, err := a.client.PackageAppr.ShowPackage(params)
	if err != nil {
		return nil, err
	}

	return pkg.Payload, nil
}

func (a *apprApiAdapterImpl) DownloadOperatorManifest(namespace string, repository string, digest string) ([]byte, error) {
	params := appr_blobs.NewPullBlobParams().
		WithNamespace(namespace).
		WithPackage(repository).
		WithDigest(digest)

	writer := &bytes.Buffer{}
	_, err := a.client.Blobs.PullBlob(params, writer)
	if err != nil {
		return nil, err
	}

	return writer.Bytes(), nil
}
