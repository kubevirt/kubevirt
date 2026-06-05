package dhcpv6

import (
	"fmt"
)

// TransactionID is a DHCPv6 Transaction ID defined by RFC 3315, Section 6.
type TransactionID [3]byte

// String prints the transaction ID as a hex value.
func (xid TransactionID) String() string {
	return fmt.Sprintf("0x%x", xid[:])
}

// MessageType represents the kind of DHCPv6 message.
type MessageType uint8

// The DHCPv6 message types defined per RFC 3315, Section 5.3.
const (
	// MessageTypeNone is used internally and is not part of the RFC.
	MessageTypeNone               MessageType = 0
	MessageTypeSolicit            MessageType = 1
	MessageTypeAdvertise          MessageType = 2
	MessageTypeRequest            MessageType = 3
	MessageTypeConfirm            MessageType = 4
	MessageTypeRenew              MessageType = 5
	MessageTypeRebind             MessageType = 6
	MessageTypeReply              MessageType = 7
	MessageTypeRelease            MessageType = 8
	MessageTypeDecline            MessageType = 9
	MessageTypeReconfigure        MessageType = 10
	MessageTypeInformationRequest MessageType = 11
	MessageTypeRelayForward       MessageType = 12
	MessageTypeRelayReply         MessageType = 13
	MessageTypeLeaseQuery         MessageType = 14
	MessageTypeLeaseQueryReply    MessageType = 15
	MessageTypeLeaseQueryDone     MessageType = 16
	MessageTypeLeaseQueryData     MessageType = 17
	_                             MessageType = 18
	_                             MessageType = 19
	MessageTypeDHCPv4Query        MessageType = 20
	MessageTypeDHCPv4Response     MessageType = 21
)

// String prints the message type name.
func (m MessageType) String() string {
	if s, ok := messageTypeToStringMap[m]; ok {
		return s
	}
	return fmt.Sprintf("unknown (%d)", m)
}

// messageTypeToStringMap contains the mapping of MessageTypes to
// human-readable strings.
var messageTypeToStringMap = map[MessageType]string{
	MessageTypeSolicit:            "SOLICIT",
	MessageTypeAdvertise:          "ADVERTISE",
	MessageTypeRequest:            "REQUEST",
	MessageTypeConfirm:            "CONFIRM",
	MessageTypeRenew:              "RENEW",
	MessageTypeRebind:             "REBIND",
	MessageTypeReply:              "REPLY",
	MessageTypeRelease:            "RELEASE",
	MessageTypeDecline:            "DECLINE",
	MessageTypeReconfigure:        "RECONFIGURE",
	MessageTypeInformationRequest: "INFORMATION-REQUEST",
	MessageTypeRelayForward:       "RELAY-FORW",
	MessageTypeRelayReply:         "RELAY-REPL",
	MessageTypeLeaseQuery:         "LEASEQUERY",
	MessageTypeLeaseQueryReply:    "LEASEQUERY-REPLY",
	MessageTypeLeaseQueryDone:     "LEASEQUERY-DONE",
	MessageTypeLeaseQueryData:     "LEASEQUERY-DATA",
	MessageTypeDHCPv4Query:        "DHCPv4-QUERY",
	MessageTypeDHCPv4Response:     "DHCPv4-RESPONSE",
}

// OptionCode is a single byte representing the code for a given Option.
type OptionCode uint16

// String returns the option code name.
func (o OptionCode) String() string {
	if s, ok := optionCodeToString[o]; ok {
		return s
	}
	return fmt.Sprintf("unknown (%d)", o)
}

