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
#  define LIBVIR_VERSION_NUMBER 11002000
#endif

#if !LIBVIR_CHECK_VERSION(5, 8, 0)
#  define VIR_CONNECT_IDENTITY_GROUP_NAME "group-name"
#endif

#if !LIBVIR_CHECK_VERSION(5, 8, 0)
#  define VIR_CONNECT_IDENTITY_PROCESS_ID "process-id"
#endif

#if !LIBVIR_CHECK_VERSION(5, 8, 0)
#  define VIR_CONNECT_IDENTITY_PROCESS_TIME "process-time"
#endif

#if !LIBVIR_CHECK_VERSION(5, 8, 0)
#  define VIR_CONNECT_IDENTITY_SASL_USER_NAME "sasl-user-name"
#endif

#if !LIBVIR_CHECK_VERSION(5, 8, 0)
#  define VIR_CONNECT_IDENTITY_SELINUX_CONTEXT "selinux-context"
#endif

#if !LIBVIR_CHECK_VERSION(5, 8, 0)
#  define VIR_CONNECT_IDENTITY_UNIX_GROUP_ID "unix-group-id"
#endif

#if !LIBVIR_CHECK_VERSION(5, 8, 0)
#  define VIR_CONNECT_IDENTITY_UNIX_USER_ID "unix-user-id"
#endif

#if !LIBVIR_CHECK_VERSION(5, 8, 0)
#  define VIR_CONNECT_IDENTITY_USER_NAME "user-name"
#endif

#if !LIBVIR_CHECK_VERSION(5, 8, 0)
#  define VIR_CONNECT_IDENTITY_X509_DISTINGUISHED_NAME "x509-distinguished-name"
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 9)
#  define VIR_DOMAIN_BANDWIDTH_IN_AVERAGE "inbound.average"
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 9)
#  define VIR_DOMAIN_BANDWIDTH_IN_BURST "inbound.burst"
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 19)
#  define VIR_DOMAIN_BANDWIDTH_IN_FLOOR "inbound.floor"
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 9)
#  define VIR_DOMAIN_BANDWIDTH_IN_PEAK "inbound.peak"
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 9)
#  define VIR_DOMAIN_BANDWIDTH_OUT_AVERAGE "outbound.average"
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 9)
#  define VIR_DOMAIN_BANDWIDTH_OUT_BURST "outbound.burst"
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 9)
#  define VIR_DOMAIN_BANDWIDTH_OUT_PEAK "outbound.peak"
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 2)
#  define VIR_DOMAIN_BLKIO_DEVICE_READ_BPS "device_read_bytes_sec"
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 2)
#  define VIR_DOMAIN_BLKIO_DEVICE_READ_IOPS "device_read_iops_sec"
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 8)
#  define VIR_DOMAIN_BLKIO_DEVICE_WEIGHT "device_weight"
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 2)
#  define VIR_DOMAIN_BLKIO_DEVICE_WRITE_BPS "device_write_bytes_sec"
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 2)
#  define VIR_DOMAIN_BLKIO_DEVICE_WRITE_IOPS "device_write_iops_sec"
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 0)
#  define VIR_DOMAIN_BLKIO_FIELD_LENGTH VIR_TYPED_PARAM_FIELD_LENGTH
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 0)
#  define VIR_DOMAIN_BLKIO_WEIGHT "weight"
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 8)
#  define VIR_DOMAIN_BLOCK_COPY_BANDWIDTH "bandwidth"
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 8)
#  define VIR_DOMAIN_BLOCK_COPY_BUF_SIZE "buf-size"
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 8)
#  define VIR_DOMAIN_BLOCK_COPY_GRANULARITY "granularity"
#endif

#if !LIBVIR_CHECK_VERSION(3, 0, 0)
#  define VIR_DOMAIN_BLOCK_IOTUNE_GROUP_NAME "group_name"
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 8)
#  define VIR_DOMAIN_BLOCK_IOTUNE_READ_BYTES_SEC "read_bytes_sec"
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 11)
#  define VIR_DOMAIN_BLOCK_IOTUNE_READ_BYTES_SEC_MAX "read_bytes_sec_max"
#endif

#if !LIBVIR_CHECK_VERSION(2, 4, 0)
#  define VIR_DOMAIN_BLOCK_IOTUNE_READ_BYTES_SEC_MAX_LENGTH "read_bytes_sec_max_length"
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 8)
#  define VIR_DOMAIN_BLOCK_IOTUNE_READ_IOPS_SEC "read_iops_sec"
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 11)
#  define VIR_DOMAIN_BLOCK_IOTUNE_READ_IOPS_SEC_MAX "read_iops_sec_max"
#endif

#if !LIBVIR_CHECK_VERSION(2, 4, 0)
#  define VIR_DOMAIN_BLOCK_IOTUNE_READ_IOPS_SEC_MAX_LENGTH "read_iops_sec_max_length"
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 11)
#  define VIR_DOMAIN_BLOCK_IOTUNE_SIZE_IOPS_SEC "size_iops_sec"
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 8)
#  define VIR_DOMAIN_BLOCK_IOTUNE_TOTAL_BYTES_SEC "total_bytes_sec"
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 11)
#  define VIR_DOMAIN_BLOCK_IOTUNE_TOTAL_BYTES_SEC_MAX "total_bytes_sec_max"
#endif

#if !LIBVIR_CHECK_VERSION(2, 4, 0)
#  define VIR_DOMAIN_BLOCK_IOTUNE_TOTAL_BYTES_SEC_MAX_LENGTH "total_bytes_sec_max_length"
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 8)
#  define VIR_DOMAIN_BLOCK_IOTUNE_TOTAL_IOPS_SEC "total_iops_sec"
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 11)
#  define VIR_DOMAIN_BLOCK_IOTUNE_TOTAL_IOPS_SEC_MAX "total_iops_sec_max"
#endif

#if !LIBVIR_CHECK_VERSION(2, 4, 0)
#  define VIR_DOMAIN_BLOCK_IOTUNE_TOTAL_IOPS_SEC_MAX_LENGTH "total_iops_sec_max_length"
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 8)
#  define VIR_DOMAIN_BLOCK_IOTUNE_WRITE_BYTES_SEC "write_bytes_sec"
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 11)
#  define VIR_DOMAIN_BLOCK_IOTUNE_WRITE_BYTES_SEC_MAX "write_bytes_sec_max"
#endif

#if !LIBVIR_CHECK_VERSION(2, 4, 0)
#  define VIR_DOMAIN_BLOCK_IOTUNE_WRITE_BYTES_SEC_MAX_LENGTH "write_bytes_sec_max_length"
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 8)
#  define VIR_DOMAIN_BLOCK_IOTUNE_WRITE_IOPS_SEC "write_iops_sec"
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 11)
#  define VIR_DOMAIN_BLOCK_IOTUNE_WRITE_IOPS_SEC_MAX "write_iops_sec_max"
#endif

