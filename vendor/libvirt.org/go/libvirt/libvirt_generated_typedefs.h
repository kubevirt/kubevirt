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

#if !LIBVIR_CHECK_VERSION(0, 9, 0)
typedef int virBlkioParameterType;
#endif

#if !LIBVIR_CHECK_VERSION(0, 7, 5)
typedef int virCPUCompareResult;
#endif

#if !LIBVIR_CHECK_VERSION(0, 4, 1)
typedef virConnectAuth * virConnectAuthPtr;
#endif

#if !LIBVIR_CHECK_VERSION(1, 1, 2)
typedef int virConnectBaselineCPUFlags;
#endif

#if !LIBVIR_CHECK_VERSION(0, 10, 0)
typedef int virConnectCloseReason;
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 6)
typedef int virConnectCompareCPUFlags;
#endif

#if !LIBVIR_CHECK_VERSION(0, 4, 1)
typedef virConnectCredential * virConnectCredentialPtr;
#endif

#if !LIBVIR_CHECK_VERSION(0, 4, 1)
typedef int virConnectCredentialType;
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 11)
typedef int virConnectDomainEventAgentLifecycleReason;
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 11)
typedef int virConnectDomainEventAgentLifecycleState;
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 4)
typedef int virConnectDomainEventBlockJobStatus;
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 7)
typedef int virConnectDomainEventDiskChangeReason;
#endif

#if !LIBVIR_CHECK_VERSION(0, 4, 1)
typedef int virConnectFlags;
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 8)
typedef int virConnectGetAllDomainStatsFlags;
#endif

#if !LIBVIR_CHECK_VERSION(11, 0, 0)
typedef int virConnectGetDomainCapabilitiesFlags;
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 13)
typedef int virConnectListAllDomainsFlags;
#endif

#if !LIBVIR_CHECK_VERSION(0, 10, 2)
typedef int virConnectListAllInterfacesFlags;
#endif

#if !LIBVIR_CHECK_VERSION(0, 10, 2)
typedef int virConnectListAllNetworksFlags;
#endif

#if !LIBVIR_CHECK_VERSION(0, 10, 2)
typedef int virConnectListAllNodeDeviceFlags;
#endif

#if !LIBVIR_CHECK_VERSION(0, 10, 2)
typedef int virConnectListAllSecretsFlags;
#endif

#if !LIBVIR_CHECK_VERSION(0, 10, 2)
typedef int virConnectListAllStoragePoolsFlags;
#endif

#if !LIBVIR_CHECK_VERSION(0, 0, 1)
typedef virConnect * virConnectPtr;
#endif

#if !LIBVIR_CHECK_VERSION(8, 5, 0)
typedef int virDomainAbortJobFlagsValues;
#endif

#if !LIBVIR_CHECK_VERSION(5, 10, 0)
typedef int virDomainAgentResponseTimeoutValues;
#endif

#if !LIBVIR_CHECK_VERSION(6, 10, 0)
typedef int virDomainAuthorizedSSHKeysSetFlags;
#endif

#if !LIBVIR_CHECK_VERSION(6, 0, 0)
typedef int virDomainBackupBeginFlags;
#endif

#if !LIBVIR_CHECK_VERSION(0, 10, 2)
typedef int virDomainBlockCommitFlags;
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 8)
typedef int virDomainBlockCopyFlags;
#endif

#if !LIBVIR_CHECK_VERSION(0, 8, 1)
typedef virDomainBlockInfo * virDomainBlockInfoPtr;
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 12)
typedef int virDomainBlockJobAbortFlags;
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 4)
typedef unsigned long long virDomainBlockJobCursor;
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 9)
typedef int virDomainBlockJobInfoFlags;
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 4)
typedef virDomainBlockJobInfo * virDomainBlockJobInfoPtr;
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 9)
typedef int virDomainBlockJobSetSpeedFlags;
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 4)
typedef int virDomainBlockJobType;
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 9)
typedef int virDomainBlockPullFlags;
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 12)
typedef int virDomainBlockRebaseFlags;
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 11)
typedef int virDomainBlockResizeFlags;
#endif

