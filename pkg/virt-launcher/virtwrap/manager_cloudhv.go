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
 * Copyright 2022 Intel Corporation.
 *
 */

package virtwrap

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"kubevirt.io/client-go/log"

	cloudinit "kubevirt.io/kubevirt/pkg/cloud-init"
	"kubevirt.io/kubevirt/pkg/config"
	containerdisk "kubevirt.io/kubevirt/pkg/container-disk"
	"kubevirt.io/kubevirt/pkg/emptydisk"
	"kubevirt.io/kubevirt/pkg/hooks"
	"kubevirt.io/kubevirt/pkg/network/cache"
	netsetup "kubevirt.io/kubevirt/pkg/network/setup"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"

	v1 "kubevirt.io/api/core/v1"

	ephemeraldisk "kubevirt.io/kubevirt/pkg/ephemeral-disk"
	cmdv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter"
	openapiClient "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/openapi/cloud-hypervisor/client"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/stats"
)

const (
	clientTimeout = 300
)

type CloudHvDomainManager struct {
	client               *openapiClient.DefaultApiService
	vmConfig             openapiClient.VmConfig
	ephemeralDiskCreator ephemeraldisk.EphemeralDiskCreatorInterface
	started              bool
	domain               *api.Domain
}

func NewCloudHvDomainManager(apiSocketPath, ephemeralDiskDir, efiDir string, ephemeralDiskCreator ephemeraldisk.EphemeralDiskCreatorInterface) (*CloudHvDomainManager, error) {
	return newCloudHvDomainManager(apiSocketPath, ephemeralDiskDir, efiDir, ephemeralDiskCreator)
}

func newCloudHvDomainManager(apiSocketPath, ephemeralDiskDir, efiDir string, ephemeralDiskCreator ephemeraldisk.EphemeralDiskCreatorInterface) (*CloudHvDomainManager, error) {
	apiSocketAddr, err := net.ResolveUnixAddr("unix", apiSocketPath)
	if err != nil {
		return nil, err
	}

	clientConfig := openapiClient.NewConfiguration()
	clientConfig.HTTPClient = &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, path string) (net.Conn, error) {
				return net.DialUnix("unix", nil, apiSocketAddr)
			},
		},
	}

	return &CloudHvDomainManager{
		client: openapiClient.NewAPIClient(clientConfig).DefaultApi,
		vmConfig: openapiClient.VmConfig{
			Kernel: openapiClient.KernelConfig{
				Path: filepath.Join(efiDir, "CLOUDHV.fd"),
			},
		},
		ephemeralDiskCreator: ephemeralDiskCreator,
		started:              false,
		domain:               &api.Domain{},
	}, nil
}

func (c *CloudHvDomainManager) PrepareMigrationTarget(
	vmi *v1.VirtualMachineInstance,
	allowEmulation bool,
	options *cmdv1.VirtualMachineOptions,
) error {
	return fmt.Errorf("PrepareMigrationTarget not implemented for CloudHvDomainManager")
}

func (c *CloudHvDomainManager) FinalizeVirtualMachineMigration(vmi *v1.VirtualMachineInstance) error {
	return fmt.Errorf("FinalizeVirtualMachineMigration not implemented for CloudHvDomainManager")
}

func (c *CloudHvDomainManager) HotplugHostDevices(vmi *v1.VirtualMachineInstance) error {
	return fmt.Errorf("HotplugHostDevices not implemented for CloudHvDomainManager")
}

func (c *CloudHvDomainManager) Exec(domainName, command string, args []string, timeoutSeconds int32) (string, error) {
	return "", fmt.Errorf("Exec not implemented for CloudHvDomainManager")
}

func (c *CloudHvDomainManager) GuestPing(domainName string) error {
	return fmt.Errorf("GuestPing not implemented for CloudHvDomainManager")
}

func (c *CloudHvDomainManager) CancelVMIMigration(vmi *v1.VirtualMachineInstance) error {
	return fmt.Errorf("CancelVMIMigration not implemented for CloudHvDomainManager")
}

func (c *CloudHvDomainManager) MigrateVMI(vmi *v1.VirtualMachineInstance, options *cmdclient.MigrationOptions) error {
	return fmt.Errorf("MigrateVMI not implemented for CloudHvDomainManager")
}