#if !LIBVIR_CHECK_VERSION(2, 4, 0)
#  define VIR_DOMAIN_BLOCK_IOTUNE_WRITE_IOPS_SEC_MAX_LENGTH "write_iops_sec_max_length"
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 5)
#  define VIR_DOMAIN_BLOCK_STATS_ERRS "errs"
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 5)
#  define VIR_DOMAIN_BLOCK_STATS_FIELD_LENGTH VIR_TYPED_PARAM_FIELD_LENGTH
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 5)
#  define VIR_DOMAIN_BLOCK_STATS_FLUSH_REQ "flush_operations"
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 5)
#  define VIR_DOMAIN_BLOCK_STATS_FLUSH_TOTAL_TIMES "flush_total_times"
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 5)
#  define VIR_DOMAIN_BLOCK_STATS_READ_BYTES "rd_bytes"
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 5)
#  define VIR_DOMAIN_BLOCK_STATS_READ_REQ "rd_operations"
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 5)
#  define VIR_DOMAIN_BLOCK_STATS_READ_TOTAL_TIMES "rd_total_times"
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 5)
#  define VIR_DOMAIN_BLOCK_STATS_WRITE_BYTES "wr_bytes"
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 5)
#  define VIR_DOMAIN_BLOCK_STATS_WRITE_REQ "wr_operations"
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 5)
#  define VIR_DOMAIN_BLOCK_STATS_WRITE_TOTAL_TIMES "wr_total_times"
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 10)
#  define VIR_DOMAIN_CPU_STATS_CPUTIME "cpu_time"
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 11)
#  define VIR_DOMAIN_CPU_STATS_SYSTEMTIME "system_time"
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 11)
#  define VIR_DOMAIN_CPU_STATS_USERTIME "user_time"
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 13)
#  define VIR_DOMAIN_CPU_STATS_VCPUTIME "vcpu_time"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_GUEST_INFO_DISK_COUNT "disk.count"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_GUEST_INFO_DISK_PREFIX "disk."
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_GUEST_INFO_DISK_SUFFIX_ALIAS ".alias"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_GUEST_INFO_DISK_SUFFIX_DEPENDENCY_COUNT ".dependency.count"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_GUEST_INFO_DISK_SUFFIX_DEPENDENCY_PREFIX ".dependency."
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_GUEST_INFO_DISK_SUFFIX_DEPENDENCY_SUFFIX_NAME ".name"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_GUEST_INFO_DISK_SUFFIX_GUEST_ALIAS ".guest_alias"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_GUEST_INFO_DISK_SUFFIX_GUEST_BUS ".guest_bus"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_GUEST_INFO_DISK_SUFFIX_NAME ".name"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_GUEST_INFO_DISK_SUFFIX_PARTITION ".partition"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_GUEST_INFO_DISK_SUFFIX_SERIAL ".serial"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_GUEST_INFO_FS_COUNT "fs.count"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_GUEST_INFO_FS_PREFIX "fs."
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_GUEST_INFO_FS_SUFFIX_DISK_COUNT ".disk.count"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_GUEST_INFO_FS_SUFFIX_DISK_PREFIX ".disk."
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_GUEST_INFO_FS_SUFFIX_DISK_SUFFIX_ALIAS ".alias"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_GUEST_INFO_FS_SUFFIX_DISK_SUFFIX_DEVICE ".device"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_GUEST_INFO_FS_SUFFIX_DISK_SUFFIX_SERIAL ".serial"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_GUEST_INFO_FS_SUFFIX_FSTYPE ".fstype"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_GUEST_INFO_FS_SUFFIX_MOUNTPOINT ".mountpoint"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_GUEST_INFO_FS_SUFFIX_NAME ".name"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_GUEST_INFO_FS_SUFFIX_TOTAL_BYTES ".total-bytes"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_GUEST_INFO_FS_SUFFIX_USED_BYTES ".used-bytes"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_GUEST_INFO_HOSTNAME_HOSTNAME "hostname"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_GUEST_INFO_IF_COUNT "if.count"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_GUEST_INFO_IF_PREFIX "if."
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_GUEST_INFO_IF_SUFFIX_ADDR_COUNT ".addr.count"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_GUEST_INFO_IF_SUFFIX_ADDR_PREFIX ".addr."
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_GUEST_INFO_IF_SUFFIX_ADDR_SUFFIX_ADDR ".addr"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_GUEST_INFO_IF_SUFFIX_ADDR_SUFFIX_PREFIX ".prefix"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_GUEST_INFO_IF_SUFFIX_ADDR_SUFFIX_TYPE ".type"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_GUEST_INFO_IF_SUFFIX_HWADDR ".hwaddr"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_GUEST_INFO_IF_SUFFIX_NAME ".name"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_GUEST_INFO_LOAD_15M "load.15m"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_GUEST_INFO_LOAD_1M "load.1m"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_GUEST_INFO_LOAD_5M "load.5m"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_GUEST_INFO_OS_ID "os.id"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_GUEST_INFO_OS_KERNEL_RELEASE "os.kernel-release"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_GUEST_INFO_OS_KERNEL_VERSION "os.kernel-version"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_GUEST_INFO_OS_MACHINE "os.machine"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_GUEST_INFO_OS_NAME "os.name"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_GUEST_INFO_OS_PRETTY_NAME "os.pretty-name"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_GUEST_INFO_OS_VARIANT "os.variant"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_GUEST_INFO_OS_VARIANT_ID "os.variant-id"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_GUEST_INFO_OS_VERSION "os.version"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_GUEST_INFO_OS_VERSION_ID "os.version-id"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_GUEST_INFO_TIMEZONE_NAME "timezone.name"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_GUEST_INFO_TIMEZONE_OFFSET "timezone.offset"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_GUEST_INFO_USER_COUNT "user.count"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_GUEST_INFO_USER_PREFIX "user."
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_GUEST_INFO_USER_SUFFIX_DOMAIN ".domain"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_GUEST_INFO_USER_SUFFIX_LOGIN_TIME ".login-time"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_GUEST_INFO_USER_SUFFIX_NAME ".name"
#endif

#if !LIBVIR_CHECK_VERSION(4, 10, 0)
#  define VIR_DOMAIN_IOTHREAD_POLL_GROW "poll_grow"
#endif

#if !LIBVIR_CHECK_VERSION(4, 10, 0)
#  define VIR_DOMAIN_IOTHREAD_POLL_MAX_NS "poll_max_ns"
#endif

#if !LIBVIR_CHECK_VERSION(4, 10, 0)
#  define VIR_DOMAIN_IOTHREAD_POLL_SHRINK "poll_shrink"
#endif

