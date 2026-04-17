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
	"regexp"
	"strings"

	"github.com/tidwall/gjson"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type RuleBasedRewriter struct {
	Rules *RewriteRules
}

type Action string

const (
	// Restore is an action to restore resources to original.
	Restore Action = "restore"
	// Rename is an action to rename original resources.
	Rename Action = "rename"
)

// RewriteAPIEndpoint renames group and resource in /apis/* endpoints.
// It assumes that ep contains original group and resourceType.
// Restoring of path is not implemented.
func (rw *RuleBasedRewriter) RewriteAPIEndpoint(ep *APIEndpoint) *APIEndpoint {
	var rwrEndpoint *APIEndpoint

	switch {
	case ep.IsRoot || ep.IsCore || ep.IsUnknown:
		// Leave paths /, /api, /api/*, and unknown paths as is.
	case ep.IsCRD:
		// Rename CRD name resourcetype.group for resources with rules.
		rwrEndpoint = rw.rewriteCRDEndpoint(ep.Clone())
	default:
		// Rewrite group and resourceType parts for resources with rules.
		rwrEndpoint = rw.rewriteCRApiEndpoint(ep.Clone())
	}

	rewritten := rwrEndpoint != nil

	if rwrEndpoint == nil {
		rwrEndpoint = ep.Clone()
	}

	// Rewrite key and values if query has labelSelector.
	if strings.Contains(ep.RawQuery, "labelSelector") {
		newRawQuery := rw.rewriteLabelSelector(rwrEndpoint.RawQuery)
		if newRawQuery != rwrEndpoint.RawQuery {
			rewritten = true
			rwrEndpoint.RawQuery = newRawQuery
		}
	}

	if rewritten {
		return rwrEndpoint
	}

	return nil
}

func (rw *RuleBasedRewriter) rewriteCRDEndpoint(ep *APIEndpoint) *APIEndpoint {
	// Rewrite fieldSelector if CRD list is requested.
	if ep.CRDGroup == "" && ep.CRDResourceType == "" {
		if strings.Contains(ep.RawQuery, "metadata.name") {
			// Rewrite name in field selector if any.
			newQuery := rw.rewriteFieldSelector(ep.RawQuery)
			if newQuery != "" {
				res := ep.Clone()
				res.RawQuery = newQuery
				return res
			}
		}
		return nil
	}

	// Check if resource has rules
	_, resourceRule := rw.Rules.ResourceRules(ep.CRDGroup, ep.CRDResourceType)
	if resourceRule == nil {
		// No rewrite for CRD without rules.
		return nil
	}
	// Rewrite group and resourceType in CRD name.
	res := ep.Clone()
	res.CRDGroup = rw.Rules.RenameApiVersion(ep.CRDGroup)
	res.CRDResourceType = rw.Rules.RenameResource(res.CRDResourceType)
	res.Name = res.CRDResourceType + "." + res.CRDGroup
	return res
}

func (rw *RuleBasedRewriter) rewriteCRApiEndpoint(ep *APIEndpoint) *APIEndpoint {
	// Early return if request has no group, e.g. discovery.
	if ep.Group == "" {
		return nil
	}

	// Rename group and resource for CR requests.
	// Check if group has rules. Return early if not.
	groupRule := rw.Rules.GroupRule(ep.Group)
	if groupRule == nil {
		// No group and resourceType rewrite for group without rules.
		return nil
	}
	newGroup := rw.Rules.RenameApiVersion(ep.Group)

	// Shortcut: return clone if only group is requested.
	newResource := ""
	if ep.ResourceType != "" {
		_, resRule := rw.Rules.ResourceRules(ep.Group, ep.ResourceType)
		if resRule == nil {
			// No group and resourceType rewrite for resourceType without rules.
			return nil
		}
		newResource = rw.Rules.RenameResource(ep.ResourceType)
	}

	// Return rewritten endpoint if group or resource are changed.
	if newGroup != "" || newResource != "" {
		res := ep.Clone()
		if newGroup != "" {
			res.Group = newGroup
		}
		if newResource != "" {
			res.ResourceType = newResource
		}

		return res
	}

	return nil
}

