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

package virtwrap

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/sys/unix"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"libvirt.org/go/libvirt"
	"libvirt.org/go/libvirtxml"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/hibernation"
	hotplugdisk "kubevirt.io/kubevirt/pkg/storage/hotplug-disk"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
)

const (
	// defaultSavedStateChannels is the number of multifd channels writing and
	// reading the sparse (mapped-ram) save image in parallel.
	defaultSavedStateChannels = 4
	maxSavedStateChannels     = 64

	// sparseImageFormat makes QEMU write each RAM page at a fixed offset of
	// the save image (mapped-ram), skipping zero pages, which allows parallel
	// channels and direct I/O.
	sparseImageFormat = "sparse"

	// pageCacheDropInterval is how often the page cache of a save image
	// written through the page cache is written back and dropped, which keeps
	// it from piling up in the memory cgroup of the pod next to the guest.
	pageCacheDropInterval = 250 * time.Millisecond

	// abortRetryInterval is how often an abort of a save that took too long
	// is retried while the save job does not end.
	abortRetryInterval = 10 * time.Second
)

// SaveVMI saves the memory and device state of the domain to savePath in the
// background. libvirt stops the domain once its state is saved, and resumes
// it if saving fails. The outcome is reported in the domain metadata.
func (l *LibvirtDomainManager) saveVMI(vmi *v1.VirtualMachineInstance, savePath string) error {
	if !l.savingState.CompareAndSwap(false, true) {
		log.Log.Object(vmi).Info("Saving the domain state is already in progress")
		return nil
	}
	if metadata, exists := l.metadataCache.SavedState.Load(); exists && metadata.Operation == api.SavedStateOperationSave &&
		metadata.FileName == filepath.Base(savePath) && metadata.Completed {
		// Saved already, or failed: a failed save is reported, not retried.
		l.savingState.Store(false)
		log.Log.Object(vmi).Infof("The domain state was saved to %s already, failed: %t", savePath, metadata.Failed)
		return nil
	}

	now := metav1.Now()
	l.metadataCache.SavedState.Set(api.SavedStateMetadata{
		Operation:      api.SavedStateOperationSave,
		FileName:       filepath.Base(savePath),
		StartTimestamp: &now,
	})

	go func() {
		defer l.savingState.Store(false)
		err := l.saveDomain(vmi, savePath)
		if err != nil {
			log.Log.Object(vmi).Reason(err).Error("Saving the domain state failed")
		}
		l.setSavedStateResult(err)
	}()
	return nil
}

