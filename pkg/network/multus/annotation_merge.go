/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright The KubeVirt Authors.
 *
 */

package multus

import (
	"encoding/json"
	"fmt"
	"maps"
	"reflect"

	"k8s.io/apimachinery/pkg/util/sets"
)

type networkSelectionElementKey struct {
	namespace        string
	name             string
	iface            string
	logicNetworkName string
}

var kubevirtOwnedNetworkSelectionFields = sets.New[string]("namespace", "name", "interface", "mac")

// MergeNetworkSelectionAnnotations merges the desired k8s.v1.cni.cncf.io/networks
// annotation into the current one.
//
// The current annotation is parsed as raw JSON so fields that are not modeled by
// KubeVirt are preserved. Only fields owned by KubeVirt are replaced. Elements
// listed in the absent annotations are removed.
func MergeNetworkSelectionAnnotations(current, desired string, absentAnnotations ...string) (string, error) {
	currentElements, err := parseNetworkSelectionAnnotation(current)
	if err != nil {
		return "", fmt.Errorf("failed to parse current network selection annotation: %w", err)
	}

	desiredElements, err := parseNetworkSelectionAnnotation(desired)
	if err != nil {
		return "", fmt.Errorf("failed to parse desired network selection annotation: %w", err)
	}

	absentElements, err := parseNetworkSelectionAnnotations(absentAnnotations)
	if err != nil {
		return "", fmt.Errorf("failed to parse absent network selection annotations: %w", err)
	}

	mergedElements, err := mergeNetworkSelectionElements(currentElements, desiredElements, absentElements)
	if err != nil {
		return "", err
	}

	return marshalNetworkSelectionAnnotation(current, currentElements, mergedElements)
}

func mergeNetworkSelectionElements(
	currentElements, desiredElements, absentElements []map[string]any,
) ([]map[string]any, error) {
	desiredElementsByKey, err := indexNetworkSelectionElements(desiredElements)
	if err != nil {
		return nil, err
	}
	absentElementsByKey, err := indexNetworkSelectionElements(absentElements)
	if err != nil {
		return nil, err
	}

	mergedElements, mergedDesiredKeys, err := mergeCurrentNetworkSelectionElements(
		currentElements,
		desiredElementsByKey,
		absentElementsByKey,
	)
	if err != nil {
		return nil, err
	}

	return appendNewNetworkSelectionElements(
		mergedElements,
		desiredElements,
		mergedDesiredKeys,
		absentElementsByKey,
	)
}

func mergeCurrentNetworkSelectionElements(
	currentElements []map[string]any,
	desiredElementsByKey map[networkSelectionElementKey]map[string]any,
	absentElementsByKey map[networkSelectionElementKey]map[string]any,
) ([]map[string]any, sets.Set[networkSelectionElementKey], error) {
	mergedElements := make([]map[string]any, 0, len(currentElements))
	mergedDesiredKeys := sets.New[networkSelectionElementKey]()

	for _, currentElement := range currentElements {
		key, err := networkSelectionElementKeyFromMap(currentElement)
		if err != nil {
			return nil, nil, err
		}

		if _, isAbsent := absentElementsByKey[key]; isAbsent {
			continue
		}

		if desiredElement, shouldMerge := desiredElementsByKey[key]; shouldMerge {
			mergedElements = append(mergedElements, mergeNetworkSelectionElement(currentElement, desiredElement))
			mergedDesiredKeys.Insert(key)
			continue
		}

		mergedElements = append(mergedElements, maps.Clone(currentElement))
	}

	return mergedElements, mergedDesiredKeys, nil
}

func appendNewNetworkSelectionElements(
	mergedElements []map[string]any,
	desiredElements []map[string]any,
	mergedDesiredKeys sets.Set[networkSelectionElementKey],
	absentElementsByKey map[networkSelectionElementKey]map[string]any,
) ([]map[string]any, error) {
	for _, desiredElement := range desiredElements {
		key, err := networkSelectionElementKeyFromMap(desiredElement)
		if err != nil {
			return nil, err
		}

		if mergedDesiredKeys.Has(key) {
			continue
		}
		if _, isAbsent := absentElementsByKey[key]; isAbsent {
			continue
		}

		mergedElements = append(mergedElements, maps.Clone(desiredElement))
	}

	return mergedElements, nil
}

