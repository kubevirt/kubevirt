package controllercmd

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/apiserver/pkg/server"
	"k8s.io/apiserver/pkg/util/logs"
	"k8s.io/klog"

	operatorv1alpha1 "github.com/openshift/api/operator/v1alpha1"

	"github.com/openshift/library-go/pkg/config/configdefaults"
	"github.com/openshift/library-go/pkg/crypto"
	"github.com/openshift/library-go/pkg/serviceability"

	// for metrics
	_ "github.com/openshift/library-go/pkg/controller/metrics"
)

// ControllerCommandConfig holds values required to construct a command to run.
type ControllerCommandConfig struct {
	componentName string
	startFunc     StartFunc
	version       version.Info

	basicFlags *ControllerFlags
}

// NewControllerConfig returns a new ControllerCommandConfig which can be used to wire up all the boiler plate of a controller
// TODO add more methods around wiring health checks and the like
func NewControllerCommandConfig(componentName string, version version.Info, startFunc StartFunc) *ControllerCommandConfig {
	return &ControllerCommandConfig{
		startFunc:     startFunc,
		componentName: componentName,
		version:       version,

		basicFlags: NewControllerFlags(),
	}
}

// NewCommand returns a new command that a caller must set the Use and Descriptions on.  It wires default log, profiling,
// leader election and other "normal" behaviors.
// Deprecated: Use the NewCommandWithContext instead, this is here to be less disturbing for existing usages.
func (c *ControllerCommandConfig) NewCommand() *cobra.Command {
	return c.NewCommandWithContext(context.TODO())

}

// NewCommandWithContext returns a new command that a caller must set the Use and Descriptions on.  It wires default log, profiling,
// leader election and other "normal" behaviors.
// The context passed will be passed down to controller loops and observers and cancelled on SIGTERM and SIGINT signals.
func (c *ControllerCommandConfig) NewCommandWithContext(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Run: func(cmd *cobra.Command, args []string) {
			// boiler plate for the "normal" command
			rand.Seed(time.Now().UTC().UnixNano())
			logs.InitLogs()

			// handle SIGTERM and SIGINT by cancelling the context.
			shutdownCtx, cancel := context.WithCancel(ctx)
			shutdownHandler := server.SetupSignalHandler()
			go func() {
				defer cancel()
				<-shutdownHandler
				klog.Infof("Received SIGTERM or SIGINT signal, shutting down controller.")
			}()

			defer logs.FlushLogs()
			defer serviceability.BehaviorOnPanic(os.Getenv("OPENSHIFT_ON_PANIC"), c.version)()
			defer serviceability.Profile(os.Getenv("OPENSHIFT_PROFILE")).Stop()

			serviceability.StartProfiler()

			if err := c.basicFlags.Validate(); err != nil {
				klog.Fatal(err)
			}

			if err := c.StartController(shutdownCtx); err != nil {
				klog.Fatal(err)
			}
		},
	}

	c.basicFlags.AddFlags(cmd)

	return cmd
}

// Config returns the configuration of this command. Use StartController if you don't need to customize the default operator.
// This method does not modify the receiver.
func (c *ControllerCommandConfig) Config() (*unstructured.Unstructured, *operatorv1alpha1.GenericOperatorConfig, []byte, error) {
	configContent, unstructuredConfig, err := c.basicFlags.ToConfigObj()
	if err != nil {
		return nil, nil, nil, err
	}
	config := &operatorv1alpha1.GenericOperatorConfig{}
	if unstructuredConfig != nil {
		// make a copy we can mutate
		configCopy := unstructuredConfig.DeepCopy()
		// force the config to our version to read it
		configCopy.SetGroupVersionKind(operatorv1alpha1.GroupVersion.WithKind("GenericOperatorConfig"))
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(configCopy.Object, config); err != nil {
			return nil, nil, nil, err
		}
	}
	return unstructuredConfig, config, configContent, nil
}

func hasServiceServingCerts(certDir string) bool {
	if _, err := os.Stat(filepath.Join(certDir, "tls.crt")); os.IsNotExist(err) {
		return false
	}
	if _, err := os.Stat(filepath.Join(certDir, "tls.key")); os.IsNotExist(err) {
		return false
	}
	return true
}

