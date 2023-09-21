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

/* enum virBlkioParameterType */
#  if !LIBVIR_CHECK_VERSION(0, 9, 0)
#    define VIR_DOMAIN_BLKIO_PARAM_INT VIR_TYPED_PARAM_INT
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 0)
#    define VIR_DOMAIN_BLKIO_PARAM_UINT VIR_TYPED_PARAM_UINT
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 0)
#    define VIR_DOMAIN_BLKIO_PARAM_LLONG VIR_TYPED_PARAM_LLONG
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 0)
#    define VIR_DOMAIN_BLKIO_PARAM_ULLONG VIR_TYPED_PARAM_ULLONG
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 0)
#    define VIR_DOMAIN_BLKIO_PARAM_DOUBLE VIR_TYPED_PARAM_DOUBLE
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 0)
#    define VIR_DOMAIN_BLKIO_PARAM_BOOLEAN VIR_TYPED_PARAM_BOOLEAN
#  endif

/* enum virCPUCompareResult */
#  if !LIBVIR_CHECK_VERSION(0, 7, 5)
#    define VIR_CPU_COMPARE_ERROR -1
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 7, 5)
#    define VIR_CPU_COMPARE_INCOMPATIBLE 0
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 7, 5)
#    define VIR_CPU_COMPARE_IDENTICAL 1
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 7, 5)
#    define VIR_CPU_COMPARE_SUPERSET 2
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 10)
#    define VIR_CPU_COMPARE_LAST 3
#  endif

/* enum virConnectBaselineCPUFlags */
#  if !LIBVIR_CHECK_VERSION(1, 1, 2)
#    define VIR_CONNECT_BASELINE_CPU_EXPAND_FEATURES (1 << 0)
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 14)
#    define VIR_CONNECT_BASELINE_CPU_MIGRATABLE (1 << 1)
#  endif

/* enum virConnectCloseReason */
#  if !LIBVIR_CHECK_VERSION(0, 10, 0)
#    define VIR_CONNECT_CLOSE_REASON_ERROR 0
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 10, 0)
#    define VIR_CONNECT_CLOSE_REASON_EOF 1
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 10, 0)
#    define VIR_CONNECT_CLOSE_REASON_KEEPALIVE 2
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 10, 0)
#    define VIR_CONNECT_CLOSE_REASON_CLIENT 3
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 10, 0)
#    define VIR_CONNECT_CLOSE_REASON_LAST 4
#  endif

/* enum virConnectCompareCPUFlags */
#  if !LIBVIR_CHECK_VERSION(1, 2, 6)
#    define VIR_CONNECT_COMPARE_CPU_FAIL_INCOMPATIBLE (1 << 0)
#  endif
#  if !LIBVIR_CHECK_VERSION(6, 9, 0)
#    define VIR_CONNECT_COMPARE_CPU_VALIDATE_XML (1 << 1)
#  endif

/* enum virConnectCredentialType */
#  if !LIBVIR_CHECK_VERSION(0, 4, 1)
#    define VIR_CRED_USERNAME 1
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 4, 1)
#    define VIR_CRED_AUTHNAME 2
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 4, 1)
#    define VIR_CRED_LANGUAGE 3
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 4, 1)
#    define VIR_CRED_CNONCE 4
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 4, 1)
#    define VIR_CRED_PASSPHRASE 5
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 4, 1)
#    define VIR_CRED_ECHOPROMPT 6
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 4, 1)
#    define VIR_CRED_NOECHOPROMPT 7
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 4, 1)
#    define VIR_CRED_REALM 8
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 4, 1)
#    define VIR_CRED_EXTERNAL 9
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 10)
#    define VIR_CRED_LAST 10
#  endif

/* enum virConnectDomainEventAgentLifecycleReason */
#  if !LIBVIR_CHECK_VERSION(1, 2, 11)
#    define VIR_CONNECT_DOMAIN_EVENT_AGENT_LIFECYCLE_REASON_UNKNOWN 0
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 11)
#    define VIR_CONNECT_DOMAIN_EVENT_AGENT_LIFECYCLE_REASON_DOMAIN_STARTED 1
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 11)
#    define VIR_CONNECT_DOMAIN_EVENT_AGENT_LIFECYCLE_REASON_CHANNEL 2
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 11)
#    define VIR_CONNECT_DOMAIN_EVENT_AGENT_LIFECYCLE_REASON_LAST 3
#  endif

/* enum virConnectDomainEventAgentLifecycleState */
#  if !LIBVIR_CHECK_VERSION(1, 2, 11)
#    define VIR_CONNECT_DOMAIN_EVENT_AGENT_LIFECYCLE_STATE_CONNECTED 1
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 11)
#    define VIR_CONNECT_DOMAIN_EVENT_AGENT_LIFECYCLE_STATE_DISCONNECTED 2
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 11)
#    define VIR_CONNECT_DOMAIN_EVENT_AGENT_LIFECYCLE_STATE_LAST 3
#  endif

/* enum virConnectDomainEventBlockJobStatus */
#  if !LIBVIR_CHECK_VERSION(0, 9, 4)
#    define VIR_DOMAIN_BLOCK_JOB_COMPLETED 0
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 4)
#    define VIR_DOMAIN_BLOCK_JOB_FAILED 1
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 12)
#    define VIR_DOMAIN_BLOCK_JOB_CANCELED 2
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 0)
#    define VIR_DOMAIN_BLOCK_JOB_READY 3
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 10)
#    define VIR_DOMAIN_BLOCK_JOB_LAST 4
#  endif

/* enum virConnectDomainEventDiskChangeReason */
#  if !LIBVIR_CHECK_VERSION(0, 9, 7)
#    define VIR_DOMAIN_EVENT_DISK_CHANGE_MISSING_ON_START 0
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 1, 2)
#    define VIR_DOMAIN_EVENT_DISK_DROP_MISSING_ON_START 1
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 10)
#    define VIR_DOMAIN_EVENT_DISK_CHANGE_LAST 2
#  endif

/* enum virConnectFlags */
#  if !LIBVIR_CHECK_VERSION(0, 4, 1)
#    define VIR_CONNECT_RO (1 << 0)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 7)
#    define VIR_CONNECT_NO_ALIASES (1 << 1)
#  endif

/* enum virConnectGetAllDomainStatsFlags */
#  if !LIBVIR_CHECK_VERSION(1, 2, 8)
#    define VIR_CONNECT_GET_ALL_DOMAINS_STATS_ACTIVE VIR_CONNECT_LIST_DOMAINS_ACTIVE
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 8)
#    define VIR_CONNECT_GET_ALL_DOMAINS_STATS_INACTIVE VIR_CONNECT_LIST_DOMAINS_INACTIVE
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 8)
#    define VIR_CONNECT_GET_ALL_DOMAINS_STATS_PERSISTENT VIR_CONNECT_LIST_DOMAINS_PERSISTENT
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 8)
#    define VIR_CONNECT_GET_ALL_DOMAINS_STATS_TRANSIENT VIR_CONNECT_LIST_DOMAINS_TRANSIENT
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 8)
#    define VIR_CONNECT_GET_ALL_DOMAINS_STATS_RUNNING VIR_CONNECT_LIST_DOMAINS_RUNNING
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 8)
#    define VIR_CONNECT_GET_ALL_DOMAINS_STATS_PAUSED VIR_CONNECT_LIST_DOMAINS_PAUSED
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 8)
#    define VIR_CONNECT_GET_ALL_DOMAINS_STATS_SHUTOFF VIR_CONNECT_LIST_DOMAINS_SHUTOFF
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 8)
#    define VIR_CONNECT_GET_ALL_DOMAINS_STATS_OTHER VIR_CONNECT_LIST_DOMAINS_OTHER
#  endif
#  if !LIBVIR_CHECK_VERSION(4, 5, 0)
#    define VIR_CONNECT_GET_ALL_DOMAINS_STATS_NOWAIT (1 << 29)
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 12)
#    define VIR_CONNECT_GET_ALL_DOMAINS_STATS_BACKING (1 << 30)
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 8)
#    define VIR_CONNECT_GET_ALL_DOMAINS_STATS_ENFORCE_STATS (1U << 31)
#  endif

/* enum virConnectListAllDomainsFlags */
#  if !LIBVIR_CHECK_VERSION(0, 9, 13)
#    define VIR_CONNECT_LIST_DOMAINS_ACTIVE (1 << 0)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 13)
#    define VIR_CONNECT_LIST_DOMAINS_INACTIVE (1 << 1)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 13)
#    define VIR_CONNECT_LIST_DOMAINS_PERSISTENT (1 << 2)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 13)
#    define VIR_CONNECT_LIST_DOMAINS_TRANSIENT (1 << 3)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 13)
#    define VIR_CONNECT_LIST_DOMAINS_RUNNING (1 << 4)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 13)
#    define VIR_CONNECT_LIST_DOMAINS_PAUSED (1 << 5)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 13)
#    define VIR_CONNECT_LIST_DOMAINS_SHUTOFF (1 << 6)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 13)
#    define VIR_CONNECT_LIST_DOMAINS_OTHER (1 << 7)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 13)
#    define VIR_CONNECT_LIST_DOMAINS_MANAGEDSAVE (1 << 8)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 13)
#    define VIR_CONNECT_LIST_DOMAINS_NO_MANAGEDSAVE (1 << 9)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 13)
#    define VIR_CONNECT_LIST_DOMAINS_AUTOSTART (1 << 10)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 13)
#    define VIR_CONNECT_LIST_DOMAINS_NO_AUTOSTART (1 << 11)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 13)
#    define VIR_CONNECT_LIST_DOMAINS_HAS_SNAPSHOT (1 << 12)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 13)
#    define VIR_CONNECT_LIST_DOMAINS_NO_SNAPSHOT (1 << 13)
#  endif
#  if !LIBVIR_CHECK_VERSION(5, 6, 0)
#    define VIR_CONNECT_LIST_DOMAINS_HAS_CHECKPOINT (1 << 14)
#  endif
#  if !LIBVIR_CHECK_VERSION(5, 6, 0)
#    define VIR_CONNECT_LIST_DOMAINS_NO_CHECKPOINT (1 << 15)
#  endif

/* enum virConnectListAllInterfacesFlags */
#  if !LIBVIR_CHECK_VERSION(0, 10, 2)
#    define VIR_CONNECT_LIST_INTERFACES_INACTIVE (1 << 0)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 10, 2)
#    define VIR_CONNECT_LIST_INTERFACES_ACTIVE (1 << 1)
#  endif

/* enum virConnectListAllNetworksFlags */
#  if !LIBVIR_CHECK_VERSION(0, 10, 2)
#    define VIR_CONNECT_LIST_NETWORKS_INACTIVE (1 << 0)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 10, 2)
#    define VIR_CONNECT_LIST_NETWORKS_ACTIVE (1 << 1)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 10, 2)
#    define VIR_CONNECT_LIST_NETWORKS_PERSISTENT (1 << 2)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 10, 2)
#    define VIR_CONNECT_LIST_NETWORKS_TRANSIENT (1 << 3)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 10, 2)
#    define VIR_CONNECT_LIST_NETWORKS_AUTOSTART (1 << 4)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 10, 2)
#    define VIR_CONNECT_LIST_NETWORKS_NO_AUTOSTART (1 << 5)
#  endif

/* enum virConnectListAllNodeDeviceFlags */
#  if !LIBVIR_CHECK_VERSION(0, 10, 2)
#    define VIR_CONNECT_LIST_NODE_DEVICES_CAP_SYSTEM (1 << 0)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 10, 2)
#    define VIR_CONNECT_LIST_NODE_DEVICES_CAP_PCI_DEV (1 << 1)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 10, 2)
#    define VIR_CONNECT_LIST_NODE_DEVICES_CAP_USB_DEV (1 << 2)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 10, 2)
#    define VIR_CONNECT_LIST_NODE_DEVICES_CAP_USB_INTERFACE (1 << 3)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 10, 2)
#    define VIR_CONNECT_LIST_NODE_DEVICES_CAP_NET (1 << 4)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 10, 2)
#    define VIR_CONNECT_LIST_NODE_DEVICES_CAP_SCSI_HOST (1 << 5)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 10, 2)
#    define VIR_CONNECT_LIST_NODE_DEVICES_CAP_SCSI_TARGET (1 << 6)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 10, 2)
#    define VIR_CONNECT_LIST_NODE_DEVICES_CAP_SCSI (1 << 7)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 10, 2)
#    define VIR_CONNECT_LIST_NODE_DEVICES_CAP_STORAGE (1 << 8)
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 4)
#    define VIR_CONNECT_LIST_NODE_DEVICES_CAP_FC_HOST (1 << 9)
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 4)
#    define VIR_CONNECT_LIST_NODE_DEVICES_CAP_VPORTS (1 << 10)
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 1, 0)
#    define VIR_CONNECT_LIST_NODE_DEVICES_CAP_SCSI_GENERIC (1 << 11)
#  endif
#  if !LIBVIR_CHECK_VERSION(3, 1, 0)
#    define VIR_CONNECT_LIST_NODE_DEVICES_CAP_DRM (1 << 12)
#  endif
#  if !LIBVIR_CHECK_VERSION(3, 4, 0)
#    define VIR_CONNECT_LIST_NODE_DEVICES_CAP_MDEV_TYPES (1 << 13)
#  endif
#  if !LIBVIR_CHECK_VERSION(3, 4, 0)
#    define VIR_CONNECT_LIST_NODE_DEVICES_CAP_MDEV (1 << 14)
#  endif
#  if !LIBVIR_CHECK_VERSION(3, 4, 0)
#    define VIR_CONNECT_LIST_NODE_DEVICES_CAP_CCW_DEV (1 << 15)
#  endif
#  if !LIBVIR_CHECK_VERSION(6, 8, 0)
#    define VIR_CONNECT_LIST_NODE_DEVICES_CAP_CSS_DEV (1 << 16)
#  endif
#  if !LIBVIR_CHECK_VERSION(6, 9, 0)
#    define VIR_CONNECT_LIST_NODE_DEVICES_CAP_VDPA (1 << 17)
#  endif
#  if !LIBVIR_CHECK_VERSION(7, 0, 0)
#    define VIR_CONNECT_LIST_NODE_DEVICES_CAP_AP_CARD (1 << 18)
#  endif
#  if !LIBVIR_CHECK_VERSION(7, 0, 0)
#    define VIR_CONNECT_LIST_NODE_DEVICES_CAP_AP_QUEUE (1 << 19)
#  endif
#  if !LIBVIR_CHECK_VERSION(7, 0, 0)
#    define VIR_CONNECT_LIST_NODE_DEVICES_CAP_AP_MATRIX (1 << 20)
#  endif
#  if !LIBVIR_CHECK_VERSION(7, 9, 0)
#    define VIR_CONNECT_LIST_NODE_DEVICES_CAP_VPD (1 << 21)
#  endif
#  if !LIBVIR_CHECK_VERSION(7, 3, 0)
#    define VIR_CONNECT_LIST_NODE_DEVICES_INACTIVE (1 << 30)
#  endif
#  if !LIBVIR_CHECK_VERSION(7, 3, 0)
#    define VIR_CONNECT_LIST_NODE_DEVICES_ACTIVE (1U << 31)
#  endif

/* enum virConnectListAllSecretsFlags */
#  if !LIBVIR_CHECK_VERSION(0, 10, 2)
#    define VIR_CONNECT_LIST_SECRETS_EPHEMERAL (1 << 0)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 10, 2)
#    define VIR_CONNECT_LIST_SECRETS_NO_EPHEMERAL (1 << 1)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 10, 2)
#    define VIR_CONNECT_LIST_SECRETS_PRIVATE (1 << 2)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 10, 2)
#    define VIR_CONNECT_LIST_SECRETS_NO_PRIVATE (1 << 3)
#  endif

