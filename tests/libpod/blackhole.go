package libpod

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"strings"
	"time"

	"github.com/onsi/ginkgo/v2"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
)

type podCommand func(context.Context, *v1.Pod, string, []string) (string, error)

type blackholeTarget struct {
	pod       *v1.Pod
	attempted bool
}

// KubernetesAPIBlackhole owns the routes injected by one spec. Assign the handle
// before calling Add, and call Recover from a nested AfterEach, before the suite's
// global configuration reset. DeferCleanup runs too late for these disruptions.
// A handle is used serially, just like the disruptive specs that own it.
type KubernetesAPIBlackhole struct {
	client            corev1.CoreV1Interface
	execute           podCommand
	container         string
	targets           []blackholeTarget
	serviceIP, family string

	pollInterval       time.Duration
	execTimeout        time.Duration
	removalTimeout     time.Duration
	replacementTimeout time.Duration
	readinessTimeout   time.Duration
}

func NewKubernetesAPIBlackhole(pods *v1.PodList, container string) *KubernetesAPIBlackhole {
	return newKubernetesAPIBlackhole(kubevirt.Client().CoreV1(), pods, container, exec.ExecuteCommandOnPodWithContext)
}

func newKubernetesAPIBlackhole(client corev1.CoreV1Interface, pods *v1.PodList, container string, execute podCommand) *KubernetesAPIBlackhole {
	b := &KubernetesAPIBlackhole{client: client, execute: execute, container: container,
		pollInterval: time.Second, execTimeout: 10 * time.Second, removalTimeout: 120 * time.Second,
		replacementTimeout: 450 * time.Second, readinessTimeout: 120 * time.Second}
	if pods != nil {
		for i := range pods.Items {
			b.targets = append(b.targets, blackholeTarget{pod: pods.Items[i].DeepCopy()})
		}
	}
	return b
}

// NeedsRecovery includes uncertain exec results, not just successful additions.
func (b *KubernetesAPIBlackhole) NeedsRecovery() bool {
	if b != nil {
		for _, target := range b.targets {
			if target.attempted {
				return true
			}
		}
	}
	return false
}

// Add records ownership before executing each mutation, including execs whose
// responses may be lost after the kernel has installed the route.
func (b *KubernetesAPIBlackhole) Add(ctx context.Context) error {
	if len(b.targets) == 0 {
		return errors.New("no virt-handler pods to blackhole")
	}
	for _, target := range b.targets {
		if err := b.validateTarget(target.pod); err != nil {
			return err
		}
		if target.attempted {
			return errors.New("blackhole handle already used")
		}
	}
	requestCtx, cancel := context.WithTimeout(ctx, b.execTimeout)
	service, err := b.client.Services("default").Get(requestCtx, "kubernetes", metav1.GetOptions{})
	cancel()
	if err != nil {
		return err
	}
	addr, err := netip.ParseAddr(service.Spec.ClusterIP)
	if err != nil {
		return fmt.Errorf("invalid Kubernetes API service IP: %w", err)
	}
	b.serviceIP, b.family = addr.String(), "-4"
	if addr.Is6() {
		b.family = "-6"
	}
	for i := range b.targets {
		target := &b.targets[i]
		pod, err := b.getPod(ctx, target.pod)
		if err != nil {
			return err
		}
		if pod.UID != target.pod.UID || pod.DeletionTimestamp != nil {
			return fmt.Errorf("pod %s changed before injection", pod.Name)
		}
		present, err := b.routePresent(ctx, pod)
		if err != nil {
			return err
		}
		if present {
			return fmt.Errorf("refusing to own pre-existing API blackhole on %s", pod.Name)
		}
		target.attempted = true
		if _, err := b.run(ctx, pod, "add", "blackhole", b.serviceIP); err != nil {
			return err
		}
	}
	return nil
}

// Remove verifies route absence without replacing pods. The health-check spec
// uses this path so replacement cannot mask a failure to recover in place.
func (b *KubernetesAPIBlackhole) Remove(ctx context.Context) error {
	if b == nil {
		return nil
	}
	var errs []error
	for _, target := range b.targets {
		if !target.attempted {
			continue
		}
		if err := b.remove(ctx, target.pod); err != nil {
			errs = append(errs, fmt.Errorf("%s/%s: %w", target.pod.Namespace, target.pod.Name, err))
			continue
		}
		pod, err := b.getPod(ctx, target.pod)
		if err != nil {
			errs = append(errs, fmt.Errorf("original handler %s is unavailable after route removal: %w", target.pod.Name, err))
		} else if pod.UID != target.pod.UID || pod.DeletionTimestamp != nil {
			errs = append(errs, fmt.Errorf("handler %s was replaced or is terminating; in-place recovery was not verified", target.pod.Name))
		}
	}
	return errors.Join(errs...)
}

