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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package eventsserver

import (
	"os"
	"path/filepath"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"kubevirt.io/kubevirt/pkg/handler-launcher-com/common"

	"k8s.io/apimachinery/pkg/watch"

	"kubevirt.io/client-go/log"
	notifyv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/notify/v1"
	grpcutil "kubevirt.io/kubevirt/pkg/util/net/grpc"
)

type Notify struct {
	EventChan chan watch.Event
	recorder  common.KubernetesEventRecorderInterface
}

func (n *Notify) HandleDomainEvent(ctx context.Context, request *notifyv1.DomainEventRequest) (*notifyv1.Response, error) {
	response := &notifyv1.Response{
		Success: true,
	}
	err := common.EnqueueHandlerDomainEvent(n.EventChan, request)
	if err != nil {
		response.Success = false
		response.Message = err.Error()
	}
	return response, nil
}

func (n *Notify) HandleK8SEvent(ctx context.Context, request *notifyv1.K8SEventRequest) (*notifyv1.Response, error) {
	response := &notifyv1.Response{
		Success: true,
	}
	if err := n.recorder.Record(request); err != nil {
		response.Message = err.Error()
		response.Success = false
	}
	return response, nil
}

func RunServer(virtShareDir string, stopChan chan struct{}, c chan watch.Event, recorder common.KubernetesEventRecorderInterface) error {

	grpcServer := grpc.NewServer([]grpc.ServerOption{}...)
	notifyServer := &Notify{
		EventChan: c,
		recorder:  recorder,
	}
	registerInfoServer(grpcServer)

	// register more versions as soon as needed
	// and add them to info.go
	notifyv1.RegisterNotifyServer(grpcServer, notifyServer)

	sockFile := filepath.Join(virtShareDir, "domain-notify.sock")
	sock, err := grpcutil.CreateSocket(sockFile)
	if err != nil {
		return err
	}

	defer func() {
		sock.Close()
		os.Remove(sockFile)
	}()

	done := make(chan struct{})
	go func() {
		defer close(done)
		grpcServer.Serve(sock)
	}()

	// wait for either the server to exit or stopChan to signal
	select {
	case <-done:
		log.Log.Info("notify server done")
	case <-stopChan:
		grpcServer.Stop()
		log.Log.Info("notify server stopped")
	}

	return nil
}
