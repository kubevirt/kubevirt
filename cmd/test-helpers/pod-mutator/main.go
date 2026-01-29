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

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/spf13/pflag"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kubevirt.io/client-go/log"
)

const (
	injectedVolumeName = "test-injected-volume"
	injectedMountPath  = "/opt/test-injected"
)

var (
	volumeType    string
	configMapName string
)

func main() {
	log.InitializeLogging("test-pod-mutator")

	var port int
	var certFile, keyFile string
	pflag.IntVar(&port, "port", 8443, "port to listen on")
	pflag.StringVar(&certFile, "cert-file", "/etc/webhook/certs/tls.crt", "TLS certificate file")
	pflag.StringVar(&keyFile, "key-file", "/etc/webhook/certs/tls.key", "TLS private key file")
	pflag.StringVar(&volumeType, "volume-type", "emptydir", "type of volume to inject: emptydir or configmap")
	pflag.StringVar(&configMapName, "configmap-name", "", "name of ConfigMap to inject (required when volume-type=configmap)")
	pflag.Parse()

	if volumeType == "configmap" && configMapName == "" {
		log.Log.Error("--configmap-name is required when --volume-type=configmap")
		os.Exit(1)
	}

	http.HandleFunc("/mutate", handleMutate)
	http.HandleFunc("/health", handleHealth)

	addr := fmt.Sprintf(":%d", port)
	log.Log.Infof("Starting webhook server on %s with TLS", addr)

	if err := http.ListenAndServeTLS(addr, certFile, keyFile, nil); err != nil {
		log.Log.Reason(err).Errorf("Failed to start webhook server")
		os.Exit(1)
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func handleMutate(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Log.Reason(err).Error("Failed to read request body")
		http.Error(w, "Failed to read request", http.StatusBadRequest)
		return
	}

	var admissionReview admissionv1.AdmissionReview
	if err := json.Unmarshal(body, &admissionReview); err != nil {
		log.Log.Reason(err).Error("Failed to unmarshal admission review")
		http.Error(w, "Failed to parse request", http.StatusBadRequest)
		return
	}

	if admissionReview.Request == nil {
		log.Log.Error("Admission review has no request")
		http.Error(w, "Invalid admission review", http.StatusBadRequest)
		return
	}

	response := mutate(admissionReview.Request)
	admissionReview.Response = response
	admissionReview.Response.UID = admissionReview.Request.UID

	responseBytes, err := json.Marshal(admissionReview)
	if err != nil {
		log.Log.Reason(err).Error("Failed to marshal response")
		http.Error(w, "Failed to create response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(responseBytes)
}

func mutate(req *admissionv1.AdmissionRequest) *admissionv1.AdmissionResponse {
	if req.Kind.Kind != "Pod" {
		return &admissionv1.AdmissionResponse{
			Allowed: true,
		}
	}

	var pod corev1.Pod
	if err := json.Unmarshal(req.Object.Raw, &pod); err != nil {
		log.Log.Reason(err).Error("Failed to unmarshal pod")
		return &admissionv1.AdmissionResponse{
			Allowed: false,
			Result: &metav1.Status{
				Message: fmt.Sprintf("Failed to unmarshal pod: %v", err),
			},
		}
	}

	// Only mutate virt-launcher pods
	if pod.Labels == nil || pod.Labels["kubevirt.io"] != "virt-launcher" {
		return &admissionv1.AdmissionResponse{
			Allowed: true,
		}
	}

	log.Log.Infof("Mutating virt-launcher pod %s/%s", pod.Namespace, pod.Name)

	patches := []map[string]interface{}{}

	// Check if volume already exists
	volumeExists := false
	for _, vol := range pod.Spec.Volumes {
		if vol.Name == injectedVolumeName {
			volumeExists = true
			break
		}
	}

	if !volumeExists {
		var volumeSource map[string]interface{}
		switch volumeType {
		case "emptydir":
			volumeSource = map[string]interface{}{
				"emptyDir": map[string]interface{}{},
			}
		case "configmap":
			volumeSource = map[string]interface{}{
				"configMap": map[string]interface{}{
					"name": configMapName,
				},
			}
		default:
			log.Log.Errorf("Unknown volume type: %s", volumeType)
			return &admissionv1.AdmissionResponse{
				Allowed: false,
				Result: &metav1.Status{
					Message: fmt.Sprintf("Unknown volume type: %s", volumeType),
				},
			}
		}

		volumePatch := map[string]interface{}{
			"op":    "add",
			"path":  "/spec/volumes/-",
			"value": volumeSource,
		}
		volumePatch["value"].(map[string]interface{})["name"] = injectedVolumeName
		patches = append(patches, volumePatch)
	}

	// Add volumeMount to compute container
	for i, container := range pod.Spec.Containers {
		if container.Name == "compute" {
			mountExists := false
			for _, mount := range container.VolumeMounts {
				if mount.Name == injectedVolumeName {
					mountExists = true
					break
				}
			}

			if !mountExists {
				mountPatch := map[string]interface{}{
					"op":   "add",
					"path": fmt.Sprintf("/spec/containers/%d/volumeMounts/-", i),
					"value": map[string]interface{}{
						"name":      injectedVolumeName,
						"mountPath": injectedMountPath,
					},
				}
				patches = append(patches, mountPatch)
			}
			break
		}
	}

	if len(patches) == 0 {
		return &admissionv1.AdmissionResponse{
			Allowed: true,
		}
	}

	patchBytes, err := json.Marshal(patches)
	if err != nil {
		log.Log.Reason(err).Error("Failed to marshal patches")
		return &admissionv1.AdmissionResponse{
			Allowed: false,
			Result: &metav1.Status{
				Message: fmt.Sprintf("Failed to create patches: %v", err),
			},
		}
	}

	patchType := admissionv1.PatchTypeJSONPatch
	return &admissionv1.AdmissionResponse{
		Allowed:   true,
		Patch:     patchBytes,
		PatchType: &patchType,
	}
}