func marshalNetworkSelectionAnnotation(
	current string,
	currentElements, mergedElements []map[string]any,
) (string, error) {
	if len(mergedElements) == 0 {
		return "", nil
	}

	if reflect.DeepEqual(currentElements, mergedElements) {
		return current, nil
	}

	mergedAnnotation, err := json.Marshal(mergedElements)
	if err != nil {
		return "", fmt.Errorf("failed to marshal network selection annotation: %w", err)
	}

	return string(mergedAnnotation), nil
}

func parseNetworkSelectionAnnotation(annotation string) ([]map[string]any, error) {
	if annotation == "" {
		return []map[string]any{}, nil
	}

	var elements []map[string]any
	if err := json.Unmarshal([]byte(annotation), &elements); err != nil {
		return nil, err
	}
	if elements == nil {
		elements = []map[string]any{}
	}

	return elements, nil
}

func parseNetworkSelectionAnnotations(annotations []string) ([]map[string]any, error) {
	elements := []map[string]any{}
	for _, annotation := range annotations {
		parsedElements, err := parseNetworkSelectionAnnotation(annotation)
		if err != nil {
			return nil, err
		}
		elements = append(elements, parsedElements...)
	}

	return elements, nil
}

func indexNetworkSelectionElements(elements []map[string]any) (map[networkSelectionElementKey]map[string]any, error) {
	elementsByKey := make(map[networkSelectionElementKey]map[string]any, len(elements))
	for _, element := range elements {
		key, err := networkSelectionElementKeyFromMap(element)
		if err != nil {
			return nil, err
		}
		if _, exists := elementsByKey[key]; exists {
			return nil, fmt.Errorf("duplicate network selection element: %+v", key)
		}
		elementsByKey[key] = element
	}

	return elementsByKey, nil
}

func networkSelectionElementKeyFromMap(element map[string]any) (networkSelectionElementKey, error) {
	name, err := requiredString(element, "name")
	if err != nil {
		return networkSelectionElementKey{}, err
	}
	namespace, err := optionalString(element, "namespace")
	if err != nil {
		return networkSelectionElementKey{}, err
	}
	iface, err := optionalString(element, "interface")
	if err != nil {
		return networkSelectionElementKey{}, err
	}
	logicNetworkName := ""
	if iface == "" {
		logicNetworkName, err = logicNetworkNameFromMap(element)
		if err != nil {
			return networkSelectionElementKey{}, err
		}
	}

	return networkSelectionElementKey{
		namespace:        namespace,
		name:             name,
		iface:            iface,
		logicNetworkName: logicNetworkName,
	}, nil
}

func mergeNetworkSelectionElement(current, desired map[string]any) map[string]any {
	merged := maps.Clone(current)
	for field := range kubevirtOwnedNetworkSelectionFields {
		if value, exists := desired[field]; exists {
			merged[field] = value
		} else {
			delete(merged, field)
		}
	}

	return merged
}

func logicNetworkNameFromMap(element map[string]any) (string, error) {
	value, exists := element["cni-args"]
	if !exists || value == nil {
		return "", nil
	}

	cniArgs, ok := value.(map[string]any)
	if !ok {
		return "", fmt.Errorf("network selection element has a non-object %q field: %v", "cni-args", value)
	}

	return optionalString(cniArgs, "logicNetworkName")
}

func requiredString(element map[string]any, field string) (string, error) {
	value, exists := element[field]
	if !exists || value == nil {
		return "", fmt.Errorf("network selection element is missing the %q field", field)
	}

	valueString, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("network selection element has a non-string %q field: %v", field, value)
	}

	return valueString, nil
}

func optionalString(element map[string]any, field string) (string, error) {
	value, exists := element[field]
	if !exists || value == nil {
		return "", nil
	}

	valueString, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("network selection element has a non-string %q field: %v", field, value)
	}

	return valueString, nil
}
