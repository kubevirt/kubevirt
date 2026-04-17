/*
Copyright 2024 Flant JSC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package rewriter

import (
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// RewriteAffinity renames or restores labels in labelSelector of affinity structure.
// See https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#node-affinity
func RewriteAffinity(rules *RewriteRules, obj []byte, path string, action Action) ([]byte, error) {
	return TransformObject(obj, path, func(affinity []byte) ([]byte, error) {
		rwrAffinity, err := TransformObject(affinity, "nodeAffinity", func(item []byte) ([]byte, error) {
			return rewriteNodeAffinity(rules, item, action)
		})
		if err != nil {
			return nil, err
		}

		rwrAffinity, err = TransformObject(rwrAffinity, "podAffinity", func(item []byte) ([]byte, error) {
			return rewritePodAffinity(rules, item, action)
		})
		if err != nil {
			return nil, err
		}

		return TransformObject(rwrAffinity, "podAntiAffinity", func(item []byte) ([]byte, error) {
			return rewritePodAffinity(rules, item, action)
		})

	})
}

// rewriteNodeAffinity rewrites labels in nodeAffinity structure.
// nodeAffinity:
//
//	requiredDuringSchedulingIgnoredDuringExecution:
//	  nodeSelectorTerms []NodeSelector -> rewrite each item: key in each matchExpressions and matchFields
//	preferredDuringSchedulingIgnoredDuringExecution: -> array of PreferredSchedulingTerm:
//	  preference NodeSelector ->  rewrite key in each matchExpressions and matchFields
//	  weight:
func rewriteNodeAffinity(rules *RewriteRules, obj []byte, action Action) ([]byte, error) {
	// Rewrite an array of nodeSelectorTerms in requiredDuringSchedulingIgnoredDuringExecution field.
	var err error
	obj, err = TransformObject(obj, "requiredDuringSchedulingIgnoredDuringExecution", func(affinityTerm []byte) ([]byte, error) {
		return RewriteArray(affinityTerm, "nodeSelectorTerms", func(item []byte) ([]byte, error) {
			return rewriteNodeSelectorTerm(rules, item, action)
		})
	})
	if err != nil {
		return nil, err
	}

	// Rewrite an array of weightedNodeSelectorTerms in preferredDuringSchedulingIgnoredDuringExecution field.
	return RewriteArray(obj, "preferredDuringSchedulingIgnoredDuringExecution", func(item []byte) ([]byte, error) {
		return TransformObject(item, "preference", func(preference []byte) ([]byte, error) {
			return rewriteNodeSelectorTerm(rules, preference, action)
		})
	})
}

// rewriteNodeSelectorTerm renames or restores selector requirements arrays in matchLabels or matchExpressions of NodeSelectorTerm.
// See [v1.NodeSelectorTerm](https://pkg.go.dev/k8s.io/api/core/v1#NodeSelectorTerm)
func rewriteNodeSelectorTerm(rules *RewriteRules, obj []byte, action Action) ([]byte, error) {
	obj, err := RewriteArray(obj, "matchLabels", func(item []byte) ([]byte, error) {
		return rewriteSelectorRequirement(rules, item, action)
	})
	if err != nil {
		return nil, err
	}
	return RewriteArray(obj, "matchExpressions", func(labelSelectorObj []byte) ([]byte, error) {
		return rewriteSelectorRequirement(rules, labelSelectorObj, action)
	})
}

// rewriteSelectorRequirement rewrites key and values in the selector requirement.
// Selector requirement example:
// {"key":"app.kubernetes.io/managed-by", "operator": "In", "values": ["Helm"]}
func rewriteSelectorRequirement(rules *RewriteRules, obj []byte, action Action) ([]byte, error) {
	key := gjson.GetBytes(obj, "key").String()
	valuesArr := gjson.GetBytes(obj, "values").Array()
	values := make([]string, len(valuesArr))
	for i, value := range valuesArr {
		values[i] = value.String()
	}
	rwrKey, rwrValues := rules.LabelsRewriter().RewriteNameValues(key, values, action)

	obj, err := sjson.SetBytes(obj, "key", rwrKey)
	if err != nil {
		return nil, err
	}

	return sjson.SetBytes(obj, "values", rwrValues)
}

// rewritePodAffinity rewrites PodAffinity and PodAntiAffinity structures.
// PodAffinity and PodAntiAffinity structures are the same:
//
//		requiredDuringSchedulingIgnoredDuringExecution -> array of PodAffinityTerm structures:
//		  labelSelector:
//		    matchLabels -> rewrite map
//		    matchExpressions -> rewrite key in each item
//		  topologyKey -> rewrite as label name
//		  namespaceSelector -> rewrite as labelSelector
//	   matchLabelKeys -> rewrite array of label keys
//	   mismatchLabelKeys -> rewrite array of label keys
//		preferredDuringSchedulingIgnoredDuringExecution -> array of WeightedPodAffinityTerm:
//		  weight
//		  podAffinityTerm PodAffinityTerm -> rewrite as described above
func rewritePodAffinity(rules *RewriteRules, obj []byte, action Action) ([]byte, error) {
	// Rewrite an array of PodAffinityTerms in requiredDuringSchedulingIgnoredDuringExecution field.
	obj, err := RewriteArray(obj, "requiredDuringSchedulingIgnoredDuringExecution", func(affinityTerm []byte) ([]byte, error) {
		return rewritePodAffinityTerm(rules, affinityTerm, action)
	})
	if err != nil {
		return nil, err
	}

	// Rewrite an array of WeightedPodAffinityTerms in requiredDuringSchedulingIgnoredDuringExecution field.
	return RewriteArray(obj, "preferredDuringSchedulingIgnoredDuringExecution", func(affinityTerm []byte) ([]byte, error) {
		return TransformObject(affinityTerm, "podAffinityTerm", func(podAffinityTerm []byte) ([]byte, error) {
			return rewritePodAffinityTerm(rules, podAffinityTerm, action)
		})
	})
}

func rewritePodAffinityTerm(rules *RewriteRules, obj []byte, action Action) ([]byte, error) {
	obj, err := TransformObject(obj, "labelSelector", func(labelSelector []byte) ([]byte, error) {
		return rewriteLabelSelector(rules, labelSelector, action)
	})
	if err != nil {
		return nil, err
	}

	obj, err = TransformString(obj, "topologyKey", func(topologyKey string) string {
		return rules.LabelsRewriter().Rewrite(topologyKey, action)
	})
	if err != nil {
		return nil, err
	}

	obj, err = TransformObject(obj, "namespaceSelector", func(selector []byte) ([]byte, error) {
		return rewriteLabelSelector(rules, selector, action)
	})
	if err != nil {
		return nil, err
	}

	obj, err = TransformArrayOfStrings(obj, "matchLabelKeys", func(labelKey string) string {
		return rules.LabelsRewriter().Rewrite(labelKey, action)
	})
	if err != nil {
		return nil, err
	}

	return TransformArrayOfStrings(obj, "mismatchLabelKeys", func(labelKey string) string {
		return rules.LabelsRewriter().Rewrite(labelKey, action)
	})
}

// rewriteLabelSelector rewrites matchLabels and matchExpressions. It is similar to rewriteNodeSelectorTerm
// but matchLabels is a map here, not an array of requirements.
func rewriteLabelSelector(rules *RewriteRules, obj []byte, action Action) ([]byte, error) {
	obj, err := RewriteLabelsMap(rules, obj, "matchLabels", action)
	if err != nil {
		return nil, err
	}

	return RewriteArray(obj, "matchExpressions", func(item []byte) ([]byte, error) {
		return rewriteSelectorRequirement(rules, item, action)
	})
}
