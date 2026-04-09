"""
Live Update NAD Reference on Running VM Tests

STP Reference: stps/sig-network/nad-live-update-stp.md
Jira: CNV-72329
"""

import logging

import pytest

from libs.net.vmspec import lookup_iface_status, lookup_iface_status_ip
from ocp_resources.virtual_machine_instance import VirtualMachineInstance
from tests.network.l2_bridge.libl2bridge import hot_plug_interface
from utilities.constants import TIMEOUT_2MIN, TIMEOUT_5MIN
from utilities.network import assert_ping_successful, is_destination_pingable_from_vm

from conftest import patch_vm_nad_reference

LOGGER = logging.getLogger(__name__)

pytestmark = pytest.mark.usefixtures("namespace")

HOT_PLUG_IFACE_NAME = "hotplug-iface"


@pytest.mark.tier2
class TestNADLiveUpdateE2E:
    """
    Tests for live update of NAD reference on a running VM's secondary network interface.

    Markers:
        - tier2

    Preconditions:
        - Two bridge-based NADs deployed on each worker node (nad1, nad2)
        - Running VM with secondary bridge interface on nad1
        - Peer VM running on nad2
        - MAC address and interface name of secondary interface recorded
    """

    def test_e2e_nad_change_connectivity(
        self,
        nad1_scope_class,
        nad2_scope_class,
        vm_on_nad1_scope_class,
        peer_vm_on_nad2_scope_class,
        peer_vm_ip,
    ):
        """
        Test TS-CNV72329-002: VM gains connectivity on new network after NAD change.

        Preconditions:
            - No connectivity to peer VM on nad2 (baseline)

        Steps:
            1. Verify no baseline connectivity to peer VM on nad2
            2. Patch VM spec to change NAD reference from nad1 to nad2
            3. Wait for update to complete

        Expected:
            - Ping from VM to peer VM on nad2 succeeds with 0% packet loss
        """
        LOGGER.info("Verifying no baseline connectivity to peer VM on nad2")
        assert not is_destination_pingable_from_vm(
            src_vm=vm_on_nad1_scope_class,
            dst_ip=str(peer_vm_ip),
            count=3,
        ), "VM should NOT have connectivity to peer on nad2 before NAD change"

        LOGGER.info("Patching VM spec to change NAD reference from nad1 to nad2")
        patch_vm_nad_reference(
            vm=vm_on_nad1_scope_class,
            network_name=nad1_scope_class.name,
            new_nad_name=nad2_scope_class.name,
        )

        LOGGER.info("Waiting for NAD change to take effect")
        lookup_iface_status(
            vm=vm_on_nad1_scope_class,
            iface_name=nad2_scope_class.name,
            timeout=TIMEOUT_5MIN,
        )

        LOGGER.info("Verifying connectivity to peer VM on nad2")
        assert_ping_successful(
            src_vm=vm_on_nad1_scope_class,
            dst_ip=lookup_iface_status_ip(
                vm=peer_vm_on_nad2_scope_class,
                iface_name=nad2_scope_class.name,
                ip_family=4,
            ),
        )

    def test_recovery_after_failed_nad_update(
        self,
        nad1_scope_class,
        nad2_scope_class,
        vm_on_nad1_scope_class,
    ):
        """
        Test TS-CNV72329-007: [NEGATIVE] VM recovers after failed NAD update.

        Steps:
            1. Patch VM spec to change NAD reference to non-existent NAD name
            2. Patch VM spec to change NAD reference to valid nad2

        Expected:
            - Error condition is reported for non-existent NAD
            - VM is "Running" and connected to nad2 after valid change
        """
        non_existent_nad = "does-not-exist-nad"

        LOGGER.info("Patching VM spec with non-existent NAD: %s", non_existent_nad)
        patch_vm_nad_reference(
            vm=vm_on_nad1_scope_class,
            network_name=nad1_scope_class.name,
            new_nad_name=non_existent_nad,
        )

        LOGGER.info("Verifying error condition on VM status")
        vmi = VirtualMachineInstance(
            name=vm_on_nad1_scope_class.name,
            namespace=vm_on_nad1_scope_class.namespace,
        )
        # VM should still be running despite the failed NAD reference
        assert vmi.instance, "VMI should still exist after failed NAD update"

        LOGGER.info("Patching VM spec with valid NAD: %s", nad2_scope_class.name)
        patch_vm_nad_reference(
            vm=vm_on_nad1_scope_class,
            network_name=nad1_scope_class.name,
            new_nad_name=nad2_scope_class.name,
        )

        LOGGER.info("Waiting for VM to recover and connect to nad2")
        lookup_iface_status(
            vm=vm_on_nad1_scope_class,
            iface_name=nad2_scope_class.name,
            timeout=TIMEOUT_5MIN,
        )

        LOGGER.info("Verifying VM is Running after recovery")
        vm_on_nad1_scope_class.wait_for_status(
            status=VirtualMachineInstance.Status.RUNNING,
            timeout=TIMEOUT_2MIN,
        )

    def test_hotplug_then_nad_change(
        self,
        nad1_scope_class,
        nad2_scope_class,
        hotplug_nad_scope_class,
        vm_on_nad1_scope_class,
    ):
        """
        Test TS-CNV72329-011: NIC hotplug and NAD change coexist on same VM.

        Steps:
            1. Hotplug a new bridge interface to the VM
            2. Patch VM spec to change NAD reference on existing secondary interface

        Expected:
            - Hotplugged interface reports valid IP address
            - Original secondary interface is connected to new NAD
        """
        LOGGER.info("Hotplugging new bridge interface to VM")
        hot_plug_interface(
            vm=vm_on_nad1_scope_class,
            hot_plugged_interface_name=HOT_PLUG_IFACE_NAME,
            net_attach_def_name=hotplug_nad_scope_class.name,
        )

        LOGGER.info("Verifying hotplugged interface has valid IP")
        hotplug_ip = lookup_iface_status_ip(
            vm=vm_on_nad1_scope_class,
            iface_name=hotplug_nad_scope_class.name,
            ip_family=4,
        )
        assert hotplug_ip, (
            f"Hotplugged interface {HOT_PLUG_IFACE_NAME} should report a valid IP"
        )

        LOGGER.info("Patching VM spec to change NAD reference from nad1 to nad2")
        patch_vm_nad_reference(
            vm=vm_on_nad1_scope_class,
            network_name=nad1_scope_class.name,
            new_nad_name=nad2_scope_class.name,
        )

        LOGGER.info("Waiting for NAD change to take effect")
        lookup_iface_status(
            vm=vm_on_nad1_scope_class,
            iface_name=nad2_scope_class.name,
            timeout=TIMEOUT_5MIN,
        )

        LOGGER.info("Verifying original secondary interface connected to nad2")
        nad2_ip = lookup_iface_status_ip(
            vm=vm_on_nad1_scope_class,
            iface_name=nad2_scope_class.name,
            ip_family=4,
        )
        assert nad2_ip, (
            "Original secondary interface should have IP on nad2 after NAD change"
        )
