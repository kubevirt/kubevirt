package main

import (
	"flag"
	"github.com/rmohr/libvirt-go"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	xmlPath := flag.String("domain-path", "/var/run/virt-launcher/dom.xml", "Where to look for the domain xml.")
	conUri := flag.String("libvirt-uri", "qemu:///system", "Libvirt connection string.")
	user := flag.String("user", "vdsm@ovirt", "Libvirt user")
	pass := flag.String("pass", "shibboleth", "Libvirt password")
	flag.Parse()
	conn := buildLocalConnection(*conUri, *user, *pass)
	log.Print("Libvirt connection established.")

	defer func() {
		if res, _ := conn.CloseConnection(); res != 0 {
			log.Fatalf("CloseConnection() == %d, expected 0", res)
		}
	}()

	xml, readErr := ioutil.ReadFile(*xmlPath)
	if readErr != nil {
		log.Fatal(readErr)
	}
	log.Print("Domain description loaded.")

	// Launch VM
	dom, createErr := conn.DomainCreateXML(string(xml), 0)
	if createErr != nil {
		log.Fatal(createErr)
	}
	log.Print("Domain started in pause mode.")

	// Wait for termination
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)

	log.Print("Waiting forever ...")
	s := <-c
	log.Print("Got signal:", s)

	destroyErr := dom.Destroy()
	if destroyErr != nil {
		log.Fatal(destroyErr)
	}
	log.Print("Domain destroyed.")
}

func buildLocalConnection(uri string, user string, pass string) libvirt.VirConnection {
	conn, err := libvirt.NewVirConnectionWithAuth(uri, user, pass)
	if err != nil {
		panic(err)
	}
	return conn
}
