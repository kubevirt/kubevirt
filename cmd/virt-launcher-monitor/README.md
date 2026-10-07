# virt-launcher-monitor

PID 1 of the compute container. It supervises `/usr/bin/virt-launcher` and
stays small: a statically linked C binary with no Go runtime and no extra
Bazel rules.

Build:

```text
gcc -static -o virt-launcher-monitor main.c
```

`hack/build-go.sh` compiles it the same way as `cmd/container-disk-v2alpha`.
Bazel uses `cc_binary` with `-static`.

Behavior preserved from the Go monitor:

- SIGINT/SIGTERM/SIGQUIT are forwarded as SIGTERM to virt-launcher
- the monitor waits with `sigsuspend`; SIGCHLD is reaped with
  `waitpid(-1, WNOHANG)` in the handler
- full launcher argv is passed through except `--keep-after-failure`
- `--keep-after-failure` parks the compute container on failure
- `CAP_NET_BIND_SERVICE` is raised inheritable + ambient on the child
- after launcher exit, dump `/var/run/kubevirt/passt.log` and
  `/run/kubevirt-private/libvirt/qemu/log/*`
- leftover QEMU (`qemu-system` or `qemu-kvm`) gets SIGTERM; the monitor
  waits up to 10s on SIGCHLD
- delete `*.sock` under `--container-disk-dir`
- Istio ready probe on `:15021` and `POST :15020/quitquitquit`

Logs are kubevirt JSON lines on stderr (`component`, `level`, `msg`, `pos`)
so they match virt-launcher in the same stream. PID 1 does not pull in the
Go logging stack.
