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

package v1alpha1

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"

	virtv1 "kubevirt.io/api/core/v1"

	grpcutil "kubevirt.io/kubevirt/pkg/util/net/grpc"
)

const DefaultTimeout = 30 * time.Second

func DialSocket(socketPath string) (*grpc.ClientConn, error) {
	return grpcutil.DialSocketWithTimeout(socketPath, 1)
}

func ExecuteNodeHook(client NodeHookServiceClient, hookPoint string, vmi *virtv1.VirtualMachineInstance, nodeName string, timeout time.Duration) error {
	if timeout == 0 {
		timeout = DefaultTimeout
	}

	vmiJSON, err := json.Marshal(vmi)
	if err != nil {
		return fmt.Errorf("failed to marshal VMI: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	_, err = client.ExecuteNodeHook(ctx, &ExecuteNodeHookRequest{
		HookPoint:   hookPoint,
		Vmi:         vmiJSON,
		NodeContext: &NodeContext{NodeName: nodeName},
	})
	if err != nil {
		if st, ok := status.FromError(err); ok {
			return fmt.Errorf("node hook %s failed with %s: %s", hookPoint, st.Code(), st.Message())
		}
		return fmt.Errorf("node hook %s failed: %w", hookPoint, err)
	}
	return nil
}
