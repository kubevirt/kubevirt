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

#ifndef LIBVIRT_GO_CONNECT_WRAPPER_H__
#define LIBVIRT_GO_CONNECT_WRAPPER_H__

#include <libvirt/libvirt.h>
#include <libvirt/virterror.h>
#include "connect_compat.h"

void
closeCallbackHelper(virConnectPtr conn,
                    int reason,
                    void *opaque);

int
virConnectRegisterCloseCallbackHelper(virConnectPtr c,
                                      virConnectCloseFunc cb,
                                      long goCallbackId);

char *
virConnectBaselineCPUWrapper(virConnectPtr conn,
                             const char **xmlCPUs,
                             unsigned int ncpus,
                             unsigned int flags,
                             virErrorPtr err);

char *
virConnectBaselineHypervisorCPUWrapper(virConnectPtr conn,
                                       const char *emulator,
                                       const char *arch,
                                       const char *machine,
                                       const char *virttype,
                                       const char **xmlCPUs,
                                       unsigned int ncpus,
                                       unsigned int flags,
                                       virErrorPtr err);

int
virConnectCloseWrapper(virConnectPtr conn,
                       virErrorPtr err);

int
virConnectCompareCPUWrapper(virConnectPtr conn,
                            const char *xmlDesc,
                            unsigned int flags,
                            virErrorPtr err);

int
virConnectCompareHypervisorCPUWrapper(virConnectPtr conn,
                                      const char *emulator,
                                      const char *arch,
                                      const char *machine,
                                      const char *virttype,
                                      const char *xmlCPU,
                                      unsigned int flags,
                                      virErrorPtr err);

char *
virConnectDomainXMLFromNativeWrapper(virConnectPtr conn,
                                     const char *nativeFormat,
                                     const char *nativeConfig,
                                     unsigned int flags,
                                     virErrorPtr err);

char *
virConnectDomainXMLToNativeWrapper(virConnectPtr conn,
                                   const char *nativeFormat,
                                   const char *domainXml,
                                   unsigned int flags,
                                   virErrorPtr err);

char *
virConnectFindStoragePoolSourcesWrapper(virConnectPtr conn,
                                        const char *type,
                                        const char *srcSpec,
                                        unsigned int flags,
                                        virErrorPtr err);

int
virConnectGetAllDomainStatsWrapper(virConnectPtr conn,
                                   unsigned int stats,
                                   virDomainStatsRecordPtr **retStats,
                                   unsigned int flags,
                                   virErrorPtr err);

int
virConnectGetCPUModelNamesWrapper(virConnectPtr conn,
                                  const char *arch,
                                  char ***models,
                                  unsigned int flags,
                                  virErrorPtr err);

char *
virConnectGetCapabilitiesWrapper(virConnectPtr conn,
                                 virErrorPtr err);

char *
virConnectGetDomainCapabilitiesWrapper(virConnectPtr conn,
                                       const char *emulatorbin,
                                       const char *arch,
                                       const char *machine,
                                       const char *virttype,
                                       unsigned int flags,
                                       virErrorPtr err);

char *
virConnectGetHostnameWrapper(virConnectPtr conn,
                             virErrorPtr err);

int
virConnectGetLibVersionWrapper(virConnectPtr conn,
                               unsigned long *libVer,
                               virErrorPtr err);

int
virConnectGetMaxVcpusWrapper(virConnectPtr conn,
                             const char *type,
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
                            unsigned long *hvVer,
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
                                virDomainPtr **domains,
                                unsigned int flags,
                                virErrorPtr err);

int
virConnectListAllInterfacesWrapper(virConnectPtr conn,
                                   virInterfacePtr **ifaces,
                                   unsigned int flags,
                                   virErrorPtr err);

int
virConnectListAllNWFilterBindingsWrapper(virConnectPtr conn,
                                         virNWFilterBindingPtr **bindings,
                                         unsigned int flags,
                                         virErrorPtr err);

int
virConnectListAllNWFiltersWrapper(virConnectPtr conn,
                                  virNWFilterPtr **filters,
                                  unsigned int flags,
                                  virErrorPtr err);

int
virConnectListAllNetworksWrapper(virConnectPtr conn,
                                 virNetworkPtr **nets,
                                 unsigned int flags,
                                 virErrorPtr err);

