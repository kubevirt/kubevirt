//go:generate counterfeiter -o ../../../fakes/fake_reconciler_reconciler.go . ReconcilerFactory
package reconciler

import (
	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/operatorclient"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/operatorlister"
)

type RegistryReconciler interface {
	EnsureRegistryServer(catalogSource *v1alpha1.CatalogSource) error
}

type ReconcilerFactory interface {
	ReconcilerForSource(source *v1alpha1.CatalogSource) RegistryReconciler
}

type RegistryReconcilerFactory struct {
	Lister               operatorlister.OperatorLister
	OpClient             operatorclient.ClientInterface
	ConfigMapServerImage string
}

func (r *RegistryReconcilerFactory) ReconcilerForSource(source *v1alpha1.CatalogSource) RegistryReconciler {
	switch source.Spec.SourceType {
	case v1alpha1.SourceTypeInternal, v1alpha1.SourceTypeConfigmap:
		return &ConfigMapRegistryReconciler{
			Lister:   r.Lister,
			OpClient: r.OpClient,
			Image:    r.ConfigMapServerImage,
		}
	case v1alpha1.SourceTypeGrpc:
		if source.Spec.Image != "" {
			return &GrpcRegistryReconciler{
				Lister:   r.Lister,
				OpClient: r.OpClient,
			}
		} else if source.Spec.Address != "" {
			return &GrpcAddressRegistryReconciler{}
		}
	}
	return nil
}
