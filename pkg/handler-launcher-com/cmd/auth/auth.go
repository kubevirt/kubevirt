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

// Package auth provides a gRPC service for authenticating virt-launcher
// pods via projected ServiceAccount tokens. The verifier (virt-handler)
// requests the token and validates it through the TokenReview API.
//
// This is a hand-written gRPC service following the same pattern as
// pkg/handler-launcher-com/cmd/info. The proto definition it represents:
//
//	service CmdAuth {
//	  rpc Authenticate (AuthRequest) returns (AuthResponse) {}
//	}
//	message AuthRequest {}
//	message AuthResponse { string token = 1; }
package auth

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
)

const Audience = "kubevirt.io/cmd-auth"

// AuthRequest is the request message for Authenticate.
type AuthRequest struct{}

func (m *AuthRequest) Reset()         {}
func (m *AuthRequest) String() string { return "AuthRequest{}" }
func (m *AuthRequest) ProtoMessage()  {}

// AuthResponse is the response message for Authenticate, carrying the
// projected SA token.
type AuthResponse struct {
	Token string `protobuf:"bytes,1,opt,name=token" json:"token,omitempty"`
}

func (m *AuthResponse) Reset()         { *m = AuthResponse{} }
func (m *AuthResponse) String() string { return fmt.Sprintf("AuthResponse{Token: %q}", m.Token) }
func (m *AuthResponse) ProtoMessage()  {}

func (m *AuthResponse) GetToken() string {
	if m != nil {
		return m.Token
	}
	return ""
}

// CmdAuthClient is the client interface for the CmdAuth service.
type CmdAuthClient interface {
	Authenticate(ctx context.Context, in *AuthRequest, opts ...grpc.CallOption) (*AuthResponse, error)
}

type cmdAuthClient struct {
	cc *grpc.ClientConn
}

func NewCmdAuthClient(cc *grpc.ClientConn) CmdAuthClient {
	return &cmdAuthClient{cc}
}

func (c *cmdAuthClient) Authenticate(ctx context.Context, in *AuthRequest, opts ...grpc.CallOption) (*AuthResponse, error) {
	out := new(AuthResponse)
	err := grpc.Invoke(ctx, "/kubevirt.cmd.auth.CmdAuth/Authenticate", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// CmdAuthServer is the server interface for the CmdAuth service.
type CmdAuthServer interface {
	Authenticate(context.Context, *AuthRequest) (*AuthResponse, error)
}

func RegisterCmdAuthServer(s *grpc.Server, srv CmdAuthServer) {
	s.RegisterService(&_CmdAuth_serviceDesc, srv)
}

func _CmdAuth_Authenticate_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AuthRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CmdAuthServer).Authenticate(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/kubevirt.cmd.auth.CmdAuth/Authenticate",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CmdAuthServer).Authenticate(ctx, req.(*AuthRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _CmdAuth_serviceDesc = grpc.ServiceDesc{
	ServiceName: "kubevirt.cmd.auth.CmdAuth",
	HandlerType: (*CmdAuthServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Authenticate",
			Handler:    _CmdAuth_Authenticate_Handler,
		},
	},
	Streams: []grpc.StreamDesc{},
}