int
virConnectListAllNodeDevicesWrapper(virConnectPtr conn,
                                    virNodeDevicePtr **devices,
                                    unsigned int flags,
                                    virErrorPtr err);

int
virConnectListAllSecretsWrapper(virConnectPtr conn,
                                virSecretPtr **secrets,
                                unsigned int flags,
                                virErrorPtr err);

int
virConnectListAllStoragePoolsWrapper(virConnectPtr conn,
                                     virStoragePoolPtr **pools,
                                     unsigned int flags,
                                     virErrorPtr err);

int
virConnectListDefinedDomainsWrapper(virConnectPtr conn,
                                    char **const names,
                                    int maxnames,
                                    virErrorPtr err);

int
virConnectListDefinedInterfacesWrapper(virConnectPtr conn,
                                       char **const names,
                                       int maxnames,
                                       virErrorPtr err);

int
virConnectListDefinedNetworksWrapper(virConnectPtr conn,
                                     char **const names,
                                     int maxnames,
                                     virErrorPtr err);

int
virConnectListDefinedStoragePoolsWrapper(virConnectPtr conn,
                                         char **const names,
                                         int maxnames,
                                         virErrorPtr err);

int
virConnectListDomainsWrapper(virConnectPtr conn,
                             int *ids,
                             int maxids,
                             virErrorPtr err);

int
virConnectListInterfacesWrapper(virConnectPtr conn,
                                char **const names,
                                int maxnames,
                                virErrorPtr err);

int
virConnectListNWFiltersWrapper(virConnectPtr conn,
                               char **const names,
                               int maxnames,
                               virErrorPtr err);

int
virConnectListNetworksWrapper(virConnectPtr conn,
                              char **const names,
                              int maxnames,
                              virErrorPtr err);

int
virConnectListSecretsWrapper(virConnectPtr conn,
                             char **uuids,
                             int maxuuids,
                             virErrorPtr err);

