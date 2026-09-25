# virt-launcher-monitor dependencies

`Cargo.lock` pins the Rust dependency graph for this binary. `cargo-bazel-lock.json`
pins the crate-universe rendering used by Bazel, so CI does not resolve a
different crate graph on each machine.

After intentionally changing `Cargo.toml`, regenerate the Bazel crate lock with:

```text
CARGO_BAZEL_REPIN=1 bazel sync --only=crate_index
```

Commit both lockfiles together with the dependency change.

The builder already provides `/usr/bin/ld` through `binutils`. The scoped
`.bazelrc` `host_action_env` setting keeps `/usr/bin` on PATH for the
`rules_rust` `process_wrapper` bootstrap link, without changing PATH for
unrelated workspace actions. The Rust compiler and exec compiler actions use
the scoped `extra_rustc_env` and `extra_exec_rustc_env` settings for the same
reason. The builder must provide `/usr/bin/ld` via `binutils`; verify with
`command -v ld` and `rpm -q binutils` before changing the builder image.

CS9 and CS10 sandboxes have different glibc. `rules_rust` does not hash the
sandbox gcc/glibc into exec-tool action keys, so a CS10-linked
`cargo-build-script` runner can be reused on CS9 and fail with
`GLIBC_2.39 not found`. `.bazelrc` therefore adds a no-op `--cfg=kubevirt_sandbox_cs9`
or `--cfg=kubevirt_sandbox_cs10` rustc flag so those cache keys stay distinct.
