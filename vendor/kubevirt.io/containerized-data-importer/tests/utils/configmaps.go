package utils

import (
	"net/url"
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"kubevirt.io/containerized-data-importer/pkg/common"
	"kubevirt.io/containerized-data-importer/pkg/util"
)

// CopyRegistryCertConfigMap copies the test registry configmap, it assumes the Registry host is in the CDI namespace
func CopyRegistryCertConfigMap(client kubernetes.Interface, destNamespace, cdiNamespace string) (string, error) {
	n, err := CopyConfigMap(client, cdiNamespace, RegistryCertConfigMap, destNamespace, "")
	if err != nil {
		return "", err
	}
	return n, nil
}

// CopyFileHostCertConfigMap copies the test file host configmap, it assumes the File host is in the CDI namespace
func CopyFileHostCertConfigMap(client kubernetes.Interface, destNamespace, cdiNamespace string) (string, error) {
	n, err := CopyConfigMap(client, cdiNamespace, FileHostCertConfigMap, destNamespace, "")
	if err != nil {
		return "", err
	}
	return n, nil
}

// CopyConfigMap copies a ConfigMap
func CopyConfigMap(client kubernetes.Interface, srcNamespace, srcName, destNamespace, destName string) (string, error) {
	src, err := client.CoreV1().ConfigMaps(srcNamespace).Get(srcName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	if destName == "" {
		destName = srcName + "-" + strings.ToLower(util.RandAlphaNum(8))
	}

	dst := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: destName,
		},
		Data: src.Data,
	}

	err = client.CoreV1().ConfigMaps(destNamespace).Delete(destName, nil)
	if err != nil && !errors.IsNotFound(err) {
		return "", err
	}

	_, err = client.CoreV1().ConfigMaps(destNamespace).Create(dst)
	if err != nil {
		return "", err
	}

	return destName, nil
}

const insecureRegistryKey = "test-registry"

// SetInsecureRegistry sets the configmap entry to mark the registry as okay to be insecure
func SetInsecureRegistry(client kubernetes.Interface, cdiNamespace, registryURL string) error {
	cm, err := client.CoreV1().ConfigMaps(cdiNamespace).Get(common.InsecureRegistryConfigMap, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if cm.Data == nil {
		cm.Data = map[string]string{}
	}

	parsedURL, err := url.Parse(registryURL)
	if err != nil {
		return err
	}

	cm.Data[insecureRegistryKey] = parsedURL.Host

	_, err = client.CoreV1().ConfigMaps(cdiNamespace).Update(cm)
	if err != nil {
		return err
	}

	return nil
}

// ClearInsecureRegistry undoes whatever SetInsecureRegistry does
func ClearInsecureRegistry(client kubernetes.Interface, cdiNamespace string) error {
	cm, err := client.CoreV1().ConfigMaps(cdiNamespace).Get(common.InsecureRegistryConfigMap, metav1.GetOptions{})
	if err != nil {
		return err
	}

	delete(cm.Data, insecureRegistryKey)

	_, err = client.CoreV1().ConfigMaps(cdiNamespace).Update(cm)
	if err != nil {
		return err
	}

	return nil
}
