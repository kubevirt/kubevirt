package indexer

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

const (
	// MetaLabelIndexFuncKey is the recommended key to use for registering the index func with an indexer.
	MetaLabelIndexFuncKey string = "metalabelindexfunc"
)

// MetaLabelIndexFunc returns indicies from the labels of the given object.
func MetaLabelIndexFunc(obj interface{}) ([]string, error) {
	indicies := []string{}
	m, err := meta.Accessor(obj)
	if err != nil {
		return indicies, fmt.Errorf("object has no meta: %v", err)
	}

	for k, v := range m.GetLabels() {
		indicies = append(indicies, fmt.Sprintf("%s=%s", k, v))
	}

	return indicies, nil
}

// LabelIndexKeys returns the union of indexed cache keys in the given indexers matching the same labels as the given selector
func LabelIndexKeys(indexers map[string]cache.Indexer, labelSets ...labels.Set) ([]string, error) {
	keySet := map[string]struct{}{}
	keys := []string{}
	for _, indexer := range indexers {
		for _, labelSet := range labelSets {
			for key, value := range labelSet {
				apiLabelKey := fmt.Sprintf("%s=%s", key, value)
				cacheKeys, err := indexer.IndexKeys(MetaLabelIndexFuncKey, apiLabelKey)
				if err != nil {
					return nil, err
				}

				for _, cacheKey := range cacheKeys {
					// Detect duplication
					if _, ok := keySet[cacheKey]; ok {
						continue
					}

					// Add to set
					keySet[cacheKey] = struct{}{}
					keys = append(keys, cacheKey)
				}

			}
		}
	}

	return keys, nil
}
