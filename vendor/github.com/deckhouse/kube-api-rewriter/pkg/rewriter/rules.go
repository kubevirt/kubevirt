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
	"strings"

	"github.com/tidwall/gjson"

	"github.com/deckhouse/kube-api-rewriter/pkg/rewriter/indexer"
)

type RewriteRules struct {
	KindPrefix         string                  `json:"kindPrefix"`
	ResourceTypePrefix string                  `json:"resourceTypePrefix"`
	ShortNamePrefix    string                  `json:"shortNamePrefix"`
	Categories         []string                `json:"categories"`
	Rules              map[string]APIGroupRule `json:"rules"`
	Webhooks           map[string]WebhookRule  `json:"webhooks"`
	Labels             MetadataReplace         `json:"labels"`
	Annotations        MetadataReplace         `json:"annotations"`
	Finalizers         MetadataReplace         `json:"finalizers"`
	Excludes           []ExcludeRule           `json:"excludes"`

	// TODO move these indexed rewriters into the RuleBasedRewriter.
	labelsRewriter      *PrefixedNameRewriter
	annotationsRewriter *PrefixedNameRewriter
	finalizersRewriter  *PrefixedNameRewriter

	apiGroupsIndex *indexer.MapIndexer
}

// Init should be called before using rules in the RuleBasedRewriter.
func (rr *RewriteRules) Init() {
	rr.labelsRewriter = NewPrefixedNameRewriter(rr.Labels)
	rr.annotationsRewriter = NewPrefixedNameRewriter(rr.Annotations)
	rr.finalizersRewriter = NewPrefixedNameRewriter(rr.Finalizers)

	// Add all original Kinds and KindList as implicit excludes.
	originalKinds := make([]string, 0)
	for _, apiGroupRule := range rr.Rules {
		for _, resourceRule := range apiGroupRule.ResourceRules {
			originalKinds = append(originalKinds, resourceRule.Kind, resourceRule.ListKind)
		}
	}
	if len(originalKinds) > 0 {
		rr.Excludes = append(rr.Excludes, ExcludeRule{Kinds: originalKinds})
	}

	// Index apiGroups originals and their renames.
	rr.apiGroupsIndex = indexer.NewMapIndexer()
	for _, apiGroupRule := range rr.Rules {
		rr.apiGroupsIndex.AddPair(apiGroupRule.GroupRule.Group, apiGroupRule.GroupRule.Renamed)
	}
}

type APIGroupRule struct {
	GroupRule     GroupRule               `json:"groupRule"`
	ResourceRules map[string]ResourceRule `json:"resourceRules"`
}

type GroupRule struct {
	Group            string   `json:"group"`
	Versions         []string `json:"versions"`
	PreferredVersion string   `json:"preferredVersion"`
	Renamed          string   `json:"renamed"`
}

type ResourceRule struct {
	Kind             string   `json:"kind"`
	ListKind         string   `json:"listKind"`
	Plural           string   `json:"plural"`
	Singular         string   `json:"singular"`
	ShortNames       []string `json:"shortNames"`
	Categories       []string `json:"categories"`
	Versions         []string `json:"versions"`
	PreferredVersion string   `json:"preferredVersion"`
}

type WebhookRule struct {
	Path     string `json:"path"`
	Group    string `json:"group"`
	Resource string `json:"resource"`
}

type MetadataReplace struct {
	Prefixes []MetadataReplaceRule
	Names    []MetadataReplaceRule
}

type MetadataReplaceRule struct {
	Original      string `json:"original"`
	Renamed       string `json:"renamed"`
	OriginalValue string `json:"originalValue"`
	RenamedValue  string `json:"renamedValue"`
}

type ExcludeRule struct {
	Kinds       []string          `json:"kinds"`
	MatchNames  []string          `json:"matchNames"`
	MatchLabels map[string]string `json:"matchLabels"`
}