func (c *CloudHvDomainManager) SyncVMI(vmi *v1.VirtualMachineInstance, allowEmulation bool, options *cmdv1.VirtualMachineOptions) (*api.DomainSpec, error) {
	// As a first step, let's implement this function to create and boot a
	// VM if it's not running yet. We can add more advanced machine state
	// handling later.

	// Convert the VMI definition into a Cloud Hypervisor configuration.
	vmConfig := c.vmConfig
	if err := converter.ConvertVirtualMachineInstanceToVmConfig(vmi, &vmConfig); err != nil {
		return nil, err
	}

	if !vmi.IsRunning() && !vmi.IsFinal() && !c.started {
		c.domain.ObjectMeta.Name = vmi.ObjectMeta.Name
		c.domain.ObjectMeta.Namespace = vmi.ObjectMeta.Namespace
		c.domain.ObjectMeta.UID = vmi.UID
		c.domain.Spec.Metadata.KubeVirt.UID = vmi.UID
		c.domain.Spec.Name = api.VMINamespaceKeyFunc(vmi)

		c.vmConfig = vmConfig

		// Run pre-start hooks
		if err := c.preStartHook(vmi); err != nil {
			return nil, err
		}

		ctx, cancel := context.WithTimeout(context.Background(), clientTimeout*time.Second)
		defer cancel()

		// Create the VM
		resp, err := c.client.CreateVM(ctx).VmConfig(c.vmConfig).Execute()
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != 204 {
			return nil, fmt.Errorf("Failed creating the VM: %s", resp.Status)
		}

		// And finally boot the VM
		resp, err = c.client.BootVM(ctx).Execute()
		if err != nil {
			if resp != nil {
				buf := new(strings.Builder)
				io.Copy(buf, resp.Body)
				log.Log.Error(buf.String())
			}
			return nil, err
		}
		if resp.StatusCode != 204 {
			return nil, fmt.Errorf("Failed booting the VM: %s", resp.Status)
		}

		c.started = true
		c.domain.Status.Status = api.Running

		// Run post-start hooks
		if err := c.postStartHook(vmi); err != nil {
			return nil, err
		}
	}

	return &c.domain.Spec, nil
}

func (c *CloudHvDomainManager) MemoryDump(vmi *v1.VirtualMachineInstance, dumpPath string) error {
	// This is supported through vm.coredump, which will land as part of
	// Cloud Hypervisor v25.0, and it requires the VMM to be built with the
	// proper feature "guest_debug".
	return nil
}

func (c *CloudHvDomainManager) PauseVMI(vmi *v1.VirtualMachineInstance) error {
	ctx, cancel := context.WithTimeout(context.Background(), clientTimeout*time.Second)
	defer cancel()

	resp, err := c.client.PauseVM(ctx).Execute()
	if err != nil {
		return err
	}
	if resp.StatusCode != 204 {
		return fmt.Errorf("Failed pausing the VM: %s", resp.Status)
	}

	c.domain.Status.Status = api.Paused

	return nil
}

func (c *CloudHvDomainManager) UnpauseVMI(vmi *v1.VirtualMachineInstance) error {
	ctx, cancel := context.WithTimeout(context.Background(), clientTimeout*time.Second)
	defer cancel()

	resp, err := c.client.ResumeVM(ctx).Execute()
	if err != nil {
		return err
	}
	if resp.StatusCode != 204 {
		return fmt.Errorf("Failed unpausing the VM: %s", resp.Status)
	}

	c.domain.Status.Status = api.Running

	return nil
}

func (c *CloudHvDomainManager) FreezeVMI(vmi *v1.VirtualMachineInstance, unfreezeTimeoutSeconds int32) error {
	return fmt.Errorf("FreezeVMI not implemented for CloudHvDomainManager")
}

func (c *CloudHvDomainManager) UnfreezeVMI(vmi *v1.VirtualMachineInstance) error {
	return fmt.Errorf("UnfreezeVMI not implemented for CloudHvDomainManager")
}

func (c *CloudHvDomainManager) SoftRebootVMI(vmi *v1.VirtualMachineInstance) error {
	return nil
}

func (c *CloudHvDomainManager) MarkGracefulShutdownVMI(vmi *v1.VirtualMachineInstance) error {
	return fmt.Errorf("MarkGracefulShutdownVMI not implemented for CloudHvDomainManager")
}

func (c *CloudHvDomainManager) SignalShutdownVMI(vmi *v1.VirtualMachineInstance) error {
	ctx, cancel := context.WithTimeout(context.Background(), clientTimeout*time.Second)
	defer cancel()

	resp, err := c.client.PowerButtonVM(ctx).Execute()
	if err != nil {
		return err
	}
	if resp.StatusCode != 204 {
		return fmt.Errorf("Failed shutting down the VM: %s", resp.Status)
	}

	return nil
}

