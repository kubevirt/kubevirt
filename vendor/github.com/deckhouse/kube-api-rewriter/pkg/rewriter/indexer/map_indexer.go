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

package indexer

type MapIndexer struct {
	idx     map[string]string
	reverse map[string]string
}

func NewMapIndexer() *MapIndexer {
	return &MapIndexer{
		idx:     make(map[string]string),
		reverse: make(map[string]string),
	}
}

func (m *MapIndexer) AddPair(original, renamed string) {
	m.idx[original] = renamed
	m.reverse[renamed] = original
}

func (m *MapIndexer) Rename(original string) string {
	if renamed, ok := m.idx[original]; ok {
		return renamed
	}
	return original
}

func (m *MapIndexer) Restore(renamed string) string {
	if original, ok := m.reverse[renamed]; ok {
		return original
	}
	return renamed
}

func (m *MapIndexer) IsOriginal(original string) bool {
	_, ok := m.idx[original]
	return ok
}

func (m *MapIndexer) IsRenamed(original string) bool {
	_, ok := m.reverse[original]
	return ok
}