#if !LIBVIR_CHECK_VERSION(0, 3, 2)
typedef virDomainBlockStatsStruct * virDomainBlockStatsPtr;
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 2)
typedef int virDomainBlockedReason;
#endif

#if !LIBVIR_CHECK_VERSION(1, 0, 2)
typedef int virDomainChannelFlags;
#endif

#if !LIBVIR_CHECK_VERSION(5, 6, 0)
typedef int virDomainCheckpointCreateFlags;
#endif

#if !LIBVIR_CHECK_VERSION(5, 6, 0)
typedef int virDomainCheckpointDeleteFlags;
#endif

#if !LIBVIR_CHECK_VERSION(5, 6, 0)
typedef int virDomainCheckpointListFlags;
#endif

#if !LIBVIR_CHECK_VERSION(5, 2, 0)
typedef virDomainCheckpoint * virDomainCheckpointPtr;
#endif

#if !LIBVIR_CHECK_VERSION(5, 6, 0)
typedef int virDomainCheckpointXMLFlags;
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 11)
typedef int virDomainConsoleFlags;
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 14)
typedef int virDomainControlErrorReason;
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 3)
typedef virDomainControlInfo * virDomainControlInfoPtr;
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 3)
typedef int virDomainControlState;
#endif

#if !LIBVIR_CHECK_VERSION(0, 7, 5)
typedef int virDomainCoreDumpFlags;
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 3)
typedef int virDomainCoreDumpFormat;
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 2)
typedef int virDomainCrashedReason;
#endif

#if !LIBVIR_CHECK_VERSION(0, 0, 1)
typedef int virDomainCreateFlags;
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 12)
typedef int virDomainDefineFlags;
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 4)
typedef int virDomainDestroyFlagsValues;
#endif

#if !LIBVIR_CHECK_VERSION(0, 7, 7)
typedef int virDomainDeviceModifyFlags;
#endif

#if !LIBVIR_CHECK_VERSION(8, 1, 0)
typedef int virDomainDirtyRateCalcFlags;
#endif

#if !LIBVIR_CHECK_VERSION(7, 2, 0)
typedef int virDomainDirtyRateStatus;
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 10)
typedef int virDomainDiskErrorCode;
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 10)
typedef virDomainDiskError * virDomainDiskErrorPtr;
#endif

#if !LIBVIR_CHECK_VERSION(1, 1, 1)
typedef int virDomainEventCrashedDetailType;
#endif

#if !LIBVIR_CHECK_VERSION(0, 5, 0)
typedef int virDomainEventDefinedDetailType;
#endif

#if !LIBVIR_CHECK_VERSION(0, 8, 0)
typedef virDomainEventGraphicsAddress * virDomainEventGraphicsAddressPtr;
#endif

#if !LIBVIR_CHECK_VERSION(0, 8, 0)
typedef int virDomainEventGraphicsAddressType;
#endif

#if !LIBVIR_CHECK_VERSION(0, 8, 0)
typedef int virDomainEventGraphicsPhase;
#endif

#if !LIBVIR_CHECK_VERSION(0, 8, 0)
typedef virDomainEventGraphicsSubjectIdentity * virDomainEventGraphicsSubjectIdentityPtr;
#endif

#if !LIBVIR_CHECK_VERSION(0, 8, 0)
typedef virDomainEventGraphicsSubject * virDomainEventGraphicsSubjectPtr;
#endif

#if !LIBVIR_CHECK_VERSION(0, 8, 0)
typedef int virDomainEventID;
#endif

#if !LIBVIR_CHECK_VERSION(0, 8, 0)
typedef int virDomainEventIOErrorAction;
#endif

