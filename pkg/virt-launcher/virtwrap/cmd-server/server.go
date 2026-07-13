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
 * Copyright The KubeVirt Authors.
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
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"

	backupv1 "kubevirt.io/api/backup/v1alpha1"
	v1 "kubevirt.io/api/core/v1"
	pluginv1alpha1 "kubevirt.io/api/plugin/v1alpha1"
	"kubevirt.io/client-go/log"

	cmdv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"
	grpcutil "kubevirt.io/kubevirt/pkg/util/net/grpc"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
	notifyclient "kubevirt.io/kubevirt/pkg/virt-launcher/notify-client"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/agent"
	launcherErrors "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/errors"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/plugins"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/storage"
)

const (
	receivedEarlyExitSignalEnvVar = "VIRT_LAUNCHER_TARGET_POD_EXIT_SIGNAL"
)

type ServerOptions struct {
	allowEmulation          bool
	vmStatsCollectorEnabled bool
	notifier                *notifyclient.Notifier
	vmiName                 string
	vmiNamespace            string
	vmiUID                  types.UID
}

func NewServerOptions(allowEmulation bool) *ServerOptions {
	return &ServerOptions{allowEmulation: allowEmulation}
}

func (o *ServerOptions) WithNotifier(n *notifyclient.Notifier) *ServerOptions {
	o.notifier = n
	return o
}

func (o *ServerOptions) WithVMStatsCollector(enabled bool) *ServerOptions {
	o.vmStatsCollectorEnabled = enabled
	return o
}

func (o *ServerOptions) WithVMI(vmi *v1.VirtualMachineInstance) *ServerOptions {
	if vmi != nil {
		o.vmiName = vmi.Name
		o.vmiNamespace = vmi.Namespace
		o.vmiUID = vmi.UID
	}
	return o
}

type Launcher struct {
	domainManager virtwrap.DomainManager
	*ServerOptions
}

func NewLauncher(domainManager virtwrap.DomainManager, options *ServerOptions) *Launcher {
	return &Launcher{
		domainManager: domainManager,
		ServerOptions: options,
	}
}

func setPluginsFromOptions(options *cmdv1.VirtualMachineOptions) error {
	if options == nil || len(options.PluginsJson) == 0 {
		plugins.SetPlugins(nil)
		return nil
	}
	var pluginList []pluginv1alpha1.Plugin
	if err := json.Unmarshal(options.PluginsJson, &pluginList); err != nil {
		return fmt.Errorf("failed to deserialize plugins from options: %w", err)
	}
	plugins.SetPlugins(pluginList)
	return nil
}

func getVMIFromRequest(request *cmdv1.VMI) (*v1.VirtualMachineInstance, error) {
	var vmi v1.VirtualMachineInstance
	if err := json.Unmarshal(request.VmiJson, &vmi); err != nil {
		log.Log.Reason(err).Error("Failed to unmarshal VMI from request")
		return nil, grpcstatus.Errorf(codes.InvalidArgument, "no valid vmi object present in request: %v", err)
	}
	return &vmi, nil
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

	vmi, err := getVMIFromRequest(request.Vmi)
	if err != nil {
		return nil, err
	}

	options, err := getMigrationOptionsFromRequest(request)
	if err != nil {
		return nil, grpcstatus.Errorf(codes.InvalidArgument, "%s", err.Error())
	}

	if err := l.domainManager.MigrateVMI(vmi, options); err != nil {
		log.Log.Object(vmi).Reason(err).Errorf("Failed to migrate vmi")
		return nil, grpcstatus.Errorf(codes.Internal, "failed to migrate vmi: %s", getErrorMessage(err))
	}

	log.Log.Object(vmi).Info("Signaled vmi migration")
	return &cmdv1.Response{Success: true}, nil
}

