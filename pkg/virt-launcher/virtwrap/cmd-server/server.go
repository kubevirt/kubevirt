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
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"google.golang.org/grpc"

	"k8s.io/apimachinery/pkg/util/json"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	cmdv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"
	grpcutil "kubevirt.io/kubevirt/pkg/util/net/grpc"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/agent"
	launcherErrors "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/errors"
)

const (
	receivedEarlyExitSignalEnvVar = "VIRT_LAUNCHER_TARGET_POD_EXIT_SIGNAL"
)

type ServerOptions struct {
	allowEmulation bool
}

func NewServerOptions(allowEmulation bool) *ServerOptions {
	return &ServerOptions{allowEmulation: allowEmulation}
}

type Launcher struct {
	domainManager  virtwrap.DomainManager
	allowEmulation bool
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

func (l *Launcher) MigrateVirtualMachine(_ context.Context, request *cmdv1.MigrationRequest) (*cmdv1.Response, error) {

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

func (l *Launcher) CancelVirtualMachineMigration(_ context.Context, request *cmdv1.VMIRequest) (*cmdv1.Response, error) {

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

func (l *Launcher) SignalTargetPodCleanup(_ context.Context, request *cmdv1.VMIRequest) (*cmdv1.Response, error) {

	vmi, response := getVMIFromRequest(request.Vmi)
	if !response.Success {
		return response, nil
	}

	myPodName := os.Getenv("POD_NAME")

	if myPodName != "" && vmi.Status.MigrationState != nil && vmi.Status.MigrationState.TargetPod == myPodName {
		os.Setenv(receivedEarlyExitSignalEnvVar, "")
		log.Log.Object(vmi).Infof("Signaled target pod %s to cleanup", myPodName)
	}

	return response, nil
}

func (l *Launcher) SyncMigrationTarget(_ context.Context, request *cmdv1.VMIRequest) (*cmdv1.Response, error) {

	vmi, response := getVMIFromRequest(request.Vmi)
	if !response.Success {
		return response, nil
	}

	if err := l.domainManager.PrepareMigrationTarget(vmi, l.allowEmulation, request.Options); err != nil {
		log.Log.Object(vmi).Reason(err).Errorf("Failed to prepare migration target pod")
		response.Success = false
		response.Message = getErrorMessage(err)
		return response, nil
	}

	log.Log.Object(vmi).Info("Prepared migration target pod")
	return response, nil

}

func (l *Launcher) SyncVirtualMachineCPUs(_ context.Context, request *cmdv1.VMIRequest) (*cmdv1.Response, error) {
	vmi, response := getVMIFromRequest(request.Vmi)
	if !response.Success {
		return response, nil
	}

	if err := l.domainManager.UpdateVCPUs(vmi, request.Options); err != nil {
		log.Log.Object(vmi).Reason(err).Errorf("Failed update VMI vCPUs")
		response.Success = false
		response.Message = getErrorMessage(err)
		return response, nil
	}

	log.Log.Object(vmi).Info("VMI vCPUs has been updated")
	return response, nil
}

func (l *Launcher) SyncVirtualMachine(_ context.Context, request *cmdv1.VMIRequest) (*cmdv1.Response, error) {

	vmi, response := getVMIFromRequest(request.Vmi)
	if !response.Success {
		return response, nil
	}

	if _, err := l.domainManager.SyncVMI(vmi, l.allowEmulation, request.Options); err != nil {
		log.Log.Object(vmi).Reason(err).Errorf("Failed to sync vmi")
		response.Success = false
		response.Message = getErrorMessage(err)
		return response, nil
	}

	log.Log.Object(vmi).Info("Synced vmi")
	return response, nil
}

func (l *Launcher) PauseVirtualMachine(_ context.Context, request *cmdv1.VMIRequest) (*cmdv1.Response, error) {
	vmi, response := getVMIFromRequest(request.Vmi)
	if !response.Success {
		return response, nil
	}

	if err := l.domainManager.PauseVMI(vmi); err != nil {
		log.Log.Object(vmi).Reason(err).Errorf("Failed to pause vmi")
		response.Success = false
		response.Message = getErrorMessage(err)
		return response, nil
	}

	log.Log.Object(vmi).Info("Paused vmi")
	return response, nil
}

func (l *Launcher) UnpauseVirtualMachine(_ context.Context, request *cmdv1.VMIRequest) (*cmdv1.Response, error) {
	vmi, response := getVMIFromRequest(request.Vmi)
	if !response.Success {
		return response, nil
	}

	if err := l.domainManager.UnpauseVMI(vmi); err != nil {
		log.Log.Object(vmi).Reason(err).Errorf("Failed to unpause vmi")
		response.Success = false
		response.Message = getErrorMessage(err)
		return response, nil
	}

	log.Log.Object(vmi).Info("Unpaused vmi")
	return response, nil
}

func (l *Launcher) VirtualMachineMemoryDump(_ context.Context, request *cmdv1.MemoryDumpRequest) (*cmdv1.Response, error) {
	vmi, response := getVMIFromRequest(request.Vmi)
	if !response.Success {
		return response, nil
	}

	if err := l.domainManager.MemoryDump(vmi, request.DumpPath); err != nil {
		log.Log.Object(vmi).Reason(err).Errorf("Failed to Dump vmi memory")
		response.Success = false
		response.Message = getErrorMessage(err)
		return response, nil
	}

	return response, nil
}

func (l *Launcher) FreezeVirtualMachine(_ context.Context, request *cmdv1.FreezeRequest) (*cmdv1.Response, error) {
	vmi, response := getVMIFromRequest(request.Vmi)
	if !response.Success {
		return response, nil
	}

	if err := l.domainManager.FreezeVMI(vmi, request.UnfreezeTimeoutSeconds); err != nil {
		log.Log.Object(vmi).Reason(err).Errorf("Failed to freeze vmi")
		response.Success = false
		response.Message = getErrorMessage(err)
		return response, nil
	}

	log.Log.Object(vmi).Info("Freezed vmi")
	return response, nil
}

func (l *Launcher) UnfreezeVirtualMachine(_ context.Context, request *cmdv1.VMIRequest) (*cmdv1.Response, error) {
	vmi, response := getVMIFromRequest(request.Vmi)
	if !response.Success {
		return response, nil
	}

	if err := l.domainManager.UnfreezeVMI(vmi); err != nil {
		log.Log.Object(vmi).Reason(err).Errorf("Failed to unfreeze vmi")
		response.Success = false
		response.Message = getErrorMessage(err)
		return response, nil
	}

	log.Log.Object(vmi).Info("Unfreezed vmi")
	return response, nil
}

func (l *Launcher) SoftRebootVirtualMachine(_ context.Context, request *cmdv1.VMIRequest) (*cmdv1.Response, error) {
	vmi, response := getVMIFromRequest(request.Vmi)
	if !response.Success {
		return response, nil
	}

	if err := l.domainManager.SoftRebootVMI(vmi); err != nil {
		log.Log.Object(vmi).Reason(err).Errorf("Failed to soft reboot vmi")
		response.Success = false
		response.Message = getErrorMessage(err)
		return response, nil
	}

	log.Log.Object(vmi).Info("Soft rebooted vmi")
	return response, nil
}

func (l *Launcher) KillVirtualMachine(_ context.Context, request *cmdv1.VMIRequest) (*cmdv1.Response, error) {

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

func (l *Launcher) ShutdownVirtualMachine(_ context.Context, request *cmdv1.VMIRequest) (*cmdv1.Response, error) {

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

func (l *Launcher) DeleteVirtualMachine(_ context.Context, request *cmdv1.VMIRequest) (*cmdv1.Response, error) {

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

func (l *Launcher) FinalizeVirtualMachineMigration(_ context.Context, request *cmdv1.VMIRequest) (*cmdv1.Response, error) {
	vmi, response := getVMIFromRequest(request.Vmi)
	if !response.Success {
		return response, nil
	}

	if err := l.domainManager.FinalizeVirtualMachineMigration(vmi); err != nil {
		log.Log.Object(vmi).Reason(err).Errorf("failed to finalize migration")
		response.Success = false
		response.Message = getErrorMessage(err)
		return response, nil
	}

	log.Log.Object(vmi).Info("migration finalized successfully")
	return response, nil
}

func (l *Launcher) HotplugHostDevices(_ context.Context, request *cmdv1.VMIRequest) (*cmdv1.Response, error) {
	vmi, response := getVMIFromRequest(request.Vmi)
	if !response.Success {
		return response, nil
	}

	if err := l.domainManager.HotplugHostDevices(vmi); err != nil {
		log.Log.Object(vmi).Errorf(err.Error())
		response.Success = false
		response.Message = getErrorMessage(err)
		return response, nil
	}

	return response, nil
}

func (l *Launcher) GetDomain(_ context.Context, _ *cmdv1.EmptyRequest) (*cmdv1.DomainResponse, error) {

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

	if len(list) > 0 {
		domainObj := list[0]
		if osInfo := l.domainManager.GetGuestOSInfo(); osInfo != nil {
			domainObj.Status.OSInfo = *osInfo
		}
		if interfaces := l.domainManager.InterfacesStatus(); interfaces != nil {
			domainObj.Status.Interfaces = interfaces
		}
		if domain, err := json.Marshal(domainObj); err != nil {
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

func (l *Launcher) GetQemuVersion(_ context.Context, _ *cmdv1.EmptyRequest) (*cmdv1.QemuVersionResponse, error) {
	response := &cmdv1.QemuVersionResponse{
		Response: &cmdv1.Response{},
	}

	if version, err := l.domainManager.GetQemuVersion(); err != nil {
		response.Response.Message = getErrorMessage(err)
	} else {
		response.Response.Success = true
		response.Version = version
	}

	return response, nil
}

func (l *Launcher) GetDomainStats(_ context.Context, _ *cmdv1.EmptyRequest) (*cmdv1.DomainStatsResponse, error) {

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

	if len(list) > 0 {
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

// GetGuestInfo collect guest info from the domain
func (l *Launcher) GetGuestInfo(_ context.Context, _ *cmdv1.EmptyRequest) (*cmdv1.GuestInfoResponse, error) {
	response := &cmdv1.GuestInfoResponse{
		Response: &cmdv1.Response{
			Success: true,
		},
	}

	guestInfo := l.domainManager.GetGuestInfo()
	if jGuestInfo, err := json.Marshal(guestInfo); err != nil {
		log.Log.Reason(err).Errorf("Failed to marshal agent info")
		response.Response.Success = false
		response.Response.Message = getErrorMessage(err)
		return response, nil
	} else {
		response.GuestInfoResponse = string(jGuestInfo)
	}

	return response, nil
}

// GetUsers returns the list of active users on the guest machine
func (l *Launcher) GetUsers(_ context.Context, _ *cmdv1.EmptyRequest) (*cmdv1.GuestUserListResponse, error) {
	response := &cmdv1.GuestUserListResponse{
		Response: &cmdv1.Response{
			Success: true,
		},
	}

	users := l.domainManager.GetUsers()
	if jUsers, err := json.Marshal(users); err != nil {
		log.Log.Reason(err).Errorf("Failed to marshal guest user list")
		response.Response.Success = false
		response.Response.Message = getErrorMessage(err)
		return response, nil
	} else {
		response.GuestUserListResponse = string(jUsers)
	}

	return response, nil
}

// GetFilesystems returns a full list of active filesystems on the guest machine
func (l *Launcher) GetFilesystems(_ context.Context, _ *cmdv1.EmptyRequest) (*cmdv1.GuestFilesystemsResponse, error) {
	response := &cmdv1.GuestFilesystemsResponse{
		Response: &cmdv1.Response{
			Success: true,
		},
	}

	fs := l.domainManager.GetFilesystems()
	if jFS, err := json.Marshal(fs); err != nil {
		log.Log.Reason(err).Errorf("Failed to marshal guest user list")
		response.Response.Success = false
		response.Response.Message = getErrorMessage(err)
		return response, nil
	} else {
		response.GuestFilesystemsResponse = string(jFS)
	}

	return response, nil
}

// Exec the provided command and return it's success
func (l *Launcher) Exec(ctx context.Context, request *cmdv1.ExecRequest) (*cmdv1.ExecResponse, error) {
	resp := &cmdv1.ExecResponse{
		Response: &cmdv1.Response{
			Success: true,
		},
	}

	stdOut, err := l.domainManager.Exec(request.DomainName, request.Command, request.Args, request.TimeoutSeconds)
	resp.StdOut = stdOut

	exitCode := agent.ExecExitCode{}
	if err != nil && !errors.As(err, &exitCode) {
		resp.Response.Success = false
		resp.Response.Message = err.Error()
		return resp, err
	}
	resp.ExitCode = int32(exitCode.ExitCode)

	return resp, nil
}

func (l *Launcher) GuestPing(ctx context.Context, request *cmdv1.GuestPingRequest) (*cmdv1.GuestPingResponse, error) {
	resp := &cmdv1.GuestPingResponse{
		Response: &cmdv1.Response{
			Success: true,
		},
	}
	err := l.domainManager.GuestPing(request.DomainName)
	if err != nil {
		resp.Response.Success = false
		resp.Response.Message = err.Error()
		return resp, err
	}
	return resp, nil
}

func RunServer(socketPath string,
	domainManager virtwrap.DomainManager,
	stopChan chan struct{},
	options *ServerOptions) (chan struct{}, error) {

	allowEmulation := false
	if options != nil {
		allowEmulation = options.allowEmulation
	}

	grpcServer := grpc.NewServer([]grpc.ServerOption{}...)
	server := &Launcher{
		domainManager:  domainManager,
		allowEmulation: allowEmulation,
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
			stopped := make(chan struct{})
			go func() {
				grpcServer.Stop()
				close(stopped)
			}()

			select {
			case <-stopped:
				log.Log.Info("cmd server stopped")
			case <-time.After(1 * time.Second):
				log.Log.Error("timeout on stopping the cmd server, continuing anyway.")
			}
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

func (l *Launcher) Ping(_ context.Context, _ *cmdv1.EmptyRequest) (*cmdv1.Response, error) {
	response := &cmdv1.Response{
		Success: true,
	}
	return response, nil
}

func (l *Launcher) GetSEVInfo(_ context.Context, _ *cmdv1.EmptyRequest) (*cmdv1.SEVInfoResponse, error) {
	sevInfoResponse := &cmdv1.SEVInfoResponse{
		Response: &cmdv1.Response{
			Success: true,
		},
	}

	sevPlatformInfo, err := l.domainManager.GetSEVInfo()
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to get SEV platform info")
		sevInfoResponse.Response.Success = false
		sevInfoResponse.Response.Message = getErrorMessage(err)
		return sevInfoResponse, nil
	}

	if sevPlatformInfoJson, err := json.Marshal(sevPlatformInfo); err != nil {
		log.Log.Reason(err).Errorf("Failed to marshal SEV platform info")
		sevInfoResponse.Response.Success = false
		sevInfoResponse.Response.Message = getErrorMessage(err)
		return sevInfoResponse, nil
	} else {
		sevInfoResponse.SevInfo = sevPlatformInfoJson
	}

	return sevInfoResponse, nil
}

func (l *Launcher) GetLaunchMeasurement(_ context.Context, request *cmdv1.VMIRequest) (*cmdv1.LaunchMeasurementResponse, error) {
	vmi, response := getVMIFromRequest(request.Vmi)
	launchMeasurementResponse := &cmdv1.LaunchMeasurementResponse{
		Response: response,
	}

	if !launchMeasurementResponse.Response.Success {
		return launchMeasurementResponse, nil
	}

	sevMeasurementInfo, err := l.domainManager.GetLaunchMeasurement(vmi)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Errorf("Failed to get launch measuement")
		launchMeasurementResponse.Response.Success = false
		launchMeasurementResponse.Response.Message = getErrorMessage(err)
		return launchMeasurementResponse, nil
	}

	if sevMeasurementInfoJson, err := json.Marshal(sevMeasurementInfo); err != nil {
		log.Log.Reason(err).Errorf("Failed to marshal launch measuement info")
		launchMeasurementResponse.Response.Success = false
		launchMeasurementResponse.Response.Message = getErrorMessage(err)
		return launchMeasurementResponse, nil
	} else {
		launchMeasurementResponse.LaunchMeasurement = sevMeasurementInfoJson
	}

	return launchMeasurementResponse, nil
}

func (l *Launcher) InjectLaunchSecret(_ context.Context, request *cmdv1.InjectLaunchSecretRequest) (*cmdv1.Response, error) {
	vmi, response := getVMIFromRequest(request.Vmi)
	if !response.Success {
		return response, nil
	}

	var sevSecretOptions v1.SEVSecretOptions
	if err := json.Unmarshal(request.Options, &sevSecretOptions); err != nil {
		response.Success = false
		response.Message = "No valid secret options present in command server request"
		return response, nil
	}

	if err := l.domainManager.InjectLaunchSecret(vmi, &sevSecretOptions); err != nil {
		log.Log.Object(vmi).Reason(err).Errorf("Failed to inject SEV launch secret")
		response.Success = false
		response.Message = getErrorMessage(err)
		return response, nil
	}

	return response, nil
}

func (l *Launcher) SyncVirtualMachineMemory(_ context.Context, request *cmdv1.VMIRequest) (*cmdv1.Response, error) {
	vmi, response := getVMIFromRequest(request.Vmi)
	if !response.Success {
		return response, nil
	}

	if err := l.domainManager.UpdateGuestMemory(vmi); err != nil {
		log.Log.Object(vmi).Reason(err).Errorf("Failed update VMI guest memory")
		response.Success = false
		response.Message = getErrorMessage(err)
		return response, nil
	}

	log.Log.Object(vmi).Info("guest memory has been updated")
	return response, nil
}

func ReceivedEarlyExitSignal() bool {
	_, earlyExit := os.LookupEnv(receivedEarlyExitSignalEnvVar)
	return earlyExit
}
