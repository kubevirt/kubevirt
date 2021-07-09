package dhcpv6

import (
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/insomniacslk/dhcp/iana"
	"github.com/insomniacslk/dhcp/rfc1035label"
	"github.com/u-root/u-root/pkg/rand"
	"github.com/u-root/u-root/pkg/uio"
)

const MessageHeaderSize = 4

// MessageOptions are the options that may appear in a normal DHCPv6 message.
//
// RFC 3315 Appendix B lists the valid options that can be used.
type MessageOptions struct {
	Options
}

// ArchTypes returns the architecture type option.
func (mo MessageOptions) ArchTypes() iana.Archs {
	opt := mo.GetOne(OptionClientArchType)
	if opt == nil {
		return nil
	}
	return opt.(*optClientArchType).Archs
}

// ClientID returns the client identifier option.
func (mo MessageOptions) ClientID() *Duid {
	opt := mo.GetOne(OptionClientID)
	if opt == nil {
		return nil
	}
	return &opt.(*optClientID).Duid
}

// ServerID returns the server identifier option.
func (mo MessageOptions) ServerID() *Duid {
	opt := mo.GetOne(OptionServerID)
	if opt == nil {
		return nil
	}
	return &opt.(*optServerID).Duid
}

// IANA returns all Identity Association for Non-temporary Address options.
func (mo MessageOptions) IANA() []*OptIANA {
	opts := mo.Get(OptionIANA)
	var ianas []*OptIANA
	for _, o := range opts {
		ianas = append(ianas, o.(*OptIANA))
	}
	return ianas
}

// OneIANA returns the first IANA option.
func (mo MessageOptions) OneIANA() *OptIANA {
	ianas := mo.IANA()
	if len(ianas) == 0 {
		return nil
	}
	return ianas[0]
}

// IATA returns all Identity Association for Temporary Address options.
func (mo MessageOptions) IATA() []*OptIATA {
	opts := mo.Get(OptionIANA)
	var iatas []*OptIATA
	for _, o := range opts {
		iatas = append(iatas, o.(*OptIATA))
	}
	return iatas
}

// OneIATA returns the first IATA option.
func (mo MessageOptions) OneIATA() *OptIATA {
	iatas := mo.IATA()
	if len(iatas) == 0 {
		return nil
	}
	return iatas[0]
}

// IAPD returns all Identity Association for Prefix Delegation options.
func (mo MessageOptions) IAPD() []*OptIAPD {
	opts := mo.Get(OptionIAPD)
	var ianas []*OptIAPD
	for _, o := range opts {
		ianas = append(ianas, o.(*OptIAPD))
	}
	return ianas
}

// OneIAPD returns the first IAPD option.
func (mo MessageOptions) OneIAPD() *OptIAPD {
	iapds := mo.IAPD()
	if len(iapds) == 0 {
		return nil
	}
	return iapds[0]
}

// Status returns the status code associated with this option.
func (mo MessageOptions) Status() *OptStatusCode {
	opt := mo.Options.GetOne(OptionStatusCode)
	if opt == nil {
		return nil
	}
	sc, ok := opt.(*OptStatusCode)
	if !ok {
		return nil
	}
	return sc
}

// RequestedOptions returns the Options Requested Option.
func (mo MessageOptions) RequestedOptions() OptionCodes {
	// Technically, RFC 8415 states that ORO may only appear once in the
	// area of a DHCP message. However, some proprietary clients have been
	// observed sending more than one OptionORO.
	//
	// So we merge them.
	opt := mo.Options.Get(OptionORO)
	if len(opt) == 0 {
		return nil
	}
	var oc OptionCodes
	for _, o := range opt {
		if oro, ok := o.(*optRequestedOption); ok {
			oc = append(oc, oro.OptionCodes...)
		}
	}
	return oc
}

// DNS returns the DNS Recursive Name Server option as defined by RFC 3646.
func (mo MessageOptions) DNS() []net.IP {
	opt := mo.Options.GetOne(OptionDNSRecursiveNameServer)
	if opt == nil {
		return nil
	}
	if dns, ok := opt.(*optDNS); ok {
		return dns.NameServers
	}
	return nil
}

// DomainSearchList returns the Domain List option as defined by RFC 3646.
func (mo MessageOptions) DomainSearchList() *rfc1035label.Labels {
	opt := mo.Options.GetOne(OptionDomainSearchList)
	if opt == nil {
		return nil
	}
	if dsl, ok := opt.(*optDomainSearchList); ok {
		return dsl.DomainSearchList
	}
	return nil
}