#if !LIBVIR_CHECK_VERSION(0, 10, 2)
typedef int virDomainEventPMSuspendedDetailType;
#endif

#if !LIBVIR_CHECK_VERSION(0, 5, 0)
typedef int virDomainEventResumedDetailType;
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 8)
typedef int virDomainEventShutdownDetailType;
#endif

#if !LIBVIR_CHECK_VERSION(0, 5, 0)
typedef int virDomainEventStartedDetailType;
#endif

#if !LIBVIR_CHECK_VERSION(0, 5, 0)
typedef int virDomainEventStoppedDetailType;
#endif

#if !LIBVIR_CHECK_VERSION(0, 5, 0)
typedef int virDomainEventSuspendedDetailType;
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 11)
typedef int virDomainEventTrayChangeReason;
#endif

#if !LIBVIR_CHECK_VERSION(0, 5, 0)
typedef int virDomainEventType;
#endif

#if !LIBVIR_CHECK_VERSION(0, 5, 0)
typedef int virDomainEventUndefinedDetailType;
#endif

#if !LIBVIR_CHECK_VERSION(0, 8, 0)
typedef int virDomainEventWatchdogAction;
#endif

#if !LIBVIR_CHECK_VERSION(9, 0, 0)
typedef int virDomainFDAssociateFlags;
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 11)
typedef virDomainFSInfo * virDomainFSInfoPtr;
#endif

#if !LIBVIR_CHECK_VERSION(6, 1, 0)
typedef int virDomainGetHostnameFlags;
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 9)
typedef int virDomainGetJobStatsFlags;
#endif

#if !LIBVIR_CHECK_VERSION(10, 2, 0)
typedef int virDomainGraphicsReloadType;
#endif

#if !LIBVIR_CHECK_VERSION(5, 7, 0)
typedef int virDomainGuestInfoTypes;
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
typedef int virDomainInterfaceAddressesSource;
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

#if !LIBVIR_CHECK_VERSION(3, 3, 0)
typedef int virDomainJobOperation;
#endif

#if !LIBVIR_CHECK_VERSION(0, 7, 7)
typedef int virDomainJobType;
#endif

#if !LIBVIR_CHECK_VERSION(3, 9, 0)
typedef int virDomainLifecycle;
#endif

#if !LIBVIR_CHECK_VERSION(3, 9, 0)
typedef int virDomainLifecycleAction;
#endif

#if !LIBVIR_CHECK_VERSION(6, 9, 0)
typedef int virDomainMemoryFailureActionType;
#endif

#if !LIBVIR_CHECK_VERSION(6, 9, 0)
typedef int virDomainMemoryFailureFlags;
#endif

#if !LIBVIR_CHECK_VERSION(6, 9, 0)
typedef int virDomainMemoryFailureRecipientType;
#endif

#if !LIBVIR_CHECK_VERSION(0, 4, 4)
typedef int virDomainMemoryFlags;
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 0)
typedef int virDomainMemoryModFlags;
#endif

#if !LIBVIR_CHECK_VERSION(0, 7, 5)
typedef virDomainMemoryStatStruct * virDomainMemoryStatPtr;
#endif

#if !LIBVIR_CHECK_VERSION(0, 7, 5)
typedef int virDomainMemoryStatTags;
#endif

#if !LIBVIR_CHECK_VERSION(7, 1, 0)
typedef int virDomainMessageType;
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 10)
typedef int virDomainMetadataType;
#endif

#if !LIBVIR_CHECK_VERSION(0, 3, 2)
typedef int virDomainMigrateFlags;
#endif

#if !LIBVIR_CHECK_VERSION(5, 1, 0)
typedef int virDomainMigrateMaxSpeedFlags;
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 2)
typedef int virDomainModificationImpact;
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 2)
typedef int virDomainNostateReason;
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 9)
typedef int virDomainNumatuneMemMode;
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 7)
typedef int virDomainOpenGraphicsFlags;
#endif

