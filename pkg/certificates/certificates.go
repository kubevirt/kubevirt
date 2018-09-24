package certificates

import (
	"k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/cert/triple"
	"k8s.io/client-go/util/certificate"
)

func GenerateSelfSignedCert(certsDirectory string, name string, namespace string) (certificate.FileStore, error) {
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