// All DHCPv6 options.
const (
	OptionClientID                                OptionCode = 1
	OptionServerID                                OptionCode = 2
	OptionIANA                                    OptionCode = 3
	OptionIATA                                    OptionCode = 4
	OptionIAAddr                                  OptionCode = 5
	OptionORO                                     OptionCode = 6
	OptionPreference                              OptionCode = 7
	OptionElapsedTime                             OptionCode = 8
	OptionRelayMsg                                OptionCode = 9
	_                                             OptionCode = 10
	OptionAuth                                    OptionCode = 11
	OptionUnicast                                 OptionCode = 12
	OptionStatusCode                              OptionCode = 13
	OptionRapidCommit                             OptionCode = 14
	OptionUserClass                               OptionCode = 15
	OptionVendorClass                             OptionCode = 16
	OptionVendorOpts                              OptionCode = 17
	OptionInterfaceID                             OptionCode = 18
	OptionReconfMessage                           OptionCode = 19
	OptionReconfAccept                            OptionCode = 20
	OptionSIPServersDomainNameList                OptionCode = 21
	OptionSIPServersIPv6AddressList               OptionCode = 22
	OptionDNSRecursiveNameServer                  OptionCode = 23
	OptionDomainSearchList                        OptionCode = 24
	OptionIAPD                                    OptionCode = 25
	OptionIAPrefix                                OptionCode = 26
	OptionNISServers                              OptionCode = 27
	OptionNISPServers                             OptionCode = 28
	OptionNISDomainName                           OptionCode = 29
	OptionNISPDomainName                          OptionCode = 30
	OptionSNTPServerList                          OptionCode = 31
	OptionInformationRefreshTime                  OptionCode = 32
	OptionBCMCSControllerDomainNameList           OptionCode = 33
	OptionBCMCSControllerIPv6AddressList          OptionCode = 34
	_                                             OptionCode = 35
	OptionGeoConfCivic                            OptionCode = 36
	OptionRemoteID                                OptionCode = 37
	OptionRelayAgentSubscriberID                  OptionCode = 38
	OptionFQDN                                    OptionCode = 39
	OptionPANAAuthenticationAgent                 OptionCode = 40
	OptionNewPOSIXTimezone                        OptionCode = 41
	OptionNewTZDBTimezone                         OptionCode = 42
	OptionEchoRequest                             OptionCode = 43
	OptionLQQuery                                 OptionCode = 44
	OptionClientData                              OptionCode = 45
	OptionCLTTime                                 OptionCode = 46
	OptionLQRelayData                             OptionCode = 47
	OptionLQClientLink                            OptionCode = 48
	OptionMIPv6HomeNetworkIDFQDN                  OptionCode = 49
	OptionMIPv6VisitedHomeNetworkInformation      OptionCode = 50
	OptionLoSTServer                              OptionCode = 51
	OptionCAPWAPAccessControllerAddresses         OptionCode = 52
	OptionRelayID                                 OptionCode = 53
	OptionIPv6AddressMOS                          OptionCode = 54
	OptionIPv6FQDNMOS                             OptionCode = 55
	OptionNTPServer                               OptionCode = 56
	OptionV6AccessDomain                          OptionCode = 57
	OptionSIPUACSList                             OptionCode = 58
	OptionBootfileURL                             OptionCode = 59
	OptionBootfileParam                           OptionCode = 60
	OptionClientArchType                          OptionCode = 61
	OptionNII                                     OptionCode = 62
	OptionGeolocation                             OptionCode = 63
	OptionAFTRName                                OptionCode = 64
	OptionERPLocalDomainName                      OptionCode = 65
	OptionRSOO                                    OptionCode = 66
	OptionPDExclude                               OptionCode = 67
	OptionVirtualSubnetSelection                  OptionCode = 68
	OptionMIPv6IdentifiedHomeNetworkInformation   OptionCode = 69
	OptionMIPv6UnrestrictedHomeNetworkInformation OptionCode = 70
	OptionMIPv6HomeNetworkPrefix                  OptionCode = 71
	OptionMIPv6HomeAgentAddress                   OptionCode = 72
	OptionMIPv6HomeAgentFQDN                      OptionCode = 73
	OptionRDNSSSelection                          OptionCode = 74
	OptionKRBPrincipalName                        OptionCode = 75
	OptionKRBRealmName                            OptionCode = 76
	OptionKRBDefaultRealmName                     OptionCode = 77
	OptionKRBKDC                                  OptionCode = 78
	OptionClientLinkLayerAddr                     OptionCode = 79
	OptionLinkAddress                             OptionCode = 80
	OptionRadius                                  OptionCode = 81
	OptionSolMaxRT                                OptionCode = 82
	OptionInfMaxRT                                OptionCode = 83
	OptionAddrSel                                 OptionCode = 84
	OptionAddrSelTable                            OptionCode = 85
	OptionV6PCPServer                             OptionCode = 86
	OptionDHCPv4Msg                               OptionCode = 87
	OptionDHCP4oDHCP6Server                       OptionCode = 88
	OptionS46Rule                                 OptionCode = 89
	OptionS46BR                                   OptionCode = 90
	OptionS46DMR                                  OptionCode = 91
	OptionS46V4V6Bind                             OptionCode = 92
	OptionS46PortParams                           OptionCode = 93
	OptionS46ContMapE                             OptionCode = 94
	OptionS46ContMapT                             OptionCode = 95
	OptionS46ContLW                               OptionCode = 96
	Option4RD                                     OptionCode = 97
	Option4RDMapRule                              OptionCode = 98
	Option4RDNonMapRule                           OptionCode = 99
	OptionLQBaseTime                              OptionCode = 100
	OptionLQStartTime                             OptionCode = 101
	OptionLQEndTime                               OptionCode = 102
	OptionCaptivePortal                           OptionCode = 103
	OptionMPLParameters                           OptionCode = 104
	OptionANIAccessTechType                       OptionCode = 105
	OptionANINetworkName                          OptionCode = 106
	OptionANIAccessPointName                      OptionCode = 107
	OptionANIAccessPointBSSID                     OptionCode = 108
	OptionANIOperatorID                           OptionCode = 109
	OptionANIOperatorRealm                        OptionCode = 110
	OptionS46Priority                             OptionCode = 111
	OptionMUDUrlV6                                OptionCode = 112
	OptionV6Prefix64                              OptionCode = 113
	OptionFailoverBindingStatus                   OptionCode = 114
	OptionFailoverConnectFlags                    OptionCode = 115
	OptionFailoverDNSRemovalInfo                  OptionCode = 116
	OptionFailoverDNSHostName                     OptionCode = 117
	OptionFailoverDNSZoneName                     OptionCode = 118
	OptionFailoverDNSFlags                        OptionCode = 119
	OptionFailoverExpirationTime                  OptionCode = 120
	OptionFailoverMaxUnackedBNDUPD                OptionCode = 121
	OptionFailoverMCLT                            OptionCode = 122
	OptionFailoverPartnerLifetime                 OptionCode = 123
	OptionFailoverPartnerLifetimeSent             OptionCode = 124
	OptionFailoverPartnerDownTime                 OptionCode = 125
	OptionFailoverPartnerRawCLTTime               OptionCode = 126
	OptionFailoverProtocolVersion                 OptionCode = 127
	OptionFailoverKeepaliveTime                   OptionCode = 128
	OptionFailoverReconfigureData                 OptionCode = 129
	OptionFailoverRelationshipName                OptionCode = 130
	OptionFailoverServerFlags                     OptionCode = 131
	OptionFailoverServerState                     OptionCode = 132
	OptionFailoverStartTimeOfState                OptionCode = 133
	OptionFailoverStateExpirationTime             OptionCode = 134
	OptionRelayPort                               OptionCode = 135
	OptionV6SZTPRedirect                          OptionCode = 136
	OptionS46BindIPv6Prefix                       OptionCode = 137
	_                                             OptionCode = 138
	_                                             OptionCode = 139
	_                                             OptionCode = 140
	_                                             OptionCode = 141
	_                                             OptionCode = 142
	OptionIPv6AddressANDSF                        OptionCode = 143
)