func (c *CloudHvDomainManager) KillVMI(vmi *v1.VirtualMachineInstance) error {
	ctx, cancel := context.WithTimeout(context.Background(), clientTimeout*time.Second)
	defer cancel()

	resp, err := c.client.ShutdownVM(ctx).Execute()
	if err != nil {
		return err
	}
	if resp.StatusCode != 204 {
		return fmt.Errorf("Failed killing the VM: %s", resp.Status)
	}

	c.domain.Status.Status = api.Shutdown

	return nil
}

func (c *CloudHvDomainManager) DeleteVMI(vmi *v1.VirtualMachineInstance) error {
	ctx, cancel := context.WithTimeout(context.Background(), clientTimeout*time.Second)
	defer cancel()

	resp, err := c.client.DeleteVM(ctx).Execute()
	if err != nil {
		return err
	}
	if resp.StatusCode != 204 {
		return fmt.Errorf("Failed deleting the VM: %s", resp.Status)
	}

	resp, err = c.client.ShutdownVMM(ctx).Execute()
	if err != nil {
		return err
	}
	if resp.StatusCode != 204 {
		return fmt.Errorf("Failed terminating the VMM: %s", resp.Status)
	}

	c.domain.Status.Status = api.NoState

	return nil
}

func (c *CloudHvDomainManager) ListAllDomains() ([]*api.Domain, error) {
	return []*api.Domain{c.domain}, nil
}

func (c *CloudHvDomainManager) GetDomainStats() ([]*stats.DomainStats, error) {
	return []*stats.DomainStats{}, nil
}

func (c *CloudHvDomainManager) GetGuestInfo() (v1.VirtualMachineInstanceGuestAgentInfo, error) {
	return v1.VirtualMachineInstanceGuestAgentInfo{}, fmt.Errorf("GetGuestInfo not implemented for CloudHvDomainManager")
}

func (c *CloudHvDomainManager) InterfacesStatus() []api.InterfaceStatus {
	return []api.InterfaceStatus{}
}

func (c *CloudHvDomainManager) GetGuestOSInfo() *api.GuestOSInfo {
	return &api.GuestOSInfo{}
}

func (c *CloudHvDomainManager) GetUsers() ([]v1.VirtualMachineInstanceGuestOSUser, error) {
	return []v1.VirtualMachineInstanceGuestOSUser{}, fmt.Errorf("GetUsers not implemented for CloudHvDomainManager")
}

func (c *CloudHvDomainManager) GetFilesystems() ([]v1.VirtualMachineInstanceFileSystem, error) {
	return []v1.VirtualMachineInstanceFileSystem{}, fmt.Errorf("GetFilesystems not implemented for CloudHvDomainManager")
}

func (c *CloudHvDomainManager) getVmState() (string, error) {
	vmInfo, err := c.getVmInfo()
	if err != nil {
		return "", err
	}

	return vmInfo.State, nil
}

func (c *CloudHvDomainManager) getVmInfo() (openapiClient.VmInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), clientTimeout*time.Second)
	defer cancel()

	vmInfo, resp, err := c.client.VmInfoGet(ctx).Execute()
	if err != nil {
		return openapiClient.VmInfo{}, err
	}
	if resp.StatusCode != 200 {
		return openapiClient.VmInfo{}, fmt.Errorf("Failed retrieving VM info: %s", resp.Status)
	}

	return vmInfo, nil
}

// All local environment setup that needs to occur before VirtualMachineInstance starts
// can be done in this function. This includes things like...
//
// - storage prep
func (c *CloudHvDomainManager) preStartHook(vmi *v1.VirtualMachineInstance) error {
	logger := log.Log.Object(vmi)

	logger.Info("Executing PreStartHook on VMI pod environment")

	// Create ephemeral disks for container disks
	if err := containerdisk.CreateRawEphemeralImages(vmi); err != nil {
		return fmt.Errorf("preparing ephemeral container disk images failed: %v", err)
	}

	// Create empty disks
	if err := emptydisk.NewEmptyRawDiskCreator().CreateTemporaryDisks(vmi); err != nil {
		return fmt.Errorf("creating empty disks failed: %v", err)
	}

	// Create cloud-init images
	// generate cloud-init data
	cloudInitData, err := cloudinit.ReadCloudInitVolumeDataSource(vmi, config.SecretSourceDir)
	if err != nil {
		return fmt.Errorf("ReadCloudInitVolumeDataSource failed: %v", err)
	}

	// Pass cloud-init data to PreCloudInitIso hook
	logger.Info("Starting PreCloudInitIso hook")
	hooksManager := hooks.GetManager()
	cloudInitData, err = hooksManager.PreCloudInitIso(vmi, cloudInitData)
	if err != nil {
		return fmt.Errorf("PreCloudInitIso hook failed: %v", err)
	}

	if cloudInitData != nil {
		// need to prepare the local path for cloud-init in advance for proper
		// detection of the disk driver cache mode
		if err := cloudinit.PrepareLocalPath(vmi.Name, vmi.Namespace); err != nil {
			return fmt.Errorf("PrepareLocalPath failed: %v", err)
		}

		// ClusterFlavor will take precedence over a namespaced Flavor
		// for setting instance_type in the metadata
		flavor := vmi.Annotations[v1.ClusterFlavorAnnotation]
		if flavor == "" {
			flavor = vmi.Annotations[v1.FlavorAnnotation]
		}

		if err := cloudinit.GenerateLocalData(vmi, flavor, cloudInitData); err != nil {
			return fmt.Errorf("generating local cloud-init data failed: %v", err)
		}
	}

	logger.Info("Executing PrepareNetwork on VMI pod environment")
	// Setup the networking (phase #2)
	if err := c.PrepareNetwork(vmi); err != nil {
		return err
	}

	return nil
}

