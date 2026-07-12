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

package driver

import (
	"context"
	"log"

	drav1 "k8s.io/kubelet/pkg/apis/dra/v1"
)

type Driver struct {
	drav1.UnimplementedDRAPluginServer
}

func New() *Driver {
	return &Driver{}
}

func (d *Driver) NodePrepareResources(ctx context.Context, req *drav1.NodePrepareResourcesRequest) (*drav1.NodePrepareResourcesResponse, error) {
	log.Println("NodePrepareResources called")
	return &drav1.NodePrepareResourcesResponse{}, nil
}

func (d *Driver) NodeUnprepareResources(ctx context.Context, req *drav1.NodeUnprepareResourcesRequest) (*drav1.NodeUnprepareResourcesResponse, error) {
	log.Println("NodeUnprepareResources called")
	return &drav1.NodeUnprepareResourcesResponse{}, nil
}
