package watch

import (
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	virtv1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/controller"
)

func NewVMIServiceController(vmiInformer cache.SharedIndexInformer,
	vmisrvInformer cache.SharedIndexInformer,
	dataVolumeInformer cache.SharedIndexInformer,
	recorder record.EventRecorder,
	clientset kubecli.KubevirtClient) *VMIServiceController {

	c := &VMIServiceController{
		Queue:            workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		recorder:         recorder,
		vmiInformer:      vmiInformer,
		vmisrvInformer:   vmisrvInformer,
		clientset:        clientset,
		vmisExpectations: controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectations()),
	}

	c.vmisrvInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addVMIService,
		DeleteFunc: c.deleteVMIService,
		UpdateFunc: c.updateVMIService,
	})

	return c
}

type VMIServiceController struct {
	clientset        kubecli.KubevirtClient
	Queue            workqueue.RateLimitingInterface
	vmiInformer      cache.SharedIndexInformer
	vmisrvInformer   cache.SharedIndexInformer
	recorder         record.EventRecorder
	vmisExpectations *controller.UIDTrackingControllerExpectations
}

func (c *VMIServiceController) Run(threadiness int, stopCh <-chan struct{}) {
	defer controller.HandlePanic()
	defer c.Queue.ShutDown()
	log.Log.Info("Starting vmiservice controller.")
	// Wait for cache sync before we start the pod controller
	cache.WaitForCacheSync(stopCh, c.vmiInformer.HasSynced)

	// Start the actual work
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh
	log.Log.Info("Stopping vmiservice controller.")
}

func (c *VMIServiceController) runWorker() {
	for c.Execute() {
	}
}

func (c *VMIServiceController) Execute() bool {
	key, quit := c.Queue.Get()
	if quit {
		return false
	}

	defer c.Queue.Done(key)
	err := c.execute(key.(string))

	if err != nil {
		log.Log.Reason(err).Infof("reenqueuing VMIService ")
		c.Queue.AddRateLimited(key)
	} else {
		log.Log.V(4).Infof("processed VMIService ")
		c.Queue.Forget(key)
	}
	return true
}

func (c *VMIServiceController) execute(key string) error {

	// Fetch the latest Vmis state from cache
	obj, exists, err := c.vmisrvInformer.GetStore().GetByKey(key)

	log.Log.Infof(" execute %v : %s", exists, key)
	if err != nil {
		log.Log.Infof(" execute err %v", err)
		return err
	}

	if !exists {
		log.Log.Infof("%s object do not exist", key)
		return fmt.Errorf("%s object do not exist", key)
	}

	vmis := obj.(*virtv1.VMIService)
	log.Log.Infof(">> execute vmis exisit processing.... ")

	// this must be first step in execution. Writing the object
	// when api version changes ensures our api stored version is updated.
	if !controller.ObservedLatestApiVersionAnnotation(vmis) {
		vmis := vmis.DeepCopy()
		controller.SetLatestApiVersionAnnotation(vmis)
		_, err = c.clientset.VMIService(vmis.ObjectMeta.Namespace).Update(vmis)
		return err
	}

	log.Log.Infof(">> execute default processing.... ")

	if vmis.Spec.Hosts == nil || vmis.Spec.Selector == nil {
		log.Log.Infof("Invalid controller spec, will not re-enqueue.")
		return nil
	}

	selector, err := metav1.LabelSelectorAsSelector(vmis.Spec.Selector)
	if err != nil {
		log.Log.Infof("Invalid selector on vmiservice, will not re-enqueue.")
		return nil
	}

	// Need function to check hosts with

	//needsSync := c.vmisExpectations.SatisfiedExpectations(key)

	//NOTE: if control raches here, its assumed there is no VMIService instance for this object.

	// Get the vmiservice instance.
	//if needsSync {
	err = c.syncVMIService(vmis, selector)
	if err != nil {
		return err
	}
	//}
	return nil
}

// When a vmiservice is created, enqueue the VMIService that manages it and update its expectations.
func (c *VMIServiceController) addVMIService(obj interface{}) {
	c.enqueueVMIService(obj)
	return

}

// When a vmiservice is updated, figure out what VMIService manage it and wake them up
func (c *VMIServiceController) updateVMIService(old, cur interface{}) {
	c.enqueueVMIService(cur)
}

