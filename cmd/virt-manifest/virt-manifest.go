package main

import (
	"flag"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/emicklei/go-restful"

	"kubevirt.io/kubevirt/pkg/logging"
	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap"
	"kubevirt.io/kubevirt/pkg/virt-manifest/rest"
)

func main() {
	logging.InitializeLogging("virt-manifest")
	libvirtUri := flag.String("libvirt-uri", "qemu:///system", "Libvirt connection string.")
	listen := flag.String("listen", "0.0.0.0", "Address where to listen on")
	port := flag.Int("port", 8186, "Port to listen on")
	flag.Parse()

	log := logging.DefaultLogger()
	log.Info().Msg("Starting virt-manifest server")

	log.Info().Msg("Connecting to libvirt")

	domainConn, err := virtwrap.NewConnection(*libvirtUri, "", "", 60*time.Second)
	if err != nil {
		log.Error().Reason(err).Msg("cannot connect to libvirt")
		panic(fmt.Sprintf("failed to connect to libvirt: %v", err))
	}
	defer domainConn.Close()

	log.Info().Msg("Connected to libvirt")

	ws, err := rest.ManifestService(domainConn)
	if err != nil {
		log.Error().Reason(err).Msg("Unable to create REST server.")
	}

	restful.DefaultContainer.Add(ws)
	server := &http.Server{Addr: *listen + ":" + strconv.Itoa(*port), Handler: restful.DefaultContainer}
	log.Info().Msg("Listening for client connections")

	if err := server.ListenAndServe(); err != nil {
		log.Error().Reason(err).Msg("Unable to start web server.")
	}
}
