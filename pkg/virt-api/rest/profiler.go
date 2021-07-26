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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package rest

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"

	restful "github.com/emicklei/go-restful"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/log"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	clientutil "kubevirt.io/client-go/util"
)

func (app *SubresourceAPIApp) getAllComponentPods() ([]*k8sv1.Pod, error) {
	namespace, err := clientutil.GetNamespace()
	if err != nil {
		return nil, err
	}

	podList, err := app.virtCli.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var pods []*k8sv1.Pod

	for _, pod := range podList.Items {
		if podIsReadyComponent(&pod) {
			pods = append(pods, pod.DeepCopy())
		}
	}

	return pods, nil
}

func podIsReadyComponent(pod *k8sv1.Pod) bool {
	componentPrefixes := []string{"virt-controller", "virt-handler", "virt-api"}

	found := false
	// filter out any kubevirt related pod that doesn't have profiling capabilities
	for _, prefix := range componentPrefixes {
		if strings.Contains(pod.Name, prefix) {
			found = true
			break
		}
	}
	if !found {
		return false
	}

	if pod == nil {
		return false
	} else if pod.Status.Phase != k8sv1.PodRunning {
		return false
	} else {
		for _, cond := range pod.Status.Conditions {
			if cond.Type == k8sv1.PodReady && cond.Status == k8sv1.ConditionTrue {
				return true
			}
		}
	}

	return false
}

func (app *SubresourceAPIApp) stopStartHandler(command string, request *restful.Request, response *restful.Response) {
	pods, err := app.getAllComponentPods()
	if err != nil {
		log.Log.Infof("Encountered error while retrieving component pods for cluster profiler: %v", err)
		response.WriteErrorString(http.StatusInternalServerError, fmt.Sprintf("Internal error while looking up component pods for profiling: %v", err))
		return
	}

	if len(pods) == 0 {
		response.WriteErrorString(http.StatusInternalServerError, "Internal error, no component pods found")
		return
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	client := http.Client{
		Timeout:   time.Duration(5 * time.Second),
		Transport: tr,
	}

	wg := sync.WaitGroup{}
	wg.Add(len(pods))

	errorChan := make(chan error, len(pods))
	defer close(errorChan)

	go func() {
		for _, pod := range pods {
			ip := pod.Status.PodIP
			name := pod.Name
			log.Log.Infof("Executing Cluster Profiler %s on Pod %s", command, name)
			go func(ip string, name string) {
				defer wg.Done()
				url := fmt.Sprintf("https://%s:8443/%s-profiler", ip, command)
				req, _ := http.NewRequest("GET", url, nil)
				resp, err := client.Do(req)
				if err != nil {
					log.Log.Infof("Encountered error during ClusterProfiler %s on Pod %s: %v", command, name, err)
					errorChan <- err
					return
				}

				defer resp.Body.Close()
				if resp.StatusCode != http.StatusOK {

					errorChan <- fmt.Errorf("Encountered [%d] status code while contacting url [%s] for pod [%s]", resp.StatusCode, url, name)
					return
				}
			}(ip, name)
		}
	}()

	wg.Wait()

	select {
	case err := <-errorChan:
		response.WriteErrorString(http.StatusInternalServerError, fmt.Sprintf("Internal error encountered: %v", err))
		return
	default:
		// no error
	}

	response.WriteHeader(http.StatusOK)
}

func (app *SubresourceAPIApp) StartClusterProfilerHandler() restful.RouteFunction {
	return func(request *restful.Request, response *restful.Response) {
		app.stopStartHandler("start", request, response)
	}
}

func (app *SubresourceAPIApp) StopClusterProfilerHandler() restful.RouteFunction {
	return func(request *restful.Request, response *restful.Response) {
		app.stopStartHandler("stop", request, response)
	}
}
func (app *SubresourceAPIApp) DumpClusterProfilerHandler() restful.RouteFunction {
	return func(request *restful.Request, response *restful.Response) {
		pods, err := app.getAllComponentPods()
		if err != nil {
			response.WriteErrorString(http.StatusInternalServerError, fmt.Sprintf("Internal error while looking up component pods for profiling: %v", err))
			return
		}

		if len(pods) == 0 {
			response.WriteErrorString(http.StatusInternalServerError, "Internal error, no component pods found")
			return
		}

		tr := &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}

		client := http.Client{
			Timeout:   time.Duration(5 * time.Second),
			Transport: tr,
		}

		command := "dump"

		wg := sync.WaitGroup{}
		wg.Add(len(pods))

		errorChan := make(chan error, len(pods))
		defer close(errorChan)

		results := v1.ClusterProfilerResults{
			ComponentResults: make(map[string]v1.ProfilerResult),
		}
		resultsLock := sync.Mutex{}

		go func() {
			for _, pod := range pods {
				ip := pod.Status.PodIP
				name := pod.Name
				log.Log.Infof("Executing Cluster Profiler %s on Pod %s", command, name)
				go func(ip string, name string) {
					defer wg.Done()
					url := fmt.Sprintf("https://%s:8443/%s-profiler", ip, command)
					req, _ := http.NewRequest("GET", url, nil)
					resp, err := client.Do(req)
					if err != nil {
						log.Log.Infof("Encountered error during ClusterProfiler %s on Pod %s: %v", command, name, err)
						errorChan <- err
						return
					}
					defer resp.Body.Close()

					if resp.StatusCode != http.StatusOK {

						errorChan <- fmt.Errorf("Encountered [%d] status code while contacting url [%s] for pod [%s]", resp.StatusCode, url, name)
						return

					} else {
						data, err := ioutil.ReadAll(resp.Body)
						if err != nil {
							errorChan <- err
							return
						}

						componentResult := v1.ProfilerResult{}
						err = json.Unmarshal(data, &componentResult)
						if err != nil {
							errorChan <- fmt.Errorf("Failure to unmarshal json body: %s\nerr: %v", string(data), err)
							return
						}

						resultsLock.Lock()
						defer resultsLock.Unlock()
						results.ComponentResults[name] = componentResult

					}
				}(ip, name)
			}
		}()

		wg.Wait()
		select {
		case err := <-errorChan:
			response.WriteErrorString(http.StatusInternalServerError, fmt.Sprintf("Internal error encountered: %v", err))
			return
		default:
			//no error
		}

		response.WriteAsJson(results)
	}
}
