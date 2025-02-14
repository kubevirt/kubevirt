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
 * Copyright 2025 The KubeVirt Authors.
 *
 */

package virthandler

import (
	"context"
	"encoding/json"
	"errors"
	goerror "errors"
	"fmt"
	"path/filepath"
	"time"

	"libvirt.org/go/libvirtxml"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/controller"
	hostdisk "kubevirt.io/kubevirt/pkg/host-disk"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/util/migrations"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"

	launcher_clients "kubevirt.io/kubevirt/pkg/virt-handler/launcher-clients"
	migrationproxy "kubevirt.io/kubevirt/pkg/virt-handler/migration-proxy"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var errWaitingForTargetPorts = errors.New("waiting for target to publish migration ports")

type MigrationSourceController struct {
	*BaseController
	capabilities                *libvirtxml.Caps
	clientset                   kubecli.KubevirtClient
	queue                       workqueue.TypedRateLimitingInterface[string]
	launcherClients             launcher_clients.LauncherClientsManager
	migrationProxy              migrationproxy.ProxyManager
	podIsolationDetector        isolation.PodIsolationDetector
	recorder                    record.EventRecorder
	virtLauncherFSRunDirPattern string
	vmiExpectations             *controller.UIDTrackingControllerExpectations
}

func NewMigrationSourceController(
	recorder record.EventRecorder,
	clientset kubecli.KubevirtClient,
	host string,
	virtShareDir string,
	launcherClients launcher_clients.LauncherClientsManager,
	vmiInformer cache.SharedIndexInformer,
	domainInformer cache.SharedInformer,
	clusterConfig *virtconfig.ClusterConfig,
	podIsolationDetector isolation.PodIsolationDetector,
	migrationProxy migrationproxy.ProxyManager,
	capabilities *libvirtxml.Caps,
) (*MigrationSourceController, error) {

	baseCtrl, err := NewBaseController(
		host,
		vmiInformer,
		domainInformer,
		clusterConfig,
		podIsolationDetector,
	)
	if err != nil {
		return nil, err
	}

	queue := workqueue.NewTypedRateLimitingQueueWithConfig[string](
		workqueue.DefaultTypedControllerRateLimiter[string](),
		workqueue.TypedRateLimitingQueueConfig[string]{Name: "virt-handler-source"},
	)

	c := &MigrationSourceController{
		BaseController:              baseCtrl,
		capabilities:                capabilities,
		clientset:                   clientset,
		queue:                       queue,
		launcherClients:             launcherClients,
		migrationProxy:              migrationProxy,
		podIsolationDetector:        podIsolationDetector,
		recorder:                    recorder,
		virtLauncherFSRunDirPattern: "/proc/%d/root/var/run",
		vmiExpectations:             controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectations()),
	}

	_, err = vmiInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addDeleteFunc,
		UpdateFunc: c.updateFunc,
	})
	if err != nil {
		return nil, err
	}

	_, err = domainInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: c.updateDomainFunc,
	})
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (c *MigrationSourceController) hasTargetDetectedReadyDomain(vmi *v1.VirtualMachineInstance) (bool, int64) {
	// give the target node 60 seconds to discover the libvirt domain via the domain informer
	// before allowing the VMI to be processed. This closes the gap between the
	// VMI's status getting updated to reflect the new source node, and the domain
	// informer firing the event to alert the source node of the new domain.
	migrationTargetDelayTimeout := 60

	if vmi.Status.MigrationState == nil ||
		vmi.Status.MigrationState.EndTimestamp == nil {
		return false, int64(migrationTargetDelayTimeout)
	}

	if vmi.Status.MigrationState != nil &&
		vmi.Status.MigrationState.TargetNodeDomainDetected &&
		vmi.Status.MigrationState.TargetNodeDomainReadyTimestamp != nil {

		return true, 0
	}

	nowUnix := time.Now().UTC().Unix()
	migrationEndUnix := vmi.Status.MigrationState.EndTimestamp.Time.UTC().Unix()

	diff := nowUnix - migrationEndUnix

	if diff > int64(migrationTargetDelayTimeout) {
		return false, 0
	}

	timeLeft := int64(migrationTargetDelayTimeout) - diff

	enqueueTime := timeLeft
	if enqueueTime < 5 {
		enqueueTime = 5
	}

	// re-enqueue the key to ensure it gets processed again within the right time.
	c.queue.AddAfter(controller.VirtualMachineInstanceKey(vmi), time.Duration(enqueueTime)*time.Second)

	return false, timeLeft
}

