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

// networkSelectionElement models the fields of a Multus network-selection
// element that KubeVirt owns; every other field is preserved in extras and
// re-emitted on marshal with value fidelity (number precision included),
// though formatting is normalized.
//
// The vendored networkv1.NetworkSelectionElement type is deliberately not used
// here: round-tripping through it would drop fields it does not model, which
// is exactly the data loss this merge exists to prevent.
//
// Modeled fields set to JSON null are dropped on marshal; null is equivalent
// to absent for Multus consumers.
type networkSelectionElement struct {
	Namespace        string   `json:"namespace,omitempty"`
	Name             string   `json:"name"`
	InterfaceRequest string   `json:"interface,omitempty"`
	MacRequest       string   `json:"mac,omitempty"`
	CNIArgs          *cniArgs `json:"cni-args,omitempty"`

	extras map[string]json.RawMessage
}

// cniArgs models the only cni-args key KubeVirt writes: logicNetworkName,
// which it uses to disambiguate binding-plugin elements sharing a NAD. It is
// modeled - rather than left to extras - because it is part of the element key
// for such elements. Other cni-args keys (e.g. deviceID) are external and
// preserved in extras. Merge policy still preserves the entire current
// cni-args object (see mergeNetworkSelectionElement), since external
// controllers may enrich the object KubeVirt last wrote.
type cniArgs struct {
	LogicNetworkName string `json:"logicNetworkName,omitempty"`

	extras map[string]json.RawMessage
}

// The JSON field names modeled by the structs above; every other field is
// external and preserved. Pinned to the struct tags by
// TestOwnedJSONFieldsMatchStructTags, so the lists cannot drift from the
// structs.
var (
	elementOwnedJSONFields = []string{"namespace", "name", "interface", "mac", "cni-args"}
	cniArgsOwnedJSONFields = []string{"logicNetworkName"}
)

func (e *networkSelectionElement) UnmarshalJSON(data []byte) error {
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

// decodeWithExtras decodes data into owned and returns every field not named
// in ownedFields as raw JSON, so it can be re-emitted on marshal.
// Owned fields are matched case-insensitively, mirroring encoding/json's field
// binding: a differently-cased owned field must not survive in extras and be
// emitted next to its canonical spelling.
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

// encodeWithExtras re-emits the extras fields alongside the marshaled owned
// fields. Extra values keep their JSON value, number precision included.
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

// key identifies the element in the annotation. An empty namespace defaults to
// the pod namespace, matching Multus semantics, so that an externally dropped
// namespace does not fork one attachment into two elements.
// Binding-plugin elements carry no interface request; they are identified by
// their KubeVirt-written cni-args.logicNetworkName alone, because their NAD is
// resolved from cluster configuration and may be re-pointed while the logical
// network name is stable.
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
// annotation into the current one.
//
// Fields that are not modeled by KubeVirt are preserved. Only fields owned by
// KubeVirt are replaced. Elements listed in the absent annotations are removed.
//
// A current annotation that an external writer malformed cannot be preserved,
// so the desired annotation is returned as-is instead of blocking the update.
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

// mergeNetworkSelectionElement overlays the KubeVirt-owned fields - exactly
// the fields modeled by the struct - from the desired element onto the current
// one; owned fields absent from the desired element are removed. A modeled
// field is desired-wins by default: a field that must instead be preserved
// from the current element needs an explicit assignment below (see CNIArgs,
// which external controllers may have enriched).
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