// BootFileURL returns the Boot File URL option as defined by RFC 5970.
func (mo MessageOptions) BootFileURL() string {
	opt := mo.Options.GetOne(OptionBootfileURL)
	if opt == nil {
		return ""
	}
	if u, ok := opt.(optBootFileURL); ok {
		return string(u)
	}
	return ""
}

// BootFileParam returns the Boot File Param option as defined by RFC 5970.
func (mo MessageOptions) BootFileParam() []string {
	opt := mo.Options.GetOne(OptionBootfileParam)
	if opt == nil {
		return nil
	}
	if u, ok := opt.(optBootFileParam); ok {
		return []string(u)
	}
	return nil
}

// UserClasses returns a list of user classes.
func (mo MessageOptions) UserClasses() [][]byte {
	opt := mo.Options.GetOne(OptionUserClass)
	if opt == nil {
		return nil
	}
	if t, ok := opt.(*OptUserClass); ok {
		return t.UserClasses
	}
	return nil
}

// VendorOpts returns the all vendor-specific options.
//
// RFC 8415 Section 21.17:
//
//   Multiple instances of the Vendor-specific Information option may appear in
//   a DHCP message.
func (mo MessageOptions) VendorOpts() []*OptVendorOpts {
	opt := mo.Options.Get(OptionVendorOpts)
	if opt == nil {
		return nil
	}
	var vo []*OptVendorOpts
	for _, o := range opt {
		if t, ok := o.(*OptVendorOpts); ok {
			vo = append(vo, t)
		}
	}
	return vo
}

// VendorOpt returns the vendor options matching the given enterprise number.
//
// RFC 8415 Section 21.17:
//
//   Servers and clients MUST NOT send more than one instance of the
//   Vendor-specific Information option with the same Enterprise Number.
func (mo MessageOptions) VendorOpt(enterpriseNumber uint32) Options {
	vo := mo.VendorOpts()
	for _, v := range vo {
		if v.EnterpriseNumber == enterpriseNumber {
			return v.VendorOpts
		}
	}
	return nil
}

// ElapsedTime returns the Elapsed Time option as defined by RFC 3315 Section 22.9.
//
// ElapsedTime returns a duration of 0 if the option is not present.
func (mo MessageOptions) ElapsedTime() time.Duration {
	opt := mo.Options.GetOne(OptionElapsedTime)
	if opt == nil {
		return 0
	}
	if t, ok := opt.(*optElapsedTime); ok {
		return t.ElapsedTime
	}
	return 0
}

// InformationRefreshTime returns the Information Refresh Time option
// as defined by RFC 815 Section 21.23.
//
// InformationRefreshTime returns the provided default if no option is present.
func (mo MessageOptions) InformationRefreshTime(def time.Duration) time.Duration {
	opt := mo.Options.GetOne(OptionInformationRefreshTime)
	if opt == nil {
		return def
	}
	if t, ok := opt.(*optInformationRefreshTime); ok {
		return t.InformationRefreshtime
	}
	return def
}

// FQDN returns the FQDN option as defined by RFC 4704.
func (mo MessageOptions) FQDN() *OptFQDN {
	opt := mo.Options.GetOne(OptionFQDN)
	if opt == nil {
		return nil
	}
	if fqdn, ok := opt.(*OptFQDN); ok {
		return fqdn
	}
	return nil
}

// DHCP4oDHCP6Server returns the DHCP 4o6 Server Address option as
// defined by RFC 7341.
func (mo MessageOptions) DHCP4oDHCP6Server() *OptDHCP4oDHCP6Server {
	opt := mo.Options.GetOne(OptionDHCP4oDHCP6Server)
	if opt == nil {
		return nil
	}
	if server, ok := opt.(*OptDHCP4oDHCP6Server); ok {
		return server
	}
	return nil
}

// Message represents a DHCPv6 Message as defined by RFC 3315 Section 6.
type Message struct {
	MessageType   MessageType
	TransactionID TransactionID
	Options       MessageOptions
}

var randomRead = rand.Read

// GenerateTransactionID generates a random 3-byte transaction ID.
func GenerateTransactionID() (TransactionID, error) {
	var tid TransactionID
	n, err := randomRead(tid[:])
	if err != nil {
		return tid, err
	}
	if n != len(tid) {
		return tid, fmt.Errorf("invalid random sequence: shorter than 3 bytes")
	}
	return tid, nil
}

