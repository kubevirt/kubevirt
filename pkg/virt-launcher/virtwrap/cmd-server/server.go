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

	backupv1 "kubevirt.io/api/backup/v1beta1"
	v1 "kubevirt.io/api/core/v1"
	pluginv1alpha1 "kubevirt.io/api/plugin/v1alpha1"
	"kubevirt.io/client-go/log"

	cmdv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"
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

	logErrorPrefix                  = "Failed to "
	grpcErrorPrefix                 = "failed to "
	migrateVMIMessage               = "migrate vmi"
	prepareMigrationTargetMessage   = "prepare migration target pod"
	updateVCPUsMessage              = "update vmi vcpus"
	syncVMIMessage                  = "sync vmi"
	pauseVMIMessage                 = "pause vmi"
	unpauseVMIMessage               = "unpause vmi"
	memoryDumpMessage               = "dump vmi memory"
	freezeVMIMessage                = "freeze vmi"
	unfreezeVMIMessage              = "unfreeze vmi"
	resetVMIMessage                 = "reset vmi"
	softRebootVMIMessage            = "soft reboot vmi"
	killVMIMessage                  = "kill vmi"
	shutdownVMIMessage              = "signal shutdown for vmi"
	deleteVMIMessage                = "signal deletion for vmi"
	finalizeMigrationMessage        = "finalize migration"
	hotplugHostDevicesMessage       = "hotplug host devices"
	marshalDomainMessage            = "marshal domain"
	marshalDomainStatsMessage       = "marshal domain stats"
	marshalGuestInfoMessage         = "marshal guest info"
	marshalGuestUsersMessage        = "marshal guest user list"
	marshalGuestFilesystemsMessage  = "marshal guest filesystem list"
	getSEVInfoMessage               = "get SEV platform info"
	marshalSEVInfoMessage           = "marshal SEV platform info"
	getLaunchMeasurementMessage     = "get launch measurement"
	marshalLaunchMeasurementMessage = "marshal launch measurement info"
	injectLaunchSecretMessage       = "inject SEV launch secret"
	updateGuestMemoryMessage        = "update vmi guest memory"
	getScreenshotMessage            = "get screenshot"
	backupVMIMessage                = "run backup job"
	externalSnapshotMessage         = "start external snapshot"
	commitSnapshotMessage           = "start snapshot overlay commit"
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
		return nil, grpcstatus.Error(codes.InvalidArgument, err.Error())
	}

	if err := l.domainManager.MigrateVMI(vmi, options); err != nil {
		log.Log.Object(vmi).Reason(err).Error(logErrorPrefix + migrateVMIMessage)
		return nil, grpcstatus.Errorf(codes.Internal, grpcErrorPrefix+migrateVMIMessage+": %s", getErrorMessage(err))
	}

	log.Log.Object(vmi).Info("Signaled vmi migration")
	return &cmdv1.Response{Success: true}, nil
}

func (l *Launcher) CancelVirtualMachineMigration(_ context.Context, request *cmdv1.VMIRequest) (*cmdv1.Response, error) {

	vmi, err := getVMIFromRequest(request.Vmi)
	if err != nil {
		return nil, err
	}

	l.domainManager.CancelVMIMigration(vmi)
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
		return nil, grpcstatus.Error(codes.InvalidArgument, err.Error())
	}

	if err := l.domainManager.PrepareMigrationTarget(vmi, l.allowEmulation, request.Options); err != nil {
		log.Log.Object(vmi).Reason(err).Error(logErrorPrefix + prepareMigrationTargetMessage)
		return nil, grpcstatus.Errorf(codes.Internal, grpcErrorPrefix+prepareMigrationTargetMessage+": %s", getErrorMessage(err))
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
		log.Log.Object(vmi).Reason(err).Error(logErrorPrefix + updateVCPUsMessage)
		return nil, grpcstatus.Errorf(codes.Internal, grpcErrorPrefix+updateVCPUsMessage+": %s", getErrorMessage(err))
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
		return nil, grpcstatus.Error(codes.InvalidArgument, err.Error())
	}

	if _, err := l.domainManager.SyncVMI(vmi, l.allowEmulation, request.Options); err != nil {
		log.Log.Object(vmi).Reason(err).Error(logErrorPrefix + syncVMIMessage)
		return nil, grpcstatus.Errorf(codes.Internal, grpcErrorPrefix+syncVMIMessage+": %s", getErrorMessage(err))
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
		log.Log.Object(vmi).Reason(err).Error(logErrorPrefix + pauseVMIMessage)
		return nil, grpcstatus.Errorf(codes.Internal, grpcErrorPrefix+pauseVMIMessage+": %s", getErrorMessage(err))
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
		log.Log.Object(vmi).Reason(err).Error(logErrorPrefix + unpauseVMIMessage)
		return nil, grpcstatus.Errorf(codes.Internal, grpcErrorPrefix+unpauseVMIMessage+": %s", getErrorMessage(err))
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
		log.Log.Object(vmi).Reason(err).Error(logErrorPrefix + memoryDumpMessage)
		return nil, grpcstatus.Errorf(codes.Internal, grpcErrorPrefix+memoryDumpMessage+": %s", getErrorMessage(err))
	}

	return &cmdv1.Response{Success: true}, nil
}