#if !LIBVIR_CHECK_VERSION(8, 5, 0)
#  define VIR_DOMAIN_IOTHREAD_THREAD_POOL_MAX "thread_pool_max"
#endif

#if !LIBVIR_CHECK_VERSION(8, 5, 0)
#  define VIR_DOMAIN_IOTHREAD_THREAD_POOL_MIN "thread_pool_min"
#endif

#if !LIBVIR_CHECK_VERSION(2, 0, 0)
#  define VIR_DOMAIN_JOB_AUTO_CONVERGE_THROTTLE "auto_converge_throttle"
#endif

#if !LIBVIR_CHECK_VERSION(1, 0, 3)
#  define VIR_DOMAIN_JOB_COMPRESSION_BYTES "compression_bytes"
#endif

#if !LIBVIR_CHECK_VERSION(1, 0, 3)
#  define VIR_DOMAIN_JOB_COMPRESSION_CACHE "compression_cache"
#endif

#if !LIBVIR_CHECK_VERSION(1, 0, 3)
#  define VIR_DOMAIN_JOB_COMPRESSION_CACHE_MISSES "compression_cache_misses"
#endif

#if !LIBVIR_CHECK_VERSION(1, 0, 3)
#  define VIR_DOMAIN_JOB_COMPRESSION_OVERFLOW "compression_overflow"
#endif

#if !LIBVIR_CHECK_VERSION(1, 0, 3)
#  define VIR_DOMAIN_JOB_COMPRESSION_PAGES "compression_pages"
#endif

#if !LIBVIR_CHECK_VERSION(1, 0, 3)
#  define VIR_DOMAIN_JOB_DATA_PROCESSED "data_processed"
#endif

#if !LIBVIR_CHECK_VERSION(1, 0, 3)
#  define VIR_DOMAIN_JOB_DATA_REMAINING "data_remaining"
#endif

#if !LIBVIR_CHECK_VERSION(1, 0, 3)
#  define VIR_DOMAIN_JOB_DATA_TOTAL "data_total"
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 9)
#  define VIR_DOMAIN_JOB_DISK_BPS "disk_bps"
#endif

#if !LIBVIR_CHECK_VERSION(1, 0, 3)
#  define VIR_DOMAIN_JOB_DISK_PROCESSED "disk_processed"
#endif

#if !LIBVIR_CHECK_VERSION(1, 0, 3)
#  define VIR_DOMAIN_JOB_DISK_REMAINING "disk_remaining"
#endif

#if !LIBVIR_CHECK_VERSION(6, 0, 0)
#  define VIR_DOMAIN_JOB_DISK_TEMP_TOTAL "disk_temp_total"
#endif

#if !LIBVIR_CHECK_VERSION(6, 0, 0)
#  define VIR_DOMAIN_JOB_DISK_TEMP_USED "disk_temp_used"
#endif

#if !LIBVIR_CHECK_VERSION(1, 0, 3)
#  define VIR_DOMAIN_JOB_DISK_TOTAL "disk_total"
#endif

#if !LIBVIR_CHECK_VERSION(1, 0, 3)
#  define VIR_DOMAIN_JOB_DOWNTIME "downtime"
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 15)
#  define VIR_DOMAIN_JOB_DOWNTIME_NET "downtime_net"
#endif

#if !LIBVIR_CHECK_VERSION(6, 3, 0)
#  define VIR_DOMAIN_JOB_ERRMSG "errmsg"
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 9)
#  define VIR_DOMAIN_JOB_MEMORY_BPS "memory_bps"
#endif

#if !LIBVIR_CHECK_VERSION(1, 0, 3)
#  define VIR_DOMAIN_JOB_MEMORY_CONSTANT "memory_constant"
#endif

#if !LIBVIR_CHECK_VERSION(1, 3, 1)
#  define VIR_DOMAIN_JOB_MEMORY_DIRTY_RATE "memory_dirty_rate"
#endif

#if !LIBVIR_CHECK_VERSION(1, 3, 1)
#  define VIR_DOMAIN_JOB_MEMORY_ITERATION "memory_iteration"
#endif

#if !LIBVIR_CHECK_VERSION(1, 0, 3)
#  define VIR_DOMAIN_JOB_MEMORY_NORMAL "memory_normal"
#endif

#if !LIBVIR_CHECK_VERSION(1, 0, 3)
#  define VIR_DOMAIN_JOB_MEMORY_NORMAL_BYTES "memory_normal_bytes"
#endif

#if !LIBVIR_CHECK_VERSION(3, 9, 0)
#  define VIR_DOMAIN_JOB_MEMORY_PAGE_SIZE "memory_page_size"
#endif

#if !LIBVIR_CHECK_VERSION(5, 0, 0)
#  define VIR_DOMAIN_JOB_MEMORY_POSTCOPY_REQS "memory_postcopy_requests"
#endif

#if !LIBVIR_CHECK_VERSION(1, 0, 3)
#  define VIR_DOMAIN_JOB_MEMORY_PROCESSED "memory_processed"
#endif

#if !LIBVIR_CHECK_VERSION(1, 0, 3)
#  define VIR_DOMAIN_JOB_MEMORY_REMAINING "memory_remaining"
#endif

#if !LIBVIR_CHECK_VERSION(1, 0, 3)
#  define VIR_DOMAIN_JOB_MEMORY_TOTAL "memory_total"
#endif

#if !LIBVIR_CHECK_VERSION(3, 3, 0)
#  define VIR_DOMAIN_JOB_OPERATION "operation"
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 9)
#  define VIR_DOMAIN_JOB_SETUP_TIME "setup_time"
#endif

#if !LIBVIR_CHECK_VERSION(6, 0, 0)
#  define VIR_DOMAIN_JOB_SUCCESS "success"
#endif

#if !LIBVIR_CHECK_VERSION(1, 0, 3)
#  define VIR_DOMAIN_JOB_TIME_ELAPSED "time_elapsed"
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 15)
#  define VIR_DOMAIN_JOB_TIME_ELAPSED_NET "time_elapsed_net"
#endif

#if !LIBVIR_CHECK_VERSION(1, 0, 3)
#  define VIR_DOMAIN_JOB_TIME_REMAINING "time_remaining"
#endif

#if !LIBVIR_CHECK_VERSION(10, 6, 0)
#  define VIR_DOMAIN_JOB_VFIO_DATA_TRANSFERRED "vfio_data_transferred"
#endif

#if !LIBVIR_CHECK_VERSION(8, 0, 0)
#  define VIR_DOMAIN_LAUNCH_SECURITY_SEV_API_MAJOR "sev-api-major"
#endif

#if !LIBVIR_CHECK_VERSION(8, 0, 0)
#  define VIR_DOMAIN_LAUNCH_SECURITY_SEV_API_MINOR "sev-api-minor"
#endif

#if !LIBVIR_CHECK_VERSION(8, 0, 0)
#  define VIR_DOMAIN_LAUNCH_SECURITY_SEV_BUILD_ID "sev-build-id"
#endif

