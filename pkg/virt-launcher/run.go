package main

import (
	"flag"
	"github.com/rgbkrk/libvirt-go"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	xmlPath := flag.String("domain-path", "/var/run/virt-launcher/dom.xml", "Where to look for the domain xml.")
	conUri := flag.String("libvirt-uri", "qemu:///system", "Libvirt connection string.")
	flag.Parse()
	conn := buildLocalConnection(*conUri)
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

	// Launch VM in paused mode
	dom, createErr := conn.DomainCreateXML(string(xml), 1)
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

func buildLocalConnection(uri string) libvirt.VirConnection {
	conn, err := libvirt.NewVirConnection(uri)
	if err != nil {
		panic(err)
	}
	return conn
}
