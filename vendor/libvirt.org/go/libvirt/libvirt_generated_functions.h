/*
 * This file is part of the libvirt-go-module project
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
 * Copyright (C) 2022 Red Hat, Inc.
 *
 */
/****************************************************************************
 * THIS CODE HAS BEEN GENERATED. DO NOT CHANGE IT DIRECTLY                  *
 ****************************************************************************/

#pragma once

int
virConnCopyLastErrorWrapper(virConnectPtr conn,
                            virErrorPtr to,
                            virErrorPtr err);

virErrorPtr
virConnGetLastErrorWrapper(virConnectPtr conn,
                           virErrorPtr err);

void
virConnResetLastErrorWrapper(virConnectPtr conn);

void
virConnSetErrorFuncWrapper(virConnectPtr conn,
                           void * userData,
                           virErrorFunc handler);

char *
virConnectBaselineCPUWrapper(virConnectPtr conn,
                             const char ** xmlCPUs,
                             unsigned int ncpus,
                             unsigned int flags,
                             virErrorPtr err);

char *
virConnectBaselineHypervisorCPUWrapper(virConnectPtr conn,
                                       const char * emulator,
                                       const char * arch,
                                       const char * machine,
                                       const char * virttype,
                                       const char ** xmlCPUs,
                                       unsigned int ncpus,
                                       unsigned int flags,
                                       virErrorPtr err);

int
virConnectCloseWrapper(virConnectPtr conn,
                       virErrorPtr err);

int
virConnectCompareCPUWrapper(virConnectPtr conn,
                            const char * xmlDesc,
                            unsigned int flags,
                            virErrorPtr err);

int
virConnectCompareHypervisorCPUWrapper(virConnectPtr conn,
                                      const char * emulator,
                                      const char * arch,
                                      const char * machine,
                                      const char * virttype,
                                      const char * xmlCPU,
                                      unsigned int flags,
                                      virErrorPtr err);

int
virConnectDomainEventDeregisterWrapper(virConnectPtr conn,
                                       virConnectDomainEventCallback cb,
                                       virErrorPtr err);

int
virConnectDomainEventDeregisterAnyWrapper(virConnectPtr conn,
                                          int callbackID,
                                          virErrorPtr err);

int
virConnectDomainEventRegisterWrapper(virConnectPtr conn,
                                     virConnectDomainEventCallback cb,
                                     void * opaque,
                                     virFreeCallback freecb,
                                     virErrorPtr err);

int
virConnectDomainEventRegisterAnyWrapper(virConnectPtr conn,
                                        virDomainPtr dom,
                                        int eventID,
                                        virConnectDomainEventGenericCallback cb,
                                        void * opaque,
                                        virFreeCallback freecb,
                                        virErrorPtr err);

char *
virConnectDomainXMLFromNativeWrapper(virConnectPtr conn,
                                     const char * nativeFormat,
                                     const char * nativeConfig,
                                     unsigned int flags,
                                     virErrorPtr err);

char *
virConnectDomainXMLToNativeWrapper(virConnectPtr conn,
                                   const char * nativeFormat,
                                   const char * domainXml,
                                   unsigned int flags,
                                   virErrorPtr err);

char *
virConnectFindStoragePoolSourcesWrapper(virConnectPtr conn,
                                        const char * type,
                                        const char * srcSpec,
                                        unsigned int flags,
                                        virErrorPtr err);

int
virConnectGetAllDomainStatsWrapper(virConnectPtr conn,
                                   unsigned int stats,
                                   virDomainStatsRecordPtr ** retStats,
                                   unsigned int flags,
                                   virErrorPtr err);

int
virConnectGetCPUModelNamesWrapper(virConnectPtr conn,
                                  const char * arch,
                                  char *** models,
                                  unsigned int flags,
                                  virErrorPtr err);

char *
virConnectGetCapabilitiesWrapper(virConnectPtr conn,
                                 virErrorPtr err);

char *
virConnectGetDomainCapabilitiesWrapper(virConnectPtr conn,
                                       const char * emulatorbin,
                                       const char * arch,
                                       const char * machine,
                                       const char * virttype,
                                       unsigned int flags,
                                       virErrorPtr err);

char *
virConnectGetHostnameWrapper(virConnectPtr conn,
                             virErrorPtr err);

int
virConnectGetLibVersionWrapper(virConnectPtr conn,
                               unsigned long * libVer,
                               virErrorPtr err);

int
virConnectGetMaxVcpusWrapper(virConnectPtr conn,
                             const char * type,
                             virErrorPtr err);

char *
virConnectGetStoragePoolCapabilitiesWrapper(virConnectPtr conn,
                                            unsigned int flags,
                                            virErrorPtr err);

char *
virConnectGetSysinfoWrapper(virConnectPtr conn,
                            unsigned int flags,
                            virErrorPtr err);

const char *
virConnectGetTypeWrapper(virConnectPtr conn,
                         virErrorPtr err);

char *
virConnectGetURIWrapper(virConnectPtr conn,
                        virErrorPtr err);

int
virConnectGetVersionWrapper(virConnectPtr conn,
                            unsigned long * hvVer,
                            virErrorPtr err);

int
virConnectIsAliveWrapper(virConnectPtr conn,
                         virErrorPtr err);

int
virConnectIsEncryptedWrapper(virConnectPtr conn,
                             virErrorPtr err);

int
virConnectIsSecureWrapper(virConnectPtr conn,
                          virErrorPtr err);

int
virConnectListAllDomainsWrapper(virConnectPtr conn,
                                virDomainPtr ** domains,
                                unsigned int flags,
                                virErrorPtr err);

int
virConnectListAllInterfacesWrapper(virConnectPtr conn,
                                   virInterfacePtr ** ifaces,
                                   unsigned int flags,
                                   virErrorPtr err);

int
virConnectListAllNWFilterBindingsWrapper(virConnectPtr conn,
                                         virNWFilterBindingPtr ** bindings,
                                         unsigned int flags,
                                         virErrorPtr err);

int
virConnectListAllNWFiltersWrapper(virConnectPtr conn,
                                  virNWFilterPtr ** filters,
                                  unsigned int flags,
                                  virErrorPtr err);

int
virConnectListAllNetworksWrapper(virConnectPtr conn,
                                 virNetworkPtr ** nets,
                                 unsigned int flags,
                                 virErrorPtr err);

int
virConnectListAllNodeDevicesWrapper(virConnectPtr conn,
                                    virNodeDevicePtr ** devices,
                                    unsigned int flags,
                                    virErrorPtr err);

int
virConnectListAllSecretsWrapper(virConnectPtr conn,
                                virSecretPtr ** secrets,
                                unsigned int flags,
                                virErrorPtr err);

int
virConnectListAllStoragePoolsWrapper(virConnectPtr conn,
                                     virStoragePoolPtr ** pools,
                                     unsigned int flags,
                                     virErrorPtr err);

int
virConnectListDefinedDomainsWrapper(virConnectPtr conn,
                                    char ** const names,
                                    int maxnames,
                                    virErrorPtr err);

int
virConnectListDefinedInterfacesWrapper(virConnectPtr conn,
                                       char ** const names,
                                       int maxnames,
                                       virErrorPtr err);

int
virConnectListDefinedNetworksWrapper(virConnectPtr conn,
                                     char ** const names,
                                     int maxnames,
                                     virErrorPtr err);

int
virConnectListDefinedStoragePoolsWrapper(virConnectPtr conn,
                                         char ** const names,
                                         int maxnames,
                                         virErrorPtr err);

int
virConnectListDomainsWrapper(virConnectPtr conn,
                             int * ids,
                             int maxids,
                             virErrorPtr err);

int
virConnectListInterfacesWrapper(virConnectPtr conn,
                                char ** const names,
                                int maxnames,
                                virErrorPtr err);

int
virConnectListNWFiltersWrapper(virConnectPtr conn,
                               char ** const names,
                               int maxnames,
                               virErrorPtr err);

int
virConnectListNetworksWrapper(virConnectPtr conn,
                              char ** const names,
                              int maxnames,
                              virErrorPtr err);

int
virConnectListSecretsWrapper(virConnectPtr conn,
                             char ** uuids,
                             int maxuuids,
                             virErrorPtr err);

int
virConnectListStoragePoolsWrapper(virConnectPtr conn,
                                  char ** const names,
                                  int maxnames,
                                  virErrorPtr err);

int
virConnectNetworkEventDeregisterAnyWrapper(virConnectPtr conn,
                                           int callbackID,
                                           virErrorPtr err);

int
virConnectNetworkEventRegisterAnyWrapper(virConnectPtr conn,
                                         virNetworkPtr net,
                                         int eventID,
                                         virConnectNetworkEventGenericCallback cb,
                                         void * opaque,
                                         virFreeCallback freecb,
                                         virErrorPtr err);

int
virConnectNodeDeviceEventDeregisterAnyWrapper(virConnectPtr conn,
                                              int callbackID,
                                              virErrorPtr err);

