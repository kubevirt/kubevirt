/*
Copyright 2018 The CDI Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"time"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/cert/triple"
	"k8s.io/client-go/util/workqueue"

	"kubevirt.io/containerized-data-importer/pkg/keys"
	"kubevirt.io/containerized-data-importer/pkg/util"
)

const (
	// AnnUploadRequest marks that a PVC should be made available for upload
	AnnUploadRequest = "cdi.kubevirt.io/storage.upload.target"

	annCreatedByUpload = "cdi.kubevirt.io/storage.createdByUploadController"

	// cert/key annotations

	uploadServerCASecret = "cdi-upload-server-ca-key"
	uploadServerCAName   = "server.upload.cdi.kubevirt.io"

	uploadServerClientCASecret = "cdi-upload-server-client-ca-key"
	uploadServerClientCAName   = "client.upload-server.cdi.kubevirt.io"

	uploadServerClientKeySecret = "cdi-upload-server-client-key"
	uploadProxyClientName       = "uploadproxy.client.upload-server.cdi.kebevirt.io"

	uploadProxyCASecret     = "cdi-upload-proxy-ca-key"
	uploadProxyServerSecret = "cdi-upload-proxy-server-key"
	uploadProxyCAName       = "proxy.upload.cdi.kubevirt.io"
)

// UploadController members
type UploadController struct {
	client                                    kubernetes.Interface
	queue                                     workqueue.RateLimitingInterface
	pvcInformer, podInformer, serviceInformer cache.SharedIndexInformer
	pvcLister                                 corelisters.PersistentVolumeClaimLister
	podLister                                 corelisters.PodLister
	serviceLister                             corelisters.ServiceLister
	pvcsSynced                                cache.InformerSynced
	podsSynced                                cache.InformerSynced
	servicesSynced                            cache.InformerSynced
	uploadServiceImage                        string
	pullPolicy                                string // Options: IfNotPresent, Always, or Never
	verbose                                   string // verbose levels: 1, 2, ...
	serverCAKeyPair                           *triple.KeyPair
	clientCAKeyPair                           *triple.KeyPair
	uploadProxyServiceName                    string
}

// GetUploadResourceName returns the name given to upload services/pods
func GetUploadResourceName(pvcName string) string {
	return "cdi-upload-" + pvcName
}

// UploadPossibleForPVC is called by the api server to see whether to return an upload token
func UploadPossibleForPVC(pvc *v1.PersistentVolumeClaim) error {
	if _, ok := pvc.Annotations[AnnUploadRequest]; !ok {
		return errors.Errorf("PVC %s is not an upload target", pvc.Name)
	}

	pvcPodStatus, ok := pvc.Annotations[AnnPodPhase]
	if !ok || v1.PodPhase(pvcPodStatus) != v1.PodRunning {
		return errors.Errorf("Upload Server pod not currently running for PVC %s", pvc.Name)
	}

	return nil
}

// NewUploadController returns a new UploadController
func NewUploadController(client kubernetes.Interface,
	pvcInformer coreinformers.PersistentVolumeClaimInformer,
	podInformer coreinformers.PodInformer,
	serviceInformer coreinformers.ServiceInformer,
	uploadServiceImage string,
	uploadProxyServiceName string,
	pullPolicy string,
	verbose string) *UploadController {
	c := &UploadController{
		client:                 client,
		queue:                  workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		pvcInformer:            pvcInformer.Informer(),
		podInformer:            podInformer.Informer(),
		serviceInformer:        serviceInformer.Informer(),
		pvcLister:              pvcInformer.Lister(),
		podLister:              podInformer.Lister(),
		serviceLister:          serviceInformer.Lister(),
		pvcsSynced:             pvcInformer.Informer().HasSynced,
		podsSynced:             podInformer.Informer().HasSynced,
		servicesSynced:         serviceInformer.Informer().HasSynced,
		uploadServiceImage:     uploadServiceImage,
		uploadProxyServiceName: uploadProxyServiceName,
		pullPolicy:             pullPolicy,
		verbose:                verbose,
	}

	// Bind the pvc SharedIndexInformer to the pvc queue
	c.pvcInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.enqueueObject,
		UpdateFunc: func(old, new interface{}) {
			c.enqueueObject(new)
		},
	})

	// Bind the pod SharedIndexInformer to the pvc queue
	c.podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.handleObject,
		UpdateFunc: func(old, new interface{}) {
			newDepl := new.(*v1.Pod)
			oldDepl := old.(*v1.Pod)
			if newDepl.ResourceVersion == oldDepl.ResourceVersion {
				// Periodic resync will send update events for all known Pods.
				// Two different versions of the same PVCs will always have different RVs.
				return
			}
			c.handleObject(new)
		},
		DeleteFunc: c.handleObject,
	})

	// Bind the service SharedIndexInformer to the service queue
	c.serviceInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.handleObject,
		UpdateFunc: func(old, new interface{}) {
			newDepl := new.(*v1.Service)
			oldDepl := old.(*v1.Service)
			if newDepl.ResourceVersion == oldDepl.ResourceVersion {
				// Periodic resync will send update events for all known Services.
				// Two different versions of the same Servicess will always have different RVs.
				return
			}
			c.handleObject(new)
		},
		DeleteFunc: c.handleObject,
	})

	return c
}

// Run sets up UploadController state and executes main event loop
func (c *UploadController) Run(threadiness int, stopCh <-chan struct{}) error {
	defer runtime.HandleCrash()
	defer c.queue.ShutDown()

	glog.V(2).Infoln("Getting/creating certs")

	if err := c.initCerts(); err != nil {
		runtime.HandleError(err)
		return errors.Wrap(err, "Error initializing certificates")
	}

	glog.V(2).Infoln("Starting cdi upload controller Run loop")

	if threadiness < 1 {
		return errors.Errorf("expected >0 threads, got %d", threadiness)
	}

	glog.V(3).Info("Waiting for informer caches to sync")

	if ok := cache.WaitForCacheSync(stopCh, c.pvcsSynced, c.podsSynced, c.servicesSynced); !ok {
		return errors.New("failed to wait for caches to sync")
	}

	glog.V(3).Infoln("UploadController cache has synced")

	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	glog.Info("Started workers")
	<-stopCh
	glog.Info("Shutting down workers")

	return nil
}

func (c *UploadController) initCerts() error {
	var err error

	// CA for Upload Servers
	c.serverCAKeyPair, err = keys.GetOrCreateCA(c.client, util.GetNamespace(), uploadServerCASecret, uploadServerCAName)
	if err != nil {
		return errors.Wrap(err, "Couldn't get/create server CA")
	}

	// CA for Upload Client
	c.clientCAKeyPair, err = keys.GetOrCreateCA(c.client, util.GetNamespace(), uploadServerClientCASecret, uploadServerClientCAName)
	if err != nil {
		return errors.Wrap(err, "Couldn't get/create client CA")
	}

	// Upload Server Client Cert
	_, err = keys.GetOrCreateClientKeyPairAndCert(c.client,
		util.GetNamespace(),
		uploadServerClientKeySecret,
		c.clientCAKeyPair,
		c.serverCAKeyPair.Cert,
		uploadProxyClientName,
		[]string{},
		nil,
	)
	if err != nil {
		return errors.Wrap(err, "Couldn't get/create client cert")
	}

	uploadProxyCAKeyPair, err := keys.GetOrCreateCA(c.client, util.GetNamespace(), uploadProxyCASecret, uploadProxyCAName)
	if err != nil {
		return errors.Wrap(err, "Couldn't create upload proxy server cert")
	}

	_, err = keys.GetOrCreateServerKeyPairAndCert(c.client,
		util.GetNamespace(),
		uploadProxyServerSecret,
		uploadProxyCAKeyPair,
		nil,
		c.uploadProxyServiceName+"."+util.GetNamespace(),
		c.uploadProxyServiceName,
		nil,
	)
	if err != nil {
		return errors.Wrap(err, "Error creating upload proxy server key pair")
	}

	return nil
}

func (c *UploadController) handleObject(obj interface{}) {
	var object metav1.Object
	var ok bool
	if object, ok = obj.(metav1.Object); !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			runtime.HandleError(errors.Errorf("error decoding object, invalid type"))
			return
		}
		object, ok = tombstone.Obj.(metav1.Object)
		if !ok {
			runtime.HandleError(errors.Errorf("error decoding object tombstone, invalid type"))
			return
		}
		glog.V(3).Infof("Recovered deleted object '%s' from tombstone", object.GetName())
	}
	glog.V(3).Infof("Processing object: %s", object.GetName())
	if ownerRef := metav1.GetControllerOf(object); ownerRef != nil {
		_, createdByUs := object.GetAnnotations()[annCreatedByUpload]

		if ownerRef.Kind != "PersistentVolumeClaim" || !createdByUs {
			return
		}

		pvc, err := c.pvcLister.PersistentVolumeClaims(object.GetNamespace()).Get(ownerRef.Name)
		if err != nil {
			glog.V(3).Infof("ignoring orphaned object '%s' of pvc '%s'", object.GetSelfLink(), ownerRef.Name)
			return
		}

		glog.V(3).Infof("queueing pvc %+v!!", pvc)

		c.enqueueObject(pvc)
		return
	}
}

func (c *UploadController) enqueueObject(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		runtime.HandleError(err)
		return
	}
	c.queue.AddRateLimited(key)
}

func (c *UploadController) runWorker() {
	for c.processNextWorkItem() {
	}
}

func (c *UploadController) processNextWorkItem() bool {
	obj, shutdown := c.queue.Get()

	if shutdown {
		return false
	}

	err := func(obj interface{}) error {
		defer c.queue.Done(obj)

		var key string
		var ok bool

		if key, ok = obj.(string); !ok {
			c.queue.Forget(obj)
			runtime.HandleError(errors.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}

		if err := c.syncHandler(key); err != nil {
			return errors.Errorf("error syncing '%s': %s", key, err.Error())
		}

		c.queue.Forget(obj)
		glog.Infof("Successfully synced '%s'", key)
		return nil

	}(obj)

	if err != nil {
		runtime.HandleError(err)
		return true
	}

	return true
}

func podPhaseFromPVC(pvc *v1.PersistentVolumeClaim) v1.PodPhase {
	phase, _ := pvc.ObjectMeta.Annotations[AnnPodPhase]
	return v1.PodPhase(phase)
}

func podSucceededFromPVC(pvc *v1.PersistentVolumeClaim) bool {
	return (podPhaseFromPVC(pvc) == v1.PodSucceeded)
}

func (c *UploadController) syncHandler(key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		runtime.HandleError(errors.Errorf("invalid resource key: %s", key))
		return nil
	}

	pvc, err := c.pvcLister.PersistentVolumeClaims(namespace).Get(name)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			runtime.HandleError(errors.Errorf("PVC '%s' in work queue no longer exists", key))
			return nil
		}
		return errors.Wrapf(err, "error getting PVC %s", key)
	}

	_, isUploadPod := pvc.ObjectMeta.Annotations[AnnUploadRequest]
	resourceName := GetUploadResourceName(pvc.Name)

	// force cleanup if PVC pending delete and pod running or the upoad annotation was removed
	if !isUploadPod || (pvc.DeletionTimestamp != nil && !podSucceededFromPVC(pvc)) {
		// delete everything

		// delete service
		err = c.deleteService(pvc.Namespace, resourceName)
		if err != nil {
			return errors.Wrapf(err, "Error deleting upload service for pvc: %s", key)
		}

		// delete pod
		// we're using a req struct for now until we can normalize the controllers a bit more and share things like lister, client etc
		// this way it's easy to stuff everything into an easy request struct, and can extend aditional behaviors if we want going forward
		// NOTE this is a special case where the user updated annotations on the pvc to abort the upload requests, we'll add another call
		// for this for the success case
		dReq := podDeleteRequest{
			namespace: pvc.Namespace,
			podName:   resourceName,
			podLister: c.podLister,
			k8sClient: c.client,
		}

		err = deletePod(dReq)
		if err != nil {
			return errors.Wrapf(err, "Error deleting upload pod for pvc: %s", key)
		}

		return nil
	}

	var pod *v1.Pod
	if !podSucceededFromPVC(pvc) {
		pod, err = c.getOrCreateUploadPod(pvc, resourceName)
		if err != nil {
			return errors.Wrapf(err, "Error creating upload pod for pvc: %s", key)
		}

		podPhase := pod.Status.Phase
		if podPhase != podPhaseFromPVC(pvc) {
			var labels map[string]string
			annotations := map[string]string{AnnPodPhase: string(podPhase)}
			pvc, err = updatePVC(c.client, pvc, annotations, labels)
			if err != nil {
				return errors.Wrapf(err, "Error updating pvc %s, pod phase %s", key, podPhase)
			}
		}
	}

	if podSucceededFromPVC(pvc) {
		// delete service
		if err = c.deleteService(pvc.Namespace, resourceName); err != nil {
			return errors.Wrapf(err, "Error deleting upload service for pvc %s", key)
		}

		dReq := podDeleteRequest{
			namespace: pvc.Namespace,
			podName:   resourceName,
			podLister: c.podLister,
			k8sClient: c.client,
		}

		// delete the pod
		err = deletePod(dReq)
		if err != nil {
			return errors.Wrapf(err, "Error deleting upload pod for pvc: %s", key)
		}

	} else {
		// make sure the service exists
		if _, err = c.getOrCreateUploadService(pvc, resourceName); err != nil {
			return errors.Wrapf(err, "Error getting/creating service resource for PVC %s", key)
		}
	}

	return nil
}

func (c *UploadController) getOrCreateUploadPod(pvc *v1.PersistentVolumeClaim, name string) (*v1.Pod, error) {
	pod, err := c.podLister.Pods(pvc.Namespace).Get(name)

	if k8serrors.IsNotFound(err) {
		pod, err = CreateUploadPod(c.client, c.serverCAKeyPair, c.clientCAKeyPair.Cert, c.uploadServiceImage, c.verbose, c.pullPolicy, name, pvc)
	}

	if pod != nil && !metav1.IsControlledBy(pod, pvc) {
		return nil, errors.Errorf("%s pod not controlled by pvc %s", name, pvc.Name)
	}

	return pod, err
}

func (c *UploadController) getOrCreateUploadService(pvc *v1.PersistentVolumeClaim, name string) (*v1.Service, error) {
	service, err := c.serviceLister.Services(pvc.Namespace).Get(name)

	if k8serrors.IsNotFound(err) {
		service, err = CreateUploadService(c.client, name, pvc)
	}

	if service != nil && !metav1.IsControlledBy(service, pvc) {
		return nil, errors.Errorf("%s service not controlled by pvc %s", name, pvc.Name)
	}

	return service, err
}

func (c *UploadController) deleteService(namespace, name string) error {
	service, err := c.serviceLister.Services(namespace).Get(name)
	if k8serrors.IsNotFound(err) {
		return nil
	}
	if err == nil && service.DeletionTimestamp == nil {
		err = c.client.CoreV1().Services(namespace).Delete(name, &metav1.DeleteOptions{})
		if k8serrors.IsNotFound(err) {
			return nil
		}
	}
	return err
}
