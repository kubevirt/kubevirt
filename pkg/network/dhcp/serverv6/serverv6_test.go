/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2020 Red Hat, Inc.
 *
 */

package serverv6

import (
	"net"

	"github.com/insomniacslk/dhcp/dhcpv6"
	"github.com/insomniacslk/dhcp/iana"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("DHCPv6", func() {
	Context("prepareDHCPv6Modifiers", func() {
		It("should contain ianaAdrress and duid", func() {
			clientIP := net.ParseIP("fd10:0:2::2")
			serverInterfaceMac, _ := net.ParseMAC("12:34:56:78:9A:BC")
			modifiers := prepareDHCPv6Modifiers(clientIP, serverInterfaceMac)
			Expect(modifiers).To(HaveLen(2))

			msg := &dhcpv6.Message{
				MessageType: dhcpv6.MessageTypeAdvertise,
			}
			expectedIaAddr := dhcpv6.OptIAAddress{IPv6Addr: clientIP, PreferredLifetime: infiniteLease, ValidLifetime: infiniteLease}
			modifiers[0](msg)
			opt := msg.GetOneOption(dhcpv6.OptionIANA)
			optIana := opt.(*dhcpv6.OptIANA)
			Expect(optIana.Options.Addresses()).To(HaveLen(1))
			Expect(optIana.Options.OneAddress().String()).To(Equal(expectedIaAddr.String()))

			duid := &dhcpv6.DUIDLL{HWType: iana.HWTypeEthernet, LinkLayerAddr: serverInterfaceMac}
			expectedServerId := dhcpv6.OptServerID(duid)
			modifiers[1](msg)
			Expect(msg.GetOneOption(dhcpv6.OptionServerID).String()).To(Equal(expectedServerId.String()))
		})
	})
	Context("buildResponse should build a response with", func() {
		var handler *DHCPv6Handler

		BeforeEach(func() {
			clientIP := net.ParseIP("fd10:0:2::2")
			serverInterfaceMac, _ := net.ParseMAC("12:34:56:78:9A:BC")
			modifiers := prepareDHCPv6Modifiers(clientIP, serverInterfaceMac)

			handler = &DHCPv6Handler{
				clientIP:  clientIP,
				modifiers: modifiers,
			}
		})

		It("advertise type on rapid commit solicit request", func() {
			clientMessage, err := newMessage(dhcpv6.MessageTypeSolicit)
			Expect(err).ToNot(HaveOccurred())
			dhcpv6.WithRapidCommit(clientMessage)

			replyMessage, err := handler.buildResponse(clientMessage)
			Expect(err).ToNot(HaveOccurred())
			Expect(replyMessage.Type()).To(Equal(dhcpv6.MessageTypeReply))
		})
		It("reply type on solicit request", func() {
			clientMessage, err := newMessage(dhcpv6.MessageTypeSolicit)
			Expect(err).ToNot(HaveOccurred())

			replyMessage, err := handler.buildResponse(clientMessage)
			Expect(err).ToNot(HaveOccurred())
			Expect(replyMessage.Type()).To(Equal(dhcpv6.MessageTypeAdvertise))
		})
		It("reply type on any other request", func() {
			clientMessage, err := newMessage(dhcpv6.MessageTypeRequest)
			Expect(err).ToNot(HaveOccurred())

			replyMessage, err := handler.buildResponse(clientMessage)
			Expect(err).ToNot(HaveOccurred())
			Expect(replyMessage.Type()).To(Equal(dhcpv6.MessageTypeReply))
		})
		It("iana option containing the iaid from the request", func() {
			clientMessage, err := newMessage(dhcpv6.MessageTypeSolicit)
			iaId := [4]byte{5, 6, 7, 8}
			clientMessage.UpdateOption(&dhcpv6.OptIANA{IaId: iaId})
			Expect(err).ToNot(HaveOccurred())

			replyMessage, err := handler.buildResponse(clientMessage)
			Expect(err).ToNot(HaveOccurred())
			Expect(replyMessage.Options.OneIANA().IaId).To(Equal([4]byte{5, 6, 7, 8}))
		})
		It("the correct number of options", func() {
			clientMessage, err := newMessage(dhcpv6.MessageTypeSolicit)
			Expect(err).ToNot(HaveOccurred())

			replyMessage, err := handler.buildResponse(clientMessage)
			Expect(err).ToNot(HaveOccurred())
			expectedLength := len(handler.modifiers) + 1
			Expect(replyMessage.Options.Options).To(HaveLen(expectedLength))
		})
		It("handle request without iana option", func() {
			clientMac, _ := net.ParseMAC("34:56:78:9A:BC:DE")
			duid := &dhcpv6.DUIDLL{HWType: iana.HWTypeEthernet, LinkLayerAddr: clientMac}
			clientMessage, err := dhcpv6.NewMessage(dhcpv6.WithClientID(duid))
			Expect(err).ToNot(HaveOccurred())
			clientMessage.MessageType = dhcpv6.MessageTypeInformationRequest

			_, err = handler.buildResponse(clientMessage)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})

func newMessage(messageType dhcpv6.MessageType) (*dhcpv6.Message, error) {
	clientMac, _ := net.ParseMAC("34:56:78:9A:BC:DE")
	duid := &dhcpv6.DUIDLL{HWType: iana.HWTypeEthernet, LinkLayerAddr: clientMac}
	clientMessage, err := dhcpv6.NewMessage(dhcpv6.WithIAID([4]byte{1, 2, 3, 4}), dhcpv6.WithClientID(duid))
	clientMessage.MessageType = messageType
	return clientMessage, err
}