int
virConnectNodeDeviceEventRegisterAnyWrapper(virConnectPtr conn,
                                            virNodeDevicePtr dev,
                                            int eventID,
                                            virConnectNodeDeviceEventGenericCallback cb,
                                            void * opaque,
                                            virFreeCallback freecb,
                                            virErrorPtr err);

int
virConnectNumOfDefinedDomainsWrapper(virConnectPtr conn,
                                     virErrorPtr err);

int
virConnectNumOfDefinedInterfacesWrapper(virConnectPtr conn,
                                        virErrorPtr err);

int
virConnectNumOfDefinedNetworksWrapper(virConnectPtr conn,
                                      virErrorPtr err);

int
virConnectNumOfDefinedStoragePoolsWrapper(virConnectPtr conn,
                                          virErrorPtr err);

int
virConnectNumOfDomainsWrapper(virConnectPtr conn,
                              virErrorPtr err);

int
virConnectNumOfInterfacesWrapper(virConnectPtr conn,
                                 virErrorPtr err);

int
virConnectNumOfNWFiltersWrapper(virConnectPtr conn,
                                virErrorPtr err);

int
virConnectNumOfNetworksWrapper(virConnectPtr conn,
                               virErrorPtr err);

int
virConnectNumOfSecretsWrapper(virConnectPtr conn,
                              virErrorPtr err);

int
virConnectNumOfStoragePoolsWrapper(virConnectPtr conn,
                                   virErrorPtr err);

virConnectPtr
virConnectOpenWrapper(const char * name,
                      virErrorPtr err);

virConnectPtr
virConnectOpenAuthWrapper(const char * name,
                          virConnectAuthPtr auth,
                          unsigned int flags,
                          virErrorPtr err);

virConnectPtr
virConnectOpenReadOnlyWrapper(const char * name,
                              virErrorPtr err);

int
virConnectRefWrapper(virConnectPtr conn,
                     virErrorPtr err);

int
virConnectRegisterCloseCallbackWrapper(virConnectPtr conn,
                                       virConnectCloseFunc cb,
                                       void * opaque,
                                       virFreeCallback freecb,
                                       virErrorPtr err);

int
virConnectSecretEventDeregisterAnyWrapper(virConnectPtr conn,
                                          int callbackID,
                                          virErrorPtr err);

int
virConnectSecretEventRegisterAnyWrapper(virConnectPtr conn,
                                        virSecretPtr secret,
                                        int eventID,
                                        virConnectSecretEventGenericCallback cb,
                                        void * opaque,
                                        virFreeCallback freecb,
                                        virErrorPtr err);

int
virConnectSetIdentityWrapper(virConnectPtr conn,
                             virTypedParameterPtr params,
                             int nparams,
                             unsigned int flags,
                             virErrorPtr err);

int
virConnectSetKeepAliveWrapper(virConnectPtr conn,
                              int interval,
                              unsigned int count,
                              virErrorPtr err);

int
virConnectStoragePoolEventDeregisterAnyWrapper(virConnectPtr conn,
                                               int callbackID,
                                               virErrorPtr err);

int
virConnectStoragePoolEventRegisterAnyWrapper(virConnectPtr conn,
                                             virStoragePoolPtr pool,
                                             int eventID,
                                             virConnectStoragePoolEventGenericCallback cb,
                                             void * opaque,
                                             virFreeCallback freecb,
                                             virErrorPtr err);

int
virConnectUnregisterCloseCallbackWrapper(virConnectPtr conn,
                                         virConnectCloseFunc cb,
                                         virErrorPtr err);

void
virDefaultErrorFuncWrapper(virErrorPtr err);

int
virDomainAbortJobWrapper(virDomainPtr domain,
                         virErrorPtr err);

int
virDomainAbortJobFlagsWrapper(virDomainPtr domain,
                              unsigned int flags,
                              virErrorPtr err);

int
virDomainAddIOThreadWrapper(virDomainPtr domain,
                            unsigned int iothread_id,
                            unsigned int flags,
                            virErrorPtr err);

int
virDomainAgentSetResponseTimeoutWrapper(virDomainPtr domain,
                                        int timeout,
                                        unsigned int flags,
                                        virErrorPtr err);

int
virDomainAttachDeviceWrapper(virDomainPtr domain,
                             const char * xml,
                             virErrorPtr err);

int
virDomainAttachDeviceFlagsWrapper(virDomainPtr domain,
                                  const char * xml,
                                  unsigned int flags,
                                  virErrorPtr err);

int
virDomainAuthorizedSSHKeysGetWrapper(virDomainPtr domain,
                                     const char * user,
                                     char *** keys,
                                     unsigned int flags,
                                     virErrorPtr err);

int
virDomainAuthorizedSSHKeysSetWrapper(virDomainPtr domain,
                                     const char * user,
                                     const char ** keys,
                                     unsigned int nkeys,
                                     unsigned int flags,
                                     virErrorPtr err);

int
virDomainBackupBeginWrapper(virDomainPtr domain,
                            const char * backupXML,
                            const char * checkpointXML,
                            unsigned int flags,
                            virErrorPtr err);

char *
virDomainBackupGetXMLDescWrapper(virDomainPtr domain,
                                 unsigned int flags,
                                 virErrorPtr err);

int
virDomainBlockCommitWrapper(virDomainPtr dom,
                            const char * disk,
                            const char * base,
                            const char * top,
                            unsigned long bandwidth,
                            unsigned int flags,
                            virErrorPtr err);

int
virDomainBlockCopyWrapper(virDomainPtr dom,
                          const char * disk,
                          const char * destxml,
                          virTypedParameterPtr params,
                          int nparams,
                          unsigned int flags,
                          virErrorPtr err);

int
virDomainBlockJobAbortWrapper(virDomainPtr dom,
                              const char * disk,
                              unsigned int flags,
                              virErrorPtr err);

int
virDomainBlockJobSetSpeedWrapper(virDomainPtr dom,
                                 const char * disk,
                                 unsigned long bandwidth,
                                 unsigned int flags,
                                 virErrorPtr err);

int
virDomainBlockPeekWrapper(virDomainPtr dom,
                          const char * disk,
                          unsigned long long offset,
                          size_t size,
                          void * buffer,
                          unsigned int flags,
                          virErrorPtr err);

int
virDomainBlockPullWrapper(virDomainPtr dom,
                          const char * disk,
                          unsigned long bandwidth,
                          unsigned int flags,
                          virErrorPtr err);

int
virDomainBlockRebaseWrapper(virDomainPtr dom,
                            const char * disk,
                            const char * base,
                            unsigned long bandwidth,
                            unsigned int flags,
                            virErrorPtr err);

int
virDomainBlockResizeWrapper(virDomainPtr dom,
                            const char * disk,
                            unsigned long long size,
                            unsigned int flags,
                            virErrorPtr err);

int
virDomainBlockStatsWrapper(virDomainPtr dom,
                           const char * disk,
                           virDomainBlockStatsPtr stats,
                           size_t size,
                           virErrorPtr err);

int
virDomainBlockStatsFlagsWrapper(virDomainPtr dom,
                                const char * disk,
                                virTypedParameterPtr params,
                                int * nparams,
                                unsigned int flags,
                                virErrorPtr err);

virDomainCheckpointPtr
virDomainCheckpointCreateXMLWrapper(virDomainPtr domain,
                                    const char * xmlDesc,
                                    unsigned int flags,
                                    virErrorPtr err);

int
virDomainCheckpointDeleteWrapper(virDomainCheckpointPtr checkpoint,
                                 unsigned int flags,
                                 virErrorPtr err);

int
virDomainCheckpointFreeWrapper(virDomainCheckpointPtr checkpoint,
                               virErrorPtr err);

virConnectPtr
virDomainCheckpointGetConnectWrapper(virDomainCheckpointPtr checkpoint,
                                     virErrorPtr err);

virDomainPtr
virDomainCheckpointGetDomainWrapper(virDomainCheckpointPtr checkpoint,
                                    virErrorPtr err);

const char *
virDomainCheckpointGetNameWrapper(virDomainCheckpointPtr checkpoint,
                                  virErrorPtr err);

virDomainCheckpointPtr
virDomainCheckpointGetParentWrapper(virDomainCheckpointPtr checkpoint,
                                    unsigned int flags,
                                    virErrorPtr err);

char *
virDomainCheckpointGetXMLDescWrapper(virDomainCheckpointPtr checkpoint,
                                     unsigned int flags,
                                     virErrorPtr err);

int
virDomainCheckpointListAllChildrenWrapper(virDomainCheckpointPtr checkpoint,
                                          virDomainCheckpointPtr ** children,
                                          unsigned int flags,
                                          virErrorPtr err);

virDomainCheckpointPtr
virDomainCheckpointLookupByNameWrapper(virDomainPtr domain,
                                       const char * name,
                                       unsigned int flags,
                                       virErrorPtr err);

