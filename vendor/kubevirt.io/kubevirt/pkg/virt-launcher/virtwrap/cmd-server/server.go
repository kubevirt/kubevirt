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
 * Copyright 2017, 2018 Red Hat, Inc.
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

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/log"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap"
	launcherErrors "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/errors"
)

type ServerOptions struct {
	useEmulation bool
}

func NewServerOptions(useEmulation bool) *ServerOptions {
	return &ServerOptions{useEmulation: useEmulation}
}

type Launcher struct {
	domainManager virtwrap.DomainManager
	useEmulation  bool
}

func getVmfromClientArgs(args *cmdclient.Args) (*v1.VirtualMachineInstance, error) {
	if args.VMI == nil {
		return nil, goerror.New(fmt.Sprintf("vmi object not present in command server args"))
	}
	return args.VMI, nil
}

func getErrorMessage(err error) string {
	if virErr := launcherErrors.FormatLibvirtError(err); virErr != "" {
		return virErr
	}
	return err.Error()
}

func (s *Launcher) Migrate(args *cmdclient.Args, reply *cmdclient.Reply) error {

	reply.Success = true

	vmi, err := getVmfromClientArgs(args)
	if err != nil {
		reply.Success = false
		reply.Message = err.Error()
		return nil
	}

	err = s.domainManager.MigrateVMI(vmi)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Errorf("Failed to migrate vmi")
		reply.Success = false
		reply.Message = getErrorMessage(err)
		return nil
	}

	log.Log.Object(vmi).Info("Signaled vmi migration")
	return nil
}

func (s *Launcher) SyncMigrationTarget(args *cmdclient.Args, reply *cmdclient.Reply) error {

	reply.Success = true

	vmi, err := getVmfromClientArgs(args)
	if err != nil {
		reply.Success = false
		reply.Message = err.Error()
		return nil
	}

	err = s.domainManager.PrepareMigrationTarget(vmi, s.useEmulation)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Errorf("Failed to prepare migration target pod")
		reply.Success = false
		reply.Message = getErrorMessage(err)
		return nil
	}

	log.Log.Object(vmi).Info("Prepared migration target pod")
	return nil

}

func (s *Launcher) Sync(args *cmdclient.Args, reply *cmdclient.Reply) error {
	reply.Success = true

	vmi, err := getVmfromClientArgs(args)
	if err != nil {
		reply.Success = false
		reply.Message = err.Error()
		return nil
	}

	_, err = s.domainManager.SyncVMI(vmi, s.useEmulation)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Errorf("Failed to sync vmi")
		reply.Success = false
		reply.Message = getErrorMessage(err)
		return nil
	}

	log.Log.Object(vmi).Info("Synced vmi")
	return nil
}

func (s *Launcher) Kill(args *cmdclient.Args, reply *cmdclient.Reply) error {
	reply.Success = true

	vmi, err := getVmfromClientArgs(args)
	if err != nil {
		reply.Success = false
		reply.Message = err.Error()
		return nil
	}

	err = s.domainManager.KillVMI(vmi)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Errorf("Failed to kill vmi")
		reply.Success = false
		reply.Message = getErrorMessage(err)
		return nil
	}

	log.Log.Object(vmi).Info("Signaled vmi kill")
	return nil
}

func (s *Launcher) Shutdown(args *cmdclient.Args, reply *cmdclient.Reply) error {
	reply.Success = true

	vmi, err := getVmfromClientArgs(args)
	if err != nil {
		reply.Success = false
		reply.Message = err.Error()
		return nil
	}

	err = s.domainManager.SignalShutdownVMI(vmi)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Errorf("Failed to signal shutdown for vmi")
		reply.Success = false
		reply.Message = getErrorMessage(err)
		return nil
	}

	log.Log.Object(vmi).Info("Signaled vmi shutdown")
	return nil
}

func (s *Launcher) Delete(args *cmdclient.Args, reply *cmdclient.Reply) error {
	reply.Success = true

	vmi, err := getVmfromClientArgs(args)
	if err != nil {
		reply.Success = false
		reply.Message = err.Error()
		return nil
	}

	err = s.domainManager.DeleteVMI(vmi)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Errorf("Failed to signal deletion for vmi")
		reply.Success = false
		reply.Message = getErrorMessage(err)
		return nil
	}

	log.Log.Object(vmi).Info("Signaled vmi deletion")
	return nil
}

func (s *Launcher) GetDomain(args *cmdclient.Args, reply *cmdclient.Reply) error {

	reply.Success = true

	list, err := s.domainManager.ListAllDomains()
	if err != nil {
		reply.Success = false
		reply.Message = getErrorMessage(err)
		return nil
	}

	if len(list) == 0 {
		reply.Domain = nil
	} else {
		reply.Domain = list[0]
	}

	return nil
}

func (s *Launcher) GetDomainStats(args *cmdclient.Args, reply *cmdclient.Reply) error {

	reply.Success = true

	list, err := s.domainManager.GetDomainStats()
	if err != nil {
		reply.Success = false
		reply.Message = getErrorMessage(err)
		return nil
	}

	if len(list) == 0 {
		reply.DomainStats = nil
	} else {
		reply.DomainStats = list[0]
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
	stopChan chan struct{},
	options *ServerOptions) (chan struct{}, error) {

	useEmulation := false
	if options != nil {
		useEmulation = options.useEmulation
	}
	rpcServer := rpc.NewServer()
	server := &Launcher{
		domainManager: domainManager,
		useEmulation:  useEmulation,
	}
	rpcServer.Register(server)
	sock, err := createSocket(socketPath)
	if err != nil {
		return nil, err
	}

	done := make(chan struct{})

	go func() {
		select {
		case <-stopChan:
			sock.Close()
			os.Remove(socketPath)
			log.Log.Info("closing cmd server socket")
			close(done)
		}
	}()

	go func() {
		rpcServer.Accept(sock)
	}()

	return done, nil
}

func (s *Launcher) Ping(args *cmdclient.Args, reply *cmdclient.Reply) error {
	reply.Success = true
	return nil
}
