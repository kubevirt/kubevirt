# Readiness and LivenessProbes

The VMI spec allows setting `livenessProbe` and `readinessProbe` which translate to the same field on the resulting pod running the VM.

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

**Note:** The Fedora image used in this example does not have the qemu-guest-agent available by default. We need to install and enable it through the console until this is resolved.
The cloud-init defined below will take care of the install for us, but we still need to enable it.

1. Create VM manifest

```yaml
# /tmp/probe-test.vm.yaml
apiVersion: kubevirt.io/v1alpha3
kind: VirtualMachine
metadata:
  name: probe-test
  namespace: default
  labels:
    app: probe-test
    vm.kubevirt.io/name: probe-test
    kubevirt.io/domain: probe-test
spec:
  running: false
  template:
    metadata:
      labels:
        vm.kubevirt.io/name: probe-test
        kubevirt.io/domain: probe-test
    spec:
      readinessProbe:
        exec:
          command:
          - cat
          - /tmp/ready.txt
        failureThreshold: 10
        initialDelaySeconds: 120
        periodSeconds: 10
        # Note that timeoutSeconds value does not have any impact before K8s v1.20.
        timeoutSeconds: 5
      domain:
        cpu:
          cores: 1
          sockets: 1
          threads: 1
        devices:
          disks:
            - disk:
                bus: virtio
              name: cloudinitdisk
            - bootOrder: 1
              disk:
                bus: virtio
              name: rootdisk
          interfaces:
            - masquerade: {}
              model: virtio
              name: nic-0
          networkInterfaceMultiqueue: true
          rng: {}
        machine:
          type: pc-q35-rhel8.2.0
        resources:
          requests:
            memory: 1Gi
      hostname: probe-test
      networks:
        - name: nic-0
          pod: {}
      terminationGracePeriodSeconds: 180
      volumes:
        - cloudInitNoCloud:
            userData: |
              #cloud-config
              user: fedora
              password: fedora
              chpasswd:
                expire: false
              packages:
                qemu-guest-agent
          name: cloudinitdisk
        - containerDisk:
            image: kubevirt/fedora-cloud-container-disk-demo
          name: rootdisk
```
2. Apply the VM manifest

```sh
kubectl apply -f /tmp/probe-test.yaml
```

3. Start the VM

```sh
kubectl virt start probe-test
```

4. (optional) Watch the VM events in a separate shell

```sh
# This will stream the events including any probe failures.
# Observe the guest-agent becomming available here.
kubectl get events --watch
```

5. Install the guest agent

```sh
kubectl virt console probe-test
```

```sh
sudo systemctl enable --now qemu-guest-agent
```

6. Create the `/tmp/ready.txt` file to satisfy the probe

```sh
touch /tmp/ready.txt
```

