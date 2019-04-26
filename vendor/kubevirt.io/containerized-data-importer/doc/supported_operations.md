# Containerized Data Importer supported operations
The Containerized Data Importer (CDI) supports importing data/disk images from a variety of sources, and it can be hard to understand which combination of operations is supported and if that operation requires scratch space. This document explains the combinations that are supported and which ones require scratch space. All Kubevirt content-type operations will support both file system Data Volumes (DV) as well as block volume DVs.

## Supported matrix

The first column represents the available content-types, Kubevirt and Archive. Kubevirt is broken down into QCOW2 vs RAW. QCOW2 needs to be converted before being written to the DV (and in a lot of cases requires scratch space for this conversion), where RAW doesn't need conversion and can be written directly to the DV.

| | http | https | http basic auth | Registry | S3 Bucket | Upload |
|--------------|---------|-|--|-------|--------|------------|
| KubeVirt(QCOW2)        |<ul><li>[x] QCOW2</li><li>[x] GZ\*</li><li>[x] XZ\*</li><li>[x] TAR\*</li></ul> |<ul><li>[x] QCOW2\*\*</li><li>[x] GZ\*</li><li>[x] XZ\*</li><li>[x] TAR\*</li></ul> |<ul><li>[x] QCOW2</li><li>[x] GZ\*</li><li>[x] XZ\*</li><li>[x] TAR\*</li></ul> | <ul><li>[ ] QCOW2</li><li>[ ] GZ</li><li>[ ] XZ</li><li>[ ] TAR</li></ul> | <ul><li>[x] QCOW2\*</li><li>[x] GZ\*</li><li>[x] XZ\*</li><li>[x] TAR\*</li></ul> | <ul><li>[x] QCOW2\*</li><li>[x] GZ\*</li><li>[x] XZ\*</li><li>[x] TAR\*</li></ul> |
| KubeVirt (RAW)          |<ul><li>[x] RAW</li><li>[x] GZ</li><li>[x] XZ</li><li>[x] TAR</li></ul> |<ul><li>[x] RAW</li><li>[x] GZ</li><li>[x] XZ</li><li>[x] TAR</li></ul> | <ul><li>[x] RAW</li><li>[x] GZ</li><li>[x] XZ</li><li>[x] TAR</li></ul> | <ul><li>[x] RAW*</li><li>[ ] GZ</li><li>[ ] XZ</li><li>[ ] TAR</li></ul> | <ul><li>[x] RAW</li><li>[x] GZ</li><li>[x] XZ</li><li>[x] TAR</li></ul> | <ul><li>[x] RAW</li><li>[x] GZ</li><li>[x] XZ</li><li>[x] TAR</li></ul> |
| Archive+ | <ul><li>[x] TAR</li></ul> | <ul><li>[x] TAR</li></ul> | <ul><li>[x] TAR</li></ul> | <ul><li>[ ] TAR</li></ul> | <ul><li>[ ] TAR</li></ul> | <ul><li>[ ] TAR</li></ul> |

\* Requires [scratch space](scratch-space.md)

\*\* Requires [scratch space](scratch-space.md) if a custom CA is required.

\+ Archive does not support block mode DVs