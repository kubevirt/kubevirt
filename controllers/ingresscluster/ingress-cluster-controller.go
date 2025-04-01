package ingresscluster

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/go-logr/logr"
	configv1 "github.com/openshift/api/config/v1"
	routev1 "github.com/openshift/api/route/v1"
	operatorhandler "github.com/operator-framework/operator-lib/handler"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/reqresolver"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/downloadhost"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
)

const (
	ingressName     = "cluster"
	componentName   = "virt-downloads"
	secretNamespace = "openshift-config"

	validConditionReason = "AsExpected"
	validConditionMsg    = "All is well"
)

var (
	selfNamespace = hcoutil.GetOperatorNamespaceFromEnv()
	logger        = logf.Log.WithName("controller_ingress_cluster")
)

type ReconcileIngressCluster struct {
	client.Client
	// used to trigger events in the hyperconverged-controller
	ingressEventCh chan<- event.TypedGenericEvent[client.Object]
}

func newIngressClusterController(cl client.Client, ingressEventCh chan<- event.TypedGenericEvent[client.Object]) *ReconcileIngressCluster {
	return &ReconcileIngressCluster{
		Client:         cl,
		ingressEventCh: ingressEventCh,
	}
}

func RegisterReconciler(mgr manager.Manager, ingressEventCh chan<- event.TypedGenericEvent[client.Object]) error {
	return add(mgr, newIngressClusterController(mgr.GetClient(), ingressEventCh))
}

func add(mgr manager.Manager, r reconcile.Reconciler) error {
	c, err := controller.New("ingress-cluster", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	return c.Watch(source.Kind(mgr.GetCache(), client.Object(&configv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name: ingressName,
		},
	}),
		&operatorhandler.InstrumentedEnqueueRequestForObject[client.Object]{},
		predicate.ResourceVersionChangedPredicate{}),
	)
}

