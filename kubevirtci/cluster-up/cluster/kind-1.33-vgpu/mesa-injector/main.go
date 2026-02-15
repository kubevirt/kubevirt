// Copyright 2024 The KubeVirt Authors.
// Licensed under the Apache License, Version 2.0.

// mesa-injector is a mutating admission webhook that injects OpenGL/mesa
// libraries into virt-launcher pods, enabling vGPU display support.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	certFile = "/etc/webhook/certs/tls.crt"
	keyFile  = "/etc/webhook/certs/tls.key"
	port     = ":8443"
)

// JSON Patch operations to inject mesa libraries
type patchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

func main() {
	log.Println("Starting mesa-injector webhook server...")

	http.HandleFunc("/mutate", handleMutate)
	http.HandleFunc("/health", handleHealth)

	log.Printf("Listening on %s", port)
	if err := http.ListenAndServeTLS(port, certFile, keyFile, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func handleMutate(w http.ResponseWriter, r *http.Request) {
	log.Println("Received mutation request")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading body: %v", err)
		http.Error(w, "could not read request body", http.StatusBadRequest)
		return
	}

	var admissionReview admissionv1.AdmissionReview
	if err := json.Unmarshal(body, &admissionReview); err != nil {
		log.Printf("Error unmarshaling request: %v", err)
		http.Error(w, "could not unmarshal request", http.StatusBadRequest)
		return
	}

	response := mutate(admissionReview.Request)
	admissionReview.Response = response
	admissionReview.Response.UID = admissionReview.Request.UID

	respBytes, err := json.Marshal(admissionReview)
	if err != nil {
		log.Printf("Error marshaling response: %v", err)
		http.Error(w, "could not marshal response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(respBytes)
}

func mutate(request *admissionv1.AdmissionRequest) *admissionv1.AdmissionResponse {
	// Only handle pod creation
	if request.Kind.Kind != "Pod" {
		return &admissionv1.AdmissionResponse{Allowed: true}
	}

	var pod corev1.Pod
	if err := json.Unmarshal(request.Object.Raw, &pod); err != nil {
		return &admissionv1.AdmissionResponse{
			Allowed: false,
			Result: &metav1.Status{
				Message: fmt.Sprintf("could not unmarshal pod: %v", err),
			},
		}
	}

	// Check if this is a virt-launcher pod
	if pod.Labels["kubevirt.io"] != "virt-launcher" {
		log.Printf("Skipping non-virt-launcher pod: %s", pod.Name)
		return &admissionv1.AdmissionResponse{Allowed: true}
	}

	log.Printf("Injecting mesa libraries into virt-launcher pod: %s", pod.Name)

	// Build the JSON patch
	patches := buildMesaPatches(&pod)

	if len(patches) == 0 {
		return &admissionv1.AdmissionResponse{Allowed: true}
	}

	patchBytes, err := json.Marshal(patches)
	if err != nil {
		return &admissionv1.AdmissionResponse{
			Allowed: false,
			Result: &metav1.Status{
				Message: fmt.Sprintf("could not marshal patch: %v", err),
			},
		}
	}

	log.Printf("Applying patch: %s", string(patchBytes))

	patchType := admissionv1.PatchTypeJSONPatch
	return &admissionv1.AdmissionResponse{
		Allowed:   true,
		Patch:     patchBytes,
		PatchType: &patchType,
	}
}

func buildMesaPatches(pod *corev1.Pod) []patchOperation {
	var patches []patchOperation

	// Check if volumes already exist (avoid duplicates)
	volumeExists := make(map[string]bool)
	for _, v := range pod.Spec.Volumes {
		volumeExists[v.Name] = true
	}

	// Mesa library paths to inject
	// Must include mesa libraries AND all their dependencies:
	// - libglvnd frontends (libEGL.so.1, libGL.so.1, etc)
	// - mesa implementations (libEGL_mesa.so.0, libGLX_mesa.so.0)
	// - libgbm.so.1 (GBM buffer management for DMABUF)
	// - libglapi.so.0 (mesa GL API)
	// - libdrm.so.2 (Direct Rendering Manager)
	// - X11/XCB libraries (libxcb*, libX11-xcb)
	// - Wayland libraries
	// - DRI drivers + EGL vendor config
	mesaLibs := []struct {
		name      string
		hostPath  string
		mountPath string
		pathType  corev1.HostPathType
	}{
		// libglvnd frontends
		{name: "mesa-libgl", hostPath: "/usr/lib64/libGL.so.1", mountPath: "/usr/lib64/libGL.so.1", pathType: corev1.HostPathFile},
		{name: "mesa-libegl", hostPath: "/usr/lib64/libEGL.so.1", mountPath: "/usr/lib64/libEGL.so.1", pathType: corev1.HostPathFile},
		{name: "mesa-libglesv2", hostPath: "/usr/lib64/libGLESv2.so.2", mountPath: "/usr/lib64/libGLESv2.so.2", pathType: corev1.HostPathFile},
		{name: "mesa-libglx", hostPath: "/usr/lib64/libGLX.so.0", mountPath: "/usr/lib64/libGLX.so.0", pathType: corev1.HostPathFile},
		// Mesa implementations
		{name: "mesa-libegl-impl", hostPath: "/usr/lib64/libEGL_mesa.so.0", mountPath: "/usr/lib64/libEGL_mesa.so.0", pathType: corev1.HostPathFile},
		{name: "mesa-libglx-impl", hostPath: "/usr/lib64/libGLX_mesa.so.0", mountPath: "/usr/lib64/libGLX_mesa.so.0", pathType: corev1.HostPathFile},
		{name: "mesa-libgbm", hostPath: "/usr/lib64/libgbm.so.1", mountPath: "/usr/lib64/libgbm.so.1", pathType: corev1.HostPathFile},
		{name: "mesa-libglapi", hostPath: "/usr/lib64/libglapi.so.0", mountPath: "/usr/lib64/libglapi.so.0", pathType: corev1.HostPathFile},
		// DRM
		{name: "mesa-libdrm", hostPath: "/usr/lib64/libdrm.so.2", mountPath: "/usr/lib64/libdrm.so.2", pathType: corev1.HostPathFile},
		// X11/XCB libraries
		{name: "mesa-libx11xcb", hostPath: "/usr/lib64/libX11-xcb.so.1", mountPath: "/usr/lib64/libX11-xcb.so.1", pathType: corev1.HostPathFile},
		{name: "mesa-libxcb", hostPath: "/usr/lib64/libxcb.so.1", mountPath: "/usr/lib64/libxcb.so.1", pathType: corev1.HostPathFile},
		{name: "mesa-libxcb-dri2", hostPath: "/usr/lib64/libxcb-dri2.so.0", mountPath: "/usr/lib64/libxcb-dri2.so.0", pathType: corev1.HostPathFile},
		{name: "mesa-libxcb-dri3", hostPath: "/usr/lib64/libxcb-dri3.so.0", mountPath: "/usr/lib64/libxcb-dri3.so.0", pathType: corev1.HostPathFile},
		{name: "mesa-libxcb-present", hostPath: "/usr/lib64/libxcb-present.so.0", mountPath: "/usr/lib64/libxcb-present.so.0", pathType: corev1.HostPathFile},
		{name: "mesa-libxcb-sync", hostPath: "/usr/lib64/libxcb-sync.so.1", mountPath: "/usr/lib64/libxcb-sync.so.1", pathType: corev1.HostPathFile},
		{name: "mesa-libxcb-xfixes", hostPath: "/usr/lib64/libxcb-xfixes.so.0", mountPath: "/usr/lib64/libxcb-xfixes.so.0", pathType: corev1.HostPathFile},
		{name: "mesa-libxcb-randr", hostPath: "/usr/lib64/libxcb-randr.so.0", mountPath: "/usr/lib64/libxcb-randr.so.0", pathType: corev1.HostPathFile},
		{name: "mesa-libxcb-glx", hostPath: "/usr/lib64/libxcb-glx.so.0", mountPath: "/usr/lib64/libxcb-glx.so.0", pathType: corev1.HostPathFile},
		{name: "mesa-libxshmfence", hostPath: "/usr/lib64/libxshmfence.so.1", mountPath: "/usr/lib64/libxshmfence.so.1", pathType: corev1.HostPathFile},
		// Wayland
		{name: "mesa-libwayland-client", hostPath: "/usr/lib64/libwayland-client.so.0", mountPath: "/usr/lib64/libwayland-client.so.0", pathType: corev1.HostPathFile},
		{name: "mesa-libwayland-server", hostPath: "/usr/lib64/libwayland-server.so.0", mountPath: "/usr/lib64/libwayland-server.so.0", pathType: corev1.HostPathFile},
		// X11 auth libraries
		{name: "mesa-libxau", hostPath: "/usr/lib64/libXau.so.6", mountPath: "/usr/lib64/libXau.so.6", pathType: corev1.HostPathFile},
		{name: "mesa-libxdmcp", hostPath: "/usr/lib64/libXdmcp.so.6", mountPath: "/usr/lib64/libXdmcp.so.6", pathType: corev1.HostPathFile},
		{name: "mesa-libbsd", hostPath: "/usr/lib64/libbsd.so.0", mountPath: "/usr/lib64/libbsd.so.0", pathType: corev1.HostPathFile},
		{name: "mesa-libmd", hostPath: "/usr/lib64/libmd.so.0", mountPath: "/usr/lib64/libmd.so.0", pathType: corev1.HostPathFile},
		// DRI drivers
		{name: "mesa-dri", hostPath: "/usr/lib64/dri", mountPath: "/usr/lib64/dri", pathType: corev1.HostPathDirectory},
		// EGL vendor configuration
		{name: "libglvnd-egl", hostPath: "/usr/share/glvnd/egl_vendor.d", mountPath: "/usr/share/glvnd/egl_vendor.d", pathType: corev1.HostPathDirectory},
		// NOTE: /dev/dri is NOT mounted - sharing host GPU causes kernel panics
	}

	// Add volumes
	for _, lib := range mesaLibs {
		if volumeExists[lib.name] {
			continue
		}

		// Add volume
		patches = append(patches, patchOperation{
			Op:   "add",
			Path: "/spec/volumes/-",
			Value: corev1.Volume{
				Name: lib.name,
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: lib.hostPath,
						Type: &lib.pathType,
					},
				},
			},
		})
	}

	// Add volume mounts to the compute container (first container)
	for i, container := range pod.Spec.Containers {
		if container.Name == "compute" {
			for _, lib := range mesaLibs {
				// Check if mount already exists
				mountExists := false
				for _, m := range container.VolumeMounts {
					if m.Name == lib.name {
						mountExists = true
						break
					}
				}
				if mountExists {
					continue
				}

				patches = append(patches, patchOperation{
					Op:   "add",
					Path: fmt.Sprintf("/spec/containers/%d/volumeMounts/-", i),
					Value: corev1.VolumeMount{
						Name:      lib.name,
						MountPath: lib.mountPath,
						ReadOnly:  true,
					},
				})
			}
			break
		}
	}

	return patches
}

func init() {
	// Check if cert files exist
	if _, err := os.Stat(certFile); os.IsNotExist(err) {
		log.Printf("Warning: cert file not found at %s", certFile)
	}
	if _, err := os.Stat(keyFile); os.IsNotExist(err) {
		log.Printf("Warning: key file not found at %s", keyFile)
	}
}
