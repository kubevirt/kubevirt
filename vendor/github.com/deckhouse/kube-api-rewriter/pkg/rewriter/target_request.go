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
	"net/http"
)

type TargetRequest struct {
	originEndpoint *APIEndpoint
	targetEndpoint *APIEndpoint

	webhookRule *WebhookRule
}

func NewTargetRequest(rwr *RuleBasedRewriter, req *http.Request) *TargetRequest {
	if req == nil || req.URL == nil {
		return nil
	}

	// Is it a request to the webhook?
	webhookRule := rwr.Rules.WebhookRule(req.URL.Path)
	if webhookRule != nil {
		return &TargetRequest{
			webhookRule: webhookRule,
		}
	}

	apiEndpoint := ParseAPIEndpoint(req.URL)
	if apiEndpoint == nil {
		return nil
	}

	// rewrite path if needed
	targetEndpoint := rwr.RewriteAPIEndpoint(apiEndpoint)

	return &TargetRequest{
		originEndpoint: apiEndpoint,
		targetEndpoint: targetEndpoint,
	}
}

// Path return possibly rewritten path for target endpoint.
func (tr *TargetRequest) Path() string {
	if tr.targetEndpoint != nil {
		return tr.targetEndpoint.Path()
	}
	if tr.originEndpoint != nil {
		return tr.originEndpoint.Path()
	}
	if tr.webhookRule != nil {
		return tr.webhookRule.Path
	}

	return ""
}

func (tr *TargetRequest) IsCore() bool {
	if tr.originEndpoint != nil {
		return tr.originEndpoint.IsCore
	}
	return false
}

func (tr *TargetRequest) IsCRD() bool {
	if tr.originEndpoint != nil {
		return tr.originEndpoint.IsCRD
	}
	return false
}

func (tr *TargetRequest) IsWatch() bool {
	if tr.originEndpoint != nil {
		return tr.originEndpoint.IsWatch
	}
	return false
}

func (tr *TargetRequest) IsWebhook() bool {
	return tr.webhookRule != nil
}

func (tr *TargetRequest) OrigGroup() string {
	if tr.IsCRD() {
		return tr.originEndpoint.CRDGroup
	}
	if tr.originEndpoint != nil {
		return tr.originEndpoint.Group
	}
	if tr.webhookRule != nil {
		return tr.webhookRule.Group
	}
	return ""
}

func (tr *TargetRequest) OrigResourceType() string {
	if tr.IsCRD() {
		return tr.originEndpoint.CRDResourceType
	}
	if tr.originEndpoint != nil {
		return tr.originEndpoint.ResourceType
	}
	if tr.webhookRule != nil {
		return tr.webhookRule.Resource
	}
	return ""
}

func (tr *TargetRequest) RawQuery() string {
	if tr.targetEndpoint != nil {
		return tr.targetEndpoint.RawQuery
	}
	if tr.originEndpoint != nil {
		return tr.originEndpoint.RawQuery
	}
	return ""
}

func (tr *TargetRequest) RequestURI() string {
	path := tr.Path()
	query := tr.RawQuery()
	if query == "" {
		return path
	}
	return fmt.Sprint(path, "?", query)
}

// ShouldRewriteRequest returns true if incoming payload should
// be rewritten.
func (tr *TargetRequest) ShouldRewriteRequest() bool {
	// Consider known webhook should be rewritten. Unknown paths will be passed as-is.
	if tr.webhookRule != nil {
		return true
	}

	if tr.originEndpoint != nil {
		if tr.originEndpoint.IsRoot || tr.originEndpoint.IsUnknown {
			return false
		}

		if tr.targetEndpoint == nil {
			// Pass resources without rules as is, except some special types.

			// Rewrite request body when creating CRD.
			if tr.originEndpoint.ResourceType == "customresourcedefinitions" && tr.originEndpoint.Name == "" {
				return true
			}

			return shouldRewriteResource(tr.originEndpoint.ResourceType)
		}
	}

	// Payload should be inspected to decide if rewrite is required.
	return true
}

