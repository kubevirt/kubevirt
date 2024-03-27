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
 * Copyright 2024 The KubeVirt Contributors
 *
 */

package util

import (
	"sync"

	"k8s.io/client-go/util/workqueue"
)

type PendingQueue struct {
	queue     []string
	queueLock *sync.Mutex
}

func NewPendingQueue() *PendingQueue {
	return &PendingQueue{
		queue:     []string{},
		queueLock: &sync.Mutex{},
	}
}

func (pq *PendingQueue) GetQueue() []string {
	pq.queueLock.Lock()
	defer pq.queueLock.Unlock()
	return pq.queue
}

func (pq *PendingQueue) FlushTo(queue workqueue.TypedRateLimitingInterface[string]) {
	pq.queueLock.Lock()
	defer pq.queueLock.Unlock()
	for _, key := range pq.queue {
		queue.AddRateLimited(key)
	}
	pq.queue = []string{}
}

func (pq *PendingQueue) Add(item string) {
	pq.queueLock.Lock()
	defer pq.queueLock.Unlock()
	pq.queue = append(pq.queue, item)
}