func (l *Launcher) CancelVirtualMachineMigration(_ context.Context, request *cmdv1.VMIRequest) (*cmdv1.Response, error) {

	vmi, err := getVMIFromRequest(request.Vmi)
	if err != nil {
		return nil, err
	}

	if err := l.domainManager.CancelVMIMigration(vmi); err != nil {
		log.Log.Object(vmi).Reason(err).Errorf("Failed to abort live migration")
		return nil, grpcstatus.Errorf(codes.Internal, "failed to abort live migration: %s", getErrorMessage(err))
	}

	log.Log.Object(vmi).Info("Live migration has been aborted")
	return &cmdv1.Response{Success: true}, nil

}

func (l *Launcher) SignalTargetPodCleanup(_ context.Context, request *cmdv1.VMIRequest) (*cmdv1.Response, error) {

	vmi, err := getVMIFromRequest(request.Vmi)
	if err != nil {
		return nil, err
	}

	myPodName := os.Getenv("POD_NAME")

	if myPodName != "" && vmi.Status.MigrationState != nil && vmi.Status.MigrationState.TargetPod == myPodName {
		os.Setenv(receivedEarlyExitSignalEnvVar, "")
		log.Log.Object(vmi).Infof("Signaled target pod %s to cleanup", myPodName)
	}

	return &cmdv1.Response{Success: true}, nil
}

func (l *Launcher) SyncMigrationTarget(_ context.Context, request *cmdv1.VMIRequest) (*cmdv1.Response, error) {

	vmi, err := getVMIFromRequest(request.Vmi)
	if err != nil {
		return nil, err
	}
	if err := setPluginsFromOptions(request.Options); err != nil {
		return nil, grpcstatus.Errorf(codes.InvalidArgument, "%s", err.Error())
	}

	if err := l.domainManager.PrepareMigrationTarget(vmi, l.allowEmulation, request.Options); err != nil {
		log.Log.Object(vmi).Reason(err).Errorf("Failed to prepare migration target pod")
		return nil, grpcstatus.Errorf(codes.Internal, "failed to prepare migration target pod: %s", getErrorMessage(err))
	}

	log.Log.Object(vmi).Info("Prepared migration target pod")
	return &cmdv1.Response{Success: true}, nil

}

func (l *Launcher) SyncVirtualMachineCPUs(_ context.Context, request *cmdv1.VMIRequest) (*cmdv1.Response, error) {
	vmi, err := getVMIFromRequest(request.Vmi)
	if err != nil {
		return nil, err
	}

	if err := l.domainManager.UpdateVCPUs(vmi, request.Options); err != nil {
		log.Log.Object(vmi).Reason(err).Errorf("Failed update VMI vCPUs")
		return nil, grpcstatus.Errorf(codes.Internal, "failed to update vmi vcpus: %s", getErrorMessage(err))
	}

	log.Log.Object(vmi).Info("VMI vCPUs has been updated")
	return &cmdv1.Response{Success: true}, nil
}

func (l *Launcher) SyncVirtualMachine(_ context.Context, request *cmdv1.VMIRequest) (*cmdv1.Response, error) {

	vmi, err := getVMIFromRequest(request.Vmi)
	if err != nil {
		return nil, err
	}
	if err := setPluginsFromOptions(request.Options); err != nil {
		return nil, grpcstatus.Errorf(codes.InvalidArgument, "%s", err.Error())
	}

	if _, err := l.domainManager.SyncVMI(vmi, l.allowEmulation, request.Options); err != nil {
		log.Log.Object(vmi).Reason(err).Errorf("Failed to sync vmi")
		return nil, grpcstatus.Errorf(codes.Internal, "failed to sync vmi: %s", getErrorMessage(err))
	}

	log.Log.Object(vmi).Info("Synced vmi")
	return &cmdv1.Response{Success: true}, nil
}

