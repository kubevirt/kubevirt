"""
Shared fixtures for Live Update NAD Reference tests.

STP Reference: stps/sig-network/nad-live-update-stp.md
Jira: CNV-72329
"""

import logging

import pytest

from libs.net.vmspec import lookup_iface_status_ip
from ocp_resources.resource import ResourceEditor
from utilities.constants import LINUX_BRIDGE
from utilities.network import network_nad
from utilities.virt import VirtualMachineForTests, fedora_vm_body

LOGGER = logging.getLogger(__name__)

SECONDARY_NETWORK_NAME = "secondary"
NAD1_BRIDGE = "br-nad1"
NAD2_BRIDGE = "br-nad2"
HOTPLUG_BRIDGE = "br-hotplug"


@pytest.fixture(scope="class")
def nad1_scope_class(admin_client, namespace, bridge_device_nad1):
    """
    First bridge-based NetworkAttachmentDefinition (nad1).

    Yields:
        NetworkAttachmentDefinition: NAD on br-nad1
    """
    with network_nad(
        namespace=namespace,
        nad_type=LINUX_BRIDGE,
        nad_name="nad1",
        interface_name=bridge_device_nad1.bridge_name,
        client=admin_client,
    ) as nad:
        yield nad


@pytest.fixture(scope="class")
def nad2_scope_class(admin_client, namespace, bridge_device_nad2):
    """
    Second bridge-based NetworkAttachmentDefinition (nad2).

    Yields:
        NetworkAttachmentDefinition: NAD on br-nad2
    """
    with network_nad(
        namespace=namespace,
        nad_type=LINUX_BRIDGE,
        nad_name="nad2",
        interface_name=bridge_device_nad2.bridge_name,
        client=admin_client,
    ) as nad:
        yield nad


@pytest.fixture(scope="class")
def hotplug_nad_scope_class(admin_client, namespace, bridge_device_hotplug):
    """
    Bridge-based NetworkAttachmentDefinition for hotplug testing.

    Yields:
        NetworkAttachmentDefinition: NAD on br-hotplug
    """
    with network_nad(
        namespace=namespace,
        nad_type=LINUX_BRIDGE,
        nad_name="hotplug-nad",
        interface_name=bridge_device_hotplug.bridge_name,
        client=admin_client,
    ) as nad:
        yield nad


@pytest.fixture(scope="class")
def vm_on_nad1_scope_class(unprivileged_client, namespace, nad1_scope_class):
    """
    Running Fedora VM with secondary bridge interface on nad1.

    Yields:
        VirtualMachineForTests: VM with secondary interface on nad1
    """
    name = "vm-nad-live-update"
    networks = {nad1_scope_class.name: nad1_scope_class.name}
    with VirtualMachineForTests(
        namespace=namespace.name,
        name=name,
        body=fedora_vm_body(name=name),
        networks=networks,
        interfaces=networks.keys(),
        client=unprivileged_client,
    ) as vm:
        vm.start(wait=True)
        vm.wait_for_agent_connected()
        yield vm


@pytest.fixture(scope="class")
def peer_vm_on_nad2_scope_class(unprivileged_client, namespace, nad2_scope_class):
    """
    Peer Fedora VM running on nad2 for connectivity verification.

    Yields:
        VirtualMachineForTests: Peer VM on nad2
    """
    name = "peer-vm-nad2"
    networks = {nad2_scope_class.name: nad2_scope_class.name}
    with VirtualMachineForTests(
        namespace=namespace.name,
        name=name,
        body=fedora_vm_body(name=name),
        networks=networks,
        interfaces=networks.keys(),
        client=unprivileged_client,
    ) as vm:
        vm.start(wait=True)
        vm.wait_for_agent_connected()
        yield vm


@pytest.fixture()
def peer_vm_ip(peer_vm_on_nad2_scope_class, nad2_scope_class):
    """
    IPv4 address of the peer VM on nad2.

    Returns:
        ipaddress.IPv4Address: Peer VM IP on nad2
    """
    return lookup_iface_status_ip(
        vm=peer_vm_on_nad2_scope_class,
        iface_name=nad2_scope_class.name,
        ip_family=4,
    )


def patch_vm_nad_reference(vm, network_name, new_nad_name):
    """Patch VM spec to change the NAD reference for a named network."""
    networks = []
    for network in vm.instance.spec.template.spec.networks:
        net_dict = network.to_dict()
        if net_dict.get("name") == network_name and "multus" in net_dict:
            net_dict["multus"]["networkName"] = new_nad_name
        networks.append(net_dict)

    ResourceEditor(
        patches={
            vm: {
                "spec": {
                    "template": {
                        "spec": {
                            "networks": networks,
                        }
                    }
                }
            }
        }
    ).update()
