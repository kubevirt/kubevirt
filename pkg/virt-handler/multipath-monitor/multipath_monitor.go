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
 */

package multipath_monitor

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/safepath"
	"kubevirt.io/kubevirt/pkg/storage/reservation"
	"kubevirt.io/kubevirt/pkg/virt-handler/filewatcher"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
	virt_chroot "kubevirt.io/kubevirt/pkg/virt-handler/virt-chroot"
)

const (
	socket = "multipathd.socket"
)

//go:generate mockgen -source $GOFILE -package=$GOPACKAGE -destination=generated_mock_$GOFILE
type mounter interface {
	Mount(sourcePath, targetPath *safepath.Path, ro bool) *exec.Cmd
	Umount(path *safepath.Path) *exec.Cmd
	IsMounted(mountPoint *safepath.Path) (isMounted bool, err error)
}

type mountManager struct{}

func (m *mountManager) Mount(sourcePath, targetPath *safepath.Path, ro bool) *exec.Cmd {
	return virt_chroot.MountChroot(sourcePath, targetPath, ro)
}

func (m *mountManager) Umount(path *safepath.Path) *exec.Cmd {
	return virt_chroot.UmountChroot(path)
}

func (m *mountManager) IsMounted(mountPoint *safepath.Path) (isMounted bool, err error) {
	return isolation.IsMounted(mountPoint)
}

type MultipathSocketMonitor struct {
	Watcher             *filewatcher.FileWatcher
	MultipathSocketPath string
	HostDir             string
	Mounter             mounter
	done                chan struct{}
}

func NewMultipathSocketMonitor() *MultipathSocketMonitor {
	const procRootRun = "/proc/1/root/run"
	multipathSocketPath := filepath.Join(procRootRun, socket)
	return &MultipathSocketMonitor{
		Watcher:             filewatcher.New(multipathSocketPath, 1*time.Second),
		MultipathSocketPath: multipathSocketPath,
		HostDir:             reservation.GetPrHelperHostSocketDir(),
		Mounter:             &mountManager{},
	}
}

func (m *MultipathSocketMonitor) Run() {
	m.done = make(chan struct{})

	// Remove old bind mount if there was one
	if err := m.unmount(); err == nil {
		// Try to mount existing socket
		m.mount()
	}

	m.Watcher.Run()
	go m.run()
}

func (m *MultipathSocketMonitor) Close() {
	m.Watcher.Close()
	<-m.done
}

func (m *MultipathSocketMonitor) run() {
	defer func() {
		m.Watcher.Close()
		// Try to unmount socket on exit and ignore errors
		_ = m.unmount()
		close(m.done)
	}()

	log.Log.Infof("Starting to monitor the multipath socket")
	for {
		select {
		case event := <-m.Watcher.Events:
			switch event {
			case filewatcher.Create:
				fallthrough
			case filewatcher.InoChange:
				if err := m.unmount(); err == nil {
					m.mount()
				}
			case filewatcher.Remove:
				_ = m.unmount()
			}
		case err := <-m.Watcher.Errors:
			if err != nil {
				log.Log.Reason(err).Errorf("Error during monitoring multipath socket")
			}
		}
		if m.Watcher.IsClosed() {
			log.Log.Infof("Stopping to monitor the multipath socket")
			return
		}
	}
}

func (m *MultipathSocketMonitor) unmount() error {
	sPath, err := m.createSPath()
	if err != nil {
		return err
	}

	for {
		if isMounted, err := m.Mounter.IsMounted(sPath); err != nil {
			log.Log.Reason(err).Errorf("failed to get the state of the multipath socket bind mount %s", sPath)
			return err
		} else if !isMounted {
			break
		}

		if out, err := m.Mounter.Umount(sPath).CombinedOutput(); err != nil {
			log.Log.Reason(err).Errorf("failed to umount persistent reservation directory %s: %s", sPath, string(out))
			return err
		}

		log.Log.Infof("Removed bind mount for the multipath socket")
	}

	return nil
}

func (m *MultipathSocketMonitor) mount() {
	sPath, err := m.createSPath()
	if err != nil {
		return
	}

	sPathMultipathSocket, err := safepath.JoinAndResolveWithRelativeRoot(m.MultipathSocketPath)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to create the safepath for the multipath socket")
		return
	}

	if out, err := m.Mounter.Mount(sPathMultipathSocket, sPath, false).CombinedOutput(); err != nil {
		log.Log.Reason(err).Errorf("failed to create the multipath socket bind mount %s: %s", sPath, string(out))
		return
	}

	log.Log.Infof("Created bind mount for the multipath socket")
}

func (m *MultipathSocketMonitor) createSPath() (*safepath.Path, error) {
	if _, err := os.Stat(m.HostDir); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		if err := os.MkdirAll(m.HostDir, 0o755); err != nil {
			log.Log.Reason(err).Errorf("failed creating persistent reservation directory %s", m.HostDir)
			return nil, err
		}
		log.Log.Infof("Created persistent reservation directory %s", m.HostDir)
	}

	path := filepath.Join(m.HostDir, socket)
	if _, err := os.Stat(path); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			log.Log.Reason(err).Errorf("failed stat for persistent reservation socket %s", path)
			return nil, err
		}
		file, err := os.Create(path)
		if err != nil {
			log.Log.Reason(err).Errorf("failed creating persistent reservation socket %s", path)
			return nil, err
		}
		if err := file.Close(); err != nil {
			log.Log.Reason(err).Errorf("failed closing persistent reservation socket %s", path)
			return nil, err
		}
		log.Log.Infof("Created persistent reservation directory socket %s", path)
	}

	sPath, err := safepath.JoinAndResolveWithRelativeRoot("/", path)
	if err != nil {
		log.Log.Reason(err).Error("failed to create the safepath for the multipath socket mount")
		return nil, err
	}

	return sPath, nil
}
