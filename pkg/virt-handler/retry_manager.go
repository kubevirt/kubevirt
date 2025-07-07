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
 * Copyright 2022 Red Hat, Inc.
 *
 */

package virthandler

import (
	"sync"
	"time"

	"kubevirt.io/client-go/log"
)

// FailRetryManager is the manger to handle asynchronous retry used by virt-handler.
// When a failure event happens, virt-handler will try to fix it. The result of the
// fix is reflected by whether we receive the failure signal again afterwards.
// If we did, we want to schedule the retry in an exponential backoff manner.
// This manager would remember the last try and return the wait time
// for next try. If we don't receive the failure signal again within the
// maxFailResponseTime, we consider the fix has worked and will reset the record.
type FailRetryManager struct {
	name string
	// initialWait is the initial wait time for a retry. The wait time will be
	// doubled afterwards for each retry until it reaches the maxWait time.
	initialWait time.Duration
	// maxWait is the max wait time between retries.
	maxWait time.Duration
	// maxFailResponseTime is the max time we expect to receive a failure signal
	// after a retry. If no failure signal is received after maxFailResponseTime
	// the retry is considered to have succeeded.
	maxFailResponseTime time.Duration

	stateLock   sync.Mutex
	retryStates map[string]*retryState
}

type retryState struct {
	firstFail     time.Time
	lastRun       time.Time
	nextRun       time.Time
	waitInterval  time.Duration
	lastRunFailed bool
}

// NewFailRetryManager creates a new FailRetryManager with the parameters explained above.
func NewFailRetryManager(name string, initialWait, maxWait, maxFailResponseTime time.Duration) *FailRetryManager {
	return &FailRetryManager{
		name:                name,
		maxFailResponseTime: maxFailResponseTime,
		initialWait:         initialWait,
		maxWait:             maxWait,
		retryStates:         make(map[string]*retryState),
	}
}

func (f *FailRetryManager) newState(now time.Time) *retryState {
	return &retryState{
		firstFail:     now,
		lastRun:       now,
		nextRun:       now.Add(f.initialWait),
		waitInterval:  f.initialWait,
		lastRunFailed: false,
	}
}

// ShouldDelay returns whether the retry on this failure should be delayed or not,
// and if true return the duration of the delay as well.
func (f *FailRetryManager) ShouldDelay(key string, isFailure func() bool) (bool, time.Duration) {
	if !isFailure() {
		log.Log.V(4).Infof("%s: Not a failure", key)
		return false, 0
	}

	now := time.Now()

	f.stateLock.Lock()
	defer f.stateLock.Unlock()

	state := f.retryStates[key]

	// When first failure occurs we set when should be the next run(`nextRun`) but do not delay
	// letting the VMI try to start again.
	// From this moment we wait for `maxFailResponseTime` amount of time.
	if state == nil {
		f.retryStates[key] = f.newState(now)
		log.Log.V(4).Infof("%s: First failure. Creating a new state and do not delay.", key)
		return false, 0
	}

	// `lastRunFailed` is a flag that allows us to join or separate different run/fail cycles.
	// When a failure occurs within a "maxFailResponseTime" amount of time, we consider it related to previous start,
	// and we start the backoff.
	if !state.lastRunFailed {
		// If another failure occurs during this waiting time we delay the enqueue
		// based on the difference between now and `nextRun`:
		//	- If now > nextRun 	=> we should not delay
		//	- If now <= nextRun => we need to wait (nextRun - now) amount of time before processing the vm again
		if state.lastRun.Add(f.maxFailResponseTime).After(now) {
			// This is a failure due to previous try.
			state.lastRunFailed = true
			log.Log.V(4).Infof("%s: Received failure withing %f seconds.", key, f.maxFailResponseTime.Seconds())
			log.Log.V(4).Infof("%s: Delaying: %t", key, state.nextRun.After(now))
			return state.nextRun.After(now), state.nextRun.Sub(now)
		} else {
			// This is a new failure. Reset the status.
			f.retryStates[key] = f.newState(now)
			log.Log.V(4).Infof("%s: New failure detected. Resetting the state and do not delay.", key)
			return false, 0
		}
	}

	// If this function has been triggered too early we delay the processing of the remaining backoff amount of time.
	if !now.After(state.nextRun) {
		log.Log.V(4).Infof("%s: Delaying vm processing for %f.", key, state.nextRun.Sub(now).Seconds())
		return true, state.nextRun.Sub(now)
	}

	// Backoff ended. Increase it and do not delay the processing.
	state.waitInterval = state.waitInterval * 2
	if state.waitInterval > f.maxWait {
		state.waitInterval = f.maxWait
	}
	state.nextRun = now.Add(state.waitInterval)
	state.lastRun = now
	state.lastRunFailed = false
	log.Log.V(4).Infof("%s: Backoff increased. New backoff time is %f", key, state.waitInterval.Seconds())
	return false, 0
}

// Run starts the manager.
func (f *FailRetryManager) Run(stopCh chan struct{}) {
	ticker := time.NewTicker(f.maxWait)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			func() {
				f.stateLock.Lock()
				defer f.stateLock.Unlock()
				for key, state := range f.retryStates {
					if !state.lastRunFailed && time.Now().After(state.lastRun.Add(f.maxFailResponseTime)) {
						log.Log.V(4).Infof("%s: Resetting the state", key)
						delete(f.retryStates, key)
					}
				}
			}()
		case <-stopCh:
			return
		}
	}
}
