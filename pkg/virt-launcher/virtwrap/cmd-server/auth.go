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

package cmdserver

import (
	"context"
	"fmt"
	"os"
	"strings"

	"google.golang.org/grpc"

	"kubevirt.io/client-go/log"

	cmdauth "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/auth"
)

const defaultTokenPath = "/var/run/secrets/tokens/cmd-auth-token"

// AuthServer implements the CmdAuth gRPC service on the virt-launcher
// side. It reads the projected ServiceAccount token and returns it to
// the caller (virt-handler) for verification via TokenReview.
type AuthServer struct {
	tokenPath string
}

func newAuthServer() *AuthServer {
	return &AuthServer{tokenPath: defaultTokenPath}
}

func (a *AuthServer) Authenticate(_ context.Context, _ *cmdauth.AuthRequest) (*cmdauth.AuthResponse, error) {
	data, err := os.ReadFile(a.tokenPath)
	if err != nil {
		log.Log.Reason(err).Error("failed to read projected SA token for authentication")
		return nil, fmt.Errorf("reading SA token: %w", err)
	}
	return &cmdauth.AuthResponse{
		Token: strings.TrimSpace(string(data)),
	}, nil
}

func registerAuthServer(grpcServer *grpc.Server) {
	cmdauth.RegisterCmdAuthServer(grpcServer, newAuthServer())
}