// GetTime returns a time integer suitable for DUID-LLT, i.e. the current time counted
// in seconds since January 1st, 2000, midnight UTC, modulo 2^32
func GetTime() uint32 {
	now := time.Since(time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC))
	return uint32((now.Nanoseconds() / 1000000000) % 0xffffffff)
}

// NewSolicit creates a new SOLICIT message, using the given hardware address to
// derive the IAID in the IA_NA option.
func NewSolicit(hwaddr net.HardwareAddr, modifiers ...Modifier) (*Message, error) {
	duid := Duid{
		Type:          DUID_LLT,
		HwType:        iana.HWTypeEthernet,
		Time:          GetTime(),
		LinkLayerAddr: hwaddr,
	}
	m, err := NewMessage()
	if err != nil {
		return nil, err
	}
	m.MessageType = MessageTypeSolicit
	m.AddOption(OptClientID(duid))
	m.AddOption(OptRequestedOption(
		OptionDNSRecursiveNameServer,
		OptionDomainSearchList,
	))
	m.AddOption(OptElapsedTime(0))
	if len(hwaddr) < 4 {
		return nil, errors.New("short hardware addrss: less than 4 bytes")
	}
	l := len(hwaddr)
	var iaid [4]byte
	copy(iaid[:], hwaddr[l-4:l])
	modifiers = append([]Modifier{WithIAID(iaid)}, modifiers...)
	// Apply modifiers
	for _, mod := range modifiers {
		mod(m)
	}
	return m, nil
}

// NewAdvertiseFromSolicit creates a new ADVERTISE packet based on an SOLICIT packet.
func NewAdvertiseFromSolicit(sol *Message, modifiers ...Modifier) (*Message, error) {
	if sol == nil {
		return nil, errors.New("SOLICIT cannot be nil")
	}
	if sol.Type() != MessageTypeSolicit {
		return nil, errors.New("The passed SOLICIT must have SOLICIT type set")
	}
	// build ADVERTISE from SOLICIT
	adv := &Message{
		MessageType:   MessageTypeAdvertise,
		TransactionID: sol.TransactionID,
	}
	// add Client ID
	cid := sol.GetOneOption(OptionClientID)
	if cid == nil {
		return nil, errors.New("Client ID cannot be nil in SOLICIT when building ADVERTISE")
	}
	adv.AddOption(cid)

	// apply modifiers
	for _, mod := range modifiers {
		mod(adv)
	}
	return adv, nil
}

// NewRequestFromAdvertise creates a new REQUEST packet based on an ADVERTISE
// packet options.
func NewRequestFromAdvertise(adv *Message, modifiers ...Modifier) (*Message, error) {
	if adv == nil {
		return nil, errors.New("ADVERTISE cannot be nil")
	}
	if adv.MessageType != MessageTypeAdvertise {
		return nil, fmt.Errorf("The passed ADVERTISE must have ADVERTISE type set")
	}
	// build REQUEST from ADVERTISE
	req, err := NewMessage()
	if err != nil {
		return nil, err
	}
	req.MessageType = MessageTypeRequest
	// add Client ID
	cid := adv.GetOneOption(OptionClientID)
	if cid == nil {
		return nil, fmt.Errorf("Client ID cannot be nil in ADVERTISE when building REQUEST")
	}
	req.AddOption(cid)
	// add Server ID
	sid := adv.GetOneOption(OptionServerID)
	if sid == nil {
		return nil, fmt.Errorf("Server ID cannot be nil in ADVERTISE when building REQUEST")
	}
	req.AddOption(sid)
	// add Elapsed Time
	req.AddOption(OptElapsedTime(0))
	// add IA_NA
	iana := adv.Options.OneIANA()
	if iana == nil {
		return nil, fmt.Errorf("IA_NA cannot be nil in ADVERTISE when building REQUEST")
	}
	req.AddOption(iana)
	// add IA_PD
	if iaPd := adv.GetOneOption(OptionIAPD); iaPd != nil {
		req.AddOption(iaPd)
	}
	req.AddOption(OptRequestedOption(
		OptionDNSRecursiveNameServer,
		OptionDomainSearchList,
	))
	// add OPTION_VENDOR_CLASS, only if present in the original request
	// TODO implement OptionVendorClass
	vClass := adv.GetOneOption(OptionVendorClass)
	if vClass != nil {
		req.AddOption(vClass)
	}

	// apply modifiers
	for _, mod := range modifiers {
		mod(req)
	}
	return req, nil
}

