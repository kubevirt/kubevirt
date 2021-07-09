package certificates

import (
	"time"

	"k8s.io/client-go/util/certificate"

	"kubevirt.io/kubevirt/pkg/certificates/triple"
	"kubevirt.io/kubevirt/pkg/certificates/triple/cert"
)

func GenerateSelfSignedCert(certsDirectory string, name string, namespace string) (certificate.FileStore, error) {
	caKeyPair, _ := triple.NewCA("kubevirt.io", time.Hour*24*7)
	keyPair, _ := triple.NewServerKeyPair(
		caKeyPair,
		name+"."+namespace+".pod.cluster.local",
		name,
		namespace,
		"cluster.local",
		nil,
		nil,
		time.Hour*24,
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
