package registry

import (
	genericregistry "k8s.io/apiserver/pkg/registry/generic/registry"
)

type REST struct {
	*genericregistry.Store
}
