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
 * Copyright 2024 The KubeVirt Authors.
 *
 */

package migration

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	k8score "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/utils/pointer"
	v1 "kubevirt.io/api/core/v1"
	virtv1 "kubevirt.io/api/core/v1"
	virtstoragev1alpha1 "kubevirt.io/api/storage/v1alpha1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"

	"kubevirt.io/kubevirt/pkg/controller"
)

type VolumeMigrationController struct {
	Queue                   workqueue.RateLimitingInterface
	clientset               kubecli.KubevirtClient
	volumeMigrationInformer cache.SharedIndexInformer
	vmiInformer             cache.SharedIndexInformer
	migrationInformer       cache.SharedIndexInformer
	vmInformer              cache.SharedIndexInformer
	pvcInformer             cache.SharedIndexInformer
	cdiInformer             cache.SharedIndexInformer
	cdiConfigInformer       cache.SharedIndexInformer
}

func NewVolumeMigrationController(clientset kubecli.KubevirtClient,
	volumeMigrationInformer cache.SharedIndexInformer,
	migrationInformer cache.SharedIndexInformer,
	vmiInformer cache.SharedIndexInformer,
	vmInformer cache.SharedIndexInformer,
	pvcInformer cache.SharedIndexInformer,
	cdiInformer cache.SharedIndexInformer,
	cdiConfigInformer cache.SharedIndexInformer,
) (*VolumeMigrationController, error) {
	c := &VolumeMigrationController{
		Queue:                   workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "virt-controller-volume-migration"),
		clientset:               clientset,
		volumeMigrationInformer: volumeMigrationInformer,
		vmiInformer:             vmiInformer,
		vmInformer:              vmInformer,
		migrationInformer:       migrationInformer,
		pvcInformer:             pvcInformer,
		cdiInformer:             cdiInformer,
		cdiConfigInformer:       cdiConfigInformer,
	}

	_, err := c.volumeMigrationInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addVolumeMigration,
		DeleteFunc: c.deleteVolumeMigration,
		UpdateFunc: c.updateVolumeMigration,
	})
	if err != nil {
		return nil, err
	}

	_, err = c.migrationInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addMigration,
		DeleteFunc: c.deleteMigration,
		UpdateFunc: c.updateMigration,
	})
	if err != nil {
		return nil, err
	}

	_, err = c.vmiInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addVMI,
		UpdateFunc: c.updateVMI,
	})
	if err != nil {
		return nil, err
	}

	_, err = c.vmInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addVM,
		UpdateFunc: c.updateVM,
	})
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (c *VolumeMigrationController) Run(threadiness int, stopCh <-chan struct{}) {
	defer controller.HandlePanic()
	defer c.Queue.ShutDown()
	log.Log.Info("Starting VolumeMigrationController controller.")

	// Wait for cache sync before we start the controller
	cache.WaitForCacheSync(stopCh,
		c.volumeMigrationInformer.HasSynced,
		c.vmiInformer.HasSynced,
		c.vmInformer.HasSynced,
		c.migrationInformer.HasSynced,
		c.pvcInformer.HasSynced,
		c.cdiInformer.HasSynced,
		c.cdiConfigInformer.HasSynced,
	)

	// Start the actual work
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh
	log.Log.Info("Stopping VolumeMigrationController controller.")
}

func (c *VolumeMigrationController) runWorker() {
	for c.Execute() {
	}
}

func (c *VolumeMigrationController) Execute() bool {
	key, quit := c.Queue.Get()
	if quit {
		return false
	}
	defer c.Queue.Done(key)
	if err := c.execute(key.(string)); err != nil {
		log.Log.Reason(err).Infof("re-enqueuing VolumeMigration %v", key)
		c.Queue.AddRateLimited(key)
	} else {
		log.Log.V(4).Infof("processed VolumeMigration %v", key)
		c.Queue.Forget(key)
	}
	return true
}

func (c *VolumeMigrationController) getVolumeMigrationPhase(volMig *virtstoragev1alpha1.VolumeMigration) virtstoragev1alpha1.VolumeMigrationPhase {
	volMigObj, exists, err := c.volumeMigrationInformer.GetStore().GetByKey(volMig.Namespace + "/" + volMig.Name)
	if err != nil || !exists {
		return virtstoragev1alpha1.VolumeMigrationPhaseUnknown
	}
	obj := volMigObj.(*virtstoragev1alpha1.VolumeMigration)

	return obj.Status.Phase
}