#if !LIBVIR_CHECK_VERSION(4, 5, 0)
#  define VIR_DOMAIN_LAUNCH_SECURITY_SEV_MEASUREMENT "sev-measurement"
#endif

#if !LIBVIR_CHECK_VERSION(8, 0, 0)
#  define VIR_DOMAIN_LAUNCH_SECURITY_SEV_POLICY "sev-policy"
#endif

#if !LIBVIR_CHECK_VERSION(8, 0, 0)
#  define VIR_DOMAIN_LAUNCH_SECURITY_SEV_SECRET "sev-secret"
#endif

#if !LIBVIR_CHECK_VERSION(8, 0, 0)
#  define VIR_DOMAIN_LAUNCH_SECURITY_SEV_SECRET_HEADER "sev-secret-header"
#endif

#if !LIBVIR_CHECK_VERSION(8, 0, 0)
#  define VIR_DOMAIN_LAUNCH_SECURITY_SEV_SECRET_SET_ADDRESS "sev-secret-set-address"
#endif

#if !LIBVIR_CHECK_VERSION(10, 5, 0)
#  define VIR_DOMAIN_LAUNCH_SECURITY_SEV_SNP_POLICY "sev-snp-policy"
#endif

#if !LIBVIR_CHECK_VERSION(0, 8, 5)
#  define VIR_DOMAIN_MEMORY_FIELD_LENGTH VIR_TYPED_PARAM_FIELD_LENGTH
#endif

#if !LIBVIR_CHECK_VERSION(0, 8, 5)
#  define VIR_DOMAIN_MEMORY_HARD_LIMIT "hard_limit"
#endif

#if !LIBVIR_CHECK_VERSION(0, 8, 5)
#  define VIR_DOMAIN_MEMORY_MIN_GUARANTEE "min_guarantee"
#endif

#if !LIBVIR_CHECK_VERSION(0, 8, 8)
#  define VIR_DOMAIN_MEMORY_PARAM_UNLIMITED 9007199254740991LL /* = INT64_MAX >> 10 */
#endif

#if !LIBVIR_CHECK_VERSION(0, 8, 5)
#  define VIR_DOMAIN_MEMORY_SOFT_LIMIT "soft_limit"
#endif

#if !LIBVIR_CHECK_VERSION(0, 8, 5)
#  define VIR_DOMAIN_MEMORY_SWAP_HARD_LIMIT "swap_hard_limit"
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 9)
#  define VIR_DOMAIN_NUMA_MODE "numa_mode"
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 9)
#  define VIR_DOMAIN_NUMA_NODESET "numa_nodeset"
#endif

#if !LIBVIR_CHECK_VERSION(8, 4, 0)
#  define VIR_DOMAIN_SAVE_PARAM_DXML "dxml"
#endif

#if !LIBVIR_CHECK_VERSION(8, 4, 0)
#  define VIR_DOMAIN_SAVE_PARAM_FILE "file"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_SAVE_PARAM_IMAGE_FORMAT "image_format"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_SAVE_PARAM_PARALLEL_CHANNELS "parallel.channels"
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 7)
#  define VIR_DOMAIN_SCHEDULER_CAP "cap"
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 7)
#  define VIR_DOMAIN_SCHEDULER_CPU_SHARES "cpu_shares"
#endif

#if !LIBVIR_CHECK_VERSION(0, 10, 0)
#  define VIR_DOMAIN_SCHEDULER_EMULATOR_PERIOD "emulator_period"
#endif

#if !LIBVIR_CHECK_VERSION(0, 10, 0)
#  define VIR_DOMAIN_SCHEDULER_EMULATOR_QUOTA "emulator_quota"
#endif

#if !LIBVIR_CHECK_VERSION(1, 3, 3)
#  define VIR_DOMAIN_SCHEDULER_GLOBAL_PERIOD "global_period"
#endif

#if !LIBVIR_CHECK_VERSION(1, 3, 3)
#  define VIR_DOMAIN_SCHEDULER_GLOBAL_QUOTA "global_quota"
#endif

#if !LIBVIR_CHECK_VERSION(2, 2, 0)
#  define VIR_DOMAIN_SCHEDULER_IOTHREAD_PERIOD "iothread_period"
#endif

#if !LIBVIR_CHECK_VERSION(2, 2, 0)
#  define VIR_DOMAIN_SCHEDULER_IOTHREAD_QUOTA "iothread_quota"
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 7)
#  define VIR_DOMAIN_SCHEDULER_LIMIT "limit"
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 7)
#  define VIR_DOMAIN_SCHEDULER_RESERVATION "reservation"
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 7)
#  define VIR_DOMAIN_SCHEDULER_SHARES "shares"
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 7)
#  define VIR_DOMAIN_SCHEDULER_VCPU_PERIOD "vcpu_period"
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 7)
#  define VIR_DOMAIN_SCHEDULER_VCPU_QUOTA "vcpu_quota"
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 7)
#  define VIR_DOMAIN_SCHEDULER_WEIGHT "weight"
#endif

