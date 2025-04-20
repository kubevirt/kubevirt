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
 */

package system

import (
	"context"

	"kubevirt.io/kubevirt/pkg/util/tls"
	v1 "kubevirt.io/kubevirt/pkg/vsock/system/v1"
)

type SystemService struct {
	caManager tls.ClientCAManager
}

func (s SystemService) CABundle(ctx context.Context, _ *v1.EmptyRequest) (*v1.Bundle, error) {
	raw, err := s.caManager.GetCurrentRaw()
	if err != nil {
		return nil, err
	}
	return &v1.Bundle{Raw: raw}, nil
}

func NewSystemService(mgr tls.ClientCAManager) *SystemService {
	return &SystemService{caManager: mgr}
}
