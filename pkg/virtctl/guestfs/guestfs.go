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

package guestfs

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"

	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/virtctl/clientconfig"
	"kubevirt.io/kubevirt/pkg/virtctl/console"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

const (
	defaultImageName = "libguestfs-tools"
	defaultImage     = "quay.io/kubevirt/" + defaultImageName + ":latest"
	// KvmDevice defines the resource as in pkg/virt-controller/services/template.go, but we don't import the package to avoid compile conflicts when the os is windows
	KvmDevice         = "devices.kubevirt.io/kvm"
	fuseDevice        = "devices.kubevirt.io/fuse"
	volume            = "volume"
	contName          = "libguestfs"
	diskDir           = "/disk"
	diskPath          = "/dev/vda"
	podNamePrefix     = "libguestfs-tools"
	applianceDir      = "/usr/local/lib/guestfs"
	guestfsVolume     = "guestfs"
	appliancePath     = applianceDir + "/appliance"
	guestfsHome       = "/home/guestfs"
	tmpDirVolumeName  = "libguestfs-tmp-dir"
	tmpDirPath        = "/tmp/guestfs"
	pullPolicyDefault = corev1.PullIfNotPresent
	timeout           = 500 * time.Second
)

type guestfsCommand struct {
	pvc        string
	image      string
	kvm        bool
	root       bool
	fsGroup    string
	uid        string
	gid        string
	pullPolicy string
	vm         string
}

// Following variables allow overriding the default functions (useful for unit testing)
var CreateClientFunc = CreateClient
var CreateAttacherFunc = CreateAttacher
var ImageSetFunc = SetImage
var ImageInfoGetFunc = GetImageInfo

// NewGuestfsShellCommand returns a cobra.Command for starting libguestfs-tool pod and attach it to a pvc
func NewGuestfsShellCommand() *cobra.Command {
	c := guestfsCommand{}
	cmd := &cobra.Command{
		Use:     "guestfs",
		Short:   "Start a shell into the libguestfs pod",
		Long:    `Create a pod with libguestfs-tools, mount the pvc and attach a shell to it. The pvc is mounted under the /disks directory inside the pod for filesystem-based pvcs, or as /dev/vda for block-based pvcs`,
		Args:    cobra.ExactArgs(1),
		Example: usage(),
		RunE:    c.run,
	}
	cmd.PersistentFlags().StringVar(&c.image, "image", "", "libguestfs-tools container image")
	cmd.PersistentFlags().StringVar(&c.pullPolicy, "pull-policy", string(pullPolicyDefault), "pull policy for the libguestfs image")
	cmd.PersistentFlags().BoolVar(&c.kvm, "kvm", true, "Use kvm for the libguestfs-tools container")
	cmd.PersistentFlags().BoolVar(&c.root, "root", false, "Set uid 0 for the libguestfs-tool container")
	cmd.PersistentFlags().StringVar(&c.uid, "uid", "", "Set uid for the libguestfs-tool container. It doesn't work with root")
	cmd.PersistentFlags().StringVar(&c.gid, "gid", "", "Set gid for the libguestfs-tool container. This works only combined when the uid is manually set")
	cmd.SetUsageTemplate(templates.UsageTemplate())
	cmd.PersistentFlags().StringVar(&c.fsGroup, "fsGroup", "", "Set the fsgroup for the libguestfs-tool container")
	cmd.PersistentFlags().StringVar(&c.vm, "vm", "", "Provide a VM to apply its scheduling constraints to the libguestfs-tool pod")

	return cmd
}

func usage() string {
	usage := `  # Create a pod with libguestfs-tools, mount the pvc and attach a shell to it:
  {{ProgramName}} guestfs <pvc-name>`
	return usage
}

func (c *guestfsCommand) run(cmd *cobra.Command, args []string) error {
	c.pvc = args[0]

	virtClient, namespace, _, err := clientconfig.ClientAndNamespaceFromContext(cmd.Context())
	if err != nil {
		return err
	}

	if c.pullPolicy != string(corev1.PullAlways) &&
		c.pullPolicy != string(corev1.PullNever) &&
		c.pullPolicy != string(corev1.PullIfNotPresent) {
		return fmt.Errorf("Invalid pull policy: %s", c.pullPolicy)
	}
	var inUse bool
	client, err := CreateClientFunc(virtClient)
	if err != nil {
		return err
	}
	if c.image == "" {
		c.image, err = ImageSetFunc(client.VirtClient)
		if err != nil {
			return err
		}
	}
	fmt.Printf("Use image: %s \n", c.image)
	exist, _ := client.existsPVC(c.pvc, namespace)
	if !exist {
		return fmt.Errorf("The PVC %s doesn't exist", c.pvc)
	}
	inUse, err = client.isPVCinUse(c.pvc, namespace)
	if err != nil {
		return err
	}
	if inUse {
		return fmt.Errorf("PVC %s is used by another pod", c.pvc)
	}
	isBlock, err := client.isPVCVolumeBlock(c.pvc, namespace)
	if err != nil {
		return err
	}
	defer client.removePod(namespace, genPodName(c.pvc))
	return c.createInteractivePodWithPVC(client, namespace, "/entrypoint.sh", []string{}, isBlock)
}

