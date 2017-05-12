package spice

import (
	"errors"
	"fmt"
	flag "github.com/spf13/pflag"
	"io/ioutil"
	kubev1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/rest"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"log"
	"os"
	"os/exec"
)

const FLAG = "spice"
const TEMP_PREFIX = "spice"

type Spice struct {
}

func DownloadSpice(vm string, restClient *rest.RESTClient) (string, error) {
	body, err := restClient.Get().
		Resource("vms").SetHeader("Accept", "text/plain").
		SubResource("spice").
		Namespace(kubev1.NamespaceDefault).
		Name(vm).Do().Raw()
	if err != nil {
		return "", errors.New(fmt.Sprintf("Can't read body: %s\n", err.Error()))
	}
	return fmt.Sprintf("%s", body), nil
}

func (o *Spice) FlagSet() *flag.FlagSet {

	cf := flag.NewFlagSet(FLAG, flag.ExitOnError)
	cf.BoolP("details", "d", false, "If present, print SPICE console to stdout, otherwise run remote-viewer")
	return cf
}

func (o *Spice) Run(flags *flag.FlagSet) int {
	server, _ := flags.GetString("server")
	kubeconfig, _ := flags.GetString("kubeconfig")
	details, _ := flags.GetBool("details")

	if len(flags.Args()) != 2 {
		log.Println("VM name is missing")
		return 1
	}
	vm := flags.Arg(1)

	restClient, err := kubecli.GetRESTClientFromFlags(server, kubeconfig)

	if err != nil {
		log.Println(err)
		return 1
	}
	body, err := DownloadSpice(vm, restClient)
	if err != nil {
		log.Fatalf(err.Error())
		return 1
	}
	if details {
		fmt.Printf("%s", body)
	} else {
		f, err := ioutil.TempFile("", TEMP_PREFIX)

		if err != nil {
			log.Fatalf("Can't open file: %s", err.Error())
			return 1
		}
		defer os.Remove(f.Name())
		defer f.Close()

		_, err = f.WriteString(body)
		if err != nil {
			log.Fatalf("Can't write to file: %s", err.Error())
			return 1
		}

		f.Sync()

		cmnd := exec.Command("remote-viewer", f.Name())
		err = cmnd.Run()

		if err != nil {
			log.Fatalf("Something goes wring with remote-viewer: %s", err.Error())
			return 1
		}
	}
	return 0
}

func (o *Spice) Usage() string {
	usage := "virtctl can connect via remote-viewer to VM, or show SPICE connection details\n\n"
	usage += "Examples:\n"
	usage += "# Show SPICE connection details of the VM testvm\n"
	usage += "./virtctl spice testvm --details\n\n"
	usage += "# Connect to testvm via remote-viewer\n"
	usage += "./virtctl spice testvm\n\n"
	usage += "The following options can be passed to any command:\n\n"
	usage += o.FlagSet().FlagUsages()
	return usage
}
