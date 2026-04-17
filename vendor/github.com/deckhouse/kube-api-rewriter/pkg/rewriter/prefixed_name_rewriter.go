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

import "strings"

const PreservedPrefix = "preserved-original-"

type PrefixedNameRewriter struct {
	namesRenameIdx   map[string]string
	namesRestoreIdx  map[string]string
	prefixRenameIdx  map[string]string
	prefixRestoreIdx map[string]string
}

func NewPrefixedNameRewriter(replaceRules MetadataReplace) *PrefixedNameRewriter {
	return &PrefixedNameRewriter{
		namesRenameIdx:   indexRules(replaceRules.Names),
		namesRestoreIdx:  indexRulesReverse(replaceRules.Names),
		prefixRenameIdx:  indexRules(replaceRules.Prefixes),
		prefixRestoreIdx: indexRulesReverse(replaceRules.Prefixes),
	}
}

func (p *PrefixedNameRewriter) Rewrite(name string, action Action) string {
	switch action {
	case Rename:
		name, _ = p.rename(name, "")
	case Restore:
		name, _ = p.restore(name, "")
	}
	return name
}

func (p *PrefixedNameRewriter) RewriteNameValue(name, value string, action Action) (string, string) {
	switch action {
	case Rename:
		return p.rename(name, value)
	case Restore:
		return p.restore(name, value)
	}
	return name, value
}

func (p *PrefixedNameRewriter) RewriteNameValues(name string, values []string, action Action) (string, []string) {
	if len(values) == 0 {
		return p.Rewrite(name, action), values
	}
	switch action {
	case Rename:
		return p.rewriteNameValues(name, values, p.rename)
	case Restore:
		return p.rewriteNameValues(name, values, p.restore)
	}
	return name, values
}

func (p *PrefixedNameRewriter) RewriteSlice(names []string, action Action) []string {
	switch action {
	case Rename:
		return p.rewriteSlice(names, p.rename)
	case Restore:
		return p.rewriteSlice(names, p.restore)
	}
	return names
}

func (p *PrefixedNameRewriter) RewriteMap(names map[string]string, action Action) map[string]string {
	switch action {
	case Rename:
		return p.rewriteMap(names, p.rename)
	case Restore:
		return p.rewriteMap(names, p.restore)
	}
	return names
}

func (p *PrefixedNameRewriter) Rename(name, value string) (string, string) {
	return p.rename(name, value)
}

func (p *PrefixedNameRewriter) Restore(name, value string) (string, string) {
	return p.restore(name, value)
}

func (p *PrefixedNameRewriter) RenameSlice(names []string) []string {
	return p.rewriteSlice(names, p.rename)
}

func (p *PrefixedNameRewriter) RestoreSlice(names []string) []string {
	return p.rewriteSlice(names, p.restore)
}

func (p *PrefixedNameRewriter) RenameMap(names map[string]string) map[string]string {
	return p.rewriteMap(names, p.rename)
}

func (p *PrefixedNameRewriter) RestoreMap(names map[string]string) map[string]string {
	return p.rewriteMap(names, p.restore)
}

// rewriteNameValues rewrite name and values, e.g. for matchExpressions.
// Method uses all rules to detect a new name, first matching rule is applied.
// Values may be rewritten partially depending on specified name-value rules.
func (p *PrefixedNameRewriter) rewriteNameValues(name string, values []string, fn func(string, string) (string, string)) (string, []string) {
	rwrName := name
	rwrValues := make([]string, 0, len(values))

	for _, value := range values {
		n, v := fn(name, value)
		// Set new name only for the first matching rule.
		if n != name && rwrName == name {
			rwrName = n
		}
		rwrValues = append(rwrValues, v)
	}

	return rwrName, rwrValues
}

func (p *PrefixedNameRewriter) rewriteMap(names map[string]string, fn func(string, string) (string, string)) map[string]string {
	if names == nil {
		return nil
	}
	result := make(map[string]string)
	for name, value := range names {
		rwrName, rwrValue := fn(name, value)
		result[rwrName] = rwrValue
	}
	return result
}

// rewriteSlice do not rewrite values, only names.
func (p *PrefixedNameRewriter) rewriteSlice(names []string, fn func(string, string) (string, string)) []string {
	if names == nil {
		return nil
	}
	result := make([]string, 0, len(names))
	for _, name := range names {
		rwrName, _ := fn(name, "")
		result = append(result, rwrName)
	}
	return result
}

