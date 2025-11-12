/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright The KubeVirt Authors.
 *
 */

package ksm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	k8scorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/controller"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

const (
	pagesBoostDefault      = 300
	pagesDecayDefault      = -50
	nPagesMinDefault       = 64
	nPagesMaxDefault       = 1250
	nPagesInitDefault      = 100
	sleepMsBaselineDefault = 100 // 10ms in oVirt seemed really low
	freePercentDefault     = 0.2
)

var (
	// These are vars so they can be changed by the unit tests

	// In some environments, sysfs is mounted read-only even for privileged
	// containers: https://github.com/containerd/containerd/issues/8445.
	// Use the path from the host filesystem.
	ksmBasePath  = "/proc/1/root/sys/kernel/mm/ksm/"
	ksmRunPath   = ksmBasePath + "run"
	ksmSleepPath = ksmBasePath + "sleep_millisecs"
	ksmPagesPath = ksmBasePath + "pages_to_scan"

	memInfoPath = "/proc/meminfo"
)

type ksmState struct {
	running bool
	sleep   uint64
	pages   int
}

type Handler struct {
	isLoopRunning bool
	clusterConfig *virtconfig.ClusterConfig
	nodeName      string
	client        k8scorev1.CoreV1Interface
	lock          sync.Mutex
	nodeStore     cache.Store
	// chan for being notified by KV config or node labels changes
	extChangesChan chan struct{}
	loopChan       chan struct{}
}

func NewHandler(nodeName string, client k8scorev1.CoreV1Interface, clusterConfig *virtconfig.ClusterConfig) *Handler {
	return &Handler{
		isLoopRunning:  false,
		clusterConfig:  clusterConfig,
		nodeName:       nodeName,
		client:         client,
		extChangesChan: make(chan struct{}),
	}
}

func (k *Handler) Run(stopCh chan struct{}) {
	// Create a ListWatch filtered to only the local node
	listWatch := cache.NewListWatchFromClient(
		k.client.RESTClient(),
		"nodes",
		metav1.NamespaceAll,
		fields.OneTermEqualSelector("metadata.name", k.nodeName),
	)

	nodeInformer := cache.NewSharedIndexInformer(listWatch, &k8sv1.Node{}, controller.ResyncPeriod(12*time.Hour), cache.Indexers{})
	go nodeInformer.Run(stopCh)
	cache.WaitForCacheSync(stopCh, nodeInformer.HasSynced)

	if _, err := nodeInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			UpdateFunc: k.handleNodeUpdate,
		}); err != nil {
		panic(err)
	}
	k.nodeStore = nodeInformer.GetStore()
	go k.Start()
	<-stopCh
}

func (k *Handler) handleNodeUpdate(oldObj, newObj interface{}) {
	oldNode := oldObj.(*k8sv1.Node)
	newNode := newObj.(*k8sv1.Node)
	if !equality.Semantic.DeepEqual(oldNode.Labels, newNode.Labels) {
		k.extChangesChan <- struct{}{}
		return
	}
	if !equality.Semantic.DeepEqual(oldNode.Annotations, newNode.Annotations) {
		k.extChangesChan <- struct{}{}
		return
	}
}

func (k *Handler) Start() {
	// Perform the initial node patch
	if ksmEligible := k.spin(); ksmEligible {
		k.isLoopRunning = true
		k.loopChan = make(chan struct{})
		go k.loop()
	}

	k.clusterConfig.SetConfigModifiedCallback(k.configCallback)
}

func (k *Handler) configCallback() {
	k.lock.Lock()
	defer k.lock.Unlock()
	ksmEligible, curState := k.isKSMEligible()
	if !ksmEligible {
		// stop the loop if running
		if k.isLoopRunning {
			k.isLoopRunning = false
			// Stop the ksm loop
			close(k.loopChan)
		}
		if curState {
			k.disableKSM()
		}

		k.patchKSM(ksmEligible, false)
		return
	}

	// loop already running, trigger another spin
	if k.isLoopRunning {
		k.extChangesChan <- struct{}{}
		return
	}

	k.isLoopRunning = true
	k.loopChan = make(chan struct{})
	go k.loop()
}

func (k *Handler) loop() {
	k.spin()
	ticker := time.NewTicker(3 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-k.extChangesChan:
			k.spin()
			ticker.Reset(3 * time.Minute)
		case <-ticker.C:
			k.spin()
		case <-k.loopChan:
			return
		}
	}
}

func (k *Handler) spin() bool {
	k.lock.Lock()
	defer k.lock.Unlock()
	// check that a concurrent config update has not disabled the ksm
	ksmEligible, curState := k.isKSMEligible()
	if !ksmEligible && curState {
		k.disableKSM()
	}
	var ksmEnabledByUs bool
	if ksmEligible {
		ksmEnabledByUs = k.handleNodePressure(curState)
	}

	k.patchKSM(ksmEligible, ksmEnabledByUs)
	return ksmEligible
}