// AddDefaultRotationToConfig starts the provided builder with the default rotation set (config + serving info). Use StartController if
// you do not need to customize the controller builder. This method modifies config with self-signed default cert locations if
// necessary.
func (c *ControllerCommandConfig) AddDefaultRotationToConfig(config *operatorv1alpha1.GenericOperatorConfig, configContent []byte) (map[string][]byte, []string, error) {
	certDir := "/var/run/secrets/serving-cert"

	observedFiles := []string{
		c.basicFlags.ConfigFile,
		// We observe these, so we they are created or modified by service serving cert signer, we can react and restart the process
		// that will pick these up instead of generating the self-signed certs.
		// NOTE: We are not observing the temporary, self-signed certificates.
		filepath.Join(certDir, "tls.crt"),
		filepath.Join(certDir, "tls.key"),
	}
	// startingFileContent holds hardcoded starting content.  If we generate our own certificates, then we want to specify empty
	// content to avoid a starting race.  When we consume them, the race is really about as good as we can do since we don't know
	// what's actually been read.
	startingFileContent := map[string][]byte{
		c.basicFlags.ConfigFile: configContent,
	}

	// if we don't have any serving cert/key pairs specified and the defaults are not present, generate a self-signed set
	// TODO maybe this should be optional?  It's a little difficult to come up with a scenario where this is worse than nothing though.
	if len(config.ServingInfo.CertFile) == 0 && len(config.ServingInfo.KeyFile) == 0 {
		servingInfoCopy := config.ServingInfo.DeepCopy()
		configdefaults.SetRecommendedHTTPServingInfoDefaults(servingInfoCopy)

		if hasServiceServingCerts(certDir) {
			klog.Infof("Using service-serving-cert provided certificates")
			config.ServingInfo.CertFile = filepath.Join(certDir, "tls.crt")
			config.ServingInfo.KeyFile = filepath.Join(certDir, "tls.key")
		} else {
			klog.Warningf("Using insecure, self-signed certificates")
			temporaryCertDir, err := ioutil.TempDir("", "serving-cert-")
			if err != nil {
				return nil, nil, err
			}
			signerName := fmt.Sprintf("%s-signer@%d", c.componentName, time.Now().Unix())
			ca, err := crypto.MakeSelfSignedCA(
				filepath.Join(temporaryCertDir, "serving-signer.crt"),
				filepath.Join(temporaryCertDir, "serving-signer.key"),
				filepath.Join(temporaryCertDir, "serving-signer.serial"),
				signerName,
				0,
			)
			if err != nil {
				return nil, nil, err
			}
			certDir = temporaryCertDir

			// force the values to be set to where we are writing the certs
			config.ServingInfo.CertFile = filepath.Join(certDir, "tls.crt")
			config.ServingInfo.KeyFile = filepath.Join(certDir, "tls.key")
			// nothing can trust this, so we don't really care about hostnames
			servingCert, err := ca.MakeServerCert(sets.NewString("localhost"), 30)
			if err != nil {
				return nil, nil, err
			}
			if err := servingCert.WriteCertConfigFile(config.ServingInfo.CertFile, config.ServingInfo.KeyFile); err != nil {
				return nil, nil, err
			}
			crtContent := &bytes.Buffer{}
			keyContent := &bytes.Buffer{}
			if err := servingCert.WriteCertConfig(crtContent, keyContent); err != nil {
				return nil, nil, err
			}

			// If we generate our own certificates, then we want to specify empty content to avoid a starting race.  This way,
			// if any change comes in, we will properly restart
			startingFileContent[filepath.Join(certDir, "tls.crt")] = crtContent.Bytes()
			startingFileContent[filepath.Join(certDir, "tls.key")] = keyContent.Bytes()
		}
	}
	return startingFileContent, observedFiles, nil
}

// StartController runs the controller. This is the recommend entrypoint when you don't need
// to customize the builder.
func (c *ControllerCommandConfig) StartController(ctx context.Context) error {
	unstructuredConfig, config, configContent, err := c.Config()
	if err != nil {
		return err
	}

	startingFileContent, observedFiles, err := c.AddDefaultRotationToConfig(config, configContent)
	if err != nil {
		return err
	}

	exitOnChangeReactorCh := make(chan struct{})
	ctx2, cancel := context.WithCancel(ctx)
	go func() {
		select {
		case <-exitOnChangeReactorCh:
			cancel()
		case <-ctx.Done():
			cancel()
		}
	}()

	builder := NewController(c.componentName, c.startFunc).
		WithKubeConfigFile(c.basicFlags.KubeConfigFile, nil).
		WithLeaderElection(config.LeaderElection, "", c.componentName+"-lock").
		WithServer(config.ServingInfo, config.Authentication, config.Authorization).
		WithRestartOnChange(exitOnChangeReactorCh, startingFileContent, observedFiles...)

	return builder.Run(unstructuredConfig, ctx2)
}
