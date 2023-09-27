package virthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/util/workqueue"

	"kubevirt.io/kubevirt/pkg/controller"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	"kubevirt.io/kubevirt/pkg/safepath"
	"kubevirt.io/kubevirt/pkg/util"
	virtcache "kubevirt.io/kubevirt/pkg/virt-handler/cache"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
)

const (
	statsUpdateInterval = 90 * time.Second
)

type GuestStatsController struct {
	virtShareDir         string
	clientset            kubecli.KubevirtClient
	vmiInformer          cache.SharedIndexInformer
	launcherClient       virtcache.LauncherClientInfoByVMI
	podIsolationDetector isolation.PodIsolationDetector
	logger               *log.FilteredLogger
	vmiQueue             workqueue.RateLimitingInterface
}

func NewGuestStatsController(virtShareDir string, clientset kubecli.KubevirtClient, vmiInformer cache.SharedIndexInformer) (*GuestStatsController, error) {
	ctrl := &GuestStatsController{
		virtShareDir:         virtShareDir,
		clientset:            clientset,
		vmiInformer:          vmiInformer,
		launcherClient:       virtcache.LauncherClientInfoByVMI{},
		podIsolationDetector: isolation.NewSocketBasedIsolationDetector(virtShareDir),
		logger:               log.Logger("guest-stat-controller"),
		vmiQueue:             workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "guest-stats-updater-vmi"),
	}

	_, err := ctrl.vmiInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := controller.KeyFunc(obj)
			if err != nil {
				ctrl.logger.Reason(err).Errorf("key func failed at AddFunc event handler")
				return
			}
			ctrl.reenqueueVmi(key)
		},
		DeleteFunc: func(obj interface{}) {
			key, err := controller.KeyFunc(obj)
			if err != nil {
				ctrl.logger.Reason(err).Errorf("key func failed at DeleteFunc event handler")
				return
			}
			ctrl.vmiQueue.Done(key)
			ctrl.vmiQueue.Forget(key)
		},
	})
	if err != nil {
		return nil, err
	}

	return ctrl, nil
}

func (c *GuestStatsController) Run(threadiness int, stopCh <-chan struct{}) {
	defer c.vmiQueue.ShutDown()
	c.logger.Info("Starting guest-stats controller.")
	defer c.logger.Info("Stopping guest-stats controller.")

	cache.WaitForCacheSync(stopCh, c.vmiInformer.HasSynced)

	// Add all running VMIs to the queue
	for _, vmiKey := range c.vmiInformer.GetStore().ListKeys() {
		c.reenqueueVmi(vmiKey)
	}

	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh
}

func (c *GuestStatsController) runWorker() {
	for c.Execute() {
	}
}

func (c *GuestStatsController) Execute() bool {
	key, quit := c.vmiQueue.Get()
	if quit {
		return false
	}

	err := c.execute(key.(string))
	if err != nil {
		c.logger.Reason(err).Errorf("Failed to update guest stats for vmi %s", key)
		return true
	}

	c.vmiQueue.Done(key)
	c.reenqueueVmi(key.(string))

	return true
}

func (c *GuestStatsController) execute(key string) error {
	obj, exists, err := c.vmiInformer.GetStore().GetByKey(key)
	if err != nil {
		return err
	} else if !exists {
		return fmt.Errorf("vmi %s does not exist", key)
	}

	vmi := obj.(*virtv1.VirtualMachineInstance)
	err = c.updateVmiGuestStats(vmi)
	if err != nil {
		return err
	}

	return nil
}