func (c *VolumeMigrationController) triggerVirtualMachineInstanceMigration(volMig *virtstoragev1alpha1.VolumeMigration, migVols []virtstoragev1alpha1.MigratedVolume, vmiName, migName, ns string) error {
	vmiObj, vmiExists, err := c.vmiInformer.GetStore().GetByKey(fmt.Sprintf("%s/%s", ns, vmiName))
	if err != nil {
		return err
	}
	if !vmiExists {
		return fmt.Errorf("VMI %s for the migration %s doesn't exist", vmiName, volMig.Name)
	}
	vmi := vmiObj.(*virtv1.VirtualMachineInstance)
	phase := c.getVolumeMigrationPhase(volMig)
	// Update the VMI status with the migrate volumes
	if err := c.updateVMIStatusWithMigratedDisksPatch(migVols, vmi, phase); err != nil {
		return err
	}

	// Create VirtualMachineiMigration object
	vmiMig := &virtv1.VirtualMachineInstanceMigration{
		ObjectMeta: metav1.ObjectMeta{
			Name:   migName,
			Labels: map[string]string{virtstoragev1alpha1.VolumeMigrationLabel: volMig.Name},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         virtstoragev1alpha1.SchemeGroupVersion.String(),
					Kind:               virtstoragev1alpha1.VolumeMigrationKind.String(),
					Name:               volMig.ObjectMeta.Name,
					UID:                volMig.ObjectMeta.UID,
					Controller:         pointer.BoolPtr(true),
					BlockOwnerDeletion: pointer.BoolPtr(true),
				},
			},
		},
		Spec: virtv1.VirtualMachineInstanceMigrationSpec{
			VMIName: vmiName,
		},
	}
	_, err = c.clientset.VirtualMachineInstanceMigration(ns).Create(vmiMig, &metav1.CreateOptions{})
	if err != nil {
		return err
	}
	return nil
}

func (c *VolumeMigrationController) updateStatusVolumeMigration(volMig *virtstoragev1alpha1.VolumeMigration,
	vmiMig *virtv1.VirtualMachineInstanceMigration) (*virtstoragev1alpha1.VolumeMigration, error) {
	var err error
	volMigCopy := volMig.DeepCopy()

	// Final phases of the volume migration
	if volMig.Status.Phase == virtstoragev1alpha1.VolumeMigrationPhaseSucceeded ||
		volMig.Status.Phase == virtstoragev1alpha1.VolumeMigrationPhaseFailed {
		return volMigCopy, nil
	}

	volMigCopy.Status.VirtualMachineMigrationName = pointer.StringPtr(vmiMig.Name)
	volMigCopy.Status.VirtualMachineInstanceName = pointer.StringPtr(vmiMig.Spec.VMIName)
	if vmiMig.Status.MigrationState != nil {
		if vmiMig.Status.MigrationState.StartTimestamp != nil {
			volMigCopy.Status.StartTimestamp = vmiMig.Status.MigrationState.StartTimestamp.DeepCopy()
		}
		if vmiMig.Status.MigrationState.EndTimestamp != nil {
			volMigCopy.Status.EndTimestamp = vmiMig.Status.MigrationState.EndTimestamp.DeepCopy()
		}
	}

	var phase virtstoragev1alpha1.VolumeMigrationPhase
	switch vmiMig.Status.Phase {
	case virtv1.MigrationFailed:
		phase = virtstoragev1alpha1.VolumeMigrationPhaseFailed
	case virtv1.MigrationSucceeded:
		phase = virtstoragev1alpha1.VolumeMigrationPhaseSucceeded
	case virtv1.MigrationScheduling, virtv1.MigrationScheduled,
		virtv1.MigrationPreparingTarget, virtv1.MigrationTargetReady:
		phase = virtstoragev1alpha1.VolumeMigrationPhaseScheduling
	case virtv1.MigrationRunning:
		phase = virtstoragev1alpha1.VolumeMigrationPhaseRunning
	default:
		phase = virtstoragev1alpha1.VolumeMigrationPhaseUnknown
	}
	setPhaseVolumeMigrationStatus(volMigCopy, phase)

	if equality.Semantic.DeepEqual(volMig, volMigCopy) {
		return volMig, nil
	}
	if volMigCopy, err = c.clientset.VolumeMigration(volMig.ObjectMeta.Namespace).UpdateStatus(context.Background(), volMigCopy, metav1.UpdateOptions{}); err != nil {
		return volMigCopy, fmt.Errorf("failed updating storage migration %s: %v", volMig.Name,
			err)
	}

	return volMigCopy, nil
}

func setPhaseVolumeMigrationStatus(volMig *virtstoragev1alpha1.VolumeMigration,
	phase virtstoragev1alpha1.VolumeMigrationPhase) {
	if volMig.Status.Phase == phase {
		return
	}
	volMig.Status.Phase = phase
	volMig.Status.PhaseTransitionTimestamps = append(volMig.Status.PhaseTransitionTimestamps,
		virtstoragev1alpha1.VolumeMigrationPhaseTransitionTimestamp{
			Phase:                    phase,
			PhaseTransitionTimestamp: metav1.Now(),
		})
}

func appendVolumeState(volMig *virtstoragev1alpha1.VolumeMigration, v *virtstoragev1alpha1.MigratedVolume,
	validation virtstoragev1alpha1.MigratedVolumeValidation, reason *string) {
	volMig.Status.VolumeMigrationStates = append(volMig.Status.VolumeMigrationStates,
		virtstoragev1alpha1.VolumeMigrationState{
			MigratedVolume: *v,
			Validation:     validation,
			Reason:         reason,
		})
}

