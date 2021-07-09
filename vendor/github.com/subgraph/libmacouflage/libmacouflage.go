package libmacouflage

import "C"
import (
	"syscall"
	"fmt"
	"net"
	"unsafe"
	rand "crypto/rand"
	"os/user"
	"encoding/json"
	"strings"
	mathrand "math/rand"
	"time"
	"regexp"
)

const SIOCSIFHWADDR = 0x8924
const SIOCETHTOOL = 0x8946
const ETHTOOL_GPERMADDR = 0x00000020
const IFHWADDRLEN = 6
var OuiDb []Oui

const (
	invalidInterfaceRegexp = "^(lo|br|veth|tun|tap|oz|voz).*$"
)

type Mode struct {
	name string
	help string
	flagShort string
	flagLong string
}

// TODO: Ad-hoc structs that work, fix
type NetInfo struct {
	name [16]byte
	family uint16
	data [6]byte
}

type ifreq struct {
	name [16]byte
        epa *EthtoolPermAddr
}

type EthtoolPermAddr struct {
	cmd uint32
	size uint32
	data [6]byte
}

type Oui struct {
	VendorPrefix string	`json:"vendor_prefix"`
	Popular bool		`json:"is_popular"`
	Vendor string		`json:"vendor_name"`
	Devices []Device	`json:"devices"`
}

type Device struct {
	DeviceType string	`json:"device_type"`
	DeviceName string	`json:"device_name"`
}

type NoVendorError struct {
	msg string
}

type InvalidInterfaceTypeError struct {
	msg string
}

func init() {
	Modes := make(map[string]Mode)
	Modes["SPECIFIC"] = Mode{"Specific",
		"Set the MAC XX:XX:XX:XX:XX:XX",
		"m",
		"mac"}
	Modes["RANDOM"] = Mode{"Random",
		"Set fully random MAC",
		"r",
		"random"}
	Modes["SAMEVENDOR"] = Mode{"Same Vendor",
		"Don't change the vendor bytes",
		"e",
		"ending"}
	Modes["ANOTHER"] = Mode{"Another",
		"Set random vendor MAC of the same kind",
		"a",
		"another"}
	Modes["ANY"] = Mode{"Any",
		"Set random vendor MAC of any kind",
        "A",
		"any"}

	OuiData, err := Asset("data/ouis.json")
	if err != nil {
		fmt.Println(err)
		return
	}
	err = json.Unmarshal(OuiData, &OuiDb)
	if err != nil {
		fmt.Println(err)
		return
	}
}

func GetCurrentMac(name string) (mac net.HardwareAddr, err error) {
	if IsInterfaceTypeInvalid(name) {
		msg := fmt.Sprintf("Invalid interface type: %s", name)
		err = InvalidInterfaceTypeError{msg}
		return
	}
	iface, err := net.InterfaceByName(name)
	if err != nil {
		return
	}
	mac = iface.HardwareAddr
	return
}

func GetAllCurrentMacs() (macs map[string]string, err error) {
	ifaces, err := GetInterfaces()
	macs = make(map[string]string)
	for _, iface := range ifaces {
		macs[iface.Name] = iface.HardwareAddr.String()
	}
	return
}

func GetInterfaces() (ifaces []net.Interface, err error) {
	allIfaces, err := net.Interfaces()
	for _, iface := range allIfaces {
		// Skip invalid interfaces
		if IsInterfaceTypeInvalid(iface.Name) {
			continue
		}
		ifaces = append(ifaces, iface)
	}
	return
}

func GetPermanentMac(name string) (mac net.HardwareAddr, err error) {
	if IsInterfaceTypeInvalid(name) {
		msg := fmt.Sprintf("Invalid interface type: %s", name)
		err = InvalidInterfaceTypeError{msg}
		return
	}
	sockfd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, 0)
	defer syscall.Close(sockfd)
	var ifr ifreq
	copy(ifr.name[:], []byte(name))
	var epa EthtoolPermAddr
	epa.cmd = ETHTOOL_GPERMADDR
	epa.size = IFHWADDRLEN
	ifr.epa = &epa
	_, _, errno  := syscall.Syscall(syscall.SYS_IOCTL, uintptr(sockfd), SIOCETHTOOL, uintptr(unsafe.Pointer(&ifr)))
	if errno != 0 {
		err = syscall.Errno(errno)
		return
	}
	mac = net.HardwareAddr(C.GoBytes(unsafe.Pointer(&ifr.epa.data), 6))
	return
}

