package main

import (
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/client/clientset/versioned"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/operatorclient"
)

const (
	pollInterval = 1 * time.Second
	pollDuration = 5 * time.Minute
)

type checkResourceFunc func() error
type deleteResourceFunc func() error

func cleanup(logger *logrus.Logger, c operatorclient.ClientInterface, crc versioned.Interface) {
	if err := waitForDelete(checkCatalogSource(crc, "olm-operators"), deleteCatalogSource(crc, "olm-operators")); err != nil {
		logger.WithError(err).Fatal("couldn't clean previous release")
	}

	if err := waitForDelete(checkConfigMap(c, "olm-operators"), deleteConfigMap(c, "olm-operators")); err != nil {
		logger.WithError(err).Fatal("couldn't clean previous release")
	}

	if err := waitForDelete(checkSubscription(crc, "packageserver"), deleteSubscription(crc, "packageserver")); err != nil {
		logger.WithError(err).Fatal("couldn't clean previous release")
	}

	if err := waitForDelete(checkClusterServiceVersion(crc, "packageserver.v0.10.0"), deleteClusterServiceVersion(crc, "packageserver.v0.10.0")); err != nil {
		logger.WithError(err).Fatal("couldn't clean previous release")
	}

	if err := waitForDelete(checkClusterServiceVersion(crc, "packageserver.v0.9.0"), deleteClusterServiceVersion(crc, "packageserver.v0.9.0")); err != nil {
		logger.WithError(err).Fatal("couldn't clean previous release")
	}
}

func waitForDelete(checkResource checkResourceFunc, deleteResource deleteResourceFunc) error {
	if err := checkResource(); err != nil && errors.IsNotFound(err) {
		return nil
	}
	if err := deleteResource(); err != nil {
		return err
	}
	var err error
	err = wait.Poll(pollInterval, pollDuration, func() (bool, error) {
		err := checkResource()
		if errors.IsNotFound(err) {
			return true, nil
		}
		if err != nil {
			return false, err
		}
		return false, nil
	})

	return err
}

func checkClusterServiceVersion(crc versioned.Interface, name string) checkResourceFunc {
	return func() error {
		_, err := crc.OperatorsV1alpha1().ClusterServiceVersions(*namespace).Get(name, metav1.GetOptions{})
		return err
	}
}

func deleteClusterServiceVersion(crc versioned.Interface, name string) deleteResourceFunc {
	return func() error {
		return crc.OperatorsV1alpha1().ClusterServiceVersions(*namespace).Delete(name, metav1.NewDeleteOptions(0))
	}
}

func checkSubscription(crc versioned.Interface, name string) checkResourceFunc {
	return func() error {
		_, err := crc.OperatorsV1alpha1().Subscriptions(*namespace).Get(name, metav1.GetOptions{})
		return err
	}
}

func deleteSubscription(crc versioned.Interface, name string) deleteResourceFunc {
	return func() error {
		return crc.OperatorsV1alpha1().Subscriptions(*namespace).Delete(name, metav1.NewDeleteOptions(0))
	}
}

func checkConfigMap(c operatorclient.ClientInterface, name string) checkResourceFunc {
	return func() error {
		_, err := c.KubernetesInterface().CoreV1().ConfigMaps(*namespace).Get(name, metav1.GetOptions{})
		return err
	}
}

func deleteConfigMap(c operatorclient.ClientInterface, name string) deleteResourceFunc {
	return func() error {
		return c.KubernetesInterface().CoreV1().ConfigMaps(*namespace).Delete(name, metav1.NewDeleteOptions(0))
	}
}

func checkCatalogSource(crc versioned.Interface, name string) checkResourceFunc {
	return func() error {
		_, err := crc.OperatorsV1alpha1().CatalogSources(*namespace).Get(name, metav1.GetOptions{})
		return err
	}
}

func deleteCatalogSource(crc versioned.Interface, name string) deleteResourceFunc {
	return func() error {
		return crc.OperatorsV1alpha1().CatalogSources(*namespace).Delete(name, metav1.NewDeleteOptions(0))
	}
}
