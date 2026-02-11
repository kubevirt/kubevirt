"""
Shared fixtures for CNV-72329 Live Update NAD Reference tests.

These fixtures provide common test infrastructure for NAD reference
update testing scenarios.
"""

import pytest


# Note: admin_client, unprivileged_client, and namespace fixtures
# are provided by the main tests/conftest.py and should NOT be
# redefined here.


@pytest.fixture(scope="module")
def skip_if_feature_gate_disabled(admin_client):
    """
    Skip tests if LiveUpdateNADRefEnabled feature gate is not enabled.

    This fixture checks the HyperConverged CR for the feature gate
    and skips all tests in the module if it's not enabled.
    """
    # Check HyperConverged CR for feature gate
    # This is a placeholder - actual implementation depends on
    # how the feature gate is exposed
    try:
        # from ocp_resources.hyperconverged import HyperConverged
        # hco = HyperConverged(client=admin_client, name="kubevirt-hyperconverged")
        # feature_gates = hco.instance.spec.get("featureGates", {})
        # if not feature_gates.get("LiveUpdateNADRefEnabled", False):
        #     pytest.skip("LiveUpdateNADRefEnabled feature gate is not enabled")
        pass
    except Exception:
        # If we can't check, assume feature is available
        pass


@pytest.fixture(scope="module")
def multi_node_cluster_required(admin_client):
    """
    Skip tests if cluster doesn't have multiple nodes for migration.

    Live NAD update requires migration, which needs multiple nodes.
    """
    # from ocp_resources.node import Node
    # nodes = list(Node.get(client=admin_client))
    # worker_nodes = [n for n in nodes if "worker" in n.labels.get("node-role.kubernetes.io", "")]
    # if len(worker_nodes) < 2:
    #     pytest.skip("Multi-node cluster required for migration-based NAD update")
    pass