// When a vmiservice is deleted, enqueue the VMIService that manages the vmiservice and update its expectations.
func (c *VMIServiceController) deleteVMIService(obj interface{}) {
	c.enqueueVMIService(obj)
}

func (c *VMIServiceController) enqueueVMIService(obj interface{}) {
	logger := log.Log
	vmis := obj.(*virtv1.VMIService)
	key, err := controller.KeyFunc(vmis)
	if err != nil {
		logger.Object(vmis).Reason(err).Error("Failed to extract key from vmiservice")
	}
	c.Queue.Add(key)
}

func (c *VMIServiceController) syncVMIService(vmis *virtv1.VMIService, selector labels.Selector) error {
	//TODO:
	// - Create The headless + haproxy thing.
	logger := log.Log
	logger.Object(vmis).Info("syncVMIService invoked")

	hostsinfo := vmis.Spec.Hosts
	for _, hostinfo := range hostsinfo {
		logger.Object(vmis).Infof("hostinfo : %s", hostinfo)

		// get vm hostname & network endpoint details.
		vmi, err := c.getVMI(vmis.Namespace, hostinfo.VM)
		if err != nil {
			logger.Object(vmis).Info("no vm found")
			return err
		}
		hostname := vmi.Spec.Hostname

		ip, err := c.getEnpoint(vmi, hostinfo.Network)
		if err != nil {
			return err
		}
		logger.Object(vmis).Info(ip)
		err = c.createHeadlessService(vmis.Namespace, hostname, ip)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *VMIServiceController) getVMI(namespace, vmname string) (*virtv1.VirtualMachineInstance, error) {
	// from name to vmi.
	vmi, err := c.clientset.VirtualMachineInstance(namespace).Get(vmname, &metav1.GetOptions{})
	if err != nil {
		log.Log.Infof("no vm found with %s name", vmname)
		return nil, err
	}

	return vmi, nil
}

func (c *VMIServiceController) getEnpoint(vmi *virtv1.VirtualMachineInstance, network string) (string, error) {
	for _, netifc := range vmi.Status.Interfaces {
		log.Log.Infof("Name: %s, IP: %s, MAC: %s, InterfaceName: %s, IPs: %v ",
			netifc.Name, netifc.IP, netifc.MAC, netifc.InterfaceName, netifc.IPs)
		if netifc.Name == network {
			return netifc.IP, nil
		}
	}
	return "", fmt.Errorf("No network %s found", network)
}

func (c *VMIServiceController) createHeadlessService(namespace, vmname, ip string) error {
	//Create Endpoint with Hostname.
	//Create headless service with name as hostname.
	log.Log.Infof("createHeadlessService : namespace: %s, selector: %s , ip: %s ", namespace, vmname, ip)

	endpoint := &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vmname + "-srv",
			Namespace: namespace,
		},
		Subsets: []corev1.EndpointSubset{
			{
				Addresses: []corev1.EndpointAddress{
					{
						IP:       ip,
						Hostname: vmname,
					},
				},
			},
		},
	}

	headlessService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vmname + "-srv",
			Namespace: namespace,
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "None",
		},
	}

	_, err := c.clientset.CoreV1().Services(namespace).Get(headlessService.ObjectMeta.Name, metav1.GetOptions{})
	if err != nil {
		_, err := c.clientset.CoreV1().Services(namespace).Create(headlessService)
		if err != nil {
			log.Log.V(4).Infof("headless service creation : %v", err)
			return err
		}
		log.Log.Infof("headless service created ")
	} else {
		log.Log.V(4).Infof(" Service already exist!!!")
	}

	_, err = c.clientset.CoreV1().Endpoints(namespace).Get(endpoint.ObjectMeta.Name, metav1.GetOptions{})
	if err != nil {
		_, err := c.clientset.CoreV1().Endpoints(namespace).Create(endpoint)
		if err != nil {
			log.Log.V(4).Infof("error in endpoint creation : %v", err)
			return err
		}
	} else {
		log.Log.V(4).Infof(" endpoint already exist!!!")

	}

	/*
		haproxy  pod.
		- backend configuration.
		  - dnsentry mapping to
		  test-app.test-app-srv.default.svc.cluster.local
		  test-app2.test-app2-srv.default.svc.cluster.local

		  TODO: Getting clustername.
				namespace
				service name.
	*/

	return nil
}
