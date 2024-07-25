package util

// tests.NamespaceTestDefault is the default namespace, to test non-infrastructure related KubeVirt objects.
var NamespaceTestDefault = "kubevirt-test-default"

func PanicOnError(err error) {
	if err != nil {
		panic(err)
	}
}
