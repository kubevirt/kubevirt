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

package virthandler

import (
	"sync"
	"time"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const (
	// vmStartPoolSize is how many VM starts may run SyncVMI at once.
	// The reconcile workers stay free to publish domain changes for other VMIs.
	// A 100ms stagger with a ~1.5s start keeps about 15 starts in flight.
	vmStartPoolSize = 32

	// vmStartCooldown stops a finished start from being submitted again
	// before the domain informer has reported the running domain.
	vmStartCooldown = time.Second

	// vmStartShutdownRetry is how long a delete waits while its start is still in the pool.
	vmStartShutdownRetry = 200 * time.Millisecond
)

type startPhase int

const (
	startIdle startPhase = iota
	// startWaiting is handed off but has not entered SyncVMI yet.
	// A delete may cancel it.
	startWaiting
	// startExecuting is inside SyncVMI. A delete must wait for it.
	startExecuting
	startSucceeded
	startFailed
)

type startRecord struct {
	phase    startPhase
	err      error
	finished time.Time
}

// vmStartTracker remembers, per VMI UID, whether a start is running in the pool,
// has succeeded and is waiting for the domain, or has failed and must be reported
// on the next reconcile.
type vmStartTracker struct {
	mu    sync.Mutex
	byUID map[string]*startRecord
}

func newVMStartTracker() *vmStartTracker {
	return &vmStartTracker{byUID: map[string]*startRecord{}}
}

func (t *vmStartTracker) inProgress(uid string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	rec := t.byUID[uid]
	return rec != nil && (rec.phase == startWaiting || rec.phase == startExecuting)
}

// begin marks uid as waiting for a pool slot. It returns false when a start
// for uid is already waiting or executing.
func (t *vmStartTracker) begin(uid string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if rec := t.byUID[uid]; rec != nil && (rec.phase == startWaiting || rec.phase == startExecuting) {
		return false
	}
	t.byUID[uid] = &startRecord{phase: startWaiting}
	return true
}

// beginExecution promotes a waiting start to SyncVMI.
// It returns false when the start was cancelled or is no longer waiting.
func (t *vmStartTracker) beginExecution(uid string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	rec := t.byUID[uid]
	if rec == nil || rec.phase != startWaiting {
		return false
	}
	rec.phase = startExecuting
	return true
}

// cancelWaiting drops a start that has not entered SyncVMI.
// It returns false when SyncVMI is already running and the caller must wait.
func (t *vmStartTracker) cancelWaiting(uid string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	rec := t.byUID[uid]
	if rec == nil || (rec.phase != startWaiting && rec.phase != startExecuting) {
		return true
	}
	if rec.phase == startExecuting {
		return false
	}
	delete(t.byUID, uid)
	return true
}

// drop forgets a start that took a slot and then found the VMI should not start.
func (t *vmStartTracker) drop(uid string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	rec := t.byUID[uid]
	if rec == nil || (rec.phase != startWaiting && rec.phase != startExecuting) {
		return
	}
	delete(t.byUID, uid)
}

func (t *vmStartTracker) finish(uid string, err error, now time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()
	rec := &startRecord{finished: now}
	if err != nil {
		rec.phase = startFailed
		rec.err = err
	} else {
		rec.phase = startSucceeded
	}
	t.byUID[uid] = rec
}

// takeError returns the error from a finished start and forgets it.
// A start that is still running, or that succeeded, is left in place.
func (t *vmStartTracker) takeError(uid string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	rec := t.byUID[uid]
	if rec == nil || rec.phase != startFailed {
		return nil
	}
	err := rec.err
	delete(t.byUID, uid)
	return err
}

func (t *vmStartTracker) cooldownRemaining(uid string, now time.Time) time.Duration {
	t.mu.Lock()
	defer t.mu.Unlock()
	rec := t.byUID[uid]
	if rec == nil || rec.phase != startSucceeded {
		return 0
	}
	elapsed := now.Sub(rec.finished)
	if elapsed >= vmStartCooldown {
		return 0
	}
	return vmStartCooldown - elapsed
}

func (t *vmStartTracker) clear(uid string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.byUID, uid)
}

// startRunner runs a VM start. Async runners return before fn finishes.
type startRunner interface {
	Async() bool
	Run(fn func())
}

// boundedStartLauncher runs at most n starts at the same time.
// Extra starts wait on the semaphore inside their own goroutine, not on a reconcile worker.
type boundedStartLauncher struct {
	sem chan struct{}
}

func newBoundedStartLauncher(n int) *boundedStartLauncher {
	return &boundedStartLauncher{sem: make(chan struct{}, n)}
}

func (l *boundedStartLauncher) Async() bool { return true }

func (l *boundedStartLauncher) Run(fn func()) {
	go func() {
		l.sem <- struct{}{}
		defer func() { <-l.sem }()
		fn()
	}()
}

// syncStartLauncher runs the start inline. Unit tests use it so SyncVMI still
// happens before execute returns.
type syncStartLauncher struct{}

func (syncStartLauncher) Async() bool { return false }

func (syncStartLauncher) Run(fn func()) { fn() }

func domainIsAlive(domain *api.Domain) bool {
	if domain == nil {
		return false
	}
	status := domain.Status.Status
	return status != api.Shutoff && status != api.Crashed && status != ""
}

// domainNeedsAsyncStart reports whether this reconcile would block a worker on a cold start.
// A running VMI, a final VMI, and a VMI that is already deleting stay on the worker.
func domainNeedsAsyncStart(vmi *v1.VirtualMachineInstance, domain *api.Domain) bool {
	if vmi.IsRunning() || vmi.IsFinal() || vmi.DeletionTimestamp != nil {
		return false
	}
	return !domainIsAlive(domain)
}