// GetAPIGroupList returns an array of groups in format applicable to use in APIGroupList:
//
//	{
//	  name
//	  versions: [ { groupVersion, version } ... ]
//	  preferredVersion: { groupVersion, version }
//	}
func (rr *RewriteRules) GetAPIGroupList() []interface{} {
	groups := make([]interface{}, 0)

	for _, rGroup := range rr.Rules {
		group := map[string]interface{}{
			"name": rGroup.GroupRule.Group,
			"preferredVersion": map[string]interface{}{
				"groupVersion": rGroup.GroupRule.Group + "/" + rGroup.GroupRule.PreferredVersion,
				"version":      rGroup.GroupRule.PreferredVersion,
			},
		}
		versions := make([]interface{}, 0)
		for _, ver := range rGroup.GroupRule.Versions {
			versions = append(versions, map[string]interface{}{
				"groupVersion": rGroup.GroupRule.Group + "/" + ver,
				"version":      ver,
			})
		}
		group["versions"] = versions
		groups = append(groups, group)
	}

	return groups
}

func (rr *RewriteRules) ResourceByKind(kind string) (string, string, bool) {
	for groupName, group := range rr.Rules {
		for resName, res := range group.ResourceRules {
			if res.Kind == kind {
				return groupName, resName, false
			}
			if res.ListKind == kind {
				return groupName, resName, true
			}
		}
	}
	return "", "", false
}

func (rr *RewriteRules) WebhookRule(path string) *WebhookRule {
	if webhookRule, ok := rr.Webhooks[path]; ok {
		return &webhookRule
	}
	return nil
}

func (rr *RewriteRules) IsRenamedGroup(apiGroup string) bool {
	// Trim version and delimeter.
	apiGroup, _, _ = strings.Cut(apiGroup, "/")
	return rr.apiGroupsIndex.IsRenamed(apiGroup)
}

func (rr *RewriteRules) HasGroup(group string) bool {
	// Trim version and delimeter.
	group, _, _ = strings.Cut(group, "/")
	_, ok := rr.Rules[group]
	return ok
}

func (rr *RewriteRules) GroupRule(group string) *GroupRule {
	if groupRule, ok := rr.Rules[group]; ok {
		return &groupRule.GroupRule
	}
	return nil
}

// KindRules returns rule for group and resource by apiGroup and kind.
// apiGroup may be a group or a group with version.
func (rr *RewriteRules) KindRules(apiGroup, kind string) (*GroupRule, *ResourceRule) {
	group, _, _ := strings.Cut(apiGroup, "/")
	groupRule, ok := rr.Rules[group]
	if !ok {
		return nil, nil
	}

	for _, resRule := range groupRule.ResourceRules {
		if resRule.Kind == kind {
			return &groupRule.GroupRule, &resRule
		}
		if resRule.ListKind == kind {
			return &groupRule.GroupRule, &resRule
		}
	}
	return nil, nil
}

func (rr *RewriteRules) ResourceRules(apiGroup, resource string) (*GroupRule, *ResourceRule) {
	group, _, _ := strings.Cut(apiGroup, "/")
	groupRule, ok := rr.Rules[group]
	if !ok {
		return nil, nil
	}
	resource, _, _ = strings.Cut(resource, "/")
	resourceRule, ok := rr.Rules[group].ResourceRules[resource]
	if !ok {
		return nil, nil
	}
	return &groupRule.GroupRule, &resourceRule
}

func (rr *RewriteRules) GroupResourceRules(resourceType string) (*GroupRule, *ResourceRule) {
	// Trim subresource and delimiter.
	resourceType, _, _ = strings.Cut(resourceType, "/")

	for _, group := range rr.Rules {
		for _, res := range group.ResourceRules {
			if res.Plural == resourceType {
				return &group.GroupRule, &res
			}
		}
	}
	return nil, nil
}

func (rr *RewriteRules) GroupResourceRulesByKind(kind string) (*GroupRule, *ResourceRule) {
	for _, group := range rr.Rules {
		for _, res := range group.ResourceRules {
			if res.Kind == kind {
				return &group.GroupRule, &res
			}
		}
	}
	return nil, nil
}

func (rr *RewriteRules) RenameResource(resource string) string {
	return rr.ResourceTypePrefix + resource
}

func (rr *RewriteRules) RenameKind(kind string) string {
	return rr.KindPrefix + kind
}

// RestoreResource restores renamed resource to its original state, keeping suffix.
// E.g. "prefixedsomeresources/scale" will be restored to "someresources/scale".
func (rr *RewriteRules) RestoreResource(resource string) string {
	return strings.TrimPrefix(resource, rr.ResourceTypePrefix)
}

func (rr *RewriteRules) RestoreKind(kind string) string {
	return strings.TrimPrefix(kind, rr.KindPrefix)
}