func (b *KubernetesAPIBlackhole) remove(ctx context.Context, original *v1.Pod) error {
	return b.poll(ctx, b.removalTimeout, func(ctx context.Context) error {
		pod, err := b.getPod(ctx, original)
		if apierrors.IsNotFound(err) {
			return nil
		}
		if err != nil {
			return err
		}
		if pod.UID != original.UID {
			return nil
		}
		present, err := b.routePresent(ctx, pod)
		if err != nil || !present {
			return err
		}
		if _, err := b.run(ctx, pod, "del", "blackhole", b.serviceIP); err != nil {
			return err
		}
		present, err = b.routePresent(ctx, pod)
		if err != nil {
			return err
		}
		if present {
			return errors.New("API blackhole still present after deletion")
		}
		return nil
	})
}

func (b *KubernetesAPIBlackhole) run(ctx context.Context, pod *v1.Pod, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, b.execTimeout)
	defer cancel()
	command := append([]string{"ip", b.family, "route"}, args...)
	return b.execute(ctx, pod, b.container, command)
}

func (b *KubernetesAPIBlackhole) routePresent(ctx context.Context, pod *v1.Pod) (bool, error) {
	output, err := b.run(ctx, pod, "show", "type", "blackhole", "to", "exact", b.serviceIP)
	return strings.TrimSpace(output) != "", err
}

func (b *KubernetesAPIBlackhole) getPod(ctx context.Context, pod *v1.Pod) (*v1.Pod, error) {
	ctx, cancel := context.WithTimeout(ctx, b.execTimeout)
	defer cancel()
	return b.client.Pods(pod.Namespace).Get(ctx, pod.Name, metav1.GetOptions{})
}

func (b *KubernetesAPIBlackhole) validateTarget(pod *v1.Pod) error {
	owner := metav1.GetControllerOf(pod)
	if b.container != "virt-handler" || pod.Spec.HostNetwork || pod.UID == "" || pod.Spec.NodeName == "" ||
		pod.Labels["kubevirt.io"] != "virt-handler" || owner == nil || owner.APIVersion != "apps/v1" || owner.Kind != "DaemonSet" || owner.Name != "virt-handler" || owner.UID == "" {
		return fmt.Errorf("refusing blackhole recovery for unsupported pod %s/%s", pod.Namespace, pod.Name)
	}
	return nil
}

func (b *KubernetesAPIBlackhole) poll(ctx context.Context, timeout time.Duration, condition func(context.Context) error) error {
	var lastErr error
	err := wait.PollUntilContextTimeout(ctx, b.pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
		lastErr = condition(ctx)
		return lastErr == nil, nil
	})
	if err != nil {
		return fmt.Errorf("%w: last observation: %v", err, lastErr)
	}
	return nil
}

// Recover restores all affected handlers, using pod replacement only when route
// removal cannot be verified. Callers must also wait for configuration propagation
// and KubeVirt readiness before allowing global cleanup to run.
func (b *KubernetesAPIBlackhole) Recover(ctx context.Context) error {
	if b == nil {
		return nil
	}
	var errs []error
	for _, target := range b.targets {
		if !target.attempted {
			continue
		}
		original := target.pod
		if err := b.remove(ctx, original); err != nil {
			b.diagnose(ctx, original, err)
			if replacementErr := b.replace(ctx, original); replacementErr != nil {
				errs = append(errs, fmt.Errorf("%s/%s: removal: %v; replacement: %w", original.Namespace, original.Name, err, replacementErr))
				continue
			}
		}
		// Readiness failure after verified removal is not a reason to delete a pod.
		if err := b.waitReady(ctx, original); err != nil {
			b.diagnose(ctx, original, err)
			errs = append(errs, fmt.Errorf("%s/%s: recovery readiness: %w", original.Namespace, original.Name, err))
		}
	}
	return errors.Join(errs...)
}

func (b *KubernetesAPIBlackhole) replace(ctx context.Context, original *v1.Pod) error {
	if err := b.validateTarget(original); err != nil {
		return err
	}
	deletionRequested := false
	return b.poll(ctx, b.replacementTimeout, func(ctx context.Context) error {
		pod, err := b.getPod(ctx, original)
		if err != nil && !apierrors.IsNotFound(err) {
			return err
		}
		if err == nil && pod.UID == original.UID {
			if !deletionRequested && pod.DeletionTimestamp == nil {
				if err := b.validateTarget(pod); err != nil {
					return err
				}
				if metav1.GetControllerOf(pod).UID != metav1.GetControllerOf(original).UID {
					return errors.New("handler controller changed before replacement")
				}
				requestCtx, cancel := context.WithTimeout(ctx, b.execTimeout)
				err := b.client.Pods(original.Namespace).Delete(requestCtx, original.Name, metav1.DeleteOptions{
					Preconditions: &metav1.Preconditions{UID: &original.UID},
				})
				cancel()
				if err != nil && !apierrors.IsNotFound(err) {
					return err
				}
				deletionRequested = true
			}
			return fmt.Errorf("waiting for original pod UID %s to terminate", original.UID)
		}
		_, err = b.replacement(ctx, original)
		return err
	})
}