// optionCodeToString maps DHCPv6 OptionCodes to human-readable strings.
var optionCodeToString = map[OptionCode]string{
	OptionClientID:                              "Client ID",
	OptionServerID:                              "Server ID",
	OptionIANA:                                  "IANA",
	OptionIATA:                                  "IATA",
	OptionIAAddr:                                "IA IP Address",
	OptionORO:                                   "Requested Options",
	OptionPreference:                            "Preference",
	OptionElapsedTime:                           "Elapsed Time",
	OptionRelayMsg:                              "Relay Message",
	OptionAuth:                                  "Auth",
	OptionUnicast:                               "Unicast",
	OptionStatusCode:                            "Status Code",
	OptionRapidCommit:                           "Rapid Commit",
	OptionUserClass:                             "User Class",
	OptionVendorClass:                           "Vendor Class",
	OptionVendorOpts:                            "Vendor Options",
	OptionInterfaceID:                           "Interface ID",
	OptionReconfMessage:                         "Reconfig Message",
	OptionReconfAccept:                          "Reconfig Accept",
	OptionSIPServersDomainNameList:              "SIP Servers Domain Name List",
	OptionSIPServersIPv6AddressList:             "SIP Servers IPv6 Address List",
	OptionDNSRecursiveNameServer:                "DNS",
	OptionDomainSearchList:                      "Domain Search List",
	OptionIAPD:                                  "IAPD",
	OptionIAPrefix:                              "IA Prefix",
	OptionNISServers:                            "NIS Servers",
	OptionNISPServers:                           "NISP Servers",
	OptionNISDomainName:                         "NIS Domain Name",
	OptionNISPDomainName:                        "NISP Domain Name",
	OptionSNTPServerList:                        "SNTP Server List",
	OptionInformationRefreshTime:                "Information Refresh Time",
	OptionBCMCSControllerDomainNameList:         "BCMCS Controller Domain Name List",
	OptionBCMCSControllerIPv6AddressList:        "BCMCS Controller IPv6 Address List",
	OptionGeoConfCivic:                          "Geoconf",
	OptionRemoteID:                              "Remote ID",
	OptionRelayAgentSubscriberID:                "Relay-Agent Subscriber ID",
	OptionFQDN:                                  "FQDN",
	OptionPANAAuthenticationAgent:               "PANA Authentication Agent",
	OptionNewPOSIXTimezone:                      "New POSIX Timezone",
	OptionNewTZDBTimezone:                       "New TZDB Timezone",
	OptionEchoRequest:                           "Echo Request",
	OptionLQQuery:                               "OPTION_LQ_QUERY",
	OptionClientData:                            "OPTION_CLIENT_DATA",
	OptionCLTTime:                               "OPTION_CLT_TIME",
	OptionLQRelayData:                           "OPTION_LQ_RELAY_DATA",
	OptionLQClientLink:                          "OPTION_LQ_CLIENT_LINK",
	OptionMIPv6HomeNetworkIDFQDN:                "MIPv6 Home Network ID FQDN",
	OptionMIPv6VisitedHomeNetworkInformation:    "MIPv6 Visited Home Network Information",
	OptionLoSTServer:                            "LoST Server",
	OptionCAPWAPAccessControllerAddresses:       "CAPWAP Access Controller Addresses",
	OptionRelayID:                               "Relay ID",
	OptionIPv6AddressMOS:                        "OPTION-IPv6_Address-MoS",
	OptionIPv6FQDNMOS:                           "OPTION-IPv6-FQDN-MoS",
	OptionNTPServer:                             "NTP Server",
	OptionV6AccessDomain:                        "OPTION_V6_ACCESS_DOMAIN",
	OptionSIPUACSList:                           "OPTION_SIP_UA_CS_LIST",
	OptionBootfileURL:                           "Boot File URL",
	OptionBootfileParam:                         "Boot File Parameters",
	OptionClientArchType:                        "Client Architecture",
	OptionNII:                                   "Network Interface ID",
	OptionGeolocation:                           "OPTION_GEOLOCATION",
	OptionAFTRName:                              "OPTION_AFTR_NAME",
	OptionERPLocalDomainName:                    "OPTION_ERP_LOCAL_DOMAIN_NAME",
	OptionRSOO:                                  "OPTION_RSOO",
	OptionPDExclude:                             "OPTION_PD_EXCLUDE",
	OptionVirtualSubnetSelection:                "Virtual Subnet Selection",
	OptionMIPv6IdentifiedHomeNetworkInformation: "MIPv6 Identified Home Network Information",
	OptionMIPv6UnrestrictedHomeNetworkInformation: "MIPv6 Unrestricted Home Network Information",
	OptionMIPv6HomeNetworkPrefix:                  "MIPv6 Home Network Prefix",
	OptionMIPv6HomeAgentAddress:                   "MIPv6 Home Agent Address",
	OptionMIPv6HomeAgentFQDN:                      "MIPv6 Home Agent FQDN",
	OptionRDNSSSelection:                          "RDNSS Selection",
	OptionKRBPrincipalName:                        "Kerberos Principal Name",
	OptionKRBRealmName:                            "Kerberos Realm Name",
	OptionKRBDefaultRealmName:                     "Kerberos Default Realm Name",
	OptionKRBKDC:                                  "Kerberos KDC",
	OptionClientLinkLayerAddr:                     "Client Link-Layer Address",
	OptionLinkAddress:                             "Link Address",
	OptionRadius:                                  "OPTION_RADIUS",
	OptionSolMaxRT:                                "Max Solicit Timeout Value",
	OptionInfMaxRT:                                "Max Information-Request Timeout Value",
	OptionAddrSel:                                 "Address Selection",
	OptionAddrSelTable:                            "Address Selection Policy Table",
	OptionV6PCPServer:                             "Port Control Protocol Server",
	OptionDHCPv4Msg:                               "Encapsulated DHCPv4 Message",
	OptionDHCP4oDHCP6Server:                       "DHCPv4-over-DHCPv6 Server",
	OptionS46Rule:                                 "Softwire46 Rule",
	OptionS46BR:                                   "Softwire46 Border Relay",
	OptionS46DMR:                                  "Softwire46 Default Mapping Rule",
	OptionS46V4V6Bind:                             "Softwire46 IPv4/IPv6 Address Binding",
	OptionS46PortParams:                           "Softwire46 Port Parameters",
	OptionS46ContMapE:                             "Softwire46 MAP-E Container",
	OptionS46ContMapT:                             "Softwire46 MAP-T Container",
	OptionS46ContLW:                               "Softwire46 Lightweight 4over6 Container",
	Option4RD:                                     "4RD",
	Option4RDMapRule:                              "4RD Mapping Rule",
	Option4RDNonMapRule:                           "4RD Non-Mapping Rule",
	OptionLQBaseTime:                              "Leasequery Server Base time",
	OptionLQStartTime:                             "Leasequery Server Query Start Time",
	OptionLQEndTime:                               "Leasequery Server Query End Time",
	OptionCaptivePortal:                           "Captive Portal URI",
	OptionMPLParameters:                           "MPL Parameters",
	OptionANIAccessTechType:                       "Access-Network-Information Access-Technology-Type",
	OptionANINetworkName:                          "Access-Network-Information Network-Name",
	OptionANIAccessPointName:                      "Access-Network-Information Access-Point-Name",
	OptionANIAccessPointBSSID:                     "Access-Network-Information Access-Point-BSSID",
	OptionANIOperatorID:                           "Access-Network-Information Operator-Identifier",
	OptionANIOperatorRealm:                        "Access-Network-Information Operator-Realm",
	OptionS46Priority:                             "Softwire46 Priority",
	OptionMUDUrlV6:                                "Manufacturer Usage Description URL",
	OptionV6Prefix64:                              "OPTION_V6_PREFIX64",
	OptionFailoverBindingStatus:                   "Failover Binding Status",
	OptionFailoverConnectFlags:                    "Failover Connection Flags",
	OptionFailoverDNSRemovalInfo:                  "Failover DNS Removal Info",
	OptionFailoverDNSHostName:                     "Failover DNS Removal Host Name",
	OptionFailoverDNSZoneName:                     "Failover DNS Removal Zone Name",
	OptionFailoverDNSFlags:                        "Failover DNS Removal Flags",
	OptionFailoverExpirationTime:                  "Failover Maximum Expiration Time",
	OptionFailoverMaxUnackedBNDUPD:                "Failover Maximum Unacked BNDUPD Messages",
	OptionFailoverMCLT:                            "Failover Maximum Client Lead Time",
	OptionFailoverPartnerLifetime:                 "Failover Partner Lifetime",
	OptionFailoverPartnerLifetimeSent:             "Failover Received Partner Lifetime",
	OptionFailoverPartnerDownTime:                 "Failover Last Partner Down Time",
	OptionFailoverPartnerRawCLTTime:               "Failover Last Client Time",
	OptionFailoverProtocolVersion:                 "Failover Protocol Version",
	OptionFailoverKeepaliveTime:                   "Failover Keepalive Time",
	OptionFailoverReconfigureData:                 "Failover Reconfigure Data",
	OptionFailoverRelationshipName:                "Failover Relationship Name",
	OptionFailoverServerFlags:                     "Failover Server Flags",
	OptionFailoverServerState:                     "Failover Server State",
	OptionFailoverStartTimeOfState:                "Failover State Start Time",
	OptionFailoverStateExpirationTime:             "Failover State Expiration Time",
	OptionRelayPort:                               "Relay Source Port",
	OptionV6SZTPRedirect:                          "IPv6 Secure Zerotouch Provisioning Redirect",
	OptionS46BindIPv6Prefix:                       "Softwire46 Source Binding Prefix Hint",
	OptionIPv6AddressANDSF:                        "IPv6 Access Network Discovery and Selection Function Address",
}
