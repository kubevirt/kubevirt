package apputil

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/golang/glog"
	nritypes "github.com/intel/network-resources-injector/pkg/types"

	"github.com/openshift/app-netutil/pkg/types"
)

const (
	HugepagesRequestPath = "hugepages_request"
	HugepagesLimitPath   = "hugepages_limit"
)

//
// API Functions
//
func GetHugepages() (*types.HugepagesResponse, error) {
	response := &types.HugepagesResponse{}

	// Try to retrieve this container's name from the environment variable
	glog.Infof("PROCESS ENV:")
	envResponse, err := getEnv()
	if err == nil {
		for envName, envVal := range envResponse.Envs {
			if envName == nritypes.EnvNameContainerName {
				response.MyContainerName = envVal
			}
		}
	} else {
		glog.Errorf("GetHugepages: Error calling getEnv: %v", err)
	}

	// Loop through all the files in the Downward API directory
	// and match the files with the "hugepages_" prefix.
	directory, err := os.Open(nritypes.DownwardAPIMountPath)
	if err != nil {
		glog.Infof("Error opening directory %s: %v", nritypes.DownwardAPIMountPath, err)
		return nil, err
	}
	defer directory.Close()

	fileList, err := directory.Readdirnames(0)
	if err != nil {
		glog.Infof("Error reading directory names in %s: %v", nritypes.DownwardAPIMountPath, err)
		return nil, err
	}

	found := false
	for _, fileName := range fileList {
		if match := strings.HasPrefix(fileName, nritypes.Hugepages1GRequestPath); match {
			// Request 1G Match
			hugepagesData, hugepageVal, err := managerContainerName(response, fileName, nritypes.Hugepages1GRequestPath)
			if err == nil {
				hugepagesData.Request1G = hugepageVal
				found = true
			}
		} else if match := strings.HasPrefix(fileName, nritypes.Hugepages2MRequestPath); match {
			// Request 2M Match
			hugepagesData, hugepageVal, err := managerContainerName(response, fileName, nritypes.Hugepages2MRequestPath)
			if err == nil {
				hugepagesData.Request2M = hugepageVal
				found = true
			}
		} else if match := strings.HasPrefix(fileName, nritypes.Hugepages1GLimitPath); match {
			// Limit 1G Match
			hugepagesData, hugepageVal, err := managerContainerName(response, fileName, nritypes.Hugepages1GLimitPath)
			if err == nil {
				hugepagesData.Limit1G = hugepageVal
				found = true
			}
		} else if match := strings.HasPrefix(fileName, nritypes.Hugepages2MLimitPath); match {
			// Limit 2M Match
			hugepagesData, hugepageVal, err := managerContainerName(response, fileName, nritypes.Hugepages2MLimitPath)
			if err == nil {
				hugepagesData.Limit2M = hugepageVal
				found = true
			}
		} else if match := strings.HasPrefix(fileName, HugepagesRequestPath); match {
			// Request Match
			hugepagesData, hugepageVal, err := managerContainerName(response, fileName, HugepagesRequestPath)
			if err == nil {
				hugepagesData.Request = hugepageVal
				found = true
			}
		} else if match := strings.HasPrefix(fileName, HugepagesLimitPath); match {
			// Limit Match
			hugepagesData, hugepageVal, err := managerContainerName(response, fileName, HugepagesLimitPath)
			if err == nil {
				hugepagesData.Limit = hugepageVal
				found = true
			}
		} else {
			glog.Infof("  \"%s\" does NOT match any hugepage file names", fileName)
		}
	}

	if !found {
		return nil, fmt.Errorf("hugepage data not found")
	}
	return response, nil
}

func managerContainerName(response *types.HugepagesResponse,
	fileName string,
	curMatchStr string) (*types.HugepagesData, int64, error) {

	// Find Container Name, use what has been matched so far (don't add '_')
	containerName := strings.TrimPrefix(fileName, curMatchStr)
	if containerName != "" {
		// Trim leading '_'
		containerName = strings.TrimPrefix(containerName, "_")
	}
	glog.Infof("Hugepage file: Using containerName \"%s\"", containerName)

	// Determine if ContainerName already has an entry in response
	var hugepagesData *types.HugepagesData
	for _, tmpResponseData := range response.Hugepages {
		if tmpResponseData.ContainerName == containerName {
			hugepagesData = tmpResponseData
			break
		}
	}
	// Create new entry if not found above
	if hugepagesData == nil {
		hugepagesData = &types.HugepagesData{
			ContainerName: containerName,
		}
		response.Hugepages = append(response.Hugepages, hugepagesData)
	}

	// Retrieve value from file
	var hugepagesVal int64
	path := filepath.Join(nritypes.DownwardAPIMountPath, fileName)
	glog.Infof("GetHugepages: Open %s", path)
	hugepagesStr, err := ioutil.ReadFile(path)
	if err != nil {
		glog.Infof("Error getting %s info: %v", path, err)
	} else {
		hugepagesVal, err = strconv.ParseInt(string(bytes.TrimSpace(hugepagesStr)), 10, 64)
		if err != nil {
			glog.Infof("Error converting limit \"%s\": %v", hugepagesStr, err)
		}
	}

	return hugepagesData, hugepagesVal, err
}