func validateRejectedVolumes(volMig *virtstoragev1alpha1.VolumeMigration,
	vmi *virtv1.VirtualMachineInstance) *virtstoragev1alpha1.VolumeMigration {

	setFailed := false
	volMigCopy := volMig.DeepCopy()
	volMigCopy.Status.VolumeMigrationStates = []virtstoragev1alpha1.VolumeMigrationState{}
	srcVols := volumeMigrationSourceVolMapping(volMig)
	disks := storagetypes.GetDisksFromVolumes(vmi)
	filesystems := storagetypes.GetFilesystemsFromVolumes(vmi)
	for _, v := range vmi.Spec.Volumes {
		name := storagetypes.PVCNameFromVirtVolume(&v)
		vol, ok := srcVols[name]
		if !ok {
			continue
		}

		// Hotplugged volumes
		if storagetypes.IsHotplugVolume(&v) {
			appendVolumeState(volMigCopy, &vol,
				virtstoragev1alpha1.MigratedVolumeValidationRejected,
				pointer.StringPtr(virtstoragev1alpha1.ReasonRejectHotplugVolumes))
			setFailed = true
			continue
		}

		// Filesystems
		if _, ok := filesystems[v.Name]; ok {
			appendVolumeState(volMigCopy, &vol,
				virtstoragev1alpha1.MigratedVolumeValidationRejected,
				pointer.StringPtr(virtstoragev1alpha1.ReasonRejectFilesystemVolumes))
			setFailed = true
			continue
		}

		d, ok := disks[v.Name]
		if !ok {
			continue
		}

		// Shareable disks
		if d.Shareable != nil && *d.Shareable {
			appendVolumeState(volMigCopy, &vol,
				virtstoragev1alpha1.MigratedVolumeValidationRejected,
				pointer.StringPtr(virtstoragev1alpha1.ReasonRejectShareableVolumes))
			setFailed = true
			continue
		}

		// LUN disks
		if d.DiskDevice.LUN != nil {
			appendVolumeState(volMigCopy, &vol,
				virtstoragev1alpha1.MigratedVolumeValidationRejected,
				pointer.StringPtr(virtstoragev1alpha1.ReasonRejectLUNVolumes))
			setFailed = true
			continue
		}

		// If it reaches this point then the volume is valid
		appendVolumeState(volMigCopy, &vol,
			virtstoragev1alpha1.MigratedVolumeValidationValid, nil)
	}

	if setFailed {
		setPhaseVolumeMigrationStatus(volMigCopy,
			virtstoragev1alpha1.VolumeMigrationPhaseFailed)
	}

	return volMigCopy
}

func validatePendingVolumes(volMig *virtstoragev1alpha1.VolumeMigration,
	pendVols []virtstoragev1alpha1.MigratedVolume, multipleVMIs bool) *virtstoragev1alpha1.VolumeMigration {
	volMigCopy := volMig.DeepCopy()
	reason := virtstoragev1alpha1.ReasonRejectedPending
	if multipleVMIs {
		reason = virtstoragev1alpha1.ReasonRejectedMultipleVMIsAndPending
	}
	for _, v := range pendVols {
		appendVolumeState(volMigCopy, &v, virtstoragev1alpha1.MigratedVolumeValidationPending,
			pointer.StringPtr(reason))
	}
	if len(pendVols) > 0 {
		setPhaseVolumeMigrationStatus(volMigCopy,
			virtstoragev1alpha1.VolumeMigrationPhaseFailed)
	}

	return volMigCopy
}

func (c *VolumeMigrationController) validateAndUpdateVolumeMigrationMigrateVolumes(volMig *virtstoragev1alpha1.VolumeMigration,
	migrVolPerVMI map[string][]virtstoragev1alpha1.MigratedVolume, pendVols []virtstoragev1alpha1.MigratedVolume) error {
	volMigCopy := volMig.DeepCopy()

	// Reinitialized the VolumeMigrationStates
	volMigCopy.Status.VolumeMigrationStates = []virtstoragev1alpha1.VolumeMigrationState{}

	volMigCopy = validatePendingVolumes(volMig, pendVols, len(migrVolPerVMI) > 1)
	for vmiName, _ := range migrVolPerVMI {
		vmiObj, vmiExists, err := c.vmiInformer.GetStore().GetByKey(fmt.Sprintf("%s/%s", volMig.Namespace, vmiName))
		if err != nil {
			return err
		}
		if !vmiExists {
			return fmt.Errorf("VMI %s for the volume migration %s doesn't exist", vmiName, volMig.Name)
		}
		vmi := vmiObj.(*virtv1.VirtualMachineInstance)
		volMigCopy = validateRejectedVolumes(volMigCopy, vmi)
	}
	if equality.Semantic.DeepEqual(volMig, volMigCopy) {
		return nil
	}
	if _, err := c.clientset.VolumeMigration(volMig.ObjectMeta.Namespace).UpdateStatus(context.Background(), volMigCopy, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("failed updating storage migration %s: %v", volMig.Name,
			err)
	}

	return nil
}

