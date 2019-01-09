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
 * Copyright 2019 Red Hat, Inc.
 *
 */
package rbac

import (
	"fmt"

	"kubevirt.io/kubevirt/pkg/log"

	secv1 "github.com/openshift/client-go/security/clientset/versioned/typed/security/v1"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	virtv1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
)

func CreateScc(clientset kubecli.KubevirtClient, kv *virtv1.KubeVirt) error {

	secClient, err := secv1.NewForConfig(clientset.Config())
	if err != nil {
		return fmt.Errorf("unable to create scc client: %v", err)
	}

	privScc, err := secClient.SecurityContextConstraints().Get("privileged", metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			// we are mot on openshift?
			log.Log.V(4).Infof("unable to get scc, we are probably not on openshift: %v", err)
			return nil
		} else {
			return fmt.Errorf("unable to get scc: %v", err)
		}
	}

	var kubeVirtAccounts []string
	prefix := "system:serviceaccount"
	kubeVirtAccounts = append(kubeVirtAccounts, fmt.Sprintf("%s:%s:%s", prefix, kv.Namespace, "kubevirt-privileged"))
	kubeVirtAccounts = append(kubeVirtAccounts, fmt.Sprintf("%s:%s:%s", prefix, kv.Namespace, "kubevirt-apiserver"))
	kubeVirtAccounts = append(kubeVirtAccounts, fmt.Sprintf("%s:%s:%s", prefix, kv.Namespace, "kubevirt-controller"))

	added := false
	users := privScc.Users
	for _, acc := range kubeVirtAccounts {
		if !contains(users, acc) {
			users = append(users, acc)
			added = true
		}
	}
	if added {
		privScc.Users = users
		_, err = secClient.SecurityContextConstraints().Update(privScc)
		if err != nil {
			return fmt.Errorf("unable to update scc: %v", err)
		}
	}

	return nil
}

func contains(users []string, user string) bool {
	for _, u := range users {
		if u == user {
			return true
		}
	}
	return false
}
