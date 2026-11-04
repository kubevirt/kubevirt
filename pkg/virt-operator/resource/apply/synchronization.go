package apply

import (
	"context"
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"

	routev1 "github.com/openshift/api/route/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/client-go/log"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/network/multus"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
	"kubevirt.io/kubevirt/pkg/virt-operator/util"
)

func (r *Reconciler) updateSynchronizationAddress() (err error) {
	if !r.isFeatureGateEnabled(featuregate.DecentralizedLiveMigration) {
		r.kv.Status.SynchronizationAddresses = nil
		return nil
	}
	// Find the lease associated with the virt-synchronization controller
	lease, err := r.k8sClient.CoordinationV1().Leases(r.kv.Namespace).Get(context.Background(), components.VirtSynchronizationControllerName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			r.kv.Status.SynchronizationAddresses = nil
			return nil
		}
		return err
	}
	podName := ""
	if lease.Spec.HolderIdentity != nil {
		podName = *lease.Spec.HolderIdentity
	}
	if podName == "" {
		r.kv.Status.SynchronizationAddresses = nil
		return nil
	}
	pod, err := r.k8sClient.CoreV1().Pods(r.kv.Namespace).Get(context.Background(), podName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			r.kv.Status.SynchronizationAddresses = nil
			return nil
		}
		return err
	}
	if pod.DeletionTimestamp != nil {
		r.kv.Status.SynchronizationAddresses = nil
		return nil
	}

	// When the proxy datapath is enabled without a cross-cluster network,
	// check for an admin-labeled Ingress, Route, or Service that provides
	// the externally reachable synchronization endpoint.
	if r.isDecentralizedLiveMigrationProxyEnabled() && !r.isCrossClusterMigrationNetworkConfigured() {
		endpointAddress, err := r.getSynchronizationEndpointAddress()
		if err != nil {
			log.Log.Warningf("synchronization endpoint misconfiguration: %v", err)
			r.kv.Status.SynchronizationAddresses = nil
			changed := util.SetSynchronizationEndpointCondition(r.kv, corev1.ConditionFalse,
				util.ConditionReasonSynchronizationEndpointMisconfigured, err.Error())
			// Emit a Warning only on transition to avoid reconcile-loop event spam.
			if changed && r.recorder != nil {
				r.recorder.Eventf(r.kv, corev1.EventTypeWarning,
					util.ConditionReasonSynchronizationEndpointMisconfigured,
					"synchronization endpoint misconfiguration: %v", err)
			}
			return nil
		}
		if endpointAddress != "" {
			r.kv.Status.SynchronizationAddresses = []string{endpointAddress}
			util.SetSynchronizationEndpointCondition(r.kv, corev1.ConditionTrue,
				util.ConditionReasonSynchronizationEndpointFound,
				fmt.Sprintf("using synchronization endpoint %s", endpointAddress))
			return nil
		}
		// No labeled endpoint found; fall through to pod IP.
	}

	// Clear when falling through to network/pod IP (proxy without a labeled
	// endpoint, or discovery not active).
	util.ClearSynchronizationEndpointCondition(r.kv)

	// Check for the migration network address in the pod annotations.
	ips := r.getIpsFromAnnotations(pod)
	if len(ips) == 0 && pod.Status.PodIPs != nil &&
		!(r.isDecentralizedLiveMigrationProxyEnabled() && r.isCrossClusterMigrationNetworkConfigured()) {
		// Did not find annotations, use the pod ip address instead
		ips = make([]string, len(pod.Status.PodIPs))
		for i, podIP := range pod.Status.PodIPs {
			ips[i] = podIP.IP
		}
	}
	if len(ips) == 0 {
		return nil
	}
	port, err := r.synchronizationPort()
	if err != nil {
		return err
	}
	addresses := make([]string, len(ips))
	for i, ip := range ips {
		addresses[i] = net.JoinHostPort(ip, strconv.Itoa(int(port)))
	}
	r.kv.Status.SynchronizationAddresses = addresses
	return nil
}