func getClaimNameFromVMIStatus(name string, vmiStatus v1.VirtualMachineInstanceStatus) (string, error) {
	for _, vStatus := range vmiStatus.VolumeStatus {
		if vStatus.PersistentVolumeClaimInfo != nil && name == vStatus.Name {
			return vStatus.PersistentVolumeClaimInfo.ClaimName, nil
		}
	}
	return "", fmt.Errorf("claim name for volume %s not found", name)
}

func findVolumeName(vmi *virtv1.VirtualMachineInstance, claimName string) (string, error) {
	for _, v := range vmi.Spec.Volumes {
		name := storagetypes.PVCNameFromVirtVolume(&v)
		if claimName == name {
			return v.Name, nil
		}
	}
	return "", fmt.Errorf("failed to find corresponding volume to claim name %s", claimName)
}

func (c *VolumeMigrationController) patchVMIstatus(oldVMI, newVMI *virtv1.VirtualMachineInstance) error {
	if !equality.Semantic.DeepEqual(oldVMI.Status, newVMI.Status) {
		newState, err := json.Marshal(newVMI.Status)
		if err != nil {
			return err
		}

		oldState, err := json.Marshal(oldVMI.Status)
		if err != nil {
			return err
		}
		var ops []string
		ops = append(ops, fmt.Sprintf(`{ "op": "test", "path": "/status", "value": %s }`, string(oldState)))
		ops = append(ops, fmt.Sprintf(`{ "op": "replace", "path": "/status", "value": %s }`, string(newState)))
		_, err = c.clientset.VirtualMachineInstance(oldVMI.Namespace).Patch(context.Background(), oldVMI.Name,
			types.JSONPatchType, controller.GeneratePatchBytes(ops), &metav1.PatchOptions{})
		return err
	}

	return nil

}

func (c *VolumeMigrationController) clearMigratedVolumes(vmi *virtv1.VirtualMachineInstance) error {
	vmiCopy := vmi.DeepCopy()
	vmiCopy.Status.MigratedVolumes = []virtv1.StorageMigratedVolumeInfo{}

	return c.patchVMIstatus(vmi, vmiCopy)
}

func (c *VolumeMigrationController) updateVMIStatusWithVolumeMigrationPhase(vmi *virtv1.VirtualMachineInstance,
	phase virtstoragev1alpha1.VolumeMigrationPhase) error {
	vmiCopy := vmi.DeepCopy()
	for i, _ := range vmi.Status.MigratedVolumes {
		vmiCopy.Status.MigratedVolumes[i].MigrationPhase = &phase
	}

	return c.patchVMIstatus(vmi, vmiCopy)
}

func (c *VolumeMigrationController) updateVMIStatusWithMigratedDisksPatch(migratedVolumes []virtstoragev1alpha1.MigratedVolume,
	vmi *virtv1.VirtualMachineInstance, phase virtstoragev1alpha1.VolumeMigrationPhase) error {
	vmiCopy := vmi.DeepCopy()
	// Always reinitialized the migrated disks
	vmiCopy.Status.MigratedVolumes = []v1.StorageMigratedVolumeInfo{}
	for _, d := range migratedVolumes {
		// Getting information on the destination PVC
		pvcInterface, pvcExists, err := c.pvcInformer.GetStore().GetByKey(fmt.Sprintf("%s/%s", vmi.Namespace, d.DestinationClaim))
		if !pvcExists {
			return fmt.Errorf("failed getting information for the destination PVC %s: %v", d.DestinationClaim, err)
		}
		pvcDst := pvcInterface.(*k8score.PersistentVolumeClaim)
		filesystemOverheadDst, err := storagetypes.GetFilesystemOverheadInformers(c.cdiInformer, c.cdiConfigInformer, pvcDst)
		if err != nil {
			log.Log.Reason(err).Errorf("Failed to get filesystem overhead for PVC %s/%s", vmi.Namespace, d.DestinationClaim)
			return err
		}

		// Getting information on the source PVC
		pvcInterface, pvcExists, err = c.pvcInformer.GetStore().GetByKey(fmt.Sprintf("%s/%s", vmi.Namespace, d.SourceClaim))
		if !pvcExists {
			return fmt.Errorf("failed getting information for the source PVC %s: %v", d.SourceClaim, err)
		}
		pvcSrc := pvcInterface.(*k8score.PersistentVolumeClaim)
		filesystemOverheadSrc, err := storagetypes.GetFilesystemOverheadInformers(c.cdiInformer, c.cdiConfigInformer, pvcSrc)
		if err != nil {
			log.Log.Reason(err).Errorf("Failed to get filesystem overhead for PVC %s/%s", vmi.Namespace, d.SourceClaim)
			return err
		}
		var volName string
		if volName, err = findVolumeName(vmi, pvcSrc.Name); err != nil {
			return err
		}

		vmiCopy.Status.MigratedVolumes = append(vmiCopy.Status.MigratedVolumes,
			v1.StorageMigratedVolumeInfo{
				VolumeName: volName,
				DestinationPVCInfo: &virtv1.PersistentVolumeClaimInfo{
					ClaimName:          pvcDst.Name,
					AccessModes:        pvcDst.Spec.AccessModes,
					VolumeMode:         pvcDst.Spec.VolumeMode,
					Capacity:           pvcDst.Status.Capacity,
					Requests:           pvcDst.Spec.Resources.Requests,
					Preallocated:       storagetypes.IsPreallocated(pvcDst.ObjectMeta.Annotations),
					FilesystemOverhead: &filesystemOverheadDst,
				},
				SourcePVCInfo: &virtv1.PersistentVolumeClaimInfo{
					ClaimName:          pvcSrc.Name,
					AccessModes:        pvcSrc.Spec.AccessModes,
					VolumeMode:         pvcSrc.Spec.VolumeMode,
					Capacity:           pvcSrc.Status.Capacity,
					Requests:           pvcSrc.Spec.Resources.Requests,
					Preallocated:       storagetypes.IsPreallocated(pvcSrc.ObjectMeta.Annotations),
					FilesystemOverhead: &filesystemOverheadSrc,
				},
				MigrationPhase: &phase,
			})

	}

	return c.patchVMIstatus(vmi, vmiCopy)
}