// K8sClient holds the information of the Kubernetes client
type K8sClient struct {
	Client     kubernetes.Interface
	config     *rest.Config
	VirtClient kubecli.KubevirtClient
}

// SetImage sets the image name based on the information retrieved by the KubeVirt server.
func SetImage(virtClient kubecli.KubevirtClient) (string, error) {
	var imageName string
	info, err := ImageInfoGetFunc(virtClient)
	if err != nil {
		return "", fmt.Errorf("could not get guestfs image info: %v", err)
	}
	if info.GsImage != "" {
		// custom image set, no need to assemble url
		return info.GsImage, nil
	}
	// Set image name including prefix if available
	imageName = fmt.Sprintf("%s%s", info.ImagePrefix, defaultImageName)
	// Set the image version.
	if info.Digest != "" {
		imageName = fmt.Sprintf("%s@%s", imageName, info.Digest)
	} else if info.Tag != "" {
		imageName = fmt.Sprintf("%s:%s", imageName, info.Tag)
	} else {
		return "", fmt.Errorf("Neither the digest nor the tag for the image has been specified")
	}

	// Set the registry
	image := imageName
	if info.Registry != "" {
		image = fmt.Sprintf("%s/%s", info.Registry, imageName)
	}

	return image, nil
}

// GetImageInfo gets the image info based on the information on KubeVirt CR
func GetImageInfo(virtClient kubecli.KubevirtClient) (*kubecli.GuestfsInfo, error) {
	info, err := virtClient.GuestfsVersion().Get()
	if err != nil {
		return nil, err
	}

	return info, nil
}

func CreateClient(virtClient kubecli.KubevirtClient) (*K8sClient, error) {
	client, err := kubernetes.NewForConfig(virtClient.Config())
	if err != nil {
		return &K8sClient{}, err
	}
	return &K8sClient{
		Client:     client,
		config:     virtClient.Config(),
		VirtClient: virtClient,
	}, nil
}

func (client *K8sClient) existsPVC(pvc, ns string) (bool, error) {
	p, err := client.Client.CoreV1().PersistentVolumeClaims(ns).Get(context.TODO(), pvc, metav1.GetOptions{})
	if err != nil {
		return false, err
	}
	if p.Name == "" {
		return false, nil
	}
	return true, nil
}

func (client *K8sClient) isPVCVolumeBlock(pvc, ns string) (bool, error) {
	p, err := client.Client.CoreV1().PersistentVolumeClaims(ns).Get(context.TODO(), pvc, metav1.GetOptions{})
	if err != nil {
		return false, err
	}
	if *p.Spec.VolumeMode == corev1.PersistentVolumeBlock {
		return true, nil
	}
	return false, nil
}

func (client *K8sClient) existsPod(pod, ns string) bool {
	p, err := client.Client.CoreV1().Pods(ns).Get(context.TODO(), pod, metav1.GetOptions{})
	if err != nil {
		return false
	}
	if p.Name == "" {
		return false
	}
	return true
}

func (client *K8sClient) isPVCinUse(pvc, ns string) (bool, error) {
	pods, err := client.getPodsForPVC(pvc, ns)
	if err != nil {
		return false, err
	}
	if len(pods) > 0 {
		return true, nil
	}
	return false, nil
}

func (client *K8sClient) waitForContainerRunning(podName, ns string, timeout time.Duration) error {
	terminated := "Terminated"
	chTerm := make(chan os.Signal, 1)
	c := make(chan string, 1)
	signal.Notify(chTerm, os.Interrupt, syscall.SIGTERM)
	// if the user killed the guestfs command, the libguestfs-tools pod is also removed
	go func() {
		<-chTerm
		client.removePod(ns, podName)
		c <- terminated
	}()

	go func() {
		for {
			pod, err := client.Client.CoreV1().Pods(ns).Get(context.TODO(), podName, metav1.GetOptions{})
			if err != nil {
				c <- err.Error()
			}
			if pod.Status.Phase != corev1.PodPending {
				c <- string(pod.Status.Phase)

			}
			for _, c := range pod.Status.ContainerStatuses {
				if c.State.Waiting != nil {
					fmt.Printf("Waiting for container %s still in pending, reason: %s, message: %s \n", c.Name, c.State.Waiting.Reason, c.State.Waiting.Message)
				}
			}

			time.Sleep(5 * time.Second)
		}
	}()
	select {
	case res := <-c:
		if res == string(corev1.PodRunning) || res == terminated {
			return nil
		}
		return fmt.Errorf("Pod is not in running state but got %s", res)
	case <-time.After(timeout):
		return fmt.Errorf("timeout in waiting for the containers to be started in pod %s", podName)
	}
}

