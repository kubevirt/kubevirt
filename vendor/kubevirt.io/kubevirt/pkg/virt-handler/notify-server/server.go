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
	"encoding/json"
	"fmt"
	"net"
	"net/rpc"
	"os"
	"path/filepath"

	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"

	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

type Notify struct {
	EventChan chan watch.Event
	recorder  record.EventRecorder
	vmiStore  cache.Store
}

type Reply struct {
	Success bool
	Message string
}

type DomainEventArgs struct {
	DomainJSON string
	StatusJSON string
	EventType  string
}

func (s *Notify) DomainEvent(args *DomainEventArgs, reply *Reply) error {
	reply.Success = true

	domain := &api.Domain{}
	status := &metav1.Status{}
	if args.DomainJSON != "" {
		err := json.Unmarshal([]byte(args.DomainJSON), domain)
		if err != nil {
			log.Log.Errorf("Failed to unmarshal domain json object")
			reply.Success = false
			reply.Message = err.Error()
			return nil
		}
	}
	if args.StatusJSON != "" {
		err := json.Unmarshal([]byte(args.StatusJSON), status)
		if err != nil {
			log.Log.Errorf("Failed to unmarshal status json object")
			reply.Success = false
			reply.Message = err.Error()
			return nil
		}
	}

	log.Log.Infof("Received Domain Event of type %s", args.EventType)
	switch args.EventType {
	case string(watch.Added):
		s.EventChan <- watch.Event{Type: watch.Added, Object: domain}
	case string(watch.Modified):
		s.EventChan <- watch.Event{Type: watch.Modified, Object: domain}
	case string(watch.Deleted):
		s.EventChan <- watch.Event{Type: watch.Deleted, Object: domain}
	case string(watch.Error):
		s.EventChan <- watch.Event{Type: watch.Error, Object: status}
	}
	return nil
}

func (s *Notify) K8sEvent(event k8sv1.Event, reply *Reply) error {
	reply.Success = true

	// get vmi and record event
	involvedObj := event.InvolvedObject

	if obj, exists, err := s.vmiStore.GetByKey(involvedObj.Namespace + "/" + involvedObj.Name); err != nil {
		reply.Success = false
		reply.Message = fmt.Sprintf("Error getting VMI: %v", err)
	} else if !exists || obj.(*v1.VirtualMachineInstance).UID != involvedObj.UID {
		reply.Success = false
		reply.Message = "VMI not found"
	} else {
		vmi := obj.(*v1.VirtualMachineInstance)
		s.recorder.Event(vmi, event.Type, event.Reason, event.Message)
	}
	return nil
}

func createSocket(socketPath string) (net.Listener, error) {
	os.RemoveAll(socketPath)

	err := os.MkdirAll(filepath.Dir(socketPath), 0755)
	if err != nil {
		log.Log.Reason(err).Error("unable to create directory for unix socket")
		return nil, err
	}

	socket, err := net.Listen("unix", socketPath)

	if err != nil {
		log.Log.Reason(err).Error("failed to create unix sock for domain event service")
		return nil, err
	}
	return socket, nil
}

func RunServer(virtShareDir string, stopChan chan struct{}, c chan watch.Event, recorder record.EventRecorder, vmiStore cache.Store) error {
	sockFile := filepath.Join(virtShareDir, "domain-notify.sock")

	rpcServer := rpc.NewServer()
	server := &Notify{
		EventChan: c,
		recorder:  recorder,
		vmiStore:  vmiStore,
	}
	rpcServer.Register(server)
	sock, err := createSocket(sockFile)
	if err != nil {
		return err
	}

	defer func() {
		sock.Close()
		os.Remove(sockFile)
	}()

	done := make(chan error)
	go func() {
		defer close(done)
		rpcServer.Accept(sock)
	}()

	// wait for either the server to exit or stopChan to signal
	select {
	case <-done:
	case <-stopChan:
	}

	return err
}
