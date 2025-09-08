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

#if !LIBVIR_CHECK_VERSION(0, 0, 1)
typedef struct _virConnect virConnect;
#endif

#if !LIBVIR_CHECK_VERSION(0, 4, 1)
typedef struct _virConnectAuth virConnectAuth;
#endif

#if !LIBVIR_CHECK_VERSION(0, 4, 1)
typedef struct _virConnectCredential virConnectCredential;
#endif

#if !LIBVIR_CHECK_VERSION(0, 0, 1)
typedef struct _virDomain virDomain;
#endif

#if !LIBVIR_CHECK_VERSION(0, 8, 1)
typedef struct _virDomainBlockInfo virDomainBlockInfo;
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 4)
typedef struct _virDomainBlockJobInfo virDomainBlockJobInfo;
#endif

#if !LIBVIR_CHECK_VERSION(0, 3, 3)
typedef struct _virDomainBlockStats virDomainBlockStatsStruct;
#endif

#if !LIBVIR_CHECK_VERSION(5, 2, 0)
typedef struct _virDomainCheckpoint virDomainCheckpoint;
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 3)
typedef struct _virDomainControlInfo virDomainControlInfo;
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 10)
typedef struct _virDomainDiskError virDomainDiskError;
#endif

#if !LIBVIR_CHECK_VERSION(0, 8, 0)
typedef struct _virDomainEventGraphicsAddress virDomainEventGraphicsAddress;
#endif

#if !LIBVIR_CHECK_VERSION(0, 8, 0)
typedef struct _virDomainEventGraphicsSubject virDomainEventGraphicsSubject;
#endif

#if !LIBVIR_CHECK_VERSION(0, 8, 0)
typedef struct _virDomainEventGraphicsSubjectIdentity virDomainEventGraphicsSubjectIdentity;
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 11)
typedef struct _virDomainFSInfo virDomainFSInfo;
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 14)
typedef struct _virDomainIOThreadInfo virDomainIOThreadInfo;
#endif

#if !LIBVIR_CHECK_VERSION(0, 0, 1)
typedef struct _virDomainInfo virDomainInfo;
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 14)
typedef struct _virDomainInterface virDomainInterface;
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 14)
typedef struct _virDomainInterfaceIPAddress virDomainIPAddress;
#endif

#if !LIBVIR_CHECK_VERSION(0, 3, 3)
typedef struct _virDomainInterfaceStats virDomainInterfaceStatsStruct;
#endif

#if !LIBVIR_CHECK_VERSION(0, 7, 7)
typedef struct _virDomainJobInfo virDomainJobInfo;
#endif

#if !LIBVIR_CHECK_VERSION(0, 7, 5)
typedef struct _virDomainMemoryStat virDomainMemoryStatStruct;
#endif

#if !LIBVIR_CHECK_VERSION(0, 8, 0)
typedef struct _virDomainSnapshot virDomainSnapshot;
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 8)
typedef struct _virDomainStatsRecord virDomainStatsRecord;
#endif

#if !LIBVIR_CHECK_VERSION(0, 1, 0)
typedef struct _virError virError;
#endif

#if !LIBVIR_CHECK_VERSION(0, 6, 4)
typedef struct _virInterface virInterface;
#endif

#if !LIBVIR_CHECK_VERSION(0, 8, 0)
typedef struct _virNWFilter virNWFilter;
#endif

#if !LIBVIR_CHECK_VERSION(4, 5, 0)
typedef struct _virNWFilterBinding virNWFilterBinding;
#endif

#if !LIBVIR_CHECK_VERSION(0, 2, 0)
typedef struct _virNetwork virNetwork;
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 6)
typedef struct _virNetworkDHCPLease virNetworkDHCPLease;
#endif

#if !LIBVIR_CHECK_VERSION(5, 5, 0)
typedef struct _virNetworkPort virNetworkPort;
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 3)
typedef struct _virNodeCPUStats virNodeCPUStats;
#endif

#if !LIBVIR_CHECK_VERSION(0, 5, 0)
typedef struct _virNodeDevice virNodeDevice;
#endif

