package watch

import (
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v13 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v12 "kubevirt.io/client-go/api/v1"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/controller"
)

const (
	virtualMachineInstanceMigrationCreationSuccess = "MigrationCreatedSuccessfully"
	virtualMachineInstanceMigrationCreationFailure = "FailureCreatingMigration"
)

// LauncherEvictionController watches for virt-launcher pods that were marked for eviction and triggers a migration.
type LauncherEvictionController struct {
	clientset        kubecli.KubevirtClient
	Queue            workqueue.RateLimitingInterface
	podInformer      cache.SharedIndexInformer
	recorder         record.EventRecorder
	heartBeatTimeout time.Duration
	recheckInterval  time.Duration
}

// NewLauncherEvictionController creates a new instance of the LauncherEvictionController struct.
func NewLauncherEvictionController(clientset kubecli.KubevirtClient, podInformer cache.SharedIndexInformer, recorder record.EventRecorder) *LauncherEvictionController {
	c := &LauncherEvictionController{
		clientset:        clientset,
		Queue:            workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		podInformer:      podInformer,
		recorder:         recorder,
		heartBeatTimeout: 5 * time.Minute,
		recheckInterval:  1 * time.Minute,
	}

	c.podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addPod,
		DeleteFunc: func(_ interface{}) {}, // nothing to do
		UpdateFunc: c.updatePod,
	})

	return c
}

func (c *LauncherEvictionController) addPod(obj interface{}) {
	c.enqueuePod(obj)
}

func (c *LauncherEvictionController) updatePod(_, curr interface{}) {
	c.enqueuePod(curr)
}

func (c *LauncherEvictionController) enqueuePod(obj interface{}) {
	logger := log.Log
	pod := obj.(*v1.Pod)
	key, err := controller.KeyFunc(pod)
	if err != nil {
		logger.Object(pod).Reason(err).Error("Failed to extract key from pod.")
	}
	c.Queue.Add(key)
}

// Run runs the passed in LauncherEvictionController.
func (c *LauncherEvictionController) Run(threadiness int, stopCh <-chan struct{}) {
	defer controller.HandlePanic()
	defer c.Queue.ShutDown()
	log.Log.Info("Starting node controller.")

	// Wait for cache sync before we start the node controller
	cache.WaitForCacheSync(stopCh, c.podInformer.HasSynced)

	// Start the actual work
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh
	log.Log.Info("Stopping node controller.")
}

func (c *LauncherEvictionController) runWorker() {
	for c.Execute() {
	}
}

// Execute runs commands from the controller queue, if there is
// an error it re-enqueues the key. Returns false if the queue
// is empty.
func (c *LauncherEvictionController) Execute() bool {
	key, quit := c.Queue.Get()
	if quit {
		return false
	}
	defer c.Queue.Done(key)
	err := c.execute(key.(string))

	if err != nil {
		log.Log.Reason(err).Infof("reenqueuing pod %v", key)
		c.Queue.AddRateLimited(key)
	} else {
		log.Log.V(4).Infof("processed pod %v", key)
		c.Queue.Forget(key)
	}
	return true
}

func (c *LauncherEvictionController) execute(key string) error {
	obj, podExists, err := c.podInformer.GetStore().GetByKey(key)
	if err != nil {
		return err
	}
	if !podExists {
		return fmt.Errorf("could not get pod %s from store", key)
	}

	var pod *v1.Pod
	pod, ok := obj.(*v1.Pod)
	if !ok {
		return fmt.Errorf("could not cast stored object to pod")
	}

	logger := log.DefaultLogger()
	logger = logger.Object(pod)
	logger.Key(key, "Pod")

	podLabels := pod.GetLabels()
	if app, ok := podLabels[v12.AppLabel]; !ok || app != "virt-launcher" {
		log.Log.V(4).Infof("Nothing to do with pod %s, not a virt-launcher app", pod.GetName())
		return nil
	}

	markedForEviction := false
	for _, condition := range pod.Status.Conditions {
		if condition.Type == v12.LauncherMarkedForEviction {
			markedForEviction = true
			break
		}
	}

	if !markedForEviction {
		logger.Level(2).Infof("Nothing to do with pod %s as it was not marked for eviction", pod.Name)
		return nil
	}

	vmiName, ok := pod.Annotations[v12.DomainAnnotation]
	if !ok {
		return fmt.Errorf("could not find VMI for pod %s", pod.Name)
	}

	vmi, err := c.clientset.VirtualMachineInstance(pod.GetNamespace()).Get(vmiName, &metav1.GetOptions{})
	if err != nil {
		// there's nothing we can do at this point so we have to remove the pod
		logger.Level(2).Errorf("Could not get VMI for pod %s", err.Error())
		c.recorder.Eventf(
			pod, v1.EventTypeWarning, virtualMachineInstanceMigrationCreationFailure, "%s", err.Error())
		return c.clientset.CoreV1().Pods(pod.GetNamespace()).Delete(pod.Name, &metav1.DeleteOptions{})
	}

	if !vmi.IsMigratable() {
		// try to stop the VM if it exists
		err = c.clientset.VirtualMachine(vmi.GetNamespace()).Stop(vmi.GetName())
		if err != nil {
			if errors.IsNotFound(err) {
				// There's nothing we can do at this point because the VMI is not migratable and is not backed by a VM
				// that we can stop.
				logger.Level(2).Errorf("Deleting evicted pod %s/%s", pod.GetNamespace(), pod.GetName())
				return c.clientset.CoreV1().Pods(pod.GetNamespace()).Delete(pod.Name, &metav1.DeleteOptions{})
			}
			logger.Level(2).Errorf("Could not stop VM for pod %s/%s: %s", pod.GetNamespace(), pod.GetName(), err.Error())
			return err
		}
		return nil
	}

	// Check if there's a migration in progress for this pod
	migrations, err := c.clientset.VirtualMachineInstanceMigration(pod.Namespace).List(&metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", v12.MigrationForEvictedPodLabel, pod.Name),
	})
	if err != nil {
		if !errors.IsNotFound(err) {
			logger.Errorf("Could not list migrations for pod %s/%s", pod.Namespace, pod.Name)
			c.recorder.Eventf(
				pod, v1.EventTypeWarning, virtualMachineInstanceMigrationCreationFailure, "%s", err.Error())
			return err
		}
	} else if len(migrations.Items) > 0 {
		logger.Level(2).Infof("A migration for evicted pod %s/%s already exists", pod.Namespace, pod.Name)
		return nil
	}

	// Create a migration for the VMI
	migration := &v12.VirtualMachineInstanceMigration{
		ObjectMeta: v13.ObjectMeta{
			GenerateName: "kubevirt-eviction-",
			Labels: map[string]string{
				v12.MigrationForEvictedPodLabel: pod.Name,
			},
		},
		Spec: v12.VirtualMachineInstanceMigrationSpec{
			VMIName: vmiName,
		},
	}
	createdMigration, err := c.clientset.VirtualMachineInstanceMigration(pod.Namespace).Create(migration)
	if err != nil {
		logger.Errorf("Could not create virtual machine instance migration: %s", err.Error())
		c.recorder.Eventf(
			pod, v1.EventTypeWarning, virtualMachineInstanceMigrationCreationFailure, "%s", err.Error())
		return err
	}
	c.recorder.Eventf(
		pod, v1.EventTypeNormal, virtualMachineInstanceMigrationCreationSuccess,
		"Created migration %s", createdMigration.Name)

	return nil
}