#if !LIBVIR_CHECK_VERSION(1, 0, 0)
typedef int virDomainPMSuspendedDiskReason;
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 11)
typedef int virDomainPMSuspendedReason;
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 2)
typedef int virDomainPausedReason;
#endif

#if !LIBVIR_CHECK_VERSION(1, 0, 1)
typedef int virDomainProcessSignal;
#endif

#if !LIBVIR_CHECK_VERSION(0, 0, 1)
typedef virDomain * virDomainPtr;
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 10)
typedef int virDomainRebootFlagValues;
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 2)
typedef int virDomainRunningReason;
#endif

#if !LIBVIR_CHECK_VERSION(5, 1, 0)
typedef int virDomainSaveImageXMLFlags;
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 4)
typedef int virDomainSaveRestoreFlags;
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 5)
typedef int virDomainSetTimeFlags;
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 16)
typedef int virDomainSetUserPasswordFlags;
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 10)
typedef int virDomainShutdownFlagValues;
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 2)
typedef int virDomainShutdownReason;
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 2)
typedef int virDomainShutoffReason;
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 5)
typedef int virDomainSnapshotCreateFlags;
#endif

#if !LIBVIR_CHECK_VERSION(0, 8, 0)
typedef int virDomainSnapshotDeleteFlags;
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 5)
typedef int virDomainSnapshotListFlags;
#endif

#if !LIBVIR_CHECK_VERSION(0, 8, 0)
typedef virDomainSnapshot * virDomainSnapshotPtr;
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 5)
typedef int virDomainSnapshotRevertFlags;
#endif

#if !LIBVIR_CHECK_VERSION(5, 1, 0)
typedef int virDomainSnapshotXMLFlags;
#endif

#if !LIBVIR_CHECK_VERSION(0, 0, 1)
typedef int virDomainState;
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 8)
typedef virDomainStatsRecord * virDomainStatsRecordPtr;
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 8)
typedef int virDomainStatsTypes;
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 4)
typedef int virDomainUndefineFlagsValues;
#endif

#if !LIBVIR_CHECK_VERSION(0, 8, 5)
typedef int virDomainVcpuFlags;
#endif

#if !LIBVIR_CHECK_VERSION(0, 3, 3)
typedef int virDomainXMLFlags;
#endif

#if !LIBVIR_CHECK_VERSION(0, 1, 0)
typedef int virErrorDomain;
#endif

#if !LIBVIR_CHECK_VERSION(0, 1, 0)
typedef int virErrorLevel;
#endif

#if !LIBVIR_CHECK_VERSION(0, 1, 0)
typedef int virErrorNumber;
#endif

#if !LIBVIR_CHECK_VERSION(0, 1, 0)
typedef virError * virErrorPtr;
#endif

#if !LIBVIR_CHECK_VERSION(0, 5, 0)
typedef int virEventHandleType;
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 6)
typedef int virIPAddrType;
#endif

#if !LIBVIR_CHECK_VERSION(7, 7, 0)
typedef int virInterfaceDefineFlags;
#endif

#if !LIBVIR_CHECK_VERSION(0, 6, 4)
typedef virInterface * virInterfacePtr;
#endif

#if !LIBVIR_CHECK_VERSION(0, 7, 3)
typedef int virInterfaceXMLFlags;
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 3)
typedef int virKeycodeSet;
#endif

#if !LIBVIR_CHECK_VERSION(0, 8, 5)
typedef virMemoryParameter * virMemoryParameterPtr;
#endif

#if !LIBVIR_CHECK_VERSION(0, 8, 5)
typedef int virMemoryParameterType;
#endif

#if !LIBVIR_CHECK_VERSION(7, 8, 0)
typedef int virNWFilterBindingCreateFlags;
#endif

#if !LIBVIR_CHECK_VERSION(4, 5, 0)
typedef virNWFilterBinding * virNWFilterBindingPtr;
#endif

