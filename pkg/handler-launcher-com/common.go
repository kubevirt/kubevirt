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
package handler_launcher_com

import (
	"path/filepath"

	"google.golang.org/grpc"

	cmdinfo "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/info"
	cmdv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"
	notifyinfo "kubevirt.io/kubevirt/pkg/handler-launcher-com/notify/info"
	notifyv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/notify/v1"
	"kubevirt.io/kubevirt/pkg/log"
	grpcutil "kubevirt.io/kubevirt/pkg/util/net/grpc"
)

func NewNotifyClients(virtShareDir string) (notifyinfo.NotifyInfoClient, notifyv1.NotifyClient, *grpc.ClientConn, error) {
	socketPath := filepath.Join(virtShareDir, "domain-notify.sock")
	conn, err := grpcutil.DialSocket(socketPath)
	if err != nil {
		log.Log.Reason(err).Infof("Failed to dial notify socket: %s", socketPath)
		return nil, nil, nil, err
	}
	infoClient := notifyinfo.NewNotifyInfoClient(conn)
	notifyClient := notifyv1.NewNotifyClient(conn)
	return infoClient, notifyClient, conn, nil
}

func NewCmdClients(socketPath string) (cmdinfo.CmdInfoClient, cmdv1.CmdClient, *grpc.ClientConn, error) {
	conn, err := grpcutil.DialSocket(socketPath)
	if err != nil {
		log.Log.Reason(err).Infof("Failed to dial cmd socket: %s", socketPath)
		return nil, nil, nil, err
	}
	infoClient := cmdinfo.NewCmdInfoClient(conn)
	cmdClient := cmdv1.NewCmdClient(conn)
	return infoClient, cmdClient, conn, nil
}

func ContainsVersion(serverVersions []string, clientVersions []string) bool {
	for _, serverVersion := range serverVersions {
		for _, clientVersion := range clientVersions {
			if serverVersion == clientVersion {
				return true
			}
		}
	}
	return false
}
