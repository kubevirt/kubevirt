package watch

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	corev1 "k8s.io/api/core/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/controller"
)

type ShadowNodeController struct {
	clientset          kubecli.KubevirtClient
	Queue              workqueue.RateLimitingInterface
	nodeInformer       cache.SharedIndexInformer
	shadowNodeInformer cache.SharedIndexInformer
}

// NewNodeController creates a new instance of the NodeController struct.
func NewShadowNodeController(clientset kubecli.KubevirtClient,
	nodeInformer cache.SharedIndexInformer,
	shadowNodeInformer cache.SharedIndexInformer,
) (*ShadowNodeController, error) {
	c := &ShadowNodeController{
		clientset:          clientset,
		Queue:              workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "virt-controller-shadownode"),
		nodeInformer:       nodeInformer,
		shadowNodeInformer: shadowNodeInformer,
	}

	enqueue := func(obj interface{}) {
		node := obj.(*v1.ShadowNode)
		key, err := controller.KeyFunc(node)
		if err != nil {
			return
		}
		c.Queue.Add(key)
	}

	_, err := shadowNodeInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    enqueue,
			UpdateFunc: func(_, newObj interface{}) { enqueue(newObj) },
			DeleteFunc: func(_ interface{}) {},
		},
	)

	return c, err
}

func (c *ShadowNodeController) Run(threadiness int, stopCh <-chan struct{}) {
	defer controller.HandlePanic()
	defer c.Queue.ShutDown()
	log.Log.Info("Starting shadownode controller.")

	cache.WaitForCacheSync(stopCh, c.nodeInformer.HasSynced, c.shadowNodeInformer.HasSynced)

	for i := 0; i < threadiness; i++ {
		go wait.Until(func() {
			for c.Execute() {
			}
		}, time.Second, stopCh)
	}

	<-stopCh
	log.Log.Info("Stopping shadownode controller.")
}

func (c *ShadowNodeController) Execute() bool {
	key, quit := c.Queue.Get()
	if quit {
		return false
	}
	defer c.Queue.Done(key)
	err := c.execute(key.(string))

	if err != nil {
		log.Log.Reason(err).Infof("reenqueuing shadownode %v", key)
		c.Queue.AddRateLimited(key)
	} else {
		log.Log.V(4).Infof("processed shadownode %v", key)
		c.Queue.Forget(key)
	}
	return true
}

func (c *ShadowNodeController) execute(key string) error {
	obj, nodeExists, err := c.nodeInformer.GetStore().GetByKey(key)
	if err != nil {
		return err
	}
	if !nodeExists {
		log.Log.Infof("%v, node counterpart does not exist", key)
		return nil
	}
	node := obj.(*corev1.Node)

	obj, shadowNodeExist, err := c.shadowNodeInformer.GetStore().GetByKey(key)
	if err != nil {
		return err
	}
	if !shadowNodeExist {
		log.Log.Infof("shadowNode %v does not exist", key)
		return nil
	}
	shadowNode := obj.(*v1.ShadowNode)

	filteredNodeAnnotations := calculateNodeAnnotations(node.Annotations, shadowNode.Annotations)
	filteredNodeLabels := calculateNodeLabels(node.Labels, shadowNode.Labels)

	var patches []patch.PatchOperation
	patches = append(patches, makeAnnotationsPath(node.Annotations, filteredNodeAnnotations)...)
	patches = append(patches, makeLabelsPath(node.Labels, filteredNodeLabels)...)

	if len(patches) > 0 {
		payload, err := json.Marshal(patches)
		if err != nil {
			return err
		}

		_, err = c.clientset.CoreV1().Nodes().Patch(context.TODO(), node.Name, types.JSONPatchType, payload, metav1.PatchOptions{})
		if err != nil {
			return err
		}
	}
	return nil
}

func calculateNodeAnnotations(nodeAnnotations, shadowNodeAnnotations map[string]string) map[string]string {
	kubevirtShadowNodeAnnotations := filterKubevirt(shadowNodeAnnotations)
	filteredNodeAnnotations := filterNotKubevirt(nodeAnnotations)

	for key, value := range kubevirtShadowNodeAnnotations {
		filteredNodeAnnotations[key] = value
	}

	return filteredNodeAnnotations
}

func calculateNodeLabels(nodeLabels, shadowNodeLabels map[string]string) map[string]string {
	kubevirtShadowNodeLabels := filterKubevirt(shadowNodeLabels)
	filteredNodeLabels := filterNotKubevirt(nodeLabels)

	for key, value := range kubevirtShadowNodeLabels {
		filteredNodeLabels[key] = value
	}

	return filteredNodeLabels
}

func filterNotKubevirt(m map[string]string) map[string]string {
	filtered := map[string]string{}
	for key, value := range m {
		if !strings.Contains(key, v1.AppLabel) && !strings.Contains(key, v1.CPUManager) {
			filtered[key] = value
		}
	}
	return filtered
}

func filterKubevirt(m map[string]string) map[string]string {
	filtered := map[string]string{}
	for key, value := range m {
		if (strings.Contains(key, v1.AppLabel) && key != v1.VirtHandlerHeartbeat) || strings.Contains(key, v1.CPUManager) {
			filtered[key] = value
		}
	}
	return filtered
}

func makeAnnotationsPath(old, new map[string]string) []patch.PatchOperation {
	var p []patch.PatchOperation
	if !equality.Semantic.DeepEqual(old, new) {
		p = append(p,
			patch.PatchOperation{
				Op:    "test",
				Path:  "/metadata/annotations",
				Value: old,
			},
			patch.PatchOperation{
				Op:    "replace",
				Path:  "/metadata/annotations",
				Value: new,
			},
		)
	}
	return p
}

func makeLabelsPath(old, new map[string]string) []patch.PatchOperation {
	var p []patch.PatchOperation
	if !equality.Semantic.DeepEqual(old, new) {
		p = append(p,
			patch.PatchOperation{
				Op:    "test",
				Path:  "/metadata/labels",
				Value: old,
			},
			patch.PatchOperation{
				Op:    "replace",
				Path:  "/metadata/labels",
				Value: new,
			},
		)
	}
	return p
}