var metadataNameRe = regexp.MustCompile(`metadata.name\%3D([a-z0-9-]+)((\.[a-z0-9-]+)*)`)

// rewriteFieldSelector rewrites value for metadata.name in fieldSelector of CRDs listing.
// Example request:
// https://APISERVER/apis/apiextensions.k8s.io/v1/customresourcedefinitions?fieldSelector=metadata.name%3Dresources.original.group.io&...
func (rw *RuleBasedRewriter) rewriteFieldSelector(rawQuery string) string {
	matches := metadataNameRe.FindStringSubmatch(rawQuery)
	if matches == nil {
		return ""
	}

	resourceType := matches[1]
	group := matches[2]
	group = strings.TrimPrefix(group, ".")

	_, resRule := rw.Rules.ResourceRules(group, resourceType)
	if resRule == nil {
		return ""
	}

	group = rw.Rules.RenameApiVersion(group)
	resourceType = rw.Rules.RenameResource(resourceType)

	newSelector := `metadata.name%3D` + resourceType + "." + group

	return metadataNameRe.ReplaceAllString(rawQuery, newSelector)
}

// rewriteLabelSelector rewrites labels in labelSelector
// Example request:
// https://<apiserver>/apis/apps/v1/namespaces/<namespace>/deployments?labelSelector=app%3Dsomething
func (rw *RuleBasedRewriter) rewriteLabelSelector(rawQuery string) string {
	q, err := url.ParseQuery(rawQuery)
	if err != nil {
		return rawQuery
	}
	lsq := q.Get("labelSelector")
	if lsq == "" {
		return rawQuery
	}

	labelSelector, err := metav1.ParseToLabelSelector(lsq)
	if err != nil {
		// The labelSelector is not well-formed. We pass it through, so
		// API Server will return an error.
		return rawQuery
	}

	// Return early if labelSelector is empty, e.g. ?labelSelector=&limit=500
	if labelSelector == nil {
		return rawQuery
	}

	rwrMatchLabels := rw.Rules.LabelsRewriter().RenameMap(labelSelector.MatchLabels)

	rwrMatchExpressions := make([]metav1.LabelSelectorRequirement, 0)
	for _, expr := range labelSelector.MatchExpressions {
		rwrExpr := expr
		rwrExpr.Key, rwrExpr.Values = rw.Rules.LabelsRewriter().RewriteNameValues(rwrExpr.Key, rwrExpr.Values, Rename)
		rwrMatchExpressions = append(rwrMatchExpressions, rwrExpr)
	}

	rwrLabelSelector := &metav1.LabelSelector{
		MatchLabels:      rwrMatchLabels,
		MatchExpressions: rwrMatchExpressions,
	}

	res, err := metav1.LabelSelectorAsSelector(rwrLabelSelector)
	if err != nil {
		return rawQuery
	}

	q.Set("labelSelector", res.String())
	return q.Encode()
}