func (r *Reconciler) getIpsFromAnnotations(pod *corev1.Pod) []string {
	// Priority for advertised SynchronizationAddresses (must match where the
	// sync controller actually listens for peer gRPC):
	// 1. crosscluster0 when Proxy datapath is enabled and crossClusterNetwork is set
	// 2. migration0 when Direct datapath (or Proxy not enabled)
	// 3. (caller falls back to pod IP when Proxy peer binds on pod network)

	if r.isDecentralizedLiveMigrationProxyEnabled() {
		if r.isCrossClusterMigrationNetworkConfigured() {
			ips := multus.GetMigrationNetworkIPs(pod, v1.CrossClusterMigrationInterfaceName)
			if len(ips) > 0 {
				return ips
			}
		}
		// Proxy without crossClusterNetwork: peer gRPC binds on pod IP, not migration0.
		return nil
	}

	return multus.GetMigrationNetworkIPs(pod, v1.MigrationInterfaceName)
}

func (r *Reconciler) isCrossClusterMigrationNetworkConfigured() bool {
	mig := r.kv.Spec.Configuration.MigrationConfiguration
	return mig != nil && mig.CrossClusterNetwork != nil
}

// isDecentralizedLiveMigrationProxyEnabled mirrors virtconfig.DecentralizedLiveMigrationProxyEnabled:
// CrossClusterMigrationProxy feature gate + decentralizedLiveMigrationDatapath=Proxy.
func (r *Reconciler) isDecentralizedLiveMigrationProxyEnabled() bool {
	if !r.isFeatureGateEnabled(featuregate.CrossClusterMigrationProxy) {
		return false
	}
	mig := r.kv.Spec.Configuration.MigrationConfiguration
	if mig == nil || mig.DecentralizedLiveMigrationDatapath == nil {
		return false
	}
	return *mig.DecentralizedLiveMigrationDatapath == v1.DecentralizedLiveMigrationDatapathProxy
}

func (r *Reconciler) synchronizationPort() (int32, error) {
	port := util.DefaultSynchronizationPort
	if r.kv.Spec.SynchronizationPort == "" {
		return port, nil
	}
	p, err := strconv.Atoi(r.kv.Spec.SynchronizationPort)
	if err == nil && (p < 1 || p > 65535) {
		err = fmt.Errorf("out of range")
	}
	if err != nil {
		return 0, fmt.Errorf("invalid synchronizationPort %q: %v", r.kv.Spec.SynchronizationPort, err)
	}
	return int32(p), nil
}

// getSynchronizationEndpointAddress discovers an admin-labeled Ingress, Route,
// or Service and returns the externally reachable address for the
// virt-synchronization-controller peer gRPC endpoint.
// At most one resource of each type may carry the label; duplicates within a
// type return an error. Priority: Route > Ingress > Service.
// Returns ("", nil) when no labeled resource is found.
func (r *Reconciler) getSynchronizationEndpointAddress() (string, error) {
	var routes []*routev1.Route
	var routeNames []string
	for _, obj := range r.stores.SynchronizationRouteCache.List() {
		if route, ok := obj.(*routev1.Route); ok {
			routes = append(routes, route)
			routeNames = append(routeNames, route.Name)
		}
	}

	var ingresses []*networkingv1.Ingress
	var ingressNames []string
	for _, obj := range r.stores.SynchronizationIngressCache.List() {
		if ingress, ok := obj.(*networkingv1.Ingress); ok {
			ingresses = append(ingresses, ingress)
			ingressNames = append(ingressNames, ingress.Name)
		}
	}

	var services []*corev1.Service
	var serviceNames []string
	for _, obj := range r.stores.SynchronizationServiceCache.List() {
		if svc, ok := obj.(*corev1.Service); ok {
			services = append(services, svc)
			serviceNames = append(serviceNames, svc.Name)
		}
	}

	// Cache order can change between reconciles. Keep condition messages stable
	// so the same misconfiguration does not emit repeated Warning events.
	if len(routes) > 1 {
		sort.Strings(routeNames)
		return "", fmt.Errorf("found %d Routes with label %s, expected at most 1: %s",
			len(routes), v1.SynchronizationEndpointLabel, strings.Join(routeNames, ","))
	}
	if len(ingresses) > 1 {
		sort.Strings(ingressNames)
		return "", fmt.Errorf("found %d Ingresses with label %s, expected at most 1: %s",
			len(ingresses), v1.SynchronizationEndpointLabel, strings.Join(ingressNames, ","))
	}
	if len(services) > 1 {
		sort.Strings(serviceNames)
		return "", fmt.Errorf("found %d Services with label %s, expected at most 1: %s",
			len(services), v1.SynchronizationEndpointLabel, strings.Join(serviceNames, ","))
	}

	if len(routes) == 1 {
		if host := getHostFromSyncRoute(routes[0]); host != "" {
			// OpenShift Routes (and typical Ingress TLS) are exposed on 443;
			// the router forwards to the Service grpc targetPort.
			return net.JoinHostPort(host, synchronizationEndpointTLSPort), nil
		}
	}

	if len(ingresses) == 1 {
		if host := getHostFromSyncIngress(ingresses[0]); host != "" {
			// Same external TLS port assumption as Routes; see synchronizationEndpointTLSPort.
			return net.JoinHostPort(host, synchronizationEndpointTLSPort), nil
		}
	}

	if len(services) == 1 {
		port, err := r.synchronizationPort()
		if err != nil {
			return "", err
		}
		return getAddressFromSyncService(services[0], port), nil
	}

	return "", nil
}

