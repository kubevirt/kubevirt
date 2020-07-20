package water

import (
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"unsafe"
)

const appleUTUNCtl = "com.apple.net.utun_control"

/*
 * From ioctl.h:
 * #define	IOCPARM_MASK	0x1fff		// parameter length, at most 13 bits
 * ...
 * #define	IOC_OUT		0x40000000	// copy out parameters
 * #define	IOC_IN		0x80000000	// copy in parameters
 * #define	IOC_INOUT	(IOC_IN|IOC_OUT)
 * ...
 * #define _IOC(inout,group,num,len) \
 * 	(inout | ((len & IOCPARM_MASK) << 16) | ((group) << 8) | (num))
 * ...
 * #define	_IOWR(g,n,t)	_IOC(IOC_INOUT,	(g), (n), sizeof(t))
 *
 * From kern_control.h:
 * #define CTLIOCGINFO     _IOWR('N', 3, struct ctl_info)	// get id from name
 *
 */

const appleCTLIOCGINFO = (0x40000000 | 0x80000000) | ((100 & 0x1fff) << 16) | uint32(byte('N'))<<8 | 3

/*
 * #define _IOW(g,n,t) _IOC(IOC_IN, (g), (n), sizeof(t))
 * #define TUNSIFMODE _IOW('t', 94, int)
 */
const appleTUNSIFMODE = (0x80000000) | ((4 & 0x1fff) << 16) | uint32(byte('t'))<<8 | 94

/*
 * struct sockaddr_ctl {
 *     u_char sc_len; // depends on size of bundle ID string
 *     u_char sc_family; // AF_SYSTEM
 *     u_int16_t ss_sysaddr; // AF_SYS_KERNCONTROL
 *     u_int32_t sc_id; // Controller unique identifier
 *     u_int32_t sc_unit; // Developer private unit number
 *     u_int32_t sc_reserved[5];
 * };
 */
type sockaddrCtl struct {
	scLen      uint8
	scFamily   uint8
	ssSysaddr  uint16
	scID       uint32
	scUnit     uint32
	scReserved [5]uint32
}

var sockaddrCtlSize uintptr = 32

func openDev(config Config) (ifce *Interface, err error) {
	if config.Driver == MacOSDriverTunTapOSX {
		return openDevTunTapOSX(config)
	}
	if config.Driver == MacOSDriverSystem {
		return openDevSystem(config)
	}
	return nil, errors.New("unrecognized driver")
}

// openDevSystem opens tun device on system
func openDevSystem(config Config) (ifce *Interface, err error) {
	if config.DeviceType != TUN {
		return nil, errors.New("only tun is implemented for SystemDriver, use TunTapOSXDriver for tap")
	}

	ifIndex := -1
	if config.Name != "" {
		const utunPrefix = "utun"
		if !strings.HasPrefix(config.Name, utunPrefix) {
			return nil, fmt.Errorf("Interface name must be utun[0-9]+")
		}
		ifIndex, err = strconv.Atoi(config.Name[len(utunPrefix):])
		if err != nil || ifIndex < 0 || ifIndex > math.MaxUint32-1 {
			return nil, fmt.Errorf("Interface name must be utun[0-9]+")
		}
	}

	var fd int
	// Supposed to be socket(PF_SYSTEM, SOCK_DGRAM, SYSPROTO_CONTROL), but ...
	//
	// In sys/socket.h:
	// #define PF_SYSTEM	AF_SYSTEM
	//
	// In sys/sys_domain.h:
	// #define SYSPROTO_CONTROL       	2	/* kernel control protocol */
	if fd, err = syscall.Socket(syscall.AF_SYSTEM, syscall.SOCK_DGRAM, 2); err != nil {
		return nil, fmt.Errorf("error in syscall.Socket: %v", err)
	}

	var ctlInfo = &struct {
		ctlID   uint32
		ctlName [96]byte
	}{}
	copy(ctlInfo.ctlName[:], []byte(appleUTUNCtl))

	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), uintptr(appleCTLIOCGINFO), uintptr(unsafe.Pointer(ctlInfo))); errno != 0 {
		err = errno
		return nil, fmt.Errorf("error in syscall.Syscall(syscall.SYS_IOCTL, ...): %v", err)
	}

	addrP := unsafe.Pointer(&sockaddrCtl{
		scLen:    uint8(sockaddrCtlSize),
		scFamily: syscall.AF_SYSTEM,

		/* #define AF_SYS_CONTROL 2 */
		ssSysaddr: 2,

		scID:   ctlInfo.ctlID,
		scUnit: uint32(ifIndex) + 1,
	})
	if _, _, errno := syscall.RawSyscall(syscall.SYS_CONNECT, uintptr(fd), uintptr(addrP), uintptr(sockaddrCtlSize)); errno != 0 {
		err = errno
		return nil, fmt.Errorf("error in syscall.RawSyscall(syscall.SYS_CONNECT, ...): %v", err)
	}

	var ifName struct {
		name [16]byte
	}
	ifNameSize := uintptr(16)
	if _, _, errno := syscall.Syscall6(syscall.SYS_GETSOCKOPT, uintptr(fd),
		2, /* #define SYSPROTO_CONTROL 2 */
		2, /* #define UTUN_OPT_IFNAME 2 */
		uintptr(unsafe.Pointer(&ifName)),
		uintptr(unsafe.Pointer(&ifNameSize)), 0); errno != 0 {
		err = errno
		return nil, fmt.Errorf("error in syscall.Syscall6(syscall.SYS_GETSOCKOPT, ...): %v", err)
	}

	if err = setNonBlock(fd); err != nil {
		return nil, fmt.Errorf("setting non-blocking error")
	}

	return &Interface{
		isTAP: false,
		name:  string(ifName.name[:ifNameSize-1 /* -1 is for \0 */]),
		ReadWriteCloser: &tunReadCloser{
			f: os.NewFile(uintptr(fd), string(ifName.name[:])),
		},
	}, nil
}

