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
	"io"
	"net/http"
	"regexp"
	"sync"
	"time"

	"github.com/emicklei/go-restful/v3"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
	clientutil "kubevirt.io/client-go/util"
)

const (
	// NOTE: We are limiting the maximum page size, as virt-api memory usage grows linearly with the page size,
	// as virt-api stores in memory profiling results. Based on experiments, profile data of one pod can grow to at least ~10Mb.
	maxClusterProfilerResultsPageSize     = 20
	defaultClusterProfilerResultsPageSize = 10
)

func (app *SubresourceAPIApp) getAllComponentPods() ([]k8sv1.Pod, error) {
	namespace, err := clientutil.GetNamespace()
	if err != nil {
		return nil, err
	}

	podList, err := app.virtCli.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{LabelSelector: "kubevirt.io"})
	if err != nil {
		return nil, err
	}

	pods := podList.Items[:0]
	for _, pod := range podList.Items {
		if podIsReadyComponent(&pod) {
			pods = append(pods, pod)
		}
	}

	return pods, nil
}

func (app *SubresourceAPIApp) unmarshalClusterProfilerRequest(request *restful.Request) (*v1.ClusterProfilerRequest, error) {
	cpRequest := &v1.ClusterProfilerRequest{}
	if request.Request.Body == nil {
		return nil, fmt.Errorf("empty request body")
	}
	return cpRequest, json.NewDecoder(request.Request.Body).Decode(cpRequest)
}

func (app *SubresourceAPIApp) getPodsNextPage(cpRequest *v1.ClusterProfilerRequest) (pods []k8sv1.Pod, cont string, err error) {
	var (
		listOptions = metav1.ListOptions{}
		namespace   string
		podList     *k8sv1.PodList
	)

	if selector, err := labels.Parse(cpRequest.LabelSelector); err != nil {
		return nil, "", err
	} else {
		listOptions.LabelSelector = selector.String()
	}

	listOptions.Continue = cpRequest.Continue
	listOptions.Limit = cpRequest.PageSize
	if listOptions.Limit <= 0 {
		listOptions.Limit = defaultClusterProfilerResultsPageSize
	} else if listOptions.Limit > maxClusterProfilerResultsPageSize {
		listOptions.Limit = maxClusterProfilerResultsPageSize
	}

	if namespace, err = clientutil.GetNamespace(); err != nil {
		return nil, "", err
	}

	if podList, err = app.virtCli.CoreV1().Pods(namespace).List(context.Background(), listOptions); err != nil {
		return nil, "", err
	}

	pods = podList.Items[:0]
	for _, pod := range podList.Items {
		if podIsReadyComponent(&pod) {
			pods = append(pods, pod)
		}
	}

	return pods, podList.Continue, nil
}

func podIsReadyComponent(pod *k8sv1.Pod) bool {
	re, _ := regexp.Compile("^(virt-api-|virt-operator-|virt-handler-|virt-controller-).*")
	isComponentPod := re.MatchString(pod.Name)
	// filter out any kubevirt related pod that doesn't have profiling capabilities
	if !isComponentPod {
		return false
	}

	if pod == nil {
		return false
	} else if pod.Status.Phase != k8sv1.PodRunning {
		return false
	} else if pod.DeletionTimestamp != nil {
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
				url := fmt.Sprintf("https://%s:%d/%s-profiler", ip, app.profilerComponentPort, command)
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

func (app *SubresourceAPIApp) StartClusterProfilerHandler(request *restful.Request, response *restful.Response) {
	if !app.clusterConfig.ClusterProfilerEnabled() {
		response.WriteErrorString(http.StatusForbidden, "Unable to start profiler. \"ClusterProfiler\" feature gate must be enabled")
		return
	}
	app.stopStartHandler("start", request, response)
}

func (app *SubresourceAPIApp) StopClusterProfilerHandler(request *restful.Request, response *restful.Response) {
	if !app.clusterConfig.ClusterProfilerEnabled() {
		response.WriteErrorString(http.StatusForbidden, "Unable to stop profiler. \"ClusterProfiler\" feature gate must be enabled")
		return
	}
	app.stopStartHandler("stop", request, response)
}

func (app *SubresourceAPIApp) DumpClusterProfilerHandler(request *restful.Request, response *restful.Response) {
	if !app.clusterConfig.ClusterProfilerEnabled() {
		response.WriteErrorString(http.StatusForbidden, "Unable to dump profiler results. \"ClusterProfiler\" feature gate must be enabled")
		return
	}

	cpRequest, err := app.unmarshalClusterProfilerRequest(request)
	if err != nil {
		response.WriteErrorString(http.StatusBadRequest, fmt.Sprintf("failed to parse cluster profiler request: %v", err))
		return
	}

	pods, cont, err := app.getPodsNextPage(cpRequest)
	if err != nil {
		response.WriteErrorString(http.StatusBadRequest, fmt.Sprintf("Internal error while looking up component pods for profiling: %v", err))
		return
	}

	if len(pods) == 0 {
		response.WriteHeaderAndJson(http.StatusNoContent, v1.ClusterProfilerResults{}, restful.MIME_JSON)
		return
	}

	const command = "dump"
	var (
		tr = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
		client = http.Client{
			Timeout:   5 * time.Second,
			Transport: tr,
		}
		results = v1.ClusterProfilerResults{
			ComponentResults: make(map[string]v1.ProfilerResult),
			Continue:         cont,
		}

		resultsLock = sync.Mutex{}
		wg          = sync.WaitGroup{}
		errorChan   = make(chan error, len(pods))
	)

	wg.Add(len(pods))
	defer close(errorChan)

	for _, pod := range pods {
		ip := pod.Status.PodIP
		name := pod.Name
		log.Log.Infof("Executing Cluster Profiler %s on Pod %s", command, name)
		go func(ip string, name string) {
			defer wg.Done()
			url := fmt.Sprintf("https://%s:%d/%s-profiler", ip, app.profilerComponentPort, command)
			req, _ := http.NewRequest("GET", url, nil)
			resp, err := client.Do(req)
			if err != nil {
				log.Log.Infof("Encountered error during ClusterProfiler %s on Pod %s: %v", command, name, err)
				errorChan <- err
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				errorChan <- fmt.Errorf("encountered [%d] status code while contacting url [%s] for pod [%s]", resp.StatusCode, url, name)
				return

			}

			data, err := io.ReadAll(resp.Body)
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

		}(ip, name)
	}

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
