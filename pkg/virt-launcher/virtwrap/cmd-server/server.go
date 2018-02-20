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

package cmdserver

import (
	goerror "errors"
	"fmt"
	"net"
	"net/rpc"
	"os"
	"path/filepath"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/log"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap"
)

type Launcher struct {
	domainManager virtwrap.DomainManager
}

func getVmfromClientArgs(args *cmdclient.Args) (*v1.VirtualMachine, error) {
	if args.VM == nil {
		return nil, goerror.New(fmt.Sprintf("vm object not present in command server args"))
	}
	return args.VM, nil
}

func (s *Launcher) Sync(args *cmdclient.Args, reply *cmdclient.Reply) error {
	reply.Success = true

	vm, err := getVmfromClientArgs(args)
	if err != nil {
		reply.Success = false
		reply.Message = err.Error()
		return nil
	}

	_, err = s.domainManager.SyncVM(vm)
	if err != nil {
		log.Log.Object(vm).Reason(err).Errorf("Failed to sync vm")
		reply.Success = false
		reply.Message = err.Error()
		return nil
	}

	log.Log.Object(vm).Info("Synced vm")
	return nil
}

func (s *Launcher) Kill(args *cmdclient.Args, reply *cmdclient.Reply) error {
	reply.Success = true

	vm, err := getVmfromClientArgs(args)
	if err != nil {
		reply.Success = false
		reply.Message = err.Error()
		return nil
	}

	err = s.domainManager.KillVM(vm)
	if err != nil {
		log.Log.Object(vm).Reason(err).Errorf("Failed to kill vm")
		reply.Success = false
		reply.Message = err.Error()
		return nil
	}

	log.Log.Object(vm).Info("Signaled vm kill")
	return nil
}

func (s *Launcher) Shutdown(args *cmdclient.Args, reply *cmdclient.Reply) error {
	reply.Success = true

	vm, err := getVmfromClientArgs(args)
	if err != nil {
		reply.Success = false
		reply.Message = err.Error()
		return nil
	}

	err = s.domainManager.SignalShutdownVM(vm)
	if err != nil {
		log.Log.Object(vm).Reason(err).Errorf("Failed to signal shutdown for vm")
		reply.Success = false
		reply.Message = err.Error()
		return nil
	}

	log.Log.Object(vm).Info("Signaled vm shutdown")
	return nil
}

func (s *Launcher) GetDomain(args *cmdclient.Args, reply *cmdclient.Reply) error {

	reply.Success = true

	list, err := s.domainManager.ListAllDomains()
	if err != nil {
		reply.Success = false
		reply.Message = err.Error()
		return nil
	}

	if len(list) == 0 {
		reply.Domain = nil
	} else {
		reply.Domain = list[0]
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
		log.Log.Reason(err).Error("failed to create unix socket for launcher cmd service")
		return nil, err
	}
	return socket, nil
}

func RunServer(socketPath string,
	domainManager virtwrap.DomainManager,
	stopChan chan struct{}) error {

	rpcServer := rpc.NewServer()
	server := &Launcher{
		domainManager: domainManager,
	}
	rpcServer.Register(server)
	sock, err := createSocket(socketPath)
	if err != nil {
		return err
	}

	go func() {
		select {
		case <-stopChan:
			sock.Close()
			os.Remove(socketPath)
			log.Log.Info("closing cmd server socket")
		}
	}()

	go func() {
		rpcServer.Accept(sock)
	}()

	return nil
}

func (s *Launcher) Ping(args *cmdclient.Args, reply *cmdclient.Reply) error {
	reply.Success = true
	return nil
}