/* enum virConnectListAllStoragePoolsFlags */
#  if !LIBVIR_CHECK_VERSION(0, 10, 2)
#    define VIR_CONNECT_LIST_STORAGE_POOLS_INACTIVE (1 << 0)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 10, 2)
#    define VIR_CONNECT_LIST_STORAGE_POOLS_ACTIVE (1 << 1)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 10, 2)
#    define VIR_CONNECT_LIST_STORAGE_POOLS_PERSISTENT (1 << 2)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 10, 2)
#    define VIR_CONNECT_LIST_STORAGE_POOLS_TRANSIENT (1 << 3)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 10, 2)
#    define VIR_CONNECT_LIST_STORAGE_POOLS_AUTOSTART (1 << 4)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 10, 2)
#    define VIR_CONNECT_LIST_STORAGE_POOLS_NO_AUTOSTART (1 << 5)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 10, 2)
#    define VIR_CONNECT_LIST_STORAGE_POOLS_DIR (1 << 6)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 10, 2)
#    define VIR_CONNECT_LIST_STORAGE_POOLS_FS (1 << 7)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 10, 2)
#    define VIR_CONNECT_LIST_STORAGE_POOLS_NETFS (1 << 8)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 10, 2)
#    define VIR_CONNECT_LIST_STORAGE_POOLS_LOGICAL (1 << 9)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 10, 2)
#    define VIR_CONNECT_LIST_STORAGE_POOLS_DISK (1 << 10)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 10, 2)
#    define VIR_CONNECT_LIST_STORAGE_POOLS_ISCSI (1 << 11)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 10, 2)
#    define VIR_CONNECT_LIST_STORAGE_POOLS_SCSI (1 << 12)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 10, 2)
#    define VIR_CONNECT_LIST_STORAGE_POOLS_MPATH (1 << 13)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 10, 2)
#    define VIR_CONNECT_LIST_STORAGE_POOLS_RBD (1 << 14)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 10, 2)
#    define VIR_CONNECT_LIST_STORAGE_POOLS_SHEEPDOG (1 << 15)
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 1)
#    define VIR_CONNECT_LIST_STORAGE_POOLS_GLUSTER (1 << 16)
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 8)
#    define VIR_CONNECT_LIST_STORAGE_POOLS_ZFS (1 << 17)
#  endif
#  if !LIBVIR_CHECK_VERSION(3, 1, 0)
#    define VIR_CONNECT_LIST_STORAGE_POOLS_VSTORAGE (1 << 18)
#  endif
#  if !LIBVIR_CHECK_VERSION(5, 6, 0)
#    define VIR_CONNECT_LIST_STORAGE_POOLS_ISCSI_DIRECT (1 << 19)
#  endif

/* enum virDomainAbortJobFlagsValues */
#  if !LIBVIR_CHECK_VERSION(8, 5, 0)
#    define VIR_DOMAIN_ABORT_JOB_POSTCOPY (1 << 0)
#  endif

/* enum virDomainAgentResponseTimeoutValues */
#  if !LIBVIR_CHECK_VERSION(5, 10, 0)
#    define VIR_DOMAIN_AGENT_RESPONSE_TIMEOUT_BLOCK -2
#  endif
#  if !LIBVIR_CHECK_VERSION(5, 10, 0)
#    define VIR_DOMAIN_AGENT_RESPONSE_TIMEOUT_DEFAULT -1
#  endif
#  if !LIBVIR_CHECK_VERSION(5, 10, 0)
#    define VIR_DOMAIN_AGENT_RESPONSE_TIMEOUT_NOWAIT 0
#  endif

/* enum virDomainAuthorizedSSHKeysSetFlags */
#  if !LIBVIR_CHECK_VERSION(6, 10, 0)
#    define VIR_DOMAIN_AUTHORIZED_SSH_KEYS_SET_APPEND (1 << 0)
#  endif
#  if !LIBVIR_CHECK_VERSION(6, 10, 0)
#    define VIR_DOMAIN_AUTHORIZED_SSH_KEYS_SET_REMOVE (1 << 1)
#  endif

/* enum virDomainBackupBeginFlags */
#  if !LIBVIR_CHECK_VERSION(6, 0, 0)
#    define VIR_DOMAIN_BACKUP_BEGIN_REUSE_EXTERNAL (1 << 0)
#  endif

/* enum virDomainBlockCommitFlags */
#  if !LIBVIR_CHECK_VERSION(0, 10, 2)
#    define VIR_DOMAIN_BLOCK_COMMIT_SHALLOW (1 << 0)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 10, 2)
#    define VIR_DOMAIN_BLOCK_COMMIT_DELETE (1 << 1)
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 6)
#    define VIR_DOMAIN_BLOCK_COMMIT_ACTIVE (1 << 2)
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 7)
#    define VIR_DOMAIN_BLOCK_COMMIT_RELATIVE (1 << 3)
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 9)
#    define VIR_DOMAIN_BLOCK_COMMIT_BANDWIDTH_BYTES (1 << 4)
#  endif

/* enum virDomainBlockCopyFlags */
#  if !LIBVIR_CHECK_VERSION(1, 2, 8)
#    define VIR_DOMAIN_BLOCK_COPY_SHALLOW (1 << 0)
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 8)
#    define VIR_DOMAIN_BLOCK_COPY_REUSE_EXT (1 << 1)
#  endif
#  if !LIBVIR_CHECK_VERSION(3, 5, 0)
#    define VIR_DOMAIN_BLOCK_COPY_TRANSIENT_JOB (1 << 2)
#  endif
#  if !LIBVIR_CHECK_VERSION(8, 0, 0)
#    define VIR_DOMAIN_BLOCK_COPY_SYNCHRONOUS_WRITES (1 << 3)
#  endif

/* enum virDomainBlockJobAbortFlags */
#  if !LIBVIR_CHECK_VERSION(0, 9, 12)
#    define VIR_DOMAIN_BLOCK_JOB_ABORT_ASYNC (1 << 0)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 12)
#    define VIR_DOMAIN_BLOCK_JOB_ABORT_PIVOT (1 << 1)
#  endif

/* enum virDomainBlockJobInfoFlags */
#  if !LIBVIR_CHECK_VERSION(1, 2, 9)
#    define VIR_DOMAIN_BLOCK_JOB_INFO_BANDWIDTH_BYTES (1 << 0)
#  endif

/* enum virDomainBlockJobSetSpeedFlags */
#  if !LIBVIR_CHECK_VERSION(1, 2, 9)
#    define VIR_DOMAIN_BLOCK_JOB_SPEED_BANDWIDTH_BYTES (1 << 0)
#  endif

/* enum virDomainBlockJobType */
#  if !LIBVIR_CHECK_VERSION(0, 9, 4)
#    define VIR_DOMAIN_BLOCK_JOB_TYPE_UNKNOWN 0
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 4)
#    define VIR_DOMAIN_BLOCK_JOB_TYPE_PULL 1
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 12)
#    define VIR_DOMAIN_BLOCK_JOB_TYPE_COPY 2
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 10, 2)
#    define VIR_DOMAIN_BLOCK_JOB_TYPE_COMMIT 3
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 6)
#    define VIR_DOMAIN_BLOCK_JOB_TYPE_ACTIVE_COMMIT 4
#  endif
#  if !LIBVIR_CHECK_VERSION(6, 0, 0)
#    define VIR_DOMAIN_BLOCK_JOB_TYPE_BACKUP 5
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 10)
#    define VIR_DOMAIN_BLOCK_JOB_TYPE_LAST 6
#  endif

/* enum virDomainBlockPullFlags */
#  if !LIBVIR_CHECK_VERSION(1, 2, 9)
#    define VIR_DOMAIN_BLOCK_PULL_BANDWIDTH_BYTES (1 << 6)
#  endif

/* enum virDomainBlockRebaseFlags */
#  if !LIBVIR_CHECK_VERSION(0, 9, 12)
#    define VIR_DOMAIN_BLOCK_REBASE_SHALLOW (1 << 0)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 12)
#    define VIR_DOMAIN_BLOCK_REBASE_REUSE_EXT (1 << 1)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 12)
#    define VIR_DOMAIN_BLOCK_REBASE_COPY_RAW (1 << 2)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 12)
#    define VIR_DOMAIN_BLOCK_REBASE_COPY (1 << 3)
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 7)
#    define VIR_DOMAIN_BLOCK_REBASE_RELATIVE (1 << 4)
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 9)
#    define VIR_DOMAIN_BLOCK_REBASE_COPY_DEV (1 << 5)
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 9)
#    define VIR_DOMAIN_BLOCK_REBASE_BANDWIDTH_BYTES (1 << 6)
#  endif

/* enum virDomainBlockResizeFlags */
#  if !LIBVIR_CHECK_VERSION(0, 9, 11)
#    define VIR_DOMAIN_BLOCK_RESIZE_BYTES (1 << 0)
#  endif

/* enum virDomainBlockedReason */
#  if !LIBVIR_CHECK_VERSION(0, 9, 2)
#    define VIR_DOMAIN_BLOCKED_UNKNOWN 0
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 10)
#    define VIR_DOMAIN_BLOCKED_LAST 1
#  endif

/* enum virDomainChannelFlags */
#  if !LIBVIR_CHECK_VERSION(1, 0, 2)
#    define VIR_DOMAIN_CHANNEL_FORCE (1 << 0)
#  endif

/* enum virDomainCheckpointCreateFlags */
#  if !LIBVIR_CHECK_VERSION(5, 6, 0)
#    define VIR_DOMAIN_CHECKPOINT_CREATE_REDEFINE (1 << 0)
#  endif
#  if !LIBVIR_CHECK_VERSION(5, 6, 0)
#    define VIR_DOMAIN_CHECKPOINT_CREATE_QUIESCE (1 << 1)
#  endif
#  if !LIBVIR_CHECK_VERSION(6, 10, 0)
#    define VIR_DOMAIN_CHECKPOINT_CREATE_REDEFINE_VALIDATE (1 << 2)
#  endif

/* enum virDomainCheckpointDeleteFlags */
#  if !LIBVIR_CHECK_VERSION(5, 6, 0)
#    define VIR_DOMAIN_CHECKPOINT_DELETE_CHILDREN (1 << 0)
#  endif
#  if !LIBVIR_CHECK_VERSION(5, 6, 0)
#    define VIR_DOMAIN_CHECKPOINT_DELETE_METADATA_ONLY (1 << 1)
#  endif
#  if !LIBVIR_CHECK_VERSION(5, 6, 0)
#    define VIR_DOMAIN_CHECKPOINT_DELETE_CHILDREN_ONLY (1 << 2)
#  endif

/* enum virDomainCheckpointListFlags */
#  if !LIBVIR_CHECK_VERSION(5, 6, 0)
#    define VIR_DOMAIN_CHECKPOINT_LIST_DESCENDANTS (1 << 0)
#  endif
#  if !LIBVIR_CHECK_VERSION(5, 6, 0)
#    define VIR_DOMAIN_CHECKPOINT_LIST_ROOTS (1 << 0)
#  endif
#  if !LIBVIR_CHECK_VERSION(5, 6, 0)
#    define VIR_DOMAIN_CHECKPOINT_LIST_TOPOLOGICAL (1 << 1)
#  endif
#  if !LIBVIR_CHECK_VERSION(5, 6, 0)
#    define VIR_DOMAIN_CHECKPOINT_LIST_LEAVES (1 << 2)
#  endif
#  if !LIBVIR_CHECK_VERSION(5, 6, 0)
#    define VIR_DOMAIN_CHECKPOINT_LIST_NO_LEAVES (1 << 3)
#  endif

/* enum virDomainCheckpointXMLFlags */
#  if !LIBVIR_CHECK_VERSION(5, 6, 0)
#    define VIR_DOMAIN_CHECKPOINT_XML_SECURE (1 << 0)
#  endif
#  if !LIBVIR_CHECK_VERSION(5, 6, 0)
#    define VIR_DOMAIN_CHECKPOINT_XML_NO_DOMAIN (1 << 1)
#  endif
#  if !LIBVIR_CHECK_VERSION(5, 6, 0)
#    define VIR_DOMAIN_CHECKPOINT_XML_SIZE (1 << 2)
#  endif

/* enum virDomainConsoleFlags */
#  if !LIBVIR_CHECK_VERSION(0, 9, 11)
#    define VIR_DOMAIN_CONSOLE_FORCE (1 << 0)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 11)
#    define VIR_DOMAIN_CONSOLE_SAFE (1 << 1)
#  endif

/* enum virDomainControlErrorReason */
#  if !LIBVIR_CHECK_VERSION(1, 2, 14)
#    define VIR_DOMAIN_CONTROL_ERROR_REASON_NONE 0
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 14)
#    define VIR_DOMAIN_CONTROL_ERROR_REASON_UNKNOWN 1
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 14)
#    define VIR_DOMAIN_CONTROL_ERROR_REASON_MONITOR 2
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 14)
#    define VIR_DOMAIN_CONTROL_ERROR_REASON_INTERNAL 3
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 14)
#    define VIR_DOMAIN_CONTROL_ERROR_REASON_LAST 4
#  endif

/* enum virDomainControlState */
#  if !LIBVIR_CHECK_VERSION(0, 9, 3)
#    define VIR_DOMAIN_CONTROL_OK 0
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 3)
#    define VIR_DOMAIN_CONTROL_JOB 1
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 3)
#    define VIR_DOMAIN_CONTROL_OCCUPIED 2
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 3)
#    define VIR_DOMAIN_CONTROL_ERROR 3
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 10)
#    define VIR_DOMAIN_CONTROL_LAST 4
#  endif

/* enum virDomainCoreDumpFlags */
#  if !LIBVIR_CHECK_VERSION(0, 7, 5)
#    define VIR_DUMP_CRASH (1 << 0)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 7, 5)
#    define VIR_DUMP_LIVE (1 << 1)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 4)
#    define VIR_DUMP_BYPASS_CACHE (1 << 2)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 7)
#    define VIR_DUMP_RESET (1 << 3)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 13)
#    define VIR_DUMP_MEMORY_ONLY (1 << 4)
#  endif

/* enum virDomainCoreDumpFormat */
#  if !LIBVIR_CHECK_VERSION(1, 2, 3)
#    define VIR_DOMAIN_CORE_DUMP_FORMAT_RAW 0
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 3)
#    define VIR_DOMAIN_CORE_DUMP_FORMAT_KDUMP_ZLIB 1
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 3)
#    define VIR_DOMAIN_CORE_DUMP_FORMAT_KDUMP_LZO 2
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 3)
#    define VIR_DOMAIN_CORE_DUMP_FORMAT_KDUMP_SNAPPY 3
#  endif
#  if !LIBVIR_CHECK_VERSION(7, 4, 0)
#    define VIR_DOMAIN_CORE_DUMP_FORMAT_WIN_DMP 4
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 3)
#    define VIR_DOMAIN_CORE_DUMP_FORMAT_LAST 5
#  endif

/* enum virDomainCrashedReason */
#  if !LIBVIR_CHECK_VERSION(0, 9, 2)
#    define VIR_DOMAIN_CRASHED_UNKNOWN 0
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 1, 1)
#    define VIR_DOMAIN_CRASHED_PANICKED 1
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 10)
#    define VIR_DOMAIN_CRASHED_LAST 2
#  endif

/* enum virDomainCreateFlags */
#  if !LIBVIR_CHECK_VERSION(0, 0, 1)
#    define VIR_DOMAIN_NONE 0
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 8, 2)
#    define VIR_DOMAIN_START_PAUSED (1 << 0)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 3)
#    define VIR_DOMAIN_START_AUTODESTROY (1 << 1)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 4)
#    define VIR_DOMAIN_START_BYPASS_CACHE (1 << 2)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 5)
#    define VIR_DOMAIN_START_FORCE_BOOT (1 << 3)
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 12)
#    define VIR_DOMAIN_START_VALIDATE (1 << 4)
#  endif
#  if !LIBVIR_CHECK_VERSION(8, 1, 0)
#    define VIR_DOMAIN_START_RESET_NVRAM (1 << 5)
#  endif

