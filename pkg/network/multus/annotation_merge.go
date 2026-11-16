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
	"errors"
	"fmt"
	"maps"
	"reflect"
	"strings"

	"k8s.io/apimachinery/pkg/util/sets"
)

type networkSelectionElement struct {
	Namespace        string   `json:"namespace,omitempty"`
	Name             string   `json:"name"`
	InterfaceRequest string   `json:"interface,omitempty"`
	MacRequest       string   `json:"mac,omitempty"`
	CNIArgs          *cniArgs `json:"cni-args,omitempty"`

	extras map[string]json.RawMessage
}

type cniArgs struct {
	LogicNetworkName string `json:"logicNetworkName,omitempty"`

	extras map[string]json.RawMessage
}

var (
	elementOwnedJSONFields = []string{"namespace", "name", "interface", "mac", "cni-args"}
	cniArgsOwnedJSONFields = []string{"logicNetworkName"}
)

func (e *networkSelectionElement) UnmarshalJSON(data []byte) error {
	// Use a type without methods to avoid calling UnmarshalJSON recursively.
	type alias networkSelectionElement
	extras, err := decodeWithExtras(data, (*alias)(e), elementOwnedJSONFields)
	if err != nil {
		return fmt.Errorf("failed to decode network selection element: %w", err)
	}
	e.extras = extras

	return nil
}

func (e networkSelectionElement) MarshalJSON() ([]byte, error) {
	type alias networkSelectionElement

	return encodeWithExtras(alias(e), e.extras)
}

func (a *cniArgs) UnmarshalJSON(data []byte) error {
	type alias cniArgs
	extras, err := decodeWithExtras(data, (*alias)(a), cniArgsOwnedJSONFields)
	if err != nil {
		return fmt.Errorf("failed to decode cni-args: %w", err)
	}
	a.extras = extras

	return nil
}

func (a cniArgs) MarshalJSON() ([]byte, error) {
	type alias cniArgs

	return encodeWithExtras(alias(a), a.extras)
}

func decodeWithExtras(data []byte, owned any, ownedFields []string) (map[string]json.RawMessage, error) {
	if err := json.Unmarshal(data, owned); err != nil {
		return nil, err
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return nil, err
	}
	for field := range fields {
		for _, ownedField := range ownedFields {
			if strings.EqualFold(field, ownedField) {
				delete(fields, field)
				break
			}
		}
	}

	return fields, nil
}

func encodeWithExtras(owned any, extras map[string]json.RawMessage) ([]byte, error) {
	base, err := json.Marshal(owned)
	if err != nil {
		return nil, err
	}
	if len(extras) == 0 {
		return base, nil
	}

	fields := map[string]json.RawMessage{}
	if err := json.Unmarshal(base, &fields); err != nil {
		return nil, err
	}
	maps.Copy(fields, extras)

	return json.Marshal(fields)
}

type networkSelectionElementKey struct {
	namespace        string
	name             string
	iface            string
	logicNetworkName string
}

func (e networkSelectionElement) key(podNamespace string) networkSelectionElementKey {
	logicNetworkName := ""
	if e.InterfaceRequest == "" && e.CNIArgs != nil {
		logicNetworkName = e.CNIArgs.LogicNetworkName
	}
	if logicNetworkName != "" {
		return networkSelectionElementKey{logicNetworkName: logicNetworkName}
	}

	namespace := e.Namespace
	if namespace == "" {
		namespace = podNamespace
	}

	return networkSelectionElementKey{
		namespace: namespace,
		name:      e.Name,
		iface:     e.InterfaceRequest,
	}
}

// MergeNetworkSelectionAnnotations merges the desired k8s.v1.cni.cncf.io/networks
// annotation into the current one, preserving external fields and removing absent
// elements. If the current annotation cannot be parsed, it returns desired as-is.
func MergeNetworkSelectionAnnotations(podNamespace, current, desired string, absentAnnotations ...string) (string, error) {
	desiredElements, err := parseNetworkSelectionAnnotation(desired)
	if err != nil {
		return "", fmt.Errorf("failed to parse desired network selection annotation: %w", err)
	}

	currentElements, err := parseNetworkSelectionAnnotation(current)
	if err != nil {
		return desired, nil
	}

	absentElements, err := parseNetworkSelectionAnnotations(absentAnnotations)
	if err != nil {
		return "", fmt.Errorf("failed to parse absent network selection annotations: %w", err)
	}

	mergedElements, err := mergeNetworkSelectionElements(podNamespace, currentElements, desiredElements, absentElements)
	if err != nil {
		return "", err
	}

	return marshalNetworkSelectionAnnotation(current, currentElements, mergedElements)
}

