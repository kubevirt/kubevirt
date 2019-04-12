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
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	notifyv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/notify/v1"
	"kubevirt.io/kubevirt/pkg/log"
	grpcutil "kubevirt.io/kubevirt/pkg/util/net/grpc"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

type Notify struct {
	EventChan chan watch.Event
	recorder  record.EventRecorder
	vmiStore  cache.Store
}

func (n *Notify) HandleDomainEvent(ctx context.Context, request *notifyv1.DomainEventRequest) (*notifyv1.Response, error) {
	response := &notifyv1.Response{
		Success: true,
	}

	domain := &api.Domain{}
	status := &metav1.Status{}

	if len(request.DomainJSON) > 0 {
		err := json.Unmarshal(request.DomainJSON, domain)
		if err != nil {
			log.Log.Errorf("Failed to unmarshal domain json object")
			response.Success = false
			response.Message = err.Error()
			return response, nil
		}
	}
	if len(request.StatusJSON) > 0 {
		err := json.Unmarshal(request.StatusJSON, status)
		if err != nil {
			log.Log.Errorf("Failed to unmarshal status json object")
			response.Success = false
			response.Message = err.Error()
			return response, nil
		}
	}

	log.Log.Infof("Received Domain Event of type %s", request.EventType)
	switch request.EventType {
	case string(watch.Added):
		n.EventChan <- watch.Event{Type: watch.Added, Object: domain}
	case string(watch.Modified):
		n.EventChan <- watch.Event{Type: watch.Modified, Object: domain}
	case string(watch.Deleted):
		n.EventChan <- watch.Event{Type: watch.Deleted, Object: domain}
	case string(watch.Error):
		n.EventChan <- watch.Event{Type: watch.Error, Object: status}
	}
	return response, nil
}

func (n *Notify) HandleK8SEvent(ctx context.Context, request *notifyv1.K8SEventRequest) (*notifyv1.Response, error) {
	response := &notifyv1.Response{
		Success: true,
	}

	// unmarshal k8s event
	var event k8sv1.Event
	err := json.Unmarshal(request.EventJSON, &event)
	if err != nil {
		response.Success = false
		response.Message = fmt.Sprintf("Error unmarshalling k8s event: %v", err)
		return response, nil
	}

	// get vmi and record event
	involvedObj := event.InvolvedObject

	if obj, exists, err := n.vmiStore.GetByKey(involvedObj.Namespace + "/" + involvedObj.Name); err != nil {
		response.Success = false
		response.Message = fmt.Sprintf("Error getting VMI: %v", err)
	} else if !exists || obj.(*v1.VirtualMachineInstance).UID != involvedObj.UID {
		response.Success = false
		response.Message = "VMI not found"
	} else {
		vmi := obj.(*v1.VirtualMachineInstance)
		n.recorder.Event(vmi, event.Type, event.Reason, event.Message)
	}
	return response, nil
}

func RunServer(virtShareDir string, stopChan chan struct{}, c chan watch.Event, recorder record.EventRecorder, vmiStore cache.Store) error {

	grpcServer := grpc.NewServer([]grpc.ServerOption{}...)
	notifyServer := &Notify{
		EventChan: c,
		recorder:  recorder,
		vmiStore:  vmiStore,
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
		log.Log.Info("stopping notify server")
		grpcServer.Stop()
		sock.Close()
		os.Remove(sockFile)
	}

	return nil
}