/* enum virDomainDefineFlags */
#  if !LIBVIR_CHECK_VERSION(1, 2, 12)
#    define VIR_DOMAIN_DEFINE_VALIDATE (1 << 0)
#  endif

/* enum virDomainDestroyFlagsValues */
#  if !LIBVIR_CHECK_VERSION(0, 9, 10)
#    define VIR_DOMAIN_DESTROY_DEFAULT 0
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 10)
#    define VIR_DOMAIN_DESTROY_GRACEFUL (1 << 0)
#  endif
#  if !LIBVIR_CHECK_VERSION(8, 3, 0)
#    define VIR_DOMAIN_DESTROY_REMOVE_LOGS (1 << 1)
#  endif

/* enum virDomainDeviceModifyFlags */
#  if !LIBVIR_CHECK_VERSION(0, 7, 7)
#    define VIR_DOMAIN_DEVICE_MODIFY_CURRENT VIR_DOMAIN_AFFECT_CURRENT
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 7, 7)
#    define VIR_DOMAIN_DEVICE_MODIFY_LIVE VIR_DOMAIN_AFFECT_LIVE
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 7, 7)
#    define VIR_DOMAIN_DEVICE_MODIFY_CONFIG VIR_DOMAIN_AFFECT_CONFIG
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 8, 6)
#    define VIR_DOMAIN_DEVICE_MODIFY_FORCE (1 << 2)
#  endif

/* enum virDomainDirtyRateCalcFlags */
#  if !LIBVIR_CHECK_VERSION(8, 1, 0)
#    define VIR_DOMAIN_DIRTYRATE_MODE_PAGE_SAMPLING 0
#  endif
#  if !LIBVIR_CHECK_VERSION(8, 1, 0)
#    define VIR_DOMAIN_DIRTYRATE_MODE_DIRTY_BITMAP (1 << 0)
#  endif
#  if !LIBVIR_CHECK_VERSION(8, 1, 0)
#    define VIR_DOMAIN_DIRTYRATE_MODE_DIRTY_RING (1 << 1)
#  endif

/* enum virDomainDirtyRateStatus */
#  if !LIBVIR_CHECK_VERSION(7, 2, 0)
#    define VIR_DOMAIN_DIRTYRATE_UNSTARTED 0
#  endif
#  if !LIBVIR_CHECK_VERSION(7, 2, 0)
#    define VIR_DOMAIN_DIRTYRATE_MEASURING 1
#  endif
#  if !LIBVIR_CHECK_VERSION(7, 2, 0)
#    define VIR_DOMAIN_DIRTYRATE_MEASURED 2
#  endif
#  if !LIBVIR_CHECK_VERSION(7, 2, 0)
#    define VIR_DOMAIN_DIRTYRATE_LAST 3
#  endif

/* enum virDomainDiskErrorCode */
#  if !LIBVIR_CHECK_VERSION(0, 9, 10)
#    define VIR_DOMAIN_DISK_ERROR_NONE 0
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 10)
#    define VIR_DOMAIN_DISK_ERROR_UNSPEC 1
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 10)
#    define VIR_DOMAIN_DISK_ERROR_NO_SPACE 2
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 10)
#    define VIR_DOMAIN_DISK_ERROR_LAST 3
#  endif

/* enum virDomainEventCrashedDetailType */
#  if !LIBVIR_CHECK_VERSION(1, 1, 1)
#    define VIR_DOMAIN_EVENT_CRASHED_PANICKED 0
#  endif
#  if !LIBVIR_CHECK_VERSION(6, 1, 0)
#    define VIR_DOMAIN_EVENT_CRASHED_CRASHLOADED 1
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 1, 1)
#    define VIR_DOMAIN_EVENT_CRASHED_LAST 2
#  endif

/* enum virDomainEventDefinedDetailType */
#  if !LIBVIR_CHECK_VERSION(0, 5, 0)
#    define VIR_DOMAIN_EVENT_DEFINED_ADDED 0
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 5, 0)
#    define VIR_DOMAIN_EVENT_DEFINED_UPDATED 1
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 19)
#    define VIR_DOMAIN_EVENT_DEFINED_RENAMED 2
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 3, 3)
#    define VIR_DOMAIN_EVENT_DEFINED_FROM_SNAPSHOT 3
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 10)
#    define VIR_DOMAIN_EVENT_DEFINED_LAST 4
#  endif

/* enum virDomainEventGraphicsAddressType */
#  if !LIBVIR_CHECK_VERSION(0, 8, 0)
#    define VIR_DOMAIN_EVENT_GRAPHICS_ADDRESS_IPV4 0
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 8, 0)
#    define VIR_DOMAIN_EVENT_GRAPHICS_ADDRESS_IPV6 1
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 7)
#    define VIR_DOMAIN_EVENT_GRAPHICS_ADDRESS_UNIX 2
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 10)
#    define VIR_DOMAIN_EVENT_GRAPHICS_ADDRESS_LAST 3
#  endif

/* enum virDomainEventGraphicsPhase */
#  if !LIBVIR_CHECK_VERSION(0, 8, 0)
#    define VIR_DOMAIN_EVENT_GRAPHICS_CONNECT 0
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 8, 0)
#    define VIR_DOMAIN_EVENT_GRAPHICS_INITIALIZE 1
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 8, 0)
#    define VIR_DOMAIN_EVENT_GRAPHICS_DISCONNECT 2
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 10)
#    define VIR_DOMAIN_EVENT_GRAPHICS_LAST 3
#  endif

/* enum virDomainEventID */
#  if !LIBVIR_CHECK_VERSION(0, 8, 0)
#    define VIR_DOMAIN_EVENT_ID_LIFECYCLE 0
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 8, 0)
#    define VIR_DOMAIN_EVENT_ID_REBOOT 1
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 8, 0)
#    define VIR_DOMAIN_EVENT_ID_RTC_CHANGE 2
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 8, 0)
#    define VIR_DOMAIN_EVENT_ID_WATCHDOG 3
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 8, 0)
#    define VIR_DOMAIN_EVENT_ID_IO_ERROR 4
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 8, 0)
#    define VIR_DOMAIN_EVENT_ID_GRAPHICS 5
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 8, 1)
#    define VIR_DOMAIN_EVENT_ID_IO_ERROR_REASON 6
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 2)
#    define VIR_DOMAIN_EVENT_ID_CONTROL_ERROR 7
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 4)
#    define VIR_DOMAIN_EVENT_ID_BLOCK_JOB 8
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 7)
#    define VIR_DOMAIN_EVENT_ID_DISK_CHANGE 9
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 11)
#    define VIR_DOMAIN_EVENT_ID_TRAY_CHANGE 10
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 11)
#    define VIR_DOMAIN_EVENT_ID_PMWAKEUP 11
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 11)
#    define VIR_DOMAIN_EVENT_ID_PMSUSPEND 12
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 10, 0)
#    define VIR_DOMAIN_EVENT_ID_BALLOON_CHANGE 13
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 0)
#    define VIR_DOMAIN_EVENT_ID_PMSUSPEND_DISK 14
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 1, 1)
#    define VIR_DOMAIN_EVENT_ID_DEVICE_REMOVED 15
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 6)
#    define VIR_DOMAIN_EVENT_ID_BLOCK_JOB_2 16
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 9)
#    define VIR_DOMAIN_EVENT_ID_TUNABLE 17
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 11)
#    define VIR_DOMAIN_EVENT_ID_AGENT_LIFECYCLE 18
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 15)
#    define VIR_DOMAIN_EVENT_ID_DEVICE_ADDED 19
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 3, 2)
#    define VIR_DOMAIN_EVENT_ID_MIGRATION_ITERATION 20
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 3, 3)
#    define VIR_DOMAIN_EVENT_ID_JOB_COMPLETED 21
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 3, 4)
#    define VIR_DOMAIN_EVENT_ID_DEVICE_REMOVAL_FAILED 22
#  endif
#  if !LIBVIR_CHECK_VERSION(3, 0, 0)
#    define VIR_DOMAIN_EVENT_ID_METADATA_CHANGE 23
#  endif
#  if !LIBVIR_CHECK_VERSION(3, 2, 0)
#    define VIR_DOMAIN_EVENT_ID_BLOCK_THRESHOLD 24
#  endif
#  if !LIBVIR_CHECK_VERSION(6, 9, 0)
#    define VIR_DOMAIN_EVENT_ID_MEMORY_FAILURE 25
#  endif
#  if !LIBVIR_CHECK_VERSION(7, 9, 0)
#    define VIR_DOMAIN_EVENT_ID_MEMORY_DEVICE_SIZE_CHANGE 26
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 8, 0)
#    define VIR_DOMAIN_EVENT_ID_LAST 27
#  endif

/* enum virDomainEventIOErrorAction */
#  if !LIBVIR_CHECK_VERSION(0, 8, 0)
#    define VIR_DOMAIN_EVENT_IO_ERROR_NONE 0
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 8, 0)
#    define VIR_DOMAIN_EVENT_IO_ERROR_PAUSE 1
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 8, 0)
#    define VIR_DOMAIN_EVENT_IO_ERROR_REPORT 2
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 10)
#    define VIR_DOMAIN_EVENT_IO_ERROR_LAST 3
#  endif

/* enum virDomainEventPMSuspendedDetailType */
#  if !LIBVIR_CHECK_VERSION(0, 10, 2)
#    define VIR_DOMAIN_EVENT_PMSUSPENDED_MEMORY 0
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 0)
#    define VIR_DOMAIN_EVENT_PMSUSPENDED_DISK 1
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 10, 2)
#    define VIR_DOMAIN_EVENT_PMSUSPENDED_LAST 2
#  endif

/* enum virDomainEventResumedDetailType */
#  if !LIBVIR_CHECK_VERSION(0, 5, 0)
#    define VIR_DOMAIN_EVENT_RESUMED_UNPAUSED 0
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 5, 0)
#    define VIR_DOMAIN_EVENT_RESUMED_MIGRATED 1
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 5)
#    define VIR_DOMAIN_EVENT_RESUMED_FROM_SNAPSHOT 2
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 3, 3)
#    define VIR_DOMAIN_EVENT_RESUMED_POSTCOPY 3
#  endif
#  if !LIBVIR_CHECK_VERSION(8, 5, 0)
#    define VIR_DOMAIN_EVENT_RESUMED_POSTCOPY_FAILED 4
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 10)
#    define VIR_DOMAIN_EVENT_RESUMED_LAST 5
#  endif

/* enum virDomainEventShutdownDetailType */
#  if !LIBVIR_CHECK_VERSION(0, 9, 8)
#    define VIR_DOMAIN_EVENT_SHUTDOWN_FINISHED 0
#  endif
#  if !LIBVIR_CHECK_VERSION(3, 4, 0)
#    define VIR_DOMAIN_EVENT_SHUTDOWN_GUEST 1
#  endif
#  if !LIBVIR_CHECK_VERSION(3, 4, 0)
#    define VIR_DOMAIN_EVENT_SHUTDOWN_HOST 2
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 10)
#    define VIR_DOMAIN_EVENT_SHUTDOWN_LAST 3
#  endif

/* enum virDomainEventStartedDetailType */
#  if !LIBVIR_CHECK_VERSION(0, 5, 0)
#    define VIR_DOMAIN_EVENT_STARTED_BOOTED 0
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 5, 0)
#    define VIR_DOMAIN_EVENT_STARTED_MIGRATED 1
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 5, 0)
#    define VIR_DOMAIN_EVENT_STARTED_RESTORED 2
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 8, 0)
#    define VIR_DOMAIN_EVENT_STARTED_FROM_SNAPSHOT 3
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 11)
#    define VIR_DOMAIN_EVENT_STARTED_WAKEUP 4
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 10)
#    define VIR_DOMAIN_EVENT_STARTED_LAST 5
#  endif

/* enum virDomainEventStoppedDetailType */
#  if !LIBVIR_CHECK_VERSION(0, 5, 0)
#    define VIR_DOMAIN_EVENT_STOPPED_SHUTDOWN 0
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 5, 0)
#    define VIR_DOMAIN_EVENT_STOPPED_DESTROYED 1
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 5, 0)
#    define VIR_DOMAIN_EVENT_STOPPED_CRASHED 2
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 5, 0)
#    define VIR_DOMAIN_EVENT_STOPPED_MIGRATED 3
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 5, 0)
#    define VIR_DOMAIN_EVENT_STOPPED_SAVED 4
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 5, 0)
#    define VIR_DOMAIN_EVENT_STOPPED_FAILED 5
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 8, 0)
#    define VIR_DOMAIN_EVENT_STOPPED_FROM_SNAPSHOT 6
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 10)
#    define VIR_DOMAIN_EVENT_STOPPED_LAST 7
#  endif

/* enum virDomainEventSuspendedDetailType */
#  if !LIBVIR_CHECK_VERSION(0, 5, 0)
#    define VIR_DOMAIN_EVENT_SUSPENDED_PAUSED 0
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 5, 0)
#    define VIR_DOMAIN_EVENT_SUSPENDED_MIGRATED 1
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 8, 0)
#    define VIR_DOMAIN_EVENT_SUSPENDED_IOERROR 2
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 8, 0)
#    define VIR_DOMAIN_EVENT_SUSPENDED_WATCHDOG 3
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 5)
#    define VIR_DOMAIN_EVENT_SUSPENDED_RESTORED 4
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 5)
#    define VIR_DOMAIN_EVENT_SUSPENDED_FROM_SNAPSHOT 5
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_EVENT_SUSPENDED_API_ERROR 6
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 3, 3)
#    define VIR_DOMAIN_EVENT_SUSPENDED_POSTCOPY 7
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 3, 3)
#    define VIR_DOMAIN_EVENT_SUSPENDED_POSTCOPY_FAILED 8
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 10)
#    define VIR_DOMAIN_EVENT_SUSPENDED_LAST 9
#  endif

/* enum virDomainEventTrayChangeReason */
#  if !LIBVIR_CHECK_VERSION(0, 9, 11)
#    define VIR_DOMAIN_EVENT_TRAY_CHANGE_OPEN 0
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 11)
#    define VIR_DOMAIN_EVENT_TRAY_CHANGE_CLOSE 1
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 11)
#    define VIR_DOMAIN_EVENT_TRAY_CHANGE_LAST 2
#  endif

/* enum virDomainEventType */
#  if !LIBVIR_CHECK_VERSION(0, 5, 0)
#    define VIR_DOMAIN_EVENT_DEFINED 0
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 5, 0)
#    define VIR_DOMAIN_EVENT_UNDEFINED 1
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 5, 0)
#    define VIR_DOMAIN_EVENT_STARTED 2
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 5, 0)
#    define VIR_DOMAIN_EVENT_SUSPENDED 3
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 5, 0)
#    define VIR_DOMAIN_EVENT_RESUMED 4
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 5, 0)
#    define VIR_DOMAIN_EVENT_STOPPED 5
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 8)
#    define VIR_DOMAIN_EVENT_SHUTDOWN 6
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 10, 2)
#    define VIR_DOMAIN_EVENT_PMSUSPENDED 7
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 1, 1)
#    define VIR_DOMAIN_EVENT_CRASHED 8
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 10)
#    define VIR_DOMAIN_EVENT_LAST 9
#  endif

/* enum virDomainEventUndefinedDetailType */
#  if !LIBVIR_CHECK_VERSION(0, 5, 0)
#    define VIR_DOMAIN_EVENT_UNDEFINED_REMOVED 0
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 19)
#    define VIR_DOMAIN_EVENT_UNDEFINED_RENAMED 1
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 10)
#    define VIR_DOMAIN_EVENT_UNDEFINED_LAST 2
#  endif