func (l *Launcher) FreezeVirtualMachine(_ context.Context, request *cmdv1.FreezeRequest) (*cmdv1.Response, error) {
	vmi, err := getVMIFromRequest(request.Vmi)
	if err != nil {
		return nil, err
	}

	if err := l.domainManager.FreezeVMI(vmi, request.UnfreezeTimeoutSeconds); err != nil {
		log.Log.Object(vmi).Reason(err).Error(logErrorPrefix + freezeVMIMessage)
		return nil, grpcstatus.Errorf(codes.Internal, grpcErrorPrefix+freezeVMIMessage+": %s", getErrorMessage(err))
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
		log.Log.Object(vmi).Reason(err).Error(logErrorPrefix + unfreezeVMIMessage)
		return nil, grpcstatus.Errorf(codes.Internal, grpcErrorPrefix+unfreezeVMIMessage+": %s", getErrorMessage(err))
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
		log.Log.Object(vmi).Reason(err).Error(logErrorPrefix + resetVMIMessage)
		return nil, grpcstatus.Errorf(codes.Internal, grpcErrorPrefix+resetVMIMessage+": %s", getErrorMessage(err))
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
		log.Log.Object(vmi).Reason(err).Error(logErrorPrefix + softRebootVMIMessage)
		return nil, grpcstatus.Errorf(codes.Internal, grpcErrorPrefix+softRebootVMIMessage+": %s", getErrorMessage(err))
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
		log.Log.Object(vmi).Reason(err).Error(logErrorPrefix + killVMIMessage)
		return nil, grpcstatus.Errorf(codes.Internal, grpcErrorPrefix+killVMIMessage+": %s", getErrorMessage(err))
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
		log.Log.Object(vmi).Reason(err).Error(logErrorPrefix + shutdownVMIMessage)
		return nil, grpcstatus.Errorf(codes.Internal, grpcErrorPrefix+shutdownVMIMessage+": %s", getErrorMessage(err))
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
		log.Log.Object(vmi).Reason(err).Error(logErrorPrefix + deleteVMIMessage)
		return nil, grpcstatus.Errorf(codes.Internal, grpcErrorPrefix+deleteVMIMessage+": %s", getErrorMessage(err))
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
		log.Log.Object(vmi).Reason(err).Error(logErrorPrefix + finalizeMigrationMessage)
		return nil, grpcstatus.Errorf(codes.Internal, grpcErrorPrefix+finalizeMigrationMessage+": %s", getErrorMessage(err))
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
		log.Log.Object(vmi).Reason(err).Error(logErrorPrefix + hotplugHostDevicesMessage)
		return nil, grpcstatus.Errorf(codes.Internal, grpcErrorPrefix+hotplugHostDevicesMessage+": %s", getErrorMessage(err))
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
			log.Log.Reason(err).Error(logErrorPrefix + marshalDomainMessage)
			return nil, grpcstatus.Errorf(codes.Internal, grpcErrorPrefix+marshalDomainMessage+": %s", getErrorMessage(err))
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
		log.Log.Reason(err).Error(logErrorPrefix + marshalDomainStatsMessage)
		return nil, grpcstatus.Errorf(codes.Internal, grpcErrorPrefix+marshalDomainStatsMessage+": %s", getErrorMessage(err))
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
		return nil, grpcstatus.Error(codes.Internal, "dirty rate MegabytesPerSecondSet is false")
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
		log.Log.Reason(err).Error(logErrorPrefix + marshalGuestInfoMessage)
		return nil, grpcstatus.Errorf(codes.Internal, grpcErrorPrefix+marshalGuestInfoMessage+": %s", getErrorMessage(err))
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
		log.Log.Reason(err).Error(logErrorPrefix + marshalGuestUsersMessage)
		return nil, grpcstatus.Errorf(codes.Internal, grpcErrorPrefix+marshalGuestUsersMessage+": %s", getErrorMessage(err))
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
		log.Log.Reason(err).Error(logErrorPrefix + marshalGuestFilesystemsMessage)
		return nil, grpcstatus.Errorf(codes.Internal, grpcErrorPrefix+marshalGuestFilesystemsMessage+": %s", getErrorMessage(err))
	}

	return &cmdv1.GuestFilesystemsResponse{
		Response:                 &cmdv1.Response{Success: true},
		GuestFilesystemsResponse: string(jFS),
	}, nil
}

