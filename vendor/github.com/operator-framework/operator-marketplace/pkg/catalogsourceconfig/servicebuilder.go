package catalogsourceconfig

import (
	marketplace "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ServiceBuilder builds a new CatalogSource object.
type ServiceBuilder struct {
	service core.Service
}

// Service returns a Service object.
func (b *ServiceBuilder) Service() *core.Service {
	return &b.service
}

// WithTypeMeta sets TypeMeta.
func (b *ServiceBuilder) WithTypeMeta() *ServiceBuilder {
	b.service.TypeMeta = metav1.TypeMeta{
		Kind:       "Service",
		APIVersion: "v1",
	}
	return b
}

// WithMeta sets TypeMeta and ObjectMeta.
func (b *ServiceBuilder) WithMeta(name, namespace string) *ServiceBuilder {
	b.WithTypeMeta()
	b.service.ObjectMeta = metav1.ObjectMeta{
		Name:      name,
		Namespace: namespace,
	}
	return b
}

// WithOwnerLabel sets the owner label of the CatalogSource object to the given owner.
func (b *ServiceBuilder) WithOwnerLabel(owner *marketplace.CatalogSourceConfig) *ServiceBuilder {
	labels := map[string]string{
		CscOwnerNameLabel:      owner.Name,
		CscOwnerNamespaceLabel: owner.Namespace,
	}

	for key, value := range b.service.GetLabels() {
		labels[key] = value
	}

	b.service.SetLabels(labels)
	return b
}

// WithSpec sets the Data.
func (b *ServiceBuilder) WithSpec(spec core.ServiceSpec) *ServiceBuilder {
	b.service.Spec = spec
	return b
}
