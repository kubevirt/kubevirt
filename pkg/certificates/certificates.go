package certificates

import (
	"io/ioutil"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/cert/triple"
	"k8s.io/client-go/util/certificate"
)

func GenerateSelfSignedCert(name string, namespace string) (certificate.FileStore, error) {
	caKeyPair, _ := triple.NewCA("kubevirt.io")
	keyPair, _ := triple.NewServerKeyPair(
		caKeyPair,
		name+"."+namespace+".pod.cluster.local",
		name,
		namespace,
		"cluster.local",
		nil,
		nil,
	)

	certsDirectory, err := ioutil.TempDir("", "certsdir")
	if err != nil {
		return nil, err
	}
	store, err := certificate.NewFileStore(name, certsDirectory, certsDirectory, "", "")
	if err != nil {
		return nil, err
	}
	_, err = store.Update(cert.EncodeCertPEM(keyPair.Cert), cert.EncodePrivateKeyPEM(keyPair.Key))
	if err != nil {
		return nil, err
	}
	return store, nil
}

func GetNamespace() string {
	if data, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
		if ns := strings.TrimSpace(string(data)); len(ns) > 0 {
			return ns
		}
	}
	return v1.NamespaceSystem
}