#if !LIBVIR_CHECK_VERSION(7, 7, 0)
typedef int virNWFilterDefineFlags;
#endif

#if !LIBVIR_CHECK_VERSION(0, 8, 0)
typedef virNWFilter * virNWFilterPtr;
#endif

#if !LIBVIR_CHECK_VERSION(7, 8, 0)
typedef int virNetworkCreateFlags;
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 6)
typedef virNetworkDHCPLease * virNetworkDHCPLeasePtr;
#endif

#if !LIBVIR_CHECK_VERSION(7, 7, 0)
typedef int virNetworkDefineFlags;
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 1)
typedef int virNetworkEventID;
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 1)
typedef int virNetworkEventLifecycleType;
#endif

#if !LIBVIR_CHECK_VERSION(9, 7, 0)
typedef int virNetworkMetadataType;
#endif

#if !LIBVIR_CHECK_VERSION(5, 5, 0)
typedef int virNetworkPortCreateFlags;
#endif

#if !LIBVIR_CHECK_VERSION(5, 5, 0)
typedef virNetworkPort * virNetworkPortPtr;
#endif

#if !LIBVIR_CHECK_VERSION(0, 2, 0)
typedef virNetwork * virNetworkPtr;
#endif

#if !LIBVIR_CHECK_VERSION(0, 10, 2)
typedef int virNetworkUpdateCommand;
#endif

#if !LIBVIR_CHECK_VERSION(0, 10, 2)
typedef int virNetworkUpdateFlags;
#endif

#if !LIBVIR_CHECK_VERSION(0, 10, 2)
typedef int virNetworkUpdateSection;
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 10)
typedef int virNetworkXMLFlags;
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 9)
typedef int virNodeAllocPagesFlags;
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 3)
typedef virNodeCPUStats * virNodeCPUStatsPtr;
#endif

#if !LIBVIR_CHECK_VERSION(8, 10, 0)
typedef int virNodeDeviceCreateXMLFlags;
#endif

#if !LIBVIR_CHECK_VERSION(8, 10, 0)
typedef int virNodeDeviceDefineXMLFlags;
#endif

#if !LIBVIR_CHECK_VERSION(2, 2, 0)
typedef int virNodeDeviceEventID;
#endif

#if !LIBVIR_CHECK_VERSION(2, 2, 0)
typedef int virNodeDeviceEventLifecycleType;
#endif

#if !LIBVIR_CHECK_VERSION(0, 5, 0)
typedef virNodeDevice * virNodeDevicePtr;
#endif

#if !LIBVIR_CHECK_VERSION(10, 1, 0)
typedef int virNodeDeviceUpdateFlags;
#endif

#if !LIBVIR_CHECK_VERSION(10, 1, 0)
typedef int virNodeDeviceXMLFlags;
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 8)
typedef int virNodeGetCPUStatsAllCPUs;
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 8)
typedef int virNodeGetMemoryStatsAllCells;
#endif

#if !LIBVIR_CHECK_VERSION(0, 1, 0)
typedef virNodeInfo * virNodeInfoPtr;
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 3)
typedef virNodeMemoryStats * virNodeMemoryStatsPtr;
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 8)
typedef int virNodeSuspendTarget;
#endif

#if !LIBVIR_CHECK_VERSION(0, 2, 3)
typedef virSchedParameter * virSchedParameterPtr;
#endif

#if !LIBVIR_CHECK_VERSION(0, 2, 3)
typedef int virSchedParameterType;
#endif

#if !LIBVIR_CHECK_VERSION(7, 7, 0)
typedef int virSecretDefineFlags;
#endif

#if !LIBVIR_CHECK_VERSION(3, 0, 0)
typedef int virSecretEventID;
#endif

#if !LIBVIR_CHECK_VERSION(3, 0, 0)
typedef int virSecretEventLifecycleType;
#endif

#if !LIBVIR_CHECK_VERSION(0, 7, 1)
typedef virSecret * virSecretPtr;
#endif

