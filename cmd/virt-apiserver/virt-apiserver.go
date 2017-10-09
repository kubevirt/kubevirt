package main

import (
	"flag"
	"fmt"
	"net"
	"net/url"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/util/wait"
	genericapiserver "k8s.io/apiserver/pkg/server"
	genericoptions "k8s.io/apiserver/pkg/server/options"
	restclient "k8s.io/client-go/rest"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/logging"
	"kubevirt.io/kubevirt/pkg/virt-apiserver/apiserver"
)

const defaultEtcdUrl = "http://127.0.0.1:2379"
const defaultEtcdPathPrefix = "/registry"
const etcdRetryLimit = 600
const etcdRetryInterval = 1 * time.Second
const connectionTimeout = 1 * time.Second

type etcdConnection struct {
	ServerList []string
}

type etcdHealthResponse struct {
	health bool
}

func etcdReady(serverUri string) bool {
	if connUrl, err := url.Parse(serverUri); err == nil {
		if conn, err := net.DialTimeout("tcp", connUrl.Host, connectionTimeout); err == nil {
			defer conn.Close()
			return true
		}
	}
	return false
}

func (e etcdConnection) checkEtcdServers() (done bool, err error) {
	for _, serverUri := range e.ServerList {
		if etcdReady(serverUri) {
			return true, nil
		}
	}
	return false, nil
}
>>>>>>> 43f8da9... Wait for etcd before starting apiserver

type ApiServerOptions struct {
	RecommendedOptions *genericoptions.RecommendedOptions
	Admission          *genericoptions.AdmissionOptions

	host string
	port int
}

func (o ApiServerOptions) Complete() error {
	return nil
}

func (o ApiServerOptions) Validate(args []string) error {
	return nil
}

func (o ApiServerOptions) Config() (*apiserver.Config, error) {
	if err := o.RecommendedOptions.SecureServing.MaybeDefaultWithSelfSignedCerts("localhost", nil, []net.IP{net.ParseIP(o.host)}); err != nil {
		return nil, fmt.Errorf("error creating self-signed certificates: %v", err)
	}

	serverConfig := genericapiserver.NewConfig(apiserver.Codecs)

	serverConfig.LoopbackClientConfig = &restclient.Config{
		Host:            fmt.Sprintf("%s:%d", o.host, o.port),
		TLSClientConfig: restclient.TLSClientConfig{
		//CAFile: o.caFile,
		},
	}

	if err := o.RecommendedOptions.ApplyTo(serverConfig); err != nil {
		return nil, err
	}
	if err := o.Admission.ApplyTo(serverConfig); err != nil {
		return nil, err
	}

	config := &apiserver.Config{
		GenericConfig: serverConfig,
	}
	return config, nil
}

func (o ApiServerOptions) RunApiServer(stopCh <-chan struct{}) error {
	config, err := o.Config()
	if err != nil {
		return err
	}

	server, err := config.Complete().New()
	if err != nil {
		return err
	}
	return server.GenericAPIServer.PrepareRun().Run(stopCh)
}

func NewVirtApiServerOptions() *ApiServerOptions {
	recommended := genericoptions.NewRecommendedOptions(defaultEtcdPathPrefix, apiserver.Scheme, apiserver.Codecs.LegacyCodec(v1.SchemeGroupVersion))
	admission := genericoptions.NewAdmissionOptions()

	options := &ApiServerOptions{
		RecommendedOptions: recommended,
		Admission:          admission,
	}
	return options
}

func (o *ApiServerOptions) addFlags(flags *pflag.FlagSet) {
	o.RecommendedOptions.AddFlags(flags)
	o.Admission.AddFlags(flags)
}

func NewVirtApiServerCommand(stopCh <-chan struct{}) *cobra.Command {
	o := NewVirtApiServerOptions()

	cmd := &cobra.Command{
		Short: "Launch a KubeVirt API server",
		Long:  "Launch a KubeVirt API server",
		RunE: func(c *cobra.Command, args []string) error {
			if err := o.Complete(); err != nil {
				return err
			}
			if err := o.Validate(args); err != nil {
				return err
			}
			if err := o.RunApiServer(stopCh); err != nil {
				return err
			}
			return nil
		},
	}
	flags := cmd.Flags()
	o.addFlags(flags)
	return cmd
}

func main() {
	logging.InitializeLogging("virt-apiserver")

	cmd := NewVirtApiServerCommand(wait.NeverStop)
	cmd.Flags().AddGoFlagSet(flag.CommandLine)

	etcdServers := []string{defaultEtcdUrl}
	//etcdServers := strings.Split(etcdServerFlag.Value.String(), ",")
	if err := wait.PollImmediate(etcdRetryInterval, etcdRetryLimit*etcdRetryInterval,
		etcdConnection{ServerList: etcdServers}.checkEtcdServers); err != nil {
		logging.DefaultLogger().Error().Reason(err).Msg("Cannot establish etcd connection")
		panic("Cannot establish etcd connection")
	}
	logging.DefaultLogger().Info().Msg("Established connection with etcd")

	logging.DefaultLogger().Info().Msg("Launching KubeVirt API Server")

	if err := cmd.Execute(); err != nil {
		logging.DefaultLogger().Error().Reason(err).Msg("Unexpected server halt")
	}
}