func (l *Launcher) PauseVirtualMachine(_ context.Context, request *cmdv1.VMIRequest) (*cmdv1.Response, error) {
	vmi, err := getVMIFromRequest(request.Vmi)
	if err != nil {
		return nil, err
	}

	if err := l.domainManager.PauseVMI(vmi); err != nil {
		log.Log.Object(vmi).Reason(err).Errorf("Failed to pause vmi")
		return nil, grpcstatus.Errorf(codes.Internal, "failed to pause vmi: %s", getErrorMessage(err))
	}

	log.Log.Object(vmi).Info("Paused vmi")
	return &cmdv1.Response{Success: true}, nil
}

func (l *Launcher) UnpauseVirtualMachine(_ context.Context, request *cmdv1.VMIRequest) (*cmdv1.Response, error) {
	vmi, err := getVMIFromRequest(request.Vmi)
	if err != nil {
		return nil, err
	}

	if err := l.domainManager.UnpauseVMI(vmi); err != nil {
		log.Log.Object(vmi).Reason(err).Errorf("Failed to unpause vmi")
		return nil, grpcstatus.Errorf(codes.Internal, "failed to unpause vmi: %s", getErrorMessage(err))
	}

	log.Log.Object(vmi).Info("Unpaused vmi")
	return &cmdv1.Response{Success: true}, nil
}

func (l *Launcher) VirtualMachineMemoryDump(_ context.Context, request *cmdv1.MemoryDumpRequest) (*cmdv1.Response, error) {
	vmi, err := getVMIFromRequest(request.Vmi)
	if err != nil {
		return nil, err
	}

	if err := l.domainManager.MemoryDump(vmi, request.DumpPath); err != nil {
		log.Log.Object(vmi).Reason(err).Errorf("Failed to Dump vmi memory")
		return nil, grpcstatus.Errorf(codes.Internal, "failed to dump vmi memory: %s", getErrorMessage(err))
	}

	return &cmdv1.Response{Success: true}, nil
}

func (l *Launcher) FreezeVirtualMachine(_ context.Context, request *cmdv1.FreezeRequest) (*cmdv1.Response, error) {
	vmi, err := getVMIFromRequest(request.Vmi)
	if err != nil {
		return nil, err
	}

	if err := l.domainManager.FreezeVMI(vmi, request.UnfreezeTimeoutSeconds); err != nil {
		log.Log.Object(vmi).Reason(err).Errorf("Failed to freeze vmi")
		return nil, grpcstatus.Errorf(codes.Internal, "failed to freeze vmi: %s", getErrorMessage(err))
	}

	log.Log.Object(vmi).Info("Freezed vmi")
	return &cmdv1.Response{Success: true}, nil
}

func (l *Launcher) UnfreezeVirtualMachine(_ context.Context, request *cmdv1.VMIRequest) (*cmdv1.Response, error) {
	vmi, err := getVMIFromRequest(request.Vmi)
	if err != nil {
		return nil, err
	}

	if err := l.domainManager.UnfreezeVMI(vmi); err != nil {
		log.Log.Object(vmi).Reason(err).Errorf("Failed to unfreeze vmi")
		return nil, grpcstatus.Errorf(codes.Internal, "failed to unfreeze vmi: %s", getErrorMessage(err))
	}

	log.Log.Object(vmi).Info("Unfreezed vmi")
	return &cmdv1.Response{Success: true}, nil
}

func (l *Launcher) ResetVirtualMachine(_ context.Context, request *cmdv1.VMIRequest) (*cmdv1.Response, error) {
	vmi, err := getVMIFromRequest(request.Vmi)
	if err != nil {
		return nil, err
	}

	if err := l.domainManager.ResetVMI(vmi); err != nil {
		log.Log.Object(vmi).Reason(err).Errorf("Failed to reset vmi")
		return nil, grpcstatus.Errorf(codes.Internal, "failed to reset vmi: %s", getErrorMessage(err))
	}

	log.Log.Object(vmi).Info("Reset vmi")
	return &cmdv1.Response{Success: true}, nil
}