// TODO: replace this function with errors.Join available from golang 1.20
func joinErrors(errors ...error) error {
	var err error
	for _, e := range errors {
		if e == nil {
			continue
		}
		if err == nil {
			err = e
		} else {
			err = fmt.Errorf("%s: %w", err.Error(), e)
		}
	}
	return err
}

func (c *VolumeMigrationController) cleanupVirtualMachineInstanceMigration(mig *virtv1.VirtualMachineInstanceMigration) error {
	var errRet error
	var podName string
	if mig.Status.MigrationState == nil {
		errRet = fmt.Errorf("migration %s has an empty state, cannot cleanup the target pod", mig.Name)
	} else {
		podName = mig.Status.MigrationState.TargetPod
	}
	if err := c.clientset.VirtualMachineInstanceMigration(mig.Namespace).Delete(mig.Name, &metav1.DeleteOptions{}); err != nil {
		errRet = joinErrors(errRet, err)
	}
	if podName == "" {
		return errRet
	}
	if err := c.clientset.CoreV1().Pods(mig.Namespace).Delete(context.TODO(), podName, metav1.DeleteOptions{}); err != nil {
		errRet = joinErrors(errRet, err)
	}
	return errRet
}

func (c *VolumeMigrationController) classifyVolumesPerVMI(volMig *virtstoragev1alpha1.VolumeMigration) (map[string][]virtstoragev1alpha1.MigratedVolume,
	[]virtstoragev1alpha1.MigratedVolume, error) {
	type destMigVol struct {
		name   string
		policy virtstoragev1alpha1.SourceReclaimPolicy
	}
	migrVolPerVMI := make(map[string][]virtstoragev1alpha1.MigratedVolume)
	var migrVolWithoutVMI []virtstoragev1alpha1.MigratedVolume

	vmiList, err := c.clientset.VirtualMachineInstance(volMig.Namespace).List(context.Background(), &metav1.ListOptions{})
	if err != nil {
		return migrVolPerVMI, migrVolWithoutVMI, fmt.Errorf("failed to get VMIs: %v", err)
	}
	vols := make(map[string]destMigVol)
	for _, volMigr := range volMig.Spec.MigratedVolume {
		vols[volMigr.SourceClaim] = destMigVol{
			name:   volMigr.DestinationClaim,
			policy: volMigr.SourceReclaimPolicy,
		}

	}

	// Group the migrated volume per VMI
	for _, vmi := range vmiList.Items {
		var migrVols []virtstoragev1alpha1.MigratedVolume
		for _, v := range vmi.Spec.Volumes {
			name := storagetypes.PVCNameFromVirtVolume(&v)
			if name == "" {
				continue
			}
			dst, ok := vols[name]
			if !ok {
				continue
			}
			migrVols = append(migrVols, virtstoragev1alpha1.MigratedVolume{
				SourceClaim:         name,
				DestinationClaim:    dst.name,
				SourceReclaimPolicy: dst.policy,
			})
			delete(vols, name)
		}
		if len(migrVols) > 0 {
			migrVolPerVMI[vmi.Name] = migrVols
		}
	}

	// The rest of the volumes in vols aren't associated to a VMI
	for k, v := range vols {
		migrVolWithoutVMI = append(migrVolWithoutVMI, virtstoragev1alpha1.MigratedVolume{
			SourceClaim:         k,
			DestinationClaim:    v.name,
			SourceReclaimPolicy: v.policy,
		})
	}

	return migrVolPerVMI, migrVolWithoutVMI, nil
}

func findFinalizer(checkFinalizer string, finalizers []string) bool {
	for _, f := range finalizers {
		if f == checkFinalizer {
			return true
		}
	}

	return false
}