func (k *Handler) shouldNodeHandleKSM() (shouldHandle, currentState bool, err error) {
	available, enabled := loadKSM()
	if !available {
		return false, false, nil
	}

	ksmConfig := k.clusterConfig.GetKSMConfiguration()
	if ksmConfig == nil {
		return false, enabled, nil
	}

	selector, err := metav1.LabelSelectorAsSelector(ksmConfig.NodeLabelSelector)
	if err != nil {
		return false, enabled, fmt.Errorf("an error occurred while converting the ksm selector: %s", err)
	}

	node, err := k.getNode()
	if err != nil {
		return false, enabled, err
	}

	if !selector.Matches(labels.Set(node.Labels)) {
		return false, enabled, nil
	}

	return true, enabled, nil
}

// isKSMEligible will return whether the node is eligible for the ksm handling:
// - ksm is enabled on the node
// - the node labels matches the node label selector ksm configuration
// Alongside, it will return the current ksm state and if the node labels need to be updated.
// Empty Selector will enable ksm for every node
func (k *Handler) isKSMEligible() (shouldHandle, currentState bool) {
	var err error
	if shouldHandle, currentState, err = k.shouldNodeHandleKSM(); err != nil {
		log.Log.Reason(err).Error(err.Error())
	}

	return
}

func (k *Handler) handleNodePressure(currentState bool) (ksmEnabledByUs bool) {
	node, err := k.getNode()
	if err != nil {
		return false
	}

	ksm, err := calculateNewRunSleepAndPages(node, currentState)
	if err != nil {
		log.DefaultLogger().Reason(err).Errorf("An error occurred while calculating the new KSM values")
		return false
	}

	if err = writeKsmValuesToFiles(ksm); err != nil {
		log.DefaultLogger().Reason(err).Errorf("An error occurred while writing the new KSM values")
		return false
	}

	return ksm.running
}

func (k *Handler) patchKSM(ksmEligible, ksmEnabledByUs bool) {
	// merge patch is being used here to handle the case in which the node has an empty/nil labels/annotations map,
	// which would cause a JSON patch to fail.
	patchPayload := struct {
		Metadata metav1.ObjectMeta `json:"metadata"`
	}{
		Metadata: metav1.ObjectMeta{
			Labels: map[string]string{
				v1.KSMEnabledLabel: fmt.Sprintf("%t", ksmEligible),
			},
			Annotations: map[string]string{
				v1.KSMHandlerManagedAnnotation: fmt.Sprintf("%t", ksmEnabledByUs),
			},
		},
	}
	patchBytes, err := json.Marshal(patchPayload)
	if err != nil {
		log.DefaultLogger().Reason(err).Error("Can't parse json patch")
	}

	_, err = k.client.Nodes().Patch(context.Background(), k.nodeName, types.MergePatchType, patchBytes, metav1.PatchOptions{})
	if err != nil {
		log.DefaultLogger().Reason(err).Errorf("Can't patch node %s", k.nodeName)
	}
}

func (k *Handler) disableKSM() {
	node, err := k.getNode()
	if err != nil {
		return
	}

	if value, found := node.GetAnnotations()[v1.KSMHandlerManagedAnnotation]; found && value == "true" {
		if err := os.WriteFile(ksmRunPath, []byte("0\n"), 0644); err != nil {
			log.DefaultLogger().Errorf("Unable to write ksm: %s", err.Error())
		}
	}
}

func (k *Handler) getNode() (*k8sv1.Node, error) {
	nodeObj, exists, err := k.nodeStore.GetByKey(k.nodeName)
	if err != nil {
		log.DefaultLogger().Errorf("Unable to get not: %s", err.Error())
		return nil, err
	}
	if !exists {
		log.DefaultLogger().Errorf("node %s does not exist", k.nodeName)
		return nil, fmt.Errorf("node %s does not exist", k.nodeName)
	}

	node, ok := nodeObj.(*k8sv1.Node)
	if !ok {
		return nil, fmt.Errorf("unknown object type found in node informer")
	}

	return node, nil
}

// Inspired from https://github.com/artyom/meminfo
func getTotalAndAvailableMem() (uint64, uint64, error) {
	var total, available uint64

	f, err := os.Open(memInfoPath)
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	found := 0
	for s.Scan() && found < 2 {
		switch {
		case bytes.HasPrefix(s.Bytes(), []byte(`MemTotal:`)):
			_, err = fmt.Sscanf(s.Text(), "MemTotal:%d", &total)
			found++
		case bytes.HasPrefix(s.Bytes(), []byte(`MemAvailable:`)):
			_, err = fmt.Sscanf(s.Text(), "MemAvailable:%d", &available)
			found++
		default:
			continue
		}
		if err != nil {
			return 0, 0, err
		}
	}
	if found != 2 {
		return 0, 0, fmt.Errorf("failed to find total and available memory")
	}

	return total, available, nil
}

func getKsmPages() (int, error) {
	pagesBytes, err := os.ReadFile(ksmPagesPath)
	if err != nil {
		return 0, err
	}

	pages, err := strconv.Atoi(strings.TrimSpace(string(pagesBytes)))
	if err != nil {
		return 0, err
	}

	return pages, nil
}