// RewriteJSONPayload does rewrite based on kind.
// TODO(future refactor): Remove targetReq in all callers.
func (rw *RuleBasedRewriter) RewriteJSONPayload(_ *TargetRequest, obj []byte, action Action) ([]byte, error) {
	// Detect Kind
	kind := gjson.GetBytes(obj, "kind").String()

	var rwrBytes []byte
	var err error

	obj, err = rw.FilterExcludes(obj, action)
	if err != nil {
		return obj, err
	}

	switch kind {
	case "APIGroupList":
		rwrBytes, err = RewriteAPIGroupList(rw.Rules, obj)

	case "APIGroup":
		rwrBytes, err = RewriteAPIGroup(rw.Rules, obj)

	case "APIResourceList":
		rwrBytes, err = RewriteAPIResourceList(rw.Rules, obj)

	case "APIGroupDiscoveryList":
		rwrBytes, err = RewriteAPIGroupDiscoveryList(rw.Rules, obj)

	case "AdmissionReview":
		rwrBytes, err = RewriteAdmissionReview(rw.Rules, obj)

	case CRDKind, CRDListKind:
		rwrBytes, err = RewriteCRDOrList(rw.Rules, obj, action)

	case MutatingWebhookConfigurationKind,
		MutatingWebhookConfigurationListKind:
		rwrBytes, err = RewriteMutatingOrList(rw.Rules, obj, action)

	case ValidatingWebhookConfigurationKind,
		ValidatingWebhookConfigurationListKind:
		rwrBytes, err = RewriteValidatingOrList(rw.Rules, obj, action)

	case EventKind, EventListKind:
		rwrBytes, err = RewriteEventOrList(rw.Rules, obj, action)

	case ClusterRoleKind, ClusterRoleListKind:
		rwrBytes, err = RewriteClusterRoleOrList(rw.Rules, obj, action)

	case RoleKind, RoleListKind:
		rwrBytes, err = RewriteRoleOrList(rw.Rules, obj, action)
	case DeploymentKind, DeploymentListKind:
		rwrBytes, err = RewriteDeploymentOrList(rw.Rules, obj, action)
	case StatefulSetKind, StatefulSetListKind:
		rwrBytes, err = RewriteStatefulSetOrList(rw.Rules, obj, action)
	case DaemonSetKind, DaemonSetListKind:
		rwrBytes, err = RewriteDaemonSetOrList(rw.Rules, obj, action)
	case PodKind, PodListKind:
		rwrBytes, err = RewritePodOrList(rw.Rules, obj, action)
	case PodDisruptionBudgetKind, PodDisruptionBudgetListKind:
		rwrBytes, err = RewritePDBOrList(rw.Rules, obj, action)
	case JobKind, JobListKind:
		rwrBytes, err = RewriteJobOrList(rw.Rules, obj, action)
	case ServiceKind, ServiceListKind:
		rwrBytes, err = RewriteServiceOrList(rw.Rules, obj, action)
	case PersistentVolumeClaimKind, PersistentVolumeClaimListKind:
		rwrBytes, err = RewritePVCOrList(rw.Rules, obj, action)

	case ServiceMonitorKind, ServiceMonitorListKind:
		rwrBytes, err = RewriteServiceMonitorOrList(rw.Rules, obj, action)

	case ValidatingAdmissionPolicyBindingKind, ValidatingAdmissionPolicyBindingListKind:
		rwrBytes, err = RewriteValidatingAdmissionPolicyBindingOrList(rw.Rules, obj, action)
	case ValidatingAdmissionPolicyKind, ValidatingAdmissionPolicyListKind:
		rwrBytes, err = RewriteValidatingAdmissionPolicyOrList(rw.Rules, obj, action)
	default:
		// TODO Add rw.Rules.IsKnownKind() to rewrite only known kinds.
		rwrBytes, err = RewriteCustomResourceOrList(rw.Rules, obj, action)
	}
	// Return obj bytes as-is in case of the error.
	if err != nil {
		return obj, err
	}

	// Always rewrite metadata: labels, annotations, finalizers, ownerReferences.
	// TODO: add rewriter for managedFields.
	return RewriteResourceOrList2(rwrBytes, func(singleObj []byte) ([]byte, error) {
		return TransformObject(singleObj, "metadata", func(metadataObj []byte) ([]byte, error) {
			return RewriteMetadata(rw.Rules, metadataObj, action)
		})
	})
}

// RestoreBookmark restores apiVersion and kind in an object in WatchEvent with type BOOKMARK. Bookmark is not a full object, so RewriteJSONPayload may add unexpected fields.
// Bookmark example: {"kind":"ConfigMap","apiVersion":"v1","metadata":{"resourceVersion":"438083871","creationTimestamp":null}}
func (rw *RuleBasedRewriter) RestoreBookmark(targetReq *TargetRequest, obj []byte) ([]byte, error) {
	return RestoreAPIVersionAndKind(rw.Rules, obj)
}

