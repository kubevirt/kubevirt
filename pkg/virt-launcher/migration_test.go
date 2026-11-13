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
	"errors"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/types"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

type domainListFunc func() ([]*api.Domain, error)

func (f domainListFunc) ListAllDomains() ([]*api.Domain, error) { return f() }

func TestMigrationSourceWaitsForDomainCleanup(t *testing.T) {
	vmi := v1.NewVMIReferenceWithUUID("default", "vm", "uid")
	domain := api.NewMinimalDomainWithUUID("vm", vmi.UID)
	domain.Status.Status, domain.Status.Reason = api.Shutoff, api.ReasonMigrated
	calls := 0
	err := waitForMigrationSourceCleanup(vmi, domainListFunc(func() ([]*api.Domain, error) {
		calls++
		if calls < 3 {
			return []*api.Domain{domain}, nil
		}
		// The handler's DeleteDomain undefines the persistent migrated source.
		return nil, nil
	}), make(chan struct{}), time.Millisecond, time.Second)
	if err != nil || calls != 3 {
		t.Fatalf("launcher did not wait for source cleanup: calls=%d, err=%v", calls, err)
	}
}

func TestMigrationSourceWaitDoesNotDelayOtherDomains(t *testing.T) {
	for _, tc := range []struct {
		name   string
		uid    types.UID
		status api.LifeCycle
		reason api.StateChangeReason
	}{
		{"normal shutdown", "uid", api.Shutoff, api.ReasonShutdown},
		{"crash", "uid", api.Crashed, api.ReasonUnknown},
		{"running target", "uid", api.Running, api.ReasonMigrated},
		{"different VMI", "other-uid", api.Shutoff, api.ReasonMigrated},
	} {
		t.Run(tc.name, func(t *testing.T) {
			vmi := v1.NewVMIReferenceWithUUID("default", "vm", "uid")
			domain := api.NewMinimalDomainWithUUID("vm", vmi.UID)
			domain.Spec.Metadata.KubeVirt.UID = tc.uid
			domain.Status.Status, domain.Status.Reason = tc.status, tc.reason
			calls := 0
			err := waitForMigrationSourceCleanup(vmi, domainListFunc(func() ([]*api.Domain, error) {
				calls++
				return []*api.Domain{domain}, nil
			}), make(chan struct{}), time.Millisecond, time.Second)
			if err != nil || calls != 1 {
				t.Fatalf("unexpected wait: calls=%d, err=%v", calls, err)
			}
		})
	}
}

func TestMigrationSourceWaitIsBoundedAndCancellable(t *testing.T) {
	vmi := v1.NewVMIReferenceWithUUID("default", "vm", "uid")
	domain := api.NewMinimalDomainWithUUID("vm", vmi.UID)
	domain.Status.Status, domain.Status.Reason = api.Shutoff, api.ReasonMigrated
	for _, cancel := range []bool{false, true} {
		t.Run(map[bool]string{false: "timeout", true: "signal"}[cancel], func(t *testing.T) {
			stopCh := make(chan struct{})
			calls := 0
			err := waitForMigrationSourceCleanup(vmi, domainListFunc(func() ([]*api.Domain, error) {
				calls++
				if cancel && calls == 1 {
					close(stopCh)
				}
				return []*api.Domain{domain}, nil
			}), stopCh, time.Millisecond, 5*time.Millisecond)
			want := context.DeadlineExceeded
			if cancel {
				want = context.Canceled
			}
			if !errors.Is(err, want) {
				t.Fatalf("expected %v, got %v", want, err)
			}
		})
	}
}

func TestMigrationSourceWaitHandlesLibvirtError(t *testing.T) {
	vmi := v1.NewVMIReferenceWithUUID("default", "vm", "uid")
	want := errors.New("libvirt unavailable")
	err := waitForMigrationSourceCleanup(vmi, domainListFunc(func() ([]*api.Domain, error) {
		return nil, want
	}), make(chan struct{}), time.Millisecond, time.Second)
	if !errors.Is(err, want) {
		t.Fatalf("expected libvirt error, got %v", err)
	}
}

func TestMigrationSourceWaitRetriesTransientLibvirtErrors(t *testing.T) {
	vmi := v1.NewVMIReferenceWithUUID("default", "vm", "uid")
	domain := api.NewMinimalDomainWithUUID("vm", vmi.UID)
	domain.Status.Status, domain.Status.Reason = api.Shutoff, api.ReasonMigrated
	calls := 0
	err := waitForMigrationSourceCleanup(vmi, domainListFunc(func() ([]*api.Domain, error) {
		calls++
		switch calls {
		case 1:
			return []*api.Domain{domain}, nil
		case 2:
			return nil, errors.New("temporary libvirt failure")
		default:
			return nil, nil
		}
	}), make(chan struct{}), time.Millisecond, time.Second)
	if err != nil || calls != 3 {
		t.Fatalf("libvirt error released launcher before source cleanup: calls=%d, err=%v", calls, err)
	}
}

func TestMigrationSourceWaitsForLibvirtToObserveQEMUExit(t *testing.T) {
	vmi := v1.NewVMIReferenceWithUUID("default", "vm", "uid")
	domain := api.NewMinimalDomainWithUUID("vm", vmi.UID)
	domain.Spec.Metadata.KubeVirt.Migration = &api.MigrationMetadata{UID: "migration"}
	calls := 0
	err := waitForMigrationSourceCleanup(vmi, domainListFunc(func() ([]*api.Domain, error) {
		calls++
		switch calls {
		case 1:
			domain.Status.Status, domain.Status.Reason = api.Running, api.ReasonUnknown
		case 2:
			domain.Status.Status, domain.Status.Reason = api.Shutoff, api.ReasonMigrated
		default:
			return nil, nil
		}
		return []*api.Domain{domain}, nil
	}), make(chan struct{}), time.Millisecond, time.Second)
	if err != nil || calls != 3 {
		t.Fatalf("launcher exited before libvirt settled: calls=%d, err=%v", calls, err)
	}
}
