package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"text/template"

	"github.com/spf13/pflag"
	v1 "kubevirt.io/api/core/v1"
	cloudinit "kubevirt.io/kubevirt/pkg/cloud-init"
)

const (
	awsRoleArn                        = "AWS_ROLE_ARN"
	awsRegion                         = "AWS_REGION"
	serviceAccountTargetDirAnnotation = "serviceaccounts.vm.kubevirt.io/targetDir"
)

const cloudInitTemplate = `
#cloud-config
write_files:
  - path: /etc/profile.d/aws_env.sh
    permissions: '0644'
    owner: root:root
    content: |
      export AWS_ROLE_ARN="{{ .arn }}"
      export AWS_WEB_IDENTITY_TOKEN_FILE="{{ .tokenFile }}"
      export AWS_REGION="{{ .region }}"

runcmd:
  - chmod +x /etc/profile.d/aws_env.sh
  - source /etc/profile.d/aws_env.sh
`

func preCloudInitIso(log *log.Logger, vmiJSON, cloudInitDataJSON []byte) (string, error) {
	log.Print("Hook's PreCloudInitIso callback method has been called")

	vmi := v1.VirtualMachineInstance{}
	err := json.Unmarshal(vmiJSON, &vmi)
	if err != nil {
		return "", fmt.Errorf("Failed to unmarshal given VMI spec: %s %s", err, string(vmiJSON))
	}

	// The hook will read the source (the path to the socket on the virt launcher) to set it as the AWS_WEB_IDENTITY_TOKEN_FILE
	if _, ok := vmi.Annotations[serviceAccountTargetDirAnnotation]; ok {
		return "", fmt.Errorf("target directory annotation not set, exiting")
	}

	cloudInitData := cloudinit.CloudInitData{}
	err = json.Unmarshal(cloudInitDataJSON, &cloudInitData)
	if err != nil {
		return "", fmt.Errorf("Failed to unmarshal given CloudInitData: %s %s", err, string(cloudInitDataJSON))
	}

	// Use a go template to parse the variables into a script to be used in cloud init user data
	var out *bytes.Buffer
	tmpl, err := template.New("aws").Parse(cloudInitTemplate)
	if err != nil {
		return "", fmt.Errorf("Failed to parse go template: %s", err)
	}

	// Read the variables from the container env, they should be added by the AWS pod identity webhook
	arn, region := getAwsEnvVars()
	awsMap := map[string]string{
		"arn":       arn,
		"tokenFile": vmi.Annotations[serviceAccountTargetDirAnnotation],
		"region":    region,
	}

	if err := tmpl.Execute(out, awsMap); err != nil {
		return "", fmt.Errorf("Failed to replace template variables")
	}

	// Handle the case where there already is a cloud init user data, by appending what the user had to the end of the script specified
	// TODO: this might not work if the keys can't be duplicated
	if cloudInitData.UserData != "" {
		withoutCloudConfig := strings.Replace(cloudInitData.UserData, "#cloud-config", "", 1)
		endConfig := out.String() + withoutCloudConfig
		cloudInitData.UserData = endConfig
	} else {
		cloudInitData.UserData = out.String()
	}

	response, err := json.Marshal(cloudInitData)
	if err != nil {
		return "", fmt.Errorf("Failed to marshal CloudInitData: %s %+v", err, cloudInitData)
	}

	return string(response), nil
}

func getAwsEnvVars() (string, string) {
	return mustGetEnv(awsRoleArn), mustGetEnv(awsRegion)
}

func mustGetEnv(key string) string {
	if key := os.Getenv(key); key != "" {
		return key
	}
	panic("Key " + key + " not set, exiting!")
}

func main() {
	var vmiJSON, cloudInitDataJSON string
	pflag.StringVar(&vmiJSON, "vmi", "", "Current VMI, in JSON format")
	pflag.StringVar(&cloudInitDataJSON, "cloud-init", "", "The CloudInitData, in JSON format")
	pflag.Parse()

	logger := log.New(os.Stderr, "eks-env-vars", log.Ldate)
	if vmiJSON == "" || cloudInitDataJSON == "" {
		logger.Printf("Bad input vmi=%d, cloud-init=%d", len(vmiJSON), len(cloudInitDataJSON))
		os.Exit(1)
	}

	cloudInitData, err := preCloudInitIso(logger, []byte(vmiJSON), []byte(cloudInitDataJSON))
	if err != nil {
		logger.Printf("preCloudInitIso failed: %s", err)
		panic(err)
	}
	fmt.Println(cloudInitData)
}
