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

package apply

import (
	"context"
	"fmt"
	"net/url"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	apiregv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/install"
)

const (
	customMetricsAPIServiceName = "v1beta1.custom.metrics.k8s.io"

	// exportProxyHPAMetricsDetectionTimeout bounds custom-metrics API probes during
	// HPA profile auto-detection so a hung aggregator does not stall reconcile.
	exportProxyHPAMetricsDetectionTimeout = 10 * time.Second
)

func (r *Reconciler) resolveExportProxyHPAMetricsProfile(namespace string) components.ExportProxyHPAMetricsProfile {
	if r.exportProxyHPAMetricsProfileResolved != "" {
		log.Log.V(2).Infof("export-proxy HPA metrics profile already resolved this sync for namespace %s: %s",
			namespace, r.exportProxyHPAMetricsProfileResolved)
		return r.exportProxyHPAMetricsProfileResolved
	}

	resolver := r.exportProxyHPAProfileResolver
	if resolver == nil {
		resolver = r.autoDetectExportProxyHPAMetricsProfile
	}

	profile := resolver(namespace)
	r.exportProxyHPAMetricsProfileResolved = profile
	return profile
}

func (r *Reconciler) autoDetectExportProxyHPAMetricsProfile(namespace string) components.ExportProxyHPAMetricsProfile {
	return r.exportProxyHPAMetricsProfileCache.Resolve(namespace, func() components.ExportProxyHPAMetricsProfile {
		ctx, cancel := context.WithTimeout(context.Background(), exportProxyHPAMetricsDetectionTimeout)
		defer cancel()
		return detectExportProxyHPAMetricsProfile(ctx, r.aggregatorclient, r.k8sClient, namespace)
	})
}

func detectExportProxyHPAMetricsProfile(ctx context.Context, aggregator install.APIServiceInterface, kubeClient kubernetes.Interface, namespace string) components.ExportProxyHPAMetricsProfile {
	if !customMetricsAPIServiceAvailable(ctx, aggregator) {
		log.Log.Infof("export-proxy HPA using %s profile: APIService %s is unavailable",
			components.ExportProxyHPAMetricsProfileResource, customMetricsAPIServiceName)
		return components.ExportProxyHPAMetricsProfileResource
	}

	// Use Discovery's unversioned REST client. CoreV1().RESTClient().AbsPath() joins onto
	// /api/v1 and would probe /api/v1/apis/custom.metrics..., which never succeeds.
	restClient := kubeClient.Discovery().RESTClient()
	if err := exportProxyCustomMetricsAvailable(ctx, restClient, namespace); err != nil {
		log.Log.Infof("export-proxy HPA using %s profile: custom metrics probes failed err=%s",
			components.ExportProxyHPAMetricsProfileResource, formatProbeError(err))
		return components.ExportProxyHPAMetricsProfileResource
	}

	podPath := exportProxyPodMetricProbePath(namespace)
	namespacePath := exportProxyNamespaceObjectMetricProbePath(namespace)
	log.Log.Infof("export-proxy HPA using %s profile: pods=%s namespace=%s",
		components.ExportProxyHPAMetricsProfileCustomMetrics, podPath, namespacePath)
	return components.ExportProxyHPAMetricsProfileCustomMetrics
}

func customMetricsAPIServiceAvailable(ctx context.Context, aggregator install.APIServiceInterface) bool {
	if aggregator == nil {
		return false
	}

	apiSvc, err := aggregator.Get(ctx, customMetricsAPIServiceName, metav1.GetOptions{})
	if err != nil {
		return false
	}

	for _, cond := range apiSvc.Status.Conditions {
		if cond.Type == apiregv1.Available && cond.Status == apiregv1.ConditionTrue {
			return true
		}
	}

	return false
}

func exportProxyCustomMetricsAvailable(ctx context.Context, restClient rest.Interface, namespace string) error {
	podPath := exportProxyPodMetricProbePath(namespace)
	if err := probeCustomMetricPath(ctx, restClient, podPath); err != nil {
		return fmt.Errorf("pods metric probe failed path=%s: %w", podPath, err)
	}

	namespacePath := exportProxyNamespaceObjectMetricProbePath(namespace)
	if err := probeCustomMetricPath(ctx, restClient, namespacePath); err != nil {
		return fmt.Errorf("namespace metric probe failed path=%s: %w", namespacePath, err)
	}

	return nil
}

func exportProxyPodMetricProbePath(namespace string) string {
	// Use a literal "*" here; client-go URL-escapes it to %2A on the wire (matching kubectl --raw).
	return fmt.Sprintf("/apis/custom.metrics.k8s.io/v1beta1/namespaces/%s/pods/*/%s",
		url.PathEscape(namespace), components.ExportProxyActiveTransfersMetricName)
}

// exportProxyNamespaceObjectMetricProbePath returns the custom.metrics.k8s.io path
// for the HPA object metric on Namespace. Metrics that describe a namespace use the
// pseudo-resource "metrics" (not /namespaces/{name}/...); see custom-metrics API
// design and kubernetes HPA GetObjectMetric for Kind=Namespace.
func exportProxyNamespaceObjectMetricProbePath(namespace string) string {
	return fmt.Sprintf("/apis/custom.metrics.k8s.io/v1beta1/namespaces/%s/metrics/%s",
		url.PathEscape(namespace), components.ExportProxyActiveTransfersPodMaxMetricName)
}

// probeCustomMetricPath reports whether a custom metric endpoint responds successfully.
// Any probe error (404, 503, timeout, etc.) is treated as unavailable so auto-detection
// stays on the resource CPU profile until both transfer metrics endpoints are confirmed
// working; upgrading to custom-metrics HPA on a flaky probe would leave scaling broken.
func probeCustomMetricPath(ctx context.Context, restClient rest.Interface, path string) error {
	_, err := restClient.Get().AbsPath(path).DoRaw(ctx)
	return err
}

func formatProbeError(err error) string {
	if status, ok := err.(apierrors.APIStatus); ok {
		s := status.Status()
		return fmt.Sprintf("code=%d reason=%s msg=%q", s.Code, s.Reason, s.Message)
	}
	return err.Error()
}
