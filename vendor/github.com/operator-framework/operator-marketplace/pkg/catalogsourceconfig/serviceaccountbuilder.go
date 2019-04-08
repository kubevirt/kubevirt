package catalogsourceconfig

import (
	marketplace "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	core "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ServiceAccountBuilder builds a new ServiceAccount object.
type ServiceAccountBuilder struct {
	sa core.ServiceAccount
}

// ServiceAccount returns a ServiceAccount object.
func (b *ServiceAccountBuilder) ServiceAccount() *core.ServiceAccount {
	return &b.sa
}

// WithTypeMeta sets basic TypeMeta.
func (b *ServiceAccountBuilder) WithTypeMeta() *ServiceAccountBuilder {
	b.sa.TypeMeta = meta.TypeMeta{
		Kind:       "ServiceAccount",
		APIVersion: "v1",
	}
	return b
}

// WithMeta sets basic TypeMeta and ObjectMeta.
func (b *ServiceAccountBuilder) WithMeta(name, namespace string) *ServiceAccountBuilder {
	b.WithTypeMeta()
	if b.sa.GetObjectMeta() == nil {
		b.sa.ObjectMeta = meta.ObjectMeta{}
	}
	b.sa.SetName(name)
	b.sa.SetNamespace(namespace)
	return b
}

// WithOwnerLabel sets the owner label of the ServiceAccount object to the given owner.
func (b *ServiceAccountBuilder) WithOwnerLabel(owner *marketplace.CatalogSourceConfig) *ServiceAccountBuilder {
	labels := map[string]string{
		CscOwnerNameLabel:      owner.Name,
		CscOwnerNamespaceLabel: owner.Namespace,
	}

	for key, value := range b.sa.GetLabels() {
		labels[key] = value
	}

	b.sa.SetLabels(labels)
	return b
}
