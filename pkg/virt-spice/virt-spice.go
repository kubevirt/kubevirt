package virt_spice

import (
	"flag"
	"fmt"
	"strconv"
	"time"

	"github.com/jsimonetti/go-spice"
	"github.com/jsimonetti/go-spice/red"
	"github.com/sirupsen/logrus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
)

type virtSpiceApp struct {
	authSpice *AuthSpice
	proxy     *spice.Proxy
}

// AuthSpice is an example implementation of a spice Authenticator
type AuthSpice struct {
	log         *logrus.Entry
	virtCli     kubecli.KubevirtClient
	expiredTime time.Duration
}

func NewVirtSpice() *virtSpiceApp {
	expiration := flag.String("t", "1m", "expiration time of the token (example: 5m)")
	logLevel := flag.String("v", "2", "log level")
	flag.Parse()

	// create a new logger to be used for the proxy and the authenticator
	log := logrus.New()
	level, err := strconv.Atoi(*logLevel)
	if err != nil {
		panic(err)
	}

	log.SetLevel(logrus.Level(uint(level)))

	d, err := time.ParseDuration(*expiration)
	if err != nil {
		panic(err)
	}

	authSpice := &AuthSpice{
		log:         log.WithField("component", "authenticator"),
		expiredTime: d,
	}

	// create the proxy using the logger and authenticator
	logger := spice.Adapt(log.WithField("component", "proxy"))
	proxy, err := spice.New(spice.WithLogger(logger),
		spice.WithAuthenticator(authSpice))
	if err != nil {
		log.Fatalf("error: %s", err)
	}

	return &virtSpiceApp{proxy: proxy, authSpice: authSpice}

}

func (v *virtSpiceApp) Execute() {
	// start listening for tenant connections
	panic(v.proxy.ListenAndServe("tcp", "0.0.0.0:5900"))
}

// Init initialises this authenticator
func (a *AuthSpice) Init() error {
	// fill in some compute nodes
	var err error
	a.virtCli, err = kubecli.GetKubevirtClient()
	if err != nil {
		panic(err)
	}

	return nil
}

// Next will check the supplied token and return authorisation information
func (a *AuthSpice) Next(c spice.AuthContext) (bool, string, error) {
	// convert the AuthContext into an AuthSpiceContext, since we do that
	var ctx spice.AuthSpiceContext
	var ok bool
	if ctx, ok = c.(spice.AuthSpiceContext); !ok {
		return false, "", fmt.Errorf("invalid auth method")
	}

	// retrieve the token sent by the tenant
	token, err := ctx.Token()
	if err != nil {
		return false, "", err
	}

	// find the compute node for this token
	if destination, ok := a.resolveComputeAddress(token); ok {
		a.log.Debugf("Ticket validated, compute node at %s", destination)
		return true, destination, nil
	}

	a.log.Warn("authentication failed")
	return false, "", nil
}

// Method returns the Spice auth method
func (a *AuthSpice) Method() red.AuthMethod {
	return red.AuthMethodSpice
}

// resolveComputeAddress is a custom function that checks the token and returns
// a compute node address
func (a *AuthSpice) resolveComputeAddress(token string) (string, bool) {
	labelSelector := fmt.Sprintf("%s=%s", v1.SpiceTokenLabel, token)
	vmiList, err := a.virtCli.VirtualMachineInstance("").List(&metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		a.log.Errorf("failed to search for vm with the configured token: %s", token)
		return "", false
	}

	if len(vmiList.Items) == 0 {
		a.log.Warningf("non vm found with the requested token %s", token)
		return "", false
	}

	if len(vmiList.Items) != 1 {
		a.log.Errorf("more then one VM return")
		return "", false
	}

	vmi := vmiList.Items[0]

	if vmi.Status.SpiceConnection == nil {
		return "", false
	}

	if vmi.Status.SpiceConnection.SpiceHandler == nil {
		return "", false
	}

	if vmi.Status.SpiceConnection.SpiceToken == nil {
		return "", false
	}

	if time.Now().Sub(vmi.Status.SpiceConnection.SpiceToken.ExpirationTime.Time) > a.expiredTime {
		return "", false
	}

	virtHandlerConnection := fmt.Sprintf("%s:%d", vmi.Status.SpiceConnection.SpiceHandler.Host, vmi.Status.SpiceConnection.SpiceHandler.Port)

	return virtHandlerConnection, true
}