func (c *VolumeMigrationController) setDeletionFinalizer(volMig *virtstoragev1alpha1.VolumeMigration) error {
	volMigCopy := volMig.DeepCopy()
	finalizers := volMigCopy.ObjectMeta.GetFinalizers()
	if !findFinalizer(virtstoragev1alpha1.VolumeMigrationFinalizer, finalizers) {
		finalizers = append(finalizers, virtstoragev1alpha1.VolumeMigrationFinalizer)
		volMigCopy.ObjectMeta.SetFinalizers(finalizers)
		if _, err := c.clientset.VolumeMigration(volMig.Namespace).Update(context.TODO(), volMigCopy, metav1.UpdateOptions{}); err != nil {
			return err
		}
	}

	return nil
}

func (c *VolumeMigrationController) deleteDeletionFinalizer(volMig *virtstoragev1alpha1.VolumeMigration) error {
	found := false
	volMigCopy := volMig.DeepCopy()
	finalizers := volMigCopy.ObjectMeta.GetFinalizers()
	var setFinalizers []string
	for _, f := range finalizers {
		if f == virtstoragev1alpha1.VolumeMigrationFinalizer {
			found = true
		} else {
			setFinalizers = append(setFinalizers, f)
		}
	}
	if found {
		volMigCopy.ObjectMeta.SetFinalizers(setFinalizers)
		if _, err := c.clientset.VolumeMigration(volMig.Namespace).Update(context.TODO(), volMigCopy, metav1.UpdateOptions{}); err != nil {
			return err
		}
	}

	return nil
}

func volMigrationStarted(volMig *virtstoragev1alpha1.VolumeMigration) bool {
	phase := volMig.Status.Phase
	return phase == virtstoragev1alpha1.VolumeMigrationPhaseRunning ||
		phase == virtstoragev1alpha1.VolumeMigrationPhaseSucceeded ||
		phase == virtstoragev1alpha1.VolumeMigrationPhaseFailed ||
		phase == virtstoragev1alpha1.VolumeMigrationPhaseScheduling
}

func (c *VolumeMigrationController) execute(key string) error {
	var err error
	obj, exists, err := c.volumeMigrationInformer.GetStore().GetByKey(key)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	volMig := obj.(*virtstoragev1alpha1.VolumeMigration)
	logger := log.Log.Object(volMig)
	logger.V(1).Infof("Processing volume migration: %s", volMig.Name)

	if volMig.DeletionTimestamp.IsZero() {
		if err := c.setDeletionFinalizer(volMig); err != nil {
			return err
		}
	} else {
		// The volume migration has been deleted then remove all the related
		// VMI migrations
		var errRet error
		vmiMigList, err := c.clientset.VirtualMachineInstanceMigration(volMig.Namespace).List(&metav1.ListOptions{
			LabelSelector: fmt.Sprintf("%s=%s", virtstoragev1alpha1.VolumeMigrationLabel, volMig.Name),
		})
		if err != nil {
			return fmt.Errorf("failed to get VMIs: %v", err)
		}
		for _, mig := range vmiMigList.Items {
			err := c.clientset.VirtualMachineInstanceMigration(volMig.Namespace).Delete(mig.Name, &metav1.DeleteOptions{})
			if err != nil {
				log.Log.Errorf("failed deleting VM migration %s:%v", mig.Name, err)
				errRet = joinErrors(errRet, err)
			} else {
				logger.V(1).Infof("deleted VM migration %s", mig.Name)

			}
		}
		if errRet != nil {
			return errRet
		}

		// Remove the finalizer
		return c.deleteDeletionFinalizer(volMig)
	}

	migrVolPerVMI, migVolWithoutVMI, err := c.classifyVolumesPerVMI(volMig)
	if err != nil {
		return err
	}
	if !volMigrationStarted(volMig) {
		if len(migVolWithoutVMI) > 0 {
			logger.Errorf("There are migrated volumes without a VMI: %v", migVolWithoutVMI)
		}
		if len(migrVolPerVMI) > 1 {
			logger.Errorf("There are migrated volumes associated to multiples VMIs")
		}
		if err := c.validateAndUpdateVolumeMigrationMigrateVolumes(volMig, migrVolPerVMI, migVolWithoutVMI); err != nil {
			return err
		}
	}
	for vmi, migVols := range migrVolPerVMI {
		if err = c.executeStorageMigPerVMI(volMig, migVols, vmi); err != nil {
			logger.Object(volMig).Reason(err).Errorf("Failed to migrate the storage for VMI %s: %v", vmi, err)
		}
	}

	return err
}

func (c *VolumeMigrationController) deleteDVOwner(claim, ns string) error {
	pvc, err := c.clientset.CoreV1().PersistentVolumeClaims(ns).Get(context.TODO(),
		claim, metav1.GetOptions{})
	if err != nil {
		return err
	}
	for _, owner := range pvc.ObjectMeta.OwnerReferences {
		if owner.Controller == nil {
			continue
		}
		if !*owner.Controller {
			continue
		}
		if owner.Kind == "DataVolume" && owner.APIVersion == cdiv1.SchemeGroupVersion.String() {
			// For a controller, the PVC can have a single owner
			return c.clientset.CdiClient().CdiV1beta1().DataVolumes(ns).Delete(context.Background(),
				claim, metav1.DeleteOptions{})
		}
	}

	return nil
}

