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
package grpcserver

import (
	"os"
	"time"

	"google.golang.org/grpc"
	"kubevirt.io/client-go/log"

	grpcutil "kubevirt.io/kubevirt/pkg/util/net/grpc"
)

type registerFunc func(grpcServer *grpc.Server)

func RunServer(stop <-chan struct{}, socketPath string, registerF ...registerFunc) (<-chan struct{}, error) {
	grpcServer := grpc.NewServer([]grpc.ServerOption{}...)

	for _, f := range registerF {
		f(grpcServer)
	}

	sock, err := grpcutil.CreateSocket(socketPath)
	if err != nil {
		return nil, err
	}

	done := make(chan struct{})
	go func() {
		<-stop
		log.Log.Info("stopping grpc server")
		stopped := make(chan struct{})
		go func() {
			grpcServer.Stop()
			close(stopped)
		}()

		select {
		case <-stopped:
			log.Log.Info("grpc server stopped")
		case <-time.After(1 * time.Second):
			log.Log.Error("timeout on stopping the grpc server, continuing anyway.")
		}
		sock.Close()
		os.Remove(socketPath)
		close(done)
	}()

	go func() {
		grpcServer.Serve(sock)
	}()

	return done, nil
}