func (l *Launcher) SoftRebootVirtualMachine(_ context.Context, request *cmdv1.VMIRequest) (*cmdv1.Response, error) {
	vmi, err := getVMIFromRequest(request.Vmi)
	if err != nil {
		return nil, err
	}

	if err := l.domainManager.SoftRebootVMI(vmi); err != nil {
		log.Log.Object(vmi).Reason(err).Errorf("Failed to soft reboot vmi")
		return nil, grpcstatus.Errorf(codes.Internal, "failed to soft reboot vmi: %s", getErrorMessage(err))
	}

	log.Log.Object(vmi).Info("Soft rebooted vmi")
	return &cmdv1.Response{Success: true}, nil
}

func (l *Launcher) KillVirtualMachine(_ context.Context, request *cmdv1.VMIRequest) (*cmdv1.Response, error) {

	vmi, err := getVMIFromRequest(request.Vmi)
	if err != nil {
		return nil, err
	}

	if err := l.domainManager.KillVMI(vmi); err != nil {
		log.Log.Object(vmi).Reason(err).Errorf("Failed to kill vmi")
		return nil, grpcstatus.Errorf(codes.Internal, "failed to kill vmi: %s", getErrorMessage(err))
	}

	log.Log.Object(vmi).Info("Signaled vmi kill")
	return &cmdv1.Response{Success: true}, nil
}

func (l *Launcher) ShutdownVirtualMachine(_ context.Context, request *cmdv1.VMIRequest) (*cmdv1.Response, error) {

	vmi, err := getVMIFromRequest(request.Vmi)
	if err != nil {
		return nil, err
	}

	if err := l.domainManager.SignalShutdownVMI(vmi); err != nil {
		log.Log.Object(vmi).Reason(err).Errorf("Failed to signal shutdown for vmi")
		return nil, grpcstatus.Errorf(codes.Internal, "failed to signal shutdown for vmi: %s", getErrorMessage(err))
	}

	log.Log.Object(vmi).Info("Signaled vmi shutdown")
	return &cmdv1.Response{Success: true}, nil
}

func (l *Launcher) DeleteVirtualMachine(_ context.Context, request *cmdv1.VMIRequest) (*cmdv1.Response, error) {

	vmi, err := getVMIFromRequest(request.Vmi)
	if err != nil {
		return nil, err
	}

	if err := l.domainManager.DeleteVMI(vmi); err != nil {
		log.Log.Object(vmi).Reason(err).Errorf("Failed to signal deletion for vmi")
		return nil, grpcstatus.Errorf(codes.Internal, "failed to signal deletion for vmi: %s", getErrorMessage(err))
	}

	log.Log.Object(vmi).Info("Signaled vmi deletion")
	return &cmdv1.Response{Success: true}, nil
}

func (l *Launcher) FinalizeVirtualMachineMigration(_ context.Context, request *cmdv1.VMIRequest) (*cmdv1.Response, error) {
	vmi, err := getVMIFromRequest(request.Vmi)
	if err != nil {
		return nil, err
	}

	if err := l.domainManager.FinalizeVirtualMachineMigration(vmi, request.Options); err != nil {
		log.Log.Object(vmi).Reason(err).Errorf("failed to finalize migration")
		return nil, grpcstatus.Errorf(codes.Internal, "failed to finalize migration: %s", getErrorMessage(err))
	}

	log.Log.Object(vmi).Info("migration finalized successfully")
	return &cmdv1.Response{Success: true}, nil
}

func (l *Launcher) HotplugHostDevices(_ context.Context, request *cmdv1.VMIRequest) (*cmdv1.Response, error) {
	vmi, err := getVMIFromRequest(request.Vmi)
	if err != nil {
		return nil, err
	}

	if err := l.domainManager.HotplugHostDevices(vmi); err != nil {
		log.Log.Object(vmi).Errorf("%s", err.Error())
		return nil, grpcstatus.Errorf(codes.Internal, "failed to hotplug host devices: %s", getErrorMessage(err))
	}

	return &cmdv1.Response{Success: true}, nil
}