func (b *KubernetesAPIBlackhole) replacement(ctx context.Context, original *v1.Pod) (*v1.Pod, error) {
	ctx, cancel := context.WithTimeout(ctx, b.execTimeout)
	defer cancel()
	pods, err := b.client.Pods(original.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "kubevirt.io=virt-handler", FieldSelector: "spec.nodeName=" + original.Spec.NodeName,
	})
	if err != nil {
		return nil, err
	}
	owner := metav1.GetControllerOf(original)
	var candidates []*v1.Pod
	for i := range pods.Items {
		pod := &pods.Items[i]
		candidateOwner := metav1.GetControllerOf(pod)
		if pod.UID == original.UID || pod.DeletionTimestamp != nil || pod.Spec.NodeName != original.Spec.NodeName || candidateOwner == nil || candidateOwner.UID != owner.UID {
			continue
		}
		if err := b.validateTarget(pod); err != nil {
			return nil, err
		}
		candidates = append(candidates, pod)
	}
	if len(candidates) != 1 {
		return nil, fmt.Errorf("expected one replacement handler on %s, found %d", original.Spec.NodeName, len(candidates))
	}
	return candidates[0], nil
}

func (b *KubernetesAPIBlackhole) waitReady(ctx context.Context, original *v1.Pod) error {
	return b.poll(ctx, b.readinessTimeout, func(ctx context.Context) error {
		pod, err := b.getPod(ctx, original)
		if apierrors.IsNotFound(err) || (err == nil && pod.UID != original.UID) {
			pod, err = b.replacement(ctx, original)
		}
		if err != nil {
			return err
		}
		if pod.DeletionTimestamp != nil {
			return fmt.Errorf("handler %s is still terminating", pod.Name)
		}
		present, err := b.routePresent(ctx, pod)
		if err != nil {
			return err
		}
		if present {
			return fmt.Errorf("handler %s still has an API blackhole", pod.Name)
		}
		for _, condition := range pod.Status.Conditions {
			if condition.Type == v1.PodReady && condition.Status == v1.ConditionTrue {
				return b.checkAPI(ctx, pod)
			}
		}
		return fmt.Errorf("handler %s is not ready", pod.Name)
	})
}

func (b *KubernetesAPIBlackhole) diagnose(ctx context.Context, original *v1.Pod, cause error) {
	fmt.Fprintf(ginkgo.GinkgoWriter, "API blackhole recovery: pod=%s/%s UID=%s node=%s destination=%s: %v\n",
		original.Namespace, original.Name, original.UID, original.Spec.NodeName, b.serviceIP, cause)
	pod, err := b.getPod(ctx, original)
	if err != nil {
		fmt.Fprintf(ginkgo.GinkgoWriter, "handler state unavailable: %v\n", err)
		return
	}
	fmt.Fprintf(ginkgo.GinkgoWriter, "current UID=%s deletion=%v containers=%+v\n", pod.UID, pod.DeletionTimestamp, pod.Status.ContainerStatuses)
	routes, err := b.run(ctx, pod, "show")
	fmt.Fprintf(ginkgo.GinkgoWriter, "handler routes: %s; error: %v\n", routes, err)
	for _, previous := range []bool{false, true} {
		requestCtx, cancel := context.WithTimeout(ctx, b.execTimeout)
		lines, limit := int64(40), int64(16384)
		logs, err := b.client.Pods(pod.Namespace).GetLogs(pod.Name, &v1.PodLogOptions{Container: b.container, Previous: previous, TailLines: &lines, LimitBytes: &limit}).DoRaw(requestCtx)
		cancel()
		fmt.Fprintf(ginkgo.GinkgoWriter, "handler logs (previous=%t): %s; error: %v\n", previous, logs, err)
	}
	requestCtx, cancel := context.WithTimeout(ctx, b.execTimeout)
	defer cancel()
	events, err := b.client.Events(pod.Namespace).List(requestCtx, metav1.ListOptions{FieldSelector: "involvedObject.uid=" + string(pod.UID)})
	if err != nil {
		fmt.Fprintf(ginkgo.GinkgoWriter, "handler events unavailable: %v\n", err)
		return
	}
	for _, event := range events.Items {
		fmt.Fprintf(ginkgo.GinkgoWriter, "handler event %s: %s\n", event.Reason, event.Message)
	}
}

// Probe the captured API destination from the handler's network namespace. Feed
// the authorization header over stdin rather than exposing its token in argv.
func (b *KubernetesAPIBlackhole) checkAPI(ctx context.Context, pod *v1.Pod) error {
	ctx, cancel := context.WithTimeout(ctx, b.execTimeout)
	defer cancel()
	const command = `token=$(cat /var/run/secrets/kubernetes.io/serviceaccount/token) || exit 1
printf 'Authorization: Bearer %s\n' "$token" | curl --fail --silent --show-error --max-time 5 --cacert /var/run/secrets/kubernetes.io/serviceaccount/ca.crt --header @- "$1/version" >/dev/null`
	_, err := b.execute(ctx, pod, b.container, []string{"sh", "-c", command, "blackhole-api-check", "https://" + net.JoinHostPort(b.serviceIP, "443")})
	if err != nil {
		return fmt.Errorf("handler %s cannot reach Kubernetes API: %w", pod.Name, err)
	}
	return nil
}