#if !LIBVIR_CHECK_VERSION(0, 2, 3)
#  define VIR_DOMAIN_SCHED_FIELD_LENGTH VIR_TYPED_PARAM_FIELD_LENGTH
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 3)
#  define VIR_DOMAIN_SEND_KEY_MAX_KEYS 16
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_BALLOON_AVAILABLE "balloon.available"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_BALLOON_CURRENT "balloon.current"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_BALLOON_DISK_CACHES "balloon.disk_caches"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_BALLOON_HUGETLB_PGALLOC "balloon.hugetlb_pgalloc"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_BALLOON_HUGETLB_PGFAIL "balloon.hugetlb_pgfail"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_BALLOON_LAST_UPDATE "balloon.last-update"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_BALLOON_MAJOR_FAULT "balloon.major_fault"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_BALLOON_MAXIMUM "balloon.maximum"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_BALLOON_MINOR_FAULT "balloon.minor_fault"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_BALLOON_RSS "balloon.rss"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_BALLOON_SWAP_IN "balloon.swap_in"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_BALLOON_SWAP_OUT "balloon.swap_out"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_BALLOON_UNUSED "balloon.unused"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_BALLOON_USABLE "balloon.usable"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_BLOCK_COUNT "block.count"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_BLOCK_PREFIX "block."
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_BLOCK_SUFFIX_ALLOCATION ".allocation"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_BLOCK_SUFFIX_BACKINGINDEX ".backingIndex"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_BLOCK_SUFFIX_CAPACITY ".capacity"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_BLOCK_SUFFIX_ERRORS ".errors"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_BLOCK_SUFFIX_FL_REQS ".fl.reqs"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_BLOCK_SUFFIX_FL_TIMES ".fl.times"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_BLOCK_SUFFIX_NAME ".name"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_BLOCK_SUFFIX_PATH ".path"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_BLOCK_SUFFIX_PHYSICAL ".physical"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_BLOCK_SUFFIX_RD_BYTES ".rd.bytes"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_BLOCK_SUFFIX_RD_REQS ".rd.reqs"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_BLOCK_SUFFIX_RD_TIMES ".rd.times"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_BLOCK_SUFFIX_THRESHOLD ".threshold"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_BLOCK_SUFFIX_WR_BYTES ".wr.bytes"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_BLOCK_SUFFIX_WR_REQS ".wr.reqs"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_BLOCK_SUFFIX_WR_TIMES ".wr.times"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_CPU_CACHE_MONITOR_COUNT "cpu.cache.monitor.count"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_CPU_CACHE_MONITOR_PREFIX "cpu.cache.monitor."
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_CPU_CACHE_MONITOR_SUFFIX_BANK_COUNT ".bank.count"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_CPU_CACHE_MONITOR_SUFFIX_BANK_PREFIX ".bank."
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_CPU_CACHE_MONITOR_SUFFIX_BANK_SUFFIX_BYTES ".bytes"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_CPU_CACHE_MONITOR_SUFFIX_BANK_SUFFIX_ID ".id"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_CPU_CACHE_MONITOR_SUFFIX_NAME ".name"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_CPU_CACHE_MONITOR_SUFFIX_VCPUS ".vcpus"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_CPU_HALTPOLL_FAIL_TIME "cpu.haltpoll.fail.time"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_CPU_HALTPOLL_SUCCESS_TIME "cpu.haltpoll.success.time"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_CPU_SYSTEM "cpu.system"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_CPU_TIME "cpu.time"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_CPU_USER "cpu.user"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_CUSTOM_SUFFIX_TYPE_CUR ".cur"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_CUSTOM_SUFFIX_TYPE_MAX ".max"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_CUSTOM_SUFFIX_TYPE_SUM ".sum"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_DIRTYRATE_CALC_MODE "dirtyrate.calc_mode"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_DIRTYRATE_CALC_PERIOD "dirtyrate.calc_period"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_DIRTYRATE_CALC_START_TIME "dirtyrate.calc_start_time"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_DIRTYRATE_CALC_STATUS "dirtyrate.calc_status"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_DIRTYRATE_MEGABYTES_PER_SECOND "dirtyrate.megabytes_per_second"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_DIRTYRATE_VCPU_PREFIX "dirtyrate.vcpu."
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_DIRTYRATE_VCPU_SUFFIX_MEGABYTES_PER_SECOND ".megabytes_per_second"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_IOTHREAD_COUNT "iothread.count"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_IOTHREAD_PREFIX "iothread."
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_IOTHREAD_SUFFIX_POLL_GROW ".poll-grow"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_IOTHREAD_SUFFIX_POLL_MAX_NS ".poll-max-ns"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_IOTHREAD_SUFFIX_POLL_SHRINK ".poll-shrink"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_MEMORY_BANDWIDTH_MONITOR_COUNT "memory.bandwidth.monitor.count"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_MEMORY_BANDWIDTH_MONITOR_PREFIX "memory.bandwidth.monitor."
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_MEMORY_BANDWIDTH_MONITOR_SUFFIX_NAME ".name"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_MEMORY_BANDWIDTH_MONITOR_SUFFIX_NODE_COUNT ".node.count"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_MEMORY_BANDWIDTH_MONITOR_SUFFIX_NODE_PREFIX ".node."
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_MEMORY_BANDWIDTH_MONITOR_SUFFIX_NODE_SUFFIX_BYTES_LOCAL ".bytes.local"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_MEMORY_BANDWIDTH_MONITOR_SUFFIX_NODE_SUFFIX_BYTES_TOTAL ".bytes.total"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_MEMORY_BANDWIDTH_MONITOR_SUFFIX_NODE_SUFFIX_ID ".id"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_MEMORY_BANDWIDTH_MONITOR_SUFFIX_VCPUS ".vcpus"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_NET_COUNT "net.count"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_NET_PREFIX "net."
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_NET_SUFFIX_NAME ".name"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_NET_SUFFIX_RX_BYTES ".rx.bytes"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_NET_SUFFIX_RX_DROP ".rx.drop"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_NET_SUFFIX_RX_ERRS ".rx.errs"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_NET_SUFFIX_RX_PKTS ".rx.pkts"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_NET_SUFFIX_TX_BYTES ".tx.bytes"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_NET_SUFFIX_TX_DROP ".tx.drop"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_NET_SUFFIX_TX_ERRS ".tx.errs"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_NET_SUFFIX_TX_PKTS ".tx.pkts"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_PERF_ALIGNMENT_FAULTS "perf.alignment_faults"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_PERF_BRANCH_INSTRUCTIONS "perf.branch_instructions"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_PERF_BRANCH_MISSES "perf.branch_misses"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_PERF_BUS_CYCLES "perf.bus_cycles"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_PERF_CACHE_MISSES "perf.cache_misses"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_PERF_CACHE_REFERENCES "perf.cache_references"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_PERF_CMT "perf.cmt"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_PERF_CONTEXT_SWITCHES "perf.context_switches"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_PERF_CPU_CLOCK "perf.cpu_clock"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_PERF_CPU_CYCLES "perf.cpu_cycles"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_PERF_CPU_MIGRATIONS "perf.cpu_migrations"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_PERF_EMULATION_FAULTS "perf.emulation_faults"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_PERF_INSTRUCTIONS "perf.instructions"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_PERF_MBML "perf.mbml"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_PERF_MBMT "perf.mbmt"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_PERF_PAGE_FAULTS "perf.page_faults"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_PERF_PAGE_FAULTS_MAJ "perf.page_faults_maj"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_PERF_PAGE_FAULTS_MIN "perf.page_faults_min"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_PERF_REF_CPU_CYCLES "perf.ref_cpu_cycles"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_PERF_STALLED_CYCLES_BACKEND "perf.stalled_cycles_backend"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_PERF_STALLED_CYCLES_FRONTEND "perf.stalled_cycles_frontend"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_PERF_TASK_CLOCK "perf.task_clock"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_STATE_REASON "state.reason"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_STATE_STATE "state.state"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_VCPU_CURRENT "vcpu.current"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_VCPU_MAXIMUM "vcpu.maximum"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_VCPU_PREFIX "vcpu."
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_VCPU_SUFFIX_DELAY ".delay"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_VCPU_SUFFIX_HALTED ".halted"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_VCPU_SUFFIX_STATE ".state"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_VCPU_SUFFIX_TIME ".time"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_VCPU_SUFFIX_WAIT ".wait"
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
#  define VIR_DOMAIN_STATS_VM_PREFIX "vm."
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 9)
#  define VIR_DOMAIN_TUNABLE_BLKDEV_DISK "blkdeviotune.disk"
#endif

