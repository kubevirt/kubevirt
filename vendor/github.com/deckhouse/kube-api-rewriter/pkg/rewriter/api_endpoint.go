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
	"net/url"
	"strings"
)

type APIEndpoint struct {
	// IsUknown indicates that path is unknown for rewriter and should be passed as is.
	IsUnknown bool
	RawPath   string

	IsRoot bool

	Prefix string
	IsCore bool

	Group        string
	Version      string
	Namespace    string
	ResourceType string
	Name         string
	Subresource  string
	Remainder    []string

	IsCRD           bool
	CRDResourceType string
	CRDGroup        string

	IsWatch  bool
	RawQuery string
}

// Core resources:
// - /api/VERSION/RESOURCETYPE
// - /api/VERSION/RESOURCETYPE/NAME
// - /api/VERSION/RESOURCETYPE/NAME/SUBRESOURCE
// - /api/VERSION/namespaces/NAMESPACE/RESOURCETYPE
// - /api/VERSION/namespaces/NAMESPACE/RESOURCETYPE/NAME
// - /api/VERSION/namespaces/NAMESPACE/RESOURCETYPE/NAME/SUBRESOURCE
// - /api/VERSION/namespaces/NAME/SUBRESOURCE - RESOURCETYPE=namespaces
//
// Cluster scoped custom resource:
// - /apis/GROUP/VERSION/RESOURCETYPE/NAME/SUBRESOURCE
//    |      |     |       |
// PrefixIdx |     |       |
// GroupIDx -+     |       |
// VersionIDx -----+       |
// ClusterResourceIdx -----+
//
// Namespaced custom resource:
// - /apis/GROUP/VERSION/namespaces/NAMESPACE/RESOURCETYPE/NAME/SUBRESOURCE
// - /apis/GROUP/VERSION/namespaces/NAMESPACE/RESOURCETYPE/NAME/SUBRESOURCE
//
// CRD (CRD is itself a cluster scoped custom resource):
// - /apis/apiextensions.k8s.io/v1/customresourcedefinitions
// - /apis/apiextensions.k8s.io/v1/customresourcedefinitions/RESOURCETYPE.GROUP

const (
	CorePrefix = "api"
	APIsPrefix = "apis"

	NamespacesPart = "namespaces"

	CRDGroup        = "apiextensions.k8s.io"
	CRDResourceType = "customresourcedefinitions"

	WatchClause = "watch=true"
)

// ParseAPIEndpoint breaks url path by parts.
func ParseAPIEndpoint(apiURL *url.URL) *APIEndpoint {
	rawPath := apiURL.Path
	rawQuery := apiURL.RawQuery
	isWatch := strings.Contains(rawQuery, WatchClause)

	cleanedPath := strings.Trim(apiURL.Path, "/")
	pathItems := strings.Split(cleanedPath, "/")

	if cleanedPath == "" || len(pathItems) == 0 {
		return &APIEndpoint{
			IsRoot:   true,
			IsWatch:  isWatch,
			RawPath:  rawPath,
			RawQuery: rawQuery,
		}
	}

	var ae *APIEndpoint
	// PREFIX is the first item in path.
	prefix := pathItems[0]
	switch prefix {
	case CorePrefix:
		ae = parseCoreEndpoint(pathItems)
	case APIsPrefix:
		ae = parseAPIsEndpoint(pathItems)
	}

	if ae == nil {
		return &APIEndpoint{
			IsUnknown: true,
			RawPath:   rawPath,
			RawQuery:  rawQuery,
		}
	}

	ae.IsWatch = isWatch
	ae.RawPath = rawPath
	ae.RawQuery = rawQuery
	return ae
}

