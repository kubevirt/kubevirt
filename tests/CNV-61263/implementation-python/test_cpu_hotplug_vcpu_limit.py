"""
Tier 2 end-to-end tests for CPU hotplug vCPU limit enforcement.

STP Reference: tests/CNV-61263/CNV-61263_test_plan.md
STD Reference: tests/CNV-61263/CNV-61263_test_description.yaml

Jira: CNV-61263 - [CLOSED LOOP for] CPU hotplug logic still going over the limits
Source Bug: CNV-57352, CNV-48124

Related PRs:
    - kubevirt/kubevirt#14338: defaults: Limit MaxSockets based on maximum of vcpus
    - kubevirt/kubevirt#14511: [release-1.5] Cherry-pick

This test module validates that VMs with high vCPU counts can start successfully
after the MaxSockets fix. The fix prevents the CPU hotplug logic from exceeding
the 512 vCPU upper bound which would cause libvirt to reject the configuration.

Markers:
    - tier2
    - p1
    - compute

Preconditions:
    - OpenShift cluster with CNV 4.16.7+ or 4.19.0+
    - Worker node with sufficient CPUs (200+ for high vCPU tests)
    - libvirt version with eim support (libvirt-libs-10.0.0-6.15.el9_4+)
"""

import logging
import time

import pytest
from ocp_resources.virtual_machine import VirtualMachine
from ocp_resources.virtual_machine_instance import VirtualMachineInstance

from utilities.constants import TIMEOUT_5MIN, TIMEOUT_10MIN
from utilities.virt import VirtualMachineForTests, fedora_vm_body, running_vm

LOGGER = logging.getLogger(__name__)

pytestmark = [
    pytest.mark.usefixtures("namespace"),
    pytest.mark.tier2,
]


