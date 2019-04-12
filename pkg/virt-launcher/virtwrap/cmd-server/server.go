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
	"fmt"
	"os"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"k8s.io/apimachinery/pkg/util/json"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	cmdv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"
	"kubevirt.io/kubevirt/pkg/log"
	grpcutil "kubevirt.io/kubevirt/pkg/util/net/grpc"
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

func getVMIFromRequest(request *cmdv1.VMI) (*v1.VirtualMachineInstance, *cmdv1.Response) {

	response := &cmdv1.Response{
		Success: true,
	}

	var vmi v1.VirtualMachineInstance
	if err := json.Unmarshal(request.VmiJson, &vmi); err != nil {
		response.Success = false
		response.Message = "No valid vmi object present in command server request"
	}

	return &vmi, response
}

func getMigrationOptionsFromRequest(request *cmdv1.MigrationRequest) (*cmdclient.MigrationOptions, error) {

	if request.Options == nil {
		return nil, fmt.Errorf("migration options object not present in command server request")
	}

	var options *cmdclient.MigrationOptions
	if err := json.Unmarshal(request.Options, &options); err != nil {
		return nil, fmt.Errorf("no valid migration options object present in command server request: %v", err)
	}

	return options, nil
}

func getErrorMessage(err error) string {
	if virErr := launcherErrors.FormatLibvirtError(err); virErr != "" {
		return virErr
	}
	return err.Error()
}

func (l *Launcher) MigrateVirtualMachine(ctx context.Context, request *cmdv1.MigrationRequest) (*cmdv1.Response, error) {

	vmi, response := getVMIFromRequest(request.Vmi)
	if !response.Success {
		return response, nil
	}

	options, err := getMigrationOptionsFromRequest(request)
	if err != nil {
		response.Success = false
		response.Message = err.Error()
		return response, nil
	}

	if err := l.domainManager.MigrateVMI(vmi, options); err != nil {
		log.Log.Object(vmi).Reason(err).Errorf("Failed to migrate vmi")
		response.Success = false
		response.Message = getErrorMessage(err)
		return response, nil
	}

	log.Log.Object(vmi).Info("Signaled vmi migration")
	return response, nil
}

func (l *Launcher) CancelVirtualMachineMigration(ctx context.Context, request *cmdv1.VMIRequest) (*cmdv1.Response, error) {

	vmi, response := getVMIFromRequest(request.Vmi)
	if !response.Success {
		return response, nil
	}

	if err := l.domainManager.CancelVMIMigration(vmi); err != nil {
		log.Log.Object(vmi).Reason(err).Errorf("Failed to abort live migration")
		response.Success = false
		response.Message = getErrorMessage(err)
		return response, nil
	}

	log.Log.Object(vmi).Info("Live migration as been aborted")
	return response, nil

}

func (l *Launcher) SyncMigrationTarget(ctx context.Context, request *cmdv1.VMIRequest) (*cmdv1.Response, error) {

	vmi, response := getVMIFromRequest(request.Vmi)
	if !response.Success {
		return response, nil
	}

	if err := l.domainManager.PrepareMigrationTarget(vmi, l.useEmulation); err != nil {
		log.Log.Object(vmi).Reason(err).Errorf("Failed to prepare migration target pod")
		response.Success = false
		response.Message = getErrorMessage(err)
		return response, nil
	}

	log.Log.Object(vmi).Info("Prepared migration target pod")
	return response, nil

}

func (l *Launcher) SyncVirtualMachine(ctx context.Context, request *cmdv1.VMIRequest) (*cmdv1.Response, error) {

	vmi, response := getVMIFromRequest(request.Vmi)
	if !response.Success {
		return response, nil
	}

	if _, err := l.domainManager.SyncVMI(vmi, l.useEmulation); err != nil {
		log.Log.Object(vmi).Reason(err).Errorf("Failed to sync vmi")
		response.Success = false
		response.Message = getErrorMessage(err)
		return response, nil
	}

	log.Log.Object(vmi).Info("Synced vmi")
	return response, nil
}

func (l *Launcher) KillVirtualMachine(ctx context.Context, request *cmdv1.VMIRequest) (*cmdv1.Response, error) {

	vmi, response := getVMIFromRequest(request.Vmi)
	if !response.Success {
		return response, nil
	}

	if err := l.domainManager.KillVMI(vmi); err != nil {
		log.Log.Object(vmi).Reason(err).Errorf("Failed to kill vmi")
		response.Success = false
		response.Message = getErrorMessage(err)
		return response, nil
	}

	log.Log.Object(vmi).Info("Signaled vmi kill")
	return response, nil
}

