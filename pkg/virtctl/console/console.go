package console

import (
	"bytes"
	"fmt"
	"github.com/gorilla/websocket"
	flag "github.com/spf13/pflag"
	"golang.org/x/crypto/ssh/terminal"
	"io"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/clientcmd"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"time"
)

type Console struct {
}

func (c *Console) FlagSet() *flag.FlagSet {
	cf := flag.NewFlagSet("console", flag.ExitOnError)
	cf.StringP("device", "d", "", "Console to connect to")

	return cf
}

func (c *Console) Usage() string {
	usage := "Connect to a serial console on a VM:\n\n"
	usage += "Examples:\n"
	usage += "# Connect to the console 'serial0' on the VM 'myvm':\n"
	usage += "virtctl console myvm --device serial0\n\n"
	usage += "Options:\n"
	usage += c.FlagSet().FlagUsages()
	return usage
}

func (c *Console) Run(flags *flag.FlagSet) int {

	server, _ := flags.GetString("server")
	kubeconfig, _ := flags.GetString("kubeconfig")
	namespace, _ := flags.GetString("namespace")
	device, _ := flags.GetString("device")
	if namespace == "" {
		namespace = v1.NamespaceDefault
	}
	if len(flags.Args()) != 2 {
		log.Println("VM name is missing")
		return 1
	}
	vm := flags.Arg(1)

	config, err := clientcmd.BuildConfigFromFlags(server, kubeconfig)
	if err != nil {
		log.Println(err)
		return 1
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	u, err := url.Parse(config.Host)
	if err != nil {
		log.Fatal("dial:", err)
	}
	u.Scheme = "ws"
	u.Path = fmt.Sprintf("/apis/kubevirt.io/v1alpha1/namespaces/%s/vms/%s/console", namespace, vm)
	if device != "" {
		u.RawQuery = "console=" + device
	}
	log.Printf("connecting to %s", u.String())

	ws, resp, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		if resp != nil && resp.StatusCode != http.StatusOK {
			buf := new(bytes.Buffer)
			buf.ReadFrom(resp.Body)
			log.Fatalf("Can't connect to console (%d): %s\n", resp.StatusCode, buf.String())
		}
		log.Fatalf("Can't connect to console: %s\n", err.Error())
	}
	defer ws.Close()

	writeStop := make(chan struct{})
	readStop := make(chan struct{})

	state, err := terminal.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		log.Fatal("Make raw terminal failed:", err)
	}
	defer terminal.Restore(int(os.Stdin.Fd()), state)
	fmt.Fprint(os.Stderr, "Escape sequence is ^]")

	go func() {
		defer close(readStop)
		for {
			_, message, err := ws.ReadMessage()
			if err != nil {
				os.Stdout.Write(message)
				return
			}
			os.Stdout.Write(message)
		}
	}()

	buf := make([]byte, 1024, 1024)
	go func() {
		defer close(writeStop)
		for {
			n, err := os.Stdin.Read(buf)
			if err != nil && err != io.EOF {
				log.Println(err)
				return
			}

			if buf[0] == 29 {
				return
			}

			err = ws.WriteMessage(websocket.TextMessage, buf[0:n])
			if err != nil && err != io.EOF {
				log.Println(err)
				return
			}
		}
	}()

	select {
	case <-interrupt:
	case <-readStop:
	case <-writeStop:
	}

	err = ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		log.Fatalf("Error on close announcement: %s", err.Error())
	}
	select {
	case <-readStop:
	case <-time.After(time.Second):
	}
	return 0
}
