/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package profiler

import (
	"fmt"
	"net/http"

	v1 "kubevirt.io/api/core/v1"

	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"

	restful "github.com/emicklei/go-restful/v3"
)

type ProfileManager struct {
	clusterConfig   *virtconfig.ClusterConfig
	processProfiler *pprofData
}

type ProfilerResults struct {
	ProcessProfilerResults map[string][]byte `json:"processProfilerResults,omitempty"`
}

func NewProfileManager(clusterConfig *virtconfig.ClusterConfig) *ProfileManager {

	return &ProfileManager{
		clusterConfig:   clusterConfig,
		processProfiler: &pprofData{},
	}
}

func (m *ProfileManager) startProfiler() error {

	// make sure all profilers are stopped before
	// we attempt to start again
	err := m.stopProfiler(true)
	if err != nil {
		return err
	}

	err = m.processProfiler.startProcessProfiler()
	if err != nil {
		m.stopProfiler(true)
		return err
	}

	return nil
}

func (m *ProfileManager) stopProfiler(clearResults bool) error {
	m.processProfiler.stopProcessProfiler(clearResults)

	return nil
}

func (m *ProfileManager) dumpProfilerResult() (*v1.ProfilerResult, error) {
	pprofResults, err := m.processProfiler.dumpProcessProfilerResults()
	if err != nil {
		return nil, err
	}
	profilerResults := &v1.ProfilerResult{
		PprofData: pprofResults,
	}

	return profilerResults, nil

}

func (m *ProfileManager) HandleStartProfiler(_ *restful.Request, response *restful.Response) {

	if !m.clusterConfig.ClusterProfilerEnabled() {
		response.WriteErrorString(http.StatusForbidden, "Unable to start profiler. \"ClusterProfiler\" feature gate must be enabled")
		return
	}

	err := m.startProfiler()
	if err != nil {
		response.WriteErrorString(http.StatusInternalServerError, fmt.Sprintf("could not start internal profiling: %v", err))
		return
	}
	response.WriteHeader(http.StatusOK)
}

func (m *ProfileManager) HandleStopProfiler(_ *restful.Request, response *restful.Response) {
	if !m.clusterConfig.ClusterProfilerEnabled() {
		response.WriteErrorString(http.StatusForbidden, "Unable to stop profiler. \"ClusterProfiler\" feature gate must be enabled")
		return
	}
	err := m.stopProfiler(false)
	if err != nil {
		response.WriteErrorString(http.StatusInternalServerError, fmt.Sprintf("could not stop internal profiling: %v", err))
		return
	}

	response.WriteHeader(http.StatusOK)
}

func (m *ProfileManager) HandleDumpProfiler(_ *restful.Request, response *restful.Response) {

	if !m.clusterConfig.ClusterProfilerEnabled() {
		response.WriteErrorString(http.StatusForbidden, "Unable to retrieve profiler data. \"ClusterProfiler\" feature gate must be enabled")
		return
	}
	res, err := m.dumpProfilerResult()
	if err != nil {
		response.WriteErrorString(http.StatusInternalServerError, fmt.Sprintf("could not dump internal profiling: %v", err))
		return
	}

	response.WriteHeaderAndJson(http.StatusOK, res, restful.MIME_JSON)
}