/* enum virDomainEventWatchdogAction */
#  if !LIBVIR_CHECK_VERSION(0, 8, 0)
#    define VIR_DOMAIN_EVENT_WATCHDOG_NONE 0
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 8, 0)
#    define VIR_DOMAIN_EVENT_WATCHDOG_PAUSE 1
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 8, 0)
#    define VIR_DOMAIN_EVENT_WATCHDOG_RESET 2
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 8, 0)
#    define VIR_DOMAIN_EVENT_WATCHDOG_POWEROFF 3
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 8, 0)
#    define VIR_DOMAIN_EVENT_WATCHDOG_SHUTDOWN 4
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 8, 0)
#    define VIR_DOMAIN_EVENT_WATCHDOG_DEBUG 5
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 17)
#    define VIR_DOMAIN_EVENT_WATCHDOG_INJECTNMI 6
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 10)
#    define VIR_DOMAIN_EVENT_WATCHDOG_LAST 7
#  endif

/* enum virDomainFDAssociateFlags */
#  if !LIBVIR_CHECK_VERSION(9, 0, 0)
#    define VIR_DOMAIN_FD_ASSOCIATE_SECLABEL_RESTORE (1 << 0)
#  endif
#  if !LIBVIR_CHECK_VERSION(9, 0, 0)
#    define VIR_DOMAIN_FD_ASSOCIATE_SECLABEL_WRITABLE (1 << 1)
#  endif

/* enum virDomainGetHostnameFlags */
#  if !LIBVIR_CHECK_VERSION(6, 1, 0)
#    define VIR_DOMAIN_GET_HOSTNAME_LEASE (1 << 0)
#  endif
#  if !LIBVIR_CHECK_VERSION(6, 1, 0)
#    define VIR_DOMAIN_GET_HOSTNAME_AGENT (1 << 1)
#  endif

/* enum virDomainGetJobStatsFlags */
#  if !LIBVIR_CHECK_VERSION(1, 2, 9)
#    define VIR_DOMAIN_JOB_STATS_COMPLETED (1 << 0)
#  endif
#  if !LIBVIR_CHECK_VERSION(6, 0, 0)
#    define VIR_DOMAIN_JOB_STATS_KEEP_COMPLETED (1 << 1)
#  endif

/* enum virDomainGuestInfoTypes */
#  if !LIBVIR_CHECK_VERSION(5, 7, 0)
#    define VIR_DOMAIN_GUEST_INFO_USERS (1 << 0)
#  endif
#  if !LIBVIR_CHECK_VERSION(5, 7, 0)
#    define VIR_DOMAIN_GUEST_INFO_OS (1 << 1)
#  endif
#  if !LIBVIR_CHECK_VERSION(5, 7, 0)
#    define VIR_DOMAIN_GUEST_INFO_TIMEZONE (1 << 2)
#  endif
#  if !LIBVIR_CHECK_VERSION(5, 7, 0)
#    define VIR_DOMAIN_GUEST_INFO_HOSTNAME (1 << 3)
#  endif
#  if !LIBVIR_CHECK_VERSION(5, 7, 0)
#    define VIR_DOMAIN_GUEST_INFO_FILESYSTEM (1 << 4)
#  endif
#  if !LIBVIR_CHECK_VERSION(7, 0, 0)
#    define VIR_DOMAIN_GUEST_INFO_DISKS (1 << 5)
#  endif
#  if !LIBVIR_CHECK_VERSION(7, 10, 0)
#    define VIR_DOMAIN_GUEST_INFO_INTERFACES (1 << 6)
#  endif

/* enum virDomainInterfaceAddressesSource */
#  if !LIBVIR_CHECK_VERSION(1, 2, 14)
#    define VIR_DOMAIN_INTERFACE_ADDRESSES_SRC_LEASE 0
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 14)
#    define VIR_DOMAIN_INTERFACE_ADDRESSES_SRC_AGENT 1
#  endif
#  if !LIBVIR_CHECK_VERSION(4, 2, 0)
#    define VIR_DOMAIN_INTERFACE_ADDRESSES_SRC_ARP 2
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 14)
#    define VIR_DOMAIN_INTERFACE_ADDRESSES_SRC_LAST 3
#  endif

/* enum virDomainJobOperation */
#  if !LIBVIR_CHECK_VERSION(3, 3, 0)
#    define VIR_DOMAIN_JOB_OPERATION_UNKNOWN 0
#  endif
#  if !LIBVIR_CHECK_VERSION(3, 3, 0)
#    define VIR_DOMAIN_JOB_OPERATION_START 1
#  endif
#  if !LIBVIR_CHECK_VERSION(3, 3, 0)
#    define VIR_DOMAIN_JOB_OPERATION_SAVE 2
#  endif
#  if !LIBVIR_CHECK_VERSION(3, 3, 0)
#    define VIR_DOMAIN_JOB_OPERATION_RESTORE 3
#  endif
#  if !LIBVIR_CHECK_VERSION(3, 3, 0)
#    define VIR_DOMAIN_JOB_OPERATION_MIGRATION_IN 4
#  endif
#  if !LIBVIR_CHECK_VERSION(3, 3, 0)
#    define VIR_DOMAIN_JOB_OPERATION_MIGRATION_OUT 5
#  endif
#  if !LIBVIR_CHECK_VERSION(3, 3, 0)
#    define VIR_DOMAIN_JOB_OPERATION_SNAPSHOT 6
#  endif
#  if !LIBVIR_CHECK_VERSION(3, 3, 0)
#    define VIR_DOMAIN_JOB_OPERATION_SNAPSHOT_REVERT 7
#  endif
#  if !LIBVIR_CHECK_VERSION(3, 3, 0)
#    define VIR_DOMAIN_JOB_OPERATION_DUMP 8
#  endif
#  if !LIBVIR_CHECK_VERSION(6, 0, 0)
#    define VIR_DOMAIN_JOB_OPERATION_BACKUP 9
#  endif
#  if !LIBVIR_CHECK_VERSION(9, 0, 0)
#    define VIR_DOMAIN_JOB_OPERATION_SNAPSHOT_DELETE 10
#  endif
#  if !LIBVIR_CHECK_VERSION(3, 3, 0)
#    define VIR_DOMAIN_JOB_OPERATION_LAST 11
#  endif

/* enum virDomainJobType */
#  if !LIBVIR_CHECK_VERSION(0, 7, 7)
#    define VIR_DOMAIN_JOB_NONE 0
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 7, 7)
#    define VIR_DOMAIN_JOB_BOUNDED 1
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 7, 7)
#    define VIR_DOMAIN_JOB_UNBOUNDED 2
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 7, 7)
#    define VIR_DOMAIN_JOB_COMPLETED 3
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 7, 7)
#    define VIR_DOMAIN_JOB_FAILED 4
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 7, 7)
#    define VIR_DOMAIN_JOB_CANCELLED 5
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 10)
#    define VIR_DOMAIN_JOB_LAST 6
#  endif

/* enum virDomainLifecycle */
#  if !LIBVIR_CHECK_VERSION(3, 9, 0)
#    define VIR_DOMAIN_LIFECYCLE_POWEROFF 0
#  endif
#  if !LIBVIR_CHECK_VERSION(3, 9, 0)
#    define VIR_DOMAIN_LIFECYCLE_REBOOT 1
#  endif
#  if !LIBVIR_CHECK_VERSION(3, 9, 0)
#    define VIR_DOMAIN_LIFECYCLE_CRASH 2
#  endif
#  if !LIBVIR_CHECK_VERSION(3, 9, 0)
#    define VIR_DOMAIN_LIFECYCLE_LAST 3
#  endif

/* enum virDomainLifecycleAction */
#  if !LIBVIR_CHECK_VERSION(3, 9, 0)
#    define VIR_DOMAIN_LIFECYCLE_ACTION_DESTROY 0
#  endif
#  if !LIBVIR_CHECK_VERSION(3, 9, 0)
#    define VIR_DOMAIN_LIFECYCLE_ACTION_RESTART 1
#  endif
#  if !LIBVIR_CHECK_VERSION(3, 9, 0)
#    define VIR_DOMAIN_LIFECYCLE_ACTION_RESTART_RENAME 2
#  endif
#  if !LIBVIR_CHECK_VERSION(3, 9, 0)
#    define VIR_DOMAIN_LIFECYCLE_ACTION_PRESERVE 3
#  endif
#  if !LIBVIR_CHECK_VERSION(3, 9, 0)
#    define VIR_DOMAIN_LIFECYCLE_ACTION_COREDUMP_DESTROY 4
#  endif
#  if !LIBVIR_CHECK_VERSION(3, 9, 0)
#    define VIR_DOMAIN_LIFECYCLE_ACTION_COREDUMP_RESTART 5
#  endif
#  if !LIBVIR_CHECK_VERSION(3, 9, 0)
#    define VIR_DOMAIN_LIFECYCLE_ACTION_LAST 6
#  endif

/* enum virDomainMemoryFailureActionType */
#  if !LIBVIR_CHECK_VERSION(6, 9, 0)
#    define VIR_DOMAIN_EVENT_MEMORY_FAILURE_ACTION_IGNORE 0
#  endif
#  if !LIBVIR_CHECK_VERSION(6, 9, 0)
#    define VIR_DOMAIN_EVENT_MEMORY_FAILURE_ACTION_INJECT 1
#  endif
#  if !LIBVIR_CHECK_VERSION(6, 9, 0)
#    define VIR_DOMAIN_EVENT_MEMORY_FAILURE_ACTION_FATAL 2
#  endif
#  if !LIBVIR_CHECK_VERSION(6, 9, 0)
#    define VIR_DOMAIN_EVENT_MEMORY_FAILURE_ACTION_RESET 3
#  endif
#  if !LIBVIR_CHECK_VERSION(6, 9, 0)
#    define VIR_DOMAIN_EVENT_MEMORY_FAILURE_ACTION_LAST 4
#  endif

/* enum virDomainMemoryFailureFlags */
#  if !LIBVIR_CHECK_VERSION(6, 9, 0)
#    define VIR_DOMAIN_MEMORY_FAILURE_ACTION_REQUIRED (1 << 0)
#  endif
#  if !LIBVIR_CHECK_VERSION(6, 9, 0)
#    define VIR_DOMAIN_MEMORY_FAILURE_RECURSIVE (1 << 1)
#  endif

/* enum virDomainMemoryFailureRecipientType */
#  if !LIBVIR_CHECK_VERSION(6, 9, 0)
#    define VIR_DOMAIN_EVENT_MEMORY_FAILURE_RECIPIENT_HYPERVISOR 0
#  endif
#  if !LIBVIR_CHECK_VERSION(6, 9, 0)
#    define VIR_DOMAIN_EVENT_MEMORY_FAILURE_RECIPIENT_GUEST 1
#  endif
#  if !LIBVIR_CHECK_VERSION(6, 9, 0)
#    define VIR_DOMAIN_EVENT_MEMORY_FAILURE_RECIPIENT_LAST 2
#  endif

/* enum virDomainMemoryFlags */
#  if !LIBVIR_CHECK_VERSION(0, 4, 4)
#    define VIR_MEMORY_VIRTUAL (1 << 0)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 7, 0)
#    define VIR_MEMORY_PHYSICAL (1 << 1)
#  endif

/* enum virDomainMemoryModFlags */
#  if !LIBVIR_CHECK_VERSION(0, 9, 1)
#    define VIR_DOMAIN_MEM_CURRENT VIR_DOMAIN_AFFECT_CURRENT
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 0)
#    define VIR_DOMAIN_MEM_LIVE VIR_DOMAIN_AFFECT_LIVE
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 0)
#    define VIR_DOMAIN_MEM_CONFIG VIR_DOMAIN_AFFECT_CONFIG
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 1)
#    define VIR_DOMAIN_MEM_MAXIMUM (1 << 2)
#  endif

/* enum virDomainMemoryStatTags */
#  if !LIBVIR_CHECK_VERSION(0, 7, 5)
#    define VIR_DOMAIN_MEMORY_STAT_SWAP_IN 0
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 7, 5)
#    define VIR_DOMAIN_MEMORY_STAT_SWAP_OUT 1
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 7, 5)
#    define VIR_DOMAIN_MEMORY_STAT_MAJOR_FAULT 2
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 7, 5)
#    define VIR_DOMAIN_MEMORY_STAT_MINOR_FAULT 3
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 7, 5)
#    define VIR_DOMAIN_MEMORY_STAT_UNUSED 4
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 7, 5)
#    define VIR_DOMAIN_MEMORY_STAT_AVAILABLE 5
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 3)
#    define VIR_DOMAIN_MEMORY_STAT_ACTUAL_BALLOON 6
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 10)
#    define VIR_DOMAIN_MEMORY_STAT_RSS 7
#  endif
#  if !LIBVIR_CHECK_VERSION(2, 1, 0)
#    define VIR_DOMAIN_MEMORY_STAT_USABLE 8
#  endif
#  if !LIBVIR_CHECK_VERSION(2, 1, 0)
#    define VIR_DOMAIN_MEMORY_STAT_LAST_UPDATE 9
#  endif
#  if !LIBVIR_CHECK_VERSION(4, 6, 0)
#    define VIR_DOMAIN_MEMORY_STAT_DISK_CACHES 10
#  endif
#  if !LIBVIR_CHECK_VERSION(5, 4, 0)
#    define VIR_DOMAIN_MEMORY_STAT_HUGETLB_PGALLOC 11
#  endif
#  if !LIBVIR_CHECK_VERSION(5, 4, 0)
#    define VIR_DOMAIN_MEMORY_STAT_HUGETLB_PGFAIL 12
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 10)
#    define VIR_DOMAIN_MEMORY_STAT_LAST VIR_DOMAIN_MEMORY_STAT_NR
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 7, 5)
#    define VIR_DOMAIN_MEMORY_STAT_NR 13
#  endif

/* enum virDomainMessageType */
#  if !LIBVIR_CHECK_VERSION(7, 1, 0)
#    define VIR_DOMAIN_MESSAGE_DEPRECATION (1 << 0)
#  endif
#  if !LIBVIR_CHECK_VERSION(7, 1, 0)
#    define VIR_DOMAIN_MESSAGE_TAINTING (1 << 1)
#  endif

/* enum virDomainMetadataType */
#  if !LIBVIR_CHECK_VERSION(0, 9, 10)
#    define VIR_DOMAIN_METADATA_DESCRIPTION 0
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 10)
#    define VIR_DOMAIN_METADATA_TITLE 1
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 10)
#    define VIR_DOMAIN_METADATA_ELEMENT 2
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 10)
#    define VIR_DOMAIN_METADATA_LAST 3
#  endif

