package spice

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net"

	"github.com/jsimonetti/go-spice/red"
)

type tenantHandshake struct {
	proxy *Proxy

	done bool

	tenantAuthMethod red.AuthMethod
	privateKey       *rsa.PrivateKey

	channelID   uint8
	channelType red.ChannelType
	sessionID   uint32

	otp         string // one time password
	destination string // compute computeAddress

	log Logger
}

func newTenantHandshake(p *Proxy, log Logger) (*tenantHandshake, error) {

	handShake := &tenantHandshake{
		proxy: p,
		log:   log,
	}

	rng := rand.Reader
	key, err := rsa.GenerateKey(rng, 1024)
	if err != nil {
		return nil, err
	}
	handShake.privateKey = key

	return handShake, nil
}

func (c *tenantHandshake) Done() bool {
	return c.done
}

func (c *tenantHandshake) clientLinkStage(tenant io.ReadWriter) (net.Conn, error) {
	// Handle first Tenant Link Message
	if err := c.clientLinkMessage(tenant); err != nil {
		return nil, err
	}

	c.otp = c.proxy.sessionTable.OTP(c.sessionID)
	c.destination = c.proxy.sessionTable.Compute(c.sessionID)

	// Handle 2nd Tenant auth method select
	if err := c.clientAuthMethod(tenant); err != nil {
		return nil, err
	}

	// Do compute handshake
	handShake := &computeHandshake{
		proxy:       c.proxy,
		channelType: c.channelType,
		channelID:   c.channelID,
		sessionID:   c.sessionID,
		tenant:      tenant,
		log:         c.log,
	}

	// Lookup destination in proxy.sessionTable
	if c.proxy.sessionTable.Lookup(c.sessionID) {
		var err error
		c.destination, err = c.proxy.sessionTable.Connect(c.sessionID)
		if err != nil {
			return nil, err
		}
	}

	handShake.log = c.log.WithFields("compute", c.destination)

	for !handShake.Done() {
		if err := handShake.clientLinkStage(c.destination); err != nil {
			handShake.log.WithError(err).Error("compute handshake error")
			return nil, err
		}
	}

	c.log = handShake.log

	c.sessionID = handShake.sessionID
	c.proxy.sessionTable.Add(c.sessionID, c.destination, c.otp)
	c.done = true

	return handShake.compute, nil
}

func (c *tenantHandshake) clientAuthMethod(tenant io.ReadWriter) error {
	var err error
	b := make([]byte, 4)

	if _, err = tenant.Read(b); err != nil {
		c.log.WithError(err).Error("error reading client AuthMethod")
		return err
	}

	c.log.Debug("received ClientAuthMethod")

	c.tenantAuthMethod = red.AuthMethod(b[0])

	var auth Authenticator
	var ok bool

	if auth, ok = c.proxy.authenticator[c.tenantAuthMethod]; !ok {
		if err := sendServerTicket(red.ErrorPermissionDenied, tenant); err != nil {
			c.log.WithError(err).Error("send ticket")
		}
		return fmt.Errorf("unavailable auth method %s", c.tenantAuthMethod)
	}

	c.log = c.log.WithFields("authmethod", c.tenantAuthMethod)
	c.log.Debug("starting authentication")

	var authCtx AuthContext
	switch c.tenantAuthMethod {
	case red.AuthMethodSpice:
		authCtx = &authSpice{tenant: tenant, privateKey: c.privateKey, token: c.otp, computeAddress: c.destination}
	case red.AuthMethodSASL:
		return errors.New("SASL is not a supported authmethod")
	default:
		return errors.New("unsupported authmethod")
	}

	result, destination, err := auth.Next(authCtx)
	if err != nil {
		c.log.WithError(err).Error("authentication error")
		return err
	}

	c.otp = authCtx.LoadToken()
	c.destination = destination

	if !result {
		if err := sendServerTicket(red.ErrorPermissionDenied, tenant); err != nil {
			c.log.WithError(err).Error("send ticket")
			return err
		}
		return fmt.Errorf("authentication failed")
	}

	return sendServerTicket(red.ErrorOk, tenant)
}

func (c *tenantHandshake) clientLinkMessage(tenant io.ReadWriter) error {
	var err error
	var b []byte

	if b, err = readLinkPacket(tenant); err != nil {
		c.log.WithError(err).Error("error reading link packet")
		return err
	}

	c.log.Debug("received ClientLinkMessage")

	linkMessage := &red.ClientLinkMessage{}
	if err := linkMessage.UnmarshalBinary(b); err != nil {
		return err
	}

	c.channelType = linkMessage.ChannelType
	c.channelID = linkMessage.ChannelID
	c.sessionID = linkMessage.SessionID

	c.log = c.log.WithFields("channel", c.channelID, "type", c.channelType, "session", c.sessionID)

	return sendServerLinkPacket(tenant, c.privateKey.Public())
}

func redPubKey(key crypto.PublicKey) (ret red.PubKey, err error) {
	cert, err := x509.MarshalPKIXPublicKey(key)
	if err != nil {
		return ret, err
	}

	copy(ret[:], cert[:])
	return ret, nil
}

func sendServerLinkPacket(wr io.Writer, key crypto.PublicKey) error {
	pubkey, err := redPubKey(key)
	if err != nil {
		return err
	}

	var channelCaps, commonCaps red.Capability
	commonCaps.Set(red.CapabilityAuthSpice).Set(red.CapabilityAuthSelection).Set(red.CapabilityMiniHeader)
	channelCaps.Set(red.CapabilityMainSeamlessMigrate).Set(red.CapabilityMainSemiSeamlessMigrate)

	reply := red.ServerLinkMessage{
		Error:               red.ErrorOk,
		PubKey:              pubkey,
		CommonCaps:          1,
		ChannelCaps:         1,
		CommonCapabilities:  []red.Capability{commonCaps},
		ChannelCapabilities: []red.Capability{channelCaps},
	}

	b, err := reply.MarshalBinary()
	if err != nil {
		return err
	}

	header := red.LinkHeader{
		Size: reply.CapsOffset + 8,
	}

	b2, err := header.MarshalBinary()
	if err != nil {
		return err
	}

	data := append(b2, b...)

	_, err = wr.Write(data)
	if err != nil {
		return err
	}

	return nil
}

func readLinkPacket(conn io.Reader) ([]byte, error) {
	headerBytes := make([]byte, 16)

	if _, err := conn.Read(headerBytes); err != nil {
		return nil, err
	}

	header := &red.LinkHeader{}
	if err := header.UnmarshalBinary(headerBytes); err != nil {
		return nil, err
	}

	var messageBytes []byte
	var n int
	var err error
	pending := int(header.Size)

	for pending > 0 {
		bytes := make([]byte, header.Size)
		if n, err = conn.Read(bytes); err != nil {
			return nil, err
		}
		pending = pending - n
		messageBytes = append(messageBytes, bytes[:n]...)
	}

	return messageBytes[:int(header.Size)], nil
}

func sendServerTicket(result red.ErrorCode, writer io.Writer) error {
	msg := red.ServerTicket{
		Result: result,
	}

	b, err := msg.MarshalBinary()
	if err != nil {
		return err
	}

	if _, err := writer.Write(b); err != nil {
		return err
	}

	return nil
}
