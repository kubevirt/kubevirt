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
 * Copyright 2019 Red Hat, Inc.
 *
 */
package grpc

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/util"
)

const (
	CONNECT_TIMEOUT_SECONDS = 2
	unixProtocol            = "unix"
)

func DialSocket(socketPath string) (*grpc.ClientConn, error) {
	return DialSocketWithTimeout(socketPath, 0)
}

func DialSocketWithTimeout(socketPath string, timeout int) (*grpc.ClientConn, error) {

	options := []grpc.DialOption{
		grpc.WithAuthority("localhost"),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, unixProtocol, addr)
		}),
		grpc.WithBlock(), // dial sync in order to catch errors early
	}

	ctxTimeout := time.Duration(timeout+CONNECT_TIMEOUT_SECONDS) * time.Second
	if timeout > 0 {
		ctxTimeout = time.Duration(timeout) * time.Second
	}

	// Combined with the Block option, this context controls how long to wait for establishing the connection.
	// The dial timeout used above, controls the overall duration of the connection (including RCP calls).
	ctx, cancel := context.WithTimeout(context.Background(), ctxTimeout)
	defer cancel()

	return grpc.DialContext(ctx, socketPath, options...)

}

func CreateSocket(socketPath string) (net.Listener, error) {
	os.RemoveAll(socketPath)

	err := util.MkdirAllWithNosec(filepath.Dir(socketPath))
	if err != nil {
		log.Log.Reason(err).Errorf("unable to create directory for unix socket %v", socketPath)
		return nil, err
	}

	socket, err := net.Listen("unix", socketPath)

	if err != nil {
		log.Log.Reason(err).Errorf("failed to create unix socket %v", socketPath)
		return nil, err
	}
	return socket, nil
}
