package scoped

import (
	"errors"

	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/client/clientset/versioned"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/operatorclient"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/scopedclient"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
)

var (
	errQuerierNotSpecified = errors.New("no service account querier func specified")
)

// NewClientAttenuator returns a new instance of ClientAttenuator.
func NewClientAttenuator(logger *logrus.Logger, config *rest.Config, kubeclient operatorclient.ClientInterface, crclient versioned.Interface) *ClientAttenuator {
	return &ClientAttenuator{
		logger:     logger,
		kubeclient: kubeclient,
		crclient:   crclient,
		factory:    scopedclient.NewFactory(config),
		retriever: &BearerTokenRetriever{
			kubeclient: kubeclient,
			logger:     logger,
		},
	}
}

// ServiceAccountQuerierFunc returns a reference to the service account from
// which scope client(s) can be created.
// This abstraction allows the attenuator to be agnostic of what the source of user
// specified service accounts are. A user can specify service account(s) for an
// operator group, subscription and CSV.
type ServiceAccountQuerierFunc func() (reference *corev1.ObjectReference, err error)

// ClientAttenuator returns appropriately scoped client(s) to be used for an
// operator that is being installed.
type ClientAttenuator struct {
	// default operator client used by the operator.
	kubeclient operatorclient.ClientInterface

	// default CR client used by the operator.
	crclient versioned.Interface

	factory   *scopedclient.Factory
	retriever *BearerTokenRetriever
	logger    *logrus.Logger
}

// AttenuateClient returns appropriately scoped client(s) to the caller.
//
// client(s) that are bound to OLM cluster-admin role are returned if the querier
// returns no error and reference to a service account is nil.
// Otherwise an attempt is made to return attenuated client instance(s).
func (s *ClientAttenuator) AttenuateClient(querier ServiceAccountQuerierFunc) (kubeclient operatorclient.ClientInterface, crclient versioned.Interface, err error) {
	if querier == nil {
		err = errQuerierNotSpecified
		return
	}

	reference, err := querier()
	if err != nil {
		return
	}

	if reference == nil {
		// No service account/token has been provided. Return the default client(s).
		kubeclient = s.kubeclient
		crclient = s.crclient
		return
	}

	token, err := s.retriever.Retrieve(reference)
	if err != nil {
		return
	}

	// Create client(s) bound to the user defined service account.
	crclient, err = s.factory.NewKubernetesClient(token)
	if err != nil {
		return
	}

	kubeclient, err = s.factory.NewOperatorClient(token)
	if err != nil {
		return
	}

	return
}

// AttenuateOperatorClient returns a scoped operator client instance based on the
// service account returned by the querier specified.
func (s *ClientAttenuator) AttenuateOperatorClient(querier ServiceAccountQuerierFunc) (kubeclient operatorclient.ClientInterface, err error) {
	if querier == nil {
		err = errQuerierNotSpecified
		return
	}

	reference, err := querier()
	if err != nil {
		return
	}

	if reference == nil {
		// No service account/token has been provided. Return the default client(s).
		kubeclient = s.kubeclient
		return
	}

	token, err := s.retriever.Retrieve(reference)
	if err != nil {
		return
	}

	// Create client(s) bound to the user defined service account.
	kubeclient, err = s.factory.NewOperatorClient(token)
	if err != nil {
		return
	}

	return
}
