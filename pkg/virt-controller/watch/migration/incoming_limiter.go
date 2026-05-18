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

package migration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"

	coordinationv1 "k8s.io/api/coordination/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	coordinationclientv1 "k8s.io/client-go/kubernetes/typed/coordination/v1"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
)

const (
	incomingMigrationLeaseNamespace       = "d8-virtualization"
	incomingMigrationLeaseNamePrefix      = "incoming-migration"
	incomingMigrationLeaseDurationSeconds = int32(300)

	incomingMigrationLimiterComponentValue = "inbound-migration-limiter"

	incomingMigrationComponentLabel      = "virtualization.deckhouse.io/component"
	incomingMigrationTargetNodeHashLabel = "virtualization.deckhouse.io/target-node-hash"
	incomingMigrationSlotIndexLabel      = "virtualization.deckhouse.io/slot-index"

	incomingMigrationTargetNodeAnnotation = "virtualization.deckhouse.io/target-node"
	incomingMigrationNamespaceAnnotation  = "virtualization.deckhouse.io/migration-namespace"
	incomingMigrationNameAnnotation       = "virtualization.deckhouse.io/migration-name"
	incomingMigrationUIDAnnotation        = "virtualization.deckhouse.io/migration-uid"

	TargetNodeIncomingMigrationLimitExceededReason  = "TargetNodeIncomingMigrationLimitExceeded"
	TargetNodeIncomingMigrationLimitExceededMessage = "Target node has no free inbound migration slots."
)

type IncomingMigrationLimiter interface {
	TryAcquire(ctx context.Context, migration *virtv1.VirtualMachineInstanceMigration, targetNode string, limit int) (bool, error)
	Release(ctx context.Context, migration *virtv1.VirtualMachineInstanceMigration, targetNode string, limit int) error
}

type LeaseIncomingMigrationLimiter struct {
	clientset kubecli.KubevirtClient
	namespace string
	now       func() metav1.MicroTime
}

func NewLeaseIncomingMigrationLimiter(clientset kubecli.KubevirtClient) *LeaseIncomingMigrationLimiter {
	return &LeaseIncomingMigrationLimiter{
		clientset: clientset,
		namespace: incomingMigrationLeaseNamespace,
		now: func() metav1.MicroTime {
			return metav1.MicroTime{Time: time.Now()}
		},
	}
}

func (l *LeaseIncomingMigrationLimiter) TryAcquire(ctx context.Context, migration *virtv1.VirtualMachineInstanceMigration, targetNode string, limit int) (bool, error) {
	if limit < 1 {
		limit = 1
	}

	slots := slotNames(targetNode, limit)
	for _, slot := range slots {
		lease, err := l.leases().Get(ctx, slot, metav1.GetOptions{})
		if k8serrors.IsNotFound(err) {
			continue
		}
		if err != nil {
			return false, err
		}
		if leaseIsHeldBy(lease, migration) {
			return true, l.renewLease(ctx, lease, migration, targetNode)
		}
	}

	for slotIndex, slot := range slots {
		acquired, err := l.tryAcquireSlot(ctx, migration, targetNode, slot, slotIndex)
		if k8serrors.IsConflict(err) || k8serrors.IsAlreadyExists(err) {
			continue
		}
		if err != nil {
			return false, err
		}
		if acquired {
			return true, nil
		}
	}

	return false, nil
}

func (l *LeaseIncomingMigrationLimiter) Release(ctx context.Context, migration *virtv1.VirtualMachineInstanceMigration, targetNode string, limit int) error {
	if limit < 1 {
		limit = 1
	}

	for _, slot := range slotNames(targetNode, limit) {
		lease, err := l.leases().Get(ctx, slot, metav1.GetOptions{})
		if k8serrors.IsNotFound(err) {
			continue
		}
		if err != nil {
			return err
		}
		if leaseIsHeldBy(lease, migration) {
			return ignoreNotFound(l.leases().Delete(ctx, lease.Name, metav1.DeleteOptions{}))
		}
	}

	selector := labels.SelectorFromSet(labels.Set{
		incomingMigrationComponentLabel:      incomingMigrationLimiterComponentValue,
		incomingMigrationTargetNodeHashLabel: targetNodeHash(targetNode),
	})
	leases, err := l.leases().List(ctx, metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		return err
	}
	for _, lease := range leases.Items {
		lease := lease
		if leaseIsHeldBy(&lease, migration) {
			if err := l.leases().Delete(ctx, lease.Name, metav1.DeleteOptions{}); err != nil && !k8serrors.IsNotFound(err) {
				return err
			}
		}
	}
	return nil
}

func (l *LeaseIncomingMigrationLimiter) tryAcquireSlot(ctx context.Context, migration *virtv1.VirtualMachineInstanceMigration, targetNode, slotName string, slotIndex int) (bool, error) {
	lease, err := l.leases().Get(ctx, slotName, metav1.GetOptions{})
	if k8serrors.IsNotFound(err) {
		_, err = l.leases().Create(ctx, l.newLease(migration, targetNode, slotName, slotIndex), metav1.CreateOptions{})
		return err == nil, err
	}
	if err != nil {
		return false, err
	}
	if leaseIsHeldBy(lease, migration) {
		return true, l.renewLease(ctx, lease, migration, targetNode)
	}

	active, err := l.holderMigrationIsActive(ctx, lease)
	if err != nil {
		return false, err
	}
	if active {
		return false, nil
	}

	return true, l.stealLease(ctx, lease, migration, targetNode, slotIndex)
}

