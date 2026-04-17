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
	"fmt"
	"strings"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const (
	CRDKind     = "CustomResourceDefinition"
	CRDListKind = "CustomResourceDefinitionList"
)

func RewriteCRDOrList(rules *RewriteRules, obj []byte, action Action) ([]byte, error) {
	// CREATE, UPDATE, or PATCH requests.
	if action == Rename {
		return RewriteResourceOrList(obj, CRDListKind, func(singleObj []byte) ([]byte, error) {
			return RenameCRD(rules, singleObj)
		})
	}

	// Responses of GET, LIST, DELETE requests. Also, rewrite in watch events.
	return RewriteResourceOrList(obj, CRDListKind, func(singleObj []byte) ([]byte, error) {
		return RestoreCRD(rules, singleObj)
	})
}

// RestoreCRD restores fields in CRD to original.
//
// Example:
// .metadata.name     prefixedvirtualmachines.x.virtualization.deckhouse.io -> virtualmachines.kubevirt.io
// .spec.group        x.virtualization.deckhouse.io -> kubevirt.io
// .spec.names
//
//	categories      kubevirt -> all
//	kind            PrefixedVirtualMachines -> VirtualMachine
//	listKind        PrefixedVirtualMachineList -> VirtualMachineList
//	plural          prefixedvirtualmachines -> virtualmachines
//	singular        prefixedvirtualmachine -> virtualmachine
//	shortNames      [xvm xvms] -> [vm vms]
func RestoreCRD(rules *RewriteRules, obj []byte) ([]byte, error) {
	crdName := gjson.GetBytes(obj, "metadata.name").String()
	resource, group, found := strings.Cut(crdName, ".")
	if !found {
		return nil, fmt.Errorf("malformed CRD name: should be resourcetype.group, got %s", crdName)
	}

	// Skip CRD with original group to avoid duplicates in restored List.
	if rules.HasGroup(group) {
		return nil, SkipItem
	}

	// Do not restore CRDs from unknown groups.
	if !rules.IsRenamedGroup(group) {
		return nil, nil
	}

	origResource := rules.RestoreResource(resource)

	groupRule, resourceRule := rules.GroupResourceRules(origResource)
	if resourceRule == nil {
		return nil, nil
	}

	newName := resourceRule.Plural + "." + groupRule.Group
	obj, err := sjson.SetBytes(obj, "metadata.name", newName)
	if err != nil {
		return nil, err
	}

	obj, err = sjson.SetBytes(obj, "spec.group", groupRule.Group)
	if err != nil {
		return nil, err
	}

	names := []byte(gjson.GetBytes(obj, "spec.names").Raw)

	names, err = sjson.SetBytes(names, "categories", rules.RestoreCategories(resourceRule))
	if err != nil {
		return nil, err
	}
	names, err = sjson.SetBytes(names, "kind", rules.RestoreKind(resourceRule.Kind))
	if err != nil {
		return nil, err
	}
	names, err = sjson.SetBytes(names, "listKind", rules.RestoreKind(resourceRule.ListKind))
	if err != nil {
		return nil, err
	}
	names, err = sjson.SetBytes(names, "plural", rules.RestoreResource(resourceRule.Plural))
	if err != nil {
		return nil, err
	}
	names, err = sjson.SetBytes(names, "singular", rules.RestoreResource(resourceRule.Singular))
	if err != nil {
		return nil, err
	}
	names, err = sjson.SetBytes(names, "shortNames", rules.RestoreShortNames(resourceRule.ShortNames))
	if err != nil {
		return nil, err
	}

	obj, err = sjson.SetRawBytes(obj, "spec.names", names)
	if err != nil {
		return nil, err
	}

	return obj, nil
}