func (l *Launcher) GetDomain(_ context.Context, _ *cmdv1.EmptyRequest) (*cmdv1.DomainResponse, error) {

	list, err := l.domainManager.ListAllDomains()
	if err != nil {
		return nil, grpcstatus.Errorf(codes.Internal, "failed to list domains: %s", getErrorMessage(err))
	}

	response := &cmdv1.DomainResponse{
		Response: &cmdv1.Response{Success: true},
	}

	if len(list) > 0 {
		domainObj := list[0]
		if osInfo := l.domainManager.GetGuestOSInfo(); osInfo != nil {
			domainObj.Status.OSInfo = *osInfo
		}
		if interfaces := l.domainManager.InterfacesStatus(); interfaces != nil {
			domainObj.Status.Interfaces = interfaces
		}
		domain, err := json.Marshal(domainObj)
		if err != nil {
			log.Log.Reason(err).Errorf("Failed to marshal domain")
			return nil, grpcstatus.Errorf(codes.Internal, "failed to marshal domain: %s", getErrorMessage(err))
		}
		response.Domain = string(domain)
	}

	return response, nil
}

func (l *Launcher) GetQemuVersion(_ context.Context, _ *cmdv1.EmptyRequest) (*cmdv1.QemuVersionResponse, error) {
	version, err := l.domainManager.GetQemuVersion()
	if err != nil {
		return nil, grpcstatus.Errorf(codes.Internal, "failed to get QEMU version: %s", getErrorMessage(err))
	}

	return &cmdv1.QemuVersionResponse{
		Response: &cmdv1.Response{Success: true},
		Version:  version,
	}, nil
}

func (l *Launcher) GetDomainStats(_ context.Context, _ *cmdv1.EmptyRequest) (*cmdv1.DomainStatsResponse, error) {

	stats, err := l.domainManager.GetDomainStats()
	if err != nil {
		return nil, grpcstatus.Errorf(codes.Internal, "failed to get domain stats: %s", getErrorMessage(err))
	}

	domainStats, err := json.Marshal(stats)
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to marshal domain stats")
		return nil, grpcstatus.Errorf(codes.Internal, "failed to marshal domain stats: %s", getErrorMessage(err))
	}

	return &cmdv1.DomainStatsResponse{
		Response:    &cmdv1.Response{Success: true},
		DomainStats: string(domainStats),
	}, nil
}

func (l *Launcher) GetDomainDirtyRateStats(_ context.Context, _ *cmdv1.EmptyRequest) (*cmdv1.DirtyRateStatsResponse, error) {
	const dirtyRateCalculationTime = time.Second
	stats, err := l.domainManager.GetDomainDirtyRateStats(dirtyRateCalculationTime)
	if err != nil {
		return nil, grpcstatus.Errorf(codes.Internal, "failed to get dirty rate stats: %s", getErrorMessage(err))
	}

	if !stats.MegabytesPerSecondSet {
		return nil, grpcstatus.Errorf(codes.Internal, "dirty rate MegabytesPerSecondSet is false")
	}

	return &cmdv1.DirtyRateStatsResponse{
		Response:     &cmdv1.Response{Success: true},
		DirtyRateMbs: stats.MegabytesPerSecond,
	}, nil
}

// GetGuestInfo collect guest info from the domain
func (l *Launcher) GetGuestInfo(_ context.Context, _ *cmdv1.EmptyRequest) (*cmdv1.GuestInfoResponse, error) {
	guestInfo := l.domainManager.GetGuestInfo()
	jGuestInfo, err := json.Marshal(guestInfo)
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to marshal agent info")
		return nil, grpcstatus.Errorf(codes.Internal, "failed to marshal guest info: %s", getErrorMessage(err))
	}

	return &cmdv1.GuestInfoResponse{
		Response:          &cmdv1.Response{Success: true},
		GuestInfoResponse: string(jGuestInfo),
	}, nil
}

