package spice

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"

	"github.com/jsimonetti/go-spice/red"
)

type computeHandshake struct {
	proxy *Proxy

	done    bool
	compute net.Conn
	tenant  io.Writer

	channelID   uint8
	channelType red.ChannelType
	sessionID   uint32

	computePubKey red.PubKey
	log           Logger
}

func (c *computeHandshake) Done() bool {
	return c.done
}

func (c *computeHandshake) clientLinkStage(destination string) error {
	var err error

	c.compute, err = c.proxy.dial(context.Background(), "tcp", destination)
	if err != nil {
		c.log.WithError(err).Error("dial compute error")
		return err
	}

	// handle send client LinkMessage
	if err := c.clientLinkMessage(c.compute); err != nil {
		return err
	}

	// handle send auth method
	if err := c.clientAuthMethod(c.compute); err != nil {
		return err
	}

	// Handle 3rd Client Ticket
	if err := c.clientTicket(c.compute); err != nil {
		return err
	}

	if c.channelType == red.ChannelMain {
		if err := c.readServerInit(c.compute, c.tenant); err != nil {
			return err
		}
	}

	c.done = true

	return nil
}

func (c *computeHandshake) readServerInit(in io.Reader, out io.Writer) error {
	var b []byte
	var mType uint16
	var err error

	if mType, b, err = readMiniHeaderPacket(in); err != nil {
		c.log.WithError(err).Error("read server Init")
		return err
	}

	if mType != 103 { // Server INIT
		err := errors.New("expected server INIT")
		c.log.WithError(err).Error("read server INIT")
		return err
	}

	c.sessionID = binary.LittleEndian.Uint32(b[6:10])

	if _, err := out.Write(b); err != nil {
		c.log.WithError(err).Error("write server Init")
		return err
	}

	return nil
}

func (c *computeHandshake) clientTicket(rw io.ReadWriter) error {

	password := []byte{} // password for compute side

	// crypto/rand.Reader is a good source of entropy for randomizing the
	// encryption function.
	rng := rand.Reader

	key, err := x509.ParsePKIXPublicKey(c.computePubKey[:])
	if err != nil {
		c.log.WithError(err).Error("Error parsing public key")
		return err
	}
	pubkey := key.(*rsa.PublicKey)

	ciphertext, err := rsa.EncryptOAEP(sha1.New(), rng, pubkey, password, []byte{})
	if err != nil {
		c.log.WithError(err).Error("Error from encryption")
		return err
	}

	var ticket [128]byte
	copy(ticket[:], ciphertext[:])

	myTicket := &red.ClientTicket{
		Ticket: ticket,
	}

	mb, err := myTicket.MarshalBinary()
	if err != nil {
		c.log.WithError(err).Error("Error from marshalling ticket")
		return err
	}
	_, err = rw.Write(mb)
	if err != nil {
		c.log.WithError(err).Error("write ticket to compute error")
		return err
	}

	srvTicket := make([]byte, 4)
	_, err = rw.Read(srvTicket)
	if err != nil {
		c.log.WithError(err).Error("compute ticket read error")
		return err
	}

	if !bytes.Equal(srvTicket[:], []byte{0x00, 0x00, 0x00, 0x00}) {
		err := fmt.Errorf("compute ticket error %#v", srvTicket)
		c.log.WithError(err).Error("compute ticket error")
		return err
	}

	return nil
}

func (c *computeHandshake) clientAuthMethod(wr io.Writer) error {
	myAuthMethod := &red.ClientAuthMethod{
		Method: red.AuthMethodSpice,
	}

	mb, err := myAuthMethod.MarshalBinary()
	if err != nil {
		return err
	}

	if _, err = wr.Write(mb); err != nil {
		c.log.WithError(err).Error("write link message to compute error")
		return err
	}

	return nil
}

func (c *computeHandshake) clientLinkMessage(rw io.ReadWriter) error {
	var channelCaps, commonCaps red.Capability
	commonCaps.Set(red.CapabilityAuthSpice).Set(red.CapabilityAuthSelection).Set(red.CapabilityMiniHeader)
	channelCaps.Set(red.CapabilityMainSeamlessMigrate).Set(red.CapabilityMainSemiSeamlessMigrate)

	myLink := &red.ClientLinkMessage{
		ChannelID:           c.channelID,
		ChannelType:         c.channelType,
		SessionID:           c.sessionID,
		CommonCaps:          1,
		ChannelCaps:         1,
		CapsOffset:          18,
		CommonCapabilities:  []red.Capability{commonCaps},
		ChannelCapabilities: []red.Capability{channelCaps},
	}

	mb, err := myLink.MarshalBinary()
	if err != nil {
		return err
	}
	header := red.LinkHeader{
		Size: myLink.CapsOffset + 8,
	}
	b2, err := header.MarshalBinary()
	if err != nil {
		return err
	}

	data := append(b2, mb...)

	if _, err = rw.Write(data); err != nil {
		c.log.WithError(err).Error("write link message to compute error")
		return err
	}

	var srvLmb []byte
	if srvLmb, err = readLinkPacket(rw); err != nil {
		c.log.WithError(err).Error("compute read serverLinkMessage error")
	}

	srvLm := &red.ServerLinkMessage{}
	if err := srvLm.UnmarshalBinary(srvLmb); err != nil {
		c.log.WithError(err).Error("serverlink unmarshal error")
		return err
	}
	if srvLm.Error != red.ErrorOk {
		err := fmt.Errorf("server connection error %#v", srvLm.Error)
		c.log.WithError(err).Error("server connection error")
		return err
	}

	c.computePubKey = srvLm.PubKey

	return nil
}

func readMiniHeaderPacket(conn io.Reader) (uint16, []byte, error) {
	headerBytes := make([]byte, 6)

	if _, err := conn.Read(headerBytes); err != nil {
		return 0, nil, err
	}

	header := &red.MiniDataHeader{}
	if err := header.UnmarshalBinary(headerBytes); err != nil {
		return 0, nil, err
	}

	var messageBytes []byte
	var n int
	var err error
	pending := int(header.Size)

	for pending > 0 {
		b := make([]byte, header.Size)
		if n, err = conn.Read(b); err != nil {
			return 0, nil, err
		}
		pending = pending - n
		messageBytes = append(messageBytes, b[:n]...)
	}

	totalBytes := append(headerBytes, messageBytes[:int(header.Size)]...)
	return header.MessageType, totalBytes, nil
}
