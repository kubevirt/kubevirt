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

package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"google.golang.org/grpc"

	pluginsv1alpha1 "kubevirt.io/kubevirt/pkg/hooks/plugins/v1alpha1"
)

type server struct{}

func (s *server) GuestDefinition(_ context.Context, req *pluginsv1alpha1.GuestDefinitionRequest) (*pluginsv1alpha1.GuestDefinitionResponse, error) {
	updated, err := mutateDomain(req.GetDomain(), req.GetVmi())
	if err != nil {
		return nil, err
	}
	return &pluginsv1alpha1.GuestDefinitionResponse{Domain: updated}, nil
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: network-queue-cap <socket-path>")
	}
	socketPath := os.Args[1]

	if err := os.MkdirAll(filepath.Dir(socketPath), 0755); err != nil {
		log.Fatalf("mkdir: %v", err)
	}
	os.RemoveAll(socketPath)

	lis, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}

	s := grpc.NewServer()
	pluginsv1alpha1.RegisterLauncherHookServiceServer(s, &server{})

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM)
	go func() {
		<-sigCh
		s.GracefulStop()
	}()

	log.Printf("Listening on %s", socketPath)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("serve: %v", err)
	}
}