func GetAllPermanentMacs() (macs map[string]string, err error) {
	ifaces, err := GetInterfaces()
	for _, iface := range ifaces {
		name, err := GetPermanentMac(iface.Name)
		if err != nil {
			fmt.Println(err)
		}
		macs[name.String()] = iface.HardwareAddr.String()
	}
	return
}

func SetMac(name string, mac string) (err error) {
	if IsInterfaceTypeInvalid(name) {
		msg := fmt.Sprintf("Invalid interface type: %s", name)
		err = InvalidInterfaceTypeError{msg}
		return
	}
	result, err := RunningAsRoot()
	if err != nil {
		return
	}
	if !result {
		err = fmt.Errorf("Not running as root, insufficient privileges to set MAC on %s",
		name)
		return
	}
	result, err = IsIfUp(name)
	if err != nil {
		return
	}
	if result {
		err = fmt.Errorf("%s interface is still up, cannot set MAC", name)
		return
	}
	sockfd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, 0)
	defer syscall.Close(sockfd)
	iface, err := net.ParseMAC(mac)
	if err != nil {
		return
	}
	var netinfo NetInfo
	copy(netinfo.name[:], []byte(name))
	netinfo.family = syscall.AF_UNIX
	copy(netinfo.data[:], []byte(iface))
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(sockfd), SIOCSIFHWADDR, uintptr(unsafe.Pointer(&netinfo)))
	if errno != 0 {
		err = syscall.Errno(errno)
		return
	}
	return
}

func SpoofMacRandom(name string, bia bool) (changed bool, err error) {
	bytes := []byte{0, 0, 0, 0, 0, 0}
	mac, err := RandomizeMac(bytes, 0, bia)
	if err != nil {
		return
	}
	err = SetMac(name, mac.String())
	if err != nil {
		return
	}
	changed, err = MacChanged(name)
	if err != nil {
		return
	}
	return
}


func SpoofMacSameVendor(name string, bia bool) (changed bool, err error) {
	oldMac, err := GetCurrentMac(name)
	if err != nil {
		return
	}
	mac, err := RandomizeMac(oldMac, 3, bia)
	if err != nil {
		return
	}
	err = SetMac(name, mac.String())
	if err != nil {
		return
	}
	changed, err = MacChanged(name)
	if err != nil {
		return
	}
	return
}

func SpoofMacSameDeviceType(name string) (changed bool, err error) {
	oldMac, err := GetCurrentMac(name)
	if err != nil {
		return
	}
	deviceType, err := FindDeviceTypeByMac(oldMac.String())
	if err != nil {
		return
	}
	vendors, err := FindAllVendorsByDeviceType(deviceType)
	if err != nil {
		return
	}
	mathrand.Seed(time.Now().UTC().UnixNano())
	vendor := vendors[mathrand.Intn(len(vendors))]
	macBytes, err := net.ParseMAC(vendor.VendorPrefix + ":00:00:00")
	if err != nil {
		return
	}
	newMac, err := RandomizeMac(macBytes, 3, true)
	if err != nil {
		return
	}
	err = SetMac(name, newMac.String())
	if err != nil {
		return
	}
	changed, err = MacChanged(name)
	if err != nil {
		return
	}
	return
}

func SpoofMacAnyDeviceType(name string) (changed bool, err error) {
	vendor := OuiDb[RandomInt(len(OuiDb))]
	macBytes, err := net.ParseMAC(vendor.VendorPrefix + ":00:00:00")
	if err != nil {
		return
	}
	newMac, err := RandomizeMac(macBytes, 3, true)
	if err != nil {
		return
	}
	err = SetMac(name, newMac.String())
	if err != nil {
		return
	}
	changed, err = MacChanged(name)
	if err != nil {
		return
	}
	return
}

func SpoofMacPopular(name string) (changed bool, err error) {
	popular, err := FindAllPopularOuis()
	if err != nil {
		return
	}
	vendor := popular[RandomInt(len(popular))]
	macBytes, err := net.ParseMAC(vendor.VendorPrefix + ":00:00:00")
	newMac, err := RandomizeMac(macBytes, 3, true)
	if err != nil {
		return
	}
	err = SetMac(name, newMac.String())
	if err != nil {
		return
	}
	changed, err = MacChanged(name)
	if err != nil {
		return
	}
	return
}