// GetUsers returns the list of active users on the guest machine
func (l *Launcher) GetUsers(_ context.Context, _ *cmdv1.EmptyRequest) (*cmdv1.GuestUserListResponse, error) {
	users := l.domainManager.GetUsers()
	jUsers, err := json.Marshal(users)
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to marshal guest user list")
		return nil, grpcstatus.Errorf(codes.Internal, "failed to marshal guest user list: %s", getErrorMessage(err))
	}

	return &cmdv1.GuestUserListResponse{
		Response:              &cmdv1.Response{Success: true},
		GuestUserListResponse: string(jUsers),
	}, nil
}

// GetFilesystems returns a full list of active filesystems on the guest machine
func (l *Launcher) GetFilesystems(_ context.Context, _ *cmdv1.EmptyRequest) (*cmdv1.GuestFilesystemsResponse, error) {
	fs := l.domainManager.GetFilesystems()
	jFS, err := json.Marshal(fs)
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to marshal guest filesystem list")
		return nil, grpcstatus.Errorf(codes.Internal, "failed to marshal guest filesystem list: %s", getErrorMessage(err))
	}

	return &cmdv1.GuestFilesystemsResponse{
		Response:                 &cmdv1.Response{Success: true},
		GuestFilesystemsResponse: string(jFS),
	}, nil
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
		log.Log.Reason(err).Warning("GuestAgentPing probe failed")
		if l.notifier != nil && l.vmiName != "" {
			eventMsg := fmt.Sprintf("GuestAgentPing probe failed for VMI %s", l.vmiName)
			vmiRef := v1.NewVMIReferenceFromNameWithNS(l.vmiNamespace, l.vmiName)
			vmiRef.UID = l.vmiUID
			if sendErr := l.notifier.SendK8sEvent(vmiRef, k8sv1.EventTypeWarning, "GuestAgentPingFailed", eventMsg); sendErr != nil {
				log.Log.Reason(sendErr).Warning("Failed to send GuestAgentPingFailed event")
			}
		}

		return resp, err
	}
	return resp, nil
}

func RunServer(socketPath string,
	domainManager virtwrap.DomainManager,
	stopChan chan struct{},
	options *ServerOptions) (chan struct{}, error) {
	grpcServer := grpc.NewServer([]grpc.ServerOption{}...)
	if options == nil {
		options = NewServerOptions(false)
	}
	server := NewLauncher(domainManager, options)
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
		<-stopChan
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
	sevPlatformInfo, err := l.domainManager.GetSEVInfo()
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to get SEV platform info")
		return nil, grpcstatus.Errorf(codes.Internal, "failed to get SEV platform info: %s", getErrorMessage(err))
	}

	sevPlatformInfoJson, err := json.Marshal(sevPlatformInfo)
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to marshal SEV platform info")
		return nil, grpcstatus.Errorf(codes.Internal, "failed to marshal SEV platform info: %s", getErrorMessage(err))
	}

	return &cmdv1.SEVInfoResponse{
		Response: &cmdv1.Response{Success: true},
		SevInfo:  sevPlatformInfoJson,
	}, nil
}

func (l *Launcher) GetLaunchMeasurement(_ context.Context, request *cmdv1.VMIRequest) (*cmdv1.LaunchMeasurementResponse, error) {
	vmi, err := getVMIFromRequest(request.Vmi)
	if err != nil {
		return nil, err
	}

	sevMeasurementInfo, err := l.domainManager.GetLaunchMeasurement(vmi)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Errorf("Failed to get launch measuement")
		return nil, grpcstatus.Errorf(codes.Internal, "failed to get launch measurement: %s", getErrorMessage(err))
	}

	sevMeasurementInfoJson, err := json.Marshal(sevMeasurementInfo)
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to marshal launch measuement info")
		return nil, grpcstatus.Errorf(codes.Internal, "failed to marshal launch measurement info: %s", getErrorMessage(err))
	}

	return &cmdv1.LaunchMeasurementResponse{
		Response:          &cmdv1.Response{Success: true},
		LaunchMeasurement: sevMeasurementInfoJson,
	}, nil
}