// ShouldRewriteResponse return true if response rewrite is needed.
// Response may be passed as is if false.
func (tr *TargetRequest) ShouldRewriteResponse() bool {
	// If there is webhook rule, response should be rewritten.
	if tr.webhookRule != nil {
		return true
	}

	if tr.originEndpoint == nil {
		return false
	}

	if tr.originEndpoint.IsRoot || tr.originEndpoint.IsUnknown {
		return false
	}

	if tr.originEndpoint.IsCRD {
		// Rewrite CRD List.
		if tr.originEndpoint.Name == "" {
			return true
		}
		// Rewrite CRD if group and resource was rewritten.
		if tr.originEndpoint.Name != "" && tr.targetEndpoint != nil {
			return true
		}
		return false
	}

	// Rewrite if path was rewritten for known resource.
	if tr.targetEndpoint != nil {
		return true
	}

	// Rewrite response from /apis discovery.
	if tr.originEndpoint.Group == "" {
		return true
	}

	return shouldRewriteResource(tr.originEndpoint.ResourceType)
}

func (tr *TargetRequest) ResourceForLog() string {
	if tr.webhookRule != nil {
		return tr.webhookRule.Resource
	}
	if tr.originEndpoint != nil {
		ep := tr.originEndpoint
		if ep.IsRoot {
			return "ROOT"
		}
		if ep.IsUnknown {
			return "UKNOWN"
		}
		if ep.IsCore {
			// /api
			if ep.Version == "" {
				return "APIVersions/core"
			}
			// /api/v1
			if ep.ResourceType == "" {
				return "APIResourceList/core"
			}
			// /api/v1/RESOURCE/NAME/SUBRESOURCE
			// /api/v1/namespaces/NS/status
			// /api/v1/namespaces/NS/RESOURCE/NAME/SUBRESOURCE
			if ep.Subresource != "" {
				return ep.ResourceType + "/" + ep.Subresource
			}
			// /api/v1/RESOURCETYPE
			// /api/v1/RESOURCETYPE/NAME
			// /api/v1/namespaces
			// /api/v1/namespaces/NAMESPACE
			// /api/v1/namespaces/NAMESPACE/RESOURCETYPE
			// /api/v1/namespaces/NAMESPACE/RESOURCETYPE/NAME
			return ep.ResourceType
		}
		// /apis
		if ep.Group == "" {
			return "APIGroupList"
		}
		// /apis/GROUP
		if ep.Version == "" {
			return "APIGroup/" + ep.Group
		}
		// /apis/GROUP/VERSION
		if ep.ResourceType == "" {
			return "APIResourceList/" + ep.Group
		}
		// /apis/GROUP/VERSION/RESOURCETYPE/NAME/SUBRESOURCE
		// /apis/GROUP/VERSION/namespaces/NAMESPACE/RESOURCETYPE/NAME/SUBRESOURCE
		if ep.Subresource != "" {
			return ep.ResourceType + "/" + ep.Subresource
		}
		// /apis/GROUP/VERSION/RESOURCETYPE
		// /apis/GROUP/VERSION/RESOURCETYPE/NAME
		// /apis/GROUP/VERSION/namespaces/NAMESPACE/RESOURCETYPE
		// /apis/GROUP/VERSION/namespaces/NAMESPACE/RESOURCETYPE/NAME
		return ep.ResourceType
	}

	return "UNKNOWN"
}

func shouldRewriteResource(resourceType string) bool {
	switch resourceType {
	case "nodes",
		"pods",
		"configmaps",
		"secrets",
		"services",
		"serviceaccounts",
		"mutatingwebhookconfigurations",
		"validatingwebhookconfigurations",
		"clusterroles",
		"roles",
		"rolebindings",
		"clusterrolebindings",
		"deployments",
		"statefulsets",
		"daemonsets",
		"jobs",
		"persistentvolumeclaims",
		"prometheusrules",
		"servicemonitors",
		"poddisruptionbudgets",
		"controllerrevisions",
		"apiservices",
		"validatingadmissionpolicybindings",
		"validatingadmissionpolicies",
		"events":
		return true
	}

	return false
}