func CompareMacs(first net.HardwareAddr, second net.HardwareAddr) (same bool) {
	same = first.String() == second.String()
	return
}

func MacChanged(iface string) (changed bool, err error) {
	current, err := GetCurrentMac(iface)
	if err != nil {
		return
	}
	permanent, err := GetPermanentMac(iface)
	if err != nil {
		return
	}
	if !CompareMacs(current, permanent) {
		changed = true
	} 
	return
}

func IsIfUp(name string) (result bool, err error) {
	iface, err := net.InterfaceByName(name)
	if err != nil {
		return
	}
        if iface.Flags&net.FlagUp != 0 {
		result = true
	}
	return
}

func RevertMac(name string) (err error) {
	_, err = net.InterfaceByName(name)
	if err != nil {
		return
	}
	mac, err := GetPermanentMac(name)
	if err != nil {
		return
	}
	err = SetMac(name, mac.String())
	return
}

func RandomizeMac(macbytes net.HardwareAddr, start int, bia bool) (mac net.HardwareAddr, err error) {
	if len(macbytes) != 6 {
		err = fmt.Errorf("Invalid size for macbytes byte array: %d", 
		len(macbytes))
		return
	}
	if (start != 0 && start != 3) {
		err = fmt.Errorf("Invalid start index: %d", start) 
		return
	}
	for i := start; i < 6; i++ {
		buf := make([]byte, 1)
		rand.Read(buf)

		/* The LSB of first octet can not be set.  Those are musticast
         * MAC addresses and not allowed for network device:
         * x1:, x3:, x5:, x7:, x9:, xB:, xD: and xF:
         */
		if i == 0 {
			macbytes[i] = buf[0] & 0xfc
		} else {
			macbytes[i] = buf[0]
		}
	}
	if bia {
		macbytes[0] = macbytes[0]&^2
	} else {
		macbytes[0] |= 2
	}
	mac = macbytes
	return
}

func RunningAsRoot() (result bool, err error) {
	current, err := user.Current()
	if err != nil {
		fmt.Println(err)
	}
	if current.Uid == "0" && current.Gid == "0" && current.Username == "root" {
		result = true
	}
	return 
}

func FindAllPopularOuis() (matches []Oui, err error) {
	for _, oui := range OuiDb {
		if(oui.Popular) {
			matches = append(matches, oui)
		}
	}
	return
}

func (e InvalidInterfaceTypeError) Error() string {
	return e.msg
}

func (e NoVendorError) Error() string {
	return e.msg
}

func FindVendorByMac(mac string) (vendor Oui, err error) {
	err = ValidateMac(mac)
	if err != nil {
		return
	}
	for _, oui := range OuiDb {
		if(strings.EqualFold(oui.VendorPrefix, mac[:8])) {
			vendor = oui
			return
		}
	}
	msg := fmt.Sprintf("No vendor found in OuiDb for vendor prefix: %s", mac[:8])
	err = NoVendorError{msg}
	return
}

func FindDeviceTypeByMac(mac string) (deviceType string, err error) {
	err = ValidateMac(mac)
	if err != nil {
		return
	}
	for _, oui := range OuiDb {
		if(strings.EqualFold(oui.VendorPrefix, mac[:8])) {
			deviceType = oui.Devices[0].DeviceType
			return
		}
	}
	// If vendor prefix is not in OuiDb, return type "Other"
	deviceType = "Other"
	return
}

func FindAllVendorsByDeviceType(deviceType string) (matches []Oui, err error) {
	for _, oui := range OuiDb {
		if(strings.EqualFold(deviceType, oui.Devices[0].DeviceType)) {
			matches = append(matches, oui)
		}
	}
	return
}

func FindVendorsByKeyword(keyword string) (matches []Oui, err error) {
	for _, oui := range OuiDb {
		if(strings.Contains(strings.ToLower(oui.Vendor), strings.ToLower(keyword))) {
			matches = append(matches, oui)
		}
	}
	return
}

func ValidateMac(mac string) (err error) {
	_, err = net.ParseMAC(mac)
	return
}

func IsInterfaceTypeInvalid(name string) (result bool) {
	result = regexp.MustCompile(invalidInterfaceRegexp).MatchString(name)
	return
}

func RandomInt(max int) (result int) {
	mathrand.Seed(time.Now().UTC().UnixNano())
	result = mathrand.Intn(max)
	return
}