// RestoreApiVersion returns apiVersion with restored apiGroup part.
// It keeps with version suffix as-is if present.
func (rr *RewriteRules) RestoreApiVersion(apiVersion string) string {
	apiGroup, version, found := strings.Cut(apiVersion, "/")

	// No version suffix find, consider apiVersion is only a group name.
	if !found {
		return rr.apiGroupsIndex.Restore(apiVersion)
	}

	// Restore apiGroup part, keep version suffix.
	return rr.apiGroupsIndex.Restore(apiGroup) + "/" + version
}

// RenameApiVersion returns apiVersion with renamed apiGroup part.
// It keeps with version suffix as-is if present.
func (rr *RewriteRules) RenameApiVersion(apiVersion string) string {
	apiGroup, version, found := strings.Cut(apiVersion, "/")

	// No version suffix find, consider apiVersion is only a group name.
	if !found {
		return rr.apiGroupsIndex.Rename(apiVersion)
	}

	// Rename apiGroup part, keep version suffix.
	return rr.apiGroupsIndex.Rename(apiGroup) + "/" + version
}

func (rr *RewriteRules) RenameCategories(categories []string) []string {
	if len(categories) == 0 {
		return []string{}
	}
	return rr.Categories
}

func (rr *RewriteRules) RestoreCategories(resourceRule *ResourceRule) []string {
	if resourceRule == nil {
		return []string{}
	}
	return resourceRule.Categories
}

func (rr *RewriteRules) RenameShortName(shortName string) string {
	return rr.ShortNamePrefix + shortName
}

func (rr *RewriteRules) RestoreShortName(shortName string) string {
	return strings.TrimPrefix(shortName, rr.ShortNamePrefix)
}

func (rr *RewriteRules) RenameShortNames(shortNames []string) []string {
	newNames := make([]string, 0, len(shortNames))
	for _, shortName := range shortNames {
		newNames = append(newNames, rr.ShortNamePrefix+shortName)
	}
	return newNames
}

func (rr *RewriteRules) RestoreShortNames(shortNames []string) []string {
	newNames := make([]string, 0, len(shortNames))
	for _, shortName := range shortNames {
		newNames = append(newNames, strings.TrimPrefix(shortName, rr.ShortNamePrefix))
	}
	return newNames
}

func (rr *RewriteRules) LabelsRewriter() *PrefixedNameRewriter {
	return rr.labelsRewriter
}

func (rr *RewriteRules) AnnotationsRewriter() *PrefixedNameRewriter {
	return rr.annotationsRewriter
}

func (rr *RewriteRules) FinalizersRewriter() *PrefixedNameRewriter {
	return rr.finalizersRewriter
}

// ShouldExclude returns true if object should be excluded from response back to the client.
// Set kind when obj has no kind, e.g. a list item.
func (rr *RewriteRules) ShouldExclude(obj []byte, kind string) bool {
	for _, exclude := range rr.Excludes {
		if exclude.Match(obj, kind) {
			return true
		}
	}
	return false
}

// Match returns true if object matches all conditions in the exclude rule.
func (r ExcludeRule) Match(obj []byte, kind string) bool {
	objKind := kind
	if objKind == "" {
		objKind = gjson.GetBytes(obj, "kind").String()
	}
	kindMatch := len(r.Kinds) == 0
	for _, kind := range r.Kinds {
		if objKind == kind {
			kindMatch = true
			break
		}
	}

	objLabels := mapStringStringFromBytes(obj, "metadata.labels")
	matchLabels := len(r.MatchLabels) == 0 || mapContainsMap(objLabels, r.MatchLabels)

	matchName := len(r.MatchNames) == 0
	objName := gjson.GetBytes(obj, "metadata.name").String()
	for _, name := range r.MatchNames {
		if objName == name {
			matchName = true
			break
		}
	}

	// Return true if every condition match.
	return kindMatch && matchLabels && matchName
}

func mapStringStringFromBytes(obj []byte, path string) map[string]string {
	result := make(map[string]string)
	for field, value := range gjson.GetBytes(obj, path).Map() {
		result[field] = value.String()
	}
	return result
}

func mapContainsMap(obj, match map[string]string) bool {
	if len(match) == 0 {
		return true
	}
	for k, v := range match {
		if obj[k] != v {
			return false
		}
	}
	return true
}
