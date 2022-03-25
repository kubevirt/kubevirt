module kubevirt.io/kubevirt/staging/src/kubevirt.io/client-go/examples/listvms

go 1.12

replace k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190221213512-86fb29eff628

require (
	github.com/spf13/pflag v1.0.3
	golang.org/x/crypto v0.0.0-20220321153916-2c7772ba3064 // indirect
	golang.org/x/net v0.0.0-20220225172249-27dd8689420f // indirect
	golang.org/x/sys v0.0.0-20220319134239-a9b59b0215f8 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	k8s.io/apimachinery v0.20.2
	kubevirt.io/client-go v0.19.0
)
