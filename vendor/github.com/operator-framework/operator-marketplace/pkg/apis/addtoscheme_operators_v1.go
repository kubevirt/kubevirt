package apis

import (
	marketplace "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
)

func init() {
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	AddToSchemes = append(AddToSchemes, marketplace.SchemeBuilder.AddToScheme)
}
