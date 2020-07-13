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
#include <stdio.h>
#include "connect_wrapper.h"
#include "callbacks_wrapper.h"

extern void closeCallback(virConnectPtr, int, long);
void closeCallbackHelper(virConnectPtr conn, int reason, void *opaque)
{
    closeCallback(conn, reason, (long)opaque);
}

extern int connectAuthCallback(virConnectCredentialPtr, unsigned int, int);
int connectAuthCallbackHelper(virConnectCredentialPtr cred, unsigned int ncred, void *cbdata)
{
    int *callbackID = cbdata;

    return connectAuthCallback(cred, ncred, *callbackID);
}


char *
virConnectBaselineCPUWrapper(virConnectPtr conn,
                             const char **xmlCPUs,
                             unsigned int ncpus,
                             unsigned int flags,
                             virErrorPtr err)
{
    char * ret = virConnectBaselineCPU(conn, xmlCPUs, ncpus, flags);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


char *
virConnectBaselineHypervisorCPUWrapper(virConnectPtr conn,
                                       const char *emulator,
                                       const char *arch,
                                       const char *machine,
                                       const char *virttype,
                                       const char **xmlCPUs,
                                       unsigned int ncpus,
                                       unsigned int flags,
                                       virErrorPtr err)
{
#if LIBVIR_VERSION_NUMBER < 4004000
    assert(0); // Caller should have checked version
#else
    char * ret = virConnectBaselineHypervisorCPU(conn, emulator, arch, machine, virttype, xmlCPUs, ncpus, flags);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
#endif
}


int
virConnectCloseWrapper(virConnectPtr conn,
                       virErrorPtr err)
{
    int ret = virConnectClose(conn);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virConnectCompareCPUWrapper(virConnectPtr conn,
                            const char *xmlDesc,
                            unsigned int flags,
                            virErrorPtr err)
{
    int ret = virConnectCompareCPU(conn, xmlDesc, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virConnectCompareHypervisorCPUWrapper(virConnectPtr conn,
                                      const char *emulator,
                                      const char *arch,
                                      const char *machine,
                                      const char *virttype,
                                      const char *xmlCPU,
                                      unsigned int flags,
                                      virErrorPtr err)
{
#if LIBVIR_VERSION_NUMBER < 4004000
    assert(0); // Caller should have checked version
#else
    int ret = virConnectCompareHypervisorCPU(conn, emulator, arch, machine, virttype, xmlCPU, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
#endif
}


char *
virConnectDomainXMLFromNativeWrapper(virConnectPtr conn,
                                     const char *nativeFormat,
                                     const char *nativeConfig,
                                     unsigned int flags,
                                     virErrorPtr err)
{
    char * ret = virConnectDomainXMLFromNative(conn, nativeFormat, nativeConfig, flags);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


char *
virConnectDomainXMLToNativeWrapper(virConnectPtr conn,
                                   const char *nativeFormat,
                                   const char *domainXml,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    char * ret = virConnectDomainXMLToNative(conn, nativeFormat, domainXml, flags);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


char *
virConnectFindStoragePoolSourcesWrapper(virConnectPtr conn,
                                        const char *type,
                                        const char *srcSpec,
                                        unsigned int flags,
                                        virErrorPtr err)
{
    char * ret = virConnectFindStoragePoolSources(conn, type, srcSpec, flags);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


int
virConnectGetAllDomainStatsWrapper(virConnectPtr conn,
                                   unsigned int stats,
                                   virDomainStatsRecordPtr **retStats,
                                   unsigned int flags,
                                   virErrorPtr err)
{
#if LIBVIR_VERSION_NUMBER < 1002008
    assert(0); // Caller should have checked version
#else
    int ret = virConnectGetAllDomainStats(conn, stats, retStats, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
#endif
}


int
virConnectGetCPUModelNamesWrapper(virConnectPtr conn,
                                  const char *arch,
                                  char ** *models,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    int ret = virConnectGetCPUModelNames(conn, arch, models, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


char *
virConnectGetCapabilitiesWrapper(virConnectPtr conn,
                                 virErrorPtr err)
{
    char * ret = virConnectGetCapabilities(conn);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


char *
virConnectGetDomainCapabilitiesWrapper(virConnectPtr conn,
                                       const char *emulatorbin,
                                       const char *arch,
                                       const char *machine,
                                       const char *virttype,
                                       unsigned int flags,
                                       virErrorPtr err)
{
#if LIBVIR_VERSION_NUMBER < 1002007
    assert(0); // Caller should have checked version
#else
    char * ret = virConnectGetDomainCapabilities(conn, emulatorbin, arch, machine, virttype, flags);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
#endif
}


char *
virConnectGetHostnameWrapper(virConnectPtr conn,
                             virErrorPtr err)
{
    char * ret = virConnectGetHostname(conn);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


int
virConnectGetLibVersionWrapper(virConnectPtr conn,
                               unsigned long *libVer,
                               virErrorPtr err)
{
    int ret = virConnectGetLibVersion(conn, libVer);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virConnectGetMaxVcpusWrapper(virConnectPtr conn,
                             const char *type,
                             virErrorPtr err)
{
    int ret = virConnectGetMaxVcpus(conn, type);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


char *
virConnectGetSysinfoWrapper(virConnectPtr conn,
                            unsigned int flags,
                            virErrorPtr err)
{
    char * ret = virConnectGetSysinfo(conn, flags);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


const char *
virConnectGetTypeWrapper(virConnectPtr conn,
                         virErrorPtr err)
{
    const char * ret = virConnectGetType(conn);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


char *
virConnectGetURIWrapper(virConnectPtr conn,
                        virErrorPtr err)
{
    char * ret = virConnectGetURI(conn);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


int
virConnectGetVersionWrapper(virConnectPtr conn,
                            unsigned long *hvVer,
                            virErrorPtr err)
{
    int ret = virConnectGetVersion(conn, hvVer);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virConnectIsAliveWrapper(virConnectPtr conn,
                         virErrorPtr err)
{
    int ret = virConnectIsAlive(conn);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virConnectIsEncryptedWrapper(virConnectPtr conn,
                             virErrorPtr err)
{
    int ret = virConnectIsEncrypted(conn);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virConnectIsSecureWrapper(virConnectPtr conn,
                          virErrorPtr err)
{
    int ret = virConnectIsSecure(conn);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virConnectListAllDomainsWrapper(virConnectPtr conn,
                                virDomainPtr **domains,
                                unsigned int flags,
                                virErrorPtr err)
{
    int ret = virConnectListAllDomains(conn, domains, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virConnectListAllInterfacesWrapper(virConnectPtr conn,
                                   virInterfacePtr **ifaces,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    int ret = virConnectListAllInterfaces(conn, ifaces, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virConnectListAllNWFilterBindingsWrapper(virConnectPtr conn,
                                         virNWFilterBindingPtr **bindings,
                                         unsigned int flags,
                                         virErrorPtr err)
{
#if LIBVIR_VERSION_NUMBER < 4005000
    assert(0); // Caller should have checked version
#else
    int ret = virConnectListAllNWFilterBindings(conn, bindings, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
#endif
}


int
virConnectListAllNWFiltersWrapper(virConnectPtr conn,
                                  virNWFilterPtr **filters,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    int ret = virConnectListAllNWFilters(conn, filters, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virConnectListAllNetworksWrapper(virConnectPtr conn,
                                 virNetworkPtr **nets,
                                 unsigned int flags,
                                 virErrorPtr err)
{
    int ret = virConnectListAllNetworks(conn, nets, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virConnectListAllNodeDevicesWrapper(virConnectPtr conn,
                                    virNodeDevicePtr **devices,
                                    unsigned int flags,
                                    virErrorPtr err)
{
    int ret = virConnectListAllNodeDevices(conn, devices, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virConnectListAllSecretsWrapper(virConnectPtr conn,
                                virSecretPtr **secrets,
                                unsigned int flags,
                                virErrorPtr err)
{
    int ret = virConnectListAllSecrets(conn, secrets, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virConnectListAllStoragePoolsWrapper(virConnectPtr conn,
                                     virStoragePoolPtr **pools,
                                     unsigned int flags,
                                     virErrorPtr err)
{
    int ret = virConnectListAllStoragePools(conn, pools, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virConnectListDefinedDomainsWrapper(virConnectPtr conn,
                                    char ** const names,
                                    int maxnames,
                                    virErrorPtr err)
{
    int ret = virConnectListDefinedDomains(conn, names, maxnames);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virConnectListDefinedInterfacesWrapper(virConnectPtr conn,
                                       char ** const names,
                                       int maxnames,
                                       virErrorPtr err)
{
    int ret = virConnectListDefinedInterfaces(conn, names, maxnames);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virConnectListDefinedNetworksWrapper(virConnectPtr conn,
                                     char ** const names,
                                     int maxnames,
                                     virErrorPtr err)
{
    int ret = virConnectListDefinedNetworks(conn, names, maxnames);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virConnectListDefinedStoragePoolsWrapper(virConnectPtr conn,
                                         char ** const names,
                                         int maxnames,
                                         virErrorPtr err)
{
    int ret = virConnectListDefinedStoragePools(conn, names, maxnames);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virConnectListDomainsWrapper(virConnectPtr conn,
                             int *ids,
                             int maxids,
                             virErrorPtr err)
{
    int ret = virConnectListDomains(conn, ids, maxids);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virConnectListInterfacesWrapper(virConnectPtr conn,
                                char ** const names,
                                int maxnames,
                                virErrorPtr err)
{
    int ret = virConnectListInterfaces(conn, names, maxnames);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virConnectListNWFiltersWrapper(virConnectPtr conn,
                               char ** const names,
                               int maxnames,
                               virErrorPtr err)
{
    int ret = virConnectListNWFilters(conn, names, maxnames);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virConnectListNetworksWrapper(virConnectPtr conn,
                              char ** const names,
                              int maxnames,
                              virErrorPtr err)
{
    int ret = virConnectListNetworks(conn, names, maxnames);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virConnectListSecretsWrapper(virConnectPtr conn,
                             char **uuids,
                             int maxuuids,
                             virErrorPtr err)
{
    int ret = virConnectListSecrets(conn, uuids, maxuuids);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virConnectListStoragePoolsWrapper(virConnectPtr conn,
                                  char ** const names,
                                  int maxnames,
                                  virErrorPtr err)
{
    int ret = virConnectListStoragePools(conn, names, maxnames);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virConnectNumOfDefinedDomainsWrapper(virConnectPtr conn,
                                     virErrorPtr err)
{
    int ret = virConnectNumOfDefinedDomains(conn);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virConnectNumOfDefinedInterfacesWrapper(virConnectPtr conn,
                                        virErrorPtr err)
{
    int ret = virConnectNumOfDefinedInterfaces(conn);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virConnectNumOfDefinedNetworksWrapper(virConnectPtr conn,
                                      virErrorPtr err)
{
    int ret = virConnectNumOfDefinedNetworks(conn);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virConnectNumOfDefinedStoragePoolsWrapper(virConnectPtr conn,
                                          virErrorPtr err)
{
    int ret = virConnectNumOfDefinedStoragePools(conn);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virConnectNumOfDomainsWrapper(virConnectPtr conn,
                              virErrorPtr err)
{
    int ret = virConnectNumOfDomains(conn);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virConnectNumOfInterfacesWrapper(virConnectPtr conn,
                                 virErrorPtr err)
{
    int ret = virConnectNumOfInterfaces(conn);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virConnectNumOfNWFiltersWrapper(virConnectPtr conn,
                                virErrorPtr err)
{
    int ret = virConnectNumOfNWFilters(conn);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virConnectNumOfNetworksWrapper(virConnectPtr conn,
                               virErrorPtr err)
{
    int ret = virConnectNumOfNetworks(conn);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virConnectNumOfSecretsWrapper(virConnectPtr conn,
                              virErrorPtr err)
{
    int ret = virConnectNumOfSecrets(conn);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virConnectNumOfStoragePoolsWrapper(virConnectPtr conn,
                                   virErrorPtr err)
{
    int ret = virConnectNumOfStoragePools(conn);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


virConnectPtr
virConnectOpenWrapper(const char *name,
                      virErrorPtr err)
{
    virConnectPtr ret = virConnectOpen(name);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


virConnectPtr
virConnectOpenAuthWrapper(const char *name,
                          int *credtype,
                          unsigned int ncredtype,
                          int callbackID,
                          unsigned int flags,
                          virErrorPtr err)
{
    virConnectAuth auth = {
       .credtype = credtype,
       .ncredtype = ncredtype,
       .cb = connectAuthCallbackHelper,
       .cbdata = &callbackID,
    };

    virConnectPtr ret = virConnectOpenAuth(name, &auth, flags);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


virConnectPtr
virConnectOpenReadOnlyWrapper(const char *name,
                              virErrorPtr err)
{
    virConnectPtr ret = virConnectOpenReadOnly(name);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


int
virConnectRefWrapper(virConnectPtr conn,
                     virErrorPtr err)
{
    int ret = virConnectRef(conn);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virConnectRegisterCloseCallbackWrapper(virConnectPtr conn,
                                       long goCallbackId,
                                       virErrorPtr err)
{
    void *id = (void*)goCallbackId;
    int ret = virConnectRegisterCloseCallback(conn, closeCallbackHelper, id, freeGoCallbackHelper);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virConnectSetKeepAliveWrapper(virConnectPtr conn,
                              int interval,
                              unsigned int count,
                              virErrorPtr err)
{
    int ret = virConnectSetKeepAlive(conn, interval, count);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virConnectUnregisterCloseCallbackWrapper(virConnectPtr conn,
                                         virErrorPtr err)
{
    int ret = virConnectUnregisterCloseCallback(conn, closeCallbackHelper);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


virDomainPtr
virDomainCreateLinuxWrapper(virConnectPtr conn,
                            const char *xmlDesc,
                            unsigned int flags,
                            virErrorPtr err)
{
    virDomainPtr ret = virDomainCreateLinux(conn, xmlDesc, flags);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


virDomainPtr
virDomainCreateXMLWrapper(virConnectPtr conn,
                          const char *xmlDesc,
                          unsigned int flags,
                          virErrorPtr err)
{
    virDomainPtr ret = virDomainCreateXML(conn, xmlDesc, flags);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


virDomainPtr
virDomainCreateXMLWithFilesWrapper(virConnectPtr conn,
                                   const char *xmlDesc,
                                   unsigned int nfiles,
                                   int *files,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    virDomainPtr ret = virDomainCreateXMLWithFiles(conn, xmlDesc, nfiles, files, flags);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


virDomainPtr
virDomainDefineXMLWrapper(virConnectPtr conn,
                          const char *xml,
                          virErrorPtr err)
{
    virDomainPtr ret = virDomainDefineXML(conn, xml);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


virDomainPtr
virDomainDefineXMLFlagsWrapper(virConnectPtr conn,
                               const char *xml,
                               unsigned int flags,
                               virErrorPtr err)
{
#if LIBVIR_VERSION_NUMBER < 1002012
    assert(0); // Caller should have checked version
#else
    virDomainPtr ret = virDomainDefineXMLFlags(conn, xml, flags);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
#endif
}


int
virDomainListGetStatsWrapper(virDomainPtr *doms,
                             unsigned int stats,
                             virDomainStatsRecordPtr **retStats,
                             unsigned int flags,
                             virErrorPtr err)
{
#if LIBVIR_VERSION_NUMBER < 1002008
    assert(0); // Caller should have checked version
#else
    int ret = virDomainListGetStats(doms, stats, retStats, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
#endif
}


virDomainPtr
virDomainLookupByIDWrapper(virConnectPtr conn,
                           int id,
                           virErrorPtr err)
{
    virDomainPtr ret = virDomainLookupByID(conn, id);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


virDomainPtr
virDomainLookupByNameWrapper(virConnectPtr conn,
                             const char *name,
                             virErrorPtr err)
{
    virDomainPtr ret = virDomainLookupByName(conn, name);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


virDomainPtr
virDomainLookupByUUIDWrapper(virConnectPtr conn,
                             const unsigned char *uuid,
                             virErrorPtr err)
{
    virDomainPtr ret = virDomainLookupByUUID(conn, uuid);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


virDomainPtr
virDomainLookupByUUIDStringWrapper(virConnectPtr conn,
                                   const char *uuidstr,
                                   virErrorPtr err)
{
    virDomainPtr ret = virDomainLookupByUUIDString(conn, uuidstr);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainRestoreWrapper(virConnectPtr conn,
                        const char *from,
                        virErrorPtr err)
{
    int ret = virDomainRestore(conn, from);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainRestoreFlagsWrapper(virConnectPtr conn,
                             const char *from,
                             const char *dxml,
                             unsigned int flags,
                             virErrorPtr err)
{
    int ret = virDomainRestoreFlags(conn, from, dxml, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainSaveImageDefineXMLWrapper(virConnectPtr conn,
                                   const char *file,
                                   const char *dxml,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    int ret = virDomainSaveImageDefineXML(conn, file, dxml, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


char *
virDomainSaveImageGetXMLDescWrapper(virConnectPtr conn,
                                    const char *file,
                                    unsigned int flags,
                                    virErrorPtr err)
{
    char * ret = virDomainSaveImageGetXMLDesc(conn, file, flags);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


void
virDomainStatsRecordListFreeWrapper(virDomainStatsRecordPtr *stats)
{
#if LIBVIR_VERSION_NUMBER < 1002008
    assert(0); // Caller should have checked version
#else
    virDomainStatsRecordListFree(stats);
#endif
}


int
virGetVersionWrapper(unsigned long *libVer,
                     const char *type,
                     unsigned long *typeVer,
                     virErrorPtr err)
{
    int ret = virGetVersion(libVer, type, typeVer);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virInterfaceChangeBeginWrapper(virConnectPtr conn,
                               unsigned int flags,
                               virErrorPtr err)
{
    int ret = virInterfaceChangeBegin(conn, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virInterfaceChangeCommitWrapper(virConnectPtr conn,
                                unsigned int flags,
                                virErrorPtr err)
{
    int ret = virInterfaceChangeCommit(conn, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virInterfaceChangeRollbackWrapper(virConnectPtr conn,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    int ret = virInterfaceChangeRollback(conn, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


virInterfacePtr
virInterfaceDefineXMLWrapper(virConnectPtr conn,
                             const char *xml,
                             unsigned int flags,
                             virErrorPtr err)
{
    virInterfacePtr ret = virInterfaceDefineXML(conn, xml, flags);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


virInterfacePtr
virInterfaceLookupByMACStringWrapper(virConnectPtr conn,
                                     const char *macstr,
                                     virErrorPtr err)
{
    virInterfacePtr ret = virInterfaceLookupByMACString(conn, macstr);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


virInterfacePtr
virInterfaceLookupByNameWrapper(virConnectPtr conn,
                                const char *name,
                                virErrorPtr err)
{
    virInterfacePtr ret = virInterfaceLookupByName(conn, name);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


virNWFilterBindingPtr
virNWFilterBindingCreateXMLWrapper(virConnectPtr conn,
                                   const char *xml,
                                   unsigned int flags,
                                   virErrorPtr err)
{
#if LIBVIR_VERSION_NUMBER < 4005000
    assert(0); // Caller should have checked version
#else
    virNWFilterBindingPtr ret = virNWFilterBindingCreateXML(conn, xml, flags);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
#endif
}


virNWFilterBindingPtr
virNWFilterBindingLookupByPortDevWrapper(virConnectPtr conn,
                                         const char *portdev,
                                         virErrorPtr err)
{
#if LIBVIR_VERSION_NUMBER < 4005000
    assert(0); // Caller should have checked version
#else
    virNWFilterBindingPtr ret = virNWFilterBindingLookupByPortDev(conn, portdev);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
#endif
}


virNWFilterPtr
virNWFilterDefineXMLWrapper(virConnectPtr conn,
                            const char *xmlDesc,
                            virErrorPtr err)
{
    virNWFilterPtr ret = virNWFilterDefineXML(conn, xmlDesc);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


virNWFilterPtr
virNWFilterLookupByNameWrapper(virConnectPtr conn,
                               const char *name,
                               virErrorPtr err)
{
    virNWFilterPtr ret = virNWFilterLookupByName(conn, name);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


virNWFilterPtr
virNWFilterLookupByUUIDWrapper(virConnectPtr conn,
                               const unsigned char *uuid,
                               virErrorPtr err)
{
    virNWFilterPtr ret = virNWFilterLookupByUUID(conn, uuid);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


virNWFilterPtr
virNWFilterLookupByUUIDStringWrapper(virConnectPtr conn,
                                     const char *uuidstr,
                                     virErrorPtr err)
{
    virNWFilterPtr ret = virNWFilterLookupByUUIDString(conn, uuidstr);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


virNetworkPtr
virNetworkCreateXMLWrapper(virConnectPtr conn,
                           const char *xmlDesc,
                           virErrorPtr err)
{
    virNetworkPtr ret = virNetworkCreateXML(conn, xmlDesc);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


virNetworkPtr
virNetworkDefineXMLWrapper(virConnectPtr conn,
                           const char *xml,
                           virErrorPtr err)
{
    virNetworkPtr ret = virNetworkDefineXML(conn, xml);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


virNetworkPtr
virNetworkLookupByNameWrapper(virConnectPtr conn,
                              const char *name,
                              virErrorPtr err)
{
    virNetworkPtr ret = virNetworkLookupByName(conn, name);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


virNetworkPtr
virNetworkLookupByUUIDWrapper(virConnectPtr conn,
                              const unsigned char *uuid,
                              virErrorPtr err)
{
    virNetworkPtr ret = virNetworkLookupByUUID(conn, uuid);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


virNetworkPtr
virNetworkLookupByUUIDStringWrapper(virConnectPtr conn,
                                    const char *uuidstr,
                                    virErrorPtr err)
{
    virNetworkPtr ret = virNetworkLookupByUUIDString(conn, uuidstr);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


int
virNodeAllocPagesWrapper(virConnectPtr conn,
                         unsigned int npages,
                         unsigned int *pageSizes,
                         unsigned long long *pageCounts,
                         int startCell,
                         unsigned int cellCount,
                         unsigned int flags,
                         virErrorPtr err)
{
#if LIBVIR_VERSION_NUMBER < 1002009
    assert(0); // Caller should have checked version
#else
    int ret = virNodeAllocPages(conn, npages, pageSizes, pageCounts, startCell, cellCount, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
#endif
}


virNodeDevicePtr
virNodeDeviceCreateXMLWrapper(virConnectPtr conn,
                              const char *xmlDesc,
                              unsigned int flags,
                              virErrorPtr err)
{
    virNodeDevicePtr ret = virNodeDeviceCreateXML(conn, xmlDesc, flags);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


virNodeDevicePtr
virNodeDeviceLookupByNameWrapper(virConnectPtr conn,
                                 const char *name,
                                 virErrorPtr err)
{
    virNodeDevicePtr ret = virNodeDeviceLookupByName(conn, name);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


virNodeDevicePtr
virNodeDeviceLookupSCSIHostByWWNWrapper(virConnectPtr conn,
                                        const char *wwnn,
                                        const char *wwpn,
                                        unsigned int flags,
                                        virErrorPtr err)
{
    virNodeDevicePtr ret = virNodeDeviceLookupSCSIHostByWWN(conn, wwnn, wwpn, flags);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


int
virNodeGetCPUMapWrapper(virConnectPtr conn,
                        unsigned char **cpumap,
                        unsigned int *online,
                        unsigned int flags,
                        virErrorPtr err)
{
    int ret = virNodeGetCPUMap(conn, cpumap, online, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virNodeGetCPUStatsWrapper(virConnectPtr conn,
                          int cpuNum,
                          virNodeCPUStatsPtr params,
                          int *nparams,
                          unsigned int flags,
                          virErrorPtr err)
{
    int ret = virNodeGetCPUStats(conn, cpuNum, params, nparams, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virNodeGetCellsFreeMemoryWrapper(virConnectPtr conn,
                                 unsigned long long *freeMems,
                                 int startCell,
                                 int maxCells,
                                 virErrorPtr err)
{
    int ret = virNodeGetCellsFreeMemory(conn, freeMems, startCell, maxCells);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


unsigned long long
virNodeGetFreeMemoryWrapper(virConnectPtr conn,
                            virErrorPtr err)
{
    unsigned long long ret = virNodeGetFreeMemory(conn);
    if (ret == 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virNodeGetFreePagesWrapper(virConnectPtr conn,
                           unsigned int npages,
                           unsigned int *pages,
                           int startCell,
                           unsigned int cellCount,
                           unsigned long long *counts,
                           unsigned int flags,
                           virErrorPtr err)
{
#if LIBVIR_VERSION_NUMBER < 1002006
    assert(0); // Caller should have checked version
#else
    int ret = virNodeGetFreePages(conn, npages, pages, startCell, cellCount, counts, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
#endif
}


int
virNodeGetInfoWrapper(virConnectPtr conn,
                      virNodeInfoPtr info,
                      virErrorPtr err)
{
    int ret = virNodeGetInfo(conn, info);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virNodeGetMemoryParametersWrapper(virConnectPtr conn,
                                  virTypedParameterPtr params,
                                  int *nparams,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    int ret = virNodeGetMemoryParameters(conn, params, nparams, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virNodeGetMemoryStatsWrapper(virConnectPtr conn,
                             int cellNum,
                             virNodeMemoryStatsPtr params,
                             int *nparams,
                             unsigned int flags,
                             virErrorPtr err)
{
    int ret = virNodeGetMemoryStats(conn, cellNum, params, nparams, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virNodeGetSEVInfoWrapper(virConnectPtr conn,
                         virTypedParameterPtr *params,
                         int *nparams,
                         unsigned int flags,
                         virErrorPtr err)
{
#if LIBVIR_VERSION_NUMBER < 4005000
    assert(0); // Caller should have checked version
#else
    int ret = virNodeGetSEVInfo(conn, params, nparams, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
#endif
}


int
virNodeGetSecurityModelWrapper(virConnectPtr conn,
                               virSecurityModelPtr secmodel,
                               virErrorPtr err)
{
    int ret = virNodeGetSecurityModel(conn, secmodel);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virNodeListDevicesWrapper(virConnectPtr conn,
                          const char *cap,
                          char ** const names,
                          int maxnames,
                          unsigned int flags,
                          virErrorPtr err)
{
    int ret = virNodeListDevices(conn, cap, names, maxnames, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virNodeNumOfDevicesWrapper(virConnectPtr conn,
                           const char *cap,
                           unsigned int flags,
                           virErrorPtr err)
{
    int ret = virNodeNumOfDevices(conn, cap, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virNodeSetMemoryParametersWrapper(virConnectPtr conn,
                                  virTypedParameterPtr params,
                                  int nparams,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    int ret = virNodeSetMemoryParameters(conn, params, nparams, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virNodeSuspendForDurationWrapper(virConnectPtr conn,
                                 unsigned int target,
                                 unsigned long long duration,
                                 unsigned int flags,
                                 virErrorPtr err)
{
    int ret = virNodeSuspendForDuration(conn, target, duration, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


virSecretPtr
virSecretDefineXMLWrapper(virConnectPtr conn,
                          const char *xml,
                          unsigned int flags,
                          virErrorPtr err)
{
    virSecretPtr ret = virSecretDefineXML(conn, xml, flags);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


virSecretPtr
virSecretLookupByUUIDWrapper(virConnectPtr conn,
                             const unsigned char *uuid,
                             virErrorPtr err)
{
    virSecretPtr ret = virSecretLookupByUUID(conn, uuid);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


virSecretPtr
virSecretLookupByUUIDStringWrapper(virConnectPtr conn,
                                   const char *uuidstr,
                                   virErrorPtr err)
{
    virSecretPtr ret = virSecretLookupByUUIDString(conn, uuidstr);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


virSecretPtr
virSecretLookupByUsageWrapper(virConnectPtr conn,
                              int usageType,
                              const char *usageID,
                              virErrorPtr err)
{
    virSecretPtr ret = virSecretLookupByUsage(conn, usageType, usageID);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


virStoragePoolPtr
virStoragePoolCreateXMLWrapper(virConnectPtr conn,
                               const char *xmlDesc,
                               unsigned int flags,
                               virErrorPtr err)
{
    virStoragePoolPtr ret = virStoragePoolCreateXML(conn, xmlDesc, flags);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


virStoragePoolPtr
virStoragePoolDefineXMLWrapper(virConnectPtr conn,
                               const char *xml,
                               unsigned int flags,
                               virErrorPtr err)
{
    virStoragePoolPtr ret = virStoragePoolDefineXML(conn, xml, flags);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


virStoragePoolPtr
virStoragePoolLookupByNameWrapper(virConnectPtr conn,
                                  const char *name,
                                  virErrorPtr err)
{
    virStoragePoolPtr ret = virStoragePoolLookupByName(conn, name);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


virStoragePoolPtr
virStoragePoolLookupByTargetPathWrapper(virConnectPtr conn,
                                        const char *path,
                                        virErrorPtr err)
{
#if LIBVIR_VERSION_NUMBER < 4001000
    assert(0); // Caller should have checked version
#else
    virStoragePoolPtr ret = virStoragePoolLookupByTargetPath(conn, path);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
#endif
}


virStoragePoolPtr
virStoragePoolLookupByUUIDWrapper(virConnectPtr conn,
                                  const unsigned char *uuid,
                                  virErrorPtr err)
{
    virStoragePoolPtr ret = virStoragePoolLookupByUUID(conn, uuid);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


virStoragePoolPtr
virStoragePoolLookupByUUIDStringWrapper(virConnectPtr conn,
                                        const char *uuidstr,
                                        virErrorPtr err)
{
    virStoragePoolPtr ret = virStoragePoolLookupByUUIDString(conn, uuidstr);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


virStorageVolPtr
virStorageVolLookupByKeyWrapper(virConnectPtr conn,
                                const char *key,
                                virErrorPtr err)
{
    virStorageVolPtr ret = virStorageVolLookupByKey(conn, key);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


virStorageVolPtr
virStorageVolLookupByPathWrapper(virConnectPtr conn,
                                 const char *path,
                                 virErrorPtr err)
{
    virStorageVolPtr ret = virStorageVolLookupByPath(conn, path);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


virStreamPtr
virStreamNewWrapper(virConnectPtr conn,
                    unsigned int flags,
                    virErrorPtr err)
{
    virStreamPtr ret = virStreamNew(conn, flags);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


char *
virConnectGetStoragePoolCapabilitiesWrapper(virConnectPtr conn,
                                            unsigned int flags,
                                            virErrorPtr err)
{
#if LIBVIR_VERSION_NUMBER < 5002000
    assert(0); // Caller should have checked version
#else
    char *ret = virConnectGetStoragePoolCapabilities(conn, flags);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
#endif
}

int
virConnectSetIdentityWrapper(virConnectPtr conn,
			     virTypedParameterPtr params,
			     int nparams,
			     unsigned int flags,
			     virErrorPtr err)
{
#if LIBVIR_VERSION_NUMBER < 5008000
    assert(0); // Caller should have checked version
#else
    int ret = virConnectSetIdentity(conn, params, nparams, flags);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
#endif
}

////////////////////////////////////////////////
*/
import "C"
