"""
Conftest for CommonInstancetypesDeployment tests.

Shared fixtures for Tier 2 tests.
"""

import pytest

from ocp_resources.namespace import Namespace

from utilities.constants import CNV_NAMESPACE


@pytest.fixture(scope="class")
def namespace(admin_client, unprivileged_client):
    """
    Create a test namespace for VM operations.

    Scope: class (shared across tests in a class)
    """
    ns_name = "test-common-instancetypes"
    with Namespace(
        client=admin_client,
        name=ns_name,
        teardown=True,
    ) as ns:
        ns.wait_for_status(status=Namespace.Status.ACTIVE, timeout=60)
        yield ns
