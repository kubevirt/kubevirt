"""shared helper to provision users into distroless images

Create an OCI layer capable of provisioning a user into the target image
"""

load("@rules_distroless//distroless:defs.bzl", "flatten", "group", "home", "passwd")

def provision_user(name, username = None, groupname = None, uid = -1, gid = -1, provision_root_user = False, shell = "/bin/bash"):
    """populates /etc/passwd and /etc/groups for an oci image layer

    Args:
        name (str): name of the bazel target
        username (str): username of the user
        groupname (str): groupname of the user
        uid (int): uid of the user
        gid (int): gid of the user
        provision_root_user (bool): should the root user be created
        shell (str): shell of the user
    """

    home(
        name = "{}_home".format(name),
        dirs = [
            dict(
                home = "/home/{}".format(username),
                uid = uid,
                gid = gid,
            ),
        ],
        visibility = ["//visibility:private"],
    )

    users = []
    groups = []
    if provision_root_user:
        users.append(dict(
            gecos = ["root"],
            gid = 0,
            home = "/root",
            shell = shell,
            username = "root",
            uid = 0,
        ))
        groups.append(dict(
            name = "root",
            gid = 0,
            users = ["root"],
        ))

    users.append(dict(
        gecos = [username],
        gid = gid,
        home = "/home/{}".format(username),
        shell = shell,
        username = username,
        uid = uid,
    ))
    groups.append(dict(
        name = groupname,
        gid = gid,
        users = [username],
    ))

    passwd(
        name = "{}_passwd".format(name),
        entries = users,
        visibility = ["//visibility:private"],
    )

    group(
        name = "{}_group".format(name),
        entries = groups,
        visibility = ["//visibility:private"],
    )

    flatten(
        name = name,
        tars = [
            "{}_home".format(name),
            "{}_passwd".format(name),
            "{}_group".format(name),
        ],
        deduplicate = True,  # make sure the tar file doesn't contain /etc twice
        visibility = ["//visibility:private"],
    )