int
virDomainCheckpointRefWrapper(virDomainCheckpointPtr checkpoint,
                              virErrorPtr err);

int
virDomainCoreDumpWrapper(virDomainPtr domain,
                         const char * to,
                         unsigned int flags,
                         virErrorPtr err);

int
virDomainCoreDumpWithFormatWrapper(virDomainPtr domain,
                                   const char * to,
                                   unsigned int dumpformat,
                                   unsigned int flags,
                                   virErrorPtr err);

int
virDomainCreateWrapper(virDomainPtr domain,
                       virErrorPtr err);

virDomainPtr
virDomainCreateLinuxWrapper(virConnectPtr conn,
                            const char * xmlDesc,
                            unsigned int flags,
                            virErrorPtr err);

int
virDomainCreateWithFilesWrapper(virDomainPtr domain,
                                unsigned int nfiles,
                                int * files,
                                unsigned int flags,
                                virErrorPtr err);

int
virDomainCreateWithFlagsWrapper(virDomainPtr domain,
                                unsigned int flags,
                                virErrorPtr err);

virDomainPtr
virDomainCreateXMLWrapper(virConnectPtr conn,
                          const char * xmlDesc,
                          unsigned int flags,
                          virErrorPtr err);

virDomainPtr
virDomainCreateXMLWithFilesWrapper(virConnectPtr conn,
                                   const char * xmlDesc,
                                   unsigned int nfiles,
                                   int * files,
                                   unsigned int flags,
                                   virErrorPtr err);

virDomainPtr
virDomainDefineXMLWrapper(virConnectPtr conn,
                          const char * xml,
                          virErrorPtr err);

virDomainPtr
virDomainDefineXMLFlagsWrapper(virConnectPtr conn,
                               const char * xml,
                               unsigned int flags,
                               virErrorPtr err);

int
virDomainDelIOThreadWrapper(virDomainPtr domain,
                            unsigned int iothread_id,
                            unsigned int flags,
                            virErrorPtr err);

int
virDomainDelThrottleGroupWrapper(virDomainPtr dom,
                                 const char * group,
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
                             const char * xml,
                             virErrorPtr err);

int
virDomainDetachDeviceAliasWrapper(virDomainPtr domain,
                                  const char * alias,
                                  unsigned int flags,
                                  virErrorPtr err);

int
virDomainDetachDeviceFlagsWrapper(virDomainPtr domain,
                                  const char * xml,
                                  unsigned int flags,
                                  virErrorPtr err);

int
virDomainFDAssociateWrapper(virDomainPtr domain,
                            const char * name,
                            unsigned int nfds,
                            int * fds,
                            unsigned int flags,
                            virErrorPtr err);

int
virDomainFSFreezeWrapper(virDomainPtr dom,
                         const char ** mountpoints,
                         unsigned int nmountpoints,
                         unsigned int flags,
                         virErrorPtr err);

void
virDomainFSInfoFreeWrapper(virDomainFSInfoPtr info);

int
virDomainFSThawWrapper(virDomainPtr dom,
                       const char ** mountpoints,
                       unsigned int nmountpoints,
                       unsigned int flags,
                       virErrorPtr err);

int
virDomainFSTrimWrapper(virDomainPtr dom,
                       const char * mountPoint,
                       unsigned long long minimum,
                       unsigned int flags,
                       virErrorPtr err);

int
virDomainFreeWrapper(virDomainPtr domain,
                     virErrorPtr err);

int
virDomainGetAutostartWrapper(virDomainPtr domain,
                             int * autostart,
                             virErrorPtr err);

int
virDomainGetAutostartOnceWrapper(virDomainPtr domain,
                                 int * autostart,
                                 virErrorPtr err);

int
virDomainGetBlkioParametersWrapper(virDomainPtr domain,
                                   virTypedParameterPtr params,
                                   int * nparams,
                                   unsigned int flags,
                                   virErrorPtr err);

int
virDomainGetBlockInfoWrapper(virDomainPtr domain,
                             const char * disk,
                             virDomainBlockInfoPtr info,
                             unsigned int flags,
                             virErrorPtr err);

int
virDomainGetBlockIoTuneWrapper(virDomainPtr dom,
                               const char * disk,
                               virTypedParameterPtr params,
                               int * nparams,
                               unsigned int flags,
                               virErrorPtr err);

int
virDomainGetBlockJobInfoWrapper(virDomainPtr dom,
                                const char * disk,
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
                                   unsigned char * cpumap,
                                   int maplen,
                                   unsigned int flags,
                                   virErrorPtr err);

int
virDomainGetFSInfoWrapper(virDomainPtr dom,
                          virDomainFSInfoPtr ** info,
                          unsigned int flags,
                          virErrorPtr err);

int
virDomainGetGuestInfoWrapper(virDomainPtr domain,
                             unsigned int types,
                             virTypedParameterPtr * params,
                             int * nparams,
                             unsigned int flags,
                             virErrorPtr err);

int
virDomainGetGuestVcpusWrapper(virDomainPtr domain,
                              virTypedParameterPtr * params,
                              unsigned int * nparams,
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
                                virDomainIOThreadInfoPtr ** info,
                                unsigned int flags,
                                virErrorPtr err);

int
virDomainGetInfoWrapper(virDomainPtr domain,
                        virDomainInfoPtr info,
                        virErrorPtr err);

int
virDomainGetInterfaceParametersWrapper(virDomainPtr domain,
                                       const char * device,
                                       virTypedParameterPtr params,
                                       int * nparams,
                                       unsigned int flags,
                                       virErrorPtr err);

int
virDomainGetJobInfoWrapper(virDomainPtr domain,
                           virDomainJobInfoPtr info,
                           virErrorPtr err);

int
virDomainGetJobStatsWrapper(virDomainPtr domain,
                            int * type,
                            virTypedParameterPtr * params,
                            int * nparams,
                            unsigned int flags,
                            virErrorPtr err);

int
virDomainGetLaunchSecurityInfoWrapper(virDomainPtr domain,
                                      virTypedParameterPtr * params,
                                      int * nparams,
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
                                    int * nparams,
                                    unsigned int flags,
                                    virErrorPtr err);

int
virDomainGetMessagesWrapper(virDomainPtr domain,
                            char *** msgs,
                            unsigned int flags,
                            virErrorPtr err);

char *
virDomainGetMetadataWrapper(virDomainPtr domain,
                            int type,
                            const char * uri,
                            unsigned int flags,
                            virErrorPtr err);

const char *
virDomainGetNameWrapper(virDomainPtr domain,
                        virErrorPtr err);

int
virDomainGetNumaParametersWrapper(virDomainPtr domain,
                                  virTypedParameterPtr params,
                                  int * nparams,
                                  unsigned int flags,
                                  virErrorPtr err);

char *
virDomainGetOSTypeWrapper(virDomainPtr domain,
                          virErrorPtr err);

int
virDomainGetPerfEventsWrapper(virDomainPtr domain,
                              virTypedParameterPtr * params,
                              int * nparams,
                              unsigned int flags,
                              virErrorPtr err);

int
virDomainGetSchedulerParametersWrapper(virDomainPtr domain,
                                       virTypedParameterPtr params,
                                       int * nparams,
                                       virErrorPtr err);

int
virDomainGetSchedulerParametersFlagsWrapper(virDomainPtr domain,
                                            virTypedParameterPtr params,
                                            int * nparams,
                                            unsigned int flags,
                                            virErrorPtr err);

char *
virDomainGetSchedulerTypeWrapper(virDomainPtr domain,
                                 int * nparams,
                                 virErrorPtr err);

int
virDomainGetSecurityLabelWrapper(virDomainPtr domain,
                                 virSecurityLabelPtr seclabel,
                                 virErrorPtr err);

int
virDomainGetSecurityLabelListWrapper(virDomainPtr domain,
                                     virSecurityLabelPtr * seclabels,
                                     virErrorPtr err);

int
virDomainGetStateWrapper(virDomainPtr domain,
                         int * state,
                         int * reason,
                         unsigned int flags,
                         virErrorPtr err);

int
virDomainGetTimeWrapper(virDomainPtr dom,
                        long long * seconds,
                        unsigned int * nseconds,
                        unsigned int flags,
                        virErrorPtr err);

int
virDomainGetUUIDWrapper(virDomainPtr domain,
                        unsigned char * uuid,
                        virErrorPtr err);

int
virDomainGetUUIDStringWrapper(virDomainPtr domain,
                              char * buf,
                              virErrorPtr err);

int
virDomainGetVcpuPinInfoWrapper(virDomainPtr domain,
                               int ncpumaps,
                               unsigned char * cpumaps,
                               int maplen,
                               unsigned int flags,
                               virErrorPtr err);