// RewritePatch rewrites patches for some known objects.
// Only rename action is required for patches.
func (rw *RuleBasedRewriter) RewritePatch(targetReq *TargetRequest, patchBytes []byte) ([]byte, error) {
	_, resRule := rw.Rules.ResourceRules(targetReq.OrigGroup(), targetReq.OrigResourceType())
	if resRule != nil {
		if targetReq.IsCRD() {
			return RenameCRDPatch(rw.Rules, resRule, patchBytes)
		}
		return RenameResourcePatch(rw.Rules, patchBytes)
	}

	switch targetReq.OrigResourceType() {
	case "services":
		return RenameServicePatch(rw.Rules, patchBytes)
	case "deployments",
		"daemonsets",
		"statefulsets":
		return RenameSpecTemplatePatch(rw.Rules, patchBytes)
	case "validatingwebhookconfigurations",
		"mutatingwebhookconfigurations":
		return RenameWebhookConfigurationPatch(rw.Rules, patchBytes)
	}

	return RenameMetadataPatch(rw.Rules, patchBytes)
}

// FilterExcludes removes excluded resources from the list or return SkipItem if resource itself is excluded.
func (rw *RuleBasedRewriter) FilterExcludes(obj []byte, action Action) ([]byte, error) {
	if action != Restore {
		return obj, nil
	}

	kind := gjson.GetBytes(obj, "kind").String()
	if !isExcludableKind(kind) {
		return obj, nil
	}

	if rw.Rules.ShouldExclude(obj, kind) {
		return obj, SkipItem
	}

	// Also check each item if obj is List
	if !strings.HasSuffix(kind, "List") {
		return obj, nil
	}

	singleKind := strings.TrimSuffix(kind, "List")
	obj, err := RewriteResourceOrList2(obj, func(singleObj []byte) ([]byte, error) {
		if rw.Rules.ShouldExclude(singleObj, singleKind) {
			return nil, SkipItem
		}
		return nil, nil
	})
	if err != nil {
		return obj, err
	}
	return obj, nil
}

func shouldRewriteOwnerReferences(resourceType string) bool {
	switch resourceType {
	case CRDKind, CRDListKind,
		RoleKind, RoleListKind,
		RoleBindingKind, RoleBindingListKind,
		PodDisruptionBudgetKind, PodDisruptionBudgetListKind,
		ControllerRevisionKind, ControllerRevisionListKind,
		ClusterRoleKind, ClusterRoleListKind,
		ClusterRoleBindingKind, ClusterRoleBindingListKind,
		APIServiceKind, APIServiceListKind,
		DeploymentKind, DeploymentListKind,
		DaemonSetKind, DaemonSetListKind,
		StatefulSetKind, StatefulSetListKind,
		PodKind, PodListKind,
		JobKind, JobListKind,
		ValidatingWebhookConfigurationKind,
		ValidatingWebhookConfigurationListKind,
		MutatingWebhookConfigurationKind,
		MutatingWebhookConfigurationListKind,
		ServiceKind, ServiceListKind,
		PersistentVolumeClaimKind, PersistentVolumeClaimListKind,
		PrometheusRuleKind, PrometheusRuleListKind,
		ServiceMonitorKind, ServiceMonitorListKind:
		return true
	}

	return false
}

// isExcludeKind returns true if kind may be excluded from rewriting.
// Discovery kinds and AdmissionReview have special schemas, it is sane to
// exclude resources in particular rewriters.
func isExcludableKind(kind string) bool {
	switch kind {
	case "APIGroupList",
		"APIGroup",
		"APIResourceList",
		"APIGroupDiscoveryList",
		"AdmissionReview":
		return false
	}

	return true
}
