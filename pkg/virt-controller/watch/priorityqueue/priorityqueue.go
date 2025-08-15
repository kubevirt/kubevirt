package priorityqueue

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/btree"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/utils/clock"
	"k8s.io/utils/ptr"
)

// AddOpts describes the options for adding items to the queue.
type AddOpts struct {
	After       time.Duration
	RateLimited bool
	Priority    int
}

// PriorityQueue is a priority queue for a controller. It
// internally de-duplicates all items that are added to
// it. It will use the max of the passed priorities and the
// min of possible durations.
type PriorityQueue interface {
	workqueue.RateLimitingInterface
	AddWithOpts(o AddOpts, Items ...interface{})
	GetWithPriority() (item interface{}, priority int, shutdown bool)
}

// Opts contains the options for a PriorityQueue.
type Opts struct {
	// Ratelimiter is being used when AddRateLimited is called. Defaults to a per-item exponential backoff
	// limiter with an initial delay of five milliseconds and a max delay of 1000 seconds.
	RateLimiter    workqueue.RateLimiter
	MetricProvider workqueue.MetricsProvider
	Log            logr.Logger
}

// Opt allows to configure a PriorityQueue.
type Opt func(*Opts)

// New constructs a new PriorityQueue.
func New(name string, o ...Opt) PriorityQueue {
	opts := &Opts{}
	for _, f := range o {
		f(opts)
	}

	if opts.RateLimiter == nil {
		opts.RateLimiter = workqueue.NewItemExponentialFailureRateLimiter(5*time.Millisecond, 1000*time.Second)
	}

	if opts.MetricProvider == nil {
		opts.MetricProvider = workqueueMetricsProvider{}
	}

	pq := &priorityqueue{
		log:         opts.Log,
		items:       map[interface{}]*item{},
		queue:       btree.NewG(32, less),
		becameReady: sets.Set[interface{}]{},
		metrics:     newQueueMetrics(opts.MetricProvider, name, clock.RealClock{}),
		// itemOrWaiterAdded indicates that an item or
		// waiter was added. It must be buffered, because
		// if we currently process items we can't tell
		// if that included the new item/waiter.
		itemOrWaiterAdded: make(chan struct{}, 1),
		rateLimiter:       opts.RateLimiter,
		locked:            sets.Set[interface{}]{},
		done:              make(chan struct{}),
		get:               make(chan item),
		now:               time.Now,
		tick:              time.Tick,
	}

	go pq.spin()
	go pq.logState()
	if _, ok := pq.metrics.(noMetrics); !ok {
		go pq.updateUnfinishedWorkLoop()
	}

	return pq
}

type priorityqueue struct {
	log logr.Logger
	// lock has to be acquired for any access any of items, queue, addedCounter
	// or becameReady
	lock  sync.Mutex
	items map[interface{}]*item
	queue bTree[*item]

	// addedCounter is a counter of elements added, we need it
	// because unixNano is not guaranteed to be unique.
	addedCounter uint64

	// becameReady holds items that are in the queue, were added
	// with non-zero after and became ready. We need it to call the
	// metrics add exactly once for them.
	becameReady sets.Set[interface{}]
	metrics     queueMetrics

	itemOrWaiterAdded chan struct{}

	rateLimiter workqueue.RateLimiter

	// locked contains the keys we handed out through Get() and that haven't
	// yet been returned through Done().
	locked     sets.Set[interface{}]
	lockedLock sync.RWMutex

	shutdown atomic.Bool
	done     chan struct{}

	get chan item

	// waiters is the number of routines blocked in Get, we use it to determine
	// if we can push items.
	waiters atomic.Int64

	// Configurable for testing
	now  func() time.Time
	tick func(time.Duration) <-chan time.Time
}

func (w *priorityqueue) AddWithOpts(o AddOpts, items ...interface{}) {
	w.lock.Lock()
	defer w.lock.Unlock()

	for _, keyObj := range items {
		key := keyObj.(string)
		after := o.After
		if o.RateLimited {
			rlAfter := w.rateLimiter.When(key)
			if after == 0 || rlAfter < after {
				after = rlAfter
			}
		}

		var readyAt *time.Time
		if after > 0 {
			readyAt = ptr.To(w.now().Add(after))
			w.metrics.retry()
		}
		if _, ok := w.items[key]; !ok {
			item := &item{
				Key:          key,
				AddedCounter: w.addedCounter,
				Priority:     o.Priority,
				ReadyAt:      readyAt,
			}
			w.items[key] = item
			w.queue.ReplaceOrInsert(item)
			if item.ReadyAt == nil {
				w.metrics.add(key)
			}
			w.addedCounter++
			continue
		}

		// The b-tree de-duplicates based on ordering and any change here
		// will affect the order - Just delete and re-add.
		item, _ := w.queue.Delete(w.items[key])
		if o.Priority > item.Priority {
			item.Priority = o.Priority
		}

		if item.ReadyAt != nil && (readyAt == nil || readyAt.Before(*item.ReadyAt)) {
			if readyAt == nil && !w.becameReady.Has(key) {
				w.metrics.add(key)
			}
			item.ReadyAt = readyAt
		}

		w.queue.ReplaceOrInsert(item)
	}

	if len(items) > 0 {
		w.notifyItemOrWaiterAdded()
	}
}

func (w *priorityqueue) notifyItemOrWaiterAdded() {
	select {
	case w.itemOrWaiterAdded <- struct{}{}:
	default:
	}
}