// openDevTunTapOSX opens tun / tap device, assuming tuntaposx is installed
func openDevTunTapOSX(config Config) (ifce *Interface, err error) {
	var fd int
	var socketFD int

	if config.DeviceType == TAP && !strings.HasPrefix(config.Name, "tap") {
		return nil, errors.New("device name does not start with tap when creating a tap device")
	}
	if config.DeviceType == TUN && !strings.HasPrefix(config.Name, "tun") {
		return nil, errors.New("device name does not start with tun when creating a tun device")
	}
	if config.DeviceType != TAP && config.DeviceType != TUN {
		return nil, errors.New("unsupported DeviceType")
	}
	if len(config.Name) >= 15 {
		return nil, errors.New("device name is too long")
	}

	if fd, err = syscall.Open(
		"/dev/"+config.Name, os.O_RDWR|syscall.O_NONBLOCK, 0); err != nil {
		return nil, err
	}
	// Note that we are not setting NONBLOCK on the fd itself since it breaks tuntaposx
	// see https://sourceforge.net/p/tuntaposx/bugs/6/

	// create socket so we can do SIO ioctls, we are not using it afterwards
	if socketFD, err = syscall.Socket(syscall.AF_SYSTEM, syscall.SOCK_DGRAM, 2); err != nil {
		return nil, fmt.Errorf("error in syscall.Socket: %v", err)
	}
	var ifReq = &struct {
		ifName    [16]byte
		ifruFlags int16
		pad       [16]byte
	}{}
	copy(ifReq.ifName[:], []byte(config.Name))
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(socketFD), uintptr(syscall.SIOCGIFFLAGS), uintptr(unsafe.Pointer(ifReq))); errno != 0 {
		err = errno
		return nil, fmt.Errorf("error in syscall.Syscall(syscall.SYS_IOCTL, ...): %v", err)
	}
	ifReq.ifruFlags |= syscall.IFF_RUNNING | syscall.IFF_UP
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(socketFD), uintptr(syscall.SIOCSIFFLAGS), uintptr(unsafe.Pointer(ifReq))); errno != 0 {
		err = errno
		return nil, fmt.Errorf("error in syscall.Syscall(syscall.SYS_IOCTL, ...): %v", err)
	}
	syscall.Close(socketFD)

	return &Interface{
		isTAP:           config.DeviceType == TAP,
		ReadWriteCloser: os.NewFile(uintptr(fd), "tun"),
		name:            config.Name,
	}, nil
}

// tunReadCloser is a hack to work around the first 4 bytes "packet
// information" because there doesn't seem to be an IFF_NO_PI for darwin.
type tunReadCloser struct {
	f io.ReadWriteCloser

	rMu  sync.Mutex
	rBuf []byte

	wMu  sync.Mutex
	wBuf []byte
}

var _ io.ReadWriteCloser = (*tunReadCloser)(nil)

func (t *tunReadCloser) Read(to []byte) (int, error) {
	t.rMu.Lock()
	defer t.rMu.Unlock()

	if cap(t.rBuf) < len(to)+4 {
		t.rBuf = make([]byte, len(to)+4)
	}
	t.rBuf = t.rBuf[:len(to)+4]

	n, err := t.f.Read(t.rBuf)
	copy(to, t.rBuf[4:])
	return n - 4, err
}

func (t *tunReadCloser) Write(from []byte) (int, error) {

	if len(from) == 0 {
		return 0, syscall.EIO
	}

	t.wMu.Lock()
	defer t.wMu.Unlock()

	if cap(t.wBuf) < len(from)+4 {
		t.wBuf = make([]byte, len(from)+4)
	}
	t.wBuf = t.wBuf[:len(from)+4]

	// Determine the IP Family for the NULL L2 Header
	ipVer := from[0] >> 4
	if ipVer == 4 {
		t.wBuf[3] = syscall.AF_INET
	} else if ipVer == 6 {
		t.wBuf[3] = syscall.AF_INET6
	} else {
		return 0, errors.New("Unable to determine IP version from packet")
	}

	copy(t.wBuf[4:], from)

	n, err := t.f.Write(t.wBuf)
	return n - 4, err
}

func (t *tunReadCloser) Close() error {
	return t.f.Close()
}