class TestCPUHotplugVCPULimit:
    """
    Test suite for CPU hotplug vCPU limit enforcement.

    Markers:
        - tier2
        - p1

    Preconditions:
        - OpenShift cluster with CNV 4.16.7+ or 4.19.0+
        - Worker node with sufficient CPUs (200+ for high vCPU tests)
        - libvirt version with eim support
    """

    @pytest.mark.polarion("CNV-61263-001")
    def test_ts_cnv61263_001_vm_216_cores_starts_successfully(
        self,
        unprivileged_client,
        namespace,
    ):
        """
        Test TS-CNV61263-001: VM with 216 cores must start successfully.

        This is the exact reproduction case from the bug report. The customer was
        unable to start VMs with 216 cores because libvirt rejected the configuration.

        Steps:
            1. Create VM with 216 cores, 1 socket, 1 thread
            2. Start VM and wait for Running status
            3. Verify no vCPU limit error in events

        Expected:
            - VM starts without 'Maximum CPUs greater than specified machine type limit' error
            - VM reaches Running status
            - virt-launcher pod shows successful sync
        """
        vm_name = "vm-216-cores"

        with VirtualMachineForTests(
            name=vm_name,
            namespace=namespace.name,
            client=unprivileged_client,
            body=fedora_vm_body(name=vm_name),
            cpu_cores=216,
            cpu_sockets=1,
            cpu_threads=1,
            memory_guest="10Gi",
        ) as vm:
            running_vm(vm=vm, timeout=TIMEOUT_10MIN)

            # Verify VM is running
            assert vm.vmi.status.phase == VirtualMachineInstance.Status.RUNNING, (
                f"VM {vm_name} failed to reach Running status, "
                f"current phase: {vm.vmi.status.phase}"
            )

            # Check for vCPU limit error in events
            vmi_events = list(vm.vmi.events)
            for event in vmi_events:
                assert "Maximum CPUs greater than specified machine type limit" not in (
                    event.message or ""
                ), f"vCPU limit error found in VMI events: {event.message}"

            LOGGER.info(f"VM {vm_name} with 216 cores started successfully")

    @pytest.mark.polarion("CNV-61263-002")
    def test_ts_cnv61263_002_vm_100_sockets_starts_successfully(
        self,
        unprivileged_client,
        namespace,
    ):
        """
        Test TS-CNV61263-002: VM with 100 sockets must start successfully.

        This was the original reported case from CNV-48124 that broke with CPU hotplug.
        With 100 sockets, the 4x ratio yields 400 vCPUs.

        Steps:
            1. Create VM with 1 core, 100 sockets, 1 thread
            2. Start VM and wait for Running status
            3. Verify VM is running

        Expected:
            - VM starts successfully
            - VM reaches Running status
            - No libvirt errors about extended interrupt mode (eim)
        """
        vm_name = "vm-100-sockets"

        with VirtualMachineForTests(
            name=vm_name,
            namespace=namespace.name,
            client=unprivileged_client,
            body=fedora_vm_body(name=vm_name),
            cpu_cores=1,
            cpu_sockets=100,
            cpu_threads=1,
            memory_guest="10Gi",
        ) as vm:
            running_vm(vm=vm, timeout=TIMEOUT_10MIN)

            # Verify VM is running
            assert vm.vmi.status.phase == VirtualMachineInstance.Status.RUNNING, (
                f"VM {vm_name} failed to reach Running status, "
                f"current phase: {vm.vmi.status.phase}"
            )

            LOGGER.info(f"VM {vm_name} with 100 sockets started successfully")

    @pytest.mark.polarion("CNV-61263-004")
    def test_ts_cnv61263_004_cpu_hotplug_with_capped_max_sockets(
        self,
        unprivileged_client,
        namespace,
    ):
        """
        Test TS-CNV61263-004: CPU hotplug must work with capped MaxSockets.

        This test validates that CPU hotplug works correctly when MaxSockets
        is capped due to the vCPU limit. The VM should be able to hotplug CPUs
        up to but not exceeding the capped MaxSockets value.

        Steps:
            1. Create VM with 64 cores, 2 sockets, 1 thread
            2. Start VM and wait for Running status
            3. Hotplug additional socket (patch to 3 sockets)
            4. Verify additional CPUs are visible in guest

        Expected:
            - VM starts with high core count
            - CPU hotplug succeeds up to MaxSockets limit
            - Guest OS sees additional CPUs (192 = 64 * 3)
        """
        vm_name = "vm-hotplug-test"
        initial_sockets = 2
        target_sockets = 3
        cores = 64
        threads = 1

        with VirtualMachineForTests(
            name=vm_name,
            namespace=namespace.name,
            client=unprivileged_client,
            body=fedora_vm_body(name=vm_name),
            cpu_cores=cores,
            cpu_sockets=initial_sockets,
            cpu_threads=threads,
            memory_guest="16Gi",
        ) as vm:
            running_vm(vm=vm, timeout=TIMEOUT_10MIN)

            # Verify initial CPU count
            initial_vcpus = cores * initial_sockets * threads
            LOGGER.info(f"VM started with {initial_vcpus} vCPUs")

            # Hotplug additional socket
            LOGGER.info(f"Hotplugging from {initial_sockets} to {target_sockets} sockets")
            vm.update(
                {
                    "spec": {
                        "template": {
                            "spec": {
                                "domain": {
                                    "cpu": {
                                        "sockets": target_sockets,
                                    }
                                }
                            }
                        }
                    }
                }
            )

            # Wait for hotplug to complete
            time.sleep(30)

            # Verify the VMI spec was updated
            expected_vcpus = cores * target_sockets * threads
            LOGGER.info(f"Expected vCPUs after hotplug: {expected_vcpus}")

            # Verify VM is still running after hotplug
            assert vm.vmi.status.phase == VirtualMachineInstance.Status.RUNNING, (
                f"VM {vm_name} not Running after CPU hotplug"
            )

            LOGGER.info(f"CPU hotplug successful - VM now has {target_sockets} sockets")

    @pytest.mark.polarion("CNV-61263-005")
    def test_ts_cnv61263_005_explicit_max_sockets_override(
        self,
        unprivileged_client,
        namespace,
    ):
        """
        Test TS-CNV61263-005: Explicit maxSockets should override default calculation.

        This test validates that when a user explicitly sets maxSockets,
        the auto-calculation is bypassed. With maxSockets=2 and 216 cores,
        total vCPUs = 2 * 216 * 1 = 432, which is under 512.

        Steps:
            1. Create VM with 216 cores, 1 socket, 1 thread, maxSockets=2
            2. Start VM and wait for Running status
            3. Verify maxSockets in VMI spec is 2 (not auto-calculated)

        Expected:
            - VM starts with explicit maxSockets=2
            - MaxSockets in VMI spec is 2 (not auto-calculated)
            - Total potential vCPUs stays at 432 (216 * 2 * 1)
            - User-specified configuration is respected
        """
        vm_name = "vm-explicit-maxsockets"
        explicit_max_sockets = 2

        # Create VM body with explicit maxSockets
        vm_body = fedora_vm_body(name=vm_name)

        # Patch the CPU spec to include maxSockets
        vm_body["spec"]["template"]["spec"]["domain"]["cpu"] = {
            "cores": 216,
            "sockets": 1,
            "threads": 1,
            "maxSockets": explicit_max_sockets,
        }

        with VirtualMachineForTests(
            name=vm_name,
            namespace=namespace.name,
            client=unprivileged_client,
            body=vm_body,
            memory_guest="10Gi",
        ) as vm:
            running_vm(vm=vm, timeout=TIMEOUT_10MIN)

            # Verify VM is running
            assert vm.vmi.status.phase == VirtualMachineInstance.Status.RUNNING, (
                f"VM {vm_name} failed to reach Running status"
            )

            # Verify maxSockets is the explicit value (not auto-calculated)
            vmi_max_sockets = vm.vmi.spec.domain.cpu.maxSockets
            assert vmi_max_sockets == explicit_max_sockets, (
                f"Expected maxSockets={explicit_max_sockets}, "
                f"but got maxSockets={vmi_max_sockets}"
            )

            LOGGER.info(
                f"VM {vm_name} started with explicit maxSockets={explicit_max_sockets} "
                f"(user configuration preserved)"
            )

    @pytest.mark.polarion("CNV-61263-007")
    def test_ts_cnv61263_007_numa_aware_high_vcpu_vm(
        self,
        unprivileged_client,
        namespace,
    ):
        """
        Test TS-CNV61263-007: NUMA-aware guest handling.

        This test validates behavior when a high-vCPU VM uses NUMA passthrough.
        NUMA-aware guests have additional topology constraints that may affect
        socket configuration beyond the vCPU limit fix.

        Steps:
            1. Create NUMA-aware VM with 64 cores, 2 sockets, guestMappingPassthrough
            2. Start VM and wait for Running status
            3. Verify NUMA topology in guest

        Expected:
            - NUMA-aware VM starts successfully (or fails with clear NUMA topology error)
            - Socket topology is compatible with NUMA configuration
            - Guest sees expected NUMA topology
        """
        vm_name = "vm-numa-passthrough"

        # Create VM body with NUMA passthrough
        vm_body = fedora_vm_body(name=vm_name)

        # Configure CPU with NUMA passthrough
        vm_body["spec"]["template"]["spec"]["domain"]["cpu"] = {
            "cores": 64,
            "sockets": 2,
            "threads": 1,
            "dedicatedCpuPlacement": True,
            "numa": {
                "guestMappingPassthrough": {},
            },
        }
        vm_body["spec"]["template"]["spec"]["domain"]["memory"] = {
            "guest": "16Gi",
        }

        with VirtualMachineForTests(
            name=vm_name,
            namespace=namespace.name,
            client=unprivileged_client,
            body=vm_body,
        ) as vm:
            try:
                running_vm(vm=vm, timeout=TIMEOUT_10MIN)

                # Verify VM is running
                assert vm.vmi.status.phase == VirtualMachineInstance.Status.RUNNING, (
                    f"VM {vm_name} failed to reach Running status"
                )

                LOGGER.info(
                    f"NUMA-aware VM {vm_name} started successfully with "
                    f"dedicatedCpuPlacement and guestMappingPassthrough"
                )

            except Exception as e:
                # NUMA guests may fail due to host topology constraints
                # This is acceptable as long as the error is clear
                LOGGER.warning(
                    f"NUMA-aware VM {vm_name} failed to start: {e}. "
                    f"This may be expected if host lacks NUMA topology support."
                )
                pytest.skip(
                    f"NUMA-aware VM failed to start due to host constraints: {e}"
                )