int
virConnectListStoragePoolsWrapper(virConnectPtr conn,
                                  char **const names,
                                  int maxnames,
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
virConnectOpenWrapper(const char *name,
                      virErrorPtr err);

virConnectPtr
virConnectOpenAuthWrapper(const char *name,
                          int *credtype,
                          unsigned int ncredtype,
                          int callbackID,
                          unsigned int flags,
                          virErrorPtr err);

virConnectPtr
virConnectOpenReadOnlyWrapper(const char *name,
                              virErrorPtr err);

int
virConnectRefWrapper(virConnectPtr conn,
                     virErrorPtr err);

int
virConnectRegisterCloseCallbackWrapper(virConnectPtr conn,
                                       long goCallbackId,
                                       virErrorPtr err);

int
virConnectSetKeepAliveWrapper(virConnectPtr conn,
                              int interval,
                              unsigned int count,
                              virErrorPtr err);

int
virConnectUnregisterCloseCallbackWrapper(virConnectPtr conn,
                                         virErrorPtr err);

virDomainPtr
virDomainCreateLinuxWrapper(virConnectPtr conn,
                            const char *xmlDesc,
                            unsigned int flags,
                            virErrorPtr err);

virDomainPtr
virDomainCreateXMLWrapper(virConnectPtr conn,
                          const char *xmlDesc,
                          unsigned int flags,
                          virErrorPtr err);

virDomainPtr
virDomainCreateXMLWithFilesWrapper(virConnectPtr conn,
                                   const char *xmlDesc,
                                   unsigned int nfiles,
                                   int *files,
                                   unsigned int flags,
                                   virErrorPtr err);

virDomainPtr
virDomainDefineXMLWrapper(virConnectPtr conn,
                          const char *xml,
                          virErrorPtr err);

virDomainPtr
virDomainDefineXMLFlagsWrapper(virConnectPtr conn,
                               const char *xml,
                               unsigned int flags,
                               virErrorPtr err);

int
virDomainListGetStatsWrapper(virDomainPtr *doms,
                             unsigned int stats,
                             virDomainStatsRecordPtr **retStats,
                             unsigned int flags,
                             virErrorPtr err);

virDomainPtr
virDomainLookupByIDWrapper(virConnectPtr conn,
                           int id,
                           virErrorPtr err);

virDomainPtr
virDomainLookupByNameWrapper(virConnectPtr conn,
                             const char *name,
                             virErrorPtr err);

virDomainPtr
virDomainLookupByUUIDWrapper(virConnectPtr conn,
                             const unsigned char *uuid,
                             virErrorPtr err);

virDomainPtr
virDomainLookupByUUIDStringWrapper(virConnectPtr conn,
                                   const char *uuidstr,
                                   virErrorPtr err);

int
virDomainRestoreWrapper(virConnectPtr conn,
                        const char *from,
                        virErrorPtr err);

int
virDomainRestoreFlagsWrapper(virConnectPtr conn,
                             const char *from,
                             const char *dxml,
                             unsigned int flags,
                             virErrorPtr err);

int
virDomainSaveImageDefineXMLWrapper(virConnectPtr conn,
                                   const char *file,
                                   const char *dxml,
                                   unsigned int flags,
                                   virErrorPtr err);

char *
virDomainSaveImageGetXMLDescWrapper(virConnectPtr conn,
                                    const char *file,
                                    unsigned int flags,
                                    virErrorPtr err);

void
virDomainStatsRecordListFreeWrapper(virDomainStatsRecordPtr *stats);

int
virGetVersionWrapper(unsigned long *libVer,
                     const char *type,
                     unsigned long *typeVer,
                     virErrorPtr err);

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

virInterfacePtr
virInterfaceDefineXMLWrapper(virConnectPtr conn,
                             const char *xml,
                             unsigned int flags,
                             virErrorPtr err);

virInterfacePtr
virInterfaceLookupByMACStringWrapper(virConnectPtr conn,
                                     const char *macstr,
                                     virErrorPtr err);

virInterfacePtr
virInterfaceLookupByNameWrapper(virConnectPtr conn,
                                const char *name,
                                virErrorPtr err);

virNWFilterBindingPtr
virNWFilterBindingCreateXMLWrapper(virConnectPtr conn,
                                   const char *xml,
                                   unsigned int flags,
                                   virErrorPtr err);

virNWFilterBindingPtr
virNWFilterBindingLookupByPortDevWrapper(virConnectPtr conn,
                                         const char *portdev,
                                         virErrorPtr err);

virNWFilterPtr
virNWFilterDefineXMLWrapper(virConnectPtr conn,
                            const char *xmlDesc,
                            virErrorPtr err);

virNWFilterPtr
virNWFilterLookupByNameWrapper(virConnectPtr conn,
                               const char *name,
                               virErrorPtr err);

virNWFilterPtr
virNWFilterLookupByUUIDWrapper(virConnectPtr conn,
                               const unsigned char *uuid,
                               virErrorPtr err);

virNWFilterPtr
virNWFilterLookupByUUIDStringWrapper(virConnectPtr conn,
                                     const char *uuidstr,
                                     virErrorPtr err);

virNetworkPtr
virNetworkCreateXMLWrapper(virConnectPtr conn,
                           const char *xmlDesc,
                           virErrorPtr err);

virNetworkPtr
virNetworkDefineXMLWrapper(virConnectPtr conn,
                           const char *xml,
                           virErrorPtr err);

virNetworkPtr
virNetworkLookupByNameWrapper(virConnectPtr conn,
                              const char *name,
                              virErrorPtr err);

virNetworkPtr
virNetworkLookupByUUIDWrapper(virConnectPtr conn,
                              const unsigned char *uuid,
                              virErrorPtr err);

virNetworkPtr
virNetworkLookupByUUIDStringWrapper(virConnectPtr conn,
                                    const char *uuidstr,
                                    virErrorPtr err);

int
virNodeAllocPagesWrapper(virConnectPtr conn,
                         unsigned int npages,
                         unsigned int *pageSizes,
                         unsigned long long *pageCounts,
                         int startCell,
                         unsigned int cellCount,
                         unsigned int flags,
                         virErrorPtr err);

virNodeDevicePtr
virNodeDeviceCreateXMLWrapper(virConnectPtr conn,
                              const char *xmlDesc,
                              unsigned int flags,
                              virErrorPtr err);

virNodeDevicePtr
virNodeDeviceLookupByNameWrapper(virConnectPtr conn,
                                 const char *name,
                                 virErrorPtr err);

virNodeDevicePtr
virNodeDeviceLookupSCSIHostByWWNWrapper(virConnectPtr conn,
                                        const char *wwnn,
                                        const char *wwpn,
                                        unsigned int flags,
                                        virErrorPtr err);

int
virNodeGetCPUMapWrapper(virConnectPtr conn,
                        unsigned char **cpumap,
                        unsigned int *online,
                        unsigned int flags,
                        virErrorPtr err);

int
virNodeGetCPUStatsWrapper(virConnectPtr conn,
                          int cpuNum,
                          virNodeCPUStatsPtr params,
                          int *nparams,
                          unsigned int flags,
                          virErrorPtr err);

int
virNodeGetCellsFreeMemoryWrapper(virConnectPtr conn,
                                 unsigned long long *freeMems,
                                 int startCell,
                                 int maxCells,
                                 virErrorPtr err);

unsigned long long
virNodeGetFreeMemoryWrapper(virConnectPtr conn,
                            virErrorPtr err);

int
virNodeGetFreePagesWrapper(virConnectPtr conn,
                           unsigned int npages,
                           unsigned int *pages,
                           int startCell,
                           unsigned int cellCount,
                           unsigned long long *counts,
                           unsigned int flags,
                           virErrorPtr err);

int
virNodeGetInfoWrapper(virConnectPtr conn,
                      virNodeInfoPtr info,
                      virErrorPtr err);

int
virNodeGetMemoryParametersWrapper(virConnectPtr conn,
                                  virTypedParameterPtr params,
                                  int *nparams,
                                  unsigned int flags,
                                  virErrorPtr err);

int
virNodeGetMemoryStatsWrapper(virConnectPtr conn,
                             int cellNum,
                             virNodeMemoryStatsPtr params,
                             int *nparams,
                             unsigned int flags,
                             virErrorPtr err);

int
virNodeGetSEVInfoWrapper(virConnectPtr conn,
                         virTypedParameterPtr *params,
                         int *nparams,
                         unsigned int flags,
                         virErrorPtr err);

int
virNodeGetSecurityModelWrapper(virConnectPtr conn,
                               virSecurityModelPtr secmodel,
                               virErrorPtr err);

int
virNodeListDevicesWrapper(virConnectPtr conn,
                          const char *cap,
                          char **const names,
                          int maxnames,
                          unsigned int flags,
                          virErrorPtr err);

int
virNodeNumOfDevicesWrapper(virConnectPtr conn,
                           const char *cap,
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

virSecretPtr
virSecretDefineXMLWrapper(virConnectPtr conn,
                          const char *xml,
                          unsigned int flags,
                          virErrorPtr err);

virSecretPtr
virSecretLookupByUUIDWrapper(virConnectPtr conn,
                             const unsigned char *uuid,
                             virErrorPtr err);

virSecretPtr
virSecretLookupByUUIDStringWrapper(virConnectPtr conn,
                                   const char *uuidstr,
                                   virErrorPtr err);

virSecretPtr
virSecretLookupByUsageWrapper(virConnectPtr conn,
                              int usageType,
                              const char *usageID,
                              virErrorPtr err);

virStoragePoolPtr
virStoragePoolCreateXMLWrapper(virConnectPtr conn,
                               const char *xmlDesc,
                               unsigned int flags,
                               virErrorPtr err);

virStoragePoolPtr
virStoragePoolDefineXMLWrapper(virConnectPtr conn,
                               const char *xml,
                               unsigned int flags,
                               virErrorPtr err);

virStoragePoolPtr
virStoragePoolLookupByNameWrapper(virConnectPtr conn,
                                  const char *name,
                                  virErrorPtr err);

virStoragePoolPtr
virStoragePoolLookupByTargetPathWrapper(virConnectPtr conn,
                                        const char *path,
                                        virErrorPtr err);

virStoragePoolPtr
virStoragePoolLookupByUUIDWrapper(virConnectPtr conn,
                                  const unsigned char *uuid,
                                  virErrorPtr err);

virStoragePoolPtr
virStoragePoolLookupByUUIDStringWrapper(virConnectPtr conn,
                                        const char *uuidstr,
                                        virErrorPtr err);

virStorageVolPtr
virStorageVolLookupByKeyWrapper(virConnectPtr conn,
                                const char *key,
                                virErrorPtr err);

virStorageVolPtr
virStorageVolLookupByPathWrapper(virConnectPtr conn,
                                 const char *path,
                                 virErrorPtr err);

virStreamPtr
virStreamNewWrapper(virConnectPtr conn,
                    unsigned int flags,
                    virErrorPtr err);


#endif /* LIBVIRT_GO_CONNECT_WRAPPER_H__ */