// Exec the provided command and return it's success
func (l *Launcher) Exec(ctx context.Context, request *cmdv1.ExecRequest) (*cmdv1.ExecResponse, error) {
	stdOut, err := l.domainManager.Exec(request.DomainName, request.Command, request.Args, request.TimeoutSeconds)

	exitCode := agent.ExecExitCode{}
	if err != nil && !errors.As(err, &exitCode) {
		return nil, grpcstatus.Errorf(codes.Internal, "failed to exec command: %s", err.Error())
	}

	return &cmdv1.ExecResponse{
		Response: &cmdv1.Response{Success: true},
		ExitCode: int32(exitCode.ExitCode),
		StdOut:   stdOut,
	}, nil
}

func (l *Launcher) GuestPing(ctx context.Context, request *cmdv1.GuestPingRequest) (*cmdv1.GuestPingResponse, error) {
	err := l.domainManager.GuestPing(request.DomainName)
	if err != nil {
		log.Log.Reason(err).Warning("GuestAgentPing probe failed")
		if l.notifier != nil && l.vmiName != "" {
			eventMsg := fmt.Sprintf("GuestAgentPing probe failed for VMI %s", l.vmiName)
			vmiRef := v1.NewVMIReferenceFromNameWithNS(l.vmiNamespace, l.vmiName)
			vmiRef.UID = l.vmiUID
			if sendErr := l.notifier.SendK8sEvent(vmiRef, k8sv1.EventTypeWarning, "GuestAgentPingFailed", eventMsg); sendErr != nil {
				log.Log.Reason(sendErr).Warning("Failed to send GuestAgentPingFailed event")
			}
		}
		return nil, grpcstatus.Errorf(codes.Internal, "guest agent ping failed: %s", err.Error())
	}
	return &cmdv1.GuestPingResponse{
		Response: &cmdv1.Response{Success: true},
	}, nil
}

func RegisterCmdServer(cmdServer cmdv1.CmdServer) func(grpcServer *grpc.Server) {
	return func(server *grpc.Server) {
		registerInfoServer(server)
		cmdv1.RegisterCmdServer(server, cmdServer)
	}
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
		log.Log.Reason(err).Error(logErrorPrefix + getSEVInfoMessage)
		return nil, grpcstatus.Errorf(codes.Internal, grpcErrorPrefix+getSEVInfoMessage+": %s", getErrorMessage(err))
	}

	sevPlatformInfoJson, err := json.Marshal(sevPlatformInfo)
	if err != nil {
		log.Log.Reason(err).Error(logErrorPrefix + marshalSEVInfoMessage)
		return nil, grpcstatus.Errorf(codes.Internal, grpcErrorPrefix+marshalSEVInfoMessage+": %s", getErrorMessage(err))
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
		log.Log.Object(vmi).Reason(err).Error(logErrorPrefix + getLaunchMeasurementMessage)
		return nil, grpcstatus.Errorf(codes.Internal, grpcErrorPrefix+getLaunchMeasurementMessage+": %s", getErrorMessage(err))
	}

	sevMeasurementInfoJson, err := json.Marshal(sevMeasurementInfo)
	if err != nil {
		log.Log.Reason(err).Error(logErrorPrefix + marshalLaunchMeasurementMessage)
		return nil, grpcstatus.Errorf(codes.Internal, grpcErrorPrefix+marshalLaunchMeasurementMessage+": %s", getErrorMessage(err))
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
		log.Log.Object(vmi).Reason(err).Error(logErrorPrefix + injectLaunchSecretMessage)
		return nil, grpcstatus.Errorf(codes.Internal, grpcErrorPrefix+injectLaunchSecretMessage+": %s", getErrorMessage(err))
	}

	return &cmdv1.Response{Success: true}, nil
}