func (r *ReconcileIngressCluster) Reconcile(ctx context.Context, _ reconcile.Request) (reconcile.Result, error) {

	clusterIngress := &configv1.Ingress{}

	if err := r.Get(ctx, client.ObjectKey{Name: ingressName}, clusterIngress); err != nil {
		if errors.IsNotFound(err) {
			logger.Info("Ingress resource not found. Ignoring since object must be deleted")
			return reconcile.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	hcExists, err := r.hyperConvergedExists(ctx, logger)
	if err != nil {
		return ctrl.Result{}, err
	}

	dh, dhErr := r.getDownloadHost(ctx, clusterIngress)

	if downloadhost.Set(dh) {
		logger.Info("Notifying the hyperconverged controller about the new hostname")
		r.ingressEventCh <- event.TypedGenericEvent[client.Object]{Object: clusterIngress}
	}

	routeInd, needUpdate := updateComponentInStatus(clusterIngress, hcExists, dh)

	if needUpdate {
		if dhErr != nil {
			meta.SetStatusCondition(&clusterIngress.Status.ComponentRoutes[routeInd].Conditions, metav1.Condition{
				Type:    "Degraded",
				Status:  metav1.ConditionTrue,
				Reason:  "WrongConfiguration",
				Message: dhErr.Error(),
			})
		} else {
			meta.SetStatusCondition(&clusterIngress.Status.ComponentRoutes[routeInd].Conditions, metav1.Condition{
				Type:    "Degraded",
				Status:  metav1.ConditionFalse,
				Reason:  validConditionReason,
				Message: validConditionMsg,
			})
		}

		meta.SetStatusCondition(&clusterIngress.Status.ComponentRoutes[routeInd].Conditions, metav1.Condition{
			Type:    "Progressing",
			Status:  metav1.ConditionFalse,
			Reason:  validConditionReason,
			Message: validConditionMsg,
		})
		meta.SetStatusCondition(&clusterIngress.Status.ComponentRoutes[routeInd].Conditions, metav1.Condition{
			Type:    "Available",
			Status:  metav1.ConditionTrue,
			Reason:  validConditionReason,
			Message: validConditionMsg,
		})
		err = r.Status().Update(ctx, clusterIngress)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func updateComponentInStatus(clusterIngress *configv1.Ingress, hcExists bool, dh downloadhost.CLIDownloadHost) (int, bool) {
	needUpdate := false
	routeInd := slices.IndexFunc(clusterIngress.Status.ComponentRoutes, func(component configv1.ComponentRouteStatus) bool {
		return component.Name == componentName && component.Namespace == selfNamespace
	})

	if !hcExists {
		if routeInd >= 0 {
			clusterIngress.Status.ComponentRoutes = slices.Delete(clusterIngress.Status.ComponentRoutes, routeInd, routeInd+1)
			needUpdate = true
			logger.Info(fmt.Sprintf("HyperConverged wasn't found; removing the %s component from the cluster Ingress", componentName))
		}
	} else {
		defaultRoute := generateDefaultRouteStatus(dh)
		if routeInd == -1 {
			logger.Info(fmt.Sprintf("Adding the %s component to the cluster Ingress", componentName))
			clusterIngress.Status.ComponentRoutes = append(clusterIngress.Status.ComponentRoutes, defaultRoute)
			needUpdate = true
			routeInd = len(clusterIngress.Status.ComponentRoutes) - 1

		} else if !reflect.DeepEqual(clusterIngress.Status.ComponentRoutes[routeInd], defaultRoute) {
			logger.Info(fmt.Sprintf("Updating the %s component to the cluster Ingress", componentName))
			clusterIngress.Status.ComponentRoutes[routeInd] = defaultRoute
			needUpdate = true
		}
	}
	return routeInd, needUpdate
}

func (r *ReconcileIngressCluster) hyperConvergedExists(ctx context.Context, looger logr.Logger) (bool, error) {
	hc := &v1beta1.HyperConverged{}
	err := r.Get(ctx, reqresolver.GetHyperConvergedNamespacedName(), hc)

	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("HyperConverged resource not found.")
			return false, nil
		}
		looger.Error(err, "Failed to get HyperConverged resource.")
		return false, err
	}

	hcExists := hc.DeletionTimestamp.IsZero()
	if !hcExists {
		looger.Info("HyperConverged resource found, but it was already been deleted.")
	}
	return hcExists, nil
}

func generateDefaultRouteStatus(host downloadhost.CLIDownloadHost) configv1.ComponentRouteStatus {
	serviceAccount := configv1.ConsumingUser(fmt.Sprintf("system:serviceaccount:%s:hyperconverged-cluster-operator", selfNamespace))

	currentHostName := host.DefaultHost
	if len(host.CurrentHost) > 0 {
		currentHostName = host.CurrentHost
	}

	return configv1.ComponentRouteStatus{
		Name:             componentName,
		Namespace:        selfNamespace,
		DefaultHostname:  host.DefaultHost,
		CurrentHostnames: []configv1.Hostname{currentHostName},
		ConsumingUsers:   []configv1.ConsumingUser{serviceAccount},
		RelatedObjects: []configv1.ObjectReference{
			{
				Group:     routev1.GroupName,
				Resource:  "routes",
				Namespace: selfNamespace,
				Name:      downloadhost.CLIDownloadsServiceName,
			},
		},
	}
}

func getDefaultCLIIDownloadHost(domain string) configv1.Hostname {
	return configv1.Hostname(fmt.Sprintf("%s-%s.%s", downloadhost.CLIDownloadsServiceName, selfNamespace, domain))
}

func (r *ReconcileIngressCluster) getDownloadHost(ctx context.Context, clusterIngress *configv1.Ingress) (downloadhost.CLIDownloadHost, error) {
	defaultHost := getDefaultCLIIDownloadHost(clusterIngress.Spec.Domain)
	dh := downloadhost.CLIDownloadHost{
		DefaultHost: defaultHost,
		CurrentHost: defaultHost,
	}

	routeInd := slices.IndexFunc(clusterIngress.Spec.ComponentRoutes, func(component configv1.ComponentRouteSpec) bool {
		return component.Name == componentName && component.Namespace == selfNamespace
	})

	if routeInd >= 0 {
		customHost := clusterIngress.Spec.ComponentRoutes[routeInd].Hostname
		secretName := clusterIngress.Spec.ComponentRoutes[routeInd].ServingCertKeyPairSecret.Name

		if len(customHost) > 0 {
			if !strings.HasSuffix(string(customHost), "."+clusterIngress.Spec.Domain) && len(secretName) == 0 {
				// ignore the customization, keep use the default host
				return dh, fmt.Errorf("must use a secret if the custom host is not a subdomain of the domain name")
			}
		}

		crt, key, err := r.getSecret(ctx, secretName)
		if err != nil {
			return dh, err
		}

		dh.CurrentHost = customHost
		dh.Cert = crt
		dh.Key = key
	}

	return dh, nil
}

func (r *ReconcileIngressCluster) getSecret(ctx context.Context, secretName string) (string, string, error) {
	lgr, err := logr.FromContext(ctx)
	if err != nil {
		return "", "", err
	}

	if len(secretName) > 0 {
		secret := &corev1.Secret{}
		err := r.Get(ctx, client.ObjectKey{Namespace: secretNamespace, Name: secretName}, secret)
		if err != nil {
			lgr.Error(err, "can't get secret", "secret name", secretName, "namespace", secretNamespace)
			return "", "", fmt.Errorf("can't get secret %s; %v", secretName, err)
		} else {
			if secret.Type != corev1.SecretTypeTLS {
				err = fmt.Errorf("non-TLS secret")
				lgr.Error(err, "the secret is with a wrong type", "secret name", secretName, "namespace", secretNamespace)
				return "", "", err
			}

			crt, ok1 := secret.Data["tls.crt"]
			key, ok2 := secret.Data["tls.key"]
			if !ok1 || !ok2 {
				err = fmt.Errorf("wrong TLS secret")
				lgr.Error(err, "can't find required data in the secret", "secret name", secretName, "namespace", secretNamespace)
				return "", "", err
			}

			if err = verifyCertificate(crt); err != nil {
				lgr.Error(err, "wrong secret: can't verify certificate", "secret name", secretName, "namespace", secretNamespace)
				return "", "", err
			}

			if err = verifyPrivateKey(key); err != nil {
				lgr.Error(err, "wrong secret: can't verify the private key", "secret name", secretName, "namespace", secretNamespace)
				return "", "", err
			}

			return string(crt), string(key), nil
		}
	}

	return "", "", nil
}
