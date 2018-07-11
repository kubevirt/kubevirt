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

#ifndef LIBVIRT_GO_DOMAIN_WRAPPER_H__
#define LIBVIRT_GO_DOMAIN_WRAPPER_H__

#include <libvirt/libvirt.h>
#include <libvirt/virterror.h>
#include "domain_compat.h"

int
virDomainAbortJobWrapper(virDomainPtr domain,
                         virErrorPtr err);

int
virDomainAddIOThreadWrapper(virDomainPtr domain,
                            unsigned int iothread_id,
                            unsigned int flags,
                            virErrorPtr err);

int
virDomainAttachDeviceWrapper(virDomainPtr domain,
                             const char *xml,
                             virErrorPtr err);

int
virDomainAttachDeviceFlagsWrapper(virDomainPtr domain,
                                  const char *xml,
                                  unsigned int flags,
                                  virErrorPtr err);

int
virDomainBlockCommitWrapper(virDomainPtr dom,
                            const char *disk,
                            const char *base,
                            const char *top,
                            unsigned long bandwidth,
                            unsigned int flags,
                            virErrorPtr err);

int
virDomainBlockCopyWrapper(virDomainPtr dom,
                          const char *disk,
                          const char *destxml,
                          virTypedParameterPtr params,
                          int nparams,
                          unsigned int flags,
                          virErrorPtr err);

int
virDomainBlockJobAbortWrapper(virDomainPtr dom,
                              const char *disk,
                              unsigned int flags,
                              virErrorPtr err);

int
virDomainBlockJobSetSpeedWrapper(virDomainPtr dom,
                                 const char *disk,
                                 unsigned long bandwidth,
                                 unsigned int flags,
                                 virErrorPtr err);

int
virDomainBlockPeekWrapper(virDomainPtr dom,
                          const char *disk,
                          unsigned long long offset,
                          size_t size,
                          void *buffer,
                          unsigned int flags,
                          virErrorPtr err);

int
virDomainBlockPullWrapper(virDomainPtr dom,
                          const char *disk,
                          unsigned long bandwidth,
                          unsigned int flags,
                          virErrorPtr err);

int
virDomainBlockRebaseWrapper(virDomainPtr dom,
                            const char *disk,
                            const char *base,
                            unsigned long bandwidth,
                            unsigned int flags,
                            virErrorPtr err);

int
virDomainBlockResizeWrapper(virDomainPtr dom,
                            const char *disk,
                            unsigned long long size,
                            unsigned int flags,
                            virErrorPtr err);

int
virDomainBlockStatsWrapper(virDomainPtr dom,
                           const char *disk,
                           virDomainBlockStatsPtr stats,
                           size_t size,
                           virErrorPtr err);

int
virDomainBlockStatsFlagsWrapper(virDomainPtr dom,
                                const char *disk,
                                virTypedParameterPtr params,
                                int *nparams,
                                unsigned int flags,
                                virErrorPtr err);

int
virDomainCoreDumpWrapper(virDomainPtr domain,
                         const char *to,
                         unsigned int flags,
                         virErrorPtr err);

int
virDomainCoreDumpWithFormatWrapper(virDomainPtr domain,
                                   const char *to,
                                   unsigned int dumpformat,
                                   unsigned int flags,
                                   virErrorPtr err);

int
virDomainCreateWrapper(virDomainPtr domain,
                       virErrorPtr err);

int
virDomainCreateWithFilesWrapper(virDomainPtr domain,
                                unsigned int nfiles,
                                int *files,
                                unsigned int flags,
                                virErrorPtr err);

int
virDomainCreateWithFlagsWrapper(virDomainPtr domain,
                                unsigned int flags,
                                virErrorPtr err);

int
virDomainDelIOThreadWrapper(virDomainPtr domain,
                            unsigned int iothread_id,
                            unsigned int flags,
                            virErrorPtr err);

int
virDomainDestroyWrapper(virDomainPtr domain,
                        virErrorPtr err);

int
virDomainDestroyFlagsWrapper(virDomainPtr domain,
                             unsigned int flags,
                             virErrorPtr err);

int
virDomainDetachDeviceWrapper(virDomainPtr domain,
                             const char *xml,
                             virErrorPtr err);

int
virDomainDetachDeviceAliasWrapper(virDomainPtr domain,
                                  const char *alias,
                                  unsigned int flags,
                                  virErrorPtr err);