func (l *Launcher) SyncVirtualMachineMemory(_ context.Context, request *cmdv1.VMIRequest) (*cmdv1.Response, error) {
	vmi, err := getVMIFromRequest(request.Vmi)
	if err != nil {
		return nil, err
	}

	if _, exists := vmi.Annotations[v1.FuncTestMemoryHotplugFailAnnotation]; exists {
		return nil, grpcstatus.Error(codes.Internal, v1.FuncTestMemoryHotplugFailAnnotation)
	}

	if err := l.domainManager.UpdateGuestMemory(vmi); err != nil {
		log.Log.Object(vmi).Reason(err).Error(logErrorPrefix + updateGuestMemoryMessage)
		return nil, grpcstatus.Errorf(codes.Internal, grpcErrorPrefix+updateGuestMemoryMessage+": %s", getErrorMessage(err))
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
		log.Log.Object(vmi).Reason(err).Error(logErrorPrefix + getScreenshotMessage)
		return nil, grpcstatus.Errorf(codes.Internal, grpcErrorPrefix+getScreenshotMessage+": %s", getErrorMessage(err))
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
	case backupv1.Abort:
		return options, nil
	default:
		return nil, fmt.Errorf("cmd unsupported, backup command only supports start or abort")
	}

	return options, nil
}

func (l *Launcher) BackupVirtualMachine(_ context.Context, request *cmdv1.BackupRequest) (*cmdv1.Response, error) {
	vmi, err := getVMIFromRequest(request.Vmi)
	if err != nil {
		return nil, err
	}

	if !storage.IsChangedBlockTrackingEnabled(vmi) {
		return nil, grpcstatus.Error(codes.FailedPrecondition, storage.ChangedBlockTrackingNotEnabledMsg)
	}

	options, err := getBackupOptionsFromRequest(request)
	if err != nil {
		return nil, grpcstatus.Error(codes.InvalidArgument, err.Error())
	}

	if err := l.domainManager.BackupVirtualMachine(vmi, options); err != nil {
		log.Log.Object(vmi).Reason(err).Error(logErrorPrefix + backupVMIMessage)
		return nil, grpcstatus.Errorf(codes.Internal, grpcErrorPrefix+backupVMIMessage+": %s", err.Error())
	}

	log.Log.Object(vmi).Info("VMI backup job initiated")
	return &cmdv1.Response{Success: true}, nil
}

func (l *Launcher) ExternalSnapshot(_ context.Context, request *cmdv1.ExternalSnapshotRequest) (*cmdv1.Response, error) {
	vmi, err := getVMIFromRequest(request.Vmi)
	if err != nil {
		return nil, err
	}

	if err := l.domainManager.ExternalSnapshot(vmi, request.OverlayDir, request.Volumes); err != nil {
		log.Log.Object(vmi).Reason(err).Error(logErrorPrefix + externalSnapshotMessage)
		return nil, grpcstatus.Errorf(codes.Internal, grpcErrorPrefix+externalSnapshotMessage+": %s", getErrorMessage(err))
	}

	log.Log.Object(vmi).Info("External snapshot transaction accepted")
	return &cmdv1.Response{Success: true}, nil
}

func (l *Launcher) CommitSnapshot(_ context.Context, request *cmdv1.CommitSnapshotRequest) (*cmdv1.Response, error) {
	vmi, err := getVMIFromRequest(request.Vmi)
	if err != nil {
		return nil, err
	}

	if err := l.domainManager.CommitSnapshot(vmi, request.OverlayDir); err != nil {
		log.Log.Object(vmi).Reason(err).Error(logErrorPrefix + commitSnapshotMessage)
		return nil, grpcstatus.Errorf(codes.Internal, grpcErrorPrefix+commitSnapshotMessage+": %s", getErrorMessage(err))
	}

	log.Log.Object(vmi).Info("Snapshot overlay commit request accepted")
	return &cmdv1.Response{Success: true}, nil
}
