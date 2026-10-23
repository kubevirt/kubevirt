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

package admission

import (
	"sync"
	"time"
)

const utilizationSampleInterval = 500 * time.Millisecond

// UtilizationReader reports cgroup CPU utilization as a percentage of the pod's
// configured CPU limit.
type UtilizationReader interface {
	Utilization() (cpuPercent float64, ok bool)
}

var utilizationReader UtilizationReader = newPlatformUtilizationReader()

// cachedUtilizationReader periodically samples a base reader so soft admission
// does not perform synchronous cgroup sysfs reads on every request.
type cachedUtilizationReader struct {
	base UtilizationReader

	mu         sync.RWMutex
	cpuPercent float64
	ok         bool
	started    bool
	stopCh     chan struct{}
}

func newCachedUtilizationReader(base UtilizationReader) *cachedUtilizationReader {
	return &cachedUtilizationReader{base: base}
}

func (c *cachedUtilizationReader) Utilization() (cpuPercent float64, ok bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.cpuPercent, c.ok
}

func (c *cachedUtilizationReader) refresh() {
	cpu, ok := c.base.Utilization()
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cpuPercent = cpu
	c.ok = ok
}

// Start begins background sampling. Safe to call once; subsequent calls are no-ops.
func (c *cachedUtilizationReader) Start() {
	c.mu.Lock()
	if c.started {
		c.mu.Unlock()
		return
	}
	c.started = true
	c.stopCh = make(chan struct{})
	c.mu.Unlock()

	c.refresh()
	go func() {
		ticker := time.NewTicker(utilizationSampleInterval)
		defer ticker.Stop()
		for {
			select {
			case <-c.stopCh:
				return
			case <-ticker.C:
				c.refresh()
			}
		}
	}()
}

func (c *cachedUtilizationReader) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.started || c.stopCh == nil {
		return
	}
	close(c.stopCh)
	c.stopCh = nil
	c.started = false
}

// StartUtilizationSampler starts periodic cgroup sampling for soft admission.
// Call once during process startup.
func StartUtilizationSampler() {
	if cached, ok := utilizationReader.(*cachedUtilizationReader); ok {
		cached.Start()
		return
	}
	cached := newCachedUtilizationReader(utilizationReader)
	utilizationReader = cached
	cached.Start()
}

// Utilization returns the current CPU utilization percentage.
// ok is false when the CPU limit cannot be read (for example outside a cgroup).
func Utilization() (cpuPercent float64, ok bool) {
	return utilizationReader.Utilization()
}

// OverSoftUtilizationLimit reports whether CPU exceeds the soft utilization
// admission threshold.
func OverSoftUtilizationLimit() bool {
	cpu, ok := Utilization()
	if !ok {
		return false
	}
	return cpu > float64(SoftCPUUtilizationPercent)
}

// SetUtilizationReaderForTest replaces the utilization reader for unit tests.
func SetUtilizationReaderForTest(reader UtilizationReader) {
	if cached, ok := utilizationReader.(*cachedUtilizationReader); ok {
		cached.Stop()
	}
	utilizationReader = reader
}

// ResetUtilizationReaderForTest restores the default platform utilization reader.
func ResetUtilizationReaderForTest() {
	if cached, ok := utilizationReader.(*cachedUtilizationReader); ok {
		cached.Stop()
	}
	utilizationReader = newPlatformUtilizationReader()
}