// NewReplyFromMessage creates a new REPLY packet based on a
// Message. The function is to be used when generating a reply to a SOLICIT with
// rapid-commit, REQUEST, CONFIRM, RENEW, REBIND, RELEASE and INFORMATION-REQUEST
// packets.
func NewReplyFromMessage(msg *Message, modifiers ...Modifier) (*Message, error) {
	if msg == nil {
		return nil, errors.New("message cannot be nil")
	}
	switch msg.Type() {
	case MessageTypeSolicit:
		if msg.GetOneOption(OptionRapidCommit) == nil {
			return nil, errors.New("cannot create REPLY from a SOLICIT without rapid-commit option")
		}
		modifiers = append([]Modifier{WithRapidCommit}, modifiers...)
	case MessageTypeRequest, MessageTypeConfirm, MessageTypeRenew,
		MessageTypeRebind, MessageTypeRelease, MessageTypeInformationRequest:
	default:
		return nil, errors.New("cannot create REPLY from the passed message type set")
	}

	// build REPLY from MESSAGE
	rep := &Message{
		MessageType:   MessageTypeReply,
		TransactionID: msg.TransactionID,
	}
	// add Client ID
	cid := msg.GetOneOption(OptionClientID)
	if cid == nil {
		return nil, errors.New("Client ID cannot be nil when building REPLY")
	}
	rep.AddOption(cid)

	// apply modifiers
	for _, mod := range modifiers {
		mod(rep)
	}
	return rep, nil
}

// Type returns this message's message type.
func (m Message) Type() MessageType {
	return m.MessageType
}

// GetInnerMessage returns the message itself.
func (m *Message) GetInnerMessage() (*Message, error) {
	return m, nil
}

// AddOption adds an option to this message.
func (m *Message) AddOption(option Option) {
	m.Options.Add(option)
}

// UpdateOption updates the existing options with the passed option, adding it
// at the end if not present already
func (m *Message) UpdateOption(option Option) {
	m.Options.Update(option)
}

// IsNetboot returns true if the machine is trying to netboot. It checks if
// "boot file" is one of the requested options, which is useful for
// SOLICIT/REQUEST packet types, it also checks if the "boot file" option is
// included in the packet, which is useful for ADVERTISE/REPLY packet.
func (m *Message) IsNetboot() bool {
	if m.IsOptionRequested(OptionBootfileURL) {
		return true
	}
	if optbf := m.GetOneOption(OptionBootfileURL); optbf != nil {
		return true
	}
	return false
}

// IsOptionRequested takes an OptionCode and returns true if that option is
// within the requested options of the DHCPv6 message.
func (m *Message) IsOptionRequested(requested OptionCode) bool {
	return m.Options.RequestedOptions().Contains(requested)
}

// String returns a short human-readable string for this message.
func (m *Message) String() string {
	return fmt.Sprintf("Message(messageType=%s transactionID=%s, %d options)",
		m.MessageType, m.TransactionID, len(m.Options.Options))
}

// Summary prints all options associated with this message.
func (m *Message) Summary() string {
	ret := fmt.Sprintf(
		"Message\n"+
			"  messageType=%s\n"+
			"  transactionid=%s\n",
		m.MessageType,
		m.TransactionID,
	)
	ret += "  options=["
	if len(m.Options.Options) > 0 {
		ret += "\n"
	}
	for _, opt := range m.Options.Options {
		ret += fmt.Sprintf("    %v\n", opt.String())
	}
	ret += "  ]\n"
	return ret
}

// ToBytes returns the serialized version of this message as defined by RFC
// 3315, Section 5.
func (m *Message) ToBytes() []byte {
	buf := uio.NewBigEndianBuffer(nil)
	buf.Write8(uint8(m.MessageType))
	buf.WriteBytes(m.TransactionID[:])
	buf.WriteBytes(m.Options.ToBytes())
	return buf.Data()
}

// GetOption returns the options associated with the code.
func (m *Message) GetOption(code OptionCode) []Option {
	return m.Options.Get(code)
}

// GetOneOption returns the first associated option with the code from this
// message.
func (m *Message) GetOneOption(code OptionCode) Option {
	return m.Options.GetOne(code)
}

// IsRelay returns whether this is a relay message or not.
func (m *Message) IsRelay() bool {
	return false
}
