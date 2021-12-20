package main

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/pflag"
	"google.golang.org/grpc"

	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/hooks"
	hooksInfo "kubevirt.io/kubevirt/pkg/hooks/info"
	hooksV1alpha1 "kubevirt.io/kubevirt/pkg/hooks/v1alpha1"
	hooksV1alpha2 "kubevirt.io/kubevirt/pkg/hooks/v1alpha2"
)

const (
	onDefineDomainLoggingMessage = "Hook's OnDefineDomain callback method has been called"
	usage                        = `updater
  --version v1alpha1|v1alpha2
  --args ''`
)

type infoServer struct {
	Version string
}

func (s infoServer) Info(ctx context.Context, params *hooksInfo.InfoParams) (*hooksInfo.InfoResult, error) {
	log.Log.Info("Hook's Info method has been called")

	return &hooksInfo.InfoResult{
		Name: "update-xml",
		Versions: []string{
			s.Version,
		},
		HookPoints: []*hooksInfo.HookPoint{
			{
				Name:     hooksInfo.OnDefineDomainHookPointName,
				Priority: 0,
			},
		},
	}, nil
}

const virtXML = "virt-xml"

type v1alpha1Server struct {
	args []string
}

type v1alpha2Server struct {
	args []string
}

func MergeKubeVirtXMLWithProvidedXML(domainXML []byte, args []string) ([]byte, error) {
	args = append(args, "--edit")
	args = append(args, "--print-xml")
	cmd := exec.Command(virtXML, args...)
	cmd.Stdin = strings.NewReader(string(domainXML))
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	log.Log.Infof("Execute command: %s", cmd.String())
	out, err := cmd.Output()
	if err != nil {
		log.Log.Errorf("Fail running command stdout:%s stderr: %s error:%v", out, stderr.String(), err)
		return []byte{}, err
	}

	return []byte(out), nil
}

func (s v1alpha2Server) OnDefineDomain(ctx context.Context, params *hooksV1alpha2.OnDefineDomainParams) (*hooksV1alpha2.OnDefineDomainResult, error) {
	log.Log.Info(onDefineDomainLoggingMessage)
	newDomainXML, err := MergeKubeVirtXMLWithProvidedXML(params.GetDomainXML(), s.args)
	if err != nil {
		return nil, err
	}
	return &hooksV1alpha2.OnDefineDomainResult{
		DomainXML: newDomainXML,
	}, nil
}
func (s v1alpha2Server) PreCloudInitIso(_ context.Context, params *hooksV1alpha2.PreCloudInitIsoParams) (*hooksV1alpha2.PreCloudInitIsoResult, error) {
	return &hooksV1alpha2.PreCloudInitIsoResult{
		CloudInitData: params.GetCloudInitData(),
	}, nil
}

func (s v1alpha1Server) OnDefineDomain(ctx context.Context, params *hooksV1alpha1.OnDefineDomainParams) (*hooksV1alpha1.OnDefineDomainResult, error) {
	log.Log.Info(onDefineDomainLoggingMessage)
	newDomainXML, err := MergeKubeVirtXMLWithProvidedXML(params.GetVmi(), s.args)
	if err != nil {
		return nil, err
	}
	return &hooksV1alpha1.OnDefineDomainResult{
		DomainXML: newDomainXML,
	}, nil
}

func main() {
	log.InitializeLogging("xml update")

	var version string
	pflag.StringVar(&version, "version", "", "hook version to use")

	var options string
	pflag.StringVar(&options, "args", "", "params to pass to virt-xml")
	pflag.Parse()

	args := strings.Split(options, "|")

	socketPath := hooks.HookSocketsSharedDirectory + "/update.sock"
	socket, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to initialized socket on path: %s", socket)
		log.Log.Error("Check whether given directory exists and socket name is not already taken by other file")
		panic(err)
	}
	defer os.Remove(socketPath)

	server := grpc.NewServer([]grpc.ServerOption{}...)

	if version == "" {
		panic(fmt.Errorf(usage))
	}
	if options == "" {
		panic(fmt.Errorf(usage))
	}

	hooksInfo.RegisterInfoServer(server, infoServer{Version: version})
	hooksV1alpha1.RegisterCallbacksServer(server, v1alpha1Server{args: args})
	hooksV1alpha2.RegisterCallbacksServer(server, v1alpha2Server{args: args})
	log.Log.Infof("Starting hook server exposing 'info' and 'v1alpha1' services on socket %s", socketPath)
	server.Serve(socket)
}