#if !LIBVIR_CHECK_VERSION(0, 1, 0)
typedef struct _virNodeInfo virNodeInfo;
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 3)
typedef struct _virNodeMemoryStats virNodeMemoryStats;
#endif

#if !LIBVIR_CHECK_VERSION(0, 7, 1)
typedef struct _virSecret virSecret;
#endif

#if !LIBVIR_CHECK_VERSION(0, 6, 1)
typedef struct _virSecurityLabel virSecurityLabel;
#endif

#if !LIBVIR_CHECK_VERSION(0, 6, 1)
typedef struct _virSecurityModel virSecurityModel;
#endif

#if !LIBVIR_CHECK_VERSION(0, 4, 1)
typedef struct _virStoragePool virStoragePool;
#endif

#if !LIBVIR_CHECK_VERSION(0, 4, 1)
typedef struct _virStoragePoolInfo virStoragePoolInfo;
#endif

#if !LIBVIR_CHECK_VERSION(0, 4, 1)
typedef struct _virStorageVol virStorageVol;
#endif

#if !LIBVIR_CHECK_VERSION(0, 4, 1)
typedef struct _virStorageVolInfo virStorageVolInfo;
#endif

#if !LIBVIR_CHECK_VERSION(0, 7, 2)
typedef struct _virStream virStream;
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 0)
typedef struct _virTypedParameter virBlkioParameter;
#endif

#if !LIBVIR_CHECK_VERSION(0, 8, 5)
typedef struct _virTypedParameter virMemoryParameter;
#endif

#if !LIBVIR_CHECK_VERSION(0, 2, 3)
typedef struct _virTypedParameter virSchedParameter;
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 2)
typedef struct _virTypedParameter virTypedParameter;
#endif

#if !LIBVIR_CHECK_VERSION(0, 1, 4)
typedef struct _virVcpuInfo virVcpuInfo;
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 0)
typedef virBlkioParameter * virBlkioParameterPtr;
#endif

#if !LIBVIR_CHECK_VERSION(0, 4, 1)
typedef virConnectAuth * virConnectAuthPtr;
#endif

#if !LIBVIR_CHECK_VERSION(0, 4, 1)
typedef virConnectCredential * virConnectCredentialPtr;
#endif

#if !LIBVIR_CHECK_VERSION(0, 0, 1)
typedef virConnect * virConnectPtr;
#endif

#if !LIBVIR_CHECK_VERSION(0, 8, 1)
typedef virDomainBlockInfo * virDomainBlockInfoPtr;
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 4)
typedef unsigned long long virDomainBlockJobCursor;
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 4)
typedef virDomainBlockJobInfo * virDomainBlockJobInfoPtr;
#endif

#if !LIBVIR_CHECK_VERSION(0, 3, 2)
typedef virDomainBlockStatsStruct * virDomainBlockStatsPtr;
#endif

#if !LIBVIR_CHECK_VERSION(5, 2, 0)
typedef virDomainCheckpoint * virDomainCheckpointPtr;
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 3)
typedef virDomainControlInfo * virDomainControlInfoPtr;
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 10)
typedef virDomainDiskError * virDomainDiskErrorPtr;
#endif

#if !LIBVIR_CHECK_VERSION(0, 8, 0)
typedef virDomainEventGraphicsAddress * virDomainEventGraphicsAddressPtr;
#endif

#if !LIBVIR_CHECK_VERSION(0, 8, 0)
typedef virDomainEventGraphicsSubjectIdentity * virDomainEventGraphicsSubjectIdentityPtr;
#endif

#if !LIBVIR_CHECK_VERSION(0, 8, 0)
typedef virDomainEventGraphicsSubject * virDomainEventGraphicsSubjectPtr;
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 11)
typedef virDomainFSInfo * virDomainFSInfoPtr;
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 14)
typedef virDomainIOThreadInfo * virDomainIOThreadInfoPtr;
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 14)
typedef virDomainIPAddress * virDomainIPAddressPtr;
#endif

