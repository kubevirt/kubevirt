/*
 * This file is part of the libvirt-go project
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE.
 *
 * Copyright (c) 2013 Alex Zorin
 * Copyright (C) 2016 Red Hat, Inc.
 *
 */

package libvirt

/*
#cgo pkg-config: libvirt
#include <assert.h>
#include "domain_wrapper.h"

int
virDomainAbortJobWrapper(virDomainPtr domain,
                         virErrorPtr err)
{
    int ret = virDomainAbortJob(domain);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainAddIOThreadWrapper(virDomainPtr domain,
                            unsigned int iothread_id,
                            unsigned int flags,
                            virErrorPtr err)
{
#if LIBVIR_VERSION_NUMBER < 1002015
    assert(0); // Caller should have checked version
#else
    int ret = virDomainAddIOThread(domain, iothread_id, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
#endif
}


int
virDomainAttachDeviceWrapper(virDomainPtr domain,
                             const char *xml,
                             virErrorPtr err)
{
    int ret = virDomainAttachDevice(domain, xml);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainAttachDeviceFlagsWrapper(virDomainPtr domain,
                                  const char *xml,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    int ret = virDomainAttachDeviceFlags(domain, xml, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainBlockCommitWrapper(virDomainPtr dom,
                            const char *disk,
                            const char *base,
                            const char *top,
                            unsigned long bandwidth,
                            unsigned int flags,
                            virErrorPtr err)
{
    int ret = virDomainBlockCommit(dom, disk, base, top, bandwidth, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainBlockCopyWrapper(virDomainPtr dom,
                          const char *disk,
                          const char *destxml,
                          virTypedParameterPtr params,
                          int nparams,
                          unsigned int flags,
                          virErrorPtr err)
{
#if LIBVIR_VERSION_NUMBER < 1002008
    assert(0); // Caller should have checked version
#else
    int ret = virDomainBlockCopy(dom, disk, destxml, params, nparams, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
#endif
}


int
virDomainBlockJobAbortWrapper(virDomainPtr dom,
                              const char *disk,
                              unsigned int flags,
                              virErrorPtr err)
{
    int ret = virDomainBlockJobAbort(dom, disk, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainBlockJobSetSpeedWrapper(virDomainPtr dom,
                                 const char *disk,
                                 unsigned long bandwidth,
                                 unsigned int flags,
                                 virErrorPtr err)
{
    int ret = virDomainBlockJobSetSpeed(dom, disk, bandwidth, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainBlockPeekWrapper(virDomainPtr dom,
                          const char *disk,
                          unsigned long long offset,
                          size_t size,
                          void *buffer,
                          unsigned int flags,
                          virErrorPtr err)
{
    int ret = virDomainBlockPeek(dom, disk, offset, size, buffer, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainBlockPullWrapper(virDomainPtr dom,
                          const char *disk,
                          unsigned long bandwidth,
                          unsigned int flags,
                          virErrorPtr err)
{
    int ret = virDomainBlockPull(dom, disk, bandwidth, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainBlockRebaseWrapper(virDomainPtr dom,
                            const char *disk,
                            const char *base,
                            unsigned long bandwidth,
                            unsigned int flags,
                            virErrorPtr err)
{
    int ret = virDomainBlockRebase(dom, disk, base, bandwidth, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainBlockResizeWrapper(virDomainPtr dom,
                            const char *disk,
                            unsigned long long size,
                            unsigned int flags,
                            virErrorPtr err)
{
    int ret = virDomainBlockResize(dom, disk, size, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainBlockStatsWrapper(virDomainPtr dom,
                           const char *disk,
                           virDomainBlockStatsPtr stats,
                           size_t size,
                           virErrorPtr err)
{
    int ret = virDomainBlockStats(dom, disk, stats, size);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainBlockStatsFlagsWrapper(virDomainPtr dom,
                                const char *disk,
                                virTypedParameterPtr params,
                                int *nparams,
                                unsigned int flags,
                                virErrorPtr err)
{
    int ret = virDomainBlockStatsFlags(dom, disk, params, nparams, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainCoreDumpWrapper(virDomainPtr domain,
                         const char *to,
                         unsigned int flags,
                         virErrorPtr err)
{
    int ret = virDomainCoreDump(domain, to, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainCoreDumpWithFormatWrapper(virDomainPtr domain,
                                   const char *to,
                                   unsigned int dumpformat,
                                   unsigned int flags,
                                   virErrorPtr err)
{
#if LIBVIR_VERSION_NUMBER < 1002003
    assert(0); // Caller should have checked version
#else
    int ret = virDomainCoreDumpWithFormat(domain, to, dumpformat, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
#endif
}


int
virDomainCreateWrapper(virDomainPtr domain,
                       virErrorPtr err)
{
    int ret = virDomainCreate(domain);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainCreateWithFilesWrapper(virDomainPtr domain,
                                unsigned int nfiles,
                                int *files,
                                unsigned int flags,
                                virErrorPtr err)
{
    int ret = virDomainCreateWithFiles(domain, nfiles, files, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainCreateWithFlagsWrapper(virDomainPtr domain,
                                unsigned int flags,
                                virErrorPtr err)
{
    int ret = virDomainCreateWithFlags(domain, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainDelIOThreadWrapper(virDomainPtr domain,
                            unsigned int iothread_id,
                            unsigned int flags,
                            virErrorPtr err)
{
#if LIBVIR_VERSION_NUMBER < 1002015
    assert(0); // Caller should have checked version
#else
    int ret = virDomainDelIOThread(domain, iothread_id, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
#endif
}


int
virDomainDestroyWrapper(virDomainPtr domain,
                        virErrorPtr err)
{
    int ret = virDomainDestroy(domain);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainDestroyFlagsWrapper(virDomainPtr domain,
                             unsigned int flags,
                             virErrorPtr err)
{
    int ret = virDomainDestroyFlags(domain, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainDetachDeviceWrapper(virDomainPtr domain,
                             const char *xml,
                             virErrorPtr err)
{
    int ret = virDomainDetachDevice(domain, xml);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainDetachDeviceAliasWrapper(virDomainPtr domain,
                                  const char *alias,
                                  unsigned int flags,
                                  virErrorPtr err)
{
#if LIBVIR_VERSION_NUMBER < 4004000
    assert(0); // Caller should have checked version
#else
    int ret = virDomainDetachDeviceAlias(domain, alias, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
#endif
}


int
virDomainDetachDeviceFlagsWrapper(virDomainPtr domain,
                                  const char *xml,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    int ret = virDomainDetachDeviceFlags(domain, xml, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainFSFreezeWrapper(virDomainPtr dom,
                         const char **mountpoints,
                         unsigned int nmountpoints,
                         unsigned int flags,
                         virErrorPtr err)
{
#if LIBVIR_VERSION_NUMBER < 1002005
    assert(0); // Caller should have checked version
#else
    int ret = virDomainFSFreeze(dom, mountpoints, nmountpoints, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
#endif
}


void
virDomainFSInfoFreeWrapper(virDomainFSInfoPtr info)
{
#if LIBVIR_VERSION_NUMBER < 1002011
    assert(0); // Caller should have checked version
#else
    virDomainFSInfoFree(info);
#endif
}


int
virDomainFSThawWrapper(virDomainPtr dom,
                       const char **mountpoints,
                       unsigned int nmountpoints,
                       unsigned int flags,
                       virErrorPtr err)
{
#if LIBVIR_VERSION_NUMBER < 1002005
    assert(0); // Caller should have checked version
#else
    int ret = virDomainFSThaw(dom, mountpoints, nmountpoints, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
#endif
}


int
virDomainFSTrimWrapper(virDomainPtr dom,
                       const char *mountPoint,
                       unsigned long long minimum,
                       unsigned int flags,
                       virErrorPtr err)
{
    int ret = virDomainFSTrim(dom, mountPoint, minimum, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainFreeWrapper(virDomainPtr domain,
                     virErrorPtr err)
{
    int ret = virDomainFree(domain);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainGetAutostartWrapper(virDomainPtr domain,
                             int *autostart,
                             virErrorPtr err)
{
    int ret = virDomainGetAutostart(domain, autostart);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainGetBlkioParametersWrapper(virDomainPtr domain,
                                   virTypedParameterPtr params,
                                   int *nparams,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    int ret = virDomainGetBlkioParameters(domain, params, nparams, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainGetBlockInfoWrapper(virDomainPtr domain,
                             const char *disk,
                             virDomainBlockInfoPtr info,
                             unsigned int flags,
                             virErrorPtr err)
{
    int ret = virDomainGetBlockInfo(domain, disk, info, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainGetBlockIoTuneWrapper(virDomainPtr dom,
                               const char *disk,
                               virTypedParameterPtr params,
                               int *nparams,
                               unsigned int flags,
                               virErrorPtr err)
{
    int ret = virDomainGetBlockIoTune(dom, disk, params, nparams, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainGetBlockJobInfoWrapper(virDomainPtr dom,
                                const char *disk,
                                virDomainBlockJobInfoPtr info,
                                unsigned int flags,
                                virErrorPtr err)
{
    int ret = virDomainGetBlockJobInfo(dom, disk, info, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainGetCPUStatsWrapper(virDomainPtr domain,
                            virTypedParameterPtr params,
                            unsigned int nparams,
                            int start_cpu,
                            unsigned int ncpus,
                            unsigned int flags,
                            virErrorPtr err)
{
    int ret = virDomainGetCPUStats(domain, params, nparams, start_cpu, ncpus, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


virConnectPtr
virDomainGetConnectWrapper(virDomainPtr dom,
                           virErrorPtr err)
{
    virConnectPtr ret = virDomainGetConnect(dom);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainGetControlInfoWrapper(virDomainPtr domain,
                               virDomainControlInfoPtr info,
                               unsigned int flags,
                               virErrorPtr err)
{
    int ret = virDomainGetControlInfo(domain, info, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainGetDiskErrorsWrapper(virDomainPtr dom,
                              virDomainDiskErrorPtr errors,
                              unsigned int maxerrors,
                              unsigned int flags,
                              virErrorPtr err)
{
    int ret = virDomainGetDiskErrors(dom, errors, maxerrors, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainGetEmulatorPinInfoWrapper(virDomainPtr domain,
                                   unsigned char *cpumap,
                                   int maplen,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    int ret = virDomainGetEmulatorPinInfo(domain, cpumap, maplen, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainGetFSInfoWrapper(virDomainPtr dom,
                          virDomainFSInfoPtr **info,
                          unsigned int flags,
                          virErrorPtr err)
{
#if LIBVIR_VERSION_NUMBER < 1002011
    assert(0); // Caller should have checked version
#else
    int ret = virDomainGetFSInfo(dom, info, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
#endif
}


int
virDomainGetGuestVcpusWrapper(virDomainPtr domain,
                              virTypedParameterPtr *params,
                              unsigned int *nparams,
                              unsigned int flags,
                              virErrorPtr err)
{
#if LIBVIR_VERSION_NUMBER < 2000000
    assert(0); // Caller should have checked version
#else
    int ret = virDomainGetGuestVcpus(domain, params, nparams, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
#endif
}


char *
virDomainGetHostnameWrapper(virDomainPtr domain,
                            unsigned int flags,
                            virErrorPtr err)
{
    char * ret = virDomainGetHostname(domain, flags);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


unsigned int
virDomainGetIDWrapper(virDomainPtr domain,
                      virErrorPtr err)
{
    unsigned int ret = virDomainGetID(domain);
    if (ret == (unsigned int)-1) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainGetIOThreadInfoWrapper(virDomainPtr dom,
                                virDomainIOThreadInfoPtr **info,
                                unsigned int flags,
                                virErrorPtr err)
{
#if LIBVIR_VERSION_NUMBER < 1002014
    assert(0); // Caller should have checked version
#else
    int ret = virDomainGetIOThreadInfo(dom, info, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
#endif
}


int
virDomainGetInfoWrapper(virDomainPtr domain,
                        virDomainInfoPtr info,
                        virErrorPtr err)
{
    int ret = virDomainGetInfo(domain, info);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainGetInterfaceParametersWrapper(virDomainPtr domain,
                                       const char *device,
                                       virTypedParameterPtr params,
                                       int *nparams,
                                       unsigned int flags,
                                       virErrorPtr err)
{
    int ret = virDomainGetInterfaceParameters(domain, device, params, nparams, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainGetJobInfoWrapper(virDomainPtr domain,
                           virDomainJobInfoPtr info,
                           virErrorPtr err)
{
    int ret = virDomainGetJobInfo(domain, info);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainGetJobStatsWrapper(virDomainPtr domain,
                            int *type,
                            virTypedParameterPtr *params,
                            int *nparams,
                            unsigned int flags,
                            virErrorPtr err)
{
    int ret = virDomainGetJobStats(domain, type, params, nparams, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainGetLaunchSecurityInfoWrapper(virDomainPtr domain,
                                      virTypedParameterPtr *params,
                                      int *nparams,
                                      unsigned int flags,
                                      virErrorPtr err)
{
#if LIBVIR_VERSION_NUMBER < 4005000
    assert(0); // Caller should have checked version
#else
    int ret = virDomainGetLaunchSecurityInfo(domain, params, nparams, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
#endif
}


unsigned long
virDomainGetMaxMemoryWrapper(virDomainPtr domain,
                             virErrorPtr err)
{
    unsigned long ret = virDomainGetMaxMemory(domain);
    if (ret == 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainGetMaxVcpusWrapper(virDomainPtr domain,
                            virErrorPtr err)
{
    int ret = virDomainGetMaxVcpus(domain);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainGetMemoryParametersWrapper(virDomainPtr domain,
                                    virTypedParameterPtr params,
                                    int *nparams,
                                    unsigned int flags,
                                    virErrorPtr err)
{
    int ret = virDomainGetMemoryParameters(domain, params, nparams, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


char *
virDomainGetMetadataWrapper(virDomainPtr domain,
                            int type,
                            const char *uri,
                            unsigned int flags,
                            virErrorPtr err)
{
    char * ret = virDomainGetMetadata(domain, type, uri, flags);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


const char *
virDomainGetNameWrapper(virDomainPtr domain,
                        virErrorPtr err)
{
    const char * ret = virDomainGetName(domain);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainGetNumaParametersWrapper(virDomainPtr domain,
                                  virTypedParameterPtr params,
                                  int *nparams,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    int ret = virDomainGetNumaParameters(domain, params, nparams, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


char *
virDomainGetOSTypeWrapper(virDomainPtr domain,
                          virErrorPtr err)
{
    char * ret = virDomainGetOSType(domain);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainGetPerfEventsWrapper(virDomainPtr domain,
                              virTypedParameterPtr *params,
                              int *nparams,
                              unsigned int flags,
                              virErrorPtr err)
{
#if LIBVIR_VERSION_NUMBER < 1003003
    assert(0); // Caller should have checked version
#else
    int ret = virDomainGetPerfEvents(domain, params, nparams, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
#endif
}


int
virDomainGetSchedulerParametersWrapper(virDomainPtr domain,
                                       virTypedParameterPtr params,
                                       int *nparams,
                                       virErrorPtr err)
{
    int ret = virDomainGetSchedulerParameters(domain, params, nparams);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainGetSchedulerParametersFlagsWrapper(virDomainPtr domain,
                                            virTypedParameterPtr params,
                                            int *nparams,
                                            unsigned int flags,
                                            virErrorPtr err)
{
    int ret = virDomainGetSchedulerParametersFlags(domain, params, nparams, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


char *
virDomainGetSchedulerTypeWrapper(virDomainPtr domain,
                                 int *nparams,
                                 virErrorPtr err)
{
    char * ret = virDomainGetSchedulerType(domain, nparams);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainGetSecurityLabelWrapper(virDomainPtr domain,
                                 virSecurityLabelPtr seclabel,
                                 virErrorPtr err)
{
    int ret = virDomainGetSecurityLabel(domain, seclabel);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainGetSecurityLabelListWrapper(virDomainPtr domain,
                                     virSecurityLabelPtr *seclabels,
                                     virErrorPtr err)
{
    int ret = virDomainGetSecurityLabelList(domain, seclabels);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainGetStateWrapper(virDomainPtr domain,
                         int *state,
                         int *reason,
                         unsigned int flags,
                         virErrorPtr err)
{
    int ret = virDomainGetState(domain, state, reason, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainGetTimeWrapper(virDomainPtr dom,
                        long long *seconds,
                        unsigned int *nseconds,
                        unsigned int flags,
                        virErrorPtr err)
{
#if LIBVIR_VERSION_NUMBER < 1002005
    assert(0); // Caller should have checked version
#else
    int ret = virDomainGetTime(dom, seconds, nseconds, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
#endif
}


int
virDomainGetUUIDWrapper(virDomainPtr domain,
                        unsigned char *uuid,
                        virErrorPtr err)
{
    int ret = virDomainGetUUID(domain, uuid);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainGetUUIDStringWrapper(virDomainPtr domain,
                              char *buf,
                              virErrorPtr err)
{
    int ret = virDomainGetUUIDString(domain, buf);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainGetVcpuPinInfoWrapper(virDomainPtr domain,
                               int ncpumaps,
                               unsigned char *cpumaps,
                               int maplen,
                               unsigned int flags,
                               virErrorPtr err)
{
    int ret = virDomainGetVcpuPinInfo(domain, ncpumaps, cpumaps, maplen, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainGetVcpusWrapper(virDomainPtr domain,
                         virVcpuInfoPtr info,
                         int maxinfo,
                         unsigned char *cpumaps,
                         int maplen,
                         virErrorPtr err)
{
    int ret = virDomainGetVcpus(domain, info, maxinfo, cpumaps, maplen);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainGetVcpusFlagsWrapper(virDomainPtr domain,
                              unsigned int flags,
                              virErrorPtr err)
{
    int ret = virDomainGetVcpusFlags(domain, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


char *
virDomainGetXMLDescWrapper(virDomainPtr domain,
                           unsigned int flags,
                           virErrorPtr err)
{
    char * ret = virDomainGetXMLDesc(domain, flags);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainHasCurrentSnapshotWrapper(virDomainPtr domain,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    int ret = virDomainHasCurrentSnapshot(domain, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainHasManagedSaveImageWrapper(virDomainPtr dom,
                                    unsigned int flags,
                                    virErrorPtr err)
{
    int ret = virDomainHasManagedSaveImage(dom, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainInjectNMIWrapper(virDomainPtr domain,
                          unsigned int flags,
                          virErrorPtr err)
{
    int ret = virDomainInjectNMI(domain, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainInterfaceAddressesWrapper(virDomainPtr dom,
                                   virDomainInterfacePtr **ifaces,
                                   unsigned int source,
                                   unsigned int flags,
                                   virErrorPtr err)
{
#if LIBVIR_VERSION_NUMBER < 1002014
    assert(0); // Caller should have checked version
#else
    int ret = virDomainInterfaceAddresses(dom, ifaces, source, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
#endif
}


void
virDomainInterfaceFreeWrapper(virDomainInterfacePtr iface)
{
#if LIBVIR_VERSION_NUMBER < 1002014
    assert(0); // Caller should have checked version
#else
    virDomainInterfaceFree(iface);
#endif
}


int
virDomainInterfaceStatsWrapper(virDomainPtr dom,
                               const char *device,
                               virDomainInterfaceStatsPtr stats,
                               size_t size,
                               virErrorPtr err)
{
    int ret = virDomainInterfaceStats(dom, device, stats, size);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


void
virDomainIOThreadInfoFreeWrapper(virDomainIOThreadInfoPtr info)
{
#if LIBVIR_VERSION_NUMBER < 1002014
    assert(0); // Caller should have checked version
#else
    virDomainIOThreadInfoFree(info);
#endif
}


int
virDomainIsActiveWrapper(virDomainPtr dom,
                         virErrorPtr err)
{
    int ret = virDomainIsActive(dom);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainIsPersistentWrapper(virDomainPtr dom,
                             virErrorPtr err)
{
    int ret = virDomainIsPersistent(dom);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainIsUpdatedWrapper(virDomainPtr dom,
                          virErrorPtr err)
{
    int ret = virDomainIsUpdated(dom);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainListAllSnapshotsWrapper(virDomainPtr domain,
                                 virDomainSnapshotPtr **snaps,
                                 unsigned int flags,
                                 virErrorPtr err)
{
    int ret = virDomainListAllSnapshots(domain, snaps, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainManagedSaveWrapper(virDomainPtr dom,
                            unsigned int flags,
                            virErrorPtr err)
{
    int ret = virDomainManagedSave(dom, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainManagedSaveDefineXMLWrapper(virDomainPtr domain,
                                     const char *dxml,
                                     unsigned int flags,
                                     virErrorPtr err)
{
#if LIBVIR_VERSION_NUMBER < 3007000
    assert(0); // Caller should have checked version
#else
    int ret = virDomainManagedSaveDefineXML(domain, dxml, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
#endif
}


char *
virDomainManagedSaveGetXMLDescWrapper(virDomainPtr domain,
                                      unsigned int flags,
                                      virErrorPtr err)
{
#if LIBVIR_VERSION_NUMBER < 3007000
    assert(0); // Caller should have checked version
#else
    char * ret = virDomainManagedSaveGetXMLDesc(domain, flags);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
#endif
}


int
virDomainManagedSaveRemoveWrapper(virDomainPtr dom,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    int ret = virDomainManagedSaveRemove(dom, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainMemoryPeekWrapper(virDomainPtr dom,
                           unsigned long long start,
                           size_t size,
                           void *buffer,
                           unsigned int flags,
                           virErrorPtr err)
{
    int ret = virDomainMemoryPeek(dom, start, size, buffer, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainMemoryStatsWrapper(virDomainPtr dom,
                            virDomainMemoryStatPtr stats,
                            unsigned int nr_stats,
                            unsigned int flags,
                            virErrorPtr err)
{
    int ret = virDomainMemoryStats(dom, stats, nr_stats, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


virDomainPtr
virDomainMigrateWrapper(virDomainPtr domain,
                        virConnectPtr dconn,
                        unsigned long flags,
                        const char *dname,
                        const char *uri,
                        unsigned long bandwidth,
                        virErrorPtr err)
{
    virDomainPtr ret = virDomainMigrate(domain, dconn, flags, dname, uri, bandwidth);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


virDomainPtr
virDomainMigrate2Wrapper(virDomainPtr domain,
                         virConnectPtr dconn,
                         const char *dxml,
                         unsigned long flags,
                         const char *dname,
                         const char *uri,
                         unsigned long bandwidth,
                         virErrorPtr err)
{
    virDomainPtr ret = virDomainMigrate2(domain, dconn, dxml, flags, dname, uri, bandwidth);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


virDomainPtr
virDomainMigrate3Wrapper(virDomainPtr domain,
                         virConnectPtr dconn,
                         virTypedParameterPtr params,
                         unsigned int nparams,
                         unsigned int flags,
                         virErrorPtr err)
{
    virDomainPtr ret = virDomainMigrate3(domain, dconn, params, nparams, flags);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainMigrateGetCompressionCacheWrapper(virDomainPtr domain,
                                           unsigned long long *cacheSize,
                                           unsigned int flags,
                                           virErrorPtr err)
{
    int ret = virDomainMigrateGetCompressionCache(domain, cacheSize, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainMigrateGetMaxDowntimeWrapper(virDomainPtr domain,
                                      unsigned long long *downtime,
                                      unsigned int flags,
                                      virErrorPtr err)
{
#if LIBVIR_VERSION_NUMBER < 3007000
    assert(0); // Caller should have checked version
#else
    int ret = virDomainMigrateGetMaxDowntime(domain, downtime, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
#endif
}


int
virDomainMigrateGetMaxSpeedWrapper(virDomainPtr domain,
                                   unsigned long *bandwidth,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    int ret = virDomainMigrateGetMaxSpeed(domain, bandwidth, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainMigrateSetCompressionCacheWrapper(virDomainPtr domain,
                                           unsigned long long cacheSize,
                                           unsigned int flags,
                                           virErrorPtr err)
{
    int ret = virDomainMigrateSetCompressionCache(domain, cacheSize, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainMigrateSetMaxDowntimeWrapper(virDomainPtr domain,
                                      unsigned long long downtime,
                                      unsigned int flags,
                                      virErrorPtr err)
{
    int ret = virDomainMigrateSetMaxDowntime(domain, downtime, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainMigrateSetMaxSpeedWrapper(virDomainPtr domain,
                                   unsigned long bandwidth,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    int ret = virDomainMigrateSetMaxSpeed(domain, bandwidth, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainMigrateStartPostCopyWrapper(virDomainPtr domain,
                                     unsigned int flags,
                                     virErrorPtr err)
{
#if LIBVIR_VERSION_NUMBER < 1003003
    assert(0); // Caller should have checked version
#else
    int ret = virDomainMigrateStartPostCopy(domain, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
#endif
}


int
virDomainMigrateToURIWrapper(virDomainPtr domain,
                             const char *duri,
                             unsigned long flags,
                             const char *dname,
                             unsigned long bandwidth,
                             virErrorPtr err)
{
    int ret = virDomainMigrateToURI(domain, duri, flags, dname, bandwidth);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainMigrateToURI2Wrapper(virDomainPtr domain,
                              const char *dconnuri,
                              const char *miguri,
                              const char *dxml,
                              unsigned long flags,
                              const char *dname,
                              unsigned long bandwidth,
                              virErrorPtr err)
{
    int ret = virDomainMigrateToURI2(domain, dconnuri, miguri, dxml, flags, dname, bandwidth);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainMigrateToURI3Wrapper(virDomainPtr domain,
                              const char *dconnuri,
                              virTypedParameterPtr params,
                              unsigned int nparams,
                              unsigned int flags,
                              virErrorPtr err)
{
    int ret = virDomainMigrateToURI3(domain, dconnuri, params, nparams, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainOpenChannelWrapper(virDomainPtr dom,
                            const char *name,
                            virStreamPtr st,
                            unsigned int flags,
                            virErrorPtr err)
{
    int ret = virDomainOpenChannel(dom, name, st, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainOpenConsoleWrapper(virDomainPtr dom,
                            const char *dev_name,
                            virStreamPtr st,
                            unsigned int flags,
                            virErrorPtr err)
{
    int ret = virDomainOpenConsole(dom, dev_name, st, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainOpenGraphicsWrapper(virDomainPtr dom,
                             unsigned int idx,
                             int fd,
                             unsigned int flags,
                             virErrorPtr err)
{
    int ret = virDomainOpenGraphics(dom, idx, fd, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainOpenGraphicsFDWrapper(virDomainPtr dom,
                               unsigned int idx,
                               unsigned int flags,
                               virErrorPtr err)
{
#if LIBVIR_VERSION_NUMBER < 1002008
    assert(0); // Caller should have checked version
#else
    int ret = virDomainOpenGraphicsFD(dom, idx, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
#endif
}


int
virDomainPMSuspendForDurationWrapper(virDomainPtr dom,
                                     unsigned int target,
                                     unsigned long long duration,
                                     unsigned int flags,
                                     virErrorPtr err)
{
    int ret = virDomainPMSuspendForDuration(dom, target, duration, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainPMWakeupWrapper(virDomainPtr dom,
                         unsigned int flags,
                         virErrorPtr err)
{
    int ret = virDomainPMWakeup(dom, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainPinEmulatorWrapper(virDomainPtr domain,
                            unsigned char *cpumap,
                            int maplen,
                            unsigned int flags,
                            virErrorPtr err)
{
    int ret = virDomainPinEmulator(domain, cpumap, maplen, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainPinIOThreadWrapper(virDomainPtr domain,
                            unsigned int iothread_id,
                            unsigned char *cpumap,
                            int maplen,
                            unsigned int flags,
                            virErrorPtr err)
{
#if LIBVIR_VERSION_NUMBER < 1002014
    assert(0); // Caller should have checked version
#else
    int ret = virDomainPinIOThread(domain, iothread_id, cpumap, maplen, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
#endif
}


int
virDomainPinVcpuWrapper(virDomainPtr domain,
                        unsigned int vcpu,
                        unsigned char *cpumap,
                        int maplen,
                        virErrorPtr err)
{
    int ret = virDomainPinVcpu(domain, vcpu, cpumap, maplen);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainPinVcpuFlagsWrapper(virDomainPtr domain,
                             unsigned int vcpu,
                             unsigned char *cpumap,
                             int maplen,
                             unsigned int flags,
                             virErrorPtr err)
{
    int ret = virDomainPinVcpuFlags(domain, vcpu, cpumap, maplen, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainRebootWrapper(virDomainPtr domain,
                       unsigned int flags,
                       virErrorPtr err)
{
    int ret = virDomainReboot(domain, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainRefWrapper(virDomainPtr domain,
                    virErrorPtr err)
{
    int ret = virDomainRef(domain);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainRenameWrapper(virDomainPtr dom,
                       const char *new_name,
                       unsigned int flags,
                       virErrorPtr err)
{
#if LIBVIR_VERSION_NUMBER < 1002019
    assert(0); // Caller should have checked version
#else
    int ret = virDomainRename(dom, new_name, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
#endif
}


int
virDomainResetWrapper(virDomainPtr domain,
                      unsigned int flags,
                      virErrorPtr err)
{
    int ret = virDomainReset(domain, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainResumeWrapper(virDomainPtr domain,
                       virErrorPtr err)
{
    int ret = virDomainResume(domain);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainSaveWrapper(virDomainPtr domain,
                     const char *to,
                     virErrorPtr err)
{
    int ret = virDomainSave(domain, to);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainSaveFlagsWrapper(virDomainPtr domain,
                          const char *to,
                          const char *dxml,
                          unsigned int flags,
                          virErrorPtr err)
{
    int ret = virDomainSaveFlags(domain, to, dxml, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


char *
virDomainScreenshotWrapper(virDomainPtr domain,
                           virStreamPtr stream,
                           unsigned int screen,
                           unsigned int flags,
                           virErrorPtr err)
{
    char * ret = virDomainScreenshot(domain, stream, screen, flags);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainSendKeyWrapper(virDomainPtr domain,
                        unsigned int codeset,
                        unsigned int holdtime,
                        unsigned int *keycodes,
                        int nkeycodes,
                        unsigned int flags,
                        virErrorPtr err)
{
    int ret = virDomainSendKey(domain, codeset, holdtime, keycodes, nkeycodes, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainSendProcessSignalWrapper(virDomainPtr domain,
                                  long long pid_value,
                                  unsigned int signum,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    int ret = virDomainSendProcessSignal(domain, pid_value, signum, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainSetAutostartWrapper(virDomainPtr domain,
                             int autostart,
                             virErrorPtr err)
{
    int ret = virDomainSetAutostart(domain, autostart);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainSetBlkioParametersWrapper(virDomainPtr domain,
                                   virTypedParameterPtr params,
                                   int nparams,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    int ret = virDomainSetBlkioParameters(domain, params, nparams, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainSetBlockIoTuneWrapper(virDomainPtr dom,
                               const char *disk,
                               virTypedParameterPtr params,
                               int nparams,
                               unsigned int flags,
                               virErrorPtr err)
{
    int ret = virDomainSetBlockIoTune(dom, disk, params, nparams, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainSetBlockThresholdWrapper(virDomainPtr domain,
                                  const char *dev,
                                  unsigned long long threshold,
                                  unsigned int flags,
                                  virErrorPtr err)
{
#if LIBVIR_VERSION_NUMBER < 3002000
    assert(0); // Caller should have checked version
#else
    int ret = virDomainSetBlockThreshold(domain, dev, threshold, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
#endif
}


int
virDomainSetGuestVcpusWrapper(virDomainPtr domain,
                              const char *cpumap,
                              int state,
                              unsigned int flags,
                              virErrorPtr err)
{
#if LIBVIR_VERSION_NUMBER < 2000000
    assert(0); // Caller should have checked version
#else
    int ret = virDomainSetGuestVcpus(domain, cpumap, state, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
#endif
}


int
virDomainSetInterfaceParametersWrapper(virDomainPtr domain,
                                       const char *device,
                                       virTypedParameterPtr params,
                                       int nparams,
                                       unsigned int flags,
                                       virErrorPtr err)
{
    int ret = virDomainSetInterfaceParameters(domain, device, params, nparams, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainSetLifecycleActionWrapper(virDomainPtr domain,
                                   unsigned int type,
                                   unsigned int action,
                                   unsigned int flags,
                                   virErrorPtr err)
{
#if LIBVIR_VERSION_NUMBER < 3009000
    assert(0); // Caller should have checked version
#else
    int ret = virDomainSetLifecycleAction(domain, type, action, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
#endif
}


int
virDomainSetMaxMemoryWrapper(virDomainPtr domain,
                             unsigned long memory,
                             virErrorPtr err)
{
    int ret = virDomainSetMaxMemory(domain, memory);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainSetMemoryWrapper(virDomainPtr domain,
                          unsigned long memory,
                          virErrorPtr err)
{
    int ret = virDomainSetMemory(domain, memory);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainSetMemoryFlagsWrapper(virDomainPtr domain,
                               unsigned long memory,
                               unsigned int flags,
                               virErrorPtr err)
{
    int ret = virDomainSetMemoryFlags(domain, memory, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainSetMemoryParametersWrapper(virDomainPtr domain,
                                    virTypedParameterPtr params,
                                    int nparams,
                                    unsigned int flags,
                                    virErrorPtr err)
{
    int ret = virDomainSetMemoryParameters(domain, params, nparams, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainSetMemoryStatsPeriodWrapper(virDomainPtr domain,
                                     int period,
                                     unsigned int flags,
                                     virErrorPtr err)
{
    int ret = virDomainSetMemoryStatsPeriod(domain, period, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainSetMetadataWrapper(virDomainPtr domain,
                            int type,
                            const char *metadata,
                            const char *key,
                            const char *uri,
                            unsigned int flags,
                            virErrorPtr err)
{
    int ret = virDomainSetMetadata(domain, type, metadata, key, uri, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainSetNumaParametersWrapper(virDomainPtr domain,
                                  virTypedParameterPtr params,
                                  int nparams,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    int ret = virDomainSetNumaParameters(domain, params, nparams, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainSetPerfEventsWrapper(virDomainPtr domain,
                              virTypedParameterPtr params,
                              int nparams,
                              unsigned int flags,
                              virErrorPtr err)
{
#if LIBVIR_VERSION_NUMBER < 1003003
    assert(0); // Caller should have checked version
#else
    int ret = virDomainSetPerfEvents(domain, params, nparams, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
#endif
}


int
virDomainSetSchedulerParametersWrapper(virDomainPtr domain,
                                       virTypedParameterPtr params,
                                       int nparams,
                                       virErrorPtr err)
{
    int ret = virDomainSetSchedulerParameters(domain, params, nparams);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainSetSchedulerParametersFlagsWrapper(virDomainPtr domain,
                                            virTypedParameterPtr params,
                                            int nparams,
                                            unsigned int flags,
                                            virErrorPtr err)
{
    int ret = virDomainSetSchedulerParametersFlags(domain, params, nparams, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainSetTimeWrapper(virDomainPtr dom,
                        long long seconds,
                        unsigned int nseconds,
                        unsigned int flags,
                        virErrorPtr err)
{
#if LIBVIR_VERSION_NUMBER < 1002005
    assert(0); // Caller should have checked version
#else
    int ret = virDomainSetTime(dom, seconds, nseconds, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
#endif
}


int
virDomainSetUserPasswordWrapper(virDomainPtr dom,
                                const char *user,
                                const char *password,
                                unsigned int flags,
                                virErrorPtr err)
{
#if LIBVIR_VERSION_NUMBER < 1002016
    assert(0); // Caller should have checked version
#else
    int ret = virDomainSetUserPassword(dom, user, password, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
#endif
}


int
virDomainSetVcpuWrapper(virDomainPtr domain,
                        const char *vcpumap,
                        int state,
                        unsigned int flags,
                        virErrorPtr err)
{
#if LIBVIR_VERSION_NUMBER < 3001000
    assert(0); // Caller should have checked version
#else
    int ret = virDomainSetVcpu(domain, vcpumap, state, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
#endif
}


int
virDomainSetVcpusWrapper(virDomainPtr domain,
                         unsigned int nvcpus,
                         virErrorPtr err)
{
    int ret = virDomainSetVcpus(domain, nvcpus);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainSetVcpusFlagsWrapper(virDomainPtr domain,
                              unsigned int nvcpus,
                              unsigned int flags,
                              virErrorPtr err)
{
    int ret = virDomainSetVcpusFlags(domain, nvcpus, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainShutdownWrapper(virDomainPtr domain,
                         virErrorPtr err)
{
    int ret = virDomainShutdown(domain);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainShutdownFlagsWrapper(virDomainPtr domain,
                              unsigned int flags,
                              virErrorPtr err)
{
    int ret = virDomainShutdownFlags(domain, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


virDomainSnapshotPtr
virDomainSnapshotCreateXMLWrapper(virDomainPtr domain,
                                  const char *xmlDesc,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    virDomainSnapshotPtr ret = virDomainSnapshotCreateXML(domain, xmlDesc, flags);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


virDomainSnapshotPtr
virDomainSnapshotCurrentWrapper(virDomainPtr domain,
                                unsigned int flags,
                                virErrorPtr err)
{
    virDomainSnapshotPtr ret = virDomainSnapshotCurrent(domain, flags);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainSnapshotListNamesWrapper(virDomainPtr domain,
                                  char **names,
                                  int nameslen,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    int ret = virDomainSnapshotListNames(domain, names, nameslen, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


virDomainSnapshotPtr
virDomainSnapshotLookupByNameWrapper(virDomainPtr domain,
                                     const char *name,
                                     unsigned int flags,
                                     virErrorPtr err)
{
    virDomainSnapshotPtr ret = virDomainSnapshotLookupByName(domain, name, flags);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainSnapshotNumWrapper(virDomainPtr domain,
                            unsigned int flags,
                            virErrorPtr err)
{
    int ret = virDomainSnapshotNum(domain, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainSuspendWrapper(virDomainPtr domain,
                        virErrorPtr err)
{
    int ret = virDomainSuspend(domain);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainUndefineWrapper(virDomainPtr domain,
                         virErrorPtr err)
{
    int ret = virDomainUndefine(domain);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainUndefineFlagsWrapper(virDomainPtr domain,
                              unsigned int flags,
                              virErrorPtr err)
{
    int ret = virDomainUndefineFlags(domain, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainUpdateDeviceFlagsWrapper(virDomainPtr domain,
                                  const char *xml,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    int ret = virDomainUpdateDeviceFlags(domain, xml, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


*/
import "C"