/* enum virDomainMigrateFlags */
#  if !LIBVIR_CHECK_VERSION(0, 3, 2)
#    define VIR_MIGRATE_LIVE (1 << 0)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 7, 2)
#    define VIR_MIGRATE_PEER2PEER (1 << 1)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 7, 2)
#    define VIR_MIGRATE_TUNNELLED (1 << 2)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 7, 3)
#    define VIR_MIGRATE_PERSIST_DEST (1 << 3)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 7, 3)
#    define VIR_MIGRATE_UNDEFINE_SOURCE (1 << 4)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 7, 5)
#    define VIR_MIGRATE_PAUSED (1 << 5)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 8, 2)
#    define VIR_MIGRATE_NON_SHARED_DISK (1 << 6)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 8, 2)
#    define VIR_MIGRATE_NON_SHARED_INC (1 << 7)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 4)
#    define VIR_MIGRATE_CHANGE_PROTECTION (1 << 8)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 11)
#    define VIR_MIGRATE_UNSAFE (1 << 9)
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_MIGRATE_OFFLINE (1 << 10)
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 3)
#    define VIR_MIGRATE_COMPRESSED (1 << 11)
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 1, 0)
#    define VIR_MIGRATE_ABORT_ON_ERROR (1 << 12)
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 3)
#    define VIR_MIGRATE_AUTO_CONVERGE (1 << 13)
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 9)
#    define VIR_MIGRATE_RDMA_PIN_ALL (1 << 14)
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 3, 3)
#    define VIR_MIGRATE_POSTCOPY (1 << 15)
#  endif
#  if !LIBVIR_CHECK_VERSION(3, 2, 0)
#    define VIR_MIGRATE_TLS (1 << 16)
#  endif
#  if !LIBVIR_CHECK_VERSION(5, 2, 0)
#    define VIR_MIGRATE_PARALLEL (1 << 17)
#  endif
#  if !LIBVIR_CHECK_VERSION(8, 0, 0)
#    define VIR_MIGRATE_NON_SHARED_SYNCHRONOUS_WRITES (1 << 18)
#  endif
#  if !LIBVIR_CHECK_VERSION(8, 5, 0)
#    define VIR_MIGRATE_POSTCOPY_RESUME (1 << 19)
#  endif
#  if !LIBVIR_CHECK_VERSION(8, 5, 0)
#    define VIR_MIGRATE_ZEROCOPY (1 << 20)
#  endif

/* enum virDomainMigrateMaxSpeedFlags */
#  if !LIBVIR_CHECK_VERSION(5, 1, 0)
#    define VIR_DOMAIN_MIGRATE_MAX_SPEED_POSTCOPY (1 << 0)
#  endif

/* enum virDomainModificationImpact */
#  if !LIBVIR_CHECK_VERSION(0, 9, 2)
#    define VIR_DOMAIN_AFFECT_CURRENT 0
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 2)
#    define VIR_DOMAIN_AFFECT_LIVE (1 << 0)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 2)
#    define VIR_DOMAIN_AFFECT_CONFIG (1 << 1)
#  endif

/* enum virDomainNostateReason */
#  if !LIBVIR_CHECK_VERSION(0, 9, 2)
#    define VIR_DOMAIN_NOSTATE_UNKNOWN 0
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 10)
#    define VIR_DOMAIN_NOSTATE_LAST 1
#  endif

/* enum virDomainNumatuneMemMode */
#  if !LIBVIR_CHECK_VERSION(0, 9, 9)
#    define VIR_DOMAIN_NUMATUNE_MEM_STRICT 0
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 9)
#    define VIR_DOMAIN_NUMATUNE_MEM_PREFERRED 1
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 9)
#    define VIR_DOMAIN_NUMATUNE_MEM_INTERLEAVE 2
#  endif
#  if !LIBVIR_CHECK_VERSION(7, 3, 0)
#    define VIR_DOMAIN_NUMATUNE_MEM_RESTRICTIVE 3
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 9)
#    define VIR_DOMAIN_NUMATUNE_MEM_LAST 4
#  endif

/* enum virDomainOpenGraphicsFlags */
#  if !LIBVIR_CHECK_VERSION(0, 9, 7)
#    define VIR_DOMAIN_OPEN_GRAPHICS_SKIPAUTH (1 << 0)
#  endif

/* enum virDomainPMSuspendedDiskReason */
#  if !LIBVIR_CHECK_VERSION(1, 0, 0)
#    define VIR_DOMAIN_PMSUSPENDED_DISK_UNKNOWN 0
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 0)
#    define VIR_DOMAIN_PMSUSPENDED_DISK_LAST 1
#  endif

/* enum virDomainPMSuspendedReason */
#  if !LIBVIR_CHECK_VERSION(0, 9, 11)
#    define VIR_DOMAIN_PMSUSPENDED_UNKNOWN 0
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 11)
#    define VIR_DOMAIN_PMSUSPENDED_LAST 1
#  endif

/* enum virDomainPausedReason */
#  if !LIBVIR_CHECK_VERSION(0, 9, 2)
#    define VIR_DOMAIN_PAUSED_UNKNOWN 0
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 2)
#    define VIR_DOMAIN_PAUSED_USER 1
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 2)
#    define VIR_DOMAIN_PAUSED_MIGRATION 2
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 2)
#    define VIR_DOMAIN_PAUSED_SAVE 3
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 2)
#    define VIR_DOMAIN_PAUSED_DUMP 4
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 2)
#    define VIR_DOMAIN_PAUSED_IOERROR 5
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 2)
#    define VIR_DOMAIN_PAUSED_WATCHDOG 6
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 2)
#    define VIR_DOMAIN_PAUSED_FROM_SNAPSHOT 7
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 5)
#    define VIR_DOMAIN_PAUSED_SHUTTING_DOWN 8
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_PAUSED_SNAPSHOT 9
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 1, 1)
#    define VIR_DOMAIN_PAUSED_CRASHED 10
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 14)
#    define VIR_DOMAIN_PAUSED_STARTING_UP 11
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 3, 3)
#    define VIR_DOMAIN_PAUSED_POSTCOPY 12
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 3, 3)
#    define VIR_DOMAIN_PAUSED_POSTCOPY_FAILED 13
#  endif
#  if !LIBVIR_CHECK_VERSION(9, 2, 0)
#    define VIR_DOMAIN_PAUSED_API_ERROR 14
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 10)
#    define VIR_DOMAIN_PAUSED_LAST 15
#  endif

/* enum virDomainProcessSignal */
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_PROCESS_SIGNAL_NOP 0
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_PROCESS_SIGNAL_HUP 1
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_PROCESS_SIGNAL_INT 2
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_PROCESS_SIGNAL_QUIT 3
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_PROCESS_SIGNAL_ILL 4
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_PROCESS_SIGNAL_TRAP 5
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_PROCESS_SIGNAL_ABRT 6
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_PROCESS_SIGNAL_BUS 7
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_PROCESS_SIGNAL_FPE 8
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_PROCESS_SIGNAL_KILL 9
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_PROCESS_SIGNAL_USR1 10
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_PROCESS_SIGNAL_SEGV 11
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_PROCESS_SIGNAL_USR2 12
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_PROCESS_SIGNAL_PIPE 13
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_PROCESS_SIGNAL_ALRM 14
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_PROCESS_SIGNAL_TERM 15
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_PROCESS_SIGNAL_STKFLT 16
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_PROCESS_SIGNAL_CHLD 17
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_PROCESS_SIGNAL_CONT 18
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_PROCESS_SIGNAL_STOP 19
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_PROCESS_SIGNAL_TSTP 20
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_PROCESS_SIGNAL_TTIN 21
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_PROCESS_SIGNAL_TTOU 22
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_PROCESS_SIGNAL_URG 23
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_PROCESS_SIGNAL_XCPU 24
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_PROCESS_SIGNAL_XFSZ 25
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_PROCESS_SIGNAL_VTALRM 26
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_PROCESS_SIGNAL_PROF 27
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_PROCESS_SIGNAL_WINCH 28
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_PROCESS_SIGNAL_POLL 29
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_PROCESS_SIGNAL_PWR 30
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_PROCESS_SIGNAL_SYS 31
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_PROCESS_SIGNAL_RT0 32
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_PROCESS_SIGNAL_RT1 33
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_PROCESS_SIGNAL_RT2 34
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_PROCESS_SIGNAL_RT3 35
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_PROCESS_SIGNAL_RT4 36
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_PROCESS_SIGNAL_RT5 37
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_PROCESS_SIGNAL_RT6 38
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_PROCESS_SIGNAL_RT7 39
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_PROCESS_SIGNAL_RT8 40
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_PROCESS_SIGNAL_RT9 41
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_PROCESS_SIGNAL_RT10 42
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_PROCESS_SIGNAL_RT11 43
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_PROCESS_SIGNAL_RT12 44
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_PROCESS_SIGNAL_RT13 45
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_PROCESS_SIGNAL_RT14 46
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_PROCESS_SIGNAL_RT15 47
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_PROCESS_SIGNAL_RT16 48
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_PROCESS_SIGNAL_RT17 49
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_PROCESS_SIGNAL_RT18 50
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_PROCESS_SIGNAL_RT19 51
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_PROCESS_SIGNAL_RT20 52
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_PROCESS_SIGNAL_RT21 53
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_PROCESS_SIGNAL_RT22 54
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_PROCESS_SIGNAL_RT23 55
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_PROCESS_SIGNAL_RT24 56
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_PROCESS_SIGNAL_RT25 57
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_PROCESS_SIGNAL_RT26 58
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_PROCESS_SIGNAL_RT27 59
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_PROCESS_SIGNAL_RT28 60
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_PROCESS_SIGNAL_RT29 61
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_PROCESS_SIGNAL_RT30 62
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_PROCESS_SIGNAL_RT31 63
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_PROCESS_SIGNAL_RT32 64
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_PROCESS_SIGNAL_LAST 65
#  endif

/* enum virDomainRebootFlagValues */
#  if !LIBVIR_CHECK_VERSION(0, 9, 10)
#    define VIR_DOMAIN_REBOOT_DEFAULT 0
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 10)
#    define VIR_DOMAIN_REBOOT_ACPI_POWER_BTN (1 << 0)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 10)
#    define VIR_DOMAIN_REBOOT_GUEST_AGENT (1 << 1)
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_REBOOT_INITCTL (1 << 2)
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_REBOOT_SIGNAL (1 << 3)
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 5)
#    define VIR_DOMAIN_REBOOT_PARAVIRT (1 << 4)
#  endif

/* enum virDomainRunningReason */
#  if !LIBVIR_CHECK_VERSION(0, 9, 2)
#    define VIR_DOMAIN_RUNNING_UNKNOWN 0
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 2)
#    define VIR_DOMAIN_RUNNING_BOOTED 1
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 2)
#    define VIR_DOMAIN_RUNNING_MIGRATED 2
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 2)
#    define VIR_DOMAIN_RUNNING_RESTORED 3
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 2)
#    define VIR_DOMAIN_RUNNING_FROM_SNAPSHOT 4
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 2)
#    define VIR_DOMAIN_RUNNING_UNPAUSED 5
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 2)
#    define VIR_DOMAIN_RUNNING_MIGRATION_CANCELED 6
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 2)
#    define VIR_DOMAIN_RUNNING_SAVE_CANCELED 7
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 11)
#    define VIR_DOMAIN_RUNNING_WAKEUP 8
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 1, 1)
#    define VIR_DOMAIN_RUNNING_CRASHED 9
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 3, 3)
#    define VIR_DOMAIN_RUNNING_POSTCOPY 10
#  endif
#  if !LIBVIR_CHECK_VERSION(8, 5, 0)
#    define VIR_DOMAIN_RUNNING_POSTCOPY_FAILED 11
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 10)
#    define VIR_DOMAIN_RUNNING_LAST 12
#  endif

/* enum virDomainSaveImageXMLFlags */
#  if !LIBVIR_CHECK_VERSION(5, 1, 0)
#    define VIR_DOMAIN_SAVE_IMAGE_XML_SECURE VIR_DOMAIN_XML_SECURE
#  endif

/* enum virDomainSaveRestoreFlags */
#  if !LIBVIR_CHECK_VERSION(0, 9, 4)
#    define VIR_DOMAIN_SAVE_BYPASS_CACHE (1 << 0)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 5)
#    define VIR_DOMAIN_SAVE_RUNNING (1 << 1)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 5)
#    define VIR_DOMAIN_SAVE_PAUSED (1 << 2)
#  endif
#  if !LIBVIR_CHECK_VERSION(8, 1, 0)
#    define VIR_DOMAIN_SAVE_RESET_NVRAM (1 << 3)
#  endif

/* enum virDomainSetTimeFlags */
#  if !LIBVIR_CHECK_VERSION(1, 2, 5)
#    define VIR_DOMAIN_TIME_SYNC (1 << 0)
#  endif

/* enum virDomainSetUserPasswordFlags */
#  if !LIBVIR_CHECK_VERSION(1, 2, 16)
#    define VIR_DOMAIN_PASSWORD_ENCRYPTED (1 << 0)
#  endif

/* enum virDomainShutdownFlagValues */
#  if !LIBVIR_CHECK_VERSION(0, 9, 10)
#    define VIR_DOMAIN_SHUTDOWN_DEFAULT 0
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 10)
#    define VIR_DOMAIN_SHUTDOWN_ACPI_POWER_BTN (1 << 0)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 10)
#    define VIR_DOMAIN_SHUTDOWN_GUEST_AGENT (1 << 1)
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_SHUTDOWN_INITCTL (1 << 2)
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_SHUTDOWN_SIGNAL (1 << 3)
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 5)
#    define VIR_DOMAIN_SHUTDOWN_PARAVIRT (1 << 4)
#  endif

/* enum virDomainShutdownReason */
#  if !LIBVIR_CHECK_VERSION(0, 9, 2)
#    define VIR_DOMAIN_SHUTDOWN_UNKNOWN 0
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 2)
#    define VIR_DOMAIN_SHUTDOWN_USER 1
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 10)
#    define VIR_DOMAIN_SHUTDOWN_LAST 2
#  endif

/* enum virDomainShutoffReason */
#  if !LIBVIR_CHECK_VERSION(0, 9, 2)
#    define VIR_DOMAIN_SHUTOFF_UNKNOWN 0
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 2)
#    define VIR_DOMAIN_SHUTOFF_SHUTDOWN 1
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 2)
#    define VIR_DOMAIN_SHUTOFF_DESTROYED 2
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 2)
#    define VIR_DOMAIN_SHUTOFF_CRASHED 3
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 2)
#    define VIR_DOMAIN_SHUTOFF_MIGRATED 4
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 2)
#    define VIR_DOMAIN_SHUTOFF_SAVED 5
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 2)
#    define VIR_DOMAIN_SHUTOFF_FAILED 6
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 2)
#    define VIR_DOMAIN_SHUTOFF_FROM_SNAPSHOT 7
#  endif
#  if !LIBVIR_CHECK_VERSION(4, 10, 0)
#    define VIR_DOMAIN_SHUTOFF_DAEMON 8
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 10)
#    define VIR_DOMAIN_SHUTOFF_LAST 9
#  endif

/* enum virDomainSnapshotCreateFlags */
#  if !LIBVIR_CHECK_VERSION(0, 9, 5)
#    define VIR_DOMAIN_SNAPSHOT_CREATE_REDEFINE (1 << 0)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 5)
#    define VIR_DOMAIN_SNAPSHOT_CREATE_CURRENT (1 << 1)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 5)
#    define VIR_DOMAIN_SNAPSHOT_CREATE_NO_METADATA (1 << 2)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 5)
#    define VIR_DOMAIN_SNAPSHOT_CREATE_HALT (1 << 3)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 5)
#    define VIR_DOMAIN_SNAPSHOT_CREATE_DISK_ONLY (1 << 4)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 10)
#    define VIR_DOMAIN_SNAPSHOT_CREATE_REUSE_EXT (1 << 5)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 10)
#    define VIR_DOMAIN_SNAPSHOT_CREATE_QUIESCE (1 << 6)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 11)
#    define VIR_DOMAIN_SNAPSHOT_CREATE_ATOMIC (1 << 7)
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_SNAPSHOT_CREATE_LIVE (1 << 8)
#  endif
#  if !LIBVIR_CHECK_VERSION(5, 6, 0)
#    define VIR_DOMAIN_SNAPSHOT_CREATE_VALIDATE (1 << 9)
#  endif

/* enum virDomainSnapshotDeleteFlags */
#  if !LIBVIR_CHECK_VERSION(0, 8, 0)
#    define VIR_DOMAIN_SNAPSHOT_DELETE_CHILDREN (1 << 0)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 5)
#    define VIR_DOMAIN_SNAPSHOT_DELETE_METADATA_ONLY (1 << 1)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 5)
#    define VIR_DOMAIN_SNAPSHOT_DELETE_CHILDREN_ONLY (1 << 2)
#  endif