func (l *LibvirtDomainManager) saveDomain(vmi *v1.VirtualMachineInstance, savePath string) error {
	logger := log.Log.Object(vmi)
	dom, err := l.virConn.LookupDomainByName(api.VMINamespaceKeyFunc(vmi))
	if err != nil {
		return err
	}
	defer dom.Free()

	// A previous attempt may have left a partial image behind.
	if err := os.Remove(savePath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("failed to remove the previous save image: %w", err)
	}

	channels := savedStateChannels(vmi)
	params := libvirt.DomainSaveRestoreParams{
		FileSet:             true,
		File:                savePath,
		ImageFormatSet:      true,
		ImageFormat:         sparseImageFormat,
		ParallelChannelsSet: true,
		ParallelChannels:    channels,
	}
	flags := saveFlags(vmi)

	timeout := hibernation.SavingTimeout(vmi)
	logger.Infof("Saving the domain state to %s, format %s, %d channels, bypass cache %t, timeout %s",
		savePath, sparseImageFormat, channels, flags&libvirt.DOMAIN_SAVE_BYPASS_CACHE != 0, timeout)
	start := time.Now()

	// An aborted save fails, and libvirt resumes the guest it paused. The
	// abort is retried until the save ends.
	var (
		abortLock sync.Mutex
		saveEnded bool
		timedOut  bool
		timer     *time.Timer
	)
	var abort func()
	abort = func() {
		abortLock.Lock()
		defer abortLock.Unlock()
		if saveEnded {
			return
		}
		timedOut = true
		logger.Warningf("Saving the domain state did not complete within %s, aborting it", timeout)
		if err := dom.AbortJob(); err != nil {
			logger.Reason(err).Errorf("Failed to abort saving the domain state, retrying in %s", abortRetryInterval)
			timer = time.AfterFunc(abortRetryInterval, abort)
		}
	}
	timer = time.AfterFunc(timeout, abort)
	defer func() {
		// The domain handle is freed on return, the abort must not use it after.
		abortLock.Lock()
		defer abortLock.Unlock()
		saveEnded = true
		timer.Stop()
	}()
	if flags&libvirt.DOMAIN_SAVE_BYPASS_CACHE == 0 {
		stop := dropPageCacheWhileWriting(savePath, pageCacheDropInterval)
		defer stop()
	}

	err = dom.SaveParams(params, flags)
	if err != nil && isUnsupportedSaveParam(err) {
		// libvirt < 11.2 or QEMU without mapped-ram: fall back to the
		// sequential raw format, which cannot be read in parallel.
		logger.Reason(err).Warning("The sparse save image format is not supported, falling back to the raw format")
		err = dom.SaveParams(libvirt.DomainSaveRestoreParams{FileSet: true, File: savePath}, flags)
	}
	abortLock.Lock()
	aborted := timedOut
	abortLock.Unlock()
	if err != nil {
		if aborted {
			return fmt.Errorf("saving the domain state did not complete within %s and was aborted: %w", timeout, err)
		}
		if isNoSpace(err) {
			return fmt.Errorf("the saved state PVC is too small for the state (%s free): %w", freeSpace(filepath.Dir(savePath)), err)
		}
		return err
	}
	logger.Infof("Saved the domain state in %s", time.Since(start))
	return nil
}

// isNoSpace reports whether err is QEMU or libvirt running out of space.
func isNoSpace(err error) bool {
	return errors.Is(err, unix.ENOSPC) || strings.Contains(err.Error(), "No space left on device")
}

// freeSpace returns the space left on the file system of dir, for diagnostics.
func freeSpace(dir string) string {
	var stat unix.Statfs_t
	if err := unix.Statfs(dir, &stat); err != nil {
		return "unknown"
	}
	return resource.NewQuantity(int64(stat.Bavail)*int64(stat.Bsize), resource.BinarySI).String()
}

// dropPageCacheWhileWriting writes back and drops the page cache of path
// every interval until the returned function is called, and once more then.
// Pages QEMU writes through the page cache are charged to the memory cgroup
// of the pod, where they add up to the size of the guest RAM on top of the
// guest itself until they are written back.
func dropPageCacheWhileWriting(path string, interval time.Duration) func() {
	done := make(chan struct{})
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				dropPageCache(path)
				return
			case <-ticker.C:
				dropPageCache(path)
			}
		}
	}()
	return func() {
		close(done)
		<-finished
	}
}

// dropPageCache starts the write back of the dirty pages of path and drops
// its clean pages from the page cache. It is best effort.
func dropPageCache(path string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()
	fd := int(file.Fd())
	_ = unix.SyncFileRange(fd, 0, 0, unix.SYNC_FILE_RANGE_WRITE)
	_ = unix.Fadvise(fd, 0, 0, unix.FADV_DONTNEED)
}

// saveFlags returns the flags of a save or restore of vmi. By default the
// image bypasses the page cache (O_DIRECT), which keeps it out of the memory
// of the pod, sized for the guest only. Some storage, e.g. NFS, can be faster
// through the page cache: an annotation turns the bypass off for tuning.
func saveFlags(vmi *v1.VirtualMachineInstance) libvirt.DomainSaveRestoreFlags {
	value, ok := vmi.Annotations[v1.SavedStateBypassCacheAnnotation]
	if !ok {
		return libvirt.DOMAIN_SAVE_BYPASS_CACHE
	}
	bypass, err := strconv.ParseBool(value)
	if err != nil {
		log.Log.Object(vmi).Warningf("Ignoring invalid %s annotation %q", v1.SavedStateBypassCacheAnnotation, value)
		return libvirt.DOMAIN_SAVE_BYPASS_CACHE
	}
	if bypass {
		return libvirt.DOMAIN_SAVE_BYPASS_CACHE
	}
	return 0
}