// synchronizationEndpointTLSPort is the externally advertised port for labeled
// Route and Ingress endpoints. OpenShift Routes and common Ingress TLS
// listeners serve HTTPS on 443; HAProxy/nginx then forward to the backend
// Service port (typically the virt-synchronization-controller grpc port).
const synchronizationEndpointTLSPort = "443"

// Preferred Service port names for the sync gRPC endpoint.
const (
	synchronizationServicePortNameGRPC = "grpc"
	synchronizationServicePortNameSync = "synchronization"
)

func getHostFromSyncRoute(route *routev1.Route) string {
	for i := range route.Status.Ingress {
		if isRouteIngressAdmitted(&route.Status.Ingress[i]) && route.Status.Ingress[i].Host != "" {
			return route.Status.Ingress[i].Host
		}
	}
	// Status entries exist but none are admitted — the route was rejected.
	if len(route.Status.Ingress) > 0 {
		return ""
	}
	// No status yet (newly created); fall back to spec.Host.
	return route.Spec.Host
}

func isRouteIngressAdmitted(ingress *routev1.RouteIngress) bool {
	for _, condition := range ingress.Conditions {
		if condition.Type == routev1.RouteAdmitted {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}

func getHostFromSyncIngress(ingress *networkingv1.Ingress) string {
	for _, rule := range ingress.Spec.Rules {
		if rule.Host != "" {
			return rule.Host
		}
	}
	// Some Ingress controllers only publish the usable hostname/IP in status.
	for _, lb := range ingress.Status.LoadBalancer.Ingress {
		if lb.Hostname != "" {
			return lb.Hostname
		}
		if lb.IP != "" {
			return lb.IP
		}
	}
	return ""
}

// getAddressFromSyncService returns the advertised address for a labeled sync
// Service. Prefer an external LoadBalancer hostname/IP when present; otherwise
// fall back to the cluster-local DNS name (name.namespace.svc), which requires
// cross-cluster DNS for peer reachability.
func getAddressFromSyncService(svc *corev1.Service, defaultPort int32) string {
	port := selectSynchronizationServicePort(svc, defaultPort)
	portStr := strconv.Itoa(int(port))
	if host := getHostFromServiceLoadBalancer(svc); host != "" {
		return net.JoinHostPort(host, portStr)
	}
	host := fmt.Sprintf("%s.%s.svc", svc.Name, svc.Namespace)
	return net.JoinHostPort(host, portStr)
}

func getHostFromServiceLoadBalancer(svc *corev1.Service) string {
	for _, lb := range svc.Status.LoadBalancer.Ingress {
		if lb.Hostname != "" {
			return lb.Hostname
		}
		if lb.IP != "" {
			return lb.IP
		}
	}
	return ""
}

// selectSynchronizationServicePort picks the Service port for the sync endpoint.
// Preference order: port named "grpc", port named "synchronization", port equal
// to defaultPort, the sole port if only one is defined, otherwise defaultPort.
func selectSynchronizationServicePort(svc *corev1.Service, defaultPort int32) int32 {
	if svc == nil || len(svc.Spec.Ports) == 0 {
		return defaultPort
	}
	for _, p := range svc.Spec.Ports {
		if p.Name == synchronizationServicePortNameGRPC || p.Name == synchronizationServicePortNameSync {
			return p.Port
		}
	}
	for _, p := range svc.Spec.Ports {
		if p.Port == defaultPort {
			return p.Port
		}
	}
	if len(svc.Spec.Ports) == 1 {
		return svc.Spec.Ports[0].Port
	}
	return defaultPort
}
