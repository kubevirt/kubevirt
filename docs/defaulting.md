# Ahead of Time libvirt Defaulting on the Cluster/API Level

This Proposal fills a gap between KubeVirt as Runtime and Higher level
management, like proposed in
[docs/config-and-instances.md](docs/config-and-instances.md). It lays out a
design which overcomes the necessity for asynchronous specification updates,
which has it's origin in the fact that a lot of the defaulting is done on the
host by libvirt, so after scheduling.

With the proposal here, which will provide a synchronous way of fully
populating a VM specification, we come much closer to the meaning of a Pod
specification in Kubernetes.

Note that a fully populated VM specification in the context of this proposal, is
a specification which has filled out all fields, which influence how the
emulated machine looks like for the Guest OS. For instance libvirt network
configurations may differ on a host to host basis, e.g. because on one host a
path to an Ethernet device is `/dev/eth0` and on another one `/dev/eth1`,
although in both cases we connect to the same network. Such backend
configuration specific fields are not of interest for this proposal.

The goals of this proposal are:

 * Synchronous defaulting of all VM specification related fields, which
   influence how a Guest OS sees a VM.
 * As a consequence, enforcing the meaning of the VM specification as
   specification. We then don't have to change the specification after it was
   provided, or force users to extract additional data from the status field,
   which they would have to put into the VM specification in subsequent runs,
   to get exactly the same VM.
 * Give us more control over what we want to expose in `status`, instead of
   blindly copying every non-sensitive domain information into the status,
   which host-only libvirt defaulting would force us to do.

## Current situation

libvirt allows you to post very minimal Domain specifications. It will then
fill out a lot of device details for you. On cluster level this can lead to
difficulties, since depending on the fact if the VM has ever run before, you
will have to operate on different specs, since you have to incorporate VM
specification changes once a VM was started the first time. Furhter, getting
notified about these changes via asynchronous specification updates from the
host would weaken the VM specification as specification, the same applies to
reflecting the full VM specification in the `status`.

The following illustrates how immense the defaulting is.

First let's have a look on a fairly minimal KubeVirt VM Specification:

```yaml
apiVersion: kubevirt.io/v1alpha1
kind: VM
metadata:
  name: testvm
spec:
  domain:
    devices:
      interfaces:
      - type: network
        source:
          network: default
    memory:
      unit: MB
      value: 64
    os:
      type:
        os: hvm
    type: qemu
```

This can be translated to a quite small Domain XML:

```xml
<domain type="qemu" xmlns:qemu='http://libvirt.org/schemas/domain/qemu/1.0'>
  <name>testvm</name>
  <memory unit="MB">64</memory>
  <os>
    <type>hvm</type>
  </os>
  <devices>
      <interface type="network">
      <source network="default"></source>
    </interface>
   </devices>
</domain>
```

After we have defined the domain in libvirt, the XML looks like this:

```xml
<domain type='qemu' xmlns:qemu='http://libvirt.org/schemas/domain/qemu/1.0'>
  <name>testvm</name>
  <uuid>a8335812-53d4-4e57-ac0b-72951c885a57</uuid>
  <memory unit='KiB'>62500</memory>
  <currentMemory unit='KiB'>62500</currentMemory>
  <vcpu placement='static'>1</vcpu>
  <os>
    <type arch='x86_64' machine='pc-i440fx-2.7'>hvm</type>
    <boot dev='hd'/>
  </os>
  <clock offset='utc'/>
  <on_poweroff>destroy</on_poweroff>
  <on_reboot>restart</on_reboot>
  <on_crash>destroy</on_crash>
  <devices>
    <emulator>/usr/bin/qemu-system-x86_64</emulator>
    <controller type='usb' index='0' model='piix3-uhci'>
      <address type='pci' domain='0x0000' bus='0x00' slot='0x01' function='0x2'/>
    </controller>
    <controller type='pci' index='0' model='pci-root'/>
    <interface type='network'>
      <mac address='52:54:00:2e:84:30'/>
      <source network='default'/>
      <model type='rtl8139'/>
      <address type='pci' domain='0x0000' bus='0x00' slot='0x02' function='0x0'/>
    </interface>
    <input type='mouse' bus='ps2'/>
    <input type='keyboard' bus='ps2'/>
    <memballoon model='virtio'>
      <address type='pci' domain='0x0000' bus='0x00' slot='0x03' function='0x0'/>
    </memballoon>
  </devices>
</domain>
```

Finally, when a Domain was successfully started by libvirt, libvirt assigns an
`alias` to every device in the XML.

