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

package executor

import (
	"context"
	"fmt"
	"time"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/tools/perfscale-load-generator/flags"
)

func CreateNamespaces(virtCli kubecli.KubevirtClient, name string, uuid string) error {
	ns := &k8sv1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				scenarioLabel: uuid,
			},
		},
	}
	_, err := virtCli.CoreV1().Namespaces().Create(context.Background(), ns, metav1.CreateOptions{})
	if !errors.IsAlreadyExists(err) {
		if err != nil {
			log.Log.V(2).Errorf("Error creating namespace %s: %v", name, err)
			return err
		}
	}
	return nil
}

// CleanupNamespaces deletes namespaces with the given selector
func CleanupNamespaces(virtCli kubecli.KubevirtClient, uuid string) error {
	listOptions := metav1.ListOptions{}
	listOptions.LabelSelector = fmt.Sprintf("%s=%s", scenarioLabel, uuid)
	log.Log.V(2).Infof("Deleting namespaces with label %s", listOptions.LabelSelector)
	ns, _ := virtCli.CoreV1().Namespaces().List(context.TODO(), listOptions)
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
	if len(ns.Items) > 0 {
		return waitForDeleteNamespaces(virtCli, listOptions)
	}
	return nil
}

func waitForDeleteNamespaces(virtCli kubecli.KubevirtClient, listOptions metav1.ListOptions) error {
	return wait.PollImmediate(10*time.Second, flags.MaxWaitTimeout, func() (bool, error) {
		ns, err := virtCli.CoreV1().Namespaces().List(context.TODO(), listOptions)
		if err != nil {
			return false, err
		}
		if len(ns.Items) == 0 {
			return true, nil
		}
		log.Log.V(4).Infof("Waiting for %d namespaces labeled with %s to be removed", len(ns.Items), listOptions.LabelSelector)
		return false, nil
	})
}
