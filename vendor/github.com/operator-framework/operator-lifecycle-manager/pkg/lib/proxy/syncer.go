package proxy

import (
	"time"

	"github.com/openshift/client-go/config/informers/externalversions"

	apiconfigv1 "github.com/openshift/api/config/v1"
	configv1client "github.com/openshift/client-go/config/clientset/versioned"
	configv1 "github.com/openshift/client-go/config/informers/externalversions/config/v1"
	listers "github.com/openshift/client-go/config/listers/config/v1"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/discovery"
)

const (
	// This is the cluster level global proxy object name.
	globalProxyName = "cluster"

	// default sync interval
	defaultSyncInterval = 30 * time.Minute
)

// NewSyncer returns informer and sync functions to enable watch of Proxy type.
func NewSyncer(logger *logrus.Logger, client configv1client.Interface, discovery discovery.DiscoveryInterface) (proxyInformer configv1.ProxyInformer, syncer *Syncer, querier Querier, err error) {
	factory := externalversions.NewSharedInformerFactoryWithOptions(client, defaultSyncInterval)
	proxyInformer = factory.Config().V1().Proxies()
	s := &Syncer{
		logger: logger,
		lister: proxyInformer.Lister(),
	}

	syncer = s
	querier = s
	return
}

// Syncer deals with watching proxy type(s) on the cluster and let the caller
// query for cluster scoped proxy objects.
type Syncer struct {
	logger *logrus.Logger
	lister listers.ProxyLister
}

// QueryProxyConfig queries the global cluster level proxy object and then
// returns the proxy env variable(s) to the user.
func (w *Syncer) QueryProxyConfig() (proxy []corev1.EnvVar, err error) {
	global, getErr := w.lister.Get(globalProxyName)
	if getErr != nil {
		if !k8serrors.IsNotFound(getErr) {
			err = getErr
			return
		}

		w.logger.Debugf("global Proxy configuration not defined in '%s' - %v", globalProxyName, getErr)
		return
	}

	// We have found the global proxy configuration object!
	proxy = ToEnvVar(global)
	return
}

// SyncProxy is invoked when a cluster scoped proxy object is added or modified.
func (w *Syncer) SyncProxy(object interface{}) error {
	_, ok := object.(*apiconfigv1.Proxy)
	if !ok {
		w.logger.Error("wrong type in proxy syncer")
		return nil
	}

	return nil
}

// HandleProxyDelete is invoked when a cluster scoped proxy object is deleted.
func (w *Syncer) HandleProxyDelete(object interface{}) {
	_, ok := object.(*apiconfigv1.Proxy)
	if !ok {
		w.logger.Error("wrong type in proxy delete syncer")
		return
	}

	return
}