func domainMigrated(domain *api.Domain) bool {
	return domain != nil && domain.Status.Status == api.Shutoff && domain.Status.Reason == api.ReasonMigrated
}

func (c *MigrationSourceController) setMigrationProgressStatus(vmi *v1.VirtualMachineInstance, domain *api.Domain) {
	if domain == nil ||
		domain.Spec.Metadata.KubeVirt.Migration == nil ||
		vmi.Status.MigrationState == nil {
		return
	}

	migrationMetadata := domain.Spec.Metadata.KubeVirt.Migration
	if migrationMetadata.UID != vmi.Status.MigrationState.MigrationUID {
		return
	}

	vmi.Status.MigrationState.StartTimestamp = migrationMetadata.StartTimestamp

	vmi.Status.MigrationState.Failed = migrationMetadata.Failed

	if migrationMetadata.Failed {
		vmi.Status.MigrationState.EndTimestamp = migrationMetadata.EndTimestamp
		vmi.Status.MigrationState.FailureReason = migrationMetadata.FailureReason
		c.recorder.Event(vmi, k8sv1.EventTypeWarning, v1.Migrated.String(), fmt.Sprintf("VirtualMachineInstance migration uid %s failed. reason:%s", string(migrationMetadata.UID), migrationMetadata.FailureReason))
	}

	vmi.Status.MigrationState.AbortStatus = v1.MigrationAbortStatus(migrationMetadata.AbortStatus)
	if migrationMetadata.AbortStatus == string(v1.MigrationAbortSucceeded) {
		vmi.Status.MigrationState.EndTimestamp = migrationMetadata.EndTimestamp
	}

	vmi.Status.MigrationState.Mode = migrationMetadata.Mode
}