// RenameCRD renames fields in CRD.
//
// Example:
// .metadata.name     virtualmachines.kubevirt.io -> prefixedvirtualmachines.x.virtualization.deckhouse.io
// .spec.group        kubevirt.io -> x.virtualization.deckhouse.io
// .spec.names
//
//	categories      all -> kubevirt
//	kind            VirtualMachine -> PrefixedVirtualMachines
//	listKind        VirtualMachineList -> PrefixedVirtualMachineList
//	plural          virtualmachines -> prefixedvirtualmachines
//	singular        virtualmachine -> prefixedvirtualmachine
//	shortNames      [vm vms] -> [xvm xvms]
func RenameCRD(rules *RewriteRules, obj []byte) ([]byte, error) {
	crdName := gjson.GetBytes(obj, "metadata.name").String()
	resource, group, found := strings.Cut(crdName, ".")
	if !found {
		return nil, fmt.Errorf("malformed CRD name: should be resourcetype.group, got %s", crdName)
	}

	_, resourceRule := rules.ResourceRules(group, resource)
	if resourceRule == nil {
		return nil, nil
	}

	newName := rules.RenameResource(resource) + "." + rules.RenameApiVersion(group)
	obj, err := sjson.SetBytes(obj, "metadata.name", newName)
	if err != nil {
		return nil, err
	}

	spec := gjson.GetBytes(obj, "spec")
	newSpec, err := renameCRDSpec(rules, resourceRule, []byte(spec.Raw))
	if err != nil {
		return nil, err
	}
	return sjson.SetRawBytes(obj, "spec", newSpec)
}

func renameCRDSpec(rules *RewriteRules, resourceRule *ResourceRule, spec []byte) ([]byte, error) {
	var err error

	spec, err = TransformString(spec, "group", func(crdSpecGroup string) string {
		return rules.RenameApiVersion(crdSpecGroup)
	})
	if err != nil {
		return nil, err
	}

	// Rename fields in the 'names' object.
	names := []byte(gjson.GetBytes(spec, "names").Raw)

	if gjson.GetBytes(names, "categories").Exists() {
		names, err = sjson.SetBytes(names, "categories", rules.RenameCategories(resourceRule.Categories))
		if err != nil {
			return nil, err
		}
	}
	if gjson.GetBytes(names, "kind").Exists() {
		names, err = sjson.SetBytes(names, "kind", rules.RenameKind(resourceRule.Kind))
		if err != nil {
			return nil, err
		}
	}
	if gjson.GetBytes(names, "listKind").Exists() {
		names, err = sjson.SetBytes(names, "listKind", rules.RenameKind(resourceRule.ListKind))
		if err != nil {
			return nil, err
		}
	}
	if gjson.GetBytes(names, "plural").Exists() {
		names, err = sjson.SetBytes(names, "plural", rules.RenameResource(resourceRule.Plural))
		if err != nil {
			return nil, err
		}
	}
	if gjson.GetBytes(names, "singular").Exists() {
		names, err = sjson.SetBytes(names, "singular", rules.RenameResource(resourceRule.Singular))
		if err != nil {
			return nil, err
		}
	}
	if gjson.GetBytes(names, "shortNames").Exists() {
		names, err = sjson.SetBytes(names, "shortNames", rules.RenameShortNames(resourceRule.ShortNames))
		if err != nil {
			return nil, err
		}
	}

	spec, err = sjson.SetRawBytes(spec, "names", names)
	if err != nil {
		return nil, err
	}

	return spec, nil
}

func RenameCRDPatch(rules *RewriteRules, resourceRule *ResourceRule, obj []byte) ([]byte, error) {
	var err error

	obj, err = RenameMetadataPatch(rules, obj)
	if err != nil {
		return nil, fmt.Errorf("rename metadata patches for CRD: %w", err)
	}

	isRenamed := false
	newPatches, err := RewriteArray(obj, Root, func(singlePatch []byte) ([]byte, error) {
		op := gjson.GetBytes(singlePatch, "op").String()
		path := gjson.GetBytes(singlePatch, "path").String()

		if (op == "replace" || op == "add") && path == "/spec" {
			isRenamed = true
			value := []byte(gjson.GetBytes(singlePatch, "value").Raw)
			newValue, err := renameCRDSpec(rules, resourceRule, value)
			if err != nil {
				return nil, err
			}
			return sjson.SetRawBytes(singlePatch, "value", newValue)
		}

		return nil, nil
	})

	if !isRenamed {
		return obj, nil
	}

	return newPatches, nil
}
