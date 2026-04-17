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

// RewritePath return rewritten TargetPath along with original group and resource type.
// TODO: this rewriter is not conform to S in SOLID. Should split to ParseAPIEndpoint and RewriteAPIEndpoint.
//func (rw *RuleBasedRewriter) RewritePath(urlPath string) (*TargetRequest, error) {
//	// Is it a webhook?
//	if webhookRule, ok := rw.Rules.Webhooks[urlPath]; ok {
//		return &TargetRequest{
//			Webhook: &webhookRule,
//		}, nil
//	}
//
//	// Is it an API request?
//	if strings.HasPrefix(urlPath, "/apis/") || urlPath == "/apis" {
//		// TODO refactor RewriteAPIPath to produce a TargetPath, not an array in PathItems.
//		cleanedPath := strings.Trim(urlPath, "/")
//		pathItems := strings.Split(cleanedPath, "/")
//
//		// First, try to rewrite CRD request.
//		res := RewriteCRDPath(pathItems, rw.Rules)
//		if res != nil {
//			return res, nil
//		}
//		// Next, rewrite usual request.
//		res, err := RewriteAPIsPath(pathItems, rw.Rules)
//		if err != nil {
//			return nil, err
//		}
//		if res == nil {
//			// e.g. no rewrite rule find.
//			return nil, nil
//		}
//		if len(res.PathItems) > 0 {
//			res.TargetPath = "/" + path.Join(res.PathItems...)
//		}
//		return res, nil
//	}
//
//	if strings.HasPrefix(urlPath, "/api/") || urlPath == "/api" {
//		return &TargetRequest{
//			IsCoreAPI: true,
//		}, nil
//	}
//
//	return nil, nil
//}

// Constants with indices of API endpoints portions.
// Request cluster scoped resource:
// - /apis/GROUP/VERSION/RESOURCETYPE/NAME/SUBRESOURCE
//    |    |     |       |
// APISIdx |     |       |
// GroupIDx      |       |
// VersionIDx ---+       |
// ClusterResourceIdx ---+

//
// Request namespaced resource:
// - /apis/GROUP/VERSION/namespaces/NAMESPACE/RESOURCETYPE/NAME/SUBRESOURCE
//                       |          |         |
// NamespacesIdx --------+          |         |
// NamespaceIdx --------------------+         |
// NamespacedResourceIdx----------------------+
//
// Request CRD:
// - /apis/apiextensions.k8s.io/v1/customresourcedefinitions/RESOURCETYPE.GROUP
//         |                       |                         |
// GroupIdx                        |                         |
// ClusterResourceIdx -------------+                         |
// CRDNameIdx -----------------------------------------------+

//const (
//	APISIdx               = 0
//	GroupIdx              = 1
//	VersionIdx            = 2
//	NamespacesIdx         = 3
//	NamespaceIdx          = 4
//	ClusterResourceIdx    = 3
//	NamespacedResourceIdx = 5
//)

// RewriteAPIsPath rewrites GROUP and RESOURCETYPE in these API calls:
// - /apis/GROUP
// - /apis/GROUP/VERSION
// - /apis/GROUP/VERSION/RESOURCETYPE
// - /apis/GROUP/VERSION/RESOURCETYPE/NAME
// - /apis/GROUP/VERSION/RESOURCETYPE/NAME/SUBRESOURCE
//
// - /apis/GROUP/VERSION/namespaces/NAMESPACE/RESOURCETYPE
// - /apis/GROUP/VERSION/namespaces/NAMESPACE/RESOURCETYPE/NAME
// - /apis/GROUP/VERSION/namespaces/NAMESPACE/RESOURCETYPE/NAME/SUBRESOURCE
//func RewriteAPIsPath(pathItems []string, rules *RewriteRules) (*TargetRequest, error) {
//	if len(pathItems) == 0 {
//		return nil, nil
//	}
//
//	res := &TargetRequest{
//		PathItems: make([]string, 0, len(pathItems)),
//	}
//
//	if len(pathItems) == 1 {
//		if pathItems[APISIdx] == "apis" {
//			// Do not rewrite URL, but rewrite response later.
//			res.PathItems = append(res.PathItems, pathItems[APISIdx])
//			return res, nil
//		}
//		// The single path item should be "apis".
//		return nil, nil
//	}
//
//	res.PathItems = append(res.PathItems, pathItems[APISIdx])
//
//	// Check if the GROUP portion match Rules.
//	apiGroupName := ""
//	apiGroupMatch := false
//	group := pathItems[GroupIdx]
//	for groupName, apiGroupRule := range rules.Rules {
//		if apiGroupRule.GroupRule.Group == group {
//			res.OrigGroup = group
//			res.PathItems = append(res.PathItems, rules.RenamedGroup)
//			apiGroupName = groupName
//			apiGroupMatch = true
//			break
//		}
//	}
//
//	if !apiGroupMatch {
//		return nil, nil
//	}
//	// Stop if GROUP is the last item in path.
//	if len(pathItems) <= GroupIdx+1 {
//		return res, nil
//	}
//
//	// Add VERSION portion.
//	res.PathItems = append(res.PathItems, pathItems[VersionIdx])
//	// Stop if VERSION is the last item in path.
//	if len(pathItems) <= VersionIdx+1 {
//		return res, nil
//	}
//
//	// Check is namespaced resource is requested.
//	resourceTypeIdx := ClusterResourceIdx
//	if pathItems[NamespacesIdx] == "namespaces" {
//		res.PathItems = append(res.PathItems, pathItems[NamespacesIdx])
//		res.PathItems = append(res.PathItems, pathItems[NamespaceIdx])
//		resourceTypeIdx = NamespacedResourceIdx
//	}
//
//	// Check if the RESOURCETYPE portion match Rules.
//	resourceType := pathItems[resourceTypeIdx]
//	resourceTypeMatched := true
//	for _, rule := range rules.Rules[apiGroupName].ResourceRules {
//		if rule.Plural == resourceType {
//			res.OrigResourceType = resourceType
//			res.PathItems = append(res.PathItems, rules.RenameResource(rule.Plural))
//			resourceTypeMatched = true
//			break
//		}
//	}
//	if !resourceTypeMatched {
//		return nil, nil
//	}
//	// Return if RESOURCETYPE is the last item in path.
//	if len(pathItems) == resourceTypeIdx+1 {
//		return res, nil
//	}
//
//	// Copy remaining items: NAME and SUBRESOURCE.
//	for i := resourceTypeIdx + 1; i < len(pathItems); i++ {
//		res.PathItems = append(res.PathItems, pathItems[i])
//	}
//
//	return res, nil
//}
