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
package controller

import (
	"testing"

	v12 "k8s.io/api/core/v1"

	v1 "kubevirt.io/client-go/api/v1"
)

func TestAddPodCondition(t *testing.T) {

	vmi := v1.NewMinimalVMI("test")

	pc1 := &v12.PodCondition{
		Type:   v12.PodScheduled,
		Status: v12.ConditionFalse,
	}
	pc2 := &v12.PodCondition{
		Type:   v12.PodScheduled,
		Status: v12.ConditionTrue,
	}

	cm := NewVirtualMachineInstanceConditionManager()

	cm.AddPodCondition(vmi, pc1)
	cm.AddPodCondition(vmi, pc2)

	if len(vmi.Status.Conditions) != 1 {
		t.Errorf("There should be exactly 1 condition when muliple conditions of the same type were added")
	}
}
