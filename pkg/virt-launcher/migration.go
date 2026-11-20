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

package virtlauncher

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

type domainLister interface {
	ListAllDomains() ([]*api.Domain, error)
}

// WaitForMigrationSourceCleanup keeps compute and hook containers alive after
// QEMU migrates away. virt-handler can defer undefining the persistent source
// domain until a same-IP target Pod is ready. Keep the command server and notifier up
// while waiting for that acknowledgement, with a timeout longer than the handler's
// one-minute readiness wait so handler failure cannot strand the launcher.
func WaitForMigrationSourceCleanup(vmi *v1.VirtualMachineInstance, domains domainLister, stopCh <-chan struct{}) {
	const cleanupTimeout = 90 * time.Second
	if err := waitForMigrationSourceCleanup(vmi, domains, stopCh, time.Second, cleanupTimeout); err != nil {
		log.Log.Object(vmi).Reason(err).Warning("Stopped waiting for migration source cleanup")
	}
}

func waitForMigrationSourceCleanup(vmi *v1.VirtualMachineInstance, domains domainLister,
	stopCh <-chan struct{}, interval, timeout time.Duration,
) error {
	waitingForMigration := false
	return wait.PollUntilContextTimeout(wait.ContextForChannel(stopCh), interval, timeout, true, func(context.Context) (bool, error) {
		list, err := domains.ListAllDomains()
		if err != nil {
			if waitingForMigration {
				// A transient libvirt error is not the handler's acknowledgement.
				log.Log.V(3).Object(vmi).Reason(err).Info("Retrying migration source cleanup observation")
				return false, nil
			}
			return false, err
		}
		for _, domain := range list {
			if domain.Spec.Metadata.KubeVirt.UID != vmi.UID {
				continue
			}
			if domain.Status.Status == api.Shutoff && domain.Status.Reason == api.ReasonMigrated {
				waitingForMigration = true
				return false, nil
			}
			// PID monitoring can observe QEMU's exit before libvirt has updated
			// the migration's terminal state. Do not let that race release hooks.
			migration := domain.Spec.Metadata.KubeVirt.Migration
			if migration != nil && !migration.Failed &&
				(domain.Status.Status == api.Running || domain.Status.Status == api.Paused || domain.Status.Status == api.Shutdown) {
				waitingForMigration = true
				return false, nil
			}
		}
		return true, nil
	})
}
