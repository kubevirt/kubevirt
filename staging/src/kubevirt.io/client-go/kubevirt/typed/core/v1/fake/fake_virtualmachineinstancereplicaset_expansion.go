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
 * Copyright The KubeVirt Authors
 *
 */

package fake

import (
	"context"

	autov1 "k8s.io/api/autoscaling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/testing"

	v1 "kubevirt.io/api/core/v1"
	fake2 "kubevirt.io/client-go/testing"
)

func (c *fakeVirtualMachineInstanceReplicaSets) GetScale(ctx context.Context, replicaSetName string, options metav1.GetOptions) (*autov1.Scale, error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetSubresourceAction(c.Resource(), c.Namespace(), "scale", replicaSetName), &autov1.Scale{})

	if obj == nil {
		return nil, err
	}
	return obj.(*autov1.Scale), err
}

func (c *fakeVirtualMachineInstanceReplicaSets) UpdateScale(ctx context.Context, replicaSetName string, scale *autov1.Scale) (*autov1.Scale, error) {
	obj, err := c.Fake.
		Invokes(fake2.NewPutSubresourceAction(c.Resource(), c.Namespace(), "scale", replicaSetName, struct{}{}), &autov1.Scale{})

	if obj == nil {
		return nil, err
	}
	return obj.(*autov1.Scale), err
}

func (c *fakeVirtualMachineInstanceReplicaSets) PatchStatus(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions) (*v1.VirtualMachineInstanceReplicaSet, error) {
	return c.Patch(ctx, name, pt, data, opts, "status")
}