func isUnsupportedSaveParam(err error) bool {
	var libvirtErr libvirt.Error
	if !errors.As(err, &libvirtErr) {
		return false
	}
	switch libvirtErr.Code {
	case libvirt.ERR_INVALID_ARG, libvirt.ERR_ARGUMENT_UNSUPPORTED, libvirt.ERR_CONFIG_UNSUPPORTED, libvirt.ERR_OPERATION_UNSUPPORTED:
		return true
	}
	return false
}

func (l *LibvirtDomainManager) setSavedStateResult(err error) {
	l.metadataCache.SavedState.WithSafeBlock(func(metadata *api.SavedStateMetadata, initialized bool) {
		if !initialized {
			return
		}
		now := metav1.Now()
		metadata.Completed = true
		metadata.EndTimestamp = &now
		metadata.Failed = err != nil
		if err != nil {
			metadata.FailureReason = err.Error()
		}
	})
}

// isSavingState reports whether a save holds the domain paused, which must
// not be undone by a sync.
func (l *LibvirtDomainManager) isSavingState() bool {
	return l.savingState.Load()
}

// savedStateChannels returns the number of parallel channels for the save
// image, tunable by an annotation.
func savedStateChannels(vmi *v1.VirtualMachineInstance) int {
	value, ok := vmi.Annotations[v1.SavedStateParallelChannelsAnnotation]
	if !ok {
		return defaultSavedStateChannels
	}
	channels, err := strconv.Atoi(value)
	if err != nil || channels < 1 || channels > maxSavedStateChannels {
		log.Log.Object(vmi).Warningf("Ignoring invalid %s annotation %q", v1.SavedStateParallelChannelsAnnotation, value)
		return defaultSavedStateChannels
	}
	return channels
}

// savedStateSourcePath returns the path of the state vmi resumes from, in the
// hot plugged saved state source volume.
func savedStateSourcePath(vmi *v1.VirtualMachineInstance) string {
	return filepath.Join(hotplugdisk.GetVolumeMountDir(hibernation.SourceVolumeName), filepath.Base(vmi.Spec.SavedStateSource.FileName))
}

// restoreDomain starts the defined domain from the state saved in the
// SavedStateSource of vmi instead of booting it. The domain definition
// generated for this pod is used, so that host specific details such as paths
// follow the new pod, while libvirt checks that its guest ABI matches the saved
// one.
func (l *LibvirtDomainManager) restoreDomain(vmi *v1.VirtualMachineInstance, dom cli.VirDomain) error {
	path := savedStateSourcePath(vmi)
	logger := log.Log.Object(vmi)

	savedXML, err := l.virConn.DomainSaveImageGetXMLDesc(path, libvirt.DOMAIN_SAVE_IMAGE_XML_SECURE)
	if err != nil {
		return fmt.Errorf("failed to read the saved state %s: %w", path, err)
	}
	domainXML, err := dom.GetXMLDesc(libvirt.DOMAIN_XML_SECURE | libvirt.DOMAIN_XML_INACTIVE)
	if err != nil {
		return err
	}
	restoreXML, err := restoreDomainXML(domainXML, savedXML)
	if err != nil {
		return err
	}

	// Resume paused whenever a boot would start paused, e.g. to pin the
	// emulator thread first. A later sync resumes it unless the user paused it.
	flags := saveFlags(vmi) | libvirt.DOMAIN_SAVE_RUNNING
	if getDomainCreateFlags(vmi)&libvirt.DOMAIN_START_PAUSED != 0 {
		flags = saveFlags(vmi) | libvirt.DOMAIN_SAVE_PAUSED
	}
	params := libvirt.DomainSaveRestoreParams{
		FileSet:             true,
		File:                path,
		DXMLSet:             true,
		DXML:                restoreXML,
		ParallelChannelsSet: true,
		ParallelChannels:    savedStateChannels(vmi),
	}
	logger.Infof("Restoring the domain from %s with %d channels", path, params.ParallelChannels)
	err = l.virConn.DomainRestoreParams(params, flags)
	if err != nil && isUnsupportedSaveParam(err) {
		// Parallel channels are only valid for a sparse image.
		logger.Reason(err).Warning("Restoring with parallel channels failed, retrying sequentially")
		params.ParallelChannelsSet = false
		params.ParallelChannels = 0
		err = l.virConn.DomainRestoreParams(params, flags)
	}
	if flags&libvirt.DOMAIN_SAVE_BYPASS_CACHE == 0 {
		// The state is used once, its pages only weigh on the memory cgroup
		// of the pod. Off the resume path, which is done.
		go dropPageCache(path)
	}
	return err
}