#if !LIBVIR_CHECK_VERSION(3, 0, 0)
#  define VIR_DOMAIN_TUNABLE_BLKDEV_GROUP_NAME "blkdeviotune.group_name"
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 9)
#  define VIR_DOMAIN_TUNABLE_BLKDEV_READ_BYTES_SEC "blkdeviotune.read_bytes_sec"
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 11)
#  define VIR_DOMAIN_TUNABLE_BLKDEV_READ_BYTES_SEC_MAX "blkdeviotune.read_bytes_sec_max"
#endif

#if !LIBVIR_CHECK_VERSION(2, 4, 0)
#  define VIR_DOMAIN_TUNABLE_BLKDEV_READ_BYTES_SEC_MAX_LENGTH "blkdeviotune.read_bytes_sec_max_length"
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 9)
#  define VIR_DOMAIN_TUNABLE_BLKDEV_READ_IOPS_SEC "blkdeviotune.read_iops_sec"
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 11)
#  define VIR_DOMAIN_TUNABLE_BLKDEV_READ_IOPS_SEC_MAX "blkdeviotune.read_iops_sec_max"
#endif

#if !LIBVIR_CHECK_VERSION(2, 4, 0)
#  define VIR_DOMAIN_TUNABLE_BLKDEV_READ_IOPS_SEC_MAX_LENGTH "blkdeviotune.read_iops_sec_max_length"
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 11)
#  define VIR_DOMAIN_TUNABLE_BLKDEV_SIZE_IOPS_SEC "blkdeviotune.size_iops_sec"
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 9)
#  define VIR_DOMAIN_TUNABLE_BLKDEV_TOTAL_BYTES_SEC "blkdeviotune.total_bytes_sec"
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 11)
#  define VIR_DOMAIN_TUNABLE_BLKDEV_TOTAL_BYTES_SEC_MAX "blkdeviotune.total_bytes_sec_max"
#endif

#if !LIBVIR_CHECK_VERSION(2, 4, 0)
#  define VIR_DOMAIN_TUNABLE_BLKDEV_TOTAL_BYTES_SEC_MAX_LENGTH "blkdeviotune.total_bytes_sec_max_length"
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 9)
#  define VIR_DOMAIN_TUNABLE_BLKDEV_TOTAL_IOPS_SEC "blkdeviotune.total_iops_sec"
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 11)
#  define VIR_DOMAIN_TUNABLE_BLKDEV_TOTAL_IOPS_SEC_MAX "blkdeviotune.total_iops_sec_max"
#endif

#if !LIBVIR_CHECK_VERSION(2, 4, 0)
#  define VIR_DOMAIN_TUNABLE_BLKDEV_TOTAL_IOPS_SEC_MAX_LENGTH "blkdeviotune.total_iops_sec_max_length"
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 9)
#  define VIR_DOMAIN_TUNABLE_BLKDEV_WRITE_BYTES_SEC "blkdeviotune.write_bytes_sec"
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 11)
#  define VIR_DOMAIN_TUNABLE_BLKDEV_WRITE_BYTES_SEC_MAX "blkdeviotune.write_bytes_sec_max"
#endif

#if !LIBVIR_CHECK_VERSION(2, 4, 0)
#  define VIR_DOMAIN_TUNABLE_BLKDEV_WRITE_BYTES_SEC_MAX_LENGTH "blkdeviotune.write_bytes_sec_max_length"
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 9)
#  define VIR_DOMAIN_TUNABLE_BLKDEV_WRITE_IOPS_SEC "blkdeviotune.write_iops_sec"
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 11)
#  define VIR_DOMAIN_TUNABLE_BLKDEV_WRITE_IOPS_SEC_MAX "blkdeviotune.write_iops_sec_max"
#endif

#if !LIBVIR_CHECK_VERSION(2, 4, 0)
#  define VIR_DOMAIN_TUNABLE_BLKDEV_WRITE_IOPS_SEC_MAX_LENGTH "blkdeviotune.write_iops_sec_max_length"
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 9)
#  define VIR_DOMAIN_TUNABLE_CPU_CPU_SHARES "cputune.cpu_shares"
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 9)
#  define VIR_DOMAIN_TUNABLE_CPU_EMULATORPIN "cputune.emulatorpin"
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 9)
#  define VIR_DOMAIN_TUNABLE_CPU_EMULATOR_PERIOD "cputune.emulator_period"
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 9)
#  define VIR_DOMAIN_TUNABLE_CPU_EMULATOR_QUOTA "cputune.emulator_quota"
#endif

#if !LIBVIR_CHECK_VERSION(1, 3, 3)
#  define VIR_DOMAIN_TUNABLE_CPU_GLOBAL_PERIOD "cputune.global_period"
#endif

#if !LIBVIR_CHECK_VERSION(1, 3, 3)
#  define VIR_DOMAIN_TUNABLE_CPU_GLOBAL_QUOTA "cputune.global_quota"
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 14)
#  define VIR_DOMAIN_TUNABLE_CPU_IOTHREADSPIN "cputune.iothreadpin%u"
#endif

#if !LIBVIR_CHECK_VERSION(2, 2, 0)
#  define VIR_DOMAIN_TUNABLE_CPU_IOTHREAD_PERIOD "cputune.iothread_period"
#endif

#if !LIBVIR_CHECK_VERSION(2, 2, 0)
#  define VIR_DOMAIN_TUNABLE_CPU_IOTHREAD_QUOTA "cputune.iothread_quota"
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 9)
#  define VIR_DOMAIN_TUNABLE_CPU_VCPUPIN "cputune.vcpupin%u"
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 9)
#  define VIR_DOMAIN_TUNABLE_CPU_VCPU_PERIOD "cputune.vcpu_period"
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 9)
#  define VIR_DOMAIN_TUNABLE_CPU_VCPU_QUOTA "cputune.vcpu_quota"
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 5)
#  define VIR_KEYCODE_SET_RFB VIR_KEYCODE_SET_QNUM
#endif

#if !LIBVIR_CHECK_VERSION(2, 0, 0)
#  define VIR_MIGRATE_PARAM_AUTO_CONVERGE_INCREMENT "auto_converge.increment"
#endif

#if !LIBVIR_CHECK_VERSION(2, 0, 0)
#  define VIR_MIGRATE_PARAM_AUTO_CONVERGE_INITIAL "auto_converge.initial"
#endif

#if !LIBVIR_CHECK_VERSION(1, 1, 0)
#  define VIR_MIGRATE_PARAM_BANDWIDTH "bandwidth"
#endif

#if !LIBVIR_CHECK_VERSION(11, 1, 0)
#  define VIR_MIGRATE_PARAM_BANDWIDTH_AVAIL_SWITCHOVER "bandwidth.avail.switchover"
#endif