// The post start hook assumes the VM is running.
// Takes care of running two redirection goroutines between the hardcoded socket
// path expected by virt-handler and the PTY device provided by Cloud Hypervisor.
// This makes the serial console accessible from virtctl console <vmi-name>
func (c *CloudHvDomainManager) postStartHook(vmi *v1.VirtualMachineInstance) error {
	logger := log.Log.Object(vmi)

	logger.Info("Executing PostStartHook on VMI pod environment")

	if c.vmConfig.Serial == nil || c.vmConfig.Serial.Mode != "Pty" {
		return nil
	}

	vmInfo, err := c.getVmInfo()
	if err != nil {
		return err
	}

	if vmInfo.Config.Serial != nil &&
		vmInfo.Config.Serial.Mode == "Pty" &&
		vmInfo.Config.Serial.File != nil &&
		*vmInfo.Config.Serial.File != "" {
		socketPath := fmt.Sprintf("/var/run/kubevirt-private/%s/virt-serial0", vmi.ObjectMeta.UID)
		listener, err := net.Listen("unix", socketPath)
		if err != nil {
			return err
		}

		ptyPath := *vmInfo.Config.Serial.File

		go func() {
			for {
				logger.Info("Socket wait for incoming connection")
				conn, err := listener.Accept()
				if err != nil {
					log.Log.Reason(err).Errorf("Failed accepting connection to %s", socketPath)
					return
				}
				logger.Info("Socket connection accepted")

				pty, err := os.OpenFile(ptyPath, os.O_RDWR, 0660)
				if err != nil {
					log.Log.Reason(err).Errorf("Failed opening PTY at %s", ptyPath)
					return
				}

				var wg sync.WaitGroup
				wg.Add(2)

				go func() {
					io.Copy(pty, conn)
					pty.Close()
					wg.Done()
				}()
				go func() {
					io.Copy(conn, pty)
					wg.Done()
				}()

				wg.Wait()
			}
		}()
	}

	return nil
}

func (c *CloudHvDomainManager) GetDomain() *api.Domain {
	return c.domain
}

func (c *CloudHvDomainManager) PrepareNetwork(vmi *v1.VirtualMachineInstance) error {
	if c.vmConfig.Net == nil {
		return nil
	}

	domain := api.Domain{}

	// Here we create a temporary domain for the sole purpose of configuring
	// the network. This is why we only create interfaces based on the list
	// of interfaces from the VMI.
	for _, iface := range vmi.Spec.Domain.Devices.Interfaces {
		if iface.Bridge == nil && iface.Masquerade == nil {
			continue
		}

		domainIface := api.Interface{
			Alias: api.NewUserDefinedAlias(iface.Name),
		}

		domain.Spec.Devices.Interfaces = append(domain.Spec.Devices.Interfaces, domainIface)
	}

	// The domain is going to be updated through this function. It means
	// MTU, MAC address and TAP interface name are going to be provisioned
	// through this step.
	if err := netsetup.NewVMNetworkConfigurator(vmi, cache.CacheCreator{}).SetupPodNetworkPhase2(&domain); err != nil {
		return fmt.Errorf("Failed preparing pod network: %v", err)
	}

	// Convert the information from the domain into the Cloud Hypervisor
	// VmConfig.
	for _, iface := range domain.Spec.Devices.Interfaces {
		for i, net := range *c.vmConfig.Net {
			if net.GetId() == iface.Alias.GetName() {
				if iface.Target != nil {
					(*c.vmConfig.Net)[i].SetTap(iface.Target.Device)
				}
				if iface.MAC != nil {
					(*c.vmConfig.Net)[i].SetMac(iface.MAC.MAC)
				}
			}
		}
	}

	return nil
}
