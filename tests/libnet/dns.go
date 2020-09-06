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
 * Copyright 2020 Red Hat, Inc.
 *
 */

package libnet

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/client-go/kubecli"
)

const (
	k8sDNSServiceName = "kube-dns"
	k8sDNSNamespace   = "kube-system"

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
	virtClient, err := kubecli.GetKubevirtClient()
	if err != nil {
		return "", err
	}

	service, err := virtClient.CoreV1().Services(k8sDNSNamespace).Get(k8sDNSServiceName, metav1.GetOptions{})
	if err != nil {
		prevErr := err
		service, err = virtClient.CoreV1().Services(openshiftDNSNamespace).Get(openshiftDNSServiceName, metav1.GetOptions{})
		if err != nil {
			return "", fmt.Errorf("unable to detect the DNS services: %v, %v", prevErr, err)
		}
	}
	return service.Spec.ClusterIP, nil
}
