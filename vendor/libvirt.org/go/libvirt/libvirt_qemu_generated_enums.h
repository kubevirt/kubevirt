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

/* enum virConnectDomainQemuMonitorEventRegisterFlags */
#  if !LIBVIR_CHECK_VERSION(1, 2, 3)
#    define VIR_CONNECT_DOMAIN_QEMU_MONITOR_EVENT_REGISTER_REGEX (1 << 0)
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 3)
#    define VIR_CONNECT_DOMAIN_QEMU_MONITOR_EVENT_REGISTER_NOCASE (1 << 1)
#  endif

/* enum virDomainQemuAgentCommandTimeoutValues */
#  if !LIBVIR_CHECK_VERSION(0, 10, 0)
#    define VIR_DOMAIN_QEMU_AGENT_COMMAND_BLOCK VIR_DOMAIN_AGENT_RESPONSE_TIMEOUT_BLOCK
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 10, 0)
#    define VIR_DOMAIN_QEMU_AGENT_COMMAND_MIN VIR_DOMAIN_AGENT_RESPONSE_TIMEOUT_BLOCK
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 10, 0)
#    define VIR_DOMAIN_QEMU_AGENT_COMMAND_DEFAULT VIR_DOMAIN_AGENT_RESPONSE_TIMEOUT_DEFAULT
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 10, 0)
#    define VIR_DOMAIN_QEMU_AGENT_COMMAND_NOWAIT VIR_DOMAIN_AGENT_RESPONSE_TIMEOUT_NOWAIT
#  endif
#  if !LIBVIR_CHECK_VERSION(1, 2, 15)
#    define VIR_DOMAIN_QEMU_AGENT_COMMAND_SHUTDOWN 60
#  endif

/* enum virDomainQemuMonitorCommandFlags */
#  if !LIBVIR_CHECK_VERSION(0, 8, 8)
#    define VIR_DOMAIN_QEMU_MONITOR_COMMAND_DEFAULT 0
#  endif
#  if !LIBVIR_CHECK_VERSION(0, 8, 8)
#    define VIR_DOMAIN_QEMU_MONITOR_COMMAND_HMP (1 << 0)
#  endif