func (client *K8sClient) getPodsForPVC(pvcName, ns string) ([]corev1.Pod, error) {
	nsPods, err := client.Client.CoreV1().Pods(ns).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return []corev1.Pod{}, err
	}

	var pods []corev1.Pod

	for _, pod := range nsPods.Items {
		for _, volume := range pod.Spec.Volumes {
			if volume.VolumeSource.PersistentVolumeClaim != nil && volume.VolumeSource.PersistentVolumeClaim.ClaimName == pvcName {
				pods = append(pods, pod)
			}
		}
	}

	return pods, nil
}

func (c *guestfsCommand) setFSGroupLibguestfs() (*int64, error) {
	if c.root && c.fsGroup != "" {
		return nil, fmt.Errorf("cannot set fsGroup id with root")
	}
	if c.fsGroup != "" {
		n, err := strconv.ParseInt(c.fsGroup, 10, 64)
		if err != nil {
			return nil, err
		}
		return &n, nil
	}
	if c.root {
		var rootFsID int64 = 0
		return &rootFsID, nil
	}
	return nil, nil
}

// setUIDLibguestfs returns the guestfs uid
func (c *guestfsCommand) setUIDLibguestfs() (*int64, error) {
	switch {
	case c.root:
		var zero int64
		if c.uid != "" {
			return nil, fmt.Errorf("cannot set uid if root is true")
		}
		return &zero, nil
	case c.uid != "":
		n, err := strconv.ParseInt(c.uid, 10, 64)
		if err != nil {
			return nil, err
		}
		return &n, nil
	default:
		return nil, nil
	}
}

func (c *guestfsCommand) setGIDLibguestfs() (*int64, error) {
	// The GID can only be specified together with the uid. See comment at: https://github.com/kubernetes/cri-api/blob/2b5244cefaeace624cb160d6b3d85dd3fd14baea/pkg/apis/runtime/v1/api.proto#L307-L309
	if c.gid != "" && c.uid == "" {
		return nil, fmt.Errorf("gid requires the uid to be set")
	}

	if c.root && c.gid != "" {
		return nil, fmt.Errorf("cannot set gid id with root")
	}
	if c.gid != "" {
		n, err := strconv.ParseInt(c.gid, 10, 64)
		if err != nil {
			return nil, err
		}
		return &n, nil
	}
	if c.root {
		var rootGID int64 = 0
		return &rootGID, nil
	}
	return nil, nil
}