func (w *priorityqueue) spin() {
	blockForever := make(chan time.Time)
	var nextReady <-chan time.Time
	nextReady = blockForever

	for {
		select {
		case <-w.done:
			return
		case <-w.itemOrWaiterAdded:
		case <-nextReady:
		}

		nextReady = blockForever

		func() {
			w.lock.Lock()
			defer w.lock.Unlock()

			w.lockedLock.Lock()
			defer w.lockedLock.Unlock()

			// manipulating the tree from within Ascend might lead to panics, so
			// track what we want to delete and do it after we are done ascending.
			var toDelete []*item
			w.queue.Ascend(func(item *item) bool {
				if item.ReadyAt != nil {
					if readyAt := item.ReadyAt.Sub(w.now()); readyAt > 0 {
						nextReady = w.tick(readyAt)
						return false
					}
					if !w.becameReady.Has(item.Key) {
						w.metrics.add(item.Key)
						w.becameReady.Insert(item.Key)
					}
				}

				if w.waiters.Load() == 0 {
					// Have to keep iterating here to ensure we update metrics
					// for further items that became ready and set nextReady.
					return true
				}

				// Item is locked, we can not hand it out
				if w.locked.Has(item.Key) {
					return true
				}

				w.metrics.get(item.Key)
				w.locked.Insert(item.Key)
				w.waiters.Add(-1)
				delete(w.items, item.Key)
				toDelete = append(toDelete, item)
				w.becameReady.Delete(item.Key)
				w.get <- *item

				return true
			})

			for _, item := range toDelete {
				w.queue.Delete(item)
			}
		}()
	}
}

func (w *priorityqueue) Add(item interface{}) {
	w.AddWithOpts(AddOpts{}, item)
}

func (w *priorityqueue) AddAfter(item interface{}, after time.Duration) {
	w.AddWithOpts(AddOpts{After: after}, item)
}

func (w *priorityqueue) AddRateLimited(item interface{}) {
	w.AddWithOpts(AddOpts{RateLimited: true}, item)
}

func (w *priorityqueue) GetWithPriority() (_ interface{}, priority int, shutdown bool) {
	w.waiters.Add(1)

	w.notifyItemOrWaiterAdded()
	item := <-w.get

	return item.Key, item.Priority, w.shutdown.Load()
}

func (w *priorityqueue) Get() (item interface{}, shutdown bool) {
	key, _, shutdown := w.GetWithPriority()
	return key, shutdown
}

func (w *priorityqueue) Forget(item interface{}) {
	w.rateLimiter.Forget(item)
}

func (w *priorityqueue) NumRequeues(item interface{}) int {
	return w.rateLimiter.NumRequeues(item)
}

func (w *priorityqueue) ShuttingDown() bool {
	return w.shutdown.Load()
}

func (w *priorityqueue) Done(item interface{}) {
	w.lockedLock.Lock()
	defer w.lockedLock.Unlock()
	w.locked.Delete(item)
	w.metrics.done(item)
	w.notifyItemOrWaiterAdded()
}

func (w *priorityqueue) ShutDown() {
	w.shutdown.Store(true)
	close(w.done)
}

// ShutDownWithDrain just calls ShutDown, as the draining
// functionality is not used by controller-runtime.
func (w *priorityqueue) ShutDownWithDrain() {
	w.ShutDown()
}

// Len returns the number of items that are ready to be
// picked up. It does not include items that are not yet
// ready.
func (w *priorityqueue) Len() int {
	w.lock.Lock()
	defer w.lock.Unlock()

	var result int
	w.queue.Ascend(func(item *item) bool {
		if item.ReadyAt == nil || item.ReadyAt.Compare(w.now()) <= 0 {
			result++
			return true
		}
		return false
	})

	return result
}

func (w *priorityqueue) logState() {
	t := time.Tick(10 * time.Second)
	for {
		select {
		case <-w.done:
			return
		case <-t:
		}

		// Log level may change at runtime, so keep the
		// loop going even if a given level is currently
		// not enabled.
		if !w.log.V(5).Enabled() {
			continue
		}
		w.lock.Lock()
		items := make([]*item, 0, len(w.items))
		w.queue.Ascend(func(item *item) bool {
			items = append(items, item)
			return true
		})
		w.lock.Unlock()

		w.log.V(5).Info("workqueue_items", "items", items)
	}
}

func less(a, b *item) bool {
	if a.ReadyAt == nil && b.ReadyAt != nil {
		return true
	}
	if b.ReadyAt == nil && a.ReadyAt != nil {
		return false
	}
	if a.ReadyAt != nil && b.ReadyAt != nil && !a.ReadyAt.Equal(*b.ReadyAt) {
		return a.ReadyAt.Before(*b.ReadyAt)
	}
	if a.Priority != b.Priority {
		return a.Priority > b.Priority
	}

	return a.AddedCounter < b.AddedCounter
}

type item struct {
	Key          interface{} `json:"key"`
	AddedCounter uint64      `json:"addedCounter"`
	Priority     int         `json:"priority"`
	ReadyAt      *time.Time  `json:"readyAt,omitempty"`
}

func (w *priorityqueue) updateUnfinishedWorkLoop() {
	t := time.Tick(500 * time.Millisecond) // borrowed from workqueue: https://github.com/kubernetes/kubernetes/blob/67a807bf142c7a2a5ecfdb2a5d24b4cdea4cc79c/staging/src/k8s.io/client-go/util/workqueue/queue.go#L182
	for {
		select {
		case <-w.done:
			return
		case <-t:
		}
		w.metrics.updateUnfinishedWork()
	}
}

type bTree[T any] interface {
	ReplaceOrInsert(item T) (_ T, _ bool)
	Delete(item T) (T, bool)
	Ascend(iterator btree.ItemIteratorG[T])
}
