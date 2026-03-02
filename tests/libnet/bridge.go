package libnet

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libregistry"
	"kubevirt.io/kubevirt/tests/testsuite"
)

const (
	daemonSetReadyTimeout = 2 * time.Minute
	daemonSetPollInterval = 2 * time.Second
)

func SetupBridgeAsMaster(bridgeName, iface string) (*appsv1.DaemonSet, error) {
	ds := createBridgeDaemonSet(bridgeName, iface)

	client := kubevirt.Client()
	var err error
	ds, err = client.AppsV1().DaemonSets(ds.Namespace).Create(context.Background(), ds, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	return waitForDaemonSetReady(ds)
}

func createBridgeDaemonSet(bridgeName, iface string) *appsv1.DaemonSet {
	initCheckCmd := fmt.Sprintf(
		"nsenter -t 1 -n -m -- ip link show %s > /dev/null 2>&1 || "+
			"{ echo 'interface %s not found on node' > /dev/termination-log; exit 1; }; "+
			"nsenter -t 1 -n -m -- ip link show %s > /dev/null 2>&1 && "+
			"{ echo 'bridge %s already exists on node' > /dev/termination-log; exit 1; }; "+
			"exit 0",
		iface, iface, bridgeName, bridgeName,
	)
	setupCmd := fmt.Sprintf(
		"nsenter -t 1 -n -m -- ip link add %s type bridge && "+
			"nsenter -t 1 -n -m -- ip link set %s master %s && "+
			"nsenter -t 1 -n -m -- ip link set %s up && tail -f /dev/null",
		bridgeName, iface, bridgeName, bridgeName,
	)

	image := libregistry.GetUtilityImageFromRegistry("vm-killer")
	labels := map[string]string{
		v1.AppLabel: "test",
		"app":       bridgeName + "-setup",
	}

	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: bridgeName + "-setup-",
			Namespace:    testsuite.NamespacePrivileged,
			Labels:       labels,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					HostNetwork: true,
					HostPID:     true,
					SecurityContext: &corev1.PodSecurityContext{
						RunAsUser: pointer.P(int64(0)),
					},
					InitContainers: []corev1.Container{{
						Name:    "iface-check",
						Image:   image,
						Command: []string{"/bin/bash", "-c"},
						Args:    []string{initCheckCmd},
						SecurityContext: &corev1.SecurityContext{
							Privileged: pointer.P(true),
						},
					}},
					Containers: []corev1.Container{{
						Name:    "bridge-setup",
						Image:   image,
						Command: []string{"/bin/bash", "-c"},
						Args:    []string{setupCmd},
						Lifecycle: &corev1.Lifecycle{
							PreStop: &corev1.LifecycleHandler{
								Exec: &corev1.ExecAction{
									Command: []string{
										"/bin/bash", "-c",
										fmt.Sprintf(
											"nsenter -t 1 -n -m -- ip link set %s nomaster && nsenter -t 1 -n -m -- ip link delete %s",
											iface, bridgeName,
										),
									},
								},
							},
						},
						SecurityContext: &corev1.SecurityContext{
							Privileged: pointer.P(true),
						},
					}},
				},
			},
		},
	}
}

func waitForDaemonSetReady(ds *appsv1.DaemonSet) (*appsv1.DaemonSet, error) {
	client := kubevirt.Client()
	ctx, cancel := context.WithTimeout(context.Background(), daemonSetReadyTimeout)
	defer cancel()

	for {
		updated, err := client.AppsV1().DaemonSets(ds.Namespace).Get(ctx, ds.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		if updated.Status.DesiredNumberScheduled > 0 && updated.Status.DesiredNumberScheduled == updated.Status.NumberReady {
			return ds, nil
		}

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timed out waiting for DaemonSet %s to be ready", ds.Name)
		case <-time.After(daemonSetPollInterval):
		}
	}
}
