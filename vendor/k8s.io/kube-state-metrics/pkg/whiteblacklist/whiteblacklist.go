/*
Copyright 2018 The Kubernetes Authors All rights reserved.

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

package whiteblacklist

import (
	"errors"
	"strings"
)

// WhiteBlackList encapsulates the logic needed to filter based on a string.
type WhiteBlackList struct {
	list        map[string]struct{}
	isWhiteList bool
}

// New constructs a new WhiteBlackList based on a white- and a
// blacklist. Only one of them can be not empty.
func New(w, b map[string]struct{}) (*WhiteBlackList, error) {
	if len(w) != 0 && len(b) != 0 {
		return nil, errors.New(
			"whitelist and blacklist are both set, they are mutually exclusive, only one of them can be set",
		)
	}

	white := copyList(w)
	black := copyList(b)

	var list map[string]struct{}
	var isWhiteList bool

	// Default to blacklisting
	if len(white) != 0 {
		list = white
		isWhiteList = true
	} else {
		list = black
		isWhiteList = false
	}

	return &WhiteBlackList{
		list:        list,
		isWhiteList: isWhiteList,
	}, nil
}

// Include includes the given items in the list.
func (l *WhiteBlackList) Include(items []string) {
	if l.isWhiteList {
		for _, item := range items {
			l.list[item] = struct{}{}
		}
	} else {
		for _, item := range items {
			delete(l.list, item)
		}
	}
}

// Exclude excludes the given items from the list.
func (l *WhiteBlackList) Exclude(items []string) {
	if l.isWhiteList {
		for _, item := range items {
			delete(l.list, item)
		}
	} else {
		for _, item := range items {
			l.list[item] = struct{}{}
		}
	}
}

// IsIncluded returns if the given item is included.
func (l *WhiteBlackList) IsIncluded(item string) bool {
	_, exists := l.list[item]

	if l.isWhiteList {
		return exists
	}

	return !exists
}

// IsExcluded returns if the given item is excluded.
func (l *WhiteBlackList) IsExcluded(item string) bool {
	return !l.IsIncluded(item)
}

// Status returns the status of the WhiteBlackList that can e.g. be passed into
// a logger.
func (l *WhiteBlackList) Status() string {
	items := []string{}
	for key := range l.list {
		items = append(items, key)
	}

	if l.isWhiteList {
		return "whitelisting the following items: " + strings.Join(items, ", ")
	}

	return "blacklisting the following items: " + strings.Join(items, ", ")
}

func copyList(l map[string]struct{}) map[string]struct{} {
	newList := map[string]struct{}{}
	for k, v := range l {
		newList[k] = v
	}
	return newList
}
