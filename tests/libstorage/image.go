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

package libstorage

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/tests/framework/k8s"
	. "kubevirt.io/kubevirt/tests/framework/matcher"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ChangeImgFilePermissionsToNonQEMU(pvc *k8sv1.PersistentVolumeClaim) {
	args := []string{fmt.Sprintf(`chmod 640 %s && chown root:root %s && sync`, filepath.Join(DefaultPvcMountPath, "disk.img"), filepath.Join(DefaultPvcMountPath, "disk.img"))}

	By("changing disk.img permissions to non qemu")
	pod := RenderPodWithPVC("change-permissions-disk-img-pod", []string{"/bin/bash", "-c"}, args, pvc)

	// overwrite securityContext and run as root user
	pod.Spec.Containers[0].SecurityContext = &k8sv1.SecurityContext{
		Capabilities: &k8sv1.Capabilities{
			Drop: []k8sv1.Capability{"ALL"},
		},
		Privileged:   pointer.P(true),
		RunAsUser:    pointer.P(int64(0)),
		RunAsNonRoot: pointer.P(false),
	}

	k8sClient := k8s.Client()
	pod, err := k8sClient.CoreV1().Pods(pvc.Namespace).Create(context.Background(), pod, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())
	Eventually(ThisPod(pod), 120).Should(BeInPhase(k8sv1.PodSucceeded), "Could not set permissions to non-qemu")
}

func ArchiveToFile(tgtFile *os.File, sourceFilesNames ...string) {
	w := tar.NewWriter(tgtFile)
	defer w.Close()

	for _, src := range sourceFilesNames {
		srcFile, err := os.Open(src)
		Expect(err).ToNot(HaveOccurred())
		defer srcFile.Close()

		srcFileInfo, err := srcFile.Stat()
		Expect(err).ToNot(HaveOccurred())

		hdr, err := tar.FileInfoHeader(srcFileInfo, "")
		Expect(err).ToNot(HaveOccurred())

		err = w.WriteHeader(hdr)
		Expect(err).ToNot(HaveOccurred())

		_, err = io.Copy(w, srcFile)
		Expect(err).ToNot(HaveOccurred())
	}
}