int
virDomainDetachDeviceFlagsWrapper(virDomainPtr domain,
                                  const char *xml,
                                  unsigned int flags,
                                  virErrorPtr err);

int
virDomainFSFreezeWrapper(virDomainPtr dom,
                         const char **mountpoints,
                         unsigned int nmountpoints,
                         unsigned int flags,
                         virErrorPtr err);

void
virDomainFSInfoFreeWrapper(virDomainFSInfoPtr info);

int
virDomainFSThawWrapper(virDomainPtr dom,
                       const char **mountpoints,
                       unsigned int nmountpoints,
                       unsigned int flags,
                       virErrorPtr err);

int
virDomainFSTrimWrapper(virDomainPtr dom,
                       const char *mountPoint,
                       unsigned long long minimum,
                       unsigned int flags,
                       virErrorPtr err);

int
virDomainFreeWrapper(virDomainPtr domain,
                     virErrorPtr err);

int
virDomainGetAutostartWrapper(virDomainPtr domain,
                             int *autostart,
                             virErrorPtr err);

int
virDomainGetBlkioParametersWrapper(virDomainPtr domain,
                                   virTypedParameterPtr params,
                                   int *nparams,
                                   unsigned int flags,
                                   virErrorPtr err);

int
virDomainGetBlockInfoWrapper(virDomainPtr domain,
                             const char *disk,
                             virDomainBlockInfoPtr info,
                             unsigned int flags,
                             virErrorPtr err);

int
virDomainGetBlockIoTuneWrapper(virDomainPtr dom,
                               const char *disk,
                               virTypedParameterPtr params,
                               int *nparams,
                               unsigned int flags,
                               virErrorPtr err);

int
virDomainGetBlockJobInfoWrapper(virDomainPtr dom,
                                const char *disk,
                                virDomainBlockJobInfoPtr info,
                                unsigned int flags,
                                virErrorPtr err);

int
virDomainGetCPUStatsWrapper(virDomainPtr domain,
                            virTypedParameterPtr params,
                            unsigned int nparams,
                            int start_cpu,
                            unsigned int ncpus,
                            unsigned int flags,
                            virErrorPtr err);

virConnectPtr
virDomainGetConnectWrapper(virDomainPtr dom,
                           virErrorPtr err);

int
virDomainGetControlInfoWrapper(virDomainPtr domain,
                               virDomainControlInfoPtr info,
                               unsigned int flags,
                               virErrorPtr err);

int
virDomainGetDiskErrorsWrapper(virDomainPtr dom,
                              virDomainDiskErrorPtr errors,
                              unsigned int maxerrors,
                              unsigned int flags,
                              virErrorPtr err);

int
virDomainGetEmulatorPinInfoWrapper(virDomainPtr domain,
                                   unsigned char *cpumap,
                                   int maplen,
                                   unsigned int flags,
                                   virErrorPtr err);

int
virDomainGetFSInfoWrapper(virDomainPtr dom,
                          virDomainFSInfoPtr **info,
                          unsigned int flags,
                          virErrorPtr err);

int
virDomainGetGuestVcpusWrapper(virDomainPtr domain,
                              virTypedParameterPtr *params,
                              unsigned int *nparams,
                              unsigned int flags,
                              virErrorPtr err);

char *
virDomainGetHostnameWrapper(virDomainPtr domain,
                            unsigned int flags,
                            virErrorPtr err);

unsigned int
virDomainGetIDWrapper(virDomainPtr domain,
                      virErrorPtr err);

int
virDomainGetIOThreadInfoWrapper(virDomainPtr dom,
                                virDomainIOThreadInfoPtr **info,
                                unsigned int flags,
                                virErrorPtr err);

int
virDomainGetInfoWrapper(virDomainPtr domain,
                        virDomainInfoPtr info,
                        virErrorPtr err);

int
virDomainGetInterfaceParametersWrapper(virDomainPtr domain,
                                       const char *device,
                                       virTypedParameterPtr params,
                                       int *nparams,
                                       unsigned int flags,
                                       virErrorPtr err);

int
virDomainGetJobInfoWrapper(virDomainPtr domain,
                           virDomainJobInfoPtr info,
                           virErrorPtr err);

int
virDomainGetJobStatsWrapper(virDomainPtr domain,
                            int *type,
                            virTypedParameterPtr *params,
                            int *nparams,
                            unsigned int flags,
                            virErrorPtr err);

int
virDomainGetLaunchSecurityInfoWrapper(virDomainPtr domain,
                                      virTypedParameterPtr *params,
                                      int *nparams,
                                      unsigned int flags,
                                      virErrorPtr err);