func parseCoreEndpoint(pathItems []string) *APIEndpoint {
	var isLast bool
	var ae APIEndpoint
	ae.IsCore = true

	// /api
	ae.Prefix, isLast = Shift(&pathItems)
	if isLast {
		return &ae
	}

	// /api/VERSION
	ae.Version, isLast = Shift(&pathItems)
	if isLast {
		return &ae
	}

	// /api/VERSION/RESOURCETYPE
	ae.ResourceType, isLast = Shift(&pathItems)
	if isLast {
		return &ae
	}

	// /api/VERSION/RESOURCETYPE/NAME
	ae.Name, isLast = Shift(&pathItems)
	if isLast {
		return &ae
	}

	// /api/VERSION/RESOURCETYPE/NAME/SUBRESOURCE
	// /api/VERSION/namespaces/NAMESPACE/status
	// /api/VERSION/namespaces/NAMESPACE/RESOURCETYPE
	ae.Subresource, isLast = Shift(&pathItems)
	if ae.ResourceType == NamespacesPart && ae.Subresource != "status" {
		// It is a namespaced resource, we got ns name and resourcetype in name and subresource.
		ae.Namespace = ae.Name
		ae.ResourceType = ae.Subresource
		ae.Name = ""
		ae.Subresource = ""
	}
	// Stop if no items available.
	if isLast {
		return &ae
	}

	// /api/VERSION/namespaces/NAMESPACE/RESOURCETYPE/NAME
	ae.Name, isLast = Shift(&pathItems)
	if isLast {
		return &ae
	}
	// /api/VERSION/namespaces/NAMESPACE/RESOURCETYPE/NAME/SUBRESOURCE
	ae.Subresource, isLast = Shift(&pathItems)
	if isLast {
		return &ae
	}

	// Save remaining items if any.
	ae.Remainder = pathItems
	return &ae
}

func parseAPIsEndpoint(pathItems []string) *APIEndpoint {
	var ae APIEndpoint
	var isLast bool

	// /apis
	ae.Prefix, isLast = Shift(&pathItems)
	if isLast {
		return &ae
	}

	// /apis/GROUP
	ae.Group, isLast = Shift(&pathItems)
	if isLast {
		return &ae
	}

	// /apis/GROUP/VERSION
	ae.Version, isLast = Shift(&pathItems)
	if isLast {
		return &ae
	}

	// /apis/GROUP/VERSION/RESOURCETYPE
	ae.ResourceType, isLast = Shift(&pathItems)
	// /apis/apiextensions.k8s.io/VERSION/customresourcedefinitions
	if ae.Group == CRDGroup && ae.ResourceType == CRDResourceType {
		ae.IsCRD = true
	}
	if isLast {
		return &ae
	}

	// /apis/GROUP/VERSION/RESOURCETYPE/NAME
	ae.Name, isLast = Shift(&pathItems)
	if ae.IsCRD {
		ae.CRDResourceType, ae.CRDGroup, _ = strings.Cut(ae.Name, ".")
	}
	if isLast {
		return &ae
	}

	// /apis/GROUP/VERSION/RESOURCETYPE/NAME/SUBRESOURCE
	// /apis/GROUP/VERSION/namespaces/NAMESPACE/RESOURCETYPE
	ae.Subresource, isLast = Shift(&pathItems)
	if ae.ResourceType == NamespacesPart {
		// It is a namespaced resource, we got ns name and resourcetype in name and subresource.
		ae.Namespace = ae.Name
		ae.ResourceType = ae.Subresource
		ae.Name = ""
		ae.Subresource = ""
	}
	// Stop if no items available.
	if isLast {
		return &ae
	}

	// /apis/GROUP/VERSION/namespaces/NAMESPACE/RESOURCETYPE/NAME
	ae.Name, isLast = Shift(&pathItems)
	if isLast {
		return &ae
	}
	// /apis/GROUP/VERSION/namespaces/NAMESPACE/RESOURCETYPE/NAME/SUBRESOURCE
	ae.Subresource, isLast = Shift(&pathItems)
	if isLast {
		return &ae
	}

	// Save remaining items if any.
	ae.Remainder = pathItems
	return &ae
}

func (a *APIEndpoint) Clone() *APIEndpoint {
	clone := *a
	return &clone
}

func (a *APIEndpoint) Path() string {
	if a.IsRoot || a.IsCore || a.IsUnknown {
		return a.RawPath
	}

	ns := ""
	if a.Namespace != "" {
		ns = NamespacesPart + "/" + a.Namespace
	}
	var parts []string
	parts = []string{
		a.Prefix,
		a.Group,
		a.Version,
		ns,
		a.ResourceType,
		a.Name,
		a.Subresource,
	}
	if len(a.Remainder) > 0 {
		parts = append(parts, a.Remainder...)
	}

	nonEmptyParts := make([]string, 0)
	for _, part := range parts {
		if part != "" {
			nonEmptyParts = append(nonEmptyParts, part)
		}
	}

	return "/" + strings.Join(nonEmptyParts, "/")
}

// Shift deletes the first item from the array and returns it.
func Shift(items *[]string) (string, bool) {
	if len(*items) == 0 {
		return "", true
	}

	first := (*items)[0]
	if len(*items) == 1 {
		*items = []string{}
	} else {
		*items = (*items)[1:]
	}
	return first, len(*items) == 0
}
