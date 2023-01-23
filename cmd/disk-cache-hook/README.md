# Disk cache hook sidecar

### usage:
```yaml
  annotations:
    hooks.kubevirt.io/hookSidecars: '[{"args": ["--cache-type", "directsync"], "image":
      "registry:5000/kubevirt/disk-cache-hook-sidecar-image:devel"}]'
  labels:
```
----
The `cache-type` argument can be:

* `writeback`
* `writethrough`
* `none`
* `unsafe`
* `directsync`

