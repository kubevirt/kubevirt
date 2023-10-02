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
#  define LIBVIR_VERSION_NUMBER 9004000
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