#if !LIBVIR_CHECK_VERSION(5, 1, 0)
#  define VIR_MIGRATE_PARAM_BANDWIDTH_POSTCOPY "bandwidth.postcopy"
#endif

#if !LIBVIR_CHECK_VERSION(1, 3, 4)
#  define VIR_MIGRATE_PARAM_COMPRESSION "compression"
#endif

#if !LIBVIR_CHECK_VERSION(1, 3, 4)
#  define VIR_MIGRATE_PARAM_COMPRESSION_MT_DTHREADS "compression.mt.dthreads"
#endif

#if !LIBVIR_CHECK_VERSION(1, 3, 4)
#  define VIR_MIGRATE_PARAM_COMPRESSION_MT_LEVEL "compression.mt.level"
#endif

#if !LIBVIR_CHECK_VERSION(1, 3, 4)
#  define VIR_MIGRATE_PARAM_COMPRESSION_MT_THREADS "compression.mt.threads"
#endif

#if !LIBVIR_CHECK_VERSION(1, 3, 4)
#  define VIR_MIGRATE_PARAM_COMPRESSION_XBZRLE_CACHE "compression.xbzrle.cache"
#endif

#if !LIBVIR_CHECK_VERSION(9, 4, 0)
#  define VIR_MIGRATE_PARAM_COMPRESSION_ZLIB_LEVEL "compression.zlib.level"
#endif

#if !LIBVIR_CHECK_VERSION(9, 4, 0)
#  define VIR_MIGRATE_PARAM_COMPRESSION_ZSTD_LEVEL "compression.zstd.level"
#endif

#if !LIBVIR_CHECK_VERSION(1, 1, 0)
#  define VIR_MIGRATE_PARAM_DEST_NAME "destination_name"
#endif

#if !LIBVIR_CHECK_VERSION(1, 1, 0)
#  define VIR_MIGRATE_PARAM_DEST_XML "destination_xml"
#endif

#if !LIBVIR_CHECK_VERSION(1, 3, 3)
#  define VIR_MIGRATE_PARAM_DISKS_PORT "disks_port"
#endif

#if !LIBVIR_CHECK_VERSION(6, 8, 0)
#  define VIR_MIGRATE_PARAM_DISKS_URI "disks_uri"
#endif

#if !LIBVIR_CHECK_VERSION(1, 1, 0)
#  define VIR_MIGRATE_PARAM_GRAPHICS_URI "graphics_uri"
#endif

#if !LIBVIR_CHECK_VERSION(1, 1, 4)
#  define VIR_MIGRATE_PARAM_LISTEN_ADDRESS "listen_address"
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 17)
#  define VIR_MIGRATE_PARAM_MIGRATE_DISKS "migrate_disks"
#endif

#if !LIBVIR_CHECK_VERSION(10, 9, 0)
#  define VIR_MIGRATE_PARAM_MIGRATE_DISKS_DETECT_ZEROES "migrate_disks_detect_zeroes"
#endif

#if !LIBVIR_CHECK_VERSION(5, 2, 0)
#  define VIR_MIGRATE_PARAM_PARALLEL_CONNECTIONS "parallel.connections"
#endif

#if !LIBVIR_CHECK_VERSION(1, 3, 4)
#  define VIR_MIGRATE_PARAM_PERSIST_XML "persistent_xml"
#endif

#if !LIBVIR_CHECK_VERSION(6, 0, 0)
#  define VIR_MIGRATE_PARAM_TLS_DESTINATION "tls.destination"
#endif

#if !LIBVIR_CHECK_VERSION(1, 1, 0)
#  define VIR_MIGRATE_PARAM_URI "migrate_uri"
#endif

#if !LIBVIR_CHECK_VERSION(5, 5, 0)
#  define VIR_NETWORK_PORT_BANDWIDTH_IN_AVERAGE "inbound.average"
#endif

#if !LIBVIR_CHECK_VERSION(5, 5, 0)
#  define VIR_NETWORK_PORT_BANDWIDTH_IN_BURST "inbound.burst"
#endif

#if !LIBVIR_CHECK_VERSION(5, 5, 0)
#  define VIR_NETWORK_PORT_BANDWIDTH_IN_FLOOR "inbound.floor"
#endif

#if !LIBVIR_CHECK_VERSION(5, 5, 0)
#  define VIR_NETWORK_PORT_BANDWIDTH_IN_PEAK "inbound.peak"
#endif

#if !LIBVIR_CHECK_VERSION(5, 5, 0)
#  define VIR_NETWORK_PORT_BANDWIDTH_OUT_AVERAGE "outbound.average"
#endif

#if !LIBVIR_CHECK_VERSION(5, 5, 0)
#  define VIR_NETWORK_PORT_BANDWIDTH_OUT_BURST "outbound.burst"
#endif

#if !LIBVIR_CHECK_VERSION(5, 5, 0)
#  define VIR_NETWORK_PORT_BANDWIDTH_OUT_PEAK "outbound.peak"
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 3)
#  define VIR_NODE_CPU_STATS_FIELD_LENGTH 80
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 3)
#  define VIR_NODE_CPU_STATS_IDLE "idle"
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 2)
#  define VIR_NODE_CPU_STATS_INTR "intr"
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 3)
#  define VIR_NODE_CPU_STATS_IOWAIT "iowait"
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 3)
#  define VIR_NODE_CPU_STATS_KERNEL "kernel"
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 3)
#  define VIR_NODE_CPU_STATS_USER "user"
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 3)
#  define VIR_NODE_CPU_STATS_UTILIZATION "utilization"
#endif

#if !LIBVIR_CHECK_VERSION(0, 10, 2)
#  define VIR_NODE_MEMORY_SHARED_FULL_SCANS "shm_full_scans"
#endif

#if !LIBVIR_CHECK_VERSION(1, 0, 0)
#  define VIR_NODE_MEMORY_SHARED_MERGE_ACROSS_NODES "shm_merge_across_nodes"
#endif

#if !LIBVIR_CHECK_VERSION(0, 10, 2)
#  define VIR_NODE_MEMORY_SHARED_PAGES_SHARED "shm_pages_shared"
#endif

#if !LIBVIR_CHECK_VERSION(0, 10, 2)
#  define VIR_NODE_MEMORY_SHARED_PAGES_SHARING "shm_pages_sharing"
#endif

#if !LIBVIR_CHECK_VERSION(0, 10, 2)
#  define VIR_NODE_MEMORY_SHARED_PAGES_TO_SCAN "shm_pages_to_scan"
#endif

#if !LIBVIR_CHECK_VERSION(0, 10, 2)
#  define VIR_NODE_MEMORY_SHARED_PAGES_UNSHARED "shm_pages_unshared"
#endif

#if !LIBVIR_CHECK_VERSION(0, 10, 2)
#  define VIR_NODE_MEMORY_SHARED_PAGES_VOLATILE "shm_pages_volatile"
#endif

