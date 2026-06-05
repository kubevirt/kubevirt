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

#if !LIBVIR_CHECK_VERSION(0, 4, 1)
struct _virConnectAuth {
  int * credtype;
  unsigned int ncredtype;
  virConnectAuthCallbackPtr cb;
  void * cbdata;
};
#endif


#if !LIBVIR_CHECK_VERSION(0, 4, 1)
struct _virConnectCredential {
  int type;
  const char * prompt;
  const char * challenge;
  const char * defresult;
  char * result;
  unsigned int resultlen;
};
#endif


#if !LIBVIR_CHECK_VERSION(0, 8, 1)
struct _virDomainBlockInfo {
  unsigned long long capacity;
  unsigned long long allocation;
  unsigned long long physical;
};
#endif


#if !LIBVIR_CHECK_VERSION(0, 9, 4)
struct _virDomainBlockJobInfo {
  int type;
  unsigned long bandwidth;
  virDomainBlockJobCursor cur;
  virDomainBlockJobCursor end;
};
#endif


#if !LIBVIR_CHECK_VERSION(0, 3, 3)
struct _virDomainBlockStats {
  long long rd_req;
  long long rd_bytes;
  long long wr_req;
  long long wr_bytes;
  long long errs;
};
#endif


#if !LIBVIR_CHECK_VERSION(0, 9, 3)
struct _virDomainControlInfo {
  unsigned int state;
  unsigned int details;
  unsigned long long stateTime;
};
#endif


#if !LIBVIR_CHECK_VERSION(0, 9, 10)
struct _virDomainDiskError {
  char * disk;
  int error;
};
#endif


#if !LIBVIR_CHECK_VERSION(0, 8, 0)
struct _virDomainEventGraphicsAddress {
  int family;
  char * node;
  char * service;
};
#endif


#if !LIBVIR_CHECK_VERSION(0, 8, 0)
struct _virDomainEventGraphicsSubject {
  int nidentity;
  virDomainEventGraphicsSubjectIdentityPtr identities;
};
#endif


#if !LIBVIR_CHECK_VERSION(0, 8, 0)
struct _virDomainEventGraphicsSubjectIdentity {
  char * type;
  char * name;
};
#endif


#if !LIBVIR_CHECK_VERSION(1, 2, 11)
struct _virDomainFSInfo {
  char * mountpoint;
  char * name;
  char * fstype;
  size_t ndevAlias;
  char ** devAlias;
};
#endif


#if !LIBVIR_CHECK_VERSION(1, 2, 14)
struct _virDomainIOThreadInfo {
  unsigned int iothread_id;
  unsigned char * cpumap;
  int cpumaplen;
};
#endif


#if !LIBVIR_CHECK_VERSION(0, 0, 1)
struct _virDomainInfo {
  unsigned char state;
  unsigned long maxMem;
  unsigned long memory;
  unsigned short nrVirtCpu;
  unsigned long long cpuTime;
};
#endif


#if !LIBVIR_CHECK_VERSION(1, 2, 14)
struct _virDomainInterface {
  char * name;
  char * hwaddr;
  unsigned int naddrs;
  virDomainIPAddressPtr addrs;
};
#endif


#if !LIBVIR_CHECK_VERSION(1, 2, 14)
struct _virDomainInterfaceIPAddress {
  int type;
  char * addr;
  unsigned int prefix;
};
#endif


#if !LIBVIR_CHECK_VERSION(0, 3, 3)
struct _virDomainInterfaceStats {
  long long rx_bytes;
  long long rx_packets;
  long long rx_errs;
  long long rx_drop;
  long long tx_bytes;
  long long tx_packets;
  long long tx_errs;
  long long tx_drop;
};
#endif


#if !LIBVIR_CHECK_VERSION(0, 7, 7)
struct _virDomainJobInfo {
  int type;
  unsigned long long timeElapsed;
  unsigned long long timeRemaining;
  unsigned long long dataTotal;
  unsigned long long dataProcessed;
  unsigned long long dataRemaining;
  unsigned long long memTotal;
  unsigned long long memProcessed;
  unsigned long long memRemaining;
  unsigned long long fileTotal;
  unsigned long long fileProcessed;
  unsigned long long fileRemaining;
};
#endif


#if !LIBVIR_CHECK_VERSION(0, 7, 5)
struct _virDomainMemoryStat {
  int tag;
  unsigned long long val;
};
#endif


#if !LIBVIR_CHECK_VERSION(1, 2, 8)
struct _virDomainStatsRecord {
  virDomainPtr dom;
  virTypedParameterPtr params;
  int nparams;
};
#endif


#if !LIBVIR_CHECK_VERSION(0, 1, 0)
struct _virError {
  int code;
  int domain;
  char * message;
  virErrorLevel level;
  virConnectPtr conn;
  virDomainPtr dom;
  char * str1;
  char * str2;
  char * str3;
  int int1;
  int int2;
  virNetworkPtr net;
};
#endif


#if !LIBVIR_CHECK_VERSION(1, 2, 6)
struct _virNetworkDHCPLease {
  char * iface;
  long long expirytime;
  int type;
  char * mac;
  char * iaid;
  char * ipaddr;
  unsigned int prefix;
  char * hostname;
  char * clientid;
};
#endif


#if !LIBVIR_CHECK_VERSION(0, 9, 3)
struct _virNodeCPUStats {
  char field[VIR_NODE_CPU_STATS_FIELD_LENGTH];
  unsigned long long value;
};
#endif


#if !LIBVIR_CHECK_VERSION(0, 1, 0)
struct _virNodeInfo {
  char model[32];
  unsigned long memory;
  unsigned int cpus;
  unsigned int mhz;
  unsigned int nodes;
  unsigned int sockets;
  unsigned int cores;
  unsigned int threads;
};
#endif


#if !LIBVIR_CHECK_VERSION(0, 9, 3)
struct _virNodeMemoryStats {
  char field[VIR_NODE_MEMORY_STATS_FIELD_LENGTH];
  unsigned long long value;
};
#endif


#if !LIBVIR_CHECK_VERSION(0, 6, 1)
struct _virSecurityLabel {
  char label[VIR_SECURITY_LABEL_BUFLEN];
  int enforcing;
};
#endif


#if !LIBVIR_CHECK_VERSION(0, 6, 1)
struct _virSecurityModel {
  char model[VIR_SECURITY_MODEL_BUFLEN];
  char doi[VIR_SECURITY_DOI_BUFLEN];
};
#endif


#if !LIBVIR_CHECK_VERSION(0, 4, 1)
struct _virStoragePoolInfo {
  int state;
  unsigned long long capacity;
  unsigned long long allocation;
  unsigned long long available;
};
#endif


#if !LIBVIR_CHECK_VERSION(0, 4, 1)
struct _virStorageVolInfo {
  int type;
  unsigned long long capacity;
  unsigned long long allocation;
};
#endif


#if !LIBVIR_CHECK_VERSION(0, 9, 0)
struct _virTypedParameter {
  char field[VIR_TYPED_PARAM_FIELD_LENGTH];
  int type;
  union {
    int i;
    unsigned int ui;
    long long int l;
    unsigned long long int ul;
    double d;
    char b;
    char * s;
  } value;
};
#endif


#if !LIBVIR_CHECK_VERSION(0, 1, 4)
struct _virVcpuInfo {
  unsigned int number;
  int state;
  unsigned long long cpuTime;
  int cpu;
};
#endif

