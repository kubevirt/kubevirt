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
	"encoding/json"
	"net"
	"net/rpc"
	"os"
	"path/filepath"

	k8sv1 "k8s.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cmd-server/client"
)

type Launcher struct {
	domainManager virtwrap.DomainManager
}

func getK8SecretsfromClientArgs(args *cmdclient.Args) (map[string]*k8sv1.Secret, error) {
	var secrets map[string]*k8sv1.Secret
	err := json.Unmarshal([]byte(args.K8SecretMapJSON), &secrets)
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to unmarshal k8 secrents json object")
		return nil, err
	}
	return secrets, nil
}

func getVmfromClientArgs(args *cmdclient.Args) (*v1.VirtualMachine, error) {
	vm := &v1.VirtualMachine{}
	err := json.Unmarshal([]byte(args.VMJSON), vm)
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to unmarshal vm json object")
		return nil, err
	}
	return vm, nil
}

func (s *Launcher) SyncSecret(args *cmdclient.Args, reply *cmdclient.Reply) error {
	reply.Success = true

	vm, err := getVmfromClientArgs(args)
	if err != nil {
		reply.Success = false
		reply.Message = err.Error()
		return nil
	}

	err = s.domainManager.SyncVMSecret(vm,
		args.SecretUsageType,
		args.SecretUsageID,
		args.SecretValue)

	if err != nil {
		log.Log.Object(vm).Reason(err).Errorf("Failed to sync vm secrets")
		reply.Success = false
		reply.Message = err.Error()
		return nil
	}

	log.Log.Object(vm).Info("Synced vm secrets")
	return nil
}

func (s *Launcher) Start(args *cmdclient.Args, reply *cmdclient.Reply) error {
	reply.Success = true

	vm, err := getVmfromClientArgs(args)
	if err != nil {
		reply.Success = false
		reply.Message = err.Error()
		return nil
	}

	secrets, err := getK8SecretsfromClientArgs(args)
	if err != nil {
		reply.Success = false
		reply.Message = err.Error()
		return nil
	}

	_, err = s.domainManager.SyncVM(vm, secrets)
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

func (s *Launcher) ListDomains(args *cmdclient.Args, reply *cmdclient.Reply) error {

	reply.Success = true

	list, err := s.domainManager.ListAllDomains()
	if err != nil {
		reply.Success = false
		reply.Message = err.Error()
		return nil
	}

	domainListJSON, err := json.Marshal(list)
	if err != nil {
		reply.Success = false
		reply.Message = err.Error()
		return nil
	}
	reply.DomainListJSON = string(domainListJSON)

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