/* enum virDomainSnapshotListFlags */
#  if !LIBVIR_CHECK_VERSION(0, 9, 7)
#    define VIR_DOMAIN_SNAPSHOT_LIST_DESCENDANTS (1 << 0)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 5)
#    define VIR_DOMAIN_SNAPSHOT_LIST_ROOTS (1 << 0)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 5)
#    define VIR_DOMAIN_SNAPSHOT_LIST_METADATA (1 << 1)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 7)
#    define VIR_DOMAIN_SNAPSHOT_LIST_LEAVES (1 << 2)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 13)
#    define VIR_DOMAIN_SNAPSHOT_LIST_NO_LEAVES (1 << 3)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 13)
#    define VIR_DOMAIN_SNAPSHOT_LIST_NO_METADATA (1 << 4)
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_SNAPSHOT_LIST_INACTIVE (1 << 5)
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_SNAPSHOT_LIST_ACTIVE (1 << 6)
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_SNAPSHOT_LIST_DISK_ONLY (1 << 7)
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_SNAPSHOT_LIST_INTERNAL (1 << 8)
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_DOMAIN_SNAPSHOT_LIST_EXTERNAL (1 << 9)
#  endif
#  if !LIBVIR_CHECK_VERSION(5, 2, 0)
#    define VIR_DOMAIN_SNAPSHOT_LIST_TOPOLOGICAL (1 << 10)
#  endif

/* enum virDomainSnapshotRevertFlags */
#  if !LIBVIR_CHECK_VERSION(0, 9, 5)
#    define VIR_DOMAIN_SNAPSHOT_REVERT_RUNNING (1 << 0)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 5)
#    define VIR_DOMAIN_SNAPSHOT_REVERT_PAUSED (1 << 1)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 7)
#    define VIR_DOMAIN_SNAPSHOT_REVERT_FORCE (1 << 2)
#  endif
#  if !LIBVIR_CHECK_VERSION(8, 1, 0)
#    define VIR_DOMAIN_SNAPSHOT_REVERT_RESET_NVRAM (1 << 3)
#  endif

/* enum virDomainSnapshotXMLFlags */
#  if !LIBVIR_CHECK_VERSION(5, 1, 0)
#    define VIR_DOMAIN_SNAPSHOT_XML_SECURE VIR_DOMAIN_XML_SECURE
#  endif

/* enum virDomainState */
#  if !LIBVIR_CHECK_VERSION(0, 0, 1)
#    define VIR_DOMAIN_NOSTATE 0
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 0, 1)
#    define VIR_DOMAIN_RUNNING 1
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 0, 1)
#    define VIR_DOMAIN_BLOCKED 2
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 0, 1)
#    define VIR_DOMAIN_PAUSED 3
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 0, 1)
#    define VIR_DOMAIN_SHUTDOWN 4
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 0, 1)
#    define VIR_DOMAIN_SHUTOFF 5
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 0, 2)
#    define VIR_DOMAIN_CRASHED 6
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 11)
#    define VIR_DOMAIN_PMSUSPENDED 7
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 5)
#    define VIR_DOMAIN_LAST 8
#  endif

/* enum virDomainStatsTypes */
#  if !LIBVIR_CHECK_VERSION(1, 2, 8)
#    define VIR_DOMAIN_STATS_STATE (1 << 0)
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 9)
#    define VIR_DOMAIN_STATS_CPU_TOTAL (1 << 1)
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 9)
#    define VIR_DOMAIN_STATS_BALLOON (1 << 2)
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 9)
#    define VIR_DOMAIN_STATS_VCPU (1 << 3)
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 9)
#    define VIR_DOMAIN_STATS_INTERFACE (1 << 4)
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 9)
#    define VIR_DOMAIN_STATS_BLOCK (1 << 5)
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 3, 3)
#    define VIR_DOMAIN_STATS_PERF (1 << 6)
#  endif
#  if !LIBVIR_CHECK_VERSION(4, 10, 0)
#    define VIR_DOMAIN_STATS_IOTHREAD (1 << 7)
#  endif
#  if !LIBVIR_CHECK_VERSION(6, 0, 0)
#    define VIR_DOMAIN_STATS_MEMORY (1 << 8)
#  endif
#  if !LIBVIR_CHECK_VERSION(7, 2, 0)
#    define VIR_DOMAIN_STATS_DIRTYRATE (1 << 9)
#  endif
#  if !LIBVIR_CHECK_VERSION(8, 9, 0)
#    define VIR_DOMAIN_STATS_VM (1 << 10)
#  endif

/* enum virDomainUndefineFlagsValues */
#  if !LIBVIR_CHECK_VERSION(0, 9, 4)
#    define VIR_DOMAIN_UNDEFINE_MANAGED_SAVE (1 << 0)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 5)
#    define VIR_DOMAIN_UNDEFINE_SNAPSHOTS_METADATA (1 << 1)
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 9)
#    define VIR_DOMAIN_UNDEFINE_NVRAM (1 << 2)
#  endif
#  if !LIBVIR_CHECK_VERSION(2, 3, 0)
#    define VIR_DOMAIN_UNDEFINE_KEEP_NVRAM (1 << 3)
#  endif
#  if !LIBVIR_CHECK_VERSION(5, 6, 0)
#    define VIR_DOMAIN_UNDEFINE_CHECKPOINTS_METADATA (1 << 4)
#  endif
#  if !LIBVIR_CHECK_VERSION(8, 9, 0)
#    define VIR_DOMAIN_UNDEFINE_TPM (1 << 5)
#  endif
#  if !LIBVIR_CHECK_VERSION(8, 9, 0)
#    define VIR_DOMAIN_UNDEFINE_KEEP_TPM (1 << 6)
#  endif

/* enum virDomainVcpuFlags */
#  if !LIBVIR_CHECK_VERSION(0, 9, 4)
#    define VIR_DOMAIN_VCPU_CURRENT VIR_DOMAIN_AFFECT_CURRENT
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 8, 5)
#    define VIR_DOMAIN_VCPU_LIVE VIR_DOMAIN_AFFECT_LIVE
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 8, 5)
#    define VIR_DOMAIN_VCPU_CONFIG VIR_DOMAIN_AFFECT_CONFIG
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 8, 5)
#    define VIR_DOMAIN_VCPU_MAXIMUM (1 << 2)
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 1, 0)
#    define VIR_DOMAIN_VCPU_GUEST (1 << 3)
#  endif
#  if !LIBVIR_CHECK_VERSION(2, 4, 0)
#    define VIR_DOMAIN_VCPU_HOTPLUGGABLE (1 << 4)
#  endif

/* enum virDomainXMLFlags */
#  if !LIBVIR_CHECK_VERSION(0, 3, 3)
#    define VIR_DOMAIN_XML_SECURE (1 << 0)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 3, 3)
#    define VIR_DOMAIN_XML_INACTIVE (1 << 1)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 8, 0)
#    define VIR_DOMAIN_XML_UPDATE_CPU (1 << 2)
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 0)
#    define VIR_DOMAIN_XML_MIGRATABLE (1 << 3)
#  endif

/* enum virErrorDomain */
#  if !LIBVIR_CHECK_VERSION(0, 1, 0)
#    define VIR_FROM_NONE 0
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 1, 0)
#    define VIR_FROM_XEN 1
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 1, 0)
#    define VIR_FROM_XEND 2
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 1, 0)
#    define VIR_FROM_XENSTORE 3
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 1, 0)
#    define VIR_FROM_SEXPR 4
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 1, 0)
#    define VIR_FROM_XML 5
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 1, 0)
#    define VIR_FROM_DOM 6
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 1, 1)
#    define VIR_FROM_RPC 7
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 1, 3)
#    define VIR_FROM_PROXY 8
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 1, 6)
#    define VIR_FROM_CONF 9
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 2, 0)
#    define VIR_FROM_QEMU 10
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 2, 0)
#    define VIR_FROM_NET 11
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 2, 3)
#    define VIR_FROM_TEST 12
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 2, 3)
#    define VIR_FROM_REMOTE 13
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 3, 1)
#    define VIR_FROM_OPENVZ 14
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 4, 1)
#    define VIR_FROM_XENXM 15
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 4, 1)
#    define VIR_FROM_STATS_LINUX 16
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 4, 2)
#    define VIR_FROM_LXC 17
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 4, 1)
#    define VIR_FROM_STORAGE 18
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 4, 6)
#    define VIR_FROM_NETWORK 19
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 4, 6)
#    define VIR_FROM_DOMAIN 20
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 5, 0)
#    define VIR_FROM_UML 21
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 5, 0)
#    define VIR_FROM_NODEDEV 22
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 5, 0)
#    define VIR_FROM_XEN_INOTIFY 23
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 6, 1)
#    define VIR_FROM_SECURITY 24
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 6, 3)
#    define VIR_FROM_VBOX 25
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 6, 4)
#    define VIR_FROM_INTERFACE 26
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 6, 4)
#    define VIR_FROM_ONE 27
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 7, 0)
#    define VIR_FROM_ESX 28
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 7, 0)
#    define VIR_FROM_PHYP 29
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 7, 1)
#    define VIR_FROM_SECRET 30
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 7, 5)
#    define VIR_FROM_CPU 31
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 8, 0)
#    define VIR_FROM_XENAPI 32
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 8, 0)
#    define VIR_FROM_NWFILTER 33
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 8, 0)
#    define VIR_FROM_HOOK 34
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 8, 0)
#    define VIR_FROM_DOMAIN_SNAPSHOT 35
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 8, 5)
#    define VIR_FROM_AUDIT 36
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 8, 6)
#    define VIR_FROM_SYSINFO 37
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 8, 6)
#    define VIR_FROM_STREAMS 38
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 8, 7)
#    define VIR_FROM_VMWARE 39
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 0)
#    define VIR_FROM_EVENT 40
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 0)
#    define VIR_FROM_LIBXL 41
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 2)
#    define VIR_FROM_LOCKING 42
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 5)
#    define VIR_FROM_HYPERV 43
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 8)
#    define VIR_FROM_CAPABILITIES 44
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 11)
#    define VIR_FROM_URI 45
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 11)
#    define VIR_FROM_AUTH 46
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 12)
#    define VIR_FROM_DBUS 47
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 10, 0)
#    define VIR_FROM_PARALLELS 48
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 10, 0)
#    define VIR_FROM_DEVICE 49
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 10, 0)
#    define VIR_FROM_SSH 50
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 0)
#    define VIR_FROM_LOCKSPACE 51
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_FROM_INITCTL 52
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 4)
#    define VIR_FROM_IDENTITY 53
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 5)
#    define VIR_FROM_CGROUP 54
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 1, 0)
#    define VIR_FROM_ACCESS 55
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 1, 1)
#    define VIR_FROM_SYSTEMD 56
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 2)
#    define VIR_FROM_BHYVE 57
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 3)
#    define VIR_FROM_CRYPTO 58
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 4)
#    define VIR_FROM_FIREWALL 59
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 9)
#    define VIR_FROM_POLKIT 60
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 14)
#    define VIR_FROM_THREAD 61
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 17)
#    define VIR_FROM_ADMIN 62
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 3, 0)
#    define VIR_FROM_LOGGING 63
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 3, 2)
#    define VIR_FROM_XENXL 64
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 3, 3)
#    define VIR_FROM_PERF 65
#  endif
#  if !LIBVIR_CHECK_VERSION(2, 5, 0)
#    define VIR_FROM_LIBSSH 66
#  endif
#  if !LIBVIR_CHECK_VERSION(3, 7, 0)
#    define VIR_FROM_RESCTRL 67
#  endif
#  if !LIBVIR_CHECK_VERSION(5, 1, 0)
#    define VIR_FROM_FIREWALLD 68
#  endif
#  if !LIBVIR_CHECK_VERSION(5, 2, 0)
#    define VIR_FROM_DOMAIN_CHECKPOINT 69
#  endif
#  if !LIBVIR_CHECK_VERSION(5, 6, 0)
#    define VIR_FROM_TPM 70
#  endif
#  if !LIBVIR_CHECK_VERSION(5, 10, 0)
#    define VIR_FROM_BPF 71
#  endif
#  if !LIBVIR_CHECK_VERSION(7, 5, 0)
#    define VIR_FROM_CH 72
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 13)
#    define VIR_ERR_DOMAIN_LAST 73
#  endif

/* enum virErrorLevel */
#  if !LIBVIR_CHECK_VERSION(0, 1, 0)
#    define VIR_ERR_NONE 0
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 1, 0)
#    define VIR_ERR_WARNING 1
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 1, 0)
#    define VIR_ERR_ERROR 2
#  endif

