package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/golang/glog"
	"github.com/pkg/errors"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"kubevirt.io/containerized-data-importer/pkg/uploadproxy"
)

const (
	// Default port that api listens on.
	defaultPort = 8443

	// Default address api listens on.
	defaultHost = "0.0.0.0"
)

var (
	configPath string
	masterURL  string
	verbose    string
)

func init() {
	// flags
	flag.StringVar(&configPath, "kubeconfig", os.Getenv("KUBECONFIG"), "(Optional) Overrides $KUBECONFIG")
	flag.StringVar(&masterURL, "server", "", "(Optional) URL address of a remote api server.  Do not set for local clusters.")
	flag.Parse()

	// get the verbose level so it can be passed to the importer pod
	defVerbose := fmt.Sprintf("%d", 1) // note flag values are strings
	verbose = defVerbose
	// visit actual flags passed in and if passed check -v and set verbose
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "v" {
			verbose = f.Value.String()
		}
	})
	if verbose == defVerbose {
		glog.V(1).Infof("Note: increase the -v level in the api deployment for more detailed logging, eg. -v=%d or -v=%d\n", 2, 3)
	}
}

func main() {
	defer glog.Flush()

	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, configPath)
	if err != nil {
		glog.Fatalf("Unable to get kube config: %v\n", errors.WithStack(err))
	}
	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		glog.Fatalf("Unable to get kube client: %v\n", errors.WithStack(err))
	}
	apiServerPublicKey, err := getAPIServerPublicKey()
	if err != nil {
		glog.Fatalf("Unable to get apiserver public key %v\n", errors.WithStack(err))
	}

	uploadProxy, err := uploadproxy.NewUploadProxy(defaultHost,
		defaultPort,
		apiServerPublicKey,
		os.Getenv("UPLOAD_SERVER_CLIENT_KEY"),
		os.Getenv("UPLOAD_SERVER_CLIENT_CERT"),
		os.Getenv("UPLOAD_SERVER_CA_CERT"),
		os.Getenv("SERVICE_TLS_KEY"),
		os.Getenv("SERVICE_TLS_CERT"),
		client)
	if err != nil {
		glog.Fatalf("UploadProxy failed to initialize: %v\n", errors.WithStack(err))
	}

	err = uploadProxy.Start()
	if err != nil {
		glog.Fatalf("TLS server failed: %v\n", errors.WithStack(err))
	}
}

func getAPIServerPublicKey() (string, error) {
	const envName = "APISERVER_PUBLIC_KEY"
	val, ok := os.LookupEnv(envName)
	if !ok {
		return "", errors.Errorf("%s not defined", envName)
	}
	return val, nil
}
