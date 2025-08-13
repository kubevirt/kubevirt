# StartupProbe, Readiness and LivenessProbes

The VMI spec allows setting `startupProbe`, `livenessProbe` and `readinessProbe` which translate to the same field on the resulting pod running the VM.

## Exec Probes

One option to probe the VM is by running commands on it and determine the ready/live state based on it's success.

Compared to pods, we need a way to execute the command inside the actual VM and not the pod.
To do that, we rely on the qemu-guest-agent to be available inside the VM.

A command supplied to an exec probe, will be wrapped by `virt-probe` in the operator and forwarded to the guest.

## Guest-Agent Ping Probe

Another option to probe the VM is by doing a qemu-guest-agent based `guest-ping`. This will ping the guest and return an error if the guest is not up and running.
To easily define this on VM spec, specify `guestAgentPing: {}` in VM's readiness probe spec; `virt-controller` will translate this into a corresponding command wrapped by `virt-probe`.

> Note: You can only define one of the type of probe.


**Important:** If the qemu-guest-agent is not installed **and** enabled inside the VM, the probe will fail.
Many images don't enabled the agent by default so make sure you either run one that does or enable it.

Make sure to provide enough delay and failureThreshold for the VM and the agent to be online.

### Example

The Fedora image used in this example does have qemu-guest-agent available by default. Nevertheless, in
case qemu-guest-agent is not installed, it will be installed and enabled via cloud-init as shown in the example below.
Also, cloud-init assigns the proper SELinux context, i.e. virt_qemu_ga_exec_t, to the `/tmp/healthy.txt` file.
Otherwise, SELinux will deny the attempts to open the `/tmp/healthy.txt` file causing the probe to fail.

> Note:  If SELinux is not installed in your container disk image, the command `chcon` should be removed from the VM
> manifest shown below. Otherwise, the `chcon`  command will fail.

1. Create the VM

```yaml
$ cat <<EOF | kubectl apply -f -
apiVersion: kubevirt.io/v1
kind: VirtualMachine
metadata:
  labels:
    kubevirt.io/vm: readiness-probe-vm
  name: readiness-probe
spec:
  runStrategy: Always
  template:
    metadata:
      labels:
        kubevirt.io/domain: readiness-probe
        kubevirt.io/vm: readiness-probe
    spec:
      domain:
        cpu:
          cores: 1
          sockets: 1
          threads: 1
        devices:
          disks:
            - name: containerdisk
              disk:
                bus: virtio
            - name: cloudinitdisk
              disk:
                bus: virtio
          rng: {}
        resources:
          requests:
            memory: 1Gi
      readinessProbe:
        exec:
          command: ["cat", "/tmp/healthy.txt"]
        failureThreshold: 10
        initialDelaySeconds: 120
        periodSeconds: 10
        # Note that timeoutSeconds value does not have any impact before K8s v1.20.
        timeoutSeconds: 5
      terminationGracePeriodSeconds: 180
      volumes:
        - containerDisk:
            image: quay.io/containerdisks/fedora
          name: containerdisk
        - cloudInitNoCloud:
            userData: |
              #cloud-config
              chpasswd:
                expire: false
              password: password
              user: fedora
              packages:
                qemu-guest-agent
              runcmd:
                - ["touch", "/tmp/healthy.txt"]
                - ["sudo", "chcon", "--type", "virt_qemu_ga_exec_t", "/tmp/healthy.txt"]
                - ["sudo", "systemctl", "enable", "--now", "qemu-guest-agent"]
          name: cloudinitdisk
EOF
```
2. (optional) Watch the VM events in a separate shell

```sh
# This will stream the events including any probe failures.
# Observe the guest-agent becoming available here.
kubectl get events --watch
```

3. Wait for the `.status.ready` field to be `true`, it may take a bit

```sh
kubectl wait vms/readiness-probe --for=condition=Ready --timeout=5m
```

4. (optional) Log in to the VM and watch the incoming qemu-ga commands

```sh
virtctl console readiness-probe
journalctl --follow
```
