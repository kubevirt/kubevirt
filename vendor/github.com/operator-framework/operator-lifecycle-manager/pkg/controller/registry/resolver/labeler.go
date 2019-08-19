package resolver

import (
	"github.com/operator-framework/operator-registry/pkg/registry"
	extv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/labels"
)

const (
	// APILabelKeyPrefix is the key prefix for a CSV's APIs label
	APILabelKeyPrefix = "olm.api."
)

// LabelSetsFor returns API label sets for the given object.
// Concrete types other than OperatorSurface and CustomResource definition no-op.
func LabelSetsFor(obj interface{}) ([]labels.Set, error) {
	switch v := obj.(type) {
	case OperatorSurface:
		return labelSetsForOperatorSurface(v)
	case *extv1beta1.CustomResourceDefinition:
		return labelSetsForCRD(v)
	default:
		return nil, nil
	}
}

func labelSetsForOperatorSurface(surface OperatorSurface) ([]labels.Set, error) {
	labelSet := labels.Set{}
	for key := range surface.ProvidedAPIs().StripPlural() {
		hash, err := APIKeyToGVKHash(key)
		if err != nil {
			return nil, err
		}
		labelSet[APILabelKeyPrefix+hash] = "provided"
	}
	for key := range surface.RequiredAPIs().StripPlural() {
		hash, err := APIKeyToGVKHash(key)
		if err != nil {
			return nil, err
		}
		labelSet[APILabelKeyPrefix+hash] = "required"
	}

	return []labels.Set{labelSet}, nil
}

func labelSetsForCRD(crd *extv1beta1.CustomResourceDefinition) ([]labels.Set, error) {
	labelSets := []labels.Set{}
	if crd == nil {
		return labelSets, nil
	}

	// Add label sets for each version
	for _, version := range crd.Spec.Versions {
		hash, err := APIKeyToGVKHash(registry.APIKey{
			Group:   crd.Spec.Group,
			Version: version.Name,
			Kind:    crd.Spec.Names.Kind,
		})
		if err != nil {
			return nil, err
		}
		key := APILabelKeyPrefix + hash
		sets := []labels.Set{
			{
				key: "provided",
			},
			{
				key: "required",
			},
		}
		labelSets = append(labelSets, sets...)
	}

	return labelSets, nil
}
