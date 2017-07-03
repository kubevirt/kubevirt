# Defaulting Service Proposal

This Proposal fills a gap between KubeVirt as Runtime and Higher level
management, like proposed in #272. It lays out a design wich overcomes the
necessity for asynchronous specification updates, which has it's origin in the
fact that a lot of the defaulting is done on the host, so after scheduling.

With the proposal here, which will provide a synchronous way of fully
populating a VM specification, we come much closer to the meaning of a Pod
specification in Kubernetes, but most important, this allows decoupling
administrative tasks completely from the runtime, when manipulating VM Specs
(the need to run VMs, to get a full VM spec).

## Current situation

Libvirt allows you to post very minimal Domain specifications. It will then
fill out a lot of device details for you. On cluster level this can lead to
difficulties, since depending on the fact if the VM has ever run before, you
will have to operate on different specs, and you have to incoroporate VM
specification changes once a VM was started the first time.

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

Finally, when a Domain was successfully started by Libvirt, Libvirt assigns an
`alias` to every device in the XML.

## The Problem

 * Hotplug operations are not possible with the minimal VM spec of kubevirt,
   since we could match wrong devices.
 * After VM restarts, different devices or addresses could be assigned to a VM
   by libvirt
 * Libvirt assigns all that data on the host, either when defining the Domain,
   or after the Domain was started. That requires asynchronous lookups, and
   when thinking about administrating offline VMs, it binds your domain
   specification management to the kubevirt runtime.

If your VMs are ephemeral (no migrations, recreating VMs vs. restarting VMs, no
hotplug), you are not affected by that.

## Proposed Solution

### Defaulting service based on libvirt

Except for some runtime information, which is normally not included in
migratable domain XMLs, libvirt fills in all defaults when defining a domain.
We can leverage that with a simple REST based service, which defines VMs in an
isolated libvirt (running libvirt and the small REST service in a Pod). The
service is a standalone component in KubeVirt and can then be integrated into
two different flows:

 1. When doing a POST to
    `/apis/kubevirt.io/v1alpha1/namespaces/mynamespace/vms`, `virt-api` can do
    a roundtrip to that service and prefill the defaults. As the response of
    the POST to `virt-api`, you will get the completely prefilled VM
    specification. This updated spec contains all the necessary bits, to allow
    consistent hotplugging and keeping the VM definition consistent between
    redefines.
 2. Add an extra endpoint to `virt-api` which only purpose is to prefill the VM
    spec. The URI will be `/apis/kubevirt.io/v1.alpha1/vms/defaults`
    (non-namespaced). You can POST VM specifications there and get a fully
    populated VM spec back. With this service URL in place, mixing
    administrative tasks like hotplug, adding devices for later runs and
    changing the spec for later runs (e.g. memory) can be solved in a
    consistent way, without the need to ever do an asynchronous roundtrip to
    libvirt to build consistent specs. **This allows decoupling administrative
    tasks completely from the runtime**. 

Defaulting rules:

 * The Kubevirt VM spec can contain abstract definitions, like a reference to a
   volume claim. In such cases most of the time, the device `source` section
   needs to be replaced by a dummy `source` inside the defaulter. Then the
   fully populated `target` section needs to be mapped back.
 * There exist mandatory Domain XML fields, which we can't allow to be set on
   the VM specification. In such cases, the defaulter needs to set dummy values
   before asking Libvirt for the defaults. Since these fields should not exist
   on the VM specification at all (see
   [docs/vm-configuration.md](docs/vm-configuration.md)), not mapping such
   dummy data back should be easy.

## Important side notes

### Device naming

With the proposed Defaulting service, it is still pretty hard, from an
administration and operations perspective to find out which device is which.
Libvirt at the moment does not support naming devices for the whole VM
lifecycle (https://bugzilla.redhat.com/show_bug.cgi?id=1434451). It actually
becomes a bigger problem with the defaulting service in place. To find out if a
device has changed, or needs an update, you would have to rely on the order of
the devices (which is not necessarily stable:
https://www.redhat.com/archives/libvirt-users/2012-December/msg00087.html), or
try to match them based on their content.

### Migrating between different QEMU versions

In order to make sure that the Libvirt version, which does the defaulting
generates appliable defaults, it must first be compatible with the libvirt/qemu
pairs, currently running on the host. Second a guest machine type must be
specified, to guarantee migratability between the different versions in the
cluster.