#if !LIBVIR_CHECK_VERSION(0, 10, 2)
#  define VIR_NODE_MEMORY_SHARED_SLEEP_MILLISECS "shm_sleep_millisecs"
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 3)
#  define VIR_NODE_MEMORY_STATS_BUFFERS "buffers"
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 3)
#  define VIR_NODE_MEMORY_STATS_CACHED "cached"
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 3)
#  define VIR_NODE_MEMORY_STATS_FIELD_LENGTH 80
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 3)
#  define VIR_NODE_MEMORY_STATS_FREE "free"
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 3)
#  define VIR_NODE_MEMORY_STATS_TOTAL "total"
#endif

#if !LIBVIR_CHECK_VERSION(4, 5, 0)
#  define VIR_NODE_SEV_CBITPOS "cbitpos"
#endif

#if !LIBVIR_CHECK_VERSION(4, 5, 0)
#  define VIR_NODE_SEV_CERT_CHAIN "cert-chain"
#endif

#if !LIBVIR_CHECK_VERSION(8, 4, 0)
#  define VIR_NODE_SEV_CPU0_ID "cpu0-id"
#endif

#if !LIBVIR_CHECK_VERSION(8, 0, 0)
#  define VIR_NODE_SEV_MAX_ES_GUESTS "max-es-guests"
#endif

#if !LIBVIR_CHECK_VERSION(8, 0, 0)
#  define VIR_NODE_SEV_MAX_GUESTS "max-guests"
#endif

#if !LIBVIR_CHECK_VERSION(4, 5, 0)
#  define VIR_NODE_SEV_PDH "pdh"
#endif

#if !LIBVIR_CHECK_VERSION(4, 5, 0)
#  define VIR_NODE_SEV_REDUCED_PHYS_BITS "reduced-phys-bits"
#endif

#if !LIBVIR_CHECK_VERSION(3, 2, 0)
#  define VIR_PERF_PARAM_ALIGNMENT_FAULTS "alignment_faults"
#endif

#if !LIBVIR_CHECK_VERSION(3, 0, 0)
#  define VIR_PERF_PARAM_BRANCH_INSTRUCTIONS "branch_instructions"
#endif

#if !LIBVIR_CHECK_VERSION(3, 0, 0)
#  define VIR_PERF_PARAM_BRANCH_MISSES "branch_misses"
#endif

#if !LIBVIR_CHECK_VERSION(3, 0, 0)
#  define VIR_PERF_PARAM_BUS_CYCLES "bus_cycles"
#endif

#if !LIBVIR_CHECK_VERSION(2, 3, 0)
#  define VIR_PERF_PARAM_CACHE_MISSES "cache_misses"
#endif

#if !LIBVIR_CHECK_VERSION(2, 3, 0)
#  define VIR_PERF_PARAM_CACHE_REFERENCES "cache_references"
#endif

#if !LIBVIR_CHECK_VERSION(1, 3, 3)
#  define VIR_PERF_PARAM_CMT "cmt"
#endif

#if !LIBVIR_CHECK_VERSION(3, 2, 0)
#  define VIR_PERF_PARAM_CONTEXT_SWITCHES "context_switches"
#endif

#if !LIBVIR_CHECK_VERSION(3, 2, 0)
#  define VIR_PERF_PARAM_CPU_CLOCK "cpu_clock"
#endif

#if !LIBVIR_CHECK_VERSION(2, 3, 0)
#  define VIR_PERF_PARAM_CPU_CYCLES "cpu_cycles"
#endif

#if !LIBVIR_CHECK_VERSION(3, 2, 0)
#  define VIR_PERF_PARAM_CPU_MIGRATIONS "cpu_migrations"
#endif

#if !LIBVIR_CHECK_VERSION(3, 2, 0)
#  define VIR_PERF_PARAM_EMULATION_FAULTS "emulation_faults"
#endif

#if !LIBVIR_CHECK_VERSION(2, 3, 0)
#  define VIR_PERF_PARAM_INSTRUCTIONS "instructions"
#endif

#if !LIBVIR_CHECK_VERSION(1, 3, 5)
#  define VIR_PERF_PARAM_MBML "mbml"
#endif

#if !LIBVIR_CHECK_VERSION(1, 3, 5)
#  define VIR_PERF_PARAM_MBMT "mbmt"
#endif

#if !LIBVIR_CHECK_VERSION(3, 2, 0)
#  define VIR_PERF_PARAM_PAGE_FAULTS "page_faults"
#endif

#if !LIBVIR_CHECK_VERSION(3, 2, 0)
#  define VIR_PERF_PARAM_PAGE_FAULTS_MAJ "page_faults_maj"
#endif

#if !LIBVIR_CHECK_VERSION(3, 2, 0)
#  define VIR_PERF_PARAM_PAGE_FAULTS_MIN "page_faults_min"
#endif

#if !LIBVIR_CHECK_VERSION(3, 0, 0)
#  define VIR_PERF_PARAM_REF_CPU_CYCLES "ref_cpu_cycles"
#endif

#if !LIBVIR_CHECK_VERSION(3, 0, 0)
#  define VIR_PERF_PARAM_STALLED_CYCLES_BACKEND "stalled_cycles_backend"
#endif

#if !LIBVIR_CHECK_VERSION(3, 0, 0)
#  define VIR_PERF_PARAM_STALLED_CYCLES_FRONTEND "stalled_cycles_frontend"
#endif

#if !LIBVIR_CHECK_VERSION(3, 2, 0)
#  define VIR_PERF_PARAM_TASK_CLOCK "task_clock"
#endif

#if !LIBVIR_CHECK_VERSION(0, 6, 1)
#  define VIR_SECURITY_DOI_BUFLEN (256 + 1)
#endif

#if !LIBVIR_CHECK_VERSION(0, 6, 1)
#  define VIR_SECURITY_LABEL_BUFLEN (4096 + 1)
#endif

#if !LIBVIR_CHECK_VERSION(0, 6, 1)
#  define VIR_SECURITY_MODEL_BUFLEN (256 + 1)
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 2)
#  define VIR_TYPED_PARAM_FIELD_LENGTH 80
#endif

#if !LIBVIR_CHECK_VERSION(0, 2, 0)
#  define VIR_UUID_BUFLEN (16)
#endif

#if !LIBVIR_CHECK_VERSION(0, 2, 0)
#  define VIR_UUID_STRING_BUFLEN (36+1)
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 0)
#  define _virBlkioParameter _virTypedParameter
#endif

#if !LIBVIR_CHECK_VERSION(0, 8, 5)
#  define _virMemoryParameter _virTypedParameter
#endif

#if !LIBVIR_CHECK_VERSION(0, 2, 3)
#  define _virSchedParameter _virTypedParameter
#endif