#if !LIBVIR_CHECK_VERSION(0, 7, 1)
typedef int virSecretUsageType;
#endif

#if !LIBVIR_CHECK_VERSION(0, 6, 1)
typedef virSecurityLabel * virSecurityLabelPtr;
#endif

#if !LIBVIR_CHECK_VERSION(0, 6, 1)
typedef virSecurityModel * virSecurityModelPtr;
#endif

#if !LIBVIR_CHECK_VERSION(0, 4, 1)
typedef int virStoragePoolBuildFlags;
#endif

#if !LIBVIR_CHECK_VERSION(1, 3, 1)
typedef int virStoragePoolCreateFlags;
#endif

#if !LIBVIR_CHECK_VERSION(7, 7, 0)
typedef int virStoragePoolDefineFlags;
#endif

#if !LIBVIR_CHECK_VERSION(0, 4, 1)
typedef int virStoragePoolDeleteFlags;
#endif

#if !LIBVIR_CHECK_VERSION(2, 0, 0)
typedef int virStoragePoolEventID;
#endif

#if !LIBVIR_CHECK_VERSION(2, 0, 0)
typedef int virStoragePoolEventLifecycleType;
#endif

#if !LIBVIR_CHECK_VERSION(0, 4, 1)
typedef virStoragePoolInfo * virStoragePoolInfoPtr;
#endif

#if !LIBVIR_CHECK_VERSION(0, 4, 1)
typedef virStoragePool * virStoragePoolPtr;
#endif

#if !LIBVIR_CHECK_VERSION(0, 4, 1)
typedef int virStoragePoolState;
#endif

#if !LIBVIR_CHECK_VERSION(1, 0, 1)
typedef int virStorageVolCreateFlags;
#endif

#if !LIBVIR_CHECK_VERSION(0, 4, 1)
typedef int virStorageVolDeleteFlags;
#endif

#if !LIBVIR_CHECK_VERSION(3, 4, 0)
typedef int virStorageVolDownloadFlags;
#endif

#if !LIBVIR_CHECK_VERSION(3, 0, 0)
typedef int virStorageVolInfoFlags;
#endif

#if !LIBVIR_CHECK_VERSION(0, 4, 1)
typedef virStorageVolInfo * virStorageVolInfoPtr;
#endif

#if !LIBVIR_CHECK_VERSION(0, 4, 1)
typedef virStorageVol * virStorageVolPtr;
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 10)
typedef int virStorageVolResizeFlags;
#endif

#if !LIBVIR_CHECK_VERSION(0, 4, 1)
typedef int virStorageVolType;
#endif

#if !LIBVIR_CHECK_VERSION(3, 4, 0)
typedef int virStorageVolUploadFlags;
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 10)
typedef int virStorageVolWipeAlgorithm;
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 13)
typedef int virStorageXMLFlags;
#endif

#if !LIBVIR_CHECK_VERSION(0, 7, 2)
typedef int virStreamEventType;
#endif

#if !LIBVIR_CHECK_VERSION(0, 7, 2)
typedef int virStreamFlags;
#endif

#if !LIBVIR_CHECK_VERSION(0, 7, 2)
typedef virStream * virStreamPtr;
#endif

#if !LIBVIR_CHECK_VERSION(3, 4, 0)
typedef int virStreamRecvFlagsValues;
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 8)
typedef int virTypedParameterFlags;
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 2)
typedef virTypedParameter * virTypedParameterPtr;
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 2)
typedef int virTypedParameterType;
#endif

#if !LIBVIR_CHECK_VERSION(6, 10, 0)
typedef int virVcpuHostCpuState;
#endif

#if !LIBVIR_CHECK_VERSION(0, 1, 4)
typedef virVcpuInfo * virVcpuInfoPtr;
#endif

#if !LIBVIR_CHECK_VERSION(0, 1, 4)
typedef int virVcpuState;
#endif
