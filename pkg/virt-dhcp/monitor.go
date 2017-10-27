package virtdhcp

import (
	"os/exec"

	"kubevirt.io/kubevirt/pkg/log"
)

type Monitor interface {
	Start(cmd string, args []string)
	Stop()
	IsRunning() bool
}

type MonitorRuntime struct {
	isRunning bool
	stopChan  chan bool
	pid       int
}

type MonitorCmd struct {
	Stop           chan bool
	processStopped chan bool
	ProcCmd        *exec.Cmd
	cmdArgs        []string
	cmdPath        string
	retries        int
}

func NewMonitor() Monitor {
	mon := MonitorRuntime{
		stopChan: make(chan bool),
		pid:      0,
	}
	return &mon
}

func (runtime *MonitorRuntime) Start(cmd string, args []string) {
	go RunMonitor(cmd, args, runtime.stopChan, &runtime.pid)
	runtime.isRunning = true
}

func (runtime *MonitorRuntime) Stop() {
	runtime.stopChan <- true
	runtime.isRunning = false
}

func (runtime *MonitorRuntime) IsRunning() bool {
	return runtime.isRunning
}

func (mon *MonitorCmd) restart(pid *int) {
	defer func() { mon.processStopped <- true }()
	mon.ProcCmd = exec.Command(mon.cmdPath, mon.cmdArgs...)
	err := mon.ProcCmd.Start()
	mon.retries += 1
	if err != nil {
		log.Log.Reason(err).Error("failed to start dnsmasq")
		return
	}

	*pid = mon.ProcCmd.Process.Pid
	mon.ProcCmd.Wait()
}

func RunMonitor(cmd string, args []string, stopChan chan bool, pid *int) error {

	mon := &MonitorCmd{}
	mon.processStopped = make(chan bool)
	mon.Stop = stopChan
	mon.cmdPath = cmd
	mon.cmdArgs = args
	mon.retries = 0
	done := false
	go mon.restart(pid)

	for !done {
		select {
		case <-mon.processStopped:
			if !done {
				go mon.restart(pid)
			}
		case <-mon.Stop:
			mon.ProcCmd.Process.Kill()
			done = true

		}
		if mon.retries >= 3 {
			panic("failed to start dnsmasq after 3 attempts")
		}
	}
	return nil
}