/* enum virErrorNumber */
#  if !LIBVIR_CHECK_VERSION(0, 1, 0)
#    define VIR_ERR_OK 0
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 1, 0)
#    define VIR_ERR_INTERNAL_ERROR 1
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 1, 0)
#    define VIR_ERR_NO_MEMORY 2
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 1, 0)
#    define VIR_ERR_NO_SUPPORT 3
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 1, 0)
#    define VIR_ERR_UNKNOWN_HOST 4
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 1, 0)
#    define VIR_ERR_NO_CONNECT 5
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 1, 0)
#    define VIR_ERR_INVALID_CONN 6
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 1, 0)
#    define VIR_ERR_INVALID_DOMAIN 7
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 1, 0)
#    define VIR_ERR_INVALID_ARG 8
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 1, 0)
#    define VIR_ERR_OPERATION_FAILED 9
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 1, 0)
#    define VIR_ERR_GET_FAILED 10
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 1, 0)
#    define VIR_ERR_POST_FAILED 11
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 1, 0)
#    define VIR_ERR_HTTP_ERROR 12
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 1, 0)
#    define VIR_ERR_SEXPR_SERIAL 13
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 1, 0)
#    define VIR_ERR_NO_XEN 14
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 1, 0)
#    define VIR_ERR_XEN_CALL 15
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 1, 0)
#    define VIR_ERR_OS_TYPE 16
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 1, 0)
#    define VIR_ERR_NO_KERNEL 17
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 1, 0)
#    define VIR_ERR_NO_ROOT 18
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 1, 0)
#    define VIR_ERR_NO_SOURCE 19
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 1, 0)
#    define VIR_ERR_NO_TARGET 20
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 1, 0)
#    define VIR_ERR_NO_NAME 21
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 1, 0)
#    define VIR_ERR_NO_OS 22
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 1, 0)
#    define VIR_ERR_NO_DEVICE 23
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 1, 0)
#    define VIR_ERR_NO_XENSTORE 24
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 1, 0)
#    define VIR_ERR_DRIVER_FULL 25
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 1, 0)
#    define VIR_ERR_CALL_FAILED 26
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 1, 1)
#    define VIR_ERR_XML_ERROR 27
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 1, 1)
#    define VIR_ERR_DOM_EXIST 28
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 1, 4)
#    define VIR_ERR_OPERATION_DENIED 29
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 1, 6)
#    define VIR_ERR_OPEN_FAILED 30
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 1, 6)
#    define VIR_ERR_READ_FAILED 31
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 1, 6)
#    define VIR_ERR_PARSE_FAILED 32
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 1, 6)
#    define VIR_ERR_CONF_SYNTAX 33
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 1, 6)
#    define VIR_ERR_WRITE_FAILED 34
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 1, 9)
#    define VIR_ERR_XML_DETAIL 35
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 2, 0)
#    define VIR_ERR_INVALID_NETWORK 36
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 2, 0)
#    define VIR_ERR_NETWORK_EXIST 37
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 2, 1)
#    define VIR_ERR_SYSTEM_ERROR 38
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 2, 3)
#    define VIR_ERR_RPC 39
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 2, 3)
#    define VIR_ERR_GNUTLS_ERROR 40
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 2, 3)
#    define VIR_WAR_NO_NETWORK 41
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 3, 0)
#    define VIR_ERR_NO_DOMAIN 42
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 3, 0)
#    define VIR_ERR_NO_NETWORK 43
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 3, 1)
#    define VIR_ERR_INVALID_MAC 44
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 4, 1)
#    define VIR_ERR_AUTH_FAILED 45
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 4, 1)
#    define VIR_ERR_INVALID_STORAGE_POOL 46
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 4, 1)
#    define VIR_ERR_INVALID_STORAGE_VOL 47
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 4, 1)
#    define VIR_WAR_NO_STORAGE 48
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 4, 1)
#    define VIR_ERR_NO_STORAGE_POOL 49
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 4, 1)
#    define VIR_ERR_NO_STORAGE_VOL 50
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 5, 0)
#    define VIR_WAR_NO_NODE 51
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 5, 0)
#    define VIR_ERR_INVALID_NODE_DEVICE 52
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 5, 0)
#    define VIR_ERR_NO_NODE_DEVICE 53
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 6, 1)
#    define VIR_ERR_NO_SECURITY_MODEL 54
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 6, 4)
#    define VIR_ERR_OPERATION_INVALID 55
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 6, 4)
#    define VIR_WAR_NO_INTERFACE 56
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 6, 4)
#    define VIR_ERR_NO_INTERFACE 57
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 6, 4)
#    define VIR_ERR_INVALID_INTERFACE 58
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 7, 0)
#    define VIR_ERR_MULTIPLE_INTERFACES 59
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 8, 0)
#    define VIR_WAR_NO_NWFILTER 60
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 8, 0)
#    define VIR_ERR_INVALID_NWFILTER 61
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 8, 0)
#    define VIR_ERR_NO_NWFILTER 62
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 8, 0)
#    define VIR_ERR_BUILD_FIREWALL 63
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 7, 1)
#    define VIR_WAR_NO_SECRET 64
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 7, 1)
#    define VIR_ERR_INVALID_SECRET 65
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 7, 1)
#    define VIR_ERR_NO_SECRET 66
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 7, 3)
#    define VIR_ERR_CONFIG_UNSUPPORTED 67
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 7, 3)
#    define VIR_ERR_OPERATION_TIMEOUT 68
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 7, 3)
#    define VIR_ERR_MIGRATE_PERSIST_FAILED 69
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 8, 0)
#    define VIR_ERR_HOOK_SCRIPT_FAILED 70
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 8, 0)
#    define VIR_ERR_INVALID_DOMAIN_SNAPSHOT 71
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 8, 0)
#    define VIR_ERR_NO_DOMAIN_SNAPSHOT 72
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 0)
#    define VIR_ERR_INVALID_STREAM 73
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 4)
#    define VIR_ERR_ARGUMENT_UNSUPPORTED 74
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 5)
#    define VIR_ERR_STORAGE_PROBE_FAILED 75
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 5)
#    define VIR_ERR_STORAGE_POOL_BUILT 76
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 7)
#    define VIR_ERR_SNAPSHOT_REVERT_RISKY 77
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 9)
#    define VIR_ERR_OPERATION_ABORTED 78
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 10)
#    define VIR_ERR_AUTH_CANCELLED 79
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 10)
#    define VIR_ERR_NO_DOMAIN_METADATA 80
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 11)
#    define VIR_ERR_MIGRATE_UNSAFE 81
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 11)
#    define VIR_ERR_OVERFLOW 82
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 12)
#    define VIR_ERR_BLOCK_COPY_ACTIVE 83
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 10, 0)
#    define VIR_ERR_OPERATION_UNSUPPORTED 84
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 10, 0)
#    define VIR_ERR_SSH 85
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 10, 0)
#    define VIR_ERR_AGENT_UNRESPONSIVE 86
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 0)
#    define VIR_ERR_RESOURCE_BUSY 87
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 1, 0)
#    define VIR_ERR_ACCESS_DENIED 88
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 1, 1)
#    define VIR_ERR_DBUS_SERVICE 89
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 1, 4)
#    define VIR_ERR_STORAGE_VOL_EXIST 90
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 6)
#    define VIR_ERR_CPU_INCOMPATIBLE 91
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 12)
#    define VIR_ERR_XML_INVALID_SCHEMA 92
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 18)
#    define VIR_ERR_MIGRATE_FINISH_OK 93
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 3, 3)
#    define VIR_ERR_AUTH_UNAVAILABLE 94
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 3, 3)
#    define VIR_ERR_NO_SERVER 95
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 3, 5)
#    define VIR_ERR_NO_CLIENT 96
#  endif
#  if !LIBVIR_CHECK_VERSION(2, 3, 0)
#    define VIR_ERR_AGENT_UNSYNCED 97
#  endif
#  if !LIBVIR_CHECK_VERSION(2, 5, 0)
#    define VIR_ERR_LIBSSH 98
#  endif
#  if !LIBVIR_CHECK_VERSION(4, 1, 0)
#    define VIR_ERR_DEVICE_MISSING 99
#  endif
#  if !LIBVIR_CHECK_VERSION(4, 5, 0)
#    define VIR_ERR_INVALID_NWFILTER_BINDING 100
#  endif
#  if !LIBVIR_CHECK_VERSION(4, 5, 0)
#    define VIR_ERR_NO_NWFILTER_BINDING 101
#  endif
#  if !LIBVIR_CHECK_VERSION(5, 2, 0)
#    define VIR_ERR_INVALID_DOMAIN_CHECKPOINT 102
#  endif
#  if !LIBVIR_CHECK_VERSION(5, 2, 0)
#    define VIR_ERR_NO_DOMAIN_CHECKPOINT 103
#  endif
#  if !LIBVIR_CHECK_VERSION(5, 2, 0)
#    define VIR_ERR_NO_DOMAIN_BACKUP 104
#  endif
#  if !LIBVIR_CHECK_VERSION(5, 5, 0)
#    define VIR_ERR_INVALID_NETWORK_PORT 105
#  endif
#  if !LIBVIR_CHECK_VERSION(5, 5, 0)
#    define VIR_ERR_NETWORK_PORT_EXIST 106
#  endif
#  if !LIBVIR_CHECK_VERSION(5, 5, 0)
#    define VIR_ERR_NO_NETWORK_PORT 107
#  endif
#  if !LIBVIR_CHECK_VERSION(6, 1, 0)
#    define VIR_ERR_NO_HOSTNAME 108
#  endif
#  if !LIBVIR_CHECK_VERSION(6, 10, 0)
#    define VIR_ERR_CHECKPOINT_INCONSISTENT 109
#  endif
#  if !LIBVIR_CHECK_VERSION(7, 1, 0)
#    define VIR_ERR_MULTIPLE_DOMAINS 110
#  endif
#  if !LIBVIR_CHECK_VERSION(5, 0, 0)
#    define VIR_ERR_NUMBER_LAST 111
#  endif

/* enum virEventHandleType */
#  if !LIBVIR_CHECK_VERSION(0, 5, 0)
#    define VIR_EVENT_HANDLE_READABLE (1 << 0)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 5, 0)
#    define VIR_EVENT_HANDLE_WRITABLE (1 << 1)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 5, 0)
#    define VIR_EVENT_HANDLE_ERROR (1 << 2)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 5, 0)
#    define VIR_EVENT_HANDLE_HANGUP (1 << 3)
#  endif

/* enum virIPAddrType */
#  if !LIBVIR_CHECK_VERSION(1, 2, 6)
#    define VIR_IP_ADDR_TYPE_IPV4 0
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 6)
#    define VIR_IP_ADDR_TYPE_IPV6 1
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 6)
#    define VIR_IP_ADDR_TYPE_LAST 2
#  endif

/* enum virInterfaceDefineFlags */
#  if !LIBVIR_CHECK_VERSION(7, 7, 0)
#    define VIR_INTERFACE_DEFINE_VALIDATE (1 << 0)
#  endif

/* enum virInterfaceXMLFlags */
#  if !LIBVIR_CHECK_VERSION(0, 7, 3)
#    define VIR_INTERFACE_XML_INACTIVE (1 << 0)
#  endif

/* enum virKeycodeSet */
#  if !LIBVIR_CHECK_VERSION(0, 9, 3)
#    define VIR_KEYCODE_SET_LINUX 0
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 3)
#    define VIR_KEYCODE_SET_XT 1
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 3)
#    define VIR_KEYCODE_SET_ATSET1 2
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 3)
#    define VIR_KEYCODE_SET_ATSET2 3
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 3)
#    define VIR_KEYCODE_SET_ATSET3 4
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 4)
#    define VIR_KEYCODE_SET_OSX 5
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 4)
#    define VIR_KEYCODE_SET_XT_KBD 6
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 4)
#    define VIR_KEYCODE_SET_USB 7
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 4)
#    define VIR_KEYCODE_SET_WIN32 8
#  endif
#  if !LIBVIR_CHECK_VERSION(4, 2, 0)
#    define VIR_KEYCODE_SET_QNUM 9
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 4)
#    define VIR_KEYCODE_SET_LAST 10
#  endif

/* enum virMemoryParameterType */
#  if !LIBVIR_CHECK_VERSION(0, 8, 5)
#    define VIR_DOMAIN_MEMORY_PARAM_INT VIR_TYPED_PARAM_INT
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 8, 5)
#    define VIR_DOMAIN_MEMORY_PARAM_UINT VIR_TYPED_PARAM_UINT
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 8, 5)
#    define VIR_DOMAIN_MEMORY_PARAM_LLONG VIR_TYPED_PARAM_LLONG
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 8, 5)
#    define VIR_DOMAIN_MEMORY_PARAM_ULLONG VIR_TYPED_PARAM_ULLONG
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 8, 5)
#    define VIR_DOMAIN_MEMORY_PARAM_DOUBLE VIR_TYPED_PARAM_DOUBLE
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 8, 5)
#    define VIR_DOMAIN_MEMORY_PARAM_BOOLEAN VIR_TYPED_PARAM_BOOLEAN
#  endif

/* enum virNWFilterBindingCreateFlags */
#  if !LIBVIR_CHECK_VERSION(7, 8, 0)
#    define VIR_NWFILTER_BINDING_CREATE_VALIDATE (1 << 0)
#  endif

/* enum virNWFilterDefineFlags */
#  if !LIBVIR_CHECK_VERSION(7, 7, 0)
#    define VIR_NWFILTER_DEFINE_VALIDATE (1 << 0)
#  endif

/* enum virNetworkCreateFlags */
#  if !LIBVIR_CHECK_VERSION(7, 8, 0)
#    define VIR_NETWORK_CREATE_VALIDATE (1 << 0)
#  endif

/* enum virNetworkDefineFlags */
#  if !LIBVIR_CHECK_VERSION(7, 7, 0)
#    define VIR_NETWORK_DEFINE_VALIDATE (1 << 0)
#  endif

/* enum virNetworkEventID */
#  if !LIBVIR_CHECK_VERSION(1, 2, 1)
#    define VIR_NETWORK_EVENT_ID_LIFECYCLE 0
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 1)
#    define VIR_NETWORK_EVENT_ID_LAST 1
#  endif

/* enum virNetworkEventLifecycleType */
#  if !LIBVIR_CHECK_VERSION(1, 2, 1)
#    define VIR_NETWORK_EVENT_DEFINED 0
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 1)
#    define VIR_NETWORK_EVENT_UNDEFINED 1
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 1)
#    define VIR_NETWORK_EVENT_STARTED 2
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 1)
#    define VIR_NETWORK_EVENT_STOPPED 3
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 1)
#    define VIR_NETWORK_EVENT_LAST 4
#  endif

/* enum virNetworkPortCreateFlags */
#  if !LIBVIR_CHECK_VERSION(5, 5, 0)
#    define VIR_NETWORK_PORT_CREATE_RECLAIM (1 << 0)
#  endif
#  if !LIBVIR_CHECK_VERSION(7, 8, 0)
#    define VIR_NETWORK_PORT_CREATE_VALIDATE (1 << 1)
#  endif

/* enum virNetworkUpdateCommand */
#  if !LIBVIR_CHECK_VERSION(0, 10, 2)
#    define VIR_NETWORK_UPDATE_COMMAND_NONE 0
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 10, 2)
#    define VIR_NETWORK_UPDATE_COMMAND_MODIFY 1
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 10, 2)
#    define VIR_NETWORK_UPDATE_COMMAND_DELETE 2
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 10, 2)
#    define VIR_NETWORK_UPDATE_COMMAND_ADD_LAST 3
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 10, 2)
#    define VIR_NETWORK_UPDATE_COMMAND_ADD_FIRST 4
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 10, 2)
#    define VIR_NETWORK_UPDATE_COMMAND_LAST 5
#  endif

/* enum virNetworkUpdateFlags */
#  if !LIBVIR_CHECK_VERSION(0, 10, 2)
#    define VIR_NETWORK_UPDATE_AFFECT_CURRENT 0
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 10, 2)
#    define VIR_NETWORK_UPDATE_AFFECT_LIVE (1 << 0)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 10, 2)
#    define VIR_NETWORK_UPDATE_AFFECT_CONFIG (1 << 1)
#  endif

/* enum virNetworkUpdateSection */
#  if !LIBVIR_CHECK_VERSION(0, 10, 2)
#    define VIR_NETWORK_SECTION_NONE 0
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 10, 2)
#    define VIR_NETWORK_SECTION_BRIDGE 1
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 10, 2)
#    define VIR_NETWORK_SECTION_DOMAIN 2
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 10, 2)
#    define VIR_NETWORK_SECTION_IP 3
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 10, 2)
#    define VIR_NETWORK_SECTION_IP_DHCP_HOST 4
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 10, 2)
#    define VIR_NETWORK_SECTION_IP_DHCP_RANGE 5
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 10, 2)
#    define VIR_NETWORK_SECTION_FORWARD 6
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 10, 2)
#    define VIR_NETWORK_SECTION_FORWARD_INTERFACE 7
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 10, 2)
#    define VIR_NETWORK_SECTION_FORWARD_PF 8
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 10, 2)
#    define VIR_NETWORK_SECTION_PORTGROUP 9
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 10, 2)
#    define VIR_NETWORK_SECTION_DNS_HOST 10
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 10, 2)
#    define VIR_NETWORK_SECTION_DNS_TXT 11
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 10, 2)
#    define VIR_NETWORK_SECTION_DNS_SRV 12
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 10, 2)
#    define VIR_NETWORK_SECTION_LAST 13
#  endif

/* enum virNetworkXMLFlags */
#  if !LIBVIR_CHECK_VERSION(0, 9, 10)
#    define VIR_NETWORK_XML_INACTIVE (1 << 0)
#  endif

/* enum virNodeAllocPagesFlags */
#  if !LIBVIR_CHECK_VERSION(1, 2, 9)
#    define VIR_NODE_ALLOC_PAGES_ADD 0
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 9)
#    define VIR_NODE_ALLOC_PAGES_SET (1 << 0)
#  endif

/* enum virNodeDeviceCreateXMLFlags */
#  if !LIBVIR_CHECK_VERSION(8, 10, 0)
#    define VIR_NODE_DEVICE_CREATE_XML_VALIDATE (1 << 0)
#  endif

/* enum virNodeDeviceDefineXMLFlags */
#  if !LIBVIR_CHECK_VERSION(8, 10, 0)
#    define VIR_NODE_DEVICE_DEFINE_XML_VALIDATE (1 << 0)
#  endif