// Inspired from https://github.com/oVirt/mom/blob/master/doc/ksm.rules
func calculateNewRunSleepAndPages(node *k8sv1.Node, running bool) (ksmState, error) {
	pagesBoost := getIntParam(node, v1.KSMPagesBoostOverride, pagesBoostDefault, 0, math.MaxInt)
	pagesDecay := getIntParam(node, v1.KSMPagesDecayOverride, pagesDecayDefault, math.MinInt, 0)
	nPagesMin := getIntParam(node, v1.KSMPagesMinOverride, nPagesMinDefault, 0, math.MaxInt)
	nPagesMax := getIntParam(node, v1.KSMPagesMaxOverride, nPagesMaxDefault, nPagesMin, math.MaxInt)
	nPagesInit := getIntParam(node, v1.KSMPagesInitOverride, nPagesInitDefault, nPagesMin, nPagesMax)
	sleepMsBaseline := uint64(getIntParam(node, v1.KSMSleepMsBaselineOverride, sleepMsBaselineDefault, 1, math.MaxInt))
	freePercent := getFloatParam(node, v1.KSMFreePercentOverride, freePercentDefault, 0, 1)
	ksm := ksmState{running: running}
	total, available, err := getTotalAndAvailableMem()
	if err != nil {
		return ksm, err
	}
	ksm.pages, err = getKsmPages()
	if err != nil {
		return ksm, err
	}

	// Set sleep_millisecs to sleepMsBaseline on a 16GB system that's out of memory.
	// This basically scales sleep down the more memory there is to look at, capped at a minimum of 10ms.
	// This is copied from oVirt but might have to be adjuested in the future.
	ksm.sleep = sleepMsBaseline * (16 * 1024 * 1024) / (total - available)
	if ksm.sleep < sleepMsBaseline/10 {
		ksm.sleep = sleepMsBaseline / 10
	}

	if float32(available) > float32(total)*freePercent {
		// No memory pressure. Reduce or stop KSM activity
		if running {
			ksm.pages += pagesDecay
			if ksm.pages < nPagesMin {
				ksm.pages = nPagesMin
				ksm.running = false
			}
			return ksm, nil
		} else {
			return ksmState{false, 0, 0}, nil
		}
	} else {
		// We're under memory pressure. Increase or start KSM activity
		if running {
			ksm.pages += pagesBoost
			if ksm.pages > nPagesMax {
				ksm.pages = nPagesMax
			}
			return ksm, nil
		} else {
			ksm.running = true
			ksm.pages = nPagesInit
			return ksm, nil
		}
	}
}

func writeKsmValuesToFiles(ksm ksmState) error {
	run := "0"
	if ksm.running {
		run = "1"

		err := os.WriteFile(ksmSleepPath, []byte(strconv.FormatUint(ksm.sleep, 10)), 0644)
		if err != nil {
			return err
		}
		err = os.WriteFile(ksmPagesPath, []byte(strconv.Itoa(ksm.pages)), 0644)
		if err != nil {
			return err
		}
	}
	err := os.WriteFile(ksmRunPath, []byte(run), 0644)
	if err != nil {
		return err
	}

	return nil
}

func loadKSM() (bool, bool) {
	ksmValue, err := os.ReadFile(ksmRunPath)
	if err != nil {
		log.DefaultLogger().Warningf("An error occurred while reading the ksm module file; Maybe it is not available: %s", err)
		// Only enable for ksm-available nodes
		return false, false
	}

	return true, bytes.Equal(bytes.TrimSpace(ksmValue), []byte("1"))
}

func boundCheck[T int | float32](value, defaultValue, lowerBound, upperBound T, message string) T {
	if value < lowerBound || value > upperBound {
		if defaultValue > lowerBound && defaultValue < upperBound {
			log.DefaultLogger().Errorf("%s, using default (%v)", message, defaultValue)
			return defaultValue
		} else {
			log.DefaultLogger().Errorf("%s, using lowest possible value (%v)", message, lowerBound)
			return lowerBound
		}
	}

	return value
}

func getFloatParam(node *k8sv1.Node, param string, defaultValue, lowerBound, upperBound float32) float32 {
	override, ok := node.Annotations[param]
	if !ok {
		return defaultValue
	}
	value, err := strconv.ParseFloat(override, 32)
	if err != nil {
		log.DefaultLogger().Reason(err).Errorf("failed to parse %s override value, using default", param)
		return defaultValue
	}

	return boundCheck(float32(value), defaultValue, lowerBound, upperBound, fmt.Sprintf("%s override value out of bounds", param))
}

func getIntParam(node *k8sv1.Node, param string, defaultValue, lowerBound, upperBound int) int {
	override, ok := node.Annotations[param]
	if !ok {
		return defaultValue
	}
	value, err := strconv.Atoi(override)
	if err != nil {
		log.DefaultLogger().Reason(err).Errorf("failed to parse %s override value, using default", param)
		return defaultValue
	}

	return boundCheck(value, defaultValue, lowerBound, upperBound, fmt.Sprintf("%s override value out of bounds", param))
}
