package virtdhcp

import (
	"net"
	"net/rpc"
	"os"
	"sync"

	"kubevirt.io/kubevirt/pkg/log"
)

var DhcpSocket string

type DHCP struct {
	name    string
	dnsmasq *DNSmasqInstance
	mutex   *sync.Mutex
}

type RemoveArgs struct {
	IP  string
	Mac string
}

type DhcpReply struct {
	Message string
	Success bool
}

type AddArgs struct {
	IP    string
	Mac   string
	Lease int
}

func (s *DHCP) AddIP(args *AddArgs, reply *DhcpReply) error {
	reply.Success = true
	s.mutex.Lock()
	defer s.mutex.Unlock()
	err := s.dnsmasq.AddHost(args.Mac, args.IP, args.Lease)

	if err != nil {
		log.Log.Reason(err).Errorf("failed to add ip: %s and mac: %s to dhcp", args.IP, args.Mac)
		reply.Success = false
		reply.Message = err.Error()
	}
	return nil
}

func (s *DHCP) RemoveIP(args *RemoveArgs, reply *DhcpReply) error {
	reply.Success = true
	s.mutex.Lock()
	defer s.mutex.Unlock()
	err := s.dnsmasq.RemoveHost(args.IP)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to remove ip: %s and mac: %s from dhcp", args.IP, args.Mac)
		reply.Success = false
		reply.Message = err.Error()
	}
	return nil
}

func (s *DHCP) SetIPs(args *[]AddArgs, reply *DhcpReply) error {
	reply.Success = true
	s.mutex.Lock()
	defer s.mutex.Unlock()
	err := s.dnsmasq.SetKnownHosts(args)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to set known hosts")
		reply.Success = false
		reply.Message = err.Error()
	}
	return nil
}

func createSocket(socketPath string) (net.Listener, error) {

	os.RemoveAll(socketPath)
	socket, err := net.Listen("unix", socketPath)

	if err != nil {
		log.Log.Reason(err).Error("failed to create a socket for dhcp service")
		return nil, err
	}
	return socket, nil
}

func Run(socket string) error {
	DhcpSocket = socket
	var mutex = &sync.Mutex{}

	rpcServer := rpc.NewServer()
	dnsmasq := NewDNSmasq()
	server := &DHCP{name: "virt-dhcp",
		dnsmasq: dnsmasq,
		mutex:   mutex}
	rpcServer.Register(server)
	dnsmasq.loadHostsFile()
	sock, err := createSocket(DhcpSocket)
	if err != nil {
		return err
	}

	defer func() {
		sock.Close()
		os.Remove(DhcpSocket)
	}()
	rpcServer.Accept(sock)

	return nil
}