/* enum virNodeDeviceEventID */
#  if !LIBVIR_CHECK_VERSION(2, 2, 0)
#    define VIR_NODE_DEVICE_EVENT_ID_LIFECYCLE 0
#  endif
#  if !LIBVIR_CHECK_VERSION(2, 2, 0)
#    define VIR_NODE_DEVICE_EVENT_ID_UPDATE 1
#  endif
#  if !LIBVIR_CHECK_VERSION(2, 2, 0)
#    define VIR_NODE_DEVICE_EVENT_ID_LAST 2
#  endif

/* enum virNodeDeviceEventLifecycleType */
#  if !LIBVIR_CHECK_VERSION(2, 2, 0)
#    define VIR_NODE_DEVICE_EVENT_CREATED 0
#  endif
#  if !LIBVIR_CHECK_VERSION(2, 2, 0)
#    define VIR_NODE_DEVICE_EVENT_DELETED 1
#  endif
#  if !LIBVIR_CHECK_VERSION(7, 3, 0)
#    define VIR_NODE_DEVICE_EVENT_DEFINED 2
#  endif
#  if !LIBVIR_CHECK_VERSION(7, 3, 0)
#    define VIR_NODE_DEVICE_EVENT_UNDEFINED 3
#  endif
#  if !LIBVIR_CHECK_VERSION(2, 2, 0)
#    define VIR_NODE_DEVICE_EVENT_LAST 4
#  endif

/* enum virNodeGetCPUStatsAllCPUs */
#  if !LIBVIR_CHECK_VERSION(0, 9, 3)
#    define VIR_NODE_CPU_STATS_ALL_CPUS -1
#  endif

/* enum virNodeGetMemoryStatsAllCells */
#  if !LIBVIR_CHECK_VERSION(0, 9, 3)
#    define VIR_NODE_MEMORY_STATS_ALL_CELLS -1
#  endif

/* enum virNodeSuspendTarget */
#  if !LIBVIR_CHECK_VERSION(0, 9, 8)
#    define VIR_NODE_SUSPEND_TARGET_MEM 0
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 8)
#    define VIR_NODE_SUSPEND_TARGET_DISK 1
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 8)
#    define VIR_NODE_SUSPEND_TARGET_HYBRID 2
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 8)
#    define VIR_NODE_SUSPEND_TARGET_LAST 3
#  endif

/* enum virSchedParameterType */
#  if !LIBVIR_CHECK_VERSION(0, 2, 3)
#    define VIR_DOMAIN_SCHED_FIELD_INT VIR_TYPED_PARAM_INT
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 2, 3)
#    define VIR_DOMAIN_SCHED_FIELD_UINT VIR_TYPED_PARAM_UINT
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 2, 3)
#    define VIR_DOMAIN_SCHED_FIELD_LLONG VIR_TYPED_PARAM_LLONG
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 2, 3)
#    define VIR_DOMAIN_SCHED_FIELD_ULLONG VIR_TYPED_PARAM_ULLONG
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 2, 3)
#    define VIR_DOMAIN_SCHED_FIELD_DOUBLE VIR_TYPED_PARAM_DOUBLE
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 2, 3)
#    define VIR_DOMAIN_SCHED_FIELD_BOOLEAN VIR_TYPED_PARAM_BOOLEAN
#  endif

/* enum virSecretDefineFlags */
#  if !LIBVIR_CHECK_VERSION(7, 7, 0)
#    define VIR_SECRET_DEFINE_VALIDATE (1 << 0)
#  endif

/* enum virSecretEventID */
#  if !LIBVIR_CHECK_VERSION(3, 0, 0)
#    define VIR_SECRET_EVENT_ID_LIFECYCLE 0
#  endif
#  if !LIBVIR_CHECK_VERSION(3, 0, 0)
#    define VIR_SECRET_EVENT_ID_VALUE_CHANGED 1
#  endif
#  if !LIBVIR_CHECK_VERSION(3, 0, 0)
#    define VIR_SECRET_EVENT_ID_LAST 2
#  endif

/* enum virSecretEventLifecycleType */
#  if !LIBVIR_CHECK_VERSION(3, 0, 0)
#    define VIR_SECRET_EVENT_DEFINED 0
#  endif
#  if !LIBVIR_CHECK_VERSION(3, 0, 0)
#    define VIR_SECRET_EVENT_UNDEFINED 1
#  endif
#  if !LIBVIR_CHECK_VERSION(3, 0, 0)
#    define VIR_SECRET_EVENT_LAST 2
#  endif

/* enum virSecretUsageType */
#  if !LIBVIR_CHECK_VERSION(0, 7, 1)
#    define VIR_SECRET_USAGE_TYPE_NONE 0
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 7, 1)
#    define VIR_SECRET_USAGE_TYPE_VOLUME 1
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 7)
#    define VIR_SECRET_USAGE_TYPE_CEPH 2
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 0, 4)
#    define VIR_SECRET_USAGE_TYPE_ISCSI 3
#  endif
#  if !LIBVIR_CHECK_VERSION(2, 3, 0)
#    define VIR_SECRET_USAGE_TYPE_TLS 4
#  endif
#  if !LIBVIR_CHECK_VERSION(5, 6, 0)
#    define VIR_SECRET_USAGE_TYPE_VTPM 5
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 7)
#    define VIR_SECRET_USAGE_TYPE_LAST 6
#  endif

/* enum virStoragePoolBuildFlags */
#  if !LIBVIR_CHECK_VERSION(0, 4, 1)
#    define VIR_STORAGE_POOL_BUILD_NEW 0
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 4, 1)
#    define VIR_STORAGE_POOL_BUILD_REPAIR (1 << 0)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 4, 1)
#    define VIR_STORAGE_POOL_BUILD_RESIZE (1 << 1)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 5)
#    define VIR_STORAGE_POOL_BUILD_NO_OVERWRITE (1 << 2)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 5)
#    define VIR_STORAGE_POOL_BUILD_OVERWRITE (1 << 3)
#  endif

/* enum virStoragePoolCreateFlags */
#  if !LIBVIR_CHECK_VERSION(1, 3, 1)
#    define VIR_STORAGE_POOL_CREATE_NORMAL 0
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 3, 1)
#    define VIR_STORAGE_POOL_CREATE_WITH_BUILD (1 << 0)
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 3, 1)
#    define VIR_STORAGE_POOL_CREATE_WITH_BUILD_OVERWRITE (1 << 1)
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 3, 1)
#    define VIR_STORAGE_POOL_CREATE_WITH_BUILD_NO_OVERWRITE (1 << 2)
#  endif

/* enum virStoragePoolDefineFlags */
#  if !LIBVIR_CHECK_VERSION(7, 7, 0)
#    define VIR_STORAGE_POOL_DEFINE_VALIDATE (1 << 0)
#  endif

/* enum virStoragePoolDeleteFlags */
#  if !LIBVIR_CHECK_VERSION(0, 4, 1)
#    define VIR_STORAGE_POOL_DELETE_NORMAL 0
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 4, 1)
#    define VIR_STORAGE_POOL_DELETE_ZEROED (1 << 0)
#  endif

/* enum virStoragePoolEventID */
#  if !LIBVIR_CHECK_VERSION(2, 0, 0)
#    define VIR_STORAGE_POOL_EVENT_ID_LIFECYCLE 0
#  endif
#  if !LIBVIR_CHECK_VERSION(2, 0, 0)
#    define VIR_STORAGE_POOL_EVENT_ID_REFRESH 1
#  endif
#  if !LIBVIR_CHECK_VERSION(2, 0, 0)
#    define VIR_STORAGE_POOL_EVENT_ID_LAST 2
#  endif

/* enum virStoragePoolEventLifecycleType */
#  if !LIBVIR_CHECK_VERSION(2, 0, 0)
#    define VIR_STORAGE_POOL_EVENT_DEFINED 0
#  endif
#  if !LIBVIR_CHECK_VERSION(2, 0, 0)
#    define VIR_STORAGE_POOL_EVENT_UNDEFINED 1
#  endif
#  if !LIBVIR_CHECK_VERSION(2, 0, 0)
#    define VIR_STORAGE_POOL_EVENT_STARTED 2
#  endif
#  if !LIBVIR_CHECK_VERSION(2, 0, 0)
#    define VIR_STORAGE_POOL_EVENT_STOPPED 3
#  endif
#  if !LIBVIR_CHECK_VERSION(3, 8, 0)
#    define VIR_STORAGE_POOL_EVENT_CREATED 4
#  endif
#  if !LIBVIR_CHECK_VERSION(3, 8, 0)
#    define VIR_STORAGE_POOL_EVENT_DELETED 5
#  endif
#  if !LIBVIR_CHECK_VERSION(2, 0, 0)
#    define VIR_STORAGE_POOL_EVENT_LAST 6
#  endif

/* enum virStoragePoolState */
#  if !LIBVIR_CHECK_VERSION(0, 4, 1)
#    define VIR_STORAGE_POOL_INACTIVE 0
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 4, 1)
#    define VIR_STORAGE_POOL_BUILDING 1
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 4, 1)
#    define VIR_STORAGE_POOL_RUNNING 2
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 4, 1)
#    define VIR_STORAGE_POOL_DEGRADED 3
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 8, 2)
#    define VIR_STORAGE_POOL_INACCESSIBLE 4
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 10)
#    define VIR_STORAGE_POOL_STATE_LAST 5
#  endif

/* enum virStorageVolCreateFlags */
#  if !LIBVIR_CHECK_VERSION(1, 0, 1)
#    define VIR_STORAGE_VOL_CREATE_PREALLOC_METADATA (1 << 0)
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 13)
#    define VIR_STORAGE_VOL_CREATE_REFLINK (1 << 1)
#  endif
#  if !LIBVIR_CHECK_VERSION(8, 10, 0)
#    define VIR_STORAGE_VOL_CREATE_VALIDATE (1 << 2)
#  endif

/* enum virStorageVolDeleteFlags */
#  if !LIBVIR_CHECK_VERSION(0, 4, 1)
#    define VIR_STORAGE_VOL_DELETE_NORMAL 0
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 4, 1)
#    define VIR_STORAGE_VOL_DELETE_ZEROED (1 << 0)
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 21)
#    define VIR_STORAGE_VOL_DELETE_WITH_SNAPSHOTS (1 << 1)
#  endif

/* enum virStorageVolDownloadFlags */
#  if !LIBVIR_CHECK_VERSION(3, 4, 0)
#    define VIR_STORAGE_VOL_DOWNLOAD_SPARSE_STREAM (1 << 0)
#  endif

/* enum virStorageVolInfoFlags */
#  if !LIBVIR_CHECK_VERSION(3, 0, 0)
#    define VIR_STORAGE_VOL_USE_ALLOCATION 0
#  endif
#  if !LIBVIR_CHECK_VERSION(3, 0, 0)
#    define VIR_STORAGE_VOL_GET_PHYSICAL (1 << 0)
#  endif

/* enum virStorageVolResizeFlags */
#  if !LIBVIR_CHECK_VERSION(0, 9, 10)
#    define VIR_STORAGE_VOL_RESIZE_ALLOCATE (1 << 0)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 10)
#    define VIR_STORAGE_VOL_RESIZE_DELTA (1 << 1)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 10)
#    define VIR_STORAGE_VOL_RESIZE_SHRINK (1 << 2)
#  endif

/* enum virStorageVolType */
#  if !LIBVIR_CHECK_VERSION(0, 4, 1)
#    define VIR_STORAGE_VOL_FILE 0
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 4, 1)
#    define VIR_STORAGE_VOL_BLOCK 1
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 5)
#    define VIR_STORAGE_VOL_DIR 2
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 13)
#    define VIR_STORAGE_VOL_NETWORK 3
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 0)
#    define VIR_STORAGE_VOL_NETDIR 4
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 3, 4)
#    define VIR_STORAGE_VOL_PLOOP 5
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 10)
#    define VIR_STORAGE_VOL_LAST 6
#  endif

/* enum virStorageVolUploadFlags */
#  if !LIBVIR_CHECK_VERSION(3, 4, 0)
#    define VIR_STORAGE_VOL_UPLOAD_SPARSE_STREAM (1 << 0)
#  endif

/* enum virStorageVolWipeAlgorithm */
#  if !LIBVIR_CHECK_VERSION(0, 9, 10)
#    define VIR_STORAGE_VOL_WIPE_ALG_ZERO 0
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 10)
#    define VIR_STORAGE_VOL_WIPE_ALG_NNSA 1
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 10)
#    define VIR_STORAGE_VOL_WIPE_ALG_DOD 2
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 10)
#    define VIR_STORAGE_VOL_WIPE_ALG_BSI 3
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 10)
#    define VIR_STORAGE_VOL_WIPE_ALG_GUTMANN 4
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 10)
#    define VIR_STORAGE_VOL_WIPE_ALG_SCHNEIER 5
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 10)
#    define VIR_STORAGE_VOL_WIPE_ALG_PFITZNER7 6
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 10)
#    define VIR_STORAGE_VOL_WIPE_ALG_PFITZNER33 7
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 10)
#    define VIR_STORAGE_VOL_WIPE_ALG_RANDOM 8
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 3, 2)
#    define VIR_STORAGE_VOL_WIPE_ALG_TRIM 9
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 10)
#    define VIR_STORAGE_VOL_WIPE_ALG_LAST 10
#  endif

/* enum virStorageXMLFlags */
#  if !LIBVIR_CHECK_VERSION(0, 9, 13)
#    define VIR_STORAGE_XML_INACTIVE (1 << 0)
#  endif

/* enum virStreamEventType */
#  if !LIBVIR_CHECK_VERSION(0, 7, 2)
#    define VIR_STREAM_EVENT_READABLE (1 << 0)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 7, 2)
#    define VIR_STREAM_EVENT_WRITABLE (1 << 1)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 7, 2)
#    define VIR_STREAM_EVENT_ERROR (1 << 2)
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 7, 2)
#    define VIR_STREAM_EVENT_HANGUP (1 << 3)
#  endif

/* enum virStreamFlags */
#  if !LIBVIR_CHECK_VERSION(0, 7, 2)
#    define VIR_STREAM_NONBLOCK (1 << 0)
#  endif

/* enum virStreamRecvFlagsValues */
#  if !LIBVIR_CHECK_VERSION(3, 4, 0)
#    define VIR_STREAM_RECV_STOP_AT_HOLE (1 << 0)
#  endif

/* enum virTypedParameterFlags */
#  if !LIBVIR_CHECK_VERSION(0, 9, 8)
#    define VIR_TYPED_PARAM_STRING_OKAY (1 << 2)
#  endif

/* enum virTypedParameterType */
#  if !LIBVIR_CHECK_VERSION(0, 9, 2)
#    define VIR_TYPED_PARAM_INT 1
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 2)
#    define VIR_TYPED_PARAM_UINT 2
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 2)
#    define VIR_TYPED_PARAM_LLONG 3
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 2)
#    define VIR_TYPED_PARAM_ULLONG 4
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 2)
#    define VIR_TYPED_PARAM_DOUBLE 5
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 2)
#    define VIR_TYPED_PARAM_BOOLEAN 6
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 8)
#    define VIR_TYPED_PARAM_STRING 7
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 10)
#    define VIR_TYPED_PARAM_LAST 8
#  endif

/* enum virVcpuHostCpuState */
#  if !LIBVIR_CHECK_VERSION(6, 10, 0)
#    define VIR_VCPU_INFO_CPU_UNAVAILABLE -2
#  endif
#  if !LIBVIR_CHECK_VERSION(6, 10, 0)
#    define VIR_VCPU_INFO_CPU_OFFLINE -1
#  endif

/* enum virVcpuState */
#  if !LIBVIR_CHECK_VERSION(0, 1, 4)
#    define VIR_VCPU_OFFLINE 0
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 1, 4)
#    define VIR_VCPU_RUNNING 1
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 1, 4)
#    define VIR_VCPU_BLOCKED 2
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 9, 10)
#    define VIR_VCPU_LAST 3
#  endif