func (c *GuestStatsController) updateVmiGuestStats(vmi *virtv1.VirtualMachineInstance) error {
	launcherClient, err := c.getLauncherClient(vmi)
	if err != nil {
		return err
	}

	guestStats, err := launcherClient.GetGuestStats()
	if err != nil {
		return err
	} else if guestStats.DirtyRate == nil {
		c.logger.Infof("dirty rate is missing, skipping")
		return nil
	}

	const minDiff = 0.1
	const maxSamples = 150

	newDirtyRate := guestStats.DirtyRate
	var curDirtyRate *virtv1.DirtyRateStats
	if guestStats := vmi.Status.GuestStats; guestStats != nil && guestStats.DirtyRate != nil {
		curDirtyRate = guestStats.DirtyRate
	} else {
		curDirtyRate = &virtv1.DirtyRateStats{}
	}

	isFirstUpdate := curDirtyRate.SampleCount == 0
	maxSamplesReached := curDirtyRate.SampleCount+maxSamples < newDirtyRate.SampleCount
	isBigAvgDiff := newDirtyRate.Average > curDirtyRate.Average*(1+minDiff) || newDirtyRate.Average < curDirtyRate.Average*(1-minDiff)

	if isFirstUpdate || maxSamplesReached || isBigAvgDiff {
		c.logger.Infof("Updating guest stats from %+v to %+v for vmi %s/%s", *curDirtyRate, *newDirtyRate, vmi.Namespace, vmi.Name)
		err = c.patchGuestStats(vmi, guestStats)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *GuestStatsController) getLauncherClient(vmi *virtv1.VirtualMachineInstance) (cmdclient.LauncherClient, error) {
	return GetLauncherClient(vmi, &c.launcherClient, c.podIsolationDetector, c.virtShareDir)
}

func (c *GuestStatsController) patchGuestStats(vmi *virtv1.VirtualMachineInstance, newGuestStats virtv1.GuestStats) error {
	patchOp := patch.PatchReplaceOp
	if guestStats := vmi.Status.GuestStats; guestStats == nil || guestStats.DirtyRate == nil {
		patchOp = patch.PatchAddOp
	}

	guestStatsStr, err := json.Marshal(newGuestStats)
	if err != nil {
		return fmt.Errorf("failed marshalling: %v", err)
	}

	patchBytes := fmt.Sprintf(`[{ "op": "%s", "path": "/status/guestStats", "value": %s }]`, patchOp, guestStatsStr)
	_, err = c.clientset.VirtualMachineInstance(vmi.Namespace).Patch(context.TODO(), vmi.Name, types.JSONPatchType, []byte(patchBytes), &v1.PatchOptions{})

	return err
}

func (c *GuestStatsController) reenqueueVmi(key string) {
	const intervalMutationPercent = 15.0 / 100.0
	updateInterval := statsUpdateInterval.Seconds()

	min := int((1.0 - intervalMutationPercent) * updateInterval)
	max := int((1.0 + intervalMutationPercent) * updateInterval)
	nextUpdateInterval := rand.IntnRange(min, max)
	c.vmiQueue.AddAfter(key, time.Duration(nextUpdateInterval)*time.Second)

	c.logger.V(4).Infof("Reenqueued vmi %s for %d seconds", key, nextUpdateInterval)
}

func GetLauncherClient(vmi *virtv1.VirtualMachineInstance, launcherClients *virtcache.LauncherClientInfoByVMI, podIsolationDetector isolation.PodIsolationDetector, virtShareDir string) (cmdclient.LauncherClient, error) {
	var err error

	clientInfo, exists := launcherClients.Load(vmi.UID)
	if exists && clientInfo.Client != nil {
		return clientInfo.Client, nil
	}

	socketFile, err := cmdclient.FindSocketOnHost(vmi)
	if err != nil {
		return nil, err
	}

	err = virtcache.AddGhostRecord(vmi.Namespace, vmi.Name, socketFile, vmi.UID)
	if err != nil {
		return nil, err
	}

	client, err := cmdclient.NewClient(socketFile)
	if err != nil {
		return nil, err
	}

	domainPipeStopChan := make(chan struct{})
	// if this isn't a legacy socket, we need to
	// pipe in the domain socket into the VMI's filesystem
	if !cmdclient.IsLegacySocket(socketFile) {
		err = StartDomainNotifyPipe(vmi, domainPipeStopChan, podIsolationDetector, virtShareDir)
		if err != nil {
			client.Close()
			close(domainPipeStopChan)
			return nil, err
		}
	}

	launcherClients.Store(vmi.UID, &virtcache.LauncherClientInfo{
		Client:              client,
		SocketFile:          socketFile,
		DomainPipeStopChan:  domainPipeStopChan,
		NotInitializedSince: time.Now(),
		Ready:               true,
	})

	return client, nil
}

func StartDomainNotifyPipe(vmi *virtv1.VirtualMachineInstance, domainPipeStopChan chan struct{}, podIsolationDetector isolation.PodIsolationDetector, virtShareDir string) error {

	res, err := podIsolationDetector.Detect(vmi)
	if err != nil {
		return fmt.Errorf("failed to detect isolation for launcher pod when setting up notify pipe: %v", err)
	}

	// inject the domain-notify.sock into the VMI pod.
	root, err := res.MountRoot()
	if err != nil {
		return err
	}
	socketDir, err := root.AppendAndResolveWithRelativeRoot(virtShareDir)
	if err != nil {
		return err
	}

	listener, err := safepath.ListenUnixNoFollow(socketDir, "domain-notify-pipe.sock")
	if err != nil {
		log.Log.Reason(err).Error("failed to create unix socket for proxy service")
		return err
	}
	socketPath, err := safepath.JoinNoFollow(socketDir, "domain-notify-pipe.sock")
	if err != nil {
		return err
	}

	if util.IsNonRootVMI(vmi) {
		err := diskutils.DefaultOwnershipManager.SetFileOwnership(socketPath)
		if err != nil {
			log.Log.Reason(err).Error("unable to change ownership for domain notify")
			return err
		}
	}

	handleDomainNotifyPipe(domainPipeStopChan, listener, virtShareDir, vmi)

	return nil
}