func mergeNetworkSelectionElements(
	podNamespace string,
	currentElements, desiredElements, absentElements []networkSelectionElement,
) ([]networkSelectionElement, error) {
	desiredElementsByKey, err := indexNetworkSelectionElements(podNamespace, desiredElements)
	if err != nil {
		return nil, err
	}
	absentElementsByKey, err := indexNetworkSelectionElements(podNamespace, absentElements)
	if err != nil {
		return nil, err
	}

	mergedElements, mergedDesiredKeys := mergeCurrentNetworkSelectionElements(
		podNamespace,
		currentElements,
		desiredElementsByKey,
		absentElementsByKey,
	)

	return appendNewNetworkSelectionElements(
		podNamespace,
		mergedElements,
		desiredElements,
		mergedDesiredKeys,
		absentElementsByKey,
	), nil
}

func mergeCurrentNetworkSelectionElements(
	podNamespace string,
	currentElements []networkSelectionElement,
	desiredElementsByKey map[networkSelectionElementKey]networkSelectionElement,
	absentElementsByKey map[networkSelectionElementKey]networkSelectionElement,
) ([]networkSelectionElement, sets.Set[networkSelectionElementKey]) {
	mergedElements := make([]networkSelectionElement, 0, len(currentElements))
	mergedDesiredKeys := sets.New[networkSelectionElementKey]()

	for _, currentElement := range currentElements {
		key := currentElement.key(podNamespace)

		if _, isAbsent := absentElementsByKey[key]; isAbsent {
			continue
		}

		if desiredElement, shouldMerge := desiredElementsByKey[key]; shouldMerge {
			mergedElements = append(mergedElements, mergeNetworkSelectionElement(currentElement, desiredElement))
			mergedDesiredKeys.Insert(key)
			continue
		}

		mergedElements = append(mergedElements, currentElement)
	}

	return mergedElements, mergedDesiredKeys
}

func appendNewNetworkSelectionElements(
	podNamespace string,
	mergedElements []networkSelectionElement,
	desiredElements []networkSelectionElement,
	mergedDesiredKeys sets.Set[networkSelectionElementKey],
	absentElementsByKey map[networkSelectionElementKey]networkSelectionElement,
) []networkSelectionElement {
	for _, desiredElement := range desiredElements {
		key := desiredElement.key(podNamespace)

		if mergedDesiredKeys.Has(key) {
			continue
		}
		if _, isAbsent := absentElementsByKey[key]; isAbsent {
			continue
		}

		mergedElements = append(mergedElements, desiredElement)
	}

	return mergedElements
}

func mergeNetworkSelectionElement(current, desired networkSelectionElement) networkSelectionElement {
	merged := desired
	merged.extras = current.extras
	merged.CNIArgs = current.CNIArgs

	return merged
}

func marshalNetworkSelectionAnnotation(
	current string,
	currentElements, mergedElements []networkSelectionElement,
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

func parseNetworkSelectionAnnotation(annotation string) ([]networkSelectionElement, error) {
	if annotation == "" {
		return []networkSelectionElement{}, nil
	}

	var elements []networkSelectionElement
	if err := json.Unmarshal([]byte(annotation), &elements); err != nil {
		return nil, err
	}
	if elements == nil {
		elements = []networkSelectionElement{}
	}
	for _, element := range elements {
		if element.Name == "" {
			return nil, errors.New(`network selection element is missing the "name" field`)
		}
	}

	return elements, nil
}

func parseNetworkSelectionAnnotations(annotations []string) ([]networkSelectionElement, error) {
	elements := []networkSelectionElement{}
	for _, annotation := range annotations {
		parsedElements, err := parseNetworkSelectionAnnotation(annotation)
		if err != nil {
			return nil, err
		}
		elements = append(elements, parsedElements...)
	}

	return elements, nil
}

func indexNetworkSelectionElements(
	podNamespace string,
	elements []networkSelectionElement,
) (map[networkSelectionElementKey]networkSelectionElement, error) {
	elementsByKey := make(map[networkSelectionElementKey]networkSelectionElement, len(elements))
	for _, element := range elements {
		key := element.key(podNamespace)
		if _, exists := elementsByKey[key]; exists {
			return nil, fmt.Errorf("duplicate network selection element: %+v", key)
		}
		elementsByKey[key] = element
	}

	return elementsByKey, nil
}