unsigned long
virDomainGetMaxMemoryWrapper(virDomainPtr domain,
                             virErrorPtr err);

int
virDomainGetMaxVcpusWrapper(virDomainPtr domain,
                            virErrorPtr err);

int
virDomainGetMemoryParametersWrapper(virDomainPtr domain,
                                    virTypedParameterPtr params,
                                    int *nparams,
                                    unsigned int flags,
                                    virErrorPtr err);

char *
virDomainGetMetadataWrapper(virDomainPtr domain,
                            int type,
                            const char *uri,
                            unsigned int flags,
                            virErrorPtr err);

const char *
virDomainGetNameWrapper(virDomainPtr domain,
                        virErrorPtr err);

int
virDomainGetNumaParametersWrapper(virDomainPtr domain,
                                  virTypedParameterPtr params,
                                  int *nparams,
                                  unsigned int flags,
                                  virErrorPtr err);

char *
virDomainGetOSTypeWrapper(virDomainPtr domain,
                          virErrorPtr err);

int
virDomainGetPerfEventsWrapper(virDomainPtr domain,
                              virTypedParameterPtr *params,
                              int *nparams,
                              unsigned int flags,
                              virErrorPtr err);

int
virDomainGetSchedulerParametersWrapper(virDomainPtr domain,
                                       virTypedParameterPtr params,
                                       int *nparams,
                                       virErrorPtr err);

int
virDomainGetSchedulerParametersFlagsWrapper(virDomainPtr domain,
                                            virTypedParameterPtr params,
                                            int *nparams,
                                            unsigned int flags,
                                            virErrorPtr err);

char *
virDomainGetSchedulerTypeWrapper(virDomainPtr domain,
                                 int *nparams,
                                 virErrorPtr err);

int
virDomainGetSecurityLabelWrapper(virDomainPtr domain,
                                 virSecurityLabelPtr seclabel,
                                 virErrorPtr err);

int
virDomainGetSecurityLabelListWrapper(virDomainPtr domain,
                                     virSecurityLabelPtr *seclabels,
                                     virErrorPtr err);

int
virDomainGetStateWrapper(virDomainPtr domain,
                         int *state,
                         int *reason,
                         unsigned int flags,
                         virErrorPtr err);

int
virDomainGetTimeWrapper(virDomainPtr dom,
                        long long *seconds,
                        unsigned int *nseconds,
                        unsigned int flags,
                        virErrorPtr err);

int
virDomainGetUUIDWrapper(virDomainPtr domain,
                        unsigned char *uuid,
                        virErrorPtr err);

int
virDomainGetUUIDStringWrapper(virDomainPtr domain,
                              char *buf,
                              virErrorPtr err);

int
virDomainGetVcpuPinInfoWrapper(virDomainPtr domain,
                               int ncpumaps,
                               unsigned char *cpumaps,
                               int maplen,
                               unsigned int flags,
                               virErrorPtr err);

int
virDomainGetVcpusWrapper(virDomainPtr domain,
                         virVcpuInfoPtr info,
                         int maxinfo,
                         unsigned char *cpumaps,
                         int maplen,
                         virErrorPtr err);

int
virDomainGetVcpusFlagsWrapper(virDomainPtr domain,
                              unsigned int flags,
                              virErrorPtr err);

char *
virDomainGetXMLDescWrapper(virDomainPtr domain,
                           unsigned int flags,
                           virErrorPtr err);

int
virDomainHasCurrentSnapshotWrapper(virDomainPtr domain,
                                   unsigned int flags,
                                   virErrorPtr err);

int
virDomainHasManagedSaveImageWrapper(virDomainPtr dom,
                                    unsigned int flags,
                                    virErrorPtr err);

int
virDomainInjectNMIWrapper(virDomainPtr domain,
                          unsigned int flags,
                          virErrorPtr err);

int
virDomainInterfaceAddressesWrapper(virDomainPtr dom,
                                   virDomainInterfacePtr **ifaces,
                                   unsigned int source,
                                   unsigned int flags,
                                   virErrorPtr err);

void
virDomainInterfaceFreeWrapper(virDomainInterfacePtr iface);

int
virDomainInterfaceStatsWrapper(virDomainPtr dom,
                               const char *device,
                               virDomainInterfaceStatsPtr stats,
                               size_t size,
                               virErrorPtr err);

void
virDomainIOThreadInfoFreeWrapper(virDomainIOThreadInfoPtr info);