func (c *MigrationSourceController) updateStatus(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {
	c.setMigrationProgressStatus(vmi, domain)

	// handle migrations differently than normal status updates.
	//
	// When a successful migration is detected, we must transfer ownership of the VMI
	// from the source node (this node) to the target node (node the domain was migrated to).
	//
	// Transfer ownership by...
	// 1. Marking vmi.Status.MigrationState as completed
	// 2. Update the vmi.Status.NodeName to reflect the target node's name
	// 3. Update the VMI's NodeNameLabel annotation to reflect the target node's name
	// 4. Clear the LauncherContainerImageVersion which virt-controller will detect
	//    and accurately based on the version used on the target pod
	//
	// After a migration, the VMI's phase is no longer owned by this node. Only the
	// MigrationState status field is eligible to be mutated.
	migrationHost := ""
	if vmi.Status.MigrationState != nil {
		migrationHost = vmi.Status.MigrationState.TargetNode
	}

	targetNodeDetectedDomain, timeLeft := c.hasTargetDetectedReadyDomain(vmi)
	// If we can't detect where the migration went to, then we have no
	// way of transferring ownership. The only option here is to move the
	// vmi to failed.  The cluster vmi controller will then tear down the
	// resulting pods.
	if migrationHost == "" {
		// migrated to unknown host.
		vmi.Status.Phase = v1.Failed
		vmi.Status.MigrationState.Completed = true
		vmi.Status.MigrationState.Failed = true

		log.Log.Object(vmi).Warning("the vmi migrated to an unknown host")
		c.recorder.Event(vmi, k8sv1.EventTypeWarning, v1.Migrated.String(), fmt.Sprintf("The VirtualMachineInstance migrated to unknown host."))
	} else if !targetNodeDetectedDomain {
		if timeLeft <= 0 {
			vmi.Status.Phase = v1.Failed
			vmi.Status.MigrationState.Completed = true
			vmi.Status.MigrationState.Failed = true

			log.Log.Object(vmi).Warning("the domain was never observed on the taget after the migration completed within the timeout period")
			c.recorder.Event(vmi, k8sv1.EventTypeWarning, v1.Migrated.String(), fmt.Sprintf("The VirtualMachineInstance's domain was never observed on the target after the migration completed within the timeout period."))
		}
	}

	return nil
}

func (c *MigrationSourceController) Run(threadiness int, stopCh chan struct{}) {
	defer c.queue.ShutDown()
	log.Log.Info("Starting virt-handler source controller.")

	cache.WaitForCacheSync(stopCh, c.hasSynced)

	// queue keys for previous Domains on the host that no longer exist
	// in the cache. This ensures we perform local cleanup of deleted VMs.
	for _, domain := range c.domainStore.List() {
		d := domain.(*api.Domain)
		vmiRef := v1.NewVMIReferenceWithUUID(
			d.ObjectMeta.Namespace,
			d.ObjectMeta.Name,
			d.Spec.Metadata.KubeVirt.UID)

		key := controller.VirtualMachineInstanceKey(vmiRef)

		_, exists, _ := c.vmiStore.GetByKey(key)
		if !exists {
			c.queue.Add(key)
		}
	}

	// Start the actual work
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh
	log.Log.Info("Stopping virt-handler source controller.")
}

func (c *MigrationSourceController) runWorker() {
	for c.Execute() {
	}
}

func (c *MigrationSourceController) Execute() bool {
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	defer c.queue.Done(key)
	if err := c.execute(key); err != nil {
		log.Log.Reason(err).Infof("re-enqueuing VirtualMachineInstance %v", key)
		c.queue.AddRateLimited(key)
	} else {
		log.Log.V(4).Infof("processed VirtualMachineInstance %v", key)
		c.queue.Forget(key)
	}
	return true
}

func (c *MigrationSourceController) sync(key string, vmi *v1.VirtualMachineInstance, domain *api.Domain) error {
	if domain != nil {
		log.Log.Object(vmi).Infof("VMI is in phase: %v | Domain status: %v, reason: %v", vmi.Status.Phase, domain.Status.Status, domain.Status.Reason)
	} else {
		log.Log.Object(vmi).Infof("VMI is in phase: %v", vmi.Status.Phase)
	}

	oldStatus := vmi.DeepCopy().Status

	syncErr := c.processVMI(vmi, domain)

	if syncErr != nil {
		c.recorder.Event(vmi, k8sv1.EventTypeWarning, v1.SyncFailed.String(), syncErr.Error())
		// `syncErr` will be propagated anyway, and it will be logged in `re-enqueueing`
		// so there is no need to log it twice in hot path without increased verbosity.
		log.Log.Object(vmi).Reason(syncErr).Error("Synchronizing the VirtualMachineInstance failed.")
	}

	updateErr := c.updateStatus(vmi, domain)

	if updateErr != nil {
		log.Log.Object(vmi).Reason(updateErr).Error("Updating the migration status failed.")
	}

	// update the VMI if necessary
	if !equality.Semantic.DeepEqual(oldStatus, vmi.Status) {
		key := controller.VirtualMachineInstanceKey(vmi)
		c.vmiExpectations.SetExpectations(key, 1, 0)
		_, err := c.clientset.VirtualMachineInstance(vmi.ObjectMeta.Namespace).Update(context.Background(), vmi, metav1.UpdateOptions{})
		if err != nil {
			c.vmiExpectations.LowerExpectations(key, 1, 0)
			return err
		}
	}

	if syncErr != nil {
		return syncErr
	}

	if updateErr != nil {
		return updateErr
	}

	log.Log.Object(vmi).V(4).Info("Source synchronization loop succeeded.")
	return nil

}

func (c *MigrationSourceController) execute(key string) error {
	vmi, vmiExists, err := c.getVMIFromCache(key)
	if err != nil {
		return err
	}

	if !vmiExists || vmi.IsFinal() || vmi.DeletionTimestamp != nil {
		log.Log.V(4).Infof("vmi for key %v is terminating, final or does not exists", key)
		return nil
	}

	if !c.vmiExpectations.SatisfiedExpectations(key) {
		log.Log.V(4).Object(vmi).Info("waiting for expectations to be satisfied")
		return nil
	}

	domain, domainExists, _, err := c.getDomainFromCache(key)
	if err != nil {
		return err
	}

	if domainExists && domain.Spec.Metadata.KubeVirt.UID != vmi.UID {
		log.Log.V(4).Object(vmi).Infof("Detected stale vmi %s that still needs cleanup before new vmi with identical name/namespace can be processed", vmi.UID)
		return nil
	}

	if vmi.Status.MigrationState == nil {
		log.Log.V(4).Object(vmi).Info("no migration is in progress")
		return nil
	}

	// post migration clean up
	if isMigrationDone(vmi.Status.MigrationState) {
		c.migrationProxy.StopSourceListener(string(vmi.UID))
		return nil
	}

	if !c.isMigrationSource(vmi) {
		log.Log.Object(vmi).V(4).Info("not a migration source")
		return nil
	}

	return c.sync(key, vmi.DeepCopy(), domain)
}

func (c *MigrationSourceController) isMigrationSource(vmi *v1.VirtualMachineInstance) bool {

	if vmi.Status.MigrationState != nil &&
		vmi.Status.NodeName == c.host &&
		vmi.Status.MigrationState.SourceNode == c.host {

		return true
	}
	return false

}

func (c *MigrationSourceController) handleSourceMigrationProxy(vmi *v1.VirtualMachineInstance) error {

	res, err := c.podIsolationDetector.Detect(vmi)
	if err != nil {
		return err
	}
	// the migration-proxy is no longer shared via host mount, so we
	// pass in the virt-launcher's baseDir to reach the unix sockets.
	baseDir := fmt.Sprintf(filepath.Join(c.virtLauncherFSRunDirPattern, "kubevirt"), res.Pid())
	if vmi.Status.MigrationState.TargetDirectMigrationNodePorts == nil {
		return errWaitingForTargetPorts
	}

	err = c.migrationProxy.StartSourceListener(
		string(vmi.UID),
		vmi.Status.MigrationState.TargetNodeAddress,
		vmi.Status.MigrationState.TargetDirectMigrationNodePorts,
		baseDir,
	)
	if err != nil {
		return err
	}

	return nil
}

func (c *MigrationSourceController) migrateVMI(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {
	isUnresponsive, isInitialized, err := c.launcherClients.IsLauncherClientUnresponsive(vmi)
	if err != nil {
		return err
	}
	if !isInitialized {
		log.Log.Object(vmi).V(4).Info("launcher client is not initialized")
		c.queue.AddAfter(controller.VirtualMachineInstanceKey(vmi), time.Second*1)
		return nil
	} else if isUnresponsive {
		return goerror.New(fmt.Sprintf("Can not update a VirtualMachineInstance with unresponsive command server."))
	}

	client, err := c.launcherClients.GetLauncherClient(vmi)
	if err != nil {
		return fmt.Errorf(unableCreateVirtLauncherConnectionFmt, err)
	}

	if vmi.Status.MigrationState.AbortRequested {
		err = c.handleMigrationAbort(vmi, client)
		return err
	}

	if isMigrationInProgress(vmi, domain) {
		// we already started this migration, no need to rerun this
		log.Log.Object(vmi).V(4).Infof("migration %s has already been started", vmi.Status.MigrationState.MigrationUID)
		return nil
	}

	err = c.handleSourceMigrationProxy(vmi)
	if errors.Is(err, errWaitingForTargetPorts) {
		log.Log.Object(vmi).V(4).Info("waiting for target node to publish migration ports")
		c.queue.AddAfter(controller.VirtualMachineInstanceKey(vmi), 1*time.Second)
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to handle migration proxy: %v", err)
	}

	migrationConfiguration := vmi.Status.MigrationState.MigrationConfiguration
	if migrationConfiguration == nil {
		migrationConfiguration = c.clusterConfig.GetMigrationConfiguration()
	}

	options := &cmdclient.MigrationOptions{
		Bandwidth:               *migrationConfiguration.BandwidthPerMigration,
		ProgressTimeout:         *migrationConfiguration.ProgressTimeout,
		CompletionTimeoutPerGiB: *migrationConfiguration.CompletionTimeoutPerGiB,
		UnsafeMigration:         *migrationConfiguration.UnsafeMigrationOverride,
		AllowAutoConverge:       *migrationConfiguration.AllowAutoConverge,
		AllowPostCopy:           *migrationConfiguration.AllowPostCopy,
		AllowWorkloadDisruption: *migrationConfiguration.AllowWorkloadDisruption,
	}

	configureParallelMigrationThreads(options, vmi)

	marshalledOptions, err := json.Marshal(options)
	if err != nil {
		log.Log.Object(vmi).Warning("failed to marshall matched migration options")
	} else {
		log.Log.Object(vmi).Infof("migration options matched for vmi %s: %s", vmi.Name, string(marshalledOptions))
	}

	vmiCopy := vmi.DeepCopy()
	err = hostdisk.ReplacePVCByHostDisk(vmiCopy)
	if err != nil {
		return err
	}

	err = client.MigrateVirtualMachine(vmiCopy, options)
	if err != nil {
		return err
	}
	c.recorder.Event(vmi, k8sv1.EventTypeNormal, v1.Migrating.String(), VMIMigrating)
	return nil
}

func isMigrationDone(state *v1.VirtualMachineInstanceMigrationState) bool {
	return state == nil || (state.EndTimestamp != nil && (state.Completed || state.Failed))
}

func (c *MigrationSourceController) processVMI(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {
	domainAlive := domain != nil &&
		domain.Status.Status != api.Shutoff &&
		domain.Status.Status != api.Crashed &&
		domain.Status.Status != ""

	if !domainAlive {
		log.Log.V(4).Object(vmi).Info("domain is not alive")
		return nil
	}

	return c.migrateVMI(vmi, domain)
}

func (c *MigrationSourceController) addDeleteFunc(obj interface{}) {
	key, err := controller.KeyFunc(obj)
	if err == nil {
		c.vmiExpectations.LowerExpectations(key, 1, 0)
		c.queue.Add(key)
	}
}

func (c *MigrationSourceController) updateFunc(_, new interface{}) {
	key, err := controller.KeyFunc(new)
	if err == nil {
		c.vmiExpectations.LowerExpectations(key, 1, 0)
		c.queue.Add(key)
	}
}

func (c *MigrationSourceController) addDomainFunc(obj interface{}) {
	key, err := controller.KeyFunc(obj)
	if err == nil {
		c.queue.Add(key)
	}
}

func (c *MigrationSourceController) updateDomainFunc(old, new interface{}) {
	key, err := controller.KeyFunc(new)
	if err == nil {
		c.queue.Add(key)
	}
}

func (c *MigrationSourceController) handleMigrationAbort(vmi *v1.VirtualMachineInstance, client cmdclient.LauncherClient) error {
	if vmi.Status.MigrationState.AbortStatus == v1.MigrationAbortInProgress || vmi.Status.MigrationState.AbortStatus == v1.MigrationAbortSucceeded {
		return nil
	}

	if err := client.CancelVirtualMachineMigration(vmi); err != nil {
		if err.Error() == migrations.CancelMigrationFailedVmiNotMigratingErr {
			// If migration did not even start there is no need to cancel it
			log.Log.Object(vmi).Infof("skipping migration cancellation since vmi is not migrating")
		}
		return err
	}

	c.recorder.Event(vmi, k8sv1.EventTypeNormal, v1.Migrating.String(), VMIAbortingMigration)
	return nil
}

func configureParallelMigrationThreads(options *cmdclient.MigrationOptions, vm *v1.VirtualMachineInstance) {
	// When the CPU is limited, there's a risk of the migration threads choking the CPU resources on the compute container.
	// For this reason, we will avoid configuring migration threads in such scenarios.
	if cpuLimit, cpuLimitExists := vm.Spec.Domain.Resources.Limits[k8sv1.ResourceCPU]; cpuLimitExists && !cpuLimit.IsZero() {
		return
	}

	options.ParallelMigrationThreads = pointer.P(parallelMultifdMigrationThreads)
}