// restoreDomainXML returns the domain XML to restore with: the definition of
// this pod with the guest CPU, the vCPU list, the MAC addresses and the
// machine type of the saved state. With the default host-model CPU, the definition holds the
// model to be expanded on this node, while the saved state holds the CPU the
// guest actually runs on. With hotpluggable vCPUs, libvirt numbers the order
// the vCPUs were enabled in when the domain boots, and checks it on restore.
// An interface without a MAC address in the VMI spec gets the MAC of its pod
// interface, which differs in every pod: the guest keeps the MAC it runs
// with, as after a live migration.
func restoreDomainXML(domainXML, savedXML string) (string, error) {
	domain := &libvirtxml.Domain{}
	if err := domain.Unmarshal(domainXML); err != nil {
		return "", fmt.Errorf("failed to parse the domain XML: %w", err)
	}
	saved := &libvirtxml.Domain{}
	if err := saved.Unmarshal(savedXML); err != nil {
		return "", fmt.Errorf("failed to parse the saved domain XML: %w", err)
	}
	if domain.UUID != saved.UUID {
		return "", fmt.Errorf("the saved state belongs to domain %s, not %s", saved.UUID, domain.UUID)
	}
	domain.CPU = saved.CPU
	if saved.VCPUs != nil {
		domain.VCPUs = saved.VCPUs
	}
	if saved.Devices != nil && domain.Devices != nil {
		savedMACs := map[string]*libvirtxml.DomainInterfaceMAC{}
		for _, iface := range saved.Devices.Interfaces {
			if iface.Alias != nil && iface.MAC != nil {
				savedMACs[iface.Alias.Name] = iface.MAC
			}
		}
		for i, iface := range domain.Devices.Interfaces {
			if iface.Alias == nil {
				continue
			}
			if mac, ok := savedMACs[iface.Alias.Name]; ok {
				domain.Devices.Interfaces[i].MAC = mac
			}
		}
	}
	if domain.OS != nil && domain.OS.Type != nil && saved.OS != nil && saved.OS.Type != nil {
		domain.OS.Type.Machine = saved.OS.Type.Machine
	}
	return domain.Marshal()
}

// startFromSavedState restores the domain from the saved state of vmi. A
// failed restore is reported in the domain metadata and leaves the domain
// defined and stopped: it is never booted instead, since that would discard
// the saved state, and it is not retried either.
func (l *LibvirtDomainManager) startFromSavedState(vmi *v1.VirtualMachineInstance, dom cli.VirDomain) error {
	fileName := vmi.Spec.SavedStateSource.FileName
	if metadata, exists := l.metadataCache.SavedState.Load(); exists && metadata.Operation == api.SavedStateOperationRestore &&
		metadata.FileName == fileName && metadata.Completed && metadata.Failed {
		return fmt.Errorf("resuming from the saved state failed, the domain is not booted instead: %s", metadata.FailureReason)
	}

	now := metav1.Now()
	l.metadataCache.SavedState.Set(api.SavedStateMetadata{
		Operation:      api.SavedStateOperationRestore,
		FileName:       fileName,
		StartTimestamp: &now,
	})
	err := l.restoreDomain(vmi, dom)
	l.setSavedStateResult(err)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error("Restoring the domain from its saved state failed, the domain is not booted instead")
		return fmt.Errorf("resuming from the saved state failed: %w", err)
	}
	log.Log.Object(vmi).Info("Domain restored from its saved state")
	return nil
}