int
virDomainIsActiveWrapper(virDomainPtr dom,
                         virErrorPtr err);

int
virDomainIsPersistentWrapper(virDomainPtr dom,
                             virErrorPtr err);

int
virDomainIsUpdatedWrapper(virDomainPtr dom,
                          virErrorPtr err);

int
virDomainListAllSnapshotsWrapper(virDomainPtr domain,
                                 virDomainSnapshotPtr **snaps,
                                 unsigned int flags,
                                 virErrorPtr err);

int
virDomainManagedSaveWrapper(virDomainPtr dom,
                            unsigned int flags,
                            virErrorPtr err);

int
virDomainManagedSaveDefineXMLWrapper(virDomainPtr domain,
                                     const char *dxml,
                                     unsigned int flags,
                                     virErrorPtr err);

char *
virDomainManagedSaveGetXMLDescWrapper(virDomainPtr domain,
                                      unsigned int flags,
                                      virErrorPtr err);

int
virDomainManagedSaveRemoveWrapper(virDomainPtr dom,
                                  unsigned int flags,
                                  virErrorPtr err);

int
virDomainMemoryPeekWrapper(virDomainPtr dom,
                           unsigned long long start,
                           size_t size,
                           void *buffer,
                           unsigned int flags,
                           virErrorPtr err);

int
virDomainMemoryStatsWrapper(virDomainPtr dom,
                            virDomainMemoryStatPtr stats,
                            unsigned int nr_stats,
                            unsigned int flags,
                            virErrorPtr err);

virDomainPtr
virDomainMigrateWrapper(virDomainPtr domain,
                        virConnectPtr dconn,
                        unsigned long flags,
                        const char *dname,
                        const char *uri,
                        unsigned long bandwidth,
                        virErrorPtr err);

virDomainPtr
virDomainMigrate2Wrapper(virDomainPtr domain,
                         virConnectPtr dconn,
                         const char *dxml,
                         unsigned long flags,
                         const char *dname,
                         const char *uri,
                         unsigned long bandwidth,
                         virErrorPtr err);

virDomainPtr
virDomainMigrate3Wrapper(virDomainPtr domain,
                         virConnectPtr dconn,
                         virTypedParameterPtr params,
                         unsigned int nparams,
                         unsigned int flags,
                         virErrorPtr err);

int
virDomainMigrateGetCompressionCacheWrapper(virDomainPtr domain,
                                           unsigned long long *cacheSize,
                                           unsigned int flags,
                                           virErrorPtr err);

int
virDomainMigrateGetMaxDowntimeWrapper(virDomainPtr domain,
                                      unsigned long long *downtime,
                                      unsigned int flags,
                                      virErrorPtr err);

int
virDomainMigrateGetMaxSpeedWrapper(virDomainPtr domain,
                                   unsigned long *bandwidth,
                                   unsigned int flags,
                                   virErrorPtr err);

int
virDomainMigrateSetCompressionCacheWrapper(virDomainPtr domain,
                                           unsigned long long cacheSize,
                                           unsigned int flags,
                                           virErrorPtr err);

int
virDomainMigrateSetMaxDowntimeWrapper(virDomainPtr domain,
                                      unsigned long long downtime,
                                      unsigned int flags,
                                      virErrorPtr err);

int
virDomainMigrateSetMaxSpeedWrapper(virDomainPtr domain,
                                   unsigned long bandwidth,
                                   unsigned int flags,
                                   virErrorPtr err);

int
virDomainMigrateStartPostCopyWrapper(virDomainPtr domain,
                                     unsigned int flags,
                                     virErrorPtr err);

int
virDomainMigrateToURIWrapper(virDomainPtr domain,
                             const char *duri,
                             unsigned long flags,
                             const char *dname,
                             unsigned long bandwidth,
                             virErrorPtr err);

int
virDomainMigrateToURI2Wrapper(virDomainPtr domain,
                              const char *dconnuri,
                              const char *miguri,
                              const char *dxml,
                              unsigned long flags,
                              const char *dname,
                              unsigned long bandwidth,
                              virErrorPtr err);

int
virDomainMigrateToURI3Wrapper(virDomainPtr domain,
                              const char *dconnuri,
                              virTypedParameterPtr params,
                              unsigned int nparams,
                              unsigned int flags,
                              virErrorPtr err);

int
virDomainOpenChannelWrapper(virDomainPtr dom,
                            const char *name,
                            virStreamPtr st,
                            unsigned int flags,
                            virErrorPtr err);

