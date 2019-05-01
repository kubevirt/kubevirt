package utils

import (
	"io/ioutil"
	"os"
	"path"

	"github.com/pkg/errors"

	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/klog"
)

// CreateCertForTestService creates a TLS key/cert for a service, writes them to files
// and creates a config map containing the cert
func CreateCertForTestService(namespace, serviceName, configMapName, certDir, certFileName, keyFileName string) error {
	klog.Info("Creating key/certificate")

	config, err := rest.InClusterConfig()
	if err != nil {
		return errors.Wrap(err, "Error creating rest config")
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return errors.Wrap(err, "Error creating kubernetes client")
	}

	if err := os.MkdirAll(certDir, 0777); err != nil {
		return errors.Wrapf(err, "Error making %s", certDir)
	}

	namespacedName := serviceName + "." + namespace

	certBytes, keyBytes, err := certutil.GenerateSelfSignedCertKey(serviceName, nil, []string{namespacedName, namespacedName + ".svc"})
	if err != nil {
		return errors.Wrap(err, "Error generating key/cert")
	}

	cm := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: configMapName,
		},
		Data: map[string]string{
			certFileName: string(certBytes),
		},
	}

	stored, err := clientset.CoreV1().ConfigMaps(namespace).Get(configMapName, metav1.GetOptions{})
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			return errors.Wrapf(err, "Error getting configmap %s", configMapName)
		}

		_, err := clientset.CoreV1().ConfigMaps(namespace).Create(cm)
		if err != nil {
			return err
		}

	} else {
		cpy := stored.DeepCopy()
		cpy.Data = cm.Data
		_, err := clientset.CoreV1().ConfigMaps(namespace).Update(cpy)
		if err != nil {
			return err
		}
	}

	if err = ioutil.WriteFile(path.Join(certDir, certFileName), certBytes, 0644); err != nil {
		return err
	}

	if err = ioutil.WriteFile(path.Join(certDir, keyFileName), keyBytes, 0600); err != nil {
		return err
	}

	klog.Info("Successfully created key/certificate")
	return nil
}