#if !LIBVIR_CHECK_VERSION(0, 0, 1)
typedef virDomainInfo * virDomainInfoPtr;
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 14)
typedef virDomainInterface * virDomainInterfacePtr;
#endif

#if !LIBVIR_CHECK_VERSION(0, 3, 2)
typedef virDomainInterfaceStatsStruct * virDomainInterfaceStatsPtr;
#endif

#if !LIBVIR_CHECK_VERSION(0, 7, 7)
typedef virDomainJobInfo * virDomainJobInfoPtr;
#endif

#if !LIBVIR_CHECK_VERSION(0, 7, 5)
typedef virDomainMemoryStatStruct * virDomainMemoryStatPtr;
#endif

#if !LIBVIR_CHECK_VERSION(0, 0, 1)
typedef virDomain * virDomainPtr;
#endif

#if !LIBVIR_CHECK_VERSION(0, 8, 0)
typedef virDomainSnapshot * virDomainSnapshotPtr;
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 8)
typedef virDomainStatsRecord * virDomainStatsRecordPtr;
#endif

#if !LIBVIR_CHECK_VERSION(0, 1, 0)
typedef int virErrorLevel;
#endif

#if !LIBVIR_CHECK_VERSION(0, 1, 0)
typedef virError * virErrorPtr;
#endif

#if !LIBVIR_CHECK_VERSION(0, 6, 4)
typedef virInterface * virInterfacePtr;
#endif

#if !LIBVIR_CHECK_VERSION(0, 8, 5)
typedef virMemoryParameter * virMemoryParameterPtr;
#endif

#if !LIBVIR_CHECK_VERSION(4, 5, 0)
typedef virNWFilterBinding * virNWFilterBindingPtr;
#endif

#if !LIBVIR_CHECK_VERSION(0, 8, 0)
typedef virNWFilter * virNWFilterPtr;
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 6)
typedef virNetworkDHCPLease * virNetworkDHCPLeasePtr;
#endif

#if !LIBVIR_CHECK_VERSION(5, 5, 0)
typedef virNetworkPort * virNetworkPortPtr;
#endif

#if !LIBVIR_CHECK_VERSION(0, 2, 0)
typedef virNetwork * virNetworkPtr;
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 3)
typedef virNodeCPUStats * virNodeCPUStatsPtr;
#endif

#if !LIBVIR_CHECK_VERSION(0, 5, 0)
typedef virNodeDevice * virNodeDevicePtr;
#endif

#if !LIBVIR_CHECK_VERSION(0, 1, 0)
typedef virNodeInfo * virNodeInfoPtr;
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 3)
typedef virNodeMemoryStats * virNodeMemoryStatsPtr;
#endif

#if !LIBVIR_CHECK_VERSION(0, 2, 3)
typedef virSchedParameter * virSchedParameterPtr;
#endif

#if !LIBVIR_CHECK_VERSION(0, 7, 1)
typedef virSecret * virSecretPtr;
#endif

#if !LIBVIR_CHECK_VERSION(0, 6, 1)
typedef virSecurityLabel * virSecurityLabelPtr;
#endif

#if !LIBVIR_CHECK_VERSION(0, 6, 1)
typedef virSecurityModel * virSecurityModelPtr;
#endif

#if !LIBVIR_CHECK_VERSION(0, 4, 1)
typedef virStoragePoolInfo * virStoragePoolInfoPtr;
#endif

#if !LIBVIR_CHECK_VERSION(0, 4, 1)
typedef virStoragePool * virStoragePoolPtr;
#endif

#if !LIBVIR_CHECK_VERSION(0, 4, 1)
typedef virStorageVolInfo * virStorageVolInfoPtr;
#endif

#if !LIBVIR_CHECK_VERSION(0, 4, 1)
typedef virStorageVol * virStorageVolPtr;
#endif

#if !LIBVIR_CHECK_VERSION(0, 7, 2)
typedef virStream * virStreamPtr;
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 2)
typedef virTypedParameter * virTypedParameterPtr;
#endif

#if !LIBVIR_CHECK_VERSION(0, 1, 4)
typedef virVcpuInfo * virVcpuInfoPtr;
#endif