func (l *Launcher) InjectLaunchSecret(_ context.Context, request *cmdv1.InjectLaunchSecretRequest) (*cmdv1.Response, error) {
	vmi, err := getVMIFromRequest(request.Vmi)
	if err != nil {
		return nil, err
	}

	var sevSecretOptions v1.SEVSecretOptions
	if err := json.Unmarshal(request.Options, &sevSecretOptions); err != nil {
		return nil, grpcstatus.Errorf(codes.InvalidArgument, "no valid secret options present in request: %v", err)
	}

	if err := l.domainManager.InjectLaunchSecret(vmi, &sevSecretOptions); err != nil {
		log.Log.Object(vmi).Reason(err).Errorf("Failed to inject SEV launch secret")
		return nil, grpcstatus.Errorf(codes.Internal, "failed to inject SEV launch secret: %s", getErrorMessage(err))
	}

	return &cmdv1.Response{Success: true}, nil
}

func (l *Launcher) SyncVirtualMachineMemory(_ context.Context, request *cmdv1.VMIRequest) (*cmdv1.Response, error) {
	vmi, err := getVMIFromRequest(request.Vmi)
	if err != nil {
		return nil, err
	}

	if _, exists := vmi.Annotations[v1.FuncTestMemoryHotplugFailAnnotation]; exists {
		return nil, grpcstatus.Errorf(codes.Internal, "%s", v1.FuncTestMemoryHotplugFailAnnotation)
	}

	if err := l.domainManager.UpdateGuestMemory(vmi); err != nil {
		log.Log.Object(vmi).Reason(err).Errorf("Failed update VMI guest memory")
		return nil, grpcstatus.Errorf(codes.Internal, "failed to update vmi guest memory: %s", getErrorMessage(err))
	}

	log.Log.Object(vmi).Info("guest memory has been updated")
	return &cmdv1.Response{Success: true}, nil
}

func (l *Launcher) GetScreenshot(_ context.Context, request *cmdv1.VMIRequest) (*cmdv1.ScreenshotResponse, error) {
	vmi, err := getVMIFromRequest(request.Vmi)
	if err != nil {
		return nil, err
	}

	domainScreenshot, err := l.domainManager.GetScreenshot(vmi)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Errorf("Failed to screenshot")
		return nil, grpcstatus.Errorf(codes.Internal, "failed to get screenshot: %s", getErrorMessage(err))
	}

	return &cmdv1.ScreenshotResponse{
		Response: &cmdv1.Response{Success: true},
		Mime:     domainScreenshot.Mime,
		Data:     domainScreenshot.Data,
	}, nil
}

func ReceivedEarlyExitSignal() bool {
	_, earlyExit := os.LookupEnv(receivedEarlyExitSignalEnvVar)
	return earlyExit
}

func getBackupOptionsFromRequest(request *cmdv1.BackupRequest) (*backupv1.BackupOptions, error) {
	if request.Options == nil {
		return nil, fmt.Errorf("backup options object not present in command server request")
	}

	var options *backupv1.BackupOptions
	if err := json.Unmarshal(request.Options, &options); err != nil {
		return nil, fmt.Errorf("no valid backup options object present in command server request: %v", err)
	}

	switch options.Cmd {
	case backupv1.Start:
		switch options.Mode {
		case backupv1.PushMode, backupv1.PullMode:
			if options.TargetPath == nil {
				return nil, fmt.Errorf("backup targetPath wasn't provided")
			}
		default:
			return nil, fmt.Errorf("unknown backup mode: only Push and Pull are supported")
		}
	case backupv1.Export:
		if err := validateBackupExportRequest(options); err != nil {
			return nil, err
		}
		return options, nil
	case backupv1.Abort:
		return options, nil
	default:
		return nil, fmt.Errorf("cmd unsupported, backup command only supports start or abort")
	}

	return options, nil
}