int
virDomainOpenConsoleWrapper(virDomainPtr dom,
                            const char *dev_name,
                            virStreamPtr st,
                            unsigned int flags,
                            virErrorPtr err);

int
virDomainOpenGraphicsWrapper(virDomainPtr dom,
                             unsigned int idx,
                             int fd,
                             unsigned int flags,
                             virErrorPtr err);

int
virDomainOpenGraphicsFDWrapper(virDomainPtr dom,
                               unsigned int idx,
                               unsigned int flags,
                               virErrorPtr err);

int
virDomainPMSuspendForDurationWrapper(virDomainPtr dom,
                                     unsigned int target,
                                     unsigned long long duration,
                                     unsigned int flags,
                                     virErrorPtr err);

int
virDomainPMWakeupWrapper(virDomainPtr dom,
                         unsigned int flags,
                         virErrorPtr err);

int
virDomainPinEmulatorWrapper(virDomainPtr domain,
                            unsigned char *cpumap,
                            int maplen,
                            unsigned int flags,
                            virErrorPtr err);

int
virDomainPinIOThreadWrapper(virDomainPtr domain,
                            unsigned int iothread_id,
                            unsigned char *cpumap,
                            int maplen,
                            unsigned int flags,
                            virErrorPtr err);

int
virDomainPinVcpuWrapper(virDomainPtr domain,
                        unsigned int vcpu,
                        unsigned char *cpumap,
                        int maplen,
                        virErrorPtr err);

int
virDomainPinVcpuFlagsWrapper(virDomainPtr domain,
                             unsigned int vcpu,
                             unsigned char *cpumap,
                             int maplen,
                             unsigned int flags,
                             virErrorPtr err);

int
virDomainRebootWrapper(virDomainPtr domain,
                       unsigned int flags,
                       virErrorPtr err);

int
virDomainRefWrapper(virDomainPtr domain,
                    virErrorPtr err);

int
virDomainRenameWrapper(virDomainPtr dom,
                       const char *new_name,
                       unsigned int flags,
                       virErrorPtr err);

int
virDomainResetWrapper(virDomainPtr domain,
                      unsigned int flags,
                      virErrorPtr err);

int
virDomainResumeWrapper(virDomainPtr domain,
                       virErrorPtr err);

int
virDomainSaveWrapper(virDomainPtr domain,
                     const char *to,
                     virErrorPtr err);

int
virDomainSaveFlagsWrapper(virDomainPtr domain,
                          const char *to,
                          const char *dxml,
                          unsigned int flags,
                          virErrorPtr err);

char *
virDomainScreenshotWrapper(virDomainPtr domain,
                           virStreamPtr stream,
                           unsigned int screen,
                           unsigned int flags,
                           virErrorPtr err);

int
virDomainSendKeyWrapper(virDomainPtr domain,
                        unsigned int codeset,
                        unsigned int holdtime,
                        unsigned int *keycodes,
                        int nkeycodes,
                        unsigned int flags,
                        virErrorPtr err);

int
virDomainSendProcessSignalWrapper(virDomainPtr domain,
                                  long long pid_value,
                                  unsigned int signum,
                                  unsigned int flags,
                                  virErrorPtr err);

int
virDomainSetAutostartWrapper(virDomainPtr domain,
                             int autostart,
                             virErrorPtr err);

int
virDomainSetBlkioParametersWrapper(virDomainPtr domain,
                                   virTypedParameterPtr params,
                                   int nparams,
                                   unsigned int flags,
                                   virErrorPtr err);

int
virDomainSetBlockIoTuneWrapper(virDomainPtr dom,
                               const char *disk,
                               virTypedParameterPtr params,
                               int nparams,
                               unsigned int flags,
                               virErrorPtr err);

int
virDomainSetBlockThresholdWrapper(virDomainPtr domain,
                                  const char *dev,
                                  unsigned long long threshold,
                                  unsigned int flags,
                                  virErrorPtr err);

int
virDomainSetGuestVcpusWrapper(virDomainPtr domain,
                              const char *cpumap,
                              int state,
                              unsigned int flags,
                              virErrorPtr err);

int
virDomainSetInterfaceParametersWrapper(virDomainPtr domain,
                                       const char *device,
                                       virTypedParameterPtr params,
                                       int nparams,
                                       unsigned int flags,
                                       virErrorPtr err);