func (c *guestfsCommand) createLibguestfsPod(client *K8sClient, ns, cmd string, args []string, isBlock bool) (*corev1.Pod, error) {
	var (
		resources    corev1.ResourceRequirements
		tolerations  []corev1.Toleration
		affinity     *corev1.Affinity
		labels       map[string]string
		nodeSelector map[string]string
	)
	if c.kvm {
		resources = corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				KvmDevice: resource.MustParse("1"),
				fuseDevice: resource.MustParse("1"),
			},
		}
	}
	if c.vm != "" {
		if vm, err := client.VirtClient.VirtualMachine(ns).Get(context.Background(), c.vm, metav1.GetOptions{}); err == nil {
			tolerations = vm.Spec.Template.Spec.Tolerations
			affinity = vm.Spec.Template.Spec.Affinity
			labels = vm.Spec.Template.ObjectMeta.Labels
			nodeSelector = vm.Spec.Template.Spec.NodeSelector
		} else {
			return nil, err
		}
	}
	u, err := c.setUIDLibguestfs()
	if err != nil {
		return nil, err
	}
	g, err := c.setGIDLibguestfs()
	if err != nil {
		return nil, err
	}
	f, err := c.setFSGroupLibguestfs()
	if err != nil {
		return nil, err
	}
	allowPrivilegeEscalation := false
	containerSecurityContext := &corev1.SecurityContext{
		AllowPrivilegeEscalation: &allowPrivilegeEscalation,
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{"ALL"},
		},
	}
	securityContext := &corev1.PodSecurityContext{
		RunAsNonRoot: pointer.P(!c.root),
		RunAsUser:    u,
		RunAsGroup:   g,
		FSGroup:      f,
		SeccompProfile: &corev1.SeccompProfile{
			Type: corev1.SeccompProfileTypeRuntimeDefault,
		},
	}
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:   genPodName(c.pvc),
			Labels: labels,
		},
		Spec: corev1.PodSpec{
			SecurityContext: securityContext,
			Volumes: []corev1.Volume{
				{
					Name: volume,
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: c.pvc,
							ReadOnly:  false,
						},
					},
				},
				// Use emptyDir to store temporary files generated by libguestfs
				{
					Name: tmpDirVolumeName,
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
				{
					Name: guestfsVolume,
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			},
			Containers: []corev1.Container{
				{
					Name:    contName,
					Image:   c.image,
					Command: []string{cmd},
					Args:    args,
					// Set env variable to start libguestfs:
					// LIBGUESTFS_BACKEND sets libguestfs to directly use qemu
					// LIBGUESTFS_PATH sets the path where the root, initrd and the kernel are located
					// LIBGUESTFS_TMPDIR sets the path where temporary files generated by libguestfs are stored
					Env: []corev1.EnvVar{
						{
							Name:  "LIBGUESTFS_BACKEND",
							Value: "direct",
						},
						{
							Name:  "LIBGUESTFS_PATH",
							Value: appliancePath,
						},
						{
							Name:  "LIBGUESTFS_TMPDIR",
							Value: tmpDirPath,
						},
						{
							Name:  "HOME",
							Value: guestfsHome,
						},
					},
					SecurityContext: containerSecurityContext,
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      tmpDirVolumeName,
							ReadOnly:  false,
							MountPath: tmpDirPath,
						},
						{
							Name:      guestfsVolume,
							ReadOnly:  false,
							MountPath: guestfsHome,
						},
					},
					ImagePullPolicy: corev1.PullPolicy(c.pullPolicy),
					Stdin:           true,
					TTY:             true,
					Resources:       resources,
				},
			},
			RestartPolicy: corev1.RestartPolicyNever,
			Tolerations:   tolerations,
			Affinity:      affinity,
			NodeSelector:  nodeSelector,
		},
	}
	if isBlock {
		pod.Spec.Containers[0].VolumeDevices = append(pod.Spec.Containers[0].VolumeDevices, corev1.VolumeDevice{
			Name:       volume,
			DevicePath: diskPath,
		})
		fmt.Printf("The PVC has been mounted at %s \n", diskPath)
	} else {
		// PVC volume mode is filesystem
		pod.Spec.Containers[0].VolumeMounts = append(pod.Spec.Containers[0].VolumeMounts, corev1.VolumeMount{
			Name:      volume,
			ReadOnly:  false,
			MountPath: diskDir,
		})

		pod.Spec.Containers[0].WorkingDir = diskDir
		fmt.Printf("The PVC has been mounted at %s \n", diskDir)
	}

	p, err := client.Client.CoreV1().Pods(ns).Create(context.TODO(), pod, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	return p, nil
}

// CreateAttacher attaches the stdin, stdout, and stderr to the container shell
func CreateAttacher(client *K8sClient, p *corev1.Pod, command string) error {
	req := client.Client.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(p.Name).
		Namespace(p.Namespace).
		SubResource("attach")
	req.VersionedParams(
		&corev1.PodAttachOptions{
			Container: contName,
			Stdin:     true,
			Stdout:    true,
			Stderr:    true,
			TTY:       true,
		}, scheme.ParameterCodec,
	)
	exec, err := remotecommand.NewSPDYExecutor(client.config, "POST", req.URL())
	if err != nil {
		return err
	}

	stdinReader, stdinWriter := io.Pipe()
	stdoutReader, stdoutWriter := io.Pipe()
	resChan := make(chan error)

	go func() {
		resChan <- exec.Stream(remotecommand.StreamOptions{
			Stdin:  stdinReader,
			Stdout: stdoutWriter,
			Stderr: stdoutWriter,
		})
	}()
	return console.Attach(stdinReader, stdoutReader, stdinWriter, stdoutWriter,
		"If you don't see a command prompt, try pressing enter.", resChan)
}

func (c *guestfsCommand) createInteractivePodWithPVC(client *K8sClient, ns, command string, args []string, isblock bool) error {
	pod, err := c.createLibguestfsPod(client, ns, command, args, isblock)
	if err != nil {
		return err
	}
	err = client.waitForContainerRunning(genPodName(c.pvc), ns, timeout)
	if err != nil {
		return err
	}
	return CreateAttacherFunc(client, pod, command)
}

func (client *K8sClient) removePod(ns, podName string) error {
	return client.Client.CoreV1().Pods(ns).Delete(context.TODO(), podName, metav1.DeleteOptions{})
}

func genPodName(pvc string) string {
	return fmt.Sprintf("%s-%s", podNamePrefix, pvc)
}
