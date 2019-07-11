package spice

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"io"
	"net"
	"time"

	"github.com/jsimonetti/go-spice/red"
	"github.com/pkg/errors"
)

// Authenticator is the interface used for creating a tenant authentication
// It is used by the proxy to do two things:
//
//   1) authenticate the user
//   2) return the compute node to forward the tenant user to
//
// When creating your own authentication you should probably use one-time tokens
// for the tenant authentication. Using a method based on the below sequence of events:
//
//  a) Tenant authenticates using token '123e4567:secretpw'
//  b) The Authenticator looks up the token '123e4567' in a shared store
//     (kv store or database)
//  c) The value of token 123e4567 is an encrypted compute node computeAddress.
//     Attempt to decrypt the computeAddress using 'secretpw'. If this results
//     in a valid compute node computeAddress, the user is granted access, and
//     de compute destination is set to the decrypted node computeAddress.
//     In the same transaction, a new token+secret should be generated, and the
//     old one destroyed
type Authenticator interface {
	// Next starts the authentication procedure for the tenant connection
	// It should only ever return an error when there is a issue performing the
	// authentication. A non-existant user/token or a bad password/token is not
	// considered an error.
	// Errors result in a connection being dropped instantly, whereas
	// `accessGranted = false` results in the connection being dropped, after
	// an 'access denied' message is returned. `accessGranted = false` is also
	// not logged by the proxy, where an error is.
	Next(AuthContext) (accessGranted bool, computeDestination string, err error)

	// Method is used to retrieve the type of authentication this
	// Authenticator supports
	Method() red.AuthMethod

	// Init is called once during configuration and can be used to do any
	// initialisation this Authenticator might need. If an error is
	// returned, the Authenticator is not used.
	Init() error
}

var _ Authenticator = &noopAuth{}

// noopAuth is a default no-op Authenticator that returns a static compute
// entry and is always successful.
type noopAuth struct{}

// Next implements the Authenticator interface
func (a *noopAuth) Next(ctx AuthContext) (bool, string, error) {
	var c AuthSpiceContext
	var ok bool
	if c, ok = ctx.(AuthSpiceContext); !ok {
		return false, "", errors.New("invalid auth method")
	}

	c.(*authSpice).readTicket()
	return true, "127.0.0.1:5900", nil
}

// Method implements the Authenticator interface
func (a *noopAuth) Method() red.AuthMethod {
	return red.AuthMethodSpice
}

// Init implements the Authenticator interface
func (a *noopAuth) Init() error { return nil }

// AuthContext is used to pass either a spiceAuthContext or a saslAuthContext
// to the Authenticator
type AuthContext interface {
	LoadToken() string
	SaveToken(string)
	LoadAddress() string
	SaveAddress(string)
}

// AuthSASLContext is the interface for SASL authentication.
// This is not yet implemented
type AuthSASLContext interface {
	toBeImplemented()
	AuthContext
}

// AuthSpiceContext is the interface for token based (Spice) authentication.
type AuthSpiceContext interface {
	Token() (string, error)
	AuthContext
}

// authSpice is a special context for the Authenticator
// Is is used to pass information from the proxy to the Authenticator and
// back again.
type authSpice struct {
	tenant          io.Reader
	ticketCrypted   []byte
	ticketUncrypted []byte

	privateKey     *rsa.PrivateKey // needed for Spice auth
	token          string          // previously authenticated ticket
	computeAddress string          // destination compute node
}

// readTicket is a helper function to read the tenant ticket bytes
func (a *authSpice) readTicket() ([]byte, error) {
	if a.ticketCrypted != nil {
		return a.ticketCrypted, nil
	}
	if _, ok := a.tenant.(net.Conn); ok {
		a.tenant.(net.Conn).SetReadDeadline(time.Now().Add(5 * time.Second))
		defer a.tenant.(net.Conn).SetReadDeadline(time.Time{})
	}

	a.ticketCrypted = make([]byte, red.ClientTicketBytes)
	if _, err := a.tenant.Read(a.ticketCrypted); err != nil {
		return nil, errors.Wrap(err, "read deadline reached")
	}
	return a.ticketCrypted, nil
}

// Token will return the unencrypted token string the tenant used
// to authenticate this session after trimming trailing zero's.
func (a *authSpice) Token() (string, error) {
	crypted, err := a.readTicket()
	if err != nil {
		return "", err
	}
	key := a.privateKey
	if key == nil {
		return "", errors.New("no private key")
	}

	rng := rand.Reader
	plaintext, err := rsa.DecryptOAEP(sha1.New(), rng, key, crypted, []byte{})
	if err != nil {
		return "", errors.New("could not decrypt token")
	}

	// trim trailing nul
	a.ticketUncrypted = bytes.Trim(plaintext, "\x00")

	return string(a.ticketUncrypted), nil
}

// LoadToken return the token saved to this session.
// If this connection belongs to a previously established session
// (any channel after the first), this returns the token that was stored
// in the session table when authenticating the first connection.
// This allows for the use of One-Time-Passwords, but still allow multiple
// connections belonging to the same session to be validated.
// The exact method of validation is up to the implementor of an Authenticator.
// See the example on how to use this.
func (a *authSpice) LoadToken() string {
	return a.token
}

// SaveToken stores a token in the context. When the result of the authentication
// is true (access is granted) this token is saved in the session table. Any subsequent
// connections using the same session id, will have this token available in its auth,
// and can be retrieved using LoadToken().
// See the example on how to use this.
func (a *authSpice) SaveToken(token string) {
	a.token = token
}

// LoadAddress returns the compute node computeAddress saved to this session.
// This is the same for LoadToken, only it is used to store the compute node computeAddress
// for this session.
// See the example on how to use this.
func (a *authSpice) LoadAddress() string {
	return a.computeAddress
}

// SaveAddress saves the compute node computeAddress in the context. When the result of the authentication
// is true (access is granted) this computeAddress is saved in the session table.
// See the example on how to use this.
func (a *authSpice) SaveAddress(address string) {
	a.computeAddress = address
}