func (c *VolumeMigrationController) handleSourceVolume(v *virtstoragev1alpha1.MigratedVolume, ns string) error {
	switch v.SourceReclaimPolicy {
	case virtstoragev1alpha1.SourceReclaimPolicyDelete:
		// Before deleting the PVC, delete the DV owner
		// if it exists
		err := c.deleteDVOwner(v.SourceClaim, ns)
		if err != nil {
			return err
		}
		err = c.clientset.CoreV1().PersistentVolumeClaims(ns).Delete(context.TODO(),
			v.SourceClaim, metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	case virtstoragev1alpha1.SourceReclaimPolicyRetain:
		// Do nothing for the retain policy
		return nil
	default:
		return fmt.Errorf("PVC policy '%s' not recognized", v.SourceReclaimPolicy)
	}

	return nil
}

func (c *VolumeMigrationController) getOwnerVM(vmi *virtv1.VirtualMachineInstance) *virtv1.VirtualMachine {
	controllerRef := metav1.GetControllerOf(vmi)
	if controllerRef == nil || controllerRef.Kind != virtv1.VirtualMachineGroupVersionKind.Kind {
		return nil
	}

	obj, exists, _ := c.vmInformer.GetStore().GetByKey(vmi.Namespace + "/" + controllerRef.Name)
	if !exists {
		return nil
	}

	ownerVM := obj.(*virtv1.VirtualMachine)

	if controllerRef.UID == ownerVM.UID {
		return ownerVM
	}

	return nil
}

func (c *VolumeMigrationController) checkUpdateMigratedVolumeUpdate(vmi *virtv1.VirtualMachineInstance) bool {
	if vmi == nil {
		return false
	}

	// Check if all the destination migrated volumes have been replaced in
	// the VMI spec
	migVols := volumeMigrationDestinationMapping(vmi)
	for _, v := range vmi.Spec.Volumes {
		name := storagetypes.PVCNameFromVirtVolume(&v)
		if _, ok := migVols[name]; ok {
			delete(migVols, name)
		}
	}
	if len(migVols) > 0 {
		return false
	}

	// If the VMI have a VM owner, then we need to control if the VM has
	// already been updated with the migrated volumes.
	vm := c.getOwnerVM(vmi)
	if vm == nil {
		return true
	}
	migVols = volumeMigrationDestinationMapping(vmi)
	for _, v := range vm.Spec.Template.Spec.Volumes {
		name := storagetypes.PVCNameFromVirtVolume(&v)
		if _, ok := migVols[name]; ok {
			delete(migVols, name)
		}
	}
	if len(migVols) > 0 {
		return false
	}

	return true
}

func (c *VolumeMigrationController) executeStorageMigPerVMI(volMig *virtstoragev1alpha1.VolumeMigration, migVols []virtstoragev1alpha1.MigratedVolume, vmiName string) error {
	var errRet error
	ns := volMig.Namespace
	vmiObj, vmiExists, err := c.vmiInformer.GetStore().GetByKey(fmt.Sprintf("%s/%s", ns, vmiName))
	if err != nil {
		return err
	}
	if !vmiExists {
		return fmt.Errorf("VMI %s for the volume migration %s doesn't exist", vmiName, volMig.Name)
	}
	vmi := vmiObj.(*virtv1.VirtualMachineInstance)

	logger := log.Log.Object(vmi)
	logger.V(1).Infof("Volume migration for volumes of VMI %s", vmi.Name)

	// Check if the migration has already been triggered
	migName := volMig.GetVirtualMachiheInstanceMigrationName(vmiName)
	migObj, exists, err := c.migrationInformer.GetStore().GetByKey(fmt.Sprintf("%s/%s", ns, migName))
	if err != nil {
		return err
	}
	// Start the migration if it doesn't exist
	if !exists {
		logger.V(1).Infof("Start VM migration %s for VMI %s", migName, vmi.Name)
		return c.triggerVirtualMachineInstanceMigration(volMig, migVols, vmiName, migName, ns)
	}

	mig := migObj.(*virtv1.VirtualMachineInstanceMigration)
	if _, err := c.updateStatusVolumeMigration(volMig, mig); err != nil {
		return err
	}
	phase := c.getVolumeMigrationPhase(volMig)
	if err := c.updateVMIStatusWithVolumeMigrationPhase(vmi, phase); err != nil {
		return err
	}

	if mig.Status.MigrationState != nil && mig.Status.MigrationState.Completed && !mig.Status.MigrationState.Failed {
		// Clean-up the virtual machine migration
		if err := c.cleanupVirtualMachineInstanceMigration(mig); err != nil {
			return err
		}
		if !c.checkUpdateMigratedVolumeUpdate(vmi) {
			return nil
		}
		// If the VM and VMIs have been update then, we can handle the source volumes
		for _, v := range migVols {
			err := c.handleSourceVolume(&v, ns)
			errRet = joinErrors(errRet, err)
		}
		if errRet != nil {
			return errRet
		}
		if err := c.clearMigratedVolumes(vmi); err != nil {
			return err
		}
	}

	return nil
}

func (c *VolumeMigrationController) enqueueVolumeMigration(obj interface{}) {
	logger := log.Log
	volMig := obj.(*virtstoragev1alpha1.VolumeMigration)
	key, err := controller.KeyFunc(volMig)
	if err != nil {
		logger.Object(volMig).Reason(err).Error("Failed to extract key from volume migration.")
		return
	}
	c.Queue.Add(key)
}

func (c *VolumeMigrationController) addVolumeMigration(obj interface{}) {
	c.enqueueVolumeMigration(obj)
}

func (c *VolumeMigrationController) deleteVolumeMigration(obj interface{}) {
	c.enqueueVolumeMigration(obj)
}

func (c *VolumeMigrationController) updateVolumeMigration(_, curr interface{}) {
	c.enqueueVolumeMigration(curr)
}

// checkAndEnqueuVolumeMigVMIMig enqueues the volume migration corresponding to
// the VMI migration
func (c *VolumeMigrationController) checkAndEnqueuVolumeMigVMIMig(obj interface{}) {
	mig := obj.(*virtv1.VirtualMachineInstanceMigration)
	volMigName, ok := mig.ObjectMeta.Labels[virtstoragev1alpha1.VolumeMigrationLabel]
	if !ok {
		return
	}
	volMigObj, exists, err := c.volumeMigrationInformer.GetStore().GetByKey(mig.Namespace + "/" + volMigName)
	if err != nil {
		return
	}
	if !exists {
		return
	}
	volMig := volMigObj.(*virtstoragev1alpha1.VolumeMigration)
	c.enqueueVolumeMigration(volMig)
}

func (c *VolumeMigrationController) addMigration(obj interface{}) {
	c.checkAndEnqueuVolumeMigVMIMig(obj)
}

func (c *VolumeMigrationController) deleteMigration(obj interface{}) {
	c.checkAndEnqueuVolumeMigVMIMig(obj)
}

func (c *VolumeMigrationController) updateMigration(_, curr interface{}) {
	c.checkAndEnqueuVolumeMigVMIMig(curr)
}

// checkAndEnqueuVolumeMigVMI enqueues the volume migration related to the VMI
func (c *VolumeMigrationController) checkAndEnqueuVolumeMigVMI(obj interface{}) {
	vmi := obj.(*virtv1.VirtualMachineInstance)

	objs := c.volumeMigrationInformer.GetStore().List()
	for _, o := range objs {
		volMig := o.(*virtstoragev1alpha1.VolumeMigration)
		if volMig.Namespace != vmi.Namespace {
			continue
		}
		if volMig.Status.VirtualMachineInstanceName != nil &&
			*volMig.Status.VirtualMachineInstanceName == vmi.Name {
			c.enqueueVolumeMigration(volMig)
			return
		}
	}
}

func (c *VolumeMigrationController) addVMI(obj interface{}) {
	c.checkAndEnqueuVolumeMigVMI(obj)
}

func (c *VolumeMigrationController) updateVMI(_, curr interface{}) {
	c.checkAndEnqueuVolumeMigVMI(curr)
}

// checkAndEnqueuVolumeMigVM enqueues the volume migration related to the VM
func (c *VolumeMigrationController) checkAndEnqueuVolumeMigVM(obj interface{}) {
	vm := obj.(*virtv1.VirtualMachine)

	objs := c.volumeMigrationInformer.GetStore().List()
	for _, o := range objs {
		volMig := o.(*virtstoragev1alpha1.VolumeMigration)
		if volMig.Namespace != vm.Namespace {
			continue
		}
		if volMig.Status.VirtualMachineInstanceName != nil &&
			*volMig.Status.VirtualMachineInstanceName == vm.Name {
			c.enqueueVolumeMigration(volMig)
			return
		}
	}
}

func (c *VolumeMigrationController) addVM(obj interface{}) {
	c.checkAndEnqueuVolumeMigVM(obj)
}

func (c *VolumeMigrationController) updateVM(_, curr interface{}) {
	c.checkAndEnqueuVolumeMigVM(curr)
}

func volumeMigrationSourceVolMapping(volMig *virtstoragev1alpha1.VolumeMigration) map[string]virtstoragev1alpha1.MigratedVolume {
	srcVols := make(map[string]virtstoragev1alpha1.MigratedVolume)
	for _, v := range volMig.Spec.MigratedVolume {
		srcVols[v.SourceClaim] = v
	}

	return srcVols
}

func volumeMigrationDestinationMapping(vmi *virtv1.VirtualMachineInstance) map[string]bool {
	migVols := make(map[string]bool)
	for _, v := range vmi.Status.MigratedVolumes {
		if v.DestinationPVCInfo == nil {
			continue
		}
		migVols[v.DestinationPVCInfo.ClaimName] = true
	}

	return migVols
}