func (l *Launcher) ShutdownVirtualMachine(ctx context.Context, request *cmdv1.VMIRequest) (*cmdv1.Response, error) {

	vmi, response := getVMIFromRequest(request.Vmi)
	if !response.Success {
		return response, nil
	}

	if err := l.domainManager.SignalShutdownVMI(vmi); err != nil {
		log.Log.Object(vmi).Reason(err).Errorf("Failed to signal shutdown for vmi")
		response.Success = false
		response.Message = getErrorMessage(err)
		return response, nil
	}

	log.Log.Object(vmi).Info("Signaled vmi shutdown")
	return response, nil
}

func (l *Launcher) DeleteVirtualMachine(ctx context.Context, request *cmdv1.VMIRequest) (*cmdv1.Response, error) {

	vmi, response := getVMIFromRequest(request.Vmi)
	if !response.Success {
		return response, nil
	}

	if err := l.domainManager.DeleteVMI(vmi); err != nil {
		log.Log.Object(vmi).Reason(err).Errorf("Failed to signal deletion for vmi")
		response.Success = false
		response.Message = getErrorMessage(err)
		return response, nil
	}

	log.Log.Object(vmi).Info("Signaled vmi deletion")
	return response, nil
}

func (l *Launcher) GetDomain(ctx context.Context, request *cmdv1.EmptyRequest) (*cmdv1.DomainResponse, error) {

	response := &cmdv1.DomainResponse{
		Response: &cmdv1.Response{
			Success: true,
		},
	}

	list, err := l.domainManager.ListAllDomains()
	if err != nil {
		response.Response.Success = false
		response.Response.Message = getErrorMessage(err)
		return response, nil
	}

	if len(list) >= 0 {
		if domain, err := json.Marshal(list[0]); err != nil {
			log.Log.Reason(err).Errorf("Failed to marshal domain")
			response.Response.Success = false
			response.Response.Message = getErrorMessage(err)
			return response, nil
		} else {
			response.Domain = string(domain)
		}
	}

	return response, nil
}

func (l *Launcher) GetDomainStats(ctx context.Context, request *cmdv1.EmptyRequest) (*cmdv1.DomainStatsResponse, error) {

	response := &cmdv1.DomainStatsResponse{
		Response: &cmdv1.Response{
			Success: true,
		},
	}

	list, err := l.domainManager.GetDomainStats()
	if err != nil {
		response.Response.Success = false
		response.Response.Message = getErrorMessage(err)
		return response, nil
	}

	if len(list) >= 0 {
		if domainStats, err := json.Marshal(list[0]); err != nil {
			log.Log.Reason(err).Errorf("Failed to marshal domain stats")
			response.Response.Success = false
			response.Response.Message = getErrorMessage(err)
			return response, nil
		} else {
			response.DomainStats = string(domainStats)
		}
	}

	return response, nil
}

func RunServer(socketPath string,
	domainManager virtwrap.DomainManager,
	stopChan chan struct{},
	options *ServerOptions) (chan struct{}, error) {

	useEmulation := false
	if options != nil {
		useEmulation = options.useEmulation
	}

	grpcServer := grpc.NewServer([]grpc.ServerOption{}...)
	server := &Launcher{
		domainManager: domainManager,
		useEmulation:  useEmulation,
	}
	registerInfoServer(grpcServer)

	// register more versions as soon as needed
	// and add them to info.go
	cmdv1.RegisterCmdServer(grpcServer, server)

	sock, err := grpcutil.CreateSocket(socketPath)
	if err != nil {
		return nil, err
	}

	done := make(chan struct{})

	go func() {
		select {
		case <-stopChan:
			log.Log.Info("stopping cmd server")
			grpcServer.Stop()
			sock.Close()
			os.Remove(socketPath)
			close(done)
		}
	}()

	go func() {
		grpcServer.Serve(sock)
	}()

	return done, nil
}

func (l *Launcher) Ping(ctx context.Context, request *cmdv1.EmptyRequest) (*cmdv1.Response, error) {
	response := &cmdv1.Response{
		Success: true,
	}
	return response, nil
}
