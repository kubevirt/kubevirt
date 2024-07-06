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

package v1

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"
)

type KubeVirtExpansion interface {
	PatchStatus(ctx context.Context, name string, pt types.PatchType, data []byte, patchOptions metav1.PatchOptions) (*v1.KubeVirt, error)
}

func (c *kubeVirts) PatchStatus(ctx context.Context, name string, pt types.PatchType, data []byte, patchOptions metav1.PatchOptions) (*v1.KubeVirt, error) {
	return c.Patch(ctx, name, pt, data, patchOptions, "status")
}
