package health

import (
	"encoding/json"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"

	"sync/atomic"

	"k8s.io/apimachinery/pkg/util/runtime"

	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/clock"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
)

var errorCount uint64

func init() {
	atomic.StoreUint64(&errorCount, 0)
	runtime.ErrorHandlers = append(runtime.ErrorHandlers, func(err error) {
		atomic.AddUint64(&errorCount, 1)
	})
}

func NewReadinessChecker(clientset kubecli.KubevirtClient, host string) *ReadinessChecker {

	return &ReadinessChecker{
		clientset: clientset,
		host:      host,
		Clock:     clock.RealClock{},
	}
}

type ReadinessChecker struct {
	clientset kubecli.KubevirtClient
	host      string
	Clock     clock.Clock
}

// HeartBeat take a heartbeat inverval, a maximum of non-userfacing errors which are
// allowed to happen and a stop channel to stop the heartbeat updates.
// It periodically performs some health checks and updates the kubevirt.io/schedulable according to its checks.
// Further it sets a timestamp on the node so that cluster components can see when it last updated the node.
func (l *ReadinessChecker) HeartBeat(interval time.Duration, maxErrorsPerInterval uint64, stopCh chan struct{}) {
	for {
		wait.JitterUntil(func() {
			schedulable := true

			errors := atomic.LoadUint64(&errorCount)
			errorRateExceeded := errors > maxErrorsPerInterval
			atomic.StoreUint64(&errorCount, 0)

			// Check if error rate got exceeded
			if errorRateExceeded {
				// TODO we also need to try to patch the node with a reason
				// TODO do we also want to panic? Maybe better to feed a liveness prove and decide on the manifest level
				// if a restart it swanted.
				schedulable = false
				log.DefaultLogger().
					Reason(fmt.Errorf("Allowed error rate per interval exceeded: %d/%v", errors, interval)).
					Errorf("Will mark myself as unschedulable.")
			}

			now, err := json.Marshal(v12.Time{Time: l.Clock.Now()})
			if err != nil {
				log.DefaultLogger().Reason(err).Errorf("Can't determine date")
				return
			}
			data := []byte(fmt.Sprintf(`{"metadata": { "labels": {"%s": "%t"}, "annotations": {"%s": %s}}}`, v1.NodeSchedulable, schedulable, v1.VirtHandlerHeartbeat, string(now)))
			_, err = l.clientset.CoreV1().Nodes().Patch(l.host, types.StrategicMergePatchType, data)

			if err != nil {
				log.DefaultLogger().Reason(err).Errorf("Can't patch node %s", l.host)
			} else {
				log.DefaultLogger().V(4).Infof("Heartbeat sent")
			}
		}, interval, 1.2, true, stopCh)
	}
}
