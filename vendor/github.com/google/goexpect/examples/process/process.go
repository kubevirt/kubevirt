// process is a simple example of spawning a process from the expect package.
package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/golang/glog"
	expect "github.com/google/goexpect"
	"github.com/google/goterm/term"
)

const (
	command = `bc -l`
	timeout = 10 * time.Minute
)

var piRE = regexp.MustCompile(`3.14[0-9]*`)

func main() {
	flag.Parse()
	if flag.NArg() != 1 {
		glog.Exitf("Usage: process <nr of digits>")
	}

	if err := os.Setenv("BC_LINE_LENGTH", "0"); err != nil {
		glog.Exit(err)
	}

	scale, err := strconv.Atoi(flag.Arg(0))
	if err != nil {
		glog.Exit(err)
	}

	if scale < 3 {
		glog.Exitf("scale must be at least 3 for this sample to work")
	}

	e, _, err := expect.Spawn(command, -1)
	if err != nil {
		glog.Exit(err)
	}

	if err := e.Send("scale=" + strconv.Itoa(scale) + "\n"); err != nil {
		glog.Exit(err)
	}
	if err := e.Send("4*a(1)\n"); err != nil {
		glog.Exit(err)
	}
	out, match, err := e.Expect(piRE, timeout)
	if err != nil {
		glog.Exitf("e.Expect(%q,%v) failed: %v, out: %q", piRE.String(), timeout, err, out)
	}

	fmt.Println(term.Bluef("Pi with %d digits: %s", scale, match[0]))
}
