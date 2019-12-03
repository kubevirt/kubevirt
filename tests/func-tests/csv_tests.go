package tests

import (
	"github.com/onsi/ginkgo"

	"encoding/json"
	"fmt"
	"github.com/ghodss/yaml"
	//"gopkg.in/yaml.v2"
	"io/ioutil"
        "os"
	"os/exec"
)

// test parameters.
type TestCfg struct {
	KubectlCmd                    string
	ClusterServiceVersionFileName string
}

func newTestCfg() (*TestCfg,bool) {

        kubectlCommand := os.Getenv("TEST_KUBECTL_CMD")
        clusterServiceVersionFileName := os.Getenv("TEST_CSV_FILE")

        runTests := kubectlCommand != "" && clusterServiceVersionFileName != ""

        return &TestCfg{KubectlCmd: kubectlCommand, ClusterServiceVersionFileName: clusterServiceVersionFileName}, runTests
}


// yaml parsing structs (for parsing of cluster service version yaml file)
type hcoType struct {
	Spec hcoSpec `yaml:"spec"`
}

type hcoSpec struct {
	Install hcoInstall `yaml:"install"`
}

type hcoInstall struct {
	Spec hcoInstallSpec `yaml:"spec"`
}

type hcoInstallSpec struct {
	Deployments []hcoDeployments `yaml:"deployments"`
}

type hcoDeployments struct {
	Name string            `yaml:"name"`
	Spec hcoDeploymentSpec `yaml:"spec"`
}

type hcoDeploymentSpec struct {
	Template hcoDeploymentSpecTemplate `yaml:"template"`
}

type hcoDeploymentSpecTemplate struct {
	Spec hcoDeploymentSpecTemplateSpec `yaml:"spec"`
}

type hcoDeploymentSpecTemplateSpec struct {
	Containers []hcoDeploymentContainers `yaml:"containers"`
}

type hcoDeploymentContainers struct {
	Image string `yaml:"image"`
}

type specDeployment struct {
	Name  string
	Image string
}

func parseClusterServiceVersionFile(fname string) ([]specDeployment, error) {
	bdata, err := ioutil.ReadFile(fname)
	if err != nil {
		fmt.Fprint(ginkgo.GinkgoWriter, err)
		return nil, err

	}
	var data hcoType

	err = yaml.Unmarshal([]byte(bdata), &data)
	if err != nil {
		fmt.Fprint(ginkgo.GinkgoWriter, err)
		return nil, err
	}

	numResult := len(data.Spec.Install.Spec.Deployments)
        if numResult  == 0 {
                err := fmt.Errorf( "no deployments in spec")
                return nil, err
        }
	retVals := make([]specDeployment, numResult)

	for pos, spec := range data.Spec.Install.Spec.Deployments {
		fmt.Fprintln(ginkgo.GinkgoWriter, "Spec: ", spec)

		retVals[pos] = specDeployment{spec.Name, spec.Spec.Template.Spec.Containers[0].Image}
		fmt.Fprintln(ginkgo.GinkgoWriter, "Spec: ",retVals[pos])
	}

	return retVals, nil
}

// parsing of deployment json
type depRoot struct {
	Items []depItem `json:"items"`
}

type depItem struct {
	Metadata depMetadata `json:"metadata"`
	Spec     depSpec     `json:"spec"`
	Status   depStatus   `json:"status"`
}

type depStatus struct {
	Conditions []depStatusCondition `json:"conditions"`
}

type depStatusCondition struct {
	Status string `json:"status"`
	Type   string `json:"type"`
}

type depMetadata struct {
	Name string `json:"name"`
}

type depSpec struct {
	Template depTemplate `json:"template"`
}

type depTemplate struct {
	Spec depTemplateSpec `json:"spec"`
}

type depTemplateSpec struct {
	Containers []depContainers `json:"containers"`
}

type depContainers struct {
	Image string `json:"Image"`
}

// parsed entity from deployment json
type deploymentData struct {
	Image     string
	Available bool
}

func parseDeployments(bdata string) (map[string]deploymentData, error) {
	var data depRoot

	err := json.Unmarshal([]byte(bdata), &data)
	if err != nil {
		fmt.Fprint(ginkgo.GinkgoWriter, err)
		return nil, err
	}

	ret := make(map[string]deploymentData)

	for _, entry := range data.Items {
		fmt.Fprintln(ginkgo.GinkgoWriter, "Deployment: ", entry)

		name := entry.Metadata.Name
		imageName := entry.Spec.Template.Spec.Containers[0].Image

		available := false
		for _, cond := range entry.Status.Conditions {
			if cond.Type == "Available" && cond.Status == "True" {
				available = true
			}
		}

		depData := deploymentData{imageName, available}
		fmt.Fprintln(ginkgo.GinkgoWriter, depData)
		ret[name] = depData
	}

	return ret, nil

}

func matchImages(entryImage string, deploymentImage string) bool {
	return entryImage == deploymentImage
}

func matchClusterServiceDataToDeployment(specDep []specDeployment, depData map[string]deploymentData) bool {
	status := true
	for _, entry := range specDep {
		if deploymentEntry, ok := depData[entry.Name]; !ok {
			fmt.Fprintf(ginkgo.GinkgoWriter, "no deployment exists for Cluster service entry %s", entry.Name)
			status = false
		} else {
			if !deploymentEntry.Available {
				fmt.Fprintf(ginkgo.GinkgoWriter, "deployment %s exists, but is is not available", entry.Name)
				status = false
			}
			if !matchImages(entry.Image, deploymentEntry.Image) {
				fmt.Fprintf(ginkgo.GinkgoWriter, "images in cluster service entry %s does not match image in deployment %s", entry.Image, deploymentEntry.Image)
				status = false
			}
		}
	}
	return status
}


func getDeploymentJson(kubectlCmd string) (string, error) {
	cmd := exec.Command(kubectlCmd, "get", "deployments", "-o", "json")

    out, err := cmd.CombinedOutput()
    if err != nil {
                errf := fmt.Errorf("get deployments failed with %s", err)
		return "", errf
    }
    return string(out), nil

}

var _ = ginkgo.Describe("ClusterServiceVersion", func() {

	ginkgo.Context("csv testing", func() {
		ginkgo.It("For each csv entry should have active deployment with same image as in csv file", func() {
                    testDeployments()
                })
        })
})

func testDeployments() {

        tstCfg, runTests := newTestCfg()
        if  !runTests {
                fmt.Fprintln(ginkgo.GinkgoWriter, "*** skipping ClusterService test  ***")
                return
        }

	cluster, cerr := parseClusterServiceVersionFile(tstCfg.ClusterServiceVersionFileName)
	if cerr != nil {
		msg := fmt.Sprint("Parsing of cluster service version failed", cerr)
		ginkgo.Fail(msg)
	}

	depjson, serr := getDeploymentJson(tstCfg.KubectlCmd)
	if serr != nil {
		msg := fmt.Sprint("failed to get deployment", serr)
	        ginkgo.Fail(msg)
        }

	dep, derr := parseDeployments(depjson) //"tools/test-hco-utils/deploy.json")
	if derr != nil {
		msg := fmt.Sprint("failed to parse the deployment data", derr)
	        ginkgo.Fail(msg)
        }

	if !matchClusterServiceDataToDeployment(cluster, dep) {
	        msg := fmt.Sprint("deployment does not match cluster service data")
                ginkgo.Fail(msg)
        }
	fmt.Fprintln(ginkgo.GinkgoWriter, "*** all deployments are up and corespond to cluster service version ***")
}