func validateBackupExportRequest(options *backupv1.BackupOptions) error {
	if options.Mode != backupv1.PullMode {
		return fmt.Errorf("can only export Pull mode backup")
	}
	if options.ExportServerAddr == nil {
		return fmt.Errorf("backup export server address wasn't provided")
	}
	if options.ExportServerName == nil {
		return fmt.Errorf("backup export server name wasn't provided")
	}
	if options.BackupName == "" {
		return fmt.Errorf("backup name wasn't provided")
	}
	if options.BackupStartTime == nil {
		return fmt.Errorf("backup start time wasn't provided")
	}
	if options.BackupCert == nil {
		return fmt.Errorf("backup certificate wasn't provided")
	}
	if options.BackupKey == nil {
		return fmt.Errorf("backup key wasn't provided")
	}
	if options.CACert == nil {
		return fmt.Errorf("backup export server CA cert wasn't provided")
	}
	return nil
}

func (l *Launcher) BackupVirtualMachine(_ context.Context, request *cmdv1.BackupRequest) (*cmdv1.Response, error) {
	vmi, err := getVMIFromRequest(request.Vmi)
	if err != nil {
		return nil, err
	}

	if !storage.IsChangedBlockTrackingEnabled(vmi) {
		return nil, grpcstatus.Errorf(codes.FailedPrecondition, "%s", storage.ChangedBlockTrackingNotEnabledMsg)
	}

	options, err := getBackupOptionsFromRequest(request)
	if err != nil {
		return nil, grpcstatus.Errorf(codes.InvalidArgument, "%s", err.Error())
	}

	if err := l.domainManager.BackupVirtualMachine(vmi, options); err != nil {
		log.Log.Object(vmi).Reason(err).Errorf("Failed to run backup job")
		return nil, grpcstatus.Errorf(codes.Internal, "failed to run backup job: %s", err.Error())
	}

	log.Log.Object(vmi).Info("VMI backup job initiated")
	return &cmdv1.Response{Success: true}, nil
}

func (l *Launcher) RedefineCheckpoint(_ context.Context, request *cmdv1.RedefineCheckpointRequest) (*cmdv1.RedefineCheckpointResponse, error) {
	vmi, err := getVMIFromRequest(request.Vmi)
	if err != nil {
		return &cmdv1.RedefineCheckpointResponse{
			Response: &cmdv1.Response{
				Success: false,
				Message: err.Error(),
			},
			CheckpointInvalid: false,
		}, nil
	}

	if !storage.IsChangedBlockTrackingEnabled(vmi) {
		return &cmdv1.RedefineCheckpointResponse{
			Response: &cmdv1.Response{
				Success: false,
				Message: "Redefine checkpoint failed: ChangedBlockTracking is not enabled",
			},
			CheckpointInvalid: false,
		}, nil
	}

	checkpoint := &backupv1.BackupCheckpoint{}
	if err := json.Unmarshal(request.Checkpoint, checkpoint); err != nil {
		return &cmdv1.RedefineCheckpointResponse{
			Response: &cmdv1.Response{
				Success: false,
				Message: fmt.Sprintf("Redefine checkpoint failed: invalid checkpoint info: %v", err),
			},
			CheckpointInvalid: false,
		}, nil
	}

	checkpointInvalid, err := l.domainManager.RedefineCheckpoint(vmi, checkpoint)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Errorf("Failed to redefine checkpoint %s", checkpoint.Name)
		return &cmdv1.RedefineCheckpointResponse{
			Response: &cmdv1.Response{
				Success: false,
				Message: err.Error(),
			},
			CheckpointInvalid: checkpointInvalid,
		}, nil
	}

	log.Log.Object(vmi).Infof("Checkpoint %s redefined successfully", checkpoint.Name)
	return &cmdv1.RedefineCheckpointResponse{
		Response: &cmdv1.Response{
			Success: true,
		},
	}, nil
}
