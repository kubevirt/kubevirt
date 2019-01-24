package main

import (
	"context"
	"os"
	"time"

	"github.com/golang/glog"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/tools/record"

	"kubevirt.io/containerized-data-importer/pkg/common"
	"kubevirt.io/containerized-data-importer/pkg/util"
)

const (
	configMapName = "cdi-controller-leader-election-helper"
	componentName = "cdi-controller"
)

func startLeaderElection(ctx context.Context, config *rest.Config, leaderFunc func()) error {
	client := kubernetes.NewForConfigOrDie(config)
	namespace := util.GetNamespace()

	// create manually so it has CDI component label
	err := createConfigMap(client, namespace, configMapName)
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return err
	}

	resourceLock, err := createResourceLock(client, namespace, configMapName)
	if err != nil {
		return err
	}

	leaderElector, err := createLeaderElector(resourceLock, leaderFunc)
	if err != nil {
		return err
	}

	glog.Info("Attempting to acquire leader lease")
	go leaderElector.Run(ctx)

	return nil
}

func createConfigMap(client kubernetes.Interface, namespace, name string) error {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				common.CDIComponentLabel: componentName,
			},
		},
	}

	_, err := client.CoreV1().ConfigMaps(namespace).Create(cm)
	return err
}

func createResourceLock(client kubernetes.Interface, namespace, name string) (resourcelock.Interface, error) {
	// Leader id, needs to be unique
	id, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	id = id + "_" + string(uuid.NewUUID())

	return resourcelock.New(resourcelock.ConfigMapsResourceLock,
		namespace,
		name,
		client.CoreV1(),
		resourcelock.ResourceLockConfig{
			Identity:      id,
			EventRecorder: createEventRecorder(client, namespace, componentName),
		})
}

func createLeaderElector(resourceLock resourcelock.Interface, leaderFunc func()) (*leaderelection.LeaderElector, error) {
	return leaderelection.NewLeaderElector(leaderelection.LeaderElectionConfig{
		Lock:          resourceLock,
		LeaseDuration: 15 * time.Second,
		RenewDeadline: 10 * time.Second,
		RetryPeriod:   2 * time.Second,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(_ context.Context) {
				glog.Info("Successfully acquired leadership lease")
				leaderFunc()
			},
			OnStoppedLeading: func() {
				glog.Fatal("NO LONGER LEADER, EXITING")
			},
		},
	})
}

func createEventRecorder(client kubernetes.Interface, namespace, name string) record.EventRecorder {
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: client.CoreV1().Events(namespace)})
	return eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: componentName})
}