int
virDomainSetLifecycleActionWrapper(virDomainPtr domain,
                                   unsigned int type,
                                   unsigned int action,
                                   unsigned int flags,
                                   virErrorPtr err);

int
virDomainSetMaxMemoryWrapper(virDomainPtr domain,
                             unsigned long memory,
                             virErrorPtr err);

int
virDomainSetMemoryWrapper(virDomainPtr domain,
                          unsigned long memory,
                          virErrorPtr err);

int
virDomainSetMemoryFlagsWrapper(virDomainPtr domain,
                               unsigned long memory,
                               unsigned int flags,
                               virErrorPtr err);

int
virDomainSetMemoryParametersWrapper(virDomainPtr domain,
                                    virTypedParameterPtr params,
                                    int nparams,
                                    unsigned int flags,
                                    virErrorPtr err);

int
virDomainSetMemoryStatsPeriodWrapper(virDomainPtr domain,
                                     int period,
                                     unsigned int flags,
                                     virErrorPtr err);

int
virDomainSetMetadataWrapper(virDomainPtr domain,
                            int type,
                            const char *metadata,
                            const char *key,
                            const char *uri,
                            unsigned int flags,
                            virErrorPtr err);

int
virDomainSetNumaParametersWrapper(virDomainPtr domain,
                                  virTypedParameterPtr params,
                                  int nparams,
                                  unsigned int flags,
                                  virErrorPtr err);

int
virDomainSetPerfEventsWrapper(virDomainPtr domain,
                              virTypedParameterPtr params,
                              int nparams,
                              unsigned int flags,
                              virErrorPtr err);

int
virDomainSetSchedulerParametersWrapper(virDomainPtr domain,
                                       virTypedParameterPtr params,
                                       int nparams,
                                       virErrorPtr err);

int
virDomainSetSchedulerParametersFlagsWrapper(virDomainPtr domain,
                                            virTypedParameterPtr params,
                                            int nparams,
                                            unsigned int flags,
                                            virErrorPtr err);

int
virDomainSetTimeWrapper(virDomainPtr dom,
                        long long seconds,
                        unsigned int nseconds,
                        unsigned int flags,
                        virErrorPtr err);

int
virDomainSetUserPasswordWrapper(virDomainPtr dom,
                                const char *user,
                                const char *password,
                                unsigned int flags,
                                virErrorPtr err);

int
virDomainSetVcpuWrapper(virDomainPtr domain,
                        const char *vcpumap,
                        int state,
                        unsigned int flags,
                        virErrorPtr err);

int
virDomainSetVcpusWrapper(virDomainPtr domain,
                         unsigned int nvcpus,
                         virErrorPtr err);

int
virDomainSetVcpusFlagsWrapper(virDomainPtr domain,
                              unsigned int nvcpus,
                              unsigned int flags,
                              virErrorPtr err);

int
virDomainShutdownWrapper(virDomainPtr domain,
                         virErrorPtr err);

int
virDomainShutdownFlagsWrapper(virDomainPtr domain,
                              unsigned int flags,
                              virErrorPtr err);

virDomainSnapshotPtr
virDomainSnapshotCreateXMLWrapper(virDomainPtr domain,
                                  const char *xmlDesc,
                                  unsigned int flags,
                                  virErrorPtr err);

virDomainSnapshotPtr
virDomainSnapshotCurrentWrapper(virDomainPtr domain,
                                unsigned int flags,
                                virErrorPtr err);

int
virDomainSnapshotListNamesWrapper(virDomainPtr domain,
                                  char **names,
                                  int nameslen,
                                  unsigned int flags,
                                  virErrorPtr err);

virDomainSnapshotPtr
virDomainSnapshotLookupByNameWrapper(virDomainPtr domain,
                                     const char *name,
                                     unsigned int flags,
                                     virErrorPtr err);

int
virDomainSnapshotNumWrapper(virDomainPtr domain,
                            unsigned int flags,
                            virErrorPtr err);

int
virDomainSuspendWrapper(virDomainPtr domain,
                        virErrorPtr err);

int
virDomainUndefineWrapper(virDomainPtr domain,
                         virErrorPtr err);

int
virDomainUndefineFlagsWrapper(virDomainPtr domain,
                              unsigned int flags,
                              virErrorPtr err);

int
virDomainUpdateDeviceFlagsWrapper(virDomainPtr domain,
                                  const char *xml,
                                  unsigned int flags,
                                  virErrorPtr err);


#endif /* LIBVIRT_GO_DOMAIN_WRAPPER_H__ */