// rename rewrites original names and values. If label was preserved, rewrite it to original state.
func (p *PrefixedNameRewriter) rename(name, value string) (string, string) {
	if p.isPreserved(name) {
		return p.restorePreservedName(name), value
	}

	// First try to find name and value.
	if value != "" {
		idxKey := joinKV(name, value)
		if renamedIdxValue, ok := p.namesRenameIdx[idxKey]; ok {
			return splitKV(renamedIdxValue)
		}
	}
	// No exact rule for name and value, try to find exact name match.
	if renamed, ok := p.namesRenameIdx[name]; ok {
		return renamed, value
	}
	// No exact name, find prefix.
	prefix, remainder, found := strings.Cut(name, "/")
	if !found {
		return name, value
	}
	if renamedPrefix, ok := p.prefixRenameIdx[prefix]; ok {
		return renamedPrefix + "/" + remainder, value
	}
	return name, value
}

// restore rewrites renamed names and values to their original state.
// If name is already original, preserve it with prefix, to make it unknown for client but keep in place for UPDATE/PATCH operations.
func (p *PrefixedNameRewriter) restore(name, value string) (string, string) {
	if p.isOriginal(name, value) {
		return p.preserveName(name), value
	}

	// First try to find name and value.
	if value != "" {
		idxKey := joinKV(name, value)
		if restoredIdxValue, ok := p.namesRestoreIdx[idxKey]; ok {
			return splitKV(restoredIdxValue)
		}
	}
	// No exact rule for name and value, try to find exact name match.
	if restored, ok := p.namesRestoreIdx[name]; ok {
		return restored, value
	}
	// No exact name, find prefix.
	prefix, remainder, found := strings.Cut(name, "/")
	if !found {
		return name, value
	}
	if restoredPrefix, ok := p.prefixRestoreIdx[prefix]; ok {
		return restoredPrefix + "/" + remainder, value
	}
	return name, value
}

// isOriginal returns true if label should be renamed.
func (p *PrefixedNameRewriter) isOriginal(name, value string) bool {
	if value != "" {
		// Label is "original" if there is rule for renaming name and value.
		idxKey := joinKV(name, value)
		if _, ok := p.namesRenameIdx[idxKey]; ok {
			return true
		}
	}

	// Try to find rule for exact name match.
	if _, ok := p.namesRenameIdx[name]; ok {
		return true
	}
	// No exact name, find rule for prefix.
	prefix, _, found := strings.Cut(name, "/")
	if !found {
		// Label is only a name, but no rule for name found, so it is not "original".
		return false
	}
	if _, ok := p.prefixRenameIdx[prefix]; ok {
		return true
	}
	return false
}

func (p *PrefixedNameRewriter) isPreserved(name string) bool {
	return strings.HasPrefix(name, PreservedPrefix)
}

func (p *PrefixedNameRewriter) preserveName(name string) string {
	return PreservedPrefix + name
}

func (p *PrefixedNameRewriter) restorePreservedName(name string) string {
	return strings.TrimPrefix(name, PreservedPrefix)
}

func indexRules(rules []MetadataReplaceRule) map[string]string {
	idx := make(map[string]string, len(rules))
	for _, rule := range rules {
		if rule.OriginalValue != "" && rule.RenamedValue != "" {
			idxKey := joinKV(rule.Original, rule.OriginalValue)
			idx[idxKey] = rule.Renamed + "=" + rule.RenamedValue
			continue
		}
		idx[rule.Original] = rule.Renamed
	}
	return idx
}

func indexRulesReverse(rules []MetadataReplaceRule) map[string]string {
	idx := make(map[string]string, len(rules))
	for _, rule := range rules {
		if rule.OriginalValue != "" && rule.RenamedValue != "" {
			idxKey := joinKV(rule.Renamed, rule.RenamedValue)
			idx[idxKey] = rule.Original + "=" + rule.OriginalValue
			continue
		}
		idx[rule.Renamed] = rule.Original
	}
	return idx
}

func joinKV(name, value string) string {
	return name + "=" + value
}

func splitKV(idxValue string) (name, value string) {
	name, value, _ = strings.Cut(idxValue, "=")
	return
}
