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
    "sync"
    "golang.org/x/sync/semaphore"
    "kubevirt.io/client-go/log"
)

type NodeMigrationLimiter struct {
    buckets    	map[string]*semaphore.Weighted
    limiterLock sync.Mutex
    maxPermits 	int64
}

func NewNodeMigrationLimiter(maxPermits int64) *NodeMigrationLimiter {
    return &NodeMigrationLimiter{
        buckets:    make(map[string]*semaphore.Weighted),
        maxPermits: maxPermits,
    }
}

// Acquire tries to get a permit for the node.
func (l *NodeMigrationLimiter) Acquire(nodeName string) bool {
    l.limiterLock.Lock()
    permits, exist := l.buckets[nodeName]
    if !exist {
        permits = semaphore.NewWeighted(l.maxPermits)
        l.buckets[nodeName] = permits
    }
    l.limiterLock.Unlock()
    return permits.TryAcquire(1)
}

// Release returns a permit to the node.
func (l *NodeMigrationLimiter) Release(nodeName string) {
    l.limiterLock.Lock()
    permits, exist := l.buckets[nodeName]
    l.limiterLock.Unlock()
    if exist {
        permits.Release(1)
    } else {
		log.Log.V(4).Infof("No permits to relaese for node: %s", nodeName)
	}
}

// Delete removes the bucket of permits of an inactive node.
func (l *NodeMigrationLimiter) Delete(nodeName string) {
    l.limiterLock.Lock()
    defer l.limiterLock.Unlock()
    delete(l.buckets, nodeName)
}

// UpdateMaxPermits will resize the number of permits per node and recreate the buckets if possible.
func (l *NodeMigrationLimiter) UpdateMaxPermits(newMax int64) {
    l.limiterLock.Lock()
    defer l.limiterLock.Unlock()
    if newMax == l.maxPermits {
        return
    }
    oldMax := l.maxPermits
    l.maxPermits = newMax
    log.Log.Infof("Updating max migration permits from %d to %d", oldMax, newMax)

    for node, permits := range l.buckets {
        // We can only resize if the bucket is empty
		// If there are ongoing migrations we will wait for completion
		// and the new max will be applied on the next Acquire
		// 
        if permits.TryAcquire(oldMax) {
            permits.Release(oldMax)
            l.buckets[node] = semaphore.NewWeighted(newMax)
        } else {
            log.Log.Warningf("The is an inprogress migration - cannot resize max permits for node %s", node)
        }
    }
}