func (l *LeaseIncomingMigrationLimiter) holderMigrationIsActive(ctx context.Context, lease *coordinationv1.Lease) (bool, error) {
	annotations := lease.Annotations
	namespace := annotations[incomingMigrationNamespaceAnnotation]
	name := annotations[incomingMigrationNameAnnotation]
	uid := annotations[incomingMigrationUIDAnnotation]
	if namespace == "" || name == "" || uid == "" {
		return false, nil
	}

	migration, err := l.clientset.VirtualMachineInstanceMigration(namespace).Get(ctx, name, metav1.GetOptions{})
	if k8serrors.IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if string(migration.UID) != uid {
		return false, nil
	}
	return !migration.IsFinal(), nil
}

func (l *LeaseIncomingMigrationLimiter) renewLease(ctx context.Context, lease *coordinationv1.Lease, migration *virtv1.VirtualMachineInstanceMigration, targetNode string) error {
	leaseCopy := lease.DeepCopy()
	l.setLeaseHolder(leaseCopy, migration, targetNode, slotIndexFromLease(leaseCopy))
	_, err := l.leases().Update(ctx, leaseCopy, metav1.UpdateOptions{})
	return err
}

func (l *LeaseIncomingMigrationLimiter) stealLease(ctx context.Context, lease *coordinationv1.Lease, migration *virtv1.VirtualMachineInstanceMigration, targetNode string, slotIndex int) error {
	leaseCopy := lease.DeepCopy()
	l.setLeaseHolder(leaseCopy, migration, targetNode, slotIndex)
	_, err := l.leases().Update(ctx, leaseCopy, metav1.UpdateOptions{})
	return err
}

func (l *LeaseIncomingMigrationLimiter) newLease(migration *virtv1.VirtualMachineInstanceMigration, targetNode, name string, slotIndex int) *coordinationv1.Lease {
	lease := &coordinationv1.Lease{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: l.namespace,
			Name:      name,
		},
	}
	l.setLeaseHolder(lease, migration, targetNode, slotIndex)
	return lease
}

func (l *LeaseIncomingMigrationLimiter) setLeaseHolder(lease *coordinationv1.Lease, migration *virtv1.VirtualMachineInstanceMigration, targetNode string, slotIndex int) {
	now := l.now()
	holder := migrationHolderIdentity(migration)
	if lease.Labels == nil {
		lease.Labels = map[string]string{}
	}
	lease.Labels[incomingMigrationComponentLabel] = incomingMigrationLimiterComponentValue
	lease.Labels[incomingMigrationTargetNodeHashLabel] = targetNodeHash(targetNode)
	lease.Labels[incomingMigrationSlotIndexLabel] = strconv.Itoa(slotIndex)

	if lease.Annotations == nil {
		lease.Annotations = map[string]string{}
	}
	lease.Annotations[incomingMigrationTargetNodeAnnotation] = targetNode
	lease.Annotations[incomingMigrationNamespaceAnnotation] = migration.Namespace
	lease.Annotations[incomingMigrationNameAnnotation] = migration.Name
	lease.Annotations[incomingMigrationUIDAnnotation] = string(migration.UID)

	lease.Spec.HolderIdentity = &holder
	lease.Spec.LeaseDurationSeconds = pointerInt32(incomingMigrationLeaseDurationSeconds)
	if lease.Spec.AcquireTime == nil {
		lease.Spec.AcquireTime = &now
	}
	lease.Spec.RenewTime = &now
}

func (l *LeaseIncomingMigrationLimiter) leases() coordinationclientv1.LeaseInterface {
	return l.clientset.CoordinationV1().Leases(l.namespace)
}

func slotNames(targetNode string, limit int) []string {
	result := make([]string, 0, limit)
	hash := targetNodeHash(targetNode)
	for i := 0; i < limit; i++ {
		result = append(result, fmt.Sprintf("%s-%s-%d", incomingMigrationLeaseNamePrefix, hash, i))
	}
	return result
}

func targetNodeHash(targetNode string) string {
	sum := sha256.Sum256([]byte(targetNode))
	return hex.EncodeToString(sum[:])[:16]
}

func migrationHolderIdentity(migration *virtv1.VirtualMachineInstanceMigration) string {
	return fmt.Sprintf("%s/%s/%s", migration.Namespace, migration.Name, migration.UID)
}

func leaseIsHeldBy(lease *coordinationv1.Lease, migration *virtv1.VirtualMachineInstanceMigration) bool {
	if lease.Spec.HolderIdentity != nil && *lease.Spec.HolderIdentity == migrationHolderIdentity(migration) {
		return true
	}
	annotations := lease.Annotations
	return annotations[incomingMigrationNamespaceAnnotation] == migration.Namespace &&
		annotations[incomingMigrationNameAnnotation] == migration.Name &&
		annotations[incomingMigrationUIDAnnotation] == string(migration.UID)
}

func slotIndexFromLease(lease *coordinationv1.Lease) int {
	if lease.Labels == nil {
		return 0
	}
	idx, err := strconv.Atoi(lease.Labels[incomingMigrationSlotIndexLabel])
	if err != nil {
		return 0
	}
	return idx
}

func pointerInt32(v int32) *int32 {
	return &v
}

func ignoreNotFound(err error) error {
	if k8serrors.IsNotFound(err) {
		return nil
	}
	return err
}
