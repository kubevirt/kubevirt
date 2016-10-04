package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"syscall"

	"bufio"
	"github.com/rgbkrk/libvirt-go"
	"html"
	"strings"
)

func main() {
	xmlPath := flag.String("domain-path", "/var/run/virt-launcher/dom.xml", "Where to look for the domain xml.")
	downwardAPIPath := flag.String("downward-api-path", "", "Load domain from this downward API file")
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

	var xml string
	if downwardAPIPath == nil {
		log.Print("Loading Domain from XML file.")
		rawXML, err := ioutil.ReadFile(*xmlPath)
		if err != nil {
			log.Fatal(err)
		}
		xml = string(rawXML)
	} else {
		log.Print("Loading Domain from downward API file.")
		f, err := os.Open(*downwardAPIPath)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, `domainXML="`) {
				xml = DecodeDomainXML(strings.Trim(strings.TrimPrefix(line, "domainXML="), `"`))
			}
		}

	}
	if xml == "" {
		log.Fatal("Could not load domain XML. The resulting XML is empty")
	}
	log.Print("Domain description loaded.")

	// Launch VM
	dom, createErr := conn.DomainCreateXML(xml, 0)
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

func DecodeDomainXML(domainXML string) string {
	decodedXML := html.UnescapeString(string(domainXML))
	decodedXML = strings.Replace(decodedXML, "\\\\", "\\", -1)
	decodedXML = strings.Replace(decodedXML, "\\n", "\n", -1)
	return decodedXML
}
