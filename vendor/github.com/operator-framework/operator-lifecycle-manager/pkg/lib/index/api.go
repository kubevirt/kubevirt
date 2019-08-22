package indexer

import (
	"fmt"
	"strings"

	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	v1beta1ext "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/client-go/tools/cache"
)

const (
	// ProvidedAPIsIndexFuncKey is the recommended key to use for registering the index func with an indexer.
	ProvidedAPIsIndexFuncKey string = "providedAPIs"
)

// ProvidedAPIsIndexFunc returns indicies from the owned CRDs and APIs of the given object (CSV)
func ProvidedAPIsIndexFunc(obj interface{}) ([]string, error) {
	indicies := []string{}

	csv, ok := obj.(*v1alpha1.ClusterServiceVersion)
	if !ok {
		return indicies, fmt.Errorf("invalid object of type: %T", obj)
	}

	for _, crd := range csv.Spec.CustomResourceDefinitions.Owned {
		parts := strings.SplitN(crd.Name, ".", 2)
		if len(parts) < 2 {
			return indicies, fmt.Errorf("couldn't parse plural.group from crd name: %s", crd.Name)
		}
		indicies = append(indicies, fmt.Sprintf("%s/%s/%s", parts[1], crd.Version, crd.Kind))
	}
	for _, api := range csv.Spec.APIServiceDefinitions.Owned {
		indicies = append(indicies, fmt.Sprintf("%s/%s/%s", api.Group, api.Version, api.Kind))
	}

	return indicies, nil
}

// CRDProviderNames returns the names of CSVs that own the given CRD
func CRDProviderNames(indexers map[string]cache.Indexer, crd v1beta1ext.CustomResourceDefinition) (map[string]struct{}, error) {
	csvSet := map[string]struct{}{}
	crdSpec := map[string]struct{}{}
	for _, v := range crd.Spec.Versions {
		crdSpec[fmt.Sprintf("%s/%s/%s", crd.Spec.Group, v.Name, crd.Spec.Names.Kind)] = struct{}{}
	}
	if crd.Spec.Version != "" {
		crdSpec[fmt.Sprintf("%s/%s/%s", crd.Spec.Group, crd.Spec.Version, crd.Spec.Names.Kind)] = struct{}{}
	}
	for _, indexer := range indexers {
		for key := range crdSpec {
			csvs, err := indexer.ByIndex(ProvidedAPIsIndexFuncKey, key)
			if err != nil {
				return nil, err
			}
			for _, item := range csvs {
				csv, ok := item.(*v1alpha1.ClusterServiceVersion)
				if !ok {
					continue
				}
				// Add to set
				csvSet[csv.GetName()] = struct{}{}
			}
		}
	}
	return csvSet, nil
}
