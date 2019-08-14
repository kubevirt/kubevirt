package packagemanifest

import (
	"context"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"

	"github.com/operator-framework/operator-lifecycle-manager/pkg/package-server/apis/packagemanifest/v1alpha1"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/package-server/provider"
)

type PackageManifestStorage struct {
	groupResource schema.GroupResource
	prov          provider.PackageManifestProvider
}

var _ rest.KindProvider = &PackageManifestStorage{}
var _ rest.Storage = &PackageManifestStorage{}
var _ rest.Getter = &PackageManifestStorage{}
var _ rest.Lister = &PackageManifestStorage{}
var _ rest.Scoper = &PackageManifestStorage{}

// NewStorage returns an in-memory implementation of storage.Interface.
func NewStorage(groupResource schema.GroupResource, prov provider.PackageManifestProvider) *PackageManifestStorage {
	return &PackageManifestStorage{
		groupResource: groupResource,
		prov:          prov,
	}
}

// Storage interface
func (m *PackageManifestStorage) New() runtime.Object {
	return &v1alpha1.PackageManifest{}
}

// KindProvider interface
func (m *PackageManifestStorage) Kind() string {
	return "PackageManifest"
}

// Lister interface
func (m *PackageManifestStorage) NewList() runtime.Object {
	return &v1alpha1.PackageManifestList{}
}

// Lister interface
func (m *PackageManifestStorage) List(ctx context.Context, options *metainternalversion.ListOptions) (runtime.Object, error) {
	// get namespace
	namespace := genericapirequest.NamespaceValue(ctx)

	// get selectors
	labelSelector := labels.Everything()
	if options != nil && options.LabelSelector != nil {
		labelSelector = options.LabelSelector
	}

	res, err := m.prov.ListPackageManifests(namespace)
	if err != nil {
		return &v1alpha1.PackageManifestList{}, err
	}

	// filter results by label
	filtered := []v1alpha1.PackageManifest{}
	for _, manifest := range res.Items {
		if labelSelector.Matches(labels.Set(manifest.GetLabels())) {
			filtered = append(filtered, manifest)
		}
	}

	res.Items = filtered
	return res, nil
}

// Getter interface
func (m *PackageManifestStorage) Get(ctx context.Context, name string, opts *metav1.GetOptions) (runtime.Object, error) {
	namespace := genericapirequest.NamespaceValue(ctx)
	manifest := v1alpha1.PackageManifest{}

	pm, err := m.prov.GetPackageManifest(namespace, name)
	if err != nil {
		return nil, err
	}
	if pm != nil {
		manifest = *pm
	} else {
		return nil, k8serrors.NewNotFound(m.groupResource, name)
	}

	return &manifest, nil
}

// Scoper interface
func (m *PackageManifestStorage) NamespaceScoped() bool {
	return true
}
