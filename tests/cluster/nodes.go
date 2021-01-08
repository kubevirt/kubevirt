/*
 * This file is part of the kubevirt project
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

package cluster

import (
	. "github.com/onsi/gomega"

	"k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v13 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
)

func GetAllSchedulableNodes(virtClient kubecli.KubevirtClient) *v1.NodeList {
	nodes, err := virtClient.CoreV1().Nodes().List(v12.ListOptions{LabelSelector: v13.NodeSchedulable + "=" + "true"})
	ExpectWithOffset(1, err).ToNot(HaveOccurred(), "Should list compute nodes")
	return nodes
}

