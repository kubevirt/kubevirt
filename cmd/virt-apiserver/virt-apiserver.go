package main

import (
	"flag"
	"fmt"
	"net"

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

//FIXME: is this defined already somewhere else?
const defaultEtcdPathPrefix = "/registry/kubevirt.io"

type ApiServerOptions struct {
	RecommendedOptions *genericoptions.RecommendedOptions
	Admission          *genericoptions.AdmissionOptions

	host string
	port int
	//caFile   string
	//certFile string
	//keyFile  string
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

	//FIXME: if the new code layout works, this can all be removed
	/*	// parse command line options
		//swaggerui := flag.String("swagger-ui", "third_party/swagger-ui", "swagger-ui location")
		host := flag.String("listen", "0.0.0.0", "Address to listen on")

		port := flag.Int("port", 8183, "Port to listen on")
		//etcdServer := flag.String("etcd-servers", "http://127.0.0.1:2379", "URL to etcd server")
		//caFile := flag.String("client-ca-file", "/etc/kubernetes/pki/ca.crt", "Client CA certificate path")
		//certFile := flag.String("tls-cert-file", "/etc/kubernetes/pki/apiserver.crt", "Client certificate path")
		//keyFile := flag.String("tls-private-key-file", "/etc/kubernetes/pki/apiserver.key", "Client key path")

		recommended := genericoptions.NewRecommendedOptions(defaultEtcdPathPrefix, apiserver.Scheme, apiserver.Codecs.LegacyCodec(v1.SchemeGroupVersion))
		recommended.SecureServing = genericoptions.NewSecureServingOptions()
		recommended.SecureServing.BindPort = *port
		recommended.AddFlags(flag.CommandLine)

		admission := genericoptions.NewAdmissionOptions()
		admission.AddFlags(flag.CommandLine)

		flag.Parse()

		etcdServers, err := flag.CommandLine.GetStringSlice("etcd-servers")
		if err != nil {
			logging.DefaultLogger().Error().Reason(err).Msg("Unable to obtain etcd server list")
		}
		recommended.Etcd.StorageConfig.ServerList = etcdServers

		// FIXME: not sure if this belongs here
		apiserver.Init()

		options := ApiServerOptions{
			host: *host,
			port: *port,
			//caFile:             *caFile,
			//certFile:           *certFile,
			//keyFile:            *keyFile,
			RecommendedOptions: recommended,
			Admission:          admission,
		}

		if err := options.RunApiServer(wait.NeverStop); err != nil {
			logging.DefaultLogger().Error().Reason(err).Msg("Unexpected server halt")
		}
	*/
	cmd := NewVirtApiServerCommand(wait.NeverStop)
	cmd.Flags().AddGoFlagSet(flag.CommandLine)
	if err := cmd.Execute(); err != nil {
		logging.DefaultLogger().Error().Reason(err).Msg("Unexpected server halt")
	}
}