## The Problem

 * Hotplug operations are difficult with the minimal VM spec of kubevirt,
   since we could match wrong devices.
 * After VM restarts, different devices or addresses could be assigned to a VM
   by libvirt
 * libvirt assigns all that data on the host, either when defining the Domain,
   or after the Domain was started. That requires asynchronous lookups and in
   the worst case even merge logic.
 * If we don't want to change Guest OS relevant fields in the VM specification
   behind the back of the user, all computed default values by libvirt, need to
   be reflected in the VM status.

If your VMs are ephemeral (no migrations, recreating VMs vs. restarting VMs, no
hotplug), you are not affected by that.

## Proposed Solution

### Defaulting service based on libvirt

Except for some runtime information, which is normally not included in
migratable domain XMLs, libvirt fills in all defaults when defining a domain,
which influence how the emulated machine looks like for the Guest OS. We can
leverage that with a simple REST based service, which defines VMs in an
isolated libvirt (running libvirt and the small REST service in a Pod). The
service is a standalone component in KubeVirt and can be integrated in the
following flow:

 * When doing a POST to
   `/apis/kubevirt.io/v1alpha1/namespaces/mynamespace/vms`, `virt-api` can do
   a roundtrip to that service and prefill the defaults. As the response of
   the POST to `virt-api`, you will get the completely prefilled VM
   specification. This updated spec contains all the necessary bits, to allow
   consistent hotplugging and keeping the VM definition consistent between
   redefines.

Additional flows, like a dedicated `dry-run` endpoint, to allow management
applications, reusing existing admission controller logic for prefilling VMs,
withouth having to actually run a VM are thinkable. That would allow complete
decoupling of managment and runtime.

Defaulting rules:

 * The Kubevirt VM spec can contain abstract definitions, like a reference to a
   volume claim, or a reference to a host network. In such cases most of the
   time, the device `source` section needs to be replaced by a dummy `source`
   inside the defaulter. Then the fully populated `target` section needs to be
   mapped back (e.g. a PVC does not have to exist, when posting a VM. It can be
   mocked out, since it only provides the backend configuration).
 * Fields which don't influence how the Guest OS sees, shall not be defaulted.
   Such fields need to be filtered out.
 * There exist mandatory Domain XML fields, which we can't allow to be set on
   the VM specification. In such cases, the defaulter needs to set dummy values
   before asking libvirt for the defaults. Since these fields should not exist
   on the VM specification at all (see
   [docs/vm-configuration.md](docs/vm-configuration.md)), not mapping such
   dummy data back should be easy.

## Implementation Details

The implementation consists of two parts. First the libvirt
[admission-controller](https://kubernetes.io/docs/admin/admission-controllers/#what-are-they)
in `virt-api`. This admission controller, then reaches out to the second part,
a special libvirt instance in a container, wrapped by a REST API, which acts as
libvirt defaulting service. Since the defaults from this service would normally
be filled in on the host, it is important that this controller is the last of
all invoked admission controllers. For instance, if another admission
controller would exist, which allows injecting Disks, like a
[PodPreset](https://kubernetes.io/docs/tasks/inject-data-application/podpreset/)
allows injecting volumes, it has to be invoked before this libvirt defaulting
service. Otherwise relevant specification details, like PCI addresses might
still be added at the host level. the final flow looks like this:

```
 client       virt-api/apiserver admission-controller 1 libvirt admission controller  libvirt defaulting service
------------- ------------------ ---------------------- ----------------------------- ----------------------------


POST/VM ----->  validate/auth ----> modify/validate -----> POST /apis/vms/defaults ------> fill in defaults with
                                                                                                 libvirt
                                                                                                    |
                                                                                                    |
                                                                                                    v
                  validate <------------------------------- validate and return <--------- respond with full spec
                      |
                      |
                      v
                  persist
                      |
                      |
                      v
response <------- respond

```

This way of pre-filling the whole VM specification, allows users and management
applications on top of KubeVirt, to immediately get the resulting
specification, without the need to asynchronously watch for the missing
defaults, to be filled in.

## Important side notes

### Device naming

With the proposed Defaulting service, it is still pretty hard, from an
administration and operations perspective to find out which device is which.
libvirt at the moment does not support naming devices for the whole VM
lifecycle (https://bugzilla.redhat.com/show_bug.cgi?id=1434451). It actually
becomes a bigger problem with the defaulting service in place. To find out if a
device has changed, or needs an update, you would have to rely on the order of
the devices (which is not necessarily stable:
https://www.redhat.com/archives/libvirt-users/2012-December/msg00087.html), or
try to match them based on their content.

### Migrating between different QEMU versions

In order to make sure that the libvirt version, which does the defaulting
generates applicable defaults, it must first be compatible with the libvirt/qemu
pairs, currently running on the host. Second a guest machine type must be
specified, to guarantee migratability between the different versions in the
cluster.
