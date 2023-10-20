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
 * Copyright 2023 Red Hat, Inc.
 *
 */

package heartbeat

import (
	"bufio"
	"bytes"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	kubevirtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

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

	pagesBoost              = pagesBoostDefault
	pagesDecay              = pagesDecayDefault
	nPagesMin               = nPagesMinDefault
	nPagesMax               = nPagesMaxDefault
	nPagesInit              = nPagesInitDefault
	sleepMsBaseline uint64  = sleepMsBaselineDefault // 10ms in oVirt seemed really low
	freePercent     float32 = freePercentDefault
)

type ksmState struct {
	running bool
	sleep   uint64
	pages   int
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
func calculateNewRunSleepAndPages(running bool) (ksmState, error) {
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

func getFloatParam(node *v1.Node, param string, defaultValue, lowerBound, upperBound float32) float32 {
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

func getIntParam(node *v1.Node, param string, defaultValue, lowerBound, upperBound int) int {
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

// handleKSM will update the ksm of the node (if available) based on the kv configuration and
// will set the outcome value to the n.KSM struct
// If the node labels match the selector terms, the ksm will be enabled.
// Empty Selector will enable ksm for every node
func handleKSM(node *v1.Node, clusterConfig *virtconfig.ClusterConfig) (bool, bool) {
	available, running := loadKSM()
	if !available {
		return running, false
	}

	ksmConfig := clusterConfig.GetKSMConfiguration()
	if ksmConfig == nil {
		if disableKSM(node, running) {
			return false, false
		} else {
			return running, false
		}
	}

	selector, err := metav1.LabelSelectorAsSelector(ksmConfig.NodeLabelSelector)
	if err != nil {
		log.DefaultLogger().Errorf("An error occurred while converting the ksm selector: %s", err)
		return running, false
	}

	if !selector.Matches(labels.Set(node.ObjectMeta.Labels)) {
		if disableKSM(node, running) {
			return false, false
		} else {
			return running, false
		}
	}

	pagesBoost = getIntParam(node, kubevirtv1.KSMPagesBoostOverride, pagesBoostDefault, 0, math.MaxInt)
	pagesDecay = getIntParam(node, kubevirtv1.KSMPagesDecayOverride, pagesDecayDefault, math.MinInt, 0)
	nPagesMin = getIntParam(node, kubevirtv1.KSMPagesMinOverride, nPagesMinDefault, 0, math.MaxInt)
	nPagesMax = getIntParam(node, kubevirtv1.KSMPagesMaxOverride, nPagesMaxDefault, nPagesMin, math.MaxInt)
	nPagesInit = getIntParam(node, kubevirtv1.KSMPagesInitOverride, nPagesInitDefault, nPagesMin, nPagesMax)
	sleepMsBaseline = uint64(getIntParam(node, kubevirtv1.KSMSleepMsBaselineOverride, sleepMsBaselineDefault, 1, math.MaxInt))
	freePercent = getFloatParam(node, kubevirtv1.KSMFreePercentOverride, freePercentDefault, 0, 1)

	ksm, err := calculateNewRunSleepAndPages(running)
	if err != nil {
		log.DefaultLogger().Reason(err).Errorf("An error occurred while calculating the new KSM values")
		return running, false
	}

	err = writeKsmValuesToFiles(ksm)
	if err != nil {
		log.DefaultLogger().Reason(err).Errorf("An error occurred while writing the new KSM values")
		return running, false
	}

	return ksm.running, ksm.running
}

func disableKSM(node *v1.Node, enabled bool) bool {
	if enabled {
		if value, found := node.GetAnnotations()[kubevirtv1.KSMHandlerManagedAnnotation]; found && value == "true" {
			err := os.WriteFile(ksmRunPath, []byte("0\n"), 0644)
			if err != nil {
				log.DefaultLogger().Errorf("Unable to write ksm: %s", err.Error())
				return false
			}
			return true
		}
	}

	return false
}
