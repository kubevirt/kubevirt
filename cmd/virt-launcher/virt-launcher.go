package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"bufio"
	"github.com/rgbkrk/libvirt-go"
	"html"
	"strconv"
	"strings"
)

type virtlauncher struct {
	domainXML  string
	domainUUID string
	connURI    string
	user       string
	pass       string
}

type monitor struct {
	timeout   time.Duration
	pid       int
	exename   string
	start     time.Time
	isDone    bool
	debugMode bool
}

func (mon *monitor) refresh() {
	if mon.isDone {
		log.Print("Called refresh after done!")
		return
	}

	if mon.debugMode {
		log.Printf("Refreshing executable %s pid %d", mon.exename, mon.pid)
	}

	// is the procecess there?
	if mon.pid == 0 {
		var err error
		mon.pid, err = pidOf(mon.exename)
		if err == nil {
			log.Printf("Found PID for %s: %d", mon.exename, mon.pid)
		} else {
			if mon.debugMode {
				log.Printf("Missing PID for %s", mon.exename)
			}
			// if the proces is not there yet, is it too late?
			elapsed := time.Since(mon.start)
			if mon.timeout > 0 && elapsed >= mon.timeout {
				log.Printf("%s not found after timeout", mon.exename)
				mon.isDone = true
			}
		}
		return
	}

	// is the process gone? mon.pid != 0 -> mon.pid == 0
	// note libvirt deliver one event for this, but since we need
	// to poll procfs anyway to detect incoming QEMUs after migrations,
	// we choose to not use this. Bonus: we can close the connection
	// and open it only when needed, which is a tiny part of the
	// virt-launcher lifetime.
	if !pidExists(mon.pid) {
		log.Printf("Process %s is gone!", mon.exename)
		mon.pid = 0
		mon.isDone = true
		return
	}

	return
}

func (mon *monitor) RunForever(startTimeout time.Duration) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)

	// random value, no real rationale
	rate := 500 * time.Millisecond

	if mon.debugMode {
		timeoutRepr := fmt.Sprintf("%v", startTimeout)
		if startTimeout == 0 {
			timeoutRepr = "disabled"
		}
		log.Printf("Monitoring loop: rate %v start timeout %s", rate, timeoutRepr)
	}

	ticker := time.NewTicker(rate)

	gotSignal := false
	mon.isDone = false
	mon.timeout = startTimeout
	mon.start = time.Now()

	log.Printf("Waiting forever...")
	for !gotSignal && !mon.isDone {
		select {
		case <-ticker.C:
			mon.refresh()
		case s := <-c:
			log.Print("Got signal: ", s)
			gotSignal = true
		}
	}

	ticker.Stop()
	log.Printf("Exiting...")
}

func main() {
	startTimeout := 0 * time.Second

	xmlPath := flag.String("domain-path", "/var/run/virt-launcher/dom.xml", "Where to look for the domain xml.")
	downwardAPIPath := flag.String("downward-api-path", "", "Load domain from this downward API file")
	conUri := flag.String("libvirt-uri", "qemu:///system", "Libvirt connection string.")
	user := flag.String("user", "vdsm@ovirt", "Libvirt user")
	pass := flag.String("pass", "shibboleth", "Libvirt password")
	receiveOnly := flag.Bool("receive-only", false, "Do not create the domain")
	qemuTimeout := flag.Duration("qemu-timeout", startTimeout, "Amount of time to wait for qemu")
	debugMode := flag.Bool("debug", false, "Enable debug messages")
	flag.Parse()

	mon := monitor{
		exename:   "qemu",
		debugMode: *debugMode,
	}

	launcher := virtlauncher{
		connURI: *conUri,
		user:    *user,
		pass:    *pass,
	}

	if !*receiveOnly {
		launcher.ReadDomainXML(*xmlPath, *downwardAPIPath)
		launcher.CreateDomain()
	}

	mon.RunForever(*qemuTimeout)
}

func (vl *virtlauncher) CreateDomain() {
	conn, err := libvirt.NewVirConnectionWithAuth(vl.connURI, vl.user, vl.pass)
	if err != nil {
		panic(fmt.Sprintf("Could not connect to libvirt using %s: %s", vl.connURI, err))
	}
	defer func() {
		if _, closeErr := conn.CloseConnection(); closeErr != nil {
			log.Fatalf("CloseConnection() failed: %s", closeErr)
		}
		log.Print("Connection closed")
	}()

	log.Print("Libvirt connection established")

	// Launch VM
	_, err = conn.DomainCreateXML(vl.domainXML, 0)
	if err != nil {
		panic(fmt.Sprintf("Could not create the libvirt domain: %s", err))
	}

	log.Print("Domain started")
}

func (vl *virtlauncher) ReadDomainXML(xmlPath string, downwardAPIPath string) {
	if downwardAPIPath == "" {
		log.Print("Loading Domain from XML file")
		rawXML, err := ioutil.ReadFile(xmlPath)
		if err != nil {
			log.Fatal(err)
		}
		vl.domainXML = string(rawXML)
	} else {
		log.Print("Loading Domain from downward API file")
		f, err := os.Open(downwardAPIPath)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, `domainXML="`) {
				vl.domainXML = DecodeDomainXML(strings.Trim(strings.TrimPrefix(line, "domainXML="), `"`))
			}
		}

	}

	if vl.domainXML == "" {
		panic("Could not load domain XML. The resulting XML is empty")
	}
	log.Print("Domain description loaded.")
}

func DecodeDomainXML(domainXML string) string {
	decodedXML := html.UnescapeString(string(domainXML))
	decodedXML = strings.Replace(decodedXML, "\\\\", "\\", -1)
	decodedXML = strings.Replace(decodedXML, "\\n", "\n", -1)
	return decodedXML
}

func readProcCmdline(pathname string) ([]string, error) {
	content, err := ioutil.ReadFile(pathname)
	if err != nil {
		return nil, err
	}

	return strings.Split(string(content), "\x00"), nil
}

func pidOf(exename string) (int, error) {
	entries, err := filepath.Glob("/proc/*/cmdline")
	if err != nil {
		return 0, err
	}
	for _, entry := range entries {
		argv, err := readProcCmdline(entry)
		if err != nil {
			return 0, err
		}

		// we need to support both
		// - /usr/bin/qemu-system-$ARCH (fedora)
		// - /usr/libexec/qemu-kvm (*EL, CentOS)
		match, _ := filepath.Match(fmt.Sprintf("%s*", exename), filepath.Base(argv[0]))

		if match {
			//   <empty> /    proc     /    $PID   /   cmdline
			// items[0] sep items[1] sep items[2] sep  items[3]
			items := strings.Split(entry, string(os.PathSeparator))
			pid, err := strconv.Atoi(items[2])
			if err != nil {
				return 0, err
			}

			return pid, nil
		}
	}
	return 0, fmt.Errorf("Process %s not found in /proc", exename)
}

func pidExists(pid int) bool {
	path := fmt.Sprintf("/proc/%d/cmdline", pid)
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}
