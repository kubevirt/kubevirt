package linux

import (
	"errors"
	"io/ioutil"
	"strconv"
	"strings"
)

type NetUnixDomainSockets struct {
	Sockets []NetUnixDomainSocket `json:"sockets"`
}

type NetUnixDomainSocket struct {
	Protocol uint64 `json:"protocol"`
	RefCount uint64 `json:"ref_count"`
	Flags    uint64 `json:"flags"`
	Type     uint64 `json:"type"`
	State    uint64 `json:"state"`
	Inode    uint64 `json:"inode"`
	Path     string `json:"path"`
}

// ReadNetUnixDomainSockets parser to /proc/net/unix
func ReadNetUnixDomainSockets(fpath string) (*NetUnixDomainSockets, error) {
	b, err := ioutil.ReadFile(fpath)

	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(b), "\n")
	unixDomainSockets := &NetUnixDomainSockets{}

	for i := 1; i < len(lines); i++ {
		line := lines[i]
		f := strings.Fields(line)

		if len(f) < 8 {
			continue
		}

		socket := NetUnixDomainSocket{}
		if socket.RefCount, err = strconv.ParseUint(f[1], 16, 64); err != nil {
			return nil, errors.New("Cannot parse unix domain socket [invalid RefCount]: " + f[1])
		}

		if socket.Protocol, err = strconv.ParseUint(f[2], 10, 64); err != nil {
			return nil, errors.New("Cannot parse unix domain socket [invalid Protocol]: " + f[2])
		}

		if socket.Flags, err = strconv.ParseUint(f[3], 10, 64); err != nil {
			return nil, errors.New("Cannot parse unix domain socket [invalid Flags]: " + f[3])
		}

		if socket.Type, err = strconv.ParseUint(f[4], 10, 64); err != nil {
			return nil, errors.New("Cannot parse unix domain socket [invalid Type]: " + f[4])
		}

		if socket.State, err = strconv.ParseUint(f[5], 10, 64); err != nil {
			return nil, errors.New("Cannot parse unix domain socket [invalid State]: " + f[5])
		}

		if socket.Inode, err = strconv.ParseUint(f[6], 10, 64); err != nil {
			return nil, errors.New("Cannot parse unix domain socket [invalid Inode]: " + f[6])
		}

		socket.Path = f[7]
		unixDomainSockets.Sockets = append(unixDomainSockets.Sockets, socket)
	}
	return unixDomainSockets, nil
}
