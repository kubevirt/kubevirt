"""Rules for generating /etc/passwd and /etc/group files for container images."""

PasswdEntryInfo = provider(
    fields = [
        "username",
        "uid",
        "gid",
        "info",
        "home",
        "shell",
    ],
)

def _passwd_entry_impl(ctx):
    return [PasswdEntryInfo(
        username = ctx.attr.username,
        uid = ctx.attr.uid,
        gid = ctx.attr.gid,
        info = ctx.attr.info,
        home = ctx.attr.home,
        shell = ctx.attr.shell,
    )]

passwd_entry = rule(
    implementation = _passwd_entry_impl,
    attrs = {
        "gid": attr.int(default = 1000),
        "home": attr.string(default = "/home"),
        "info": attr.string(default = "user"),
        "shell": attr.string(default = "/bin/bash"),
        "uid": attr.int(default = 1000),
        "username": attr.string(mandatory = True),
    },
)

def _passwd_file_impl(ctx):
    content = "".join([
        "%s:x:%s:%s:%s:%s:%s\n" % (
            entry[PasswdEntryInfo].username,
            entry[PasswdEntryInfo].uid,
            entry[PasswdEntryInfo].gid,
            entry[PasswdEntryInfo].info,
            entry[PasswdEntryInfo].home,
            entry[PasswdEntryInfo].shell,
        )
        for entry in ctx.attr.entries
    ])
    out = ctx.actions.declare_file(ctx.label.name)
    ctx.actions.write(output = out, content = content)
    return DefaultInfo(files = depset([out]))

passwd_file = rule(
    implementation = _passwd_file_impl,
    attrs = {
        "entries": attr.label_list(
            allow_empty = False,
            providers = [PasswdEntryInfo],
        ),
    },
)

GroupEntryInfo = provider(
    fields = [
        "groupname",
        "gid",
        "users",
    ],
)

def _group_entry_impl(ctx):
    return [GroupEntryInfo(
        groupname = ctx.attr.groupname,
        gid = ctx.attr.gid,
        users = ctx.attr.users,
    )]

group_entry = rule(
    implementation = _group_entry_impl,
    attrs = {
        "gid": attr.int(default = 1000),
        "groupname": attr.string(mandatory = True),
        "users": attr.string_list(),
    },
)

def _group_file_impl(ctx):
    content = "".join([
        "%s:x:%s:%s\n" % (
            entry[GroupEntryInfo].groupname,
            entry[GroupEntryInfo].gid,
            ",".join(entry[GroupEntryInfo].users),
        )
        for entry in ctx.attr.entries
    ])
    out = ctx.actions.declare_file(ctx.label.name)
    ctx.actions.write(output = out, content = content)
    return DefaultInfo(files = depset([out]))

group_file = rule(
    implementation = _group_file_impl,
    attrs = {
        "entries": attr.label_list(
            allow_empty = False,
            providers = [GroupEntryInfo],
        ),
    },
)
