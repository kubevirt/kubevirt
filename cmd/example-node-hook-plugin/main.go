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
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"google.golang.org/grpc"

	pluginsv1alpha1 "kubevirt.io/kubevirt/pkg/hooks/plugins/v1alpha1"
)

const markerDir = "/var/run/kubevirt/plugin-test-markers"

type server struct{}

func (s *server) ExecuteNodeHook(_ context.Context, req *pluginsv1alpha1.ExecuteNodeHookRequest) (*pluginsv1alpha1.ExecuteNodeHookResponse, error) {
	log.Printf("ExecuteNodeHook called: hookPoint=%s, nodeName=%s", req.HookPoint, req.GetNodeContext().GetNodeName())

	if err := os.MkdirAll(markerDir, 0755); err != nil {
		return &pluginsv1alpha1.ExecuteNodeHookResponse{
			Success: false,
			Message: fmt.Sprintf("failed to create marker directory: %v", err),
		}, nil
	}

	markerFile := filepath.Join(markerDir, fmt.Sprintf("%s.stamp", req.HookPoint))

	if err := os.WriteFile(markerFile, req.Vmi, 0644); err != nil {
		return &pluginsv1alpha1.ExecuteNodeHookResponse{
			Success: false,
			Message: fmt.Sprintf("failed to write marker file: %v", err),
		}, nil
	}

	log.Printf("Marker file written: %s", markerFile)
	return &pluginsv1alpha1.ExecuteNodeHookResponse{
		Success: true,
		Message: fmt.Sprintf("marker file created at %s", markerFile),
	}, nil
}

func main() {
	socketPath := flag.String("socket", "/var/run/kubevirt/plugins/example-node-hook-plugin.sock", "Path to the gRPC unix socket")
	flag.Parse()

	if err := os.MkdirAll(filepath.Dir(*socketPath), 0755); err != nil {
		log.Fatalf("Failed to create socket directory: %v", err)
	}
	os.Remove(*socketPath)

	lis, err := net.Listen("unix", *socketPath)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", *socketPath, err)
	}
	defer lis.Close()

	grpcServer := grpc.NewServer()
	pluginsv1alpha1.RegisterNodeHookServiceServer(grpcServer, &server{})

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-sigCh
		log.Println("Received shutdown signal")
		grpcServer.GracefulStop()
	}()

	log.Printf("Listening on %s", *socketPath)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("gRPC server failed: %v", err)
	}
}
