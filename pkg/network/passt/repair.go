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
 *
 */

package passt

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"kubevirt.io/client-go/log"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/network/vmispec"
)

type clusterConfigurer interface {
	GetNetworkBindings() map[string]v1.InterfaceBindingPlugin
}

type activeGuard interface {
	TestAndSetActive(instance *v1.VirtualMachineInstance) bool
	SetInactive(instance *v1.VirtualMachineInstance)
}

type RepairManager struct {
	activeVMs            activeGuard
	clusterConfigurer    clusterConfigurer
	createSymlinkFunc    func(string) (string, error)
	findRepairSocketFunc func(string) (string, error)
	execCommandFunc      func(string, *v1.VirtualMachineInstance, func(instance *v1.VirtualMachineInstance))
}

func NewRepairManager(clusterConfigurer clusterConfigurer) *RepairManager {
	return NewRepairManagerWithOptions(
		clusterConfigurer,
		createShortenedSymlink,
		findRepairSocketInDir,
		executePasstRepair,
		newActiveVMProvider(),
	)
}

func NewRepairManagerWithOptions(
	clusterConfigurer clusterConfigurer,
	createSymlinkFunc,
	findRepairSocketFunc func(string) (string, error),
	execCommandFunc func(string, *v1.VirtualMachineInstance, func(instance *v1.VirtualMachineInstance)),
	activeVMs activeGuard,
) *RepairManager {
	return &RepairManager{
		activeVMs:            activeVMs,
		clusterConfigurer:    clusterConfigurer,
		createSymlinkFunc:    createSymlinkFunc,
		findRepairSocketFunc: findRepairSocketFunc,
		execCommandFunc:      execCommandFunc,
	}
}

func (r *RepairManager) HandleMigrationSource(vmi *v1.VirtualMachineInstance,
	dirFunc func(*v1.VirtualMachineInstance) (string, error),
) error {
	if !shouldRunPasstRepair(vmi, r.clusterConfigurer.GetNetworkBindings()) {
		return nil
	}

	if r.activeVMs.TestAndSetActive(vmi) {
		return nil
	}

	passtDir, err := dirFunc(vmi)
	if err != nil {
		return err
	}

	symlinkDir, err := r.createSymlinkFunc(passtDir)
	if err != nil {
		return err
	}

	repairSocket, err := r.findRepairSocketFunc(symlinkDir)
	if err != nil {
		return err
	}
	r.execCommandFunc(repairSocket, vmi, r.activeVMs.SetInactive)

	return nil
}

func (r *RepairManager) HandleMigrationTarget(vmi *v1.VirtualMachineInstance,
	dirFunc func(*v1.VirtualMachineInstance) (string, error),
) error {
	if !shouldRunPasstRepair(vmi, r.clusterConfigurer.GetNetworkBindings()) {
		return nil
	}

	if r.activeVMs.TestAndSetActive(vmi) {
		return nil
	}

	passtDir, err := dirFunc(vmi)
	if err != nil {
		return err
	}

	symlinkDir, err := r.createSymlinkFunc(passtDir)
	if err != nil {
		return err
	}
	r.execCommandFunc(symlinkDir, vmi, r.activeVMs.SetInactive)
	return nil
}

func shouldRunPasstRepair(vmi *v1.VirtualMachineInstance, registeredPlugins map[string]v1.InterfaceBindingPlugin) bool {
	podNetwork := vmispec.LookUpDefaultNetwork(vmi.Spec.Networks)
	if podNetwork == nil {
		return false
	}

	iface := vmispec.LookupInterfaceByName(vmi.Spec.Domain.Devices.Interfaces, podNetwork.Name)
	if iface == nil {
		return false
	}

	binding := iface.Binding
	if binding == nil {
		return false
	}

	registeredPlugin, exists := registeredPlugins[binding.Name]
	if !exists || registeredPlugin.DomainAttachmentType != "" {
		return false
	}

	return true
}

func executePasstRepair(arg string, vmi *v1.VirtualMachineInstance, setInactive func(instance *v1.VirtualMachineInstance)) {
	go func() {
		defer setInactive(vmi)

		const passtRepairEnfocedTimeout = 60 * time.Second
		ctx, cancel := context.WithTimeout(context.Background(), passtRepairEnfocedTimeout)
		defer cancel()

		const passtRepairBinaryName = "passt-repair"
		passtRepairCommand := exec.CommandContext(ctx, passtRepairBinaryName, arg)

		const debugLevel = 6
		log.Log.V(debugLevel).Infof("executing passt-repair : %s", passtRepairCommand.String())

		if stdOutErr, err := passtRepairCommand.CombinedOutput(); err != nil {
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				log.Log.Errorf("deadline exceeded running: %s, %q, %s", passtRepairCommand.String(), context.DeadlineExceeded, stdOutErr)
				return
			}
			log.Log.Errorf("failed to run %s, %v, %s", passtRepairCommand.String(), err, stdOutErr)
			return
		}
		log.Log.V(debugLevel).Infof("execution of: %s has completed", passtRepairCommand.String())
	}()
}

func createShortenedSymlink(inputPath string) (string, error) {
	cleanPath := filepath.Clean(inputPath)
	parts := strings.Split(cleanPath, string(filepath.Separator))

	if len(parts) < 3 || parts[1] != "pods" {
		return "", fmt.Errorf("invalid input path %q", inputPath)
	}

	dir := filepath.Join(string(filepath.Separator), parts[1], parts[2])
	link := filepath.Join(dir, "p")

	const dirPermissions = 0o755
	if err := os.MkdirAll(dir, dirPermissions); err != nil {
		return "", fmt.Errorf("mkdir %q: %w", dir, err)
	}
	if err := os.Symlink(inputPath, link); err != nil && !os.IsExist(err) {
		return "", fmt.Errorf("symlink %q→%q: %w", link, inputPath, err)
	}
	return link, nil
}

func findRepairSocketInDir(dirPath string) (string, error) {
	const passtRepairSocketSuffix = ".socket.repair"

	pattern := filepath.Join(dirPath, "*"+passtRepairSocketSuffix)
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return "", fmt.Errorf("glob %q: %w", pattern, err)
	}
	if len(matches) > 0 {
		return matches[0], nil
	}
	return "", fmt.Errorf("passt-repair socket not found in %s", dirPath)
}