int
virDomainGetVcpusWrapper(virDomainPtr domain,
                         virVcpuInfoPtr info,
                         int maxinfo,
                         unsigned char * cpumaps,
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
virDomainGraphicsReloadWrapper(virDomainPtr domain,
                               unsigned int type,
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

void
virDomainIOThreadInfoFreeWrapper(virDomainIOThreadInfoPtr info);

int
virDomainInjectNMIWrapper(virDomainPtr domain,
                          unsigned int flags,
                          virErrorPtr err);

int
virDomainInterfaceAddressesWrapper(virDomainPtr dom,
                                   virDomainInterfacePtr ** ifaces,
                                   unsigned int source,
                                   unsigned int flags,
                                   virErrorPtr err);

void
virDomainInterfaceFreeWrapper(virDomainInterfacePtr iface);

int
virDomainInterfaceStatsWrapper(virDomainPtr dom,
                               const char * device,
                               virDomainInterfaceStatsPtr stats,
                               size_t size,
                               virErrorPtr err);

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
virDomainListAllCheckpointsWrapper(virDomainPtr domain,
                                   virDomainCheckpointPtr ** checkpoints,
                                   unsigned int flags,
                                   virErrorPtr err);

int
virDomainListAllSnapshotsWrapper(virDomainPtr domain,
                                 virDomainSnapshotPtr ** snaps,
                                 unsigned int flags,
                                 virErrorPtr err);

int
virDomainListGetStatsWrapper(virDomainPtr * doms,
                             unsigned int stats,
                             virDomainStatsRecordPtr ** retStats,
                             unsigned int flags,
                             virErrorPtr err);

virDomainPtr
virDomainLookupByIDWrapper(virConnectPtr conn,
                           int id,
                           virErrorPtr err);

virDomainPtr
virDomainLookupByNameWrapper(virConnectPtr conn,
                             const char * name,
                             virErrorPtr err);

virDomainPtr
virDomainLookupByUUIDWrapper(virConnectPtr conn,
                             const unsigned char * uuid,
                             virErrorPtr err);

virDomainPtr
virDomainLookupByUUIDStringWrapper(virConnectPtr conn,
                                   const char * uuidstr,
                                   virErrorPtr err);

int
virDomainManagedSaveWrapper(virDomainPtr dom,
                            unsigned int flags,
                            virErrorPtr err);

int
virDomainManagedSaveDefineXMLWrapper(virDomainPtr domain,
                                     const char * dxml,
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
                           void * buffer,
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
                        const char * dname,
                        const char * uri,
                        unsigned long bandwidth,
                        virErrorPtr err);

virDomainPtr
virDomainMigrate2Wrapper(virDomainPtr domain,
                         virConnectPtr dconn,
                         const char * dxml,
                         unsigned long flags,
                         const char * dname,
                         const char * uri,
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
                                           unsigned long long * cacheSize,
                                           unsigned int flags,
                                           virErrorPtr err);

int
virDomainMigrateGetMaxDowntimeWrapper(virDomainPtr domain,
                                      unsigned long long * downtime,
                                      unsigned int flags,
                                      virErrorPtr err);

int
virDomainMigrateGetMaxSpeedWrapper(virDomainPtr domain,
                                   unsigned long * bandwidth,
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
                             const char * duri,
                             unsigned long flags,
                             const char * dname,
                             unsigned long bandwidth,
                             virErrorPtr err);

int
virDomainMigrateToURI2Wrapper(virDomainPtr domain,
                              const char * dconnuri,
                              const char * miguri,
                              const char * dxml,
                              unsigned long flags,
                              const char * dname,
                              unsigned long bandwidth,
                              virErrorPtr err);

int
virDomainMigrateToURI3Wrapper(virDomainPtr domain,
                              const char * dconnuri,
                              virTypedParameterPtr params,
                              unsigned int nparams,
                              unsigned int flags,
                              virErrorPtr err);

int
virDomainOpenChannelWrapper(virDomainPtr dom,
                            const char * name,
                            virStreamPtr st,
                            unsigned int flags,
                            virErrorPtr err);

int
virDomainOpenConsoleWrapper(virDomainPtr dom,
                            const char * dev_name,
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
                            unsigned char * cpumap,
                            int maplen,
                            unsigned int flags,
                            virErrorPtr err);

int
virDomainPinIOThreadWrapper(virDomainPtr domain,
                            unsigned int iothread_id,
                            unsigned char * cpumap,
                            int maplen,
                            unsigned int flags,
                            virErrorPtr err);

int
virDomainPinVcpuWrapper(virDomainPtr domain,
                        unsigned int vcpu,
                        unsigned char * cpumap,
                        int maplen,
                        virErrorPtr err);

int
virDomainPinVcpuFlagsWrapper(virDomainPtr domain,
                             unsigned int vcpu,
                             unsigned char * cpumap,
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
                       const char * new_name,
                       unsigned int flags,
                       virErrorPtr err);

int
virDomainResetWrapper(virDomainPtr domain,
                      unsigned int flags,
                      virErrorPtr err);

int
virDomainRestoreWrapper(virConnectPtr conn,
                        const char * from,
                        virErrorPtr err);

int
virDomainRestoreFlagsWrapper(virConnectPtr conn,
                             const char * from,
                             const char * dxml,
                             unsigned int flags,
                             virErrorPtr err);

int
virDomainRestoreParamsWrapper(virConnectPtr conn,
                              virTypedParameterPtr params,
                              int nparams,
                              unsigned int flags,
                              virErrorPtr err);

int
virDomainResumeWrapper(virDomainPtr domain,
                       virErrorPtr err);

int
virDomainRevertToSnapshotWrapper(virDomainSnapshotPtr snapshot,
                                 unsigned int flags,
                                 virErrorPtr err);

int
virDomainSaveWrapper(virDomainPtr domain,
                     const char * to,
                     virErrorPtr err);

int
virDomainSaveFlagsWrapper(virDomainPtr domain,
                          const char * to,
                          const char * dxml,
                          unsigned int flags,
                          virErrorPtr err);

int
virDomainSaveImageDefineXMLWrapper(virConnectPtr conn,
                                   const char * file,
                                   const char * dxml,
                                   unsigned int flags,
                                   virErrorPtr err);

char *
virDomainSaveImageGetXMLDescWrapper(virConnectPtr conn,
                                    const char * file,
                                    unsigned int flags,
                                    virErrorPtr err);

int
virDomainSaveParamsWrapper(virDomainPtr domain,
                           virTypedParameterPtr params,
                           int nparams,
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
                        unsigned int * keycodes,
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
virDomainSetAutostartOnceWrapper(virDomainPtr domain,
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
                               const char * disk,
                               virTypedParameterPtr params,
                               int nparams,
                               unsigned int flags,
                               virErrorPtr err);

int
virDomainSetBlockThresholdWrapper(virDomainPtr domain,
                                  const char * dev,
                                  unsigned long long threshold,
                                  unsigned int flags,
                                  virErrorPtr err);

int
virDomainSetGuestVcpusWrapper(virDomainPtr domain,
                              const char * cpumap,
                              int state,
                              unsigned int flags,
                              virErrorPtr err);

int
virDomainSetIOThreadParamsWrapper(virDomainPtr domain,
                                  unsigned int iothread_id,
                                  virTypedParameterPtr params,
                                  int nparams,
                                  unsigned int flags,
                                  virErrorPtr err);

int
virDomainSetInterfaceParametersWrapper(virDomainPtr domain,
                                       const char * device,
                                       virTypedParameterPtr params,
                                       int nparams,
                                       unsigned int flags,
                                       virErrorPtr err);

int
virDomainSetLaunchSecurityStateWrapper(virDomainPtr domain,
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
                            const char * metadata,
                            const char * key,
                            const char * uri,
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
virDomainSetThrottleGroupWrapper(virDomainPtr dom,
                                 const char * group,
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
                                const char * user,
                                const char * password,
                                unsigned int flags,
                                virErrorPtr err);

int
virDomainSetVcpuWrapper(virDomainPtr domain,
                        const char * vcpumap,
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
                                  const char * xmlDesc,
                                  unsigned int flags,
                                  virErrorPtr err);

virDomainSnapshotPtr
virDomainSnapshotCurrentWrapper(virDomainPtr domain,
                                unsigned int flags,
                                virErrorPtr err);

int
virDomainSnapshotDeleteWrapper(virDomainSnapshotPtr snapshot,
                               unsigned int flags,
                               virErrorPtr err);

int
virDomainSnapshotFreeWrapper(virDomainSnapshotPtr snapshot,
                             virErrorPtr err);

virConnectPtr
virDomainSnapshotGetConnectWrapper(virDomainSnapshotPtr snapshot,
                                   virErrorPtr err);

virDomainPtr
virDomainSnapshotGetDomainWrapper(virDomainSnapshotPtr snapshot,
                                  virErrorPtr err);

const char *
virDomainSnapshotGetNameWrapper(virDomainSnapshotPtr snapshot,
                                virErrorPtr err);

virDomainSnapshotPtr
virDomainSnapshotGetParentWrapper(virDomainSnapshotPtr snapshot,
                                  unsigned int flags,
                                  virErrorPtr err);

char *
virDomainSnapshotGetXMLDescWrapper(virDomainSnapshotPtr snapshot,
                                   unsigned int flags,
                                   virErrorPtr err);

int
virDomainSnapshotHasMetadataWrapper(virDomainSnapshotPtr snapshot,
                                    unsigned int flags,
                                    virErrorPtr err);

int
virDomainSnapshotIsCurrentWrapper(virDomainSnapshotPtr snapshot,
                                  unsigned int flags,
                                  virErrorPtr err);

int
virDomainSnapshotListAllChildrenWrapper(virDomainSnapshotPtr snapshot,
                                        virDomainSnapshotPtr ** snaps,
                                        unsigned int flags,
                                        virErrorPtr err);

int
virDomainSnapshotListChildrenNamesWrapper(virDomainSnapshotPtr snapshot,
                                          char ** names,
                                          int nameslen,
                                          unsigned int flags,
                                          virErrorPtr err);

int
virDomainSnapshotListNamesWrapper(virDomainPtr domain,
                                  char ** names,
                                  int nameslen,
                                  unsigned int flags,
                                  virErrorPtr err);

virDomainSnapshotPtr
virDomainSnapshotLookupByNameWrapper(virDomainPtr domain,
                                     const char * name,
                                     unsigned int flags,
                                     virErrorPtr err);

int
virDomainSnapshotNumWrapper(virDomainPtr domain,
                            unsigned int flags,
                            virErrorPtr err);

int
virDomainSnapshotNumChildrenWrapper(virDomainSnapshotPtr snapshot,
                                    unsigned int flags,
                                    virErrorPtr err);

int
virDomainSnapshotRefWrapper(virDomainSnapshotPtr snapshot,
                            virErrorPtr err);

int
virDomainStartDirtyRateCalcWrapper(virDomainPtr domain,
                                   int seconds,
                                   unsigned int flags,
                                   virErrorPtr err);

void
virDomainStatsRecordListFreeWrapper(virDomainStatsRecordPtr * stats);

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
                                  const char * xml,
                                  unsigned int flags,
                                  virErrorPtr err);

int
virEventAddHandleWrapper(int fd,
                         int events,
                         virEventHandleCallback cb,
                         void * opaque,
                         virFreeCallback ff,
                         virErrorPtr err);

int
virEventAddTimeoutWrapper(int timeout,
                          virEventTimeoutCallback cb,
                          void * opaque,
                          virFreeCallback ff,
                          virErrorPtr err);

int
virEventRegisterDefaultImplWrapper(virErrorPtr err);

void
virEventRegisterImplWrapper(virEventAddHandleFunc addHandle,
                            virEventUpdateHandleFunc updateHandle,
                            virEventRemoveHandleFunc removeHandle,
                            virEventAddTimeoutFunc addTimeout,
                            virEventUpdateTimeoutFunc updateTimeout,
                            virEventRemoveTimeoutFunc removeTimeout);

int
virEventRemoveHandleWrapper(int watch,
                            virErrorPtr err);

int
virEventRemoveTimeoutWrapper(int timer,
                             virErrorPtr err);

int
virEventRunDefaultImplWrapper(virErrorPtr err);

void
virEventUpdateHandleWrapper(int watch,
                            int events);

void
virEventUpdateTimeoutWrapper(int timer,
                             int timeout);

void
virFreeErrorWrapper(virErrorPtr err);

virErrorPtr
virGetLastErrorWrapper(virErrorPtr err);

int
virGetLastErrorCodeWrapper(virErrorPtr err);

int
virGetLastErrorDomainWrapper(virErrorPtr err);

const char *
virGetLastErrorMessageWrapper(virErrorPtr err);

int
virGetVersionWrapper(unsigned long * libVer,
                     const char * type,
                     unsigned long * typeVer,
                     virErrorPtr err);

int
virInitializeWrapper(virErrorPtr err);

int
virInterfaceChangeBeginWrapper(virConnectPtr conn,
                               unsigned int flags,
                               virErrorPtr err);

int
virInterfaceChangeCommitWrapper(virConnectPtr conn,
                                unsigned int flags,
                                virErrorPtr err);

int
virInterfaceChangeRollbackWrapper(virConnectPtr conn,
                                  unsigned int flags,
                                  virErrorPtr err);

int
virInterfaceCreateWrapper(virInterfacePtr iface,
                          unsigned int flags,
                          virErrorPtr err);

virInterfacePtr
virInterfaceDefineXMLWrapper(virConnectPtr conn,
                             const char * xml,
                             unsigned int flags,
                             virErrorPtr err);

int
virInterfaceDestroyWrapper(virInterfacePtr iface,
                           unsigned int flags,
                           virErrorPtr err);

int
virInterfaceFreeWrapper(virInterfacePtr iface,
                        virErrorPtr err);

virConnectPtr
virInterfaceGetConnectWrapper(virInterfacePtr iface,
                              virErrorPtr err);

const char *
virInterfaceGetMACStringWrapper(virInterfacePtr iface,
                                virErrorPtr err);

const char *
virInterfaceGetNameWrapper(virInterfacePtr iface,
                           virErrorPtr err);

char *
virInterfaceGetXMLDescWrapper(virInterfacePtr iface,
                              unsigned int flags,
                              virErrorPtr err);

int
virInterfaceIsActiveWrapper(virInterfacePtr iface,
                            virErrorPtr err);

virInterfacePtr
virInterfaceLookupByMACStringWrapper(virConnectPtr conn,
                                     const char * macstr,
                                     virErrorPtr err);

virInterfacePtr
virInterfaceLookupByNameWrapper(virConnectPtr conn,
                                const char * name,
                                virErrorPtr err);

int
virInterfaceRefWrapper(virInterfacePtr iface,
                       virErrorPtr err);

int
virInterfaceUndefineWrapper(virInterfacePtr iface,
                            virErrorPtr err);

virNWFilterBindingPtr
virNWFilterBindingCreateXMLWrapper(virConnectPtr conn,
                                   const char * xml,
                                   unsigned int flags,
                                   virErrorPtr err);

int
virNWFilterBindingDeleteWrapper(virNWFilterBindingPtr binding,
                                virErrorPtr err);

int
virNWFilterBindingFreeWrapper(virNWFilterBindingPtr binding,
                              virErrorPtr err);

const char *
virNWFilterBindingGetFilterNameWrapper(virNWFilterBindingPtr binding,
                                       virErrorPtr err);

const char *
virNWFilterBindingGetPortDevWrapper(virNWFilterBindingPtr binding,
                                    virErrorPtr err);

char *
virNWFilterBindingGetXMLDescWrapper(virNWFilterBindingPtr binding,
                                    unsigned int flags,
                                    virErrorPtr err);

virNWFilterBindingPtr
virNWFilterBindingLookupByPortDevWrapper(virConnectPtr conn,
                                         const char * portdev,
                                         virErrorPtr err);

int
virNWFilterBindingRefWrapper(virNWFilterBindingPtr binding,
                             virErrorPtr err);

virNWFilterPtr
virNWFilterDefineXMLWrapper(virConnectPtr conn,
                            const char * xmlDesc,
                            virErrorPtr err);

virNWFilterPtr
virNWFilterDefineXMLFlagsWrapper(virConnectPtr conn,
                                 const char * xmlDesc,
                                 unsigned int flags,
                                 virErrorPtr err);

int
virNWFilterFreeWrapper(virNWFilterPtr nwfilter,
                       virErrorPtr err);

const char *
virNWFilterGetNameWrapper(virNWFilterPtr nwfilter,
                          virErrorPtr err);

int
virNWFilterGetUUIDWrapper(virNWFilterPtr nwfilter,
                          unsigned char * uuid,
                          virErrorPtr err);

int
virNWFilterGetUUIDStringWrapper(virNWFilterPtr nwfilter,
                                char * buf,
                                virErrorPtr err);

char *
virNWFilterGetXMLDescWrapper(virNWFilterPtr nwfilter,
                             unsigned int flags,
                             virErrorPtr err);

virNWFilterPtr
virNWFilterLookupByNameWrapper(virConnectPtr conn,
                               const char * name,
                               virErrorPtr err);

virNWFilterPtr
virNWFilterLookupByUUIDWrapper(virConnectPtr conn,
                               const unsigned char * uuid,
                               virErrorPtr err);

virNWFilterPtr
virNWFilterLookupByUUIDStringWrapper(virConnectPtr conn,
                                     const char * uuidstr,
                                     virErrorPtr err);

int
virNWFilterRefWrapper(virNWFilterPtr nwfilter,
                      virErrorPtr err);

int
virNWFilterUndefineWrapper(virNWFilterPtr nwfilter,
                           virErrorPtr err);

int
virNetworkCreateWrapper(virNetworkPtr network,
                        virErrorPtr err);

virNetworkPtr
virNetworkCreateXMLWrapper(virConnectPtr conn,
                           const char * xmlDesc,
                           virErrorPtr err);

virNetworkPtr
virNetworkCreateXMLFlagsWrapper(virConnectPtr conn,
                                const char * xmlDesc,
                                unsigned int flags,
                                virErrorPtr err);

void
virNetworkDHCPLeaseFreeWrapper(virNetworkDHCPLeasePtr lease);

virNetworkPtr
virNetworkDefineXMLWrapper(virConnectPtr conn,
                           const char * xml,
                           virErrorPtr err);

virNetworkPtr
virNetworkDefineXMLFlagsWrapper(virConnectPtr conn,
                                const char * xml,
                                unsigned int flags,
                                virErrorPtr err);

int
virNetworkDestroyWrapper(virNetworkPtr network,
                         virErrorPtr err);

int
virNetworkFreeWrapper(virNetworkPtr network,
                      virErrorPtr err);

int
virNetworkGetAutostartWrapper(virNetworkPtr network,
                              int * autostart,
                              virErrorPtr err);

char *
virNetworkGetBridgeNameWrapper(virNetworkPtr network,
                               virErrorPtr err);

virConnectPtr
virNetworkGetConnectWrapper(virNetworkPtr net,
                            virErrorPtr err);

int
virNetworkGetDHCPLeasesWrapper(virNetworkPtr network,
                               const char * mac,
                               virNetworkDHCPLeasePtr ** leases,
                               unsigned int flags,
                               virErrorPtr err);

char *
virNetworkGetMetadataWrapper(virNetworkPtr network,
                             int type,
                             const char * uri,
                             unsigned int flags,
                             virErrorPtr err);

const char *
virNetworkGetNameWrapper(virNetworkPtr network,
                         virErrorPtr err);

int
virNetworkGetUUIDWrapper(virNetworkPtr network,
                         unsigned char * uuid,
                         virErrorPtr err);

int
virNetworkGetUUIDStringWrapper(virNetworkPtr network,
                               char * buf,
                               virErrorPtr err);

char *
virNetworkGetXMLDescWrapper(virNetworkPtr network,
                            unsigned int flags,
                            virErrorPtr err);

int
virNetworkIsActiveWrapper(virNetworkPtr net,
                          virErrorPtr err);

int
virNetworkIsPersistentWrapper(virNetworkPtr net,
                              virErrorPtr err);

int
virNetworkListAllPortsWrapper(virNetworkPtr network,
                              virNetworkPortPtr ** ports,
                              unsigned int flags,
                              virErrorPtr err);

virNetworkPtr
virNetworkLookupByNameWrapper(virConnectPtr conn,
                              const char * name,
                              virErrorPtr err);

virNetworkPtr
virNetworkLookupByUUIDWrapper(virConnectPtr conn,
                              const unsigned char * uuid,
                              virErrorPtr err);

virNetworkPtr
virNetworkLookupByUUIDStringWrapper(virConnectPtr conn,
                                    const char * uuidstr,
                                    virErrorPtr err);

virNetworkPortPtr
virNetworkPortCreateXMLWrapper(virNetworkPtr net,
                               const char * xmldesc,
                               unsigned int flags,
                               virErrorPtr err);

int
virNetworkPortDeleteWrapper(virNetworkPortPtr port,
                            unsigned int flags,
                            virErrorPtr err);

int
virNetworkPortFreeWrapper(virNetworkPortPtr port,
                          virErrorPtr err);

virNetworkPtr
virNetworkPortGetNetworkWrapper(virNetworkPortPtr port,
                                virErrorPtr err);

int
virNetworkPortGetParametersWrapper(virNetworkPortPtr port,
                                   virTypedParameterPtr * params,
                                   int * nparams,
                                   unsigned int flags,
                                   virErrorPtr err);

int
virNetworkPortGetUUIDWrapper(virNetworkPortPtr port,
                             unsigned char * uuid,
                             virErrorPtr err);

int
virNetworkPortGetUUIDStringWrapper(virNetworkPortPtr port,
                                   char * buf,
                                   virErrorPtr err);

char *
virNetworkPortGetXMLDescWrapper(virNetworkPortPtr port,
                                unsigned int flags,
                                virErrorPtr err);

virNetworkPortPtr
virNetworkPortLookupByUUIDWrapper(virNetworkPtr net,
                                  const unsigned char * uuid,
                                  virErrorPtr err);

virNetworkPortPtr
virNetworkPortLookupByUUIDStringWrapper(virNetworkPtr net,
                                        const char * uuidstr,
                                        virErrorPtr err);

int
virNetworkPortRefWrapper(virNetworkPortPtr port,
                         virErrorPtr err);

int
virNetworkPortSetParametersWrapper(virNetworkPortPtr port,
                                   virTypedParameterPtr params,
                                   int nparams,
                                   unsigned int flags,
                                   virErrorPtr err);

int
virNetworkRefWrapper(virNetworkPtr network,
                     virErrorPtr err);

int
virNetworkSetAutostartWrapper(virNetworkPtr network,
                              int autostart,
                              virErrorPtr err);

int
virNetworkSetMetadataWrapper(virNetworkPtr network,
                             int type,
                             const char * metadata,
                             const char * key,
                             const char * uri,
                             unsigned int flags,
                             virErrorPtr err);

int
virNetworkUndefineWrapper(virNetworkPtr network,
                          virErrorPtr err);

int
virNetworkUpdateWrapper(virNetworkPtr network,
                        unsigned int command,
                        unsigned int section,
                        int parentIndex,
                        const char * xml,
                        unsigned int flags,
                        virErrorPtr err);

int
virNodeAllocPagesWrapper(virConnectPtr conn,
                         unsigned int npages,
                         unsigned int * pageSizes,
                         unsigned long long * pageCounts,
                         int startCell,
                         unsigned int cellCount,
                         unsigned int flags,
                         virErrorPtr err);

int
virNodeDeviceCreateWrapper(virNodeDevicePtr dev,
                           unsigned int flags,
                           virErrorPtr err);

virNodeDevicePtr
virNodeDeviceCreateXMLWrapper(virConnectPtr conn,
                              const char * xmlDesc,
                              unsigned int flags,
                              virErrorPtr err);

virNodeDevicePtr
virNodeDeviceDefineXMLWrapper(virConnectPtr conn,
                              const char * xmlDesc,
                              unsigned int flags,
                              virErrorPtr err);

int
virNodeDeviceDestroyWrapper(virNodeDevicePtr dev,
                            virErrorPtr err);

int
virNodeDeviceDetachFlagsWrapper(virNodeDevicePtr dev,
                                const char * driverName,
                                unsigned int flags,
                                virErrorPtr err);

int
virNodeDeviceDettachWrapper(virNodeDevicePtr dev,
                            virErrorPtr err);

int
virNodeDeviceFreeWrapper(virNodeDevicePtr dev,
                         virErrorPtr err);

int
virNodeDeviceGetAutostartWrapper(virNodeDevicePtr dev,
                                 int * autostart,
                                 virErrorPtr err);

const char *
virNodeDeviceGetNameWrapper(virNodeDevicePtr dev,
                            virErrorPtr err);

const char *
virNodeDeviceGetParentWrapper(virNodeDevicePtr dev,
                              virErrorPtr err);

char *
virNodeDeviceGetXMLDescWrapper(virNodeDevicePtr dev,
                               unsigned int flags,
                               virErrorPtr err);

int
virNodeDeviceIsActiveWrapper(virNodeDevicePtr dev,
                             virErrorPtr err);

int
virNodeDeviceIsPersistentWrapper(virNodeDevicePtr dev,
                                 virErrorPtr err);

int
virNodeDeviceListCapsWrapper(virNodeDevicePtr dev,
                             char ** const names,
                             int maxnames,
                             virErrorPtr err);

virNodeDevicePtr
virNodeDeviceLookupByNameWrapper(virConnectPtr conn,
                                 const char * name,
                                 virErrorPtr err);

virNodeDevicePtr
virNodeDeviceLookupSCSIHostByWWNWrapper(virConnectPtr conn,
                                        const char * wwnn,
                                        const char * wwpn,
                                        unsigned int flags,
                                        virErrorPtr err);

int
virNodeDeviceNumOfCapsWrapper(virNodeDevicePtr dev,
                              virErrorPtr err);

int
virNodeDeviceReAttachWrapper(virNodeDevicePtr dev,
                             virErrorPtr err);

int
virNodeDeviceRefWrapper(virNodeDevicePtr dev,
                        virErrorPtr err);

int
virNodeDeviceResetWrapper(virNodeDevicePtr dev,
                          virErrorPtr err);

int
virNodeDeviceSetAutostartWrapper(virNodeDevicePtr dev,
                                 int autostart,
                                 virErrorPtr err);

int
virNodeDeviceUndefineWrapper(virNodeDevicePtr dev,
                             unsigned int flags,
                             virErrorPtr err);

int
virNodeDeviceUpdateWrapper(virNodeDevicePtr dev,
                           const char * xmlDesc,
                           unsigned int flags,
                           virErrorPtr err);

int
virNodeGetCPUMapWrapper(virConnectPtr conn,
                        unsigned char ** cpumap,
                        unsigned int * online,
                        unsigned int flags,
                        virErrorPtr err);

int
virNodeGetCPUStatsWrapper(virConnectPtr conn,
                          int cpuNum,
                          virNodeCPUStatsPtr params,
                          int * nparams,
                          unsigned int flags,
                          virErrorPtr err);

int
virNodeGetCellsFreeMemoryWrapper(virConnectPtr conn,
                                 unsigned long long * freeMems,
                                 int startCell,
                                 int maxCells,
                                 virErrorPtr err);

unsigned long long
virNodeGetFreeMemoryWrapper(virConnectPtr conn,
                            virErrorPtr err);

int
virNodeGetFreePagesWrapper(virConnectPtr conn,
                           unsigned int npages,
                           unsigned int * pages,
                           int startCell,
                           unsigned int cellCount,
                           unsigned long long * counts,
                           unsigned int flags,
                           virErrorPtr err);

int
virNodeGetInfoWrapper(virConnectPtr conn,
                      virNodeInfoPtr info,
                      virErrorPtr err);

int
virNodeGetMemoryParametersWrapper(virConnectPtr conn,
                                  virTypedParameterPtr params,
                                  int * nparams,
                                  unsigned int flags,
                                  virErrorPtr err);

int
virNodeGetMemoryStatsWrapper(virConnectPtr conn,
                             int cellNum,
                             virNodeMemoryStatsPtr params,
                             int * nparams,
                             unsigned int flags,
                             virErrorPtr err);

int
virNodeGetSEVInfoWrapper(virConnectPtr conn,
                         virTypedParameterPtr * params,
                         int * nparams,
                         unsigned int flags,
                         virErrorPtr err);

int
virNodeGetSecurityModelWrapper(virConnectPtr conn,
                               virSecurityModelPtr secmodel,
                               virErrorPtr err);

int
virNodeListDevicesWrapper(virConnectPtr conn,
                          const char * cap,
                          char ** const names,
                          int maxnames,
                          unsigned int flags,
                          virErrorPtr err);

int
virNodeNumOfDevicesWrapper(virConnectPtr conn,
                           const char * cap,
                           unsigned int flags,
                           virErrorPtr err);

int
virNodeSetMemoryParametersWrapper(virConnectPtr conn,
                                  virTypedParameterPtr params,
                                  int nparams,
                                  unsigned int flags,
                                  virErrorPtr err);

int
virNodeSuspendForDurationWrapper(virConnectPtr conn,
                                 unsigned int target,
                                 unsigned long long duration,
                                 unsigned int flags,
                                 virErrorPtr err);

void
virResetErrorWrapper(virErrorPtr err);

void
virResetLastErrorWrapper(void);

virErrorPtr
virSaveLastErrorWrapper(virErrorPtr err);

virSecretPtr
virSecretDefineXMLWrapper(virConnectPtr conn,
                          const char * xml,
                          unsigned int flags,
                          virErrorPtr err);

int
virSecretFreeWrapper(virSecretPtr secret,
                     virErrorPtr err);

virConnectPtr
virSecretGetConnectWrapper(virSecretPtr secret,
                           virErrorPtr err);

int
virSecretGetUUIDWrapper(virSecretPtr secret,
                        unsigned char * uuid,
                        virErrorPtr err);

int
virSecretGetUUIDStringWrapper(virSecretPtr secret,
                              char * buf,
                              virErrorPtr err);

const char *
virSecretGetUsageIDWrapper(virSecretPtr secret,
                           virErrorPtr err);

int
virSecretGetUsageTypeWrapper(virSecretPtr secret,
                             virErrorPtr err);

unsigned char *
virSecretGetValueWrapper(virSecretPtr secret,
                         size_t * value_size,
                         unsigned int flags,
                         virErrorPtr err);

char *
virSecretGetXMLDescWrapper(virSecretPtr secret,
                           unsigned int flags,
                           virErrorPtr err);

virSecretPtr
virSecretLookupByUUIDWrapper(virConnectPtr conn,
                             const unsigned char * uuid,
                             virErrorPtr err);

virSecretPtr
virSecretLookupByUUIDStringWrapper(virConnectPtr conn,
                                   const char * uuidstr,
                                   virErrorPtr err);

virSecretPtr
virSecretLookupByUsageWrapper(virConnectPtr conn,
                              int usageType,
                              const char * usageID,
                              virErrorPtr err);

int
virSecretRefWrapper(virSecretPtr secret,
                    virErrorPtr err);

int
virSecretSetValueWrapper(virSecretPtr secret,
                         const unsigned char * value,
                         size_t value_size,
                         unsigned int flags,
                         virErrorPtr err);

int
virSecretUndefineWrapper(virSecretPtr secret,
                         virErrorPtr err);

void
virSetErrorFuncWrapper(void * userData,
                       virErrorFunc handler);

int
virStoragePoolBuildWrapper(virStoragePoolPtr pool,
                           unsigned int flags,
                           virErrorPtr err);

int
virStoragePoolCreateWrapper(virStoragePoolPtr pool,
                            unsigned int flags,
                            virErrorPtr err);

virStoragePoolPtr
virStoragePoolCreateXMLWrapper(virConnectPtr conn,
                               const char * xmlDesc,
                               unsigned int flags,
                               virErrorPtr err);

virStoragePoolPtr
virStoragePoolDefineXMLWrapper(virConnectPtr conn,
                               const char * xml,
                               unsigned int flags,
                               virErrorPtr err);

int
virStoragePoolDeleteWrapper(virStoragePoolPtr pool,
                            unsigned int flags,
                            virErrorPtr err);

int
virStoragePoolDestroyWrapper(virStoragePoolPtr pool,
                             virErrorPtr err);

int
virStoragePoolFreeWrapper(virStoragePoolPtr pool,
                          virErrorPtr err);

int
virStoragePoolGetAutostartWrapper(virStoragePoolPtr pool,
                                  int * autostart,
                                  virErrorPtr err);

virConnectPtr
virStoragePoolGetConnectWrapper(virStoragePoolPtr pool,
                                virErrorPtr err);

int
virStoragePoolGetInfoWrapper(virStoragePoolPtr pool,
                             virStoragePoolInfoPtr info,
                             virErrorPtr err);

const char *
virStoragePoolGetNameWrapper(virStoragePoolPtr pool,
                             virErrorPtr err);

int
virStoragePoolGetUUIDWrapper(virStoragePoolPtr pool,
                             unsigned char * uuid,
                             virErrorPtr err);

int
virStoragePoolGetUUIDStringWrapper(virStoragePoolPtr pool,
                                   char * buf,
                                   virErrorPtr err);

char *
virStoragePoolGetXMLDescWrapper(virStoragePoolPtr pool,
                                unsigned int flags,
                                virErrorPtr err);

int
virStoragePoolIsActiveWrapper(virStoragePoolPtr pool,
                              virErrorPtr err);

int
virStoragePoolIsPersistentWrapper(virStoragePoolPtr pool,
                                  virErrorPtr err);

int
virStoragePoolListAllVolumesWrapper(virStoragePoolPtr pool,
                                    virStorageVolPtr ** vols,
                                    unsigned int flags,
                                    virErrorPtr err);

int
virStoragePoolListVolumesWrapper(virStoragePoolPtr pool,
                                 char ** const names,
                                 int maxnames,
                                 virErrorPtr err);

virStoragePoolPtr
virStoragePoolLookupByNameWrapper(virConnectPtr conn,
                                  const char * name,
                                  virErrorPtr err);

virStoragePoolPtr
virStoragePoolLookupByTargetPathWrapper(virConnectPtr conn,
                                        const char * path,
                                        virErrorPtr err);

virStoragePoolPtr
virStoragePoolLookupByUUIDWrapper(virConnectPtr conn,
                                  const unsigned char * uuid,
                                  virErrorPtr err);

virStoragePoolPtr
virStoragePoolLookupByUUIDStringWrapper(virConnectPtr conn,
                                        const char * uuidstr,
                                        virErrorPtr err);

virStoragePoolPtr
virStoragePoolLookupByVolumeWrapper(virStorageVolPtr vol,
                                    virErrorPtr err);

int
virStoragePoolNumOfVolumesWrapper(virStoragePoolPtr pool,
                                  virErrorPtr err);

int
virStoragePoolRefWrapper(virStoragePoolPtr pool,
                         virErrorPtr err);

int
virStoragePoolRefreshWrapper(virStoragePoolPtr pool,
                             unsigned int flags,
                             virErrorPtr err);

int
virStoragePoolSetAutostartWrapper(virStoragePoolPtr pool,
                                  int autostart,
                                  virErrorPtr err);

int
virStoragePoolUndefineWrapper(virStoragePoolPtr pool,
                              virErrorPtr err);

virStorageVolPtr
virStorageVolCreateXMLWrapper(virStoragePoolPtr pool,
                              const char * xmlDesc,
                              unsigned int flags,
                              virErrorPtr err);

virStorageVolPtr
virStorageVolCreateXMLFromWrapper(virStoragePoolPtr pool,
                                  const char * xmlDesc,
                                  virStorageVolPtr clonevol,
                                  unsigned int flags,
                                  virErrorPtr err);

int
virStorageVolDeleteWrapper(virStorageVolPtr vol,
                           unsigned int flags,
                           virErrorPtr err);

int
virStorageVolDownloadWrapper(virStorageVolPtr vol,
                             virStreamPtr stream,
                             unsigned long long offset,
                             unsigned long long length,
                             unsigned int flags,
                             virErrorPtr err);

int
virStorageVolFreeWrapper(virStorageVolPtr vol,
                         virErrorPtr err);

virConnectPtr
virStorageVolGetConnectWrapper(virStorageVolPtr vol,
                               virErrorPtr err);

int
virStorageVolGetInfoWrapper(virStorageVolPtr vol,
                            virStorageVolInfoPtr info,
                            virErrorPtr err);

int
virStorageVolGetInfoFlagsWrapper(virStorageVolPtr vol,
                                 virStorageVolInfoPtr info,
                                 unsigned int flags,
                                 virErrorPtr err);

const char *
virStorageVolGetKeyWrapper(virStorageVolPtr vol,
                           virErrorPtr err);

const char *
virStorageVolGetNameWrapper(virStorageVolPtr vol,
                            virErrorPtr err);

char *
virStorageVolGetPathWrapper(virStorageVolPtr vol,
                            virErrorPtr err);

char *
virStorageVolGetXMLDescWrapper(virStorageVolPtr vol,
                               unsigned int flags,
                               virErrorPtr err);

virStorageVolPtr
virStorageVolLookupByKeyWrapper(virConnectPtr conn,
                                const char * key,
                                virErrorPtr err);

virStorageVolPtr
virStorageVolLookupByNameWrapper(virStoragePoolPtr pool,
                                 const char * name,
                                 virErrorPtr err);

virStorageVolPtr
virStorageVolLookupByPathWrapper(virConnectPtr conn,
                                 const char * path,
                                 virErrorPtr err);

int
virStorageVolRefWrapper(virStorageVolPtr vol,
                        virErrorPtr err);

int
virStorageVolResizeWrapper(virStorageVolPtr vol,
                           unsigned long long capacity,
                           unsigned int flags,
                           virErrorPtr err);

int
virStorageVolUploadWrapper(virStorageVolPtr vol,
                           virStreamPtr stream,
                           unsigned long long offset,
                           unsigned long long length,
                           unsigned int flags,
                           virErrorPtr err);

int
virStorageVolWipeWrapper(virStorageVolPtr vol,
                         unsigned int flags,
                         virErrorPtr err);

int
virStorageVolWipePatternWrapper(virStorageVolPtr vol,
                                unsigned int algorithm,
                                unsigned int flags,
                                virErrorPtr err);

int
virStreamAbortWrapper(virStreamPtr stream,
                      virErrorPtr err);

int
virStreamEventAddCallbackWrapper(virStreamPtr stream,
                                 int events,
                                 virStreamEventCallback cb,
                                 void * opaque,
                                 virFreeCallback ff,
                                 virErrorPtr err);

int
virStreamEventRemoveCallbackWrapper(virStreamPtr stream,
                                    virErrorPtr err);

int
virStreamEventUpdateCallbackWrapper(virStreamPtr stream,
                                    int events,
                                    virErrorPtr err);

int
virStreamFinishWrapper(virStreamPtr stream,
                       virErrorPtr err);

int
virStreamFreeWrapper(virStreamPtr stream,
                     virErrorPtr err);

virStreamPtr
virStreamNewWrapper(virConnectPtr conn,
                    unsigned int flags,
                    virErrorPtr err);

int
virStreamRecvWrapper(virStreamPtr stream,
                     char * data,
                     size_t nbytes,
                     virErrorPtr err);

int
virStreamRecvAllWrapper(virStreamPtr stream,
                        virStreamSinkFunc handler,
                        void * opaque,
                        virErrorPtr err);

int
virStreamRecvFlagsWrapper(virStreamPtr stream,
                          char * data,
                          size_t nbytes,
                          unsigned int flags,
                          virErrorPtr err);

int
virStreamRecvHoleWrapper(virStreamPtr stream,
                         long long * length,
                         unsigned int flags,
                         virErrorPtr err);

int
virStreamRefWrapper(virStreamPtr stream,
                    virErrorPtr err);

int
virStreamSendWrapper(virStreamPtr stream,
                     const char * data,
                     size_t nbytes,
                     virErrorPtr err);

int
virStreamSendAllWrapper(virStreamPtr stream,
                        virStreamSourceFunc handler,
                        void * opaque,
                        virErrorPtr err);

int
virStreamSendHoleWrapper(virStreamPtr stream,
                         long long length,
                         unsigned int flags,
                         virErrorPtr err);

int
virStreamSparseRecvAllWrapper(virStreamPtr stream,
                              virStreamSinkFunc handler,
                              virStreamSinkHoleFunc holeHandler,
                              void * opaque,
                              virErrorPtr err);

int
virStreamSparseSendAllWrapper(virStreamPtr stream,
                              virStreamSourceFunc handler,
                              virStreamSourceHoleFunc holeHandler,
                              virStreamSourceSkipFunc skipHandler,
                              void * opaque,
                              virErrorPtr err);

int
virTypedParamsAddBooleanWrapper(virTypedParameterPtr * params,
                                int * nparams,
                                int * maxparams,
                                const char * name,
                                int value,
                                virErrorPtr err);

int
virTypedParamsAddDoubleWrapper(virTypedParameterPtr * params,
                               int * nparams,
                               int * maxparams,
                               const char * name,
                               double value,
                               virErrorPtr err);

int
virTypedParamsAddFromStringWrapper(virTypedParameterPtr * params,
                                   int * nparams,
                                   int * maxparams,
                                   const char * name,
                                   int type,
                                   const char * value,
                                   virErrorPtr err);

int
virTypedParamsAddIntWrapper(virTypedParameterPtr * params,
                            int * nparams,
                            int * maxparams,
                            const char * name,
                            int value,
                            virErrorPtr err);

int
virTypedParamsAddLLongWrapper(virTypedParameterPtr * params,
                              int * nparams,
                              int * maxparams,
                              const char * name,
                              long long value,
                              virErrorPtr err);

int
virTypedParamsAddStringWrapper(virTypedParameterPtr * params,
                               int * nparams,
                               int * maxparams,
                               const char * name,
                               const char * value,
                               virErrorPtr err);

int
virTypedParamsAddStringListWrapper(virTypedParameterPtr * params,
                                   int * nparams,
                                   int * maxparams,
                                   const char * name,
                                   const char ** values,
                                   virErrorPtr err);

int
virTypedParamsAddUIntWrapper(virTypedParameterPtr * params,
                             int * nparams,
                             int * maxparams,
                             const char * name,
                             unsigned int value,
                             virErrorPtr err);

int
virTypedParamsAddULLongWrapper(virTypedParameterPtr * params,
                               int * nparams,
                               int * maxparams,
                               const char * name,
                               unsigned long long value,
                               virErrorPtr err);

void
virTypedParamsClearWrapper(virTypedParameterPtr params,
                           int nparams);

void
virTypedParamsFreeWrapper(virTypedParameterPtr params,
                          int nparams);

virTypedParameterPtr
virTypedParamsGetWrapper(virTypedParameterPtr params,
                         int nparams,
                         const char * name,
                         virErrorPtr err);

int
virTypedParamsGetBooleanWrapper(virTypedParameterPtr params,
                                int nparams,
                                const char * name,
                                int * value,
                                virErrorPtr err);

int
virTypedParamsGetDoubleWrapper(virTypedParameterPtr params,
                               int nparams,
                               const char * name,
                               double * value,
                               virErrorPtr err);

int
virTypedParamsGetIntWrapper(virTypedParameterPtr params,
                            int nparams,
                            const char * name,
                            int * value,
                            virErrorPtr err);

int
virTypedParamsGetLLongWrapper(virTypedParameterPtr params,
                              int nparams,
                              const char * name,
                              long long * value,
                              virErrorPtr err);

int
virTypedParamsGetStringWrapper(virTypedParameterPtr params,
                               int nparams,
                               const char * name,
                               const char ** value,
                               virErrorPtr err);

int
virTypedParamsGetUIntWrapper(virTypedParameterPtr params,
                             int nparams,
                             const char * name,
                             unsigned int * value,
                             virErrorPtr err);

int
virTypedParamsGetULLongWrapper(virTypedParameterPtr params,
                               int nparams,
                               const char * name,
                               unsigned long long * value,
                               virErrorPtr err);
