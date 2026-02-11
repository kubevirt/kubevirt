"""
Pytest configuration and fixtures for CNV-61263 CPU hotplug vCPU limit tests.

Jira: CNV-61263 - [CLOSED LOOP for] CPU hotplug logic still going over the limits

This conftest provides test-specific fixtures for high vCPU VM tests.
Global fixtures (admin_client, unprivileged_client, namespace) are inherited
from the main tests/conftest.py.
"""

import logging

import pytest
from ocp_resources.namespace import Namespace

LOGGER = logging.getLogger(__name__)


# Constants for CPU hotplug tests
HIGH_CPU_VM_MEMORY = "10Gi"
NUMA_VM_MEMORY = "16Gi"
VM_STARTUP_TIMEOUT_HIGH_CPU = 600  # 10 minutes for high vCPU VMs


@pytest.fixture(scope="class")
def high_cpu_worker_node(admin_client, workers):
    """
    Find a worker node with sufficient CPUs for high vCPU tests.

    Returns:
        str: Name of worker node with 200+ CPUs, or None if not found.

    Note:
        Tests requiring 200+ vCPUs should skip if no suitable node is available.
    """
    min_cpus_required = 200

    for worker in workers:
        cpu_capacity = int(worker.instance.status.capacity.get("cpu", 0))
        if cpu_capacity >= min_cpus_required:
            LOGGER.info(
                f"Found high-CPU worker node: {worker.name} with {cpu_capacity} CPUs"
            )
            return worker.name

    LOGGER.warning(
        f"No worker node found with {min_cpus_required}+ CPUs. "
        f"High vCPU tests may fail or be skipped."
    )
    return None


@pytest.fixture(scope="class")
def skip_if_insufficient_cpus(high_cpu_worker_node):
    """
    Skip test if no high-CPU worker node is available.

    Use this fixture in tests that require 200+ vCPU VMs.
    """
    if high_cpu_worker_node is None:
        pytest.skip(
            "No worker node with 200+ CPUs available for high vCPU tests"
        )


@pytest.fixture(scope="class")
def numa_capable_node(admin_client, workers):
    """
    Find a worker node with NUMA topology for NUMA-aware tests.

    Returns:
        str: Name of NUMA-capable worker node, or None if not found.
    """
    for worker in workers:
        # Check for NUMA topology by looking at topology hints
        # or node labels indicating multi-socket configuration
        node_labels = worker.instance.metadata.labels or {}

        # Common indicators of NUMA capability
        if any(
            label.startswith("topology.kubernetes.io/")
            for label in node_labels
        ):
            LOGGER.info(f"Found NUMA-capable worker node: {worker.name}")
            return worker.name

    LOGGER.warning(
        "No NUMA-capable worker node found. NUMA tests may fail or be skipped."
    )
    return None
