package virtctl

import (
	"fmt"
	flag "github.com/spf13/pflag"
	"os"
)

type App interface {
	FlagSet() *flag.FlagSet
	Run(flags *flag.FlagSet) int
	Usage() string
}

type Options struct {
}

func (o *Options) FlagSet() *flag.FlagSet {

	cf := flag.NewFlagSet("options", flag.ExitOnError)
	cf.StringP("server", "s", "", "The address and port of the Kubernetes API server")
	cf.StringP("namespace", "n", "default", "If present, the namespace scope for this CLI request")
	cf.String("kubeconfig", "", "Path to the kubeconfig file to use for CLI requests")
	return cf
}

func (o *Options) Run(flags *flag.FlagSet) int {
	fmt.Fprintln(os.Stderr, o.Usage())
	return 0
}

func (o *Options) Usage() string {
	usage := "The following options can be passed to any command:\n\n"
	usage += o.FlagSet().FlagUsages()
	return usage
}
