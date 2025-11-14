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

package dns

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	k8sClient "kubevirt.io/kubevirt/tests/framework/k8s"

	"kubevirt.io/kubevirt/tests/flags"
)

const (
	openshiftDNSServiceName = "dns-default"
	openshiftDNSNamespace   = "openshift-dns"
)

// SearchDomains returns a list of default search name domains.
func SearchDomains() []string {
	return []string{"default.svc.cluster.local", "svc.cluster.local", "cluster.local"}
}

// ClusterDNSServiceIP returns the cluster IP address of the DNS service.
// Attempts first to detect the DNS service on a k8s cluster and if not found on an openshift cluster.
func ClusterDNSServiceIP() (string, error) {
	k8sClientSet := k8sClient.Client()

	service, err := k8sClientSet.CoreV1().Services(flags.DNSServiceNamespace).Get(
		context.Background(), flags.DNSServiceName, metav1.GetOptions{},
	)
	if err != nil {
		prevErr := err
		service, err = k8sClientSet.CoreV1().Services(openshiftDNSNamespace).Get(context.Background(), openshiftDNSServiceName, metav1.GetOptions{})
		if err != nil {
			return "", fmt.Errorf("unable to detect the DNS services: %v, %v", prevErr, err)
		}
	}
	return service.Spec.ClusterIP, nil
}
