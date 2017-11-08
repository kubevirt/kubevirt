package virtdhcp

import (
	"os/exec"
	"sync"
	"time"

	"kubevirt.io/kubevirt/pkg/log"
)

type Monitor interface {
	Start(cmd string, args []string)
	Stop()
}

type MonitorRuntime struct {
	isRunning bool
	stopChan  chan bool
	pid       int
	wg        sync.WaitGroup
	lock      sync.Mutex
}

func NewMonitor() Monitor {
	mon := &MonitorRuntime{
		stopChan: make(chan bool),
		pid:      0,
	}
	return mon
}

func (runtime *MonitorRuntime) Start(cmd string, args []string) {
	runtime.lock.Lock()
	defer runtime.lock.Unlock()

	if runtime.isRunning == true {
		return
	}

	runtime.wg.Add(1)
	go func() {
		defer runtime.wg.Done()
		runMonitor(cmd, args, runtime.stopChan, &runtime.pid)
	}()
	runtime.isRunning = true
}

func (runtime *MonitorRuntime) Stop() {
	runtime.lock.Lock()
	defer runtime.lock.Unlock()

	if runtime.isRunning == false {
		return
	}

	runtime.stopChan <- true
	runtime.isRunning = false

	runtime.wg.Wait()
}

func runMonitor(cmd string, args []string, stopChan chan bool, pid *int) {
	retries := 0
	done := false
	processStopped := make(chan bool)

	for !done {
		if retries >= 3 {
			panic("failed to start dnsmasq after 3 attempts")
		}

		// start process
		procCmd := exec.Command(cmd, args...)
		err := procCmd.Start()
		if err != nil {
			retries += 1
			log.Log.Reason(err).Error("failed to start dnsmasq")
			continue
		}

		*pid = procCmd.Process.Pid

		// wait for process to exit.
		go func() {
			procCmd.Wait()
			processStopped <- true
		}()

		// react to process exiting early or stop request.
		select {
		case <-processStopped:
			retries += 1
			log.Log.Reason(err).Error("dnsmasq exited early")
			// add a second between unexpected restarts
			time.Sleep(time.Second)
		case <-stopChan:
			procCmd.Process.Kill()
			// we expect the process to stop now, so wait for it.
			<-processStopped
			done = true
		}
	}
}
