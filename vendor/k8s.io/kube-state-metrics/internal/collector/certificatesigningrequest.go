/*
Copyright 2019 The Kubernetes Authors All rights reserved.

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

package collector

import (
	"k8s.io/kube-state-metrics/pkg/metric"

	certv1beta1 "k8s.io/api/certificates/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

var (
	descCSRLabelsName          = "kube_certificatesigningrequest_labels"
	descCSRLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descCSRLabelsDefaultLabels = []string{"certificatesigningrequest"}

	csrMetricFamilies = []metric.FamilyGenerator{
		{
			Name: descCSRLabelsName,
			Type: metric.Gauge,
			Help: descCSRLabelsHelp,
			GenerateFunc: wrapCSRFunc(func(j *certv1beta1.CertificateSigningRequest) *metric.Family {
				labelKeys, labelValues := kubeLabelsToPrometheusLabels(j.Labels)
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   labelKeys,
							LabelValues: labelValues,
							Value:       1,
						},
					},
				}
			}),
		},
		{
			Name: "kube_certificatesigningrequest_created",
			Type: metric.Gauge,
			Help: "Unix creation timestamp",
			GenerateFunc: wrapCSRFunc(func(csr *certv1beta1.CertificateSigningRequest) *metric.Family {
				ms := []*metric.Metric{}
				if !csr.CreationTimestamp.IsZero() {
					ms = append(ms, &metric.Metric{
						LabelKeys:   []string{},
						LabelValues: []string{},
						Value:       float64(csr.CreationTimestamp.Unix()),
					})
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		},
		{
			Name: "kube_certificatesigningrequest_condition",
			Type: metric.Gauge,
			Help: "The number of each certificatesigningrequest condition",
			GenerateFunc: wrapCSRFunc(func(csr *certv1beta1.CertificateSigningRequest) *metric.Family {
				return &metric.Family{
					Metrics: addCSRConditionMetrics(csr.Status),
				}
			}),
		},
		{
			Name: "kube_certificatesigningrequest_cert_length",
			Type: metric.Gauge,
			Help: "Length of the issued cert",
			GenerateFunc: wrapCSRFunc(func(csr *certv1beta1.CertificateSigningRequest) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   []string{},
							LabelValues: []string{},
							Value:       float64(len(csr.Status.Certificate)),
						},
					},
				}
			}),
		},
	}
)

func wrapCSRFunc(f func(*certv1beta1.CertificateSigningRequest) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		csr := obj.(*certv1beta1.CertificateSigningRequest)

		metricFamily := f(csr)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys = append(descCSRLabelsDefaultLabels, m.LabelKeys...)
			m.LabelValues = append([]string{csr.Name}, m.LabelValues...)
		}

		return metricFamily
	}
}

func createCSRListWatch(kubeClient clientset.Interface, ns string) cache.ListWatch {
	return cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.CertificatesV1beta1().CertificateSigningRequests().List(opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.CertificatesV1beta1().CertificateSigningRequests().Watch(opts)
		},
	}
}

// addCSRConditionMetrics generates one metric for each possible csr condition status
func addCSRConditionMetrics(cs certv1beta1.CertificateSigningRequestStatus) []*metric.Metric {
	cApproved := 0
	cDenied := 0
	for _, s := range cs.Conditions {
		if s.Type == certv1beta1.CertificateApproved {
			cApproved++
		}
		if s.Type == certv1beta1.CertificateDenied {
			cDenied++
		}
	}

	return []*metric.Metric{
		{
			LabelValues: []string{"approved"},
			Value:       float64(cApproved),
			LabelKeys:   []string{"condition"},
		},
		{
			LabelValues: []string{"denied"},
			Value:       float64(cDenied),
			LabelKeys:   []string{"condition"},
		},
	}
}
