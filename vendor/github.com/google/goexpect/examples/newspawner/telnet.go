// telnet creates a new Expect spawner for Telnet.
package main

import (
	"flag"
	"fmt"
	"time"

	expect "github.com/google/goexpect"

	"github.com/golang/glog"
	"github.com/google/goterm/term"
	"github.com/ziutek/telnet"
)

const (
	network = "tcp"
	address = "telehack.com:23"
	timeout = 10 * time.Second
	command = "geoip"
)

func main() {
	flag.Parse()

	fmt.Println(term.Bluef("Telnet spawner example"))
	exp, _, err := telnetSpawn(address, timeout, expect.Verbose(true))
	if err != nil {
		glog.Exitf("telnetSpawn(%q,%v) failed: %v", address, timeout, err)
	}

	defer func() {
		if err := exp.Close(); err != nil {
			glog.Infof("exp.Close failed: %v", err)
		}
	}()

	res, err := exp.ExpectBatch([]expect.Batcher{
		&expect.BExp{R: `\n\.`},
		&expect.BSnd{S: command + "\r\n"},
		&expect.BExp{R: `\n\.`},
	}, timeout)
	if err != nil {
		glog.Exitf("exp.ExpectBatch failed: %v , res: %v", err, res)
	}
	fmt.Println(term.Greenf("Res: %s", res[len(res)-1].Output))

}

func telnetSpawn(addr string, timeout time.Duration, opts ...expect.Option) (expect.Expecter, <-chan error, error) {
	conn, err := telnet.Dial(network, addr)
	if err != nil {
		return nil, nil, err
	}

	resCh := make(chan error)

	return expect.SpawnGeneric(&expect.GenOptions{
		In:  conn,
		Out: conn,
		Wait: func() error {
			return <-resCh
		},
		Close: func() error {
			close(resCh)
			return conn.Close()
		},
		Check: func() bool { return true },
	}, timeout, opts...)
}
