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
 * Copyright 2021 IBM, Inc.
 *
 */

package object

import (
	"context"
	"time"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
)

func CreateNamespaceIfNotExist(virtCli kubecli.KubevirtClient, name, scenarioLabel, uuid string) error {
	ns := &k8sv1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				scenarioLabel: uuid,
			},
		},
	}
	log.Log.V(2).Infof("Namespace %s created", name)
	_, err := virtCli.CoreV1().Namespaces().Create(context.Background(), ns, metav1.CreateOptions{})
	if !errors.IsAlreadyExists(err) {
		if err != nil {
			log.Log.Errorf("Error creating namespace %s: %v", name, err)
			return err
		}
	}
	return nil
}

// CleanupNamespaces deletes a collection of namespaces with the given selector
func CleanupNamespaces(virtCli kubecli.KubevirtClient, timeout time.Duration, listOpts *metav1.ListOptions) error {
	log.Log.V(2).Infof("Deleting namespaces with label %s", listOpts.LabelSelector)
	ns, _ := virtCli.CoreV1().Namespaces().List(context.TODO(), *listOpts)
	if len(ns.Items) > 0 {
		for _, ns := range ns.Items {
			err := virtCli.CoreV1().Namespaces().Delete(context.TODO(), ns.Name, metav1.DeleteOptions{})
			if errors.IsNotFound(err) {
				log.Log.V(2).Infof("Namespace %s not found", ns.Name)
				continue
			}
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// WaitForDeleteNamespaces waits to all namespaces with the given selector be deleted
func WaitForDeleteNamespaces(virtCli kubecli.KubevirtClient, timeout time.Duration, listOpts metav1.ListOptions) error {
	return wait.PollImmediate(10*time.Second, timeout, func() (bool, error) {
		ns, err := virtCli.CoreV1().Namespaces().List(context.TODO(), listOpts)
		if err != nil {
			return false, err
		}
		if len(ns.Items) == 0 {
			return true, nil
		}
		log.Log.V(4).Infof("Waiting for %d namespaces labeled with %s to be removed", len(ns.Items), listOpts.LabelSelector)
		return false, nil
	})
}
